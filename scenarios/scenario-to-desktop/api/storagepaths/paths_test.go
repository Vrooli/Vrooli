package storagepaths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
)

func TestLocatorResolvesAndCreatesCanonicalPaths(t *testing.T) {
	root := t.TempDir()
	locator, err := NewLocatorWith(storage.ResolverConfig{
		AppID:         AppID,
		Profile:       storage.ProfileDesktop,
		UserHomeDir:   func() (string, error) { return root, nil },
		UserConfigDir: func() (string, error) { return filepath.Join(root, "config"), nil },
		UserCacheDir:  func() (string, error) { return filepath.Join(root, "cache"), nil },
	}, storage.Options{ScenarioID: ScenarioID})
	if err != nil {
		t.Fatalf("NewLocatorWith: %v", err)
	}
	telemetry, err := locator.TelemetryFilePath("demo")
	if err != nil {
		t.Fatalf("TelemetryFilePath: %v", err)
	}
	if filepath.Base(telemetry) != "demo.jsonl" {
		t.Fatalf("telemetry path = %q", telemetry)
	}
	if _, err := locator.TelemetryFilePath(""); err == nil {
		t.Fatal("expected empty scenario to fail")
	}
	for _, ensure := range []func() (string, error){locator.EnsureTelemetryDir, locator.EnsureScenarioStateDir, locator.EnsurePipelineStateDir, locator.EnsurePipelineIndexDir, locator.EnsureInvestigationsDir, locator.EnsureCapturesDir, locator.EnsureLiveDesktopDir, locator.EnsureStagingRoot} {
		path, err := ensure()
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("expected directory %q: %v", path, err)
		}
	}
}
