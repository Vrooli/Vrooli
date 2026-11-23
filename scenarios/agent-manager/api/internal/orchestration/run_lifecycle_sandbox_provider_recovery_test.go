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
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"connectrpc.com/connect"

	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

func TestContinueRun_RetainsExactRequestAcrossSandboxReadLossAndRetry(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	sandboxID := uuid.New()
	getCalls := 0
	provider := &mocks.FakeSandboxProvider{
		GetFunc: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
			getCalls++
			if getCalls == 1 {
				return nil, connect.NewError(connect.CodeUnavailable, errors.New("workspace provider temporarily unavailable"))
			}
			return &sandbox.Sandbox{ID: sandboxID, Status: sandbox.SandboxStatusActive, WorkDir: "/tmp/recovered-workspace"}, nil
		},
	}
	continued := make(chan runner.ContinueRequest, 1)
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{SupportsContinuation: true, SupportedModels: []string{"mock-model"}})
	mockRunner.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- req
		return &runner.ExecuteResult{Success: true, ExitCode: 0, SessionID: "retained-native-session"}, nil
	}
	registry := runner.NewRegistry()
	mustRegisterRunner(t, registry, mockRunner)
	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithSandbox(provider),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunStateRoot(t.TempDir()),
		newTestRolePolicyOption(t),
	)
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name: "provider recovery profile", ProfileKey: "provider-recovery-" + uuid.New().String()[:8],
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{Title: "Retain the operator's continuation", ScopePath: "src/", ProjectRoot: t.TempDir()})
	runID := uuid.New()
	run := newLifecycleRunFixture(runID, task.ID, profile.ID, domain.RunModeSandboxed, domain.RunStatusComplete, domain.RunPhaseCompleted, "retained-native-session")
	run.SandboxID = &sandboxID
	run.ResolvedConfig = &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "mock-model", RoleRef: "code.default", NetworkAccess: domain.NetworkAccessNone,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected, NetworkMode: domain.NetworkAccessNone, WritePolicy: &domain.WorkspaceWritePolicy{Paths: []string{"src"}}},
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatalf("create retained run: %v", err)
	}
	prior := []*domain.RunEvent{
		domain.NewMessageEvent(runID, "user", "Inspect the existing implementation and preserve it."),
		domain.NewMessageEvent(runID, "assistant", "I will retain the existing implementation."),
	}
	if err := eventStore.Append(ctx, runID, prior...); err != nil {
		t.Fatalf("persist prior task transcript: %v", err)
	}

	req := orchestration.ContinueRunRequest{RunID: runID, Message: "Continue the same task without replacing prior work.", IdempotencyKey: "sandbox-recovery-" + uuid.NewString()}
	if _, err := svc.ContinueRun(ctx, req); err == nil || connect.CodeOf(err) != connect.CodeUnavailable || !domain.IsPreEffectRefusal(err) {
		t.Fatalf("provider-loss continuation err=%v; want pre-effect Unavailable", err)
	}
	receipts := repos.Idempotency.(repository.ContinuationIdempotencyRepository)
	receipt, err := receipts.CheckContinuation(ctx, req.IdempotencyKey)
	if err != nil || receipt == nil || receipt.Status != domain.IdempotencyStatusFailed {
		t.Fatalf("receipt after read-only provider loss = %+v, %v; want reusable failed receipt", receipt, err)
	}
	current, err := repos.Runs.Get(ctx, runID)
	if err != nil || current.Status != domain.RunStatusComplete || current.SessionID != "retained-native-session" || current.SandboxID == nil || *current.SandboxID != sandboxID {
		t.Fatalf("run after unavailable workspace read = %+v, %v", current, err)
	}
	messages, err := eventStore.Get(ctx, runID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || len(messages) != len(prior) {
		t.Fatalf("messages after loss = %d, %v; want original %d only", len(messages), err, len(prior))
	}

	if _, err := svc.ContinueRun(ctx, req); err != nil {
		t.Fatalf("exact idempotent retry after provider recovery: %v", err)
	}
	select {
	case got := <-continued:
		if got.RunID != runID || got.SessionID != "retained-native-session" || got.SandboxID == nil || *got.SandboxID != sandboxID || got.Prompt != req.Message {
			t.Fatalf("recovered runner request lost retained identity/work: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("recovered continuation was not dispatched")
	}
	waitForStatusEvents(t, ctx, eventStore, runID, 2)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, err = repos.Runs.Get(ctx, runID)
		if err != nil {
			t.Fatalf("read recovered run: %v", err)
		}
		if current.Status == domain.RunStatusComplete && current.FinalizationStatus == domain.RunFinalizationStatusSucceeded {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != domain.RunStatusComplete || current.SessionID != "retained-native-session" || current.SandboxID == nil || *current.SandboxID != sandboxID {
		t.Fatalf("recovered run lost saved workspace/session: %+v", current)
	}
	receipt, err = receipts.CheckContinuation(ctx, req.IdempotencyKey)
	if err != nil || receipt == nil || receipt.Status != domain.IdempotencyStatusComplete {
		t.Fatalf("receipt after recovery = %+v, %v; want complete", receipt, err)
	}
	messages, err = eventStore.Get(ctx, runID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || len(messages) != len(prior)+1 {
		t.Fatalf("messages after exact retry = %d, %v; want original plus one continuation", len(messages), err)
	}
	if getCalls != 2 {
		t.Fatalf("workspace provider reads = %d, want one failed read and one exact retry", getCalls)
	}
}

func TestContinueRun_DeletedWorkspaceUsesPinnedMappingAcrossTaskChangeAndCreateLoss(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	oldRoot, newRoot := t.TempDir(), t.TempDir()
	oldScope, newScope := "scenarios/old-target", "scenarios/new-target"
	oldSandboxID, replacementID, newRunSandboxID := uuid.New(), uuid.New(), uuid.New()
	deletedSandboxes := map[uuid.UUID]bool{oldSandboxID: true}
	createCalls := 0
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		status := sandbox.SandboxStatusActive
		if deletedSandboxes[id] {
			status = sandbox.SandboxStatusDeleted
		}
		return &sandbox.Sandbox{ID: id, Status: status, WorkDir: "/tmp/sandbox/" + id.String() + "/merged"}, nil
	}
	provider.CreateFunc = func(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
		createCalls++
		if createCalls == 1 {
			return nil, &domain.SandboxError{
				Operation: "create", Cause: connect.NewError(connect.CodeUnavailable, errors.New("workspace create temporarily unavailable")),
				IsTransient: true, CanRetry: true,
			}
		}
		id := replacementID
		if createCalls > 2 {
			id = newRunSandboxID
		}
		return &sandbox.Sandbox{
			ID: id, ProjectRoot: req.ProjectRoot, ScopePath: req.ScopePath,
			Status: sandbox.SandboxStatusActive, WorkDir: "/tmp/sandbox/" + id.String() + "/merged",
		}, nil
	}
	continued := make(chan runner.ContinueRequest, 1)
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{SupportsContinuation: true, SupportedModels: []string{"mock-model"}})
	mockRunner.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- req
		return &runner.ExecuteResult{Success: true, ExitCode: 0, SessionID: "retained-config-session"}, nil
	}
	registry := runner.NewRegistry()
	mustRegisterRunner(t, registry, mockRunner)
	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(registry),
		orchestration.WithSandbox(provider), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t),
		fixtureOwnerIdentityOption(),
	)
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name: "workspace binding recovery", ProfileKey: "workspace-binding-" + uuid.New().String()[:8],
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title: "Keep the original project", ScopePath: oldScope, ProjectRoot: oldRoot,
	})
	runID := uuid.New()
	run := newLifecycleRunFixture(runID, task.ID, profile.ID, domain.RunModeSandboxed, domain.RunStatusComplete, domain.RunPhaseCompleted, "retained-config-session")
	run.SandboxID = &oldSandboxID
	run.ResolvedConfig = &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "mock-model", RoleRef: "code.default", NetworkAccess: domain.NetworkAccessNone,
		WorkspaceBinding: &domain.RunWorkspaceBinding{ProjectRoot: oldRoot, ScopePath: oldScope},
		SandboxConfig:    &domain.SandboxConfig{Mode: domain.SandboxModeProtected, NetworkMode: domain.NetworkAccessNone},
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, task, run, nil); err != nil {
		t.Fatalf("create retained run: %v", err)
	}
	prior := []*domain.RunEvent{domain.NewMessageEvent(runID, "user", "Preserve my existing project and work.")}
	if err := eventStore.Append(ctx, runID, prior...); err != nil {
		t.Fatalf("persist original transcript: %v", err)
	}

	updated := *task
	updated.ProjectRoot, updated.ScopePath = newRoot, newScope
	if _, err := svc.UpdateTask(ctx, &updated); err != nil {
		t.Fatalf("change task mapping: %v", err)
	}
	req := orchestration.ContinueRunRequest{
		RunID: runID, Message: "Continue the retained task in its saved workspace.",
		IdempotencyKey: "workspace-binding-" + uuid.NewString(),
	}
	if _, err := svc.ContinueRun(ctx, req); err != nil {
		t.Fatalf("continuation did not recover from transient provider loss: %v", err)
	}
	select {
	case got := <-continued:
		if got.RunID != runID || got.SessionID != "retained-config-session" || got.SandboxID == nil || *got.SandboxID != replacementID || got.Prompt != req.Message {
			t.Fatalf("continued request lost retained run/session/work: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("recovered continuation was not dispatched")
	}
	requests := provider.CreateRequests()
	if len(requests) != 2 {
		t.Fatalf("create attempts = %d, want transient loss plus one recovery", len(requests))
	}
	for i, got := range requests {
		if got.ProjectRoot != oldRoot || got.ScopePath != oldScope {
			t.Fatalf("create attempt %d retargeted to current task mapping: root=%q scope=%q", i+1, got.ProjectRoot, got.ScopePath)
		}
		if got.IdempotencyKey == "" || (i > 0 && got.IdempotencyKey != requests[0].IdempotencyKey) {
			t.Fatalf("replacement retries did not use one stable provider key: %+v", requests)
		}
	}
	if createCalls != 2 {
		t.Fatalf("provider create calls = %d, want one failed attempt and one recovered attempt", createCalls)
	}
	waitForStatusEvents(t, ctx, eventStore, runID, 2)
	current, err := repos.Runs.Get(ctx, runID)
	if err != nil || current.Status != domain.RunStatusComplete || current.SessionID != "retained-config-session" || current.SandboxID == nil || *current.SandboxID != replacementID {
		t.Fatalf("run after recovered continuation lost saved identity/workspace: %+v, %v", current, err)
	}
	messages, err := eventStore.Get(ctx, runID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || len(messages) != len(prior)+1 {
		t.Fatalf("transcript after recovered continuation has %d messages, %v; want original plus one", len(messages), err)
	}
	updatedRun, err := repos.Runs.Get(ctx, runID)
	if err != nil || updatedRun.ResolvedConfig.WorkspaceBinding.ProjectRoot != oldRoot || updatedRun.ResolvedConfig.WorkspaceBinding.ScopePath != oldScope {
		t.Fatalf("persisted run binding changed with mutable task: %+v, %v", updatedRun.ResolvedConfig, err)
	}

	newRun, err := svc.CreateRun(ctx, authenticatedCreateRunFixture(orchestration.CreateRunRequest{
		TaskID: task.ID, AgentProfileID: &profile.ID, Prompt: "Use the newly selected project mapping.",
	}))
	if err != nil {
		t.Fatalf("create new run after task mapping change: %v", err)
	}
	if newRun.ResolvedConfig == nil || newRun.ResolvedConfig.WorkspaceBinding == nil ||
		newRun.ResolvedConfig.WorkspaceBinding.ProjectRoot != newRoot || newRun.ResolvedConfig.WorkspaceBinding.ScopePath != newScope {
		t.Fatalf("new run did not pin the updated task mapping: %+v", newRun.ResolvedConfig)
	}
	requests = provider.CreateRequests()
	if len(requests) != 3 || requests[2].ProjectRoot != newRoot || requests[2].ScopePath != newScope {
		t.Fatalf("new run creation did not use the updated target: %+v", requests)
	}

	legacyTask := mustCreateTask(t, svc, ctx, &domain.Task{Title: "Legacy retained workspace", ProjectRoot: newRoot, ScopePath: newScope})
	legacySandboxID := uuid.New()
	deletedSandboxes[legacySandboxID] = true
	legacyRunID := uuid.New()
	legacyRun := newLifecycleRunFixture(legacyRunID, legacyTask.ID, profile.ID, domain.RunModeSandboxed, domain.RunStatusComplete, domain.RunPhaseCompleted, "legacy-session")
	legacyRun.SandboxID = &legacySandboxID
	legacyRun.ResolvedConfig = &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode, Model: "mock-model", RoleRef: "code.default",
		NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, legacyTask, legacyTask, legacyRun, nil); err != nil {
		t.Fatalf("create legacy retained run: %v", err)
	}
	legacyWork := domain.NewMessageEvent(legacyRunID, "user", "Preserve work whose original project binding is unknown.")
	if err := eventStore.Append(ctx, legacyRunID, legacyWork); err != nil {
		t.Fatalf("persist legacy transcript: %v", err)
	}
	legacyReq := orchestration.ContinueRunRequest{
		RunID: legacyRunID, Message: "Do not guess a project for this retained session.",
		IdempotencyKey: "workspace-binding-legacy-" + uuid.NewString(),
	}
	if _, err := svc.ContinueRun(ctx, legacyReq); err == nil || !domain.IsPreEffectRefusal(err) {
		t.Fatalf("legacy deleted workspace without pinned binding error=%v; want explicit pre-effect refusal", err)
	}
	if len(provider.CreateRequests()) != 3 {
		t.Fatalf("legacy workspace attempted creation despite unknown binding: %+v", provider.CreateRequests())
	}
	legacyCurrent, err := repos.Runs.Get(ctx, legacyRunID)
	legacyMessages, messagesErr := eventStore.Get(ctx, legacyRunID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || messagesErr != nil || legacyCurrent.Status != domain.RunStatusComplete || legacyCurrent.SessionID != "legacy-session" || len(legacyMessages) != 1 {
		t.Fatalf("unsupported legacy recovery changed saved work: run=%+v messages=%d errors=%v/%v", legacyCurrent, len(legacyMessages), err, messagesErr)
	}
}
