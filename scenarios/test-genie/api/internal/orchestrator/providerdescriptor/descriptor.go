package providerdescriptor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/dimensions"

	"test-genie/internal/orchestrator/phasekeys"
	"test-genie/internal/orchestrator/phasepolicy"
)

const (
	RelPath       = ".vrooli/test-genie.json"
	SchemaVersion = "1.0.0"
	// BoilerplateDeterminismReason is rejected for static providers because it
	// does not state which external observation makes a file-bound result
	// non-deterministic. Revisit if the descriptor contract gains a structured
	// observation list that makes this prose unnecessary.
	BoilerplateDeterminismReason = "Provider inputs and external observations are not proven to be completely represented by a file digest."
)

var evidenceKindPattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)

type Descriptor struct {
	SchemaVersion string `json:"schemaVersion"`
	Scenario      string `json:"scenario"`
	Phase         string `json:"phase"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Source        string `json:"source"`
	// ConformanceTarget is an optional provider-owned fixture used by the
	// background conformance scan. When omitted, the provider scenario is the
	// target. The orchestrator never needs a provider-name switch.
	ConformanceTarget string `json:"conformanceTarget,omitempty"`
	// DBIsolationProvider identifies the descriptor that owns routed database
	// isolation checks. It is a declaration, not an orchestrator special case.
	DBIsolationProvider        bool             `json:"dbIsolationProvider,omitempty"`
	ArtifactComparisonProvider bool             `json:"artifactComparisonProvider,omitempty"`
	OrderHint                  int              `json:"orderHint,omitempty"`
	Timeout                    string           `json:"timeout"`
	FindingSource              string           `json:"findingSource,omitempty"`
	ProfileMembership          []string         `json:"profileMembership,omitempty"`
	FreshnessRequirement       string           `json:"freshnessRequirement,omitempty"`
	PhaseClass                 string           `json:"phaseClass,omitempty"`
	RuntimeClass               string           `json:"runtimeClass,omitempty"`
	Concurrency                Concurrency      `json:"concurrency,omitempty"`
	Determinism                Determinism      `json:"determinism,omitempty"`
	Dimensions                 []string         `json:"dimensions,omitempty"`
	EvidenceKinds              []string         `json:"evidenceKinds,omitempty"`
	Aliases                    []string         `json:"aliases,omitempty"`
	Supersedes                 []string         `json:"supersedes,omitempty"`
	Comparison                 Comparison       `json:"comparison,omitempty"`
	Validation                 Validation       `json:"validation"`
	Targets                    Targets          `json:"targets,omitempty"`
	Applicability              Applicability    `json:"applicability"`
	Policy                     Policy           `json:"policy"`
	Runnability                Runnability      `json:"runnability"`
	Docs                       Docs             `json:"docs,omitempty"`
	Maturity                   json.RawMessage  `json:"maturity"`
	Path                       string           `json:"-"`
	TimeoutValue               time.Duration    `json:"-"`
	MaturitySpec               *assessment.Spec `json:"-"`
	concurrencyDeclared        bool             `json:"-"`
	determinismDeclared        bool             `json:"-"`
}

type Concurrency struct {
	Mode   string `json:"mode,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Determinism is the provider-owned cache eligibility declaration. The safe
// default is observational: a provider must explicitly name every file input
// before Test Genie can reuse a passed result.
type Determinism struct {
	Default      string                         `json:"default,omitempty"`
	Inputs       []string                       `json:"inputs,omitempty"`
	Reason       string                         `json:"reason,omitempty"`
	Capabilities map[string]DeterminismOverride `json:"capabilities,omitempty"`
}

type DeterminismOverride struct {
	Mode   string   `json:"mode,omitempty"`
	Inputs []string `json:"inputs,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

// DeterminismDeclared reports whether the provider opted into the explicit
// cache-determinism contract. An omitted declaration intentionally remains
// observational for execution, but callers that enforce the contract need to
// distinguish that safe default from an auditable provider declaration.
func (d Descriptor) DeterminismDeclared() bool {
	return d.determinismDeclared
}

type Targets struct {
	Kinds     []string `json:"kinds,omitempty"`
	Selection string   `json:"selection,omitempty"`
}

var validTargetKinds = map[string]struct{}{
	"scenario": {}, "resource": {}, "tool": {}, "safeguard": {},
	"team": {}, "package": {}, "control-plane": {}, "docs": {}, "project": {},
}

var validHostOS = map[string]struct{}{"linux": {}, "macos": {}, "windows": {}}

func (t Targets) EffectiveKinds() []string {
	if len(t.Kinds) == 0 {
		return []string{"scenario"}
	}
	return append([]string(nil), t.Kinds...)
}

func (d *Descriptor) UnmarshalJSON(raw []byte) error {
	type alias Descriptor
	aux := struct {
		*alias
		FindingSource *string `json:"findingSource"`
	}{alias: (*alias)(d)}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return err
	}
	if aux.FindingSource == nil {
		d.FindingSource = ""
	} else {
		d.FindingSource = strings.TrimSpace(*aux.FindingSource)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	_, d.concurrencyDeclared = fields["concurrency"]
	_, d.determinismDeclared = fields["determinism"]
	return nil
}

type Validation struct {
	Contract         string   `json:"contract"`
	CapabilitySubset []string `json:"capabilitySubset,omitempty"`
	// DeliveryMode defaults to inline so existing descriptors remain source and
	// behavior compatible. Durable providers must declare execution and the
	// generic durable-run service explicitly.
	DeliveryMode string `json:"deliveryMode,omitempty"`
	Execution    bool   `json:"execution,omitempty"`
	RunService   string `json:"runService,omitempty"`
	// IncludeExecution is the retired inline delivery flag. It is retained only
	// while existing inline provider descriptors migrate to the explicit
	// execution field; durable descriptors must not use it.
	IncludeExecution bool `json:"includeExecution,omitempty"`
}

type Applicability struct {
	Default          string      `json:"default"`
	Any              []Predicate `json:"any,omitempty"`
	All              []Predicate `json:"all,omitempty"`
	ApplicabilityRPC string      `json:"applicabilityRpc,omitempty"`
}

type Predicate struct {
	HostOS               string `json:"hostOS,omitempty"`
	TargetKind           string `json:"targetKind,omitempty"`
	FileExists           string `json:"fileExists,omitempty"`
	PathGlob             string `json:"pathGlob,omitempty"`
	ScenarioDependency   string `json:"scenarioDependency,omitempty"`
	ServiceCapability    string `json:"serviceCapability,omitempty"`
	ServiceTag           string `json:"serviceTag,omitempty"`
	HasUI                *bool  `json:"hasUI,omitempty"`
	HasAPI               *bool  `json:"hasAPI,omitempty"`
	TestingConfigSection string `json:"testingConfigSection,omitempty"`
	unknownFields        []string
}

func (p *Predicate) UnmarshalJSON(raw []byte) error {
	type alias Predicate
	var decoded alias
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for key := range fields {
		switch key {
		case "hostOS", "targetKind", "fileExists", "pathGlob", "scenarioDependency", "serviceCapability", "serviceTag", "hasUI", "hasAPI", "testingConfigSection":
		default:
			decoded.unknownFields = append(decoded.unknownFields, key)
		}
	}
	*p = Predicate(decoded)
	return nil
}

type Policy struct {
	phasepolicy.Policy
}

// Comparison is the provider's explicit declaration for a same-key oracle
// revision. The default is changed-unreviewed: a changed validator must be
// reviewed instead of silently being treated as behavior evidence.
type Comparison struct {
	Mode string `json:"mode,omitempty"`
}

type Runnability struct {
	NeedsUI                   bool     `json:"needsUI,omitempty"`
	NeedsAPI                  bool     `json:"needsAPI,omitempty"`
	MutatesLifecycle          bool     `json:"mutatesLifecycle,omitempty"`
	LifecycleDecisionDeferred bool     `json:"lifecycleDecisionDeferred,omitempty"`
	DBIsolation               string   `json:"dbIsolation,omitempty"`
	RequiredResources         []string `json:"requiredResources,omitempty"`
}

type Docs struct {
	Path string `json:"path,omitempty"`
}

type Diagnostic struct {
	Path    string `json:"path,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LoadOptions struct {
	RepoRoot string
	Paths    []string
	// SkillIDs optionally supplies the active Prompt Manager catalog. Repository
	// loads derive it from Prompt Manager's pack order; callers that validate an
	// isolated descriptor can supply the same catalog explicitly.
	SkillIDs map[string]struct{}
}

type LoadResult struct {
	Descriptors []Descriptor
	Diagnostics []Diagnostic
}

func (r LoadResult) Err() error {
	if len(r.Diagnostics) == 0 {
		return nil
	}
	parts := make([]string, 0, len(r.Diagnostics))
	for _, d := range r.Diagnostics {
		if d.Path == "" {
			parts = append(parts, d.Code+": "+d.Message)
			continue
		}
		parts = append(parts, d.Path+": "+d.Code+": "+d.Message)
	}
	return errors.New(strings.Join(parts, "; "))
}

func Load(opts LoadOptions) LoadResult {
	skillIDs := opts.SkillIDs
	if skillIDs == nil && strings.TrimSpace(opts.RepoRoot) != "" {
		var err error
		skillIDs, err = loadPromptManagerSkillIDs(opts.RepoRoot)
		if err != nil {
			catalogDir := filepath.Join(opts.RepoRoot, "scenarios", "prompt-manager")
			if _, statErr := os.Stat(catalogDir); statErr == nil {
				return LoadResult{Diagnostics: []Diagnostic{{Code: "skill_catalog_unavailable", Message: err.Error()}}}
			}
			// Isolated orchestrator fixtures intentionally contain only the
			// descriptors under test. Production repositories always include
			// Prompt Manager and therefore always validate skill references.
			skillIDs = nil
		}
	}
	paths := append([]string(nil), opts.Paths...)
	var diagnostics []Diagnostic
	if len(paths) == 0 {
		repoRoot := strings.TrimSpace(opts.RepoRoot)
		if repoRoot == "" {
			repoRoot = "."
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "test-genie*.json"))
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "glob_failed", Message: err.Error()})
			return LoadResult{Diagnostics: diagnostics}
		}
		paths = matches
	}
	sort.Strings(paths)

	seen := map[string]string{}
	descriptors := make([]Descriptor, 0, len(paths))
	schemaPath := descriptorSchemaPath(opts.RepoRoot, paths)
	for _, path := range paths {
		descriptor, ds := loadOne(path, skillIDs, schemaPath)
		diagnostics = append(diagnostics, ds...)
		if len(ds) > 0 {
			continue
		}
		phaseKey := phasekeys.NormalizeKey(descriptor.Phase)
		if firstPath, ok := seen[phaseKey]; ok {
			diagnostics = append(diagnostics, Diagnostic{
				Path:    path,
				Code:    "duplicate_phase",
				Message: fmt.Sprintf("phase %q already declared by %s", descriptor.Phase, firstPath),
			})
			continue
		}
		seen[phaseKey] = path
		descriptors = append(descriptors, descriptor)
	}
	diagnostics = append(diagnostics, validateLineage(descriptors)...)
	return LoadResult{Descriptors: descriptors, Diagnostics: diagnostics}
}

