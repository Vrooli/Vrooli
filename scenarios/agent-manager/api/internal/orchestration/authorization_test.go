//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestExistingTokenlessProcessCanAdoptDurableGrantAndRevocationIsImmediate(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("authorization-test-secret")), WithAuthorizations(database.NewAuthorizationRepository(db)))
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "fixture", Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600})
	if err != nil {
		t.Fatal(err)
	}
	if token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err != nil || found || token != "" {
		t.Fatal("pending consent granted authority")
	}
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	p := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(30 * time.Minute)}
	owner := requestidentity.WithPrincipal(ctx, p)
	if _, err := o.ApproveAuthorization(ctx, g.ID); err == nil {
		t.Fatal("unverified caller approved its grant")
	}
	g, err = o.ApproveAuthorization(owner, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.ApproveAuthorization(owner, g.ID); err == nil {
		t.Fatal("replayed approval was accepted")
	}
	token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid())
	if err != nil || !found || token == "" {
		t.Fatalf("approved process cannot resolve credential: %v", err)
	}
	verified, err := o.VerifyIdentityToken(ctx, token)
	if err != nil || !verified.Valid || verified.Claims.RunID != g.RunID {
		t.Fatal("approved identity is not live")
	}
	if verified.Claims.ExpiresAt < time.Now().Add(55*time.Minute).Unix() {
		t.Fatal("grant ended with the approver's login instead of its approved lifetime")
	}
	stored, err := o.GetAuthorization(ctx, g.ID)
	if err != nil || stored.TokenHash == "" || stored.ClaimsJSON == "" {
		t.Fatal("grant is not durable")
	}
	body, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	json.Unmarshal(body, &receipt)
	if receipt["token"] != nil || receipt["ClaimsJSON"] != nil || receipt["TokenHash"] != nil {
		t.Fatal("receipt exposed a credential")
	}
	directory, err := os.MkdirTemp("", "agb-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	socket := filepath.Join(directory, "identity.sock")
	stop, err := o.startAuthorizationBrokerAt(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop() })
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	response, err := client.Get("http://local-agent-manager/credential?pid=1")
	if err != nil {
		t.Fatal(err)
	}
	var credential struct {
		Token string `json:"token"`
	}
	json.NewDecoder(response.Body).Decode(&credential)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || credential.Token != token {
		t.Fatal("kernel-identified existing process did not obtain approved credential")
	}
	other := p
	other.Subject = "other"
	if _, err := o.RevokeAuthorization(requestidentity.WithPrincipal(ctx, other), g.ID); err == nil {
		t.Fatal("different owner revoked grant")
	}
	if _, err := o.RevokeAuthorization(owner, g.ID); err != nil {
		t.Fatal(err)
	}
	if token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err == nil || !found || token != "" {
		t.Fatal("revoked credential still resolves")
	}
	response, err = client.Get("http://local-agent-manager/credential")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatal("broker continued issuing revoked credentials")
	}
	verified, err = o.VerifyIdentityToken(ctx, token)
	if err != nil || verified.Valid {
		t.Fatal("revoked bearer remained active")
	}
}

func TestAuthorizationAttachmentRetainsVerifiedLauncherSession(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	const harnessSession, webConsoleSession = "host-launch-session", "verified-wc-session"
	proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, gotHarnessSession, expectedWC, harness string) (SessionBindingProof, error) {
		if gotHarnessSession != harnessSession || harness != "codex" || (expectedWC != "" && expectedWC != webConsoleSession) {
			return SessionBindingProof{}, errors.New("session tuple changed")
		}
		runSelector := ""
		if runID != uuid.Nil {
			runSelector = runID.String()
		}
		return SessionBindingProof{
			ProofVersion: sessionBindingProofVersion, RunID: runSelector,
			HarnessSessionID: harnessSession, WebConsoleSessionID: webConsoleSession,
			HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding,
		}, nil
	})
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("authorization-binding-secret")),
		WithAuthorizations(database.NewAuthorizationRepository(db)), WithSessionBindingProofReader(proofReader))
	grant, err := o.RequestAuthorization(ctx, AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: harnessSession,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("request authorization with live launcher proof: %v", err)
	}
	if grant.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("pending grant locator = %q, want verified launcher locator %q", grant.WebConsoleSessionID, webConsoleSession)
	}
	stored, err := o.GetAuthorization(ctx, grant.ID)
	if err != nil || stored.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("durable grant lost verified session locator: grant=%+v err=%v", stored, err)
	}
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	principal := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
	approved, err := o.ApproveAuthorization(requestidentity.WithPrincipal(ctx, principal), grant.ID)
	if err != nil {
		t.Fatalf("approve with same exact launcher proof: %v", err)
	}
	attached, err := o.GetRun(ctx, approved.RunID)
	if err != nil || attached.WebConsoleSessionID != webConsoleSession || attached.HarnessSessionID != harnessSession {
		t.Fatalf("attached run did not retain the exact launcher tuple: run=%+v err=%v", attached, err)
	}
	if attached.OwnerSubject != "owner" || len(attached.OwnerScopes) == 0 {
		t.Fatalf("approved operator attribution was lost: subject=%q scopes=%v", attached.OwnerSubject, attached.OwnerScopes)
	}
}

