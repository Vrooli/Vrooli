package cdp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
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

const (
	defaultScrollAmountPixels     = 400
	defaultScrollVisibilityChecks = 12
	scrollVisibilityDelay         = 150 * time.Millisecond
)

type scrollOptions struct {
	scrollType     string
	selector       string
	targetSelector string
	direction      string
	behavior       string
	amount         int
	x              int
	y              int
	maxAttempts    int
	waitAfterMs    int
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

// ExecuteKeyboard dispatches low-level keyboard events via CDP.
func (s *Session) ExecuteKeyboard(ctx context.Context, keyValue, eventType string, modifiers []string, delayMs, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"key":       keyValue,
			"eventType": eventType,
			"modifiers": modifiers,
		},
	}

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
		action := chromedp.ActionFunc(func(ctx context.Context) error {
			return params.Do(ctx)
		})
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

// ExecuteHover hovers over an element to trigger pointer interactions.
func (s *Session) ExecuteHover(ctx context.Context, selector string, timeoutMs, waitAfterMs, steps, durationMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if steps < 1 {
		steps = 1
	} else if steps > 50 {
		steps = 50
	}
	if durationMs < 0 {
		durationMs = 0
	} else if durationMs > 10000 {
		durationMs = 10000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
		chromedp.Nodes(selector, &nodes, chromedp.ByQuery),
	)
	if err != nil {
		result.Error = fmt.Sprintf("hover target %s unavailable: %v", selector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if len(nodes) == 0 {
		err = fmt.Errorf("hover selector %s not found", selector)
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	nodeID := nodes[0].NodeID
	var box *dom.BoxModel
	err = chromedp.Run(timeoutCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			model, err := dom.GetBoxModel().WithNodeID(nodeID).Do(ctx)
			if err != nil {
				return err
			}
			box = model
			return nil
		}),
	)
	if err != nil {
		result.Error = fmt.Sprintf("failed to measure hover selector: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	centerX, centerY, err := centerFromBoxModel(box)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	err = chromedp.Run(timeoutCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return s.movePointerSmooth(ctx, centerX, centerY, steps, durationMs)
		}),
	)
	if err != nil {
		result.Error = fmt.Sprintf("hover movement failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("hover wait interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
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

// ExecuteScroll scrolls the page or targeted elements using the requested strategy.
func (s *Session) ExecuteScroll(ctx context.Context, opts scrollOptions, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"scrollType":     opts.scrollType,
			"selector":       opts.selector,
			"targetSelector": opts.targetSelector,
			"direction":      opts.direction,
			"amount":         opts.amount,
			"behavior":       opts.behavior,
			"x":              opts.x,
			"y":              opts.y,
			"maxAttempts":    opts.maxAttempts,
		},
	}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	behavior := scrollBehaviorOrAuto(opts.behavior)
	if opts.amount <= 0 {
		opts.amount = defaultScrollAmountPixels
	}
	if opts.direction == "" {
		opts.direction = "down"
	}
	result.DebugContext["behavior"] = behavior
	result.DebugContext["direction"] = opts.direction
	result.DebugContext["amount"] = opts.amount

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var actionErr error
	switch strings.ToLower(opts.scrollType) {
	case "element":
		actionErr = s.scrollElementIntoView(timeoutCtx, opts.selector, behavior)
	case "position":
		actionErr = s.scrollToPosition(timeoutCtx, opts.x, opts.y, behavior)
	case "untilvisible":
		actionErr = s.scrollUntilVisible(timeoutCtx, scrollOptions{
			scrollType:     opts.scrollType,
			selector:       opts.selector,
			targetSelector: opts.targetSelector,
			direction:      opts.direction,
			behavior:       behavior,
			amount:         opts.amount,
			maxAttempts:    opts.maxAttempts,
		})
	default:
		actionErr = s.scrollPage(timeoutCtx, opts.direction, opts.amount, behavior)
	}
	if actionErr != nil {
		result.Error = fmt.Sprintf("scroll failed: %v", actionErr)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, actionErr
	}

	if opts.waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(opts.waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("scroll wait interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
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

func (s *Session) scrollPage(ctx context.Context, direction string, amount int, behavior string) error {
	if direction == "" {
		direction = "down"
	}
	behavior = scrollBehaviorOrAuto(behavior)

	script := ""
	switch direction {
	case "top":
		script = fmt.Sprintf(`window.scrollTo({ top: 0, behavior: %q });`, behavior)
	case "bottom":
		script = fmt.Sprintf(`window.scrollTo({ top: document.body.scrollHeight, behavior: %q });`, behavior)
	default:
		dx, dy := scrollDeltaFromDirection(direction, amount)
		script = fmt.Sprintf(`window.scrollBy({ left: %d, top: %d, behavior: %q });`, dx, dy, behavior)
	}

	return chromedp.Run(ctx, chromedp.Evaluate(script, nil))
}

func (s *Session) scrollToPosition(ctx context.Context, x, y int, behavior string) error {
	script := fmt.Sprintf(`window.scrollTo({ left: %d, top: %d, behavior: %q });`, x, y, scrollBehaviorOrAuto(behavior))
	return chromedp.Run(ctx, chromedp.Evaluate(script, nil))
}

func (s *Session) scrollElementIntoView(ctx context.Context, selector, behavior string) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("scroll selector is required for element mode")
	}
	script := fmt.Sprintf(`(() => {
		const element = document.querySelector(%q);
		if (!element) {
			throw new Error('Selector %s not found');
		}
		element.scrollIntoView({ behavior: %q, block: 'center', inline: 'nearest' });
		return true;
	})()`, selector, selector, scrollBehaviorOrAuto(behavior))
	return chromedp.Run(ctx, chromedp.Evaluate(script, nil))
}

func (s *Session) scrollUntilVisible(ctx context.Context, opts scrollOptions) error {
	target := strings.TrimSpace(opts.targetSelector)
	if target == "" {
		return fmt.Errorf("scroll untilVisible requires target selector")
	}
	attempts := opts.maxAttempts
	if attempts <= 0 {
		attempts = defaultScrollVisibilityChecks
	}
	direction := strings.ToLower(opts.direction)
	if direction == "top" {
		direction = "up"
	} else if direction == "bottom" {
		direction = "down"
	}

	for i := 0; i < attempts; i++ {
		found, visible, err := checkSelectorVisibility(ctx, target)
		if err != nil {
			return err
		}
		if visible {
			return nil
		}
		if !found && i == attempts-1 {
			return fmt.Errorf("selector %s not found after %d scrolls", target, attempts)
		}
		if err := s.scrollPage(ctx, direction, opts.amount, opts.behavior); err != nil {
			return err
		}
		if err := waitWithContext(ctx, scrollVisibilityDelay); err != nil {
			return err
		}
	}

	return fmt.Errorf("selector %s not visible after %d scrolls", target, attempts)
}

func checkSelectorVisibility(ctx context.Context, selector string) (bool, bool, error) {
	var res struct {
		Found   bool `json:"found"`
		Visible bool `json:"visible"`
	}
	script := fmt.Sprintf(`(() => {
		const element = document.querySelector(%q);
		if (!element) {
			return { found: false, visible: false };
		}
		const rect = element.getBoundingClientRect();
		const vertVisible = rect.bottom > 0 && rect.top < (window.innerHeight || document.documentElement.clientHeight);
		const horizVisible = rect.right > 0 && rect.left < (window.innerWidth || document.documentElement.clientWidth);
		return { found: true, visible: vertVisible && horizVisible };
	})()`, selector)

	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &res)); err != nil {
		return false, false, err
	}
	return res.Found, res.Visible, nil
}

