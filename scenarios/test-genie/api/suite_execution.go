package main

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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	defaultPhaseTimeout     = 15 * time.Minute
	defaultExecutionPresets = map[string][]string{
		"quick":         {"structure", "unit"},
		"smoke":         {"structure", "integration"},
		"comprehensive": {"structure", "dependencies", "unit", "integration", "business", "performance"},
	}
	validScenarioName = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	goPhaseOrdering   = map[string]int{
		"structure":    0,
		"dependencies": 10,
		"unit":         20,
		"integration":  30,
		"business":     40,
		"performance":  50,
	}
)

const maxExecutionHistory = 50

// SuiteOrchestrator runs scenario-local test phases without relying on external bash runners.
type SuiteOrchestrator struct {
	scenariosRoot string
	projectRoot   string
	phaseTimeout  time.Duration
	goPhases      map[string]phaseRunnerFunc
	requirements  requirementsSyncer
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

// PhaseExecutionResult captures per-phase outcome information.
type PhaseExecutionResult struct {
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	DurationSeconds int      `json:"durationSeconds"`
	LogPath         string   `json:"logPath"`
	Error           string   `json:"error,omitempty"`
	Classification  string   `json:"classification,omitempty"`
	Remediation     string   `json:"remediation,omitempty"`
	Observations    []string `json:"observations,omitempty"`
}

// PhaseEnvironment exposes scenario paths so phase runners can inspect files without shell scripts.
type PhaseEnvironment struct {
	ScenarioName string
	ScenarioDir  string
	TestDir      string
	AppRoot      string
}

const (
	failureClassMisconfiguration  = "misconfiguration"
	failureClassMissingDependency = "missing_dependency"
	failureClassTimeout           = "timeout"
	failureClassSystem            = "system"
)

type PhaseRunReport struct {
	Err                   error
	Observations          []string
	FailureClassification string
	Remediation           string
}

// PhaseSummary aggregates phase telemetry for quick status surfaces.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}

type phaseRunnerFunc func(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport

type phaseDefinition struct {
	Name     string
	Runner   phaseRunnerFunc
	Timeout  time.Duration
	Optional bool
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
		goPhases:      defaultGoPhases(),
		requirements:  newNodeRequirementsSyncer(filepath.Dir(absRoot)),
	}, nil
}

// Execute performs a phased test run and returns the recorded result.
func (o *SuiteOrchestrator) Execute(ctx context.Context, req SuiteExecutionRequest) (*SuiteExecutionResult, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, newValidationError("scenarioName is required")
	}
	if !validScenarioName.MatchString(scenario) {
		return nil, newValidationError("scenarioName may only contain letters, numbers, hyphens, or underscores")
	}

	scenarioDir := filepath.Join(o.scenariosRoot, scenario)
	if _, err := os.Stat(scenarioDir); err != nil {
		return nil, newValidationError(fmt.Sprintf("scenario '%s' was not found under %s", scenario, o.scenariosRoot))
	}

	testDir := filepath.Join(scenarioDir, "test")
	phaseDir := filepath.Join(testDir, "phases")
	if _, err := os.Stat(phaseDir); err != nil {
		return nil, fmt.Errorf("scenario '%s' is missing test phases directory at %s", scenario, phaseDir)
	}

	env := PhaseEnvironment{
		ScenarioName: scenario,
		ScenarioDir:  scenarioDir,
		TestDir:      testDir,
		AppRoot:      appRootFromScenario(scenarioDir),
	}

	config, err := loadTestingConfig(env.ScenarioDir)
	if err != nil {
		return nil, err
	}

	defs, err := o.discoverPhaseDefinitions(env)
	if err != nil {
		return nil, err
	}
	defs = o.applyTestingConfig(defs, config)
	if len(defs) == 0 {
		return nil, fmt.Errorf("scenario '%s' has no enabled phase definitions", scenario)
	}

	available := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		available[strings.ToLower(def.Name)] = struct{}{}
	}

	presets := o.loadPresets(testDir, config, available)
	selected, presetUsed, err := selectPhases(defs, presets, req)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, newValidationError("no phases selected for execution")
	}

	artifactDir := filepath.Join(testDir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	result := &SuiteExecutionResult{
		ScenarioName: scenario,
		StartedAt:    time.Now().UTC(),
		PresetUsed:   presetUsed,
	}
	var phases []PhaseExecutionResult
	anyFailure := false

	for _, phase := range selected {
		phaseResult := o.runPhase(ctx, env, artifactDir, phase)
		if phaseResult.Status != "passed" {
			anyFailure = true
		}
		phases = append(phases, phaseResult)
		if req.FailFast && phaseResult.Status == "failed" {
			break
		}
	}

	result.CompletedAt = time.Now().UTC()
	result.Success = !anyFailure
	result.Phases = phases
	result.PhaseSummary = summarizePhases(phases)

	if o.requirements != nil && shouldSyncRequirements(req, defs, selected, phases, result.Success) {
		history := buildCommandHistory(req, presetUsed, selected)
		input := requirementsSyncInput{
			ScenarioName:     env.ScenarioName,
			ScenarioDir:      env.ScenarioDir,
			PhaseDefinitions: defs,
			PhaseResults:     phases,
			CommandHistory:   history,
		}
		if err := o.requirements.Sync(ctx, input); err != nil {
			log.Printf("requirements sync skipped: %v", err)
		}
	}

	return result, nil
}

