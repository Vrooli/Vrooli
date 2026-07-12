package remediation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryRepo struct {
	mu       sync.Mutex
	jobs     map[string]Job
	attempts map[string][]Attempt
}

func (m *memoryRepo) Create(_ context.Context, job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, current := range m.jobs {
		if current.Scenario == job.Scenario && IsActiveStatus(current.Status) {
			return ErrActiveJob
		}
	}
	m.jobs[job.ID] = job
	return nil
}
func (m *memoryRepo) Get(_ context.Context, id string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	job.Attempts = append([]Attempt(nil), m.attempts[id]...)
	return job, nil
}
func (m *memoryRepo) ListByScenario(_ context.Context, scenario string, _ int) ([]Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Job
	for _, job := range m.jobs {
		if job.Scenario == scenario {
			job.Attempts = append([]Attempt(nil), m.attempts[job.ID]...)
			out = append(out, job)
		}
	}
	return out, nil
}
func (m *memoryRepo) ActiveForScenario(_ context.Context, scenario string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range m.jobs {
		if job.Scenario == scenario && IsActiveStatus(job.Status) {
			job.Attempts = append([]Attempt(nil), m.attempts[job.ID]...)
			return job, nil
		}
	}
	return Job{}, ErrNotFound
}
func (m *memoryRepo) Update(_ context.Context, job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[job.ID]; !ok {
		return ErrNotFound
	}
	m.jobs[job.ID] = job
	return nil
}
func (m *memoryRepo) UpdateIfStatus(_ context.Context, job Job, expected string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.jobs[job.ID]
	if !ok {
		return ErrNotFound
	}
	if current.Status != expected {
		return ErrInvalidState
	}
	m.jobs[job.ID] = job
	return nil
}
func (m *memoryRepo) AppendAttempt(_ context.Context, attempt Attempt, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[jobID] = append(m.attempts[jobID], attempt)
	return nil
}
func (m *memoryRepo) ListAttempts(_ context.Context, jobID string) ([]Attempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Attempt(nil), m.attempts[jobID]...), nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func readyPlan() Plan {
	return BuildPlan(Evidence{SourceExecutionID: "exec", SourceRunID: "run", Scenario: "demo", CompletedAt: time.Now(), Phases: []Phase{{Name: "unit"}}, Findings: []Finding{{StableID: "afid:1", Phase: "unit"}}})
}
func TestServicePreventsCrossEntryActiveJobsAndRequiresVerification(t *testing.T) {
	svc := NewService(&memoryRepo{jobs: map[string]Job{}, attempts: map[string][]Attempt{}}, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	job, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, "context")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, ""); !errors.Is(err, ErrActiveJob) {
		t.Fatalf("second job err = %v", err)
	}
	if _, err := svc.MarkAgentCompleted(context.Background(), job.ID, "output"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("agent completion before launch err = %v", err)
	}
	job, err = svc.PrepareLaunch(context.Background(), job.ID, "code.default")
	if err != nil {
		t.Fatal(err)
	}
	job, err = svc.MarkRunning(context.Background(), job.ID, Attribution{RunID: "agent-run", RoleRef: "code.default"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = svc.MarkAgentCompleted(context.Background(), job.ID, "output")
	if err != nil {
		t.Fatal(err)
	}
	job, err = svc.StartVerification(context.Background(), job.ID, Verification{RunID: "verify-run"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = svc.CompleteVerification(context.Background(), job.ID, Verification{ExecutionID: "verification-execution", RunID: "verify-run"}, FindingDelta{Resolved: []string{"afid:1"}}, RequirementDelta{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != JobStatusVerified || job.Verification.RunID != "verify-run" || job.Verification.ExecutionID != "verification-execution" {
		t.Fatalf("job = %+v", job)
	}
}

func TestServiceReservesExactlyOneVerificationRerun(t *testing.T) {
	svc := NewService(&memoryRepo{jobs: map[string]Job{}, attempts: map[string][]Attempt{}}, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	job, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.PrepareLaunch(context.Background(), job.ID, "code.default"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.MarkRunning(context.Background(), job.ID, Attribution{RunID: "agent-run", RoleRef: "code.default"}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.MarkAgentCompleted(context.Background(), job.ID, "output"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ReserveVerification(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ReserveVerification(context.Background(), job.ID); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("second reservation err=%v, want invalid state", err)
	}
	job, err = svc.SetVerificationRun(context.Background(), job.ID, Verification{RunID: "verify-run"})
	if err != nil || job.Verification.RunID != "verify-run" {
		t.Fatalf("set verification run job=%+v err=%v", job, err)
	}
	job, err = svc.ReleaseVerificationReservation(context.Background(), job.ID)
	if err != nil || job.Status != JobStatusAgentCompleted {
		t.Fatalf("release job=%+v err=%v", job, err)
	}
}

func TestServicePersistsLaunchIntentBeforeRemoteRecovery(t *testing.T) {
	repo := &memoryRepo{jobs: map[string]Job{}, attempts: map[string][]Attempt{}}
	svc := NewService(repo, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	job, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PrepareLaunch(context.Background(), job.ID, "code.default")
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != JobStatusLaunchPending || pending.Attribution.RoleRef != "code.default" {
		t.Fatalf("pending launch = %+v", pending)
	}
	if len(pending.Attempts) != 1 || pending.Attempts[0].State != "prepared" || pending.Attempts[0].IdempotencyKey == "" {
		t.Fatalf("durable launch intent = %+v", pending.Attempts)
	}
	// A retry after a process restart is a no-op locally; callers can safely
	// send the stable key to Agent Manager to reconcile the remote run.
	recovered, err := svc.PrepareLaunch(context.Background(), job.ID, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.Attempts) != 1 || recovered.Attribution.RoleRef != "code.default" {
		t.Fatalf("recovered launch = %+v", recovered)
	}
}

func TestServiceRetryUsesNewImmutableRemoteAttempt(t *testing.T) {
	repo := &memoryRepo{jobs: map[string]Job{}, attempts: map[string][]Attempt{}}
	svc := NewService(repo, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	job, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.PrepareLaunch(context.Background(), job.ID, "code.default"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Fail(context.Background(), job.ID, "Agent Manager unavailable"); err != nil {
		t.Fatal(err)
	}
	retried, err := svc.RetryLaunch(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != JobStatusLaunchPending || retried.LaunchAttempt != 2 {
		t.Fatalf("retry = %+v", retried)
	}
	if len(retried.Attempts) < 3 || retried.Attempts[len(retried.Attempts)-1].State != "retry_prepared" || retried.Attempts[0].IdempotencyKey == retried.Attempts[len(retried.Attempts)-1].IdempotencyKey {
		t.Fatalf("retry timeline = %+v", retried.Attempts)
	}
}
