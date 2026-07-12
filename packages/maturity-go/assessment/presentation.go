package assessment

import (
	"fmt"
	"sort"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
)

// PhasePresentationContractVersion is the first portable maturity
// presentation contract. It is deliberately independent of a provider's
// assessment version: the assessment remains the semantic source while this
// version names the rendering projection.
const PhasePresentationContractVersion = "v1"

// BuildPhasePresentation derives the canonical, provider-owned presentation
// from an assessment. It never mutates the assessment and is deliberately
// deterministic when providers return findings or capabilities in a different
// raw order. Full evidence remains available through assessment.findings.
func BuildPhasePresentation(a *commonv1.MaturityAssessment) *commonv1.PhasePresentation {
	if a == nil || a.GetLocal() == nil {
		return nil
	}
	local := a.GetLocal()
	p := &commonv1.PhasePresentation{
		ContractVersion:      PhasePresentationContractVersion,
		Provider:             a.GetProvider(),
		Phase:                a.GetPhase(),
		CurrentLevel:         local.GetCurrentLevel(),
		NextLevel:            local.GetNextLevel(),
		Clean:                local.GetClean(),
		UnknownCount:         local.GetUnknownCount(),
		BlockingFindingCodes: sortedStrings(local.GetBlockingFindingCodes()),
		AtMaximum:            local.GetClean() && strings.TrimSpace(local.GetNextLevel()) == "",
	}
	if level := levelByID(local.GetLevels(), local.GetCurrentLevel()); level != nil {
		p.CurrentLevelLabel = firstNonBlank(level.GetStatusLabel(), level.GetName())
	}
	if levels := local.GetLevels(); len(levels) > 0 {
		top := levels[len(levels)-1]
		p.CeilingLevel = top.GetId()
		p.NorthStar = firstNonBlank(top.GetCapabilitySummary(), top.GetName())
	}

	capabilities := orderedCapabilities(a.GetCapabilities())
	if len(capabilities) == 0 {
		capabilities = []*commonv1.CapabilityMaturityAssessment{{
			Id:                   "local",
			Label:                "Local Maturity",
			CurrentLevel:         local.GetCurrentLevel(),
			NextLevel:            local.GetNextLevel(),
			Levels:               local.GetLevels(),
			BlockingFindingCodes: local.GetBlockingFindingCodes(),
			Clean:                local.GetClean(),
			UnknownCount:         local.GetUnknownCount(),
			PriorityRank:         1,
		}}
	}
	for _, capability := range capabilities {
		p.Capabilities = append(p.Capabilities, buildCapabilityPresentation(capability, a.GetFindings()))
	}

	if focus := a.GetHighestPriorityCapability(); focus != nil && strings.TrimSpace(focus.GetCapabilityId()) != "" {
		p.FocusCapabilityId = focus.GetCapabilityId()
		p.FocusCapabilityLabel = focus.GetCapabilityLabel()
		p.NextActionReason = focus.GetReason()
	}
	if focus := presentationCapabilityByID(p.GetCapabilities(), p.GetFocusCapabilityId()); focus != nil {
		p.NextAction = focus.GetNextUnlock()
	}
	if strings.TrimSpace(p.GetNextAction()) == "" {
		if level := levelByID(local.GetLevels(), local.GetCurrentLevel()); level != nil {
			p.NextAction = level.GetNextUnlock()
		}
	}
	if !p.GetAtMaximum() && strings.TrimSpace(p.GetPhase()) != "" {
		p.DocumentationTopics = documentationTopics(p.GetPhase(), p.GetBlockingFindingCodes())
	}
	return p
}

// ValidatePhasePresentation verifies that a provider returned the canonical
// projection for its assessment. Comparing against a freshly derived view makes
// ordering, grouping, focus, and fix affordances non-negotiable without making
// a second implementation authoritative.
func ValidatePhasePresentation(a *commonv1.MaturityAssessment) error {
	if err := ValidateAssessment(a); err != nil {
		return err
	}
	got := a.GetPresentation()
	if got == nil {
		return fmt.Errorf("assessment.presentation is required")
	}
	if got.GetContractVersion() != PhasePresentationContractVersion {
		return fmt.Errorf("assessment.presentation.contract_version %q is unsupported", got.GetContractVersion())
	}
	want := BuildPhasePresentation(a)
	if !proto.Equal(got, want) {
		return fmt.Errorf("assessment.presentation is not the canonical projection")
	}
	return nil
}

