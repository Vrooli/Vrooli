package rehearsals

import (
	"context"
	"testing"
	"time"
)

func TestFilterRoutesKeptAndDiscardedItemsOnDeclaredPorts(t *testing.T) {
	raw := `{"id":"filter-ports","nodes":[
		{"id":"trigger","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},
		{"id":"filter","name":"Filter","type":"n8n-nodes-base.filter","typeVersion":2,"parameters":{"conditions":{"combinator":"and","conditions":[{"id":"condition","leftValue":"={{ $json.valid }}","operator":{"type":"boolean","operation":"true","singleValue":true},"rightValue":true}],"options":{"caseSensitive":true,"leftValue":"","typeValidation":"loose"}},"options":{}}},
		{"id":"keep","name":"Kept","type":"n8n-nodes-base.noOp","parameters":{}},
		{"id":"drop","name":"Discarded","type":"n8n-nodes-base.noOp","parameters":{}}
	],"connections":{"Manual":{"main":[[{"node":"Filter","type":"main","index":0}]]},"Filter":{"main":[[{"node":"Kept","type":"main","index":0}],[{"node":"Discarded","type":"main","index":0}]]}}}`
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize generic Filter ports: %v", err)
	}
	trace, err := executeGraphWithInputForTest(workflow, map[string]any{"valid": true})
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasNode(trace, "Kept") || traceHasNode(trace, "Discarded") {
		t.Fatalf("true predicate must route only the kept output: %#v", trace)
	}

	workflow, _, _, err = normalizeWorkflow(raw)
	if err != nil {
		t.Fatal(err)
	}
	trace, err = executeGraphWithInputForTest(workflow, map[string]any{"valid": false})
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasNode(trace, "Discarded") || traceHasNode(trace, "Kept") {
		t.Fatalf("false predicate must route only the discarded output: %#v", trace)
	}
}

func executeGraphWithInputForTest(workflow Workflow, input map[string]any) ([]TraceEvent, error) {
	trace := []TraceEvent{}
	_, err := executeWorkflowEntry(context.Background(), workflow, workflow.EntryNodeIDs[0], input, nil, false, time.Time{}, &trace, map[string]bool{}, 0)
	return trace, err
}
