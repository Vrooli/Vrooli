package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeprecateResourceArchivesAndRemovesActiveState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	writeRegistryEntry(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")

	controller := NewController(root, home)
	report, err := controller.DeprecateResource("fixture")
	if err != nil {
		t.Fatalf("DeprecateResource: %v", err)
	}
	if !report.Archived {
		t.Fatal("expected archived report")
	}
	if _, err := os.Stat(filepath.Join(root, "resources", "fixture")); !os.IsNotExist(err) {
		t.Fatalf("expected resource directory to be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "resource-registry", "fixture.json")); !os.IsNotExist(err) {
		t.Fatalf("expected registry file to be removed, got %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service config: %v", err)
	}
	if strings.Contains(string(data), `"fixture"`) {
		t.Fatalf("expected config entry to be removed, got %s", string(data))
	}
	deprecated, err := controller.ListDeprecatedResources()
	if err != nil {
		t.Fatalf("ListDeprecatedResources: %v", err)
	}
	if len(deprecated) != 1 || deprecated[0].Name != "fixture" {
		t.Fatalf("deprecated = %#v", deprecated)
	}
	if _, err := os.Stat(filepath.Join(report.ArchiveDir, "files", "resource", "fixture", "cli.sh")); err != nil {
		t.Fatalf("expected archived cli.sh: %v", err)
	}
}

func TestRestoreDeprecatedResourceWritesQuarantinedCopy(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	writeResourceCLI(t, root, "fixture")

	controller := NewController(root, home)
	if _, err := controller.DeprecateResource("fixture"); err != nil {
		t.Fatalf("DeprecateResource: %v", err)
	}

	report, err := controller.RestoreDeprecatedResource("fixture")
	if err != nil {
		t.Fatalf("RestoreDeprecatedResource: %v", err)
	}
	if !report.Restored {
		t.Fatal("expected restore report")
	}
	if _, err := os.Stat(filepath.Join(report.RestoredPath, "resource", "fixture", "cli.sh")); err != nil {
		t.Fatalf("expected restored cli.sh copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "resources", "fixture")); !os.IsNotExist(err) {
		t.Fatalf("restore should remain quarantined outside active resources/, got %v", err)
	}
}

func TestGarbageCollectDeprecatedArchivesPurgesExpiredEntries(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	archiveDir := filepath.Join(home, ".vrooli", "archive", "resources", "old-fixture")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", archiveDir, err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	payload := DeprecatedResourceList{
		Resources: []DeprecatedResource{
			{
				Name:                "fixture",
				DeprecatedAt:        "2026-01-01",
				Reason:              "test",
				ArchivePath:         archiveDir,
				ArchiveHash:         "abc",
				RetentionPolicyDays: 90,
				RestoreSupported:    true,
				PurgeAfter:          "2026-01-02",
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "deprecated-resources.json"), data, 0o644); err != nil {
		t.Fatalf("write deprecated metadata: %v", err)
	}

	report, err := NewController(root, home).GarbageCollectDeprecatedArchives(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GarbageCollectDeprecatedArchives: %v", err)
	}
	if len(report.Removed) != 1 {
		t.Fatalf("removed = %#v", report.Removed)
	}
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Fatalf("expected archive dir to be removed, got %v", err)
	}
	items, err := NewController(root, home).ListDeprecatedResources()
	if err != nil {
		t.Fatalf("ListDeprecatedResources: %v", err)
	}
	if items[0].PurgedAt == "" {
		t.Fatalf("expected purged_at to be recorded, got %#v", items[0])
	}
}

func TestDiscoverExcludesDeprecatedResources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "active", true)
	writeResourceConfig(t, root, "deprecated-fixture", false)
	writeResourceManifest(t, root, "active", `{
  "name": "active",
  "display_name": "Active",
  "description": "Active manifest-backed resource",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "runtime": {
    "image": "active:latest"
  }
}`)
	writeResourceCLI(t, root, "active")
	writeResourceCLI(t, root, "deprecated-fixture")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	payload := DeprecatedResourceList{
		Resources: []DeprecatedResource{
			{
				Name:                "deprecated-fixture",
				DeprecatedAt:        "2026-04-11",
				Reason:              "test",
				ArchivePath:         filepath.Join(home, ".vrooli", "archive", "resources", "deprecated-fixture"),
				ArchiveHash:         "abc",
				RetentionPolicyDays: 90,
				RestoreSupported:    true,
				PurgeAfter:          "2026-07-10",
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "deprecated-resources.json"), data, 0o644); err != nil {
		t.Fatalf("write deprecated metadata: %v", err)
	}

	items, err := NewController(root, home).Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) != 1 || items[0].Name != "active" {
		t.Fatalf("discover items = %#v", items)
	}
}

func writeRegistryEntry(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, ".vrooli", "resource-registry", name+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("{\"name\":\""+name+"\"}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceCLI(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "cli.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
