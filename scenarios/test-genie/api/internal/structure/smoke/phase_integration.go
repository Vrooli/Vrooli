package smoke

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// DefaultBrowserlessURL is the default Browserless service URL.
const DefaultBrowserlessURL = "http://localhost:4110"

// PhaseResult represents the result of running UI smoke as part of a phase.
type PhaseResult struct {
	// Success indicates whether the smoke test passed.
	Success bool

	// Message is a human-readable summary of the result.
	Message string

	// Result is the detailed smoke test result.
	Result *Result

	// Skipped indicates the test was skipped (not a failure).
	Skipped bool

	// Blocked indicates the test was blocked by preconditions.
	Blocked bool
}

// RunForPhase executes UI smoke test and returns a result suitable for phase integration.
// It looks up browserless URL from environment or uses default.
func RunForPhase(ctx context.Context, scenarioName, scenarioDir string, logWriter io.Writer) (*PhaseResult, error) {
	browserlessURL := os.Getenv("BROWSERLESS_URL")
	if browserlessURL == "" {
		browserlessURL = DefaultBrowserlessURL
	}

	runner := NewRunner(browserlessURL, WithRunnerLogger(logWriter))
	result, err := runner.Run(ctx, scenarioName, scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("ui smoke execution failed: %w", err)
	}

	return resultToPhaseResult(result), nil
}

// resultToPhaseResult converts a Result to a PhaseResult.
func resultToPhaseResult(r *Result) *PhaseResult {
	pr := &PhaseResult{
		Result:  r,
		Message: r.Message,
	}

	switch r.Status {
	case StatusPassed:
		pr.Success = true
		if r.DurationMs > 0 {
			pr.Message = fmt.Sprintf("ui smoke passed (%dms)", r.DurationMs)
		} else {
			pr.Message = "ui smoke passed"
		}
	case StatusSkipped:
		pr.Success = true // skipped is not a failure
		pr.Skipped = true
	case StatusBlocked:
		pr.Success = false
		pr.Blocked = true
	case StatusFailed:
		pr.Success = false
	}

	return pr
}

// FormatObservation formats a PhaseResult as an observation message.
func (pr *PhaseResult) FormatObservation() string {
	if pr.Skipped {
		return fmt.Sprintf("UI smoke skipped: %s", pr.Message)
	}
	if pr.Blocked {
		return fmt.Sprintf("UI smoke blocked: %s", pr.Message)
	}
	if pr.Success {
		return pr.Message
	}
	return fmt.Sprintf("UI smoke failed: %s", pr.Message)
}

// ToError returns an error if the result indicates failure, otherwise nil.
func (pr *PhaseResult) ToError() error {
	if pr.Success || pr.Skipped {
		return nil
	}
	return fmt.Errorf("ui smoke %s: %s", pr.Result.Status, pr.Message)
}

// GetBundleStatus returns bundle status message if bundle is stale.
func (pr *PhaseResult) GetBundleStatus() (bool, string) {
	if pr.Result == nil || pr.Result.Bundle == nil {
		return true, ""
	}
	if !pr.Result.Bundle.Fresh {
		return false, strings.TrimSpace(pr.Result.Bundle.Reason)
	}
	return true, ""
}
