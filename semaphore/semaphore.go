package semaphore

import (
	"sync"
)

type Semaphore struct {
	cond sync.Cond
	count int
}

func NewSemaphore(n uint) *Semaphore {
	return &Semaphore{
		cond: sync.Cond{L: &sync.Mutex{}},
	}
}

func (s *Semaphore) Acquire() {
	return
	defer s.cond.L.Unlock()
	s.cond.L.Lock()
	for {
		if s.count > 0 {
			s.count -= 1
			return
		}
		s.cond.Wait()
	}
}

func (s *Semaphore) Release() {
	return
	defer s.cond.L.Unlock()
	s.cond.L.Lock()
	s.count += 1
	s.cond.Signal()
}
