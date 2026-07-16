package providerdescriptor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/dimensions"

	"test-genie/internal/orchestrator/phasekeys"
	"test-genie/internal/orchestrator/phasepolicy"
)

const (
	RelPath       = ".vrooli/test-genie.json"
	SchemaVersion = "1.0.0"
)

var evidenceKindPattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)

type Descriptor struct {
	SchemaVersion        string           `json:"schemaVersion"`
	Scenario             string           `json:"scenario"`
	Phase                string           `json:"phase"`
	DisplayName          string           `json:"displayName"`
	Description          string           `json:"description"`
	Source               string           `json:"source"`
	OrderHint            int              `json:"orderHint,omitempty"`
	Timeout              string           `json:"timeout"`
	FindingSource        string           `json:"findingSource,omitempty"`
	ProfileMembership    []string         `json:"profileMembership,omitempty"`
	FreshnessRequirement string           `json:"freshnessRequirement,omitempty"`
	PhaseClass           string           `json:"phaseClass,omitempty"`
	RuntimeClass         string           `json:"runtimeClass,omitempty"`
	Dimensions           []string         `json:"dimensions,omitempty"`
	EvidenceKinds        []string         `json:"evidenceKinds,omitempty"`
	Aliases              []string         `json:"aliases,omitempty"`
	Supersedes           []string         `json:"supersedes,omitempty"`
	Validation           Validation       `json:"validation"`
	Applicability        Applicability    `json:"applicability"`
	Policy               Policy           `json:"policy"`
	Runnability          Runnability      `json:"runnability"`
	Docs                 Docs             `json:"docs,omitempty"`
	Maturity             json.RawMessage  `json:"maturity"`
	Path                 string           `json:"-"`
	TimeoutValue         time.Duration    `json:"-"`
	MaturitySpec         *assessment.Spec `json:"-"`
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
	return nil
}

type Validation struct {
	Contract         string `json:"contract"`
	// DeliveryMode defaults to inline so existing descriptors remain source and
	// behavior compatible. Durable providers must declare execution and the
	// generic durable-run service explicitly.
	DeliveryMode     string `json:"deliveryMode,omitempty"`
	Execution        bool   `json:"execution,omitempty"`
	RunService       string `json:"runService,omitempty"`
	// IncludeExecution is the retired inline delivery flag. It is retained only
	// while existing inline provider descriptors migrate to the explicit
	// execution field; durable descriptors must not use it.
	IncludeExecution bool   `json:"includeExecution,omitempty"`
}

type Applicability struct {
	Default          string      `json:"default"`
	Any              []Predicate `json:"any,omitempty"`
	All              []Predicate `json:"all,omitempty"`
	ApplicabilityRPC string      `json:"applicabilityRpc,omitempty"`
}

type Predicate struct {
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
		case "fileExists", "pathGlob", "scenarioDependency", "serviceCapability", "serviceTag", "hasUI", "hasAPI", "testingConfigSection":
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
	paths := append([]string(nil), opts.Paths...)
	var diagnostics []Diagnostic
	if len(paths) == 0 {
		repoRoot := strings.TrimSpace(opts.RepoRoot)
		if repoRoot == "" {
			repoRoot = "."
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", filepath.FromSlash(RelPath)))
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "glob_failed", Message: err.Error()})
			return LoadResult{Diagnostics: diagnostics}
		}
		paths = matches
	}
	sort.Strings(paths)

	seen := map[string]string{}
	descriptors := make([]Descriptor, 0, len(paths))
	for _, path := range paths {
		descriptor, ds := loadOne(path)
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

func loadOne(path string) (Descriptor, []Diagnostic) {
	var diagnostics []Diagnostic
	raw, err := os.ReadFile(path)
	if err != nil {
		return Descriptor{}, []Diagnostic{{Path: path, Code: "read_failed", Message: err.Error()}}
	}
	var descriptor Descriptor
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return Descriptor{}, []Diagnostic{{Path: path, Code: "invalid_json", Message: err.Error()}}
	}
	descriptor.Path = path
	scenario := scenarioFromPath(path)
	if scenario == "" {
		diagnostics = append(diagnostics, Diagnostic{Path: path, Code: "invalid_path", Message: "descriptor must live at scenarios/<provider>/.vrooli/test-genie.json"})
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
	return descriptor, nil
}

func scenarioFromPath(path string) string {
	clean := filepath.Clean(path)
	if filepath.Base(clean) != "test-genie.json" {
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
	if strings.TrimSpace(d.FreshnessRequirement) == "" {
		d.FreshnessRequirement = "never"
	}
	if strings.TrimSpace(d.PhaseClass) == "" {
		d.PhaseClass = "quality"
	}
	if strings.TrimSpace(d.RuntimeClass) == "" {
		d.RuntimeClass = "static"
	}
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
	}
	return out
}

func countPredicateFields(p Predicate) int {
	count := 0
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
