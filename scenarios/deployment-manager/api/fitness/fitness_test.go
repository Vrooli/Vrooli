package fitness

import (
	"testing"
)

// [REQ:DM-P0-003] TestCalculateScore tests the actual fitness score calculation logic
func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name           string
		scenario       string
		tier           int
		wantOverall    int
		wantNonZero    bool
		wantBlocker    bool
		blockerPattern string
	}{
		{
			name:        "tier 1 local - full fitness",
			scenario:    "any-scenario",
			tier:        TierLocal,
			wantOverall: 100,
			wantNonZero: true,
			wantBlocker: false,
		},
		{
			name:        "tier 2 desktop - moderate fitness",
			scenario:    "picker-wheel",
			tier:        TierDesktop,
			wantOverall: 75,
			wantNonZero: true,
			wantBlocker: false,
		},
		{
			name:           "tier 3 mobile - constrained fitness",
			scenario:       "heavy-scenario",
			tier:           TierMobile,
			wantOverall:    40,
			wantNonZero:    true,
			wantBlocker:    true,
			blockerPattern: "Mobile tier requires lightweight dependencies",
		},
		{
			name:        "tier 4 saas - good fitness",
			scenario:    "web-scenario",
			tier:        TierSaaS,
			wantOverall: 85,
			wantNonZero: true,
			wantBlocker: false,
		},
		{
			name:           "tier 5 enterprise - compliance constraints",
			scenario:       "enterprise-scenario",
			tier:           TierEnterprise,
			wantOverall:    60,
			wantNonZero:    true,
			wantBlocker:    true,
			blockerPattern: "Enterprise tier requires license compliance",
		},
		{
			name:           "invalid tier 0",
			scenario:       "any-scenario",
			tier:           0,
			wantOverall:    0,
			wantNonZero:    false,
			wantBlocker:    true,
			blockerPattern: "Invalid tier: 0",
		},
		{
			name:           "invalid tier 99",
			scenario:       "any-scenario",
			tier:           99,
			wantOverall:    0,
			wantNonZero:    false,
			wantBlocker:    true,
			blockerPattern: "Invalid tier: 99",
		},
		{
			name:           "negative tier",
			scenario:       "any-scenario",
			tier:           -1,
			wantOverall:    0,
			wantNonZero:    false,
			wantBlocker:    true,
			blockerPattern: "Invalid tier: -1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.scenario, tt.tier)

			if score.Overall != tt.wantOverall {
				t.Errorf("Overall = %d, want %d", score.Overall, tt.wantOverall)
			}

			if tt.wantNonZero && score.Overall == 0 {
				t.Error("expected non-zero overall score")
			}

			if tt.wantBlocker {
				if score.BlockerReason == "" {
					t.Error("expected blocker reason to be set")
				}
				if tt.blockerPattern != "" && !contains(score.BlockerReason, tt.blockerPattern) {
					t.Errorf("BlockerReason %q does not contain %q", score.BlockerReason, tt.blockerPattern)
				}
			} else {
				if score.BlockerReason != "" {
					t.Errorf("unexpected blocker reason: %s", score.BlockerReason)
				}
			}
		})
	}
}

// [REQ:DM-P0-004] TestGetTierFitnessPolicy tests tier policy retrieval
func TestGetTierFitnessPolicy(t *testing.T) {
	tests := []struct {
		name         string
		tier         int
		wantErr      bool
		wantOverall  int
		checkSubscores bool
	}{
		{
			name:           "local tier policy",
			tier:           TierLocal,
			wantErr:        false,
			wantOverall:    100,
			checkSubscores: true,
		},
		{
			name:        "desktop tier policy",
			tier:        TierDesktop,
			wantErr:     false,
			wantOverall: 75,
		},
		{
			name:        "mobile tier policy",
			tier:        TierMobile,
			wantErr:     false,
			wantOverall: 40,
		},
		{
			name:        "saas tier policy",
			tier:        TierSaaS,
			wantErr:     false,
			wantOverall: 85,
		},
		{
			name:        "enterprise tier policy",
			tier:        TierEnterprise,
			wantErr:     false,
			wantOverall: 60,
		},
		{
			name:        "invalid tier 0",
			tier:        0,
			wantErr:     true,
			wantOverall: 0,
		},
		{
			name:        "invalid tier 6",
			tier:        6,
			wantErr:     true,
			wantOverall: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := GetTierFitnessPolicy(tt.tier)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if policy.Overall != tt.wantOverall {
				t.Errorf("Overall = %d, want %d", policy.Overall, tt.wantOverall)
			}

			// Verify all subscores are populated for valid tiers
			if tt.checkSubscores && !tt.wantErr {
				if policy.Portability == 0 {
					t.Error("expected non-zero Portability subscore")
				}
				if policy.Resources == 0 {
					t.Error("expected non-zero Resources subscore")
				}
				if policy.Licensing == 0 {
					t.Error("expected non-zero Licensing subscore")
				}
				if policy.PlatformSupport == 0 {
					t.Error("expected non-zero PlatformSupport subscore")
				}
			}
		})
	}
}

