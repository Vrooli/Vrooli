package main

import (
	"sync"
	"time"
)

// ReviewJobStore is a thread-safe in-memory store for review job tracking.
type ReviewJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*ReviewJobStatus
}

// NewReviewJobStore creates a new empty ReviewJobStore.
func NewReviewJobStore() *ReviewJobStore {
	return &ReviewJobStore{
		jobs: make(map[string]*ReviewJobStatus),
	}
}

// Create initialises a new job with the given checks set to pending.
func (s *ReviewJobStore) Create(jobID string, checks []string) *ReviewJobStatus {
	checkMap := make(map[string]CheckStatus, len(checks))
	for _, c := range checks {
		checkMap[c] = CheckPending
	}

	job := &ReviewJobStatus{
		JobID:     jobID,
		Status:    "running",
		Checks:    checkMap,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	return job
}

// Get returns a copy of the job status, or false if not found.
func (s *ReviewJobStore) Get(jobID string) (*ReviewJobStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, false
	}
	// Return a shallow copy so callers don't race on the map.
	cp := *job
	checks := make(map[string]CheckStatus, len(job.Checks))
	for k, v := range job.Checks {
		checks[k] = v
	}
	cp.Checks = checks
	return &cp, true
}

// UpdateCheck sets the status of an individual check within a job.
func (s *ReviewJobStore) UpdateCheck(jobID, check string, status CheckStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Checks[check] = status
	}
}

// Complete marks the job as completed and attaches the summary.
func (s *ReviewJobStore) Complete(jobID string, summary *ReviewSummaryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = "completed"
		job.Summary = summary
	}
}

// Fail marks the job as failed with an error message.
func (s *ReviewJobStore) Fail(jobID, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = "failed"
		job.Error = errMsg
	}
}

// Cleanup removes jobs older than 1 hour.
func (s *ReviewJobStore) Cleanup() {
	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, job := range s.jobs {
		started, err := time.Parse(time.RFC3339, job.StartedAt)
		if err != nil || started.Before(cutoff) {
			delete(s.jobs, id)
		}
	}
}
