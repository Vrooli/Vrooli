package main

import (
	"testing"

	"landing-page-business-suite-api/internal/emaildelivery"
)

func TestProviderMailDNSRequirementsDecodeRegistryShape(t *testing.T) {
	requirements := providerMailDNSRequirements(emaildelivery.Provider{
		ID: "sendgrid",
		DNSRequirements: map[string]any{
			"spf_mechanism": "include:sendgrid.net",
			"dkim":          []any{map[string]any{"selector": "s1", "type": "CNAME", "expected": "sendgrid.net"}},
		},
	})
	if len(requirements) != 1 || requirements[0].SPFMechanism != "include:sendgrid.net" || len(requirements[0].DKIM) != 1 || requirements[0].DKIM[0].Selector != "s1" {
		t.Fatalf("requirements = %#v", requirements)
	}
}

func TestClassifyEmailTransportErrorPreservesAmbiguousHandoffs(t *testing.T) {
	if outcome := classifyEmailTransportError(assertError("context deadline exceeded")); outcome.Kind != emaildelivery.OutcomeUnknown {
		t.Fatalf("timeout outcome = %#v, want unknown", outcome)
	}
	if outcome := classifyEmailTransportError(assertError("HTTP 400 invalid sender")); outcome.Kind != emaildelivery.OutcomePermanent {
		t.Fatalf("sender outcome = %#v, want permanent", outcome)
	}
}

type assertedError string

func (e assertedError) Error() string { return string(e) }

func assertError(value string) error { return assertedError(value) }
