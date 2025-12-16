package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/requirements"
	workspacepkg "test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

var (
	defaultPhaseTimeout     = phases.DefaultTimeout
	defaultExecutionPresets = map[string][]string{
		"quick":         {"structure", "standards", "docs", "unit"},
		"smoke":         {"structure", "standards", "lint", "docs", "integration"},
		"comprehensive": {"structure", "standards", "dependencies", "lint", "docs", "unit", "integration", "playbooks", "business", "performance"},
	}
	defaultPhaseSortFallback = 1000
)

type (
	PhaseEnvironment = workspacepkg.Environment
	PhaseRunReport   = phases.RunReport
	PhaseName        = phases.Name
	PhaseDescriptor  = phases.Descriptor
	PhaseCatalog     = phases.Catalog
	phaseDefinition  = phases.Definition
	phaseSpec        = phases.Spec
)

const (
	failureClassMisconfiguration  = phases.FailureClassMisconfiguration
	failureClassMissingDependency = phases.FailureClassMissingDependency
	failureClassTimeout           = phases.FailureClassTimeout
	failureClassSystem            = phases.FailureClassSystem
	PhaseStructure                = phases.Structure
	PhaseStandards                = phases.Standards
	PhaseDependencies             = phases.Dependencies
	PhaseLint                     = phases.Lint
	PhaseDocs                     = phases.Docs
	PhaseUnit                     = phases.Unit
	PhaseIntegration              = phases.Integration
	PhaseBusiness                 = phases.Business
	PhasePerformance              = phases.Performance
)

const MaxExecutionHistory = 50

// Default Browserless URL and environment variable name.
const (
	defaultBrowserlessURL = "http://localhost:4110"
	browserlessURLEnvVar  = "BROWSERLESS_URL"
)

// minimal service config to detect runtime ports for auto UI/API URL resolution
type serviceConfig struct {
	Ports struct {
		UI struct {
			EnvVar string `json:"env_var"`
		} `json:"ui"`
		API struct {
			EnvVar string `json:"env_var"`
		} `json:"api"`
	} `json:"ports"`
}

// resolveBrowserlessURL returns the Browserless URL to use, checking in order:
// 1. Explicit value from request
// 2. BROWSERLESS_URL environment variable
// 3. Default localhost URL
func resolveBrowserlessURL(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if url := os.Getenv(browserlessURLEnvVar); url != "" {
		return url
	}
	return defaultBrowserlessURL
}

// detectRuntimeURLs attempts to infer UI/API URLs from the scenario's service config and environment.
// If ports are defined in .vrooli/service.json and corresponding env vars are set, it returns
// http://localhost:<port> for each. Missing values are returned as empty strings.
func detectRuntimeURLs(scenarioDir string) (uiURL, apiURL string) {
	// Default env var names
	uiEnv := "UI_PORT"
	apiEnv := "API_PORT"

	// Try process metadata from vrooli lifecycle (captures assigned ports)
	if home, err := os.UserHomeDir(); err == nil {
		scenario := filepath.Base(scenarioDir)
		processDir := filepath.Join(home, ".vrooli", "processes", "scenarios", scenario)
		type procMeta struct {
			Port int `json:"port"`
		}
		if data, err := os.ReadFile(filepath.Join(processDir, "start-ui.json")); err == nil {
			var meta procMeta
			if json.Unmarshal(data, &meta) == nil && meta.Port > 0 {
				uiURL = fmt.Sprintf("http://localhost:%d", meta.Port)
			}
		}
		if data, err := os.ReadFile(filepath.Join(processDir, "start-api.json")); err == nil {
			var meta procMeta
			if json.Unmarshal(data, &meta) == nil && meta.Port > 0 {
				apiURL = fmt.Sprintf("http://localhost:%d", meta.Port)
			}
		}
		if uiURL != "" && apiURL != "" {
			return uiURL, apiURL
		}
	}

	// Load service.json if present to discover custom env var names
	servicePath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err == nil {
		var svc serviceConfig
		if json.Unmarshal(data, &svc) == nil {
			if strings.TrimSpace(svc.Ports.UI.EnvVar) != "" {
				uiEnv = svc.Ports.UI.EnvVar
			}
			if strings.TrimSpace(svc.Ports.API.EnvVar) != "" {
				apiEnv = svc.Ports.API.EnvVar
			}
		}
	}

	if port := strings.TrimSpace(os.Getenv(uiEnv)); port != "" {
		uiURL = fmt.Sprintf("http://localhost:%s", port)
	}
	if port := strings.TrimSpace(os.Getenv(apiEnv)); port != "" {
		apiURL = fmt.Sprintf("http://localhost:%s", port)
	}
	return uiURL, apiURL
}

// SuiteOrchestrator runs scenario-local test phases without relying on external bash runners.
type SuiteOrchestrator struct {
	scenariosRoot string
	projectRoot   string
	phaseTimeout  time.Duration
	catalog       *phases.Catalog
	requirements  requirements.Syncer
	phaseToggles  *phaseToggleStore
}

