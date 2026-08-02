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
