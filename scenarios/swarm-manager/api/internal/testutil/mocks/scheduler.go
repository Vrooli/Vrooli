package mocks

import "sync"

type RecordingScheduler struct {
	mu    sync.Mutex
	calls int
}

func (r *RecordingScheduler) ScheduleAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
}

func (r *RecordingScheduler) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}
