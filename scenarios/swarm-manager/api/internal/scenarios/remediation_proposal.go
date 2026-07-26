package scenarios

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// RemediationProposal is a pure, source-neutral preview. Construction has no
// storage side effects; application is an explicit later boundary.
type RemediationProposal struct {
	Target               RemediationTarget
	Fingerprint          string
	Provenance           string
	Title                string
	Description          string
	AcceptanceCriteria   []string
	AcceptanceAllow      []string
	RecommendedWorkflows []string
}

// MaturityCampaignProposal represents broad, explicitly confirmed work. It is
// deliberately separate from a phase item: it records the selected outcome
// and declared workflow without claiming optional tracker availability.
type MaturityCampaignProposal struct {
	Scenario            string
	Target              string
	Fingerprint         string
	Title               string
	Description         string
	AcceptanceCriteria  []string
	DeclaredWorkflow    string
	TrackerAvailability string
	TrackerRef          string
}

type MaturityCampaignTarget struct {
	Scenario       string
	Target         string
	ProviderPhases []string
}

func BuildMaturityCampaignProposal(snapshot ScenarioHealthSnapshot, scenario, target string) (MaturityCampaignProposal, error) {
	return BuildMaturityCampaignProposalForTarget(snapshot, MaturityCampaignTarget{Scenario: scenario, Target: target})
}

func BuildMaturityCampaignProposalForTarget(snapshot ScenarioHealthSnapshot, target MaturityCampaignTarget) (MaturityCampaignProposal, error) {
	if !snapshot.IsActionable() {
		return MaturityCampaignProposal{}, fmt.Errorf("fresh Test Genie evidence is required for a maturity campaign")
	}
	target.Scenario, target.Target = strings.TrimSpace(target.Scenario), strings.TrimSpace(target.Target)
	if target.Scenario == "" || target.Target == "" {
		return MaturityCampaignProposal{}, fmt.Errorf("scenario and selected maturity target are required")
	}
	phases := make([]string, 0, len(target.ProviderPhases))
	for _, phase := range target.ProviderPhases {
		phase = strings.TrimSpace(phase)
		if phase == "" {
			continue
		}
		if _, ok := findHealthPhase(snapshot.Phases, phase); !ok {
			return MaturityCampaignProposal{}, fmt.Errorf("provider phase %q is absent from current evidence", phase)
		}
		phases = append(phases, strings.ToLower(phase))
	}
	if len(phases) == 0 {
		return MaturityCampaignProposal{}, fmt.Errorf("at least one provider phase is required for a maturity campaign")
	}
	sort.Strings(phases)
	digest := sha256.Sum256([]byte("scenario-campaign/v1\x00" + strings.ToLower(target.Scenario) + "\x00" + strings.ToLower(target.Target) + "\x00" + strings.Join(phases, "\x00")))
	return MaturityCampaignProposal{Scenario: target.Scenario, Target: target.Target, Fingerprint: "smc:" + hex.EncodeToString(digest[:]), Title: fmt.Sprintf("[%s] Reach maturity target: %s", target.Scenario, target.Target), Description: "Reach the operator-selected maturity outcome across the selected provider phases using current Test Genie evidence. This is a broad campaign, not a single phase remediation.", AcceptanceCriteria: []string{fmt.Sprintf("Given fresh comparable Test Genie evidence for %s, when the maturity target %q is evaluated across %s, then provider-owned phase standing demonstrates the selected outcome.", target.Scenario, target.Target, strings.Join(phases, ", "))}, DeclaredWorkflow: "scenario-improvement-campaign", TrackerAvailability: "unavailable until Architecture Cartographer confirms a campaign reference"}, nil
}

func BuildPhaseRemediationProposal(snapshot ScenarioHealthSnapshot, target RemediationTarget, provenance string) (RemediationProposal, error) {
	if !snapshot.IsActionable() {
		return RemediationProposal{}, fmt.Errorf("fresh Test Genie evidence is required for phase remediation")
	}
	normalized, err := target.Normalize()
	if err != nil {
		return RemediationProposal{}, err
	}
	fingerprint, err := normalized.Fingerprint()
	if err != nil {
		return RemediationProposal{}, err
	}
	phase, ok := findHealthPhase(snapshot.Phases, normalized.ProviderPhase)
	if !ok {
		return RemediationProposal{}, fmt.Errorf("provider phase %q is absent from current evidence", normalized.ProviderPhase)
	}
	if strings.TrimSpace(phase.PriorityCapabilityID) != normalized.CapabilityID {
		return RemediationProposal{}, fmt.Errorf("capability target %q is not the provider priority for phase %q", normalized.CapabilityID, normalized.ProviderPhase)
	}
	label := firstNonEmpty(phase.PriorityCapabilityLabel, phase.PriorityCapabilityID)
	title := fmt.Sprintf("[%s] Improve %s in %s", normalized.Scenario, label, phase.Phase)
	description := fmt.Sprintf("Improve the provider-defined %s capability for the %s phase. Preserve Test Genie as the evidence authority; validate the outcome against fresh comparable evidence.", label, phase.Phase)
	criteria := []string{fmt.Sprintf("Given fresh Test Genie evidence for %s, when the %s phase is evaluated, then the provider reports progress from %s toward %s for %s.", normalized.Scenario, phase.Phase, firstNonEmpty(phase.CurrentRung, "the current rung"), firstNonEmpty(phase.NextRung, "the next provider rung"), label)}
	return RemediationProposal{Target: normalized, Fingerprint: fingerprint, Provenance: strings.TrimSpace(provenance), Title: title, Description: description, AcceptanceCriteria: criteria, AcceptanceAllow: []string{"scenarios/" + normalized.Scenario + "/**"}, RecommendedWorkflows: []string{"scenario-improvement-campaign"}}, nil
}

func findHealthPhase(phases []ScenarioHealthPhase, name string) (ScenarioHealthPhase, bool) {
	for _, phase := range phases {
		if strings.EqualFold(strings.TrimSpace(phase.Phase), strings.TrimSpace(name)) {
			return phase, true
		}
	}
	return ScenarioHealthPhase{}, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
