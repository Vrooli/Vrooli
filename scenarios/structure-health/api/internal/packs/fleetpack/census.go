// Package fleetpack provides manifest-only measurements over the scenario
// fleet. It deliberately reads service.json files from disk and never invokes
// code-facts or the per-scenario validation engine, so the census is fast,
// deterministic, and usable as migration evidence while scenarios are stopped.
package fleetpack

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"structure-health/internal/packs/configpack/manifestschema"
)

const censusSchemaVersion = 1

var (
	cdPattern          = regexp.MustCompile(`(?:^|[;&|()[:space:]])cd(?:[[:space:]]|$)`)
	pipePattern        = regexp.MustCompile(`(^|[^|])\|([^|]|$)`)
	redirectPattern    = regexp.MustCompile(`(?:^|[^>])>{1,2}`)
	conditionalPattern = regexp.MustCompile(`(?:if[[:space:]]+\[[[:space:]]|(?:^|[;&|()[:space:]])test[[:space:]]+-)`)
	shellRefPattern    = regexp.MustCompile(`(?:^|[[:space:]"'()])([A-Za-z0-9_./-]+\.sh)(?:$|[[:space:]"';&|)])`)
)

// Report is the frozen, deterministic census contract consumed by every phase
// of the declared-scenario migration. Do not rename fields after Phase 1.
type Report struct {
	SchemaVersion          int                    `json:"schema_version"`
	ScenarioDirectoryCount int                    `json:"scenario_directory_count"`
	ManifestCount          int                    `json:"manifest_count"`
	NoManifest             []string               `json:"no_manifest"`
	Lifecycle              LifecycleReport        `json:"lifecycle"`
	LiveShellSyntax        ShellSyntaxReport      `json:"live_shell_syntax"`
	Cohorts                CohortReport           `json:"cohorts"`
	Ports                  PortReport             `json:"ports"`
	PeerDependencies       PeerDependencyReport   `json:"peer_dependencies"`
	Components             ComponentReport        `json:"components"`
	ShellFiles             ShellFileReport        `json:"shell_files"`
	SchemaValidation       SchemaValidationReport `json:"schema_validation"`
	StepsInventory         StepInventory          `json:"steps_inventory,omitempty"`
}

type LifecycleReport struct {
	StepsByPhase           map[string]int `json:"steps_by_phase"`
	TotalSteps             int            `json:"total_steps"`
	LiveSteps              int            `json:"live_steps"`
	DistinctLiveStepShapes int            `json:"distinct_live_step_shapes"`
}

type ShellSyntaxReport struct {
	CDOccurrences          int `json:"cd_occurrences"`
	PipeOccurrences        int `json:"pipe_occurrences"`
	RedirectOccurrences    int `json:"redirect_occurrences"`
	ConditionalOccurrences int `json:"if_bracket_or_test_occurrences"`
}

type CohortReport struct {
	TemplateCurrent    []string `json:"template_current"`
	TemplatePlusExtras []string `json:"template_plus_extras"`
	LightDrift         []string `json:"light_drift"`
	HeavyDrift         []string `json:"heavy_drift"`
	PreTemplate        []string `json:"pre_template"`
	NoManifest         []string `json:"no_manifest"`
}

type PortReport struct {
	DeclarationCount     int                       `json:"declaration_count"`
	ConventionViolations []PortConventionViolation `json:"convention_violations"`
	RangeAllocatedCount  int                       `json:"range_allocated_count"`
	PinnedCount          int                       `json:"pinned_count"`
}

type PortConventionViolation struct {
	Scenario string `json:"scenario"`
	Port     string `json:"port"`
	EnvVar   string `json:"env_var"`
	Expected string `json:"expected"`
}

type PeerDependencyReport struct {
	EdgeCount                 int      `json:"edge_count"`
	DeclaringScenarioCount    int      `json:"declaring_scenario_count"`
	DistinctTargetCount       int      `json:"distinct_target_count"`
	RuntimeOnlyCount          int      `json:"runtime_only_count"`
	RuntimeOnlyRationaleCount int      `json:"runtime_only_rationale_count"`
	VersionRangeCount         int      `json:"version_range_count"`
	DeclaringScenarios        []string `json:"declaring_scenarios"`
	DistinctTargets           []string `json:"distinct_targets"`
}

