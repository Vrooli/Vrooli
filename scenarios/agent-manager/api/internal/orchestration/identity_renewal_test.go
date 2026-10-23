package orchestration

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestExpiredAttachedBaselineRenewsSameRunAndSurvivesBrokerRestart(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	secret := []byte("attached-renewal-test-secret")
	now := time.Now().UTC()
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	var proofCalls int
	var proofMu sync.Mutex
	badProof := false
	proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, harnessSession, wcSession, harness string) (SessionBindingProof, error) {
		proofMu.Lock()
		proofCalls++
		wrongSession := badProof
		proofMu.Unlock()
		if wrongSession {
			wcSession += "-mismatch"
		}
		return SessionBindingProof{
			ProofVersion: sessionBindingProofVersion, RunID: runID.String(),
			HarnessSessionID: harnessSession, WebConsoleSessionID: wcSession,
			HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding,
		}, nil
	})
	clockNow := now.Add(-48 * time.Hour)
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret(secret), WithSessionBindingProofReader(proofReader),
		WithAuthorizations(database.NewAuthorizationRepository(db)), WithClock(func() time.Time { return clockNow }))
	attached, err := o.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "renewal-fixture", ProcessID: os.Getpid(),
		WebConsoleSessionID: "web-console-fixture",
	})
	if err != nil {
		t.Fatalf("attach ownerless root: %v", err)
	}
	oldToken := attached.Token
	clockNow = now
	oldVerified, err := o.VerifyIdentityToken(ctx, oldToken)
	if err != nil || oldVerified.Valid {
		t.Fatalf("expired baseline was accepted by API: result=%+v err=%v", oldVerified, err)
	}

	type resolution struct {
		credentials AuthorizationCredentials
		err         error
	}
	const concurrentRequests = 6
	start := make(chan struct{})
	results := make(chan resolution, concurrentRequests)
	var callers sync.WaitGroup
	for i := 0; i < concurrentRequests; i++ {
		callers.Add(1)
		go func() {
			defer callers.Done()
			<-start
			value, resolveErr := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), oldToken)
			results <- resolution{credentials: value, err: resolveErr}
		}()
	}
	close(start)
	callers.Wait()
	close(results)
	var credentials AuthorizationCredentials
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent resolution failed: %v", result.err)
		}
		value := result.credentials
		if value.BaselineToken == "" || value.BaselineToken == oldToken || value.LegacyToken != "" || value.OperationToken != "" {
			t.Fatalf("renewal did not return only an ordinary baseline (baseline=%t legacy=%t operation=%t)", value.BaselineToken != "", value.LegacyToken != "", value.OperationToken != "")
		}
		if credentials.BaselineToken == "" {
			credentials = value
		} else if credentials.BaselineToken != value.BaselineToken {
			t.Fatalf("concurrent requests produced multiple active generations")
		}
	}
	current, err := o.VerifyIdentityToken(ctx, credentials.BaselineToken)
	if err != nil || current == nil || !current.Valid || current.Claims.RunID != attached.Run.ID {
		t.Fatalf("replacement baseline is not valid for the same run: result=%+v err=%v", current, err)
	}
	stored, err := repos.Runs.Get(ctx, attached.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.IdentityTokenHash != identity.HashToken(credentials.BaselineToken) || stored.IdentityRenewalAnchorHash != identity.HashToken(oldToken) || stored.ParentRunID != nil {
		t.Fatalf("renewal changed run identity/lineage incorrectly: active=%t anchor=%t parent=%v", stored.IdentityTokenHash == identity.HashToken(credentials.BaselineToken), stored.IdentityRenewalAnchorHash == identity.HashToken(oldToken), stored.ParentRunID)
	}

	// Simulate an Agent Manager restart by constructing a fresh orchestrator
	// against the same durable repositories, secret and proof source.
	restarted := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret(secret), WithSessionBindingProofReader(proofReader),
		WithAuthorizations(database.NewAuthorizationRepository(db)), WithClock(func() time.Time { return clockNow }))
	again, err := restarted.ResolveAuthorizationCredentials(ctx, os.Getpid(), oldToken)
	if err != nil || again.BaselineToken != credentials.BaselineToken {
		t.Fatalf("fresh broker did not reconstruct the persisted generation (baseline=%t legacy=%t operation=%t) err=%v", again.BaselineToken != "", again.LegacyToken != "", again.OperationToken != "", err)
	}
	proofMu.Lock()
	observedProofCalls := proofCalls
	proofMu.Unlock()
	if observedProofCalls < concurrentRequests+1 {
		t.Fatalf("renewal did not recheck live session binding for concurrent requests and restart: proof calls=%d", observedProofCalls)
	}

	// Expire the first renewed generation while retaining the original inherited
	// anchor. The next resolution creates one later generation for the same run.
	clockNow = now.Add(identity.DefaultTTL + time.Minute)
	second, err := restarted.ResolveAuthorizationCredentials(ctx, os.Getpid(), oldToken)
	if err != nil || second.BaselineToken == "" || second.BaselineToken == credentials.BaselineToken {
		t.Fatalf("expired active generation was not rotated (baseline=%t legacy=%t operation=%t) err=%v", second.BaselineToken != "", second.LegacyToken != "", second.OperationToken != "", err)
	}
	secondVerified, err := restarted.VerifyIdentityToken(ctx, second.BaselineToken)
	if err != nil || secondVerified == nil || !secondVerified.Valid || secondVerified.Claims.RunID != attached.Run.ID {
		t.Fatalf("second generation is not valid for same run: valid=%t err=%v", secondVerified != nil && secondVerified.Valid, err)
	}
	proofMu.Lock()
	badProof = true
	proofMu.Unlock()
	clockNow = clockNow.Add(identity.DefaultTTL + time.Minute)
	refused, proofErr := restarted.ResolveAuthorizationCredentials(ctx, os.Getpid(), oldToken)
	if proofErr == nil || refused.BaselineToken != "" || refused.OperationToken != "" || refused.LegacyToken != "" || refused.OperationRefusal != "identity_refresh_unavailable" {
		t.Fatalf("mismatched owner proof exposed a credential (baseline=%t legacy=%t operation=%t refusal=%q) err=%v", refused.BaselineToken != "", refused.LegacyToken != "", refused.OperationToken != "", refused.OperationRefusal, proofErr)
	}
	proofMu.Lock()
	badProof = false
	proofMu.Unlock()
	if _, err := db.ExecContext(ctx, `UPDATE runs SET identity_token_issued_at = NULL WHERE id = ?`, attached.Run.ID.String()); err != nil {
		t.Fatal(err)
	}
	corrupt, corruptErr := restarted.ResolveAuthorizationCredentials(ctx, os.Getpid(), oldToken)
	if corruptErr == nil || corrupt.BaselineToken != "" || corrupt.LegacyToken != "" || corrupt.OperationToken != "" || corrupt.OperationRefusal != "identity_refresh_unavailable" {
		t.Fatalf("corrupt anchored generation was silently reissued (baseline=%t legacy=%t operation=%t refusal=%q) err=%v", corrupt.BaselineToken != "", corrupt.LegacyToken != "", corrupt.OperationToken != "", corrupt.OperationRefusal, corruptErr)
	}
}

