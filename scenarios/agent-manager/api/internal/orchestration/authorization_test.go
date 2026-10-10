//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
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
