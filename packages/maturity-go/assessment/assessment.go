// Package assessment validates provider-owned health maturity specs and
// normalizes finding maturity metadata into the global maturity vocabulary.
package assessment

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/dimensions"
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	GlobalImpact     string
	CleanRequirement string
)

const (
	ImpactFoundationBlocker GlobalImpact = "foundation_blocker"
	ImpactSafetyBlocker     GlobalImpact = "safety_blocker"
	ImpactEvolvabilityGap   GlobalImpact = "evolvability_gap"
	ImpactHardeningGap      GlobalImpact = "hardening_gap"
	ImpactCapabilityGap     GlobalImpact = "capability_gap"
	ImpactAdvisory          GlobalImpact = "advisory"
	ImpactUnknown           GlobalImpact = "unknown"
)

const (
	CleanRequirementRequired    CleanRequirement = "required"
	CleanRequirementAdvisory    CleanRequirement = "advisory"
	CleanRequirementUncheckable CleanRequirement = "uncheckable"
)

var validImpacts = map[GlobalImpact]struct{}{
	ImpactFoundationBlocker: {},
	ImpactSafetyBlocker:     {},
	ImpactEvolvabilityGap:   {},
	ImpactHardeningGap:      {},
	ImpactCapabilityGap:     {},
	ImpactAdvisory:          {},
	ImpactUnknown:           {},
}

var validCleanRequirements = map[CleanRequirement]struct{}{
	CleanRequirementRequired:    {},
	CleanRequirementAdvisory:    {},
	CleanRequirementUncheckable: {},
}

const defaultCapabilityID = "local"

// FixClass declares whether a finding category can EVER be auto-remediated. It is
// the spec-level intent, orthogonal to whether the fixer is built yet (see
// FixerStatus). It deliberately separates "needs human judgment" (manual) from
// "could be automated" (auto/external) so under-building a fixer cannot masquerade
// as not-fixable.
type FixClass string

const (
	// FixClassAuto marks a finding a deterministic in-process fixer can remediate.
	FixClassAuto FixClass = "auto"
	// FixClassExternal marks a finding remediated by delegating to another tool or
	// scenario (e.g. scenario-auditor).
	FixClassExternal FixClass = "external"
	// FixClassManual marks a finding that inherently needs judgment — the honest
	// "never autofixable". It MUST carry a reason.
	FixClassManual FixClass = "manual"
)

// FixerStatus declares whether the remediation logic for an auto/external finding
// exists yet. It defaults to pending so a newly-declared fixable finding is a
// visible gap until a fixer is wired. It is meaningless for manual findings.
type FixerStatus string

const (
	// FixerStatusImplemented marks a fixable finding whose fixer is built.
	FixerStatusImplemented FixerStatus = "implemented"
	// FixerStatusPending marks a fixable finding whose fixer is not built yet —
	// the actionable backlog the optimization lens watches.
	FixerStatusPending FixerStatus = "pending"
)

var validFixClasses = map[FixClass]struct{}{
	FixClassAuto:     {},
	FixClassExternal: {},
	FixClassManual:   {},
}

var validFixerStatuses = map[FixerStatus]struct{}{
	FixerStatusImplemented: {},
	FixerStatusPending:     {},
}

// IsValidFixClass reports whether the value is a declared fix-class vocabulary
// member.
func IsValidFixClass(c FixClass) bool {
	_, ok := validFixClasses[c]
	return ok
}

// IsValidFixerStatus reports whether the value is a declared fixer-status
// vocabulary member.
func IsValidFixerStatus(s FixerStatus) bool {
	_, ok := validFixerStatuses[s]
	return ok
}

var impactDimensions = map[GlobalImpact][]dimensions.Dimension{
	ImpactFoundationBlocker: dims("tests", "standards", "structure"),
	ImpactSafetyBlocker:     dims("security", "dependencies"),
	ImpactEvolvabilityGap:   dims("structure", "cycles", "contracts", "docs", "proto-health", "dependency-accuracy"),
	ImpactHardeningGap:      dims("tests", "coverage", "tidiness", "ui", "visual", "performance"),
	ImpactCapabilityGap:     dims("operational-targets", "business", "measures"),
	ImpactAdvisory:          nil,
	ImpactUnknown:           nil,
}

// Spec is the maturity block schema embedded in provider-owned Test Genie
// descriptors.
type Spec struct {
	Provider     string                    `json:"provider"`
	Phase        string                    `json:"phase"`
	Version      string                    `json:"version"`
	Levels       []Level                   `json:"levels"`
	Capabilities []CapabilitySpec          `json:"capabilities,omitempty"`
	Findings     map[string]FindingMapping `json:"findings"`
	Fallback     FallbackPolicy            `json:"fallback"`
}

type Level struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	EntryCriteria     []string `json:"entry_criteria"`
	ExitCriteria      []string `json:"exit_criteria"`
	StatusLabel       string   `json:"status_label,omitempty"`
	CapabilitySummary string   `json:"capability_summary,omitempty"`
	NextUnlock        string   `json:"next_unlock,omitempty"`
}

type CapabilitySpec struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Levels      []Level `json:"levels"`
}

type FindingMapping struct {
	CapabilityID        string       `json:"capability_id,omitempty"`
	LocalLevelImpact    string       `json:"local_level_impact"`
	GlobalImpact        GlobalImpact `json:"global_impact"`
	Dimension           string       `json:"dimension"`
	SeverityDefault     string       `json:"severity_default"`
	CleanRequirement    string       `json:"clean_requirement,omitempty"`
	RecommendedSkillIDs []string     `json:"recommended_skill_ids"`
	// FixClass declares whether this finding category can ever be auto-remediated
	// (auto|external|manual). Absent → manual (conservative; never inflates the
	// fixable universe).
	FixClass FixClass `json:"fix_class,omitempty"`
	// FixerStatus declares whether the fixer exists yet (implemented|pending) for
	// auto/external classes. Absent on auto/external → pending. Meaningless for
	// manual.
	FixerStatus FixerStatus `json:"fixer_status,omitempty"`
	// FixReason justifies a manual classification — required when FixClass is
	// manual so excluding a finding from the fixable universe is reviewable.
	FixReason string `json:"reason,omitempty"`
}

// EffectiveFixClass returns the declared fix class, defaulting an absent
// declaration to manual (the conservative choice that never inflates the
// fixable universe).
func (m FindingMapping) EffectiveFixClass() FixClass {
	if m.FixClass == "" {
		return FixClassManual
	}
	return m.FixClass
}

// EffectiveFixerStatus returns the fixer status for auto/external findings,
// defaulting absent to pending. Manual findings have no fixer status ("").
func (m FindingMapping) EffectiveFixerStatus() FixerStatus {
	switch m.EffectiveFixClass() {
	case FixClassAuto, FixClassExternal:
		if m.FixerStatus == "" {
			return FixerStatusPending
		}
		return m.FixerStatus
	default:
		return ""
	}
}

