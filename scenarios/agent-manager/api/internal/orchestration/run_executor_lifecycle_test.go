package orchestration_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/adapters/sandbox"
	cfgpkg "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

func TestRunExecutor_BroadcastsStatusOnFailure(t *testing.T) {
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution failed")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := mocks.NewFakeBroadcaster()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	broadcasts := broadcaster.StatusBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one status broadcast on failure")
	}

	last := broadcasts[len(broadcasts)-1]
	if last.Status != domain.RunStatusFailed {
		t.Errorf("expected final broadcast status failed, got %s", last.Status)
	}
}

func TestRunExecutor_BroadcastsStatusOnCancellation(t *testing.T) {
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	var wg sync.WaitGroup
	wg.Add(1)
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		wg.Done()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := mocks.NewFakeBroadcaster()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 30 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithBroadcaster(broadcaster)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		executor.Execute(ctx)
		close(done)
	}()

	wg.Wait()
	cancel()
	<-done

	broadcasts := broadcaster.StatusBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one status broadcast on cancellation")
	}

	last := broadcasts[len(broadcasts)-1]
	if last.Status != domain.RunStatusCancelled {
		t.Errorf("expected final broadcast status cancelled, got %s", last.Status)
	}
}

func TestRunExecutor_NoBroadcaster_NoPanic(t *testing.T) {
	// Verify that nil broadcaster doesn't cause a panic
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	// No WithBroadcaster call — broadcaster stays nil
	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithRunStateRoot(t.TempDir())

	executor.Execute(context.Background())
	// If we get here without panic, the nil guard works
}

// =============================================================================
// IN-PLACE RUN APPROVAL BYPASS TESTS
// =============================================================================
// In-place runs apply changes directly to the working tree (no sandbox).
// Since there is no sandbox to diff against or merge from, the approval
// workflow is skipped entirely and the run auto-completes.

