package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchiveResourceToBlueprintArchivesAndRemovesActiveState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")
	writeRegistryEntry(t, root, "fixture")

	controller := NewController(root, home)
	report, err := controller.ArchiveResourceToBlueprint("fixture")
	if err != nil {
		t.Fatalf("ArchiveResourceToBlueprint: %v", err)
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
	items, err := controller.ListBlueprintArchivedResources()
	if err != nil {
		t.Fatalf("ListBlueprintArchivedResources: %v", err)
	}
	if len(items) != 1 || items[0].Name != "fixture" {
		t.Fatalf("items = %#v", items)
	}
	if _, err := os.Stat(filepath.Join(report.ArchiveDir, "files", "resource", "fixture", "cli.sh")); err != nil {
		t.Fatalf("expected archived cli.sh: %v", err)
	}
}

func TestArchiveResourceToBlueprintRequiresBlueprint(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceCLI(t, root, "fixture")

	_, err := NewController(root, home).ArchiveResourceToBlueprint("fixture")
	if err == nil || !strings.Contains(err.Error(), "does not have a blueprint record") {
		t.Fatalf("err = %v", err)
	}
}

func TestArchiveResourceToBlueprintRejectsEnabledProjectResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceConfig(t, root, "fixture", true)
	writeResourceCLI(t, root, "fixture")

	_, err := NewController(root, home).ArchiveResourceToBlueprint("fixture")
	if err == nil || !strings.Contains(err.Error(), "still active in .vrooli/service.json") {
		t.Fatalf("err = %v", err)
	}
}

func TestArchiveResourceToBlueprintRejectsScenarioReferences(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")
	writeScenarioResourceManifest(t, root, "alpha", "fixture")

	_, err := NewController(root, home).ArchiveResourceToBlueprint("fixture")
	if err == nil || !strings.Contains(err.Error(), "still referenced by 1 scenario manifest") {
		t.Fatalf("err = %v", err)
	}
}

func TestRestoreBlueprintArchivedResourceWritesQuarantinedCopy(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")

	controller := NewController(root, home)
	if _, err := controller.ArchiveResourceToBlueprint("fixture"); err != nil {
		t.Fatalf("ArchiveResourceToBlueprint: %v", err)
	}
	report, err := controller.RestoreBlueprintArchivedResource("fixture")
	if err != nil {
		t.Fatalf("RestoreBlueprintArchivedResource: %v", err)
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

func TestGarbageCollectBlueprintArchivesPurgesExpiredEntries(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	archiveDir := filepath.Join(home, ".vrooli", "archive", "resources", "old-fixture")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", archiveDir, err)
	}
	writeBlueprintArchivedMetadata(t, root, BlueprintArchivedResource{
		Name:                "fixture",
		ArchivedAt:          "2026-01-01",
		Reason:              "test",
		BlueprintName:       "fixture",
		ArchivePath:         archiveDir,
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-01-02",
	})

	report, err := NewController(root, home).GarbageCollectBlueprintArchives(time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GarbageCollectBlueprintArchives: %v", err)
	}
	if len(report.Removed) != 1 {
		t.Fatalf("removed = %#v", report.Removed)
	}
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Fatalf("expected archive dir to be removed, got %v", err)
	}
	items, err := NewController(root, home).ListBlueprintArchivedResources()
	if err != nil {
		t.Fatalf("ListBlueprintArchivedResources: %v", err)
	}
	if items[0].PurgedAt == "" {
		t.Fatalf("expected purged_at to be recorded, got %#v", items[0])
	}
}

func TestStatusReportsBlueprintArchivedResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchivedMetadata(t, root, BlueprintArchivedResource{
		Name:                "fixture",
		ArchivedAt:          "2026-04-11",
		Reason:              "test",
		BlueprintName:       "fixture",
		ArchivePath:         filepath.Join(home, ".vrooli", "archive", "resources", "fixture"),
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-07-10",
	})

	_, err := NewController(root, home).Status("fixture", true)
	if err == nil || !strings.Contains(err.Error(), "blueprint-archived") {
		t.Fatalf("err = %v", err)
	}
}

func writeBlueprintArchiveFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, ".vrooli", "resource-blueprints", name+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	contents := `{
  "name": "` + name + `",
  "display_name": "Fixture",
  "category": "testing",
  "summary": "Fixture resource",
  "why_it_matters": "Used in tests.",
  "when_to_use": ["Testing archive lifecycle"],
  "integration_kind": "docker-service",
  "platform_support": {
    "portability_tier": "partial",
    "notes": "Fixture",
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "suggested_template": "docker-service",
  "implementation_notes": ["Implement with a docker-service template."],
  "operational_notes": ["Keep archived until needed again."],
  "risks": ["Stale if not reviewed."],
  "status": "candidate",
  "references": [{"kind": "note", "value": "fixture"}],
  "last_reviewed": "2026-04-11"
}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioResourceManifest(t *testing.T, root, scenarioName, resourceName string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", scenarioName, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	payload := map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				resourceName: map[string]any{
					"enabled":  true,
					"required": true,
				},
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal scenario manifest: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeBlueprintArchivedMetadata(t *testing.T, root string, item BlueprintArchivedResource) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	payload, err := json.MarshalIndent(map[string]any{
		"resources": []BlueprintArchivedResource{item},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "blueprint-archived-resources.json"), payload, 0o644); err != nil {
		t.Fatalf("write blueprint archived metadata: %v", err)
	}
}
