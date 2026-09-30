package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"time"

	"git-control-tower/internal/dbschema"
	"git-control-tower/internal/reviewjobs"
)

// ReviewJobStore is a thread-safe in-memory store for review job tracking.
type ReviewJobStore struct {
	mu      sync.RWMutex
	jobs    map[string]*reviewJobEntry
	durable *reviewjobs.Store
	stopCh  chan struct{}
	stopped sync.Once
}

// reviewJobEntry pairs status with the scenario it belongs to.
type reviewJobEntry struct {
	status       *ReviewJobStatus
	idempotency  string
	executionIDs map[string]string
	scenarioName string
	detailCount  int
	thresholds   ReadinessThresholds
	durable      *reviewjobs.Store
}

// NewReviewJobStore creates a new empty ReviewJobStore.
func NewReviewJobStore() *ReviewJobStore {
	return &ReviewJobStore{
		jobs: make(map[string]*reviewJobEntry),
	}
}

// NewReviewJobStoreWithDB adds durable lifecycle persistence while retaining
// the compatibility surface used by the existing review handlers. The durable
// row is authoritative for restart/idempotency standing; the in-memory copy
// continues to serve the existing response model until that model migrates.
func NewReviewJobStoreWithDB(db dbschema.DB) (*ReviewJobStore, error) {
	durable, err := reviewjobs.New(db)
	if err != nil {
		return nil, err
	}
	store := NewReviewJobStore()
	store.durable = durable
	if err := store.hydrate(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ReviewJobStore) hydrate(ctx context.Context) error {
	if s.durable == nil {
		return nil
	}
	jobs, err := s.durable.List(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, durable := range jobs {
		checks := make(map[string]CheckStatus, len(durable.Checks))
		for _, check := range durable.Checks {
			status := CheckStatus(check.Verdict)
			if status == "" || status == CheckPending && check.Availability == "requested" {
				status = CheckPending
			}
			checks[check.Name] = status
		}
		status := "running"
		switch durable.State {
		case reviewjobs.Partial:
			status = "partial"
		case reviewjobs.Succeeded:
			status = "completed"
		case reviewjobs.Failed:
			status = "failed"
		case reviewjobs.Cancelled:
			status = "cancelled"
		case reviewjobs.Interrupted:
			status = "interrupted"
		}
		executionIDs := make(map[string]string, len(durable.Checks))
		for _, check := range durable.Checks {
			if check.ExecutionID != "" {
				executionIDs[check.Name] = check.ExecutionID
			}
		}
		job := &ReviewJobStatus{JobID: durable.ID, Status: status, Checks: checks, CheckExecutionIDs: executionIDs, StartedAt: durable.CreatedAt.UTC().Format(time.RFC3339), Error: durable.Error}
		if durable.ResultJSON != "" {
			var summary ReviewSummaryResponse
			if json.Unmarshal([]byte(durable.ResultJSON), &summary) == nil {
				job.Summary = &summary
			}
		}
		thresholds := DefaultReadinessThresholds()
		if durable.ThresholdsJSON != "" {
			_ = json.Unmarshal([]byte(durable.ThresholdsJSON), &thresholds)
		}
		s.jobs[durable.ID] = &reviewJobEntry{status: job, idempotency: durable.IdempotencyKey, executionIDs: executionIDs, scenarioName: durable.ScenarioName, detailCount: durable.DetailCount, thresholds: thresholds}
	}
	return nil
}

// Create initialises a new job with the given checks set to pending.
func (s *ReviewJobStore) Create(jobID string, checks []string, scenarioName string, detailCount int, thresholds ReadinessThresholds) *ReviewJobStatus {
	return s.CreateWithIdempotency(jobID, jobID, checks, scenarioName, detailCount, thresholds)
}

// CreateWithIdempotency records a request identity alongside the compatibility
// status. The durable store uses the same key to make retries return the
// original job identity instead of starting a second review.
func (s *ReviewJobStore) CreateWithIdempotency(jobID, idempotencyKey string, checks []string, scenarioName string, detailCount int, thresholds ReadinessThresholds) *ReviewJobStatus {
	checkMap := make(map[string]CheckStatus, len(checks))
	for _, c := range checks {
		checkMap[c] = CheckPending
	}

	job := &ReviewJobStatus{
		JobID:             jobID,
		Status:            "running",
		Checks:            checkMap,
		CheckExecutionIDs: make(map[string]string),
		StartedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.jobs[jobID] = &reviewJobEntry{status: job, idempotency: idempotencyKey, executionIDs: job.CheckExecutionIDs, scenarioName: scenarioName, detailCount: detailCount, thresholds: thresholds}
	s.mu.Unlock()
	if s.durable != nil {
		checksJSON, _ := json.Marshal(checks)
		h := sha256.Sum256(append(checksJSON, []byte(scenarioName)...))
		thresholdsJSON, _ := json.Marshal(thresholds)
		_, _, err := s.durable.Create(context.Background(), reviewjobs.Job{
			ID: jobID, IdempotencyKey: idempotencyKey, InputDigest: "sha256:" + hex.EncodeToString(h[:]),
			PolicyVersion: "review-v2", State: reviewjobs.Running, ScenarioName: scenarioName,
			DetailCount: detailCount, ThresholdsJSON: string(thresholdsJSON), Checks: reviewChecks(checks),
		})
		if err != nil {
			log.Printf("review durable create failed: %v", err)
		}
	}

	return job
}

// FindByIdempotency returns the durable job identity for a retry, if present.
func (s *ReviewJobStore) FindByIdempotency(key string) (string, bool) {
	s.mu.RLock()
	for id, entry := range s.jobs {
		if entry.idempotency == key {
			s.mu.RUnlock()
			return id, true
		}
	}
	s.mu.RUnlock()
	if s.durable == nil {
		return "", false
	}
	job, err := s.durable.GetByIdempotency(context.Background(), key)
	if err != nil {
		return "", false
	}
	return job.ID, true
}

// DetailCount returns the detail count stored for the given job.
func (s *ReviewJobStore) DetailCount(jobID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.jobs[jobID]; ok {
		return entry.detailCount
	}
	return 0
}

// Thresholds returns the readiness thresholds stored for the given job.
func (s *ReviewJobStore) Thresholds(jobID string) ReadinessThresholds {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.jobs[jobID]; ok {
		return entry.thresholds
	}
	return DefaultReadinessThresholds()
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
	cp.CheckExecutionIDs = make(map[string]string, len(entry.executionIDs))
	for k, v := range entry.executionIDs {
		cp.CheckExecutionIDs[k] = v
	}
	return &cp, true
}

// UpdateCheck sets the status of an individual check within a job.
func (s *ReviewJobStore) UpdateCheck(jobID, check string, status CheckStatus) {
	s.UpdateCheckWithExecution(jobID, check, status, "")
}

func (s *ReviewJobStore) UpdateCheckWithExecution(jobID, check string, status CheckStatus, executionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Checks[check] = status
		if executionID != "" {
			entry.executionIDs[check] = executionID
			entry.status.CheckExecutionIDs[check] = executionID
		}
		if s.durable != nil {
			if err := s.durable.UpdateCheck(context.Background(), jobID, reviewjobs.Check{Name: check, ExecutionID: executionID, Availability: string(status), Verdict: string(status)}); err != nil {
				log.Printf("review durable check update failed: %v", err)
			}
		}
	}
}

func (s *ReviewJobStore) CheckExecutionID(jobID, check string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if entry, ok := s.jobs[jobID]; ok {
		return entry.executionIDs[check]
	}
	return ""
}

// Complete marks the job as completed and attaches the summary.
func (s *ReviewJobStore) Complete(jobID string, summary *ReviewSummaryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Status = "completed"
		entry.status.Summary = summary
		if summary != nil {
			summary.CheckStatuses = make(map[string]CheckStatus, len(entry.status.Checks))
			for name, status := range entry.status.Checks {
				summary.CheckStatuses[name] = status
			}
		}
		if s.durable != nil {
			resultJSON, _ := json.Marshal(summary)
			if err := s.durable.SetResult(context.Background(), jobID, string(resultJSON)); err != nil {
				log.Printf("review durable result failed: %v", err)
			}
			if err := s.durable.Transition(context.Background(), jobID, []reviewjobs.State{reviewjobs.Running, reviewjobs.Partial}, reviewjobs.Succeeded, jobID, ""); err != nil {
				log.Printf("review durable completion failed: %v", err)
			}
		}
	}
}

// Fail marks the job as failed with an error message.
func (s *ReviewJobStore) Fail(jobID, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.jobs[jobID]; ok {
		entry.status.Status = "failed"
		entry.status.Error = errMsg
		if s.durable != nil {
			if err := s.durable.Transition(context.Background(), jobID, []reviewjobs.State{reviewjobs.Running, reviewjobs.Partial}, reviewjobs.Failed, "", errMsg); err != nil {
				log.Printf("review durable failure failed: %v", err)
			}
		}
	}
}

func reviewChecks(checks []string) []reviewjobs.Check {
	out := make([]reviewjobs.Check, 0, len(checks))
	for _, check := range checks {
		out = append(out, reviewjobs.Check{Name: check, Availability: "requested", Verdict: string(CheckPending)})
	}
	return out
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
