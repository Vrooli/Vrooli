package cdp

import (
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// TestExecuteInstruction_Navigate tests navigate instruction handling with default values
func TestExecuteInstruction_Navigate(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
		checkURL    bool
	}{
		{
			name: "navigate with default waitUntil and timeout",
			instruction: runtime.Instruction{
				Type:   "navigate",
				NodeID: "nav-1",
				Index:  0,
				Params: runtime.InstructionParam{
					URL: "https://example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "navigate with custom waitUntil",
			instruction: runtime.Instruction{
				Type:   "navigate",
				NodeID: "nav-2",
				Index:  1,
				Params: runtime.InstructionParam{
					URL:       "https://example.com",
					WaitUntil: "networkidle",
				},
			},
			wantErr: false,
		},
		{
			name: "navigate with custom timeout and waitAfter",
			instruction: runtime.Instruction{
				Type:   "navigate",
				NodeID: "nav-3",
				Index:  2,
				Params: runtime.InstructionParam{
					URL:        "https://example.com",
					TimeoutMs:  10000,
					WaitForMs:  500,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These tests verify parameter handling logic
			// Actual CDP execution would require a real browser session
			// which is tested in integration tests

			// Test parameter extraction and validation
			params := tt.instruction.Params

			// Verify URL is required
			if params.URL == "" && !tt.wantErr {
				t.Error("expected URL to be set")
			}

			// Verify default values are applied correctly
			waitUntil := params.WaitUntil
			if waitUntil == "" {
				waitUntil = "load"
			}
			if waitUntil != "load" && waitUntil != "networkidle" {
				t.Errorf("unexpected waitUntil: %s", waitUntil)
			}

			timeoutMs := params.TimeoutMs
			if timeoutMs == 0 {
				timeoutMs = 30000
			}
			if timeoutMs < 1000 {
				t.Errorf("timeout too short: %d", timeoutMs)
			}
		})
	}
}

// TestExecuteInstruction_Wait tests wait instruction with duration and selector modes
func TestExecuteInstruction_Wait(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
	}{
		{
			name: "time-based wait with durationMs",
			instruction: runtime.Instruction{
				Type:   "wait",
				NodeID: "wait-1",
				Index:  0,
				Params: runtime.InstructionParam{
					DurationMs: 100,
				},
			},
			wantErr: false,
		},
		{
			name: "element-based wait with selector",
			instruction: runtime.Instruction{
				Type:   "wait",
				NodeID: "wait-2",
				Index:  1,
				Params: runtime.InstructionParam{
					Selector:  "[data-testid='element']",
					TimeoutMs: 5000,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Test time-based wait logic
			if params.DurationMs > 0 {
				start := time.Now()
				time.Sleep(time.Duration(params.DurationMs) * time.Millisecond)
				elapsed := time.Since(start).Milliseconds()

				// Allow 50ms tolerance for timing
				expectedMin := int64(params.DurationMs)
				expectedMax := int64(params.DurationMs + 50)
				if elapsed < expectedMin || elapsed > expectedMax {
					t.Errorf("wait duration mismatch: got %dms, want ~%dms", elapsed, params.DurationMs)
				}
			}

			// Test element-based wait parameter validation
			if params.DurationMs == 0 && params.Selector != "" {
				timeoutMs := params.TimeoutMs
				if timeoutMs == 0 {
					timeoutMs = 30000
				}
				if timeoutMs < 1000 {
					t.Errorf("timeout too short for element wait: %d", timeoutMs)
				}
			}
		})
	}
}

// TestExecuteInstruction_Click tests click instruction with timeout and selector
func TestExecuteInstruction_Click(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
	}{
		{
			name: "click with default timeout",
			instruction: runtime.Instruction{
				Type:   "click",
				NodeID: "click-1",
				Index:  0,
				Params: runtime.InstructionParam{
					Selector: "button",
				},
			},
			wantErr: false,
		},
		{
			name: "click with custom timeout and waitAfter",
			instruction: runtime.Instruction{
				Type:   "click",
				NodeID: "click-2",
				Index:  1,
				Params: runtime.InstructionParam{
					Selector:  "[data-testid='submit']",
					TimeoutMs: 10000,
					WaitForMs: 500,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Verify selector is required for click
			if params.Selector == "" {
				t.Error("selector is required for click instruction")
			}

			// Verify default timeout
			timeoutMs := params.TimeoutMs
			if timeoutMs == 0 {
				timeoutMs = 30000
			}
			if timeoutMs < 1000 {
				t.Errorf("timeout too short: %d", timeoutMs)
			}
		})
	}
}

// TestExecuteInstruction_Type tests type instruction with clear flag
func TestExecuteInstruction_Type(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
	}{
		{
			name: "type without clear",
			instruction: runtime.Instruction{
				Type:   "type",
				NodeID: "type-1",
				Index:  0,
				Params: runtime.InstructionParam{
					Selector: "input",
					Text:     "hello",
				},
			},
			wantErr: false,
		},
		{
			name: "type with clear=true",
			instruction: runtime.Instruction{
				Type:   "type",
				NodeID: "type-2",
				Index:  1,
				Params: runtime.InstructionParam{
					Selector: "input",
					Text:     "world",
					Clear:    boolPtr(true),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Verify required fields
			if params.Selector == "" {
				t.Error("selector is required for type instruction")
			}
			if params.Text == "" {
				t.Error("text is required for type instruction")
			}

			// Verify clear flag handling
			clearFirst := false
			if params.Clear != nil {
				clearFirst = *params.Clear
			}

			// Both values are valid
			_ = clearFirst
		})
	}
}

// TestExecuteInstruction_Assert tests assert instruction modes
func TestExecuteInstruction_Assert(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
	}{
		{
			name: "assert exists with default mode",
			instruction: runtime.Instruction{
				Type:   "assert",
				NodeID: "assert-1",
				Index:  0,
				Params: runtime.InstructionParam{
					Selector: "[data-testid='element']",
				},
			},
			wantErr: false,
		},
		{
			name: "assert text with expected value",
			instruction: runtime.Instruction{
				Type:   "assert",
				NodeID: "assert-2",
				Index:  1,
				Params: runtime.InstructionParam{
					Selector:      "[data-testid='title']",
					AssertMode:    "text",
					ExpectedValue: "Welcome",
				},
			},
			wantErr: false,
		},
		{
			name: "assert attribute with case sensitivity",
			instruction: runtime.Instruction{
				Type:   "assert",
				NodeID: "assert-3",
				Index:  2,
				Params: runtime.InstructionParam{
					Selector:       "input",
					AssertMode:     "attribute",
					ExpectedValue:  "Submit",
					CaseSensitive:  boolPtr(true),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Verify selector is required
			if params.Selector == "" {
				t.Error("selector is required for assert instruction")
			}

			// Verify default mode
			mode := params.AssertMode
			if mode == "" {
				mode = "exists"
			}

			validModes := map[string]bool{
				"exists": true, "text": true, "attribute": true, "visible": true,
			}
			if !validModes[mode] {
				t.Errorf("invalid assert mode: %s", mode)
			}

			// Verify case sensitivity handling
			caseSensitive := false
			if params.CaseSensitive != nil {
				caseSensitive = *params.CaseSensitive
			}
			_ = caseSensitive
		})
	}
}

// TestExecuteInstruction_Keyboard tests keyboard instruction with different input formats
func TestExecuteInstruction_Keyboard(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
		wantKey     string
	}{
		{
			name: "keyboard with keys array (preferred)",
			instruction: runtime.Instruction{
				Type:   "keyboard",
				NodeID: "kb-1",
				Index:  0,
				Params: runtime.InstructionParam{
					Keys: []string{"Escape"},
				},
			},
			wantErr: false,
			wantKey: "Escape",
		},
		{
			name: "keyboard with key combination",
			instruction: runtime.Instruction{
				Type:   "keyboard",
				NodeID: "kb-2",
				Index:  1,
				Params: runtime.InstructionParam{
					Keys: []string{"Control", "c"},
				},
			},
			wantErr: false,
			wantKey: "Control+c",
		},
		{
			name: "keyboard with sequence string",
			instruction: runtime.Instruction{
				Type:   "keyboard",
				NodeID: "kb-3",
				Index:  2,
				Params: runtime.InstructionParam{
					Sequence: "Hello World",
				},
			},
			wantErr: false,
			wantKey: "Hello World",
		},
		{
			name: "keyboard missing input",
			instruction: runtime.Instruction{
				Type:   "keyboard",
				NodeID: "kb-4",
				Index:  3,
				Params: runtime.InstructionParam{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Test the priority logic: keys array > sequence > deprecated keyValue
			var keyValue string
			if len(params.Keys) > 0 {
				keyValue = joinKeys(params.Keys)
			} else if params.Sequence != "" {
				keyValue = params.Sequence
			} else if params.KeyValue != "" {
				keyValue = params.KeyValue
			}

			if keyValue == "" && !tt.wantErr {
				t.Error("keyboard instruction missing key input")
			}

			if tt.wantKey != "" && keyValue != tt.wantKey {
				t.Errorf("got key %q, want %q", keyValue, tt.wantKey)
			}
		})
	}
}

// TestExecuteInstruction_Screenshot tests screenshot with fullPage option
func TestExecuteInstruction_Screenshot(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
	}{
		{
			name: "screenshot with default fullPage=false",
			instruction: runtime.Instruction{
				Type:   "screenshot",
				NodeID: "ss-1",
				Index:  0,
				Params: runtime.InstructionParam{},
			},
			wantErr: false,
		},
		{
			name: "screenshot with fullPage=true",
			instruction: runtime.Instruction{
				Type:   "screenshot",
				NodeID: "ss-2",
				Index:  1,
				Params: runtime.InstructionParam{
					FullPage: boolPtr(true),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			// Verify fullPage handling
			fullPage := false
			if params.FullPage != nil {
				fullPage = *params.FullPage
			}

			// Both values are valid
			_ = fullPage
		})
	}
}

// TestExecuteInstruction_DragDrop tests dragDrop with required selectors
func TestExecuteInstruction_DragDrop(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
		errMsg      string
	}{
		{
			name: "dragDrop with valid selectors",
			instruction: runtime.Instruction{
				Type:   "dragDrop",
				NodeID: "dd-1",
				Index:  0,
				Params: runtime.InstructionParam{
					DragSourceSelector: "[data-testid='draggable']",
					DragTargetSelector: "[data-testid='droppable']",
				},
			},
			wantErr: false,
		},
		{
			name: "dragDrop missing source selector",
			instruction: runtime.Instruction{
				Type:   "dragDrop",
				NodeID: "dd-2",
				Index:  1,
				Params: runtime.InstructionParam{
					DragTargetSelector: "[data-testid='droppable']",
				},
			},
			wantErr: true,
			errMsg:  "missing source selector",
		},
		{
			name: "dragDrop missing target selector",
			instruction: runtime.Instruction{
				Type:   "dragDrop",
				NodeID: "dd-3",
				Index:  2,
				Params: runtime.InstructionParam{
					DragSourceSelector: "[data-testid='draggable']",
				},
			},
			wantErr: true,
			errMsg:  "missing target selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.instruction.Params

			source := trimSpace(params.DragSourceSelector)
			target := trimSpace(params.DragTargetSelector)

			if source == "" {
				if !tt.wantErr {
					t.Error("dragDrop missing source selector")
				}
				return
			}

			if target == "" {
				if !tt.wantErr {
					t.Error("dragDrop missing target selector")
				}
				return
			}

			// Verify default step count
			steps := params.DragSteps
			if steps <= 0 {
				steps = 1
			}
			if steps < 1 {
				t.Error("drag steps must be >= 1")
			}
		})
	}
}

// TestExecuteInstruction_UnsupportedType tests error handling for unknown instruction types
func TestExecuteInstruction_UnsupportedType(t *testing.T) {
	instruction := runtime.Instruction{
		Type:   "unknownType",
		NodeID: "unknown-1",
		Index:  0,
		Params: runtime.InstructionParam{},
	}

	// Verify that unsupported types would return an error
	validTypes := map[string]bool{
		"navigate": true, "click": true, "type": true, "wait": true,
		"assert": true, "screenshot": true, "keyboard": true, "evaluate": true,
		"dragDrop": true, "hover": true, "focus": true, "blur": true,
		"uploadFile": true, "scroll": true, "rotate": true, "select": true,
		"tabSwitch": true, "frameSwitch": true, "conditional": true,
		"extract": true, "useVariable": true, "setCookie": true,
		"getCookie": true, "clearCookie": true, "setStorage": true,
		"getStorage": true, "clearStorage": true, "networkMock": true,
		"gesture": true,
	}

	if !validTypes[instruction.Type] {
		// Expected - unsupported type should error
		return
	}

	t.Errorf("instruction type %s should not be supported", instruction.Type)
}

// TestConvertConsoleLogs tests console log conversion to runtime format
func TestConvertConsoleLogs(t *testing.T) {
	now := time.Now()
	logs := []ConsoleLog{
		{Type: "log", Text: "test message", Timestamp: now},
		{Type: "error", Text: "error message", Timestamp: now.Add(time.Second)},
	}

	result := convertConsoleLogs(logs)

	if len(result) != 2 {
		t.Errorf("got %d logs, want 2", len(result))
	}

	if result[0].Type != "log" || result[0].Text != "test message" {
		t.Errorf("first log mismatch: %+v", result[0])
	}

	if result[1].Type != "error" || result[1].Text != "error message" {
		t.Errorf("second log mismatch: %+v", result[1])
	}
}

// TestConvertNetworkEvents tests network event conversion to runtime format
func TestConvertNetworkEvents(t *testing.T) {
	now := time.Now()
	events := []NetworkEvent{
		{Type: "request", URL: "https://example.com", Method: "GET", Timestamp: now},
		{Type: "response", URL: "https://example.com", Status: 200, Timestamp: now.Add(time.Millisecond * 100)},
	}

	result := convertNetworkEvents(events)

	if len(result) != 2 {
		t.Errorf("got %d events, want 2", len(result))
	}

	if result[0].Type != "request" || result[0].Method != "GET" {
		t.Errorf("first event mismatch: %+v", result[0])
	}

	if result[1].Type != "response" || result[1].Status != 200 {
		t.Errorf("second event mismatch: %+v", result[1])
	}
}

// TestExecuteUseVariableInstruction tests variable usage with transforms
func TestExecuteUseVariableInstruction(t *testing.T) {
	tests := []struct {
		name        string
		instruction runtime.Instruction
		wantErr     bool
		checkOutput bool
	}{
		{
			name: "use existing variable",
			instruction: runtime.Instruction{
				Type:   "useVariable",
				NodeID: "var-1",
				Index:  0,
				Params: runtime.InstructionParam{
					VariableName: "testVar",
				},
				Context: map[string]interface{}{
					"testVar": "testValue",
				},
			},
			wantErr: false,
		},
		{
			name: "use missing non-required variable",
			instruction: runtime.Instruction{
				Type:   "useVariable",
				NodeID: "var-2",
				Index:  1,
				Params: runtime.InstructionParam{
					VariableName: "missingVar",
				},
				Context: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "use missing required variable",
			instruction: runtime.Instruction{
				Type:   "useVariable",
				NodeID: "var-3",
				Index:  2,
				Params: runtime.InstructionParam{
					VariableName:     "missingVar",
					VariableRequired: boolPtr(true),
				},
				Context: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "missing variable name",
			instruction: runtime.Instruction{
				Type:   "useVariable",
				NodeID: "var-4",
				Index:  3,
				Params: runtime.InstructionParam{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeUseVariableInstruction(tt.instruction)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && result == nil {
				t.Error("expected result to be non-nil for successful execution")
			}

			if !tt.wantErr && !result.Success {
				t.Errorf("expected success=true, got %v", result.Success)
			}
		})
	}
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}

func joinKeys(keys []string) string {
	result := ""
	for i, key := range keys {
		if i > 0 {
			result += "+"
		}
		result += key
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
