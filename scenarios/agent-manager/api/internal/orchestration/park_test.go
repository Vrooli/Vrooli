package orchestration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/modules"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// newParkableRun inserts a RunStatusRunning run (with custom env + session +
// resolved config) directly, bypassing CreateRun's async execution. It is the
// starting point for park/wake round-trip tests.
func newParkableRun(t *testing.T, ctx context.Context, svc *orchestration.Orchestrator, repos *database.Repositories, workReferences ...*eventdomain.WorkReference) *domain.Run {
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
		WorkReferences: workReferences,
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

type parkedCurrentDenyResolver struct{}

func (parkedCurrentDenyResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: "allowed-model", ExcludedModels: []string{"blocked-model"}}, nil
}

func TestWakeRunRejectsCurrentModelExclusionBeforeClearingAwaitHandle(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	continued := atomic.Int32{}
	mockRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued.Add(1)
		return &runner.ExecuteResult{Success: true}, nil
	}
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	rolePath := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"park-current-deny","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"claude-code","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(rolePath, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(rolePath, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(registry),
		orchestration.WithRolePolicyState(state, parkedCurrentDenyResolver{}),
		orchestration.WithRunStateRoot(t.TempDir()))
	run := newParkableRun(t, ctx, svc, repos)
	run.ResolvedConfig = &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, RoleRef: "code.default", Model: "blocked-model"}
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "current-deny"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "must not wake"}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("current deny wake error = %v", err)
	}
	retained, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Status != domain.RunStatusParked || retained.AwaitHandle == nil {
		t.Fatalf("rejected wake mutated parked run: status=%s handle=%v", retained.Status, retained.AwaitHandle)
	}
	if continued.Load() != 0 {
		t.Fatal("current deny wake reached runner continuation")
	}
}

// TestParkRun_AndWake_PreservesIdentityEnvAndResetsHeartbeat is the Phase 2
// round-trip anchor: running→parked records an await-handle (sandbox/identity
// untouched, never revoked); parked→running (wake) clears the handle, injects
// the awaited result as the next turn, re-injects custom env + a fresh verifiable
// identity token (Phase 0 assembler), and resets the heartbeat.
func TestParkRunRegisteredChildrenProducerIsAccepted(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore))
	waiter := mocks.NewFakeWaiter(orchestration.ProducerChildren)
	registry := orchestration.NewAwaitRegistry(svc, waiter)
	svc.SetAwaitRegistry(registry)
	defer registry.Stop()
	run := newParkableRun(t, ctx, svc, repos)

	parked, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID: run.ID, Producer: orchestration.ProducerChildren, Key: run.ID.String(),
	})
	if err != nil {
		t.Fatalf("registered children producer park: %v", err)
	}
	if parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Producer != orchestration.ProducerChildren {
		t.Fatalf("registered children park = status %s handle %+v", parked.Status, parked.AwaitHandle)
	}
	eventually(t, func() bool { return registry.Watching(run.ID) }, "registered children producer did not get a waiter")
}

func TestParkRunReviewTimerValidationBeforeTransition(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore))
	registry := orchestration.NewAwaitRegistry(svc, orchestration.NewReviewTimerWaiter())
	svc.SetAwaitRegistry(registry)
	defer registry.Stop()
	run := newParkableRun(t, ctx, svc, repos)

	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: orchestration.ProducerReviewTimer, Key: "invalid"}); err == nil {
		t.Fatal("invalid timer accepted")
	}
	current, err := svc.GetRun(ctx, run.ID)
	if err != nil || current.Status != domain.RunStatusRunning || current.AwaitHandle != nil {
		t.Fatalf("invalid timer changed run: %+v, err %v", current, err)
	}
	at := time.Now().Add(time.Minute)
	parked, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: orchestration.ProducerReviewTimer, Key: at.UTC().Format(time.RFC3339Nano)})
	if err != nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil {
		t.Fatalf("valid review timer park: %+v, err %v", parked, err)
	}
	if !registry.Watching(run.ID) {
		t.Fatal("timer waiter not registered")
	}
}