// SuiteExecutionRequest configures a single test execution run.
type SuiteExecutionRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Preset       string   `json:"preset,omitempty"`
	Phases       []string `json:"phases,omitempty"`
	Skip         []string `json:"skip,omitempty"`
	FailFast     bool     `json:"failFast"`

	// Runtime URLs for phases that need to connect to running services.
	// If not provided, BrowserlessURL falls back to BROWSERLESS_URL env var or default.
	// UIURL is required for Lighthouse audits; if empty, Lighthouse is skipped.
	UIURL          string `json:"uiUrl,omitempty"`
	APIURL         string `json:"apiUrl,omitempty"`
	BrowserlessURL string `json:"browserlessUrl,omitempty"`
}

// SuiteExecutionResult captures the outcome of a run.
type SuiteExecutionResult struct {
	ExecutionID    uuid.UUID              `json:"executionId,omitempty"`
	SuiteRequestID *uuid.UUID             `json:"suiteRequestId,omitempty"`
	ScenarioName   string                 `json:"scenarioName"`
	StartedAt      time.Time              `json:"startedAt"`
	CompletedAt    time.Time              `json:"completedAt"`
	Success        bool                   `json:"success"`
	PresetUsed     string                 `json:"preset,omitempty"`
	Phases         []PhaseExecutionResult `json:"phases"`
	PhaseSummary   PhaseSummary           `json:"phaseSummary"`
	Warnings       []string               `json:"warnings,omitempty"`
}

type PhaseExecutionResult = phases.ExecutionResult

// PhaseSummary aggregates phase telemetry for quick status surfaces.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}

// ExecutionEventType identifies the kind of streaming event.
type ExecutionEventType string

const (
	EventPhaseStart  ExecutionEventType = "phase_start"
	EventPhaseEnd    ExecutionEventType = "phase_end"
	EventObservation ExecutionEventType = "observation"
	EventProgress    ExecutionEventType = "progress"
	EventComplete    ExecutionEventType = "complete"
)

// ExecutionEvent represents a streaming event during suite execution.
type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`

	// For phase_start events
	Phase      string `json:"phase,omitempty"`
	PhaseIndex int    `json:"phaseIndex,omitempty"`
	PhaseTotal int    `json:"phaseTotal,omitempty"`

	// For phase_end events
	Status          string `json:"status,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	Error           string `json:"error,omitempty"`

	// For observation events
	Message string `json:"message,omitempty"`

	// For complete events
	Result *SuiteExecutionResult `json:"result,omitempty"`
}

// ExecutionEventCallback is called for each event during streaming execution.
type ExecutionEventCallback func(event ExecutionEvent)

func NewSuiteOrchestrator(scenariosRoot string) (*SuiteOrchestrator, error) {
	if strings.TrimSpace(scenariosRoot) == "" {
		return nil, fmt.Errorf("scenarios root path is required")
	}
	absRoot, err := filepath.Abs(scenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid scenarios root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("scenarios root not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenarios root must be a directory: %s", absRoot)
	}
	return &SuiteOrchestrator{
		scenariosRoot: absRoot,
		projectRoot:   filepath.Dir(absRoot),
		phaseTimeout:  defaultPhaseTimeout,
		catalog:       phases.NewDefaultCatalog(defaultPhaseTimeout),
		requirements:  requirements.NewNodeSyncer(filepath.Dir(absRoot)),
		phaseToggles:  newPhaseToggleStore(filepath.Dir(absRoot)),
	}, nil
}

// Execute performs a phased test run and returns the recorded result.
func (o *SuiteOrchestrator) Execute(ctx context.Context, req SuiteExecutionRequest) (*SuiteExecutionResult, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}

	ws, err := workspacepkg.New(o.scenariosRoot, scenario)
	if err != nil {
		return nil, err
	}

	// Configure runtime URLs for phases that connect to running services
	autoUI, autoAPI := detectRuntimeURLs(ws.ScenarioDir)
	uiURL := req.UIURL
	if uiURL == "" {
		uiURL = autoUI
	}
	apiURL := req.APIURL
	if apiURL == "" {
		apiURL = autoAPI
	}
	ws.SetRuntimeURLs(uiURL, apiURL, resolveBrowserlessURL(req.BrowserlessURL))
	env := ws.Environment()

	config, err := workspacepkg.LoadTestingConfig(env.ScenarioDir)
	if err != nil {
		return nil, err
	}

	plan, err := o.buildPhasePlan(env, config, req)
	if err != nil {
		return nil, err
	}

	runID := time.Now().UTC().Format("20060102-150405")

	if err := sharedartifacts.EnsureCoverageStructure(env.ScenarioDir); err != nil {
		return nil, err
	}
	runLogDir := sharedartifacts.RunLogsDir(env.ScenarioDir, runID)
	if err := os.MkdirAll(runLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create run log directory: %w", err)
	}

	result := &SuiteExecutionResult{
		ScenarioName: scenario,
		StartedAt:    time.Now().UTC(),
		PresetUsed:   plan.PresetUsed,
		Warnings:     buildPlanWarnings(plan),
	}

	phaseResults, anyFailure := o.runSelectedPhases(ctx, env, runLogDir, plan.Selected, req.FailFast, buildPhaseWarningMap(plan))

	result.CompletedAt = time.Now().UTC()
	result.Success = !anyFailure
	result.Phases = phaseResults
	result.PhaseSummary = SummarizePhases(phaseResults)

	if err := o.writeLatestManifest(env.ScenarioDir, runLogDir, runID, result.StartedAt, result.CompletedAt, phaseResults); err != nil {
		log.Printf("failed to write latest manifest: %v", err)
	}

	o.syncRequirementsIfNeeded(ctx, env, config, req, plan, phaseResults)

	return result, nil
}

