package policygate

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestVerifiedOwnerProofPreservesVerifiedChannelWithoutWriteGate(t *testing.T) {
	for _, test := range []struct {
		name    string
		source  identity.AuthSource
		headers http.Header
		want    string
		key     string
	}{
		{"cookie", identity.SourceScenarioAuthenticator, http.Header{"Cookie": []string{"gct_access_token=owner-cookie; unrelated=private"}}, "Bearer owner-cookie", "Authorization"},
		{"bearer", identity.SourceScenarioAuthenticator, http.Header{"Authorization": []string{"Bearer owner-bearer"}, "Cookie": []string{"gct_access_token=ignored"}}, "Bearer owner-bearer", "Authorization"},
		{"cloudflare", identity.SourceCloudflareAccess, http.Header{"Cf-Access-Jwt-Assertion": []string{"verified-access"}, "Authorization": []string{"Bearer unrelated"}}, "verified-access", "Cf-Access-Jwt-Assertion"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: test.source, ExpiresAt: time.Now().Add(time.Hour), Scopes: []string{"agent-manager:read"}}
			ctx := WithVerifiedOwnerProof(identity.WithPrincipal(context.Background(), p), test.headers)
			proof := VerifiedOwnerHeaders(ctx)
			if len(proof) != 1 || proof.Get(test.key) != test.want || proof.Get("Cookie") != "" {
				t.Fatalf("verified channel not preserved/narrowed: %v", proof)
			}
			proof.Set(test.key, "changed")
			if VerifiedOwnerHeaders(ctx).Get(test.key) != test.want {
				t.Fatal("caller modified retained proof")
			}
			p.Subject = "other"
			if len(VerifiedOwnerHeaders(identity.WithPrincipal(ctx, p))) != 0 {
				t.Fatal("proof survived identity replacement")
			}
		})
	}
}

func TestVerifiedOwnerProofRejectsUnverifiedExpiredAgentAndAmbiguousChannels(t *testing.T) {
	base := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour)}
	for _, name := range []string{"unverified", "expired", "agent", "agent header", "duplicate bearer", "personal local", "missing expiry"} {
		t.Run(name, func(t *testing.T) {
			p := base
			headers := http.Header{"Authorization": []string{"Bearer fixture"}}
			switch name {
			case "unverified":
				p.Verified = false
			case "expired":
				p.ExpiresAt = time.Now().Add(-time.Second)
			case "agent":
				p.Kind = identity.ActorAgent
			case "agent header":
				headers.Set(cliutil.HeaderAgentIdentityToken, "agent-fixture")
			case "duplicate bearer":
				headers.Add("Authorization", "Bearer second")
			case "personal local":
				p.Source = identity.SourcePersonalLocal
			case "missing expiry":
				p.ExpiresAt = time.Time{}
			}
			ctx := WithVerifiedOwnerProof(identity.WithPrincipal(context.Background(), p), headers)
			if len(VerifiedOwnerHeaders(ctx)) != 0 {
				t.Fatal("unverified/ambiguous proof forwarded")
			}
		})
	}
}

func TestVerifiedOwnerExpirySurvivesSharedAuthenticatorProjection(t *testing.T) {
	expires := time.Now().Add(time.Hour).UTC()
	provider := NewSharedAuthenticatorProvider(PrincipalVerifierFunc(func(context.Context, string) (Principal, error) {
		return Principal{Kind: cliutil.CallerKindHuman, Subject: "owner", Verified: true, ExpiresAt: expires}, nil
	}))
	req, _ := http.NewRequest(http.MethodGet, "http://fixture", nil)
	req.Header.Set("Authorization", "Bearer fixture")
	p, err := provider.VerifyRequest(context.Background(), req)
	if err != nil || !p.ExpiresAt.Equal(expires) {
		t.Fatalf("signed lifetime discarded: %+v %v", p, err)
	}
	legacy, ok := PrincipalFromContext(identity.WithPrincipal(context.Background(), p))
	if !ok || !legacy.ExpiresAt.Equal(expires) {
		t.Fatal("legacy projection discarded signed lifetime")
	}
}

func TestVerifiedOwnerProofPreservesIndependentlyVerifiedProviderChannels(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: identity.SourceCloudflareAccess, Sources: []identity.AuthSource{identity.SourceCloudflareAccess, identity.SourceScenarioAuthenticator}, Scopes: []string{"git-control-tower:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)})
	ctx = WithVerifiedOwnerProof(ctx, http.Header{"Authorization": []string{"Bearer verified-scenario"}, "Cf-Access-Jwt-Assertion": []string{"verified-cloudflare"}})
	proof := VerifiedOwnerHeaders(ctx)
	if len(proof) != 2 || proof.Get("Authorization") != "Bearer verified-scenario" || proof.Get("Cf-Access-Jwt-Assertion") != "verified-cloudflare" {
		t.Fatalf("merged provider authority narrowed: %v", proof)
	}
}
