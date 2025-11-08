package cdp

import (
	"context"
	"fmt"

	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// ExecuteInstruction adapts runtime.Instruction to CDP execution
// This provides compatibility with the existing client code
func (s *Session) ExecuteInstruction(ctx context.Context, instruction runtime.Instruction) (*runtime.ExecutionResponse, error) {
	var result *StepResult
	var err error

	switch instruction.Type {
	case "navigate":
		url := instruction.Params.URL
		waitUntil := instruction.Params.WaitUntil
		if waitUntil == "" {
			waitUntil = "load"
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteNavigate(ctx, url, waitUntil, timeoutMs, waitAfterMs)

	case "click":
		selector := instruction.Params.Selector
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteClick(ctx, selector, timeoutMs, waitAfterMs)

	case "wait":
		selector := instruction.Params.Selector
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteWait(ctx, selector, timeoutMs, waitAfterMs)

	case "assert":
		selector := instruction.Params.Selector
		mode := instruction.Params.AssertMode
		if mode == "" {
			mode = "exists"
		}
		// Handle expectedValue which could be any type
		expectedValue := ""
		if instruction.Params.ExpectedValue != nil {
			expectedValue = fmt.Sprintf("%v", instruction.Params.ExpectedValue)
		}
		// Handle caseSensitive pointer
		caseSensitive := false
		if instruction.Params.CaseSensitive != nil {
			caseSensitive = *instruction.Params.CaseSensitive
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}

		result, err = s.ExecuteAssert(ctx, selector, mode, expectedValue, caseSensitive, timeoutMs)

	case "screenshot":
		// Handle fullPage pointer
		fullPage := false
		if instruction.Params.FullPage != nil {
			fullPage = *instruction.Params.FullPage
		}
		result, err = s.ExecuteScreenshot(ctx, fullPage)

	case "type":
		selector := instruction.Params.Selector
		text := instruction.Params.Text
		// Handle Clear pointer
		clearFirst := false
		if instruction.Params.Clear != nil {
			clearFirst = *instruction.Params.Clear
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteType(ctx, selector, text, clearFirst, timeoutMs, waitAfterMs)

	case "evaluate":
		script := instruction.Params.Expression
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}

		result, err = s.ExecuteEvaluate(ctx, script, timeoutMs)

	default:
		return nil, fmt.Errorf("unsupported node type: %s", instruction.Type)
	}

	if err != nil && result == nil {
		// Create error result
		result = &StepResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Convert CDP result to runtime format
	response := &runtime.ExecutionResponse{
		Success: result.Success,
		Error:   result.Error,
		Steps: []runtime.StepResult{
			{
				Index:            instruction.Index,
				NodeID:           instruction.NodeID,
				Type:             instruction.Type,
				Success:          result.Success,
				Error:            result.Error,
				DurationMs:       result.DurationMs,
				FinalURL:         result.URL,
				ScreenshotBase64: result.Screenshot,
				ConsoleLogs:      convertConsoleLogs(result.ConsoleLogs),
				NetworkEvents:    convertNetworkEvents(result.NetworkEvents),
				// Store debug context in assertion for now (could add custom field)
				Assertion: buildAssertionResult(result.DebugContext),
				// Note: No DOMSnapshot needed in CDP mode - browser state persists!
			},
		},
	}

	return response, nil
}

// buildAssertionResult converts debug context to assertion result
func buildAssertionResult(debugCtx map[string]interface{}) *runtime.AssertionResult {
	if debugCtx == nil {
		return nil
	}

	// If this was an assert step, build proper assertion result
	if exists, ok := debugCtx["exists"].(bool); ok {
		return &runtime.AssertionResult{
			Success: true, // If we got here, assertion passed
			Actual: map[string]interface{}{
				"exists":     exists,
				"url":        debugCtx["url"],
				"allTestIds": debugCtx["allTestIds"],
			},
		}
	}

	return nil
}

// convertConsoleLogs converts CDP console logs to runtime format
func convertConsoleLogs(logs []ConsoleLog) []runtime.ConsoleLog {
	result := make([]runtime.ConsoleLog, len(logs))
	for i, log := range logs {
		result[i] = runtime.ConsoleLog{
			Type:      log.Type,
			Text:      log.Text,
			Timestamp: log.Timestamp.UnixMilli(),
		}
	}
	return result
}

// convertNetworkEvents converts CDP network events to runtime format
func convertNetworkEvents(events []NetworkEvent) []runtime.NetworkEvent {
	result := make([]runtime.NetworkEvent, len(events))
	for i, ev := range events {
		result[i] = runtime.NetworkEvent{
			Type:         ev.Type,
			URL:          ev.URL,
			Method:       ev.Method,
			Status:       int(ev.Status), // Convert int64 to int
			ResourceType: ev.ResourceType,
			Timestamp:    ev.Timestamp.UnixMilli(),
		}
	}
	return result
}

// SetViewport sets the browser viewport size
func (s *Session) SetViewport(width, height int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.viewportWidth = width
	s.viewportHeight = height
}

// LastHTML is not needed in CDP mode - stub for compatibility
func (s *Session) LastHTML() string {
	return ""
}

// SetLastHTML is not needed in CDP mode - stub for compatibility
func (s *Session) SetLastHTML(html string) {
	// No-op: CDP keeps browser state naturally
}
