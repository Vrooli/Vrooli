package validation

import "testing"

func TestIsSafeScenarioName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple name", "my-scenario", true},
		{"valid with numbers", "scenario123", true},
		{"valid with dots", "my.scenario", true},
		{"empty string", "", false},
		{"path traversal dotdot", "../etc/passwd", false},
		{"dotdot in middle", "foo/../bar", false},
		{"dotdot only", "..", false},
		{"forward slash", "foo/bar", false},
		{"backslash", "foo\\bar", false},
		{"backslash only", "\\", false},
		{"forward slash only", "/", false},
		{"valid with hyphens and underscores", "my_scenario-name", true},
		{"space is allowed", "my scenario", true},
		{"single character", "a", true},
		{"single dot is allowed", ".", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSafeScenarioName(tt.input)
			if got != tt.want {
				t.Errorf("IsSafeScenarioName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
