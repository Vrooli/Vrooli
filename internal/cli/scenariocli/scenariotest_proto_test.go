package scenariocli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"
)

func TestWriteTestPhaseResultJSONShape_Passed(t *testing.T) {
	result := lifecycle.PhaseResult{
		Scenario:  "demo",
		Phase:     "test",
		ExitCode:  0,
		StartedAt: time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 6, 14, 0, 4, 0, 0, time.UTC),
		LogFile:   "/home/u/.vrooli/logs/demo.log",
	}
	var buf bytes.Buffer
	if err := WriteTestPhaseResultJSON(&buf, result, nil); err != nil {
		t.Fatalf("write: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, buf.String())
	}
	// snake_case + EmitUnpopulated: every field present, including empty ones.
	for _, key := range []string{"scenario", "status", "exit_code", "started_at", "ended_at", "duration", "log_file"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing field %q in %s", key, buf.String())
		}
	}
	if decoded["status"] != "passed" {
		t.Fatalf("expected status passed, got %v", decoded["status"])
	}
	if decoded["duration"] != "4m0s" {
		t.Fatalf("expected duration 4m0s, got %v", decoded["duration"])
	}
}

func TestWriteTestPhaseResultJSONShape_Failed(t *testing.T) {
	result := lifecycle.PhaseResult{
		Scenario: "demo",
		Phase:    "test",
		ExitCode: 0, // a failure with no surfaced exit code should normalize to 1
	}
	var buf bytes.Buffer
	if err := WriteTestPhaseResultJSON(&buf, result, errors.New("a step failed")); err != nil {
		t.Fatalf("write: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded["status"] != "failed" {
		t.Fatalf("expected status failed, got %v", decoded["status"])
	}
	if decoded["exit_code"].(float64) != 1 {
		t.Fatalf("expected normalized exit_code 1, got %v", decoded["exit_code"])
	}
}
