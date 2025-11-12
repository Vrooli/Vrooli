package cdp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
)

// ExecuteFocus sets focus on a specific element to trigger focus/keyboard events.
func (s *Session) ExecuteFocus(ctx context.Context, selector string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	if err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.ScrollIntoView(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.Focus(selector, s.frameQueryOptions(chromedp.ByQuery)...),
	); err != nil {
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
	if err := chromedp.Run(timeoutCtx,
		chromedp.WaitReady(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.ScrollIntoView(selector, s.frameQueryOptions(chromedp.ByQuery)...),
	); err != nil {
		result.Error = fmt.Sprintf("blur failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if err := s.evalWithFrame(timeoutCtx, script, &evalResult); err != nil {
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
