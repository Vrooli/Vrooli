// Package assessment validates provider-owned health maturity specs and
// normalizes finding maturity metadata into the global maturity vocabulary.
package assessment

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type GlobalImpact string

const (
	ImpactFoundationBlocker GlobalImpact = "foundation_blocker"
	ImpactSafetyBlocker     GlobalImpact = "safety_blocker"
	ImpactEvolvabilityGap   GlobalImpact = "evolvability_gap"
	ImpactHardeningGap      GlobalImpact = "hardening_gap"
	ImpactCapabilityGap     GlobalImpact = "capability_gap"
	ImpactAdvisory          GlobalImpact = "advisory"
	ImpactUnknown           GlobalImpact = "unknown"
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

var impactDimensions = map[GlobalImpact][]dimensions.Dimension{
	ImpactFoundationBlocker: dims("tests", "standards", "structure"),
	ImpactSafetyBlocker:     dims("security", "dependencies"),
	ImpactEvolvabilityGap:   dims("structure", "cycles", "contracts", "docs", "proto-health", "dependency-accuracy"),
	ImpactHardeningGap:      dims("tests", "coverage", "tidiness", "ui", "visual", "performance"),
	ImpactCapabilityGap:     dims("operational-targets", "business", "measures"),
	ImpactAdvisory:          nil,
	ImpactUnknown:           nil,
}

// Spec is the `.vrooli/maturity.json` schema shared by health scenarios.
type Spec struct {
	Provider string                    `json:"provider"`
	Phase    string                    `json:"phase"`
	Version  string                    `json:"version"`
	Levels   []Level                   `json:"levels"`
	Findings map[string]FindingMapping `json:"findings"`
	Fallback FallbackPolicy            `json:"fallback"`
}

type Level struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	EntryCriteria []string `json:"entry_criteria"`
	ExitCriteria  []string `json:"exit_criteria"`
}

type FindingMapping struct {
	LocalLevelImpact    string       `json:"local_level_impact"`
	GlobalImpact        GlobalImpact `json:"global_impact"`
	Dimension           string       `json:"dimension"`
	SeverityDefault     string       `json:"severity_default"`
	RecommendedSkillIDs []string     `json:"recommended_skill_ids"`
}

type FallbackPolicy struct {
	LocalLevelImpact string       `json:"local_level_impact"`
	GlobalImpact     GlobalImpact `json:"global_impact"`
	Dimension        string       `json:"dimension"`
	SeverityDefault  string       `json:"severity_default"`
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
}

type FindingAssessment struct {
	Code     string
	Mapping  FindingMapping
	Severity architecturev1.FindingSeverity
}

type LocalResult struct {
	CurrentLevel         string
	NextLevel            string
	BlockingFindingCodes []string
	Findings             []FindingAssessment
}

type BuildInput struct {
	Scenario string
	Spec     Spec
	Findings []Finding
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
	if len(spec.Levels) == 0 {
		return fmt.Errorf("at least one level is required")
	}
	levels := make(map[string]int, len(spec.Levels))
	for i, level := range spec.Levels {
		id := strings.TrimSpace(level.ID)
		if id == "" {
			return fmt.Errorf("levels[%d].id is required", i)
		}
		if _, exists := levels[id]; exists {
			return fmt.Errorf("duplicate level id %q", id)
		}
		levels[id] = i
	}
	for code, mapping := range spec.Findings {
		if strings.TrimSpace(code) == "" {
			return fmt.Errorf("finding code cannot be empty")
		}
		if err := validateMapping(mapping, levels, "findings."+code); err != nil {
			return err
		}
	}
	return validateMapping(FindingMapping{
		LocalLevelImpact: spec.Fallback.LocalLevelImpact,
		GlobalImpact:     spec.Fallback.GlobalImpact,
		Dimension:        spec.Fallback.Dimension,
		SeverityDefault:  spec.Fallback.SeverityDefault,
	}, levels, "fallback")
}

