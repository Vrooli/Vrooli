package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// StepResult represents the outcome of executing a workflow step
type StepResult struct {
	Success            bool                     `json:"success"`
	Error              string                   `json:"error,omitempty"`
	DurationMs         int                      `json:"durationMs"`
	Screenshot         string                   `json:"screenshot,omitempty"` // base64 encoded
	URL                string                   `json:"url"`
	Title              string                   `json:"title"`
	ConsoleLogs        []ConsoleLog             `json:"consoleLogs,omitempty"`
	NetworkEvents      []NetworkEvent           `json:"networkEvents,omitempty"`
	DebugContext       map[string]interface{}   `json:"debugContext,omitempty"`
	ElementBoundingBox *runtime.BoundingBox     `json:"elementBoundingBox,omitempty"`
	ExtractedData      any                      `json:"extractedData,omitempty"`
	Condition          *runtime.ConditionResult `json:"conditionResult,omitempty"`
}

const (
	defaultScrollAmountPixels      = 400
	defaultScrollVisibilityChecks  = 12
	scrollVisibilityDelay          = 150 * time.Millisecond
	defaultVariableTimeoutMs       = 30000
	defaultConditionalTimeoutMs    = 10000
	defaultConditionalPollInterval = 250
	rotateOrientationPortrait      = "portrait"
	rotateOrientationLandscape     = "landscape"
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

type dragDropOptions struct {
	sourceSelector string
	targetSelector string
	holdMs         int
	steps          int
	durationMs     int
	offsetX        int
	offsetY        int
	waitAfterMs    int
}

type gestureOptions struct {
	Type             string
	Direction        string
	StartX           float64
	StartY           float64
	EndX             float64
	EndY             float64
	Distance         int
	Scale            float64
	DurationMs       int
	HoldMs           int
	Steps            int
	Selector         string
	TimeoutMs        int
	HasExplicitStart bool
	HasExplicitEnd   bool
	HasStartX        bool
	HasStartY        bool
	HasEndX          bool
	HasEndY          bool
}

type selectEvalPayload struct {
	Selector string   `json:"selector"`
	Mode     string   `json:"mode"`
	Value    string   `json:"value"`
	Text     string   `json:"text"`
	Index    int      `json:"index"`
	Values   []string `json:"values"`
	Multi    bool     `json:"multi"`
}

type selectEvalResult struct {
	SelectedValues []string `json:"selectedValues"`
	SelectedTexts  []string `json:"selectedTexts"`
	Multiple       bool     `json:"multiple"`
}

type tabSwitchOptions struct {
	SwitchBy   string
	Index      int
	TitleMatch string
	URLMatch   string
	WaitForNew bool
	TimeoutMs  int
	CloseOld   bool
}

type conditionalOptions struct {
	Type           string
	Expression     string
	Selector       string
	Variable       string
	Operator       string
	Value          any
	Negate         bool
	TimeoutMs      int
	PollIntervalMs int
	Variables      map[string]any
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

// ExecuteUploadFile selects files for an <input type="file"> element.
func (s *Session) ExecuteUploadFile(ctx context.Context, selector string, files []string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"selector": selector,
			"files":    files,
		},
	}

	if len(files) == 0 {
		err := fmt.Errorf("no files supplied for upload")
		result.Error = err.Error()
		return result, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.SetUploadFiles(selector, files, chromedp.ByQuery),
	)
	if err != nil {
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

	box, err := s.resolveElementBox(timeoutCtx, selector)
	if err != nil {
		result.Error = fmt.Sprintf("hover target %s unavailable: %v", selector, err)
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

// ExecuteDragAndDrop simulates HTML5 drag interactions, including pointer motion and drop events.
// [REQ:BAS-NODE-DRAG-DROP]
func (s *Session) ExecuteDragAndDrop(ctx context.Context, opts dragDropOptions, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"sourceSelector": opts.sourceSelector,
			"targetSelector": opts.targetSelector,
			"holdMs":         opts.holdMs,
			"steps":          opts.steps,
			"durationMs":     opts.durationMs,
			"offsetX":        opts.offsetX,
			"offsetY":        opts.offsetY,
		},
	}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if opts.steps < 1 {
		opts.steps = 1
	}
	if opts.durationMs < 0 {
		opts.durationMs = 0
	}
	if opts.holdMs < 0 {
		opts.holdMs = 0
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	sourceBox, err := s.resolveElementBox(timeoutCtx, opts.sourceSelector)
	if err != nil {
		result.Error = fmt.Sprintf("drag source %s unavailable: %v", opts.sourceSelector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	targetBox, err := s.resolveElementBox(timeoutCtx, opts.targetSelector)
	if err != nil {
		result.Error = fmt.Sprintf("drag target %s unavailable: %v", opts.targetSelector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	sourceX, sourceY, err := centerFromBoxModel(sourceBox)
	if err != nil {
		result.Error = fmt.Sprintf("unable to compute drag source center: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	targetX, targetY, err := centerFromBoxModel(targetBox)
	if err != nil {
		result.Error = fmt.Sprintf("unable to compute drag target center: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	dropX := targetX + float64(opts.offsetX)
	dropY := targetY + float64(opts.offsetY)
	result.DebugContext["dropX"] = dropX
	result.DebugContext["dropY"] = dropY

	approachSteps := opts.steps / 3
	if approachSteps < 1 {
		approachSteps = 1
	}
	approachDuration := opts.durationMs / 3
	if err := s.movePointerSmooth(timeoutCtx, sourceX, sourceY, approachSteps, approachDuration); err != nil {
		result.Error = fmt.Sprintf("failed to move pointer to source: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if err := input.DispatchMouseEvent(input.MousePressed, sourceX, sourceY).
		WithButton(input.Left).
		WithButtons(1).
		WithClickCount(1).
		Do(timeoutCtx); err != nil {
		result.Error = fmt.Sprintf("failed to press mouse: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if opts.holdMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(opts.holdMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("drag hold interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}

	if err := s.movePointerSmoothWithButtons(timeoutCtx, dropX, dropY, opts.steps, opts.durationMs, 1); err != nil {
		result.Error = fmt.Sprintf("failed to drag pointer: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if err := input.DispatchMouseEvent(input.MouseReleased, dropX, dropY).
		WithButton(input.Left).
		WithButtons(0).
		WithClickCount(1).
		Do(timeoutCtx); err != nil {
		result.Error = fmt.Sprintf("failed to release mouse: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	dragEventScript := fmt.Sprintf(`(() => {
	const sourceSelector = %s;
	const targetSelector = %s;
	const dropPoint = { x: %f, y: %f };
	const source = document.querySelector(sourceSelector);
	const target = document.querySelector(targetSelector);
	if (!source || !target) {
		throw new Error('Drag selectors missing during HTML5 dispatch');
	}
	const dataTransfer = typeof DataTransfer === 'function' ? new DataTransfer() : {
		_data: {},
		setData(type, value) { this._data[type] = String(value); },
		getData(type) { return this._data[type] ?? ''; },
		dropEffect: 'move',
		effectAllowed: 'all',
	};
	const fire = (node, type) => {
		const eventInit = { bubbles: true, cancelable: true, dataTransfer, clientX: dropPoint.x, clientY: dropPoint.y };
		let event;
		try {
			event = new DragEvent(type, eventInit);
		} catch (err) {
			event = document.createEvent('CustomEvent');
			event.initCustomEvent(type, true, true, null);
			Object.assign(event, eventInit);
		}
		node.dispatchEvent(event);
	};
	fire(source, 'dragstart');
	fire(source, 'drag');
	fire(target, 'dragenter');
	fire(target, 'dragover');
	fire(target, 'drop');
	fire(source, 'dragend');
	return true;
})()`, strconv.Quote(opts.sourceSelector), strconv.Quote(opts.targetSelector), dropX, dropY)

	if err := chromedp.Run(timeoutCtx, chromedp.Evaluate(dragEventScript, nil)); err != nil {
		result.Error = fmt.Sprintf("failed to dispatch HTML5 drag events: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	result.DebugContext["html5Dispatch"] = true

	if targetBox != nil {
		result.ElementBoundingBox = &runtime.BoundingBox{
			X:      dropX - float64(targetBox.Width)/2,
			Y:      dropY - float64(targetBox.Height)/2,
			Width:  float64(targetBox.Width),
			Height: float64(targetBox.Height),
		}
	}

	if opts.waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(opts.waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("post-drag wait interrupted: %v", err)
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

// ExecuteGesture simulates common mobile/touch gestures. [REQ:BAS-NODE-GESTURE-MOBILE]
func (s *Session) ExecuteGesture(ctx context.Context, opts gestureOptions) (*StepResult, error) {
	start := time.Now()
	gestureType := strings.ToLower(strings.TrimSpace(opts.Type))
	if gestureType == "" {
		gestureType = "tap"
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = defaultGestureTimeoutMs
	}
	if opts.DurationMs <= 0 {
		opts.DurationMs = defaultGestureDurationMs
	}
	if opts.Steps <= 0 {
		opts.Steps = defaultGestureSteps
	}
	if opts.Distance <= 0 {
		opts.Distance = defaultGestureDistance
	}
	if opts.Scale <= 0 {
		opts.Scale = 1
	}
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"type":      gestureType,
			"direction": opts.Direction,
			"selector":  strings.TrimSpace(opts.Selector),
			"distance":  opts.Distance,
			"scale":     opts.Scale,
			"steps":     opts.Steps,
			"duration":  opts.DurationMs,
		},
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutMs)*time.Millisecond)
	defer cancel()

	anchorX, anchorY, err := s.resolveGestureAnchor(timeoutCtx, opts)
	if err != nil {
		result.Error = fmt.Sprintf("failed to resolve gesture anchor: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	result.DebugContext["anchorX"] = anchorX
	result.DebugContext["anchorY"] = anchorY

	var execErr error
	switch gestureType {
	case "swipe":
		startX, startY, endX, endY := s.resolveSwipePath(anchorX, anchorY, opts)
		result.DebugContext["startX"] = startX
		result.DebugContext["startY"] = startY
		result.DebugContext["endX"] = endX
		result.DebugContext["endY"] = endY
		execErr = s.dispatchSwipe(timeoutCtx, startX, startY, endX, endY, opts.DurationMs, opts.Steps)
	case "pinch":
		execErr = s.dispatchPinch(timeoutCtx, anchorX, anchorY, opts)
	case "doubletap", "double_tap":
		execErr = s.dispatchDoubleTap(timeoutCtx, anchorX, anchorY, opts.DurationMs)
	case "longpress", "long_press", "long-press", "press":
		execErr = s.dispatchLongPress(timeoutCtx, anchorX, anchorY, opts.HoldMs)
	default:
		execErr = s.dispatchTap(timeoutCtx, anchorX, anchorY, opts.DurationMs)
	}
	if execErr != nil {
		result.Error = execErr.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, execErr
	}

	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Debug("failed to capture page metadata after gesture")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteFocus sets focus on a specific element to trigger focus/keyboard events.
// [REQ:BAS-NODE-FOCUS-INPUT]
func (s *Session) ExecuteFocus(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
		chromedp.Focus(selector, chromedp.ByQuery),
	)
	if err != nil {
		result.Error = fmt.Sprintf("focus failed: %v", err)
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

// ExecuteBlur blurs an element to trigger validation or blur handlers.
// [REQ:BAS-NODE-BLUR-VALIDATION]
func (s *Session) ExecuteBlur(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	script := fmt.Sprintf(`(() => {
		const selector = %s;
		const element = document.querySelector(selector);
		if (!element) {
			throw new Error('Element not found for selector ' + selector);
		}
		if (typeof element.focus === 'function') {
			try { element.focus(); } catch (err) {}
		}
		if (typeof element.blur === 'function') {
			element.blur();
		} else if (document.activeElement && typeof document.activeElement.blur === 'function') {
			document.activeElement.blur();
		}
		return true;
	})()`, strconv.Quote(selector))

	var evalResult interface{}
	err := chromedp.Run(timeoutCtx,
		chromedp.WaitReady(selector, chromedp.ByQuery),
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
		chromedp.Evaluate(script, &evalResult),
	)
	if err != nil {
		result.Error = fmt.Sprintf("blur failed: %v", err)
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

// ExecuteRotate updates the viewport orientation for mobile validation flows.
// [REQ:BAS-NODE-ROTATE-MOBILE]
func (s *Session) ExecuteRotate(ctx context.Context, orientation string, angle int, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"orientation": orientation,
			"angle":       angle,
		},
	}

	if orientation == "" {
		orientation = rotateOrientationPortrait
	}
	width, height := resolveViewportForOrientation(s.viewportWidth, s.viewportHeight, orientation)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	orientationType := deriveScreenOrientationType(orientation, angle)
	cmd := emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1, true).
		WithScreenOrientation(&emulation.ScreenOrientation{
			Type:  emulation.OrientationType(orientationType),
			Angle: int64(angle),
		})

	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
		return cmd.Do(runCtx)
	}))
	if err != nil {
		result.Error = fmt.Sprintf("rotate failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	s.SetViewport(width, height)
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
	result.DebugContext["viewportWidth"] = width
	result.DebugContext["viewportHeight"] = height
	result.DebugContext["orientationType"] = orientationType

	return result, nil
}

// ExecuteSelect updates the value of a native select element (single or multi-select).
func (s *Session) ExecuteSelect(ctx context.Context, selector, mode, value, text string, index int, values []string, multi bool, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"selector": selector,
			"mode":     mode,
			"multi":    multi,
			"value":    value,
			"text":     text,
			"index":    index,
			"values":   values,
		},
	}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var nodes []*cdp.Node
	if err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
		chromedp.Nodes(selector, &nodes, chromedp.ByQuery),
	); err != nil {
		result.Error = fmt.Sprintf("select target %s unavailable: %v", selector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if len(nodes) == 0 {
		err := fmt.Errorf("select target %s not found", selector)
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	payload := selectEvalPayload{
		Selector: selector,
		Mode:     mode,
		Value:    value,
		Text:     text,
		Index:    index,
		Values:   values,
		Multi:    multi,
	}

	scriptBytes, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal select payload: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	script := fmt.Sprintf(`(() => {
	const config = %s;
	const normalize = (value) => (value ?? '').toString().trim().toLowerCase();
	const element = document.querySelector(config.selector);
	if (!element) {
		throw new Error('Select node: selector ' + config.selector + ' not found');
	}
	const tag = (element.tagName || '').toLowerCase();
	const optionList = tag === 'select' && element.options ? Array.from(element.options) : Array.from(element.querySelectorAll('option'));
	if (!optionList.length) {
		throw new Error('Select node: no <option> elements found for ' + config.selector);
	}
	if (config.multi) {
		if (tag === 'select') {
			if (element.multiple !== true) {
				throw new Error('Select node: element is not configured for multiple selections');
			}
		} else {
			throw new Error('Select node: multi-select requires a <select multiple> element');
		}
	}
	const emit = () => ({
		selectedValues: optionList.filter((opt) => opt.selected).map((opt) => opt.value ?? ''),
		selectedTexts: optionList.filter((opt) => opt.selected).map((opt) => opt.text ?? ''),
		multiple: Boolean(element.multiple),
	});
	if (config.multi) {
		const targets = Array.isArray(config.values) ? config.values.map(normalize).filter(Boolean) : [];
		if (!targets.length) {
			throw new Error('Select node: multi-select requires at least one value');
		}
		optionList.forEach((option) => {
			const compare = config.mode === 'text' ? normalize(option.text) : normalize(option.value);
			const match = config.mode === 'text'
				? targets.some((target) => compare.includes(target))
				: targets.includes(compare);
			option.selected = match;
		});
		const firstSelected = optionList.findIndex((opt) => opt.selected);
		if (tag === 'select' && firstSelected >= 0) {
			element.selectedIndex = firstSelected;
		}
		element.dispatchEvent(new Event('input', { bubbles: true }));
		element.dispatchEvent(new Event('change', { bubbles: true }));
		return emit();
	}
	let targetIndex = -1;
	if (config.mode === 'index') {
		targetIndex = config.index;
		if (typeof targetIndex !== 'number' || targetIndex < 0 || targetIndex >= optionList.length) {
			throw new Error('Select node: index ' + config.index + ' is out of range');
		}
	} else {
		const needle = normalize(config.mode === 'text' ? config.text : config.value);
		if (!needle) {
			throw new Error('Select node: selection value missing for mode ' + config.mode);
		}
		targetIndex = optionList.findIndex((option) => {
			const compare = config.mode === 'text' ? normalize(option.text) : normalize(option.value);
			return config.mode === 'text' ? compare.includes(needle) : compare === needle;
		});
		if (targetIndex === -1) {
			throw new Error('Select node: no option matched ' + needle);
		}
	}
	optionList.forEach((option, idx) => {
		option.selected = idx === targetIndex;
	});
	if (tag === 'select') {
		element.selectedIndex = targetIndex;
	}
	element.dispatchEvent(new Event('input', { bubbles: true }));
	element.dispatchEvent(new Event('change', { bubbles: true }));
	return emit();
})()`, string(scriptBytes))

	var evalResult selectEvalResult
	if err := chromedp.Run(timeoutCtx, chromedp.Evaluate(script, &evalResult)); err != nil {
		result.Error = fmt.Sprintf("select script failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("select wait interrupted: %v", err)
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
	result.DebugContext["selectedValues"] = evalResult.SelectedValues
	result.DebugContext["selectedTexts"] = evalResult.SelectedTexts
	result.DebugContext["elementMultiple"] = evalResult.Multiple

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

func resolveViewportForOrientation(width, height int, orientation string) (int, int) {
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}
	switch strings.ToLower(orientation) {
	case rotateOrientationPortrait:
		if width > height {
			return height, width
		}
	case rotateOrientationLandscape:
		if height > width {
			return height, width
		}
	}
	return width, height
}

func deriveScreenOrientationType(orientation string, angle int) string {
	switch strings.ToLower(orientation) {
	case rotateOrientationLandscape:
		if angle == 270 {
			return "landscapeSecondary"
		}
		return "landscapePrimary"
	default:
		if angle == 180 {
			return "portraitSecondary"
		}
		return "portraitPrimary"
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
	result.ExtractedData = evalResult

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()

	return result, nil
}

// ExecuteExtract queries DOM elements and returns their data based on the requested strategy.
func (s *Session) ExecuteExtract(ctx context.Context, selector, extractType, attribute string, allMatches bool, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}
	if strings.TrimSpace(selector) == "" {
		result.Error = "selector is required"
		return result, fmt.Errorf("extract selector is required")
	}
	if timeoutMs <= 0 {
		timeoutMs = defaultVariableTimeoutMs
	}
	script, err := buildExtractionScript(selector, extractType, attribute, allMatches)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	payload := map[string]any{}
	if err := chromedp.Run(timeoutCtx, chromedp.Evaluate(script, &payload)); err != nil {
		result.Error = fmt.Sprintf("extract failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if errText, ok := payload["error"].(string); ok && errText != "" {
		result.Error = errText
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, fmt.Errorf("%s", errText)
	}
	if multiple, ok := payload["multiple"].(bool); ok && multiple {
		result.ExtractedData = payload["values"]
	} else {
		result.ExtractedData = payload["value"]
		if bbox, ok := payload["boundingBox"].(map[string]any); ok {
			if parsed := parseBoundingBox(bbox); parsed != nil {
				result.ElementBoundingBox = parsed
			}
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

// ExecuteTabSwitch switches the active browser target according to the requested criteria.
func (s *Session) ExecuteTabSwitch(ctx context.Context, opts tabSwitchOptions) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{
		DebugContext: map[string]interface{}{
			"switchBy":   opts.SwitchBy,
			"waitNew":    opts.WaitForNew,
			"closeOld":   opts.CloseOld,
			"titleMatch": opts.TitleMatch,
			"urlMatch":   opts.URLMatch,
			"index":      opts.Index,
		},
	}

	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 30000
	}
	if err := s.refreshTabInventory(ctx); err != nil {
		s.log.WithError(err).Warn("failed to refresh tab inventory before switch")
	}
	if opts.WaitForNew {
		known := s.currentTabSet()
		waitCtx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutMs)*time.Millisecond)
		defer cancel()
		if _, err := s.waitForNewTab(waitCtx, known, time.Duration(opts.TimeoutMs)*time.Millisecond); err != nil {
			result.Error = err.Error()
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}
	if err := s.refreshTabInventory(ctx); err != nil {
		s.log.WithError(err).Warn("failed to refresh tab inventory after wait")
	}
	tabs := s.snapshotTabs()
	if len(tabs) == 0 {
		err := fmt.Errorf("no browser tabs are available to switch to")
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	selection, err := pickTabTarget(tabs, opts)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if err := s.switchToTarget(selection.ID, opts.CloseOld); err != nil {
		result.Error = fmt.Sprintf("failed to switch tab: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	result.Success = true
	result.URL = selection.URL
	result.Title = selection.Title
	result.DebugContext["targetId"] = string(selection.ID)
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteConditional evaluates branching conditions without mutating page state.
func (s *Session) ExecuteConditional(ctx context.Context, opts conditionalOptions) (*StepResult, error) {
	start := time.Now()
	conditionType := strings.ToLower(strings.TrimSpace(opts.Type))
	if conditionType == "" {
		conditionType = "expression"
	}
	timeoutMs := opts.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultConditionalTimeoutMs
	}
	pollInterval := opts.PollIntervalMs
	if pollInterval <= 0 {
		pollInterval = defaultConditionalPollInterval
	}
	debug := map[string]interface{}{
		"conditionType": conditionType,
		"selector":      strings.TrimSpace(opts.Selector),
		"variable":      strings.TrimSpace(opts.Variable),
		"operator":      normalizeConditionOperatorName(opts.Operator),
		"negate":        opts.Negate,
	}

	result := &StepResult{
		Success:      true,
		DebugContext: debug,
	}

	var (
		outcome bool
		actual  any
		evalErr error
	)

	switch conditionType {
	case "element", "selector":
		trimmed := strings.TrimSpace(opts.Selector)
		outcome, evalErr = s.evaluateElementCondition(ctx, trimmed, timeoutMs, pollInterval)
		actual = outcome
	case "variable":
		outcome, actual, evalErr = evaluateVariableCondition(strings.TrimSpace(opts.Variable), normalizeConditionOperatorName(opts.Operator), opts.Value, opts.Variables)
	default:
		outcome, actual, evalErr = s.evaluateExpressionCondition(ctx, strings.TrimSpace(opts.Expression), timeoutMs)
	}

	if evalErr != nil {
		result.Success = false
		result.Error = evalErr.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
		return result, evalErr
	}

	if opts.Negate {
		outcome = !outcome
	}
	result.DebugContext["conditionOutcome"] = outcome
	result.Condition = &runtime.ConditionResult{
		Type:       conditionType,
		Outcome:    outcome,
		Negated:    opts.Negate,
		Operator:   normalizeConditionOperatorName(opts.Operator),
		Variable:   strings.TrimSpace(opts.Variable),
		Selector:   strings.TrimSpace(opts.Selector),
		Expression: strings.TrimSpace(opts.Expression),
		Expected:   opts.Value,
		Actual:     actual,
	}
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteSetVariable resolves a setVariable instruction by delegating to the correct primitive.
func (s *Session) ExecuteSetVariable(ctx context.Context, instruction runtime.Instruction) (*StepResult, error) {
	source := strings.ToLower(strings.TrimSpace(instruction.Params.VariableSource))
	timeout := instruction.Params.TimeoutMs
	if timeout <= 0 {
		timeout = defaultVariableTimeoutMs
	}
	switch source {
	case "static":
		return &StepResult{Success: true, ExtractedData: instruction.Params.VariableValue}, nil
	case "expression":
		return s.ExecuteEvaluate(ctx, instruction.Params.Expression, timeout)
	case "extract":
		return s.ExecuteExtract(ctx, instruction.Params.Selector, instruction.Params.ExtractType, instruction.Params.Attribute, boolFromPointer(instruction.Params.AllMatches), timeout)
	default:
		return nil, fmt.Errorf("unsupported variable source %s", source)
	}
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

func buildExtractionScript(selector, extractType, attribute string, allMatches bool) (string, error) {
	selectorJSON, err := json.Marshal(selector)
	if err != nil {
		return "", err
	}
	modeJSON, err := json.Marshal(strings.ToLower(strings.TrimSpace(extractType)))
	if err != nil {
		return "", err
	}
	attributeJSON, err := json.Marshal(attribute)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`(() => {
	const selector = %s;
	const mode = %s || 'text';
	const attribute = %s || '';
	const allMatches = %t;
	const nodes = Array.from(document.querySelectorAll(selector));
	if (!nodes.length) {
		return { error: 'No elements matched selector ' + selector };
	}
	const extractValue = (element) => {
		switch (mode) {
			case 'html':
				return element.innerHTML;
			case 'value':
				return element.value ?? element.getAttribute('value');
			case 'attribute':
				return attribute ? element.getAttribute(attribute) : null;
			default:
				return element.textContent;
		}
	};
	if (allMatches) {
		return { multiple: true, values: nodes.map(extractValue) };
	}
	const first = nodes[0];
	const rect = first.getBoundingClientRect();
	return {
		value: extractValue(first),
		boundingBox: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
	};
})()`, string(selectorJSON), string(modeJSON), string(attributeJSON), allMatches)
	return script, nil
}

func parseBoundingBox(payload map[string]any) *runtime.BoundingBox {
	if payload == nil {
		return nil
	}
	box := &runtime.BoundingBox{}
	if x, ok := payload["x"].(float64); ok {
		box.X = x
	}
	if y, ok := payload["y"].(float64); ok {
		box.Y = y
	}
	if width, ok := payload["width"].(float64); ok {
		box.Width = width
	}
	if height, ok := payload["height"].(float64); ok {
		box.Height = height
	}
	return box
}

func boolFromPointer(value *bool) bool {
	return value != nil && *value
}

func (s *Session) evaluateExpressionCondition(ctx context.Context, expression string, timeoutMs int) (bool, any, error) {
	trimmed := strings.TrimSpace(expression)
	if trimmed == "" {
		return false, nil, fmt.Errorf("conditional expression is required")
	}
	var evaluation struct {
		Success bool   `json:"success"`
		Value   any    `json:"value"`
		Error   string `json:"error"`
	}
	script := fmt.Sprintf(`(function(expr){
	try {
		const fn = new Function('return (' + expr + ');');
		const value = fn();
		return { success: !!value, value };
	} catch (err) {
		return { success: false, error: String(err) };
	}
})(%s)`, strconv.Quote(trimmed))
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	if err := chromedp.Run(ctxWithTimeout, chromedp.Evaluate(script, &evaluation)); err != nil {
		return false, nil, err
	}
	if evaluation.Error != "" {
		return false, nil, fmt.Errorf("expression evaluation failed: %s", evaluation.Error)
	}
	return evaluation.Success, evaluation.Value, nil
}

func (s *Session) evaluateElementCondition(ctx context.Context, selector string, timeoutMs, pollInterval int) (bool, error) {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return false, fmt.Errorf("conditional selector is required")
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	interval := time.Duration(pollInterval) * time.Millisecond
	for {
		var exists bool
		err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf("!!document.querySelector(%s)", strconv.Quote(trimmed)), &exists))
		if err == nil && exists {
			return true, nil
		}
		if err != nil && ctx.Err() != nil {
			return false, err
		}
		if time.Now().After(deadline) {
			return exists, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func evaluateVariableCondition(variableName, operator string, expected any, variables map[string]any) (bool, any, error) {
	trimmed := strings.TrimSpace(variableName)
	if trimmed == "" {
		return false, nil, fmt.Errorf("conditional variable name is required")
	}
	if variables == nil {
		return false, nil, fmt.Errorf("execution context is empty")
	}
	actual, ok := variables[trimmed]
	if !ok {
		return false, nil, fmt.Errorf("variable %s is not defined", trimmed)
	}
	match, err := compareConditionValues(actual, expected, operator)
	return match, actual, err
}

func normalizeConditionOperatorName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "equals", "eq", "==":
		return "equals"
	case "not_equals", "!=", "neq":
		return "not_equals"
	case "contains", "includes":
		return "contains"
	case "starts_with", "prefix":
		return "starts_with"
	case "ends_with", "suffix":
		return "ends_with"
	case "gt", ">", "greater_than":
		return "gt"
	case "lt", "<", "less_than":
		return "lt"
	case "gte", ">=", "greater_than_or_equal":
		return "gte"
	case "lte", "<=", "less_than_or_equal":
		return "lte"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func compareConditionValues(actual, expected any, operator string) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	switch operator {
	case "equals", "", "eq":
		return actualStr == expectedStr, nil
	case "not_equals", "neq":
		return actualStr != expectedStr, nil
	case "contains":
		return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "gt", "gte", "lt", "lte":
		actualNum, ok := toFloat64(actual)
		if !ok {
			return false, fmt.Errorf("actual value is not numeric")
		}
		expectedNum, ok := toFloat64(expected)
		if !ok {
			return false, fmt.Errorf("expected value is not numeric")
		}
		switch operator {
		case "gt":
			return actualNum > expectedNum, nil
		case "gte":
			return actualNum >= expectedNum, nil
		case "lt":
			return actualNum < expectedNum, nil
		case "lte":
			return actualNum <= expectedNum, nil
		}
	}
	return false, fmt.Errorf("unsupported operator %s", operator)
}

func toFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return parsed, true
		}
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return parsed, true
		}
	}
	return 0, false
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
	return s.movePointerSmoothWithButtons(ctx, targetX, targetY, steps, durationMs, 0)
}

func (s *Session) movePointerSmoothWithButtons(ctx context.Context, targetX, targetY float64, steps, durationMs int, buttons int64) error {
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
		event := input.DispatchMouseEvent(input.MouseMoved, nextX, nextY)
		if buttons > 0 {
			event = event.WithButtons(buttons)
		}
		if err := event.Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}
	s.setPointerPosition(targetX, targetY)
	return nil
}

func pickTabTarget(tabs []tabRecord, opts tabSwitchOptions) (*tabRecord, error) {
	if len(tabs) == 0 {
		return nil, fmt.Errorf("no tabs available")
	}
	mode := strings.ToLower(strings.TrimSpace(opts.SwitchBy))
	switch mode {
	case "index":
		if opts.Index < 0 || opts.Index >= len(tabs) {
			return nil, fmt.Errorf("tab index %d out of range", opts.Index)
		}
		return &tabs[opts.Index], nil
	case "title":
		pattern := strings.TrimSpace(opts.TitleMatch)
		if pattern == "" {
			return nil, fmt.Errorf("title pattern is required for switchBy=title")
		}
		for i := range tabs {
			if matchesPattern(tabs[i].Title, pattern) {
				return &tabs[i], nil
			}
		}
		return nil, fmt.Errorf("no tab matched title pattern %s", pattern)
	case "url":
		pattern := strings.TrimSpace(opts.URLMatch)
		if pattern == "" {
			return nil, fmt.Errorf("url pattern is required for switchBy=url")
		}
		for i := range tabs {
			if matchesPattern(tabs[i].URL, pattern) {
				return &tabs[i], nil
			}
		}
		return nil, fmt.Errorf("no tab matched url pattern %s", pattern)
	case "oldest":
		return &tabs[0], nil
	case "newest", "":
		return &tabs[len(tabs)-1], nil
	default:
		return &tabs[len(tabs)-1], nil
	}
}

func matchesPattern(value, pattern string) bool {
	if value == "" || pattern == "" {
		return false
	}
	if re, err := regexp.Compile(pattern); err == nil {
		return re.MatchString(value)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}

func (s *Session) resolveElementBox(ctx context.Context, selector string) (*dom.BoxModel, error) {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return nil, fmt.Errorf("selector is required")
	}
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(trimmed, chromedp.ByQuery),
		chromedp.ScrollIntoView(trimmed, chromedp.ByQuery),
		chromedp.Nodes(trimmed, &nodes, chromedp.ByQuery),
	); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("selector %s not found", trimmed)
	}
	var box *dom.BoxModel
	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			model, err := dom.GetBoxModel().WithNodeID(nodes[0].NodeID).Do(ctx)
			if err != nil {
				return err
			}
			box = model
			return nil
		}),
	); err != nil {
		return nil, err
	}
	return box, nil
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

func (s *Session) resolveGestureAnchor(ctx context.Context, opts gestureOptions) (float64, float64, error) {
	trimmed := strings.TrimSpace(opts.Selector)
	if trimmed != "" {
		box, err := s.resolveElementBox(ctx, trimmed)
		if err != nil {
			return 0, 0, err
		}
		x, y, err := centerFromBoxModel(box)
		if err != nil {
			return 0, 0, err
		}
		return x, y, nil
	}
	centerX, centerY := s.viewportCenter()
	if opts.HasExplicitStart {
		if opts.HasStartX {
			centerX = opts.StartX
		}
		if opts.HasStartY {
			centerY = opts.StartY
		}
	}
	return s.clampViewportX(centerX), s.clampViewportY(centerY), nil
}

func (s *Session) resolveSwipePath(anchorX, anchorY float64, opts gestureOptions) (float64, float64, float64, float64) {
	startX := anchorX
	startY := anchorY
	if opts.HasExplicitStart {
		if opts.HasStartX {
			startX = opts.StartX
		}
		if opts.HasStartY {
			startY = opts.StartY
		}
	}
	startX = s.clampViewportX(startX)
	startY = s.clampViewportY(startY)

	endX := startX
	endY := startY
	if opts.HasExplicitEnd {
		if opts.HasEndX {
			endX = opts.EndX
		}
		if opts.HasEndY {
			endY = opts.EndY
		}
	} else {
		delta := float64(opts.Distance)
		if delta <= 0 {
			delta = float64(defaultGestureDistance)
		}
		switch strings.ToLower(strings.TrimSpace(opts.Direction)) {
		case "up":
			endY = startY - delta
		case "left":
			endX = startX - delta
		case "right":
			endX = startX + delta
		default:
			endY = startY + delta
		}
	}
	return startX, startY, s.clampViewportX(endX), s.clampViewportY(endY)
}

func (s *Session) dispatchTap(ctx context.Context, x, y float64, holdMs int) error {
	if holdMs <= 0 {
		holdMs = minGestureDurationMs
	}
	point := s.touchPoint(1, x, y)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}
	if err := waitWithContext(ctx, time.Duration(holdMs)*time.Millisecond); err != nil {
		return err
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchDoubleTap(ctx context.Context, x, y float64, durationMs int) error {
	if err := s.dispatchTap(ctx, x, y, durationMs); err != nil {
		return err
	}
	if err := waitWithContext(ctx, 120*time.Millisecond); err != nil {
		return err
	}
	return s.dispatchTap(ctx, x, y, durationMs)
}

func (s *Session) dispatchLongPress(ctx context.Context, x, y float64, holdMs int) error {
	if holdMs < 600 {
		holdMs = 600
	}
	point := s.touchPoint(1, x, y)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}
	if err := waitWithContext(ctx, time.Duration(holdMs)*time.Millisecond); err != nil {
		return err
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchSwipe(ctx context.Context, startX, startY, endX, endY float64, durationMs, steps int) error {
	if steps < 1 {
		steps = 1
	}
	point := s.touchPoint(1, startX, startY)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}
	stepDelay := time.Duration(durationMs) * time.Millisecond / time.Duration(steps)
	if durationMs <= 0 {
		stepDelay = 0
	}
	for i := 1; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		point.X = s.clampViewportX(startX + (endX-startX)*progress)
		point.Y = s.clampViewportY(startY + (endY-startY)*progress)
		if err := input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{point}).Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchPinch(ctx context.Context, centerX, centerY float64, opts gestureOptions) error {
	steps := opts.Steps
	if steps < minGestureSteps {
		steps = minGestureSteps
	}
	baseRadius := math.Max(float64(opts.Distance)/2, 20)
	targetRadius := baseRadius * opts.Scale
	pointA := s.touchPoint(1, centerX-baseRadius, centerY)
	pointB := s.touchPoint(2, centerX+baseRadius, centerY)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{pointA, pointB}).Do(ctx); err != nil {
		return err
	}
	stepDelay := time.Duration(opts.DurationMs) * time.Millisecond / time.Duration(steps)
	if opts.DurationMs <= 0 {
		stepDelay = 0
	}
	for i := 1; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		radius := baseRadius + (targetRadius-baseRadius)*progress
		pointA.X = s.clampViewportX(centerX - radius)
		pointA.Y = s.clampViewportY(centerY)
		pointB.X = s.clampViewportX(centerX + radius)
		pointB.Y = s.clampViewportY(centerY)
		if err := input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{pointA, pointB}).Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) touchPoint(id float64, x, y float64) *input.TouchPoint {
	return &input.TouchPoint{
		ID:      id,
		X:       s.clampViewportX(x),
		Y:       s.clampViewportY(y),
		RadiusX: 8,
		RadiusY: 8,
		Force:   1,
	}
}

func (s *Session) viewportCenter() (float64, float64) {
	width := float64(s.viewportWidth)
	height := float64(s.viewportHeight)
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}
	return width / 2, height / 2
}

func (s *Session) clampViewportX(value float64) float64 {
	if value < 0 {
		return 0
	}
	maxX := float64(s.viewportWidth)
	if maxX <= 0 {
		maxX = 1920
	}
	if value > maxX {
		return maxX
	}
	return value
}

func (s *Session) clampViewportY(value float64) float64 {
	if value < 0 {
		return 0
	}
	maxY := float64(s.viewportHeight)
	if maxY <= 0 {
		maxY = 1080
	}
	if value > maxY {
		return maxY
	}
	return value
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
