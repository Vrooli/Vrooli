package testutil

import (
	"errors"
	"net/http"
	"sync"

	"swarm-manager/internal/dispatch"
)

// NoopInvalidator is a no-op implementation of dispatch.Invalidator for tests.
type NoopInvalidator struct{}

func (NoopInvalidator) DispatchInvalidate(...string) {}

var _ dispatch.Invalidator = NoopInvalidator{}

// NoopNodeDispatcher is a no-op implementation of dispatch.NodeDispatcher for tests.
type NoopNodeDispatcher struct{ NoopInvalidator }

func (NoopNodeDispatcher) DispatchNodeUpdate(string, string, any) {}

var _ dispatch.NodeDispatcher = NoopNodeDispatcher{}

// RecordingInvalidator captures DispatchInvalidate calls for assertions.
type RecordingInvalidator struct {
	Calls [][]string
}

func (r *RecordingInvalidator) DispatchInvalidate(lenses ...string) {
	r.Calls = append(r.Calls, lenses)
}

var _ dispatch.Invalidator = &RecordingInvalidator{}

// ErrorWriter is an http.ResponseWriter that always fails on Write,
// for testing JSON encoding error paths.
type ErrorWriter struct {
	header   http.Header
	Statuses []int
}

func (e *ErrorWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *ErrorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (e *ErrorWriter) WriteHeader(statusCode int) {
	e.Statuses = append(e.Statuses, statusCode)
}

func (e *ErrorWriter) HasStatus(code int) bool {
	for _, status := range e.Statuses {
		if status == code {
			return true
		}
	}
	return false
}

// AgentSpawner is a minimal enabled agent-manager double for tests that only
// need availability while a declared workflow owns execution.
type AgentSpawner struct {
	Enabled bool
}

// NewAgentSpawner returns an enabled AgentSpawner with a default success result.
func NewAgentSpawner() *AgentSpawner {
	return &AgentSpawner{
		Enabled: true,
	}
}

func (s *AgentSpawner) IsEnabled() bool {
	return s.Enabled
}

// RecordingScheduler counts ScheduleAll invocations for assertions.
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
