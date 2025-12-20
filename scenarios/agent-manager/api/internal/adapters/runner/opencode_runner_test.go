// Package runner provides runner adapter implementations.
//
// This file contains tests for the OpenCode runner adapter.
// JSON samples captured from actual OpenCode CLI output (2025-12-20).
package runner

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Real OpenCode JSON samples captured from CLI output
var opencodeSamples = map[string]string{
	// step_start event - marks the beginning of a new step
	"step_start": `{"type":"step_start","timestamp":1766204176981,"sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","part":{"id":"prt_b39f8fa54001v58yTLO90exewJ","sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","messageID":"msg_b39f8f728001cd9lI4FmZaJE3d","type":"step-start"}}`,

	// tool_use event with write tool - actual OpenCode format
	"tool_use_write": `{"type":"tool_use","timestamp":1766204178887,"sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","part":{"id":"prt_b39f901c2001jKS5vAPiKOlCUx","sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","messageID":"msg_b39f8f728001cd9lI4FmZaJE3d","type":"tool","callID":"call_01361367","tool":"write","state":{"status":"completed","input":{"content":"hello world","filePath":"/tmp/opencode-debug/debug.txt"},"output":"","title":"tmp/opencode-debug/debug.txt","metadata":{"diagnostics":{},"filepath":"/tmp/opencode-debug/debug.txt","exists":false},"time":{"start":1766204178885,"end":1766204178886}}}}`,

	// tool_use event with bash tool
	"tool_use_bash": `{"type":"tool_use","timestamp":1766204179000,"sessionID":"ses_test","part":{"id":"prt_test","sessionID":"ses_test","messageID":"msg_test","type":"tool","callID":"call_bash123","tool":"bash","state":{"status":"completed","input":{"command":"ls -la"},"output":"total 0","time":{"start":1766204179000,"end":1766204179001}}}}`,

	// step_finish event with cost and token data
	"step_finish_with_cost": `{"type":"step_finish","timestamp":1766204178909,"sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","part":{"id":"prt_b39f901dc001t7d8JGuM8jJnyv","sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","messageID":"msg_b39f8f728001cd9lI4FmZaJE3d","type":"step-finish","reason":"tool-calls","cost":0.00197528,"tokens":{"input":3562,"output":401,"reasoning":359,"cache":{"read":6144,"write":0}}}}`,

	// step_finish with stop reason
	"step_finish_stop": `{"type":"step_finish","timestamp":1766204180782,"sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","part":{"id":"prt_b39f9092d001vwFFLTOKpANxLU","sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","messageID":"msg_b39f901df001NyR5rtDPoz6axQ","type":"step-finish","reason":"stop","cost":0.00157864,"tokens":{"input":3486,"output":254,"reasoning":250,"cache":{"read":6272,"write":0}}}}`,

	// text event with assistant message
	"text": `{"type":"text","timestamp":1766204180782,"sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","part":{"id":"prt_b39f9090b0011AtJjxm1PHf6DA","sessionID":"ses_4c60708edffePbrrcxPKCtw2F7","messageID":"msg_b39f901df001NyR5rtDPoz6axQ","type":"text","text":"File created successfully.","time":{"start":1766204180781,"end":1766204180781}}}`,

	// Error event
	"error": `{"type":"error","timestamp":1766204200000,"sessionID":"ses_test","error":{"code":"rate_limit","message":"Rate limit exceeded"}}`,

	// Non-JSON line (should be skipped)
	"non_json": `Starting OpenCode session...`,

	// Empty line (should be skipped)
	"empty": ``,
}