// ExecuteWithEvents performs a phased test run while streaming events via callback.
// This enables real-time progress reporting for SSE/WebSocket clients.
func (o *SuiteOrchestrator) ExecuteWithEvents(ctx context.Context, req SuiteExecutionRequest, emit ExecutionEventCallback) (*SuiteExecutionResult, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}

	ws, err := workspacepkg.New(o.scenariosRoot, scenario)
	if err != nil {
		return nil, err
	}

	// Configure runtime URLs for phases that connect to running services
	autoUI, autoAPI := detectRuntimeURLs(ws.ScenarioDir)
	uiURL := req.UIURL
	if uiURL == "" {
		uiURL = autoUI
	}
	apiURL := req.APIURL
	if apiURL == "" {
		apiURL = autoAPI
	}
	ws.SetRuntimeURLs(uiURL, apiURL, resolveBrowserlessURL(req.BrowserlessURL))
	env := ws.Environment()

	config, err := workspacepkg.LoadTestingConfig(env.ScenarioDir)
	if err != nil {
		return nil, err
	}

	plan, err := o.buildPhasePlan(env, config, req)
	if err != nil {
		return nil, err
	}

	runID := time.Now().UTC().Format("20060102-150405")

	if err := sharedartifacts.EnsureCoverageStructure(env.ScenarioDir); err != nil {
		return nil, err
	}
	runLogDir := sharedartifacts.RunLogsDir(env.ScenarioDir, runID)
	if err := os.MkdirAll(runLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create run log directory: %w", err)
	}

	result := &SuiteExecutionResult{
		ScenarioName: scenario,
		StartedAt:    time.Now().UTC(),
		PresetUsed:   plan.PresetUsed,
		Warnings:     buildPlanWarnings(plan),
	}

	phaseResults, anyFailure := o.runSelectedPhasesWithEvents(ctx, env, runLogDir, plan.Selected, req.FailFast, emit, buildPhaseWarningMap(plan))

	result.CompletedAt = time.Now().UTC()
	result.Success = !anyFailure
	result.Phases = phaseResults
	result.PhaseSummary = SummarizePhases(phaseResults)

	// Emit completion event
	if emit != nil {
		emit(ExecutionEvent{
			Type:      EventComplete,
			Timestamp: time.Now(),
			Result:    result,
		})
	}

	if err := o.writeLatestManifest(env.ScenarioDir, runLogDir, runID, result.StartedAt, result.CompletedAt, phaseResults); err != nil {
		log.Printf("failed to write latest manifest: %v", err)
	}

	o.syncRequirementsIfNeeded(ctx, env, config, req, plan, phaseResults)

	return result, nil
}

func (o *SuiteOrchestrator) runSelectedPhases(
	ctx context.Context,
	env workspacepkg.Environment,
	runLogDir string,
	defs []phases.Definition,
	failFast bool,
	warnings map[string][]phases.Observation,
) ([]PhaseExecutionResult, bool) {
	if len(defs) == 0 {
		return nil, false
	}
	results := make([]PhaseExecutionResult, 0, len(defs))
	anyFailure := false
	for _, phase := range defs {
		phaseResult := o.runPhase(ctx, env, runLogDir, phase, warnings[phase.Name.Key()])
		if phaseResult.Status != "passed" {
			anyFailure = true
		}
		results = append(results, phaseResult)
		if failFast && phaseResult.Status == "failed" {
			break
		}
	}
	return results, anyFailure
}

func (o *SuiteOrchestrator) runSelectedPhasesWithEvents(
	ctx context.Context,
	env workspacepkg.Environment,
	runLogDir string,
	defs []phases.Definition,
	failFast bool,
	emit ExecutionEventCallback,
	warnings map[string][]phases.Observation,
) ([]PhaseExecutionResult, bool) {
	if len(defs) == 0 {
		return nil, false
	}
	results := make([]PhaseExecutionResult, 0, len(defs))
	anyFailure := false
	total := len(defs)

	for idx, phase := range defs {
		// Emit phase start event
		if emit != nil {
			emit(ExecutionEvent{
				Type:       EventPhaseStart,
				Timestamp:  time.Now(),
				Phase:      phase.Name.String(),
				PhaseIndex: idx + 1,
				PhaseTotal: total,
			})
		}

		phaseResult := o.runPhaseWithEvents(ctx, env, runLogDir, phase, emit, warnings[phase.Name.Key()])

		// Emit phase end event
		if emit != nil {
			emit(ExecutionEvent{
				Type:            EventPhaseEnd,
				Timestamp:       time.Now(),
				Phase:           phase.Name.String(),
				Status:          phaseResult.Status,
				DurationSeconds: phaseResult.DurationSeconds,
				Error:           phaseResult.Error,
			})
		}

		if phaseResult.Status != "passed" {
			anyFailure = true
		}
		results = append(results, phaseResult)
		if failFast && phaseResult.Status == "failed" {
			break
		}
	}
	return results, anyFailure
}

