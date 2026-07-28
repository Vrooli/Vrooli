package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// newParkableRun inserts a RunStatusRunning run (with custom env + session +
// resolved config) directly, bypassing CreateRun's async execution. It is the
// starting point for park/wake round-trip tests.
func newParkableRun(t *testing.T, ctx context.Context, svc *orchestration.Orchestrator, repos *database.Repositories) *domain.Run {
	t.Helper()
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "park-profile",
		ProfileKey: "park-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "park-task",
		Description: "task for park/wake round-trip",
		ScopePath:   "src/",
	})

	old := time.Now().Add(-30 * time.Minute)
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		Phase:          domain.RunPhaseExecuting,
		SessionID:      "sess-" + uuid.New().String()[:8],
		StartedAt:      &old,
		LastHeartbeat:  &old,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:  domain.ApprovalStateNone,
		CustomEnv: map[string]string{
			"VROOLI_SHADOW_SCENARIOS": "agent-manager",
		},
		CreatedAt: old,
		UpdatedAt: old,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return run
}

// TestParkRun_AndWake_PreservesIdentityEnvAndResetsHeartbeat is the Phase 2
// round-trip anchor: running→parked records an await-handle (sandbox/identity
// untouched, never revoked); parked→running (wake) clears the handle, injects
// the awaited result as the next turn, re-injects custom env + a fresh verifiable
// identity token (Phase 0 assembler), and resets the heartbeat.
func TestParkRun_AndWake_PreservesIdentityEnvAndResetsHeartbeat(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner, captured, captureMu, done := newContinuationRunner(t)
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	identitySecret := []byte("phase2-park-identity-secret-0123456789")
	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:    5 * time.Minute,
			MaxConcurrentRuns: 10,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithIdentitySecret(identitySecret),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	run := newParkableRun(t, ctx, svc, repos)

	// Park.
	parked, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID:    run.ID,
		Producer: "test-genie",
		Key:      "run-abc123",
	})
	if err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	if parked.Status != domain.RunStatusParked {
		t.Fatalf("after park, status = %s, want parked", parked.Status)
	}
	if parked.AwaitHandle == nil {
		t.Fatal("after park, await handle is nil")
	}
	if parked.AwaitHandle.Producer != "test-genie" || parked.AwaitHandle.Key != "run-abc123" {
		t.Errorf("await handle = %+v, want producer=test-genie key=run-abc123", parked.AwaitHandle)
	}
	if parked.AwaitHandle.Deadline == nil {
		t.Error("park must apply a default deadline when none supplied")
	}

	// Handle must survive a reload (persisted).
	reloaded, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after park: %v", err)
	}
	if reloaded.Status != domain.RunStatusParked || reloaded.AwaitHandle == nil {
		t.Fatalf("parked run did not persist: status=%s handle=%v", reloaded.Status, reloaded.AwaitHandle)
	}

	// Wake with the resolved result.
	woken, err := svc.WakeRun(ctx, orchestration.WakeRunInput{
		RunID:  run.ID,
		Result: "suite PASSED: 42/42 green",
	})
	if err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	if woken.Status != domain.RunStatusRunning {
		t.Fatalf("after wake, status = %s, want running", woken.Status)
	}
	if woken.AwaitHandle != nil {
		t.Error("wake must clear the await handle")
	}
	if woken.LastHeartbeat == nil || time.Since(*woken.LastHeartbeat) > time.Minute {
		t.Errorf("wake must reset the heartbeat to ~now, got %v", woken.LastHeartbeat)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue on wake")
	}

	captureMu.Lock()
	env := captured.Environment
	prompt := captured.Prompt
	captureMu.Unlock()

	// Injected turn carries the resolved result.
	if !strings.Contains(prompt, "suite PASSED: 42/42 green") {
		t.Errorf("woken turn prompt does not carry the result: %q", prompt)
	}

	// Custom env survives the wake.
	if env["VROOLI_SHADOW_SCENARIOS"] != "agent-manager" {
		t.Errorf("custom env dropped on wake: %v", env)
	}
	// A fresh, verifiable identity token is present (park did not revoke it).
	token := env["VROOLI_AGENT_IDENTITY_TOKEN"]
	if token == "" {
		t.Fatal("identity token missing from woken turn env")
	}
	res, verr := svc.VerifyIdentityToken(ctx, token)
	if verr != nil {
		t.Fatalf("VerifyIdentityToken: %v", verr)
	}
	if !res.Valid || res.Claims == nil || res.Claims.RunID != run.ID {
		t.Errorf("woken identity token does not validate/bind to the run: valid=%v claims=%+v", res.Valid, res.Claims)
	}
}

// TestWakeRun_Timeout_TypedResult verifies the park-deadline path: waking with
// TimedOut=true injects a typed "timed-out / unknown" turn (agent stays in
// control) rather than presenting a fake success.
func TestWakeRun_Timeout_TypedResult(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner, captured, captureMu, done := newContinuationRunner(t)
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	run := newParkableRun(t, ctx, svc, repos)
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "git-control-tower", Key: "diff-1"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}

	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, TimedOut: true}); err != nil {
		t.Fatalf("WakeRun(timeout): %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue on timeout-wake")
	}

	captureMu.Lock()
	prompt := captured.Prompt
	captureMu.Unlock()

	if !strings.Contains(strings.ToLower(prompt), "timed out") {
		t.Errorf("timeout-wake prompt should signal a timeout, got: %q", prompt)
	}
	if !strings.Contains(prompt, "git-control-tower:diff-1") {
		t.Errorf("timeout-wake prompt should name the handle, got: %q", prompt)
	}
}

// TestStopRun_Parked_CancelsAndClearsHandle verifies abort-while-parked: stop
// moves the parked run to cancelled and clears the handle (no process to kill).
func TestStopRun_Parked_CancelsAndClearsHandle(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
	)

	run := newParkableRun(t, ctx, svc, repos)
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "run-x"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}

	if err := svc.StopRun(ctx, run.ID); err != nil {
		t.Fatalf("StopRun(parked): %v", err)
	}

	got, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != domain.RunStatusCancelled {
		t.Errorf("after stop, status = %s, want cancelled", got.Status)
	}
	if got.AwaitHandle != nil {
		t.Error("stop must clear the await handle")
	}
}

// TestWakeRun_Idempotent verifies replay-safety: waking a run that is not parked
// (already woken, or cancelled) is a no-op that returns the current run rather
// than re-resuming it — so a waiter double-resolve cannot double-wake.
func TestWakeRun_Idempotent(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
	)

	run := newParkableRun(t, ctx, svc, repos)
	// Run is currently RUNNING (never parked). Wake must be a no-op.
	got, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "ignored"})
	if err != nil {
		t.Fatalf("WakeRun(non-parked): %v", err)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("idempotent wake changed status to %s, want unchanged running", got.Status)
	}
}

// TestParkRun_RejectsDoublePark verifies one-open-handle-per-run: parking an
// already-parked run is rejected.
func TestParkRun_RejectsDoublePark(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
	)

	run := newParkableRun(t, ctx, svc, repos)
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "k1"}); err != nil {
		t.Fatalf("first ParkRun: %v", err)
	}
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "k2"}); err == nil {
		t.Fatal("expected second ParkRun to be rejected (one open handle per run)")
	}
}
