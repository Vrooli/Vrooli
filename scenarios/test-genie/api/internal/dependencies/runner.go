package dependencies

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/dependencies/commands"
	"test-genie/internal/dependencies/packages"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/dependencies/runtime"
	"test-genie/internal/shared"
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

	// SkipResourceHealthWhenNoRequired avoids runtime status telemetry when the
	// manifest declares no required resources.
	SkipResourceHealthWhenNoRequired bool

	// Settings optionally overrides dependency policy loaded from testing.json.
	Settings *Settings

	// ScenarioStatusFetcher reads typed scenario runtime status when scenario
	// dependency checks are enabled.
	ScenarioStatusFetcher ScenarioStatusFetcher

	// ResourceStatusFetcher reads typed resource runtime status when resource
	// health checks are enabled.
	ResourceStatusFetcher resources.ResourceStatusFetcher
}

// Runner orchestrates dependency validation across commands, runtime, packages, and resources.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	commandChecker  commands.Checker
	runtimeDetector runtime.Detector
	packageDetector packages.Detector
	resourceLoader  resources.ExpectationsLoader
	resourceChecker resources.HealthChecker
	goModuleChecker GoModuleChecker
	nodeChecker     NodePackageChecker
	scenarioChecker ScenarioDependencyChecker
	commandRunner   CommandRunner
	settings        Settings
	settingsErr     error

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
	if config.Settings != nil {
		r.settings = *config.Settings
	} else {
		settings, err := LoadSettings(config.ScenarioDir)
		if err != nil {
			r.settings = DefaultSettings()
			r.settingsErr = err
		} else {
			r.settings = settings
		}
	}
	if r.commandRunner == nil {
		r.commandRunner = execCommandRunner{}
	}
	if r.goModuleChecker == nil {
		r.goModuleChecker = NewGoModuleChecker(config.ScenarioDir, r.settings.GoModules)
	}
	if r.nodeChecker == nil {
		r.nodeChecker = NewNodePackageChecker(config.ScenarioDir, r.settings.NodePackages)
	}
	if r.scenarioChecker == nil && config.ScenarioStatusFetcher != nil {
		r.scenarioChecker = NewScenarioDependencyChecker(config.ScenarioDir, r.settings.Scenarios, config.ScenarioStatusFetcher)
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

// WithGoModuleChecker sets a custom Go module checker (for testing).
func WithGoModuleChecker(c GoModuleChecker) Option {
	return func(r *Runner) {
		r.goModuleChecker = c
	}
}

// WithNodePackageChecker sets a custom Node package checker (for testing).
func WithNodePackageChecker(c NodePackageChecker) Option {
	return func(r *Runner) {
		r.nodeChecker = c
	}
}

// WithScenarioDependencyChecker sets a custom scenario dependency checker (for testing).
func WithScenarioDependencyChecker(c ScenarioDependencyChecker) Option {
	return func(r *Runner) {
		r.scenarioChecker = c
	}
}

// WithCommandRunner sets a custom command runner for version probes (for testing).
func WithCommandRunner(c CommandRunner) Option {
	return func(r *Runner) {
		r.commandRunner = c
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
	if r.settingsErr != nil {
		return &RunResult{
			Success:      false,
			Error:        r.settingsErr,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Fix the dependencies section in .vrooli/testing.json.",
		}
	}

	var observations []Observation
	var summary ValidationSummary

	shared.LogInfo(r.logWriter, "Starting dependency validation for %s", r.config.ScenarioName)

	// Section: Baseline Commands
	observations = append(observations, NewSectionObservation("🔧", "Checking baseline commands..."))
	shared.LogInfo(r.logWriter, "Checking baseline commands...")

	baselineReqs := commands.BaselineRequirements()
	cmdResult := r.commandChecker.CheckAll(baselineReqs)
	summary.CommandsChecked += len(baselineReqs)

	// Append observations directly (types are compatible via type alias)
	observations = append(observations, cmdResult.Observations...)

	if !cmdResult.Success {
		return r.failFromCommandResult(cmdResult, observations)
	}
	shared.LogSuccess(r.logWriter, "All baseline commands available (%d)", len(baselineReqs))

	// Section: Language Runtimes
	observations = append(observations, NewSectionObservation("🏃", "Detecting language runtimes..."))
	shared.LogInfo(r.logWriter, "Detecting language runtimes...")

	runtimes := r.runtimeDetector.Detect()
	summary.RuntimesDetected = len(runtimes)

	if len(runtimes) == 0 {
		shared.LogWarn(r.logWriter, "no language runtimes detected for this scenario")
		observations = append(observations, NewInfoObservation("no runtime-specific checks detected"))
	} else {
		// Check runtime commands
		runtimeReqs := runtime.ToCommandRequirements(runtimes)
		runtimeResult := r.commandChecker.CheckAll(runtimeReqs)
		summary.CommandsChecked += len(runtimeReqs)

		observations = append(observations, runtimeResult.Observations...)

		if !runtimeResult.Success {
			return r.failFromCommandResult(runtimeResult, observations)
		}
		for _, rt := range runtimes {
			if failed := r.checkVersion(ctx, rt.Command, r.settings.RuntimeVersions[rt.Command], &observations, summary); failed != nil {
				return failed
			}
		}
		shared.LogSuccess(r.logWriter, "All required runtimes available (%d)", len(runtimes))
	}

	// Section: Go Module Freshness
	if r.goModuleChecker != nil {
		observations = append(observations, NewSectionObservation("🧩", "Checking Go module state..."))
		goResult := r.goModuleChecker.Check(ctx)
		summary.GoModulesChecked += goResult.Checked
		observations = append(observations, goResult.Observations...)
		if !goResult.Success {
			return &RunResult{
				Success:      false,
				Error:        goResult.Error,
				FailureClass: FailureClassMissingDependency,
				Remediation:  goResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
	}

	// Section: Package Managers
	observations = append(observations, NewSectionObservation("📦", "Detecting package managers..."))
	shared.LogInfo(r.logWriter, "Detecting package managers...")

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

		observations = append(observations, managerResult.Observations...)

		if !managerResult.Success {
			return r.failFromCommandResult(managerResult, observations)
		}
		for _, manager := range managers {
			if failed := r.checkVersion(ctx, manager.Name, r.settings.RuntimeVersions[manager.Name], &observations, summary); failed != nil {
				return failed
			}
		}
		shared.LogSuccess(r.logWriter, "All required package managers available (%d)", len(managers))
	}

	// Section: JavaScript Package State
	if r.nodeChecker != nil {
		observations = append(observations, NewSectionObservation("📚", "Checking JavaScript package state..."))
		nodeResult := r.nodeChecker.Check()
		summary.NodePackagesChecked += nodeResult.Checked
		observations = append(observations, nodeResult.Observations...)
		if !nodeResult.Success {
			return &RunResult{
				Success:      false,
				Error:        nodeResult.Error,
				FailureClass: nodeResult.FailureClass,
				Remediation:  nodeResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
	}

	// Section: Resource Expectations
	observations = append(observations, NewSectionObservation("🔗", "Loading resource expectations..."))
	shared.LogInfo(r.logWriter, "Loading resource expectations...")

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

	// Section: Scenario Dependencies
	if r.scenarioChecker != nil {
		observations = append(observations, NewSectionObservation("🧭", "Checking scenario dependencies..."))
		scenarioResult := r.scenarioChecker.Check(ctx)
		summary.ScenariosChecked += scenarioResult.Checked
		observations = append(observations, scenarioResult.Observations...)
		if !scenarioResult.Success {
			return &RunResult{
				Success:      false,
				Error:        scenarioResult.Error,
				FailureClass: scenarioResult.FailureClass,
				Remediation:  scenarioResult.Remediation,
				Observations: observations,
				Summary:      summary,
			}
		}
	}

	summary.ResourcesChecked = len(requiredResources)
	if r.resourceChecker == nil && r.config.ResourceStatusFetcher != nil {
		r.resourceChecker = resources.NewChecker(
			requiredResources,
			r.config.ResourceStatusFetcher,
			r.logWriter,
			resources.WithAllowUnknownHealthWhenRunning(r.settings.Resources.AllowUnknownHealthWhenRunning),
			resources.WithSkippedResources(r.settings.Resources.Skip),
		)
	}

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
	if r.settings.Resources.HealthPolicy == "skip" {
		observations = append(observations, NewSkipObservation("resource health checks skipped via .vrooli/testing.json"))
	} else if r.resourceChecker != nil && r.config.SkipResourceHealthWhenNoRequired && len(requiredResources) == 0 {
		observations = append(observations, NewInfoObservation("resource health telemetry not applicable without required resources"))
	} else if r.resourceChecker != nil {
		observations = append(observations, NewSectionObservation("💚", "Checking resource health..."))
		shared.LogInfo(r.logWriter, "Checking resource health...")

		healthResult := r.resourceChecker.Check(ctx)
		observations = append(observations, healthResult.Observations...)

		if !healthResult.Success && r.settings.Resources.HealthPolicy == "warn" {
			observations = append(observations, NewWarningObservation(fmt.Sprintf("resource health check did not pass: %v", healthResult.Error)))
		} else if !healthResult.Success {
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

	shared.LogSuccess(r.logWriter, "Dependency validation complete")

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