func (o *SuiteOrchestrator) syncRequirementsIfNeeded(
	ctx context.Context,
	env workspacepkg.Environment,
	cfg *workspacepkg.Config,
	req SuiteExecutionRequest,
	plan *phasePlan,
	phaseResults []PhaseExecutionResult,
) {
	if o.requirements == nil {
		return
	}
	decision := newRequirementsSyncDecision(cfg, plan, phaseResults)
	if !decision.Execute {
		if decision.Reason != "" {
			log.Printf("requirements sync skipped: %s", decision.Reason)
		}
		return
	}
	if decision.Forced && decision.Reason != "" {
		log.Printf("forcing requirements sync despite: %s", decision.Reason)
	}
	// Allow partial sync on early abort (fail-fast). The syncer only updates
	// requirements for phases that actually ran; unaffected requirements keep
	// their previous status. This ensures PRD checkboxes stay honest even when
	// the test suite exits early.
	if len(phaseResults) < len(plan.Selected) {
		log.Printf("requirements sync proceeding with partial results: recorded %d of %d phases (fail-fast likely)", len(phaseResults), len(plan.Selected))
	}
	history := buildCommandHistory(req, plan)
	input := requirements.SyncInput{
		ScenarioName:     env.ScenarioName,
		ScenarioDir:      env.ScenarioDir,
		PhaseDefinitions: plan.Definitions,
		PhaseResults:     phaseResults,
		CommandHistory:   history,
	}
	if err := o.requirements.Sync(ctx, input); err != nil {
		log.Printf("requirements sync skipped: %v", err)
	}
}

func (o *SuiteOrchestrator) discoverPhaseDefinitions(env workspacepkg.Environment) ([]phases.Definition, error) {
	phaseDir := filepath.Join(env.TestDir, "phases")
	var entries []os.DirEntry
	if dirEntries, err := os.ReadDir(phaseDir); err == nil {
		entries = dirEntries
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read phase directory: %w", err)
	}
	definitions := make(map[string]phases.Definition)
	if o.catalog != nil {
		for _, spec := range o.catalog.All() {
			definitions[spec.Name.Key()] = phases.Definition{
				Name:     spec.Name,
				Runner:   spec.Runner,
				Timeout:  spec.DefaultTimeout,
				Optional: spec.Optional,
			}
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "test-") || !strings.HasSuffix(name, ".sh") {
			continue
		}
		phaseName := strings.TrimSuffix(strings.TrimPrefix(name, "test-"), ".sh")
		normalized, ok := phases.NormalizeName(phaseName)
		if !ok {
			continue
		}
		if _, exists := definitions[normalized.Key()]; exists {
			continue
		}
		definitions[normalized.Key()] = phases.Definition{
			Name:    normalized,
			Runner:  o.scriptPhaseRunner(filepath.Join(phaseDir, name)),
			Timeout: o.phaseTimeout,
		}
	}
	var defs []phases.Definition
	for _, def := range definitions {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		left := phaseSortValue(defs[i].Name, o.catalog)
		right := phaseSortValue(defs[j].Name, o.catalog)
		if left == right {
			return defs[i].Name.String() < defs[j].Name.String()
		}
		return left < right
	})
	return defs, nil
}

type phaseSelectionNotices struct {
	Skipped  []phaseDisableNotice
	Explicit []phaseDisableNotice
}

