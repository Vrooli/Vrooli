package cdp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// StepResult represents the outcome of executing a workflow step
type StepResult struct {
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	DurationMs    int                    `json:"durationMs"`
	Screenshot    string                 `json:"screenshot,omitempty"` // base64 encoded
	URL           string                 `json:"url"`
	Title         string                 `json:"title"`
	ConsoleLogs   []ConsoleLog           `json:"consoleLogs,omitempty"`
	NetworkEvents []NetworkEvent         `json:"networkEvents,omitempty"`
	DebugContext  map[string]interface{} `json:"debugContext,omitempty"`
}

// ExecuteNavigate navigates to a URL
func (s *Session) ExecuteNavigate(ctx context.Context, url string, waitUntil string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Execute navigation
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.WaitReady("body", chromedp.ByQuery).Do(ctx)
		}),
	)

	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Additional wait if specified
	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	// Get current URL and title
	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Warn("Failed to get page info after navigate")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()

	return result, nil
}

// ExecuteClick clicks an element
func (s *Session) ExecuteClick(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Wait for element and click
	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	)

	if err != nil {
		result.Error = fmt.Sprintf("Click failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Additional wait if specified
	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	// Get current URL and title
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

// ExecuteWait waits for an element to appear
func (s *Session) ExecuteWait(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Wait for element
	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	)

	if err != nil {
		result.Error = fmt.Sprintf("Wait failed: selector %s not found", selector)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Additional wait if specified
	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

	// Get current URL and title
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

// ExecuteAssert checks element existence or text content
func (s *Session) ExecuteAssert(ctx context.Context, selector, mode, expectedValue string, caseSensitive bool, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: make(map[string]interface{}),
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Get current URL and all test IDs for debugging
	var currentURL string
	chromedp.Run(s.ctx, chromedp.Location(&currentURL))
	result.DebugContext["url"] = currentURL

	// Get all elements with data-testid for debugging
	var testIDs []string
	chromedp.Run(s.ctx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('[data-testid]')).map(el => el.getAttribute('data-testid'))`, &testIDs),
	)
	result.DebugContext["allTestIds"] = testIDs

	switch mode {
	case "exists":
		err := chromedp.Run(timeoutCtx, chromedp.WaitVisible(selector, chromedp.ByQuery))
		if err != nil {
			result.Error = fmt.Sprintf("Expected selector %q to exist", selector)
			result.DebugContext["exists"] = false
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
		result.DebugContext["exists"] = true

	case "not_exists":
		// Check if element exists (should not)
		var nodes []*cdp.Node
		err := chromedp.Run(timeoutCtx, chromedp.Nodes(selector, &nodes, chromedp.ByQuery))
		if err == nil && len(nodes) > 0 {
			result.Error = fmt.Sprintf("Expected selector %q to be absent but it exists", selector)
			result.DebugContext["exists"] = true
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, fmt.Errorf("element should not exist")
		}
		result.DebugContext["exists"] = false

	case "exists_or_not":
		// Always passes - just check and report
		var nodes []*cdp.Node
		chromedp.Run(s.ctx, chromedp.Nodes(selector, &nodes, chromedp.ByQuery))
		result.DebugContext["exists"] = len(nodes) > 0

	case "text_equals", "text_contains":
		var text string
		err := chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Text(selector, &text, chromedp.ByQuery),
		)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get text from %q: %v", selector, err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}

		// Normalize for comparison
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
		} else { // text_contains
			if !strings.Contains(compareText, compareExpected) {
				result.Error = fmt.Sprintf("Expected text to contain %q but got %q", expectedValue, text)
				result.DurationMs = int(time.Since(start).Milliseconds())
				return result, fmt.Errorf("text mismatch")
			}
		}

	default:
		return nil, fmt.Errorf("unsupported assert mode: %s", mode)
	}

	// Get current URL and title
	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()

	return result, nil
}

// ExecuteScreenshot captures a screenshot
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

	// Encode to base64
	result.Screenshot = base64.StdEncoding.EncodeToString(buf)

	// Get current URL and title
	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()

	return result, nil
}

// ExecuteType types text into an input field
func (s *Session) ExecuteType(ctx context.Context, selector, text string, clearFirst bool, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	actions := []chromedp.Action{
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	}

	if clearFirst {
		actions = append(actions, chromedp.Clear(selector, chromedp.ByQuery))
	}

	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))

	err := chromedp.Run(timeoutCtx, actions...)
	if err != nil {
		result.Error = fmt.Sprintf("Type failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Additional wait if specified
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

// ExecuteEvaluate executes arbitrary JavaScript
func (s *Session) ExecuteEvaluate(ctx context.Context, script string, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var evalResult interface{}
	err := chromedp.Run(timeoutCtx, chromedp.Evaluate(script, &evalResult))
	if err != nil {
		result.Error = fmt.Sprintf("Evaluate failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	result.DebugContext = map[string]interface{}{
		"result": evalResult,
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
