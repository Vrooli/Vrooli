package playbooks

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/registry"
	"test-genie/internal/playbooks/seeds"
	"test-genie/internal/playbooks/workflow"
	"test-genie/internal/shared"
)

const (
	// BASScenarioName is the name of the browser-automation-studio scenario.
	BASScenarioName = "browser-automation-studio"
)

// Config holds configuration for the playbooks runner.
type Config struct {
	ScenarioDir  string
	ScenarioName string
	TestDir      string
	AppRoot      string
}

// Runner orchestrates playbook execution using injected dependencies.
type Runner struct {
	config Config

	// Injected dependencies (interfaces for testing)
	registryLoader   registry.Loader
	workflowResolver workflow.Resolver
	basClient        execution.Client
	seedManager      seeds.Manager
	artifactWriter   artifacts.Writer

	// Hooks for scenario management (injected for testing)
	resolvePort      func(ctx context.Context, scenario, portName string) (string, error)
	startScenario    func(ctx context.Context, scenario string) error
	resolveUIBaseURL func(ctx context.Context, scenario string) (string, error)

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new playbooks runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for validators if not provided via options
	if r.registryLoader == nil {
		r.registryLoader = registry.NewLoader(config.TestDir)
	}
	if r.workflowResolver == nil {
		r.workflowResolver = workflow.NewResolver(config.AppRoot, config.ScenarioDir)
	}
	if r.seedManager == nil {
		r.seedManager = seeds.NewManager(config.ScenarioDir, config.AppRoot, config.TestDir, r.logWriter)
	}
	if r.artifactWriter == nil {
		r.artifactWriter = artifacts.NewWriter(config.ScenarioDir, config.ScenarioName, config.AppRoot)
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithRegistryLoader sets a custom registry loader (for testing).
func WithRegistryLoader(l registry.Loader) Option {
	return func(r *Runner) {
		r.registryLoader = l
	}
}

// WithWorkflowResolver sets a custom workflow resolver (for testing).
func WithWorkflowResolver(res workflow.Resolver) Option {
	return func(r *Runner) {
		r.workflowResolver = res
	}
}

// WithBASClient sets a custom BAS client (for testing).
func WithBASClient(c execution.Client) Option {
	return func(r *Runner) {
		r.basClient = c
	}
}

// WithSeedManager sets a custom seed manager (for testing).
func WithSeedManager(m seeds.Manager) Option {
	return func(r *Runner) {
		r.seedManager = m
	}
}

// WithArtifactWriter sets a custom artifact writer (for testing).
func WithArtifactWriter(w artifacts.Writer) Option {
	return func(r *Runner) {
		r.artifactWriter = w
	}
}

// WithPortResolver sets a custom port resolver (for testing).
func WithPortResolver(f func(ctx context.Context, scenario, portName string) (string, error)) Option {
	return func(r *Runner) {
		r.resolvePort = f
	}
}

// WithScenarioStarter sets a custom scenario starter (for testing).
func WithScenarioStarter(f func(ctx context.Context, scenario string) error) Option {
	return func(r *Runner) {
		r.startScenario = f
	}
}

// WithUIBaseURLResolver sets a custom UI base URL resolver (for testing).
func WithUIBaseURLResolver(f func(ctx context.Context, scenario string) (string, error)) Option {
	return func(r *Runner) {
		r.resolveUIBaseURL = f
	}
}

// Run executes all playbook workflows and returns the result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	// Check for skip flag
	if os.Getenv("TEST_GENIE_SKIP_PLAYBOOKS") == "1" {
		shared.LogWarn(r.logWriter, "playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS")
		return &RunResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS")},
		}
	}

	// Check if scenario has UI
	if !r.hasUI() {
		shared.LogWarn(r.logWriter, "ui/ directory missing; skipping UI workflow validation")
		return &RunResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("ui/ directory missing; skipping UI workflow validation")},
		}
	}

	// Load registry
	reg, err := r.registryLoader.Load()
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Regenerate test/playbooks/registry.json via playbook builder.",
		}
	}

	if len(reg.Playbooks) == 0 {
		shared.LogWarn(r.logWriter, "no workflows registered under test/playbooks/")
		return &RunResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("no workflows registered under test/playbooks/")},
		}
	}

	// Ensure BAS is available
	if err := r.ensureBAS(ctx); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMissingDependency,
			Remediation:  "Start the browser-automation-studio scenario so workflows can execute.",
		}
	}

	// Apply seeds
	cleanup, err := r.seedManager.Apply(ctx)
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Fix playbook seeds under test/playbooks/__seeds.",
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Resolve UI base URL
	uiBaseURL := ""
	if r.resolveUIBaseURL != nil {
		uiBaseURL, _ = r.resolveUIBaseURL(ctx, r.config.ScenarioName)
	}

	// Execute workflows
	var observations []Observation
	var results []Result
	var summary ExecutionSummary
	phaseStart := time.Now()

	for _, entry := range reg.Playbooks {
		select {
		case <-ctx.Done():
			summary.TotalDuration = time.Since(phaseStart)
			return &RunResult{
				Success:      false,
				Error:        ctx.Err(),
				FailureClass: FailureClassSystem,
				Results:      results,
				Summary:      summary,
			}
		default:
		}

		result := r.executeWorkflow(ctx, entry, uiBaseURL)
		results = append(results, result)
		summary.WorkflowsExecuted++

		if result.Err != nil {
			summary.WorkflowsFailed++
			summary.TotalDuration = time.Since(phaseStart)

			// Write results so far
			_ = r.artifactWriter.WritePhaseResults(results)

			classification := FailureClassExecution
			return &RunResult{
				Success:      false,
				Error:        result.Err,
				FailureClass: classification,
				Remediation:  "Inspect the workflow definition and Browser Automation Studio logs.",
				Observations: observations,
				Results:      results,
				Summary:      summary,
			}
		}

		summary.WorkflowsPassed++
		if result.Outcome != nil && result.Outcome.Stats != "" {
			observations = append(observations, NewSuccessObservation(fmt.Sprintf("%s completed%s", entry.File, result.Outcome.Stats)))
		} else {
			observations = append(observations, NewSuccessObservation(fmt.Sprintf("%s completed", entry.File)))
		}
		shared.LogStep(r.logWriter, "workflow %s completed", entry.File)
	}

	summary.TotalDuration = time.Since(phaseStart)

	// Write final results
	if err := r.artifactWriter.WritePhaseResults(results); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
			Remediation:  "Ensure coverage/phase-results directory is writable.",
			Observations: observations,
			Results:      results,
			Summary:      summary,
		}
	}

	// Add summary observation
	observations = append(observations, NewSuccessObservation(summary.String()))
	shared.LogStep(r.logWriter, "playbook workflows executed: %d", len(reg.Playbooks))
	return &RunResult{
		Success:      true,
		Observations: observations,
		Results:      results,
		Summary:      summary,
	}
}