// Exercise the real timer, persistent await registry and continuation together.
// This is an in-process owner test, not a managed-harness lifecycle proof.
func TestReviewTimerParksAndContinuesAtChosenTime(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	mockRunner, captured, captureMu, done := newContinuationRunner(t)
	runners := runner.NewRegistry()
	if err := runners.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(runners),
		orchestration.WithIdentitySecret([]byte("review-timer-integration-secret-0123456789")),
		orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
	waiters := orchestration.NewAwaitRegistry(svc, orchestration.NewReviewTimerWaiter())
	svc.SetAwaitRegistry(waiters)
	defer waiters.Stop()
	run := newParkableRun(t, ctx, svc, repos)
	at := time.Now().Add(orchestration.DefaultParkTurnEndGrace + 2*time.Second)
	parked, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: orchestration.ProducerReviewTimer, Key: at.UTC().Format(time.RFC3339Nano)})
	if err != nil || parked.Status != domain.RunStatusParked {
		t.Fatalf("park: run=%+v err=%v", parked, err)
	}
	reloaded, err := svc.GetRun(ctx, run.ID)
	if err != nil || reloaded.AwaitHandle == nil || reloaded.AwaitHandle.Producer != orchestration.ProducerReviewTimer {
		t.Fatalf("persisted await: run=%+v err=%v", reloaded, err)
	}
	select {
	case <-done:
		t.Fatal("continued before the selected review time")
	case <-time.After(time.Until(at) / 2):
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timer did not resume the parked run")
	}
	captureMu.Lock()
	prompt := captured.Prompt
	token := captured.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]
	captureMu.Unlock()
	if !strings.Contains(prompt, "Review time") || !strings.Contains(prompt, "does not establish worker health or completion") {
		t.Fatalf("continuation prompt = %q", prompt)
	}
	identity, err := svc.VerifyIdentityToken(ctx, token)
	if err != nil || !identity.Valid || identity.Claims == nil || identity.Claims.RunID != run.ID {
		t.Fatalf("continuation identity invalid: err=%v", err)
	}
	eventually(t, func() bool {
		current, err := svc.GetRun(ctx, run.ID)
		return err == nil && current.Status == domain.RunStatusComplete && current.AwaitHandle == nil
	}, "timer continuation did not complete and clear its await handle")
}

// Rebuild the orchestration and await registry around the same persisted owner
// records after a timer expires. This exercises recovery without restarting a
// live Agent Manager or launching a model.
func TestReviewTimerRecoveryContinuesPersistedRunOnce(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "recovery.db") + "?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)"
	openOwnerStore := func() (*database.DB, *database.Repositories, event.Store) {
		raw, err := sqlx.Connect("sqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		raw.SetMaxOpenConns(1)
		if err := coredb.EnsureSchemas(ctx, raw, modules.AllSchemas()...); err != nil {
			raw.Close()
			t.Fatal(err)
		}
		log := logrus.New()
		log.SetLevel(logrus.WarnLevel)
		db := database.NewDB(raw, log)
		return db, database.NewRepositories(db, log), event.NewSQLiteStore(db, log)
	}
	db, repos, eventStore := openOwnerStore()
	t.Cleanup(func() { _ = db.Close() })
	mockRunner, captured, captureMu, done := newContinuationRunner(t)
	runners := runner.NewRegistry()
	if err := runners.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	secret := []byte("review-timer-recovery-secret-0123456789")
	newOwner := func() *orchestration.Orchestrator {
		return orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore), orchestration.WithRunners(runners), orchestration.WithIdentitySecret(secret), orchestration.WithRunStateRoot(stateRoot), newTestRolePolicyOption(t))
	}
	oldOwner := newOwner()
	oldRegistry := orchestration.NewAwaitRegistry(oldOwner, orchestration.NewReviewTimerWaiter())
	oldOwner.SetAwaitRegistry(oldRegistry)
	run := newParkableRun(t, ctx, oldOwner, repos)
	at := time.Now().Add(orchestration.DefaultParkTurnEndGrace + time.Second)
	if _, err := oldOwner.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: orchestration.ProducerReviewTimer, Key: at.UTC().Format(time.RFC3339Nano)}); err != nil {
		oldRegistry.Stop()
		t.Fatal(err)
	}
	oldRegistry.Stop()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if delay := time.Until(at); delay > 0 {
		time.Sleep(delay + 20*time.Millisecond)
	}
	select {
	case <-done:
		t.Fatal("stopped owner continued the run")
	default:
	}
	db, repos, eventStore = openOwnerStore()
	newOwnerInstance := newOwner()
	newRegistry := orchestration.NewAwaitRegistry(newOwnerInstance, orchestration.NewReviewTimerWaiter())
	newOwnerInstance.SetAwaitRegistry(newRegistry)
	defer newRegistry.Stop()
	count, err := newRegistry.RecoverParkedRuns(ctx)
	if err != nil || count != 1 {
		t.Fatalf("recover = %d, %v", count, err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("recovered timer did not continue")
	}
	captureMu.Lock()
	prompt, token := captured.Prompt, captured.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]
	captureMu.Unlock()
	if !strings.Contains(prompt, "does not establish worker health or completion") {
		t.Fatalf("continuation = %q", prompt)
	}
	identity, err := newOwnerInstance.VerifyIdentityToken(ctx, token)
	if err != nil || !identity.Valid || identity.Claims == nil || identity.Claims.RunID != run.ID {
		t.Fatalf("recovered continuation identity = %+v, %v", identity, err)
	}
	eventually(t, func() bool {
		current, err := newOwnerInstance.GetRun(ctx, run.ID)
		return err == nil && current.Status == domain.RunStatusComplete && current.AwaitHandle == nil
	}, "recovered run did not complete and clear its handle")
	if count, err := newRegistry.RecoverParkedRuns(ctx); err != nil || count != 0 {
		t.Fatalf("completed run was recovered again: %d, %v", count, err)
	}
}