type FallbackPolicy struct {
	CapabilityID     string       `json:"capability_id,omitempty"`
	LocalLevelImpact string       `json:"local_level_impact"`
	GlobalImpact     GlobalImpact `json:"global_impact"`
	Dimension        string       `json:"dimension"`
	SeverityDefault  string       `json:"severity_default"`
	CleanRequirement string       `json:"clean_requirement,omitempty"`
}

type Finding struct {
	Code        string
	Severity    string
	Title       string
	Message     string
	Location    string
	Remediation string
	Source      architecturev1.FindingSource
	Phase       string
	Maturity    FindingMapping
	HasMaturity bool
	// AutofixAvailable marks that a provider auto-fix can remediate this finding.
	AutofixAvailable bool
	// FixClass is the provider fix classification, e.g. "autofix" or
	// "detection_only".
	FixClass string
}

type FindingAssessment struct {
	Code     string
	Mapping  FindingMapping
	Severity architecturev1.FindingSeverity
}

type LocalResult struct {
	CapabilityID          string
	CapabilityLabel       string
	CapabilityDescription string
	CurrentLevel          string
	NextLevel             string
	CurrentSummary        string
	NextUnlock            string
	Levels                []Level
	BlockingFindingCodes  []string
	Findings              []FindingAssessment
	Clean                 bool
	UnknownCount          int
	PriorityRank          int
	PriorityReason        string
}

type DebtCounts struct {
	Total      int
	BySeverity map[architecturev1.FindingSeverity]int
}

type BuildInput struct {
	Scenario string
	Spec     Spec
	Findings []Finding
}

type validationResponseOptions struct {
	status scenariovalidationv1.ValidationStatus
}

// ValidationResponseOption customizes BuildValidationResponse.
type ValidationResponseOption func(*validationResponseOptions)

// WithValidationStatus overrides the status derived from assessment severity
// counts. Providers use this for DEGRADED, ERROR, or SKIPPED outcomes that are
// not expressible as normal maturity findings.
func WithValidationStatus(status scenariovalidationv1.ValidationStatus) ValidationResponseOption {
	return func(opts *validationResponseOptions) {
		opts.status = status
	}
}

// ParseSpec parses a JSON maturity spec supplied by a caller. The package stays
// pure logic: it never discovers or reads scenario files itself.
func ParseSpec(raw []byte) (*Spec, error) {
	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}
	if err := ValidateSpec(spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func normalizedCapabilities(spec Spec) ([]CapabilitySpec, error) {
	if len(spec.Capabilities) == 0 {
		if len(spec.Levels) == 0 {
			return nil, fmt.Errorf("at least one level is required")
		}
		return []CapabilitySpec{{
			ID:     defaultCapabilityID,
			Label:  "Local Maturity",
			Levels: append([]Level(nil), spec.Levels...),
		}}, nil
	}
	capabilities := make([]CapabilitySpec, 0, len(spec.Capabilities))
	seen := make(map[string]struct{}, len(spec.Capabilities))
	for i, capability := range spec.Capabilities {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			return nil, fmt.Errorf("capabilities[%d].id is required", i)
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("duplicate capability id %q", id)
		}
		seen[id] = struct{}{}
		label := strings.TrimSpace(capability.Label)
		if label == "" {
			return nil, fmt.Errorf("capabilities[%d].label is required", i)
		}
		capability.ID = id
		capability.Label = label
		capabilities = append(capabilities, capability)
	}
	return capabilities, nil
}

func defaultCapabilityFor(capabilities []CapabilitySpec) string {
	if len(capabilities) == 0 {
		return defaultCapabilityID
	}
	return capabilities[0].ID
}

func capabilityPath(id, child string) string {
	if id == defaultCapabilityID {
		return child
	}
	return "capabilities." + id + "." + child
}

