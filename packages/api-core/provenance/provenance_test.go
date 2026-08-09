package provenance

import (
	"errors"
	"testing"
)

func TestRequireScopeRejectsUnverifiedAndMissingScopes(t *testing.T) {
	for _, p := range []Provenance{
		{},
		{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1", Scopes: []string{"vrooli-bridge:read"}},
	} {
		if err := p.RequireScope("vrooli-bridge:write"); err == nil {
			t.Fatal("expected scope refusal")
		} else {
			var denied ScopeDeniedError
			if !errors.As(err, &denied) {
				t.Fatalf("error = %T, want ScopeDeniedError", err)
			}
		}
	}
}

func TestRequireScopeAcceptsVerifiedWildcard(t *testing.T) {
	p := Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1", Scopes: []string{"vrooli-bridge:*"}}
	if err := p.RequireScope("vrooli-bridge:dispatch"); err != nil {
		t.Fatal(err)
	}
}
