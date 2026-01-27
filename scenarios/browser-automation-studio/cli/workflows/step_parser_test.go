package workflows

import (
	"testing"
)

func TestParseSteps_BasicNavigate(t *testing.T) {
	args := []string{"--step", "navigate", "https://example.com", "--wait"}
	steps, remaining, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Type != "navigate" {
		t.Errorf("expected type 'navigate', got %q", steps[0].Type)
	}
	if steps[0].Positional != "https://example.com" {
		t.Errorf("expected positional 'https://example.com', got %q", steps[0].Positional)
	}
	if len(remaining) != 1 || remaining[0] != "--wait" {
		t.Errorf("expected remaining ['--wait'], got %v", remaining)
	}
}

func TestParseSteps_WithKVPairs(t *testing.T) {
	args := []string{"--step", "navigate", "https://example.com", "waitUntil=networkidle"}
	steps, _, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].KVPairs["waitUntil"] != "networkidle" {
		t.Errorf("expected waitUntil=networkidle, got %q", steps[0].KVPairs["waitUntil"])
	}
}

func TestParseSteps_MultipleSteps(t *testing.T) {
	args := []string{
		"--step", "navigate", "https://example.com",
		"--step", "click", "#submit",
		"--step", "screenshot",
		"--output", "/tmp/results",
	}
	steps, remaining, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[0].Type != "navigate" {
		t.Errorf("step 0: expected type 'navigate', got %q", steps[0].Type)
	}
	if steps[1].Type != "click" {
		t.Errorf("step 1: expected type 'click', got %q", steps[1].Type)
	}
	if steps[1].Positional != "#submit" {
		t.Errorf("step 1: expected positional '#submit', got %q", steps[1].Positional)
	}
	if steps[2].Type != "screenshot" {
		t.Errorf("step 2: expected type 'screenshot', got %q", steps[2].Type)
	}
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining args, got %d: %v", len(remaining), remaining)
	}
}

func TestParseSteps_ScenarioNavigation(t *testing.T) {
	args := []string{"--step", "navigate", "scenario=knowledge-observatory", "path=/dashboard"}
	steps, _, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].KVPairs["scenario"] != "knowledge-observatory" {
		t.Errorf("expected scenario=knowledge-observatory, got %q", steps[0].KVPairs["scenario"])
	}
	if steps[0].KVPairs["path"] != "/dashboard" {
		t.Errorf("expected path=/dashboard, got %q", steps[0].KVPairs["path"])
	}
}

func TestParseSteps_ResilienceSettings(t *testing.T) {
	args := []string{"--step", "click", "#btn", "resilience.maxAttempts=3", "resilience.delayMs=1000"}
	steps, _, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].KVPairs["resilience.maxAttempts"] != "3" {
		t.Errorf("expected resilience.maxAttempts=3, got %q", steps[0].KVPairs["resilience.maxAttempts"])
	}
	if steps[0].KVPairs["resilience.delayMs"] != "1000" {
		t.Errorf("expected resilience.delayMs=1000, got %q", steps[0].KVPairs["resilience.delayMs"])
	}
}

func TestParseSteps_ErrorMissingType(t *testing.T) {
	args := []string{"--step"}
	_, _, err := ParseSteps(args)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestParseSteps_TypeNode(t *testing.T) {
	args := []string{"--step", "type", "#email", "text=user@example.com", "clear=true"}
	steps, _, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Type != "type" {
		t.Errorf("expected type 'type', got %q", steps[0].Type)
	}
	if steps[0].Positional != "#email" {
		t.Errorf("expected positional '#email', got %q", steps[0].Positional)
	}
	if steps[0].KVPairs["text"] != "user@example.com" {
		t.Errorf("expected text=user@example.com, got %q", steps[0].KVPairs["text"])
	}
	if steps[0].KVPairs["clear"] != "true" {
		t.Errorf("expected clear=true, got %q", steps[0].KVPairs["clear"])
	}
}

func TestParseSteps_NoSteps(t *testing.T) {
	args := []string{"--wait", "--output", "/tmp/results"}
	steps, remaining, err := ParseSteps(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
	if len(remaining) != 3 {
		t.Errorf("expected 3 remaining args, got %d: %v", len(remaining), remaining)
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"42", 42},
		{"3.14", 3.14},
		{"true", true},
		{"false", false},
		{"hello", "hello"},
		{"1000", 1000},
		{"0", 0},
	}

	for _, tt := range tests {
		result := parseValue(tt.input)
		if result != tt.expected {
			t.Errorf("parseValue(%q) = %v (%T), expected %v (%T)", tt.input, result, result, tt.expected, tt.expected)
		}
	}
}

func TestSetNestedValue(t *testing.T) {
	data := make(map[string]any)
	setNestedValue(data, "resilience.maxAttempts", 3)

	resilience, ok := data["resilience"].(map[string]any)
	if !ok {
		t.Fatal("expected resilience to be a map")
	}
	if resilience["maxAttempts"] != 3 {
		t.Errorf("expected maxAttempts=3, got %v", resilience["maxAttempts"])
	}
}
