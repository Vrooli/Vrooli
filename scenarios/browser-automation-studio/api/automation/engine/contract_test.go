package engine

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// TestContractCompatibility verifies that Go structs can correctly encode/decode
// JSON payloads that match the TypeScript driver's expected format.
// This catches schema drift between the Go API and TypeScript playwright-driver.
//
// Reference: playwright-driver/src/types/contracts.ts
// Reference: playwright-driver/src/types/session.ts
func TestContractCompatibility(t *testing.T) {
	t.Run("StartSessionRequest", testStartSessionRequestContract)
	t.Run("StartSessionResponse", testStartSessionResponseContract)
	t.Run("RunRequest", testRunRequestContract)
	t.Run("DriverOutcome", testDriverOutcomeContract)
	t.Run("CompiledInstruction", testCompiledInstructionContract)
	t.Run("StepOutcome", testStepOutcomeContract)
}

// testStartSessionRequestContract verifies the Go startSessionRequest struct
// produces JSON matching the TypeScript StartSessionRequest interface.
func testStartSessionRequestContract(t *testing.T) {
	execID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	wfID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	req := startSessionRequest{
		ExecutionID: execID,
		WorkflowID:  wfID,
		Viewport: viewport{
			Width:  1280,
			Height: 720,
		},
		ReuseMode: "fresh",
		BaseURL:   "https://example.com",
		Labels:    map[string]string{"env": "test"},
		RequiredCapabilities: capabilityRequest{
			Tabs:      true,
			Iframes:   true,
			Uploads:   false,
			Downloads: true,
			HAR:       true,
			Video:     false,
			Tracing:   true,
			ViewportW: 1280,
			ViewportH: 720,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal StartSessionRequest: %v", err)
	}

	// Verify required fields are present with correct JSON keys
	jsonStr := string(data)
	requiredFields := []string{
		`"execution_id"`,
		`"workflow_id"`,
		`"viewport"`,
		`"width"`,
		`"height"`,
		`"reuse_mode"`,
		`"required_capabilities"`,
	}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("StartSessionRequest missing required field: %s\nJSON: %s", field, jsonStr)
		}
	}

	// Verify it can round-trip
	var decoded startSessionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StartSessionRequest: %v", err)
	}

	if decoded.ExecutionID != execID {
		t.Errorf("execution_id mismatch: got %v, want %v", decoded.ExecutionID, execID)
	}
	if decoded.Viewport.Width != 1280 {
		t.Errorf("viewport.width mismatch: got %d, want 1280", decoded.Viewport.Width)
	}
}

// testStartSessionResponseContract verifies the TypeScript response format
// can be decoded into Go.
func testStartSessionResponseContract(t *testing.T) {
	// Simulate TypeScript driver response
	tsResponse := `{"session_id": "sess-abc-123"}`

	var resp startSessionResponse
	if err := json.Unmarshal([]byte(tsResponse), &resp); err != nil {
		t.Fatalf("failed to unmarshal StartSessionResponse: %v", err)
	}

	if resp.SessionID != "sess-abc-123" {
		t.Errorf("session_id mismatch: got %q, want %q", resp.SessionID, "sess-abc-123")
	}
}

// testRunRequestContract verifies the instruction run request format.
func testRunRequestContract(t *testing.T) {
	instruction := contracts.CompiledInstruction{
		Index:  0,
		NodeID: "node-1",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: "https://example.com"},
			},
		},
		Context: map[string]any{
			"previousUrl": "about:blank",
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	req := runRequest{Instruction: instruction}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal RunRequest: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{
		`"instruction"`,
		`"index"`,
		`"node_id"`,
		`"action"`,
	}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("RunRequest missing required field: %s\nJSON: %s", field, jsonStr)
		}
	}

	// Note: Standard json.Unmarshal cannot round-trip proto oneof fields.
	// The actual driver uses protojson for proper serialization.
	// We verify the JSON structure above is correct for the driver contract.
}

