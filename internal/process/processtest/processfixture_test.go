package processfixture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

func TestWriteScenarioProcessRecordCreatesProcessMetadata(t *testing.T) {
	home := t.TempDir()
	WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: 1234})

	for _, rel := range []string{
		".vrooli/processes/scenarios/alpha/start-api.json",
		".vrooli/processes/scenarios/alpha/start-api.pid",
	} {
		if _, err := os.Stat(filepath.Join(home, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestScenarioRecordProvidesTypedDefaults(t *testing.T) {
	startedAt := time.Now().Add(-time.Minute)
	record := ScenarioRecord(
		"alpha",
		"start-api",
		1234,
		WithProcessPort(18080),
		WithProcessStartedAt(startedAt),
	)

	if record.ProcessID != "vrooli.develop.alpha.start-api" {
		t.Fatalf("ProcessID = %q", record.ProcessID)
	}
	if record.WorkingDir != "/repo/scenarios/alpha" {
		t.Fatalf("WorkingDir = %q", record.WorkingDir)
	}
	if record.Port != 18080 {
		t.Fatalf("Port = %d", record.Port)
	}
	if record.StartedAt.IsZero() || !record.StartedAt.Equal(startedAt.UTC()) {
		t.Fatalf("StartedAt = %v", record.StartedAt)
	}
}
