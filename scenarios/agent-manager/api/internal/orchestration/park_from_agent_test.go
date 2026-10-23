package orchestration_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// parkFromAgentSecret is the shared identity secret used to mint test tokens.
var parkFromAgentSecret = []byte("phase4-park-from-agent-secret-0123456789")

type parkProcessVisibleRunner struct {
	*runner.MockRunner
	pid atomic.Int32
}

func (r *parkProcessVisibleRunner) PID(uuid.UUID) int { return int(r.pid.Load()) }

type immediateParkWaiter struct{ started chan struct{} }

func (immediateParkWaiter) Producer() string { return "immediate-park-test" }

func (w immediateParkWaiter) Wait(context.Context, string) (string, error) {
	close(w.started)
	return "ready", nil
}

type blockingParkWaiter struct{ started chan struct{} }

func (blockingParkWaiter) Producer() string { return "manual-park-test" }

func (w blockingParkWaiter) Wait(ctx context.Context, _ string) (string, error) {
	close(w.started)
	<-ctx.Done()
	return "", ctx.Err()
}

type wakeOnParkedPersistRepository struct {
	repository.RunRepository
	onParked func(*domain.Run)
}

type failParkedStatusEventStore struct {
	event.Store
	err            error
	failNextStatus *atomic.Int32
}

func (s failParkedStatusEventStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	for _, evt := range events {
		if evt != nil && evt.EventType == domain.EventTypeStatus && s.failNextStatus.CompareAndSwap(1, 0) {
			return s.err
		}
	}
	return s.Store.Append(ctx, runID, events...)
}

func (r wakeOnParkedPersistRepository) Update(ctx context.Context, run *domain.Run) error {
	if err := r.RunRepository.Update(ctx, run); err != nil {
		return err
	}
	if run.Status == domain.RunStatusParked && r.onParked != nil {
		r.onParked(run)
	}
	return nil
}

// newParkFromAgentSvc builds an orchestrator wired with a claude-code runner and
// the shared identity secret, plus a running run, for ParkRunFromAgent auth tests.
func newParkFromAgentSvc(t *testing.T) (*orchestration.Orchestrator, *domain.Run) {
	svc, run, _ := newParkFromAgentSvcWithRepos(t)
	return svc, run
}

// newParkFromAgentSvcWithRepos also returns the repositories so tests can seed
// durable state without driving the racy wake-continuation goroutine.
func newParkFromAgentSvcWithRepos(t *testing.T) (*orchestration.Orchestrator, *domain.Run, *database.Repositories) {
	svc, run, repos, _ := newParkFromAgentSvcWithStopObserver(t)
	return svc, run, repos
}

// newParkFromAgentSvcWithStopObserver adds a runner-stop observer for tests
// that assert a refusal happens before any process-control effect.
func newParkFromAgentSvcWithStopObserver(t *testing.T) (*orchestration.Orchestrator, *domain.Run, *database.Repositories, *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	stopCalls := &atomic.Int32{}
	mockRunner.StopFunc = func(context.Context, uuid.UUID) error {
		stopCalls.Add(1)
		return nil
	}
	processRunner := &parkProcessVisibleRunner{MockRunner: mockRunner}
	registry := runner.NewRegistry()
	if err := registry.Register(processRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

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
		orchestration.WithIdentitySecret(parkFromAgentSecret),
	)

	run := newParkableRun(t, ctx, svc, repos)
	return svc, run, repos, stopCalls
}

// mintToken mints a valid identity token binding to runID with a 24h TTL.
func mintToken(t *testing.T, runID uuid.UUID) string {
	t.Helper()
	now := time.Now()
	token, err := identity.GenerateToken(&identity.Claims{
		RunID:     runID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(24 * time.Hour).Unix(),
	}, parkFromAgentSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return token
}

// activateToken mirrors production token issuance: a signed token is useful
// only after its hash is persisted on the owning run.
func activateToken(t *testing.T, ctx context.Context, repos *database.Repositories, run *domain.Run) string {
	t.Helper()
	token := mintToken(t, run.ID)
	run.IdentityTokenHash = identity.HashToken(token)
	run.IdentityTokenRevokedAt = nil
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("persist active token hash: %v", err)
	}
	return token
}

