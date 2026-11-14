package rehearsals

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublic16667CodeNodeFitsBoundedGenericInterpreter(t *testing.T) {
	var exported struct {
		Nodes []struct {
			Type       string         `json:"type"`
			Parameters map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/16667.json")), &exported); err != nil {
		t.Fatalf("decode sanitized public export: %v", err)
	}
	codeNodes := 0
	for _, node := range exported.Nodes {
		if node.Type != "n8n-nodes-base.code" {
			continue
		}
		codeNodes++
		source, _ := node.Parameters["jsCode"].(string)
		if source == "" {
			t.Fatal("public Code node has no source")
		}
		if _, err := parseSafeCodeMode(source, false, false); err != nil {
			t.Fatalf("public Code node outside generic bounded subset: %v", err)
		}
	}
	if codeNodes != 1 {
		t.Fatalf("sanitized public workflow contains %d Code nodes, want 1", codeNodes)
	}
}

func TestPublic16667CodeProducesSyntheticMultiChannelRestartSummaries(t *testing.T) {
	var exported struct {
		Nodes []struct {
			Type       string         `json:"type"`
			Parameters map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/16667.json")), &exported); err != nil {
		t.Fatalf("decode sanitized public export: %v", err)
	}
	var source string
	for _, node := range exported.Nodes {
		if node.Type == "n8n-nodes-base.code" {
			source, _ = node.Parameters["jsCode"].(string)
			break
		}
	}
	if source == "" {
		t.Fatal("public Code node source missing")
	}
	prior := map[string][]map[string]any{
		"Set Execution Parameters": {{
			"time_instance_start":        "2026-11-11T12:00:00.000Z",
			"minutes_to_fetch_execution": float64(120),
			"discord_mentions":           []any{"10001"},
			"slack_mentions":             []any{"20002"},
			"telegram_mentions":          []any{"30003"},
		}},
		"Fetch Execution Logs":   {{"startedAt": "2026-11-11T10:33:21.000Z"}},
		"Sort Execution Summary": {{"workflowId": "wf-synthetic-42", "count_id": float64(3)}},
	}
	got, err := executeSafeCodeWithNodeOutputs(source, nil, prior, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("execute bounded public Code transform: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Code returned %d items, want one", len(got))
	}
	output, _ := got[0]["json"].(map[string]any)
	for field, marker := range map[string]string{
		"discord_message":  "https://n8n.synthetic.invalid/workflow/wf-synthetic-42/executions",
		"slack_message":    "<https://n8n.synthetic.invalid/workflow/wf-synthetic-42/executions|wf-synthetic-42>",
		"telegram_message": "1h 26m 39s",
	} {
		message, _ := output[field].(string)
		if !strings.Contains(message, marker) {
			t.Errorf("%s missing %q: %q", field, marker, message)
		}
	}
	prior["Sort Execution Summary"] = nil
	empty, err := executeSafeCodeWithNodeOutputs(source, nil, prior, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("execute empty-summary branch: %v", err)
	}
	emptyJSON, _ := empty[0]["json"].(map[string]any)
	for _, field := range []string{"discord_message", "slack_message", "telegram_message"} {
		if !strings.Contains(emptyJSON[field].(string), "All good no failed workflow recently") {
			t.Errorf("%s missing empty-state summary: %q", field, emptyJSON[field])
		}
	}
}