func ValidateSpec(spec Spec) error {
	if strings.TrimSpace(spec.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(spec.Phase) == "" {
		return fmt.Errorf("phase is required")
	}
	if strings.TrimSpace(spec.Version) == "" {
		return fmt.Errorf("version is required")
	}
	capabilities, err := normalizedCapabilities(spec)
	if err != nil {
		return err
	}
	levelIndexes := make(map[string]map[string]int, len(capabilities))
	for _, capability := range capabilities {
		levels, err := validateLevels(capability.Levels, capabilityPath(capability.ID, "levels"))
		if err != nil {
			return err
		}
		levelIndexes[capability.ID] = levels
	}
	defaultID := defaultCapabilityFor(capabilities)
	strictCleanRequirement := len(spec.Capabilities) > 0
	for code, mapping := range spec.Findings {
		if strings.TrimSpace(code) == "" {
			return fmt.Errorf("finding code cannot be empty")
		}
		if err := validateMapping(mapping, levelIndexes, defaultID, strictCleanRequirement, "findings."+code); err != nil {
			return err
		}
	}
	return validateMapping(FindingMapping{
		CapabilityID:     spec.Fallback.CapabilityID,
		LocalLevelImpact: spec.Fallback.LocalLevelImpact,
		GlobalImpact:     spec.Fallback.GlobalImpact,
		Dimension:        spec.Fallback.Dimension,
		SeverityDefault:  spec.Fallback.SeverityDefault,
		CleanRequirement: spec.Fallback.CleanRequirement,
	}, levelIndexes, defaultID, strictCleanRequirement, "fallback")
}

func validateLevels(levels []Level, path string) (map[string]int, error) {
	if len(levels) == 0 {
		return nil, fmt.Errorf("%s must declare at least one level", path)
	}
	out := make(map[string]int, len(levels))
	for i, level := range levels {
		id := strings.TrimSpace(level.ID)
		if id == "" {
			return nil, fmt.Errorf("%s[%d].id is required", path, i)
		}
		if _, exists := out[id]; exists {
			return nil, fmt.Errorf("%s duplicate level id %q", path, id)
		}
		out[id] = i
	}
	return out, nil
}

func validateMapping(mapping FindingMapping, levelsByCapability map[string]map[string]int, defaultID string, strictCleanRequirement bool, path string) error {
	capabilityID := strings.TrimSpace(mapping.CapabilityID)
	if capabilityID == "" {
		capabilityID = defaultID
	}
	levels, ok := levelsByCapability[capabilityID]
	if !ok {
		return fmt.Errorf("%s.capability_id %q is not a declared capability", path, capabilityID)
	}
	if strings.TrimSpace(mapping.LocalLevelImpact) != "" {
		if _, ok := levels[mapping.LocalLevelImpact]; !ok {
			return fmt.Errorf("%s.local_level_impact %q is not a declared level for capability %q", path, mapping.LocalLevelImpact, capabilityID)
		}
	}
	if mapping.GlobalImpact == "" {
		return fmt.Errorf("%s.global_impact is required", path)
	}
	if !IsValidImpact(mapping.GlobalImpact) {
		return fmt.Errorf("%s.global_impact %q is not valid", path, mapping.GlobalImpact)
	}
	if dim := strings.TrimSpace(mapping.Dimension); dim != "" && !dimensions.IsValid(dimensions.Dimension(dim)) {
		return fmt.Errorf("%s.dimension %q is not in dimensions.json", path, dim)
	}
	if sev := strings.TrimSpace(mapping.SeverityDefault); sev != "" && normalizeSeverity(sev) == architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED {
		return fmt.Errorf("%s.severity_default %q is not valid", path, sev)
	}
	req := strings.TrimSpace(mapping.CleanRequirement)
	if strictCleanRequirement && req == "" {
		return fmt.Errorf("%s.clean_requirement is required for capability maturity specs", path)
	}
	if req != "" && !IsValidCleanRequirement(CleanRequirement(strings.ToLower(req))) {
		return fmt.Errorf("%s.clean_requirement %q is not valid", path, req)
	}
	if err := validateFixability(mapping, path); err != nil {
		return err
	}
	return nil
}

// validateFixability enforces the fixability declaration vocabulary. It only
// constrains explicitly-declared values, so specs predating fix_class stay valid
// (absent fix_class derives to manual for coverage counts but is not required to
// carry a reason — declaration completeness is an advisory conformance dimension,
// not a hard ValidateSpec gate).
func validateFixability(mapping FindingMapping, path string) error {
	if mapping.FixClass != "" && !IsValidFixClass(mapping.FixClass) {
		return fmt.Errorf("%s.fix_class %q is not valid (want auto|external|manual)", path, mapping.FixClass)
	}
	if mapping.FixerStatus != "" {
		if !IsValidFixerStatus(mapping.FixerStatus) {
			return fmt.Errorf("%s.fixer_status %q is not valid (want implemented|pending)", path, mapping.FixerStatus)
		}
		if mapping.FixClass == FixClassManual || mapping.FixClass == "" {
			return fmt.Errorf("%s.fixer_status is only valid for fix_class auto|external", path)
		}
	}
	if mapping.FixClass == FixClassManual && strings.TrimSpace(mapping.FixReason) == "" {
		return fmt.Errorf("%s.reason is required when fix_class is manual", path)
	}
	return nil
}

func IsValidImpact(impact GlobalImpact) bool {
	_, ok := validImpacts[impact]
	return ok
}

func IsValidCleanRequirement(requirement CleanRequirement) bool {
	_, ok := validCleanRequirements[requirement]
	return ok
}

func BuildProtoAssessment(input BuildInput) (*commonv1.MaturityAssessment, error) {
	if strings.TrimSpace(input.Scenario) == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	if err := ValidateSpec(input.Spec); err != nil {
		return nil, err
	}
	local := LocalMaturity(input.Spec, input.Findings)
	capabilities := CapabilityMaturity(input.Spec, input.Findings)
	out := &commonv1.MaturityAssessment{
		Scenario:               strings.TrimSpace(input.Scenario),
		Provider:               input.Spec.Provider,
		Phase:                  input.Spec.Phase,
		Version:                input.Spec.Version,
		Local:                  buildProtoLocal(input.Spec, local),
		Findings:               make([]*commonv1.AssessmentFinding, 0, len(input.Findings)),
		FindingsByGlobalImpact: map[string]int32{},
		FindingsBySeverity:     map[string]int32{},
		FindingsByCleanRequirement: map[string]int32{
			string(CleanRequirementAdvisory):    0,
			string(CleanRequirementRequired):    0,
			string(CleanRequirementUncheckable): 0,
		},
	}
	skills := map[string]struct{}{}
	autofixable := int32(0)
	for i, finding := range input.Findings {
		assessed := local.Findings[i]
		severity := assessed.Severity.String()
		impact := string(assessed.Mapping.GlobalImpact)
		cleanRequirement := normalizeCleanRequirement(assessed.Mapping.CleanRequirement)
		out.FindingsByGlobalImpact[impact]++
		out.FindingsBySeverity[severity]++
		out.FindingsByCleanRequirement[string(cleanRequirement)]++
		if finding.AutofixAvailable {
			autofixable++
		}
		for _, skill := range assessed.Mapping.RecommendedSkillIDs {
			skill = strings.TrimSpace(skill)
			if skill != "" {
				skills[skill] = struct{}{}
			}
		}
		out.Findings = append(out.Findings, &commonv1.AssessmentFinding{
			Code:             finding.Code,
			Severity:         severity,
			Title:            finding.Title,
			Message:          finding.Message,
			Location:         finding.Location,
			Remediation:      finding.Remediation,
			AutofixAvailable: finding.AutofixAvailable,
			FixClass:         strings.TrimSpace(finding.FixClass),
			Maturity: &commonv1.FindingMaturity{
				LocalLevel:          assessed.Mapping.LocalLevelImpact,
				GlobalImpact:        GlobalImpactToProto(assessed.Mapping.GlobalImpact),
				Dimension:           assessed.Mapping.Dimension,
				RecommendedSkillIds: append([]string(nil), assessed.Mapping.RecommendedSkillIDs...),
				CleanRequirement:    CleanRequirementToProto(cleanRequirement),
				CapabilityId:        assessed.Mapping.CapabilityID,
			},
		})
	}
	if len(input.Spec.Capabilities) > 0 {
		out.Capabilities = buildProtoCapabilities(capabilities)
		if focus := protoCapabilityByID(out.Capabilities, local.CapabilityID); focus != nil {
			out.HighestPriorityCapability = &commonv1.PriorityFocus{
				CapabilityId:    focus.GetId(),
				CapabilityLabel: focus.GetLabel(),
				CurrentLevel:    focus.GetCurrentLevel(),
				NextLevel:       focus.GetNextLevel(),
				Reason:          focus.GetPriorityReason(),
			}
		}
	}
	out.AutofixableCount = autofixable
	out.AutofixableTotal = int32(len(input.Findings))
	out.RecommendedSkillIds = sortedKeys(skills)
	if err := ValidateAssessment(out); err != nil {
		return nil, err
	}
	return out, nil
}

func protoCapabilityByID(capabilities []*commonv1.CapabilityMaturityAssessment, id string) *commonv1.CapabilityMaturityAssessment {
	for _, capability := range capabilities {
		if capability.GetId() == id {
			return capability
		}
	}
	if len(capabilities) == 0 {
		return nil
	}
	return capabilities[0]
}

func ValidateAssessment(a *commonv1.MaturityAssessment) error {
	if a == nil {
		return fmt.Errorf("assessment is required")
	}
	if strings.TrimSpace(a.GetScenario()) == "" {
		return fmt.Errorf("assessment.scenario is required")
	}
	if strings.TrimSpace(a.GetProvider()) == "" {
		return fmt.Errorf("assessment.provider is required")
	}
	if strings.TrimSpace(a.GetPhase()) == "" {
		return fmt.Errorf("assessment.phase is required")
	}
	if strings.TrimSpace(a.GetVersion()) == "" {
		return fmt.Errorf("assessment.version is required")
	}
	if a.GetLocal() == nil {
		return fmt.Errorf("assessment.local is required")
	}
	if strings.TrimSpace(a.GetLocal().GetCurrentLevel()) == "" {
		return fmt.Errorf("assessment.local.current_level is required")
	}
	capabilityLevels := map[string]map[string]struct{}{}
	for i, capability := range a.GetCapabilities() {
		if capability == nil {
			return fmt.Errorf("assessment.capabilities[%d] is required", i)
		}
		id := strings.TrimSpace(capability.GetId())
		if id == "" {
			return fmt.Errorf("assessment.capabilities[%d].id is required", i)
		}
		if strings.TrimSpace(capability.GetLabel()) == "" {
			return fmt.Errorf("assessment.capabilities[%d].label is required", i)
		}
		if _, exists := capabilityLevels[id]; exists {
			return fmt.Errorf("assessment.capabilities[%d].id %q is duplicated", i, id)
		}
		levels := map[string]struct{}{}
		for j, level := range capability.GetLevels() {
			if level == nil {
				return fmt.Errorf("assessment.capabilities[%d].levels[%d] is required", i, j)
			}
			levelID := strings.TrimSpace(level.GetId())
			if levelID == "" {
				return fmt.Errorf("assessment.capabilities[%d].levels[%d].id is required", i, j)
			}
			if _, exists := levels[levelID]; exists {
				return fmt.Errorf("assessment.capabilities[%d].levels[%d].id %q is duplicated", i, j, levelID)
			}
			levels[levelID] = struct{}{}
		}
		if _, ok := levels[capability.GetCurrentLevel()]; strings.TrimSpace(capability.GetCurrentLevel()) != "" && !ok {
			return fmt.Errorf("assessment.capabilities[%d].current_level %q is not declared", i, capability.GetCurrentLevel())
		}
		if _, ok := levels[capability.GetNextLevel()]; strings.TrimSpace(capability.GetNextLevel()) != "" && !ok {
			return fmt.Errorf("assessment.capabilities[%d].next_level %q is not declared", i, capability.GetNextLevel())
		}
		capabilityLevels[id] = levels
	}
	for i, finding := range a.GetFindings() {
		if finding == nil {
			return fmt.Errorf("assessment.findings[%d] is required", i)
		}
		if strings.TrimSpace(finding.GetCode()) == "" {
			return fmt.Errorf("assessment.findings[%d].code is required", i)
		}
		if strings.TrimSpace(finding.GetSeverity()) == "" {
			return fmt.Errorf("assessment.findings[%d].severity is required", i)
		}
		if finding.GetMaturity() == nil {
			return fmt.Errorf("assessment.findings[%d].maturity is required", i)
		}
		if finding.GetMaturity().GetGlobalImpact() == commonv1.GlobalImpact_GLOBAL_IMPACT_UNSPECIFIED {
			return fmt.Errorf("assessment.findings[%d].maturity.global_impact is required", i)
		}
		if len(capabilityLevels) > 0 {
			capabilityID := strings.TrimSpace(finding.GetMaturity().GetCapabilityId())
			if capabilityID == "" {
				return fmt.Errorf("assessment.findings[%d].maturity.capability_id is required when capabilities are present", i)
			}
			levels, ok := capabilityLevels[capabilityID]
			if !ok {
				return fmt.Errorf("assessment.findings[%d].maturity.capability_id %q is not emitted", i, capabilityID)
			}
			localLevel := strings.TrimSpace(finding.GetMaturity().GetLocalLevel())
			if localLevel != "" {
				if _, ok := levels[localLevel]; !ok {
					return fmt.Errorf("assessment.findings[%d].maturity.local_level %q is not declared for capability %q", i, localLevel, capabilityID)
				}
			}
		}
	}
	return nil
}

// AutofixCoverage is the per-provider autofix declaration rollup derived from a
// maturity spec. Counts are absolute (the headline is Pending — the actionable
// backlog); ImplementationRate is secondary/informational and never a gate.
type AutofixCoverage struct {
	// Total is the number of declared finding mappings.
	Total int `json:"total"`
	// FixableUniverse is auto+external (findings that can ever be auto-remediated).
	FixableUniverse int `json:"fixableUniverse"`
	// Implemented is fixable findings whose fixer exists (fixer_status=implemented).
	Implemented int `json:"implemented"`
	// Pending is fixable findings whose fixer is not built yet — the headline gap.
	Pending int `json:"pending"`
	// Manual is findings declared (or defaulted) to manual.
	Manual int `json:"manual"`
	// Declared is findings carrying an explicit fix_class.
	Declared int `json:"declared"`
	// DeclarationComplete is true when every finding carries an explicit fix_class
	// (and, since ValidateSpec enforces it, every manual carries a reason). This is
	// the advisory conformance dimension — NOT the coverage ratio.
	DeclarationComplete bool `json:"declarationComplete"`
}

// ImplementationRate returns implemented / (implemented + pending). Pending stays
// in the denominator so an unbuilt fixer drags the rate down — the only honest
// way up is pending→implemented. It returns 0 when the fixable universe is empty.
func (c AutofixCoverage) ImplementationRate() float64 {
	denom := c.Implemented + c.Pending
	if denom == 0 {
		return 0
	}
	return float64(c.Implemented) / float64(denom)
}

// ComputeAutofixCoverage derives the autofix declaration rollup from a spec. It
// applies the conservative defaults (absent fix_class → manual; absent
// fixer_status on auto/external → pending) so under-declaration can never
// masquerade as fixable.
func ComputeAutofixCoverage(spec Spec) AutofixCoverage {
	cov := AutofixCoverage{DeclarationComplete: true}
	for _, m := range spec.Findings {
		cov.Total++
		if m.FixClass != "" {
			cov.Declared++
		} else {
			cov.DeclarationComplete = false
		}
		switch m.EffectiveFixClass() {
		case FixClassAuto, FixClassExternal:
			cov.FixableUniverse++
			if m.EffectiveFixerStatus() == FixerStatusImplemented {
				cov.Implemented++
			} else {
				cov.Pending++
			}
		default:
			cov.Manual++
		}
	}
	return cov
}

// ConsistencyWarnings reports findings whose runtime AutofixAvailable flag
// disagrees with their declared fixability — a finding that claims a fixer ran
// while its mapping says manual or fixer_status=pending. These are contract
// warnings (advisory), not hard failures (Plan Stage 1.2).
func ConsistencyWarnings(spec Spec, findings []Finding) []string {
	var warnings []string
	for _, f := range findings {
		if !f.AutofixAvailable {
			continue
		}
		mapping := NormalizeFinding(spec, f).Mapping
		switch mapping.EffectiveFixClass() {
		case FixClassManual:
			warnings = append(warnings, fmt.Sprintf(
				"finding %q reports a runtime autofix but is declared fix_class=manual", f.Code))
		default:
			if mapping.EffectiveFixerStatus() == FixerStatusPending {
				warnings = append(warnings, fmt.Sprintf(
					"finding %q reports a runtime autofix but its fixer_status is pending", f.Code))
			}
		}
	}
	return warnings
}

// BuildValidationResponse wraps the shared maturity assessment in the common
// scenario-validation response, optionally packs a provider-native payload, and
// optionally attaches execution metrics. A nil metrics argument leaves the
// response's metrics field unset (the contract for providers that have not yet
// adopted metrics).
func BuildValidationResponse(
	scenario string,
	a *commonv1.MaturityAssessment,
	native proto.Message,
	metrics *commonv1.ExecutionMetrics,
	opts ...ValidationResponseOption,
) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	if err := ValidateAssessment(a); err != nil {
		return nil, err
	}
	config := validationResponseOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	status := config.status
	if status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		status = DeriveValidationStatus(a)
	}
	if status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		return nil, fmt.Errorf("validation status is required")
	}
	var detail *anypb.Any
	if native != nil {
		packed, err := anypb.New(native)
		if err != nil {
			return nil, fmt.Errorf("pack native detail: %w", err)
		}
		detail = packed
	}
	return &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:     scenario,
		Status:       status,
		Assessment:   a,
		NativeDetail: detail,
		Metrics:      metrics,
	}, nil
}