func TestRenewedOwnerlessBaselineKeepsIndependentGrantOnlyCredential(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	secret := []byte(pairTestSecret)
	now := time.Now().UTC()
	clockNow := now.Add(-48 * time.Hour)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	failProof := false
	proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, harnessSession, wcSession, harness string) (SessionBindingProof, error) {
		if failProof {
			return SessionBindingProof{}, context.DeadlineExceeded
		}
		return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: runID.String(), HarnessSessionID: harnessSession, WebConsoleSessionID: wcSession, HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding}, nil
	})
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret(secret), WithSessionBindingProofReader(proofReader),
		WithAuthorizations(database.NewAuthorizationRepository(db)), WithClock(func() time.Time { return clockNow }))
	root, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "renewal-grant-root", ProcessID: os.Getpid(), WebConsoleSessionID: "renewal-grant-root-session"})
	if err != nil {
		t.Fatalf("attach ownerless root: %v", err)
	}
	clockNow = now

	policy, ok := authn.InteractivePolicy("swarm-bookkeeping")
	if !ok {
		t.Fatal("fixture policy missing")
	}
	principal := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "renewal-grant-owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: now.Add(time.Hour)}
	ownerCtx := requestidentity.WithPrincipal(ctx, principal)
	alias, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "renewal-grant-alias", ProcessID: os.Getpid(), WebConsoleSessionID: "renewal-grant-alias-session"})
	if err != nil {
		t.Fatalf("attach independently authorized alias: %v", err)
	}
	grant := approvePairGrant(t, o, ctx, ownerCtx, alias.Run.ID, alias.Run.HarnessSessionID)

	ordinary, err := o.resolveOrdinaryRunIdentityCredential(ctx, os.Getpid(), root.Token)
	if err != nil || ordinary.BaselineToken == "" || ordinary.BaselineToken == root.Token || ordinary.LegacyToken != "" || ordinary.OperationToken != "" {
		t.Fatalf("identity-purpose resolution did not return only the renewed baseline (baseline=%t legacy=%t operation=%t) err=%v", ordinary.BaselineToken != "", ordinary.LegacyToken != "", ordinary.OperationToken != "", err)
	}
	ordinaryVerified, err := o.VerifyIdentityToken(ctx, ordinary.BaselineToken)
	if err != nil || ordinaryVerified == nil || !ordinaryVerified.Valid || ordinaryVerified.Claims.RunID != root.Run.ID || ordinaryVerified.Claims.Subject != "" || len(ordinaryVerified.Claims.Scopes) != 0 {
		t.Fatalf("identity-purpose result changed the original zero-scope run: valid=%t same_run=%t zero_scope=%t err=%v", ordinaryVerified != nil && ordinaryVerified.Valid, ordinaryVerified != nil && ordinaryVerified.Claims != nil && ordinaryVerified.Claims.RunID == root.Run.ID, ordinaryVerified != nil && ordinaryVerified.Claims != nil && len(ordinaryVerified.Claims.Scopes) == 0, err)
	}
	if grantIdentity, grantErr := o.resolveOrdinaryRunIdentityCredential(ctx, os.Getpid(), grant); grantErr == nil || grantIdentity.BaselineToken != "" || grantIdentity.LegacyToken != "" || grantIdentity.OperationToken != "" {
		t.Fatalf("identity-purpose selector accepted a grant identity (baseline=%t legacy=%t operation=%t) err=%v", grantIdentity.BaselineToken != "", grantIdentity.LegacyToken != "", grantIdentity.OperationToken != "", grantErr)
	}
	defaultWithProof, defaultErr := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), root.Token)
	if defaultErr != nil || defaultWithProof.LegacyToken != grant || defaultWithProof.BaselineToken != "" || defaultWithProof.OperationToken != "" {
		t.Fatalf("default resolution stopped selecting the active independent grant (baseline=%t legacy=%t operation=%t) err=%v", defaultWithProof.BaselineToken != "", defaultWithProof.LegacyToken != "", defaultWithProof.OperationToken != "", defaultErr)
	}

	failProof = true
	refusedIdentity, identityErr := o.resolveOrdinaryRunIdentityCredential(ctx, os.Getpid(), root.Token)
	if identityErr == nil || refusedIdentity.BaselineToken != "" || refusedIdentity.LegacyToken != "" || refusedIdentity.OperationToken != "" {
		t.Fatalf("identity-purpose selector returned a credential after owner-proof failure (baseline=%t legacy=%t operation=%t) err=%v", refusedIdentity.BaselineToken != "", refusedIdentity.LegacyToken != "", refusedIdentity.OperationToken != "", identityErr)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), root.Token)
	if err != nil || credentials.LegacyToken != grant || credentials.BaselineToken != "" || credentials.OperationToken != "" {
		t.Fatalf("renewal bypassed or paired the independent grant path (baseline=%t legacy=%t operation=%t) err=%v", credentials.BaselineToken != "", credentials.LegacyToken != "", credentials.OperationToken != "", err)
	}
	claims, err := identity.VerifyToken(credentials.LegacyToken, secret)
	if err != nil || claims.RunID != alias.Run.ID || claims.Meta["authorization_id"] == "" {
		t.Fatalf("returned grant was not the independently authorized alias grant: valid=%t err=%v", err == nil, err)
	}
}

