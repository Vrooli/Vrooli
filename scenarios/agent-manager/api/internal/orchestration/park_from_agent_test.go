package orchestration_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// parkFromAgentSecret is the shared identity secret used to mint test tokens.
var parkFromAgentSecret = []byte("phase4-park-from-agent-secret-0123456789")

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
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
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
	mockRunner.StopFunc = func(context.Context, uuid.UUID) error { stopped.Store(true); return nil }
	mockRunner.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued <- stopped.Load()
		return &runner.ExecuteResult{Success: true, SessionID: req.SessionID}, nil
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