// TestParkRunFromAgent_OwningTokenParks: an in-run caller presenting its own
// valid identity token parks the run (running→parked, handle recorded) and gets
// the clean turn-ending message back.
func TestParkRunFromAgent_OwningTokenParks(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)
	token := activateToken(t, ctx, repos, run)

	res, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID:         run.ID,
		Producer:      "git-control-tower",
		Key:           "agent-manager/am-park-resume",
		IdentityToken: token,
	})
	if err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	if res == nil || res.Run == nil {
		t.Fatal("expected a park result with a run")
	}
	if res.Run.Status != domain.RunStatusParked {
		t.Fatalf("status = %s, want parked", res.Run.Status)
	}
	if res.Run.AwaitHandle == nil || res.Run.AwaitHandle.Producer != "git-control-tower" {
		t.Fatalf("await handle not recorded: %+v", res.Run.AwaitHandle)
	}
	if !strings.Contains(res.Message, "PARKED") {
		t.Errorf("turn-ending message missing PARKED marker: %q", res.Message)
	}

	// Persisted.
	reloaded, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if reloaded.Status != domain.RunStatusParked {
		t.Fatalf("persisted status = %s, want parked", reloaded.Status)
	}
}

func TestParkRunFromAgent_ImmediateWaiterWaitsForOriginalProcessExit(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	continued := make(chan struct{}, 1)
	stopCalled := make(chan struct{}, 1)
	baseRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	baseRunner.SetAvailable(true, "available")
	baseRunner.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	baseRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- struct{}{}
		return &runner.ExecuteResult{Success: true}, nil
	}
	baseRunner.StopFunc = func(context.Context, uuid.UUID) error {
		stopCalled <- struct{}{}
		return nil
	}
	processRunner := &parkProcessVisibleRunner{MockRunner: baseRunner}
	processRunner.pid.Store(1234)
	runners := runner.NewRegistry()
	if err := runners.Register(processRunner); err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(runners),
		orchestration.WithRunStateRoot(t.TempDir()), orchestration.WithIdentitySecret(parkFromAgentSecret),
		newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	run.ResolvedConfig.Model = "mock-model"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	token := activateToken(t, ctx, repos, run)
	waiter := immediateParkWaiter{started: make(chan struct{})}
	registry := orchestration.NewAwaitRegistry(svc, waiter)
	svc.SetAwaitRegistry(registry)
	t.Cleanup(func() {
		processRunner.pid.Store(0)
		registry.Stop()
	})

	if _, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: waiter.Producer(), Key: "already-ready", IdentityToken: token,
	}); err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	select {
	case <-waiter.started:
	case <-time.After(time.Second):
		t.Fatal("immediately-ready waiter was not started")
	}
	select {
	case <-continued:
		t.Fatal("continuation started while the parked run's original process was live")
	case <-time.After(250 * time.Millisecond):
	}
	select {
	case <-stopCalled:
	case <-time.After(4 * time.Second):
		t.Fatal("park turn-end stop was not attempted")
	}
	select {
	case <-continued:
		t.Fatal("continuation started before the runner reported process exit")
	case <-time.After(150 * time.Millisecond):
	}
	processRunner.pid.Store(0)
	select {
	case <-continued:
	case <-time.After(5 * time.Second):
		t.Fatal("continuation did not start after the runner reported process exit")
	}
	completed, err := waitForRunCompletion(t, ctx, svc, run.ID, 3*time.Second)
	if err != nil {
		t.Fatalf("wait for durable continuation completion: %v", err)
	}
	if completed.Status != domain.RunStatusComplete {
		t.Fatalf("durable continuation status = %s, want complete", completed.Status)
	}
}

func TestParkRunFromAgent_InstallsProcessBarrierBeforePublishingParkedRun(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	continued := make(chan struct{}, 1)
	baseRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	baseRunner.SetAvailable(true, "available")
	baseRunner.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	baseRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- struct{}{}
		return &runner.ExecuteResult{Success: true}, nil
	}
	baseRunner.StopFunc = func(context.Context, uuid.UUID) error { return nil }
	processRunner := &parkProcessVisibleRunner{MockRunner: baseRunner}
	processRunner.pid.Store(1234)
	runners := runner.NewRegistry()
	if err := runners.Register(processRunner); err != nil {
		t.Fatal(err)
	}

	var svc *orchestration.Orchestrator
	var wakeErr error
	wakeDuringPersist := wakeOnParkedPersistRepository{
		RunRepository: repos.Runs,
		onParked: func(run *domain.Run) {
			wakeCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			defer cancel()
			_, wakeErr = svc.WakeRun(wakeCtx, orchestration.WakeRunInput{RunID: run.ID, Result: "ready"})
		},
	}
	svc = orchestration.New(repos.Profiles, repos.Tasks, wakeDuringPersist,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(runners),
		orchestration.WithRunStateRoot(t.TempDir()), orchestration.WithIdentitySecret(parkFromAgentSecret),
		newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	run.ResolvedConfig.Model = "mock-model"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	token := activateToken(t, ctx, repos, run)
	t.Cleanup(func() { processRunner.pid.Store(0) })

	if _, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: "test-genie", Key: "already-ready", IdentityToken: token,
	}); err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	if !errors.Is(wakeErr, context.DeadlineExceeded) {
		t.Fatalf("wake error = %v, want deadline while waiting on the turn-end barrier", wakeErr)
	}
	select {
	case <-continued:
		t.Fatalf("continuation started before the original runner exited (wake error: %v)", wakeErr)
	default:
	}
	if got := processRunner.pid.Load(); got == 0 {
		t.Fatal("runner unexpectedly exited before the wake was held at the publication boundary")
	}
}