func TestParkRunWithoutRegistryRetainsIsolatedSeam(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore))
	run := newParkableRun(t, ctx, svc, repos)

	parked, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID: run.ID, Producer: "isolated-test-producer", Key: "isolated/key",
	})
	if err != nil {
		t.Fatalf("nil-registry isolated park: %v", err)
	}
	if parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Producer != "isolated-test-producer" {
		t.Fatalf("nil-registry isolated park = status %s handle %+v", parked.Status, parked.AwaitHandle)
	}
}

func TestParkRun_AndWake_PreservesIdentityEnvAndResetsHeartbeat(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))

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
		orchestration.WithMaintenanceGate(gate),
		newTestRolePolicyOption(t),
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
	// A parked lifetime is already admitted. Planned maintenance must preserve
	// its durable waiter/identity and allow completion, not fence its wake.
	if _, err := gate.Enter(ctx, "owner", "rollout while parked"); err != nil {
		t.Fatal(err)
	}
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

func TestWakeEffortCheckinCommitsQueuedCursorWithRetainedRunWake(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{SupportsContinuation: true})
	started := make(chan runner.ContinueRequest, 1)
	release := make(chan struct{})
	finished := make(chan struct{})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		started <- req
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		close(finished)
		return &runner.ExecuteResult{Success: true, ExitCode: 0, SessionID: req.SessionID}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	eventErr := errors.New("wake status event append unavailable")
	failNextStatus := &atomic.Int32{}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: 5 * time.Minute, MaxConcurrentRuns: 10}),
		orchestration.WithEvents(failParkedStatusEventStore{Store: eventStore, err: eventErr, failNextStatus: failNextStatus}), orchestration.WithRunners(registry),
		orchestration.WithIdentitySecret([]byte("effort-checkin-identity-secret-0123456789")),
		orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
	const effortRef = "effort:retained-checkin"
	run := newParkableRun(t, ctx, svc, repos, &eventdomain.WorkReference{
		Kind: "effort", Id: effortRef, Relationship: "orchestrator", Verified: true,
		Visibility: eventdomain.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:      eventdomain.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	})
	if _, _, err := repos.Runs.(repository.EffortSignalCursorRepository).QueueEffortSignalCursor(ctx, run.ID, effortRef, 3); err != nil {
		t.Fatalf("queue signal cursor: %v", err)
	}
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "supervision", Key: "effort-checkin"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	failNextStatus.Store(1)
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{
		RunID: run.ID, Result: "must not wake for a mismatched effort",
		EffortSignalReceipt: &domain.EffortSignalReceipt{EffortRef: "effort:other", Through: 3},
	}); err == nil || !domain.IsPreEffectRefusal(err) {
		t.Fatalf("mismatched effort receipt should be refused before effects: %v", err)
	}
	if err := svc.WakeEffortCheckin(ctx, run.ID, effortRef, 4); err == nil {
		t.Fatal("unqueued signal receipt should fail before runner dispatch")
	}
	stillParked, err := svc.GetStoredRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("read run after rejected receipt: %v", err)
	}
	if stillParked.Status != domain.RunStatusParked || stillParked.AwaitHandle == nil || stillParked.EffortSignalCursors[effortRef] != (domain.EffortSignalCursor{QueuedSeq: 3, DeliveredSeq: 0}) {
		t.Fatalf("rejected receipt changed the parked owner or cursor: status=%s handle=%+v cursor=%+v", stillParked.Status, stillParked.AwaitHandle, stillParked.EffortSignalCursors[effortRef])
	}
	select {
	case <-started:
		t.Fatal("uncommitted receipt reached runner continuation")
	default:
	}

	if err := svc.WakeEffortCheckin(ctx, run.ID, effortRef, 3); !errors.Is(err, eventErr) {
		t.Fatalf("WakeEffortCheckin error = %v, want committed status-event publication error", err)
	}
	var continued runner.ContinueRequest
	select {
	case continued = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for retained run continuation")
	}

	prompt := continued.Prompt
	lowerPrompt := strings.ToLower(prompt)
	if !strings.Contains(prompt, effortRef) || !strings.Contains(prompt, "sequence 3") ||
		!strings.Contains(lowerPrompt, "reread the current canonical effort feedback") ||
		!strings.Contains(lowerPrompt, "check-in consumed the previous await") ||
		!strings.Contains(lowerPrompt, "inspect your current run state") ||
		!strings.Contains(lowerPrompt, "park this same run again with the same producer and key before ending the turn") ||
		!strings.Contains(lowerPrompt, "do not treat a prior successful park as proof that this run is currently parked") {
		t.Fatalf("check-in prompt must direct a current canonical reread: %q", prompt)
	}
	for _, untrusted := range []string{"private feedback detail", "operator says: continue", "signal payload"} {
		if strings.Contains(strings.ToLower(prompt), untrusted) {
			t.Fatalf("signal detail or authority language leaked into check-in prompt: %q", prompt)
		}
	}

	stored, err := svc.GetStoredRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("read woken run: %v", err)
	}
	if stored.Status != domain.RunStatusRunning || stored.AwaitHandle != nil {
		t.Fatalf("check-in did not wake the exact parked run: status=%s handle=%v", stored.Status, stored.AwaitHandle)
	}
	if got := stored.EffortSignalCursors[effortRef]; got != (domain.EffortSignalCursor{QueuedSeq: 3, DeliveredSeq: 3}) {
		t.Fatalf("wake receipt did not commit with status: %+v", got)
	}
	if err := svc.WakeEffortCheckin(ctx, run.ID, effortRef, 3); err != nil {
		t.Fatalf("duplicate check-in should be idempotent: %v", err)
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("continuation runner did not finish")
	}
	eventually(t, func() bool {
		finishedRun, err := svc.GetStoredRun(ctx, run.ID)
		return err == nil && finishedRun.Status.IsTerminal()
	}, "continuation did not persist its final status")
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
		newTestRolePolicyOption(t),
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

