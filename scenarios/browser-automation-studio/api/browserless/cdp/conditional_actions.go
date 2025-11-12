package cdp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

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

	result := &StepResult{Success: true, DebugContext: debug}

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

	ctxWithTimeout, cancel := context.WithTimeout(s.ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	if err := s.evalWithFrame(ctxWithTimeout, script, &evaluation); err != nil {
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
		err := s.evalWithFrame(ctx, fmt.Sprintf("!!document.querySelector(%s)", strconv.Quote(trimmed)), &exists)
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
