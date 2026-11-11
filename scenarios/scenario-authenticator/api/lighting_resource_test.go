package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scopecatalog"
)

// The normal owner manifest must feed the issuer's actual Login resolver, not
// merely a relying-party test signer. Every account/token below is disposable.
func TestManagedLightingResourceLoginPreservesAccountAuthority(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Environment map[string]string `json:"environment"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	audiences := parseResourceAudiences(manifest.Environment["VROOLI_AUTH_RESOURCE_AUDIENCES"])
	for _, resource := range []string{"device-sync-hub", "web-console", "agent-manager", "notification-hub-provider-decision", "lighting-control"} {
		if audiences[resource] != "scenario-authenticator:"+resource {
			t.Fatalf("managed audience missing or changed for %s", resource)
		}
	}
	var lighting struct {
		Authentication struct {
			Audience     string `json:"audience"`
			Capabilities []struct {
				ID string `json:"id"`
			} `json:"capabilities"`
		} `json:"authentication"`
	}
	raw, err = os.ReadFile(filepath.Join("..", "..", "lighting-control", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &lighting); err != nil {
		t.Fatal(err)
	}
	if lighting.Authentication.Audience != audiences["lighting-control"] {
		t.Fatal("issuer and relying-party audiences differ")
	}
	declared := map[string]bool{}
	for _, c := range lighting.Authentication.Capabilities {
		declared[c.ID] = true
	}
	if !declared["lighting-control:read"] || !declared["lighting-control:write"] {
		t.Fatal("lighting resource lacks its coarse capability declarations")
	}

	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(accounts.Schema), database.SchemaProviderFunc(authorization.Schema)); err != nil {
		t.Fatal(err)
	}
	clock := schedule.System()
	repo := accounts.NewSQLiteRepository(db, clock)
	hash, err := accounts.HashPassword("disposable-fixture-password")
	if err != nil {
		t.Fatal(err)
	}
	account, err := repo.Create(ctx, accounts.CreateInput{RealmID: "default", Email: "lighting-fixture@example.test", PasswordHash: hash, Roles: []string{"user"}})
	if err != nil {
		t.Fatal(err)
	}
	scopes := repo.(authorization.ScopeStore)
	if _, err := scopes.GrantScope(ctx, account.ID, "lighting-control:write"); err != nil {
		t.Fatal(err)
	}
	keys, err := authcrypto.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour, Now: clock.Now})
	service := accounts.NewService(accounts.ServiceConfig{Repo: repo, Signer: signer, Sessions: sessions.NewManager(redisstate.NewMemory(), clock.Now), Authorization: authorization.NewService(scopes, nil), Clock: clock, ResourceAudiences: audiences})
	result, err := service.Login(ctx, accounts.LoginParams{Email: account.Email, Password: "disposable-fixture-password", Resource: "lighting-control"}, accounts.RequestMeta{})
	if err != nil {
		t.Fatalf("managed resource login refused: %v", err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(keys.JWKS()) }))
	defer jwks.Close()
	verifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: lighting.Authentication.Audience, JWKSURL: jwks.URL})
	principal, err := verifier.Verify(ctx, result.AccessToken)
	if err != nil {
		t.Fatalf("issuer login did not verify at relying audience: %v", err)
	}
	if !principal.IsHuman() || !principal.IsVerified() || !scopecatalog.Resolve(principal.Scopes, "lighting-control:write") || !reflect.DeepEqual(principal.Roles, []string{"user"}) {
		t.Fatal("login changed fixture role/capability authority")
	}
	held, err := scopes.ListScopes(ctx, account.ID)
	if err != nil || !reflect.DeepEqual(held, []string{"lighting-control:write"}) {
		t.Fatal("resource registration changed existing account scope assignments")
	}
	// Registering an audience must not turn a normal signed-in account into
	// a lighting writer. The issuer preserves existing scope assignments.
	reader, err := repo.Create(ctx, accounts.CreateInput{RealmID: "default", Email: "reader-fixture@example.test", PasswordHash: hash, Roles: []string{"user"}})
	if err != nil {
		t.Fatal(err)
	}
	readLogin, err := service.Login(ctx, accounts.LoginParams{Email: reader.Email, Password: "disposable-fixture-password", Resource: "lighting-control"}, accounts.RequestMeta{})
	if err != nil {
		t.Fatal(err)
	}
	readPrincipal, err := verifier.Verify(ctx, readLogin.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if scopecatalog.Resolve(readPrincipal.Scopes, "lighting-control:write") || !reflect.DeepEqual(readPrincipal.Roles, []string{"user"}) {
		t.Fatal("resource registration assigned authority to an unscoped account")
	}
	if _, err := service.Login(ctx, accounts.LoginParams{Email: account.Email, Password: "disposable-fixture-password", Resource: "unregistered-fixture"}, accounts.RequestMeta{}); err == nil {
		t.Fatal("unknown resource bypassed the explicit issuer allowlist")
	}
}