func scrollBehaviorOrAuto(behavior string) string {
	switch strings.ToLower(strings.TrimSpace(behavior)) {
	case "smooth":
		return "smooth"
	default:
		return "auto"
	}
}

func scrollDeltaFromDirection(direction string, amount int) (int, int) {
	switch strings.ToLower(direction) {
	case "up":
		return 0, -amount
	case "down":
		return 0, amount
	case "left":
		return -amount, 0
	case "right":
		return amount, 0
	default:
		return 0, amount
	}
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

type keyDefinition struct {
	Key     string
	Code    string
	Text    string
	KeyCode int
}

var specialKeyDefinitions = map[string]keyDefinition{
	"enter":      {Key: "Enter", Code: "Enter", KeyCode: 13},
	"return":     {Key: "Enter", Code: "Enter", KeyCode: 13},
	"tab":        {Key: "Tab", Code: "Tab", KeyCode: 9},
	"escape":     {Key: "Escape", Code: "Escape", KeyCode: 27},
	"esc":        {Key: "Escape", Code: "Escape", KeyCode: 27},
	"backspace":  {Key: "Backspace", Code: "Backspace", KeyCode: 8},
	"delete":     {Key: "Delete", Code: "Delete", KeyCode: 46},
	"space":      {Key: " ", Code: "Space", KeyCode: 32, Text: " "},
	"spacebar":   {Key: " ", Code: "Space", KeyCode: 32, Text: " "},
	"arrowup":    {Key: "ArrowUp", Code: "ArrowUp", KeyCode: 38},
	"arrowdown":  {Key: "ArrowDown", Code: "ArrowDown", KeyCode: 40},
	"arrowleft":  {Key: "ArrowLeft", Code: "ArrowLeft", KeyCode: 37},
	"arrowright": {Key: "ArrowRight", Code: "ArrowRight", KeyCode: 39},
	"home":       {Key: "Home", Code: "Home", KeyCode: 36},
	"end":        {Key: "End", Code: "End", KeyCode: 35},
	"pageup":     {Key: "PageUp", Code: "PageUp", KeyCode: 33},
	"pagedown":   {Key: "PageDown", Code: "PageDown", KeyCode: 34},
}

func normalizeKeyboardEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "keydown":
		return "keydown"
	case "keyup":
		return "keyup"
	default:
		return "keypress"
	}
}

