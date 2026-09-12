package aisearch

import "testing"

// clearNamespaceEnv resets every environment variable that participates in
// variant-aware namespace resolution so each case starts from a known-empty
// base. t.Setenv restores the prior values when the test finishes.
func clearNamespaceEnv(t *testing.T) {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_VARIANT", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("AI_SEARCH_BACKLOG_COLLECTION", "")
	t.Setenv("AI_SEARCH_GOAL_COLLECTION", "")
	t.Setenv("AI_SEARCH_RECORD_COLLECTION", "")
}

func TestResolveCollection_VariantAware(t *testing.T) {
	cases := []struct {
		name      string
		namespace string // VROOLI_STORAGE_NAMESPACE
		variant   string // VROOLI_VARIANT
		scenario  string // VROOLI_SCENARIO
		override  string // AI_SEARCH_BACKLOG_COLLECTION
		want      string
	}{
		{
			name:      "live via injected storage namespace",
			namespace: "swarm-manager",
			variant:   "live",
			want:      "swarm-manager_backlog",
		},
		{
			name:      "shadow gets its own collection",
			namespace: "swarm-manager_shadow",
			variant:   "shadow",
			want:      "swarm-manager_shadow_backlog",
		},
		{
			name:     "live fallback from bare scenario slug",
			scenario: "swarm-manager",
			want:     "swarm-manager_backlog",
		},
		{
			name:     "operator override wins verbatim",
			scenario: "swarm-manager",
			override: "custom-backlog",
			want:     "custom-backlog",
		},
		{
			name: "no identity env at all falls back to live local slug",
			want: "swarm-manager_backlog",
		},
		{
			name:    "inconsistent shadow env stays isolated from live",
			variant: "shadow", // declared shadow but no namespace root injected
			want:    "swarm-manager_shadow_backlog",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearNamespaceEnv(t)
			if tc.namespace != "" {
				t.Setenv("VROOLI_STORAGE_NAMESPACE", tc.namespace)
			}
			if tc.variant != "" {
				t.Setenv("VROOLI_VARIANT", tc.variant)
			}
			if tc.scenario != "" {
				t.Setenv("VROOLI_SCENARIO", tc.scenario)
			}
			if tc.override != "" {
				t.Setenv("AI_SEARCH_BACKLOG_COLLECTION", tc.override)
			}

			got := resolveCollection(EnvAISearchBacklogColl, CollectionDomainBacklog)
			if got != tc.want {
				t.Fatalf("resolveCollection(backlog) = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestResolveCollection_NeverAliasesShadowOntoLive is the safety invariant the
// whole namespace mechanism exists to protect: a shadow instance must never
// resolve to the live collection, even when its environment is mis-configured.
func TestResolveCollection_NeverAliasesShadowOntoLive(t *testing.T) {
	clearNamespaceEnv(t)
	t.Setenv("VROOLI_VARIANT", "shadow") // shadow declared, root deliberately missing

	live := "swarm-manager_backlog"
	got := resolveCollection(EnvAISearchBacklogColl, CollectionDomainBacklog)
	if got == live {
		t.Fatalf("shadow collection aliased onto live %q — shadow writes would corrupt live state", live)
	}
}

func TestLoadConfigFromEnv_VariantAwareCollections(t *testing.T) {
	clearNamespaceEnv(t)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "swarm-manager_shadow")
	t.Setenv("VROOLI_VARIANT", "shadow")

	cfg := LoadConfigFromEnv()
	checks := map[string]string{
		"backlog": cfg.BacklogCollection,
		"goals":   cfg.GoalCollection,
		"records": cfg.RecordCollection,
	}
	want := map[string]string{
		"backlog": "swarm-manager_shadow_backlog",
		"goals":   "swarm-manager_shadow_goals",
		"records": "swarm-manager_shadow_records",
	}
	for domain, got := range checks {
		if got != want[domain] {
			t.Errorf("LoadConfigFromEnv %s collection = %q, want %q", domain, got, want[domain])
		}
	}
}

// TestLoadConfigFromEnv_LiveCollectionsUnprefixedVariant pins the live names so a
// future change can't silently re-introduce the old hyphenated form or a shadow
// suffix on the live instance.
func TestLoadConfigFromEnv_LiveCollections(t *testing.T) {
	clearNamespaceEnv(t)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "swarm-manager")
	t.Setenv("VROOLI_VARIANT", "live")

	cfg := LoadConfigFromEnv()
	if cfg.BacklogCollection != "swarm-manager_backlog" {
		t.Errorf("live backlog collection = %q, want swarm-manager_backlog", cfg.BacklogCollection)
	}
	if cfg.GoalCollection != "swarm-manager_goals" {
		t.Errorf("live goal collection = %q, want swarm-manager_goals", cfg.GoalCollection)
	}
	if cfg.RecordCollection != "swarm-manager_records" {
		t.Errorf("live record collection = %q, want swarm-manager_records", cfg.RecordCollection)
	}
}
