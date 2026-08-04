package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerdescriptor"
	"test-genie/internal/shared"

	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type phasePlan struct {
	Definitions           []phases.Definition
	Applicable            []phases.Definition
	Applicability         map[string]phaseApplicabilityNotice
	NotApplicable         []phaseApplicabilityNotice
	Selected              []phases.Definition
	ExplicitNotApplicable []phaseApplicabilityNotice
	PresetUsed            string
	GlobalToggles         PhaseToggleConfig
	DisabledByDefault     []phaseDisableNotice
	ExplicitDisabled      []phaseDisableNotice
}

func (o *SuiteOrchestrator) buildPhasePlan(env workspacepkg.Environment, cfg *workspacepkg.Config, req SuiteExecutionRequest) (*phasePlan, error) {
	defs, err := o.discoverPhaseDefinitions(env)
	if err != nil {
		return nil, err
	}
	if err := validateTestingConfigPhases(defs, cfg); err != nil {
		return nil, err
	}
	defs = o.applyTestingConfig(defs, cfg)
	if len(defs) == 0 {
		return nil, fmt.Errorf("scenario '%s' has no enabled phase definitions", env.ScenarioName)
	}
	applicabilityDecisions, err := o.evaluatePhaseApplicability(defs, env, cfg)
	if err != nil {
		return nil, err
	}
	applicableDefs, notApplicable := splitApplicableDefinitions(defs, applicabilityDecisions)
	if len(applicableDefs) == 0 {
		return nil, shared.NewValidationError("no phases are applicable to this scenario")
	}

	globalToggles, err := o.GlobalPhaseToggles()
	if err != nil {
		return nil, fmt.Errorf("load global phase toggles: %w", err)
	}

	available := make(map[string]struct{}, len(applicableDefs))
	for _, def := range applicableDefs {
		available[def.Name.Key()] = struct{}{}
	}

	presets := o.loadPresets(env.TestDir, cfg, available)
	presets[phases.PresetComprehensive.String()] = definitionNames(applicableDefs)
	selected, presetUsed, notices, err := selectPhases(applicableDefs, presets, req, globalToggles)
	if err != nil {
		if len(req.Phases) > 0 {
			explicitNotApplicable := requestedNotApplicable(req.Phases, notApplicable)
			if len(explicitNotApplicable) > 0 && allRequestedPhasesAreNotApplicable(req.Phases, explicitNotApplicable) {
				return &phasePlan{
					Definitions:           defs,
					Applicable:            applicableDefs,
					Applicability:         applicabilityDecisions,
					NotApplicable:         notApplicable,
					ExplicitNotApplicable: explicitNotApplicable,
					PresetUsed:            presetUsed,
					GlobalToggles:         globalToggles,
					DisabledByDefault:     notices.Skipped,
					ExplicitDisabled:      notices.Explicit,
				}, shared.NewValidationError("requested phase is not applicable to this scenario")
			}
		}
		return nil, err
	}
	explicitNotApplicable := requestedNotApplicable(req.Phases, notApplicable)
	if len(selected) == 0 {
		if len(explicitNotApplicable) > 0 {
			return nil, shared.NewValidationError("requested phase is not applicable to this scenario")
		}
		if len(notices.Skipped) > 0 {
			return nil, shared.NewValidationError("no phases selected for execution; requested phases are disabled or skipped")
		}
		return nil, shared.NewValidationError("no phases selected for execution")
	}

	return &phasePlan{
		Definitions:           defs,
		Applicable:            applicableDefs,
		Applicability:         applicabilityDecisions,
		NotApplicable:         notApplicable,
		Selected:              selected,
		ExplicitNotApplicable: explicitNotApplicable,
		PresetUsed:            presetUsed,
		GlobalToggles:         globalToggles,
		DisabledByDefault:     notices.Skipped,
		ExplicitDisabled:      notices.Explicit,
	}, nil
}

