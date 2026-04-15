package storagemigrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"

	"scenario-to-desktop-api/storagepaths"
)

func TestRunMovesLegacyData(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()
	destRoot := t.TempDir()

	locator, err := storagepaths.NewLocatorWith(storage.ResolverConfig{
		AppID:   storagepaths.AppID,
		Profile: storage.ProfileDesktop,
		UserHomeDir: func() (string, error) {
			return homeDir, nil
		},
		UserConfigDir: func() (string, error) {
			return filepath.Join(destRoot, "config-home"), nil
		},
		UserCacheDir: func() (string, error) {
			return filepath.Join(destRoot, "cache-home"), nil
		},
	}, storage.Options{ScenarioID: storagepaths.ScenarioID})
	if err != nil {
		t.Fatalf("NewLocatorWith() error: %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, ".vrooli", "deploy-targets.json"), `{"prod":{"label":"Prod"}}`)
	writeFile(t, filepath.Join(repoRoot, ".vrooli", "deployment", "telemetry", "demo.jsonl"), `{"event":"app_start"}`+"\n")
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "desktop_records_v2.json"), `[]`)
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "smoke_tests_v2.json"), `[]`)
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "smoke_tests.json"), `[]`)
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "pipelines", "pipe.json"), `{}`)
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "indexes", "demo.json"), `{}`)
	writeFile(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "livedesktop", "sessions", "sess-1", "shot.png"), `png`)
	writeFile(t, filepath.Join(homeDir, ".vrooli", "scenario-to-desktop", "state", "demo.json"), `{}`)

	result, err := Run(Options{
		RepoRoot: repoRoot,
		HomeDir:  homeDir,
		Locator:  locator,
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(result.Moved) == 0 {
		t.Fatalf("expected moved entries")
	}

	deployTargetsPath, _ := locator.DeployTargetsPath()
	telemetryDir, _ := locator.TelemetryDir()
	recordsPath, _ := locator.RecordsPath()
	smokeTestsPath, _ := locator.SmokeTestsPath()
	pipelineDir, _ := locator.PipelineStateDir()
	indexDir, _ := locator.PipelineIndexDir()
	stateDir, _ := locator.ScenarioStateDir()
	liveDesktopDir, _ := locator.LiveDesktopDir()
	dataRoot, _ := locator.DataRoot()

	assertExists(t, deployTargetsPath)
	assertExists(t, filepath.Join(telemetryDir, "demo.jsonl"))
	assertExists(t, recordsPath)
	assertExists(t, smokeTestsPath)
	assertExists(t, filepath.Join(dataRoot, "smoke_tests.json"))
	assertExists(t, filepath.Join(pipelineDir, "pipe.json"))
	assertExists(t, filepath.Join(indexDir, "demo.json"))
	assertExists(t, filepath.Join(stateDir, "demo.json"))
	assertExists(t, filepath.Join(liveDesktopDir, "sessions", "sess-1", "shot.png"))

	assertNotExists(t, filepath.Join(repoRoot, ".vrooli", "deploy-targets.json"))
	assertNotExists(t, filepath.Join(repoRoot, ".vrooli", "deployment", "telemetry", "demo.jsonl"))
	assertNotExists(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "desktop_records_v2.json"))
	assertNotExists(t, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "pipelines", "pipe.json"))
	assertNotExists(t, filepath.Join(homeDir, ".vrooli", "scenario-to-desktop", "state", "demo.json"))
}

func TestRunFailsOnDestinationConflict(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()
	destRoot := t.TempDir()

	locator, err := storagepaths.NewLocatorWith(storage.ResolverConfig{
		AppID:   storagepaths.AppID,
		Profile: storage.ProfileDesktop,
		UserHomeDir: func() (string, error) {
			return homeDir, nil
		},
		UserConfigDir: func() (string, error) {
			return filepath.Join(destRoot, "config-home"), nil
		},
		UserCacheDir: func() (string, error) {
			return filepath.Join(destRoot, "cache-home"), nil
		},
	}, storage.Options{ScenarioID: storagepaths.ScenarioID})
	if err != nil {
		t.Fatalf("NewLocatorWith() error: %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, ".vrooli", "deploy-targets.json"), `{"prod":{"label":"Prod"}}`)
	deployTargetsPath, _ := locator.DeployTargetsPath()
	writeFile(t, deployTargetsPath, `{"existing":true}`)

	if _, err := Run(Options{
		RepoRoot: repoRoot,
		HomeDir:  homeDir,
		Locator:  locator,
	}); err == nil {
		t.Fatalf("expected destination conflict error")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %q to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be removed, got err=%v", path, err)
	}
}
