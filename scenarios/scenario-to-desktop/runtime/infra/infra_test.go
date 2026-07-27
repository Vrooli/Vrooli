package infra

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealEnvironmentAndFilesystemHonorOSContracts(t *testing.T) {
	t.Setenv("SCENARIO_TO_DESKTOP_INFRA_TEST", "present")
	if got := (RealEnvReader{}).Getenv("SCENARIO_TO_DESKTOP_INFRA_TEST"); got != "present" {
		t.Fatalf("Getenv() = %q, want present", got)
	}

	fs := RealFileSystem{}
	path := filepath.Join(t.TempDir(), "nested", "artifact.txt")
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := fs.WriteFile(path, []byte("desktop artifact"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if got, err := fs.ReadFile(path); err != nil || string(got) != "desktop artifact" {
		t.Fatalf("ReadFile() = %q, %v", got, err)
	}
	if info, err := fs.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Stat() = %v, %v", info, err)
	}
	if err := fs.Remove(path); err != nil || !os.IsNotExist(mustStat(fs, path)) {
		t.Fatalf("Remove() error = %v", err)
	}
}

func mustStat(fs RealFileSystem, path string) error {
	_, err := fs.Stat(path)
	return err
}

func TestRealClockAndCommandRunnerExecuteBoundedOperations(t *testing.T) {
	clock := RealClock{}
	before := clock.Now()
	select {
	case after := <-clock.After(time.Millisecond):
		if after.Before(before) {
			t.Fatalf("After() = %v before Now() = %v", after, before)
		}
	case <-time.After(time.Second):
		t.Fatal("After() did not fire")
	}
	ticker := clock.NewTicker(time.Millisecond)
	defer ticker.Stop()
	select {
	case <-ticker.C():
	case <-time.After(time.Second):
		t.Fatal("NewTicker() did not tick")
	}

	runner := RealCommandRunner{}
	if err := runner.Run(context.Background(), "true", nil); err != nil {
		t.Fatalf("Run(true) error = %v", err)
	}
	if output, err := runner.Output(context.Background(), "printf", "%s", "ready"); err != nil || string(output) != "ready" {
		t.Fatalf("Output() = %q, %v", output, err)
	}
}
