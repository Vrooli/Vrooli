package cdp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

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

		// Handle scenario destination type
		if instruction.Params.DestinationType == "scenario" {
			scenarioURL, urlErr := resolveScenarioURL(instruction.Params.Scenario, instruction.Params.ScenarioPath)
			if urlErr != nil {
				return nil, fmt.Errorf("failed to resolve scenario URL for %s: %w", instruction.Params.Scenario, urlErr)
			}
			url = scenarioURL
		}

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

	case "focus":
		selector := strings.TrimSpace(instruction.Params.Selector)
		if selector == "" {
			return nil, fmt.Errorf("focus node missing selector")
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteFocus(ctx, selector, timeoutMs, waitAfterMs)

	case "blur":
		selector := strings.TrimSpace(instruction.Params.Selector)
		if selector == "" {
			return nil, fmt.Errorf("blur node missing selector")
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteBlur(ctx, selector, timeoutMs, waitAfterMs)

	case "wait":
		// Check if this is a time-based wait (durationMs) or element wait (selector)
		if instruction.Params.DurationMs > 0 {
			// Time-based wait - just sleep
			time.Sleep(time.Duration(instruction.Params.DurationMs) * time.Millisecond)
			result = &StepResult{
				Success: true,
				DurationMs: instruction.Params.DurationMs,
			}
		} else {
			// Element-based wait - wait for selector to appear
			selector := instruction.Params.Selector
			timeoutMs := instruction.Params.TimeoutMs
			if timeoutMs == 0 {
				timeoutMs = 30000
			}
			waitAfterMs := instruction.Params.WaitForMs
			result, err = s.ExecuteWait(ctx, selector, timeoutMs, waitAfterMs)
		}

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

	case "uploadFile":
		selector := strings.TrimSpace(instruction.Params.Selector)
		if selector == "" {
			return nil, fmt.Errorf("uploadFile node missing selector")
		}
		files := make([]string, 0, len(instruction.Params.FilePaths))
		for _, path := range instruction.Params.FilePaths {
			trimmed := strings.TrimSpace(path)
			if trimmed != "" {
				files = append(files, trimmed)
			}
		}
		if len(files) == 0 {
			if trimmed := strings.TrimSpace(instruction.Params.FilePath); trimmed != "" {
				files = append(files, trimmed)
			}
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("uploadFile node requires at least one absolute file path")
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteUploadFile(ctx, selector, files, timeoutMs, waitAfterMs)

	case "hover":
		selector := strings.TrimSpace(instruction.Params.Selector)
		if selector == "" {
			return nil, fmt.Errorf("hover node missing selector")
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		steps := instruction.Params.MovementSteps
		if steps == 0 {
			steps = 10
		}
		durationMs := instruction.Params.DurationMs
		if durationMs == 0 {
			durationMs = 350
		}
		waitAfterMs := instruction.Params.WaitForMs

		result, err = s.ExecuteHover(ctx, selector, timeoutMs, waitAfterMs, steps, durationMs)

	case "dragDrop":
		source := strings.TrimSpace(instruction.Params.DragSourceSelector)
		target := strings.TrimSpace(instruction.Params.DragTargetSelector)
		if source == "" {
			return nil, fmt.Errorf("dragDrop node missing source selector")
		}
		if target == "" {
			return nil, fmt.Errorf("dragDrop node missing target selector")
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		opts := dragDropOptions{
			sourceSelector: source,
			targetSelector: target,
			holdMs:         instruction.Params.DragHoldMs,
			steps:          instruction.Params.DragSteps,
			durationMs:     instruction.Params.DragDurationMs,
			offsetX:        instruction.Params.DragOffsetX,
			offsetY:        instruction.Params.DragOffsetY,
			waitAfterMs:    instruction.Params.WaitForMs,
		}
		if opts.steps <= 0 {
			opts.steps = 1
		}
		result, err = s.ExecuteDragAndDrop(ctx, opts, timeoutMs)

	case "gesture":
		opts := gestureOptions{
			Type:             instruction.Params.GestureType,
			Direction:        instruction.Params.GestureDirection,
			StartX:           float64(instruction.Params.GestureStartX),
			StartY:           float64(instruction.Params.GestureStartY),
			EndX:             float64(instruction.Params.GestureEndX),
			EndY:             float64(instruction.Params.GestureEndY),
			Distance:         instruction.Params.GestureDistance,
			Scale:            instruction.Params.GestureScale,
			DurationMs:       instruction.Params.GestureDurationMs,
			HoldMs:           instruction.Params.GestureHoldMs,
			Steps:            instruction.Params.GestureSteps,
			Selector:         instruction.Params.GestureSelector,
			TimeoutMs:        instruction.Params.TimeoutMs,
			HasExplicitStart: instruction.Params.GestureHasStart,
			HasExplicitEnd:   instruction.Params.GestureHasEnd,
			HasStartX:        instruction.Params.GestureHasStartX,
			HasStartY:        instruction.Params.GestureHasStartY,
			HasEndX:          instruction.Params.GestureHasEndX,
			HasEndY:          instruction.Params.GestureHasEndY,
		}
		result, err = s.ExecuteGesture(ctx, opts)

	case "evaluate":
		script := instruction.Params.Expression
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}

		result, err = s.ExecuteEvaluate(ctx, script, timeoutMs)

	case "keyboard":
		keyValue := instruction.Params.KeyValue
		if keyValue == "" {
			return nil, fmt.Errorf("keyboard node missing key value")
		}
		eventType := instruction.Params.KeyEventType
		modifiers := instruction.Params.KeyModifiers
		delayMs := instruction.Params.DelayMs
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}

		result, err = s.ExecuteKeyboard(ctx, keyValue, eventType, modifiers, delayMs, timeoutMs)

	case "scroll":
		scrollType := strings.ToLower(strings.TrimSpace(instruction.Params.ScrollType))
		if scrollType == "" {
			scrollType = "page"
		}
		direction := strings.ToLower(strings.TrimSpace(instruction.Params.ScrollDirection))
		behavior := instruction.Params.ScrollBehavior
		opts := scrollOptions{
			scrollType:     scrollType,
			selector:       strings.TrimSpace(instruction.Params.Selector),
			targetSelector: strings.TrimSpace(instruction.Params.ScrollTargetSelector),
			direction:      direction,
			behavior:       behavior,
			amount:         instruction.Params.ScrollAmount,
			x:              instruction.Params.ScrollX,
			y:              instruction.Params.ScrollY,
			maxAttempts:    instruction.Params.ScrollMaxAttempts,
			waitAfterMs:    instruction.Params.WaitForMs,
		}
		if opts.amount <= 0 {
			opts.amount = defaultScrollAmountPixels
		}
		if opts.maxAttempts < 0 {
			opts.maxAttempts = 0
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		result, err = s.ExecuteScroll(ctx, opts, timeoutMs)

	case "rotate":
		orientation := strings.TrimSpace(instruction.Params.RotateOrientation)
		if orientation == "" {
			orientation = rotateOrientationPortrait
		}
		angle := instruction.Params.RotateAngle
		waitAfterMs := instruction.Params.WaitForMs
		result, err = s.ExecuteRotate(ctx, orientation, angle, waitAfterMs)

	case "select":
		selector := strings.TrimSpace(instruction.Params.Selector)
		if selector == "" {
			return nil, fmt.Errorf("select node missing selector")
		}
		mode := instruction.Params.SelectionMode
		if mode == "" {
			mode = "value"
		}
		timeoutMs := instruction.Params.TimeoutMs
		if timeoutMs == 0 {
			timeoutMs = 30000
		}
		waitAfterMs := instruction.Params.WaitForMs
		result, err = s.ExecuteSelect(
			ctx,
			selector,
			mode,
			instruction.Params.OptionValue,
			instruction.Params.OptionText,
			instruction.Params.OptionIndex,
			instruction.Params.OptionValues,
			instruction.Params.MultiSelect,
			timeoutMs,
			waitAfterMs,
		)
	case "tabSwitch":
		opts := tabSwitchOptions{
			SwitchBy:   instruction.Params.TabSwitchBy,
			Index:      instruction.Params.TabIndex,
			TitleMatch: instruction.Params.TabTitleMatch,
			URLMatch:   instruction.Params.TabURLMatch,
			WaitForNew: instruction.Params.TabWaitForNew,
			TimeoutMs:  instruction.Params.TimeoutMs,
			CloseOld:   instruction.Params.TabCloseOld,
		}
		result, err = s.ExecuteTabSwitch(ctx, opts)
	case "frameSwitch":
		opts := frameSwitchOptions{
			SwitchBy:  instruction.Params.FrameSwitchBy,
			Index:     instruction.Params.FrameIndex,
			Name:      instruction.Params.FrameName,
			Selector:  instruction.Params.FrameSelector,
			URLMatch:  instruction.Params.FrameURLMatch,
			TimeoutMs: instruction.Params.TimeoutMs,
		}
		result, err = s.ExecuteFrameSwitch(ctx, opts)
	case "conditional":
		opts := conditionalOptions{
			Type:           instruction.Params.ConditionType,
			Expression:     instruction.Params.ConditionExpression,
			Selector:       instruction.Params.ConditionSelector,
			Variable:       instruction.Params.ConditionVariable,
			Operator:       instruction.Params.ConditionOperator,
			Value:          instruction.Params.ConditionValue,
			Negate:         boolFromPointer(instruction.Params.ConditionNegate),
			TimeoutMs:      instruction.Params.TimeoutMs,
			PollIntervalMs: instruction.Params.ConditionPollIntervalMs,
			Variables:      instruction.Context,
		}
		result, err = s.ExecuteConditional(ctx, opts)
	case "extract":
		allMatches := instruction.Params.AllMatches != nil && *instruction.Params.AllMatches
		result, err = s.ExecuteExtract(ctx, instruction.Params.Selector, instruction.Params.ExtractType, instruction.Params.Attribute, allMatches, instruction.Params.TimeoutMs)
	case "setVariable":
		result, err = s.ExecuteSetVariable(ctx, instruction)
	case "useVariable":
		result, err = executeUseVariableInstruction(instruction)
	case "setCookie":
		result, err = s.ExecuteSetCookie(ctx, instruction.Params)
	case "getCookie":
		result, err = s.ExecuteGetCookie(ctx, instruction.Params)
	case "clearCookie":
		result, err = s.ExecuteClearCookie(ctx, instruction.Params)
	case "setStorage":
		result, err = s.ExecuteSetStorage(ctx, instruction.Params)
	case "getStorage":
		result, err = s.ExecuteGetStorage(ctx, instruction.Params)
	case "clearStorage":
		result, err = s.ExecuteClearStorage(ctx, instruction.Params)
	case "networkMock":
		result, err = s.ExecuteNetworkMock(ctx, instruction.Params)
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
				Index:              instruction.Index,
				NodeID:             instruction.NodeID,
				Type:               instruction.Type,
				Success:            result.Success,
				Error:              result.Error,
				DurationMs:         result.DurationMs,
				FinalURL:           result.URL,
				ScreenshotBase64:   result.Screenshot,
				ConsoleLogs:        convertConsoleLogs(result.ConsoleLogs),
				NetworkEvents:      convertNetworkEvents(result.NetworkEvents),
				ElementBoundingBox: result.ElementBoundingBox,
				ExtractedData:      result.ExtractedData,
				// Store debug context in assertion for now (could add custom field)
				Assertion:       buildAssertionResult(result.DebugContext),
				ConditionResult: result.Condition,
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

func executeUseVariableInstruction(instruction runtime.Instruction) (*StepResult, error) {
	name := strings.TrimSpace(instruction.Params.VariableName)
	if name == "" {
		return &StepResult{Success: false, Error: "variable name missing"}, fmt.Errorf("variable name missing")
	}
	value := any(nil)
	if instruction.Context != nil {
		value = instruction.Context[name]
	}
	required := instruction.Params.VariableRequired != nil && *instruction.Params.VariableRequired
	if value == nil {
		if required {
			return &StepResult{Success: false, Error: fmt.Sprintf("variable %s is not defined", name)}, fmt.Errorf("variable %s is not defined", name)
		}
	}
	output := value
	if transform := strings.TrimSpace(instruction.Params.VariableTransform); transform != "" {
		replacement := fmt.Sprintf("%v", value)
		output = strings.ReplaceAll(transform, "{{value}}", replacement)
	}
	return &StepResult{Success: true, ExtractedData: output}, nil
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

// resolveScenarioURL resolves a scenario name to its UI URL using the vrooli CLI
func resolveScenarioURL(scenarioName, scenarioPath string) (string, error) {
	if scenarioName == "" {
		return "", fmt.Errorf("scenario name is required for destinationType=scenario")
	}

	// Call vrooli CLI to get scenario URL
	cmd := exec.Command("vrooli", "scenario", "open", scenarioName, "--print-url")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve scenario %s URL: %w", scenarioName, err)
	}

	baseURL := strings.TrimSpace(string(output))
	if baseURL == "" {
		return "", fmt.Errorf("vrooli CLI returned empty URL for scenario %s", scenarioName)
	}

	// Append scenario path if provided
	if scenarioPath != "" {
		// Ensure scenarioPath starts with /
		if !strings.HasPrefix(scenarioPath, "/") {
			scenarioPath = "/" + scenarioPath
		}
		// Remove trailing slash from baseURL if present
		baseURL = strings.TrimSuffix(baseURL, "/")
		return baseURL + scenarioPath, nil
	}

	return baseURL, nil
}
