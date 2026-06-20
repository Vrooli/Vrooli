package execution

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/queue"
	"test-genie/internal/shared"

	"github.com/google/uuid"
)

func TestSuiteExecutionService_ExecuteWithoutLinkedRequest(t *testing.T) {
	ctx := context.Background()
	recorder := &fakeExecutionRecorder{}
	engine := &stubExecutionEngine{
		result: &orchestrator.SuiteExecutionResult{
			ScenarioName:        "demo",
			StartedAt:           time.Now().Add(-time.Minute),
			CompletedAt:         time.Now(),
			Success:             true,
			PresetUsed:          "quick",
			RequestedPreset:     "quick",
			RequestedPhases:     []string{"structure", "unit"},
			RequestedSkipPhases: []string{"performance"},
			PlannedPhases:       []string{"structure", "unit"},
			FailFast:            true,
		},
	}

	service := NewSuiteExecutionService(
		engine,
		recorder,
		&fakeSuiteRequestManager{},
	)

	output, err := service.Execute(ctx, SuiteExecutionInput{
		Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output == nil {
		t.Fatalf("expected execution result")
	}
	if output.ExecutionID == uuid.Nil {
		t.Fatalf("execution id was not assigned")
	}
	if output.RequestedPreset != "quick" || !output.FailFast {
		t.Fatalf("expected execution metadata to be preserved: %#v", output)
	}
	if len(recorder.records) != 1 {
		t.Fatalf("expected execution to be persisted, got %#v", recorder.records)
	}
}

func TestSuiteExecutionService_RejectsMismatchedSuiteRequest(t *testing.T) {
	ctx := context.Background()
	suiteID := uuid.New()
	mgr := &fakeSuiteRequestManager{
		suites: map[uuid.UUID]*queue.SuiteRequest{
			suiteID: {
				ID:           suiteID,
				ScenarioName: "ecosystem-manager",
			},
		},
	}

	service := NewSuiteExecutionService(
		&stubExecutionEngine{},
		&fakeExecutionRecorder{},
		mgr,
	)

	_, err := service.Execute(ctx, SuiteExecutionInput{
		Request:        orchestrator.SuiteExecutionRequest{ScenarioName: "test-genie"},
		SuiteRequestID: &suiteID,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var vErr shared.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(mgr.updates) != 0 {
		t.Fatalf("expected no status updates, got %v", mgr.updates)
	}
}

func TestSuiteExecutionService_MarksSuiteRequestFailedOnRunnerError(t *testing.T) {
	ctx := context.Background()
	suiteID := uuid.New()
	mgr := &fakeSuiteRequestManager{
		suites: map[uuid.UUID]*queue.SuiteRequest{
			suiteID: {
				ID:           suiteID,
				ScenarioName: "demo",
			},
		},
	}

	service := NewSuiteExecutionService(
		&stubExecutionEngine{err: errors.New("runner failed")},
		&fakeExecutionRecorder{},
		mgr,
	)

	_, err := service.Execute(ctx, SuiteExecutionInput{
		Request:        orchestrator.SuiteExecutionRequest{ScenarioName: "demo"},
		SuiteRequestID: &suiteID,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(mgr.updates) != 2 {
		t.Fatalf("expected two status updates, got %d", len(mgr.updates))
	}
	if mgr.updates[0].status != queue.StatusRunning {
		t.Fatalf("first status should be running, got %s", mgr.updates[0].status)
	}
	if mgr.updates[1].status != queue.StatusFailed {
		t.Fatalf("second status should be failed, got %s", mgr.updates[1].status)
	}
}

func TestSuiteExecutionService_PersistsTerminalOutcomeOnRunnerError(t *testing.T) {
	cases := []struct {
		name   string
		ctx    func() (context.Context, context.CancelFunc)
		engine *stubExecutionEngine
		want   TerminalOutcome
	}{
		{
			name:   "engine error",
			ctx:    func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			engine: &stubExecutionEngine{err: errors.New("runner failed")},
			want:   TerminalOutcomeErrored,
		},
		{
			name:   "nil result",
			ctx:    func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
			engine: &stubExecutionEngine{result: nil},
			want:   TerminalOutcomeErrored,
		},
		{
			name: "aborted",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			engine: &stubExecutionEngine{err: context.Canceled},
			want:   TerminalOutcomeAborted,
		},
		{
			name: "timeout",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				return ctx, cancel
			},
			engine: &stubExecutionEngine{err: context.DeadlineExceeded},
			want:   TerminalOutcomeTimeout,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.ctx()
			defer cancel()
			recorder := &fakeExecutionRecorder{}
			service := NewSuiteExecutionService(tc.engine, recorder, &fakeSuiteRequestManager{})

			_, err := service.Execute(ctx, SuiteExecutionInput{
				Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"},
			})
			if err == nil {
				t.Fatalf("expected error")
			}
			if len(recorder.records) != 1 {
				t.Fatalf("expected exactly one terminal-outcome row, got %d", len(recorder.records))
			}
			rec := recorder.records[0]
			if rec.TerminalOutcome != tc.want {
				t.Fatalf("terminal outcome = %q, want %q", rec.TerminalOutcome, tc.want)
			}
			if rec.Success {
				t.Fatalf("catastrophic record must have success=false")
			}
			if rec.ScenarioName != "demo" {
				t.Fatalf("scenario = %q, want demo", rec.ScenarioName)
			}
		})
	}
}

type stubExecutionEngine struct {
	result *orchestrator.SuiteExecutionResult
	err    error
}

func (s *stubExecutionEngine) Execute(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*orchestrator.SuiteExecutionResult, error) {
	return s.result, s.err
}

func (s *stubExecutionEngine) ExecuteWithEvents(ctx context.Context, req orchestrator.SuiteExecutionRequest, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	return s.result, s.err
}

type fakeExecutionRecorder struct {
	records []*SuiteExecutionRecord
	err     error
}

func (f *fakeExecutionRecorder) Create(ctx context.Context, record *SuiteExecutionRecord) error {
	if f.err != nil {
		return f.err
	}
	clone := *record
	if record.Phases != nil {
		clone.Phases = append([]orchestrator.PhaseExecutionResult(nil), record.Phases...)
	}
	f.records = append(f.records, &clone)
	return nil
}

type statusChange struct {
	id     uuid.UUID
	status string
}

type fakeSuiteRequestManager struct {
	suites  map[uuid.UUID]*queue.SuiteRequest
	updates []statusChange
}

func (f *fakeSuiteRequestManager) Get(ctx context.Context, id uuid.UUID) (*queue.SuiteRequest, error) {
	if f.suites == nil {
		return nil, sql.ErrNoRows
	}
	if req, ok := f.suites[id]; ok {
		copy := *req
		return &copy, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeSuiteRequestManager) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if f.suites == nil {
		return sql.ErrNoRows
	}
	if _, ok := f.suites[id]; !ok {
		return sql.ErrNoRows
	}
	f.updates = append(f.updates, statusChange{id: id, status: status})
	return nil
}
