package rehearsals

import (
	"context"
	"strings"
	"testing"
	"time"
)

// [REQ:CR-P0-001]
func TestGraphEngineBranchesAcrossDistinctAPIProfiles(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "workflow_branch.json"))
	if err != nil {
		t.Fatal(err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := (SyntheticEngine{}).Execute(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	if traceHasOperation(trace, "Sheet append") || !traceHasOperation(trace, "CRM create") {
		t.Fatalf("expected active branch to call only CRM, got %#v", trace)
	}

	// Exercise the alternate branch with a different synthetic trigger fixture.
	var alternate []TraceEvent
	if _, err := executeGraph(context.Background(), workflow, workflow.EntryNodeIDs[0], map[string]any{"status": "inactive"}, simulator, false, time.Unix(10, 0), &alternate, map[string]bool{}, 0); err != nil {
		t.Fatal(err)
	}
	if traceHasOperation(alternate, "CRM create") || !traceHasOperation(alternate, "Sheet append") {
		t.Fatalf("expected inactive branch to call only Sheets, got %#v", alternate)
	}
}

func TestGenericNativeServiceNodesResolveThroughAPIProfile(t *testing.T) {
	workflowJSON := readFixture(t, "native_service_workflow.json")
	workflow, _, replacements, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatal(err)
	}
	if replacements != 3 {
		t.Fatalf("expected three credential references to be replaced, got %d", replacements)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := (SyntheticEngine{}).Execute(context.Background(), workflow, "manual", simulator, false, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(trace) != 7 {
		t.Fatalf("expected trigger and 3 request/operation pairs, got %d trace events: %#v", len(trace), trace)
	}
	for _, expected := range []string{
		"n8n-nodes-base.gmail:message.send",
		"n8n-nodes-base.googleSheets:sheet.append",
		"n8n-nodes-base.hubspot:contact.upsert",
	} {
		found := false
		for _, event := range trace {
			if strings.Contains(event.Operation, expected) && event.Status == "succeeded" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("generic native operation %q did not resolve: %#v", expected, trace)
		}
	}
	for _, event := range trace {
		for _, forbidden := range []string{"fictional@example.invalid", "Synthetic rehearsal", "Made-up content", "stand-in-reference"} {
			if strings.Contains(event.RequestSummary, forbidden) || strings.Contains(event.ResponseSummary, forbidden) {
				t.Fatalf("trace leaked workflow input %q", forbidden)
			}
		}
	}
}

func TestGraphEngineExecutesIncludedSubworkflowBundle(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "workflow_subworkflow_bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "api_catalog_openapi.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := (SyntheticEngine{}).Execute(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Unix(20, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasOperation(trace, "child-flow/POST /v1/rows") || !traceHasOperation(trace, "Execute Workflow child-flow") {
		t.Fatalf("expected parent and included child workflow evidence, got %#v", trace)
	}
}

func TestGraphEngineEvaluatesLegacyAndCurrentSwitchExports(t *testing.T) {
	cases := []struct {
		name    string
		rules   string
		input   map[string]any
		chosen  string
		skipped string
	}{
		{
			name:   "legacy rules",
			rules:  `{"value1":"={{$node[\"Webhook\"].json[\"body\"][\"action\"]}}","rules":{"rules":[{"value2":"edited"},{"output":1,"value2":"deleted"}]}}`,
			input:  map[string]any{"body": map[string]any{"action": "deleted"}},
			chosen: "Delete path", skipped: "Edit path",
		},
		{
			name:   "current condition rules",
			rules:  `{"rules":{"values":[{"conditions":{"conditions":[{"leftValue":"={{ $json.body.action }}","rightValue":"create","operator":{"type":"string","operation":"equals"}}]}},{"conditions":{"conditions":[{"leftValue":"={{ $json.body.action }}","rightValue":"delete","operator":{"type":"string","operation":"equals"}}]}}]}}`,
			input:  map[string]any{"body": map[string]any{"action": "create"}},
			chosen: "Edit path", skipped: "Delete path",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			workflowJSON := `{"id":"switch-flow","nodes":[{"id":"trigger","name":"Webhook","type":"n8n-nodes-base.webhook","parameters":{}},{"id":"switch","name":"Switch","type":"n8n-nodes-base.switch","parameters":` + test.rules + `},{"id":"left","name":"Edit path","type":"n8n-nodes-base.noOp","parameters":{}},{"id":"right","name":"Delete path","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Webhook":{"main":[[{"node":"Switch"}]]},"Switch":{"main":[[{"node":"Edit path"}],[{"node":"Delete path"}]]}}}`
			workflow, _, _, err := normalizeWorkflow(workflowJSON)
			if err != nil {
				t.Fatal(err)
			}
			var trace []TraceEvent
			if _, err := executeGraph(context.Background(), workflow, workflow.EntryNodeIDs[0], test.input, nil, false, time.Unix(25, 0), &trace, map[string]bool{}, 0); err != nil {
				t.Fatal(err)
			}
			if !traceHasNode(trace, test.chosen) || traceHasNode(trace, test.skipped) {
				t.Fatalf("switch selected wrong output: %#v", trace)
			}
		})
	}
}

func TestGraphEngineRequiresAndUsesSelectedTrigger(t *testing.T) {
	workflowJSON := `{"id":"multi-trigger","nodes":[{"id":"manual","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"scheduled","name":"Schedule","type":"n8n-nodes-base.scheduleTrigger","parameters":{"rule":{"interval":[]}}},{"id":"finish","name":"Finish","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Manual":{"main":[[{"node":"Finish"}]]},"Schedule":{"main":[[{"node":"Finish"}]]}}}`
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(workflow.EntryNodeIDs) != 2 || len(workflow.TriggerKinds) != 2 {
		t.Fatalf("expected two independently classified triggers: %#v", workflow)
	}
	if _, err := (SyntheticEngine{}).Execute(context.Background(), workflow, "", nil, false, time.Time{}); err == nil {
		t.Fatal("must require a specific entry trigger for a multi-trigger workflow")
	}
	trace, err := (SyntheticEngine{}).Execute(context.Background(), workflow, "scheduled", nil, false, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasOperation(trace, "scheduled Trigger") || traceHasOperation(trace, "manual Trigger") {
		t.Fatalf("expected only selected scheduled trigger to execute: %#v", trace)
	}
}

func TestGraphEngineRunsConfiguredErrorWorkflowAfterSyntheticFailure(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "workflow_error_bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "error_path_recorded_exchanges.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	trace, runErr := (SyntheticEngine{}).Execute(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Unix(30, 0))
	if runErr == nil || !strings.Contains(runErr.Error(), "synthetic HTTP 503") {
		t.Fatalf("expected original synthetic failure to remain visible, got %v", runErr)
	}
	if !traceHasOperation(trace, "error-flow/POST /v1/events") {
		t.Fatalf("expected configured error workflow to run after failure, got %#v", trace)
	}
	if !traceHasSuccessfulOperation(trace, "error-flow/POST /v1/events") {
		t.Fatalf("expected error handler success evidence, got %#v", trace)
	}
}

func TestGraphEngineRejectsMissingSubworkflowBeforeAnyNetworkUse(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(`{"id":"main","nodes":[{"id":"t","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"call","name":"Call child","type":"n8n-nodes-base.executeWorkflow","parameters":{"workflowId":{"value":"missing"}}}],"connections":{"Manual":{"main":[[{"node":"Call child"}]]}}}`)
	if err == nil {
		t.Fatal("expected missing child workflow to fail closed at import")
	}
	_ = workflow
}

func traceHasOperation(trace []TraceEvent, part string) bool {
	for _, event := range trace {
		if strings.Contains(event.Operation, part) {
			return true
		}
	}
	return false
}

func traceHasSuccessfulOperation(trace []TraceEvent, part string) bool {
	for _, event := range trace {
		if strings.Contains(event.Operation, part) && event.Status == "succeeded" {
			return true
		}
	}
	return false
}

func traceHasNode(trace []TraceEvent, name string) bool {
	for _, event := range trace {
		if event.RequestSummary == name {
			return true
		}
	}
	return false
}