func validateMapping(mapping FindingMapping, levels map[string]int, path string) error {
	if strings.TrimSpace(mapping.LocalLevelImpact) != "" {
		if _, ok := levels[mapping.LocalLevelImpact]; !ok {
			return fmt.Errorf("%s.local_level_impact %q is not a declared level", path, mapping.LocalLevelImpact)
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
	return nil
}

func IsValidImpact(impact GlobalImpact) bool {
	_, ok := validImpacts[impact]
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
	out := &commonv1.MaturityAssessment{
		Scenario:               strings.TrimSpace(input.Scenario),
		Provider:               input.Spec.Provider,
		Phase:                  input.Spec.Phase,
		Version:                input.Spec.Version,
		Local:                  buildProtoLocal(input.Spec, local),
		Findings:               make([]*commonv1.AssessmentFinding, 0, len(input.Findings)),
		FindingsByGlobalImpact: map[string]int32{},
		FindingsBySeverity:     map[string]int32{},
	}
	skills := map[string]struct{}{}
	for i, finding := range input.Findings {
		assessed := local.Findings[i]
		severity := assessed.Severity.String()
		impact := string(assessed.Mapping.GlobalImpact)
		out.FindingsByGlobalImpact[impact]++
		out.FindingsBySeverity[severity]++
		for _, skill := range assessed.Mapping.RecommendedSkillIDs {
			skill = strings.TrimSpace(skill)
			if skill != "" {
				skills[skill] = struct{}{}
			}
		}
		out.Findings = append(out.Findings, &commonv1.AssessmentFinding{
			Code:        finding.Code,
			Severity:    severity,
			Title:       finding.Title,
			Message:     finding.Message,
			Location:    finding.Location,
			Remediation: finding.Remediation,
			Maturity: &commonv1.FindingMaturity{
				LocalLevel:          assessed.Mapping.LocalLevelImpact,
				GlobalImpact:        GlobalImpactToProto(assessed.Mapping.GlobalImpact),
				Dimension:           assessed.Mapping.Dimension,
				RecommendedSkillIds: append([]string(nil), assessed.Mapping.RecommendedSkillIDs...),
			},
		})
	}
	out.RecommendedSkillIds = sortedKeys(skills)
	if err := ValidateAssessment(out); err != nil {
		return nil, err
	}
	return out, nil
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
	}
	return nil
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
	levels := make([]*commonv1.LocalMaturityLevel, 0, len(spec.Levels))
	for _, level := range spec.Levels {
		levels = append(levels, &commonv1.LocalMaturityLevel{
			Id:            level.ID,
			Name:          level.Name,
			Description:   level.Description,
			EntryCriteria: append([]string(nil), level.EntryCriteria...),
			ExitCriteria:  append([]string(nil), level.ExitCriteria...),
		})
	}
	return &commonv1.LocalMaturityAssessment{
		CurrentLevel:         local.CurrentLevel,
		NextLevel:            local.NextLevel,
		Levels:               levels,
		BlockingFindingCodes: append([]string(nil), local.BlockingFindingCodes...),
	}
}

func NormalizeFinding(spec Spec, finding Finding) FindingAssessment {
	mapping := finding.Maturity
	if !finding.HasMaturity {
		if m, ok := spec.Findings[finding.Code]; ok {
			mapping = m
		} else {
			mapping = FindingMapping{
				LocalLevelImpact: spec.Fallback.LocalLevelImpact,
				GlobalImpact:     spec.Fallback.GlobalImpact,
				Dimension:        spec.Fallback.Dimension,
				SeverityDefault:  spec.Fallback.SeverityDefault,
			}
		}
	}
	if strings.TrimSpace(mapping.Dimension) == "" {
		if dim, ok := dimensions.ForSource(finding.Source); ok {
			mapping.Dimension = string(dim)
		} else if dim, ok := dimensions.ForPhase(finding.Phase); ok {
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
	levelIndex := make(map[string]int, len(spec.Levels))
	for i, level := range spec.Levels {
		levelIndex[level.ID] = i
	}
	lowestBlocked := len(spec.Levels)
	var blocking []string
	assessed := make([]FindingAssessment, 0, len(findings))
	for _, finding := range findings {
		item := NormalizeFinding(spec, finding)
		assessed = append(assessed, item)
		if idx, ok := levelIndex[item.Mapping.LocalLevelImpact]; ok && idx < lowestBlocked && blocksLocalMaturity(item) {
			lowestBlocked = idx
		}
	}
	if lowestBlocked < len(spec.Levels) {
		blockedID := spec.Levels[lowestBlocked].ID
		for _, item := range assessed {
			if item.Mapping.LocalLevelImpact == blockedID && blocksLocalMaturity(item) {
				blocking = append(blocking, item.Code)
			}
		}
	}
	sort.Strings(blocking)
	currentIdx := len(spec.Levels) - 1
	if lowestBlocked < len(spec.Levels) {
		currentIdx = lowestBlocked - 1
	}
	current := ""
	if currentIdx >= 0 {
		current = spec.Levels[currentIdx].ID
	}
	next := ""
	if currentIdx+1 >= 0 && currentIdx+1 < len(spec.Levels) {
		next = spec.Levels[currentIdx+1].ID
	}
	return LocalResult{
		CurrentLevel:         current,
		NextLevel:            next,
		BlockingFindingCodes: blocking,
		Findings:             assessed,
	}
}

func blocksLocalMaturity(item FindingAssessment) bool {
	if item.Mapping.GlobalImpact == ImpactAdvisory {
		return false
	}
	return item.Severity == architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR ||
		item.Severity == architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
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
