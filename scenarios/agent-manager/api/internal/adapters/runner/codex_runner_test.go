package runner

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// mockPricingService is a test implementation of PricingService.
type mockPricingService struct {
	pricing map[string]*PricingCostCalculation
}

func (m *mockPricingService) CalculateCost(ctx context.Context, req PricingCostRequest) (*PricingCostCalculation, error) {
	// Look up by canonical model first
	if calc, ok := m.pricing[req.Model]; ok {
		// Calculate costs based on token counts and pricing
		inputPrice := 0.00000025  // Default input price per token
		outputPrice := 0.000002   // Default output price per token
		cacheReadPrice := 0.000000025

		return &PricingCostCalculation{
			InputCostUSD:     float64(req.InputTokens) * inputPrice,
			OutputCostUSD:    float64(req.OutputTokens) * outputPrice,
			CacheReadCostUSD: float64(req.CacheReadTokens) * cacheReadPrice,
			TotalCostUSD:     float64(req.InputTokens)*inputPrice + float64(req.OutputTokens)*outputPrice + float64(req.CacheReadTokens)*cacheReadPrice,
			CostSource:       calc.CostSource,
			Provider:         calc.Provider,
			CanonicalModel:   calc.CanonicalModel,
			PricingFetchedAt: calc.PricingFetchedAt,
		}, nil
	}
	return nil, nil
}

// =============================================================================
// CODEX JSON OUTPUT SAMPLES - CAPTURED 2025-12-19
// =============================================================================
//
// These test samples were captured from real Codex CLI output on 2025-12-19.
// Codex CLI version at time of capture: codex exec --json
// If tests start failing, the Codex JSON output format may have changed.
// Consider re-capturing samples from a current Codex version.
//
// To re-capture samples, run:
//   echo "Create a file called test.txt with 'hello'" | codex exec --json --skip-git-repo-check --full-auto - 2>&1
//
// =============================================================================

// Real Codex output samples captured 2025-12-19
var codexSamples = map[string]string{
	"thread.started": `{"type":"thread.started","thread_id":"019b3906-b365-7403-b3d1-70d60f6f06c4"}`,

	"turn.started": `{"type":"turn.started"}`,

	"reasoning": `{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"**Creating a file in /tmp**\n\nI'm thinking about how to create a file in the /tmp directory since the workspace-write seems to allow it. It's clear that the current working directory is /tmp, so I should be good to go. I don't need an elaborate plan for this; it's a straightforward task. I'll just use the apply_patch function to add the file directly. It feels manageable!"}}`,

	"file_change": `{"type":"item.completed","item":{"id":"item_1","type":"file_change","changes":[{"path":"/tmp/test123.txt","kind":"add"}],"status":"completed"}}`,

	"file_change_multiple": `{"type":"item.completed","item":{"id":"item_2","type":"file_change","changes":[{"path":"/tmp/file1.txt","kind":"add"},{"path":"/tmp/file2.txt","kind":"modify"},{"path":"/tmp/file3.txt","kind":"delete"}],"status":"completed"}}`,

	"agent_message": `{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Created ` + "`test123.txt`" + ` containing ` + "`hello`" + `."}}`,

	"turn.completed": `{"type":"turn.completed","usage":{"input_tokens":12810,"cached_input_tokens":12416,"output_tokens":83}}`,

	"error": `{"type":"error","error":{"code":"RATE_LIMIT","message":"Rate limit exceeded, please try again later"}}`,

	"data_prefix_tool_call":       `data: {"type":"item.started","item":{"id":"item_3","type":"tool_call","name":"bash","input":{"command":"ls -la"}}}`,
	"command_execution_started":   `{"type":"item.started","item":{"id":"item_4","type":"command_execution","command":"/bin/bash -lc \"echo test\"","aggregated_output":"","exit_code":null,"status":"in_progress"}}`,
	"command_execution_completed": `{"type":"item.completed","item":{"id":"item_4","type":"command_execution","command":"/bin/bash -lc \"echo test\"","aggregated_output":"test\n","exit_code":0,"status":"completed"}}`,
}