func TestLegacyPendingAuthorizationResolvesLauncherLocatorOnlyAtFreshApproval(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	const harnessSession, webConsoleSession = "legacy-pending-host-session", "current-wc-owner"
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("legacy-pending-binding-secret")),
		WithAuthorizations(database.NewAuthorizationRepository(db)))
	grant, err := o.RequestAuthorization(ctx, AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: harnessSession,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err != nil {
		t.Fatal(err)
	}
	if grant.WebConsoleSessionID != "" {
		t.Fatalf("legacy fixture unexpectedly has a session locator: %q", grant.WebConsoleSessionID)
	}
	// This simulates a request already pending before the locator was retained.
	// The fresh approval resolves it from the original host-session key and the
	// immutable process binding, not from a replacement consent or run mutation.
	o.sessionBindingProofReader = SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, gotHarnessSession, expectedWC, harness string) (SessionBindingProof, error) {
		if gotHarnessSession != harnessSession || harness != "codex" || (expectedWC != "" && expectedWC != webConsoleSession) {
			return SessionBindingProof{}, errors.New("legacy request tuple changed")
		}
		runSelector := ""
		if runID != uuid.Nil {
			runSelector = runID.String()
		}
		return SessionBindingProof{
			ProofVersion: sessionBindingProofVersion, RunID: runSelector,
			HarnessSessionID: harnessSession, WebConsoleSessionID: webConsoleSession,
			HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding,
		}, nil
	})
	reused, err := o.RequestAuthorization(ctx, AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: harnessSession,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err != nil || reused.ID != grant.ID {
		t.Fatalf("refreshing the pending locator replaced consent: reused=%v err=%v", reused != nil && reused.ID == grant.ID, err)
	}
	if reused.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("pending consent did not gain its verified locator: %q", reused.WebConsoleSessionID)
	}
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	principal := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
	approved, err := o.ApproveAuthorization(requestidentity.WithPrincipal(ctx, principal), grant.ID)
	if err != nil {
		t.Fatalf("fresh approval could not resolve exact current launcher owner: %v", err)
	}
	stored, err := o.GetAuthorization(ctx, grant.ID)
	if err != nil || stored.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("approval did not persist resolved locator: grant=%+v err=%v", stored, err)
	}
	attached, err := o.GetRun(ctx, approved.RunID)
	if err != nil || attached.WebConsoleSessionID != webConsoleSession || attached.HarnessSessionID != harnessSession {
		t.Fatalf("freshly attached run lost verified locator: run=%+v err=%v", attached, err)
	}
}

func TestAuthorizationRefusesMismatchedLauncherProofBeforeCreatingConsent(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("mismatched-binding-secret")),
		WithAuthorizations(database.NewAuthorizationRepository(db)),
		WithSessionBindingProofReader(SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, _ string, _ string, harness string) (SessionBindingProof, error) {
			runSelector := ""
			if runID != uuid.Nil {
				runSelector = runID.String()
			}
			return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: runSelector, HarnessSessionID: "different-host-session", WebConsoleSessionID: "wrong-session", HarnessKind: harness, OwnerKind: "web-console-session", Binding: binding}, nil
		})))
	_, err = o.RequestAuthorization(ctx, AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "expected-host-session",
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err == nil {
		t.Fatal("mismatched host-session proof created consent")
	}
	grants, err := o.authorizations.List(ctx, os.Getpid())
	if err != nil || len(grants) != 0 {
		t.Fatalf("mismatched proof left a consent row: grants=%d err=%v", len(grants), err)
	}
}

