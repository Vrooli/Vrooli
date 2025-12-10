package security

import (
	"testing"
)

func TestClassifyPathOverlap(t *testing.T) {
	tests := []struct {
		name         string
		pathA        string
		pathB        string
		wantOverlaps bool
		wantReason   PathOverlapReason
	}{
		{
			name:         "Exact match",
			pathA:        "api/handlers",
			pathB:        "api/handlers",
			wantOverlaps: true,
			wantReason:   OverlapReasonExactMatch,
		},
		{
			name:         "Exact match with trailing slash",
			pathA:        "api/handlers/",
			pathB:        "api/handlers",
			wantOverlaps: true,
			wantReason:   OverlapReasonExactMatch,
		},
		{
			name:         "A contains B (A is parent)",
			pathA:        "api",
			pathB:        "api/handlers",
			wantOverlaps: true,
			wantReason:   OverlapReasonAContainsB,
		},
		{
			name:         "B contains A (B is parent)",
			pathA:        "api/handlers/auth",
			pathB:        "api/handlers",
			wantOverlaps: true,
			wantReason:   OverlapReasonBContainsA,
		},
		{
			name:         "No overlap - sibling directories",
			pathA:        "api/handlers",
			pathB:        "api/services",
			wantOverlaps: false,
			wantReason:   OverlapReasonNoOverlap,
		},
		{
			name:         "No overlap - completely different paths",
			pathA:        "api/handlers",
			pathB:        "ui/components",
			wantOverlaps: false,
			wantReason:   OverlapReasonNoOverlap,
		},
		{
			name:         "No overlap - partial name match but different directory",
			pathA:        "api/handlers",
			pathB:        "api/handlers2",
			wantOverlaps: false,
			wantReason:   OverlapReasonNoOverlap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyPathOverlap(tt.pathA, tt.pathB)

			if result.Overlaps != tt.wantOverlaps {
				t.Errorf("Overlaps = %v, want %v", result.Overlaps, tt.wantOverlaps)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %v, want %v", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestClassifyScopeConflict(t *testing.T) {
	tests := []struct {
		name           string
		scopeA         []string
		scopeB         []string
		wantConflict   bool
		wantHasPaths   bool // Whether ConflictingPaths should be populated
		wantReasonType string
	}{
		{
			name:           "Empty scope A conflicts with anything",
			scopeA:         []string{},
			scopeB:         []string{"api/handlers"},
			wantConflict:   true,
			wantHasPaths:   false,
			wantReasonType: "entire scenario",
		},
		{
			name:           "Empty scope B conflicts with anything",
			scopeA:         []string{"api/handlers"},
			scopeB:         []string{},
			wantConflict:   true,
			wantHasPaths:   false,
			wantReasonType: "entire scenario",
		},
		{
			name:           "Both empty scopes conflict",
			scopeA:         []string{},
			scopeB:         []string{},
			wantConflict:   true,
			wantHasPaths:   false,
			wantReasonType: "entire scenario",
		},
		{
			name:           "Overlapping paths conflict",
			scopeA:         []string{"api/handlers"},
			scopeB:         []string{"api/handlers/auth"},
			wantConflict:   true,
			wantHasPaths:   true,
			wantReasonType: "paths overlap",
		},
		{
			name:           "No overlap - disjoint paths",
			scopeA:         []string{"api/handlers"},
			scopeB:         []string{"ui/components"},
			wantConflict:   false,
			wantHasPaths:   false,
			wantReasonType: "",
		},
		{
			name:           "Multiple paths - one conflict",
			scopeA:         []string{"api/handlers", "api/services"},
			scopeB:         []string{"ui/components", "api/handlers/auth"},
			wantConflict:   true,
			wantHasPaths:   true,
			wantReasonType: "paths overlap",
		},
		{
			name:           "Multiple paths - no conflict",
			scopeA:         []string{"api/handlers", "api/services"},
			scopeB:         []string{"ui/components", "ui/hooks"},
			wantConflict:   false,
			wantHasPaths:   false,
			wantReasonType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyScopeConflict(tt.scopeA, tt.scopeB)

			if result.HasConflict != tt.wantConflict {
				t.Errorf("HasConflict = %v, want %v", result.HasConflict, tt.wantConflict)
			}

			if tt.wantHasPaths && len(result.ConflictingPaths) == 0 {
				t.Error("Expected ConflictingPaths to be populated")
			}

			if !tt.wantHasPaths && len(result.ConflictingPaths) > 0 {
				t.Error("Expected ConflictingPaths to be empty")
			}

			if tt.wantReasonType != "" && !containsStr(result.Reason, tt.wantReasonType) {
				t.Errorf("Reason = %q, want substring %q", result.Reason, tt.wantReasonType)
			}
		})
	}
}

func TestPathSetsOverlap(t *testing.T) {
	// Simple boolean wrapper tests
	if !PathSetsOverlap([]string{}, []string{"api"}) {
		t.Error("Expected empty scope to conflict with any scope")
	}

	if !PathSetsOverlap([]string{"api/handlers"}, []string{"api/handlers/auth"}) {
		t.Error("Expected overlapping paths to conflict")
	}

	if PathSetsOverlap([]string{"api"}, []string{"ui"}) {
		t.Error("Expected disjoint paths to not conflict")
	}
}

// helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