func newCodexRunnerWithPricing(runID uuid.UUID) *CodexRunner {
	mockSvc := &mockPricingService{
		pricing: map[string]*PricingCostCalculation{
			"gpt-5.1-codex-mini": {
				CostSource:       domain.CostSourcePricingTableEstimate,
				Provider:         "openrouter",
				CanonicalModel:   "openai/gpt-5.1-codex-mini",
				PricingFetchedAt: time.Now().UTC(),
			},
		},
	}
	runner := &CodexRunner{
		pricingService: mockSvc,
		runModels:      make(map[uuid.UUID]string),
	}
	runner.trackRunModel(runID, "gpt-5.1-codex-mini")
	return runner
}

// =============================================================================
// CODEX STREAM EVENT PARSING TESTS
// =============================================================================

func TestCodexRunner_ParseStreamEvent_ThreadStarted(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["thread.started"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeLog)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Level != "debug" {
		t.Errorf("Level = %s, want debug", logData.Level)
	}
	if logData.Message != "Thread started: 019b3906-b365-7403-b3d1-70d60f6f06c4" {
		t.Errorf("Message = %s, want 'Thread started: 019b3906-b365-7403-b3d1-70d60f6f06c4'", logData.Message)
	}
}

func TestCodexRunner_ParseStreamEvent_TurnStarted(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["turn.started"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeLog)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Message != "Turn started" {
		t.Errorf("Message = %s, want 'Turn started'", logData.Message)
	}
}

func TestCodexRunner_ParseStreamEvent_Reasoning(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["reasoning"])

	if event == nil {
		t.Fatal("expected non-nil event for reasoning")
	}
	if event.EventType != domain.EventTypeLog {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeLog)
	}

	logData, ok := event.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("expected LogEventData, got %T", event.Data)
	}
	if logData.Level != "debug" {
		t.Errorf("Level = %s, want debug", logData.Level)
	}
	// Should start with "Reasoning: "
	if len(logData.Message) < 11 || logData.Message[:11] != "Reasoning: " {
		t.Errorf("Message should start with 'Reasoning: ', got: %s", logData.Message[:min(50, len(logData.Message))])
	}
}

func TestCodexRunner_ParseStreamEvent_FileChange(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["file_change"])

	if event == nil {
		t.Fatal("expected non-nil event for file_change")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeToolCall)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}
	if toolData.ToolName != "file_change" {
		t.Errorf("ToolName = %s, want file_change", toolData.ToolName)
	}

	// Check that input contains files array
	files, ok := toolData.Input["files"]
	if !ok {
		t.Fatal("expected 'files' key in Input")
	}

	filesSlice, ok := files.([]map[string]string)
	if !ok {
		t.Fatalf("expected []map[string]string for files, got %T", files)
	}
	if len(filesSlice) != 1 {
		t.Fatalf("expected 1 file, got %d", len(filesSlice))
	}
	if filesSlice[0]["path"] != "/tmp/test123.txt" {
		t.Errorf("file path = %s, want /tmp/test123.txt", filesSlice[0]["path"])
	}
	if filesSlice[0]["kind"] != "add" {
		t.Errorf("file kind = %s, want add", filesSlice[0]["kind"])
	}

	// Check status
	status, ok := toolData.Input["status"]
	if !ok {
		t.Fatal("expected 'status' key in Input")
	}
	if status != "completed" {
		t.Errorf("status = %v, want completed", status)
	}
}

func TestCodexRunner_ParseStreamEvent_FileChangeMultiple(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["file_change_multiple"])

	if event == nil {
		t.Fatal("expected non-nil event for file_change_multiple")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeToolCall)
	}

	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}

	files, ok := toolData.Input["files"]
	if !ok {
		t.Fatal("expected 'files' key in Input")
	}

	filesSlice, ok := files.([]map[string]string)
	if !ok {
		t.Fatalf("expected []map[string]string for files, got %T", files)
	}
	if len(filesSlice) != 3 {
		t.Fatalf("expected 3 files, got %d", len(filesSlice))
	}

	// Verify each file change
	expectedChanges := []struct {
		path string
		kind string
	}{
		{"/tmp/file1.txt", "add"},
		{"/tmp/file2.txt", "modify"},
		{"/tmp/file3.txt", "delete"},
	}

	for i, expected := range expectedChanges {
		if filesSlice[i]["path"] != expected.path {
			t.Errorf("file[%d].path = %s, want %s", i, filesSlice[i]["path"], expected.path)
		}
		if filesSlice[i]["kind"] != expected.kind {
			t.Errorf("file[%d].kind = %s, want %s", i, filesSlice[i]["kind"], expected.kind)
		}
	}
}

