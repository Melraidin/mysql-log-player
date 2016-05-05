package worker

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	logger "github.com/500px/go-utils/chatty_logger"
	"github.com/500px/go-utils/metrics"
	"github.com/melraidin/mysql-log-player/conn"
	"github.com/melraidin/mysql-log-player/query"
	"math"
)

var (
	// Size of command buffer for new workers.
	BufferSize = 1000
	concurrent = int64(0)
)

type Worker struct {
	Logtime0    time.Time
	Runtime0    time.Time
	ReplaySpeed float64
	client      string
	client_type string
	queryChan   <-chan query.Query
	dbConns     *conn.DBConns
	wg          *sync.WaitGroup
	stats       chan<- Stat
	metrics     metrics.StatsdClient
}

func NewWorker(client string, logtime0, runtime0 time.Time, replaySpeed float64, dbConns *conn.DBConns, wg *sync.WaitGroup, stats chan<- Stat, statsdClient metrics.StatsdClient) chan<- query.Query {
	queryChan := make(chan query.Query, BufferSize)

	var client_type string
	if strings.HasPrefix(client, "10.1.1.81:") || strings.HasPrefix(client, "10.1.1.82:") ||
		strings.HasPrefix(client, "10.1.1.83:") || strings.HasPrefix(client, "10.1.1.84:") ||
		strings.HasPrefix(client, "10.1.1.85:") || strings.HasPrefix(client, "10.1.1.86:") {
		client_type = "sq"
	} else if strings.HasPrefix(client, "10.1.1.21:") || strings.HasPrefix(client, "10.1.1.22:") ||
	          strings.HasPrefix(client, "10.1.1.23:") || strings.HasPrefix(client, "10.1.1.24:") ||
			  strings.HasPrefix(client, "10.1.1.25:") || strings.HasPrefix(client, "10.1.1.26:") {
		client_type = "app"
	} else if strings.HasPrefix(client, "10.1.1.141:") || strings.HasPrefix(client, "10.1.1.142:") ||
				strings.HasPrefix(client, "10.1.1.147:") || strings.HasPrefix(client, "10.1.1.144:") ||
				strings.HasPrefix(client, "10.1.1.148:") {
		client_type = "prime"
	} else if strings.HasPrefix(client, "10.1.1.68:") || strings.HasPrefix(client, "10.1.1.67:") {
		client_type = "search"
	} else if strings.HasPrefix(client, "10.1.1.171:") || strings.HasPrefix(client, "10.1.1.172:") {
		client_type = "upload"
	} else {
		client_type = "other"
		for _, i := range []int{31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
						220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 116, 117} {
			if strings.HasPrefix(client, "10.1.1."+strconv.Itoa(i)+":") {
				client_type = "api"
				break
			}
		}
	}

	worker := Worker{
		Logtime0:    logtime0,
		Runtime0:    runtime0,
		ReplaySpeed: replaySpeed,
		client:      client,
		client_type: client_type,
		queryChan:   queryChan,
		dbConns:     dbConns,
		wg:          wg,
		stats:       stats,
		metrics:     statsdClient,
	}

	wg.Add(1)

	go func() {
		worker.Run()
	}()
	time.Sleep(25 * time.Millisecond)

	return queryChan
}

func (w *Worker) AcquireDB() (conn.DBConn, error) {
	logger.Debugf("[%v] AcquireDB: entered", w.client)
	defer logger.Debugf("[%v] AcquireDB: exiting", w.client)

	timer := metrics.NewStopwatch()
	db, err := w.dbConns.AcquireDB()
	if err != nil {
		return nil, fmt.Errorf("[%s] error from dbOpener: %v", w.client, err)
	}

	timer.Stop()
	ms := timer.Ms()
	w.metrics.Histogram("worker.acquiredb", float64(ms), "client:"+ w.client)
	if ms > 100 {
		logger.Infof("Took %d ms to open DB connection.", ms)
	}

	return db, nil
}