// testDriverOutcomeContract verifies the TypeScript DriverOutcome format
// can be decoded into Go (with base64 screenshot/DOM fields).
func testDriverOutcomeContract(t *testing.T) {
	// Simulate TypeScript driver response with inline data
	tsResponse := `{
		"schema_version": "automation-step-outcome-v1",
		"payload_version": "1",
		"step_index": 0,
		"attempt": 1,
		"node_id": "node-1",
		"step_type": "navigate",
		"success": true,
		"started_at": "2025-01-15T10:00:00Z",
		"completed_at": "2025-01-15T10:00:05Z",
		"duration_ms": 5000,
		"final_url": "https://example.com/",
		"screenshot_base64": "iVBORw0KGgo=",
		"screenshot_media_type": "image/png",
		"screenshot_width": 1280,
		"screenshot_height": 720,
		"dom_html": "<html><body>Test</body></html>",
		"dom_preview": "Test page content"
	}`

	var outcome driverOutcome
	if err := json.Unmarshal([]byte(tsResponse), &outcome); err != nil {
		t.Fatalf("failed to unmarshal DriverOutcome: %v", err)
	}

	// Verify all fields decoded correctly
	if !outcome.Success {
		t.Error("success should be true")
	}
	if outcome.StepIndex != 0 {
		t.Errorf("step_index mismatch: got %d, want 0", outcome.StepIndex)
	}
	if outcome.NodeID != "node-1" {
		t.Errorf("node_id mismatch: got %q, want %q", outcome.NodeID, "node-1")
	}
	if outcome.ScreenshotBase64 != "iVBORw0KGgo=" {
		t.Errorf("screenshot_base64 mismatch: got %q", outcome.ScreenshotBase64)
	}
	if outcome.ScreenshotMediaType != "image/png" {
		t.Errorf("screenshot_media_type mismatch: got %q", outcome.ScreenshotMediaType)
	}
	if outcome.DOMHTML != "<html><body>Test</body></html>" {
		t.Errorf("dom_html mismatch: got %q", outcome.DOMHTML)
	}
}

