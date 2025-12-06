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
	Verbose      bool // Enable detailed progress logging
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
	traceWriter      artifacts.TraceWriter

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

// WithTraceWriter sets a custom trace writer (for testing).
func WithTraceWriter(w artifacts.TraceWriter) Option {
	return func(r *Runner) {
		r.traceWriter = w
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

	// Note: We don't check for a local ui/ directory because playbooks can target
	// any scenario using destinationType: "scenario" in navigate nodes. The presence
	// of playbooks in the registry is the gate, not whether this scenario has a UI.

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

	// Initialize trace writer if not injected
	if r.traceWriter == nil {
		tw, err := artifacts.NewTraceWriter(r.config.ScenarioDir, r.config.ScenarioName)
		if err != nil {
			shared.LogWarn(r.logWriter, "failed to create trace writer: %v", err)
			r.traceWriter = &artifacts.NullTraceWriter{}
		} else {
			r.traceWriter = tw
			defer r.traceWriter.Close()
		}
	}

	// Log phase start
	_ = r.traceWriter.Write(artifacts.TracePhaseStartEvent(len(reg.Playbooks)))

	// Ensure BAS is available
	if err := r.ensureBAS(ctx); err != nil {
		_ = r.traceWriter.Write(artifacts.TraceBASHealthEvent(false, ""))
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMissingDependency,
			Remediation:  "Start the browser-automation-studio scenario so workflows can execute.",
		}
	}
	_ = r.traceWriter.Write(artifacts.TraceBASHealthEvent(true, ""))

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
		_ = r.traceWriter.Write(artifacts.TracePhaseFailedEvent(err, summary.TotalDuration))
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

	// Log phase complete
	_ = r.traceWriter.Write(artifacts.TracePhaseCompleteEvent(summary.WorkflowsPassed, summary.WorkflowsFailed, summary.TotalDuration))

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
	// Trace workflow start
	_ = r.traceWriter.Write(artifacts.TraceWorkflowStartEvent(entry.File))

	// Resolve workflow path
	workflowPath := entry.File
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.config.ScenarioDir, filepath.FromSlash(workflowPath))
	}

	// Resolve workflow definition
	definition, err := r.workflowResolver.Resolve(ctx, workflowPath)
	if err != nil {
		execErr := NewResolveError(entry.File, err)
		_ = r.traceWriter.Write(artifacts.TraceWorkflowFailedEvent(entry.File, "", execErr, 0))
		return Result{Entry: entry, Err: execErr}
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
		execErr := NewExecuteError(entry.File, err)
		_ = r.traceWriter.Write(artifacts.TraceWorkflowFailedEvent(entry.File, "", execErr, 0))
		return Result{Entry: entry, Err: execErr}
	}

	shared.LogStep(r.logWriter, "workflow %s queued with execution id %s", entry.File, executionID)
	_ = r.traceWriter.Write(artifacts.TraceWorkflowQueuedEvent(entry.File, executionID))

	outcome := &Outcome{ExecutionID: executionID}
	start := time.Now()

	// Create progress callback for verbose mode
	var progressCallback execution.ProgressCallback
	if r.config.Verbose {
		progressCallback = func(status *ExecutionStatus, elapsed time.Duration) error {
			if status == nil {
				return nil
			}
			// BAS returns Progress as 0-100 percentage directly
			progress := float64(status.GetProgress()) / 100.0

			// CurrentStep is the step name/label from BAS
			currentStep := status.GetCurrentStep()
			if currentStep == "" {
				currentStep = status.GetStatus() // Fallback to status text
			}

			shared.LogStep(r.logWriter, "[%s] progress: %.0f%% (%s) - %s",
				entry.File, progress*100, elapsed.Round(time.Second), currentStep)
			_ = r.traceWriter.Write(artifacts.TraceWorkflowProgressEvent(entry.File, executionID, progress, currentStep))
			return nil
		}
	}

	// Wait for completion with optional progress reporting
	execErr := r.basClient.WaitForCompletionWithProgress(ctx, executionID, progressCallback)
	outcome.Duration = time.Since(start)

	// Collect artifacts (both on success and failure for debugging)
	artifactResult := r.collectWorkflowArtifacts(ctx, entry, executionID, outcome, execErr)

	if execErr != nil {
		playErr := NewWaitError(entry.File, executionID, execErr)
		playErr.Artifacts = ExecutionArtifacts{
			Timeline: artifactResult.Timeline,
			Trace:    r.traceWriter.Path(),
		}
		// Attach proto timeline for rich error diagnostics
		if artifactResult.Proto != nil {
			playErr.WithTimeline(artifactResult.Proto)
		}
		_ = r.traceWriter.Write(artifacts.TraceWorkflowFailedEvent(entry.File, executionID, execErr, outcome.Duration))

		return Result{
			Entry:        entry,
			Outcome:      outcome,
			Err:          playErr,
			ArtifactPath: artifactResult.Dir,
		}
	}

	_ = r.traceWriter.Write(artifacts.TraceWorkflowCompleteEvent(entry.File, executionID, outcome.Duration, outcome.Stats))

	return Result{Entry: entry, Outcome: outcome, ArtifactPath: artifactResult.Dir}
}

