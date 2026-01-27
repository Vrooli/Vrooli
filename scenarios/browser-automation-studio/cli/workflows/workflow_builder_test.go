package workflows

import (
	"testing"
)

func TestBuildWorkflowFromSteps_NavigatePositionalURL(t *testing.T) {
	// Test that positional argument for navigate maps to "url" (not "selector")
	steps := []*StepSpec{
		{Type: "navigate", Positional: "https://example.com", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes, ok := workflow["nodes"].([]map[string]any)
	if !ok {
		t.Fatal("expected nodes to be a slice")
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	action, ok := node["action"].(map[string]any)
	if !ok {
		t.Fatal("expected action to be a map")
	}

	if action["type"] != "ACTION_TYPE_NAVIGATE" {
		t.Errorf("expected type 'ACTION_TYPE_NAVIGATE', got %q", action["type"])
	}

	// The navigate params are under the "navigate" key (protojson uses camelCase)
	navigate, ok := action["navigate"].(map[string]any)
	if !ok {
		t.Fatal("expected navigate params to be a map")
	}

	// Positional for navigate should map to "url", not "selector"
	if navigate["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %q", navigate["url"])
	}
}

func TestBuildWorkflowFromSteps_NavigateWithURL(t *testing.T) {
	steps := []*StepSpec{
		{
			Type: "navigate",
			KVPairs: map[string]string{
				"url": "https://example.com",
			},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_NAVIGATE" {
		t.Errorf("expected type 'ACTION_TYPE_NAVIGATE', got %q", action["type"])
	}

	navigate := action["navigate"].(map[string]any)
	if navigate["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %q", navigate["url"])
	}
}

func TestBuildWorkflowFromSteps_NavigateScenario(t *testing.T) {
	// Test scenario-based navigation (e.g., --step navigate scenario=my-app path=/dashboard)
	steps := []*StepSpec{
		{
			Type: "navigate",
			KVPairs: map[string]string{
				"scenario": "browser-automation-studio",
				"path":     "/workflows",
			},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_NAVIGATE" {
		t.Errorf("expected type 'ACTION_TYPE_NAVIGATE', got %q", action["type"])
	}

	navigate := action["navigate"].(map[string]any)

	// Verify scenario field is set
	if navigate["scenario"] != "browser-automation-studio" {
		t.Errorf("expected scenario 'browser-automation-studio', got %q", navigate["scenario"])
	}

	// Verify scenarioPath field is set (protojson uses camelCase)
	if navigate["scenarioPath"] != "/workflows" {
		t.Errorf("expected scenarioPath '/workflows', got %q", navigate["scenarioPath"])
	}

	// Verify destinationType is set to SCENARIO
	if navigate["destinationType"] != "NAVIGATE_DESTINATION_TYPE_SCENARIO" {
		t.Errorf("expected destinationType 'NAVIGATE_DESTINATION_TYPE_SCENARIO', got %q", navigate["destinationType"])
	}
}

func TestBuildWorkflowFromSteps_Click(t *testing.T) {
	steps := []*StepSpec{
		{Type: "click", Positional: "#submit", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_CLICK" {
		t.Errorf("expected type 'ACTION_TYPE_CLICK', got %q", action["type"])
	}

	click := action["click"].(map[string]any)
	if click["selector"] != "#submit" {
		t.Errorf("expected selector '#submit', got %q", click["selector"])
	}
}

func TestBuildWorkflowFromSteps_Type(t *testing.T) {
	steps := []*StepSpec{
		{
			Type:       "type",
			Positional: "#email",
			KVPairs:    map[string]string{"text": "user@example.com"},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	// "type" maps to ACTION_TYPE_INPUT
	if action["type"] != "ACTION_TYPE_INPUT" {
		t.Errorf("expected type 'ACTION_TYPE_INPUT', got %q", action["type"])
	}

	input := action["input"].(map[string]any)
	if input["selector"] != "#email" {
		t.Errorf("expected selector '#email', got %q", input["selector"])
	}
	// "text" is mapped to "value" in BuildInputParams
	if input["value"] != "user@example.com" {
		t.Errorf("expected value 'user@example.com', got %q", input["value"])
	}
}

func TestBuildWorkflowFromSteps_Assert(t *testing.T) {
	steps := []*StepSpec{
		{
			Type:       "assert",
			Positional: "#result",
			KVPairs:    map[string]string{"assertMode": "exists"},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_ASSERT" {
		t.Errorf("expected type 'ACTION_TYPE_ASSERT', got %q", action["type"])
	}

	assert := action["assert"].(map[string]any)
	if assert["selector"] != "#result" {
		t.Errorf("expected selector '#result', got %q", assert["selector"])
	}
	// assertMode maps to "mode" enum in proto
	if assert["mode"] != "ASSERTION_MODE_EXISTS" {
		t.Errorf("expected mode 'ASSERTION_MODE_EXISTS', got %q", assert["mode"])
	}
}

func TestBuildWorkflowFromSteps_Wait(t *testing.T) {
	steps := []*StepSpec{
		{Type: "wait", KVPairs: map[string]string{"durationMs": "1000"}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_WAIT" {
		t.Errorf("expected type 'ACTION_TYPE_WAIT', got %q", action["type"])
	}

	wait := action["wait"].(map[string]any)
	// durationMs is in a oneof, so check the value (protojson uses camelCase)
	if wait["durationMs"] != float64(1000) {
		t.Errorf("expected durationMs 1000, got %v (type %T)", wait["durationMs"], wait["durationMs"])
	}
}

func TestBuildWorkflowFromSteps_Screenshot(t *testing.T) {
	steps := []*StepSpec{
		{Type: "screenshot", KVPairs: map[string]string{"fullPage": "true"}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_SCREENSHOT" {
		t.Errorf("expected type 'ACTION_TYPE_SCREENSHOT', got %q", action["type"])
	}

	screenshot := action["screenshot"].(map[string]any)
	if screenshot["fullPage"] != true {
		t.Errorf("expected fullPage true, got %v", screenshot["fullPage"])
	}
}

func TestBuildWorkflowFromSteps_Evaluate(t *testing.T) {
	steps := []*StepSpec{
		{Type: "evaluate", Positional: "document.title", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_EVALUATE" {
		t.Errorf("expected type 'ACTION_TYPE_EVALUATE', got %q", action["type"])
	}

	// For evaluate, the positional becomes selector, but BuildEvaluateParams expects "expression"
	// The workflow builder sets positional as selector, let's see what we get
	evaluate := action["evaluate"].(map[string]any)
	if evaluate == nil {
		t.Error("expected evaluate params to be populated")
	}
}

func TestBuildWorkflowFromSteps_EvaluateWithExpression(t *testing.T) {
	steps := []*StepSpec{
		{Type: "evaluate", KVPairs: map[string]string{"expression": "document.title"}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	evaluate := action["evaluate"].(map[string]any)
	if evaluate["expression"] != "document.title" {
		t.Errorf("expected expression 'document.title', got %q", evaluate["expression"])
	}
}

func TestBuildWorkflowFromSteps_MultipleSteps(t *testing.T) {
	steps := []*StepSpec{
		{Type: "navigate", KVPairs: map[string]string{"url": "https://example.com"}},
		{Type: "click", Positional: "#submit", KVPairs: map[string]string{}},
		{Type: "screenshot", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	edges := workflow["edges"].([]map[string]any)
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	// Verify edge connections
	if edges[0]["source"] != "step-1" || edges[0]["target"] != "step-2" {
		t.Errorf("edge 0: expected step-1 -> step-2, got %s -> %s", edges[0]["source"], edges[0]["target"])
	}
	if edges[1]["source"] != "step-2" || edges[1]["target"] != "step-3" {
		t.Errorf("edge 1: expected step-2 -> step-3, got %s -> %s", edges[1]["source"], edges[1]["target"])
	}
}

func TestBuildWorkflowFromSteps_EmptySteps(t *testing.T) {
	_, err := BuildWorkflowFromSteps([]*StepSpec{})
	if err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestBuildWorkflowFromSteps_UnsupportedType(t *testing.T) {
	steps := []*StepSpec{
		{Type: "unknown", KVPairs: map[string]string{}},
	}

	_, err := BuildWorkflowFromSteps(steps)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestBuildWorkflowFromSteps_Select(t *testing.T) {
	steps := []*StepSpec{
		{
			Type:       "select",
			Positional: "#dropdown",
			KVPairs:    map[string]string{"value": "option1"},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_SELECT" {
		t.Errorf("expected type 'ACTION_TYPE_SELECT', got %q", action["type"])
	}

	selectOption := action["selectOption"].(map[string]any)
	if selectOption["selector"] != "#dropdown" {
		t.Errorf("expected selector '#dropdown', got %q", selectOption["selector"])
	}
	if selectOption["value"] != "option1" {
		t.Errorf("expected value 'option1', got %q", selectOption["value"])
	}
}

func TestBuildWorkflowFromSteps_Hover(t *testing.T) {
	steps := []*StepSpec{
		{Type: "hover", Positional: "#menu", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_HOVER" {
		t.Errorf("expected type 'ACTION_TYPE_HOVER', got %q", action["type"])
	}

	hover := action["hover"].(map[string]any)
	if hover["selector"] != "#menu" {
		t.Errorf("expected selector '#menu', got %q", hover["selector"])
	}
}

func TestBuildWorkflowFromSteps_Focus(t *testing.T) {
	steps := []*StepSpec{
		{Type: "focus", Positional: "#input", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_FOCUS" {
		t.Errorf("expected type 'ACTION_TYPE_FOCUS', got %q", action["type"])
	}

	focus := action["focus"].(map[string]any)
	if focus["selector"] != "#input" {
		t.Errorf("expected selector '#input', got %q", focus["selector"])
	}
}

func TestBuildWorkflowFromSteps_Blur(t *testing.T) {
	steps := []*StepSpec{
		{Type: "blur", Positional: "#input", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_BLUR" {
		t.Errorf("expected type 'ACTION_TYPE_BLUR', got %q", action["type"])
	}

	blur := action["blur"].(map[string]any)
	if blur["selector"] != "#input" {
		t.Errorf("expected selector '#input', got %q", blur["selector"])
	}
}

func TestBuildWorkflowFromSteps_Keyboard(t *testing.T) {
	steps := []*StepSpec{
		{Type: "keyboard", KVPairs: map[string]string{"key": "Enter"}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_KEYBOARD" {
		t.Errorf("expected type 'ACTION_TYPE_KEYBOARD', got %q", action["type"])
	}

	keyboard := action["keyboard"].(map[string]any)
	if keyboard["key"] != "Enter" {
		t.Errorf("expected key 'Enter', got %q", keyboard["key"])
	}
}

func TestBuildWorkflowFromSteps_WaitForElement(t *testing.T) {
	steps := []*StepSpec{
		{Type: "wait", Positional: "#element", KVPairs: map[string]string{}},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	action := nodes[0]["action"].(map[string]any)

	if action["type"] != "ACTION_TYPE_WAIT" {
		t.Errorf("expected type 'ACTION_TYPE_WAIT', got %q", action["type"])
	}

	wait := action["wait"].(map[string]any)
	// selector is in a oneof
	if wait["selector"] != "#element" {
		t.Errorf("expected selector '#element', got %q", wait["selector"])
	}
}
