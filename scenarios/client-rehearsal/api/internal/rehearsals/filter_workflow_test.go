package rehearsals

import (
	"context"
	"strings"
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

func TestFilterAlwaysOutputDataContinuesOnKeptPortWhenNoItemsMatch(t *testing.T) {
	raw := `{"id":"filter-always-output","nodes":[
		{"id":"trigger","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},
		{"id":"filter","name":"Filter","type":"n8n-nodes-base.filter","typeVersion":2,"alwaysOutputData":true,"parameters":{"conditions":{"combinator":"and","conditions":[{"id":"condition","leftValue":"={{ $json.valid }}","operator":{"type":"boolean","operation":"true","singleValue":true},"rightValue":true}],"options":{"caseSensitive":true,"leftValue":"","typeValidation":"loose"}},"options":{}}},
		{"id":"kept","name":"Kept","type":"n8n-nodes-base.noOp","parameters":{}},
		{"id":"discarded","name":"Discarded","type":"n8n-nodes-base.noOp","parameters":{}}
	],"connections":{"Manual":{"main":[[{"node":"Filter","type":"main","index":0}]]},"Filter":{"main":[[{"node":"Kept","type":"main","index":0}],[{"node":"Discarded","type":"main","index":0}]]}}}`
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize always-output Filter: %v", err)
	}
	trace, err := executeGraphWithInputForTest(workflow, map[string]any{"valid": false})
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasNode(trace, "Kept") || traceHasNode(trace, "Discarded") {
		t.Fatalf("empty always-output fallback must continue on the kept output only: %#v", trace)
	}

	var captured []map[string]any
	trace = []TraceEvent{}
	_, err = executeGraphMany(context.Background(), workflow, workflow.EntryNodeIDs[0], []map[string]any{{"valid": false}, {"valid": true}}, nil, false, time.Time{}, &trace, map[string]bool{}, 0, "kept", &captured, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(captured) != 1 || captured[0]["valid"] != true {
		t.Fatalf("when some input passes, only matching items should continue without a fallback placeholder: %#v", captured)
	}
}

func TestIfRoutesAcrossBoundedAndOrStringConditions(t *testing.T) {
	raw := `{"id":"if-multiple-conditions","nodes":[
		{"id":"trigger","name":"Webhook","type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST"}},
		{"id":"if","name":"Required email and phone","type":"n8n-nodes-base.if","typeVersion":2,"parameters":{"conditions":{"combinator":"and","conditions":[
			{"id":"email","leftValue":"={{ $json.body.email }}","operator":{"type":"string","operation":"notEmpty"},"rightValue":""},
			{"id":"phone","leftValue":"={{ $json.body.phone }}","operator":{"type":"string","operation":"notEmpty"},"rightValue":""}
		],"options":{"caseSensitive":true,"leftValue":"","typeValidation":"strict","version":1}}}},
		{"id":"accepted","name":"Accepted","type":"n8n-nodes-base.noOp","parameters":{}},
		{"id":"rejected","name":"Rejected","type":"n8n-nodes-base.noOp","parameters":{}}
	],"connections":{"Webhook":{"main":[[{"node":"Required email and phone","type":"main","index":0}]]},"Required email and phone":{"main":[[{"node":"Accepted","type":"main","index":0}],[{"node":"Rejected","type":"main","index":0}]]}}}`
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize bounded multi-condition IF: %v", err)
	}
	if len(workflow.Nodes) < 2 || len(workflow.Nodes[1].IfConditions) != 2 || workflow.Nodes[1].IfCombinator != "and" {
		t.Fatalf("expected both AND conditions to be retained, got %#v", workflow.Nodes)
	}

	for _, test := range []struct {
		name     string
		input    map[string]any
		accepted bool
	}{
		{name: "both values present", input: map[string]any{"body": map[string]any{"email": "a@example.invalid", "phone": "+15550100"}}, accepted: true},
		{name: "missing email", input: map[string]any{"body": map[string]any{"phone": "+15550100"}}},
		{name: "empty phone", input: map[string]any{"body": map[string]any{"email": "a@example.invalid", "phone": ""}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			trace, runErr := executeGraphWithInputForTest(workflow, test.input)
			if runErr != nil {
				t.Fatal(runErr)
			}
			if traceHasNode(trace, "Accepted") != test.accepted || traceHasNode(trace, "Rejected") == test.accepted {
				t.Fatalf("IF output did not match expected AND behavior: %#v", trace)
			}
		})
	}

	orWorkflow, _, _, err := normalizeWorkflow(strings.ReplaceAll(raw, `"combinator":"and"`, `"combinator":"or"`))
	if err != nil {
		t.Fatalf("normalize bounded OR IF: %v", err)
	}
	trace, err := executeGraphWithInputForTest(orWorkflow, map[string]any{"body": map[string]any{"email": "a@example.invalid"}})
	if err != nil || !traceHasNode(trace, "Accepted") || traceHasNode(trace, "Rejected") {
		t.Fatalf("OR IF with one present field must accept: trace=%#v err=%v", trace, err)
	}
}

func executeGraphWithInputForTest(workflow Workflow, input map[string]any) ([]TraceEvent, error) {
	trace := []TraceEvent{}
	_, err := executeWorkflowEntry(context.Background(), workflow, workflow.EntryNodeIDs[0], input, nil, false, time.Time{}, &trace, map[string]bool{}, 0)
	return trace, err
}
