package main

import (
	"testing"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// Web Console must verify scenario-authenticator tokens for its own resource
// audience so Agent Manager's service principal can pin stable-create owners.
func TestManagedAuthenticationIncludesScenarioAuthenticator(t *testing.T) {
	cfg := pointerToManagedAuthentication(authn.Config{})
	found := false
	for _, provider := range cfg.Providers {
		if provider != nil && provider.Source() == identity.SourceScenarioAuthenticator {
			found = true
		}
	}
	if !found {
		t.Fatal("web-console authentication chain has no scenario-authenticator provider")
	}
}