// validateLineage keeps immutable machine keys primary. Aliases and supersedes
// may name retired keys only; they cannot collide with an active phase or with
// another descriptor's lineage declaration.
func validateLineage(descriptors []Descriptor) []Diagnostic {
	active := make(map[string]Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		active[descriptor.Phase] = descriptor
	}
	claimed := map[string]Descriptor{}
	var diagnostics []Diagnostic
	for _, descriptor := range descriptors {
		for _, lineage := range append(append([]string(nil), descriptor.Aliases...), descriptor.Supersedes...) {
			if owner, ok := active[lineage]; ok {
				diagnostics = append(diagnostics, Diagnostic{
					Path: descriptor.Path, Code: "lineage_active_phase_collision",
					Message: fmt.Sprintf("lineage key %q is still an active phase declared by %s", lineage, owner.Path),
				})
				continue
			}
			if owner, ok := claimed[lineage]; ok && owner.Phase != descriptor.Phase {
				diagnostics = append(diagnostics, Diagnostic{
					Path: descriptor.Path, Code: "duplicate_lineage_key",
					Message: fmt.Sprintf("lineage key %q is already claimed by phase %q in %s", lineage, owner.Phase, owner.Path),
				})
				continue
			}
			claimed[lineage] = descriptor
		}
	}
	return diagnostics
}

