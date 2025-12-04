package dependencies

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/dependencies/commands"
	"test-genie/internal/dependencies/packages"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/dependencies/runtime"
)

// Config holds configuration for dependency validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// AppRoot is the Vrooli application root directory.
	AppRoot string

	// CommandLookup is an optional custom command lookup function.
	// If nil, exec.LookPath is used.
	CommandLookup commands.LookupFunc
}

// Runner orchestrates dependency validation across commands, runtime, packages, and resources.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	commandChecker    commands.Checker
	runtimeDetector   runtime.Detector
	packageDetector   packages.Detector
	resourceLoader    resources.ExpectationsLoader
	resourceChecker   resources.HealthChecker

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new dependency validation runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for validators if not provided via options
	if r.commandChecker == nil {
		var cmdOpts []commands.Option
		if config.CommandLookup != nil {
			cmdOpts = append(cmdOpts, commands.WithLookup(config.CommandLookup))
		}
		r.commandChecker = commands.New(r.logWriter, cmdOpts...)
	}
	if r.runtimeDetector == nil {
		r.runtimeDetector = runtime.New(config.ScenarioDir, r.logWriter)
	}
	if r.packageDetector == nil {
		r.packageDetector = packages.New(config.ScenarioDir, r.logWriter)
	}
	if r.resourceLoader == nil {
		r.resourceLoader = resources.NewLoader(config.ScenarioDir, r.logWriter)
	}
	// resourceChecker is intentionally nil by default - it requires a StatusFetcher
	// which depends on runtime context

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithCommandChecker sets a custom command checker (for testing).
func WithCommandChecker(c commands.Checker) Option {
	return func(r *Runner) {
		r.commandChecker = c
	}
}

// WithRuntimeDetector sets a custom runtime detector (for testing).
func WithRuntimeDetector(d runtime.Detector) Option {
	return func(r *Runner) {
		r.runtimeDetector = d
	}
}

// WithPackageDetector sets a custom package detector (for testing).
func WithPackageDetector(d packages.Detector) Option {
	return func(r *Runner) {
		r.packageDetector = d
	}
}

// WithResourceLoader sets a custom resource loader (for testing).
func WithResourceLoader(l resources.ExpectationsLoader) Option {
	return func(r *Runner) {
		r.resourceLoader = l
	}
}

// WithResourceChecker sets a custom resource checker (for testing).
func WithResourceChecker(c resources.HealthChecker) Option {
	return func(r *Runner) {
		r.resourceChecker = c
	}
}

// Run executes all dependency validations and returns the aggregated result.
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

	logInfo(r.logWriter, "Starting dependency validation for %s", r.config.ScenarioName)

	// Section: Baseline Commands
	observations = append(observations, NewSectionObservation("🔧", "Checking baseline commands..."))
	logInfo(r.logWriter, "Checking baseline commands...")

	baselineReqs := commands.BaselineRequirements()
	cmdResult := r.commandChecker.CheckAll(baselineReqs)
	summary.CommandsChecked += len(baselineReqs)

	// Convert command observations
	for _, obs := range cmdResult.Observations {
		observations = append(observations, convertObservation(obs))
	}

	if !cmdResult.Success {
		return r.failFromCommandResult(cmdResult, observations)
	}
	logSuccess(r.logWriter, "All baseline commands available (%d)", len(baselineReqs))

	// Section: Language Runtimes
	observations = append(observations, NewSectionObservation("🏃", "Detecting language runtimes..."))
	logInfo(r.logWriter, "Detecting language runtimes...")

	runtimes := r.runtimeDetector.Detect()
	summary.RuntimesDetected = len(runtimes)

	if len(runtimes) == 0 {
		logWarn(r.logWriter, "no language runtimes detected for this scenario")
		observations = append(observations, NewInfoObservation("no runtime-specific checks detected"))
	} else {
		// Check runtime commands
		runtimeReqs := runtime.ToCommandRequirements(runtimes)
		runtimeResult := r.commandChecker.CheckAll(runtimeReqs)
		summary.CommandsChecked += len(runtimeReqs)

		for _, obs := range runtimeResult.Observations {
			observations = append(observations, convertObservation(obs))
		}

		if !runtimeResult.Success {
			return r.failFromCommandResult(runtimeResult, observations)
		}
		logSuccess(r.logWriter, "All required runtimes available (%d)", len(runtimes))
	}

	// Section: Package Managers
	observations = append(observations, NewSectionObservation("📦", "Detecting package managers..."))
	logInfo(r.logWriter, "Detecting package managers...")

	managers := r.packageDetector.Detect()
	summary.ManagersDetected = len(managers)

	if len(managers) == 0 {
		if r.packageDetector.HasNodeWorkspace() {
			observations = append(observations, NewInfoObservation(
				"JavaScript workspace detected but package manager requirement defaulted to pnpm",
			))
		} else {
			observations = append(observations, NewInfoObservation("no JavaScript package managers required"))
		}
	} else {
		// Check package manager commands
		managerReqs := packages.ToCommandRequirements(managers)
		managerResult := r.commandChecker.CheckAll(managerReqs)
		summary.CommandsChecked += len(managerReqs)

		for _, obs := range managerResult.Observations {
			observations = append(observations, convertObservation(obs))
		}

		if !managerResult.Success {
			return r.failFromCommandResult(managerResult, observations)
		}
		logSuccess(r.logWriter, "All required package managers available (%d)", len(managers))
	}

	// Section: Resource Expectations
	observations = append(observations, NewSectionObservation("🔗", "Loading resource expectations..."))
	logInfo(r.logWriter, "Loading resource expectations...")

	requiredResources, err := r.resourceLoader.Load()
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Fix .vrooli/service.json so required resources can be read.",
			Observations: observations,
		}
	}

	summary.ResourcesChecked = len(requiredResources)

	if len(requiredResources) == 0 {
		observations = append(observations, NewInfoObservation("manifest declares no required resources"))
	} else {
		for _, resource := range requiredResources {
			observations = append(observations, NewInfoObservation(
				fmt.Sprintf("requires resource: %s", resource),
			))
		}
	}

	// Section: Resource Health (if checker is configured)
	if r.resourceChecker != nil {
		observations = append(observations, NewSectionObservation("💚", "Checking resource health..."))
		logInfo(r.logWriter, "Checking resource health...")

		healthResult := r.resourceChecker.Check(ctx)
		for _, obs := range healthResult.Observations {
			observations = append(observations, convertObservation(obs))
		}

		if !healthResult.Success {
			return &RunResult{
				Success:      false,
				Error:        healthResult.Error,
				FailureClass: FailureClassMissingDependency,
				Remediation:  healthResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
	}

	// Final summary
	totalChecks := summary.TotalChecks()
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Message: fmt.Sprintf("Dependency validation completed (%d checks)", totalChecks),
	})

	logSuccess(r.logWriter, "Dependency validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// failFromCommandResult constructs a failure RunResult from a command Result.
func (r *Runner) failFromCommandResult(result commands.Result, observations []Observation) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        result.Error,
		FailureClass: FailureClassMissingDependency,
		Remediation:  result.Remediation,
		Observations: observations,
	}
}

// convertObservation converts a types.Observation to dependencies.Observation.
func convertObservation(obs interface{}) Observation {
	// Handle both structure/types.Observation and local Observation
	switch o := obs.(type) {
	case Observation:
		return o
	default:
		// For structure/types.Observation, we need to convert
		// Since we're using type aliases, we can just create a new one
		return NewInfoObservation(fmt.Sprintf("%v", obs))
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

// logWarn writes a warning message.
func logWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}