func TestParkRunFromAgent_RefusesWhenRunnerIsUnavailableBeforePark(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithIdentitySecret(parkFromAgentSecret),
		newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	token := activateToken(t, ctx, repos, run)

	if _, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: "test-genie", Key: "already-ready", IdentityToken: token,
	}); err == nil {
		t.Fatal("ParkRunFromAgent succeeded without a runner to stop and drain")
	}
	after, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after unavailable-runner refusal: %v", err)
	}
	if after.Status != domain.RunStatusRunning || after.AwaitHandle != nil {
		t.Fatalf("unavailable-runner refusal changed run state: status=%s await=%+v", after.Status, after.AwaitHandle)
	}
}

func TestParkRunFromAgent_CompletesObligationsAfterStatusEventAppendFailure(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	continued := make(chan struct{}, 1)
	stopCalled := make(chan struct{}, 1)
	baseRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	baseRunner.SetAvailable(true, "available")
	baseRunner.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	baseRunner.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- struct{}{}
		return &runner.ExecuteResult{Success: true}, nil
	}
	baseRunner.StopFunc = func(context.Context, uuid.UUID) error {
		stopCalled <- struct{}{}
		return nil
	}
	processRunner := &parkProcessVisibleRunner{MockRunner: baseRunner}
	processRunner.pid.Store(1234)
	runners := runner.NewRegistry()
	if err := runners.Register(processRunner); err != nil {
		t.Fatal(err)
	}
	eventErr := errors.New("status event append unavailable")
	failNextStatus := &atomic.Int32{}
	failNextStatus.Store(1)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(failParkedStatusEventStore{Store: eventStore, err: eventErr, failNextStatus: failNextStatus}),
		orchestration.WithRunners(runners), orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithIdentitySecret(parkFromAgentSecret), newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	run.ResolvedConfig.Model = "mock-model"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	token := activateToken(t, ctx, repos, run)
	waiter := immediateParkWaiter{started: make(chan struct{})}
	registry := orchestration.NewAwaitRegistry(svc, waiter)
	svc.SetAwaitRegistry(registry)
	t.Cleanup(func() {
		processRunner.pid.Store(0)
		registry.Stop()
	})

	if _, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: waiter.Producer(), Key: "committed-before-event-error", IdentityToken: token,
	}); !errors.Is(err, eventErr) {
		t.Fatalf("ParkRunFromAgent error = %v, want the post-commit status-event error", err)
	}
	parked, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after status-event failure: %v", err)
	}
	if parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Key != "committed-before-event-error" {
		t.Fatalf("durable park = status %s handle %+v, want committed matching handle", parked.Status, parked.AwaitHandle)
	}
	select {
	case <-waiter.started:
	case <-time.After(time.Second):
		t.Fatal("committed park's waiter was not registered after event failure")
	}
	select {
	case <-continued:
		t.Fatal("continuation started before runner process exit")
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case <-stopCalled:
	case <-time.After(4 * time.Second):
		t.Fatal("turn-end stop was not attempted after committed park")
	}
	select {
	case <-continued:
		t.Fatal("continuation started before PID reported exit")
	case <-time.After(100 * time.Millisecond):
	}
	processRunner.pid.Store(0)
	select {
	case <-continued:
	case <-time.After(5 * time.Second):
		t.Fatal("committed park waiter did not resume after PID exit")
	}
	completed, err := waitForRunCompletion(t, ctx, svc, run.ID, 3*time.Second)
	if err != nil {
		t.Fatalf("wait for durable continuation completion: %v", err)
	}
	if completed.Status != domain.RunStatusComplete {
		t.Fatalf("durable continuation status = %s, want complete", completed.Status)
	}
}

