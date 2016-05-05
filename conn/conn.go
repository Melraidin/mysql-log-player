package conn


import (
	"errors"
)

var (
	NoRowsError = errors.New("no rows")
	NoColumnError = errors.New("no columns")
)

type DBOpener interface {
	Open() (DBConn, error)
}

type QConn interface {
	Query(string, ...interface{}) (Rows, error)
}

type DBConn interface {
	Begin() (Tx, error)
	QConn
	Close() error
}

type Tx interface {
	QConn
	Commit() error
	Rollback() error
}

type Rows interface {
	First() (Row, error)
	Close() error
}

type Row interface {
	Int(string) (int, error)
}
