package retention

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func TestParseManifestAutohealShape(t *testing.T) {
	manifest := `{
	  "retention": {
	    "budgets": {
	      "system_events": {
	        "target": {
	          "kind": "sqlite_table",
	          "database": "autoheal.sqlite",
	          "table": "system_events",
	          "time_column": "occurred_at"
	        },
	        "max_age": "30d",
	        "max_bytes": "2GiB",
	        "pruner": "builtin",
	        "rationale": "Host event ingest."
	      }
	    }
	  }
	}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
	spec := specs[0]
	if spec.Budget.Name != "system_events" {
		t.Errorf("Name = %q, want system_events", spec.Budget.Name)
	}
	if spec.Budget.MaxAge != 30*24*time.Hour {
		t.Errorf("MaxAge = %v, want 720h", spec.Budget.MaxAge)
	}
	if spec.Budget.MaxBytes != 2*1024*1024*1024 {
		t.Errorf("MaxBytes = %d, want 2GiB", spec.Budget.MaxBytes)
	}
	if spec.Mode != PrunerBuiltin {
		t.Errorf("Mode = %q, want builtin", spec.Mode)
	}
	if spec.Target.Kind != TargetSQLiteTable || spec.Target.Table != "system_events" || spec.Target.TimeColumn != "occurred_at" {
		t.Errorf("Target = %+v, want the declared sqlite_table", spec.Target)
	}
	// The class defaults to data, where primary mutable state lives.
	if spec.Target.Class != storage.ClassData {
		t.Errorf("Class = %q, want data", spec.Target.Class)
	}
	if spec.Rationale == "" {
		t.Error("Rationale was dropped; a finding needs it to name what drives the volume")
	}
}

func TestParseManifestNoRetentionBlock(t *testing.T) {
	// A component that declares nothing must keep working unchanged.
	for _, manifest := range []string{`{}`, `{"retention":null}`, `{"service":{"name":"x"}}`} {
		specs, err := ParseManifest([]byte(manifest))
		if err != nil {
			t.Errorf("ParseManifest(%s): unexpected error %v", manifest, err)
		}
		if len(specs) != 0 {
			t.Errorf("ParseManifest(%s) = %d specs, want 0", manifest, len(specs))
		}
	}
}

func TestParseManifestOrdersBudgetsDeterministically(t *testing.T) {
	manifest := `{"retention":{"budgets":{
	  "zeta":{"target":{"kind":"directory","path":"z"},"max_bytes":"1GiB"},
	  "alpha":{"target":{"kind":"directory","path":"a"},"max_bytes":"1GiB"},
	  "mid":{"target":{"kind":"directory","path":"m"},"max_bytes":"1GiB"}}}}`
	for i := 0; i < 5; i++ {
		specs, err := ParseManifest([]byte(manifest))
		if err != nil {
			t.Fatalf("ParseManifest: %v", err)
		}
		got := []string{specs[0].Budget.Name, specs[1].Budget.Name, specs[2].Budget.Name}
		if got[0] != "alpha" || got[1] != "mid" || got[2] != "zeta" {
			t.Fatalf("budget order = %v, want sorted; map iteration leaked into cycle order", got)
		}
	}
}

func TestParseManifestRejections(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		wantErr  error
	}{
		{
			// The central gate: this is the autoheal failure mode, and it must
			// not be expressible as a compliant declaration.
			name:     "no bound",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"}}}}}`,
			wantErr:  ErrNoBound,
		},
		{
			name:     "max_bytes without a unit",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"},"max_bytes":"2000000"}}}}`,
			wantErr:  ErrUnknownUnit,
		},
		{
			name:     "max_bytes with a decimal unit",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"},"max_bytes":"2GB"}}}}`,
			wantErr:  ErrUnknownUnit,
		},
		{
			name:     "max_age without a unit",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"},"max_age":"30"}}}}`,
			wantErr:  ErrUnknownUnit,
		},
		{
			name:     "sqlite_table without time_column",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"sqlite_table","database":"a.sqlite","table":"t"},"max_age":"30d"}}}}`,
			wantErr:  ErrInvalidTarget,
		},
		{
			name:     "directory without path",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory"},"max_age":"30d"}}}}`,
			wantErr:  ErrInvalidTarget,
		},
		{
			name:     "unknown target kind",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"postgres_table","table":"t"},"max_age":"30d"}}}}`,
			wantErr:  ErrInvalidTarget,
		},
		{
			name:     "unknown storage class",
			manifest: `{"retention":{"budgets":{"b":{"target":{"kind":"directory","class":"scratch","path":"x"},"max_age":"30d"}}}}`,
			wantErr:  ErrInvalidTarget,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specs, err := ParseManifest([]byte(tc.manifest))
			if err == nil {
				t.Fatalf("expected error, got specs %+v", specs)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), `"b"`) {
				t.Errorf("error %q does not name the offending budget", err)
			}
		})
	}
}

func TestParseManifestUnknownPrunerMode(t *testing.T) {
	manifest := `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"},"max_age":"30d","pruner":"scenario"}}}}`
	if _, err := ParseManifest([]byte(manifest)); err == nil {
		t.Fatal("expected an error for an unknown pruner mode")
	}
}

