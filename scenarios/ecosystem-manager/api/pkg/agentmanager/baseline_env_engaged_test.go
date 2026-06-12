package agentmanager

import (
	"reflect"
	"testing"
)

func TestWithEngagedShadowScenario(t *testing.T) { // [REQ:EM-BASE-002]
	cases := []struct {
		name     string
		base     map[string]string
		scenario string
		want     map[string]string
	}{
		{
			name:     "blank scenario returns base unchanged (nil)",
			base:     nil,
			scenario: "  ",
			want:     nil,
		},
		{
			name:     "adds to nil base",
			base:     nil,
			scenario: "swarm-manager",
			want:     map[string]string{EnvShadowScenarios: "swarm-manager"},
		},
		{
			name:     "unions with existing comma-list, preserving order",
			base:     map[string]string{EnvShadowScenarios: "agent-manager,test-genie"},
			scenario: "swarm-manager",
			want:     map[string]string{EnvShadowScenarios: "agent-manager,test-genie,swarm-manager"},
		},
		{
			name:     "dedups an already-present scenario",
			base:     map[string]string{EnvShadowScenarios: "swarm-manager,test-genie"},
			scenario: "swarm-manager",
			want:     map[string]string{EnvShadowScenarios: "swarm-manager,test-genie"},
		},
		{
			name:     "preserves unrelated keys",
			base:     map[string]string{"VROOLI_OTHER": "x"},
			scenario: "swarm-manager",
			want:     map[string]string{"VROOLI_OTHER": "x", EnvShadowScenarios: "swarm-manager"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WithEngagedShadowScenario(tc.base, tc.scenario)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("WithEngagedShadowScenario(%v, %q) = %v, want %v", tc.base, tc.scenario, got, tc.want)
			}
		})
	}
}

// TestWithEngagedShadowScenario_DoesNotMutateBase verifies the input map is not
// mutated (the spawn site reuses AmbientShadowEnv()'s result).
func TestWithEngagedShadowScenario_DoesNotMutateBase(t *testing.T) {
	base := map[string]string{EnvShadowScenarios: "agent-manager"}
	_ = WithEngagedShadowScenario(base, "swarm-manager")
	if base[EnvShadowScenarios] != "agent-manager" {
		t.Fatalf("base map was mutated: %v", base)
	}
}