func TestParkRun_RegistersWaiterAfterCommittedStatusEventAppendFailure(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	eventErr := errors.New("status event append unavailable")
	failNextStatus := &atomic.Int32{}
	failNextStatus.Store(1)
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(failParkedStatusEventStore{Store: eventStore, err: eventErr, failNextStatus: failNextStatus}),
		newTestRolePolicyOption(t))
	run := newParkableRun(t, ctx, svc, repos)
	waiter := blockingParkWaiter{started: make(chan struct{})}
	registry := orchestration.NewAwaitRegistry(svc, waiter)
	svc.SetAwaitRegistry(registry)
	t.Cleanup(registry.Stop)

	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID: run.ID, Producer: waiter.Producer(), Key: "committed-before-event-error",
	}); !errors.Is(err, eventErr) {
		t.Fatalf("ParkRun error = %v, want post-commit status-event error", err)
	}
	parked, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after status-event failure: %v", err)
	}
	if parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Key != "committed-before-event-error" {
		t.Fatalf("durable park = status %s handle %+v, want committed matching handle", parked.Status, parked.AwaitHandle)
	}
	select {
	case <-waiter.started:
	case <-time.After(time.Second):
		t.Fatal("committed manual park's waiter was not registered after event failure")
	}
}

func TestParkRunFromAgent_AttachedRunRefusedBeforeParkOrStop(t *testing.T) {
	ctx := context.Background()
	svc, run, repos, stopCalls := newParkFromAgentSvcWithStopObserver(t)
	run.ExecutionMode = domain.ExecutionModeAttached
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("set attached execution mode: %v", err)
	}
	token := activateToken(t, ctx, repos, run)

	_, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: "test-genie", Key: "attached/should-not-park", IdentityToken: token,
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "attached") {
		t.Fatalf("attached run park error = %v, want explicit attached-mode refusal", err)
	}

	after, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after refusal: %v", err)
	}
	if after.Status != domain.RunStatusRunning || after.AwaitHandle != nil {
		t.Fatalf("refused attached run changed state: status=%s await=%+v", after.Status, after.AwaitHandle)
	}
	if got := stopCalls.Load(); got != 0 {
		t.Fatalf("runner stop calls = %d, want 0 after pre-effect refusal", got)
	}
}

func TestParkRunFromAgent_UnknownProducerRefusedBeforeParkOrStop(t *testing.T) {
	ctx := context.Background()
	svc, run, repos, stopCalls := newParkFromAgentSvcWithStopObserver(t)
	registry := orchestration.NewAwaitRegistry(svc)
	svc.SetAwaitRegistry(registry)
	defer registry.Stop()
	token := activateToken(t, ctx, repos, run)

	_, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: "a227-r17", Key: "lantern-ledger-v5", IdentityToken: token,
	})
	if err == nil || !strings.Contains(err.Error(), "no waiter is registered") {
		t.Fatalf("unknown producer park error = %v, want pre-effect registered-waiter refusal", err)
	}
	retained, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after refusal: %v", err)
	}
	if retained.Status != domain.RunStatusRunning || retained.AwaitHandle != nil {
		t.Fatalf("unknown producer changed run before refusal: status=%s handle=%+v", retained.Status, retained.AwaitHandle)
	}
	if got := stopCalls.Load(); got != 0 {
		t.Fatalf("runner stop calls = %d, want 0 after unsupported producer refusal", got)
	}
}

func TestVerifyIdentityTokenRejectsRevokedRunToken(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)
	token := activateToken(t, ctx, repos, run)
	verified, err := svc.VerifyIdentityToken(ctx, token)
	if err != nil || verified == nil || !verified.Valid {
		t.Fatalf("active token verification = %+v, %v", verified, err)
	}
	now := time.Now()
	run.IdentityTokenRevokedAt = &now
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("persist token revocation: %v", err)
	}
	verified, err = svc.VerifyIdentityToken(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if verified == nil || verified.Valid {
		t.Fatalf("revoked token verification = %+v, want invalid", verified)
	}
}

// TestParkRunFromAgent_ForeignTokenRejected: a token that does not own the run
// (claims.run_id != path id) is rejected and the run is NOT parked.
func TestParkRunFromAgent_ForeignTokenRejected(t *testing.T) {
	ctx := context.Background()
	svc, run := newParkFromAgentSvc(t)

	foreign := mintToken(t, uuid.New()) // a different run's token
	_, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID:         run.ID,
		Producer:      "test-genie",
		Key:           "x/y",
		IdentityToken: foreign,
	})
	if err == nil {
		t.Fatal("expected rejection for a foreign identity token")
	}

	reloaded, _ := svc.GetRun(ctx, run.ID)
	if reloaded.Status != domain.RunStatusRunning {
		t.Fatalf("run must stay running after a rejected park, got %s", reloaded.Status)
	}
}

