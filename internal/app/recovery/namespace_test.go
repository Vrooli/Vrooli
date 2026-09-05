//nolint:goconst // test data deliberately reuses stable instance fixtures.
package recovery

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestNamespaceShadowDerivesSSOTNames(t *testing.T) {
	home := t.TempDir()
	svc := Service{HomeDir: func() (string, error) { return home, nil }}

	out, err := svc.Namespace(NamespaceRequest{Scenario: "swarm-manager", Variant: "shadow"})
	if err != nil {
		t.Fatalf("Namespace: %v", err)
	}
	if out.Variant != "shadow" { //nolint:goconst // fixture variant
		t.Fatalf("variant = %q, want shadow", out.Variant)
	}
	if out.InstanceKey != "swarm-manager@shadow" {
		t.Fatalf("instanceKey = %q, want swarm-manager@shadow", out.InstanceKey)
	}
	// Postgres maps "-" and the variant join to "_"; "@" never reaches a DB name.
	if out.PostgresDb != "vrooli_swarm_manager_shadow" {
		t.Fatalf("postgresDb = %q, want vrooli_swarm_manager_shadow", out.PostgresDb)
	}
	if out.StorageNamespace != "swarm-manager_shadow" {
		t.Fatalf("storageNamespace = %q, want swarm-manager_shadow", out.StorageNamespace)
	}
	if out.DataDirName != "swarm-manager@shadow" {
		t.Fatalf("dataDirName = %q, want swarm-manager@shadow", out.DataDirName)
	}
	// The data dir is an absolute path ending in the variant subdirectory under
	// the conventional <data-root>/vrooli/<dataDirName> layout.
	if !filepath.IsAbs(out.DataDir) {
		t.Fatalf("dataDir = %q, want absolute", out.DataDir)
	}
	if base := filepath.Base(out.DataDir); base != "swarm-manager@shadow" {
		t.Fatalf("dataDir base = %q, want swarm-manager@shadow", base)
	}
	if parent := filepath.Base(filepath.Dir(out.DataDir)); parent != storageAppID {
		t.Fatalf("dataDir parent = %q, want %q", parent, storageAppID)
	}
}

func TestNamespaceLiveIsUnsuffixed(t *testing.T) {
	home := t.TempDir()
	svc := Service{HomeDir: func() (string, error) { return home, nil }}

	out, err := svc.Namespace(NamespaceRequest{Scenario: "swarm-manager", Variant: "live"})
	if err != nil {
		t.Fatalf("Namespace: %v", err)
	}
	if out.Variant != "live" { //nolint:goconst // fixture variant
		t.Fatalf("variant = %q, want live", out.Variant)
	}
	if out.InstanceKey != "swarm-manager" {
		t.Fatalf("instanceKey = %q, want bare slug for live", out.InstanceKey)
	}
	if out.PostgresDb != "vrooli_swarm_manager" {
		t.Fatalf("postgresDb = %q, want vrooli_swarm_manager", out.PostgresDb)
	}
	if out.StorageNamespace != "swarm-manager" {
		t.Fatalf("storageNamespace = %q, want swarm-manager", out.StorageNamespace)
	}
	if base := filepath.Base(out.DataDir); base != "swarm-manager" {
		t.Fatalf("dataDir base = %q, want swarm-manager", base)
	}
}

func TestNamespaceEmptyVariantNormalizesToLive(t *testing.T) {
	home := t.TempDir()
	svc := Service{HomeDir: func() (string, error) { return home, nil }}

	out, err := svc.Namespace(NamespaceRequest{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Namespace: %v", err)
	}
	if out.Variant != "live" || out.InstanceKey != "demo" {
		t.Fatalf("empty variant should normalize to live: %+v", out)
	}
}

func TestNamespaceRequiresScenario(t *testing.T) {
	svc := Service{HomeDir: func() (string, error) { return t.TempDir(), nil }}
	if _, err := svc.Namespace(NamespaceRequest{Variant: "shadow"}); err == nil {
		t.Fatalf("expected error for blank scenario")
	}
}

func TestNamespaceOmitsDataDirWhenHomeUnresolvable(t *testing.T) {
	// An unresolvable home must not fail the call — Postgres alone is a usable
	// mapping, so the data location is simply omitted.
	svc := Service{HomeDir: func() (string, error) { return "", errors.New("no home") }}
	out, err := svc.Namespace(NamespaceRequest{Scenario: "demo", Variant: "shadow"})
	if err != nil {
		t.Fatalf("Namespace should tolerate an unresolvable home: %v", err)
	}
	if out.DataDir != "" {
		t.Fatalf("dataDir = %q, want empty when home is unresolvable", out.DataDir)
	}
	if out.PostgresDb != "vrooli_demo_shadow" {
		t.Fatalf("postgresDb = %q, want vrooli_demo_shadow", out.PostgresDb)
	}
}
