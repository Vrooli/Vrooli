package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestArchiveResourceToBlueprintArchivesAndRemovesActiveState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")

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

func TestArchiveResourceToBlueprintSkipsGeneratedVirtualenvContent(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")
	venvDir := filepath.Join(root, "resources", "fixture", ".venv")
	if err := os.MkdirAll(filepath.Join(venvDir, "lib"), 0o755); err != nil {
		t.Fatalf("mkdir .venv/lib: %v", err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "lib", "module.py"), []byte("print('fixture')\n"), 0o644); err != nil {
		t.Fatalf("write .venv file: %v", err)
	}
	if err := os.Symlink("lib", filepath.Join(venvDir, "lib64")); err != nil {
		t.Fatalf("symlink .venv/lib64: %v", err)
	}

	report, err := NewController(root, home).ArchiveResourceToBlueprint("fixture")
	if err != nil {
		t.Fatalf("ArchiveResourceToBlueprint: %v", err)
	}
	if _, err := os.Stat(filepath.Join(report.ArchiveDir, "files", "resource", "fixture", ".venv")); !os.IsNotExist(err) {
		t.Fatalf("expected .venv to be skipped from archive, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(report.ArchiveDir, "archive-skipped-paths.json")); err != nil {
		t.Fatalf("expected skipped paths metadata: %v", err)
	}
}

func TestArchiveResourceToBlueprintHandlesUnreadableRuntimeDirectory(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeBlueprintArchiveFixture(t, root, "fixture")
	writeResourceCLI(t, root, "fixture")
	protectedDir := filepath.Join(root, "resources", "fixture", "data", "protected")
	if err := os.MkdirAll(protectedDir, 0o755); err != nil {
		t.Fatalf("mkdir protected dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protectedDir, "state.db"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatalf("write protected file: %v", err)
	}
	if err := os.Chmod(protectedDir, 0o000); err != nil {
		t.Fatalf("chmod protected dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(protectedDir, 0o755)
	})

	report, err := NewController(root, home).ArchiveResourceToBlueprint("fixture")
	if err != nil {
		t.Fatalf("ArchiveResourceToBlueprint: %v", err)
	}
	t.Cleanup(func() {
		remnantsRoot := filepath.Join(home, ".vrooli", "archive", "resources", "remnants")
		matches, _ := filepath.Glob(filepath.Join(remnantsRoot, "*-fixture"))
		for _, match := range matches {
			_ = os.Chmod(filepath.Join(match, "data", "protected"), 0o755)
			_ = os.Chmod(filepath.Join(match, "data"), 0o755)
			_ = os.Chmod(match, 0o755)
		}
	})
	if _, err := os.Stat(filepath.Join(root, "resources", "fixture")); !os.IsNotExist(err) {
		t.Fatalf("expected resource directory to be removed or moved away, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(report.ArchiveDir, "archive-skipped-paths.json")); err != nil {
		t.Fatalf("expected skipped paths metadata: %v", err)
	}
}

func writeBlueprintArchiveFixture(t *testing.T, root, name string) {
	t.Helper()
	testkitgo.WriteRawJSON(t, filepath.Join(root, filepath.FromSlash(blueprintDirPath), name+".json"), `{
  "name": "`+name+`",
  "display_name": "Fixture",
  "category": "testing",
  "summary": "Fixture resource",
  "why_it_matters": "Used in tests.",
  "when_to_use": ["Testing archive lifecycle"],
  "integration_kind": "docker-service",
  "platform_support": {
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
}`, 0o644)
}