// DeriveValidationStatus returns FAILED when the assessment reports error or
// blocker severity findings, otherwise PASSED.
func DeriveValidationStatus(a *commonv1.MaturityAssessment) scenariovalidationv1.ValidationStatus {
	if a == nil {
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR
	}
	for severity, count := range a.GetFindingsBySeverity() {
		if count <= 0 {
			continue
		}
		switch normalizeSeverity(severity) {
		case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
			architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
			return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		}
	}
	return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
}

// AssessmentToArchitectureFindings converts the shared assessment projection
// into the normalized finding envelope Test Genie already emits.
func AssessmentToArchitectureFindings(
	scenario string,
	a *commonv1.MaturityAssessment,
	defaultSource architecturev1.FindingSource,
) []*architecturev1.ArchitectureFinding {
	if a == nil {
		return nil
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		scenario = strings.TrimSpace(a.GetScenario())
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(a.GetFindings()))
	for _, finding := range a.GetFindings() {
		if finding == nil {
			continue
		}
		source := sourceForAssessmentFinding(finding, defaultSource)
		archFinding := &architecturev1.ArchitectureFinding{
			Scenario:     scenario,
			Source:       source,
			Code:         strings.TrimSpace(finding.GetCode()),
			Severity:     normalizeSeverity(finding.GetSeverity()),
			Locations:    nonEmptyStrings(finding.GetLocation()),
			Message:      assessmentFindingMessage(finding),
			Suggestion:   strings.TrimSpace(finding.GetRemediation()),
			Effort:       defaultEffortForSource(source),
			FindingClass: architecturev1.FindingClass_FINDING_CLASS_DETERMINISTIC,
		}
		findingid.Stamp(archFinding)
		out = append(out, archFinding)
	}
	return out
}