func loadOne(path string, skillIDs map[string]struct{}, schemaPath string) (Descriptor, []Diagnostic) {
	var diagnostics []Diagnostic
	raw, err := os.ReadFile(path)
	if err != nil {
		return Descriptor{}, []Diagnostic{{Path: path, Code: "read_failed", Message: err.Error()}}
	}
	if schemaPath != "" {
		if err := validateDescriptorSchema(raw, schemaPath); err != nil {
			return Descriptor{}, []Diagnostic{{Path: path, Code: "schema_validation_failed", Message: err.Error()}}
		}
	}
	var descriptor Descriptor
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return Descriptor{}, []Diagnostic{{Path: path, Code: "invalid_json", Message: err.Error()}}
	}
	descriptor.Path = path
	scenario := scenarioFromPath(path)
	if scenario == "" {
		diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "invalid_path", Message: "descriptor must live under a scenario .vrooli directory with a test-genie*.json name"})
	} else if descriptor.Scenario != scenario {
		diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "scenario_mismatch", Message: fmt.Sprintf("scenario %q must match directory %q", descriptor.Scenario, scenario)})
	}
	if maturityExists(path) {
		diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "leftover_maturity_json", Message: ".vrooli/maturity.json must be removed after descriptor cutover"})
	}
	diagnostics = append(diagnostics, validateDescriptor(&descriptor)...)
	if len(diagnostics) > 0 {
		return Descriptor{}, diagnostics
	}
	spec, err := assessment.ParseEmbeddedSpec(descriptor.Maturity, descriptor.Scenario, descriptor.Phase)
	if err != nil {
		return Descriptor{}, []Diagnostic{{Path: path, Code: "invalid_maturity", Message: err.Error()}}
	}
	descriptor.MaturitySpec = spec
	if skillIDs != nil {
		if diagnostics := validateRecommendedSkillIDs(path, spec, skillIDs); len(diagnostics) > 0 {
			return Descriptor{}, diagnostics
		}
	}
	return descriptor, nil
}

