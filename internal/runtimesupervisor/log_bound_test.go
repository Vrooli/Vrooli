package runtimesupervisor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
)

// Detached scenario processes write straight into their step logs, so the
// supervisor is what keeps them bounded. It sweeps at most once per interval.
func TestSupervisorBoundsScenarioLogsAtMostOncePerInterval(t *testing.T) {
	home := t.TempDir()
	root, err := process.LogsRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "scenarios", "ui-health")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "vrooli.develop.ui-health.start-api.log")
	if err := os.WriteFile(log, bytes.Repeat([]byte("probe\n"), int(process.DefaultLogFileLimit/6)+4096), 0o600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	svc := New(Config{HomeDir: home, Stderr: &stderr})

	first := svc.boundLogs()
	if !first.Ran || first.Trimmed != 1 || first.ReclaimedBytes <= 0 {
		t.Fatalf("first sweep = %+v, stderr = %s", first, stderr.String())
	}
	if info, err := os.Stat(log); err != nil || info.Size() > 1024 {
		t.Fatalf("log after trim: size=%v err=%v", info.Size(), err)
	}
	if _, err := os.Stat(log + ".1"); err != nil {
		t.Fatalf("tail file missing: %v", err)
	}
	if second := svc.boundLogs(); second.Ran {
		t.Fatalf("second sweep inside the interval ran: %+v", second)
	}
}