func GlobalImpactToProto(impact GlobalImpact) commonv1.GlobalImpact {
	switch impact {
	case ImpactFoundationBlocker:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER
	case ImpactSafetyBlocker:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER
	case ImpactEvolvabilityGap:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	case ImpactHardeningGap:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP
	case ImpactCapabilityGap:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP
	case ImpactAdvisory:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY
	case ImpactUnknown:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_UNKNOWN
	default:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_UNSPECIFIED
	}
}

func CleanRequirementToProto(requirement CleanRequirement) commonv1.CleanRequirement {
	switch requirement {
	case CleanRequirementRequired:
		return commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case CleanRequirementAdvisory:
		return commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY
	case CleanRequirementUncheckable:
		return commonv1.CleanRequirement_CLEAN_REQUIREMENT_UNCHECKABLE
	default:
		return commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY
	}
}

// DimensionsForImpact returns the current global maturity dimensions associated
// with a semantic impact. Advisory and unknown impacts intentionally map to no
// dimensions and rely on caller/fallback context.
func DimensionsForImpact(impact GlobalImpact) []dimensions.Dimension {
	source := impactDimensions[impact]
	out := make([]dimensions.Dimension, len(source))
	copy(out, source)
	return out
}

func buildProtoLocal(spec Spec, local LocalResult) *commonv1.LocalMaturityAssessment {
	levels := buildProtoLevels(local.Levels)
	return &commonv1.LocalMaturityAssessment{
		CurrentLevel:         local.CurrentLevel,
		NextLevel:            local.NextLevel,
		Levels:               levels,
		BlockingFindingCodes: append([]string(nil), local.BlockingFindingCodes...),
		Clean:                local.Clean,
		UnknownCount:         int32(local.UnknownCount),
	}
}

func buildProtoCapabilities(results []LocalResult) []*commonv1.CapabilityMaturityAssessment {
	out := make([]*commonv1.CapabilityMaturityAssessment, 0, len(results))
	for _, result := range results {
		capability := &commonv1.CapabilityMaturityAssessment{
			Id:                         result.CapabilityID,
			Label:                      result.CapabilityLabel,
			Description:                result.CapabilityDescription,
			CurrentLevel:               result.CurrentLevel,
			NextLevel:                  result.NextLevel,
			Levels:                     buildProtoLevels(result.Levels),
			CurrentSummary:             result.CurrentSummary,
			NextUnlock:                 result.NextUnlock,
			BlockingFindingCodes:       append([]string(nil), result.BlockingFindingCodes...),
			Clean:                      result.Clean,
			UnknownCount:               int32(result.UnknownCount),
			FindingsByGlobalImpact:     map[string]int32{},
			FindingsBySeverity:         map[string]int32{},
			FindingsByCleanRequirement: map[string]int32{},
			PriorityRank:               int32(result.PriorityRank),
			PriorityReason:             result.PriorityReason,
		}
		for _, finding := range result.Findings {
			capability.FindingsByGlobalImpact[string(finding.Mapping.GlobalImpact)]++
			capability.FindingsBySeverity[finding.Severity.String()]++
			capability.FindingsByCleanRequirement[string(normalizeCleanRequirement(finding.Mapping.CleanRequirement))]++
		}
		out = append(out, capability)
	}
	return out
}

