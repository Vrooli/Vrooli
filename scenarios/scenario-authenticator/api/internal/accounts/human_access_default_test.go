package accounts

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/trustposture"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
)

// All credentials, grants and bindings here exist only in a disposable fixture.
// Never print AuthResult, tokens, session identifiers or token hashes on failure.
func TestSharedHumanExchangeAndRefreshLifetime(t *testing.T) {
	t.Setenv("JWT_EXPIRY_MINUTES", "")
	ctx := context.Background()
	clock := schedule.NewFake(time.Now().UTC().Truncate(time.Second))
	repo, _ := newRepo(t)
	account, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "human-ttl@example.test", PasswordHash: "unused-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	grants := authorization.NewService(repo.(authorization.ScopeStore), nil)
	if _, err = grants.Grant(ctx, account.ID, "agent-manager:write", authorization.Meta{}); err != nil {
		t.Fatal(err)
	}
	bindings := repo.(MachineBindingStore)
	if _, err = bindings.LinkMachineBinding(ctx, MachineBinding{MachineID: "disposable-host", LocalPrincipal: "unix:1000", AccountID: account.ID, RealmID: "default", IsDefault: true, LinkedAt: clock.Now()}); err != nil {
		t.Fatal(err)
	}
	keys, err := authcrypto.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defaults, err := trustposture.DefaultsFor(trustposture.Shared)
	if err != nil {
		t.Fatal(err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: authcrypto.ResolveExpiry(defaults.AccessTokenTTL), Now: clock.Now})
	manager := sessions.NewManager(redisstate.NewMemory(), clock.Now)
	service := NewService(ServiceConfig{Repo: repo, Signer: signer, Sessions: manager, MachineBindings: bindings, Authorization: grants, Clock: clock})
	initial, err := service.ExchangeMachinePrincipal(ctx, "disposable-host", "unix:1000", "", RequestMeta{})
	if err != nil {
		t.Fatal(err)
	}
	assertClaims := func(result AuthResult) string {
		t.Helper()
		claims, err := signer.Validate(result.AccessToken, "scenario-authenticator:default")
		if err != nil {
			t.Fatal("signed fixture access token failed verification")
		}
		if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != 4*time.Hour || result.AccessExpiresAt.Sub(clock.Now()) != 4*time.Hour {
			t.Fatal("human access expiry is not four hours")
		}
		if claims.Subject != account.ID || !slices.Equal(claims.Scopes, []string{"agent-manager:write"}) || result.Audience != initial.Audience {
			t.Fatal("account, scope or audience changed")
		}
		return claims.SessionID
	}
	sessionID := assertClaims(initial)
	family, _, err := manager.RefreshFamilyForToken(ctx, initial.RefreshToken)
	if err != nil {
		t.Fatal("fixture family lookup failed")
	}
	originalSession, found, err := manager.FindSession(ctx, sessionID)
	if err != nil || !found || originalSession.ExpiresAt.Sub(originalSession.CreatedAt) != 7*24*time.Hour {
		t.Fatal("session lifetime changed from seven days")
	}
	clock.Advance(4*time.Hour + time.Second)
	if _, err = signer.Validate(initial.AccessToken, initial.Audience); err == nil {
		t.Fatal("expired initial access remained valid")
	}
	renewed, err := service.Refresh(ctx, initial.RefreshToken, RequestMeta{})
	if err != nil {
		t.Fatal("supported refresh failed")
	}
	if renewed.RefreshToken == initial.RefreshToken {
		t.Fatal("refresh did not rotate")
	}
	if assertClaims(renewed) != sessionID {
		t.Fatal("refresh created a new session")
	}
	renewedFamily, _, err := manager.RefreshFamilyForToken(ctx, renewed.RefreshToken)
	if err != nil || renewedFamily != family {
		t.Fatal("refresh created a new family")
	}
	afterSession, found, err := manager.FindSession(ctx, sessionID)
	if err != nil || !found || !afterSession.ExpiresAt.Equal(originalSession.ExpiresAt) {
		t.Fatal("refresh extended the established session deadline")
	}
	all, err := manager.ListSessions(ctx, account.ID)
	if err != nil || len(all) != 1 {
		t.Fatal("refresh duplicated sessions")
	}
	if _, err = service.Refresh(ctx, initial.RefreshToken, RequestMeta{}); !errors.Is(err, ErrRefreshRejected) {
		t.Fatal("old refresh replay accepted")
	}
	if _, err = service.Refresh(ctx, renewed.RefreshToken, RequestMeta{}); !errors.Is(err, ErrRefreshRejected) {
		t.Fatal("replayed family remained usable")
	}
}
