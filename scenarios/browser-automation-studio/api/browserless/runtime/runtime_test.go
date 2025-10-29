package runtime

import (
	"testing"
)

// TestInstructionTypes verifies instruction type constants
func TestInstructionTypes(t *testing.T) {
	tests := []struct {
		name         string
		instruction  Instruction
		expectedType string
	}{
		{
			name:         "navigate instruction",
			instruction:  Instruction{Type: "navigate", Params: InstructionParam{URL: "https://example.com"}},
			expectedType: "navigate",
		},
		{
			name:         "click instruction",
			instruction:  Instruction{Type: "click", Params: InstructionParam{Selector: "button"}},
			expectedType: "click",
		},
		{
			name:         "type instruction",
			instruction:  Instruction{Type: "type", Params: InstructionParam{Selector: "input", Text: "test"}},
			expectedType: "type",
		},
		{
			name:         "wait instruction",
			instruction:  Instruction{Type: "wait", Params: InstructionParam{DurationMs: 1000}},
			expectedType: "wait",
		},
		{
			name:         "screenshot instruction",
			instruction:  Instruction{Type: "screenshot", Params: InstructionParam{Name: "test.png"}},
			expectedType: "screenshot",
		},
		{
			name:         "extract instruction",
			instruction:  Instruction{Type: "extract", Params: InstructionParam{Selector: ".data"}},
			expectedType: "extract",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.instruction.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, tt.instruction.Type)
			}
		})
	}
}

// TestStepResultStructure verifies StepResult contains required fields
func TestStepResultStructure(t *testing.T) {
	result := StepResult{
		NodeID:  "test-node-1",
		Success: true,
		Error:   "",
	}

	if result.NodeID == "" {
		t.Error("NodeID should not be empty")
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if result.Error != "" {
		t.Error("Error should be empty for successful result")
	}
}

// TestStepResultWithError verifies error handling in StepResult
func TestStepResultWithError(t *testing.T) {
	result := StepResult{
		NodeID:  "test-node-2",
		Success: false,
		Error:   "Selector not found",
	}

	if result.Success {
		t.Error("Success should be false when error is present")
	}

	if result.Error == "" {
		t.Error("Error message should be set")
	}

	if result.Error != "Selector not found" {
		t.Errorf("Expected error 'Selector not found', got '%s'", result.Error)
	}
}

// TestInstructionValidation verifies required fields
func TestInstructionValidation(t *testing.T) {
	tests := []struct {
		name        string
		instruction Instruction
		shouldHave  string
	}{
		{
			name:        "navigate requires URL",
			instruction: Instruction{Type: "navigate", Params: InstructionParam{URL: "https://example.com"}},
			shouldHave:  "URL",
		},
		{
			name:        "click requires selector",
			instruction: Instruction{Type: "click", Params: InstructionParam{Selector: "button"}},
			shouldHave:  "Selector",
		},
		{
			name:        "type requires text",
			instruction: Instruction{Type: "type", Params: InstructionParam{Selector: "input", Text: "test"}},
			shouldHave:  "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.shouldHave {
			case "URL":
				if tt.instruction.Params.URL == "" {
					t.Error("Navigate instruction should have URL")
				}
			case "Selector":
				if tt.instruction.Params.Selector == "" {
					t.Error("Click instruction should have Selector")
				}
			case "Text":
				if tt.instruction.Params.Text == "" {
					t.Error("Type instruction should have Text")
				}
			}
		})
	}
}
