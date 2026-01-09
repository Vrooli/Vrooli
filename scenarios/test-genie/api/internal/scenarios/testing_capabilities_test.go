package scenarios

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectTestingCapabilitiesPrefersPhased(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("TEST_GENIE_BIN", "")
	t.Setenv("TEST_GENIE_DISABLE", "")
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("mkdir coverage dir: %v", err)
	}
	scriptPath := filepath.Join(coverageDir, "run-tests.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	caps := DetectTestingCapabilities(dir)
	if !caps.Phased || !caps.HasTests || caps.Preferred != "phased" {
		t.Fatalf("expected phased tests, got %#v", caps)
	}
	if len(caps.Commands) == 0 || caps.Commands[0].Type != "phased" || caps.Commands[0].WorkingDir != dir {
		t.Fatalf("expected phased command with scenario dir, got %#v", caps.Commands)
	}
}

func TestDetectTestingCapabilitiesLifecycle(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("TEST_GENIE_BIN", "")
	t.Setenv("TEST_GENIE_DISABLE", "")
	dir := t.TempDir()
	manifest := `{"lifecycle":{"test":"./scripts/run.sh"}}`
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}

	caps := DetectTestingCapabilities(dir)
	if caps.Phased || !caps.Lifecycle || caps.Preferred != "lifecycle" {
		t.Fatalf("expected lifecycle detection, got %#v", caps)
	}
	if len(caps.Commands) == 0 {
		t.Fatalf("expected lifecycle command, got %#v", caps.Commands)
	}
}

func TestDetectTestingCapabilitiesLegacy(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("TEST_GENIE_BIN", "")
	t.Setenv("TEST_GENIE_DISABLE", "")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scenario-test.yaml"), []byte("legacy: true"), 0o644); err != nil {
		t.Fatalf("write scenario-test.yaml: %v", err)
	}

	caps := DetectTestingCapabilities(dir)
	if !caps.Legacy || caps.Preferred != "legacy" {
		t.Fatalf("expected legacy detection, got %#v", caps)
	}
	found := false
	for _, cmd := range caps.Commands {
		if cmd.Type == "legacy" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected legacy command, got %#v", caps.Commands)
	}
}

func TestDetectTestingCapabilitiesMultipleModes(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("TEST_GENIE_BIN", "")
	t.Setenv("TEST_GENIE_DISABLE", "")
	dir := t.TempDir()
	// Lifecycle + legacy but no phased should prefer lifecycle
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(`{"lifecycle":{"test":"./run"}}`), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario-test.yaml"), []byte("legacy: true"), 0o644); err != nil {
		t.Fatalf("write scenario-test.yaml: %v", err)
	}

	caps := DetectTestingCapabilities(dir)
	if !caps.Lifecycle || !caps.Legacy || caps.Preferred != "lifecycle" {
		t.Fatalf("expected lifecycle preference, got %#v", caps)
	}

	// Add phased runner and ensure preference flips
	if err := os.MkdirAll(filepath.Join(dir, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir coverage dir: %v", err)
	}
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	if err := os.WriteFile(filepath.Join(dir, "coverage", "run-tests.sh"), []byte("#!/usr/bin/env bash"), mode); err != nil {
		t.Fatalf("write run-tests: %v", err)
	}

	caps = DetectTestingCapabilities(dir)
	if !caps.Phased || caps.Preferred != "phased" {
		t.Fatalf("expected phased preference, got %#v", caps)
	}
	foundPhased := false
	for _, cmd := range caps.Commands {
		if cmd.Type == "phased" {
			foundPhased = true
		}
	}
	if !foundPhased {
		t.Fatalf("expected phased command after upgrade, got %#v", caps.Commands)
	}
}

func TestDetectTestingCapabilitiesPrefersGenie(t *testing.T) {
	dir := t.TempDir()
	// Create fake test-genie binary
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	binPath := filepath.Join(binDir, "test-genie")
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	if err := os.WriteFile(binPath, []byte("#!/usr/bin/env bash\nexit 0\n"), mode); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("TEST_GENIE_BIN", "")
	t.Setenv("TEST_GENIE_DISABLE", "")

	caps := DetectTestingCapabilities(dir)
	if !caps.Genie || caps.Preferred != "genie" {
		t.Fatalf("expected genie preference, got %#v", caps)
	}
	if len(caps.Commands) == 0 || caps.Commands[0].Type != "genie" {
		t.Fatalf("expected genie command first, got %#v", caps.Commands)
	}
}