func selectPhases(defs []phases.Definition, presets map[string][]string, req SuiteExecutionRequest, toggles PhaseToggleConfig) ([]phases.Definition, string, phaseSelectionNotices, error) {
	if len(defs) == 0 {
		return nil, "", phaseSelectionNotices{}, nil
	}
	index := make(map[string]phases.Definition, len(defs))
	for _, def := range defs {
		index[def.Name.Key()] = def
	}

	desired, presetUsed, err := resolveDesiredPhaseList(req, presets)
	if err != nil {
		return nil, "", phaseSelectionNotices{}, err
	}

	var notices phaseSelectionNotices
	var resolved []phases.Definition
	isDisabled := func(name string) (PhaseToggle, bool) {
		if toggles.Phases == nil {
			return PhaseToggle{}, false
		}
		toggle, ok := toggles.Phases[name]
		return toggle, ok && toggle.Disabled
	}

	if len(desired) == 0 {
		for _, def := range defs {
			if toggle, disabled := isDisabled(def.Name.Key()); disabled {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
				continue
			}
			resolved = append(resolved, def)
		}
	} else {
		explicitRequest := len(req.Phases) > 0
		for _, phase := range desired {
			normalized := normalizePhaseName(phase)
			if normalized == "" {
				continue
			}
			def, ok := index[normalized]
			if !ok {
				return nil, "", phaseSelectionNotices{}, shared.NewValidationError(fmt.Sprintf("phase '%s' is not defined", phase))
			}
			if toggle, disabled := isDisabled(def.Name.Key()); disabled && !explicitRequest {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
				continue
			}
			if toggle, disabled := isDisabled(def.Name.Key()); disabled && explicitRequest {
				notices.Explicit = append(notices.Explicit, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
			}
			resolved = append(resolved, def)
		}
	}

	return applySkipFilters(resolved, req.Skip), presetUsed, notices, nil
}

func (o *SuiteOrchestrator) scriptPhaseRunner(scriptPath string) phases.Runner {
	return func(ctx context.Context, env workspacepkg.Environment, logWriter io.Writer) phases.RunReport {
		cmd := exec.CommandContext(ctx, "bash", scriptPath)
		cmd.Dir = env.TestDir
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("TEST_GENIE_SCENARIO_DIR=%s", env.ScenarioDir),
			fmt.Sprintf("TEST_GENIE_APP_ROOT=%s", env.AppRoot),
		)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		return phases.RunReport{Err: cmd.Run()}
	}
}

func (o *SuiteOrchestrator) runPhase(ctx context.Context, env workspacepkg.Environment, runLogDir string, def phases.Definition, preObservations []phases.Observation) PhaseExecutionResult {
	start := time.Now()
	logPath := filepath.Join(runLogDir, fmt.Sprintf("%s.log", def.Name.String()))
	logFile, err := os.Create(logPath)
	if err != nil {
		return PhaseExecutionResult{
			Name:            def.Name.String(),
			Status:          "failed",
			DurationSeconds: 0,
			LogPath:         logPath,
			Error:           fmt.Sprintf("failed to create log file: %v", err),
		}
	}
	defer logFile.Close()

	timeout := def.Timeout
	if timeout <= 0 {
		timeout = o.phaseTimeout
	}

	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if env.AppRoot == "" {
		env.AppRoot = workspacepkg.AppRootFromScenario(env.ScenarioDir)
	}

	if len(preObservations) > 0 {
		for _, obs := range preObservations {
			if obsStr := obs.String(); obsStr != "" {
				_, _ = fmt.Fprintln(logFile, obsStr)
			}
		}
	}

	report := def.Runner(phaseCtx, env, logFile)
	runErr := report.Err
	duration := int(math.Ceil(time.Since(start).Seconds()))
	if duration < 0 {
		duration = 0
	}

	status := "passed"
	errMsg := ""
	classification := report.FailureClassification
	remediation := report.Remediation

	if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(phaseCtx.Err(), context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("phase timed out after %s", timeout)
			classification = phases.FailureClassTimeout
			if remediation == "" {
				remediation = "Increase the timeout or break the phase into smaller steps."
			}
		}
		if classification == "" {
			classification = phases.FailureClassSystem
		}
		if remediation == "" {
			remediation = "Refer to the phase logs to triage the failure."
		}
	}

	relLog, relErr := filepath.Rel(o.projectRoot, logPath)
	if relErr == nil {
		logPath = relLog
	}

	result := PhaseExecutionResult{
		Name:            def.Name.String(),
		Status:          status,
		DurationSeconds: duration,
		LogPath:         logPath,
		Error:           errMsg,
		Classification:  classification,
		Remediation:     remediation,
		Observations:    report.Observations,
	}
	if len(preObservations) > 0 {
		result.Observations = append(preObservations, result.Observations...)
	}
	appendObservationsToLog(logPath, o.projectRoot, result.Observations)
	return result
}

