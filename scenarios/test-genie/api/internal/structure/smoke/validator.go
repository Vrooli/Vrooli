package smoke

import (
	"context"
	"fmt"
	"io"
	"os"

	"test-genie/internal/structure/types"
)

// Validator validates that a scenario's UI loads correctly via smoke testing.
type Validator interface {
	// Validate runs the UI smoke test and returns the result.
	Validate(ctx context.Context) types.Result
}

// validator is the default implementation of Validator.
type validator struct {
	scenarioDir    string
	scenarioName   string
	browserlessURL string
	logWriter      io.Writer
}

// ValidatorOption configures a smoke Validator.
type ValidatorOption func(*validator)

// NewValidator creates a new smoke validator for the given scenario.
func NewValidator(scenarioDir, scenarioName string, logWriter io.Writer, opts ...ValidatorOption) Validator {
	v := &validator{
		scenarioDir:    scenarioDir,
		scenarioName:   scenarioName,
		browserlessURL: DefaultBrowserlessURL,
		logWriter:      logWriter,
	}

	// Check environment for browserless URL
	if url := os.Getenv("BROWSERLESS_URL"); url != "" {
		v.browserlessURL = url
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithBrowserlessURL sets a custom browserless URL.
func WithBrowserlessURL(url string) ValidatorOption {
	return func(v *validator) {
		v.browserlessURL = url
	}
}

// Validate implements Validator.
func (v *validator) Validate(ctx context.Context) types.Result {
	runner := NewRunner(v.browserlessURL, WithRunnerLogger(v.logWriter))
	result, err := runner.Run(ctx, v.scenarioName, v.scenarioDir)
	if err != nil {
		return types.FailSystem(err, "Check browserless availability and scenario UI configuration.")
	}

	pr := resultToPhaseResult(result)

	// Handle different outcomes
	if pr.Skipped {
		return types.Result{
			Success:      true,
			ItemsChecked: 1,
			Observations: []types.Observation{
				types.NewSkipObservation(pr.Message),
			},
		}
	}

	if pr.Blocked {
		return types.Result{
			Success:      false,
			Error:        pr.ToError(),
			FailureClass: types.FailureClassMisconfiguration,
			Remediation:  pr.Message,
			Observations: []types.Observation{
				types.NewErrorObservation(pr.Message),
			},
		}
	}

	if !pr.Success {
		// Check for bundle staleness
		if fresh, reason := pr.GetBundleStatus(); !fresh {
			return types.Result{
				Success:      false,
				Error:        fmt.Errorf("ui bundle stale: %s", reason),
				FailureClass: types.FailureClassMisconfiguration,
				Remediation:  "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
				Observations: []types.Observation{
					types.NewErrorObservation(fmt.Sprintf("UI bundle stale: %s", reason)),
				},
			}
		}

		return types.Result{
			Success:      false,
			Error:        pr.ToError(),
			FailureClass: types.FailureClassSystem,
			Remediation:  "Investigate the UI smoke failure (see artifacts in coverage/<scenario>/ui-smoke/) and fix the underlying issue.",
			Observations: []types.Observation{
				types.NewErrorObservation(pr.Message),
			},
		}
	}

	return types.Result{
		Success:      true,
		ItemsChecked: 1,
		Observations: []types.Observation{
			types.NewSuccessObservation(pr.FormatObservation()),
		},
	}
}
