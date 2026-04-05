package match

import "testing"

func TestGlob(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		// Empty pattern matches everything
		{"", "anything.at.all", true},
		{"", "", true},

		// Exact match
		{"app.domain.action.v1", "app.domain.action.v1", true},
		{"app.domain.action.v1", "app.domain.action.v2", false},

		// Single wildcard (*)
		{"app.*.action.v1", "app.domain.action.v1", true},
		{"app.*.action.v1", "app.other.action.v1", true},
		{"app.*.action.v1", "app.domain.other.v1", false},
		{"*.domain.action.v1", "app.domain.action.v1", true},
		{"app.domain.action.*", "app.domain.action.v1", true},
		{"app.domain.action.*", "app.domain.action.v2", true},

		// * should NOT match multiple segments
		{"app.*.v1", "app.domain.action.v1", false},

		// Double wildcard (**)
		{"app.**", "app.domain.action.v1", true},
		{"app.**", "app.x", true},
		{"**", "any.thing", true},
		{"**.v1", "app.domain.action.v1", true},
		{"**.v1", "single.v1", true},
		{"app.**.v1", "app.domain.action.v1", true},
		{"app.**.v1", "app.x.v1", true},

		// ** must match at least one segment
		{"app.**.v1", "app.v1", false},

		// Mixed
		{"app.*.**.v1", "app.domain.action.sub.v1", true},
		{"app.*.**.v1", "app.domain.action.v1", true},
		{"app.*.**.v1", "app.domain.v1", false}, // ** needs at least 1

		// No match
		{"app.domain", "app.domain.action", false},
		{"app.domain.action", "app.domain", false},
	}

	for _, tt := range tests {
		got := Glob(tt.pattern, tt.value)
		if got != tt.want {
			t.Errorf("Glob(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
		}
	}
}
