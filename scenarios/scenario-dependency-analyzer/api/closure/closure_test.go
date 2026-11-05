package closure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolverProducesStableClosedInventory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := NewResolver().Resolve(root, "demo", "sha256:source")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewResolver().Resolve(root, "demo", "sha256:source")
	if err != nil {
		t.Fatal(err)
	}
	if a.ClosureDigest != b.ClosureDigest || len(a.Files) != 1 {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
}

func TestSelectionClosureCanonicalizesUnionAndRetainsReasons(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "alpha", `{"version":"1.0.0","service":{"name":"alpha"},"dependencies":{"scenarios":{"shared":{"required":true}},"resources":{"database":{"required":true,"versionRange":"^1.0.0"}}}}`)
	writeScenario(t, root, "beta", `{"version":"1.0.0","service":{"name":"beta"},"dependencies":{"scenarios":{"shared":{"required":true}},"resources":{"database":{"required":true,"versionRange":">=1.0.0 <2.0.0"}}}}`)
	writeScenario(t, root, "shared", `{"version":"1.0.0","service":{"name":"shared"}}`)
	writeResource(t, root, "database", `{"name":"database","platforms":{"linux-amd64":"supported"}}`)

	resolver := NewResolver()
	a, err := resolver.ResolveSelection(root, []string{"beta", "alpha"}, Target{OS: "linux", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolver.ResolveSelection(root, []string{"alpha", "beta"}, Target{OS: "linux", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ClosureDigest != b.ClosureDigest || !a.Complete || len(a.Items) != 4 {
		t.Fatalf("canonical union differs: a=%+v b=%+v", a, b)
	}
	var shared Item
	for _, item := range a.Items {
		if item.ID == "scenario:shared" {
			shared = item
		}
	}
	if len(shared.Reasons) != 2 || shared.Reasons[0].Parent != "alpha" || shared.Reasons[1].Parent != "beta" {
		t.Fatalf("shared inclusion reasons = %+v", shared.Reasons)
	}
	var database Item
	for _, item := range a.Items {
		if item.ID == "resource:database" {
			database = item
		}
	}
	if len(database.Constraints) != 2 || len(a.Files) != 7 {
		t.Fatalf("database/files = %+v files=%d", database, len(a.Files))
	}
}

func TestSelectionClosureFailsBeforeShipmentForMissingDeclarationAndCycle(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "app", `{"service":{"name":"app"},"dependencies":{"scenarios":{"missing":{"required":true}}}}`)
	_, err := NewResolver().ResolveSelection(root, []string{"app"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "missing required scenario declaration") || !strings.Contains(err.Error(), "app -> missing") {
		t.Fatalf("missing declaration error = %v", err)
	}

	writeScenario(t, root, "first", `{"service":{"name":"first"},"dependencies":{"scenarios":{"second":{"required":true}}}}`)
	writeScenario(t, root, "second", `{"service":{"name":"second"},"dependencies":{"scenarios":{"first":{"required":true}}}}`)
	_, err = NewResolver().ResolveSelection(root, []string{"first"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "first -> second -> first") {
		t.Fatalf("cycle error = %v", err)
	}
}

func TestSelectionClosureRejectsVersionPlatformAndSafeguardGaps(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "one", `{"service":{"name":"one"},"dependencies":{"resources":{"engine":{"required":true,"versionRange":"^1.0.0"}}}}`)
	writeScenario(t, root, "two", `{"service":{"name":"two"},"dependencies":{"resources":{"engine":{"required":true,"versionRange":">=2.0.0"}}}}`)
	writeResource(t, root, "engine", `{"name":"engine","platforms":{"linux-amd64":"supported"}}`)
	_, err := NewResolver().ResolveSelection(root, []string{"one", "two"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "incompatible version constraints") {
		t.Fatalf("version error = %v", err)
	}

	writeScenario(t, root, "mac-only", `{"service":{"name":"mac-only"},"dependencies":{"resources":{"engine":{"required":true}}}}`)
	writeResource(t, root, "engine-mac", `{"name":"engine-mac","platforms":{"macos":"supported"}}`)
	writeScenario(t, root, "mac-capability", `{"service":{"name":"mac-capability"},"dependencies":{"resources":{"engine-mac":{"required":true}}}}`)
	_, err = NewResolver().ResolveSelection(root, []string{"mac-capability"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "incompatible with linux/amd64") {
		t.Fatalf("platform error = %v", err)
	}

	writeScenario(t, root, "partial-capability", `{"service":{"name":"partial-capability"},"dependencies":{"resources":{"partial-engine":{"required":true}}}}`)
	writeResource(t, root, "partial-engine", `{"name":"partial-engine","platforms":{"linux-amd64":"partial"}}`)
	_, err = NewResolver().ResolveSelection(root, []string{"partial-capability"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "incompatible with linux/amd64") {
		t.Fatalf("partial platform error = %v", err)
	}

	writeScenario(t, root, "guarded", `{"service":{"name":"guarded"},"hostSafeguards":[{"name":"missing-guard","required":true}]}`)
	_, err = NewResolver().ResolveSelection(root, []string{"guarded"}, Target{OS: "linux", Architecture: "amd64"})
	if err == nil || !strings.Contains(err.Error(), "missing required host_safeguard declaration") {
		t.Fatalf("safeguard error = %v", err)
	}
}

func TestSelectionClosureCarriesOpaqueCredentialReferenceOnly(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "credentialed", `{"service":{"name":"credentialed"},"credentials":{"descriptors":[{"logical_id":"provider/api","field":"token","required":true}]}}`)
	if err := os.WriteFile(filepath.Join(root, "scenarios", "credentialed", ".env"), []byte("provider/api=secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := NewResolver().ResolveSelection(root, []string{"credentialed"}, Target{OS: "linux", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Items[0].Kind != "credential_ref" && result.Items[1].Kind != "credential_ref" {
		t.Fatalf("credential item = %+v", result.Items)
	}
	for _, item := range result.Items {
		if strings.Contains(item.ID, "secret") || strings.Contains(item.Name, "secret") {
			t.Fatalf("secret leaked into closure: %+v", item)
		}
	}
}

func writeScenario(t *testing.T, root, name, manifest string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(path)), "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeResource(t *testing.T, root, name, manifest string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "resource.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}
