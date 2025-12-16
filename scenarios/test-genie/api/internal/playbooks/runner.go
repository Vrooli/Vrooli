package playbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/registry"
	"test-genie/internal/playbooks/seeds"
	"test-genie/internal/playbooks/types"
	"test-genie/internal/playbooks/workflow"
	"test-genie/internal/shared"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

const (
	// BASScenarioName is the name of the browser-automation-studio scenario.
	BASScenarioName = "browser-automation-studio"
)

// applyEnv temporarily sets env vars, returning a restore function.
func applyEnv(env map[string]string) func() {
	if len(env) == 0 {
		return func() {}
	}
	prev := make(map[string]*string, len(env))
	for k, v := range env {
		if existing, ok := os.LookupEnv(k); ok {
			val := existing
			prev[k] = &val
		} else {
			prev[k] = nil
		}
		_ = os.Setenv(k, v)
	}
	return func() {
		for k, v := range prev {
			if v == nil {
				_ = os.Unsetenv(k)
				continue
			}
			_ = os.Setenv(k, *v)
		}
	}
}

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
	config          Config
	playbooksConfig *config.Config // Configuration from testing.json
	seedEnv         map[string]string

	// Injected dependencies (interfaces for testing)
	registryLoader   registry.Loader
	workflowResolver workflow.Resolver
	basClient        execution.Client
	seedManager      seeds.Manager
	artifactWriter   artifacts.Writer
	traceWriter      artifacts.TraceWriter

	// Hooks for scenario management (injected for testing)
	resolvePort   func(ctx context.Context, scenario, portName string) (string, error)
	startScenario func(ctx context.Context, scenario string) error

	logWriter io.Writer

	// Runtime state
	seedState map[string]any // Cached seed state for passing to BAS as initial_params
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new playbooks runner.
func New(cfg Config, opts ...Option) *Runner {
	r := &Runner{
		config:          cfg,
		playbooksConfig: config.Default(), // Use defaults initially
		logWriter:       io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for validators if not provided via options
	if r.registryLoader == nil {
		r.registryLoader = registry.NewLoader(cfg.ScenarioDir)
	}
	if r.workflowResolver == nil {
		r.workflowResolver = workflow.NewResolver(cfg.AppRoot, cfg.ScenarioDir)
	}
	if r.seedManager == nil {
		r.seedManager = seeds.NewManager(cfg.ScenarioDir, cfg.AppRoot, cfg.TestDir, r.logWriter)
	}
	if r.artifactWriter == nil {
		r.artifactWriter = artifacts.NewWriter(cfg.ScenarioDir, cfg.ScenarioName, cfg.AppRoot)
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

// WithPlaybooksConfig sets the playbooks configuration from testing.json.
func WithPlaybooksConfig(cfg *config.Config) Option {
	return func(r *Runner) {
		if cfg != nil {
			r.playbooksConfig = cfg
		}
	}
}

// WithSeedEnv injects environment variables to apply while running seeds.
func WithSeedEnv(env map[string]string) Option {
	return func(r *Runner) {
		r.seedEnv = env
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

	// Respect config toggle
	if r.playbooksConfig != nil && !r.playbooksConfig.Enabled {
		shared.LogWarn(r.logWriter, "playbooks phase disabled via .vrooli/testing.json (playbooks.enabled=false)")
		return &RunResult{
			Success:      true,
			Observations: []Observation{NewSkipObservation("playbooks phase disabled via .vrooli/testing.json")},
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
			Remediation:  "Regenerate bas/registry.json via playbook builder.",
		}
	}

	if len(reg.Playbooks) == 0 {
		shared.LogWarn(r.logWriter, "no workflows registered under bas/")
		return &RunResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("no workflows registered under bas/")},
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
			Success:       false,
			Error:         err,
			FailureClass:  FailureClassMissingDependency,
			Remediation:   "Start the browser-automation-studio scenario so workflows can execute.",
			TracePath:     r.traceWriter.Path(),
			ArtifactPaths: ArtifactPaths{Trace: r.traceWriter.Path()},
		}
	}
	_ = r.traceWriter.Write(artifacts.TraceBASHealthEvent(true, ""))

	// Apply seeds (optional). Seeds are responsible for writing coverage/runtime/seed-state.json,
	// which is then passed to BAS via execution parameters as initial_params.
	var cleanup func()
	if r.playbooksConfig.Seeds.Enabled {
		seedCtx, cancel := context.WithTimeout(ctx, r.playbooksConfig.Seeds.SeedTimeout())
		defer cancel()

		var err error
		restoreSeedEnv := applyEnv(r.seedEnv)
		cleanup, err = r.seedManager.Apply(seedCtx)
		restoreSeedEnv()
		if err != nil {
			return &RunResult{
				Success:       false,
				Error:         err,
				FailureClass:  FailureClassMisconfiguration,
				Remediation:   "Fix playbook seeds under bas/seeds.",
				TracePath:     r.traceWriter.Path(),
				ArtifactPaths: ArtifactPaths{Trace: r.traceWriter.Path()},
			}
		}

		// Load seed state for passing to BAS as initial_params.
		if seedState, err := r.loadSeedState(); err != nil {
			shared.LogWarn(r.logWriter, "failed to load seed state: %v (continuing with empty params)", err)
		} else if len(seedState) > 0 {
			r.seedState = seedState
			shared.LogStep(r.logWriter, "loaded %d seed values for BAS initial_params", len(seedState))
		}
	} else if !r.playbooksConfig.Seeds.Enabled {
		shared.LogInfo(r.logWriter, "playbooks seeds disabled via config")
	}
	if cleanup != nil {
		defer cleanup()
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
				Success:       false,
				Error:         ctx.Err(),
				FailureClass:  FailureClassSystem,
				Results:       results,
				Summary:       summary,
				TracePath:     r.traceWriter.Path(),
				ArtifactPaths: ArtifactPaths{Trace: r.traceWriter.Path()},
			}
		default:
		}

		result := r.executeWorkflow(ctx, entry)
		results = append(results, result)
		summary.WorkflowsExecuted++

		if result.Err != nil {
			summary.WorkflowsFailed++
			summary.TotalDuration = time.Since(phaseStart)

			// Write results so far
			_ = r.artifactWriter.WritePhaseResults(results)

			// Build result with diagnostic information
			runResult := &RunResult{
				Success:      false,
				Error:        result.Err,
				FailureClass: FailureClassExecution,
				Remediation:  "Inspect the workflow definition and Vrooli Ascension logs.",
				Observations: observations,
				Results:      results,
				Summary:      summary,
				TracePath:    r.traceWriter.Path(),
			}

			// Extract rich diagnostic output if error is a PlaybookExecutionError
			if playErr, ok := result.Err.(*PlaybookExecutionError); ok {
				runResult.DiagnosticOutput = playErr.DiagnosticString()
				// Add artifact observation for visibility
				if playErr.Artifacts.Timeline != "" {
					observations = append(observations, NewInfoObservation(
						fmt.Sprintf("Timeline artifact: %s", playErr.Artifacts.Timeline)))
				}
				runResult.Observations = observations
			}

			// Set artifact paths
			runResult.ArtifactPaths.Trace = r.traceWriter.Path()
			if result.ArtifactPath != "" {
				runResult.AddWorkflowArtifact(entry.File, result.ArtifactPath)
			}

			return runResult
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
			Success:       false,
			Error:         err,
			FailureClass:  FailureClassSystem,
			Remediation:   "Ensure coverage/phase-results directory is writable.",
			Observations:  observations,
			Results:       results,
			Summary:       summary,
			TracePath:     r.traceWriter.Path(),
			ArtifactPaths: ArtifactPaths{Trace: r.traceWriter.Path()},
		}
	}

	// Log phase complete
	_ = r.traceWriter.Write(artifacts.TracePhaseCompleteEvent(summary.WorkflowsPassed, summary.WorkflowsFailed, summary.TotalDuration))

	// Add summary observation
	observations = append(observations, NewSuccessObservation(summary.String()))
	shared.LogStep(r.logWriter, "playbook workflows executed: %d", len(reg.Playbooks))

	// Build artifact paths from results
	artifactPaths := ArtifactPaths{
		Trace:             r.traceWriter.Path(),
		WorkflowArtifacts: make(map[string]string),
	}
	for _, res := range results {
		if res.ArtifactPath != "" {
			artifactPaths.WorkflowArtifacts[res.Entry.File] = res.ArtifactPath
		}
	}

	return &RunResult{
		Success:       true,
		Observations:  observations,
		Results:       results,
		Summary:       summary,
		TracePath:     r.traceWriter.Path(),
		ArtifactPaths: artifactPaths,
	}
}