func buildProtoLevels(levels []Level) []*commonv1.LocalMaturityLevel {
	out := make([]*commonv1.LocalMaturityLevel, 0, len(levels))
	for _, level := range levels {
		out = append(out, &commonv1.LocalMaturityLevel{
			Id:                level.ID,
			Name:              level.Name,
			Description:       level.Description,
			EntryCriteria:     append([]string(nil), level.EntryCriteria...),
			ExitCriteria:      append([]string(nil), level.ExitCriteria...),
			StatusLabel:       level.StatusLabel,
			CapabilitySummary: level.CapabilitySummary,
			NextUnlock:        level.NextUnlock,
		})
	}
	return out
}

func NormalizeFinding(spec Spec, finding Finding) FindingAssessment {
	capabilities, _ := normalizedCapabilities(spec)
	return normalizeFinding(spec, finding, defaultCapabilityFor(capabilities))
}

func normalizeFinding(spec Spec, finding Finding, defaultID string) FindingAssessment {
	mapping := finding.Maturity
	if !finding.HasMaturity {
		if m, ok := spec.Findings[finding.Code]; ok {
			mapping = m
		} else {
			mapping = FindingMapping{
				CapabilityID:     spec.Fallback.CapabilityID,
				LocalLevelImpact: spec.Fallback.LocalLevelImpact,
				GlobalImpact:     spec.Fallback.GlobalImpact,
				Dimension:        spec.Fallback.Dimension,
				SeverityDefault:  spec.Fallback.SeverityDefault,
				CleanRequirement: spec.Fallback.CleanRequirement,
			}
		}
	}
	if strings.TrimSpace(mapping.CapabilityID) == "" {
		mapping.CapabilityID = defaultID
	}
	if strings.TrimSpace(mapping.Dimension) == "" {
		if dim, ok := dimensions.ForSource(finding.Source); ok {
			mapping.Dimension = string(dim)
		}
	}
	severity := normalizeSeverity(finding.Severity)
	if severity == architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED {
		severity = normalizeSeverity(mapping.SeverityDefault)
	}
	return FindingAssessment{
		Code:     finding.Code,
		Mapping:  mapping,
		Severity: severity,
	}
}

func LocalMaturity(spec Spec, findings []Finding) LocalResult {
	capabilities := CapabilityMaturity(spec, findings)
	if len(capabilities) == 0 {
		return LocalResult{}
	}
	assessed := make([]FindingAssessment, 0, len(findings))
	defaultID := defaultCapabilityFor(mustCapabilities(spec))
	for _, finding := range findings {
		assessed = append(assessed, normalizeFinding(spec, finding, defaultID))
	}
	if len(capabilities) == 1 {
		out := capabilities[0]
		out.Findings = assessed
		return out
	}
	focus := prioritySortedCapabilities(capabilities)[0]
	clean := true
	unknownCount := 0
	var blocking []string
	for _, capability := range capabilities {
		if !capability.Clean {
			clean = false
		}
		unknownCount += capability.UnknownCount
	}
	blocking = append(blocking, focus.BlockingFindingCodes...)
	sort.Strings(blocking)
	return LocalResult{
		CapabilityID:          focus.CapabilityID,
		CapabilityLabel:       focus.CapabilityLabel,
		CapabilityDescription: focus.CapabilityDescription,
		CurrentLevel:          focus.CurrentLevel,
		NextLevel:             focus.NextLevel,
		CurrentSummary:        focus.CurrentSummary,
		NextUnlock:            focus.NextUnlock,
		Levels:                append([]Level(nil), focus.Levels...),
		BlockingFindingCodes:  blocking,
		Findings:              assessed,
		Clean:                 clean,
		UnknownCount:          unknownCount,
		PriorityRank:          focus.PriorityRank,
		PriorityReason:        focus.PriorityReason,
	}
}

func CapabilityMaturity(spec Spec, findings []Finding) []LocalResult {
	capabilities := mustCapabilities(spec)
	defaultID := defaultCapabilityFor(capabilities)
	byCapability := make(map[string][]FindingAssessment, len(capabilities))
	for _, finding := range findings {
		item := normalizeFinding(spec, finding, defaultID)
		byCapability[item.Mapping.CapabilityID] = append(byCapability[item.Mapping.CapabilityID], item)
	}
	results := make([]LocalResult, 0, len(capabilities))
	for _, capability := range capabilities {
		result := capabilityMaturity(capability, byCapability[capability.ID])
		results = append(results, result)
	}
	for rank, result := range prioritySortedCapabilities(results) {
		for i := range results {
			if results[i].CapabilityID == result.CapabilityID {
				results[i].PriorityRank = rank + 1
				results[i].PriorityReason = priorityReason(result)
				break
			}
		}
	}
	return results
}

func capabilityMaturity(capability CapabilitySpec, assessed []FindingAssessment) LocalResult {
	levelIndex := make(map[string]int, len(capability.Levels))
	for i, level := range capability.Levels {
		levelIndex[level.ID] = i
	}
	lowestBlocked := len(capability.Levels)
	clean := true
	unknownCount := 0
	for _, item := range assessed {
		switch normalizeCleanRequirement(item.Mapping.CleanRequirement) {
		case CleanRequirementRequired:
			clean = false
		case CleanRequirementUncheckable:
			unknownCount++
		}
		if idx, ok := levelIndex[item.Mapping.LocalLevelImpact]; ok && idx < lowestBlocked && blocksLocalMaturity(item) {
			lowestBlocked = idx
		}
	}
	var blocking []string
	if lowestBlocked < len(capability.Levels) {
		blockedID := capability.Levels[lowestBlocked].ID
		for _, item := range assessed {
			if item.Mapping.LocalLevelImpact == blockedID && blocksLocalMaturity(item) {
				blocking = append(blocking, item.Code)
			}
		}
	}
	sort.Strings(blocking)
	currentIdx := len(capability.Levels) - 1
	if lowestBlocked < len(capability.Levels) {
		currentIdx = lowestBlocked - 1
		if currentIdx < 0 {
			currentIdx = 0
		}
	}
	current := ""
	currentSummary := ""
	if currentIdx >= 0 {
		currentLevel := capability.Levels[currentIdx]
		current = currentLevel.ID
		currentSummary = currentLevel.CapabilitySummary
	}
	next := ""
	nextUnlock := ""
	if currentIdx+1 >= 0 && currentIdx+1 < len(capability.Levels) {
		nextLevel := capability.Levels[currentIdx+1]
		next = nextLevel.ID
		nextUnlock = nextLevel.NextUnlock
	}
	return LocalResult{
		CapabilityID:          capability.ID,
		CapabilityLabel:       capability.Label,
		CapabilityDescription: capability.Description,
		CurrentLevel:          current,
		NextLevel:             next,
		CurrentSummary:        currentSummary,
		NextUnlock:            nextUnlock,
		Levels:                append([]Level(nil), capability.Levels...),
		BlockingFindingCodes:  blocking,
		Findings:              assessed,
		Clean:                 clean,
		UnknownCount:          unknownCount,
	}
}

