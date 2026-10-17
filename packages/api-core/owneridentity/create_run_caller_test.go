package owneridentity

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestCanonicalCloudflareCallerUsesVerifiedContextAndPreservesOnlyImmediateProof(t *testing.T) {
	now := time.Now()
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: identity.SourceCloudflareAccess, Scopes: []string{"agent-manager:write"}, ExpiresAt: now.Add(time.Hour)}
	ctx, err := AuthorizeCreateRunCaller(identity.WithPrincipal(context.Background(), p), http.Header{"Cf-Access-Jwt-Assertion": []string{"disposable-verified-assertion"}}, nil, now, true)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := CreateRunRequestHeaders(ctx, now)
	if err != nil || proof.Get("Cf-Access-Jwt-Assertion") == "" || proof.Get("Authorization") != "" {
		t.Fatal("canonical provider proof became a legacy bearer")
	}
	if _, err := CreateRunAuthorization(ctx, now); err == nil {
		t.Fatal("Cloudflare assertion mislabeled as a native human bearer")
	}
	if got := RedactCreateRunCaller(ctx, "disposable-verified-assertion"); got != "[redacted]" {
		t.Fatal("provider proof was not redacted")
	}
	p.Kind = identity.ActorAgent
	if _, err := AuthorizeCreateRunCaller(identity.WithPrincipal(context.Background(), p), proof, nil, now, true); err == nil {
		t.Fatal("agent canonical context became human authority")
	}
}

type auth01Validator struct {
	result Identity
	err    error
	calls  int
}

func (v *auth01Validator) Validate(_ context.Context, token string) (Identity, error) {
	v.calls++
	if token != "fixture-owner" {
		return Identity{}, errors.New("invalid")
	}
	return v.result, v.err
}

func TestAuth01CreateCallerQualification(t *testing.T) {
	now := time.Now()
	good := Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: now.Add(time.Hour)}
	for _, scope := range []string{"agent-manager:write", "agent-manager:*", "*:write", "*"} {
		p := good
		p.Scopes = []string{scope}
		v := &auth01Validator{result: p}
		ctx, err := AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture-owner"}}, v, now, true)
		if err != nil {
			t.Fatal(err)
		}
		auth, err := CreateRunAuthorization(ctx, now)
		if err != nil || auth != "Bearer fixture-owner" {
			t.Fatal("proof lost")
		}
		if RequireCreateRunCaller(ctx, good.ExpiresAt, true) == nil {
			t.Fatal("expiry extended")
		}
		if RedactCreateRunCaller(ctx, "fixture-owner") != "[redacted]" {
			t.Fatal("secret leaked")
		}
	}
	for _, h := range []http.Header{{}, {"Authorization": []string{"fixture-owner"}}, {"Authorization": []string{"Bearer fixture-owner", ""}}, {"Authorization": []string{"Bearer fixture-owner"}, "X-Agent-Identity-Token": []string{""}}, {"Authorization": []string{"Bearer fixture owner"}}} {
		v := &auth01Validator{result: good}
		ctx, err := AuthorizeCreateRunCaller(context.Background(), h, v, now, true)
		if err == nil || ctx != nil || v.calls != 0 {
			t.Fatal("invalid offered channel reached verifier")
		}
	}
	for _, p := range []Identity{{Subject: "", Scopes: good.Scopes, ExpiresAt: good.ExpiresAt}, {Subject: "fixture", Scopes: []string{"agent-manager:read"}, ExpiresAt: good.ExpiresAt}, {Subject: "fixture", Scopes: good.Scopes, ExpiresAt: now}} {
		ctx, err := AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture-owner"}}, &auth01Validator{result: p}, now, true)
		if err == nil || ctx != nil {
			t.Fatal("invalid human claims accepted")
		}
	}
	if _, err := AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture-owner"}}, &auth01Validator{result: good, err: errors.New("fixture-owner")}, now, true); err != ErrCreateRunCaller {
		t.Fatal("verifier failure exposed")
	}
	if RequireCreateRunCaller(context.Background(), now, true) == nil {
		t.Fatal("anonymous accepted")
	}
}

func TestAuth01ImmediateWorkerRetainsOriginalCallerCeiling(t *testing.T) {
	now := time.Now()
	ctx, err := AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture-owner"}}, &auth01Validator{result: Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: now.Add(time.Minute)}}, now, true)
	if err != nil {
		t.Fatal(err)
	}
	target, cancel := context.WithCancel(context.Background())
	worker, err := CarryCreateRunCaller(ctx, target, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateRunAuthorization(worker, now); err != nil {
		t.Fatal("worker proof lost")
	}
	if RequireCreateRunCaller(worker, now.Add(time.Minute), true) == nil {
		t.Fatal("worker expiry extended")
	}
	cancel()
	if RequireCreateRunCaller(worker, now, true) == nil {
		t.Fatal("shutdown cancellation ignored")
	}
	if _, err := CarryCreateRunCaller(context.Background(), context.Background(), now, true); err == nil {
		t.Fatal("anonymous worker authority fabricated")
	}
}

func TestAuth01SwitchDefaultsOffAndKeepsVerifiedAttribution(t *testing.T) {
	now := time.Now()
	good := Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: now.Add(time.Hour)}
	t.Setenv(CreateRunCallerEnforceEnv, "")
	if CreateRunCallerEnforced() {
		t.Fatal("enforcement must default off")
	}
	for _, h := range []http.Header{{}, {"X-Agent-Identity-Token": []string{"agent-run"}}, {"Authorization": []string{"Bearer wrong"}}} {
		ctx, err := AuthorizeCreateRunCaller(context.Background(), h, &auth01Validator{result: good}, now, CreateRunCallerEnforced())
		if err != nil || ctx == nil {
			t.Fatal("unattended or agent caller refused while off")
		}
		if RequireCreateRunCaller(ctx, now, CreateRunCallerEnforced()) != nil {
			t.Fatal("absent caller held while off")
		}
		if _, err := CreateRunAuthorization(ctx, now); err == nil {
			t.Fatal("unverified proof attributed")
		}
		if worker, err := CarryCreateRunCaller(ctx, context.Background(), now, CreateRunCallerEnforced()); err != nil || worker == nil {
			t.Fatal("absent caller not carried while off")
		}
	}
	ctx, err := AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture-owner"}, "X-Agent-Identity-Token": []string{"agent-run"}}, &auth01Validator{result: good}, now, CreateRunCallerEnforced())
	if err != nil {
		t.Fatal(err)
	}
	worker, err := CarryCreateRunCaller(ctx, context.Background(), now, CreateRunCallerEnforced())
	if err != nil {
		t.Fatal(err)
	}
	if auth, err := CreateRunAuthorization(worker, now); err != nil || auth != "Bearer fixture-owner" {
		t.Fatal("verified attribution lost while off")
	}

	t.Setenv(CreateRunCallerEnforceEnv, "true")
	if _, err := AuthorizeCreateRunCaller(context.Background(), http.Header{}, &auth01Validator{result: good}, now, CreateRunCallerEnforced()); err != ErrCreateRunCaller {
		t.Fatal("anonymous caller admitted while on")
	}
	if RequireCreateRunCaller(context.Background(), now, CreateRunCallerEnforced()) == nil {
		t.Fatal("anonymous caller not held while on")
	}
	if _, err := CarryCreateRunCaller(context.Background(), context.Background(), now, CreateRunCallerEnforced()); err == nil {
		t.Fatal("anonymous worker carried while on")
	}
}
