package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	applicabilityDecisions := o.evaluatePhaseApplicability(defs, env, cfg)
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

func (o *SuiteOrchestrator) evaluatePhaseApplicability(defs []phases.Definition, env workspacepkg.Environment, cfg *workspacepkg.Config) map[string]phaseApplicabilityNotice {
	ctx := buildApplicabilityContext(env, cfg, o.descriptorPredicates())
	results := make(map[string]phaseApplicabilityNotice, len(defs))
	for _, def := range defs {
		key := def.Name.Key()
		entry, ok := o.descriptorEntry(key)
		if !ok {
			results[key] = phaseApplicabilityNotice{
				Definition: def,
				Result: applicability.Result{
					Phase:  def.Name.String(),
					Status: applicability.StatusApplies,
					Reasons: []applicability.Reason{{
						Code:    "applicability.legacy_catalog_default",
						Message: "legacy catalog phase applies by default until descriptor migration",
					}},
				},
			}
			continue
		}
		results[key] = phaseApplicabilityNotice{
			Definition: def,
			Result:     applicability.Evaluate(def.Name.String(), entry.Descriptor.Applicability, ctx),
			Descriptor: entry.Descriptor,
		}
	}
	return results
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

func buildApplicabilityContext(env workspacepkg.Environment, cfg *workspacepkg.Config, predicates []providerdescriptor.Predicate) applicability.Context {
	ctx := applicability.Context{
		ScenarioName:          env.ScenarioName,
		ScenarioDir:           env.ScenarioDir,
		HasUI:                 dirExists(filepath.Join(env.ScenarioDir, "ui")),
		HasAPI:                dirExists(filepath.Join(env.ScenarioDir, "api")),
		Files:                 map[string]bool{},
		ServiceCapabilities:   serviceCapabilities(env.ScenarioDir),
		TestingConfigSections: testingConfigSections(cfg),
	}
	for _, predicate := range predicates {
		if strings.TrimSpace(predicate.FileExists) == "" {
			continue
		}
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(predicate.FileExists)))
		if rel == "." {
			continue
		}
		ctx.Files[rel] = fileExists(filepath.Join(env.ScenarioDir, filepath.FromSlash(rel)))
	}
	return ctx
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
	if len(cfg.Phases) > 0 {
		sections["phases"] = true
	}
	if len(cfg.Presets) > 0 {
		sections["presets"] = true
	}
	if cfg.Requirements.Enforce != nil || cfg.Requirements.Sync != nil {
		sections["requirements"] = true
	}
	return sections
}

func serviceCapabilities(scenarioDir string) map[string]bool {
	capabilities := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return capabilities
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return capabilities
	}
	addStringSlice(capabilities, nestedStringSlice(doc, "service", "tags"))
	addStringSlice(capabilities, nestedStringSlice(doc, "service", "capabilities"))
	addStringSlice(capabilities, nestedStringSlice(doc, "capabilities"))
	return capabilities
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