type ComponentReport struct {
	AdoptingManifestCount int `json:"adopting_manifest_count"`
	ComponentCount        int `json:"component_count"`
}

type ShellFileReport struct {
	ScenarioCount              int              `json:"scenario_count"`
	ResourceCount              int              `json:"resource_count"`
	LifecycleInvokedReferences []ShellReference `json:"lifecycle_invoked_references"`
}

type ShellReference struct {
	Scenario string `json:"scenario"`
	Phase    string `json:"phase"`
	Step     string `json:"step"`
	Path     string `json:"path"`
}

type SchemaValidationReport struct {
	ViolationCount       int            `json:"violation_count"`
	FailingManifestCount int            `json:"failing_manifest_count"`
	ByMessage            map[string]int `json:"by_message"`
	ByManifest           map[string]int `json:"by_manifest"`
}

// StepInventory preserves every lifecycle step object, keyed by scenario,
// phase, and step name. The raw JSON object is kept so
// later mechanical edits can prove exactly what was removed.
type StepInventory map[string]map[string]map[string]json.RawMessage

type manifestDocument struct {
	Service struct {
		Name string `json:"name"`
	} `json:"service"`
	Lifecycle    map[string]json.RawMessage `json:"lifecycle"`
	Ports        map[string]json.RawMessage `json:"ports"`
	Components   map[string]json.RawMessage `json:"components"`
	Dependencies struct {
		Scenarios map[string]dependencyDocument `json:"scenarios"`
	} `json:"dependencies"`
}

type phaseDocument struct {
	Steps []json.RawMessage `json:"steps"`
}

type stepDocument struct {
	Name string   `json:"name"`
	Exec []string `json:"exec"`
}

type portDocument struct {
	EnvVar string `json:"env_var"`
	Range  string `json:"range"`
	Port   *int   `json:"port"`
}

type dependencyDocument struct {
	RuntimeOnly          bool   `json:"runtime_only"`
	RuntimeOnlyRationale string `json:"runtime_only_rationale"`
	VersionRange         string `json:"versionRange"`
}

type templateSteps map[string]struct{}