// collectWorkflowArtifacts fetches and writes all artifacts for a workflow execution.
func (r *Runner) collectWorkflowArtifacts(
	ctx context.Context,
	entry Entry,
	executionID string,
	outcome *Outcome,
	execErr error,
) *artifacts.WorkflowArtifacts {
	// Fetch timeline data
	timeline, timelineData, fetchErr := r.basClient.GetTimeline(ctx, executionID)
	if fetchErr != nil {
		shared.LogWarn(r.logWriter, "failed to fetch timeline for %s: %v", entry.File, fetchErr)
		return &artifacts.WorkflowArtifacts{}
	}

	// Parse timeline for structured data
	parsed, parseErr := execution.ParseFullTimeline(timelineData)
	if parseErr != nil {
		shared.LogWarn(r.logWriter, "failed to parse timeline for %s: %v", entry.File, parseErr)
		// Continue with nil parsed - will still write raw timeline
	} else if parsed != nil && timeline != nil {
		// Reuse the already-fetched proto timeline to avoid duplicate parsing work downstream.
		parsed.Proto = timeline
	}

	// Update outcome stats from parsed timeline
	if parsed != nil {
		outcome.Stats = parsed.Summary.String()
	}

	// Download screenshots
	var screenshots []artifacts.ScreenshotData
	if parsed != nil {
		screenshots = r.downloadScreenshots(ctx, entry.File, parsed)
	}

	// Build result for README generation
	status := "passed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	result := &artifacts.WorkflowResult{
		WorkflowFile:  entry.File,
		Description:   entry.Description,
		Requirements:  entry.Requirements,
		ExecutionID:   executionID,
		Success:       execErr == nil,
		Status:        status,
		Error:         errMsg,
		DurationMs:    outcome.Duration.Milliseconds(),
		Timestamp:     time.Now().UTC(),
		Summary:       getSummaryFromParsed(parsed),
		ParsedSummary: parsed,
	}

	// Get the FileWriter from the interface (if available)
	fileWriter, ok := r.artifactWriter.(*artifacts.FileWriter)
	if !ok {
		// Fallback to legacy timeline-only writing
		shared.LogWarn(r.logWriter, "artifact writer does not support full artifact collection, using legacy mode")
		timelinePath, _ := r.artifactWriter.WriteTimeline(entry.File, timelineData)
		return &artifacts.WorkflowArtifacts{Dir: timelinePath, Timeline: timelinePath}
	}

	// Write all artifacts
	workflowArtifacts, writeErr := fileWriter.WriteWorkflowArtifacts(
		entry.File,
		timelineData,
		parsed,
		screenshots,
		result,
	)
	if writeErr != nil {
		shared.LogWarn(r.logWriter, "failed to write artifacts for %s: %v", entry.File, writeErr)
	}

	if workflowArtifacts != nil {
		shared.LogStep(r.logWriter, "artifacts written to %s", workflowArtifacts.Dir)
		// Attach proto timeline for error diagnostics
		workflowArtifacts.Proto = timeline
	}

	return workflowArtifacts
}

// downloadScreenshots downloads screenshot images from BAS.
func (r *Runner) downloadScreenshots(ctx context.Context, workflowFile string, parsed *execution.ParsedTimeline) []artifacts.ScreenshotData {
	var screenshots []artifacts.ScreenshotData

	// Extract screenshot references from parsed timeline
	refs := artifacts.ExtractScreenshotsFromTimeline(parsed)
	if len(refs) == 0 {
		return screenshots
	}

	shared.LogStep(r.logWriter, "downloading %d screenshots for %s", len(refs), workflowFile)

	for _, ref := range refs {
		if ref.URL == "" {
			continue
		}

		data, err := r.basClient.DownloadAsset(ctx, ref.URL)
		if err != nil {
			shared.LogWarn(r.logWriter, "failed to download screenshot for step %d: %v", ref.StepIndex, err)
			continue
		}

		screenshots = append(screenshots, artifacts.ScreenshotData{
			StepIndex: ref.StepIndex,
			StepName:  ref.StepType,
			Filename:  artifacts.GenerateScreenshotFilename(ref),
			Data:      data,
		})
	}

	return screenshots
}

// getSummaryFromParsed extracts TimelineSummary from parsed timeline.
func getSummaryFromParsed(parsed *execution.ParsedTimeline) execution.TimelineSummary {
	if parsed == nil {
		return execution.TimelineSummary{}
	}
	return parsed.Summary
}

// ensureBAS ensures the BAS API is available.
// It always attempts to start BAS to ensure it's running, then waits for health.
func (r *Runner) ensureBAS(ctx context.Context) error {
	if r.basClient != nil {
		// Client already configured (likely in tests)
		return r.basClient.WaitForHealth(ctx)
	}

	// Always start BAS first - vrooli scenario start is idempotent (safe if already running)
	if r.startScenario != nil {
		shared.LogStep(r.logWriter, "ensuring browser-automation-studio is running")
		if startErr := r.startScenario(ctx, BASScenarioName); startErr != nil {
			return fmt.Errorf("failed to start browser-automation-studio: %w", startErr)
		}
	}

	// Resolve BAS port
	var apiPort string
	var err error
	if r.resolvePort != nil {
		apiPort, err = r.resolvePort(ctx, BASScenarioName, "API_PORT")
	}

	if err != nil || apiPort == "" {
		return fmt.Errorf("browser-automation-studio API port unavailable")
	}

	apiBase := fmt.Sprintf("http://127.0.0.1:%s/api/v1", apiPort)
	r.basClient = execution.NewClient(apiBase)

	return r.basClient.WaitForHealth(ctx)
}
