package rehearsals

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPublic16667NormalizesInstanceRestartAsAnEventTrigger(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/16667.json"))
	if err != nil {
		t.Fatalf("normalize sanitized public instance-event workflow: %v", err)
	}
	if len(workflow.EntryNodeIDs) != 1 {
		t.Fatalf("public instance event has %d entry nodes, want one", len(workflow.EntryNodeIDs))
	}
	var entry WorkflowNode
	for _, node := range workflow.Nodes {
		if node.ID == workflow.EntryNodeIDs[0] {
			entry = node
			break
		}
	}
	if entry.Type != "n8n-nodes-base.n8nTrigger" || entry.Trigger != TriggerEvent {
		t.Fatalf("instance-start node normalized as %#v, want event trigger", entry)
	}
	if hasServiceOperation(workflow, "n8n-nodes-base.discord:webhook.sendLegacy") || !hasServiceOperation(workflow, "n8n-nodes-base.n8n:execution.getAll") || hasServiceOperation(workflow, "n8n-nodes-base.telegram:message.sendMessage") || !hasServiceOperation(workflow, "n8n-nodes-base.slack:message.post") {
		t.Fatalf("public workflow native operations did not resolve through bounded source defaults: %#v", workflow.Nodes)
	}
	for _, node := range workflow.Nodes {
		if node.Name == "Send Telegram Message" || node.Name == "Ping Discord Channel" {
			t.Fatalf("disabled notification node %q must be omitted from the executable graph", node.Name)
		}
	}
}

func TestInstanceEventTriggerRejectsUnsimulatedEvents(t *testing.T) {
	for _, params := range []map[string]any{
		{"events": []any{"workflowCreated"}},
		{"events": []any{"init", "workflowCreated"}},
		{"events": []any{"init"}, "credentials": map[string]any{"unexpected": true}},
	} {
		if _, err := normalizeWorkflowNodeVersion("event", "Instance event", "n8n-nodes-base.n8nTrigger", params, 1); err == nil {
			t.Errorf("accepted n8n instance trigger parameters outside the simulated init event: %#v", params)
		}
	}
}

func hasServiceOperation(workflow Workflow, operation string) bool {
	for _, node := range workflow.Nodes {
		if node.Request != nil && node.Request.Operation == operation {
			return true
		}
	}
	return false
}

func TestPublic16667InstanceEventRunsGenericExecutionSummaryAndEnabledSyntheticAlerts(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/16667.json"))
	if err != nil {
		t.Fatalf("normalize sanitized public instance-event workflow: %v", err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-16667-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic n8n and notification profiles: %v", err)
	}
	startedAt := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, startedAt, map[string]any{"timestamp": startedAt.Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatalf("execute public instance-event workflow through generated synthetic services: %v; trace=%#v", err, run.Trace)
	}
	trace := strings.Join(traceSummaries(run.Trace), "\n")
	for _, nodeName := range []string{"Fetch Execution Logs", "Filter Recent Executions", "Summarize Execution Data", "Sort Execution Summary", "Merge Execution Data", "Process Data in JavaScript", "Post to Slack Channel"} {
		if !strings.Contains(trace, nodeName) {
			t.Errorf("public event execution omitted node %q; trace:\n%s", nodeName, trace)
		}
	}
	for _, operation := range []string{"n8n-nodes-base.n8n:execution.getAll", "n8n-nodes-base.slack:message.post"} {
		if !traceHasSuccessfulOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q to succeed; trace=%#v", operation, run.Trace)
		}
	}
}