func TestOpenCodeRunner_ParseStreamEvent_StepStart(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["step_start"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("expected EventTypeLog, got %v", event.EventType)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Level != "info" {
		t.Errorf("expected level 'info', got '%s'", logData.Level)
	}
	if logData.Message != "OpenCode step started" {
		t.Errorf("expected message 'OpenCode step started', got '%s'", logData.Message)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_ToolUseWrite(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["tool_use_write"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("expected EventTypeToolCall, got %v", event.EventType)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}

	// Verify tool name is "write" (not "tool" or empty)
	if toolData.ToolName != "write" {
		t.Errorf("expected toolName 'write', got '%s'", toolData.ToolName)
	}

	// Verify input contains the expected data
	if toolData.Input == nil {
		t.Fatal("expected input, got nil")
	}
	if content, ok := toolData.Input["content"].(string); !ok || content != "hello world" {
		t.Errorf("expected input.content 'hello world', got '%v'", toolData.Input["content"])
	}
	if filePath, ok := toolData.Input["filePath"].(string); !ok || filePath != "/tmp/opencode-debug/debug.txt" {
		t.Errorf("expected input.filePath '/tmp/opencode-debug/debug.txt', got '%v'", toolData.Input["filePath"])
	}
}

func TestOpenCodeRunner_ParseStreamEvent_ToolUseBash(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["tool_use_bash"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("expected EventTypeToolCall, got %v", event.EventType)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}

	// Verify tool name is "bash"
	if toolData.ToolName != "bash" {
		t.Errorf("expected toolName 'bash', got '%s'", toolData.ToolName)
	}

	// Verify input contains command
	if toolData.Input == nil {
		t.Fatal("expected input, got nil")
	}
	if cmd, ok := toolData.Input["command"].(string); !ok || cmd != "ls -la" {
		t.Errorf("expected input.command 'ls -la', got '%v'", toolData.Input["command"])
	}
}

func TestOpenCodeRunner_ParseStreamEvent_Text(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["text"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeMessage {
		t.Errorf("expected EventTypeMessage, got %v", event.EventType)
	}

	msgData, ok := event.Data.(*domain.MessageEventData)
	if !ok {
		t.Fatalf("expected MessageEventData, got %T", event.Data)
	}
	if msgData.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", msgData.Role)
	}
	if msgData.Content != "File created successfully." {
		t.Errorf("expected content 'File created successfully.', got '%s'", msgData.Content)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_StepFinishWithCost(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["step_finish_with_cost"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeMetric {
		t.Errorf("expected EventTypeMetric, got %v", event.EventType)
	}

	costData, ok := event.Data.(*domain.CostEventData)
	if !ok {
		t.Fatalf("expected CostEventData, got %T", event.Data)
	}

	// Verify token counts
	if costData.InputTokens != 3562 {
		t.Errorf("expected inputTokens 3562, got %d", costData.InputTokens)
	}
	if costData.OutputTokens != 401 {
		t.Errorf("expected outputTokens 401, got %d", costData.OutputTokens)
	}

	// Verify cache tokens
	if costData.CacheReadTokens != 6144 {
		t.Errorf("expected cacheReadTokens 6144, got %d", costData.CacheReadTokens)
	}
	if costData.CacheCreationTokens != 0 {
		t.Errorf("expected cacheCreationTokens 0, got %d", costData.CacheCreationTokens)
	}

	// Verify cost
	if costData.TotalCostUSD < 0.00197 || costData.TotalCostUSD > 0.00198 {
		t.Errorf("expected totalCostUSD ~0.00197528, got %f", costData.TotalCostUSD)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_Error(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, opencodeSamples["error"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeError {
		t.Errorf("expected EventTypeError, got %v", event.EventType)
	}

	errData, ok := event.Data.(*domain.ErrorEventData)
	if !ok {
		t.Fatalf("expected ErrorEventData, got %T", event.Data)
	}
	if errData.Code != "rate_limit" {
		t.Errorf("expected code 'rate_limit', got '%s'", errData.Code)
	}
	if errData.Message != "Rate limit exceeded" {
		t.Errorf("expected message 'Rate limit exceeded', got '%s'", errData.Message)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_NonJSON(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	// Non-JSON lines should be silently skipped (return nil, nil)
	event, err := runner.parseStreamEvent(runID, opencodeSamples["non_json"])
	if err != nil {
		t.Fatalf("expected no error for non-JSON, got: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for non-JSON, got: %v", event)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_Empty(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	// Empty lines should be silently skipped
	event, err := runner.parseStreamEvent(runID, opencodeSamples["empty"])
	if err != nil {
		t.Fatalf("expected no error for empty line, got: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for empty line, got: %v", event)
	}
}

func TestOpenCodeRunner_ParseStreamEvent_FullStream(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	// Simulate a full stream of events
	streamLines := []string{
		opencodeSamples["step_start"],
		opencodeSamples["tool_use_write"],
		opencodeSamples["step_finish_with_cost"],
		opencodeSamples["step_start"], // Second step
		opencodeSamples["text"],
		opencodeSamples["step_finish_stop"],
	}

	var events []*domain.RunEvent
	metrics := &ExecutionMetrics{}
	lastAssistant := ""

	for _, line := range streamLines {
		event, err := runner.parseStreamEvent(runID, line)
		if err != nil {
			t.Fatalf("unexpected error parsing line: %v", err)
		}
		if event != nil {
			events = append(events, event)
			runner.updateMetrics(event, metrics, &lastAssistant)
		}
	}

	// Should have 6 events
	if len(events) != 6 {
		t.Errorf("expected 6 events, got %d", len(events))
	}

	// Verify event types in order
	expectedTypes := []domain.RunEventType{
		domain.EventTypeLog,     // step_start
		domain.EventTypeToolCall, // tool_use
		domain.EventTypeMetric,  // step_finish
		domain.EventTypeLog,     // step_start (2nd)
		domain.EventTypeMessage, // text
		domain.EventTypeMetric,  // step_finish
	}

	for i, expectedType := range expectedTypes {
		if i < len(events) && events[i].EventType != expectedType {
			t.Errorf("event %d: expected type %v, got %v", i, expectedType, events[i].EventType)
		}
	}

	// Verify last assistant message was captured
	if lastAssistant != "File created successfully." {
		t.Errorf("expected lastAssistant 'File created successfully.', got '%s'", lastAssistant)
	}

	// Verify metrics were updated
	if metrics.TurnsUsed != 1 {
		t.Errorf("expected 1 turn, got %d", metrics.TurnsUsed)
	}
	if metrics.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCallCount)
	}
}

func TestOpenCodeRunner_UpdateMetrics(t *testing.T) {
	runner := &OpenCodeRunner{}
	runID := uuid.New()

	metrics := &ExecutionMetrics{}
	lastAssistant := ""

	// Test message event updates
	msgEvent := domain.NewMessageEvent(runID, "assistant", "Hello world")
	runner.updateMetrics(msgEvent, metrics, &lastAssistant)
	if lastAssistant != "Hello world" {
		t.Errorf("expected lastAssistant 'Hello world', got '%s'", lastAssistant)
	}
	if metrics.TurnsUsed != 1 {
		t.Errorf("expected 1 turn, got %d", metrics.TurnsUsed)
	}

	// Test tool call event updates
	toolEvent := domain.NewToolCallEvent(runID, "bash", map[string]interface{}{"cmd": "ls"})
	runner.updateMetrics(toolEvent, metrics, &lastAssistant)
	if metrics.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCallCount)
	}

	// Test cost event updates
	costEvent, _ := runner.parseStreamEvent(runID, opencodeSamples["step_finish_with_cost"])
	runner.updateMetrics(costEvent, metrics, &lastAssistant)
	if metrics.TokensInput != 3562 {
		t.Errorf("expected TokensInput 3562, got %d", metrics.TokensInput)
	}
	if metrics.TokensOutput != 401 {
		t.Errorf("expected TokensOutput 401, got %d", metrics.TokensOutput)
	}
}
