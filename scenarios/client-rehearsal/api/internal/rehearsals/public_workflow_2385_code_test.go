package rehearsals

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPublic2385WorkflowComparisonCodeHandlesNestedObjectsAndIgnoredFields(t *testing.T) {
	var exported struct {
		Nodes []struct {
			ID, Name, Type string
			Parameters     map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/2385.json")), &exported); err != nil {
		t.Fatal(err)
	}
	var code WorkflowNode
	for _, node := range exported.Nodes {
		if node.Type == "n8n-nodes-base.code" {
			var err error
			code, err = normalizeWorkflowNode(node.ID, node.Name, node.Type, node.Parameters)
			if err != nil {
				t.Fatalf("normalize exported Code node: %v", err)
			}
			break
		}
	}
	if code.SafeCode == "" {
		t.Fatal("public comparison Code node not found")
	}
	original := map[string]any{"id": "workflow-7", "name": "example", "updatedAt": "2026-01-01", "global": map[string]any{"version": 1}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "open"}}}}
	changedIgnored := map[string]any{"id": "workflow-7", "name": "example", "updatedAt": "2026-11-11", "global": map[string]any{"version": 2}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "open"}}}}
	for _, tc := range []struct {
		name  string
		saved map[string]any
		want  string
	}{{"same except ignored fields", changedIgnored, "same"}, {"nested meaningful edit", map[string]any{"id": "workflow-7", "name": "example", "updatedAt": "2026-11-11", "global": map[string]any{"version": 2}, "nodes": []any{map[string]any{"id": "n1", "parameters": map[string]any{"status": "closed"}}}}, "diff"}} {
		t.Run(tc.name, func(t *testing.T) {
			item := map[string]any{"workflow-from-n8n": original, "workflow-from-gitlab": tc.saved}
			got, err := executeSafeCodePerItemWithNodeOutputs(code.SafeCode, item, nil, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			data := got["json"].(map[string]any)
			if data["status"] != tc.want {
				t.Fatalf("status=%v want %s", data["status"], tc.want)
			}
		})
	}
}
