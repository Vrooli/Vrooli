package rehearsals

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAggregateNodeUsesBoundedGenericIndividualFieldSemantics(t *testing.T) {
	params := map[string]any{
		"fieldsToAggregate": map[string]any{"fieldToAggregate": []any{
			map[string]any{"fieldToAggregate": "person.email"},
		}},
	}
	fields, keepMissing, mergeLists, disableDot, err := normalizeAggregate(params)
	if err != nil || len(fields) != 1 || fields[0].InputPath != "person.email" || fields[0].OutputName != "email" || keepMissing || mergeLists || disableDot {
		t.Fatalf("unexpected normalized Aggregate semantics: %#v %t %t %t %v", fields, keepMissing, mergeLists, disableDot, err)
	}
	workflowNode := WorkflowNode{AggregateFields: fields}
	got, err := aggregateItems([]map[string]any{
		{"person": map[string]any{"email": "person.one@example.invalid"}},
		{"person": map[string]any{"email": "person.two@example.invalid"}},
		{"person": map[string]any{}},
	}, workflowNode)
	if err != nil {
		t.Fatal(err)
	}
	values, ok := got["email"].([]any)
	if !ok || len(values) != 2 || values[0] != "person.one@example.invalid" || values[1] != "person.two@example.invalid" {
		t.Fatalf("Aggregate must collect present field values in input order: %#v", got)
	}

	params["aggregate"] = "aggregateAllItemData"
	if _, _, _, _, err := normalizeAggregate(params); err == nil {
		t.Fatal("unsupported Aggregate mode must fail closed")
	}
}

func TestPublic5889SanitizedExportRunsScheduleBranchesAndAggregateGenerically(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/5889.json")
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public 5889 export: %v", err)
	}
	if len(workflow.Nodes) != 11 || len(workflow.Connections) != 10 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerScheduled {
		t.Fatalf("expected sanitized scheduled 14-node export minus three notes, got %d nodes, %d edges, entries=%v triggers=%v", len(workflow.Nodes), len(workflow.Connections), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	var aggregateNode WorkflowNode
	for _, node := range workflow.Nodes {
		if node.Type == "n8n-nodes-base.aggregate" {
			aggregateNode = node
		}
	}
	if len(aggregateNode.AggregateFields) != 1 || aggregateNode.AggregateFields[0].InputPath != "Email" {
		t.Fatalf("expected generic Email field aggregation: %#v", aggregateNode)
	}
	if strings.Contains(workflowJSON, "From6Agency") || strings.Contains(workflowJSON, "cachedResultUrl") || strings.Contains(strings.ToLower(workflowJSON), "credentials") || strings.Contains(workflowJSON, "notifications@noreply") {
		t.Fatal("sanitized fixture retained source repository, credential reference, or note contents")
	}

	profile := readFixture(t, "public-workflow-5889-profile.json")
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatalf("load synthetic Notion/GitHub/Gmail descriptions: %v", err)
	}
	run, err := (SyntheticEngine{}).ExecuteDetailed(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("run sanitized public workflow: %v", err)
	}
	for _, operation := range []string{"notion:databasePage.getAll", "github:issue.create", "notion:databasePage.update", "notion:user.getAll", "gmail:message.send"} {
		if !traceHasOperation(run.Trace, operation) {
			t.Errorf("expected generic synthetic operation %q: %#v", operation, run.Trace)
		}
	}
	if !traceHasNode(run.Trace, "Group Recipients") {
		t.Fatalf("expected Aggregate node to run on users from the Notion list response: %#v", run.Trace)
	}
	if len(run.Output) == 0 {
		t.Fatal("expected synthetic final output")
	}
	emails, ok := run.Output["Email"].([]any)
	if !ok || len(emails) != 2 || emails[0] != "person@example.invalid" || emails[1] != "person@example.invalid" {
		t.Fatalf("expected two synthetic person recipients and no bot: %#v", run.Output)
	}
	for _, event := range run.Trace {
		if strings.Contains(event.ResponseSummary, "From6Agency") || strings.Contains(event.ResponseSummary, "person.one@example.invalid") {
			t.Fatalf("trace leaked source or simulator fixture detail: %#v", event)
		}
	}
}
