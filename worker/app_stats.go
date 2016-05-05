package worker

import (
	"time"
	"github.com/500px/go-utils/metrics"
)

type Stat struct {
	userId  int
    photoId int
}

type AppStats struct {
	start        time.Time
	first        time.Time
	stats        chan Stat
	userIds      map[int]struct{}
	photoIds     map[int]struct{}
	statsdClient metrics.StatsdClient
}

var (
	zeroTime = time.Time{}
)

func NewAppStats(stats chan Stat, statsdClient metrics.StatsdClient) *AppStats {
	return &AppStats{
		stats:        stats,
		userIds:      map[int]struct{}{},
		photoIds:     map[int]struct{}{},
		statsdClient: statsdClient,
	}
}

func (a AppStats) ReportSpeed(t time.Time) {
	if a.first == zeroTime {
		a.start = time.Now().UTC()
		a.first = t
	} else {
		elapsed := time.Now().Unix() - a.start.Unix()
		replayed := t.Unix() - a.first.Unix()
		if elapsed > 0 && replayed > 0 {
			a.statsdClient.Gauge("replay-speed", float64(replayed) / float64(elapsed))
		}
	}
}

func (a AppStats) Run() {
	for stat := range a.stats {
		if stat.userId > 0 {
			a.userIds[stat.userId] = struct{}{}
			a.statsdClient.Gauge("number.user_ids", float64(len(a.userIds)))
		}
		if stat.photoId > 0 {
			a.photoIds[stat.photoId] = struct{}{}
			a.statsdClient.Gauge("number.photo_ids", float64(len(a.photoIds)))
		}
	}
}