func TestAuthorizationRejectsClaimedWebConsoleLocatorWithoutProofReader(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	authRepo := database.NewAuthorizationRepository(db)
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("missing-proof-reader-secret")), WithAuthorizations(authRepo))
	_, err := o.RequestAuthorization(ctx, AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "claimed-wc-host-session", WebConsoleSessionID: "unverified-wc-session",
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err == nil {
		t.Fatal("claimed Web Console locator fell back to an unattached session")
	}
	grants, err := authRepo.List(ctx, os.Getpid())
	if err != nil || len(grants) != 0 {
		t.Fatalf("missing proof reader left consent rows: count=%d err=%v", len(grants), err)
	}
}

func TestAuthorizationApprovalRequiresCurrentWebConsoleProofReader(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	const harnessSession, webConsoleSession = "approval-proof-host-session", "approval-proof-wc-session"
	proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, gotHarnessSession, expectedWC, harness string) (SessionBindingProof, error) {
		if gotHarnessSession != harnessSession || harness != "codex" || (expectedWC != "" && expectedWC != webConsoleSession) {
			return SessionBindingProof{}, errors.New("session tuple changed")
		}
		runSelector := ""
		if runID != uuid.Nil {
			runSelector = runID.String()
		}
		return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: runSelector,
			HarnessSessionID: harnessSession, WebConsoleSessionID: webConsoleSession, HarnessKind: harness,
			OwnerKind: "web-console-session", Binding: binding}, nil
	})
	authRepo := database.NewAuthorizationRepository(db)
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("approval-missing-proof-reader-secret")),
		WithAuthorizations(authRepo), WithSessionBindingProofReader(proofReader))
	grant, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: harnessSession,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600})
	if err != nil {
		t.Fatalf("request authorization: %v", err)
	}
	o.sessionBindingProofReader = nil
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	principal := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess,
		Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
	if _, err := o.ApproveAuthorization(requestidentity.WithPrincipal(ctx, principal), grant.ID); err == nil {
		t.Fatal("approval proceeded after required Web Console proof reader became unavailable")
	}
	stored, err := authRepo.Get(ctx, grant.ID)
	if err != nil || stored.State != domain.AuthorizationPending || stored.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("unavailable proof changed pending consent: state=%q wc=%q err=%v", stored.State, stored.WebConsoleSessionID, err)
	}
	runs, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 10}})
	if err != nil || len(runs) != 0 {
		t.Fatalf("unavailable proof created attached runs: count=%d err=%v", len(runs), err)
	}
}

func TestAuthorizationApprovalRefusesChangedLauncherOwnerBeforeEffects(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	const harnessSession, webConsoleSession = "approval-host-session", "approval-wc-session"
	currentWC := webConsoleSession
	proofReader := SessionBindingProofReaderFunc(func(_ context.Context, runID uuid.UUID, gotHarnessSession, expectedWC, harness string) (SessionBindingProof, error) {
		if gotHarnessSession != harnessSession || harness != "codex" || (expectedWC != "" && expectedWC != webConsoleSession) {
			return SessionBindingProof{}, errors.New("session tuple changed")
		}
		runSelector := ""
		if runID != uuid.Nil {
			runSelector = runID.String()
		}
		return SessionBindingProof{ProofVersion: sessionBindingProofVersion, RunID: runSelector,
			HarnessSessionID: harnessSession, WebConsoleSessionID: currentWC, HarnessKind: harness,
			OwnerKind: "web-console-session", Binding: binding}, nil
	})
	authRepo := database.NewAuthorizationRepository(db)
	o := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte("approval-changed-binding-secret")),
		WithAuthorizations(authRepo), WithSessionBindingProofReader(proofReader))
	grant, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: harnessSession,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600})
	if err != nil {
		t.Fatalf("request authorization: %v", err)
	}
	currentWC = "replacement-wc-session"
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	principal := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
	if _, err := o.ApproveAuthorization(requestidentity.WithPrincipal(ctx, principal), grant.ID); err == nil {
		t.Fatal("approval accepted a changed launch-owner locator")
	}
	stored, err := authRepo.Get(ctx, grant.ID)
	if err != nil || stored.State != domain.AuthorizationPending || stored.WebConsoleSessionID != webConsoleSession {
		t.Fatalf("failed owner recheck mutated pending consent: state=%q wc=%q err=%v", stored.State, stored.WebConsoleSessionID, err)
	}
	runs, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 20}})
	if err != nil || len(runs) != 0 {
		t.Fatalf("failed owner recheck created a run: runs=%d err=%v", len(runs), err)
	}
}

