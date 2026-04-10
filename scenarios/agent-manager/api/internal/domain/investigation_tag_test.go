package domain

import "testing"

func TestMatchesInvestigationTag_DefaultAllowlist(t *testing.T) {
	tests := []struct {
		tag   string
		match bool
	}{
		{"investigation", true},
		{"Investigation", true},
		{"agent-manager-investigation", true},
		{"scenario-to-cloud-investigation", true},
		{"agent-manager-investigation-apply", false},
		{"random-tag", false},
	}

	for _, tc := range tests {
		if got := MatchesInvestigationTag(tc.tag, nil); got != tc.match {
			t.Fatalf("MatchesInvestigationTag(%q) = %v, want %v", tc.tag, got, tc.match)
		}
	}
}

func TestMatchesInvestigationTag_RegexCaseSensitive(t *testing.T) {
	rules := []InvestigationTagRule{
		{
			Pattern:       "^INVESTIGATION$",
			IsRegex:       true,
			CaseSensitive: true,
		},
	}

	if MatchesInvestigationTag("INVESTIGATION", rules) != true {
		t.Fatalf("expected case-sensitive regex match")
	}
	if MatchesInvestigationTag("investigation", rules) != false {
		t.Fatalf("expected case-sensitive regex mismatch")
	}
}
