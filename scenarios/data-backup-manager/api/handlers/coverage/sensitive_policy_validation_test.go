package coverage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActiveSensitivePolicyRequiresVerifiedConsentBeforeAutomation(t *testing.T) {
	validTime := "2026-11-23T05:00:00Z"
	cases := []struct {
		name string
		data string
	}{
		{name: "active bit without consent record", data: `{"active":true}`},
		{name: "unverified identity", data: `{"active":true,"grant_ask_id":"ask-1","granted_by":"agent-run","granted_by_identity_kind":"agent","granted_by_identity_source":"notification-hub","granted_at":"` + validTime + `","grant_basis":"option"}`},
		{name: "free-text grant without operator words", data: `{"active":true,"grant_ask_id":"ask-1","granted_by":"operator","granted_by_identity_kind":"verified_human","granted_by_identity_source":"notification-hub","granted_at":"` + validTime + `","grant_basis":"free_text"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flow, service, publisher, _ := sensitiveFlowFixture(t)
			if err := os.MkdirAll(filepath.Dir(flow.policy.Path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(flow.policy.Path, []byte(tc.data), 0o600); err != nil {
				t.Fatal(err)
			}

			_, _, err := flow.Request(context.Background())
			if err == nil || !strings.Contains(err.Error(), "standing policy") {
				t.Fatalf("Request with malformed active policy error=%v, want fail-closed policy error", err)
			}
			if len(service.registered) != 0 || len(publisher.events) != 0 {
				t.Fatalf("malformed policy caused effects: registered=%v events=%d", service.registered, len(publisher.events))
			}
			if _, err := flow.Sweep(context.Background()); err == nil {
				t.Fatal("Sweep accepted an active policy without valid human consent")
			}
			if len(service.registered) != 0 {
				t.Fatalf("Sweep registered targets without consent: %v", service.registered)
			}
		})
	}
}

func TestPolicyStoreDoesNotPersistActiveRuleWithoutConsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	store := &PolicyStore{Path: path}
	if _, err := store.Update(func(policy *StandingPolicy) error {
		policy.RevokedBy = "previous owner"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(policy *StandingPolicy) error {
		policy.Active = true
		return nil
	}); err == nil || !strings.Contains(err.Error(), "active rule is missing") {
		t.Fatalf("Update error=%v, want active-consent validation error", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("rejected update changed policy bytes: before=%s after=%s", before, after)
	}
}
