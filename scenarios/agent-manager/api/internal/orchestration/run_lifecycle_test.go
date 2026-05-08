package orchestration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"

	"github.com/google/uuid"
)

func TestStopRun_WithTerminatorEmitsStatusEventAndBroadcast(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mustRegisterRunner(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}
	terminator := orchestration.NewTerminator(repos.Runs, registry, orchestration.TerminatorConfig{
		GracePeriod:      time.Millisecond,
		MaxRetries:       1,
		BaseBackoff:      time.Millisecond,
		MaxBackoff:       time.Millisecond,
		VerifyTimeout:    time.Millisecond,
		KillProcessGroup: false,
	})
	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithBroadcaster(broadcaster),
		orchestration.WithTerminator(terminator),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:          "stop-profile",
		ProfileKey:    "stop-" + uuid.New().String()[:8],
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:     "stop task",
		ScopePath: "src/",
	})

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		Phase:          domain.RunPhaseExecuting,
		SessionID:      "sess-stop",
		StartedAt:      &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := svc.StopRun(ctx, runID); err != nil {
		t.Fatalf("StopRun: %v", err)
	}

	updated, err := repos.Runs.Get(ctx, runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status != domain.RunStatusCancelled {
		t.Fatalf("expected cancelled run, got %s", updated.Status)
	}
	if updated.Actions != nil {
		t.Fatal("repository run should not have computed actions persisted")
	}

	events, err := eventStore.Get(ctx, runID, event.GetOptions{
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeStatus},
	})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one status event, got %d", len(events))
	}
	status, ok := events[0].Data.(*domain.StatusEventData)
	if !ok {
		t.Fatalf("expected status event data, got %T", events[0].Data)
	}
	if status.OldStatus != string(domain.RunStatusRunning) || status.NewStatus != string(domain.RunStatusCancelled) {
		t.Fatalf("unexpected status transition %s -> %s", status.OldStatus, status.NewStatus)
	}

	broadcasts := broadcaster.getStatusBroadcasts()
	if len(broadcasts) != 1 {
		t.Fatalf("expected one run status broadcast, got %d", len(broadcasts))
	}
	if broadcasts[0].Status != domain.RunStatusCancelled {
		t.Fatalf("expected cancelled broadcast, got %s", broadcasts[0].Status)
	}
	if broadcasts[0].Actions == nil || broadcasts[0].Actions.CanStop {
		t.Fatalf("expected broadcast with updated actions and CanStop=false")
	}
}

func TestContinueRun_EmitsRunningAndTerminalStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	continueDone := make(chan struct{})
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		defer close(continueDone)
		return &runner.ExecuteResult{
			Success:   true,
			ExitCode:  0,
			SessionID: "sess-after-continue",
		}, nil
	}

	registry := runner.NewRegistry()
	mustRegisterRunner(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}
	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithBroadcaster(broadcaster),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:          "continue-profile",
		ProfileKey:    "continue-" + uuid.New().String()[:8],
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:     "continue task",
		ScopePath: "src/",
	})

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-before-continue",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	returned, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:   runID,
		Message: "please continue",
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}
	if returned.Status != domain.RunStatusRunning {
		t.Fatalf("expected immediate returned run to be running, got %s", returned.Status)
	}
	if returned.Actions == nil || !returned.Actions.CanStop {
		t.Fatalf("expected returned run actions to allow stop while continuation is running")
	}

	select {
	case <-continueDone:
	case <-time.After(10 * time.Second):
		t.Fatal("continuation runner was never called")
	}

	events := waitForStatusEvents(t, ctx, eventStore, runID, 2)

	first := events[0].Data.(*domain.StatusEventData)
	second := events[1].Data.(*domain.StatusEventData)
	if first.OldStatus != string(domain.RunStatusComplete) || first.NewStatus != string(domain.RunStatusRunning) {
		t.Fatalf("unexpected first transition %s -> %s", first.OldStatus, first.NewStatus)
	}
	if second.OldStatus != string(domain.RunStatusRunning) || second.NewStatus != string(domain.RunStatusComplete) {
		t.Fatalf("unexpected second transition %s -> %s", second.OldStatus, second.NewStatus)
	}

	updated, err := repos.Runs.Get(ctx, runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.SessionID != "sess-after-continue" {
		t.Fatalf("expected updated session ID, got %q", updated.SessionID)
	}

	broadcasts := broadcaster.getStatusBroadcasts()
	if len(broadcasts) != 2 {
		t.Fatalf("expected two run status broadcasts, got %d", len(broadcasts))
	}
	if broadcasts[0].Status != domain.RunStatusRunning || broadcasts[1].Status != domain.RunStatusComplete {
		t.Fatalf("unexpected broadcast statuses %s, %s", broadcasts[0].Status, broadcasts[1].Status)
	}
	if broadcasts[1].Actions == nil || !broadcasts[1].Actions.CanContinue {
		t.Fatalf("expected terminal broadcast actions to allow continuation")
	}
}

