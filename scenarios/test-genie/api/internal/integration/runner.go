package integration

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/integration/api"
	"test-genie/internal/integration/bats"
	"test-genie/internal/integration/cli"
	"test-genie/internal/integration/websocket"
	"test-genie/internal/shared"
)

// Config holds configuration for integration validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// APIBaseURL is the base URL for API health checks (e.g., "http://localhost:8080").
	// If empty, API health checks are skipped.
	APIBaseURL string

	// APIHealthEndpoint is the health check endpoint path (default: "/health").
	APIHealthEndpoint string

	// APIMaxResponseMs is the maximum acceptable response time in milliseconds (default: 1000).
	APIMaxResponseMs int64

	// WebSocketURL is the WebSocket URL for connection validation (e.g., "ws://localhost:8080/ws").
	// If empty, WebSocket validation is skipped.
	WebSocketURL string

	// WebSocketMaxConnectionMs is the maximum acceptable connection time in milliseconds (default: 2000).
	WebSocketMaxConnectionMs int64

	// CLI holds CLI validation configuration.
	CLI CLIConfig
}

// CLIConfig holds configuration for CLI validation.
type CLIConfig struct {
	// HelpArgs specifies argument patterns to try for help command, in order of preference.
	// Default: ["help", "--help", "-h"]
	HelpArgs []string

	// VersionArgs specifies argument patterns to try for version command, in order of preference.
	// Default: ["version", "--version", "-v"]
	VersionArgs []string

	// RequireVersionKeyword controls whether version output must contain the word "version".
	// Default: false
	RequireVersionKeyword bool

	// CheckUnknownCommand controls whether to verify the CLI handles unknown commands gracefully.
	// Default: true
	CheckUnknownCommand bool

	// CheckNoArgs controls whether to verify the CLI handles no arguments gracefully.
	// Default: true
	CheckNoArgs bool

	// NoArgsTimeoutMs is the maximum time to wait for the no-args check in milliseconds.
	// Default: 5000
	NoArgsTimeoutMs int64
}

