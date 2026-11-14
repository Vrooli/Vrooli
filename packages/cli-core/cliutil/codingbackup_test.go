package cliutil

import (
	"fmt"
	"testing"
	"time"
)

// SM8 is a product of owner facts, not five persisted decision strings.
// Each equivalence class is crossed with every harness, consent, mode and
// confirmation. Expected eligibility belongs to the fixtures, independently
// of Exhausted/Valid/SupportsModelSource and the production decision function.
func TestCodingBackupDecisionCompleteMatrix(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	limits := []struct {
		name      string
		change    func(*CodingLimit)
		exhausted bool
	}{
		{"fresh_provider", func(*CodingLimit) {}, true},
		{"fresh_usage", func(l *CodingLimit) { l.Kind = "quota_exhausted" }, true},
		{"five_minute_boundary", func(l *CodingLimit) { l.ObservedAt = now.Add(-5 * time.Minute) }, true},
		{"stale", func(l *CodingLimit) { l.ObservedAt = now.Add(-5*time.Minute - time.Nanosecond) }, false},
		{"future", func(l *CodingLimit) { l.ObservedAt = now.Add(time.Nanosecond) }, false},
		{"missing", func(l *CodingLimit) { l.ObservedAt = time.Time{} }, false},
		{"generic_rate_limit", func(l *CodingLimit) { l.Kind = "rate_limit_error" }, false},
		{"reset_now", func(l *CodingLimit) { l.ResetsAt = now }, false},
		{"reset_future", func(l *CodingLimit) { l.ResetsAt = now.Add(time.Second) }, true},
	}
	consents := []struct {
		name             string
		change           func(*CodingBackupConsent)
		valid, automatic bool
	}{
		{"absent", nil, false, false},
		{"manual", func(*CodingBackupConsent) {}, true, false},
		{"automatic", func(c *CodingBackupConsent) { c.Automatic = true }, true, true},
		{"expired", func(c *CodingBackupConsent) { c.ExpiresAt = now }, false, false},
		{"over_24h", func(c *CodingBackupConsent) { c.ExpiresAt = now.Add(24*time.Hour + time.Nanosecond) }, false, false},
		{"exact_24h", func(c *CodingBackupConsent) { c.ExpiresAt = now.Add(24 * time.Hour) }, true, false},
		{"zero_cap", func(c *CodingBackupConsent) { c.MaxCredits = 0 }, false, false},
		{"negative_cap", func(c *CodingBackupConsent) { c.MaxCredits = -1 }, false, false},
		{"no_account", func(c *CodingBackupConsent) { c.Account = "" }, false, false},
		{"no_model", func(c *CodingBackupConsent) { c.Model = "" }, false, false},
		{"wrong_harness", func(c *CodingBackupConsent) { c.Harness = "other" }, false, false},
	}
	count := 0
	for _, harness := range []string{"codex", "opencode", "claude", "unknown", ""} {
		for _, l := range limits {
			for _, c := range consents {
				for _, mode := range []string{"live", "new", "unattended", "unknown"} {
					for _, confirmed := range []bool{false, true} {
						t.Run(fmt.Sprintf("%s/%s/%s/%s/confirmed=%t", harness, l.name, c.name, mode, confirmed), func(t *testing.T) {
							limit := CodingLimit{Harness: harness, Kind: "provider_limit", ObservedAt: now}
							l.change(&limit)
							var consent *CodingBackupConsent
							if c.change != nil {
								consent = &CodingBackupConsent{Harness: harness, Account: "account", Model: "model", MaxCredits: 100, ExpiresAt: now.Add(time.Hour)}
								c.change(consent)
							}
							want := "subscription"
							if l.exhausted && harness != "" {
								want = "different_harness_required"
								if harness == "codex" || harness == "opencode" {
									want = "consent_required"
									if c.valid {
										switch mode {
										case "live":
											want = "restart_required"
											if confirmed {
												want = "credits"
											}
										case "new", "unattended":
											if c.automatic {
												want = "credits"
											}
										}
									}
								}
							}
							if got := CodingBackupDecision(now, limit, consent, mode, confirmed); got != want {
								t.Fatalf("decision=%s want=%s", got, want)
							}
						})
						count++
					}
				}
			}
		}
	}
	if count != 3960 {
		t.Fatalf("matrix coverage=%d want=3960", count)
	}
}

func TestCodingBackupDecisionLifecycleReplay(t *testing.T) {
	now := time.Now()
	limit := CodingLimit{Harness: "codex"}
	var consent *CodingBackupConsent
	for _, step := range []struct {
		name, mode, want string
		confirmed        bool
		apply            func()
	}{
		{"chosen_subscription", "new", "subscription", false, func() {}},
		{"fresh_limit", "live", "consent_required", false, func() { limit.Kind = "provider_limit"; limit.ObservedAt = now }},
		{"manual_grant", "live", "restart_required", false, func() {
			consent = &CodingBackupConsent{Harness: "codex", Account: "account", Model: "model", MaxCredits: 100, ExpiresAt: now.Add(time.Hour)}
		}},
		{"explicit_restart", "live", "credits", true, func() {}},
		{"manual_not_unattended", "unattended", "consent_required", false, func() {}},
		{"automatic_grant", "unattended", "credits", false, func() { consent.Automatic = true }},
		{"expired", "new", "consent_required", false, func() { consent.ExpiresAt = now }},
		{"revoked", "live", "consent_required", true, func() { consent = nil }},
		{"provider_recovered", "new", "subscription", false, func() { limit.ResetsAt = now }},
	} {
		step.apply()
		if got := CodingBackupDecision(now, limit, consent, step.mode, step.confirmed); got != step.want {
			t.Fatalf("%s: %s want %s", step.name, got, step.want)
		}
		t.Logf("%s -> %s", step.name, step.want)
	}
}