// runPhaseWithEvents is like runPhase but emits observation events during execution.
func (o *SuiteOrchestrator) runPhaseWithEvents(ctx context.Context, env workspacepkg.Environment, runLogDir string, def phases.Definition, emit ExecutionEventCallback, preObservations []phases.Observation) PhaseExecutionResult {
	start := time.Now()
	logPath := filepath.Join(runLogDir, fmt.Sprintf("%s.log", def.Name.String()))
	logFile, err := os.Create(logPath)
	if err != nil {
		return PhaseExecutionResult{
			Name:            def.Name.String(),
			Status:          "failed",
			DurationSeconds: 0,
			LogPath:         logPath,
			Error:           fmt.Sprintf("failed to create log file: %v", err),
		}
	}
	defer logFile.Close()

	timeout := def.Timeout
	if timeout <= 0 {
		timeout = o.phaseTimeout
	}

	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if env.AppRoot == "" {
		env.AppRoot = workspacepkg.AppRootFromScenario(env.ScenarioDir)
	}

	// Create a writer that always captures full logs to disk, but only streams
	// select lines as observations. MultiWriter duplicates writes so the stream
	// path can stay minimal while the log captures everything.
	var logWriter io.Writer = logFile
	if emit != nil {
		logWriter = io.MultiWriter(
			logFile,
			&observationEmitter{
				underlying: io.Discard,
				emit:       emit,
				phase:      def.Name.String(),
			},
		)
	}

	if len(preObservations) > 0 {
		for _, obs := range preObservations {
			if obsStr := obs.String(); obsStr != "" {
				_, _ = fmt.Fprintln(logWriter, obsStr)
			}
		}
	}

	report := def.Runner(phaseCtx, env, logWriter)
	runErr := report.Err
	duration := int(math.Ceil(time.Since(start).Seconds()))
	if duration < 0 {
		duration = 0
	}

	// Observations are included in the phase result and rendered by the CLI
	// after phase completion, so we don't stream them here to avoid duplication.

	status := "passed"
	errMsg := ""
	classification := report.FailureClassification
	remediation := report.Remediation

	if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(phaseCtx.Err(), context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("phase timed out after %s", timeout)
			classification = phases.FailureClassTimeout
			if remediation == "" {
				remediation = "Increase the timeout or break the phase into smaller steps."
			}
		}
		if classification == "" {
			classification = phases.FailureClassSystem
		}
		if remediation == "" {
			remediation = "Refer to the phase logs to triage the failure."
		}
	}

	relLog, relErr := filepath.Rel(o.projectRoot, logPath)
	if relErr == nil {
		logPath = relLog
	}

	result := PhaseExecutionResult{
		Name:            def.Name.String(),
		Status:          status,
		DurationSeconds: duration,
		LogPath:         logPath,
		Error:           errMsg,
		Classification:  classification,
		Remediation:     remediation,
		Observations:    report.Observations,
	}
	if len(preObservations) > 0 {
		result.Observations = append(preObservations, result.Observations...)
	}
	return result
}

func (o *SuiteOrchestrator) writeLatestManifest(scenarioDir, runLogDir, runID string, startedAt, completedAt time.Time, results []PhaseExecutionResult) error {
	latestDir := sharedartifacts.LatestDirPath(scenarioDir)
	if err := os.MkdirAll(latestDir, 0o755); err != nil {
		return fmt.Errorf("failed to create latest dir: %w", err)
	}

	logs := make(map[string]string, len(results))
	phaseEntries := make([]map[string]any, 0, len(results))

	for _, res := range results {
		logAbs := filepath.Join(runLogDir, fmt.Sprintf("%s.log", res.Name))
		logRel := sharedartifacts.RelPath(scenarioDir, logAbs)
		logs[res.Name] = logRel

		phaseEntries = append(phaseEntries, map[string]any{
			"name":             res.Name,
			"status":           res.Status,
			"duration_seconds": res.DurationSeconds,
			"log":              logRel,
		})

		if err := updateLatestPointer(latestDir, fmt.Sprintf("%s.log", res.Name), logAbs); err != nil {
			return err
		}
	}

	manifest := map[string]any{
		"run_id":       runID,
		"started_at":   startedAt.UTC().Format(time.RFC3339),
		"completed_at": completedAt.UTC().Format(time.RFC3339),
		"logs":         logs,
		"phases":       phaseEntries,
	}

	writer := sharedartifacts.NewBaseWriter(scenarioDir, filepath.Base(scenarioDir))
	return writer.WriteJSON(sharedartifacts.LatestManifestPath(scenarioDir), manifest)
}

func updateLatestPointer(latestDir, linkName, target string) error {
	linkPath := filepath.Join(latestDir, linkName)
	_ = os.Remove(linkPath)
	if err := os.Symlink(target, linkPath); err != nil {
		// Fallback: write target path into a small text file
		if writeErr := os.WriteFile(linkPath, []byte(target), 0o644); writeErr != nil {
			return fmt.Errorf("failed to create latest pointer for %s: %v (symlink error: %v)", linkName, writeErr, err)
		}
	}
	return nil
}

// observationEmitter wraps an io.Writer and emits observation events for lines with markers.
type observationEmitter struct {
	underlying io.Writer
	emit       ExecutionEventCallback
	phase      string
	buffer     []byte
}

func (e *observationEmitter) Write(p []byte) (n int, err error) {
	// Write to underlying log
	n, err = e.underlying.Write(p)
	if err != nil || e.emit == nil {
		return n, err
	}

	// Buffer and scan for complete lines with observation markers
	e.buffer = append(e.buffer, p...)
	for {
		idx := -1
		for i, b := range e.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}

		line := string(e.buffer[:idx])
		e.buffer = e.buffer[idx+1:]

		// Emit observation events for significant lines
		// Look for common markers that indicate progress
		if e.isSignificantLine(line) {
			e.emit(ExecutionEvent{
				Type:      EventObservation,
				Timestamp: time.Now(),
				Phase:     e.phase,
				Message:   line,
			})
		}
	}

	return n, nil
}

