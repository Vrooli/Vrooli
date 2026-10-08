package processfixture

import (
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

type ScenarioRecordOption func(*process.Record)

func ScenarioRecord(name, step string, pid int, opts ...ScenarioRecordOption) process.Record {
	record := process.Record{
		PID:        pid,
		PGID:       pid,
		ProcessID:  fmt.Sprintf("vrooli.develop.%s.%s", name, step),
		Phase:      "develop",
		Scenario:   name,
		Step:       step,
		Command:    "sleep 10",
		WorkingDir: fmt.Sprintf("/repo/scenarios/%s", name),
		LogFile:    fmt.Sprintf("/tmp/%s.log", name),
		StartedAt:  time.Now().UTC(),
		Status:     "running",
	}
	for _, opt := range opts {
		opt(&record)
	}
	return record
}

func WithProcessStartedAt(startedAt time.Time) ScenarioRecordOption {
	return func(record *process.Record) {
		record.StartedAt = startedAt.UTC()
	}
}

func WithProcessPort(port int) ScenarioRecordOption {
	return func(record *process.Record) {
		record.Port = port
	}
}

func WithProcessWorkingDir(path string) ScenarioRecordOption {
	return func(record *process.Record) {
		record.WorkingDir = path
	}
}

func WriteScenarioProcessRecord(t *testing.T, home, name, step string, record process.Record) {
	t.Helper()
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}
	if err := process.WriteScenarioRecord(home, name, step, record); err != nil {
		t.Fatalf("write scenario process record: %v", err)
	}
}

func WriteScenarioProcessRecordWithWorkingDir(t *testing.T, home, name, step string, pid, port int, startedAt time.Time, workingDir string) {
	t.Helper()
	WriteScenarioProcessRecord(t, home, name, step, ScenarioRecord(
		name,
		step,
		pid,
		WithProcessWorkingDir(workingDir),
		WithProcessPort(port),
		WithProcessStartedAt(startedAt),
	))
}
