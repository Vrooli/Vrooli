package orchestration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestAttachRunMintsScopedIdentityAndDetachRevokesIt(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	secret := []byte("attached-run-test-secret")
	orchestrator := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret(secret))

	result, err := orchestrator.AttachRun(ctx, AttachRunRequest{
		HarnessKind:    "claude-code",
		HarnessSession: "session-123",
		HarnessTitle:   "Safe attached fixture",
	})
	if err != nil {
		t.Fatalf("attach run: %v", err)
	}
	if result.Run.TaskID != uuid.Nil || result.Run.ExecutionMode != domain.ExecutionModeAttached {
		t.Fatalf("unexpected attached run identity: task=%s mode=%q", result.Run.TaskID, result.Run.ExecutionMode)
	}
	claims, err := identity.VerifyToken(result.Token, secret)
	if err != nil {
		t.Fatalf("verify attached token: %v", err)
	}
	if claims.RunID != result.Run.ID || claims.TaskID != uuid.Nil || claims.ProfileKey != "claude-code" {
		t.Fatalf("unexpected token claims: %#v", claims)
	}
	if claims.Meta["execution_mode"] != string(domain.ExecutionModeAttached) {
		t.Fatalf("token execution mode metadata = %q", claims.Meta["execution_mode"])
	}

	terminal, err := orchestrator.DetachRun(ctx, result.Run.ID, "fixture complete")
	if err != nil {
		t.Fatalf("detach run: %v", err)
	}
	if !terminal.Status.IsTerminal() || terminal.IdentityTokenRevokedAt == nil {
		t.Fatalf("detach did not terminalize and revoke token: status=%q revoked=%v", terminal.Status, terminal.IdentityTokenRevokedAt)
	}
	stored, err := repos.Runs.Get(ctx, result.Run.ID)
	if err != nil {
		t.Fatalf("reload detached run: %v", err)
	}
	if stored.Status != domain.RunStatusComplete || stored.IdentityTokenRevokedAt == nil {
		t.Fatalf("stored detached run: status=%q revoked=%v", stored.Status, stored.IdentityTokenRevokedAt)
	}
	count, err := events.Count(ctx, result.Run.ID)
	if err != nil || count != 2 {
		t.Fatalf("lifecycle event count = %d, err=%v; want attach and detach", count, err)
	}
}

func TestAttachRunPersistsWebConsoleSessionWithoutDelegation(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	orchestrator := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("attached-session-correlation")))

	const webConsoleSessionID = "wc-session-without-delegation"
	result, err := orchestrator.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-without-delegation",
		WebConsoleSessionID: webConsoleSessionID, ProcessID: os.Getpid(),
	})
	if err != nil {
		t.Fatalf("attach run: %v", err)
	}
	stored, err := repos.Runs.Get(ctx, result.Run.ID)
	if err != nil {
		t.Fatalf("reload attached run: %v", err)
	}
	if stored.WebConsoleSessionID != webConsoleSessionID {
		t.Fatalf("persisted Web Console session ID = %q, want %q", stored.WebConsoleSessionID, webConsoleSessionID)
	}
	if stored.OwnerSubject != "" || len(stored.OwnerScopes) != 0 {
		t.Fatalf("correlation-only Web Console session unexpectedly granted authority: subject=%q scopes=%v", stored.OwnerSubject, stored.OwnerScopes)
	}
}

func TestAttachRunRejectsInitProcessAsPhysicalExecutorIdentity(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	orchestrator := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("attached-run-pid-validation")))

	if _, err := orchestrator.AttachRun(t.Context(), AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "invalid-init-pid", ProcessID: 1,
	}); err == nil {
		t.Fatal("PID 1 was accepted as an attached harness process identity")
	}
}

func TestAttachedRunLivenessSweepOnlyObservesHarmlessProcess(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// /bin/true is a harmless, self-terminating fixture. The test never starts
	// a shell, writes a command, or sends a signal to a user process.
	fixture := exec.Command("/bin/true")
	if err := fixture.Start(); err != nil {
		t.Fatalf("start harmless fixture: %v", err)
	}
	pid := fixture.Process.Pid
	if err := fixture.Wait(); err != nil {
		t.Fatalf("wait harmless fixture: %v", err)
	}

	now := time.Now().UTC()
	run := &domain.Run{
		ID:               uuid.New(),
		Tag:              "attached-liveness",
		RunMode:          domain.RunModeInPlace,
		ExecutionMode:    domain.ExecutionModeAttached,
		HarnessKind:      "test-harness",
		HarnessSessionID: "session-liveness",
		Status:           domain.RunStatusRunning,
		Phase:            domain.RunPhaseExecuting,
		RunnerPID:        pid,
		CreatedAt:        now.Add(-2 * time.Minute),
		UpdatedAt:        now.Add(-2 * time.Minute),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create liveness fixture: %v", err)
	}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(events),
		// Advance the injected clock past the one-minute attach handoff grace
		// period without changing the database or any host process.
		WithReconcilerClock(func() time.Time { return now.Add(2 * time.Minute) }),
		WithReconcilerConfig(ReconcilerConfig{PendingThreshold: time.Hour, MaxStaleRuns: 10}),
	)
	if reconciler.attachedRunProcessAlive(run) {
		t.Fatal("exited harmless fixture was reported alive")
	}
	stats := reconciler.reconcile(ctx)
	if stats.StaleRuns != 1 {
		t.Fatalf("stale attached runs = %d, want 1 (errors=%v)", stats.StaleRuns, stats.Errors)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload swept run: %v", err)
	}
	// A harness process that exited is a normal exit: complete, with the
	// identity revoked, never failed (D-7).
	if stored.Status != domain.RunStatusComplete || stored.ErrorMsg != "" || stored.EndedAt == nil {
		t.Fatalf("swept run = status %q error %q ended %v, want complete", stored.Status, stored.ErrorMsg, stored.EndedAt)
	}
	if stored.IdentityTokenHash != "" && stored.IdentityTokenRevokedAt == nil {
		t.Fatal("exited attached run kept a live identity token")
	}
}