// Runner orchestrates integration validation across API, CLI, BATS, and WebSocket checks.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	apiValidator       api.Validator
	cliValidator       cli.Validator
	batsRunner         bats.Runner
	websocketValidator websocket.Validator

	// Command functions (needed to create validators if not injected)
	commandExecutor cli.CommandExecutor
	commandCapture  cli.CommandCapture
	commandLookup   bats.CommandLookup

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new integration validation runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for API validator if URL is provided
	if r.apiValidator == nil && config.APIBaseURL != "" {
		r.apiValidator = api.New(
			api.Config{
				BaseURL:        config.APIBaseURL,
				HealthEndpoint: config.APIHealthEndpoint,
				MaxResponseMs:  config.APIMaxResponseMs,
			},
			api.WithLogger(r.logWriter),
		)
	}

	// Set defaults for CLI validator if not provided via options
	if r.cliValidator == nil && r.commandExecutor != nil && r.commandCapture != nil {
		r.cliValidator = cli.New(
			cli.Config{
				ScenarioDir:          config.ScenarioDir,
				ScenarioName:         config.ScenarioName,
				HelpArgs:             config.CLI.HelpArgs,
				VersionArgs:          config.CLI.VersionArgs,
				RequireVersionKeyword: config.CLI.RequireVersionKeyword,
				CheckUnknownCommand:  config.CLI.CheckUnknownCommand,
				CheckNoArgs:          config.CLI.CheckNoArgs,
				NoArgsTimeoutMs:      config.CLI.NoArgsTimeoutMs,
			},
			cli.WithLogger(r.logWriter),
			cli.WithExecutor(cli.AdaptExecutor(r.commandExecutor)),
			cli.WithCapture(cli.AdaptCapture(r.commandCapture)),
		)
	}

	if r.batsRunner == nil && r.commandExecutor != nil {
		batsOpts := []bats.Option{
			bats.WithLogger(r.logWriter),
			bats.WithExecutor(bats.AdaptExecutor(r.commandExecutor)),
		}
		if r.commandLookup != nil {
			batsOpts = append(batsOpts, bats.WithLookup(bats.AdaptLookup(r.commandLookup)))
		}
		r.batsRunner = bats.New(
			bats.Config{
				ScenarioDir:  config.ScenarioDir,
				ScenarioName: config.ScenarioName,
			},
			batsOpts...,
		)
	}

	// Set defaults for WebSocket validator if URL is provided
	if r.websocketValidator == nil && config.WebSocketURL != "" {
		r.websocketValidator = websocket.New(
			websocket.Config{
				URL:             config.WebSocketURL,
				MaxConnectionMs: config.WebSocketMaxConnectionMs,
			},
			websocket.WithLogger(r.logWriter),
		)
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithAPIValidator sets a custom API validator (for testing).
func WithAPIValidator(v api.Validator) Option {
	return func(r *Runner) {
		r.apiValidator = v
	}
}

// WithCLIValidator sets a custom CLI validator (for testing).
func WithCLIValidator(v cli.Validator) Option {
	return func(r *Runner) {
		r.cliValidator = v
	}
}

// WithBatsRunner sets a custom BATS runner (for testing).
func WithBatsRunner(br bats.Runner) Option {
	return func(r *Runner) {
		r.batsRunner = br
	}
}

// WithWebSocketValidator sets a custom WebSocket validator (for testing).
func WithWebSocketValidator(v websocket.Validator) Option {
	return func(r *Runner) {
		r.websocketValidator = v
	}
}

// WithCommandExecutor sets the command executor function.
func WithCommandExecutor(exec cli.CommandExecutor) Option {
	return func(r *Runner) {
		r.commandExecutor = exec
	}
}

// WithCommandCapture sets the command capture function.
func WithCommandCapture(cap cli.CommandCapture) Option {
	return func(r *Runner) {
		r.commandCapture = cap
	}
}

// WithCommandLookup sets the command lookup function.
func WithCommandLookup(lookup bats.CommandLookup) Option {
	return func(r *Runner) {
		r.commandLookup = lookup
	}
}

// Run executes all integration validations and returns the aggregated result.
// All checks are run even if some fail - failures are collected and reported together.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var observations []Observation
	var summary ValidationSummary
	var failures []validationFailure

	shared.LogInfo(r.logWriter, "Starting integration validation for %s", r.config.ScenarioName)

	// Section: API Health Checks (if configured)
	if r.apiValidator != nil {
		observations = append(observations, NewSectionObservation("🌐", "Validating API health..."))
		shared.LogInfo(r.logWriter, "Validating API health...")

		apiResult := r.apiValidator.Validate(ctx)
		observations = append(observations, apiResult.Observations...)
		if !apiResult.Success {
			failures = append(failures, validationFailure{
				component:    "API health",
				err:          apiResult.Error,
				failureClass: FailureClass(apiResult.FailureClass),
				remediation:  apiResult.Remediation,
			})
			shared.LogError(r.logWriter, "API health check failed: %v", apiResult.Error)
		} else {
			summary.APIHealthChecked = true
			shared.LogSuccess(r.logWriter, "API health check passed (status %d, %dms)", apiResult.StatusCode, apiResult.ResponseTimeMs)
		}
	} else {
		observations = append(observations, NewSkipObservation("API health check skipped (no API URL configured)"))
		shared.LogInfo(r.logWriter, "Skipping API health check (no URL configured)")
	}

	// Section: CLI Validation
	observations = append(observations, NewSectionObservation("🖥️", "Validating CLI..."))
	shared.LogInfo(r.logWriter, "Validating CLI...")

	if r.cliValidator == nil {
		failures = append(failures, validationFailure{
			component:    "CLI",
			err:          fmt.Errorf("CLI validator not configured (missing command executor/capture)"),
			failureClass: FailureClassSystem,
			remediation:  "Ensure command executor and capture functions are provided.",
		})
		observations = append(observations, NewErrorObservation("CLI validator not configured"))
		shared.LogError(r.logWriter, "CLI validator not configured")
	} else {
		cliResult := r.cliValidator.Validate(ctx)
		observations = append(observations, cliResult.Observations...)
		if !cliResult.Success {
			failures = append(failures, validationFailure{
				component:    "CLI",
				err:          cliResult.Error,
				failureClass: FailureClass(cliResult.FailureClass),
				remediation:  cliResult.Remediation,
			})
			shared.LogError(r.logWriter, "CLI validation failed: %v", cliResult.Error)
		} else {
			summary.CLIValidated = true
			shared.LogSuccess(r.logWriter, "CLI validation complete")
		}
	}

	// Note about Go-native phases
	observations = append(observations, NewInfoObservation("go-native phase execution"))
	shared.LogInfo(r.logWriter, "skipping bash orchestrator validation (Go-native phases)")

	// Section: BATS Validation
	observations = append(observations, NewSectionObservation("🧪", "Running BATS acceptance tests..."))
	shared.LogInfo(r.logWriter, "Running BATS acceptance tests...")

	if r.batsRunner == nil {
		failures = append(failures, validationFailure{
			component:    "BATS",
			err:          fmt.Errorf("BATS runner not configured (missing command executor)"),
			failureClass: FailureClassSystem,
			remediation:  "Ensure command executor function is provided.",
		})
		observations = append(observations, NewErrorObservation("BATS runner not configured"))
		shared.LogError(r.logWriter, "BATS runner not configured")
	} else {
		batsResult := r.batsRunner.Run(ctx)
		observations = append(observations, batsResult.Observations...)
		if !batsResult.Success {
			failures = append(failures, validationFailure{
				component:    "BATS",
				err:          batsResult.Error,
				failureClass: FailureClass(batsResult.FailureClass),
				remediation:  batsResult.Remediation,
			})
			shared.LogError(r.logWriter, "BATS validation failed: %v", batsResult.Error)
		} else if !batsResult.Skipped {
			summary.PrimaryBatsRan = true
			summary.AdditionalBatsSuites = batsResult.AdditionalSuitesRun
			shared.LogSuccess(r.logWriter, "BATS validation complete")
		} else {
			shared.LogInfo(r.logWriter, "BATS validation skipped (no .bats files)")
		}
	}

	// Section: WebSocket Validation (if configured)
	if r.websocketValidator != nil {
		observations = append(observations, NewSectionObservation("🔌", "Validating WebSocket connection..."))
		shared.LogInfo(r.logWriter, "Validating WebSocket connection...")

		wsResult := r.websocketValidator.Validate(ctx)
		observations = append(observations, wsResult.Observations...)
		if !wsResult.Success {
			failures = append(failures, validationFailure{
				component:    "WebSocket",
				err:          wsResult.Error,
				failureClass: FailureClass(wsResult.FailureClass),
				remediation:  wsResult.Remediation,
			})
			shared.LogError(r.logWriter, "WebSocket validation failed: %v", wsResult.Error)
		} else {
			summary.WebSocketValidated = true
			shared.LogSuccess(r.logWriter, "WebSocket validation complete (%dms connection)", wsResult.ConnectionTimeMs)
		}
	} else {
		observations = append(observations, NewSkipObservation("WebSocket validation skipped (no WebSocket URL configured)"))
		shared.LogInfo(r.logWriter, "Skipping WebSocket validation (no URL configured)")
	}

	// Build final result
	if len(failures) > 0 {
		// Aggregate failures into a combined error message
		var errMsg string
		var remediations []string
		worstClass := FailureClassNone
		for _, f := range failures {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %v", f.component, f.err)
			if f.remediation != "" {
				remediations = append(remediations, fmt.Sprintf("[%s] %s", f.component, f.remediation))
			}
			// Use the most severe failure class
			if worstClass == FailureClassNone || f.failureClass == FailureClassSystem {
				worstClass = f.failureClass
			}
		}

		observations = append(observations, Observation{
			Type:    ObservationError,
			Icon:    "❌",
			Message: fmt.Sprintf("Integration validation failed (%d/%d checks failed)", len(failures), len(failures)+summary.TotalChecks()),
		})

		combinedRemediation := ""
		for i, rem := range remediations {
			if i > 0 {
				combinedRemediation += "\n"
			}
			combinedRemediation += rem
		}

		shared.LogError(r.logWriter, "Integration validation failed: %d check(s) failed", len(failures))

		return &RunResult{
			Success:      false,
			Error:        fmt.Errorf("%s", errMsg),
			FailureClass: worstClass,
			Remediation:  combinedRemediation,
			Observations: observations,
			Summary:      summary,
		}
	}

	// Final summary - all checks passed
	totalChecks := summary.TotalChecks()
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Integration validation completed (%d checks)", totalChecks),
	})

	shared.LogSuccess(r.logWriter, "Integration validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// validationFailure tracks a single validation failure for aggregation.
type validationFailure struct {
	component    string
	err          error
	failureClass FailureClass
	remediation  string
}
