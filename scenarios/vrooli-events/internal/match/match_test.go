package match

import "testing"

// [REQ:REQ-PS-002] Verify glob-pattern matching logic (single *, double **, exact, mixed)
func TestGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		// Empty pattern matches everything
		{"empty_pattern_matches_dotted", "", "anything.at.all", true},
		{"empty_pattern_matches_empty", "", "", true},

		// Exact match
		{"exact_match", "app.domain.action.v1", "app.domain.action.v1", true},
		{"exact_mismatch_last_seg", "app.domain.action.v1", "app.domain.action.v2", false},

		// Single wildcard (*)
		{"star_mid_match", "app.*.action.v1", "app.domain.action.v1", true},
		{"star_mid_other", "app.*.action.v1", "app.other.action.v1", true},
		{"star_mid_wrong_seg", "app.*.action.v1", "app.domain.other.v1", false},
		{"star_first_seg", "*.domain.action.v1", "app.domain.action.v1", true},
		{"star_last_seg_v1", "app.domain.action.*", "app.domain.action.v1", true},
		{"star_last_seg_v2", "app.domain.action.*", "app.domain.action.v2", true},

		// * should NOT match multiple segments
		{"star_no_multi_seg", "app.*.v1", "app.domain.action.v1", false},

		// Double wildcard (**)
		{"dstar_suffix_multi", "app.**", "app.domain.action.v1", true},
		{"dstar_suffix_single", "app.**", "app.x", true},
		{"dstar_all", "**", "any.thing", true},
		{"dstar_prefix_multi", "**.v1", "app.domain.action.v1", true},
		{"dstar_prefix_short", "**.v1", "single.v1", true},
		{"dstar_mid_multi", "app.**.v1", "app.domain.action.v1", true},
		{"dstar_mid_short", "app.**.v1", "app.x.v1", true},

		// ** must match at least one segment
		{"dstar_needs_one_seg", "app.**.v1", "app.v1", false},

		// Mixed
		{"mixed_star_dstar_long", "app.*.**.v1", "app.domain.action.sub.v1", true},
		{"mixed_star_dstar_short", "app.*.**.v1", "app.domain.action.v1", true},
		{"mixed_star_dstar_too_short", "app.*.**.v1", "app.domain.v1", false},

		// Length mismatches
		{"pattern_shorter", "app.domain", "app.domain.action", false},
		{"pattern_longer", "app.domain.action", "app.domain", false},

		// Edge: single segment
		{"single_exact", "app", "app", true},
		{"single_mismatch", "app", "other", false},
		{"star_single_seg", "*", "anything", true},
		{"dstar_single_seg", "**", "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Glob(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("Glob(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}