func (o *SuiteOrchestrator) discoverPhaseDefinitions(env PhaseEnvironment) ([]phaseDefinition, error) {
	phaseDir := filepath.Join(env.TestDir, "phases")
	entries, err := os.ReadDir(phaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read phase directory: %w", err)
	}
	definitions := make(map[string]phaseDefinition)
	for name, handler := range o.goPhases {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		definitions[normalized] = phaseDefinition{
			Name:    normalized,
			Runner:  handler,
			Timeout: o.phaseTimeout,
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
		normalized := strings.ToLower(strings.TrimSpace(phaseName))
		if normalized == "" {
			continue
		}
		if _, exists := definitions[normalized]; exists {
			continue
		}
		definitions[normalized] = phaseDefinition{
			Name:    normalized,
			Runner:  o.scriptPhaseRunner(filepath.Join(phaseDir, name)),
			Timeout: o.phaseTimeout,
		}
	}
	var defs []phaseDefinition
	for _, def := range definitions {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		left := phaseSortValue(defs[i].Name)
		right := phaseSortValue(defs[j].Name)
		if left == right {
			return defs[i].Name < defs[j].Name
		}
		return left < right
	})
	return defs, nil
}

func selectPhases(defs []phaseDefinition, presets map[string][]string, req SuiteExecutionRequest) ([]phaseDefinition, string, error) {
	order := make(map[string]phaseDefinition, len(defs))
	for _, def := range defs {
		order[strings.ToLower(def.Name)] = def
	}

	var presetUsed string
	var desired []string
	if len(req.Phases) > 0 {
		desired = req.Phases
	} else if req.Preset != "" {
		if presetList, ok := presets[strings.ToLower(req.Preset)]; ok {
			desired = presetList
			presetUsed = strings.ToLower(req.Preset)
		} else {
			return nil, "", newValidationError(fmt.Sprintf("preset '%s' is not defined", req.Preset))
		}
	}

	var resolved []phaseDefinition
	allowed := map[string]struct{}{}
	for _, def := range defs {
		allowed[strings.ToLower(def.Name)] = struct{}{}
	}

	// Determine base run list
	if len(desired) > 0 {
		for _, phase := range desired {
			normalized := strings.ToLower(strings.TrimSpace(phase))
			if normalized == "" {
				continue
			}
			def, ok := order[normalized]
			if !ok {
				return nil, "", newValidationError(fmt.Sprintf("phase '%s' is not defined", phase))
			}
			resolved = append(resolved, def)
		}
	} else {
		resolved = append(resolved, defs...)
	}

	// Handle skip list
	if len(req.Skip) > 0 {
		skipSet := make(map[string]struct{}, len(req.Skip))
		for _, phase := range req.Skip {
			normalized := strings.ToLower(strings.TrimSpace(phase))
			if normalized != "" {
				skipSet[normalized] = struct{}{}
			}
		}
		var filtered []phaseDefinition
		for _, def := range resolved {
			if _, skip := skipSet[strings.ToLower(def.Name)]; skip {
				continue
			}
			filtered = append(filtered, def)
		}
		resolved = filtered
	}

	return resolved, presetUsed, nil
}

func (o *SuiteOrchestrator) scriptPhaseRunner(scriptPath string) phaseRunnerFunc {
	return func(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport {
		cmd := exec.CommandContext(ctx, "bash", scriptPath)
		cmd.Dir = env.TestDir
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("TEST_GENIE_SCENARIO_DIR=%s", env.ScenarioDir),
			fmt.Sprintf("TEST_GENIE_APP_ROOT=%s", env.AppRoot),
		)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		return PhaseRunReport{Err: cmd.Run()}
	}
}

