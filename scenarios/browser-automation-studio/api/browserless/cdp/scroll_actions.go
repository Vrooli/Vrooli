package cdp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// ExecuteScroll scrolls the page or targeted elements using the requested strategy.
func (s *Session) ExecuteScroll(ctx context.Context, opts scrollOptions, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"scrollType":     opts.scrollType,
		"selector":       opts.selector,
		"targetSelector": opts.targetSelector,
		"direction":      opts.direction,
		"amount":         opts.amount,
		"behavior":       opts.behavior,
		"x":              opts.x,
		"y":              opts.y,
		"maxAttempts":    opts.maxAttempts,
	}}

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
func (s *Session) ExecuteRotate(ctx context.Context, orientation string, angle int, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"orientation": orientation,
		"angle":       angle,
	}}

	if orientation == "" {
		orientation = rotateOrientationPortrait
	}

	width, height := resolveViewportForOrientation(s.viewportWidth, s.viewportHeight, orientation)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	orientationType := deriveScreenOrientationType(orientation, angle)
	cmd := emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1, true).
		WithScreenOrientation(&emulation.ScreenOrientation{Type: emulation.OrientationType(orientationType), Angle: int64(angle)})

	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
		return cmd.Do(runCtx)
	})); err != nil {
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

func (s *Session) scrollPage(ctx context.Context, direction string, amount int, behavior string) error {
	if direction == "" {
		direction = "down"
	}
	behavior = scrollBehaviorOrAuto(behavior)

	var script string
	switch direction {
	case "top":
		script = fmt.Sprintf(`window.scrollTo({ top: 0, behavior: %q });`, behavior)
	case "bottom":
		script = fmt.Sprintf(`window.scrollTo({ top: document.body.scrollHeight, behavior: %q });`, behavior)
	default:
		dx, dy := scrollDeltaFromDirection(direction, amount)
		script = fmt.Sprintf(`window.scrollBy({ left: %d, top: %d, behavior: %q });`, dx, dy, behavior)
	}
	return s.evalWithFrame(ctx, script, nil)
}

func (s *Session) scrollToPosition(ctx context.Context, x, y int, behavior string) error {
	script := fmt.Sprintf(`window.scrollTo({ left: %d, top: %d, behavior: %q });`, x, y, scrollBehaviorOrAuto(behavior))
	return s.evalWithFrame(ctx, script, nil)
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
	return s.evalWithFrame(ctx, script, nil)
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
		found, visible, err := s.checkSelectorVisibility(ctx, target)
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
