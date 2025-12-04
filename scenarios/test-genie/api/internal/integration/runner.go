package integration

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/integration/api"
	"test-genie/internal/integration/bats"
	"test-genie/internal/integration/cli"
	"test-genie/internal/integration/websocket"
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
				ScenarioDir:  config.ScenarioDir,
				ScenarioName: config.ScenarioName,
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

	logInfo(r.logWriter, "Starting integration validation for %s", r.config.ScenarioName)

	// Section: API Health Checks (if configured)
	if r.apiValidator != nil {
		observations = append(observations, NewSectionObservation("🌐", "Validating API health..."))
		logInfo(r.logWriter, "Validating API health...")

		apiResult := r.apiValidator.Validate(ctx)
		observations = append(observations, apiResult.Observations...)
		if !apiResult.Success {
			return &RunResult{
				Success:      false,
				Error:        apiResult.Error,
				FailureClass: FailureClass(apiResult.FailureClass),
				Remediation:  apiResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
		summary.APIHealthChecked = true
		logSuccess(r.logWriter, "API health check passed (status %d, %dms)", apiResult.StatusCode, apiResult.ResponseTimeMs)
	} else {
		observations = append(observations, NewSkipObservation("API health check skipped (no API URL configured)"))
		logInfo(r.logWriter, "Skipping API health check (no URL configured)")
	}

	// Section: CLI Validation
	observations = append(observations, NewSectionObservation("🖥️", "Validating CLI..."))
	logInfo(r.logWriter, "Validating CLI...")

	if r.cliValidator == nil {
		return &RunResult{
			Success:      false,
			Error:        fmt.Errorf("CLI validator not configured (missing command executor/capture)"),
			FailureClass: FailureClassSystem,
			Remediation:  "Ensure command executor and capture functions are provided.",
			Observations: observations,
			Summary:      summary,
		}
	}

	cliResult := r.cliValidator.Validate(ctx)
	observations = append(observations, cliResult.Observations...)
	if !cliResult.Success {
		return &RunResult{
			Success:      false,
			Error:        cliResult.Error,
			FailureClass: FailureClass(cliResult.FailureClass),
			Remediation:  cliResult.Remediation,
			Observations: observations,
			Summary:      summary,
		}
	}
	summary.CLIValidated = true
	logSuccess(r.logWriter, "CLI validation complete")

	// Note about Go-native phases
	observations = append(observations, NewInfoObservation("go-native phase execution"))
	logInfo(r.logWriter, "skipping bash orchestrator validation (Go-native phases)")

	// Section: BATS Validation
	observations = append(observations, NewSectionObservation("🧪", "Running BATS acceptance tests..."))
	logInfo(r.logWriter, "Running BATS acceptance tests...")

	if r.batsRunner == nil {
		return &RunResult{
			Success:      false,
			Error:        fmt.Errorf("BATS runner not configured (missing command executor)"),
			FailureClass: FailureClassSystem,
			Remediation:  "Ensure command executor function is provided.",
			Observations: observations,
			Summary:      summary,
		}
	}

	batsResult := r.batsRunner.Run(ctx)
	observations = append(observations, batsResult.Observations...)
	if !batsResult.Success {
		return &RunResult{
			Success:      false,
			Error:        batsResult.Error,
			FailureClass: FailureClass(batsResult.FailureClass),
			Remediation:  batsResult.Remediation,
			Observations: observations,
			Summary:      summary,
		}
	}
	summary.PrimaryBatsRan = true
	summary.AdditionalBatsSuites = batsResult.AdditionalSuitesRun
	logSuccess(r.logWriter, "BATS validation complete")

	// Section: WebSocket Validation (if configured)
	if r.websocketValidator != nil {
		observations = append(observations, NewSectionObservation("🔌", "Validating WebSocket connection..."))
		logInfo(r.logWriter, "Validating WebSocket connection...")

		wsResult := r.websocketValidator.Validate(ctx)
		observations = append(observations, wsResult.Observations...)
		if !wsResult.Success {
			return &RunResult{
				Success:      false,
				Error:        wsResult.Error,
				FailureClass: FailureClass(wsResult.FailureClass),
				Remediation:  wsResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
		summary.WebSocketValidated = true
		logSuccess(r.logWriter, "WebSocket validation complete (%dms connection)", wsResult.ConnectionTimeMs)
	} else {
		observations = append(observations, NewSkipObservation("WebSocket validation skipped (no WebSocket URL configured)"))
		logInfo(r.logWriter, "Skipping WebSocket validation (no URL configured)")
	}

	// Final summary
	totalChecks := summary.TotalChecks()
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Integration validation completed (%d checks)", totalChecks),
	})

	logSuccess(r.logWriter, "Integration validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// logInfo writes an info message.
func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

// logSuccess writes a success message.
func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}
