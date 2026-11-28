package routing

import (
	"strings"
	"testing"
)

func TestAdmitsUnifiesPrivacyAndProfileRules(t *testing.T) {
	tests := []struct {
		name, profile, privacy, locality string
		want                             bool
	}{
		{"secret blocks remote under any profile", "PROFILE_LOCAL_FIRST", "PRIVACY_CLASS_SECRET", "remote", false},
		{"confidential blocks remote under local-first", "PROFILE_LOCAL_FIRST", "PRIVACY_CLASS_CONFIDENTIAL", "remote", false},
		{"confidential blocks remote under quality-first", "PROFILE_QUALITY_FIRST", "PRIVACY_CLASS_CONFIDENTIAL", "remote", false},
		{"confidential blocks remote under cheap-first", "PROFILE_CHEAP_FIRST", "PRIVACY_CLASS_CONFIDENTIAL", "remote", false},
		{"confidential blocks remote under remote-only", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_CONFIDENTIAL", "remote", false},
		{"remote-only still forbids local for confidential data", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_CONFIDENTIAL", "local", false},
		{"secret permits local", "PROFILE_QUALITY_FIRST", "PRIVACY_CLASS_SECRET", "local", true},
		{"unspecified privacy blocks remote by default", "PROFILE_UNSPECIFIED", "PRIVACY_CLASS_UNSPECIFIED", "remote", false},
		{"unspecified privacy blocks remote even with remote-only profile", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_UNSPECIFIED", "remote", false},
		{"unknown privacy blocks remote", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_FUTURE", "remote", false},
		{"explicit privacy permits remote under remote-only profile", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_PUBLIC", "remote", true},
		{"sensitive confidential blocks remote", "PROFILE_PRIVACY_SENSITIVE", "PRIVACY_CLASS_CONFIDENTIAL", "remote", false},
		{"sensitive internal retains fallback eligibility", "PROFILE_PRIVACY_SENSITIVE", "PRIVACY_CLASS_INTERNAL", "remote", true},
		{"local only blocks remote", "PROFILE_LOCAL_ONLY", "PRIVACY_CLASS_PUBLIC", "remote", false},
		{"remote only blocks local", "PROFILE_REMOTE_ONLY", "PRIVACY_CLASS_PUBLIC", "local", false},
		{"unknown locality fails closed", "PROFILE_LOCAL_FIRST", "PRIVACY_CLASS_PUBLIC", "unknown", false},
		{"unknown profile fails closed", "PROFILE_FUTURE", "PRIVACY_CLASS_PUBLIC", "local", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := Admits(test.profile, test.privacy, test.locality)
			if got != test.want {
				t.Fatalf("Admits(%q, %q, %q) = %t, want %t", test.profile, test.privacy, test.locality, got, test.want)
			}
		})
	}
}

func TestRankUsesLocalityPolicyAndStableProviderTieBreak(t *testing.T) {
	tests := []struct {
		profile, locality, provider string
		want                        int
	}{
		{"PROFILE_LOCAL_FIRST", "local", "ollama", 0},
		{"PROFILE_LOCAL_FIRST", "remote", "openrouter", 6},
		{"PROFILE_QUALITY_FIRST", "remote", "ollama", 0},
		{"PROFILE_QUALITY_FIRST", "local", "ollama", 5},
		{"PROFILE_REMOTE_ONLY", "remote", "openrouter", 1},
		{"PROFILE_UNSPECIFIED", "local", "ollama", 10},
	}
	for _, test := range tests {
		if got := Rank(test.profile, test.locality, test.provider); got != test.want {
			t.Errorf("Rank(%q, %q, %q) = %d, want %d", test.profile, test.locality, test.provider, got, test.want)
		}
	}
}

func TestValidPrivacyProvenanceRequiresShortNonSensitiveIdentifier(t *testing.T) {
	for _, value := range []string{"agent_manager.operator_and_agent_conversation", "user-input/document-42", "policy:public-source"} {
		if !ValidPrivacyProvenance(value) {
			t.Errorf("ValidPrivacyProvenance(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", " spaces are not identifiers", "private user name", "path\nwith-newline", "a/" + strings.Repeat("x", 128)} {
		if ValidPrivacyProvenance(value) {
			t.Errorf("ValidPrivacyProvenance(%q) = true, want false", value)
		}
	}
}