// executeWorkflow executes a single playbook workflow.
func (r *Runner) executeWorkflow(ctx context.Context, entry Entry) Result {
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

	projectRoot := r.basProjectRoot()
	if strings.TrimSpace(projectRoot) == "" {
		execErr := NewResolveError(entry.File, fmt.Errorf("bas/ directory not found for project_root"))
		_ = r.traceWriter.Write(artifacts.TraceWorkflowFailedEvent(entry.File, "", execErr, 0))
		return Result{Entry: entry, Err: execErr}
	}

	// Execute workflow "as-authored" (no placeholder substitution or scenario URL rewriting in test-genie).
	execName := strings.TrimSpace(entry.File)
	execDescription := strings.TrimSpace(entry.Description)
	if execDescription == "" {
		execDescription = execName
	}
	execParams := &execution.ExecutionParams{
		ProjectRoot:   projectRoot,
		InitialParams: r.seedState,
	}
	executionID, err := r.basClient.ExecuteWorkflowWithParams(ctx, definition, execName, execDescription, execParams)
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
				currentStep = types.ExecutionStatusToString(status.GetStatus()) // Fallback to status text
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
	artifactResult, parseErr := r.collectWorkflowArtifacts(ctx, entry, executionID, outcome, execErr)

	if execErr != nil || parseErr != nil {
		var playErr *PlaybookExecutionError
		if parseErr != nil && execErr == nil {
			playErr = NewArtifactError(entry.File, fmt.Errorf("timeline parse failed: %w", parseErr)).WithExecutionID(executionID)
		} else {
			playErr = NewWaitError(entry.File, executionID, execErr)
		}
		playErr.Artifacts = ExecutionArtifacts{
			Timeline: artifactResult.Timeline,
			Trace:    r.traceWriter.Path(),
		}
		// Attach proto timeline for rich error diagnostics
		if artifactResult.Proto != nil {
			playErr.WithTimeline(artifactResult.Proto)
		}
		// Provide raw timeline path when parse failed (for diagnostics)
		if parseErr != nil && playErr.Artifacts.Timeline != "" {
			playErr.Artifacts.RawTimeline = playErr.Artifacts.Timeline
		}
		_ = r.traceWriter.Write(artifacts.TraceWorkflowFailedEvent(entry.File, executionID, playErr, outcome.Duration))

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

// ensureBAS ensures the BAS API is available.
// It always attempts to start BAS to ensure it's running, then waits for health.
func (r *Runner) ensureBAS(ctx context.Context) error {
	if r.basClient != nil {
		// Client already configured (likely in tests)
		return r.basClient.WaitForHealth(ctx)
	}

	clientCfg := execution.ClientConfig{
		Timeout:                  r.playbooksConfig.BAS.Timeout(),
		HealthCheckWaitTimeout:   r.playbooksConfig.BAS.HealthCheckWaitTimeout(),
		WorkflowExecutionTimeout: r.playbooksConfig.BAS.WorkflowTimeout(),
	}

	endpoint := strings.TrimSpace(r.playbooksConfig.BAS.Endpoint)
	isCustomEndpoint := endpoint != "" && endpoint != config.DefaultBASEndpoint
	if isCustomEndpoint {
		shared.LogStep(r.logWriter, "using BAS endpoint from config: %s", endpoint)
		r.basClient = execution.NewClientWithConfig(endpoint, clientCfg)
		if err := r.basClient.WaitForHealth(ctx); err != nil {
			return fmt.Errorf("browser-automation-studio unhealthy at %s: %w", endpoint, err)
		}
		return nil
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
		return fmt.Errorf("browser-automation-studio API port unavailable (tried scenario port resolution)")
	}

	apiBase := fmt.Sprintf("http://127.0.0.1:%s/api/v1", apiPort)

	// Create client with configured timeouts
	r.basClient = execution.NewClientWithConfig(apiBase, clientCfg)

	if err := r.basClient.WaitForHealth(ctx); err != nil {
		return fmt.Errorf("browser-automation-studio unhealthy at %s: %w", apiBase, err)
	}

	return nil
}

// loadSeedState loads the seed state JSON file produced by seed scripts.
// This allows test-genie to pass seed values to BAS as initial_params for
// the new namespace-aware variable interpolation system.
func (r *Runner) loadSeedState() (map[string]any, error) {
	seedPath := sharedartifacts.SeedStatePath(r.config.ScenarioDir)
	data, err := os.ReadFile(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse seed state JSON: %w", err)
	}
	return state, nil
}

// basProjectRoot returns the BAS workflow root directory for the current scenario.
// This is typically the scenario's "bas" subdirectory where workflow JSON files are stored.
func (r *Runner) basProjectRoot() string {
	// The BAS workflows are stored under <scenarioDir>/bas/.
	basDir := filepath.Join(r.config.ScenarioDir, "bas")
	if _, err := os.Stat(basDir); err == nil {
		return basDir
	}
	return ""
}