// executeWorkflow executes a single playbook workflow.
func (r *Runner) executeWorkflow(ctx context.Context, entry Entry, uiBaseURL string) Result {
	// Resolve workflow path
	workflowPath := entry.File
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.config.ScenarioDir, filepath.FromSlash(workflowPath))
	}

	// Resolve workflow definition
	definition, err := r.workflowResolver.Resolve(ctx, workflowPath)
	if err != nil {
		return Result{Entry: entry, Err: fmt.Errorf("failed to resolve workflow: %w", err)}
	}

	// Substitute placeholders
	if uiBaseURL != "" {
		workflow.SubstitutePlaceholders(definition, uiBaseURL)
	}

	// Clean definition for BAS
	cleanDef := workflow.CleanDefinition(definition)

	// Execute workflow
	executionID, err := r.basClient.ExecuteWorkflow(ctx, cleanDef, entry.Description)
	if err != nil {
		return Result{Entry: entry, Err: fmt.Errorf("failed to execute workflow: %w", err)}
	}

	shared.LogStep(r.logWriter, "workflow %s queued with execution id %s", entry.File, executionID)

	outcome := &Outcome{ExecutionID: executionID}
	start := time.Now()

	// Wait for completion
	if err := r.basClient.WaitForCompletion(ctx, executionID); err != nil {
		outcome.Duration = time.Since(start)

		// Try to dump timeline for debugging
		artifactPath := ""
		if timelineData, fetchErr := r.basClient.GetTimeline(ctx, executionID); fetchErr == nil {
			artifactPath, _ = r.artifactWriter.WriteTimeline(entry.File, timelineData)
		}

		return Result{
			Entry:        entry,
			Outcome:      outcome,
			Err:          err,
			ArtifactPath: artifactPath,
		}
	}

	outcome.Duration = time.Since(start)

	// Get timeline for stats
	if timelineData, err := r.basClient.GetTimeline(ctx, executionID); err == nil {
		outcome.Stats = execution.SummarizeTimeline(timelineData)
	}

	return Result{Entry: entry, Outcome: outcome}
}

// ensureBAS ensures the BAS API is available.
func (r *Runner) ensureBAS(ctx context.Context) error {
	if r.basClient != nil {
		// Client already configured (likely in tests)
		return r.basClient.WaitForHealth(ctx)
	}

	// Try to resolve BAS port
	var apiPort string
	var err error
	if r.resolvePort != nil {
		apiPort, err = r.resolvePort(ctx, BASScenarioName, "API_PORT")
	}

	if err != nil || apiPort == "" {
		// Try to start BAS
		if r.startScenario != nil {
			shared.LogWarn(r.logWriter, "browser-automation-studio port lookup failed, attempting to start")
			if startErr := r.startScenario(ctx, BASScenarioName); startErr != nil {
				return fmt.Errorf("failed to start browser-automation-studio: %w", startErr)
			}
			if r.resolvePort != nil {
				apiPort, err = r.resolvePort(ctx, BASScenarioName, "API_PORT")
			}
		}
	}

	if err != nil || apiPort == "" {
		return fmt.Errorf("browser-automation-studio API port unavailable")
	}

	apiBase := fmt.Sprintf("http://127.0.0.1:%s/api/v1", apiPort)
	r.basClient = execution.NewClient(apiBase)

	return r.basClient.WaitForHealth(ctx)
}

// hasUI checks if the scenario has a UI directory.
func (r *Runner) hasUI() bool {
	info, err := os.Stat(filepath.Join(r.config.ScenarioDir, "ui"))
	return err == nil && info.IsDir()
}
