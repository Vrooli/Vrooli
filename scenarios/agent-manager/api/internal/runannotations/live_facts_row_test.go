package runannotations

import (
	"encoding/json"
	"testing"
)

func TestFactsRowCarriesTheToolVariables(t *testing.T) {
	features := RunFeatures{
		RunID:         "run-1",
		LastToolError: "permission denied",
		Tools:         map[string]any{"failed_commands": int64(3), "tool_source": "events"},
	}
	line, err := marshalFactsRow(features)
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(line, &row); err != nil {
		t.Fatal(err)
	}
	if row["run_id"] != "run-1" || row["last_tool_error"] != "permission denied" {
		t.Fatalf("feature fields missing: %v", row)
	}
	if row["failed_commands"] != float64(3) || row["tool_source"] != "events" {
		t.Fatalf("tool variables missing: %v", row)
	}
}
