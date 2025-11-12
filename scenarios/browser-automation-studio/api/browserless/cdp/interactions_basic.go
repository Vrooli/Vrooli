package cdp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// ExecuteClick clicks an element and reports telemetry.
func (s *Session) ExecuteClick(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.Click(selector, s.frameQueryOptions(chromedp.ByQuery)...),
	)
	if err != nil {
		result.Error = fmt.Sprintf("Click failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Warn("Failed to get page info after click")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteWait waits for an element to appear/become visible.
func (s *Session) ExecuteWait(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	err := chromedp.Run(timeoutCtx, chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...))
	if err != nil {
		result.Error = fmt.Sprintf("Wait failed: selector %s not found", selector)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Warn("Failed to get page info after wait")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteAssert checks element existence or text content depending on the provided mode.
func (s *Session) ExecuteAssert(ctx context.Context, selector, mode, expectedValue string, caseSensitive bool, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: make(map[string]interface{})}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var currentURL string
	chromedp.Run(s.ctx, chromedp.Location(&currentURL))
	result.DebugContext["url"] = currentURL

	var testIDs []string
	if err := s.evalWithFrame(s.ctx, `Array.from(document.querySelectorAll('[data-testid]')).map(el => el.getAttribute('data-testid'))`, &testIDs); err != nil {
		s.log.WithError(err).Warn("failed to enumerate data-testid attributes")
	}
	result.DebugContext["allTestIds"] = testIDs

	switch mode {
	case "exists":
		if err := chromedp.Run(timeoutCtx, chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...)); err != nil {
			result.Error = fmt.Sprintf("Expected selector %q to exist", selector)
			result.DebugContext["exists"] = false
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
		result.DebugContext["exists"] = true
	case "not_exists":
		var nodes []*cdp.Node
		err := chromedp.Run(timeoutCtx, chromedp.Nodes(selector, &nodes, s.frameQueryOptions(chromedp.ByQuery)...))
		if err == nil && len(nodes) > 0 {
			result.Error = fmt.Sprintf("Expected selector %q to be absent but it exists", selector)
			result.DebugContext["exists"] = true
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, fmt.Errorf("element should not exist")
		}
		result.DebugContext["exists"] = false
	case "exists_or_not":
		var nodes []*cdp.Node
		chromedp.Run(s.ctx, chromedp.Nodes(selector, &nodes, s.frameQueryOptions(chromedp.ByQuery)...))
		result.DebugContext["exists"] = len(nodes) > 0
	case "text_equals", "text_contains":
		var text string
		err := chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
			chromedp.Text(selector, &text, s.frameQueryOptions(chromedp.ByQuery)...),
		)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get text from %q: %v", selector, err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}

		compareText := text
		compareExpected := expectedValue
		if !caseSensitive {
			compareText = strings.ToLower(text)
			compareExpected = strings.ToLower(expectedValue)
		}

		if mode == "text_equals" {
			if compareText != compareExpected {
				result.Error = fmt.Sprintf("Expected text to equal %q but got %q", expectedValue, text)
				result.DurationMs = int(time.Since(start).Milliseconds())
				return result, fmt.Errorf("text mismatch")
			}
		} else if !strings.Contains(compareText, compareExpected) {
			result.Error = fmt.Sprintf("Expected text to contain %q but got %q", expectedValue, text)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, fmt.Errorf("text mismatch")
		}
	default:
		return nil, fmt.Errorf("unsupported assert mode: %s", mode)
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteScreenshot captures a screenshot of either the viewport or full page.
func (s *Session) ExecuteScreenshot(ctx context.Context, fullPage bool) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	var buf []byte
	var err error
	if fullPage {
		err = chromedp.Run(s.ctx, chromedp.FullScreenshot(&buf, 100))
	} else {
		err = chromedp.Run(s.ctx, chromedp.CaptureScreenshot(&buf))
	}
	if err != nil {
		result.Error = fmt.Sprintf("Screenshot failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	result.Screenshot = base64.StdEncoding.EncodeToString(buf)

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteType types text into an input field, optionally clearing it first.
func (s *Session) ExecuteType(ctx context.Context, selector, text string, clearFirst bool, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	actions := []chromedp.Action{
		chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.Click(selector, s.frameQueryOptions(chromedp.ByQuery)...),
	}
	if clearFirst {
		actions = append(actions, chromedp.Clear(selector, s.frameQueryOptions(chromedp.ByQuery)...))
	}
	actions = append(actions, chromedp.SendKeys(selector, text, s.frameQueryOptions(chromedp.ByQuery)...))

	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		result.Error = fmt.Sprintf("Type failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteUploadFile selects files for an <input type="file"> element.
func (s *Session) ExecuteUploadFile(ctx context.Context, selector string, files []string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{"selector": selector, "files": files}}

	if len(files) == 0 {
		err := fmt.Errorf("no files supplied for upload")
		result.Error = err.Error()
		return result, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	if err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.SetUploadFiles(selector, files, s.frameQueryOptions(chromedp.ByQuery)...),
	); err != nil {
		result.Error = fmt.Sprintf("Upload failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Warn("Failed to get page info after upload")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteKeyboard dispatches low-level keyboard events via CDP.
func (s *Session) ExecuteKeyboard(ctx context.Context, keyValue, eventType string, modifiers []string, delayMs, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{"key": keyValue, "eventType": eventType, "modifiers": modifiers}}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	definition, err := resolveKeyDefinition(keyValue)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	normalizedEvent := normalizeKeyboardEventType(eventType)
	result.DebugContext["eventType"] = normalizedEvent

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	dispatch := func(event input.KeyType) error {
		params := buildKeyEventParams(event, definition, modifiers)
		action := chromedp.ActionFunc(func(ctx context.Context) error { return params.Do(ctx) })
		return chromedp.Run(timeoutCtx, action)
	}

	sendSequence := func() error {
		switch normalizedEvent {
		case "keydown":
			return dispatch(input.KeyDown)
		case "keyup":
			return dispatch(input.KeyUp)
		default:
			if err := dispatch(input.KeyDown); err != nil {
				return err
			}
			if definition.Text != "" {
				if err := dispatch(input.KeyChar); err != nil {
					return err
				}
			}
			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			return dispatch(input.KeyUp)
		}
	}

	if err := sendSequence(); err != nil {
		result.Error = fmt.Sprintf("keyboard event failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if normalizedEvent != "keypress" && delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}
