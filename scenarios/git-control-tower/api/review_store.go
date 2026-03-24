package main

import (
	"sync"
	"time"
)

// ReviewJobStore is a thread-safe in-memory store for review job tracking.
type ReviewJobStore struct {
	mu      sync.RWMutex
	jobs    map[string]*reviewJobEntry
	stopCh  chan struct{}
	stopped sync.Once
}

// reviewJobEntry pairs status with the scenario it belongs to.
type reviewJobEntry struct {
	status       *ReviewJobStatus
	scenarioName string
}

// NewReviewJobStore creates a new empty ReviewJobStore.
func NewReviewJobStore() *ReviewJobStore {
	return &ReviewJobStore{
		jobs: make(map[string]*reviewJobEntry),
	}
}

// Create initialises a new job with the given checks set to pending.
func (s *ReviewJobStore) Create(jobID string, checks []string, scenarioName string) *ReviewJobStatus {
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
	s.jobs[jobID] = &reviewJobEntry{status: job, scenarioName: scenarioName}
	s.mu.Unlock()

	return job
}

// ActiveJobForScenario returns the job ID of a running job for the given scenario, or "".
func (s *ReviewJobStore) ActiveJobForScenario(scenarioName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for id, entry := range s.jobs {
		if entry.scenarioName == scenarioName && entry.status.Status == "running" {
			return id
		}
	}
	return ""
}

// Get returns a copy of the job status, or false if not found.
func (s *ReviewJobStore) Get(jobID string) (*ReviewJobStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.jobs[jobID]
	if !ok {
		return nil, false
	}
	// Return a shallow copy so callers don't race on the map.
	cp := *entry.status
	checks := make(map[string]CheckStatus, len(entry.status.Checks))
	for k, v := range entry.status.Checks {
		checks[k] = v
	}
	cp.Checks = checks
	return &cp, true
}

// UpdateCheck sets the status of an individual check within a job.
func (s *ReviewJobStore) UpdateCheck(jobID, check string, status CheckStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Checks[check] = status
	}
}

// Complete marks the job as completed and attaches the summary.
func (s *ReviewJobStore) Complete(jobID string, summary *ReviewSummaryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Status = "completed"
		entry.status.Summary = summary
	}
}

// Fail marks the job as failed with an error message.
func (s *ReviewJobStore) Fail(jobID, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Status = "failed"
		entry.status.Error = errMsg
	}
}

// Cleanup removes jobs older than 1 hour.
func (s *ReviewJobStore) Cleanup() {
	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, entry := range s.jobs {
		started, err := time.Parse(time.RFC3339, entry.status.StartedAt)
		if err != nil || started.Before(cutoff) {
			delete(s.jobs, id)
		}
	}
}

// StartCleanup runs periodic cleanup in the background.
func (s *ReviewJobStore) StartCleanup(interval time.Duration) {
	s.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// StopCleanup stops the periodic cleanup goroutine.
func (s *ReviewJobStore) StopCleanup() {
	s.stopped.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
}
