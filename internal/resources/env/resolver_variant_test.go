package env

import "testing"

// TestApplyDependencyOverridesVariantDatabaseName proves the Postgres database
// name is derived through the InstanceKey SSOT: the live instance keeps the
// unchanged "vrooli_<scenario>" name while a shadow gets "vrooli_<scenario>_<variant>",
// so a shadow can never write the live database. A non-postgres resource is
// untouched, and an explicit dependency-declared database name still wins.
func TestApplyDependencyOverridesVariantDatabaseName(t *testing.T) {
	cases := []struct {
		name     string
		resource string
		opts     ResolveOptions
		wantDB   string // "" means POSTGRES_DB must be absent
	}{
		{
			name:     "live default name unchanged",
			resource: "postgres",
			opts:     ResolveOptions{ScenarioName: "swarm-manager"},
			wantDB:   "vrooli_swarm_manager",
		},
		{
			name:     "explicit live variant matches default",
			resource: "postgres",
			opts:     ResolveOptions{ScenarioName: "swarm-manager", Variant: "live"},
			wantDB:   "vrooli_swarm_manager",
		},
		{
			name:     "shadow variant suffixes the db",
			resource: "postgres",
			opts:     ResolveOptions{ScenarioName: "swarm-manager", Variant: "shadow"},
			wantDB:   "vrooli_swarm_manager_shadow",
		},
		{
			name:     "non-postgres resource is untouched",
			resource: "redis",
			opts:     ResolveOptions{ScenarioName: "swarm-manager", Variant: "shadow"},
			wantDB:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{}
			applyDependencyOverrides(tc.resource, tc.opts, values)
			got, ok := values["POSTGRES_DB"]
			if tc.wantDB == "" {
				if ok {
					t.Fatalf("POSTGRES_DB = %q, want absent", got)
				}
				return
			}
			if got != tc.wantDB {
				t.Fatalf("POSTGRES_DB = %q, want %q", got, tc.wantDB)
			}
		})
	}
}
