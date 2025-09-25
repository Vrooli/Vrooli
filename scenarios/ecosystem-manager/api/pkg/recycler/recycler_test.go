package recycler

import "testing"

func TestShouldFinalize(t *testing.T) {
	tests := []struct {
		name      string
		streak    int
		threshold int
		expected  bool
	}{
		{"zeroThreshold", 3, 0, false},
		{"belowThreshold", 2, 3, false},
		{"meetsThreshold", 3, 3, true},
		{"aboveThreshold", 4, 3, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldFinalize(tc.streak, tc.threshold); got != tc.expected {
				t.Fatalf("shouldFinalize(%d,%d) = %v, want %v", tc.streak, tc.threshold, got, tc.expected)
			}
		})
	}
}

func TestIsTypeEnabled(t *testing.T) {
	cases := []struct {
		name     string
		taskType string
		mode     string
		expected bool
	}{
		{"bothResource", "resource", enabledBoth, true},
		{"bothScenario", "scenario", enabledBoth, true},
		{"resourcesOnly", "resource", enabledResources, true},
		{"resourcesWrong", "scenario", enabledResources, false},
		{"scenariosOnly", "scenario", enabledScenarios, true},
		{"scenariosWrong", "resource", enabledScenarios, false},
		{"off", "resource", enabledOff, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTypeEnabled(tc.taskType, tc.mode); got != tc.expected {
				t.Fatalf("isTypeEnabled(%q,%q) = %v, want %v", tc.taskType, tc.mode, got, tc.expected)
			}
		})
	}
}

func TestExtractOutput(t *testing.T) {
	if out := extractOutput(nil); out != "" {
		t.Fatalf("expected empty string for nil map, got %q", out)
	}

	if out := extractOutput(map[string]interface{}{"output": 123}); out != "" {
		t.Fatalf("expected empty string for non-string output, got %q", out)
	}

	if out := extractOutput(map[string]interface{}{"output": "hello"}); out != "hello" {
		t.Fatalf("expected 'hello', got %q", out)
	}
}
