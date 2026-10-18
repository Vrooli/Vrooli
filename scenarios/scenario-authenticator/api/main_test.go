package main

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"scenario-authenticator/internal/realm"

	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/trustposture"
)

type fixedCloudflareProvider struct {
	principal coreidentity.Principal
}

func (fixedCloudflareProvider) Source() coreidentity.AuthSource {
	return coreidentity.SourceCloudflareAccess
}

func (p fixedCloudflareProvider) VerifyRequest(_ context.Context, req *http.Request) (coreidentity.Principal, error) {
	if req == nil || req.Header.Get("Cf-Access-Jwt-Assertion") != "verified-assertion" {
		return coreidentity.Principal{}, coreidentity.NewFailure(coreidentity.FailureInvalid, p.Source())
	}
	return p.principal, nil
}

func TestCloudflareOwnerAuthUsesTargetedSharedBindingsOnly(t *testing.T) {
	provider := fixedCloudflareProvider{principal: coreidentity.Principal{
		Kind: coreidentity.ActorHuman, Subject: "cf-subject", Email: "operator@example.test",
		Verified: true, Source: coreidentity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour),
	}}
	rawBindings := `[
		{"source":"cloudflare_access","subject":"cf-subject","canonical_subject":"owner-account","scopes":["agent-manager:read","agent-manager:write"],"resources":["agent-manager"]},
		{"source":"cloudflare_access","subject":"cf-subject","canonical_subject":"wrong-resource","scopes":["prompt-manager:write"],"resources":["prompt-manager"]},
		{"source":"scenario_authenticator","subject":"cf-subject","canonical_subject":"wrong-source","scopes":["agent-manager:destructive"],"resources":["agent-manager"]}
	]`
	env := map[string]string{
		"VROOLI_AUTH_PROVIDERS":      "scenario_authenticator",
		"VROOLI_AUTH_OWNER_BINDINGS": rawBindings,
		"VROOLI_SCENARIO":            "scenario-authenticator",
		"SCENARIO_NAME":              "scenario-authenticator",
	}
	config, err := cloudflareOwnerAuthConfig(provider, func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("load targeted owner bindings: %v", err)
	}
	if len(config.Providers) != 1 || config.Providers[0].Source() != coreidentity.SourceCloudflareAccess {
		t.Fatalf("providers = %v, want only Cloudflare", config.Providers)
	}
	if len(config.OwnerBindings) != 2 {
		t.Fatalf("agent-manager owner bindings = %+v, want only the two entries targeted to agent-manager", config.OwnerBindings)
	}
	for _, binding := range config.OwnerBindings {
		if binding.CanonicalSubject == "wrong-resource" {
			t.Fatalf("binding for another relying resource leaked into config: %+v", binding)
		}
	}

	verified, ok := (cloudflareHumanAssertion{config: config}).VerifyHuman(context.Background(), "verified-assertion")
	if !ok || !verified.IsHuman() || !verified.IsVerified() {
		t.Fatalf("verified principal = %+v, ok=%t", verified, ok)
	}
	if verified.Subject != "owner-account" || !reflect.DeepEqual(verified.Scopes, []string{"agent-manager:read", "agent-manager:write"}) {
		t.Fatalf("owner-bound principal = subject %q scopes %v", verified.Subject, verified.Scopes)
	}
	if !verified.ExpiresAt.Equal(provider.principal.ExpiresAt) || verified.Source != coreidentity.SourceCloudflareAccess {
		t.Fatalf("owner binding changed proof metadata: %+v", verified)
	}
}