// isSignificantLine determines if a log line should be emitted as an observation.
func (e *observationEmitter) isSignificantLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Only emit lines that use explicit structured markers intended for the stream.
	// Raw command output still goes to the log file but will not be streamed.
	significantPrefixes := []string{
		"[SUCCESS]", "[ERROR]",
		"✅", "❌", "🧪", "SECTION",
	}
	for _, prefix := range significantPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	return false
}

func SummarizePhases(phases []PhaseExecutionResult) PhaseSummary {
	summary := PhaseSummary{}
	for _, phase := range phases {
		summary.Total++
		if phase.DurationSeconds > 0 {
			summary.DurationSeconds += phase.DurationSeconds
		}
		summary.ObservationCount += len(phase.Observations)
		switch strings.ToLower(phase.Status) {
		case "passed":
			summary.Passed++
		case "failed":
			summary.Failed++
		}
	}
	return summary
}

func phaseSortValue(name phases.Name, catalog *phases.Catalog) int {
	if catalog != nil {
		if weight, ok := catalog.Weight(name); ok {
			return weight
		}
	}
	return defaultPhaseSortFallback
}

func (o *SuiteOrchestrator) loadPresets(testDir string, cfg *workspacepkg.Config, allowed map[string]struct{}) map[string][]string {
	presets := make(map[string][]string)
	configPath := filepath.Join(testDir, "presets.json")
	applyPresets := func(source map[string][]string, allowDelete bool, replace bool) {
		for key, phases := range source {
			name := normalizePhaseName(key)
			if name == "" {
				continue
			}
			filtered := filterPresetPhases(phases, allowed)
			if len(filtered) == 0 {
				if allowDelete {
					delete(presets, name)
				}
				continue
			}
			if _, exists := presets[name]; exists && !replace {
				continue
			}
			presets[name] = filtered
		}
	}

	if raw, err := os.ReadFile(configPath); err == nil {
		var parsed map[string][]string
		if err := json.Unmarshal(raw, &parsed); err == nil {
			applyPresets(parsed, false, true)
		}
	}

	if cfg != nil && len(cfg.Presets) > 0 {
		applyPresets(cfg.Presets, true, true)
	}

	applyPresets(defaultExecutionPresets, false, false)

	return presets
}

// DescribePhases exposes registered Go-native phases for HTTP clients.
func (o *SuiteOrchestrator) DescribePhases() []phases.Descriptor {
	if o == nil || o.catalog == nil {
		return nil
	}
	return o.catalog.Descriptors()
}

// GlobalPhaseToggles returns the persisted global phase toggle configuration.
func (o *SuiteOrchestrator) GlobalPhaseToggles() (PhaseToggleConfig, error) {
	if o == nil || o.phaseToggles == nil {
		return PhaseToggleConfig{Phases: map[string]PhaseToggle{}}, nil
	}
	return o.phaseToggles.Load()
}

// SaveGlobalPhaseToggles persists the provided toggle configuration, preserving
// the original AddedAt timestamp when a phase remains disabled.
func (o *SuiteOrchestrator) SaveGlobalPhaseToggles(cfg PhaseToggleConfig) (PhaseToggleConfig, error) {
	if o == nil || o.phaseToggles == nil {
		return normalizePhaseToggleConfig(cfg, time.Now().UTC()), nil
	}
	current, err := o.phaseToggles.Load()
	if err != nil {
		return PhaseToggleConfig{}, err
	}

	now := time.Now().UTC()
	normalized := normalizePhaseToggleConfig(cfg, now)
	if normalized.Phases == nil {
		normalized.Phases = map[string]PhaseToggle{}
	}

	for name, existing := range current.Phases {
		updated, ok := normalized.Phases[name]
		if !ok {
			// Missing entries mean "enabled by default"—skip
			continue
		}
		if updated.Disabled && updated.AddedAt.IsZero() {
			if !existing.AddedAt.IsZero() {
				updated.AddedAt = existing.AddedAt
			} else {
				updated.AddedAt = now
			}
		}
		normalized.Phases[name] = updated
	}

	return o.phaseToggles.Save(normalized)
}

func (o *SuiteOrchestrator) applyTestingConfig(defs []phases.Definition, cfg *workspacepkg.Config) []phases.Definition {
	if cfg == nil || len(cfg.Phases) == 0 {
		return defs
	}
	var configured []phases.Definition
	for _, def := range defs {
		name := def.Name.Key()
		settings, ok := cfg.Phases[name]
		if ok {
			if settings.Enabled != nil && !*settings.Enabled {
				continue
			}
			if settings.Timeout > 0 {
				def.Timeout = settings.Timeout
			}
		}
		configured = append(configured, def)
	}
	return configured
}

func filterPresetPhases(phases []string, allowed map[string]struct{}) []string {
	if len(phases) == 0 || len(allowed) == 0 {
		return nil
	}
	var filtered []string
	seen := make(map[string]struct{}, len(phases))
	for _, phase := range phases {
		normalized := normalizePhaseName(phase)
		if normalized == "" {
			continue
		}
		if _, exists := allowed[normalized]; !exists {
			continue
		}
		if _, present := seen[normalized]; present {
			continue
		}
		seen[normalized] = struct{}{}
		filtered = append(filtered, normalized)
	}
	return filtered
}