// testCompiledInstructionContract verifies field name mapping between
// Go and TypeScript for CompiledInstruction.
func testCompiledInstructionContract(t *testing.T) {
	// Test all instruction types used by the driver
	testCases := []struct {
		name   string
		instr  contracts.CompiledInstruction
		checks []string // JSON keys that must be present
	}{
		{
			name: "navigate",
			instr: contracts.CompiledInstruction{
				Index:  0,
				NodeID: "nav-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{
						Navigate: &basactions.NavigateParams{Url: "https://example.com"},
					},
				},
			},
			checks: []string{`"index":0`, `"node_id":"nav-1"`, `"action"`},
		},
		{
			name: "click",
			instr: contracts.CompiledInstruction{
				Index:  1,
				NodeID: "click-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{Selector: "button.submit"},
					},
				},
			},
			checks: []string{`"index":1`, `"action"`},
		},
		{
			name: "input",
			instr: contracts.CompiledInstruction{
				Index:  2,
				NodeID: "type-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_INPUT,
					Params: &basactions.ActionDefinition_Input{
						Input: &basactions.InputParams{Selector: "#email", Value: "test@example.com"},
					},
				},
			},
			checks: []string{`"action"`},
		},
		{
			name: "wait",
			instr: contracts.CompiledInstruction{
				Index:  3,
				NodeID: "wait-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{
						Wait: &basactions.WaitParams{WaitFor: &basactions.WaitParams_Selector{Selector: ".loaded"}},
					},
				},
			},
			checks: []string{`"action"`},
		},
		{
			name: "assert",
			instr: contracts.CompiledInstruction{
				Index:  4,
				NodeID: "assert-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_ASSERT,
					Params: &basactions.ActionDefinition_Assert{
						Assert: &basactions.AssertParams{Selector: "h1"},
					},
				},
			},
			checks: []string{`"action"`},
		},
		{
			name: "extract",
			instr: contracts.CompiledInstruction{
				Index:  5,
				NodeID: "extract-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_EXTRACT,
					Params: &basactions.ActionDefinition_Extract{
						Extract: &basactions.ExtractParams{Selector: ".price"},
					},
				},
			},
			checks: []string{`"action"`},
		},
		{
			name: "with_preload_html",
			instr: contracts.CompiledInstruction{
				Index:  6,
				NodeID: "preload-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{
						Navigate: &basactions.NavigateParams{Url: "https://example.com"},
					},
				},
				PreloadHTML: "<html><body>Preloaded</body></html>",
			},
			checks: []string{`"preload_html"`},
		},
		{
			name: "with_context",
			instr: contracts.CompiledInstruction{
				Index:  7,
				NodeID: "ctx-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{Selector: "button"},
					},
				},
				Context: map[string]any{"loopIndex": float64(0), "loopItem": "item1"},
			},
			checks: []string{`"context"`, `"loopIndex"`},
		},
		{
			name: "with_metadata",
			instr: contracts.CompiledInstruction{
				Index:  8,
				NodeID: "meta-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT,
					Params: &basactions.ActionDefinition_Screenshot{
						Screenshot: &basactions.ScreenshotParams{},
					},
				},
				Metadata: map[string]string{"source": "test", "label": "important"},
			},
			checks: []string{`"metadata"`, `"source"`, `"label"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.instr)
			if err != nil {
				t.Fatalf("failed to marshal instruction: %v", err)
			}

			jsonStr := string(data)
			for _, check := range tc.checks {
				if !strings.Contains(jsonStr, check) {
					t.Errorf("instruction missing expected content: %s\nJSON: %s", check, jsonStr)
				}
			}
		})
	}
}

// testStepOutcomeContract verifies the full StepOutcome structure including
// nested types like Screenshot, DOMSnapshot, ConsoleLogEntry, etc.
func testStepOutcomeContract(t *testing.T) {
	now := time.Now().UTC()
	execID := uuid.New()
	highlightRGBA := "#ff0000"

	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		CorrelationID:  "corr-123",
		StepIndex:      0,
		Attempt:        1,
		NodeID:         "node-1",
		StepType:       "navigate",
		Instruction:    "navigate to example.com",
		Success:        true,
		StartedAt:      now,
		CompletedAt:    &now,
		DurationMs:     5000,
		FinalURL:       "https://example.com/",
		ConsoleLogs: []contracts.ConsoleLogEntry{
			{Type: "log", Text: "Page loaded", Timestamp: now},
			{Type: "warn", Text: "Deprecated API", Timestamp: now, Location: "app.js:42"},
		},
		Network: []contracts.NetworkEvent{
			{
				Type:      "request",
				URL:       "https://example.com/",
				Method:    "GET",
				Timestamp: now,
			},
			{
				Type:      "response",
				URL:       "https://example.com/",
				Status:    200,
				OK:        true,
				Timestamp: now,
			},
		},
		ExtractedData: map[string]any{
			"title": "Example Domain",
		},
		Assertion: &contracts.AssertionOutcome{
			Mode:     "text",
			Selector: "h1",
			Expected: "Example Domain",
			Actual:   "Example Domain",
			Success:  true,
		},
		Condition: &contracts.ConditionOutcome{
			Type:     "element_visible",
			Outcome:  true,
			Selector: ".content",
		},
		ElementBoundingBox: &contracts.BoundingBox{X: 100, Y: 50, Width: 200, Height: 40},
		ClickPosition:      &contracts.Point{X: 200, Y: 70},
		FocusedElement: &contracts.ElementFocus{
			Selector:    "h1",
			BoundingBox: &contracts.BoundingBox{X: 100, Y: 50, Width: 200, Height: 40},
			},
			HighlightRegions: []*contracts.HighlightRegion{
				{Selector: "h1", Padding: 4, CustomRgba: &highlightRGBA},
			},
			MaskRegions: []*contracts.MaskRegion{
				{Selector: ".sidebar", Opacity: 0.5},
			},
			ZoomFactor: 1.5,
			CursorTrail: []contracts.CursorPosition{
				{Point: &contracts.Point{X: 0, Y: 0}, RecordedAt: now, ElapsedMs: 0},
				{Point: &contracts.Point{X: 200, Y: 70}, RecordedAt: now, ElapsedMs: 500},
			},
			Notes: map[string]string{"dedupe": "abc123"},
		}

	data, err := json.Marshal(outcome)
	if err != nil {
		t.Fatalf("failed to marshal StepOutcome: %v", err)
	}

	// Verify all major fields are present
	jsonStr := string(data)
	expectedFields := []string{
		`"schema_version"`,
		`"payload_version"`,
		`"execution_id"`,
		`"correlation_id"`,
		`"step_index"`,
		`"attempt"`,
		`"node_id"`,
		`"step_type"`,
		`"success"`,
		`"started_at"`,
		`"completed_at"`,
		`"duration_ms"`,
		`"final_url"`,
		`"console_logs"`,
		`"network"`,
		`"extracted_data"`,
		`"assertion"`,
		`"condition"`,
		`"element_bounding_box"`,
		`"click_position"`,
		`"focused_element"`,
		`"highlight_regions"`,
		`"mask_regions"`,
		`"zoom_factor"`,
		`"cursor_trail"`,
		`"notes"`,
	}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("StepOutcome missing expected field: %s", field)
		}
	}

	// Verify round-trip
	var decoded contracts.StepOutcome
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StepOutcome: %v", err)
	}

	if decoded.ExecutionID != execID {
		t.Errorf("execution_id mismatch after round-trip")
	}
	if decoded.StepIndex != 0 {
		t.Errorf("step_index mismatch after round-trip: got %d, want 0", decoded.StepIndex)
	}
	if len(decoded.ConsoleLogs) != 2 {
		t.Errorf("console_logs count mismatch: got %d, want 2", len(decoded.ConsoleLogs))
	}
}

// TestStepFailureContract verifies StepFailure encoding matches TypeScript.
func TestStepFailureContract(t *testing.T) {
	now := time.Now().UTC()

	failure := contracts.StepFailure{
		Kind:       contracts.FailureKindTimeout,
		Code:       "ELEMENT_NOT_FOUND",
		Message:    "Selector .missing not found within 30000ms",
		Fatal:      false,
		Retryable:  true,
		OccurredAt: &now,
		Details: map[string]any{
			"selector":  ".missing",
			"timeoutMs": float64(30000),
		},
		Source: contracts.FailureSourceEngine,
	}

	data, err := json.Marshal(failure)
	if err != nil {
		t.Fatalf("failed to marshal StepFailure: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{
		`"kind":"timeout"`,
		`"code":"ELEMENT_NOT_FOUND"`,
		`"message"`,
		`"retryable":true`,
		`"occurred_at"`,
		`"details"`,
		`"source":"engine"`,
	}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("StepFailure missing expected content: %s\nJSON: %s", field, jsonStr)
		}
	}

	// Note: fatal:false is omitted due to Go's omitempty - this is expected behavior
	// TypeScript should treat missing 'fatal' as false
}

// TestHealthResponseContract verifies the health endpoint response format.
func TestHealthResponseContract(t *testing.T) {
	// Simulate TypeScript driver health response
	tsResponse := `{
		"status": "ok",
		"timestamp": "2025-01-15T10:00:00Z",
		"sessions": 2,
		"version": "1.0.0",
		"browser": {
			"healthy": true,
			"version": "120.0.6099.71"
		}
	}`

	var resp map[string]any
	if err := json.Unmarshal([]byte(tsResponse), &resp); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	// Verify expected fields
	if resp["status"] != "ok" {
		t.Errorf("status mismatch: got %v", resp["status"])
	}
	if resp["sessions"] != float64(2) {
		t.Errorf("sessions mismatch: got %v", resp["sessions"])
	}

	browser, ok := resp["browser"].(map[string]any)
	if !ok {
		t.Fatal("browser field missing or wrong type")
	}
	if browser["healthy"] != true {
		t.Errorf("browser.healthy mismatch: got %v", browser["healthy"])
	}
}

// TestDegradedHealthResponseContract verifies degraded health responses.
func TestDegradedHealthResponseContract(t *testing.T) {
	tsResponse := `{
		"status": "error",
		"timestamp": "2025-01-15T10:00:00Z",
		"sessions": 0,
		"browser": {
			"healthy": false,
			"error": "Failed to launch browser: Chromium executable not found"
		}
	}`

	var resp map[string]any
	if err := json.Unmarshal([]byte(tsResponse), &resp); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	if resp["status"] != "error" {
		t.Errorf("status should be error: got %v", resp["status"])
	}

	browser, ok := resp["browser"].(map[string]any)
	if !ok {
		t.Fatal("browser field missing")
	}
	if browser["healthy"] != false {
		t.Error("browser.healthy should be false")
	}
	if browser["error"] == nil {
		t.Error("browser.error should be present")
	}
}

// TestFailureKindValues verifies FailureKind enum values match TypeScript.
func TestFailureKindValues(t *testing.T) {
	// These values must match playwright-driver/src/types/contracts.ts FailureKind
	expectedKinds := map[contracts.FailureKind]string{
		contracts.FailureKindEngine:        "engine",
		contracts.FailureKindInfra:         "infra",
		contracts.FailureKindOrchestration: "orchestration",
		contracts.FailureKindUser:          "user",
		contracts.FailureKindTimeout:       "timeout",
		contracts.FailureKindCancelled:     "cancelled",
	}

	for kind, expected := range expectedKinds {
		if string(kind) != expected {
			t.Errorf("FailureKind %q should serialize as %q, got %q", kind, expected, string(kind))
		}
	}
}

// TestSchemaVersionConstants verifies version constants match TypeScript.
func TestSchemaVersionConstants(t *testing.T) {
	// These must match playwright-driver/src/types/contracts.ts
	if contracts.StepOutcomeSchemaVersion != "automation-step-outcome-v1" {
		t.Errorf("StepOutcomeSchemaVersion mismatch: got %q", contracts.StepOutcomeSchemaVersion)
	}
	if contracts.PayloadVersion != "1" {
		t.Errorf("PayloadVersion mismatch: got %q", contracts.PayloadVersion)
	}
}

// TestFullTypeScriptDriverResponseRoundtrip tests decoding a comprehensive
// TypeScript driver response that includes all possible fields.
// This is the primary test for detecting Go↔TypeScript schema drift.
func TestFullTypeScriptDriverResponseRoundtrip(t *testing.T) {
	// This JSON represents a full response from the TypeScript driver
	// with ALL optional fields populated. If any field names change in
	// either Go or TypeScript, this test will fail.
	tsFullResponse := `{
		"schema_version": "automation-step-outcome-v1",
		"payload_version": "1",
		"execution_id": "11111111-1111-1111-1111-111111111111",
		"correlation_id": "corr-abc-123",
		"step_index": 5,
		"attempt": 2,
		"node_id": "extract-price-node",
		"step_type": "extract",
		"instruction": "extract price from .product-price",
		"success": true,
		"started_at": "2025-01-15T10:00:00.000Z",
		"completed_at": "2025-01-15T10:00:02.500Z",
		"duration_ms": 2500,
		"final_url": "https://shop.example.com/product/123",
		"screenshot_base64": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		"screenshot_media_type": "image/png",
		"screenshot_width": 1920,
		"screenshot_height": 1080,
		"dom_html": "<html><body><div class=\"product-price\">$99.99</div></body></html>",
		"dom_preview": "<div class=\"product-price\">$99.99</div>",
		"console_logs": [
			{"type": "log", "text": "Price loaded", "timestamp": "2025-01-15T10:00:01.000Z"},
			{"type": "warn", "text": "Deprecated API", "timestamp": "2025-01-15T10:00:01.500Z", "location": "bundle.js:1234"}
		],
		"network": [
			{"type": "request", "url": "https://api.example.com/price", "method": "GET", "timestamp": "2025-01-15T10:00:00.500Z"},
			{"type": "response", "url": "https://api.example.com/price", "status": 200, "ok": true, "timestamp": "2025-01-15T10:00:01.000Z", "size": 256}
		],
		"extracted_data": {
			"price": "$99.99",
			"currency": "USD",
			"amount": 99.99
		},
		"assertion": {
			"mode": "contains",
			"selector": ".product-price",
			"expected": "$",
			"actual": "$99.99",
			"success": true
		},
		"condition": {
			"type": "element_visible",
			"outcome": true,
			"selector": ".product-price"
		},
		"element_bounding_box": {"x": 100, "y": 200, "width": 150, "height": 30},
		"click_position": {"x": 175, "y": 215},
		"focused_element": {
			"selector": ".product-price",
			"bounding_box": {"x": 100, "y": 200, "width": 150, "height": 30}
		},
		"highlight_regions": [
			{"selector": ".product-price", "padding": 4, "color": "#ff0000"}
		],
		"mask_regions": [
			{"selector": ".sensitive-data", "opacity": 1}
		],
		"zoom_factor": 1.25,
		"cursor_trail": [
			{"point": {"x": 0, "y": 0}, "recorded_at": "2025-01-15T10:00:00.000Z", "elapsed_ms": 0},
			{"point": {"x": 175, "y": 215}, "recorded_at": "2025-01-15T10:00:02.000Z", "elapsed_ms": 2000}
		],
		"notes": {
			"dedupe_key": "price-extract-123",
			"custom_data": "test"
		}
	}`

	// Step 1: Decode TypeScript response into Go driverOutcome struct
	var outcome driverOutcome
	if err := json.Unmarshal([]byte(tsFullResponse), &outcome); err != nil {
		t.Fatalf("failed to decode TypeScript driver response: %v", err)
	}

	// Step 2: Verify critical fields decoded correctly
	if outcome.SchemaVersion != "automation-step-outcome-v1" {
		t.Errorf("schema_version: got %q, want %q", outcome.SchemaVersion, "automation-step-outcome-v1")
	}
	if outcome.ExecutionID.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("execution_id: got %q, want 11111111-...", outcome.ExecutionID.String())
	}
	if outcome.StepIndex != 5 {
		t.Errorf("step_index: got %d, want 5", outcome.StepIndex)
	}
	if outcome.Attempt != 2 {
		t.Errorf("attempt: got %d, want 2", outcome.Attempt)
	}
	if outcome.NodeID != "extract-price-node" {
		t.Errorf("node_id: got %q", outcome.NodeID)
	}
	if outcome.StepType != "extract" {
		t.Errorf("step_type: got %q", outcome.StepType)
	}
	if !outcome.Success {
		t.Error("success should be true")
	}
	if outcome.DurationMs != 2500 {
		t.Errorf("duration_ms: got %d, want 2500", outcome.DurationMs)
	}
	if outcome.FinalURL != "https://shop.example.com/product/123" {
		t.Errorf("final_url: got %q", outcome.FinalURL)
	}

	// Step 3: Verify inline screenshot data
	if outcome.ScreenshotBase64 == "" {
		t.Error("screenshot_base64 should be populated")
	}
	if outcome.ScreenshotMediaType != "image/png" {
		t.Errorf("screenshot_media_type: got %q", outcome.ScreenshotMediaType)
	}
	if outcome.ScreenshotWidth != 1920 {
		t.Errorf("screenshot_width: got %d, want 1920", outcome.ScreenshotWidth)
	}
	if outcome.ScreenshotHeight != 1080 {
		t.Errorf("screenshot_height: got %d, want 1080", outcome.ScreenshotHeight)
	}

	// Step 4: Verify DOM snapshot
	if outcome.DOMHTML == "" {
		t.Error("dom_html should be populated")
	}
	if outcome.DOMPreview == "" {
		t.Error("dom_preview should be populated")
	}

	// Step 5: Verify extracted data (complex nested structure)
	if outcome.ExtractedData == nil {
		t.Fatal("extracted_data should be populated")
	}
	if price, ok := outcome.ExtractedData["price"]; !ok || price != "$99.99" {
		t.Errorf("extracted_data.price: got %v", price)
	}
	if amount, ok := outcome.ExtractedData["amount"].(float64); !ok || amount != 99.99 {
		t.Errorf("extracted_data.amount: got %v (type %T)", outcome.ExtractedData["amount"], outcome.ExtractedData["amount"])
	}

	// Step 6: Verify console logs array
	if len(outcome.ConsoleLogs) != 2 {
		t.Errorf("console_logs count: got %d, want 2", len(outcome.ConsoleLogs))
	} else {
		if outcome.ConsoleLogs[0].Type != "log" {
			t.Errorf("console_logs[0].type: got %q", outcome.ConsoleLogs[0].Type)
		}
		if outcome.ConsoleLogs[1].Location != "bundle.js:1234" {
			t.Errorf("console_logs[1].location: got %q", outcome.ConsoleLogs[1].Location)
		}
	}

	// Step 7: Verify network events array
	if len(outcome.Network) != 2 {
		t.Errorf("network count: got %d, want 2", len(outcome.Network))
	} else {
		if outcome.Network[0].Type != "request" || outcome.Network[0].Method != "GET" {
			t.Errorf("network[0] request: type=%q method=%q", outcome.Network[0].Type, outcome.Network[0].Method)
		}
		if outcome.Network[1].Type != "response" || outcome.Network[1].Status != 200 {
			t.Errorf("network[1] response: type=%q status=%d", outcome.Network[1].Type, outcome.Network[1].Status)
		}
	}

	// Step 8: Verify assertion outcome
	if outcome.Assertion == nil {
		t.Error("assertion should be populated")
	} else {
		if outcome.Assertion.Mode != "contains" {
			t.Errorf("assertion.mode: got %q", outcome.Assertion.Mode)
		}
		if !outcome.Assertion.Success {
			t.Error("assertion.success should be true")
		}
	}

	// Step 9: Verify condition outcome
	if outcome.Condition == nil {
		t.Error("condition should be populated")
	} else {
		if outcome.Condition.Type != "element_visible" {
			t.Errorf("condition.type: got %q", outcome.Condition.Type)
		}
	}

	// Step 10: Verify bounding box and click position
	if outcome.ElementBoundingBox == nil {
		t.Error("element_bounding_box should be populated")
	} else if outcome.ElementBoundingBox.Width != 150 {
		t.Errorf("element_bounding_box.width: got %v, want 150", outcome.ElementBoundingBox.Width)
	}
	if outcome.ClickPosition == nil {
		t.Error("click_position should be populated")
	} else if outcome.ClickPosition.X != 175 {
		t.Errorf("click_position.x: got %v, want 175", outcome.ClickPosition.X)
	}

	// Step 11: Verify focused element
	if outcome.FocusedElement == nil {
		t.Error("focused_element should be populated")
	} else if outcome.FocusedElement.Selector != ".product-price" {
		t.Errorf("focused_element.selector: got %q", outcome.FocusedElement.Selector)
	}

	// Step 12: Verify highlight and mask regions
	if len(outcome.HighlightRegions) != 1 {
		t.Errorf("highlight_regions count: got %d, want 1", len(outcome.HighlightRegions))
	}
	if len(outcome.MaskRegions) != 1 {
		t.Errorf("mask_regions count: got %d, want 1", len(outcome.MaskRegions))
	}

	// Step 13: Verify zoom factor
	if outcome.ZoomFactor != 1.25 {
		t.Errorf("zoom_factor: got %f, want 1.25", outcome.ZoomFactor)
	}

	// Step 14: Verify cursor trail
	if len(outcome.CursorTrail) != 2 {
		t.Errorf("cursor_trail count: got %d, want 2", len(outcome.CursorTrail))
	} else if outcome.CursorTrail[1].ElapsedMs != 2000 {
		t.Errorf("cursor_trail[1].elapsed_ms: got %d, want 2000", outcome.CursorTrail[1].ElapsedMs)
	}

	// Step 15: Verify notes
	if outcome.Notes == nil || outcome.Notes["dedupe_key"] != "price-extract-123" {
		t.Errorf("notes.dedupe_key: got %v", outcome.Notes)
	}

	// Step 16: Re-encode to JSON and verify structure preserved
	reencoded, err := json.Marshal(outcome)
	if err != nil {
		t.Fatalf("failed to re-encode outcome: %v", err)
	}

	// Decode again and verify key fields
	var roundtrip driverOutcome
	if err := json.Unmarshal(reencoded, &roundtrip); err != nil {
		t.Fatalf("failed to decode roundtrip: %v", err)
	}
	if roundtrip.StepIndex != 5 {
		t.Errorf("roundtrip step_index: got %d, want 5", roundtrip.StepIndex)
	}
	if roundtrip.FinalURL != "https://shop.example.com/product/123" {
		t.Errorf("roundtrip final_url changed: got %q", roundtrip.FinalURL)
	}
}

// TestTypeScriptErrorResponseContract verifies error responses from the driver.
func TestTypeScriptErrorResponseContract(t *testing.T) {
	// TypeScript driver error response format
	tsErrorResponse := `{
		"schema_version": "automation-step-outcome-v1",
		"payload_version": "1",
		"step_index": 3,
		"attempt": 1,
		"node_id": "click-submit",
		"step_type": "click",
		"success": false,
		"started_at": "2025-01-15T10:00:00.000Z",
		"completed_at": "2025-01-15T10:00:30.000Z",
		"duration_ms": 30000,
		"failure": {
			"kind": "timeout",
			"code": "SELECTOR_NOT_FOUND",
			"message": "Selector button.submit not found within 30000ms",
			"retryable": true,
			"occurred_at": "2025-01-15T10:00:30.000Z",
			"details": {
				"selector": "button.submit",
				"timeout_ms": 30000,
				"state": "visible"
			},
			"source": "engine"
		}
	}`

	var outcome driverOutcome
	if err := json.Unmarshal([]byte(tsErrorResponse), &outcome); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if outcome.Success {
		t.Error("success should be false for error response")
	}
	if outcome.Failure == nil {
		t.Fatal("failure should be populated")
	}
	if outcome.Failure.Kind != contracts.FailureKindTimeout {
		t.Errorf("failure.kind: got %q, want timeout", outcome.Failure.Kind)
	}
	if outcome.Failure.Code != "SELECTOR_NOT_FOUND" {
		t.Errorf("failure.code: got %q", outcome.Failure.Code)
	}
	if !outcome.Failure.Retryable {
		t.Error("failure.retryable should be true")
	}
	if outcome.Failure.Source != contracts.FailureSourceEngine {
		t.Errorf("failure.source: got %q, want engine", outcome.Failure.Source)
	}
	if outcome.Failure.Details == nil {
		t.Error("failure.details should be populated")
	}
}
