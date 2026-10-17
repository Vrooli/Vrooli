package accounts

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
)

type memoryServiceCustody struct{ values map[string]string }

func (m *memoryServiceCustody) key(name, resource string) string { return name + "/" + resource }

func (m *memoryServiceCustody) Put(name, resource, credential string) error {
	m.values[m.key(name, resource)] = credential
	return nil
}

func (m *memoryServiceCustody) Resolve(name, resource string) (string, error) {
	value, ok := m.values[m.key(name, resource)]
	if !ok {
		return "", errors.New("unconfigured")
	}
	return value, nil
}

func (m *memoryServiceCustody) Address(name, resource string) (string, string) {
	return "scenario-authenticator/service-principal/" + name + "/" + resource, "credential"
}

type servicePrincipalFixture struct {
	service *Service
	signer  *authcrypto.Signer
	custody *memoryServiceCustody
	admin   string
	issuer  string
	member  string
}

// Every credential here exists only in a disposable fixture. Never print
// tokens or credentials on failure.
func newServicePrincipalFixture(t *testing.T) servicePrincipalFixture {
	t.Helper()
	ctx := context.Background()
	clock := schedule.NewFake(time.Now().UTC().Truncate(time.Second))
	repo, _ := newRepo(t)
	bindings := repo.(MachineBindingStore)
	link := func(email, principal string, roles []string) string {
		acc, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: email, PasswordHash: "unused-fixture", Roles: roles})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = bindings.LinkMachineBinding(ctx, MachineBinding{MachineID: "disposable-host", LocalPrincipal: principal, AccountID: acc.ID, RealmID: "default", IsDefault: true, LinkedAt: clock.Now()}); err != nil {
			t.Fatal(err)
		}
		return acc.ID
	}
	link("admin@example.test", "unix:1000", []string{"admin"})
	issuerID := link("operator@example.test", "unix:1001", []string{"user"})
	link("member@example.test", "unix:1002", []string{"user"})
	keys, err := authcrypto.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour, Now: clock.Now})
	custody := &memoryServiceCustody{values: map[string]string{}}
	profiles, err := ParseServicePrincipalProfiles(`[{"name":"agent-manager","resource":"web-console","scopes":["web-console:write"],"issuers":["` + issuerID + `"]}]`)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(ServiceConfig{
		Repo: repo, Signer: signer, Sessions: sessions.NewManager(redisstate.NewMemory(), clock.Now), MachineBindings: bindings,
		Authorization: authorization.NewService(repo.(authorization.ScopeStore), nil), Clock: clock,
		ResourceAudiences: map[string]string{"web-console": "scenario-authenticator:web-console"},
		ServicePrincipals: profiles, ServiceCustody: custody,
	})
	tokenFor := func(principal string) string {
		res, err := svc.ExchangeMachinePrincipal(ctx, "disposable-host", principal, "", RequestMeta{})
		if err != nil {
			t.Fatal("fixture exchange failed")
		}
		return res.AccessToken
	}
	return servicePrincipalFixture{
		service: svc, signer: signer, custody: custody,
		admin: tokenFor("unix:1000"), issuer: tokenFor("unix:1001"), member: tokenFor("unix:1002"),
	}
}

