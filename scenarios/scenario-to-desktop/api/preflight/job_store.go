package preflight

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryJobStore is the default in-memory implementation of JobStore.
type InMemoryJobStore struct {
	jobs       map[string]*Job
	mux        sync.Mutex
	expiration time.Duration
}

// InMemoryJobStoreOption configures an InMemoryJobStore.
type InMemoryJobStoreOption func(*InMemoryJobStore)

// WithJobExpiration sets the job expiration duration.
func WithJobExpiration(d time.Duration) InMemoryJobStoreOption {
	return func(s *InMemoryJobStore) {
		s.expiration = d
	}
}

// NewInMemoryJobStore creates a new in-memory job store.
func NewInMemoryJobStore(opts ...InMemoryJobStoreOption) *InMemoryJobStore {
	s := &InMemoryJobStore{
		jobs:       make(map[string]*Job),
		expiration: 15 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create creates a new async preflight job.
func (s *InMemoryJobStore) Create() *Job {
	job := &Job{
		ID:     uuid.NewString(),
		Status: "running",
		Steps: map[string]Step{
			"validation":  {ID: "validation", Name: "Load bundle + validate", State: "pending"},
			"runtime":     {ID: "runtime", Name: "Start runtime control API", State: "pending"},
			"secrets":     {ID: "secrets", Name: "Apply secrets", State: "pending"},
			"services":    {ID: "services", Name: "Start services", State: "pending"},
			"diagnostics": {ID: "diagnostics", Name: "Collect diagnostics", State: "pending"},
		},
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.mux.Lock()
	s.jobs[job.ID] = job
	s.mux.Unlock()
	return job
}

// Get retrieves a job by ID.
func (s *InMemoryJobStore) Get(id string) (*Job, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	job, ok := s.jobs[id]
	return job, ok
}

// Update updates a job using a modifier function.
func (s *InMemoryJobStore) Update(id string, fn func(job *Job)) {
	s.mux.Lock()
	defer s.mux.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return
	}
	fn(job)
	job.UpdatedAt = time.Now()
}

// SetStep updates the state of a specific step in a job.
func (s *InMemoryJobStore) SetStep(id, stepID, state, detail string) {
	s.Update(id, func(job *Job) {
		step, ok := job.Steps[stepID]
		if !ok {
			step = Step{ID: stepID, Name: stepID}
		}
		step.State = state
		step.Detail = detail
		job.Steps[stepID] = step
	})
}

// SetResult updates the result of a job using an updater function.
func (s *InMemoryJobStore) SetResult(id string, updater func(prev *Response) *Response) {
	s.Update(id, func(job *Job) {
		job.Result = updater(job.Result)
	})
}

// Finish marks a job as complete with final status.
func (s *InMemoryJobStore) Finish(id string, status, errMsg string) {
	s.Update(id, func(job *Job) {
		job.Status = status
		job.Err = errMsg
	})
}

// Cleanup removes completed jobs older than the expiration threshold.
func (s *InMemoryJobStore) Cleanup() {
	now := time.Now()

	s.mux.Lock()
	for id, job := range s.jobs {
		if job.Status == "running" {
			continue
		}
		if now.Sub(job.UpdatedAt) > s.expiration {
			delete(s.jobs, id)
		}
	}
	s.mux.Unlock()
}