func mustCapabilities(spec Spec) []CapabilitySpec {
	capabilities, err := normalizedCapabilities(spec)
	if err != nil {
		return nil
	}
	return capabilities
}

func currentLevelIndex(result LocalResult) int {
	for i, level := range result.Levels {
		if level.ID == result.CurrentLevel {
			return i
		}
	}
	return -1
}

type capabilityPriorityScore struct {
	requiredState int
	levelIndex    int
	debt          int
	severity      [4]int
	impact        [7]int
	fixable       int
	declaration   int
}

func prioritySortedCapabilities(results []LocalResult) []LocalResult {
	out := append([]LocalResult(nil), results...)
	declaration := make(map[string]int, len(results))
	for i, result := range results {
		declaration[result.CapabilityID] = i
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := scoreCapabilityPriority(out[i], declaration[out[i].CapabilityID])
		right := scoreCapabilityPriority(out[j], declaration[out[j].CapabilityID])
		return compareCapabilityPriority(left, right) < 0
	})
	return out
}

func scoreCapabilityPriority(result LocalResult, declaration int) capabilityPriorityScore {
	score := capabilityPriorityScore{
		requiredState: 2,
		levelIndex:    currentLevelIndex(result),
		declaration:   declaration,
	}
	if score.levelIndex < 0 {
		score.levelIndex = len(result.Levels)
	}
	if !result.Clean || len(result.BlockingFindingCodes) > 0 {
		score.requiredState = 0
	} else if len(result.Findings) > 0 || result.UnknownCount > 0 {
		score.requiredState = 1
	}
	for _, finding := range result.Findings {
		if isDebtFinding(finding) {
			score.debt++
		}
		score.severity[prioritySeverityIndex(finding.Severity)]++
		score.impact[priorityImpactIndex(finding.Mapping.GlobalImpact)]++
		switch finding.Mapping.EffectiveFixClass() {
		case FixClassAuto, FixClassExternal:
			score.fixable++
		}
	}
	return score
}

func compareCapabilityPriority(left, right capabilityPriorityScore) int {
	if left.requiredState != right.requiredState {
		return left.requiredState - right.requiredState
	}
	if left.levelIndex != right.levelIndex {
		return left.levelIndex - right.levelIndex
	}
	if left.debt != right.debt {
		return right.debt - left.debt
	}
	for i := range left.severity {
		if left.severity[i] != right.severity[i] {
			return right.severity[i] - left.severity[i]
		}
	}
	for i := range left.impact {
		if left.impact[i] != right.impact[i] {
			return right.impact[i] - left.impact[i]
		}
	}
	if left.fixable != right.fixable {
		return right.fixable - left.fixable
	}
	return left.declaration - right.declaration
}

func priorityReason(result LocalResult) string {
	state := "clean capability"
	if !result.Clean || len(result.BlockingFindingCodes) > 0 {
		state = "required/blocking findings"
	} else if len(result.Findings) > 0 {
		state = "advisory findings"
	} else if result.UnknownCount > 0 {
		state = "unknown findings"
	}
	severity, severityCount := highestPrioritySeverity(result.Findings)
	impact, impactCount := highestPriorityImpact(result.Findings)
	parts := []string{"lowest current level"}
	if state != "clean capability" {
		parts = append(parts, "with "+state)
	}
	if severityCount > 0 {
		parts = append(parts, prioritySeverityLabel(severity))
	}
	if impactCount > 0 {
		parts = append(parts, string(impact))
	}
	if debt := DebtScore(result.Findings); debt > 0 {
		parts = append(parts, "debt="+fmt.Sprint(debt))
	}
	if fixable := fixableFindingCount(result.Findings); fixable > 0 {
		parts = append(parts, "fixable="+fmt.Sprint(fixable))
	}
	return strings.Join(parts, " ")
}

func prioritySeverityLabel(severity architecturev1.FindingSeverity) string {
	switch severity {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return "BLOCKER"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return "ERROR"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return "WARNING"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "INFO"
	default:
		return "UNSPECIFIED"
	}
}

func highestPrioritySeverity(findings []FindingAssessment) (architecturev1.FindingSeverity, int) {
	for _, severity := range []architecturev1.FindingSeverity{
		architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER,
		architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
	} {
		count := 0
		for _, finding := range findings {
			if finding.Severity == severity {
				count++
			}
		}
		if count > 0 {
			return severity, count
		}
	}
	return architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED, 0
}

func highestPriorityImpact(findings []FindingAssessment) (GlobalImpact, int) {
	for _, impact := range []GlobalImpact{
		ImpactFoundationBlocker,
		ImpactSafetyBlocker,
		ImpactEvolvabilityGap,
		ImpactHardeningGap,
		ImpactCapabilityGap,
		ImpactAdvisory,
		ImpactUnknown,
	} {
		count := 0
		for _, finding := range findings {
			if finding.Mapping.GlobalImpact == impact {
				count++
			}
		}
		if count > 0 {
			return impact, count
		}
	}
	return ImpactUnknown, 0
}

func fixableFindingCount(findings []FindingAssessment) int {
	count := 0
	for _, finding := range findings {
		switch finding.Mapping.EffectiveFixClass() {
		case FixClassAuto, FixClassExternal:
			count++
		}
	}
	return count
}

func prioritySeverityIndex(severity architecturev1.FindingSeverity) int {
	switch severity {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return 0
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return 1
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return 2
	default:
		return 3
	}
}

func priorityImpactIndex(impact GlobalImpact) int {
	switch impact {
	case ImpactFoundationBlocker:
		return 0
	case ImpactSafetyBlocker:
		return 1
	case ImpactEvolvabilityGap:
		return 2
	case ImpactHardeningGap:
		return 3
	case ImpactCapabilityGap:
		return 4
	case ImpactAdvisory:
		return 5
	default:
		return 6
	}
}

func DebtByLevel(findings []FindingAssessment) map[string]DebtCounts {
	out := make(map[string]DebtCounts)
	for _, item := range findings {
		if !isDebtFinding(item) {
			continue
		}
		level := strings.TrimSpace(item.Mapping.LocalLevelImpact)
		if level == "" {
			level = "unknown"
		}
		counts := out[level]
		if counts.BySeverity == nil {
			counts.BySeverity = make(map[architecturev1.FindingSeverity]int)
		}
		counts.Total++
		counts.BySeverity[item.Severity]++
		out[level] = counts
	}
	return out
}

func DebtScore(findings []FindingAssessment) int {
	score := 0
	for _, counts := range DebtByLevel(findings) {
		score += counts.Total
	}
	return score
}