func TestParseManifestDefaultsPrunerToBuiltin(t *testing.T) {
	manifest := `{"retention":{"budgets":{"b":{"target":{"kind":"directory","path":"x"},"max_age":"30d"}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if specs[0].Mode != PrunerBuiltin {
		t.Fatalf("Mode = %q, want builtin so the common case declares nothing extra", specs[0].Mode)
	}
}

func TestParseManifestStorageEntryDesugarsAndRejectsConflicts(t *testing.T) {
	manifest := `{"storage":{"entries":{"snapshots":{"rung":"owned","path":"snapshots","kind":"dir","class":"data","budget":{"max_bytes":"1GiB"}}}},"retention":{"budgets":{"legacy":{"target":{"kind":"directory","path":"snapshots"},"max_age":"30d"}}}}`
	if _, err := ParseManifest([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "snapshots") || !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("expected named same-path conflict, got %v", err)
	}
	manifest = `{"storage":{"entries":{"snapshots":{"rung":"owned","path":"snapshots","kind":"dir","class":"data","budget":{"max_bytes":"1GiB"}}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].Target.Kind != TargetDirectory || specs[0].Target.Path != "snapshots" {
		t.Fatalf("unexpected desugared specs: %+v", specs)
	}
}

// A token- or home-rooted storage path is outside the class root. The class-root
// engine must leave it to its owner and storage-manager instead of pruning a
// literal "$USER_HOME/..." directory under the scenario's data root.
func TestParseManifestSkipsTokenAndHomeRootedEntryPaths(t *testing.T) {
	for _, path := range []string{"$USER_HOME/.vrooli/plan-artifacts", "$USER_DATA_DIR/vrooli/x/retired", "~/.vrooli/x"} {
		manifest := `{"storage":{"entries":{"external":{"rung":"owned","kind":"dir","class":"data","path":"` + path + `","regenerable":false,"budget":{"max_bytes":"60GiB","max_age":"30d"}}}}}`
		specs, err := ParseManifest([]byte(manifest))
		if err != nil {
			t.Fatalf("%s: ParseManifest: %v", path, err)
		}
		if len(specs) != 0 {
			t.Fatalf("%s: class-root engine must not own an external path, got %+v", path, specs)
		}
	}
}

func TestParseManifestCoalescesCompatibleSameNameDeclarations(t *testing.T) {
	manifest := `{"storage":{"entries":{"phase_cache":{"rung":"owned","kind":"dir","class":"cache","budget":{"max_age":"7d","max_bytes":"2GiB","rationale":"Canonical owner budget."}}}},"retention":{"budgets":{"phase_cache":{"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"2GiB","pruner":"builtin","rationale":"Compatible legacy declaration."}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want one coalesced phase_cache budget: %+v", len(specs), specs)
	}
	spec := specs[0]
	if spec.Budget.Name != "phase_cache" || spec.Target.Kind != TargetDirectory || spec.Target.Class != storage.ClassCache || spec.Target.Path != "." {
		t.Fatalf("coalesced target = %+v / %+v", spec.Budget, spec.Target)
	}
	if spec.Budget.MaxAge != 7*24*time.Hour || spec.Budget.MaxBytes != 2*1024*1024*1024 || spec.Mode != PrunerBuiltin {
		t.Fatalf("coalesced policy = %+v mode=%q", spec.Budget, spec.Mode)
	}
	if spec.Rationale != "Canonical owner budget." {
		t.Fatalf("rationale = %q, want storage entry's owner rationale", spec.Rationale)
	}
}

func TestParseManifestRejectsDisagreeingSameNameDeclarations(t *testing.T) {
	base := `{"storage":{"entries":{"phase_cache":{"kind":"dir","class":"cache","budget":{"max_age":"7d","max_bytes":"2GiB"}}}},"retention":{"budgets":{"phase_cache":{"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"2GiB"}}}}`
	cases := []struct {
		name     string
		manifest string
		want     string
	}{
		{
			name:     "target mismatch",
			manifest: strings.Replace(base, `"class":"cache","path":"."`, `"class":"data","path":"."`, 1),
			want:     "targets differ",
		},
		{
			name: "bounds mismatch",
			manifest: strings.Replace(base,
				`"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"2GiB"`,
				`"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"3GiB"`, 1),
			want: "bounds differ",
		},
		{
			name: "pruner mismatch",
			manifest: strings.Replace(base,
				`"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"2GiB"`,
				`"target":{"kind":"directory","class":"cache","path":"."},"max_age":"7d","max_bytes":"2GiB","pruner":"custom"`, 1),
			want: "pruner modes differ",
		},
		{
			name:     "unsupported storage declaration",
			manifest: strings.Replace(base, `"kind":"dir","class":"cache"`, `"kind":"file","class":"cache"`, 1),
			want:     "not a budgeted directory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseManifest([]byte(tc.manifest)); err == nil || !strings.Contains(err.Error(), "phase_cache") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseManifest error = %v, want phase_cache %s conflict", err, tc.want)
			}
		})
	}
}

func TestTargetResolveUsesStorageClassRoots(t *testing.T) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	root := t.TempDir()
	opts := storage.Options{ScenarioID: "vrooli-autoheal", RootOverride: root}

	target := Target{Kind: TargetSQLiteTable, Class: storage.ClassData, Database: "autoheal.sqlite", Table: "e", TimeColumn: "at"}
	got, err := target.Resolve(resolver, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(got, root) {
		t.Fatalf("resolved %q outside the override root %q", got, root)
	}
	if !strings.HasSuffix(got, "autoheal.sqlite") {
		t.Fatalf("resolved %q does not end at the declared database", got)
	}

	// A path escaping its class root would let one component prune another's
	// data, which is exactly what the resolver exists to prevent.
	escaping := Target{Kind: TargetDirectory, Class: storage.ClassData, Path: "../../elsewhere"}
	if _, err := escaping.Resolve(resolver, opts); err == nil {
		t.Fatal("expected a traversal outside the class root to be rejected")
	}
}

func TestTargetResolveIsolatesVariants(t *testing.T) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	root := t.TempDir()
	target := Target{Kind: TargetDirectory, Class: storage.ClassData, Path: "snapshots"}

	live, err := target.Resolve(resolver, storage.Options{ScenarioID: "architecture-cartographer", RootOverride: root})
	if err != nil {
		t.Fatalf("resolve live: %v", err)
	}
	shadow, err := target.Resolve(resolver, storage.Options{ScenarioID: "architecture-cartographer_shadow", RootOverride: root})
	if err != nil {
		t.Fatalf("resolve shadow: %v", err)
	}
	if live == shadow {
		t.Fatalf("live and shadow resolved to the same path %q; a shadow would prune live's data", live)
	}
}

func TestParseManifestPreservesCustomStoragePruner(t *testing.T) {
	specs, err := ParseManifest([]byte(`{"storage":{"entries":{"staging":{"kind":"dir","class":"cache","subpath":"staging","reclaim":{"pruner":"custom"},"budget":{"max_age":"7d","max_bytes":"20GiB"}}}}}`))
	if err != nil || len(specs) != 1 || specs[0].Mode != PrunerCustom {
		t.Fatalf("specs=%+v err=%v", specs, err)
	}
}

func TestParseManifestDelegatesCustomTestRunsWithoutBuiltinSpec(t *testing.T) {
	manifest := `{"storage":{"entries":{"run_artifacts":{"kind":"dir","class":"test_runs","regenerable":true,"reclaim":{"pruner":"custom"},"budget":{"max_age":"30d","max_bytes":"20GiB"}}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest custom test_runs: %v", err)
	}
	if len(specs) != 0 {
		t.Fatalf("ParseManifest returned generic specs for owner-retained run evidence: %#v", specs)
	}
}

func TestParseManifestRefusesBuiltinTestRunsRetention(t *testing.T) {
	for _, manifest := range []string{
		`{"storage":{"entries":{"run_artifacts":{"kind":"dir","class":"test_runs","regenerable":true,"budget":{"max_age":"30d"}}}}}`,
		`{"retention":{"budgets":{"run_artifacts":{"target":{"kind":"directory","class":"test_runs","path":"."},"max_age":"30d"}}}}`,
	} {
		if _, err := ParseManifest([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "builtin retention") {
			t.Fatalf("ParseManifest(%s) error=%v, want explicit builtin test_runs refusal", manifest, err)
		}
	}
}

func TestParseManifestRejectsGenericDuplicateForCustomTestRuns(t *testing.T) {
	manifest := `{"storage":{"entries":{"run_artifacts":{"kind":"dir","class":"test_runs","regenerable":true,"reclaim":{"pruner":"custom"},"budget":{"max_age":"30d","max_bytes":"20GiB"}}}},"retention":{"budgets":{"run_artifacts":{"target":{"kind":"directory","class":"test_runs","path":"."},"max_age":"30d"}}}}`
	if _, err := ParseManifest([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "cannot also declare a generic retention budget") {
		t.Fatalf("ParseManifest custom+generic test_runs error=%v", err)
	}
}

func TestConfigureBudgetRequiresPositiveOverrides(t *testing.T) {
	initial := Budget{Name: "staging", MaxAge: 7 * 24 * time.Hour, MaxBytes: 20 << 30}
	for _, bad := range []string{"0", "-1", "nonsense"} {
		if _, err := ConfigureBudget(initial, bad, ""); err == nil {
			t.Fatalf("accepted age %q", bad)
		}
		if _, err := ConfigureBudget(initial, "", bad); err == nil {
			t.Fatalf("accepted bytes %q", bad)
		}
	}
	got, err := ConfigureBudget(initial, "2d", "1GiB")
	if err != nil || got.MaxAge != 48*time.Hour || got.MaxBytes != 1<<30 {
		t.Fatalf("budget=%+v err=%v", got, err)
	}
	got, err = ConfigureBudget(initial, "", "")
	if err != nil || got != initial {
		t.Fatalf("defaults changed: %+v %v", got, err)
	}
}

// A budget on non-regenerable data is an alarm, never a builtin age or byte
// pruner: deleting the oldest files of a durable data root to meet a ceiling
// would destroy the state the budget exists to watch.
func TestParseManifestBudgetOnDurableEntryIsAlarmOnly(t *testing.T) {
	entry := func(regenerable, reclaim string) string {
		return `{"storage":{"entries":{"data":{"rung":"owned","kind":"dir","class":"data"` + regenerable + reclaim +
			`,"budget":{"max_bytes":"5GiB","max_age":"30d"}}}}}`
	}
	cases := []struct {
		name     string
		manifest string
		want     []PrunerMode
	}{
		{"durable builtin", entry(`,"regenerable":false`, ""), nil},
		{"durable explicit builtin", entry(`,"regenerable":false`, `,"reclaim":{"pruner":"builtin"}`), nil},
		{"durable owner operation", entry(`,"regenerable":false`, `,"reclaim":{"operation":"/api/v1/storage/reclaim"}`), nil},
		{"durable custom", entry(`,"regenerable":false`, `,"reclaim":{"pruner":"custom"}`), []PrunerMode{PrunerCustom}},
		{"regenerable builtin", entry(`,"regenerable":true`, ""), []PrunerMode{PrunerBuiltin}},
		{"regenerable custom", entry(`,"regenerable":true`, `,"reclaim":{"pruner":"custom"}`), []PrunerMode{PrunerCustom}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specs, err := ParseManifest([]byte(tc.manifest))
			if err != nil {
				t.Fatalf("ParseManifest: %v", err)
			}
			if len(specs) != len(tc.want) {
				t.Fatalf("specs = %+v, want modes %v", specs, tc.want)
			}
			for i, spec := range specs {
				if spec.Mode != tc.want[i] {
					t.Fatalf("spec %d mode = %q, want %q", i, spec.Mode, tc.want[i])
				}
			}
		})
	}
}

// An explicit retention budget is the owner choosing a pruner; marking the
// storage entry durable does not silently drop it.
func TestParseManifestKeepsExplicitBudgetBesideDurableEntry(t *testing.T) {
	manifest := `{"storage":{"entries":{"events":{"rung":"owned","kind":"dir","class":"data","subpath":"events","regenerable":false,"budget":{"max_bytes":"1GiB"}}}},` +
		`"retention":{"budgets":{"events":{"target":{"kind":"directory","class":"data","path":"events"},"max_age":"7d"}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(specs) != 1 || specs[0].Mode != PrunerBuiltin || specs[0].Budget.MaxAge.Hours() != 168 {
		t.Fatalf("explicit budget lost: %+v", specs)
	}
}