func descriptorSchemaPath(repoRoot string, descriptorPaths []string) string {
	if strings.TrimSpace(repoRoot) != "" {
		path := filepath.Join(repoRoot, "scenarios", "test-genie", "schemas", "test-genie-phase-descriptor.schema.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}
	for _, descriptorPath := range descriptorPaths {
		for dir := filepath.Dir(descriptorPath); ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "..", "test-genie", "schemas", "test-genie-phase-descriptor.schema.json")
			if _, err := os.Stat(candidate); err == nil {
				return filepath.Clean(candidate)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return ""
}

func validateDescriptorSchema(raw []byte, schemaPath string) error {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read descriptor schema %s: %w", schemaPath, err)
	}
	const schemaURL = "https://vrooli.dev/schemas/test-genie-phase-descriptor.schema.json"
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("compile descriptor schema: %w", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile descriptor schema: %w", err)
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode descriptor: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("descriptor does not match schema: %w", err)
	}
	return nil
}

type promptManagerPackOrder struct {
	ActivePacks []string `json:"activePacks"`
}

type promptManagerSkill struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func loadPromptManagerSkillIDs(repoRoot string) (map[string]struct{}, error) {
	path := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "_pack-order.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prompt-manager skill catalog: %w", err)
	}
	var order promptManagerPackOrder
	if err := json.Unmarshal(raw, &order); err != nil {
		return nil, fmt.Errorf("parse prompt-manager skill catalog: %w", err)
	}
	ids := make(map[string]struct{})
	for _, pack := range order.ActivePacks {
		entries, err := os.ReadDir(filepath.Join(filepath.Dir(path), "packs", pack))
		if err != nil {
			return nil, fmt.Errorf("read prompt-manager skill pack %q: %w", pack, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(filepath.Dir(path), "packs", pack, entry.Name(), "skill.json")
			skillRaw, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}
			var skill promptManagerSkill
			if err := json.Unmarshal(skillRaw, &skill); err != nil || strings.TrimSpace(skill.ID) == "" {
				continue
			}
			if _, seen := ids[skill.ID]; !seen {
				ids[skill.ID] = struct{}{}
			}
		}
	}
	return ids, nil
}

