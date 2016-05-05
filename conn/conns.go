package conn

import (
	"sync"
	"github.com/500px/go-utils/chatty_logger"
)

type DBConns struct {
	maxConnections int
	made           int
	conns          chan DBConn
	dbOpener       DBOpener
	lock           sync.Mutex
}

func NewDBConns(maxConnections int, dbOpener DBOpener) *DBConns {
	return &DBConns{maxConnections, 0, make(chan DBConn, maxConnections), dbOpener, sync.Mutex{}}
}

func (dbs *DBConns) AcquireDB() (DBConn, error) {
	select {
	case db := <-dbs.conns:
		return db, nil
	default:
	}

	dbs.lock.Lock()
	if dbs.made < dbs.maxConnections {
		dbs.made++
		number := dbs.made
		dbs.lock.Unlock()
		logger.Infof("Creating connection #%d", number)
		return dbs.dbOpener.Open()
	}
	dbs.lock.Unlock()
	return <-dbs.conns, nil
}

func (dbs *DBConns) ReleaseDB(db DBConn) DBConn {
	if len(dbs.conns) <= dbs.maxConnections / 2 {
		return db
	}
	dbs.conns<- db
	return nil
}
