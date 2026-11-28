package apiauth_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

// Exercise the actual production provenance/authentication middleware and public
// Connect handlers. A verified current baseline is enough: no scope, operation
// proof, approval or grant is part of this ordinary lighting request.
func TestEveryLightingMutationAcceptsVerifiedUnscopedProjectAgent(t *testing.T) {
	for _, operation := range publicMutations {
		t.Run(operation.method, func(t *testing.T) {
			verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
				if token != "private-project-identity" {
					t.Fatalf("unexpected credential: %q", token)
				}
				return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "current-project-run", Subject: "operator", ExpiresAt: time.Now().Add(time.Hour).Unix()}}, nil
			})
			f := newFixtureWrapped(t, identity.Principal{}, false, false, provenance.Middleware(verifier))
			f.headers = http.Header{cliutil.HeaderAgentIdentityToken: []string{"private-project-identity"}}
			if operation.method == "ConfigureZone" {
				f.backend.configurable = true
			}
			if err := operation.call(context.Background(), f); err != nil {
				t.Fatalf("ordinary verified project agent required consent/scopes: %v", err)
			}
			if operation.method == "ConfigureZone" && (f.backend.count != 1 || !f.backend.emittedOff) {
				t.Fatal("ordinary project-agent count configuration did not preserve durable Off")
			}
		})
	}
}

func TestEveryLightingMutationRejectsInvalidOrClaimedProjectIdentityWithoutEffects(t *testing.T) {
	for _, mode := range []string{"anonymous", "invalid", "expired", "missing expiry", "foreign issuer", "source label without provenance"} {
		for _, operation := range publicMutations {
			t.Run(mode+"/"+operation.method, func(t *testing.T) {
				verifier := provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
					expiry := time.Now().Add(time.Hour).Unix()
					if mode == "expired" {
						expiry = time.Now().Add(-time.Minute).Unix()
					}
					if mode == "missing expiry" {
						expiry = 0
					}
					return &cliutil.VerifyResult{Valid: mode != "invalid", Claims: &cliutil.VerifiedClaims{RunID: "current-project-run", Subject: "operator", ExpiresAt: expiry}}, nil
				})
				var f *fixture
				if mode == "foreign issuer" || mode == "source label without provenance" {
					source := identity.SourceScenarioAuthenticator
					if mode == "source label without provenance" {
						source = identity.SourceAgentProvenance
					}
					p := identity.Principal{Kind: identity.ActorAgent, Source: source, Subject: "current-project-run", Owner: "operator", Verified: true, ExpiresAt: time.Now().Add(time.Hour), Scopes: []string{"lighting-control:write"}}
					f = newFixture(t, p, true, false)
				} else {
					f = newFixtureWrapped(t, identity.Principal{}, false, false, provenance.Middleware(verifier))
					if mode != "anonymous" {
						f.headers = http.Header{cliutil.HeaderAgentIdentityToken: []string{"private-project-identity"}}
					}
				}
				before := databaseSnapshot(t, f.db)
				calls := f.backend.calls.Load()
				if err := operation.call(context.Background(), f); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("untrusted identity admitted: %v", err)
				}
				if databaseSnapshot(t, f.db) != before || f.backend.calls.Load() != calls {
					t.Fatal("refusal changed stored state or backend")
				}
			})
		}
	}
}
