package rehearsals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublic20819ErrorWorkflowRunsAgainstSyntheticSheetsAndTelegram(t *testing.T) {
	raw := readFixture(t, "public-workflows/20819.json")
	var handler map[string]any
	if err := json.Unmarshal([]byte(raw), &handler); err != nil {
		t.Fatal(err)
	}
	handler["id"] = "synthetic-template"
	parent := map[string]any{
		"id": "synthetic-primary-flow", "name": "Synthetic primary flow",
		"settings": map[string]any{"errorWorkflow": "synthetic-template"},
		"nodes": []any{
			map[string]any{"id": "start", "name": "Manual", "type": "n8n-nodes-base.manualTrigger", "parameters": map[string]any{}},
			map[string]any{"id": "operation", "name": "Synthetic primary operation", "type": "n8n-nodes-base.httpRequest", "parameters": map[string]any{"method": "GET", "url": "https://unmapped.synthetic.invalid/failure"}},
		},
		"connections": map[string]any{"Manual": map[string]any{"main": []any{[]any{map[string]any{"node": "Synthetic primary operation", "type": "main", "index": 0}}}}},
	}
	bundle, err := json.Marshal(map[string]any{"entryWorkflowId": "synthetic-primary-flow", "workflows": []any{parent, handler}})
	if err != nil {
		t.Fatal(err)
	}
	workflow, _, _, err := normalizeWorkflow(string(bundle))
	if err != nil {
		t.Fatalf("normalize public error workflow: %v", err)
	}
	if len(workflow.Nodes) != 2 || workflow.ErrorWorkflowID != "synthetic-template" || workflow.Subworkflows["synthetic-template"] == nil {
		t.Fatalf("expected parent graph bound to imported public error handler; %#v", workflow)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-20819-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic Sheets and Telegram profiles: %v", err)
	}
	input := map[string]any{"execution": map[string]any{
		"id": "synthetic-execution-42", "lastNodeExecuted": "Synthetic CRM Upsert", "url": "https://n8n.synthetic.invalid/execution/42",
		"error": map[string]any{"name": "SyntheticFailure", "message": "synthetic CRM rejected test record", "node": map[string]any{"name": "Synthetic CRM Upsert"}},
	}, "workflow": map[string]any{"name": "Synthetic contact rehearsal"}}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC), input)
	if err == nil {
		t.Fatal("expected the primary synthetic operation to fail while the attached public error handler succeeds")
	}

	var unavailableProfile map[string]any
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflow-20819-profile.json")), &unavailableProfile); err != nil {
		t.Fatal(err)
	}
	doc := unavailableProfile["document"].(map[string]any)
	paths := doc["paths"].(map[string]any)
	delete(paths, "/v1/sheets/error-log")
	unavailableJSON, err := json.Marshal(unavailableProfile)
	if err != nil {
		t.Fatal(err)
	}
	unavailableSimulator, err := NewDescriptionSimulator(string(unavailableJSON), false)
	if err != nil {
		t.Fatalf("load synthetic Telegram-only profile to exercise Sheets failure: %v", err)
	}
	failedSheet, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], unavailableSimulator, false, time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC), input)
	if err == nil || !strings.Contains(err.Error(), "Synthetic CRM") && !strings.Contains(err.Error(), "no matching synthetic API operation") {
		t.Fatalf("expected the primary operation failure to remain visible after the configured handler runs, got %v", err)
	}
	for _, node := range []string{"Create Sheet Down Alert Message", "Send Sheet Down Alert to Telegram"} {
		if !strings.Contains(strings.Join(traceSummaries(failedSheet.Trace), " "), node) {
			t.Errorf("expected missing Sheets operation to follow public error output through %q; trace=%#v", node, failedSheet.Trace)
		}
	}
	failedAppend, sentFallback := false, false
	for _, event := range failedSheet.Trace {
		if strings.Contains(event.Operation, "googleSheets:sheet.append") && event.Status == "failed" {
			failedAppend = true
		}
		if strings.Contains(event.Operation, "telegram:message.sendMessage") && event.Status == "succeeded" {
			sentFallback = true
		}
	}
	if !failedAppend || !sentFallback {
		t.Fatalf("expected the unsupported Sheet operation to fail locally then the alternate Telegram alert to complete: %#v", failedSheet.Trace)
	}
	for _, node := range []string{"Prepare Error Record", "Append Error to Sheets", "Create Failure Alert Message", "Send Failure Alert to Telegram"} {
		if !strings.Contains(strings.Join(traceSummaries(run.Trace), " "), node) {
			t.Errorf("expected workflow node %q to execute; trace=%#v", node, run.Trace)
		}
	}
	if !traceHasSuccessfulOperation(run.Trace, "googleSheets:sheet.append") || !traceHasSuccessfulOperation(run.Trace, "telegram:message.sendMessage") {
		t.Fatalf("expected synthetic Sheets append and Telegram alert; trace=%#v", run.Trace)
	}
}
