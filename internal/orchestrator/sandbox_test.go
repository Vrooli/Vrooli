package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSandboxAffectedScenariosReturnsSortedMatches(t *testing.T) {
	home := t.TempDir()
	startedAt := time.Now().Add(-1 * time.Minute)
	writeSandboxRecord(t, home, "beta", "/merged/scenarios/beta", startedAt)
	writeSandboxRecord(t, home, "alpha", "/merged/scenarios/alpha", startedAt)
	writeSandboxRecord(t, home, "gamma", "/repo/scenarios/gamma", startedAt)

	affected, err := SandboxAffectedScenarios(home, "/merged")
	if err != nil {
		t.Fatalf("SandboxAffectedScenarios: %v", err)
	}
	if got := strings.Join(affected, ","); got != "alpha,beta" {
		t.Fatalf("affected = %q", got)
	}
}

func writeSandboxRecord(t *testing.T, home, name, workingDir string, startedAt time.Time) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", name, "start-api.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": 1234,
  "pgid": 1234,
  "process_id": "vrooli.develop.` + name + `.start-api",
  "phase": "develop",
  "scenario": "` + name + `",
  "step": "start-api",
  "command": "sleep 10",
  "working_dir": "` + workingDir + `",
  "log_file": "/tmp/` + name + `.log",
  "port": 18080,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%s\n", data)), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