func validateTestingConfigPhases(defs []phases.Definition, cfg *workspacepkg.Config) error {
	if cfg == nil || len(cfg.Phases) == 0 {
		return nil
	}
	available := make(map[string]struct{}, len(defs))
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		key := def.Name.Key()
		if key == "" {
			continue
		}
		available[key] = struct{}{}
		names = append(names, key)
	}
	for name := range cfg.Phases {
		if _, ok := available[name]; ok {
			continue
		}
		return shared.NewValidationError(fmt.Sprintf("unknown phase %q in .vrooli/testing.json phases; available phases: %s", name, strings.Join(names, ", ")))
	}
	return nil
}

type phaseDisableNotice struct {
	Name      string
	Toggle    PhaseToggle
	EnvVar    string
	Requested bool
}

type phaseApplicabilityNotice struct {
	Definition phases.Definition
	Result     applicability.Result
	Descriptor providerdescriptor.Descriptor
}

func (o *SuiteOrchestrator) evaluatePhaseApplicability(defs []phases.Definition, env workspacepkg.Environment, cfg *workspacepkg.Config) (map[string]phaseApplicabilityNotice, error) {
	ctx, err := buildApplicabilityContext(env, cfg, o.descriptorPredicates())
	if err != nil {
		return nil, err
	}
	results := make(map[string]phaseApplicabilityNotice, len(defs))
	for _, def := range defs {
		key := def.Name.Key()
		entry, ok := o.descriptorEntry(key)
		if !ok {
			if def.ProviderScenario != "" {
				return nil, fmt.Errorf("phase_applicability_descriptor_missing: phase %q has a provider-backed catalog definition but no provider descriptor", def.Name.String())
			}
			results[key] = phaseApplicabilityNotice{
				Definition: def,
				Result: applicability.Result{
					Phase:  def.Name.String(),
					Status: applicability.StatusApplies,
					Reasons: []applicability.Reason{{
						Code:    "applicability.test_fixture_default",
						Message: "non-provider test fixture phase applies by default",
					}},
				},
			}
			continue
		}
		notice := phaseApplicabilityNotice{
			Definition: def,
			Result:     applicability.Evaluate(def.Name.String(), entry.Descriptor.Applicability, ctx),
			Descriptor: entry.Descriptor,
		}
		if notice.Result.Status == applicability.StatusInvalid {
			return nil, fmt.Errorf("phase_applicability_invalid: phase %q from %s: %s", def.Name.String(), entry.Descriptor.Path, applicabilityReasons(notice.Result.Reasons))
		}
		results[key] = notice
	}
	return results, nil
}

func splitApplicableDefinitions(defs []phases.Definition, decisions map[string]phaseApplicabilityNotice) ([]phases.Definition, []phaseApplicabilityNotice) {
	applicable := make([]phases.Definition, 0, len(defs))
	var notApplicable []phaseApplicabilityNotice
	for _, def := range defs {
		notice, ok := decisions[def.Name.Key()]
		if !ok || notice.Result.Status == applicability.StatusApplies || notice.Result.Status == applicability.StatusUnknown {
			applicable = append(applicable, def)
			continue
		}
		notApplicable = append(notApplicable, notice)
	}
	return applicable, notApplicable
}

func requestedNotApplicable(requested []string, notices []phaseApplicabilityNotice) []phaseApplicabilityNotice {
	if len(requested) == 0 || len(notices) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(requested))
	for _, phase := range requested {
		if key := phases.NormalizeKey(phase); key != "" {
			wanted[key] = struct{}{}
		}
	}
	var out []phaseApplicabilityNotice
	for _, notice := range notices {
		if _, ok := wanted[notice.Definition.Name.Key()]; ok {
			out = append(out, notice)
		}
	}
	return out
}

func allRequestedPhasesAreNotApplicable(requested []string, notices []phaseApplicabilityNotice) bool {
	if len(requested) == 0 || len(notices) == 0 {
		return false
	}
	notApplicable := make(map[string]struct{}, len(notices))
	for _, notice := range notices {
		notApplicable[notice.Definition.Name.Key()] = struct{}{}
	}
	for _, phase := range requested {
		key := phases.NormalizeKey(phase)
		if key == "" {
			continue
		}
		if _, ok := notApplicable[key]; !ok {
			return false
		}
	}
	return true
}

