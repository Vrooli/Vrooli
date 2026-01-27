package workflows

import (
	"testing"
)

func TestBuildWorkflowFromSteps_Navigate(t *testing.T) {
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
	if node["type"] != "navigate" {
		t.Errorf("expected type 'navigate', got %q", node["type"])
	}

	data, ok := node["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["destinationType"] != "url" {
		t.Errorf("expected destinationType 'url', got %q", data["destinationType"])
	}
	if data["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %q", data["url"])
	}
}

func TestBuildWorkflowFromSteps_ScenarioNavigation(t *testing.T) {
	steps := []*StepSpec{
		{
			Type: "navigate",
			KVPairs: map[string]string{
				"scenario": "knowledge-observatory",
				"path":     "/dashboard",
			},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	data := nodes[0]["data"].(map[string]any)

	if data["destinationType"] != "scenario" {
		t.Errorf("expected destinationType 'scenario', got %q", data["destinationType"])
	}
	if data["scenario"] != "knowledge-observatory" {
		t.Errorf("expected scenario 'knowledge-observatory', got %q", data["scenario"])
	}
	if data["scenarioPath"] != "/dashboard" {
		t.Errorf("expected scenarioPath '/dashboard', got %q", data["scenarioPath"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["selector"] != "#submit" {
		t.Errorf("expected selector '#submit', got %q", data["selector"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["selector"] != "#email" {
		t.Errorf("expected selector '#email', got %q", data["selector"])
	}
	if data["text"] != "user@example.com" {
		t.Errorf("expected text 'user@example.com', got %q", data["text"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["selector"] != "#result" {
		t.Errorf("expected selector '#result', got %q", data["selector"])
	}
	if data["assertMode"] != "exists" {
		t.Errorf("expected assertMode 'exists', got %q", data["assertMode"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["waitType"] != "duration" {
		t.Errorf("expected waitType 'duration', got %q", data["waitType"])
	}
	if data["durationMs"] != 1000 {
		t.Errorf("expected durationMs 1000, got %v", data["durationMs"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["fullPage"] != true {
		t.Errorf("expected fullPage true, got %v", data["fullPage"])
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
	data := nodes[0]["data"].(map[string]any)

	if data["expression"] != "document.title" {
		t.Errorf("expected expression 'document.title', got %q", data["expression"])
	}
}

func TestBuildWorkflowFromSteps_MultipleSteps(t *testing.T) {
	steps := []*StepSpec{
		{Type: "navigate", Positional: "https://example.com", KVPairs: map[string]string{}},
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

func TestBuildWorkflowFromSteps_Resilience(t *testing.T) {
	steps := []*StepSpec{
		{
			Type:       "click",
			Positional: "#btn",
			KVPairs: map[string]string{
				"resilience.maxAttempts": "3",
				"resilience.delayMs":     "1000",
			},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	data := nodes[0]["data"].(map[string]any)

	resilience, ok := data["resilience"].(map[string]any)
	if !ok {
		t.Fatal("expected resilience to be a map")
	}
	if resilience["maxAttempts"] != 3 {
		t.Errorf("expected maxAttempts 3, got %v", resilience["maxAttempts"])
	}
	if resilience["delayMs"] != 1000 {
		t.Errorf("expected delayMs 1000, got %v", resilience["delayMs"])
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

func TestBuildWorkflowFromSteps_NavigateMissingURL(t *testing.T) {
	steps := []*StepSpec{
		{Type: "navigate", KVPairs: map[string]string{}},
	}

	_, err := BuildWorkflowFromSteps(steps)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestBuildWorkflowFromSteps_ClickMissingSelector(t *testing.T) {
	steps := []*StepSpec{
		{Type: "click", KVPairs: map[string]string{}},
	}

	_, err := BuildWorkflowFromSteps(steps)
	if err == nil {
		t.Fatal("expected error for missing selector")
	}
}

func TestBuildWorkflowFromSteps_AssertMissingMode(t *testing.T) {
	steps := []*StepSpec{
		{Type: "assert", Positional: "#result", KVPairs: map[string]string{}},
	}

	_, err := BuildWorkflowFromSteps(steps)
	if err == nil {
		t.Fatal("expected error for missing assertMode")
	}
}

func TestBuildWorkflowFromSteps_WaitMissingType(t *testing.T) {
	steps := []*StepSpec{
		{Type: "wait", KVPairs: map[string]string{}},
	}

	_, err := BuildWorkflowFromSteps(steps)
	if err == nil {
		t.Fatal("expected error for missing wait type")
	}
}

func TestBuildWorkflowFromSteps_Select(t *testing.T) {
	steps := []*StepSpec{
		{
			Type:       "select",
			Positional: "#dropdown",
			KVPairs:    map[string]string{"optionValue": "option1"},
		},
	}

	workflow, err := BuildWorkflowFromSteps(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := workflow["nodes"].([]map[string]any)
	data := nodes[0]["data"].(map[string]any)

	if data["selector"] != "#dropdown" {
		t.Errorf("expected selector '#dropdown', got %q", data["selector"])
	}
	if data["optionValue"] != "option1" {
		t.Errorf("expected optionValue 'option1', got %q", data["optionValue"])
	}
}
