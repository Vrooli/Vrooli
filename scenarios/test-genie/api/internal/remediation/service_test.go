package remediation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryRepo struct {
	mu   sync.Mutex
	jobs map[string]Job
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
	return job, nil
}
func (m *memoryRepo) ListByScenario(_ context.Context, scenario string, _ int) ([]Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Job
	for _, job := range m.jobs {
		if job.Scenario == scenario {
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

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func readyPlan() Plan {
	return BuildPlan(Evidence{SourceExecutionID: "exec", SourceRunID: "run", Scenario: "demo", CompletedAt: time.Now(), Phases: []Phase{{Name: "unit"}}, Findings: []Finding{{StableID: "afid:1", Phase: "unit"}}})
}
func TestServicePreventsCrossEntryActiveJobsAndRequiresVerification(t *testing.T) {
	svc := NewService(&memoryRepo{jobs: map[string]Job{}}, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
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
	job, err = svc.MarkRunning(context.Background(), job.ID, Attribution{RunID: "agent-run"})
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
	job, err = svc.CompleteVerification(context.Background(), job.ID, Verification{ExecutionID: "verification-execution", RunID: "verify-run"}, FindingDelta{Resolved: []string{"afid:1"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != JobStatusVerified || job.Verification.RunID != "verify-run" || job.Verification.ExecutionID != "verification-execution" {
		t.Fatalf("job = %+v", job)
	}
}

func TestServiceReservesExactlyOneVerificationRerun(t *testing.T) {
	svc := NewService(&memoryRepo{jobs: map[string]Job{}}, fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	job, err := svc.Create(context.Background(), readyPlan(), []string{"afid:1"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.MarkRunning(context.Background(), job.ID, Attribution{RunID: "agent-run"}); err != nil {
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
