package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/database"
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

// newParkFromAgentSvcWithRepos is newParkFromAgentSvc but also returns the
// repositories, so tests that need to seed run state directly (e.g. the re-park
// guard's persisted streak / last-await fields) can do so without driving the
// racy wake-continuation goroutine.
func newParkFromAgentSvcWithRepos(t *testing.T) (*orchestration.Orchestrator, *domain.Run, *database.Repositories) {
	t.Helper()
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
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
	return svc, run, repos
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
