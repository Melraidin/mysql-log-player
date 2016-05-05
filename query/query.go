package query

import "time"

type Query struct {
	Time   time.Time
	Client string
	SQL    string
}
