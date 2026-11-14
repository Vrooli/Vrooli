package policy

import (
	"strings"
	"testing"
	"time"
)

func TestVerifyProviderDecisionReceiptRequiresExactApprovedHumanDecision(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	digest := strings.Repeat("a", 64)
	expected := ProviderDecisionReceipt{Source: "ai-gateway", EventType: ProviderDecisionEventType, Requester: "event-1", CanonicalTermsSHA256: digest, Scope: ProviderDecisionScopeRule}
	actual := ProviderDecisionReceipt{
		Source: "ai-gateway", EventType: ProviderDecisionEventType, Requester: "event-1", EventID: "event-1",
		CanonicalTermsSHA256: digest, Scope: ProviderDecisionScopeRule, State: "answered", Answer: "approve",
		AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		CreatedAt: "2026-11-01T10:00:00Z", AnsweredAt: "2026-11-01T11:00:00Z", Deadline: "2026-11-02T10:00:00Z",
	}
	if err := VerifyProviderDecisionReceipt(expected, actual, now); err != nil {
		t.Fatalf("valid receipt refused: %v", err)
	}

	deadline := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	for name, verificationTime := range map[string]time.Time{
		"before deadline": deadline.Add(-time.Nanosecond),
		"at deadline":     deadline,
	} {
		t.Run(name, func(t *testing.T) {
			if err := VerifyProviderDecisionReceipt(expected, actual, verificationTime); err != nil {
				t.Fatalf("receipt refused at valid deadline boundary: %v", err)
			}
		})
	}
	t.Run("after deadline", func(t *testing.T) {
		if err := VerifyProviderDecisionReceipt(expected, actual, deadline.Add(time.Nanosecond)); err == nil {
			t.Fatal("expired receipt accepted")
		}
	})

	cases := map[string]func(*ProviderDecisionReceipt){
		"wrong event":          func(r *ProviderDecisionReceipt) { r.EventID = "event-2" },
		"wrong digest":         func(r *ProviderDecisionReceipt) { r.CanonicalTermsSHA256 = strings.Repeat("b", 64) },
		"wrong scope":          func(r *ProviderDecisionReceipt) { r.Scope = ProviderDecisionScopeAgreement },
		"rejected":             func(r *ProviderDecisionReceipt) { r.Answer = "reject" },
		"defaulted":            func(r *ProviderDecisionReceipt) { r.State, r.Answer = "defaulted", "approve" },
		"recipient fallback":   func(r *ProviderDecisionReceipt) { r.AnsweredByIdentityKind = "unknown" },
		"missing human source": func(r *ProviderDecisionReceipt) { r.AnsweredByIdentitySource = "" },
		"outside window":       func(r *ProviderDecisionReceipt) { r.AnsweredAt = "2026-11-03T11:00:00Z" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := actual
			mutate(&changed)
			if err := VerifyProviderDecisionReceipt(expected, changed, now); err == nil {
				t.Fatal("invalid receipt accepted")
			}
		})
	}
}