func TestCloudflareOwnerAuthLeavesUnboundHumanUnscopedAndRefusesMalformedPolicy(t *testing.T) {
	provider := fixedCloudflareProvider{principal: coreidentity.Principal{
		Kind: coreidentity.ActorHuman, Subject: "unbound-subject", Verified: true,
		Source: coreidentity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour),
	}}
	config, err := cloudflareOwnerAuthConfig(provider, func(string) string { return "" })
	if err != nil {
		t.Fatalf("load absent optional owner bindings: %v", err)
	}
	principal, ok := (cloudflareHumanAssertion{config: config}).VerifyHuman(context.Background(), "verified-assertion")
	if !ok || principal.Subject != "unbound-subject" || len(principal.Scopes) != 0 {
		t.Fatalf("unbound identity = %+v, ok=%t; want provider identity with no scopes", principal, ok)
	}

	if _, err := cloudflareOwnerAuthConfig(provider, func(key string) string {
		if key == "VROOLI_AUTH_OWNER_BINDINGS" {
			return "{malformed"
		}
		return ""
	}); err == nil {
		t.Fatal("malformed explicitly configured owner bindings were accepted")
	}

	path := filepath.Join(t.TempDir(), "owner-bindings.json")
	if err := os.WriteFile(path, []byte("{malformed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cloudflareOwnerAuthConfig(provider, func(key string) string {
		if key == "VROOLI_AUTH_OWNER_BINDINGS_FILE" {
			return path
		}
		return ""
	}); err == nil {
		t.Fatal("malformed explicit owner-binding file was accepted")
	}
}

// TestBreakGlassIssuersConform proves the authenticated scenario issuer and
// the operator passphrase issuer produce credentials accepted by the same real
// verifier with the same audience, target, and scope semantics.
func TestBreakGlassIssuersConform(t *testing.T) {
	now := time.Unix(10_000, 0).UTC()
	target := "host-a"
	audience := realm.DefaultAudience
	scopes := []string{"vrooli:uninstall"}

	authDir := t.TempDir()
	authPaths := trustposture.KeyPaths{Dir: authDir, Private: filepath.Join(authDir, "private.key"), Public: filepath.Join(authDir, "public.key"), Metadata: filepath.Join(authDir, "provisioning.json")}
	authIssuer := breakGlassProvisioner{paths: authPaths, available: true, ttl: 10 * time.Minute, target: target}
	if err := authIssuer.Provision(context.TODO(), "operator-1", "default", scopes, now); err != nil {
		t.Fatal(err)
	}
	authToken, _, err := authIssuer.Issue(context.TODO(), "operator-1", "default", scopes, now)
	if err != nil {
		t.Fatal(err)
	}

	operatorDir := t.TempDir()
	operatorPaths := trustposture.KeyPaths{Dir: operatorDir, Private: filepath.Join(operatorDir, "private.key"), WrappedPrivate: filepath.Join(operatorDir, "private.key"), Public: filepath.Join(operatorDir, "public.key"), Metadata: filepath.Join(operatorDir, "provisioning.json")}
	if err := trustposture.ProvisionWrapped(operatorPaths, "correct horse", "operator-1", audience, target, scopes, now); err != nil {
		t.Fatal(err)
	}
	operatorToken, err := trustposture.IssueFromWrappedProvision(operatorPaths, "correct horse", audience, target, scopes, now, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	authPublic, err := os.ReadFile(authPaths.Public)
	if err != nil {
		t.Fatal(err)
	}
	authClaims, err := trustposture.Verify(ed25519.PublicKey(authPublic), authToken, audience, target, now)
	if err != nil {
		t.Fatal(err)
	}
	operatorPublic, err := os.ReadFile(operatorPaths.Public)
	if err != nil {
		t.Fatal(err)
	}
	operatorClaims, err := trustposture.Verify(ed25519.PublicKey(operatorPublic), operatorToken, audience, target, now)
	if err != nil {
		t.Fatal(err)
	}
	if authClaims.Subject != operatorClaims.Subject || authClaims.Audience != operatorClaims.Audience || authClaims.Target != operatorClaims.Target || len(authClaims.Scopes) != len(operatorClaims.Scopes) {
		t.Fatalf("issuer claim shape diverged: auth=%+v operator=%+v", authClaims, operatorClaims)
	}
}

func TestParseResourceAudiences(t *testing.T) {
	got := parseResourceAudiences("bridge=scenario-authenticator:bridge, device-sync-hub=scenario-authenticator:devices, deployment-manager=scenario-authenticator:deployment-manager,broken,=missing,bad=aud|pipe")
	if got["bridge"] != "scenario-authenticator:bridge" || got["device-sync-hub"] != "scenario-authenticator:devices" || got["deployment-manager"] != "scenario-authenticator:deployment-manager" {
		t.Fatalf("parsed resources = %#v", got)
	}
	if len(got) != 3 {
		t.Fatalf("parsed malformed resource entries: %#v", got)
	}
}
