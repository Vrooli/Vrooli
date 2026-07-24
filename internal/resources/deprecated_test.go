package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
)

func TestDeprecateResourceArchivesAndRemovesActiveState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeResourceConfig(t, root, "fixture", true)
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nexit 0\n")

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
	testresource.WriteResourceCLI(t, root, "fixture", "#!/usr/bin/env bash\nexit 0\n")

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
	writeDeprecatedMetadata(t, root, DeprecatedResource{
		Name:                "fixture",
		DeprecatedAt:        "2026-01-01",
		Reason:              "test",
		ArchivePath:         archiveDir,
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-01-02",
	})

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
	writeResourceManifest(t, root, "active", testresource.ResourceManifest(
		"active",
		testresource.WithResourceDisplayName("Active"),
		testresource.WithResourceDescription("Active manifest-backed resource"),
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "partial",
		}),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "active:1.0.0",
		}),
	))
	testresource.WriteResourceCLI(t, root, "active", "#!/usr/bin/env bash\nexit 0\n")
	testresource.WriteResourceCLI(t, root, "deprecated-fixture", "#!/usr/bin/env bash\nexit 0\n")
	writeDeprecatedMetadata(t, root, DeprecatedResource{
		Name:                "deprecated-fixture",
		DeprecatedAt:        "2026-04-11",
		Reason:              "test",
		ArchivePath:         filepath.Join(home, ".vrooli", "archive", "resources", "deprecated-fixture"),
		ArchiveHash:         "abc",
		RetentionPolicyDays: 90,
		RestoreSupported:    true,
		PurgeAfter:          "2026-07-10",
	})

	items, err := NewController(root, home).Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(items) != 1 || items[0].Name != "active" {
		t.Fatalf("discover items = %#v", items)
	}
}
