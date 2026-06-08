package agentmanager

import "testing"

func clearNamespaceEnv(t *testing.T) {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_VARIANT", "")
	t.Setenv("VROOLI_SCENARIO", "")
}

func TestBuildResearchTag_VariantAware(t *testing.T) {
	cases := []struct {
		name      string
		namespace string
		variant   string
		idea      string
		want      string
	}{
		{
			name:      "live with idea is byte-identical to pre-adoption form",
			namespace: "swarm-manager",
			variant:   "live",
			idea:      "fall-foliage",
			want:      "swarm-manager:idea:fall-foliage:research",
		},
		{
			name:      "live without idea",
			namespace: "swarm-manager",
			variant:   "live",
			idea:      "",
			want:      "swarm-manager:idea:research",
		},
		{
			name:      "shadow tags its runs under the shadow namespace",
			namespace: "swarm-manager_shadow",
			variant:   "shadow",
			idea:      "fall-foliage",
			want:      "swarm-manager_shadow:idea:fall-foliage:research",
		},
		{
			name:      "idea name is trimmed",
			namespace: "swarm-manager",
			variant:   "live",
			idea:      "  fall-foliage  ",
			want:      "swarm-manager:idea:fall-foliage:research",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearNamespaceEnv(t)
			t.Setenv("VROOLI_STORAGE_NAMESPACE", tc.namespace)
			t.Setenv("VROOLI_VARIANT", tc.variant)

			got, err := buildResearchTag(tc.idea)
			if err != nil {
				t.Fatalf("buildResearchTag(%q) returned error: %v", tc.idea, err)
			}
			if got != tc.want {
				t.Fatalf("buildResearchTag(%q) = %q, want %q", tc.idea, got, tc.want)
			}
		})
	}
}

// TestBuildResearchTag_FailLoudOnInconsistentEnv verifies that a declared-shadow
// process with no namespace root injected does not silently tag its runs as live
// — it fails loud so the caller can refuse rather than contaminate live's tags.
func TestBuildResearchTag_FailLoudOnInconsistentEnv(t *testing.T) {
	clearNamespaceEnv(t)
	t.Setenv("VROOLI_VARIANT", "shadow") // shadow declared, root deliberately missing

	if _, err := buildResearchTag("fall-foliage"); err == nil {
		t.Fatal("expected error for inconsistent shadow environment, got nil")
	}
}