// Census reads the repository rooted at repoRoot and returns a deterministic
// fleet report. When includeSteps is false the large step inventory is omitted
// from JSON while every derived count remains identical.
func Census(repoRoot string, includeSteps bool) (Report, error) {
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve repository root: %w", err)
	}
	templates, err := loadTemplates(repoRoot)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		SchemaVersion: censusSchemaVersion,
		NoManifest:    []string{},
		Lifecycle:     LifecycleReport{StepsByPhase: map[string]int{}},
		Ports:         PortReport{ConventionViolations: []PortConventionViolation{}},
		ShellFiles:    ShellFileReport{LifecycleInvokedReferences: []ShellReference{}},
		SchemaValidation: SchemaValidationReport{
			ByMessage:  map[string]int{},
			ByManifest: map[string]int{},
		},
	}
	if includeSteps {
		report.StepsInventory = StepInventory{}
	}

	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return Report{}, fmt.Errorf("read scenarios directory: %w", err)
	}

	liveShapes := map[string]struct{}{}
	peerTargets := map[string]struct{}{}
	peerDeclarers := map[string]struct{}{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		report.ScenarioDirectoryCount++
		scenarioName := entry.Name()
		manifestPath := filepath.Join(scenariosDir, scenarioName, ".vrooli", "service.json")
		raw, readErr := os.ReadFile(manifestPath) // #nosec G304 -- path is rooted in enumerated scenario directories.
		if readErr != nil {
			if os.IsNotExist(readErr) {
				report.NoManifest = append(report.NoManifest, scenarioName)
				continue
			}
			return Report{}, fmt.Errorf("read %s: %w", manifestPath, readErr)
		}

		var document manifestDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			return Report{}, fmt.Errorf("parse %s: %w", manifestPath, err)
		}
		if strings.TrimSpace(document.Service.Name) != "" && !strings.Contains(document.Service.Name, "{{") {
			scenarioName = document.Service.Name
		}
		report.ManifestCount++

		messages, validateErr := manifestschema.ValidationMessages(raw, manifestPath)
		if validateErr != nil {
			return Report{}, fmt.Errorf("validate %s: %w", manifestPath, validateErr)
		}
		if len(messages) > 0 {
			report.SchemaValidation.FailingManifestCount++
			report.SchemaValidation.ByManifest[scenarioName] = len(messages)
			for _, message := range messages {
				report.SchemaValidation.ByMessage[message]++
				report.SchemaValidation.ViolationCount++
			}
		}

		liveStepCount, matchingTemplateSteps, err := measureLifecycle(&report, liveShapes, templates, scenarioName, document.Lifecycle, includeSteps)
		if err != nil {
			return Report{}, fmt.Errorf("measure lifecycle for %s: %w", scenarioName, err)
		}
		classifyCohort(&report.Cohorts, scenarioName, liveStepCount, matchingTemplateSteps)
		measurePorts(&report.Ports, scenarioName, document.Ports)
		measurePeers(&report.PeerDependencies, peerDeclarers, peerTargets, scenarioName, document.Dependencies.Scenarios)
		if document.Components != nil {
			report.Components.AdoptingManifestCount++
			report.Components.ComponentCount += len(document.Components)
		}
	}

	report.Lifecycle.DistinctLiveStepShapes = len(liveShapes)
	report.Cohorts.NoManifest = append([]string(nil), report.NoManifest...)
	report.PeerDependencies.DeclaringScenarioCount = len(peerDeclarers)
	report.PeerDependencies.DistinctTargetCount = len(peerTargets)
	report.PeerDependencies.DeclaringScenarios = sortedKeys(peerDeclarers)
	report.PeerDependencies.DistinctTargets = sortedKeys(peerTargets)

	report.ShellFiles.ScenarioCount, err = countShellFiles(scenariosDir)
	if err != nil {
		return Report{}, fmt.Errorf("count scenario shell files: %w", err)
	}
	resourcesDir := filepath.Join(repoRoot, "resources")
	report.ShellFiles.ResourceCount, err = countShellFiles(resourcesDir)
	if err != nil && !os.IsNotExist(err) {
		return Report{}, fmt.Errorf("count resource shell files: %w", err)
	}

	sortReport(&report)
	return report, nil
}

func loadTemplates(repoRoot string) ([]templateSteps, error) {
	paths := []string{
		filepath.Join(repoRoot, "templates", "scenarios", "react-vite", ".vrooli", "service.json"),
		filepath.Join(repoRoot, "templates", "scenarios", "landing-page-react-vite", ".vrooli", "service.json"),
	}
	out := make([]templateSteps, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path) // #nosec G304 -- fixed repository-relative template path.
		if err != nil {
			return nil, fmt.Errorf("read census template %s: %w", path, err)
		}
		var document manifestDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			return nil, fmt.Errorf("parse census template %s: %w", path, err)
		}
		steps := templateSteps{}
		for _, phaseName := range []string{"setup", "develop"} {
			rawPhase, exists := document.Lifecycle[phaseName]
			if !exists || len(rawPhase) == 0 {
				continue
			}
			var phase phaseDocument
			if err := json.Unmarshal(rawPhase, &phase); err != nil {
				return nil, fmt.Errorf("parse census template %s phase %s: %w", path, phaseName, err)
			}
			for _, rawStep := range phase.Steps {
				var step stepDocument
				if err := json.Unmarshal(rawStep, &step); err != nil {
					return nil, fmt.Errorf("parse template %s step: %w", phaseName, err)
				}
				steps[phaseName+"|"+normalizeExec(step.Exec, "{{SCENARIO_ID}}")] = struct{}{}
			}
		}
		out = append(out, steps)
	}
	return out, nil
}

