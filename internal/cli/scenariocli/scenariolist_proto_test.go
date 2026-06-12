package scenariocli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
)

// TestRenderListResponseJSONContract pins the `vrooli scenario list --json` wire
// shape so a change to the vrooli.cli.v1 proto (or the producer mapping) fails
// here rather than silently breaking out-of-process consumers.
func TestRenderListResponseJSONContract(t *testing.T) {
	resp := ListResponse{
		Items: []ListItemOutput{
			{
				Name:        "agent-inbox",
				Description: "inbox",
				Version:     "0.0.1",
				Status:      "running",
				Tags:        []string{"react-ui", "go-api"},
				Path:        "/s/agent-inbox",
				Ports:       []ListPortOutput{{Key: "API_PORT", Step: "develop", Port: 16542, ListenerStatus: "listening"}},
			},
			// Sparse: nil tags. The contract normalizes empty tags to [] (never null).
			{Name: "accessibility-compliance-hub", Status: "available", Path: "/s/ach"},
		},
		RunningCount: 1,
		Failures:     []discovery.Failure{{Kind: "scenario", Name: "broken", Error: "boom"}},
	}

	var buf bytes.Buffer
	if err := RenderListResponse(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderListResponse: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}

	// Summary rollup.
	summary, ok := got["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary missing/wrong type: %v", got["summary"])
	}
	for key, want := range map[string]float64{"total_scenarios": 2, "running": 1, "available": 1} {
		if summary[key] != want {
			t.Errorf("summary[%q] = %v, want %v", key, summary[key], want)
		}
	}

	scenarios, ok := got["scenarios"].([]any)
	if !ok || len(scenarios) != 2 {
		t.Fatalf("scenarios: want 2, got %v", got["scenarios"])
	}

	// snake_case field names (UseProtoNames=true).
	first := scenarios[0].(map[string]any)
	for _, key := range []string{"name", "description", "version", "status", "tags", "path", "ports"} {
		if _, ok := first[key]; !ok {
			t.Errorf("first scenario missing key %q (camelCase regression?)", key)
		}
	}
	port := first["ports"].([]any)[0].(map[string]any)
	if port["listener_status"] != "listening" || port["port"] != float64(16542) {
		t.Errorf("port not mapped (snake_case?): %v", port)
	}

	// tags normalization: the sparse scenario's empty tags must be [] (never null).
	sparse := scenarios[1].(map[string]any)
	tags, ok := sparse["tags"].([]any)
	if !ok || len(tags) != 0 {
		t.Errorf("empty tags must serialize as [], got %v (%T)", sparse["tags"], sparse["tags"])
	}

	// Discovery failures carried through.
	fails, ok := got["discovery_failures"].([]any)
	if !ok || len(fails) != 1 {
		t.Fatalf("discovery_failures: want 1, got %v", got["discovery_failures"])
	}
}