func TestAuthorizationCannotBypassRetainedRunAdmissionByOmittingRunID(t *testing.T) {
	for _, finite := range []bool{false, true} {
		t.Run(map[bool]string{false: "frozen_scope", true: "finite_lineage"}[finite], func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, events, _ := testutil.SetupTestReposWithDB(t, db)
			o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("test-secret")), WithAuthorizations(database.NewAuthorizationRepository(db)))
			policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
			p := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
			owner := requestidentity.WithPrincipal(ctx, p)
			attached, err := o.AttachRun(owner, AttachRunRequest{HarnessKind: "codex", HarnessSession: "managed-fixture", ProcessID: os.Getpid()})
			if err != nil {
				t.Fatal(err)
			}
			run, err := repos.Runs.Get(ctx, attached.Run.ID)
			if err != nil {
				t.Fatal(err)
			}
			run.ExecutionMode = domain.ExecutionModeCodecPipe
			originalScopes := []string{"swarm-manager:read"}
			run.ResolvedConfig = &domain.RunConfig{Admission: &domain.RunAdmission{FrozenIdentityScopes: &originalScopes}}
			if finite {
				run.ParentRunID = &run.ID
			}
			if err := repos.Runs.Update(ctx, run); err != nil {
				t.Fatal(err)
			}
			g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "forged-unmanaged", Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:*"}, DurationSeconds: 3600})
			if err != nil {
				t.Fatal(err)
			}
			if g.RunID != run.ID {
				t.Fatal("omitted run ID hid managed admission")
			}
			if _, err := o.ApproveAuthorization(owner, g.ID); err == nil {
				t.Fatal("interactive consent replaced retained admission")
			}
			if token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err != nil || found || token != "" {
				t.Fatal("refused admission acquired authority")
			}
		})
	}
}