func AssessmentDebtByLevel(a *commonv1.MaturityAssessment) map[string]DebtCounts {
	if a == nil {
		return nil
	}
	findings := make([]FindingAssessment, 0, len(a.GetFindings()))
	for _, finding := range a.GetFindings() {
		if finding == nil {
			continue
		}
		maturity := finding.GetMaturity()
		mapping := FindingMapping{}
		if maturity != nil {
			mapping.LocalLevelImpact = maturity.GetLocalLevel()
			mapping.GlobalImpact = ProtoToGlobalImpact(maturity.GetGlobalImpact())
			mapping.Dimension = maturity.GetDimension()
			mapping.CleanRequirement = string(ProtoToCleanRequirement(maturity.GetCleanRequirement()))
			mapping.RecommendedSkillIDs = append([]string(nil), maturity.GetRecommendedSkillIds()...)
		}
		findings = append(findings, FindingAssessment{
			Code:     finding.GetCode(),
			Mapping:  mapping,
			Severity: normalizeSeverity(finding.GetSeverity()),
		})
	}
	return DebtByLevel(findings)
}

func AssessmentDebtScore(a *commonv1.MaturityAssessment) int {
	score := 0
	for _, counts := range AssessmentDebtByLevel(a) {
		score += counts.Total
	}
	return score
}

func blocksLocalMaturity(item FindingAssessment) bool {
	if normalizeCleanRequirement(item.Mapping.CleanRequirement) == CleanRequirementUncheckable {
		return false
	}
	return item.Severity == architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR ||
		item.Severity == architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER ||
		normalizeCleanRequirement(item.Mapping.CleanRequirement) == CleanRequirementRequired
}

func isDebtFinding(item FindingAssessment) bool {
	if normalizeCleanRequirement(item.Mapping.CleanRequirement) == CleanRequirementUncheckable {
		return false
	}
	switch item.Severity {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED:
		return true
	default:
		return false
	}
}

func normalizeCleanRequirement(raw string) CleanRequirement {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(CleanRequirementRequired):
		return CleanRequirementRequired
	case string(CleanRequirementUncheckable):
		return CleanRequirementUncheckable
	case string(CleanRequirementAdvisory), "":
		return CleanRequirementAdvisory
	default:
		return CleanRequirementAdvisory
	}
}

func normalizeSeverity(raw string) architecturev1.FindingSeverity {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "BLOCKER", "FINDING_SEVERITY_BLOCKER", "SEVERITY_BLOCKER":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
	case "ERROR", "FINDING_SEVERITY_ERROR", "SEVERITY_ERROR":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR
	case "WARNING", "WARN", "FINDING_SEVERITY_WARNING", "SEVERITY_WARNING":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING
	case "INFO", "FINDING_SEVERITY_INFO", "SEVERITY_INFO":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO
	default:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED
	}
}

func ProtoToGlobalImpact(impact commonv1.GlobalImpact) GlobalImpact {
	switch impact {
	case commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER:
		return ImpactFoundationBlocker
	case commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER:
		return ImpactSafetyBlocker
	case commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP:
		return ImpactEvolvabilityGap
	case commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP:
		return ImpactHardeningGap
	case commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP:
		return ImpactCapabilityGap
	case commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY:
		return ImpactAdvisory
	case commonv1.GlobalImpact_GLOBAL_IMPACT_UNKNOWN:
		return ImpactUnknown
	default:
		return ImpactUnknown
	}
}

func ProtoToCleanRequirement(requirement commonv1.CleanRequirement) CleanRequirement {
	switch requirement {
	case commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED:
		return CleanRequirementRequired
	case commonv1.CleanRequirement_CLEAN_REQUIREMENT_UNCHECKABLE:
		return CleanRequirementUncheckable
	case commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY,
		commonv1.CleanRequirement_CLEAN_REQUIREMENT_UNSPECIFIED:
		return CleanRequirementAdvisory
	default:
		return CleanRequirementAdvisory
	}
}

func sourceForAssessmentFinding(finding *commonv1.AssessmentFinding, fallback architecturev1.FindingSource) architecturev1.FindingSource {
	if finding == nil || finding.GetMaturity() == nil {
		return fallback
	}
	switch dimensions.Dimension(strings.TrimSpace(finding.GetMaturity().GetDimension())) {
	case "contracts":
		return architecturev1.FindingSource_FINDING_SOURCE_CLI
	case "ui":
		return architecturev1.FindingSource_FINDING_SOURCE_UI
	case "docs":
		return architecturev1.FindingSource_FINDING_SOURCE_DOCS
	case "standards":
		return architecturev1.FindingSource_FINDING_SOURCE_STANDARDS
	case "cycles":
		return architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE
	case "tidiness":
		return architecturev1.FindingSource_FINDING_SOURCE_TIDINESS
	case "coverage":
		return architecturev1.FindingSource_FINDING_SOURCE_COVERAGE
	case "security":
		return architecturev1.FindingSource_FINDING_SOURCE_SECURITY
	case "measures":
		return architecturev1.FindingSource_FINDING_SOURCE_MEASURES
	case "business":
		return architecturev1.FindingSource_FINDING_SOURCE_BUSINESS
	case "proto-health":
		return architecturev1.FindingSource_FINDING_SOURCE_PROTO
	case "dependency-accuracy":
		return architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY
	case "structure":
		return architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE
	case "storage":
		return architecturev1.FindingSource_FINDING_SOURCE_STORAGE
	default:
		return fallback
	}
}

func assessmentFindingMessage(finding *commonv1.AssessmentFinding) string {
	title := strings.TrimSpace(finding.GetTitle())
	message := strings.TrimSpace(finding.GetMessage())
	switch {
	case title == "":
		return message
	case message == "":
		return title
	default:
		return title + ": " + message
	}
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func defaultEffortForSource(source architecturev1.FindingSource) architecturev1.EffortHint {
	switch source {
	case architecturev1.FindingSource_FINDING_SOURCE_DOCS:
		return architecturev1.EffortHint_EFFORT_HINT_TRIVIAL
	case architecturev1.FindingSource_FINDING_SOURCE_CLI,
		architecturev1.FindingSource_FINDING_SOURCE_UI,
		architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY:
		return architecturev1.EffortHint_EFFORT_HINT_SMALL
	case architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		architecturev1.FindingSource_FINDING_SOURCE_SECURITY:
		return architecturev1.EffortHint_EFFORT_HINT_MEDIUM
	case architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE:
		return architecturev1.EffortHint_EFFORT_HINT_LARGE
	default:
		return architecturev1.EffortHint_EFFORT_HINT_UNSPECIFIED
	}
}

func dims(ids ...string) []dimensions.Dimension {
	out := make([]dimensions.Dimension, 0, len(ids))
	for _, id := range ids {
		out = append(out, dimensions.Dimension(id))
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
