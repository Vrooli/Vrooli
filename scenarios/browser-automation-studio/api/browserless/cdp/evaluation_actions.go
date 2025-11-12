package cdp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ExecuteEvaluate executes arbitrary JavaScript and returns the evaluation result.
func (s *Session) ExecuteEvaluate(ctx context.Context, script string, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var evalResult interface{}
	if err := s.evalWithFrame(timeoutCtx, script, &evalResult); err != nil {
		result.Error = fmt.Sprintf("Evaluate failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	result.DebugContext = map[string]interface{}{"result": evalResult}
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

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	payload := map[string]any{}
	if err := s.evalWithFrame(timeoutCtx, script, &payload); err != nil {
		result.Error = fmt.Sprintf("extract failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if errText, ok := payload["error"].(string); ok && errText != "" {
		result.Error = errText
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, errors.New(errText)
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
	result := &StepResult{DebugContext: map[string]interface{}{
		"switchBy":   opts.SwitchBy,
		"waitNew":    opts.WaitForNew,
		"closeOld":   opts.CloseOld,
		"titleMatch": opts.TitleMatch,
		"urlMatch":   opts.URLMatch,
		"index":      opts.Index,
	}}

	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 30000
	}

	if err := s.refreshTabInventory(ctx); err != nil {
		s.log.WithError(err).Warn("failed to refresh tab inventory before switch")
	}

	if opts.WaitForNew {
		known := s.currentTabSet()
		waitCtx, cancel := context.WithTimeout(s.ctx, time.Duration(opts.TimeoutMs)*time.Millisecond)
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
