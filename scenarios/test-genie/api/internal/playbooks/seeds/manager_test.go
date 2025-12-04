package seeds

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerHasSeeds(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	manager := NewManager(tempDir, tempDir, testDir, nil)

	// Without apply script
	if manager.HasSeeds() {
		t.Error("expected HasSeeds=false without apply script")
	}

	// With apply script
	applyPath := filepath.Join(seedsDir, ApplyScript)
	if err := os.WriteFile(applyPath, []byte("#!/bin/bash\necho apply"), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	if !manager.HasSeeds() {
		t.Error("expected HasSeeds=true with apply script")
	}
}

func TestManagerApplyNoScript(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	manager := NewManager(tempDir, tempDir, testDir, nil)
	cleanup, err := manager.Apply(context.Background())
	if err != nil {
		t.Fatalf("expected no error without apply script, got: %v", err)
	}
	if cleanup != nil {
		t.Error("expected nil cleanup without apply script")
	}
}

func TestManagerApplySuccess(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	// Create apply script that writes a marker file
	markerPath := filepath.Join(tempDir, "applied.txt")
	applyScript := `#!/bin/bash
echo "applied" > "` + markerPath + `"
`
	if err := os.WriteFile(filepath.Join(seedsDir, ApplyScript), []byte(applyScript), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	var output bytes.Buffer
	manager := NewManager(tempDir, tempDir, testDir, &output)

	cleanup, err := manager.Apply(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify apply ran
	if _, err := os.Stat(markerPath); err != nil {
		t.Error("apply script did not create marker file")
	}

	// Cleanup should be a no-op since no cleanup script exists
	if cleanup == nil {
		t.Error("expected non-nil cleanup function")
	}
	cleanup()
}

func TestManagerApplyWithCleanup(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	// Create apply script
	applyMarker := filepath.Join(tempDir, "applied.txt")
	applyScript := `#!/bin/bash
echo "applied" > "` + applyMarker + `"
`
	if err := os.WriteFile(filepath.Join(seedsDir, ApplyScript), []byte(applyScript), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	// Create cleanup script
	cleanupMarker := filepath.Join(tempDir, "cleaned.txt")
	cleanupScript := `#!/bin/bash
echo "cleaned" > "` + cleanupMarker + `"
`
	if err := os.WriteFile(filepath.Join(seedsDir, CleanupScript), []byte(cleanupScript), 0o755); err != nil {
		t.Fatalf("failed to create cleanup script: %v", err)
	}

	manager := NewManager(tempDir, tempDir, testDir, nil)

	cleanup, err := manager.Apply(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Run cleanup
	cleanup()

	// Verify cleanup ran
	if _, err := os.Stat(cleanupMarker); err != nil {
		t.Error("cleanup script did not create marker file")
	}
}

func TestManagerApplyFailure(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	// Create failing apply script
	applyScript := `#!/bin/bash
exit 1
`
	if err := os.WriteFile(filepath.Join(seedsDir, ApplyScript), []byte(applyScript), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	manager := NewManager(tempDir, tempDir, testDir, nil)
	_, err := manager.Apply(context.Background())
	if err == nil {
		t.Fatal("expected error for failing script")
	}
	if !strings.Contains(err.Error(), "apply seed script failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestManagerApplyCanceled(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	// Create slow apply script
	applyScript := `#!/bin/bash
sleep 10
`
	if err := os.WriteFile(filepath.Join(seedsDir, ApplyScript), []byte(applyScript), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	manager := NewManager(tempDir, tempDir, testDir, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := manager.Apply(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestManagerEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	seedsDir := filepath.Join(testDir, "playbooks", SeedsFolder)
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		t.Fatalf("failed to create seeds dir: %v", err)
	}

	// Create scenario and app directories (must exist for cmd.Dir)
	scenarioDir := filepath.Join(tempDir, "scenario")
	appRoot := filepath.Join(tempDir, "app")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		t.Fatalf("failed to create app root: %v", err)
	}

	// Create script that uses environment variables
	envMarker := filepath.Join(tempDir, "env.txt")
	applyScript := `#!/bin/bash
echo "SCENARIO_DIR=$TEST_GENIE_SCENARIO_DIR" > "` + envMarker + `"
echo "APP_ROOT=$TEST_GENIE_APP_ROOT" >> "` + envMarker + `"
`
	if err := os.WriteFile(filepath.Join(seedsDir, ApplyScript), []byte(applyScript), 0o755); err != nil {
		t.Fatalf("failed to create apply script: %v", err)
	}

	manager := NewManager(scenarioDir, appRoot, testDir, nil)

	cleanup, err := manager.Apply(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Verify environment variables were passed
	content, err := os.ReadFile(envMarker)
	if err != nil {
		t.Fatalf("failed to read env marker: %v", err)
	}

	envStr := string(content)
	if !strings.Contains(envStr, "SCENARIO_DIR="+scenarioDir) {
		t.Errorf("expected SCENARIO_DIR=%s in output, got: %s", scenarioDir, envStr)
	}
	if !strings.Contains(envStr, "APP_ROOT="+appRoot) {
		t.Errorf("expected APP_ROOT=%s in output, got: %s", appRoot, envStr)
	}
}

func TestManagerPaths(t *testing.T) {
	manager := NewManager("/scenario", "/app", "/test", nil)

	expectedSeedsDir := "/test/playbooks/__seeds"
	if got := manager.SeedsDir(); got != expectedSeedsDir {
		t.Errorf("SeedsDir: expected %s, got %s", expectedSeedsDir, got)
	}

	expectedApply := "/test/playbooks/__seeds/apply.sh"
	if got := manager.ApplyPath(); got != expectedApply {
		t.Errorf("ApplyPath: expected %s, got %s", expectedApply, got)
	}

	expectedCleanup := "/test/playbooks/__seeds/cleanup.sh"
	if got := manager.CleanupPath(); got != expectedCleanup {
		t.Errorf("CleanupPath: expected %s, got %s", expectedCleanup, got)
	}
}