func validateRecommendedSkillIDs(path string, spec *assessment.Spec, skillIDs map[string]struct{}) []Diagnostic {
	if spec == nil {
		return nil
	}
	var diagnostics []Diagnostic
	for code, mapping := range spec.Findings {
		for _, id := range mapping.RecommendedSkillIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "invalid_recommended_skill", Message: fmt.Sprintf("maturity.findings.%s has an empty recommended_skill_ids entry", code)})
				continue
			}
			if _, ok := skillIDs[id]; !ok {
				diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "unknown_recommended_skill", Message: fmt.Sprintf("maturity.findings.%s references unknown active prompt-manager skill %q", code, id)})
			}
		}
	}
	return diagnostics
}

func scenarioFromPath(path string) string {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if !strings.HasPrefix(base, "test-genie") || !strings.HasSuffix(base, ".json") {
		return ""
	}
	vrooliDir := filepath.Dir(clean)
	if filepath.Base(vrooliDir) != ".vrooli" {
		return ""
	}
	return filepath.Base(filepath.Dir(vrooliDir))
}

func maturityExists(descriptorPath string) bool {
	path := filepath.Join(filepath.Dir(descriptorPath), "maturity.json")
	_, err := os.Stat(path)
	return err == nil
}

func validateDescriptor(d *Descriptor) []Diagnostic {
	var out []Diagnostic
	add := func(code, msg string) {
		out = append(out, Diagnostic{Path: d.Path, Code: code, Message: msg})
	}
	if d.SchemaVersion != SchemaVersion {
		add("invalid_schema_version", fmt.Sprintf("schemaVersion must be %q", SchemaVersion))
	}
	if strings.TrimSpace(d.Scenario) == "" {
		add("missing_scenario", "scenario is required")
	}
	phase := strings.TrimSpace(d.Phase)
	if phase == "" {
		add("missing_phase", "phase is required")
	} else if normalized := phasekeys.NormalizeKey(phase); normalized != phase {
		add("invalid_phase", fmt.Sprintf("phase must already be normalized as %q", normalized))
	}
	if strings.TrimSpace(d.Description) == "" {
		add("missing_description", "description is required")
	}
	if !oneOf(d.Source, "validation-provider") {
		add("invalid_source", "source must be validation-provider")
	}
	if d.Timeout == "" {
		add("missing_timeout", "timeout is required")
	} else if timeout, err := time.ParseDuration(d.Timeout); err != nil || timeout <= 0 {
		add("invalid_timeout", "timeout must be a positive Go duration")
	} else {
		d.TimeoutValue = timeout
	}
	if d.Validation.Contract != "scenario-validation/v1" {
		add("invalid_validation_contract", "validation.contract must be scenario-validation/v1")
	}
	if len(d.Targets.Kinds) == 0 {
		add("missing_targets", "targets.kinds is required; declare the target kinds this provider actually validates")
	}
	if d.Targets.Selection != "" && d.Targets.Selection != "enumerate" {
		add("invalid_target_selection", "targets.selection must be enumerate when present")
	}
	seenTargetKinds := map[string]struct{}{}
	for _, kind := range d.Targets.Kinds {
		kind = strings.TrimSpace(kind)
		if _, ok := validTargetKinds[kind]; !ok {
			add("invalid_target_kind", fmt.Sprintf("targets.kinds contains unsupported kind %q", kind))
		}
		if _, ok := seenTargetKinds[kind]; ok {
			add("duplicate_target_kind", fmt.Sprintf("targets.kinds contains duplicate %q", kind))
		}
		seenTargetKinds[kind] = struct{}{}
	}
	normalizeValidationDefaults(&d.Validation)
	if !oneOf(d.Validation.DeliveryMode, "inline", "durable-run") {
		add("invalid_validation_delivery_mode", "validation.deliveryMode must be inline or durable-run")
	} else if d.Validation.DeliveryMode == "durable-run" {
		if !d.Validation.Execution {
			add("durable_delivery_requires_execution", "validation.execution must be true for durable-run delivery")
		}
		if d.Validation.IncludeExecution {
			add("durable_delivery_rejects_legacy_include_execution", "durable-run delivery must use validation.execution, not includeExecution")
		}
		if d.Validation.RunService != "scenario-validation/v1.DurableValidationRunService" {
			add("durable_delivery_requires_run_service", "durable-run delivery requires validation.runService=scenario-validation/v1.DurableValidationRunService")
		}
	} else if strings.TrimSpace(d.Validation.RunService) != "" {
		add("inline_delivery_rejects_run_service", "inline delivery must not declare validation.runService")
	}
	if !oneOf(d.Runnability.DBIsolation, "", "none", "routed") {
		add("invalid_db_isolation", "runnability.dbIsolation must be none or routed")
	}
	if strings.TrimSpace(d.Docs.Path) == "" {
		add("missing_docs_path", "docs.path is required")
	}
	normalizeOrchestrationDefaults(d)
	out = append(out, validateOrchestration(d)...)
	out = append(out, validateApplicability(d)...)
	out = append(out, validatePolicy(d)...)
	if len(d.Maturity) == 0 || string(d.Maturity) == "null" {
		add("missing_maturity", "maturity is required")
	}
	return out
}

