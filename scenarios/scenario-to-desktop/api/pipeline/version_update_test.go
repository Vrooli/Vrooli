package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionUpdater_SetPersistBoth(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	scenarioRoot := filepath.Join(root, "scenarios", scenario)

	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}

	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	packagePath := filepath.Join(scenarioRoot, "ui", "package.json")

	if err := os.WriteFile(servicePath, []byte(`{
  "service": {
    "name": "demo",
    "version": "1.0.0"
  },
  "deployment": {
    "dependencies": {
      "resources": {}
    }
  }
}`+"\n"), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}

	if err := os.WriteFile(packagePath, []byte(`{
  "name": "demo-ui",
  "version": "1.0.0"
}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

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

	serviceData, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}
	var service map[string]interface{}
	if err := json.Unmarshal(serviceData, &service); err != nil {
		t.Fatalf("parse service.json: %v", err)
	}
	serviceObj, _ := service["service"].(map[string]interface{})
	if got, _ := serviceObj["version"].(string); got != "1.2.0" {
		t.Fatalf("expected service.json version 1.2.0, got %q", got)
	}

	packageData, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal(packageData, &pkg); err != nil {
		t.Fatalf("parse package.json: %v", err)
	}
	if got, _ := pkg["version"].(string); got != "1.2.0" {
		t.Fatalf("expected package.json version 1.2.0, got %q", got)
	}

	if rollback == nil {
		t.Fatal("expected rollback to be returned for persisted update")
	}
	if err := rollback.Restore(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	serviceData, err = os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read service.json after rollback: %v", err)
	}
	if err := json.Unmarshal(serviceData, &service); err != nil {
		t.Fatalf("parse service.json after rollback: %v", err)
	}
	serviceObj, _ = service["service"].(map[string]interface{})
	if got, _ := serviceObj["version"].(string); got != "1.0.0" {
		t.Fatalf("expected service.json version 1.0.0 after rollback, got %q", got)
	}

	packageData, err = os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("read package.json after rollback: %v", err)
	}
	if err := json.Unmarshal(packageData, &pkg); err != nil {
		t.Fatalf("parse package.json after rollback: %v", err)
	}
	if got, _ := pkg["version"].(string); got != "1.0.0" {
		t.Fatalf("expected package.json version 1.0.0 after rollback, got %q", got)
	}
}

func TestVersionUpdater_BumpMinor(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	scenarioRoot := filepath.Join(root, "scenarios", scenario)

	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}

	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	packagePath := filepath.Join(scenarioRoot, "ui", "package.json")

	if err := os.WriteFile(servicePath, []byte(`{"service":{"name":"demo","version":"1.4.3"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"name":"demo-ui","version":"1.4.3"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	updater := newVersionUpdater(root)
	version, _, derr := updater.Apply(scenario, &VersionUpdateRequest{
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
	scenario := "demo"
	scenarioRoot := filepath.Join(root, "scenarios", scenario)

	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}

	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	packagePath := filepath.Join(scenarioRoot, "ui", "package.json")

	if err := os.WriteFile(servicePath, []byte(`{"service":{"name":"demo","version":"2.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"name":"demo-ui","version":"2.0.0"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	updater := newVersionUpdater(root)
	_, _, derr := updater.Apply(scenario, &VersionUpdateRequest{
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
	scenario := "demo"
	scenarioRoot := filepath.Join(root, "scenarios", scenario)

	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}

	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	packagePath := filepath.Join(scenarioRoot, "ui", "package.json")

	if err := os.WriteFile(servicePath, []byte(`{"service":{"name":"demo","version":"1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(packagePath, []byte(`{"name":"demo-ui","version":"1.0.0"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	updater := newVersionUpdater(root)
	version, _, derr := updater.Apply(scenario, &VersionUpdateRequest{
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
