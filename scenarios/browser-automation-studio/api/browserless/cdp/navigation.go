package cdp

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

// ExecuteNavigate navigates to a URL and waits for the requested readiness state.
func (s *Session) ExecuteNavigate(ctx context.Context, url string, waitUntil string, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

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

	if waitAfterMs > 0 {
		time.Sleep(time.Duration(waitAfterMs) * time.Millisecond)
	}

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
