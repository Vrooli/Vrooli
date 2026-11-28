package optimization

import (
	"context"
	"sync"
)

type runLockSet struct {
	mu    sync.Mutex
	locks map[string]*runLock
}

type runLock struct {
	token chan struct{}
	refs  int
}

// lock returns an unlock function and removes unused keys so a long-lived
// optimizer does not retain one mutex for every historical run.
func (s *runLockSet) lock(ctx context.Context, runID string) (func(), error) {
	s.mu.Lock()
	if s.locks == nil {
		s.locks = make(map[string]*runLock)
	}
	entry := s.locks[runID]
	if entry == nil {
		entry = &runLock{token: make(chan struct{}, 1)}
		s.locks[runID] = entry
	}
	entry.refs++
	s.mu.Unlock()

	select {
	case entry.token <- struct{}{}:
		return func() {
			<-entry.token
			s.releaseRef(runID, entry)
		}, nil
	case <-ctx.Done():
		s.releaseRef(runID, entry)
		return nil, ctx.Err()
	}
}

func (s *runLockSet) releaseRef(runID string, entry *runLock) {
	s.mu.Lock()
	entry.refs--
	if entry.refs == 0 {
		delete(s.locks, runID)
	}
	s.mu.Unlock()
}
