package rehearsals

import (
	"testing"
	"time"
)

func TestIfConditionSupportsPairedNamedNodeItemReference(t *testing.T) {
	path, nodeName, mode, sourceOK := parseConditionJSONExpressionSource("={{ $('Route by Failure Type').item.json.order_value }}")
	if !sourceOK || path != "order_value" || nodeName != "Route by Failure Type" || mode != "item" {
		t.Fatalf("unexpected parsed item reference: path=%q node=%q mode=%q", path, nodeName, mode)
	}
	workflow := `{"name":"Named item condition","nodes":[{"name":"Start","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Route by Failure Type","type":"n8n-nodes-base.noOp","parameters":{}},{"name":"High value?","type":"n8n-nodes-base.if","parameters":{"conditions":{"number":[{"leftValue":"={{ $('Route by Failure Type').item.json.order_value }}","rightValue":500,"operation":"gte"}]}}},{"name":"True","type":"n8n-nodes-base.noOp","parameters":{}},{"name":"False","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Start":{"main":[[{"node":"Route by Failure Type"}]]},"Route by Failure Type":{"main":[[{"node":"High value?"}]]},"High value?":{"main":[[{"node":"True"}],[{"node":"False"}]]}}}`
	parsed, _, _, err := normalizeWorkflow(workflow)
	if err != nil {
		t.Fatalf("normalize named-item condition: %v", err)
	}
	var condition Condition
	for _, node := range parsed.Nodes {
		if node.Name == "High value?" && node.Condition != nil {
			condition = *node.Condition
		}
	}
	if condition.NodeName != "Route by Failure Type" || condition.NodeOutputMode != "item" || condition.Path != "order_value" || condition.Operator != "gte" || condition.Value != float64(500) {
		t.Fatalf("unexpected normalized item condition: %#v", condition)
	}
	now := time.Unix(0, 0).UTC()
	single := map[string][]map[string]any{"Route by Failure Type": {{"order_value": float64(700)}}}
	if !evaluateGraphCondition(condition, nil, single, now) {
		t.Fatal("paired current-item reference with exactly one predecessor output should evaluate true")
	}
	if evaluateGraphCondition(condition, nil, map[string][]map[string]any{"Route by Failure Type": {{"order_value": float64(700)}, {"order_value": float64(100)}}}, now) {
		t.Fatal("item reference must fail closed when item pairing is ambiguous across multiple predecessor outputs")
	}
}

func TestIfConditionSupportsExplicitFirstNamedNodeOutput(t *testing.T) {
	condition, err := conditionFromEntry(map[string]any{
		"leftValue":  "={{ $('Route by Failure Type').first().json.order_value }}",
		"rightValue": float64(500),
		"operator":   map[string]any{"type": "number", "operation": "gte"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if condition.NodeName != "Route by Failure Type" || condition.NodeOutputMode != "first" || condition.Path != "order_value" {
		t.Fatalf("unexpected normalized first-item condition: %#v", condition)
	}
	prior := map[string][]map[string]any{"Route by Failure Type": {{"order_value": float64(700)}, {"order_value": float64(100)}}}
	if !evaluateGraphCondition(condition, nil, prior, time.Unix(0, 0).UTC()) {
		t.Fatal("explicit first() reference should read the first bounded node output")
	}
}