func measureLifecycle(report *Report, liveShapes map[string]struct{}, templates []templateSteps, scenarioName string, lifecycle map[string]json.RawMessage, includeSteps bool) (int, int, error) {
	matchCounts := make([]int, len(templates))
	liveStepCount := 0
	phaseNames := make([]string, 0, len(lifecycle))
	for phaseName := range lifecycle {
		phaseNames = append(phaseNames, phaseName)
	}
	sort.Strings(phaseNames)
	for _, phaseName := range phaseNames {
		var phase phaseDocument
		if err := json.Unmarshal(lifecycle[phaseName], &phase); err != nil {
			// Lifecycle also carries scalar metadata such as version. Only objects
			// with a steps array are phases owned by this census.
			continue
		}
		for index, rawStep := range phase.Steps {
			var step stepDocument
			if err := json.Unmarshal(rawStep, &step); err != nil {
				return 0, 0, fmt.Errorf("parse %s step %d: %w", phaseName, index, err)
			}
			report.Lifecycle.StepsByPhase[phaseName]++
			report.Lifecycle.TotalSteps++
			if includeSteps {
				addInventoryStep(report.StepsInventory, scenarioName, phaseName, step.Name, rawStep)
			}
			if phaseName != "setup" && phaseName != "develop" {
				continue
			}
			liveStepCount++
			report.Lifecycle.LiveSteps++
			normalized := normalizeExec(step.Exec, scenarioName)
			liveShapes[phaseName+"|"+normalized] = struct{}{}
			commandText := strings.Join(step.Exec, " ")
			measureShellSyntax(&report.LiveShellSyntax, commandText)
			measureShellReferences(&report.ShellFiles, scenarioName, phaseName, step.Name, commandText)
			for templateIndex, template := range templates {
				if _, matches := template[phaseName+"|"+normalized]; matches {
					matchCounts[templateIndex]++
				}
			}
		}
	}
	best := 0
	for _, count := range matchCounts {
		if count > best {
			best = count
		}
	}
	return liveStepCount, best, nil
}

func normalizeExec(argv []string, scenarioName string) string {
	normalized := strings.Join(argv, "\x00")
	if scenarioName != "" && scenarioName != "{{SCENARIO_ID}}" {
		normalized = strings.ReplaceAll(normalized, scenarioName, "{{SCENARIO_ID}}")
	}
	return normalized
}

func measureShellSyntax(report *ShellSyntaxReport, run string) {
	report.CDOccurrences += len(cdPattern.FindAllStringIndex(run, -1))
	report.PipeOccurrences += len(pipePattern.FindAllStringIndex(run, -1))
	report.RedirectOccurrences += len(redirectPattern.FindAllStringIndex(run, -1))
	report.ConditionalOccurrences += len(conditionalPattern.FindAllStringIndex(run, -1))
}

func measureShellReferences(report *ShellFileReport, scenarioName, phaseName, stepName, run string) {
	seen := map[string]struct{}{}
	for _, match := range shellRefPattern.FindAllStringSubmatch(run, -1) {
		if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
			continue
		}
		path := match[1]
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		report.LifecycleInvokedReferences = append(report.LifecycleInvokedReferences, ShellReference{
			Scenario: scenarioName,
			Phase:    phaseName,
			Step:     stepName,
			Path:     path,
		})
	}
}

func addInventoryStep(inventory StepInventory, scenarioName, phaseName, stepName string, raw json.RawMessage) {
	if inventory[scenarioName] == nil {
		inventory[scenarioName] = map[string]map[string]json.RawMessage{}
	}
	if inventory[scenarioName][phaseName] == nil {
		inventory[scenarioName][phaseName] = map[string]json.RawMessage{}
	}
	if strings.TrimSpace(stepName) == "" {
		stepName = "unnamed"
	}
	key := stepName
	for suffix := 2; inventory[scenarioName][phaseName][key] != nil; suffix++ {
		key = fmt.Sprintf("%s#%d", stepName, suffix)
	}
	inventory[scenarioName][phaseName][key] = append(json.RawMessage(nil), raw...)
}