func TestContinueRun_ProtectedSandboxCarriesLauncherInputsAndLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	continueDone := make(chan error, 1)
	var captured runner.ContinueRequest
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		defer close(continueDone)
		captured = req
		if req.EventSink == nil {
			continueDone <- errors.New("expected continuation event sink")
			return nil, errors.New("expected continuation event sink")
		}
		if err := req.EventSink.Emit(domain.NewMessageEvent(req.RunID, "assistant", "continued from protected sandbox")); err != nil {
			continueDone <- err
			return nil, err
		}
		continueDone <- nil
		return &runner.ExecuteResult{
			Success:   true,
			ExitCode:  0,
			SessionID: "sess-protected-after-continue",
		}, nil
	}

	registry := runner.NewRegistry()
	mustRegisterRunner(t, registry, mockRunner)

	sandboxID := uuid.New()
	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithSandbox(&mocks.FakeSandboxProvider{
			GetFunc: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return &sandbox.Sandbox{
					ID:        sandboxID,
					Status:    sandbox.SandboxStatusCheckpointed,
					WorkDir:   "",
					CreatedAt: time.Now(),
				}, nil
			},
			ResumeFunc: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return &sandbox.Sandbox{
					ID:        sandboxID,
					Status:    sandbox.SandboxStatusActive,
					WorkDir:   "/tmp/sandbox/" + sandboxID.String() + "/merged",
					CreatedAt: time.Now(),
				}, nil
			},
		}),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:          "protected-continue-profile",
		ProfileKey:    "protected-continue-" + uuid.New().String()[:8],
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "protected continue task",
		Description: "continuation should preserve protected launcher inputs",
		ScopePath:   "src/",
	})

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeSandboxed,
		SandboxID:      &sandboxID,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-protected-before-continue",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:    domain.RunnerTypeClaudeCode,
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if _, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:   runID,
		Message: "please continue in protected mode",
	}); err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	select {
	case err := <-continueDone:
		if err != nil {
			t.Fatalf("continuation runner: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("continuation runner was never called")
	}

	if captured.ResolvedConfig == nil {
		t.Fatal("expected continuation request to carry resolved config")
	}
	if captured.ResolvedConfig.SandboxConfig == nil || captured.ResolvedConfig.SandboxConfig.Mode != domain.SandboxModeProtected {
		t.Fatalf("expected protected sandbox config, got %#v", captured.ResolvedConfig.SandboxConfig)
	}
	if captured.SandboxID == nil || *captured.SandboxID != sandboxID {
		t.Fatalf("expected sandbox ID %s, got %v", sandboxID, captured.SandboxID)
	}

	statusEvents := waitForStatusEvents(t, ctx, eventStore, runID, 2)
	first := statusEvents[0].Data.(*domain.StatusEventData)
	second := statusEvents[1].Data.(*domain.StatusEventData)
	if first.NewStatus != string(domain.RunStatusRunning) || second.NewStatus != string(domain.RunStatusComplete) {
		t.Fatalf("unexpected continuation status events %s then %s", first.NewStatus, second.NewStatus)
	}

	messageEvents, err := eventStore.Get(ctx, runID, event.GetOptions{
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeMessage},
	})
	if err != nil {
		t.Fatalf("get message events: %v", err)
	}
	if len(messageEvents) != 2 {
		t.Fatalf("expected user and assistant message events, got %d", len(messageEvents))
	}
}

func TestContinueRun_ResumeFailureDoesNotMarkRunRunning(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsContinuation: true,
		SupportedModels:      []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		t.Fatal("runner continuation must not start when sandbox resume fails")
		return nil, nil
	}
	registry := runner.NewRegistry()
	mustRegisterRunner(t, registry, mockRunner)

	sandboxID := uuid.New()
	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithSandbox(&mocks.FakeSandboxProvider{
			GetFunc: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return &sandbox.Sandbox{
					ID:     sandboxID,
					Status: sandbox.SandboxStatusCheckpointed,
				}, nil
			},
			ResumeFunc: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return nil, errors.New("resume failed")
			},
		}),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:          "resume-fail-profile",
		ProfileKey:    "resume-fail-" + uuid.New().String()[:8],
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:     "resume fail task",
		ScopePath: "src/",
	})

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeSandboxed,
		SandboxID:      &sandboxID,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-before-continue",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:    domain.RunnerTypeClaudeCode,
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if _, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:   runID,
		Message: "please continue",
	}); err == nil {
		t.Fatal("ContinueRun error = nil, want resume failure")
	}

	updated, err := repos.Runs.Get(ctx, runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status != domain.RunStatusComplete {
		t.Fatalf("run status = %s, want complete", updated.Status)
	}
	events, err := eventStore.Get(ctx, runID, event.GetOptions{
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeStatus},
	})
	if err != nil {
		t.Fatalf("get status events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no status events, got %d", len(events))
	}
}

func waitForStatusEvents(t *testing.T, ctx context.Context, store event.Store, runID uuid.UUID, count int) []*domain.RunEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var events []*domain.RunEvent
	var err error
	for time.Now().Before(deadline) {
		events, err = store.Get(ctx, runID, event.GetOptions{
			AfterSequence: -1,
			EventTypes:    []domain.RunEventType{domain.EventTypeStatus},
		})
		if err != nil {
			t.Fatalf("get events: %v", err)
		}
		if len(events) >= count {
			return events
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected %d status events, got %d", count, len(events))
	return nil
}