func TestWakeRunBindsExactLegacyTaskBeforeContinuation(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
	mockRunner, _, _, _ := newContinuationRunner(t)
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	var releaseOnce sync.Once
	releaseRunner := func() { releaseOnce.Do(func() { close(release) }) }
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		close(started)
		<-release
		close(done)
		return &runner.ExecuteResult{Success: true, SessionID: req.SessionID}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	initialTask, err := repos.Tasks.Get(ctx, run.TaskID)
	if err != nil || initialTask.Status != domain.TaskStatusQueued || initialTask.CurrentRunID != nil {
		t.Fatalf("legacy Task precondition = %+v err=%v, want queued and unowned", initialTask, err)
	}
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "legacy-wake"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "resume exact run"}); err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	t.Cleanup(func() {
		releaseRunner()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("blocked runner did not release during cleanup")
			return
		}
		// The mock's done signal fires before the orchestration goroutine has
		// persisted its terminal Run result. Wait for the durable transition so
		// the test database is not closed under the result handler.
		completed, err := waitForRunCompletion(t, ctx, svc, run.ID, 3*time.Second)
		if err != nil {
			t.Errorf("wait for durable run completion: %v", err)
			return
		}
		if completed.Status != domain.RunStatusComplete {
			t.Errorf("durable run status = %s, want complete", completed.Status)
		}
	})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("runner continuation did not start")
	}
	task, err := repos.Tasks.Get(ctx, run.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.TaskStatusRunning || task.CurrentRunID == nil || *task.CurrentRunID != run.ID {
		t.Fatalf("wake Task projection = status %s currentRunID %v, want running and exact Run %s", task.Status, task.CurrentRunID, run.ID)
	}
	releaseRunner()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runner continuation did not finish")
	}
}

