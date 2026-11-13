package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The public export is a reusable child workflow, so a synthetic parent invokes
// its Execute Workflow Trigger without modifying the child's source graph.
func TestPublic5852CentralizedErrorWorkflowRunsThroughSyntheticSubworkflowParent(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/5852.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "841b9a2120ef937d578f8279e71aa844edae1b613808945a045a3fdd8204e24d" {
		t.Fatalf("sanitized fixture changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"@gmail.com", "+1234567890", "YOUR_CHAT_ID", "61Flju0VYVpEMHzD"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public fixture retained source destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize unchanged public child graph: %v", err)
	}
	if len(workflow.Nodes) != 3 || len(workflow.EntryNodeIDs) != 0 || workflow.SubworkflowEntryID == "" {
		t.Fatalf("disabled nodes were not omitted while retaining the subworkflow entry: nodes=%d entries=%v subworkflowEntry=%q", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.SubworkflowEntryID)
	}
	if workflow.Nodes[0].Name != "When Executed by Another Workflow" || workflow.Nodes[1].Name != "Prepare Messages For Notify" || workflow.Nodes[2].Name != "Send Notify to Telegram" {
		t.Fatalf("disabled nodes were not omitted and bypassed: %#v", workflow.Nodes)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-5852-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic-only service profile: %v", err)
	}
	parentBundle := fmt.Sprintf(`{"entryWorkflowId":"synthetic-parent","workflows":[{"id":"synthetic-parent","name":"Synthetic parent","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Invoke public child","type":"n8n-nodes-base.executeWorkflow","parameters":{"workflowId":{"value":"public-template-5852","mode":"list"}}}],"connections":{"Start":{"main":[[{"node":"Invoke public child"}]]}}},%s]}`, workflowJSON)
	parent, _, _, err := normalizeWorkflow(parentBundle)
	if err != nil {
		t.Fatalf("normalize synthetic invocation harness with unchanged public child: %v", err)
	}
	child := parent.Subworkflows["public-template-5852"]
	if child == nil || len(child.Nodes) != 3 || child.SubworkflowEntryID == "" {
		t.Fatalf("synthetic harness did not retain disabled-reduced public child graph: %#v", child)
	}
	run, err := executeDetailedWithInput(context.Background(), parent, parent.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC), map[string]any{
		"workflow":  map[string]any{"name": "Synthetic workflow"},
		"execution": map[string]any{"id": "exec-synthetic-01", "url": "https://n8n.synthetic.invalid/executions/exec-synthetic-01", "lastNodeExecuted": "Synthetic step", "error": map[string]any{"message": "Synthetic failure", "level": "error", "timestamp": "2026-11-12T12:00:00Z", "context": map[string]any{"messageTemplate": "Synthetic description"}}},
	})
	if err != nil {
		t.Fatalf("execute unchanged public child through its subworkflow trigger: %v", err)
	}
	if !traceHasSuccessfulOperation(run.Trace, "n8n-nodes-base.telegram:message.sendMessage") {
		t.Fatalf("public child did not reach synthetic Telegram handoff: %#v", run.Trace)
	}
}