func normalizeValidationDefaults(validation *Validation) {
	if validation == nil {
		return
	}
	validation.DeliveryMode = strings.TrimSpace(validation.DeliveryMode)
	if validation.DeliveryMode == "" {
		validation.DeliveryMode = "inline"
	}
	validation.RunService = strings.TrimSpace(validation.RunService)
	// Existing inline descriptors use includeExecution. Preserve their behavior
	// while giving the orchestrator a single semantic execution field.
	if validation.DeliveryMode == "inline" && validation.IncludeExecution {
		validation.Execution = true
	}
}

func normalizeOrchestrationDefaults(d *Descriptor) {
	d.Comparison.Mode = strings.TrimSpace(d.Comparison.Mode)
	if d.Comparison.Mode == "" {
		d.Comparison.Mode = "changed-unreviewed"
	}
	if strings.TrimSpace(d.FreshnessRequirement) == "" {
		d.FreshnessRequirement = "never"
	}
	if strings.TrimSpace(d.PhaseClass) == "" {
		d.PhaseClass = "quality"
	}
	if strings.TrimSpace(d.RuntimeClass) == "" {
		d.RuntimeClass = "static"
	}
	d.Concurrency.Mode = strings.TrimSpace(strings.ToLower(d.Concurrency.Mode))
	d.Concurrency.Reason = strings.TrimSpace(d.Concurrency.Reason)
	if d.Concurrency.Mode == "" {
		d.Concurrency.Mode = "exclusive"
	}
	normalizeDeterminism(&d.Determinism)
	for i, profile := range d.ProfileMembership {
		d.ProfileMembership[i] = phasekeys.NormalizeKey(profile)
	}
	for i, dim := range d.Dimensions {
		d.Dimensions[i] = strings.TrimSpace(dim)
	}
	for i, kind := range d.EvidenceKinds {
		d.EvidenceKinds[i] = strings.ToLower(strings.TrimSpace(kind))
	}
	for i, alias := range d.Aliases {
		d.Aliases[i] = phasekeys.NormalizeKey(alias)
	}
	for i, superseded := range d.Supersedes {
		d.Supersedes[i] = phasekeys.NormalizeKey(superseded)
	}
}

func normalizeDeterminism(d *Determinism) {
	if d == nil {
		return
	}
	d.Default = strings.ToLower(strings.TrimSpace(d.Default))
	if d.Default == "" {
		d.Default = "observational"
	}
	d.Reason = strings.TrimSpace(d.Reason)
	for i, input := range d.Inputs {
		d.Inputs[i] = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	}
	for capability, override := range d.Capabilities {
		override.Mode = strings.ToLower(strings.TrimSpace(override.Mode))
		override.Reason = strings.TrimSpace(override.Reason)
		for i, input := range override.Inputs {
			override.Inputs[i] = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
		}
		d.Capabilities[capability] = override
	}
}