func (o *SuiteOrchestrator) runPhase(ctx context.Context, env PhaseEnvironment, artifactDir string, def phaseDefinition) PhaseExecutionResult {
	start := time.Now()
	logPath := filepath.Join(artifactDir, fmt.Sprintf("%s-%s.log", start.UTC().Format("20060102-150405"), def.Name))
	logFile, err := os.Create(logPath)
	if err != nil {
		return PhaseExecutionResult{
			Name:            def.Name,
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
		env.AppRoot = appRootFromScenario(env.ScenarioDir)
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
			classification = failureClassTimeout
			if remediation == "" {
				remediation = "Increase the timeout or break the phase into smaller steps."
			}
		}
		if classification == "" {
			classification = failureClassSystem
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
		Name:            def.Name,
		Status:          status,
		DurationSeconds: duration,
		LogPath:         logPath,
		Error:           errMsg,
		Classification:  classification,
		Remediation:     remediation,
		Observations:    report.Observations,
	}
}

func summarizePhases(phases []PhaseExecutionResult) PhaseSummary {
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

func defaultGoPhases() map[string]phaseRunnerFunc {
	return map[string]phaseRunnerFunc{
		"structure":    runStructurePhase,
		"dependencies": runDependenciesPhase,
		"unit":         runUnitPhase,
		"integration":  runIntegrationPhase,
		"business":     runBusinessPhase,
		"performance":  runPerformancePhase,
	}
}

func phaseSortValue(name string) int {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if weight, ok := goPhaseOrdering[normalized]; ok {
		return weight
	}
	return 1000
}

func (o *SuiteOrchestrator) loadPresets(testDir string, cfg *testingConfig, allowed map[string]struct{}) map[string][]string {
	presets := make(map[string][]string)
	configPath := filepath.Join(testDir, "presets.json")
	if raw, err := os.ReadFile(configPath); err == nil {
		var parsed map[string][]string
		if err := json.Unmarshal(raw, &parsed); err == nil {
			for key, phases := range parsed {
				name := strings.ToLower(strings.TrimSpace(key))
				if name == "" {
					continue
				}
				filtered := filterPresetPhases(phases, allowed)
				if len(filtered) > 0 {
					presets[name] = filtered
				}
			}
		}
	}

	if cfg != nil {
		for key, phases := range cfg.Presets {
			filtered := filterPresetPhases(phases, allowed)
			if len(filtered) == 0 {
				delete(presets, key)
				continue
			}
			presets[key] = filtered
		}
	}

	for key, phases := range defaultExecutionPresets {
		if _, exists := presets[key]; exists {
			continue
		}
		filtered := filterPresetPhases(phases, allowed)
		if len(filtered) > 0 {
			presets[key] = filtered
		}
	}

	return presets
}

func appRootFromScenario(scenarioDir string) string {
	return filepath.Clean(filepath.Join(scenarioDir, "..", ".."))
}

func (o *SuiteOrchestrator) applyTestingConfig(defs []phaseDefinition, cfg *testingConfig) []phaseDefinition {
	if cfg == nil || len(cfg.Phases) == 0 {
		return defs
	}
	var configured []phaseDefinition
	for _, def := range defs {
		name := strings.ToLower(def.Name)
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
	seen := make(map[string]struct{})
	for _, phase := range phases {
		normalized := strings.ToLower(strings.TrimSpace(phase))
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

func shouldSyncRequirements(req SuiteExecutionRequest, defs []phaseDefinition, selected []phaseDefinition, phases []PhaseExecutionResult, success bool) bool {
	if len(defs) == 0 || len(selected) == 0 {
		return false
	}
	if req.Preset != "" || len(req.Phases) > 0 || len(req.Skip) > 0 {
		return false
	}
	if len(selected) != len(defs) {
		return false
	}
	if len(phases) != len(selected) {
		return false
	}
	if !success {
		return false
	}
	return true
}

func buildCommandHistory(req SuiteExecutionRequest, presetUsed string, selected []phaseDefinition) []string {
	var history []string
	var descriptor []string
	if req.ScenarioName != "" {
		descriptor = append(descriptor, fmt.Sprintf("scenario=%s", req.ScenarioName))
	}
	if presetUsed != "" {
		descriptor = append(descriptor, fmt.Sprintf("preset=%s", presetUsed))
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
	if len(selected) > 0 {
		var names []string
		for _, def := range selected {
			if def.Name != "" {
				names = append(names, def.Name)
			}
		}
		if len(names) > 0 {
			history = append(history, fmt.Sprintf("phase-order:%s", strings.Join(names, ",")))
		}
	}
	return history
}
