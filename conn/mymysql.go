package conn

import (
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native"
)

type MyMySQLOpener struct {
	hostPort string
	user     string
	pass     string
	dbName   string
}

type MyMySQLConn struct {
	conn mysql.Conn
}

type MyMySQLTx struct {
	tx mysql.Transaction
}

type MyMySQLRows struct {
	rows mysql.Result
}

type MyMySQLRow  struct {
	rows mysql.Result
	row mysql.Row
}

func NewMyMySQLOpener(hostPort string, user, pass string, dbName string) *MyMySQLOpener {
	return &MyMySQLOpener{
		hostPort: hostPort,
		user:     user,
		pass:     pass,
		dbName:   dbName,
	}
}

func (m *MyMySQLOpener) Open() (DBConn, error) {
	conn := mysql.New("tcp", "", m.hostPort, m.user, m.pass, m.dbName)
	err := conn.Connect()
	return &MyMySQLConn{conn}, err
}

func (c MyMySQLConn) Begin() (Tx, error) {
	tx, err := c.conn.Begin()
	if err != nil {
		return nil, err
	}
	return MyMySQLTx{tx}, nil
}

func (tx MyMySQLTx) Query(query string, args ...interface{}) (Rows, error) {
	_, result, err := tx.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &MyMySQLRows{result}, nil
}

func (tx MyMySQLTx) Commit() error {
	return tx.tx.Commit()
}

func (tx MyMySQLTx) Rollback() error {
	return tx.tx.Rollback()
}

func (c MyMySQLConn) Query(query string, args ...interface{}) (Rows, error) {
	_, result, err := c.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &MyMySQLRows{result}, nil
}

func (c MyMySQLConn) Close() error {
	return c.conn.Close()
}

func (rs MyMySQLRows) Close() error {
	return nil
}

func (rs MyMySQLRows) First() (Row, error) {
	row, err := rs.rows.GetFirstRow()
	if err != nil {
		return nil, err
	}
	return &MyMySQLRow{rs.rows, row}, nil
}

func (r MyMySQLRow) Int(name string) (int, error) {
	for i, field := range r.rows.Fields() {
		if field.Name == name {
			intValue := r.row.Int(i)
			return int(intValue), nil
		}
	}
	return 0, NoColumnError
}