func validateOrchestration(d *Descriptor) []Diagnostic {
	var out []Diagnostic
	add := func(code, msg string) {
		out = append(out, Diagnostic{Path: d.Path, Code: code, Message: msg})
	}
	seenProfiles := map[string]struct{}{}
	for _, profile := range d.ProfileMembership {
		if !oneOf(profile, "architecture-audit") {
			add("invalid_profile_membership", fmt.Sprintf("profileMembership contains unsupported profile %q", profile))
			continue
		}
		if _, exists := seenProfiles[profile]; exists {
			add("duplicate_profile_membership", fmt.Sprintf("profileMembership contains duplicate profile %q", profile))
		}
		seenProfiles[profile] = struct{}{}
	}
	if !oneOf(d.FreshnessRequirement, "never", "always", "when_applicable") {
		add("invalid_freshness_requirement", "freshnessRequirement must be never, always, or when_applicable")
	}
	if !oneOf(d.PhaseClass, "quality", "architecture", "runtime", "provider-contract", "capability") {
		add("invalid_phase_class", "phaseClass must be quality, architecture, runtime, provider-contract, or capability")
	}
	if !oneOf(d.RuntimeClass, "static", "execution", "lifecycle") {
		add("invalid_runtime_class", "runtimeClass must be static, execution, or lifecycle")
	}
	if !oneOf(d.Concurrency.Mode, "parallel-safe", "exclusive", "provider-serial") {
		add("invalid_concurrency_mode", "concurrency.mode must be parallel-safe, exclusive, or provider-serial")
	}
	if d.concurrencyDeclared && d.Concurrency.Mode == "exclusive" && d.Concurrency.Reason == "" {
		add("missing_concurrency_reason", "concurrency.reason is required when mode is exclusive")
	}
	if d.determinismDeclared {
		if !oneOf(d.Determinism.Default, "file-determined", "observational") {
			add("invalid_determinism_mode", "determinism.default must be file-determined or observational")
		}
		if d.Determinism.Default == "file-determined" && len(d.Determinism.Inputs) == 0 {
			add("missing_determinism_inputs", "determinism.inputs is required when determinism.default is file-determined")
		}
		if d.Determinism.Default == "observational" && d.Determinism.Reason == "" {
			add("missing_determinism_reason", "determinism.reason is required when determinism.default is observational")
		}
		if d.RuntimeClass == "static" && d.Determinism.Default == "observational" {
			add("static_provider_declared_observational", "runtimeClass=static must declare file-determined determinism or explain a concrete external observation")
		}
		if d.RuntimeClass == "static" && d.Determinism.Reason == BoilerplateDeterminismReason {
			add("boilerplate_determinism_reason", "static providers must name the external observation that prevents file-determined caching")
		}
		for capability, override := range d.Determinism.Capabilities {
			if !oneOf(override.Mode, "file-determined", "observational") {
				add("invalid_determinism_capability", fmt.Sprintf("determinism capability %q has invalid mode %q", capability, override.Mode))
			}
			if override.Mode == "file-determined" && len(override.Inputs) == 0 {
				add("missing_determinism_capability_inputs", fmt.Sprintf("determinism capability %q requires inputs", capability))
			}
			if override.Mode == "observational" && override.Reason == "" {
				add("missing_determinism_capability_reason", fmt.Sprintf("determinism capability %q requires a reason", capability))
			}
		}
	}
	if !oneOf(d.Comparison.Mode, "compatible", "changed-unreviewed", "invalidated", "superseded") {
		add("invalid_comparison_mode", "comparison.mode must be compatible, changed-unreviewed, invalidated, or superseded")
	}
	seenDims := map[string]struct{}{}
	for _, raw := range d.Dimensions {
		if raw == "" {
			add("invalid_dimension", "dimensions cannot contain empty values")
			continue
		}
		if !dimensions.IsValid(dimensions.Dimension(raw)) {
			add("invalid_dimension", fmt.Sprintf("dimension %q is not in dimensions.json", raw))
			continue
		}
		if _, exists := seenDims[raw]; exists {
			add("duplicate_dimension", fmt.Sprintf("dimensions contains duplicate dimension %q", raw))
		}
		seenDims[raw] = struct{}{}
	}
	validateUniqueKeys := func(field string, values []string) {
		seen := map[string]struct{}{}
		for _, value := range values {
			if value == "" {
				add("invalid_"+field, field+" cannot contain empty values")
				continue
			}
			if value == d.Phase {
				add("invalid_"+field, fmt.Sprintf("%s cannot contain the phase's own machine key %q", field, value))
			}
			if _, exists := seen[value]; exists {
				add("duplicate_"+field, fmt.Sprintf("%s contains duplicate %q", field, value))
			}
			seen[value] = struct{}{}
		}
	}
	validateUniqueKeys("aliases", d.Aliases)
	validateUniqueKeys("supersedes", d.Supersedes)
	lineageKeys := map[string]string{}
	for _, item := range []struct {
		field  string
		values []string
	}{{field: "aliases", values: d.Aliases}, {field: "supersedes", values: d.Supersedes}} {
		for _, value := range item.values {
			if firstField, exists := lineageKeys[value]; exists && firstField != item.field {
				add("duplicate_lineage_key", fmt.Sprintf("lineage key %q appears in both %s and %s", value, firstField, item.field))
			}
			lineageKeys[value] = item.field
		}
	}
	seenKinds := map[string]struct{}{}
	for _, kind := range d.EvidenceKinds {
		if !evidenceKindPattern.MatchString(kind) {
			add("invalid_evidence_kind", fmt.Sprintf("evidence kind %q must be a stable lowercase token", kind))
			continue
		}
		if _, exists := seenKinds[kind]; exists {
			add("duplicate_evidence_kind", fmt.Sprintf("evidenceKinds contains duplicate %q", kind))
		}
		seenKinds[kind] = struct{}{}
	}
	return out
}

