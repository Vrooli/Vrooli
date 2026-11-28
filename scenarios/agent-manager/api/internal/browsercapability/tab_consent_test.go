package browsercapability

import (
	"errors"
	"testing"
	"time"
)

func TestTabConsentAuthorityBindsOneSelectedTabToRunAndLease(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	authority := NewTabConsentAuthority()
	authority.now = func() time.Time { return now }
	profile := PermissionProfileSpec{ProfileID: "attached-profile", ProfileMode: SessionProfileUserAttached, Level: ProfileAct, AllowedOrigins: []string{"https://example.test"}, Control: true}
	issue := TabConsentIssue{RunID: "run-1", GrantReceipt: "grant-1", Profile: profile, SessionID: "session-1", LeaseID: "lease-1", OwnerID: "run:run-1", ProfileID: profile.ProfileID, TabID: "tab-5", URL: "https://example.test/form", PairingNonce: "pair-1", ExpiresAt: now.Add(5 * time.Minute)}
	receipt, err := authority.Issue(issue)
	if err != nil {
		t.Fatalf("issue selected-tab consent: %v", err)
	}
	if _, err := authority.Issue(issue); !errors.Is(err, ErrTabConsentDenied) {
		t.Fatalf("reused pairing nonce error = %v, want denied", err)
	}
	binding := TabConsentBinding{RunID: issue.RunID, GrantReceipt: issue.GrantReceipt, Profile: profile, SessionID: issue.SessionID, LeaseID: issue.LeaseID, OwnerID: issue.OwnerID, ProfileID: issue.ProfileID, TabID: issue.TabID, CurrentURL: issue.URL}
	if err := authority.Verify(receipt, binding); err != nil {
		t.Fatalf("verify selected-tab consent: %v", err)
	}
	for name, mutate := range map[string]func(*TabConsentBinding){
		"wrong run":     func(b *TabConsentBinding) { b.RunID = "run-2" },
		"wrong grant":   func(b *TabConsentBinding) { b.GrantReceipt = "grant-2" },
		"wrong session": func(b *TabConsentBinding) { b.SessionID = "session-2" },
		"wrong lease":   func(b *TabConsentBinding) { b.LeaseID = "lease-2" },
		"wrong owner":   func(b *TabConsentBinding) { b.OwnerID = "run:run-2" },
		"wrong profile": func(b *TabConsentBinding) { b.ProfileID = "profile-2" },
		"wrong tab":     func(b *TabConsentBinding) { b.TabID = "tab-6" },
		"out of scope":  func(b *TabConsentBinding) { b.CurrentURL = "https://other.test/form" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := binding
			mutate(&candidate)
			if err := authority.Verify(receipt, candidate); !errors.Is(err, ErrTabConsentDenied) {
				t.Fatalf("mismatched binding error = %v, want denied", err)
			}
		})
	}
	if err := authority.RevokeSession(receipt, "run-1", "session-1"); err != nil {
		t.Fatalf("revoke selected-tab receipt: %v", err)
	}
	if err := authority.Verify(receipt, binding); !errors.Is(err, ErrTabConsentDenied) {
		t.Fatalf("revoked receipt error = %v, want denied", err)
	}
}

func TestTabConsentAuthorityRejectsUnsupportedProfilesAndExpiry(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	base := TabConsentIssue{RunID: "run-1", GrantReceipt: "grant-1", Profile: PermissionProfileSpec{ProfileID: "attached-profile", ProfileMode: SessionProfileUserAttached, Level: ProfileAct, AllowedOrigins: []string{"https://example.test"}, Control: true}, SessionID: "session-1", LeaseID: "lease-1", OwnerID: "run:run-1", ProfileID: "attached-profile", TabID: "tab-5", URL: "https://example.test/form", PairingNonce: "pair-1", ExpiresAt: now.Add(time.Minute)}
	for name, mutate := range map[string]func(*TabConsentIssue){
		"fresh mode":      func(i *TabConsentIssue) { i.Profile.ProfileMode = SessionProfileFresh },
		"unrestricted":    func(i *TabConsentIssue) { i.Profile.Level = ProfileUnrestricted },
		"no tab access":   func(i *TabConsentIssue) { i.Profile.Control, i.Profile.View = false, false },
		"origin denied":   func(i *TabConsentIssue) { i.URL = "https://other.test/" },
		"too long":        func(i *TabConsentIssue) { i.ExpiresAt = now.Add(maxTabConsentLifetime + time.Second) },
		"already expired": func(i *TabConsentIssue) { i.ExpiresAt = now },
	} {
		t.Run(name, func(t *testing.T) {
			authority := NewTabConsentAuthority()
			authority.now = func() time.Time { return now }
			candidate := base
			mutate(&candidate)
			if _, err := authority.Issue(candidate); !errors.Is(err, ErrTabConsentDenied) {
				t.Fatalf("issue error = %v, want denied", err)
			}
		})
	}
	authority := NewTabConsentAuthority()
	authority.now = func() time.Time { return now }
	receipt, err := authority.Issue(base)
	if err != nil {
		t.Fatal(err)
	}
	authority.now = func() time.Time { return now.Add(2 * time.Minute) }
	binding := TabConsentBinding{RunID: base.RunID, GrantReceipt: base.GrantReceipt, Profile: base.Profile, SessionID: base.SessionID, LeaseID: base.LeaseID, OwnerID: base.OwnerID, ProfileID: base.ProfileID, TabID: base.TabID, CurrentURL: base.URL}
	if err := authority.Verify(receipt, binding); !errors.Is(err, ErrTabConsentDenied) {
		t.Fatalf("expired receipt error = %v, want denied", err)
	}
}