func newWakeOwnershipGuardRunner(t *testing.T) (runner.Registry, *atomic.Int32) {
	t.Helper()
	mockRunner, _, _, _ := newContinuationRunner(t)
	calls := &atomic.Int32{}
	mockRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		calls.Add(1)
		return &runner.ExecuteResult{Success: true}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	return registry, calls
}

func TestWakeRunRefusesAmbiguousLegacyTaskOwner(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
	registry, calls := newWakeOwnershipGuardRunner(t)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	competitor := *run
	competitor.ID = uuid.New()
	competitor.Tag = competitor.ID.String()
	competitor.Status = domain.RunStatusPending
	competitor.Phase = domain.RunPhaseQueued
	competitor.AwaitHandle = nil
	competitor.SessionID = "competitor-" + uuid.NewString()
	if err := repos.Runs.Create(ctx, &competitor); err != nil {
		t.Fatalf("create competing nonterminal Run: %v", err)
	}
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "ambiguous-wake"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "must refuse"}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "could not be safely bound") {
		t.Fatalf("ambiguous legacy wake error = %v, want exact-owner pre-effect refusal", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("ambiguous wake invoked continuation %d times", calls.Load())
	}
	task, err := repos.Tasks.Get(ctx, run.TaskID)
	if err != nil || task.Status != domain.TaskStatusQueued || task.CurrentRunID != nil {
		t.Fatalf("ambiguous wake changed Task ownership: task=%+v err=%v", task, err)
	}
	parked, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil {
		t.Fatalf("ambiguous wake changed parked Run: run=%+v err=%v", parked, err)
	}
}

func TestWakeRunRefusesCancelledTaskAndDifferentCurrentOwner(t *testing.T) {
	ctx := context.Background()
	t.Run("cancelled task", func(t *testing.T) {
		db, cleanup := testutil.SetupTestDB(t)
		t.Cleanup(cleanup)
		repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
		registry, calls := newWakeOwnershipGuardRunner(t)
		svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
		run := newParkableRun(t, ctx, svc, repos)
		if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "cancelled-wake"}); err != nil {
			t.Fatalf("ParkRun: %v", err)
		}
		if claimed, err := repos.Runs.EnsureCurrentRun(ctx, run.TaskID, run.ID, time.Now()); err != nil || !claimed {
			t.Fatalf("bind exact parked Run for cancellation: claimed=%v err=%v", claimed, err)
		}
		if _, changed, err := repos.Runs.CancelTaskAttempt(ctx, run.TaskID, &run.ID, true, time.Now()); err != nil || !changed {
			t.Fatalf("cancel exact parked Task attempt: changed=%v err=%v", changed, err)
		}
		if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "must refuse"}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "terminal Task state refuses continuation") {
			t.Fatalf("cancelled Task wake error = %v, want exact terminal-owner refusal", err)
		}
		if calls.Load() != 0 {
			t.Fatalf("cancelled Task wake invoked continuation %d times", calls.Load())
		}
		task, err := repos.Tasks.Get(ctx, run.TaskID)
		if err != nil || task.Status != domain.TaskStatusCancelled || task.CurrentRunID == nil || *task.CurrentRunID != run.ID {
			t.Fatalf("cancelled Task changed after wake refusal: task=%+v err=%v", task, err)
		}
		parked, err := repos.Runs.Get(ctx, run.ID)
		if err != nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil {
			t.Fatalf("cancelled Task wake changed parked Run: run=%+v err=%v", parked, err)
		}
	})
	t.Run("different current owner", func(t *testing.T) {
		db, cleanup := testutil.SetupTestDB(t)
		t.Cleanup(cleanup)
		repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
		registry, calls := newWakeOwnershipGuardRunner(t)
		svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithRunStateRoot(t.TempDir()), newTestRolePolicyOption(t))
		task := mustCreateTask(t, svc, ctx, &domain.Task{Title: "wake owner", Description: "current owner fence", ScopePath: "src/"})
		owner := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, Tag: "current-owner", RunMode: domain.RunModeInPlace, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.CreateTaskAttempt(ctx, task, task, owner, nil); err != nil {
			t.Fatalf("create exact current attempt: %v", err)
		}
		old := time.Now().Add(-time.Minute)
		orphan := &domain.Run{
			ID: uuid.New(), TaskID: task.ID, Tag: "legacy-parked-orphan", RunMode: domain.RunModeInPlace,
			Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, SessionID: "orphan-" + uuid.NewString(),
			StartedAt: &old, LastHeartbeat: &old, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
			ApprovalState: domain.ApprovalStateNone, CreatedAt: old, UpdatedAt: old,
		}
		if err := repos.Runs.Create(ctx, orphan); err != nil {
			t.Fatalf("create named legacy Run: %v", err)
		}
		if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: orphan.ID, Producer: "test-genie", Key: "different-owner"}); err != nil {
			t.Fatalf("ParkRun: %v", err)
		}
		if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: orphan.ID, Result: "must refuse"}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "requested Run no longer owns the Task") {
			t.Fatalf("different-owner wake error = %v, want exact current-owner refusal", err)
		}
		if calls.Load() != 0 {
			t.Fatalf("different-owner wake invoked continuation %d times", calls.Load())
		}
		got, err := repos.Tasks.Get(ctx, task.ID)
		if err != nil || got.CurrentRunID == nil || *got.CurrentRunID != owner.ID || got.Status != domain.TaskStatusRunning {
			t.Fatalf("wake displaced current owner: task=%+v err=%v", got, err)
		}
		parked, err := repos.Runs.Get(ctx, orphan.ID)
		if err != nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil {
			t.Fatalf("different-owner wake changed parked Run: run=%+v err=%v", parked, err)
		}
	})
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