func validateApplicability(d *Descriptor) []Diagnostic {
	var out []Diagnostic
	add := func(code, msg string) {
		out = append(out, Diagnostic{Path: d.Path, Code: code, Message: msg})
	}
	if !oneOf(d.Applicability.Default, "applies", "not_applicable", "unknown") {
		add("invalid_applicability_default", "applicability.default must be applies, not_applicable, or unknown")
	}
	if strings.TrimSpace(d.Applicability.ApplicabilityRPC) != "" {
		add("unsupported_applicability_rpc", "applicabilityRpc is reserved but not implemented")
	}
	if len(d.Applicability.Any) > 0 && len(d.Applicability.All) > 0 {
		add("ambiguous_applicability", "applicability.any and applicability.all are mutually exclusive")
	}
	for i, predicate := range append(append([]Predicate(nil), d.Applicability.Any...), d.Applicability.All...) {
		if len(predicate.unknownFields) > 0 {
			add("invalid_predicate", fmt.Sprintf("predicate %d uses unsupported fields: %s", i, strings.Join(predicate.unknownFields, ", ")))
			continue
		}
		if countPredicateFields(predicate) != 1 {
			add("invalid_predicate", fmt.Sprintf("predicate %d must set exactly one supported field", i))
		}
		if predicate.HostOS != "" {
			if _, ok := validHostOS[strings.ToLower(strings.TrimSpace(predicate.HostOS))]; !ok {
				add("invalid_predicate_host_os", fmt.Sprintf("predicate %d uses unsupported hostOS %q", i, predicate.HostOS))
			}
		}
		if predicate.TargetKind != "" {
			if _, ok := validTargetKinds[predicate.TargetKind]; !ok {
				add("invalid_predicate_target_kind", fmt.Sprintf("predicate %d uses unsupported target kind %q", i, predicate.TargetKind))
			}
		}
	}
	return out
}

func countPredicateFields(p Predicate) int {
	count := 0
	if strings.TrimSpace(p.HostOS) != "" {
		count++
	}
	if strings.TrimSpace(p.TargetKind) != "" {
		count++
	}
	if strings.TrimSpace(p.FileExists) != "" {
		count++
	}
	if strings.TrimSpace(p.PathGlob) != "" {
		count++
	}
	if strings.TrimSpace(p.ScenarioDependency) != "" {
		count++
	}
	if strings.TrimSpace(p.ServiceCapability) != "" {
		count++
	}
	if strings.TrimSpace(p.ServiceTag) != "" {
		count++
	}
	if p.HasUI != nil {
		count++
	}
	if p.HasAPI != nil {
		count++
	}
	if strings.TrimSpace(p.TestingConfigSection) != "" {
		count++
	}
	return count
}

func validatePolicy(d *Descriptor) []Diagnostic {
	var out []Diagnostic
	for _, err := range d.Policy.Policy.Validate() {
		out = append(out, Diagnostic{Path: d.Path, Code: err.Code, Message: err.Message})
	}
	return out
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