func TestCodexRunner_ParseStreamEvent_AgentMessage(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["agent_message"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeMessage {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeMessage)
	}

	msgData, ok := event.Data.(*domain.MessageEventData)
	if !ok {
		t.Fatalf("expected MessageEventData, got %T", event.Data)
	}
	if msgData.Role != "assistant" {
		t.Errorf("Role = %s, want assistant", msgData.Role)
	}
	if msgData.Content != "Created `test123.txt` containing `hello`." {
		t.Errorf("Content = %s, want 'Created `test123.txt` containing `hello`.'", msgData.Content)
	}
}

func TestCodexRunner_ParseStreamEvent_TurnCompleted(t *testing.T) {
	runID := uuid.New()
	runner := newCodexRunnerWithPricing(runID)

	event := runner.parseCodexStreamEvent(runID, codexSamples["turn.completed"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeMetric {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeMetric)
	}

	costData, ok := event.Data.(*domain.CostEventData)
	if !ok {
		t.Fatalf("expected CostEventData, got %T", event.Data)
	}
	// Check input tokens (12810)
	if costData.InputTokens != 12810 {
		t.Errorf("InputTokens = %d, want 12810", costData.InputTokens)
	}
	// Check output tokens (83)
	if costData.OutputTokens != 83 {
		t.Errorf("OutputTokens = %d, want 83", costData.OutputTokens)
	}
	// Check cost is non-zero (calculated by pricing lookup)
	if costData.TotalCostUSD <= 0 {
		t.Errorf("TotalCostUSD = %f, want > 0", costData.TotalCostUSD)
	}
	if costData.CostSource != domain.CostSourcePricingTableEstimate {
		t.Errorf("CostSource = %s, want %s", costData.CostSource, domain.CostSourcePricingTableEstimate)
	}
	if costData.PricingProvider != "openrouter" {
		t.Errorf("PricingProvider = %s, want openrouter", costData.PricingProvider)
	}
	if costData.PricingModel != "openai/gpt-5.1-codex-mini" {
		t.Errorf("PricingModel = %s, want openai/gpt-5.1-codex-mini", costData.PricingModel)
	}
	// Check model is set to the requested config
	if costData.Model != "gpt-5.1-codex-mini" {
		t.Errorf("Model = %s, want gpt-5.1-codex-mini", costData.Model)
	}
}

func TestCodexRunner_ParseStreamEvent_DataPrefixToolCall(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["data_prefix_tool_call"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeToolCall)
	}
}

func TestCodexRunner_ParseStreamEvent_CommandExecutionStarted(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["command_execution_started"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeToolCall {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeToolCall)
	}
	toolData, ok := event.Data.(*domain.ToolCallEventData)
	if !ok {
		t.Fatalf("expected ToolCallEventData, got %T", event.Data)
	}
	if toolData.ToolName != "bash" {
		t.Errorf("ToolName = %s, want bash", toolData.ToolName)
	}
}

func TestCodexRunner_ParseStreamEvent_CommandExecutionCompleted(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["command_execution_completed"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeToolResult {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeToolResult)
	}
	toolData, ok := event.Data.(*domain.ToolResultEventData)
	if !ok {
		t.Fatalf("expected ToolResultEventData, got %T", event.Data)
	}
	if toolData.ToolName != "bash" {
		t.Errorf("ToolName = %s, want bash", toolData.ToolName)
	}
}

