package worker

import (
	"fmt"
	"sync"
	"time"

	logger "github.com/500px/go-utils/chatty_logger"
	"github.com/500px/go-utils/metrics"
	"github.com/melraidin/mysql-log-player/conn"
	"github.com/melraidin/mysql-log-player/query"
)

type Concurrency struct {
	limit int
	count int
	cond  *sync.Cond
}

func (c *Concurrency) Inc() int {
	defer c.cond.L.Unlock()
	c.cond.L.Lock()

	waiting := false
	for c.count >= c.limit {
		if !waiting {
			waiting = true
			fmt.Printf("Concurrency: waiting...")
		}
		c.cond.Wait()
	}
	if waiting {
		fmt.Printf("Concurrency: .")
	}
	c.count += 1
	return c.count
}

func (c *Concurrency) Dec() int {
	defer c.cond.L.Unlock()
	c.cond.L.Lock()

	c.count -= 1
	c.cond.Broadcast()
	return c.count
}

type WorkerPool struct {
	Logtime0    time.Time
	Runtime0    time.Time
	ReplaySpeed float64
	dbConns     *conn.DBConns
	wg          *sync.WaitGroup
	concurrency *Concurrency
	connections map[string]chan<- query.Query
	appStats    *AppStats
	metrics     metrics.StatsdClient
}

func NewWorkerPool(dbOpener conn.DBOpener, limit int, metrics metrics.StatsdClient) *WorkerPool {
	stats := make(chan Stat, 2000)
	appStats := NewAppStats(stats, metrics)
	go appStats.Run()

	return &WorkerPool{
		dbConns:     conn.NewDBConns(limit, dbOpener),
		wg:          &sync.WaitGroup{},
		concurrency: &Concurrency{limit, 0, sync.NewCond(&sync.Mutex{})},
		connections: make(map[string]chan<- query.Query),
		appStats:    appStats,
		metrics:     metrics,
	}
}

func (p *WorkerPool) Dispatch(q *query.Query) {
	workerChan, ok := p.connections[q.Client]
	if !ok {
		logger.Infof("Created new worker for client: %s", q.Client)
		workerChan = NewWorker(q.Client, p.Logtime0, p.Runtime0, p.ReplaySpeed, p.dbConns, p.wg, p.appStats.stats, p.metrics)
		p.connections[q.Client] = workerChan
		p.metrics.Gauge("clients", float64(len(p.connections)))
	}
	workerChan <- *q
	p.appStats.ReportSpeed(q.Time)
}

func (p *WorkerPool) Close(client string) {
	if workerChan, ok := p.connections[client]; ok {
	        delete(p.connections, client)
	        close(workerChan)
        }
}

func (p *WorkerPool) Wait() {
	for _, connection := range p.connections {
		close(connection)
	}

	p.wg.Wait()
}
