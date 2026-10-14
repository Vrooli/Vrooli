package accounts

import (
	"context"
	"github.com/vrooli/api-core/schedule"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
	"slices"
	"testing"
	"time"
)

func TestFiniteEffortActualOwnerApprovalAndCurrentGrantLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, db := newRepo(t)
	account, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "finite-fixture@example.test", PasswordHash: "disposable-unused-password-hash"})
	if err != nil {
		t.Fatal(err)
	}
	grants := authorization.NewService(repo.(authorization.ScopeStore), nil)
	for _, scope := range []string{"agent-manager:read", "agent-manager:write", "agent-manager:orchestrate"} {
		if _, err = grants.Grant(ctx, account.ID, scope, authorization.Meta{}); err != nil {
			t.Fatal(err)
		}
	}
	keys, err := authcrypto.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
	session := sessions.NewManager(redisstate.NewMemory(), nil)
	service := NewService(ServiceConfig{Repo: repo, Signer: signer, Sessions: session, Authorization: grants, Clock: schedule.System()})
	token, err := signer.Sign(authcrypto.TokenInput{UserID: account.ID, Audience: "scenario-authenticator:default", Scopes: []string{"agent-manager:read"}})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := service.ApproveEffortOwner(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if owner.Subject != account.ID || !owner.Enabled || !slices.Equal(owner.Ceiling, []string{"agent-manager:read"}) || len(owner.Scopes) != 3 {
		t.Fatal("approval enlarged offered token ceiling or lost current owner grants")
	}
	if _, err = service.ApproveEffortOwner(ctx, "invalid"); err == nil {
		t.Fatal("invalid JWT approved")
	}
	if err = session.BlacklistAccess(ctx, token, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ApproveEffortOwner(ctx, token); err == nil {
		t.Fatal("logged-out JWT approved new grant")
	}
	owner, err = service.CurrentEffortOwner(ctx, account.ID)
	if err != nil || !owner.Enabled {
		t.Fatal("explicit finite current-owner read did not survive logout", err)
	}
	if _, err = grants.Revoke(ctx, account.ID, "agent-manager:orchestrate", authorization.Meta{}); err != nil {
		t.Fatal(err)
	}
	owner, err = service.CurrentEffortOwner(ctx, account.ID)
	if err != nil || slices.Contains(owner.Scopes, "agent-manager:orchestrate") {
		t.Fatal("current grant removal was stale", err)
	}
	if err = repo.SetLoginFailure(ctx, account.ID, 5, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CurrentEffortOwner(ctx, account.ID); err == nil {
		t.Fatal("locked owner remained enabled")
	}
	if err = repo.SetLoginSuccess(ctx, account.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CurrentEffortOwner(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM accounts WHERE id = ?", account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CurrentEffortOwner(ctx, account.ID); err == nil {
		t.Fatal("deleted owner remained enabled")
	}
}
