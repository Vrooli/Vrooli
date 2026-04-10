package deepsearch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type memoryJobStore struct {
	mu   sync.Mutex
	jobs map[string]DeepSearchJob
	seq  int
}

func newMemoryJobStore() *memoryJobStore {
	return &memoryJobStore{jobs: make(map[string]DeepSearchJob)}
}

func (m *memoryJobStore) CreateJob(_ context.Context, req DeepSearchRequest) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	id := "job-" + string(rune('A'+m.seq))
	m.jobs[id] = DeepSearchJob{JobID: id, Status: StatusPending, MaxResults: req.MaxResults}
	return id, nil
}

func (m *memoryJobStore) GetJob(_ context.Context, jobID string) (DeepSearchJob, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	return job, ok, nil
}

func (m *memoryJobStore) MarkRunning(_ context.Context, jobID, runID string, startedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = StatusRunning
	job.AgentRunID = runID
	job.StartedAt = &startedAt
	m.jobs[jobID] = job
	return nil
}

func (m *memoryJobStore) UpdateProgress(_ context.Context, jobID, progress string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Progress = progress
	m.jobs[jobID] = job
	return nil
}

func (m *memoryJobStore) CompleteJob(_ context.Context, jobID string, results []DeepSearchResult, completedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = StatusCompleted
	job.Results = results
	job.CompletedAt = &completedAt
	m.jobs[jobID] = job
	return nil
}

func (m *memoryJobStore) FailJob(_ context.Context, jobID, message string, completedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = StatusFailed
	job.Error = message
	job.CompletedAt = &completedAt
	m.jobs[jobID] = job
	return nil
}

type fakeAgent struct {
	runID  string
	status RunStatus
	events []AgentRunEvent
}

func (f *fakeAgent) EnsureProfile(_ context.Context) error { return nil }

func (f *fakeAgent) CreateRun(_ context.Context, _ AgentRunRequest) (string, error) {
	return f.runID, nil
}

func (f *fakeAgent) GetRun(_ context.Context, _ string) (*AgentRun, error) {
	return &AgentRun{ID: f.runID, Status: f.status}, nil
}

func (f *fakeAgent) GetRunEvents(_ context.Context, _ string, _ int64) ([]AgentRunEvent, error) {
	return f.events, nil
}

func TestDeepSearchCompletes(t *testing.T) {
	scenariosRoot := filepath.Join(t.TempDir(), "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}

	results := []DeepSearchResult{
		{
			Path:        "scenarios/alpha/README.md",
			Relevance:   0.92,
			Summary:     "Alpha readme",
			MatchReason: "Contains overview",
		},
	}
	payload, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal results: %v", err)
	}

	agent := &fakeAgent{
		runID:  "run-1",
		status: RunStatusComplete,
		events: []AgentRunEvent{
			{Sequence: 1, Type: EventMessage, Role: "assistant", Content: string(payload)},
		},
	}
	store := newMemoryJobStore()

	service, err := NewService(scenariosRoot, agent, store, nil, &JSONParser{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.pollInterval = 5 * time.Millisecond

	job, err := service.StartSearch(context.Background(), DeepSearchRequest{
		Query:      "alpha overview",
		Scope:      ScopeGlobal,
		FollowRefs: true,
	})
	if err != nil {
		t.Fatalf("start search: %v", err)
	}
	if job.Status != StatusRunning {
		t.Fatalf("expected running status, got %s", job.Status)
	}

	deadline := time.After(250 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for completion")
		default:
		}
		current, _, _ := store.GetJob(context.Background(), job.JobID)
		if current.Status == StatusCompleted {
			if len(current.Results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(current.Results))
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDeepSearchValidation(t *testing.T) {
	scenariosRoot := filepath.Join(t.TempDir(), "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}
	service, err := NewService(scenariosRoot, &fakeAgent{}, newMemoryJobStore(), nil, &JSONParser{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.StartSearch(context.Background(), DeepSearchRequest{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
