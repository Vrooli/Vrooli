package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOwnerInventoryDiscoversAllOwnerKindsDeterministically(t *testing.T) {
	root := t.TempDir()
	writeOwnerFixture(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"storage":{"entries":{"data":{"rung":"owned","path":"db","kind":"file","class":"data"}}},"retention":{"budgets":{"events":{"target":{"kind":"directory","path":"events"},"max_bytes":"1MiB"}}}}`)
	writeOwnerFixture(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres","storage":{"entries":{"main":{"rung":"pinned","path":{"linux":"$USER_HOME/data","windows":"%USERPROFILE%\\data"},"kind":"dir"}}},"durable_data":{"base":"$USER_HOME","entries":{"config":{"path":"config","regenerable":false}}}}`)
	writeOwnerFixture(t, filepath.Join(root, "internal", "tools", "go", "tool.json"), `{"name":"go","storage":{"entries":{"cache":{"rung":"relocatable","path":"$USER_CACHE_DIR/go-build","kind":"dir","relocation":{"key":"GOCACHE","scope":"process-tree"}}}}}`)
	writeOwnerFixture(t, filepath.Join(root, "internal", "safeguards", "clock", "safeguard.json"), `{"name":"clock","description":"clock","handler":"clock","privilege":"none","bundling":"prohibited","deployment":{"profiles":{}}}`)
	writeOwnerFixture(t, filepath.Join(root, "internal", "safeguards", "unmanifested", "handler.go"), "package unmanifested")

	first, err := LoadOwnerInventory(InventoryOptions{RepoRoot: root, Platform: PlatformLinux, PlatformSeams: PlatformSeams{
		UserHomeDir:   func() (string, error) { return "/home/test", nil },
		UserCacheDir:  func() (string, error) { return "/home/test/.cache", nil },
		UserConfigDir: func() (string, error) { return "/home/test/.config", nil },
	}})
	if err != nil {
		t.Fatalf("LoadOwnerInventory: %v", err)
	}
	second, err := LoadOwnerInventory(InventoryOptions{RepoRoot: root, Platform: PlatformLinux})
	if err != nil {
		t.Fatalf("LoadOwnerInventory second: %v", err)
	}
	if len(first.Owners) != 4 {
		t.Fatalf("owners = %d, want 4", len(first.Owners))
	}
	if first.Owners[0].Kind != OwnerResource || first.Owners[0].ID != "postgres" {
		t.Fatalf("owners are not sorted by kind/id: %#v", first.Owners)
	}
	resource := ownerByKind(first.Owners, OwnerResource)
	if resource == nil || !resource.StorageDeclared || len(resource.StorageEntries) != 1 || resource.StorageEntries[0].Path.ByOS[PlatformLinux] == nil {
		t.Fatalf("resource storage declaration was not normalized: %#v", resource)
	}
	if len(resource.DurableData) != 1 || resource.DurableData[0].Regenerable {
		t.Fatalf("durable_data was not normalized: %#v", resource.DurableData)
	}
	scenario := ownerByKind(first.Owners, OwnerScenario)
	if scenario == nil || !scenario.StorageDeclared || len(scenario.Retention) != 1 || scenario.Retention[0].Name != "events" {
		t.Fatalf("legacy retention was not normalized: %#v", scenario)
	}
	if !hasFinding(first.Findings, "missing_manifest", OwnerSafeguard, "unmanifested") {
		t.Fatalf("missing safeguard manifest finding absent: %#v", first.Findings)
	}
	if len(first.Findings) != len(second.Findings) {
		t.Fatalf("finding order/count changed between runs: %d vs %d", len(first.Findings), len(second.Findings))
	}
}

func TestLoadOwnerInventoryPreservesExplicitEmptyDeclaration(t *testing.T) {
	root := t.TempDir()
	writeOwnerFixture(t, filepath.Join(root, "scenarios", "empty", ".vrooli", "service.json"), `{"service":{"name":"empty"},"storage":{"entries":{}}}`)
	inventory, err := LoadOwnerInventory(InventoryOptions{RepoRoot: root, Platform: PlatformLinux})
	if err != nil {
		t.Fatalf("LoadOwnerInventory: %v", err)
	}
	owner := ownerByKind(inventory.Owners, OwnerScenario)
	if owner == nil || !owner.StorageDeclared || len(owner.StorageEntries) != 0 {
		t.Fatalf("explicit empty declaration = %#v", owner)
	}
	if len(inventory.Findings) != 0 {
		t.Fatalf("explicit empty declaration findings = %#v", inventory.Findings)
	}
}

func TestLoadOwnerInventoryReportsMalformedManifestWithContext(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "broken", "resource.json")
	writeOwnerFixture(t, path, "{")
	inventory, err := LoadOwnerInventory(InventoryOptions{RepoRoot: root})
	if err != nil {
		t.Fatalf("LoadOwnerInventory: %v", err)
	}
	if !hasFinding(inventory.Findings, "malformed_manifest", OwnerResource, "") {
		t.Fatalf("malformed manifest finding absent: %#v", inventory.Findings)
	}
	if inventory.Findings[0].ManifestPath != path {
		t.Fatalf("finding path = %q, want %q", inventory.Findings[0].ManifestPath, path)
	}
}

func TestLoadOwnerInventoryFlagsInvalidRelocatableDeclaration(t *testing.T) {
	root := t.TempDir()
	writeOwnerFixture(t, filepath.Join(root, "internal", "tools", "go", "tool.json"), `{"name":"go","storage":{"entries":{"cache":{"rung":"relocatable","path":"$USER_CACHE_DIR/go-build","kind":"dir"}}}}`)
	inventory, err := LoadOwnerInventory(InventoryOptions{RepoRoot: root, Platform: PlatformLinux})
	if err != nil {
		t.Fatalf("LoadOwnerInventory: %v", err)
	}
	if !hasFinding(inventory.Findings, "missing_relocation", OwnerTool, "go") {
		t.Fatalf("missing relocation finding absent: %#v", inventory.Findings)
	}
}

func writeOwnerFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasFinding(findings []InventoryFinding, code string, kind OwnerKind, id string) bool {
	for _, finding := range findings {
		if finding.Code == code && finding.OwnerKind == kind && (id == "" || finding.OwnerID == id) {
			return true
		}
	}
	return false
}

func ownerByKind(owners []OwnerManifest, kind OwnerKind) *OwnerManifest {
	for i := range owners {
		if owners[i].Kind == kind {
			return &owners[i]
		}
	}
	return nil
}