// TestParkRunFromAgent_InvalidTokenRejected: a malformed/unsigned token is
// rejected.
func TestParkRunFromAgent_InvalidTokenRejected(t *testing.T) {
	ctx := context.Background()
	svc, run := newParkFromAgentSvc(t)

	for _, tok := range []string{"", "garbage", "not.a.real.token"} {
		_, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
			RunID:         run.ID,
			Producer:      "test-genie",
			Key:           "x/y",
			IdentityToken: tok,
		})
		if err == nil {
			t.Fatalf("expected rejection for invalid token %q", tok)
		}
	}

	reloaded, _ := svc.GetRun(ctx, run.ID)
	if reloaded.Status != domain.RunStatusRunning {
		t.Fatalf("run must stay running after rejected parks, got %s", reloaded.Status)
	}
}

// TestParkRunFromAgent_NonRunningRejected: parking a run that is already parked
// is rejected (CanParkRun: one open handle per run), even with a valid token.
func TestParkRunFromAgent_NonRunningRejected(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)
	token := activateToken(t, ctx, repos, run)

	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID: run.ID, Producer: "test-genie", Key: "a/b",
	}); err != nil {
		t.Fatalf("seed park: %v", err)
	}

	_, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID:         run.ID,
		Producer:      "test-genie",
		Key:           "a/b",
		IdentityToken: token,
	})
	if err == nil {
		t.Fatal("expected rejection when parking an already-parked run")
	}
}

// A wake that lands inside the park turn-end grace (the awaited result was
// already available) must not start the continuation until the parked turn's
// process has been stopped: two live processes on one conversation corrupt it,
// and Codex refuses the second writer outright.
func TestWakeRun_WaitsForParkedTurnToEnd(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	var stopped atomic.Bool
	continued := make(chan bool, 1)
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	processRunner := &parkProcessVisibleRunner{MockRunner: mockRunner}
	processRunner.pid.Store(1234)
	mockRunner.StopFunc = func(context.Context, uuid.UUID) error {
		stopped.Store(true)
		processRunner.pid.Store(0)
		return nil
	}
	mockRunner.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- stopped.Load()
		return &runner.ExecuteResult{Success: true, SessionID: req.SessionID}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(processRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithIdentitySecret(parkFromAgentSecret),
		newTestRolePolicyOption(t),
	)
	run := newParkableRun(t, ctx, svc, repos)
	token := activateToken(t, ctx, repos, run)

	if _, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: "test-genie", Key: "already-finished", IdentityToken: token,
	}); err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: "passed"}); err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	select {
	case wasStopped := <-continued:
		if !wasStopped {
			t.Fatal("continuation started while the parked turn was still running")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for the continuation")
	}
	completed, err := waitForRunCompletion(t, ctx, svc, run.ID, 3*time.Second)
	if err != nil {
		t.Fatalf("wait for durable continuation completion: %v", err)
	}
	if completed.Status != domain.RunStatusComplete {
		t.Fatalf("durable continuation status = %s, want complete", completed.Status)
	}
}

func TestInteractiveIdentityRequiresPersistedSessionClaim(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)
	run.ExecutionMode = domain.ExecutionModeInteractive
	run.WebConsoleSessionID = "a244-bound-session"
	makeToken := func(sessionID string) string {
		t.Helper()
		now := time.Now()
		token, err := identity.GenerateToken(&identity.Claims{
			RunID: run.ID, TaskID: run.TaskID, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
			Meta: map[string]string{"web_console_session_id": sessionID},
		}, parkFromAgentSecret)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	for _, tc := range []struct {
		name, claim string
		want        bool
	}{
		{name: "matching", claim: "a244-bound-session", want: true},
		{name: "missing", claim: "", want: false},
		{name: "other session", claim: "other-session", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := makeToken(tc.claim)
			run.IdentityTokenHash = identity.HashToken(token)
			run.IdentityTokenRevokedAt = nil
			if err := repos.Runs.Update(ctx, run); err != nil {
				t.Fatal(err)
			}
			verified, err := svc.VerifyIdentityToken(ctx, token)
			if err != nil || verified == nil || verified.Valid != tc.want {
				t.Fatalf("verification = %+v, %v; want valid=%t", verified, err, tc.want)
			}
		})
	}
}