func applicabilityReasons(reasons []applicability.Reason) string {
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, reason.Code+": "+reason.Message)
	}
	return strings.Join(parts, "; ")
}

func buildApplicabilityContext(env workspacepkg.Environment, cfg *workspacepkg.Config, predicates []providerdescriptor.Predicate) (applicability.Context, error) {
	ctx := applicability.Context{
		TargetKind:            env.TargetKind,
		TargetID:              env.TargetID,
		TargetRoot:            env.TargetRoot,
		ScenarioName:          env.ScenarioName,
		ScenarioDir:           env.ScenarioDir,
		HasUI:                 dirExists(filepath.Join(env.ScenarioDir, "ui")),
		HasAPI:                dirExists(filepath.Join(env.ScenarioDir, "api")),
		Files:                 map[string]bool{},
		PathGlobs:             map[string][]string{},
		ScenarioDependencies:  scenarioDependencies(env.ScenarioDir),
		ServiceCapabilities:   serviceCapabilities(env.ScenarioDir),
		ServiceTags:           serviceTags(env.ScenarioDir),
		TestingConfigSections: testingConfigSections(cfg),
	}
	for _, predicate := range predicates {
		if filePath := strings.TrimSpace(predicate.FileExists); filePath != "" {
			rel := filepath.ToSlash(filepath.Clean(filePath))
			if rel == "." {
				continue
			}
			ctx.Files[rel] = fileExists(filepath.Join(env.ScenarioDir, filepath.FromSlash(rel)))
		}
		if glob := strings.TrimSpace(predicate.PathGlob); glob != "" {
			matches, err := safePathGlob(env.ScenarioDir, glob)
			if err != nil {
				return applicability.Context{}, fmt.Errorf("invalid applicability pathGlob %q: %w", glob, err)
			}
			ctx.PathGlobs[glob] = matches
		}
	}
	return ctx, nil
}

func safePathGlob(scenarioDir, pattern string) ([]string, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" || filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "/") {
		return nil, fmt.Errorf("must be a non-empty target-relative pattern")
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == ".." {
			return nil, fmt.Errorf("must not traverse outside the target")
		}
	}
	matches, err := filepath.Glob(filepath.Join(scenarioDir, filepath.FromSlash(pattern)))
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(scenarioDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		resolved, err := filepath.EvalSymlinks(match)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("match escapes target: %s", match)
		}
		result = append(result, filepath.ToSlash(rel))
	}
	sort.Strings(result)
	return result, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func testingConfigSections(cfg *workspacepkg.Config) map[string]bool {
	sections := map[string]bool{}
	if cfg == nil {
		return sections
	}
	for section, present := range cfg.Sections {
		if present {
			sections[section] = true
		}
	}
	return sections
}

func serviceCapabilities(scenarioDir string) map[string]bool {
	return serviceStringFacts(scenarioDir, "capabilities")
}

func serviceTags(scenarioDir string) map[string]bool {
	return serviceStringFacts(scenarioDir, "tags")
}

func serviceStringFacts(scenarioDir, field string) map[string]bool {
	values := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return values
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return values
	}
	addStringSlice(values, nestedStringSlice(doc, "service", field))
	return values
}

func scenarioDependencies(scenarioDir string) map[string]applicability.DependencyStatus {
	result := map[string]applicability.DependencyStatus{}
	raw, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return result
	}
	var doc struct {
		Dependencies struct {
			Scenarios map[string]struct {
				Enabled *bool `json:"enabled"`
			} `json:"scenarios"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return result
	}
	for name, dependency := range doc.Dependencies.Scenarios {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		if dependency.Enabled != nil && !*dependency.Enabled {
			result[key] = applicability.DependencyDisabled
		} else {
			result[key] = applicability.DependencyPresent
		}
	}
	return result
}

func nestedStringSlice(doc map[string]any, path ...string) []string {
	var current any = doc
	for _, segment := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[segment]
	}
	values, ok := current.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func addStringSlice(out map[string]bool, values []string) {
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key != "" {
			out[key] = true
		}
	}
}

func definitionNames(defs []phases.Definition) []string {
	out := make([]string, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.Name.String())
	}
	return out
}
