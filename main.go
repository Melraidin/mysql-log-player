package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
	//_ "github.com/ziutek/mymysql/godrv"
	_ "github.com/go-sql-driver/mysql"
	logger "github.com/500px/go-utils/chatty_logger"
	"github.com/500px/go-utils/metrics"
	"github.com/melraidin/mysql-log-player/conn"
	"github.com/melraidin/mysql-log-player/query"
	"github.com/melraidin/mysql-log-player/worker"
)

func main() {
	parseFlags()

	startDebugListener()

	statsdClient, err := metrics.NewStatsdClient("mysql_log_player", "", "127.0.0.1:8125")
	exitOnError(err)
	logger.Debugf("Created statsd client.")
	statsdClient.Incr("start", 1)

	logger.Debugf("Creating query pool:")

	hostPort := fmt.Sprintf("%s:3306", *dbHost)
	dbOpener := conn.NewMyMySQLOpener(hostPort, *dbUser, *dbPass, *dbName)

	//connectInfo := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?allowOldPasswords=1&autocommit=true", *dbUser, *dbPass, *dbHost, *dbName)
	//dbOpener := conn.NewGoMySQLOpener("mysql", connectInfo)

	//dbOpener := conn.NewSiddontangOpener(hostPort, *dbUser, *dbPass, *dbName)

	logger.Infof("Counting client queries...")
	var logtime1 time.Time
	clientQueries := map[string]int{}
	err = foreach(func(query *query.Query) {
		if query.Time.After(logtime1) {
			logtime1 = query.Time
		}
		clientQueries[query.Client]++
	})
	logtime0 := logtime1.Add(-1 * time.Hour)

	if err != nil && err != io.EOF {
		logger.Errorf("Failed read: %s", err)
		os.Exit(1)
	}

	logger.Infof("log query interval [%v..%v]", logtime0, logtime1)
	logger.Infof("%d unique clients", len(clientQueries))

	queryPool := worker.NewWorkerPool(dbOpener, *dbConcurrency, statsdClient)
	logger.Debugf("Created query pool.")

	time.Sleep(1 * time.Second)
	queryPool.Logtime0 = logtime0
	queryPool.Runtime0 = time.Now()
	queryPool.ReplaySpeed = *replaySpeed
	logger.Infof("%v Dispatching queries at %gx speed...", queryPool.Runtime0, queryPool.ReplaySpeed)

	i := 0
	err = foreach(func(query *query.Query) {
		i += 1
		//logger.Debugf("Dispatching query: %d", i)
		queryPool.Dispatch(query)
		//logger.Debugf("Dispatched query: %d", i)

		count := clientQueries[query.Client]
		clientQueries[query.Client]--
		if count == 1 {
			queryPool.Close(query.Client)
		}
	})
	if err != nil && err != io.EOF {
		logger.Errorf("Failed read: %s", err)
		os.Exit(1)
	}

	queryPool.Wait()
}

func foreach(f func(q *query.Query)) error {
	reader, err := getReader(*sourcePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	var query *query.Query
	for query, err = reader.Read(); err == nil; query, err = reader.Read() {
		f(query)
	}

	return nil
}

func startDebugListener() {
	go func() {
 		log.Println(http.ListenAndServe("localhost:6060", nil))
 	}()
}

func getReader(path string) (*query.Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %s", err)
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("could not unzip file: %s", err)
	}

	return query.NewReader(reader)
}

func exitOnError(err error) {
	if err != nil {
		logger.Errorf("Error pinging db: %v", err)
		os.Exit(1)
	}
}
