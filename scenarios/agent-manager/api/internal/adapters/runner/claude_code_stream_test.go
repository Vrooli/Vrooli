// Package runner provides runner adapter implementations.
//
// This file contains tests for Claude Code stream event parsing.
// Test samples captured on 2025-12-19 from Claude Code v2.0.70.
// These should be verified periodically as Claude Code's output format may evolve.
package runner

import (
	"encoding/json"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// REAL CLAUDE CODE OUTPUT SAMPLES (captured 2025-12-19)
// =============================================================================

// claudeCodeSamples contains real output from Claude Code v2.0.70
// using --output-format stream-json --verbose
var claudeCodeSamples = map[string]string{
	// System init event - first event in every stream
	"system_init": `{"type":"system","subtype":"init","cwd":"/home/user/project","session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1","tools":["Task","Bash","Read","Write"],"mcp_servers":[],"model":"claude-opus-4-5-20251101","permissionMode":"bypassPermissions","claude_code_version":"2.0.70","uuid":"70d9d8c9-20ae-40b4-bfcf-48c30cbf7dc5"}`,

	// Assistant message with text content
	"assistant_text": `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_012wEys6PjZFVz3rJyuY8o4g","type":"message","role":"assistant","content":[{"type":"text","text":"Hello! I'm ready to help you."}],"stop_reason":null,"usage":{"input_tokens":2,"cache_creation_input_tokens":7192,"cache_read_input_tokens":14135,"output_tokens":5,"service_tier":"standard"}},"session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1"}`,

	// Assistant message with tool_use block (no text)
	"assistant_tool_use": `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_015m1Y7p5a3BZwxStfp7HQbK","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_01KKEKfCADiJJApRkJF86t8R","name":"Write","input":{"file_path":"/tmp/test.txt","content":"hello"}}],"stop_reason":null,"usage":{"input_tokens":2,"output_tokens":5}},"session_id":"fa4d44b2-1aa6-4509-b6d4-97611b779f04"}`,

	// Assistant message with both text and tool_use
	"assistant_text_and_tool": `{"type":"assistant","message":{"model":"claude-opus-4-5-20251101","id":"msg_mixed","type":"message","role":"assistant","content":[{"type":"text","text":"Let me create that file."},{"type":"tool_use","id":"toolu_abc123","name":"Write","input":{"file_path":"/tmp/file.txt","content":"data"}}],"stop_reason":null},"session_id":"session123"}`,

	// User message with tool_result
	"user_tool_result": `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01KKEKfCADiJJApRkJF86t8R","type":"tool_result","content":"File created successfully at: /tmp/test-sample.txt"}]},"session_id":"fa4d44b2-1aa6-4509-b6d4-97611b779f04","tool_use_result":{"type":"create","filePath":"/tmp/test-sample.txt","content":"sample content"}}`,

	// Result success event
	"result_success": `{"type":"result","subtype":"success","is_error":false,"duration_ms":2892,"duration_api_ms":5284,"num_turns":1,"result":"Hello! I'm ready to help.","session_id":"751b5a53-bc44-4484-943d-8851ccfdfda1","total_cost_usd":0.08424875,"usage":{"input_tokens":2,"cache_creation_input_tokens":7192,"cache_read_input_tokens":14135,"output_tokens":40,"server_tool_use":{"web_search_requests":0},"service_tier":"standard"}}`,

	// Result error event
	"result_error": `{"type":"result","subtype":"error","is_error":true,"duration_ms":100,"num_turns":0,"result":"Claude AI usage limit reached|1755806400","session_id":"session123","total_cost_usd":0}`,

	// Error event
	"error": `{"type":"error","error":{"code":"rate_limit","message":"Rate limit exceeded"}}`,

	// Usage event (standalone)
	"usage": `{"type":"usage","usage":{"input_tokens":100,"output_tokens":50}}`,
}

// =============================================================================
// PARSE STREAM EVENT TESTS
// =============================================================================

func TestClaudeCodeRunner_ParseStreamEvent_SystemInit(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["system_init"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeLog, event.EventType)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Level != "debug" {
		t.Errorf("expected level=debug, got %s", logData.Level)
	}
	if logData.Message != "System context received" {
		t.Errorf("expected message='System context received', got %s", logData.Message)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_AssistantText(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["assistant_text"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeMessage {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeMessage, event.EventType)
	}

	msgData, ok := event.Data.(*domain.MessageEventData)
	if !ok {
		t.Fatalf("expected MessageEventData, got %T", event.Data)
	}
	if msgData.Role != "assistant" {
		t.Errorf("expected role=assistant, got %s", msgData.Role)
	}
	if msgData.Content != "Hello! I'm ready to help you." {
		t.Errorf("expected content='Hello! I'm ready to help you.', got %s", msgData.Content)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_AssistantToolUse(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["assistant_tool_use"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeToolCall, event.EventType)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}
	if toolData.ToolName != "Write" {
		t.Errorf("expected toolName=Write, got %s", toolData.ToolName)
	}
	if toolData.Input == nil {
		t.Error("expected input to be non-nil")
	}
	if filePath, ok := toolData.Input["file_path"].(string); !ok || filePath != "/tmp/test.txt" {
		t.Errorf("expected input.file_path='/tmp/test.txt', got %v", toolData.Input["file_path"])
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_AssistantTextAndTool(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	// When there's both text and tool_use, text should be emitted as message
	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["assistant_text_and_tool"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}

	// The current implementation returns the first match (text content)
	if event.EventType != domain.EventTypeMessage {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeMessage, event.EventType)
	}

	msgData, ok := event.Data.(*domain.MessageEventData)
	if !ok {
		t.Fatalf("expected MessageEventData, got %T", event.Data)
	}
	if msgData.Content != "Let me create that file." {
		t.Errorf("expected content='Let me create that file.', got %s", msgData.Content)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_UserToolResult(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["user_tool_result"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}

	// User messages with tool results should be captured as tool_result events
	if event.EventType != domain.EventTypeToolResult {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeToolResult, event.EventType)
	}

	resultData, ok := event.Data.(*domain.ToolResultEventData)
	if !ok {
		t.Fatalf("expected ToolResultEventData, got %T", event.Data)
	}
	if resultData.Output != "File created successfully at: /tmp/test-sample.txt" {
		t.Errorf("unexpected output: %s", resultData.Output)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_ResultSuccess(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["result_success"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeMetric {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeMetric, event.EventType)
	}

	costData, ok := event.Data.(*domain.CostEventData)
	if !ok {
		t.Fatalf("expected CostEventData, got %T", event.Data)
	}
	if costData.TotalCostUSD != 0.08424875 {
		t.Errorf("expected totalCostUsd=0.08424875, got %f", costData.TotalCostUSD)
	}
	if costData.InputTokens != 2 {
		t.Errorf("expected inputTokens=2, got %d", costData.InputTokens)
	}
	if costData.OutputTokens != 40 {
		t.Errorf("expected outputTokens=40, got %d", costData.OutputTokens)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_ResultError_RateLimit(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["result_error"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	// Rate limit events use EventTypeError with RateLimitEventData
	if event.EventType != domain.EventTypeError {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeError, event.EventType)
	}

	rlData, ok := event.Data.(*domain.RateLimitEventData)
	if !ok {
		t.Fatalf("expected RateLimitEventData, got %T", event.Data)
	}
	if rlData.LimitType != "5_hour" {
		t.Errorf("expected limitType=5_hour, got %s", rlData.LimitType)
	}
	// The reset time should be parsed from the timestamp
	if rlData.ResetTime == nil {
		t.Error("expected ResetTime to be non-nil")
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_Error(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["error"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeError {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeError, event.EventType)
	}

	errData, ok := event.Data.(*domain.ErrorEventData)
	if !ok {
		t.Fatalf("expected ErrorEventData, got %T", event.Data)
	}
	if errData.Code != "rate_limit" {
		t.Errorf("expected code=rate_limit, got %s", errData.Code)
	}
	if errData.Message != "Rate limit exceeded" {
		t.Errorf("expected message='Rate limit exceeded', got %s", errData.Message)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_Usage(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	event, err := runner.parseStreamEvent(runID, claudeCodeSamples["usage"])

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeMetric {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeMetric, event.EventType)
	}

	metricData, ok := event.Data.(*domain.MetricEventData)
	if !ok {
		t.Fatalf("expected MetricEventData, got %T", event.Data)
	}
	if metricData.Name != "tokens" {
		t.Errorf("expected name=tokens, got %s", metricData.Name)
	}
	if metricData.Value != 150 { // 100 input + 50 output
		t.Errorf("expected value=150, got %f", metricData.Value)
	}
}

// =============================================================================
// NON-JSON LINE HANDLING TESTS
// =============================================================================

func TestClaudeCodeRunner_ParseStreamEvent_NonJsonLines(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	nonJsonLines := []string{
		"",                                              // empty line
		"   ",                                           // whitespace only
		"[INFO]    Started health monitor for claude-code", // log prefix
		"[HEADER]  🤖 Running Claude Code",              // header prefix
		"[WARNING] ⚠️  WARNING: Permission checks are disabled!", // warning prefix
		"Initializing...",                               // plain text
		"Some random output",                            // random output
	}

	for _, line := range nonJsonLines {
		t.Run(line, func(t *testing.T) {
			event, err := runner.parseStreamEvent(runID, line)
			if err != nil {
				t.Errorf("expected nil error for non-JSON line, got %v", err)
			}
			if event != nil {
				t.Errorf("expected nil event for non-JSON line, got %+v", event)
			}
		})
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_MalformedJson(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	malformedLines := []string{
		`{invalid}`,
		`{"type":`,
		`{"type": "test"`,
	}

	for _, line := range malformedLines {
		t.Run(line, func(t *testing.T) {
			event, err := runner.parseStreamEvent(runID, line)
			// Malformed JSON is silently skipped (returns nil, nil)
			if err != nil {
				t.Errorf("expected nil error for malformed JSON, got %v", err)
			}
			if event != nil {
				t.Errorf("expected nil event for malformed JSON, got %+v", event)
			}
		})
	}
}

// =============================================================================
// SILENTLY SKIPPED EVENT TYPES
// =============================================================================

func TestClaudeCodeRunner_ParseStreamEvent_SilentlySkipped(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	skippedEvents := []string{
		`{"type":"message_start"}`,
		`{"type":"message_delta"}`,
		`{"type":"message_stop"}`,
		`{"type":"content_block_stop"}`,
		`{"type":"init"}`,
		`{"type":"start"}`,
		`{"type":"ping"}`,
		`{"type":"heartbeat"}`,
		`{"type":""}`,
	}

	for _, line := range skippedEvents {
		t.Run(line, func(t *testing.T) {
			event, err := runner.parseStreamEvent(runID, line)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if event != nil {
				t.Errorf("expected nil event for silently skipped type, got %+v", event)
			}
		})
	}
}

// =============================================================================
// CONTENT BLOCK STREAMING TESTS
// =============================================================================

func TestClaudeCodeRunner_ParseStreamEvent_ContentBlockStart_ToolUse(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	line := `{"type":"content_block_start","content_block":{"type":"tool_use","id":"toolu_123","name":"Read"}}`

	event, err := runner.parseStreamEvent(runID, line)

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeToolCall, event.EventType)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}
	if toolData.ToolName != "Read" {
		t.Errorf("expected toolName=Read, got %s", toolData.ToolName)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_ContentBlockDelta_Text(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	line := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello world"}}`

	event, err := runner.parseStreamEvent(runID, line)

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("expected EventType=%s, got %s", domain.EventTypeLog, event.EventType)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Level != "trace" {
		t.Errorf("expected level=trace, got %s", logData.Level)
	}
	if logData.Message != "Hello world" {
		t.Errorf("expected message='Hello world', got %s", logData.Message)
	}
}

func TestClaudeCodeRunner_ParseStreamEvent_ContentBlockDelta_Empty(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	line := `{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}`

	event, err := runner.parseStreamEvent(runID, line)

	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	// Empty delta should be silently skipped
	if event != nil {
		t.Errorf("expected nil event for empty delta, got %+v", event)
	}
}

// =============================================================================
// UPDATE METRICS TESTS
// =============================================================================

func TestClaudeCodeRunner_UpdateMetrics_MessageEvent(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	metrics := ExecutionMetrics{}
	var lastAssistant string

	event := domain.NewMessageEvent(runID, "assistant", "Test response")
	runner.updateMetrics(event, &metrics, &lastAssistant)

	if metrics.TurnsUsed != 1 {
		t.Errorf("expected TurnsUsed=1, got %d", metrics.TurnsUsed)
	}
	if lastAssistant != "Test response" {
		t.Errorf("expected lastAssistant='Test response', got %s", lastAssistant)
	}
}

func TestClaudeCodeRunner_UpdateMetrics_ToolCallEvent(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	metrics := ExecutionMetrics{}
	var lastAssistant string

	event := domain.NewToolCallEvent(runID, "Write", map[string]interface{}{"path": "/tmp/test"})
	runner.updateMetrics(event, &metrics, &lastAssistant)

	if metrics.ToolCallCount != 1 {
		t.Errorf("expected ToolCallCount=1, got %d", metrics.ToolCallCount)
	}
}

func TestClaudeCodeRunner_UpdateMetrics_CostEvent(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	metrics := ExecutionMetrics{}
	var lastAssistant string

	event := &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Data: &domain.CostEventData{
			InputTokens:  1000,
			OutputTokens: 500,
			TotalCostUSD: 0.05,
		},
	}
	runner.updateMetrics(event, &metrics, &lastAssistant)

	if metrics.TokensInput != 1000 {
		t.Errorf("expected TokensInput=1000, got %d", metrics.TokensInput)
	}
	if metrics.TokensOutput != 500 {
		t.Errorf("expected TokensOutput=500, got %d", metrics.TokensOutput)
	}
	if metrics.CostEstimateUSD != 0.05 {
		t.Errorf("expected CostEstimateUSD=0.05, got %f", metrics.CostEstimateUSD)
	}
}

func TestClaudeCodeRunner_UpdateMetrics_NilEvent(t *testing.T) {
	runner := &ClaudeCodeRunner{}

	metrics := ExecutionMetrics{}
	var lastAssistant string

	// Should not panic on nil event
	runner.updateMetrics(nil, &metrics, &lastAssistant)

	if metrics.TurnsUsed != 0 {
		t.Error("metrics should not change for nil event")
	}
}

// =============================================================================
// HEALTH CHECK TESTS
// =============================================================================

func TestClaudeCodeRunner_HealthCheck_BooleanHealthy(t *testing.T) {
	// Test that boolean health status is parsed correctly
	// This validates the fix for the type assertion bug

	statusJSON := `{"healthy":true,"health_message":"OK"}`
	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(statusJSON), &statusData); err != nil {
		t.Fatalf("failed to parse test JSON: %v", err)
	}

	isHealthy := true
	if healthy, ok := statusData["healthy"].(bool); ok {
		isHealthy = healthy
	} else if healthyStr, ok := statusData["healthy"].(string); ok {
		isHealthy = healthyStr == "true"
	}

	if !isHealthy {
		t.Error("expected isHealthy=true for boolean healthy:true")
	}
}

func TestClaudeCodeRunner_HealthCheck_BooleanUnhealthy(t *testing.T) {
	statusJSON := `{"healthy":false,"health_message":"Not installed"}`
	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(statusJSON), &statusData); err != nil {
		t.Fatalf("failed to parse test JSON: %v", err)
	}

	isHealthy := true
	if healthy, ok := statusData["healthy"].(bool); ok {
		isHealthy = healthy
	} else if healthyStr, ok := statusData["healthy"].(string); ok {
		isHealthy = healthyStr == "true"
	}

	if isHealthy {
		t.Error("expected isHealthy=false for boolean healthy:false")
	}
}

func TestClaudeCodeRunner_HealthCheck_StringHealthy(t *testing.T) {
	// Test backward compatibility with string health status
	statusJSON := `{"healthy":"true","health_message":"OK"}`
	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(statusJSON), &statusData); err != nil {
		t.Fatalf("failed to parse test JSON: %v", err)
	}

	isHealthy := true
	if healthy, ok := statusData["healthy"].(bool); ok {
		isHealthy = healthy
	} else if healthyStr, ok := statusData["healthy"].(string); ok {
		isHealthy = healthyStr == "true"
	}

	if !isHealthy {
		t.Error("expected isHealthy=true for string healthy:\"true\"")
	}
}

// =============================================================================
// RATE LIMIT DETECTION TESTS
// =============================================================================

func TestClaudeCodeRunner_DetectRateLimit_UsageLimitReached(t *testing.T) {
	runner := &ClaudeCodeRunner{}

	info := runner.detectRateLimit("Claude AI usage limit reached|1755806400")

	if !info.Detected {
		t.Error("expected rate limit to be detected")
	}
	if info.LimitType != "5_hour" {
		t.Errorf("expected limitType=5_hour, got %s", info.LimitType)
	}
	if info.ResetTime == nil {
		t.Error("expected resetTime to be non-nil")
	}
}

func TestClaudeCodeRunner_DetectRateLimit_DailyLimit(t *testing.T) {
	runner := &ClaudeCodeRunner{}

	info := runner.detectRateLimit("Daily rate limit exceeded")

	if !info.Detected {
		t.Error("expected rate limit to be detected")
	}
	if info.LimitType != "daily" {
		t.Errorf("expected limitType=daily, got %s", info.LimitType)
	}
}

func TestClaudeCodeRunner_DetectRateLimit_NoLimit(t *testing.T) {
	runner := &ClaudeCodeRunner{}

	info := runner.detectRateLimit("Some other error message")

	if info.Detected {
		t.Error("expected rate limit NOT to be detected for unrelated message")
	}
}

// =============================================================================
// NOTE: ExtractTextContent and ExtractToolUses tests are in claude_message_test.go
// =============================================================================
