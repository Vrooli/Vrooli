package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
)

func TestValidateLaunchTraceFileRequiresContractAndRunKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.json")
	trace := smoketest.LaunchTrace{
		SchemaVersion: smoketest.LaunchTraceSchemaVersion,
		RunID:         "run:protocol",
		RunKind:       smoketest.LaunchRunProtocol,
		StartedAt:     time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
		Events: []smoketest.LaunchEvent{
			{Name: smoketest.EventRecorderStarted, Component: "test", Role: "trace", MonotonicNs: 1, WallTime: time.Date(2026, 8, 6, 12, 0, 0, 1, time.UTC)},
			{Name: smoketest.EventProtocolStarted, Component: "test", Role: "main", MonotonicNs: 2, WallTime: time.Date(2026, 8, 6, 12, 0, 0, 2, time.UTC)},
			{Name: smoketest.EventProtocolCompleted, Component: "test", Role: "main", MonotonicNs: 3, WallTime: time.Date(2026, 8, 6, 12, 0, 0, 3, time.UTC)},
		},
	}
	data, err := trace.MarshalValidated()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateLaunchTraceFile(path, smoketest.LaunchRunProtocol); err != nil {
		t.Fatalf("valid trace rejected: %v", err)
	}
	if err := validateLaunchTraceFile(path, smoketest.LaunchRunDemo); err == nil {
		t.Fatal("expected run-kind mismatch to fail")
	}

	trace.Events[2].MonotonicNs = 1
	data, err = json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateLaunchTraceFile(path, smoketest.LaunchRunProtocol); err == nil {
		t.Fatal("expected malformed trace to fail validation")
	}
}