func TestCodexRunner_ParseStreamEvent_Error(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, codexSamples["error"])

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.EventType != domain.EventTypeError {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeError)
	}

	errData, ok := event.Data.(*domain.ErrorEventData)
	if !ok {
		t.Fatalf("expected ErrorEventData, got %T", event.Data)
	}
	if errData.Code != "RATE_LIMIT" {
		t.Errorf("Code = %s, want RATE_LIMIT", errData.Code)
	}
	if errData.Message != "Rate limit exceeded, please try again later" {
		t.Errorf("Message = %s, want 'Rate limit exceeded, please try again later'", errData.Message)
	}
}

func TestCodexRunner_ParseStreamEvent_EmptyLine(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, "")

	if event != nil {
		t.Errorf("expected nil event for empty line, got %+v", event)
	}
}

func TestCodexRunner_ParseStreamEvent_NonJSON(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, "Shell cwd was reset to /home/user/project")

	if event != nil {
		t.Errorf("expected nil event for non-JSON line, got %+v", event)
	}
}

func TestCodexRunner_ParseStreamEvent_InvalidJSON(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, `{"type":"invalid json`)

	if event != nil {
		t.Errorf("expected nil event for invalid JSON, got %+v", event)
	}
}

func TestCodexRunner_ParseStreamEvent_UnknownType(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()

	event := runner.parseCodexStreamEvent(runID, `{"type":"unknown.event.type"}`)

	if event != nil {
		t.Errorf("expected nil event for unknown type, got %+v", event)
	}
}

// =============================================================================
// CODEX STRUCT TESTS
// =============================================================================

func TestCodexStreamEvent_Unmarshal(t *testing.T) {
	var event CodexStreamEvent
	err := json.Unmarshal([]byte(codexSamples["file_change"]), &event)

	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if event.Type != "item.completed" {
		t.Errorf("Type = %s, want item.completed", event.Type)
	}
	if event.Item == nil {
		t.Fatal("Item is nil")
	}
	if event.Item.Type != "file_change" {
		t.Errorf("Item.Type = %s, want file_change", event.Item.Type)
	}
	if len(event.Item.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(event.Item.Changes))
	}
	if event.Item.Changes[0].Path != "/tmp/test123.txt" {
		t.Errorf("Changes[0].Path = %s, want /tmp/test123.txt", event.Item.Changes[0].Path)
	}
	if event.Item.Changes[0].Kind != "add" {
		t.Errorf("Changes[0].Kind = %s, want add", event.Item.Changes[0].Kind)
	}
	if event.Item.Status != "completed" {
		t.Errorf("Item.Status = %s, want completed", event.Item.Status)
	}
}

func TestCodexStreamEvent_UnmarshalUsage(t *testing.T) {
	var event CodexStreamEvent
	err := json.Unmarshal([]byte(codexSamples["turn.completed"]), &event)

	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if event.Type != "turn.completed" {
		t.Errorf("Type = %s, want turn.completed", event.Type)
	}
	if event.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if event.Usage.InputTokens != 12810 {
		t.Errorf("Usage.InputTokens = %d, want 12810", event.Usage.InputTokens)
	}
	if event.Usage.CachedInputTokens != 12416 {
		t.Errorf("Usage.CachedInputTokens = %d, want 12416", event.Usage.CachedInputTokens)
	}
	if event.Usage.OutputTokens != 83 {
		t.Errorf("Usage.OutputTokens = %d, want 83", event.Usage.OutputTokens)
	}
}

func TestCodexFileChange_Fields(t *testing.T) {
	change := CodexFileChange{
		Path: "/home/user/project/file.go",
		Kind: "modify",
	}

	if change.Path != "/home/user/project/file.go" {
		t.Errorf("Path = %s, want /home/user/project/file.go", change.Path)
	}
	if change.Kind != "modify" {
		t.Errorf("Kind = %s, want modify", change.Kind)
	}
}

// =============================================================================
// METRICS UPDATE TESTS
// =============================================================================

func TestCodexRunner_UpdateMetrics_ToolCall(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()
	metrics := &ExecutionMetrics{}
	lastAssistant := ""

	// Parse a file_change event (which becomes a tool_call)
	event := runner.parseCodexStreamEvent(runID, codexSamples["file_change"])
	runner.updateCodexMetrics(event, metrics, &lastAssistant)

	if metrics.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", metrics.ToolCallCount)
	}
}