func (w *Worker) Run() {
	var db conn.DBConn

	var tx conn.Tx
	for query := range w.queryChan {
		w.metrics.Gauge("query.chan", float64(len(w.queryChan)), "client:" + w.client)
		//query = strings.TrimSpace(query)
		//if !strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		//	//logger.Debugf("[%s] Skipping non-SELECT query: %v", query)
		//	continue
		//}
		if db == nil {
			var err error
			if db, err = w.AcquireDB(); err != nil {
				logger.Errorf("Error from AcquireDB: %v", err)
				continue
			}
			time.Sleep(5 * time.Millisecond)
		}

		if strings.HasPrefix(query.SQL, "BEGIN") {
			var err error
			if tx, err = db.Begin(); err != nil {
				logger.Errorf("Error from db.Begin: %v", err)
			}
		} else if strings.HasPrefix(query.SQL, "COMMIT") {
			if tx != nil {
				if err := tx.Commit(); err != nil {
					logger.Errorf("Error from db.Commit: %v", err)
				}
				tx = nil
			}
		} else if strings.HasPrefix(query.SQL, "ROLLBACK") {
			if tx != nil {
				if err := tx.Rollback(); err != nil {
					logger.Errorf("Error from db.Begin: %v", err)
				}
				tx = nil
			}
		} else {
			if tx != nil {
				_ = w.doQuery(tx, query)
			} else {
				_ = w.doQuery(db, query)
			}
		}

		if tx == nil && len(w.queryChan) == 0 {
			db = w.dbConns.ReleaseDB(db)
			if db == nil {
				w.metrics.Incr("releasedb", 1, "client:" + w.client)
			}
		}
	}
	w.wg.Done()
}

func (w *Worker) doQuery(db conn.QConn, q query.Query) error {
	replayTime := w.Runtime0.Add(q.Time.Sub(w.Logtime0) * 10 / time.Duration(10 * w.ReplaySpeed))
	now := time.Now()
	if replayTime.After(now) {
		d := replayTime.Sub(now)
		logtimeSecond := time.Duration(math.Ceil(float64(time.Second) / w.ReplaySpeed))
		if d > logtimeSecond {
			r := rand.Int63n(int64(logtimeSecond))
			time.Sleep(time.Duration(r))
		}
	}

	timer := metrics.NewStopwatch()
	active := atomic.AddInt64(&concurrent, 1)

	sql := q.SQL
	rows, err := db.Query(sql)

	atomic.AddInt64(&concurrent, -1)
	timer.Stop()
	w.metrics.Gauge("query_worker.concurrent", float64(active))

	if strings.HasPrefix(sql, "SELECT  `photos`.* FROM `photos` WHERE `photos`.`id` = ") {
		stat := Stat{}
		if stat.photoId, err = strconv.Atoi(strings.SplitN(sql[55:], " ", 2)[0]); err != nil {
			logger.Warnf("[%s] Error extracting photo 'id': %v", w.client, err)
		} else {
			w.stats <- stat
		}
	} else if strings.HasPrefix(sql, "SELECT  `users`.* FROM `users` WHERE `users`.`id` = ") {
		stat := Stat{}
		if stat.userId, err = strconv.Atoi(strings.SplitN(sql[52:], " ", 2)[0]); err != nil {
			logger.Warnf("[%s] Error extracting user 'id': %v", w.client, err)
		} else {
			w.stats <- stat
		}
	}

	if err != nil {
		tags := []string{"status:error", "client:"+ w.client, "client_type:"+ w.client_type}

		s := err.Error()
		var code string
		if strings.HasPrefix(s, "Received #") {
			code = strings.SplitN(s[10:], " ", 2)[0]
			tags = append(tags, "code:"+code)
		}
		w.metrics.Histogram("query.duration", float64(timer.Ms()), tags...)

		if code == "1062" || code == "1227" {
			logger.Debugf("[%s] error '%+v' running query: [%s]", w.client, err, sql)
		} else {
			logger.Warnf("[%s] error '%+v' running query: [%s]", w.client, err, sql)
		}
		return err
	}
	w.metrics.Histogram("query.duration", float64(timer.Ms()), "status:ok", "client:"+ w.client, "client_type:"+ w.client_type)

	if err := rows.Close(); err != nil {
		logger.Errorf("[%s] error closing rows: %v", w.client, err)
	}

	return nil
}
