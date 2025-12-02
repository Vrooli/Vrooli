package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
)

var (
	defaultPhaseTimeout     = phases.DefaultTimeout
	defaultExecutionPresets = map[string][]string{
		"quick":         {"structure", "unit"},
		"smoke":         {"structure", "integration"},
		"comprehensive": {"structure", "dependencies", "unit", "integration", "playbooks", "business", "performance"},
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
	PhaseDependencies             = phases.Dependencies
	PhaseUnit                     = phases.Unit
	PhaseIntegration              = phases.Integration
	PhaseBusiness                 = phases.Business
	PhasePerformance              = phases.Performance
)

const MaxExecutionHistory = 50

// SuiteOrchestrator runs scenario-local test phases without relying on external bash runners.
type SuiteOrchestrator struct {
	scenariosRoot string
	projectRoot   string
	phaseTimeout  time.Duration
	catalog       *phases.Catalog
	requirements  requirements.Syncer
}

// SuiteExecutionRequest configures a single test execution run.
type SuiteExecutionRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Preset       string   `json:"preset,omitempty"`
	Phases       []string `json:"phases,omitempty"`
	Skip         []string `json:"skip,omitempty"`
	FailFast     bool     `json:"failFast"`
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
	env := ws.Environment()

	config, err := workspacepkg.LoadTestingConfig(env.ScenarioDir)
	if err != nil {
		return nil, err
	}

	plan, err := o.buildPhasePlan(env, config, req)
	if err != nil {
		return nil, err
	}

	artifactDir, err := ws.EnsureArtifactDir()
	if err != nil {
		return nil, err
	}

	result := &SuiteExecutionResult{
		ScenarioName: scenario,
		StartedAt:    time.Now().UTC(),
		PresetUsed:   plan.PresetUsed,
	}

	phaseResults, anyFailure := o.runSelectedPhases(ctx, env, artifactDir, plan.Selected, req.FailFast)

	result.CompletedAt = time.Now().UTC()
	result.Success = !anyFailure
	result.Phases = phaseResults
	result.PhaseSummary = SummarizePhases(phaseResults)

	o.syncRequirementsIfNeeded(ctx, env, config, req, plan, phaseResults)

	return result, nil
}

func (o *SuiteOrchestrator) runSelectedPhases(
	ctx context.Context,
	env workspacepkg.Environment,
	artifactDir string,
	defs []phases.Definition,
	failFast bool,
) ([]PhaseExecutionResult, bool) {
	if len(defs) == 0 {
		return nil, false
	}
	results := make([]PhaseExecutionResult, 0, len(defs))
	anyFailure := false
	for _, phase := range defs {
		phaseResult := o.runPhase(ctx, env, artifactDir, phase)
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
	if len(phaseResults) != len(plan.Selected) {
		log.Printf("requirements sync skipped: recorded %d phase results but expected %d", len(phaseResults), len(plan.Selected))
		return
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
	entries, err := os.ReadDir(phaseDir)
	if err != nil {
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

func selectPhases(defs []phases.Definition, presets map[string][]string, req SuiteExecutionRequest) ([]phases.Definition, string, error) {
	if len(defs) == 0 {
		return nil, "", nil
	}
	index := make(map[string]phases.Definition, len(defs))
	for _, def := range defs {
		index[def.Name.Key()] = def
	}

	desired, presetUsed, err := resolveDesiredPhaseList(req, presets)
	if err != nil {
		return nil, "", err
	}

	var resolved []phases.Definition
	if len(desired) == 0 {
		resolved = append(resolved, defs...)
	} else {
		for _, phase := range desired {
			normalized := normalizePhaseName(phase)
			if normalized == "" {
				continue
			}
			def, ok := index[normalized]
			if !ok {
				return nil, "", shared.NewValidationError(fmt.Sprintf("phase '%s' is not defined", phase))
			}
			resolved = append(resolved, def)
		}
	}

	return applySkipFilters(resolved, req.Skip), presetUsed, nil
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

func (o *SuiteOrchestrator) runPhase(ctx context.Context, env workspacepkg.Environment, artifactDir string, def phases.Definition) PhaseExecutionResult {
	start := time.Now()
	logPath := filepath.Join(artifactDir, fmt.Sprintf("%s-%s.log", start.UTC().Format("20060102-150405"), def.Name.String()))
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

	report := def.Runner(phaseCtx, env, logFile)
	runErr := report.Err
	duration := int(time.Since(start).Seconds())
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

	return PhaseExecutionResult{
		Name:            def.Name.String(),
		Status:          status,
		DurationSeconds: duration,
		LogPath:         logPath,
		Error:           errMsg,
		Classification:  classification,
		Remediation:     remediation,
		Observations:    report.Observations,
	}
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
