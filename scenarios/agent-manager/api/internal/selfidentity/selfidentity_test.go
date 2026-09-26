package selfidentity

import "testing"

func TestIs(t *testing.T) {
	cases := []struct {
		name     string
		scenario string
		want     bool
	}{
		{"own scenario", "agent-manager", true},
		{"other scenario", "test-genie", false},
		{"empty", "", false},
		{"prefix is not a match", "agent-manager-foo", false},
		{"case sensitive", "Agent-Manager", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Is(tc.scenario); got != tc.want {
				t.Fatalf("Is(%q) = %v, want %v", tc.scenario, got, tc.want)
			}
		})
	}
}

func TestScenarioNameMatchesPreflight(t *testing.T) {
	// Guards against the constant drifting away from main.go's preflight slug.
	if ScenarioName != "agent-manager" {
		t.Fatalf("ScenarioName = %q, expected the agent-manager lifecycle slug", ScenarioName)
	}
}