func TestRenewalRefusesRevokedOrTerminalAttachedRoot(t *testing.T) {
	for _, tc := range []struct {
		name            string
		revoke          bool
		terminal        bool
		cancelRequested bool
	}{{name: "revoked", revoke: true}, {name: "terminal", terminal: true}, {name: "cancel requested", cancelRequested: true}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, events, _ := testutil.SetupTestReposWithDB(t, db)
			secret := []byte("renewal-refusal-test-secret")
			now := time.Now().UTC()
			clockNow := now.Add(-48 * time.Hour)
			binding, err := cliutil.ObserveProcessBinding(os.Getpid())
			if err != nil {
				t.Fatal(err)
			}
			proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, harnessSession, wcSession, harness string) (SessionBindingProof, error) {
				return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: runID.String(), HarnessSessionID: harnessSession, WebConsoleSessionID: wcSession, HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding}, nil
			})
			o := New(nil, nil, repos.Runs,
				WithEvents(events), WithIdentitySecret(secret), WithSessionBindingProofReader(proofReader),
				WithAuthorizations(database.NewAuthorizationRepository(db)), WithClock(func() time.Time { return clockNow }))
			attached, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "refusal-fixture", ProcessID: os.Getpid(), WebConsoleSessionID: "refusal-session"})
			if err != nil {
				t.Fatal(err)
			}
			clockNow = now
			stored, err := repos.Runs.Get(ctx, attached.Run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.revoke {
				revokedAt := now.Add(-time.Minute)
				stored.IdentityTokenRevokedAt = &revokedAt
			}
			if tc.terminal {
				stored.Status = domain.RunStatusComplete
				stored.EndedAt = &now
			}
			if tc.cancelRequested {
				cancelAt := now.Add(-time.Minute)
				stored.CancelRequestedAt = &cancelAt
			}
			if err := repos.Runs.Update(ctx, stored); err != nil {
				t.Fatal(err)
			}
			credentials, resolveErr := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), attached.Token)
			if tc.cancelRequested {
				if resolveErr != nil || credentials.BaselineToken != "" || credentials.LegacyToken != "" || credentials.OperationToken != "" || credentials.OperationRefusal != "grant_unavailable" {
					t.Fatalf("cancel-requested run returned a credential (baseline=%t legacy=%t operation=%t refusal=%q) err=%v", credentials.BaselineToken != "", credentials.LegacyToken != "", credentials.OperationToken != "", credentials.OperationRefusal, resolveErr)
				}
			} else if resolveErr != nil || credentials.OperationToken != "" || credentials.LegacyToken != "" || credentials.BaselineToken != attached.Token {
				t.Fatalf("ineligible run returned a replacement or grant (baseline_same=%t legacy=%t operation=%t) err=%v", credentials.BaselineToken == attached.Token, credentials.LegacyToken != "", credentials.OperationToken != "", resolveErr)
			}
			verified, verifyErr := o.VerifyIdentityToken(ctx, credentials.BaselineToken)
			if verifyErr == nil && verified != nil && verified.Valid {
				t.Fatal("refused run's inherited token became valid")
			}
		})
	}
}