func TestAuthorizationRefusesManagedRunAppearingAfterPreparation(t *testing.T) {
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("test-secret")), WithAuthorizations(database.NewAuthorizationRepository(db)))
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "fixture", Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:*"}, DurationSeconds: 3600})
	if err != nil {
		t.Fatal(err)
	}
	policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
	p := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour)}
	owner := requestidentity.WithPrincipal(ctx, p)
	attached, err := o.AttachRun(owner, AttachRunRequest{HarnessKind: "codex", HarnessSession: "fixture", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	run, err := repos.Runs.Get(ctx, attached.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	run.ExecutionMode = domain.ExecutionModeCodecPipe
	run.ParentRunID = &run.ID
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := o.ApproveAuthorization(owner, g.ID); err == nil {
		t.Fatal("late managed admission was replaced by pending interactive consent")
	}
}

func TestAuthorizationReviewAllowsDaysAwayAndStartsAuthorityAtApproval(t *testing.T) {
	for _, delay := range []time.Duration{3 * 24 * time.Hour, 7 * 24 * time.Hour} {
		t.Run(delay.String(), func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, events, _ := testutil.SetupTestReposWithDB(t, db)
			now := time.Now().UTC()
			o := New(nil, nil, repos.Runs, WithClock(func() time.Time { return now }), WithEvents(events), WithIdentitySecret([]byte("review-window-test")), WithAuthorizations(database.NewAuthorizationRepository(db)))
			req := AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "delayed-review", Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:*"}, DurationSeconds: 7200}
			g, err := o.RequestAuthorization(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if g.RequestExpiresAt.Sub(g.CreatedAt) != 7*24*time.Hour {
				t.Fatal("review window does not accommodate a week away")
			}
			now = now.Add(delay)
			if token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err != nil || found || token != "" {
				t.Fatal("waiting for review granted authority")
			}
			policy, _ := authn.InteractivePolicy("swarm-bookkeeping")
			owner := requestidentity.WithPrincipal(ctx, requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: now.Add(6 * time.Hour)})
			approved, err := o.ApproveAuthorization(owner, g.ID)
			if delay >= 7*24*time.Hour {
				if err == nil {
					t.Fatal("expired request approved")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if approved.ApprovedAt == nil || !approved.ApprovedAt.Equal(now) || !approved.Grant.ExpiresAt.Equal(now.Add(2*time.Hour)) {
				t.Fatal("time awaiting review consumed the approved permission lifetime")
			}
		})
	}
}

func TestLocalOwnerEligibilityBrokerExcludesActiveRunPeers(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode domain.ExecutionMode
	}{
		{name: "attached exact root", mode: domain.ExecutionModeAttached},
		{name: "managed exact root", mode: domain.ExecutionModeCodecPipe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, events, _ := testutil.SetupTestReposWithDB(t, db)
			run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, RunnerPID: os.Getpid(), ExecutionMode: tc.mode, ApprovalState: domain.ApprovalStateNone}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("local-owner-test-secret")))
			response := localOwnerEligibilityResponse(t, o, ctx)
			if response.StatusCode != http.StatusForbidden {
				t.Fatalf("active %s peer eligibility status=%d, want refusal", tc.mode, response.StatusCode)
			}
			response.Body.Close()
		})
	}

	t.Run("unbound ordinary process gets explicit empty result", func(t *testing.T) {
		ctx := context.Background()
		db, cleanup := testutil.SetupTestDB(t)
		t.Cleanup(cleanup)
		repos, events, _ := testutil.SetupTestReposWithDB(t, db)
		o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("local-owner-test-secret")))
		response := localOwnerEligibilityResponse(t, o, ctx)
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			t.Fatalf("unbound ordinary process eligibility status=%d, want explicit 204", response.StatusCode)
		}
	})

	t.Run("managed root refuses descendant peer", func(t *testing.T) {
		ctx := context.Background()
		db, cleanup := testutil.SetupTestDB(t)
		t.Cleanup(cleanup)
		repos, events, _ := testutil.SetupTestReposWithDB(t, db)
		run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, RunnerPID: os.Getpid(), ExecutionMode: domain.ExecutionModeCodecPipe, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte("local-owner-test-secret")))
		dir, err := os.MkdirTemp("", "am-local-owner-child-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		socket := filepath.Join(dir, "identity.sock")
		stop, err := o.startAuthorizationBrokerAt(socket)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = stop() })
		cmd := exec.Command(os.Args[0], "-test.run=^TestLocalOwnerEligibilityBrokerChild$")
		cmd.Env = append(os.Environ(), "AM_LOCAL_OWNER_CHILD_SOCKET="+socket)
		if _, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("managed descendant did not fail closed: %v", err)
		}
	})

	t.Run("run lookup failure is not absence", func(t *testing.T) {
		ctx := context.Background()
		db, cleanup := testutil.SetupTestDB(t)
		t.Cleanup(cleanup)
		repos, events, _ := testutil.SetupTestReposWithDB(t, db)
		o := New(nil, nil, failingRunList{RunRepository: repos.Runs}, WithEvents(events), WithIdentitySecret([]byte("local-owner-test-secret")))
		response := localOwnerEligibilityResponse(t, o, ctx)
		defer response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("unavailable run lookup eligibility status=%d, want refusal", response.StatusCode)
		}
	})
}

type failingRunList struct{ repository.RunRepository }

func (f failingRunList) List(context.Context, repository.RunListFilter) ([]*domain.Run, error) {
	return nil, errors.New("run registry unavailable")
}

func localOwnerEligibilityResponse(t *testing.T, o *Orchestrator, ctx context.Context) *http.Response {
	t.Helper()
	dir, err := os.MkdirTemp("", "am-local-owner-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "identity.sock")
	stop, err := o.startAuthorizationBrokerAt(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stop() })
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local-agent-manager/credential?purpose=local_owner_eligibility", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestLocalOwnerEligibilityBrokerChild(t *testing.T) {
	socket := os.Getenv("AM_LOCAL_OWNER_CHILD_SOCKET")
	if socket == "" {
		return
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	req, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential?purpose=local_owner_eligibility", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("child peer eligibility status=%d, want refusal", resp.StatusCode)
	}
}
