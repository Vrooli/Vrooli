package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/providerprobe"
)

func failingResult(provider string) providerprobe.Result {
	return providerprobe.Result{
		Provider:   provider,
		Capability: "customer sign-in email",
		Status:     providerprobe.StatusFail,
		Detail:     "the SMTP relay rejected the stored password: 535 Authentication failed",
		CheckedAt:  time.Now().UTC(),
	}
}

// Health must report a rejected credential. This is the state vrooli.com was
// in for hours while every deploy-time check passed.
func TestProviderCredentialCheckDegradesOnRejectedCredential(t *testing.T) {
	server := &Server{providerVerification: newProviderVerificationCache()}
	server.providerVerification.store(failingResult("smtp"))

	result := server.providerCredentialCheck("sign_in_smtp_credentials", "smtp").Check(context.Background())

	if result.Connected || result.Error == nil {
		t.Fatalf("a rejected credential must degrade health, got %+v", result)
	}
	if !strings.Contains(result.Error.Error(), "535") {
		t.Errorf("error = %v, want the provider's own rejection", result.Error)
	}
	if !strings.Contains(result.Error.Error(), "customer sign-in email") {
		t.Errorf("error = %v, want the capability named", result.Error)
	}
}

// Silence is required for the three non-actionable states, so that every
// warning this check produces is worth acting on.
func TestProviderCredentialCheckStaysSilentWhenNotActionable(t *testing.T) {
	cases := map[string]providerprobe.Result{
		"not configured": {Provider: "sendgrid", Status: providerprobe.StatusNotConfigured},
		"unreachable":    {Provider: "sendgrid", Status: providerprobe.StatusUnknown},
		"passing":        {Provider: "sendgrid", Status: providerprobe.StatusPass},
	}
	for name, stored := range cases {
		t.Run(name, func(t *testing.T) {
			server := &Server{providerVerification: newProviderVerificationCache()}
			server.providerVerification.store(stored)

			if result := server.providerCredentialCheck("sign_in_email_credentials", "sendgrid").Check(context.Background()); !result.Connected {
				t.Errorf("%s must not degrade health: %v", name, result.Error)
			}
		})
	}
}

// Before the first probe completes, health must not claim anything.
func TestProviderCredentialCheckSilentBeforeTheFirstProbe(t *testing.T) {
	server := &Server{providerVerification: newProviderVerificationCache()}

	if result := server.providerCredentialCheck("payments_credentials", "stripe").Check(context.Background()); !result.Connected {
		t.Errorf("an unprobed provider must not degrade health: %v", result.Error)
	}
}

func TestProviderVerificationCacheKeepsTheLatestVerdictPerProvider(t *testing.T) {
	cache := newProviderVerificationCache()
	cache.store(providerprobe.Result{Provider: "smtp", Status: providerprobe.StatusFail, Detail: "rejected"})
	cache.store(providerprobe.Result{Provider: "smtp", Status: providerprobe.StatusPass, Detail: "accepted"})
	cache.store(providerprobe.Result{Provider: "stripe", Status: providerprobe.StatusPass})

	smtp, ok := cache.get("smtp")
	if !ok || smtp.Status != providerprobe.StatusPass {
		t.Errorf("smtp = %+v, want the latest verdict", smtp)
	}
	if snapshot := cache.snapshot(); len(snapshot) != 2 {
		t.Errorf("snapshot = %d providers, want 2", len(snapshot))
	}
}

// A nil cache must not panic: health is registered before the first probe
// runs, and a partially constructed server must still answer.
func TestProviderVerificationCacheToleratesNil(t *testing.T) {
	var cache *providerVerificationCache
	cache.store(providerprobe.Result{Provider: "smtp"})
	if _, ok := cache.get("smtp"); ok {
		t.Error("a nil cache must report nothing")
	}
	if cache.snapshot() != nil {
		t.Error("a nil cache must snapshot to nil")
	}
}