func TestServicePrincipalIssueAndExchange(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)

	if _, err := f.service.IssueServicePrincipal(ctx, f.member, "agent-manager", RequestMeta{}); !errors.Is(err, ErrServicePrincipalForbidden) {
		t.Fatalf("unlisted human issuance: got %v", err)
	}
	if _, err := f.service.IssueServicePrincipal(ctx, "", "agent-manager", RequestMeta{}); !errors.Is(err, ErrServicePrincipalForbidden) {
		t.Fatalf("anonymous issuance: got %v", err)
	}
	if _, err := f.service.IssueServicePrincipal(ctx, f.admin, "swarm-manager", RequestMeta{}); !errors.Is(err, ErrServicePrincipalUnknown) {
		t.Fatalf("unconfigured principal: got %v", err)
	}
	if len(f.custody.values) != 0 {
		t.Fatal("refused issuance stored a credential")
	}

	if _, err := f.service.IssueServicePrincipal(ctx, f.issuer, "swarm-manager", RequestMeta{}); !errors.Is(err, ErrServicePrincipalForbidden) {
		t.Fatalf("issuer reached an unconfigured principal: %v", err)
	}
	if _, err := f.service.IssueServicePrincipal(ctx, f.admin, "agent-manager", RequestMeta{}); err != nil {
		t.Fatalf("administrator issuance: %v", err)
	}
	issued, err := f.service.IssueServicePrincipal(ctx, f.issuer, "agent-manager", RequestMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if issued.Audience != "scenario-authenticator:web-console" || !slices.Equal(issued.Scopes, []string{"web-console:write"}) ||
		issued.CredentialIdentity != "scenario-authenticator/service-principal/agent-manager/web-console" || issued.CredentialField != "credential" {
		t.Fatalf("unexpected issue metadata: %+v", issued)
	}
	credential := f.custody.values["agent-manager/web-console"]
	if !strings.HasPrefix(credential, serviceCredentialPrefix+"agent-manager.") || len(credential) < 50 {
		t.Fatal("credential not stored in custody with the expected format")
	}

	access, err := f.service.ExchangeServiceCredential(ctx, credential, RequestMeta{})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := f.signer.Validate(access.AccessToken, "scenario-authenticator:web-console")
	if err != nil {
		t.Fatal("service token failed verification at its audience")
	}
	if claims.ActorKind != ServiceActorKind || claims.Subject != "agent-manager" || claims.UserID != "agent-manager" ||
		!slices.Equal(claims.Scopes, []string{"web-console:write"}) || claims.SessionID != "" || claims.Email != "" || len(claims.Roles) != 0 {
		t.Fatal("service token claims are not the narrow service shape")
	}
	if _, err := f.signer.Validate(access.AccessToken, "scenario-authenticator:default"); err == nil {
		t.Fatal("service token verified at the default realm audience")
	}

	// A service token is never account authority at the authenticator.
	if _, ok, _ := f.service.Validate(ctx, access.AccessToken); ok {
		t.Fatal("service token validated as an account token")
	}
	if _, err := f.service.IssueServicePrincipal(ctx, access.AccessToken, "agent-manager", RequestMeta{}); !errors.Is(err, ErrServicePrincipalForbidden) {
		t.Fatalf("service token issued a principal: %v", err)
	}

	for _, bad := range []string{"", "garbage", serviceCredentialPrefix + "agent-manager.wrong", serviceCredentialPrefix + "swarm-manager.x"} {
		if _, err := f.service.ExchangeServiceCredential(ctx, bad, RequestMeta{}); !errors.Is(err, ErrServiceCredentialRejected) {
			t.Fatalf("bad credential accepted (%d chars): %v", len(bad), err)
		}
	}

	// Rotation invalidates the previous credential immediately.
	if _, err := f.service.IssueServicePrincipal(ctx, f.issuer, "agent-manager", RequestMeta{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.ExchangeServiceCredential(ctx, credential, RequestMeta{}); !errors.Is(err, ErrServiceCredentialRejected) {
		t.Fatal("rotated credential still exchanged")
	}
	// Deleting the custody entry revokes the principal.
	rotated := f.custody.values["agent-manager/web-console"]
	delete(f.custody.values, "agent-manager/web-console")
	if _, err := f.service.ExchangeServiceCredential(ctx, rotated, RequestMeta{}); !errors.Is(err, ErrServiceCredentialRejected) {
		t.Fatal("revoked credential still exchanged")
	}
}

func TestParseServicePrincipalProfilesRefusesWideOrAmbiguousProfiles(t *testing.T) {
	if profiles, err := ParseServicePrincipalProfiles(""); err != nil || profiles != nil {
		t.Fatal("empty configuration must configure nothing")
	}
	for _, raw := range []string{
		`not json`,
		`[{"name":"agent-manager","resource":"web-console","scopes":["*"]}]`,
		`[{"name":"agent-manager","resource":"web-console","scopes":["web-console:*"]}]`,
		`[{"name":"agent-manager","resource":"","scopes":[]}]`,
		`[{"name":"Agent Manager","resource":"web-console","scopes":[]}]`,
		`[{"name":"agent-manager","resource":"web-console"},{"name":"agent-manager","resource":"other"}]`,
		`[{"name":"agent-manager","resource":"web-console","issuers":[" "]}]`,
	} {
		if _, err := ParseServicePrincipalProfiles(raw); err == nil {
			t.Fatalf("accepted invalid profile configuration %q", raw)
		}
	}
}