func TestCodexRunner_UpdateMetrics_Message(t *testing.T) {
	runner := &CodexRunner{}
	runID := uuid.New()
	metrics := &ExecutionMetrics{}
	lastAssistant := ""

	event := runner.parseCodexStreamEvent(runID, codexSamples["agent_message"])
	runner.updateCodexMetrics(event, metrics, &lastAssistant)

	if metrics.TurnsUsed != 1 {
		t.Errorf("TurnsUsed = %d, want 1", metrics.TurnsUsed)
	}
	if lastAssistant != "Created `test123.txt` containing `hello`." {
		t.Errorf("lastAssistant = %s, want 'Created `test123.txt` containing `hello`.'", lastAssistant)
	}
}

func TestCodexRunner_UpdateMetrics_Tokens(t *testing.T) {
	runID := uuid.New()
	runner := newCodexRunnerWithPricing(runID)
	metrics := &ExecutionMetrics{}
	lastAssistant := ""

	event := runner.parseCodexStreamEvent(runID, codexSamples["turn.completed"])
	runner.updateCodexMetrics(event, metrics, &lastAssistant)

	// CostEventData now tracks input and output tokens separately
	if metrics.TokensInput != 12810 {
		t.Errorf("TokensInput = %d, want 12810", metrics.TokensInput)
	}
	if metrics.TokensOutput != 83 {
		t.Errorf("TokensOutput = %d, want 83", metrics.TokensOutput)
	}
	if metrics.CostEstimateUSD <= 0 {
		t.Errorf("CostEstimateUSD = %f, want > 0", metrics.CostEstimateUSD)
	}
}

// =============================================================================
// FULL STREAM SIMULATION TEST
// =============================================================================

func TestCodexRunner_ParseFullStream(t *testing.T) {
	runID := uuid.New()
	runner := newCodexRunnerWithPricing(runID)

	// Simulate a full Codex stream in order
	streamLines := []string{
		codexSamples["thread.started"],
		codexSamples["turn.started"],
		codexSamples["reasoning"],
		codexSamples["file_change"],
		codexSamples["agent_message"],
		codexSamples["turn.completed"],
	}

	metrics := &ExecutionMetrics{}
	lastAssistant := ""
	var events []*domain.RunEvent

	for _, line := range streamLines {
		event := runner.parseCodexStreamEvent(runID, line)
		if event != nil {
			events = append(events, event)
			runner.updateCodexMetrics(event, metrics, &lastAssistant)
		}
	}

	// Should have 6 events: 2 logs (thread, turn), 1 reasoning log, 1 tool_call, 1 message, 1 metric
	if len(events) != 6 {
		t.Errorf("expected 6 events, got %d", len(events))
	}

	// Verify event types in order
	expectedTypes := []domain.RunEventType{
		domain.EventTypeLog,      // thread.started
		domain.EventTypeLog,      // turn.started
		domain.EventTypeLog,      // reasoning
		domain.EventTypeToolCall, // file_change
		domain.EventTypeMessage,  // agent_message
		domain.EventTypeMetric,   // turn.completed
	}

	for i, expectedType := range expectedTypes {
		if i >= len(events) {
			break
		}
		if events[i].EventType != expectedType {
			t.Errorf("events[%d].EventType = %s, want %s", i, events[i].EventType, expectedType)
		}
	}

	// Verify final metrics
	if metrics.TurnsUsed != 1 {
		t.Errorf("TurnsUsed = %d, want 1", metrics.TurnsUsed)
	}
	if metrics.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", metrics.ToolCallCount)
	}
	// CostEventData tracks tokens separately
	if metrics.TokensInput != 12810 {
		t.Errorf("TokensInput = %d, want 12810", metrics.TokensInput)
	}
	if metrics.TokensOutput != 83 {
		t.Errorf("TokensOutput = %d, want 83", metrics.TokensOutput)
	}
	if metrics.CostEstimateUSD <= 0 {
		t.Errorf("CostEstimateUSD = %f, want > 0", metrics.CostEstimateUSD)
	}
	if lastAssistant != "Created `test123.txt` containing `hello`." {
		t.Errorf("lastAssistant mismatch")
	}
}

// Helper function for Go 1.21+
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
