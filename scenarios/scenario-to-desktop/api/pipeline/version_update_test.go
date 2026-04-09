package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readServiceVersion reads service.json and returns the version from the "service" object.
func readServiceVersion(t *testing.T, servicePath string) string {
	t.Helper()
	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}
	var service map[string]interface{}
	if err := json.Unmarshal(data, &service); err != nil {
		t.Fatalf("parse service.json: %v", err)
	}
	serviceObj, _ := service["service"].(map[string]interface{})
	got, _ := serviceObj["version"].(string)
	return got
}

// readPackageVersion reads package.json and returns the top-level version field.
func readPackageVersion(t *testing.T, packagePath string) string {
	t.Helper()
	data, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("parse package.json: %v", err)
	}
	got, _ := pkg["version"].(string)
	return got
}

// setupVersionTestScenario creates a temp scenario directory with service.json and package.json at the given version.
func setupVersionTestScenario(t *testing.T, root, scenario, version string) (servicePath, packagePath string) {
	t.Helper()
	scenarioRoot := filepath.Join(root, "scenarios", scenario)
	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}

	servicePath = filepath.Join(scenarioRoot, ".vrooli", "service.json")
	packagePath = filepath.Join(scenarioRoot, "ui", "package.json")

	serviceContent := `{
  "service": {
    "name": "` + scenario + `",
    "version": "` + version + `"
  },
  "deployment": {
    "dependencies": {
      "resources": {}
    }
  }
}
`
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}

	packageContent := `{
  "name": "` + scenario + `-ui",
  "version": "` + version + `"
}
`
	if err := os.WriteFile(packagePath, []byte(packageContent), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return servicePath, packagePath
}

func TestVersionUpdater_SetPersistBoth(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	servicePath, packagePath := setupVersionTestScenario(t, root, scenario, "1.0.0")

	updater := newVersionUpdater(root)
	version, rollback, derr := updater.Apply(scenario, &VersionUpdateRequest{
		Mode:    VersionUpdateModeSet,
		Version: "1.2.0",
		Persist: true,
		Source:  VersionSourceBoth,
	})
	if derr != nil {
		t.Fatalf("Apply returned error: %v", derr)
	}
	if version != "1.2.0" {
		t.Fatalf("expected version 1.2.0, got %q", version)
	}

	t.Run("service.json updated", func(t *testing.T) {
		if got := readServiceVersion(t, servicePath); got != "1.2.0" {
			t.Fatalf("expected service.json version 1.2.0, got %q", got)
		}
	})

	t.Run("package.json updated", func(t *testing.T) {
		if got := readPackageVersion(t, packagePath); got != "1.2.0" {
			t.Fatalf("expected package.json version 1.2.0, got %q", got)
		}
	})

	t.Run("rollback restores versions", func(t *testing.T) {
		if rollback == nil {
			t.Fatal("expected rollback to be returned for persisted update")
		}
		if err := rollback.Restore(); err != nil {
			t.Fatalf("rollback failed: %v", err)
		}
		if got := readServiceVersion(t, servicePath); got != "1.0.0" {
			t.Fatalf("expected service.json version 1.0.0 after rollback, got %q", got)
		}
		if got := readPackageVersion(t, packagePath); got != "1.0.0" {
			t.Fatalf("expected package.json version 1.0.0 after rollback, got %q", got)
		}
	})
}

func TestVersionUpdater_BumpMinor(t *testing.T) {
	root := t.TempDir()
	setupVersionTestScenario(t, root, "demo", "1.4.3")

	updater := newVersionUpdater(root)
	version, _, derr := updater.Apply("demo", &VersionUpdateRequest{
		Mode:    VersionUpdateModeBump,
		Bump:    VersionBumpMinor,
		Persist: true,
		Source:  VersionSourceBoth,
	})
	if derr != nil {
		t.Fatalf("Apply returned error: %v", derr)
	}
	if version != "1.5.0" {
		t.Fatalf("expected version 1.5.0, got %q", version)
	}
}

func TestVersionUpdater_DowngradeBlocked(t *testing.T) {
	root := t.TempDir()
	setupVersionTestScenario(t, root, "demo", "2.0.0")

	updater := newVersionUpdater(root)
	_, _, derr := updater.Apply("demo", &VersionUpdateRequest{
		Mode:    VersionUpdateModeSet,
		Version: "1.0.0",
		Persist: false,
		Source:  VersionSourceBoth,
	})
	if derr == nil {
		t.Fatal("expected downgrade error, got nil")
	}
}

func TestVersionUpdater_BumpAutoAlias(t *testing.T) {
	root := t.TempDir()
	setupVersionTestScenario(t, root, "demo", "1.0.0")

	updater := newVersionUpdater(root)
	version, _, derr := updater.Apply("demo", &VersionUpdateRequest{
		Mode:    VersionUpdateModeBump,
		Bump:    VersionBumpAuto,
		Persist: false,
		Source:  VersionSourceBoth,
	})
	if derr != nil {
		t.Fatalf("Apply returned error: %v", derr)
	}
	if version != "1.0.1" {
		t.Fatalf("expected version 1.0.1, got %q", version)
	}
}