func buildKeyEventParams(event input.KeyType, def keyDefinition, modifiers []string) *input.DispatchKeyEventParams {
	params := input.DispatchKeyEvent(event).
		WithKey(def.Key).
		WithCode(def.Code)
	if def.KeyCode > 0 {
		params = params.
			WithWindowsVirtualKeyCode(int64(def.KeyCode)).
			WithNativeVirtualKeyCode(int64(def.KeyCode))
	}
	if def.Text != "" {
		params = params.
			WithText(def.Text).
			WithUnmodifiedText(def.Text)
	}
	return applyModifierFlags(params, modifiers)
}

func applyModifierFlags(params *input.DispatchKeyEventParams, modifiers []string) *input.DispatchKeyEventParams {
	for _, mod := range modifiers {
		switch strings.ToLower(mod) {
		case "ctrl", "control":
			params.Modifiers |= input.ModifierCtrl
		case "shift":
			params.Modifiers |= input.ModifierShift
		case "alt", "option":
			params.Modifiers |= input.ModifierAlt
		case "meta", "cmd", "command", "super":
			params.Modifiers |= input.ModifierMeta
		}
	}
	return params
}

func resolveKeyDefinition(raw string) (keyDefinition, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return keyDefinition{}, fmt.Errorf("keyboard key is required")
	}
	lower := strings.ToLower(trimmed)
	if def, ok := specialKeyDefinitions[lower]; ok {
		return def, nil
	}
	runes := []rune(trimmed)
	if len(runes) == 1 {
		r := runes[0]
		code := deriveCodeFromRune(r)
		keyCode := int(unicode.ToUpper(r))
		if keyCode == 0 {
			keyCode = int(r)
		}
		text := string(r)
		return keyDefinition{
			Key:     string(r),
			Code:    code,
			Text:    text,
			KeyCode: keyCode,
		}, nil
	}
	if strings.HasPrefix(lower, "f") {
		if number, err := strconv.Atoi(strings.TrimPrefix(lower, "f")); err == nil && number >= 1 && number <= 24 {
			return keyDefinition{
				Key:     strings.ToUpper(fmt.Sprintf("F%d", number)),
				Code:    strings.ToUpper(fmt.Sprintf("F%d", number)),
				KeyCode: 111 + number,
			}, nil
		}
	}
	return keyDefinition{Key: trimmed, Code: trimmed}, nil
}

func deriveCodeFromRune(r rune) string {
	switch {
	case unicode.IsLetter(r):
		return fmt.Sprintf("Key%s", strings.ToUpper(string(r)))
	case unicode.IsDigit(r):
		return fmt.Sprintf("Digit%c", r)
	}
	switch r {
	case ' ':
		return "Space"
	case '`':
		return "Backquote"
	case '-':
		return "Minus"
	case '=':
		return "Equal"
	case '[':
		return "BracketLeft"
	case ']':
		return "BracketRight"
	case '\\':
		return "Backslash"
	case ';':
		return "Semicolon"
	case '\'':
		return "Quote"
	case ',':
		return "Comma"
	case '.':
		return "Period"
	case '/':
		return "Slash"
	default:
		return fmt.Sprintf("Key%s", strings.ToUpper(string(r)))
	}
}

func centerFromBoxModel(box *dom.BoxModel) (float64, float64, error) {
	if box == nil || len(box.Content) < 8 {
		return 0, 0, fmt.Errorf("element box model unavailable")
	}
	x := (box.Content[0] + box.Content[4]) / 2
	y := (box.Content[1] + box.Content[5]) / 2
	return x, y, nil
}

func (s *Session) movePointerSmooth(ctx context.Context, targetX, targetY float64, steps, durationMs int) error {
	startX, startY := s.pointerStartPosition(targetX, targetY)
	if steps < 1 {
		steps = 1
	}
	deltaX := (targetX - startX) / float64(steps)
	deltaY := (targetY - startY) / float64(steps)
	var stepDelay time.Duration
	if durationMs > 0 {
		stepDelay = time.Duration(durationMs) * time.Millisecond / time.Duration(steps)
	}
	for i := 1; i <= steps; i++ {
		nextX := startX + deltaX*float64(i)
		nextY := startY + deltaY*float64(i)
		if err := input.DispatchMouseEvent(input.MouseMoved, nextX, nextY).Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}
	s.setPointerPosition(targetX, targetY)
	return nil
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Session) pointerStartPosition(targetX, targetY float64) (float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pointerInitialized {
		s.pointerX = targetX
		s.pointerY = targetY
		s.pointerInitialized = true
		return targetX, targetY
	}
	return s.pointerX, s.pointerY
}

func (s *Session) setPointerPosition(x, y float64) {
	s.mu.Lock()
	s.pointerX = x
	s.pointerY = y
	s.pointerInitialized = true
	s.mu.Unlock()
}