func buildCommandHistory(req SuiteExecutionRequest, plan *phasePlan) []string {
	var history []string
	var descriptor []string
	if req.ScenarioName != "" {
		descriptor = append(descriptor, fmt.Sprintf("scenario=%s", req.ScenarioName))
	}
	if plan != nil && plan.PresetUsed != "" {
		descriptor = append(descriptor, fmt.Sprintf("preset=%s", plan.PresetUsed))
	}
	if len(req.Phases) > 0 {
		descriptor = append(descriptor, fmt.Sprintf("phases=%s", strings.Join(req.Phases, ",")))
	}
	if len(req.Skip) > 0 {
		descriptor = append(descriptor, fmt.Sprintf("skip=%s", strings.Join(req.Skip, ",")))
	}
	if req.FailFast {
		descriptor = append(descriptor, "failFast=true")
	}
	if len(descriptor) > 0 {
		history = append(history, strings.Join(descriptor, " "))
	}
	if plan != nil && len(plan.Selected) > 0 {
		var names []string
		for _, def := range plan.Selected {
			if def.Name.IsZero() {
				continue
			}
			names = append(names, def.Name.String())
		}
		if len(names) > 0 {
			history = append(history, fmt.Sprintf("phase-order:%s", strings.Join(names, ",")))
		}
	}
	return history
}

func normalizePhaseName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func resolveDesiredPhaseList(req SuiteExecutionRequest, presets map[string][]string) ([]string, string, error) {
	if len(req.Phases) > 0 {
		return req.Phases, "", nil
	}
	if req.Preset == "" {
		return nil, "", nil
	}
	name := normalizePhaseName(req.Preset)
	if name == "" {
		return nil, "", shared.NewValidationError(fmt.Sprintf("preset '%s' is not defined", req.Preset))
	}
	phases, ok := presets[name]
	if !ok {
		return nil, "", shared.NewValidationError(fmt.Sprintf("preset '%s' is not defined", req.Preset))
	}
	return phases, name, nil
}

func applySkipFilters(selected []phases.Definition, skip []string) []phases.Definition {
	if len(selected) == 0 || len(skip) == 0 {
		return selected
	}
	skipSet := make(map[string]struct{}, len(skip))
	for _, phase := range skip {
		if normalized := normalizePhaseName(phase); normalized != "" {
			skipSet[normalized] = struct{}{}
		}
	}
	var filtered []phases.Definition
	for _, def := range selected {
		if _, skip := skipSet[def.Name.Key()]; skip {
			continue
		}
		filtered = append(filtered, def)
	}
	return filtered
}

func buildPlanWarnings(plan *phasePlan) []string {
	if plan == nil {
		return nil
	}
	var warnings []string
	for _, notice := range plan.DisabledByDefault {
		warnings = append(warnings, formatSkipWarning(notice))
	}
	for _, notice := range plan.ExplicitDisabled {
		warnings = append(warnings, formatExplicitWarning(notice))
	}
	return warnings
}

func buildPhaseWarningMap(plan *phasePlan) map[string][]phases.Observation {
	warnings := make(map[string][]phases.Observation)
	if plan == nil {
		return warnings
	}
	for _, notice := range plan.ExplicitDisabled {
		text := formatExplicitWarning(notice)
		warnings[normalizePhaseName(notice.Name)] = []phases.Observation{phases.NewWarningObservation(text)}
	}
	return warnings
}

func formatSkipWarning(notice phaseDisableNotice) string {
	base := fmt.Sprintf("Phase '%s' is globally disabled and was skipped by default.", notice.Name)
	return base + formatToggleContext(notice.Toggle)
}

func formatExplicitWarning(notice phaseDisableNotice) string {
	base := fmt.Sprintf("Phase '%s' is globally disabled but was explicitly requested; results may be unstable.", notice.Name)
	return base + formatToggleContext(notice.Toggle)
}

func formatToggleContext(toggle PhaseToggle) string {
	var parts []string
	if toggle.Reason != "" {
		parts = append(parts, fmt.Sprintf("reason: %s", toggle.Reason))
	}
	if toggle.Owner != "" {
		parts = append(parts, fmt.Sprintf("owner: %s", toggle.Owner))
	}
	if !toggle.AddedAt.IsZero() {
		parts = append(parts, fmt.Sprintf("disabledAt: %s", toggle.AddedAt.UTC().Format("2006-01-02")))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

func appendObservationsToLog(logPath string, projectRoot string, observations []phases.Observation) {
	if logPath == "" || len(observations) == 0 {
		return
	}
	target := logPath
	if !filepath.IsAbs(target) {
		target = filepath.Join(projectRoot, target)
	}
	f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "OBSERVATIONS:")
	for _, obs := range observations {
		if text := obs.String(); strings.TrimSpace(text) != "" {
			fmt.Fprintln(f, " -", text)
		}
	}
}