func TestRunExecutor_InPlace_SkipsApproval(t *testing.T) {
	// An in-place run should auto-complete because there is no sandbox to
	// diff against — the approval / apply workflow doesn't apply.
	f := newInPlaceFixtures(t)
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithRunStateRoot(t.TempDir())

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusComplete {
		t.Errorf("expected in-place run status 'complete', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStateNone {
		t.Errorf("expected approval state 'none' for in-place run, got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_Sandboxed_ManualReviewDefersApply(t *testing.T) {
	// Per the auditability contract, the only way a sandboxed run lands
	// in NeedsReview/Pending after success is ManualReview=true. The
	// contract is "auto-apply by default unless operator opts into manual
	// review".
	f := newTestFixtures(t) // sandboxed mode
	manualReviewCfg := domain.DefaultSandboxConfig()
	manualReviewCfg.ManualReview = true
	f.run.SandboxConfig = manualReviewCfg
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: manualReviewCfg,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()
	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, sandboxProvider, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithRunStateRoot(t.TempDir())

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected sandboxed run status 'needs_review', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected approval state 'pending' for sandboxed run, got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_Sandboxed_DefaultAutoApplies_Completes(t *testing.T) {
	// Sandboxed runs with the contract defaults (AutoApply=true,
	// ManualReview=false) should auto-apply at run end and land in Complete
	// with ApprovalState=Approved.
	f := newTestFixtures(t) // sandboxed mode
	f.run.SandboxConfig = domain.DefaultSandboxConfig()
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: f.run.SandboxConfig,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		// Duration must clear validateRunOutcome's silent-launch
		// heuristic (a real sandbox run takes seconds; a sub-2s run
		// with zero message events is treated as a bwrap launch
		// failure). Without this, validateRunOutcome demotes the run
		// to FAILED before applyAtRunEnd ever sees it.
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Duration: 3 * time.Second}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()
	// Mock returns Applied=1 so applyAtRunEnd sees a real auditable
	// change and follows the success → Complete branch. (The default
	// mock returns Applied=0; with the 2026-04-28 fix that path no
	// longer promotes a failure to Complete, so the test must opt in
	// to a non-empty apply explicitly.)
	sandboxProvider.ApplyAtRunEndFunc = func(ctx context.Context, req sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
		return &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1, AppliedAt: time.Now()}, nil
	}
	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, sandboxProvider, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithRunStateRoot(t.TempDir())

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusComplete {
		t.Errorf("expected status 'complete', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected approval state 'approved' (auto-applied), got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_InPlace_EmitsSkipApplyEvent(t *testing.T) {
	// Verify that in-place runs emit a system event explaining the
	// approval skip, so operators can trace the decision.
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers)

	executor.Execute(context.Background())

	// Check that a system event was emitted explaining the approval skip
	events, err := eventStore.Get(context.Background(), f.run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}

	found := false
	for _, evt := range events {
		if evt.EventType == domain.EventTypeLog {
			if logData, ok := evt.Data.(*domain.LogEventData); ok {
				if logData.Level == "info" &&
					strings.Contains(logData.Message, "in-place run completed") &&
					strings.Contains(logData.Message, "skipping apply") {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected system event explaining in-place approval skip, but none found")
	}
}

func TestRunExecutor_BroadcastsPostRunnerEvents(t *testing.T) {
	// Verify that system events emitted after the runner finishes
	// (phase changes, completion messages) are broadcast via WebSocket,
	// not just stored to the database. This ensures real-time UI updates
	// for post-execution events.
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := mocks.NewFakeBroadcaster()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	eventBroadcasts := broadcaster.EventBroadcasts()
	if len(eventBroadcasts) == 0 {
		t.Fatal("expected post-runner events to be broadcast via WebSocket")
	}

	// Verify that at least one log event (system event) was broadcast.
	// These are the events emitted by emitSystemEvent (phase changes, etc.)
	foundLogBroadcast := false
	for _, evt := range eventBroadcasts {
		if evt.EventType == domain.EventTypeLog {
			foundLogBroadcast = true
			break
		}
	}
	if !foundLogBroadcast {
		t.Error("expected at least one log event to be broadcast, but none found")
	}
}

func TestRunExecutor_BroadcastsErrorEventsOnFailure(t *testing.T) {
	// Verify that error events emitted when a runner fails are broadcast
	// via WebSocket so the UI can show failure details in real-time.
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, fmt.Errorf("agent crashed unexpectedly")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := mocks.NewFakeBroadcaster()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	eventBroadcasts := broadcaster.EventBroadcasts()

	// Verify that an error event was broadcast
	foundErrorBroadcast := false
	for _, evt := range eventBroadcasts {
		if evt.EventType == domain.EventTypeError {
			foundErrorBroadcast = true
			break
		}
	}
	if !foundErrorBroadcast {
		t.Error("expected error event to be broadcast on runner failure, but none found")
	}
}

// TestRunExecutor_ProcessReplayTrackingAppliesAttribution proves the complete
// sandboxed persistence path using the fake corpus process, not a mock runner.
func TestRunExecutor_ProcessReplayTrackingAppliesAttribution(t *testing.T) {
	f := newTestFixtures(t)
	autoApply := true
	f.run.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeTracking, AutoApply: &autoApply}
	f.run.ResolvedConfig = &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, SandboxConfig: f.run.SandboxConfig}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	fakeAgent := testutil.BuildFakeAgent(t)
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTestWithBinary(fakeAgent), runner.NewHostLauncher(), nil)); err != nil {
		t.Fatal(err)
	}
	sandboxProvider := newMockSandboxProvider()
	workDir := t.TempDir()
	sandboxProvider.GetWorkspacePathFn = func(context.Context, uuid.UUID) (string, error) { return workDir, nil }
	sandboxProvider.ApplyAtRunEndFunc = func(context.Context, sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
		return &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2, TotalSizeBytes: 4096, DiffPath: "/api/v1/sandboxes/replay/diff", CommitHash: "replay-commit", AppliedAt: time.Now()}, nil
	}
	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 10 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond
	corpus, err := filepath.Abs(filepath.Join("..", "adapters", "runner", "codecs", "testdata", "corpus", "codex-stdout.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	executor := orchestration.NewRunExecutor(repos.Runs, registry, sandboxProvider, eventStore, f.run, f.task, f.profile, "replay", "").
		WithLevers(levers).WithRunStateRoot(t.TempDir()).WithCustomEnvironment(map[string]string{"FAKE_AGENT_CORPUS": corpus})
	executor.Execute(context.Background())
	got, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, error=%q", got.Status, got.ErrorMsg)
	}
	if got.ChangedFiles != 2 || got.TotalSizeBytes != 4096 || got.DiffPath != "/api/v1/sandboxes/replay/diff" || got.CommitHash != "replay-commit" {
		t.Fatalf("attribution = files=%d bytes=%d diff=%q commit=%q", got.ChangedFiles, got.TotalSizeBytes, got.DiffPath, got.CommitHash)
	}
}
