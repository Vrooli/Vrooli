//go:build linux

package orchestration

import (
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/orchestration/testutil"
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"os"
	"testing"
	"time"
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
	if verified.Claims.ExpiresAt > p.ExpiresAt.Unix() {
		t.Fatal("grant outlives approver proof")
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
	verified, err = o.VerifyIdentityToken(ctx, token)
	if err != nil || verified.Valid {
		t.Fatal("revoked bearer remained active")
	}
}
