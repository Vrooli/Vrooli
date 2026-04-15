package orchestrator

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	testprocess "github.com/vrooli/vrooli/packages/testkit-go/processfixture"
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
	testprocess.WriteScenarioProcessRecord(t, home, name, "start-api", process.Record{
		PID:        1234,
		PGID:       1234,
		ProcessID:  "vrooli.develop." + name + ".start-api",
		Phase:      "develop",
		Scenario:   name,
		Step:       "start-api",
		Command:    "sleep 10",
		WorkingDir: workingDir,
		LogFile:    "/tmp/" + name + ".log",
		Port:       18080,
		StartedAt:  startedAt,
		Status:     "running",
	})
}