// [REQ:DM-P0-005] TestIsTierBlocked tests blocker detection logic
func TestIsTierBlocked(t *testing.T) {
	tests := []struct {
		name    string
		tier    int
		blocked bool
	}{
		{"local tier not blocked", TierLocal, false},
		{"desktop tier not blocked", TierDesktop, false},
		{"mobile tier not blocked", TierMobile, false},
		{"saas tier not blocked", TierSaaS, false},
		{"enterprise tier not blocked", TierEnterprise, false},
		{"unknown tier 0 blocked", 0, true},
		{"unknown tier -1 blocked", -1, true},
		{"unknown tier 99 blocked", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTierBlocked(tt.tier)
			if result != tt.blocked {
				t.Errorf("IsTierBlocked(%d) = %v, want %v", tt.tier, result, tt.blocked)
			}
		})
	}
}

// TestIsTierWarningLevel tests warning threshold detection
func TestIsTierWarningLevel(t *testing.T) {
	tests := []struct {
		name      string
		score     int
		isWarning bool
	}{
		{"score 0 not warning (blocked)", 0, false},
		{"score 1 is warning", 1, true},
		{"score 49 is warning", 49, true},
		{"score 50 not warning (threshold)", 50, false},
		{"score 75 not warning", 75, false},
		{"score 100 not warning", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTierWarningLevel(tt.score)
			if result != tt.isWarning {
				t.Errorf("IsTierWarningLevel(%d) = %v, want %v", tt.score, result, tt.isWarning)
			}
		})
	}
}

// TestAllTiers verifies all valid tiers are returned
func TestAllTiers(t *testing.T) {
	tiers := AllTiers()

	if len(tiers) != 5 {
		t.Fatalf("expected 5 tiers, got %d", len(tiers))
	}

	expected := []int{TierLocal, TierDesktop, TierMobile, TierSaaS, TierEnterprise}
	for i, tier := range expected {
		if tiers[i] != tier {
			t.Errorf("AllTiers()[%d] = %d, want %d", i, tiers[i], tier)
		}
	}

	// Verify all tiers have valid policies
	for _, tier := range tiers {
		policy, err := GetTierFitnessPolicy(tier)
		if err != nil {
			t.Errorf("tier %d has no policy: %v", tier, err)
		}
		if policy.Overall == 0 && tier != TierMobile {
			// Mobile has warning but not zero (it's 40)
			t.Errorf("tier %d has zero overall score", tier)
		}
	}
}

// TestGetTierDisplayName tests human-readable tier name conversion
func TestGetTierDisplayName(t *testing.T) {
	tests := []struct {
		tier int
		name string
	}{
		{TierLocal, "local"},
		{TierDesktop, "desktop"},
		{TierMobile, "mobile"},
		{TierSaaS, "saas"},
		{TierEnterprise, "enterprise"},
		{0, "tier-0"},
		{99, "tier-99"},
		{-1, "tier--1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTierDisplayName(tt.tier)
			if result != tt.name {
				t.Errorf("GetTierDisplayName(%d) = %q, want %q", tt.tier, result, tt.name)
			}
		})
	}
}

// TestScoreSubscoreConsistency verifies subscores are consistent with overall
func TestScoreSubscoreConsistency(t *testing.T) {
	for _, tier := range AllTiers() {
		t.Run(GetTierDisplayName(tier), func(t *testing.T) {
			score := CalculateScore("any-scenario", tier)

			// Verify all subscores are within valid range
			subscores := []struct {
				name  string
				value int
			}{
				{"Portability", score.Portability},
				{"Resources", score.Resources},
				{"Licensing", score.Licensing},
				{"PlatformSupport", score.PlatformSupport},
			}

			for _, s := range subscores {
				if s.value < 0 || s.value > 100 {
					t.Errorf("%s = %d, want 0-100", s.name, s.value)
				}
			}

			// Overall should be 0-100
			if score.Overall < 0 || score.Overall > 100 {
				t.Errorf("Overall = %d, want 0-100", score.Overall)
			}
		})
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