func TestWakeRun_ConcurrentNotificationsStartOneContinuation(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	var continueCalls atomic.Int32
	continued := make(chan struct{})
	var signalOnce sync.Once
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	mockRunner.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continueCalls.Add(1)
		signalOnce.Do(func() { close(continued) })
		return &runner.ExecuteResult{Success: true, ExitCode: 0, SessionID: req.SessionID}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		newTestRolePolicyOption(t),
	)
	run := newParkableRun(t, ctx, svc, repos)
	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{RunID: run.ID, Producer: "test-genie", Key: "run-concurrent"}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "terminal"}); err != nil {
				t.Errorf("WakeRun: %v", err)
			}
		}()
	}
	wg.Wait()
	select {
	case <-continued:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the single continuation")
	}
	if got := continueCalls.Load(); got != 1 {
		t.Fatalf("concurrent wake notifications started %d continuations, want 1", got)
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

// ParkedRunsOnHandle matches producer and key exactly; the friction wake
// relies on it instead of scanning a client-side page of runs.
func TestParkedRunsOnHandleMatchesProducerAndKey(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore))
	running := newParkableRun(t, ctx, svc, repos)
	park := func(producer, key string) *domain.Run {
		t.Helper()
		run := *running
		run.ID = uuid.New()
		run.Tag = run.ID.String()
		run.Status = domain.RunStatusParked
		run.AwaitHandle = &domain.AwaitHandle{Producer: producer, Key: key}
		if err := repos.Runs.Create(ctx, &run); err != nil {
			t.Fatal(err)
		}
		return &run
	}
	want := park(orchestration.ProducerChildren, "orchestrator-1")
	park("supervision", "orchestrator-1")
	park(orchestration.ProducerChildren, "orchestrator-2")

	tests := []struct {
		name, producer, key string
		want                []uuid.UUID
	}{
		{"exact handle", orchestration.ProducerChildren, "orchestrator-1", []uuid.UUID{want.ID}},
		{"key under another producer", "test-genie", "orchestrator-1", nil},
		{"unknown key", orchestration.ProducerChildren, "missing", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ParkedRunsOnHandle(ctx, tt.producer, tt.key)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]uuid.UUID, 0, len(got))
			for _, run := range got {
				ids = append(ids, run.ID)
			}
			if len(ids) != len(tt.want) || (len(ids) == 1 && ids[0] != tt.want[0]) {
				t.Fatalf("matched %v, want %v", ids, tt.want)
			}
		})
	}
}
