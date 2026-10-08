package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// TestCredentialStatusNamesAnUnusableStoreInsteadOfTheTransport pins what
// minimouse showed: with no credential store, the client refused to exist, and
// every credential read "credential transport is unavailable". Status is
// metadata-only, so readiness now asks the authority directly and the store's
// own condition and fix reach the operator.
func TestCredentialStatusNamesAnUnusableStoreInsteadOfTheTransport(t *testing.T) {
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) {
		return credentialauthority.Unavailable("no credential store on this host; run `vrooli credentials store init`")
	}
	t.Cleanup(func() { onboardingAuthority = previous })

	output, err := onboardingStatusJSON(context.Background(), "vrooli/openrouter", "api-key")
	if err != nil {
		t.Fatalf("status error = %v, want the store's condition", err)
	}
	var status struct {
		Configured     bool   `json:"configured"`
		ProviderState  string `json:"provider_state"`
		ProviderDetail string `json:"provider_detail"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		t.Fatalf("status is not JSON: %v (%s)", err, output)
	}
	if status.Configured || status.ProviderState == "" || status.ProviderState == "available" {
		t.Fatalf("status = %+v, want an unusable provider state", status)
	}
	if !strings.Contains(status.ProviderDetail, "vrooli credentials store init") {
		t.Fatalf("provider detail = %q, want the store's own remediation", status.ProviderDetail)
	}
}