func buildCapabilityPresentation(capability *commonv1.CapabilityMaturityAssessment, findings []*commonv1.AssessmentFinding) *commonv1.PhaseCapabilityPresentation {
	if capability == nil {
		return nil
	}
	p := &commonv1.PhaseCapabilityPresentation{
		Id:                   capability.GetId(),
		Label:                capability.GetLabel(),
		CurrentLevel:         capability.GetCurrentLevel(),
		NextLevel:            capability.GetNextLevel(),
		CurrentSummary:       capability.GetCurrentSummary(),
		NextUnlock:           capability.GetNextUnlock(),
		Clean:                capability.GetClean(),
		UnknownCount:         capability.GetUnknownCount(),
		BlockingFindingCodes: sortedStrings(capability.GetBlockingFindingCodes()),
		PriorityRank:         capability.GetPriorityRank(),
		PriorityReason:       capability.GetPriorityReason(),
	}
	if level := levelByID(capability.GetLevels(), capability.GetCurrentLevel()); level != nil {
		p.CurrentLevelLabel = firstNonBlank(level.GetStatusLabel(), level.GetName())
	}
	p.Findings = rollupFindings(capability.GetId(), findings)
	return p
}

func orderedCapabilities(in []*commonv1.CapabilityMaturityAssessment) []*commonv1.CapabilityMaturityAssessment {
	out := make([]*commonv1.CapabilityMaturityAssessment, 0, len(in))
	for _, capability := range in {
		if capability != nil {
			out = append(out, capability)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, right := out[i].GetPriorityRank(), out[j].GetPriorityRank()
		if left != right {
			if left == 0 {
				return false
			}
			if right == 0 {
				return true
			}
			return left < right
		}
		return out[i].GetId() < out[j].GetId()
	})
	return out
}

func rollupFindings(capabilityID string, findings []*commonv1.AssessmentFinding) []*commonv1.PhasePresentationFinding {
	type rollup struct {
		finding *commonv1.PhasePresentationFinding
	}
	byCode := map[string]*rollup{}
	for _, finding := range findings {
		if finding == nil || finding.GetMaturity().GetCapabilityId() != capabilityID {
			continue
		}
		code := strings.TrimSpace(finding.GetCode())
		if code == "" {
			continue
		}
		entry := byCode[code]
		if entry == nil {
			entry = &rollup{finding: &commonv1.PhasePresentationFinding{
				Code:          code,
				Severity:      finding.GetSeverity(),
				Title:         finding.GetTitle(),
				Message:       finding.GetMessage(),
				Remediation:   finding.GetRemediation(),
				FixAffordance: fixAffordance(finding),
			}}
			byCode[code] = entry
		}
		entry.finding.Count++
		if location := strings.TrimSpace(finding.GetLocation()); location != "" {
			entry.finding.Locations = append(entry.finding.Locations, location)
		}
	}
	codes := make([]string, 0, len(byCode))
	for code := range byCode {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	out := make([]*commonv1.PhasePresentationFinding, 0, len(codes))
	for _, code := range codes {
		entry := byCode[code].finding
		entry.Locations = sortedStrings(entry.GetLocations())
		out = append(out, entry)
	}
	return out
}

func fixAffordance(finding *commonv1.AssessmentFinding) commonv1.FixAffordance {
	if finding.GetAutofixAvailable() {
		return commonv1.FixAffordance_FIX_AFFORDANCE_PREVIEW_AVAILABLE
	}
	if strings.EqualFold(strings.TrimSpace(finding.GetFixClass()), string(FixClassManual)) {
		return commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL
	}
	return commonv1.FixAffordance_FIX_AFFORDANCE_DETECTION_ONLY
}

func documentationTopics(phase string, blockingCodes []string) []string {
	topics := []string{phase + " maturity next move"}
	for _, code := range blockingCodes {
		if code = strings.TrimSpace(code); code != "" {
			topics = append(topics, phase+" "+code+" canonical fix")
		}
		if len(topics) == 3 {
			break
		}
	}
	return topics
}

func presentationCapabilityByID(capabilities []*commonv1.PhaseCapabilityPresentation, id string) *commonv1.PhaseCapabilityPresentation {
	for _, capability := range capabilities {
		if capability != nil && capability.GetId() == id {
			return capability
		}
	}
	return nil
}

func levelByID(levels []*commonv1.LocalMaturityLevel, id string) *commonv1.LocalMaturityLevel {
	for _, level := range levels {
		if level != nil && level.GetId() == id {
			return level
		}
	}
	return nil
}

func sortedStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