func classifyCohort(report *CohortReport, scenarioName string, liveStepCount, matches int) {
	switch {
	case matches == 5 && liveStepCount == 5:
		report.TemplateCurrent = append(report.TemplateCurrent, scenarioName)
	case matches == 5 && liveStepCount > 5:
		report.TemplatePlusExtras = append(report.TemplatePlusExtras, scenarioName)
	case matches >= 3:
		report.LightDrift = append(report.LightDrift, scenarioName)
	case matches >= 1:
		report.HeavyDrift = append(report.HeavyDrift, scenarioName)
	default:
		report.PreTemplate = append(report.PreTemplate, scenarioName)
	}
}

func measurePorts(report *PortReport, scenarioName string, ports map[string]json.RawMessage) {
	portNames := make([]string, 0, len(ports))
	for name := range ports {
		portNames = append(portNames, name)
	}
	sort.Strings(portNames)
	for _, name := range portNames {
		var definition portDocument
		if err := json.Unmarshal(ports[name], &definition); err != nil {
			continue
		}
		report.DeclarationCount++
		expected := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_PORT"
		if definition.EnvVar != expected {
			report.ConventionViolations = append(report.ConventionViolations, PortConventionViolation{
				Scenario: scenarioName,
				Port:     name,
				EnvVar:   definition.EnvVar,
				Expected: expected,
			})
		}
		if strings.TrimSpace(definition.Range) != "" {
			report.RangeAllocatedCount++
		} else if definition.Port != nil {
			report.PinnedCount++
		}
	}
}

func measurePeers(report *PeerDependencyReport, declarers, targets map[string]struct{}, scenarioName string, dependencies map[string]dependencyDocument) {
	if len(dependencies) == 0 {
		return
	}
	declarers[scenarioName] = struct{}{}
	for target, dependency := range dependencies {
		report.EdgeCount++
		targets[target] = struct{}{}
		if dependency.RuntimeOnly {
			report.RuntimeOnlyCount++
		}
		if strings.TrimSpace(dependency.RuntimeOnlyRationale) != "" {
			report.RuntimeOnlyRationaleCount++
		}
		if strings.TrimSpace(dependency.VersionRange) != "" {
			report.VersionRangeCount++
		}
	}
}

func countShellFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && skipShellCensusDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".sh") {
			count++
		}
		return nil
	})
	return count, err
}

// skipShellCensusDir excludes dependency caches, not build-shaped directories.
// This is an artifact inventory rather than a code-quality scan: committed
// fixtures under target/, build helpers under build/, and bundled launchers
// under bin/ are all shell files that the migration must disposition.
func skipShellCensusDir(name string) bool {
	switch name {
	case ".git", ".venv", "node_modules", "vendor", "__pycache__", "venv":
		return true
	default:
		return false
	}
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortReport(report *Report) {
	sort.Strings(report.NoManifest)
	sort.Strings(report.Cohorts.TemplateCurrent)
	sort.Strings(report.Cohorts.TemplatePlusExtras)
	sort.Strings(report.Cohorts.LightDrift)
	sort.Strings(report.Cohorts.HeavyDrift)
	sort.Strings(report.Cohorts.PreTemplate)
	sort.Strings(report.Cohorts.NoManifest)
	sort.Slice(report.Ports.ConventionViolations, func(i, j int) bool {
		left, right := report.Ports.ConventionViolations[i], report.Ports.ConventionViolations[j]
		if left.Scenario != right.Scenario {
			return left.Scenario < right.Scenario
		}
		return left.Port < right.Port
	})
	sort.Slice(report.ShellFiles.LifecycleInvokedReferences, func(i, j int) bool {
		left, right := report.ShellFiles.LifecycleInvokedReferences[i], report.ShellFiles.LifecycleInvokedReferences[j]
		if left.Scenario != right.Scenario {
			return left.Scenario < right.Scenario
		}
		if left.Phase != right.Phase {
			return left.Phase < right.Phase
		}
		if left.Step != right.Step {
			return left.Step < right.Step
		}
		return left.Path < right.Path
	})
}