func TestAttachedRunLivenessUsesInjectedPositiveAndNegativeEvidence(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	processID := 4242
	hash := "attached-identity-hash"
	effects := &controlledProcessEffects{}
	effects.setLiveness(processID, true, hash, true)
	reconciler := NewReconciler(repos.Runs, nil, withReconcilerProcessEffects(effects))

	pidRun := &domain.Run{RunnerPID: processID}
	if !reconciler.attachedRunProcessAlive(pidRun) {
		t.Fatal("attached runner PID with positive injected liveness was reported dead")
	}
	effects.setLiveness(processID, false, hash, false)
	if reconciler.attachedRunProcessAlive(pidRun) {
		t.Fatal("attached runner PID with negative injected liveness was reported alive")
	}

	tokenRun := &domain.Run{IdentityTokenHash: hash}
	if reconciler.attachedRunProcessAlive(tokenRun) {
		t.Fatal("attached identity with negative injected liveness was reported alive")
	}
	effects.setLiveness(0, false, hash, true)
	if !reconciler.attachedRunProcessAlive(tokenRun) {
		t.Fatal("attached identity with positive injected liveness was reported dead")
	}
}

func TestAttachedIdentityProjectsOnlyItsPersistedSession(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	proofSession := "1e5904ed-8378-458a-86d7-8dbb039cff19"
	calls := 0
	reader := SessionBindingProofReaderFunc(func(_ context.Context, r SessionBindingRequest) (SessionBindingProof, error) {
		calls++
		return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: r.RunID.String(), HarnessSessionID: r.HarnessSessionID, LeaseSessionID: r.HarnessSessionID, WebConsoleSessionID: proofSession, HarnessKind: r.HarnessKind, OwnerKind: "web-console-session", Binding: binding}, nil
	})
	o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("attached-projection-test-secret")), WithSessionBindingProofReader(reader))
	const session = "1e5904ed-8378-458a-86d7-8dbb039cff19"
	attached, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "thread-123", WebConsoleSessionID: session, ProcessID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	normal, err := o.VerifyIdentityToken(ctx, attached.Token)
	if err != nil || !normal.Valid || calls != 0 || normal.Claims.Meta["verified_web_console_session_id"] != "" {
		t.Fatalf("ordinary identity requested session proof: calls=%d err=%v", calls, err)
	}
	proofCtx := WithSessionBindingVerification(ctx)
	verified, err := o.VerifyIdentityToken(proofCtx, attached.Token)
	if err != nil || !verified.Valid || verified.Claims.Meta["verified_web_console_session_id"] != session {
		t.Fatalf("own session not verified: %+v %v", verified, err)
	}
	proofSession = "different-session"
	forged, err := o.VerifyIdentityToken(proofCtx, attached.Token)
	if err != nil {
		t.Fatal(err)
	}
	if forged.Valid && forged.Claims.Meta["verified_web_console_session_id"] != "" {
		t.Fatal("forged session received proof")
	}
	proofSession = session
	run, err := repos.Runs.Get(ctx, attached.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	run.HarnessSessionID = "different"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	verified, err = o.VerifyIdentityToken(proofCtx, attached.Token)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Valid && verified.Claims.Meta["verified_web_console_session_id"] != "" {
		t.Fatal("mismatched harness granted session")
	}
}

// Every attach mints a new run id, so a harness session that re-attaches would
// lose the lineage of its earlier runs if the conversation followed the run.
// The harness session is the thread: its attached runs share one conversation,
// and another session has another.
func TestReattachingTheSameHarnessSessionContinuesItsConversation(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	secret := []byte("attached-conversation-secret")
	orchestrator := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret(secret))
	attach := func(session string) *AttachRunResult {
		t.Helper()
		result, err := orchestrator.AttachRun(ctx, AttachRunRequest{HarnessKind: "claude-code", HarnessSession: session})
		if err != nil {
			t.Fatalf("attach %s: %v", session, err)
		}
		return result
	}
	first, again, other := attach("session-123"), attach("session-123"), attach("session-456")

	if first.Run.ID == again.Run.ID {
		t.Fatal("each attach is a new run")
	}
	if first.Run.ConversationID == "" || first.Run.ConversationID != again.Run.ConversationID {
		t.Fatalf("a re-attached session must reuse its conversation: %q then %q", first.Run.ConversationID, again.Run.ConversationID)
	}
	if other.Run.ConversationID == first.Run.ConversationID {
		t.Fatalf("another harness session must not share the conversation %q", first.Run.ConversationID)
	}
	for _, result := range []*AttachRunResult{first, again, other} {
		claims, err := identity.VerifyToken(result.Token, secret)
		if err != nil || claims.ConversationID != result.Run.ConversationID {
			t.Fatalf("the token must sign the run's conversation: %+v, %v", claims, err)
		}
		verified, err := orchestrator.VerifyIdentityToken(ctx, result.Token)
		if err != nil || !verified.Valid || verified.Claims.ConversationID != result.Run.ConversationID {
			t.Fatalf("verification must return the conversation: %+v, %v", verified, err)
		}
	}
}
