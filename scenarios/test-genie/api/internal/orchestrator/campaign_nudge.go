package orchestrator

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"test-genie/internal/orchestrator/phases"
)

// Campaign-nudge defaults. When an audit battery surfaces more findings than
// one pass can responsibly fix, the suite output steers the agent to open a
// TRACKED improvement campaign in architecture-cartographer rather than
// fixing ad-hoc (the workflow that failed on swarm-manager because the
// surface area was too large to track by hand).
//
// The trigger is source-agnostic: it fires on ANY audit that crosses a
// threshold, regardless of descriptor class — a docs/standards/cli/ui load
// is just as worth tracking. Always logged when it fires — never silent.
//
// Two triggers, both env-tunable; the nudge fires when EITHER holds:
//   - SEVERE: count(BLOCKER|ERROR) ≥ THRESHOLD_SEVERE — fires regardless of
//     verdict, because real rot is worth tracking even on an otherwise-green
//     suite.
//   - VOLUME: count(BLOCKER|ERROR|WARN) > THRESHOLD_TOTAL AND the suite did
//     not PASS. INFO findings are advisory by definition and never counted; a
//     green suite is by definition blocking no one, so it never nags on volume
//     alone. This is the fix for the assessment's "near-permanent banner".
const (
	defaultCampaignSevereThreshold = 5  // count(BLOCKER|ERROR)        ≥ this
	defaultCampaignTotalThreshold  = 15 // count(BLOCKER|ERROR|WARN)   > this

	envCampaignSevereThreshold = "TESTGENIE_CAMPAIGN_THRESHOLD_SEVERE"
	envCampaignTotalThreshold  = "TESTGENIE_CAMPAIGN_THRESHOLD_TOTAL"
)

// CampaignNudge is the structured steer appended to a suite result when the
// finding load exceeds the single-pass threshold.
type CampaignNudge struct {
	// Triggered is always true when this struct is present on the result;
	// kept explicit so JSON consumers can branch without a nil check.
	Triggered bool `json:"triggered"`
	// Total is the count of normalized findings across every phase.
	Total int `json:"total"`
	// Severe is the count of BLOCKER/ERROR findings.
	Severe int `json:"severe"`
	// BySeverity is the count per normalized severity (lower-case names).
	BySeverity map[string]int `json:"bySeverity"`
	// Reason explains, in one line, why the nudge fired.
	Reason string `json:"reason"`
	// ArtifactPath is the scenario-relative findings.json that already exists
	// on disk; the Command ingests it via --from-audit.
	ArtifactPath string `json:"artifactPath"`
	// Command is the literal command to open the tracked campaign.
	Command string `json:"command"`
}

// campaignThreshold reads an int env override, falling back to def when
// unset or invalid (≤0 is rejected so a misconfig can't disable the nudge
// silently).
func campaignThreshold(env string, def int) int {
	v := strings.TrimSpace(os.Getenv(env))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// severityName maps the normalized FindingSeverity enum to its lower-case
// wire name (matching the cartographer's by_severity vocabulary).
func severityName(s architecturev1.FindingSeverity) string {
	switch s {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return "blocker"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return "error"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return "warn"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

// computeCampaignNudge aggregates the normalized findings across all phases
// and returns a nudge when either the SEVERE or VOLUME trigger holds (see the
// const block). It returns nil otherwise — so a clean or light run, and a
// green run carrying only advisory warnings, never nudge. verdict is the
// tri-state suite outcome (PASS/PARTIAL/FAIL); artifactPath is the
// scenario-relative findings.json the nudge command points at.
func computeCampaignNudge(scenario, verdict, artifactPath string, phaseResults []phases.ExecutionResult) *CampaignNudge {
	return computeCampaignNudgeFromViews(scenario, verdict, artifactPath, buildPhaseResultViews("", phaseResults))
}

func computeCampaignNudgeFromViews(scenario, verdict, artifactPath string, phaseResults []phaseResultView) *CampaignNudge {
	bySeverity := map[string]int{}
	total := 0
	for _, p := range phaseResults {
		for _, f := range p.Findings {
			if f == nil {
				continue
			}
			total++
			bySeverity[severityName(f.GetSeverity())]++
		}
	}
	severe := bySeverity["blocker"] + bySeverity["error"]
	// Volume counts actionable findings only: blocker+error+warn. INFO is
	// advisory by definition and excluded so an info flood can't nag.
	actionable := severe + bySeverity["warn"]

	severeThreshold := campaignThreshold(envCampaignSevereThreshold, defaultCampaignSevereThreshold)
	totalThreshold := campaignThreshold(envCampaignTotalThreshold, defaultCampaignTotalThreshold)

	severeTrip := severe >= severeThreshold
	// Volume only trips on a non-passing suite: a green run blocks no one.
	volumeTrip := actionable > totalThreshold && verdict != SuiteVerdictPass
	if !severeTrip && !volumeTrip {
		return nil
	}

	var reason string
	switch {
	case severeTrip:
		reason = fmt.Sprintf(
			"%d blocker/error findings (≥%d) — track these as a campaign instead of fixing ad-hoc.",
			severe, severeThreshold)
	default:
		reason = fmt.Sprintf(
			"%d actionable findings (blocker/error/warn) exceed the single-pass threshold (>%d) on a non-passing suite — track this as a campaign instead of fixing ad-hoc.",
			actionable, totalThreshold)
	}

	return &CampaignNudge{
		Triggered:    true,
		Total:        total,
		Severe:       severe,
		BySeverity:   bySeverity,
		Reason:       reason,
		ArtifactPath: artifactPath,
		Command: fmt.Sprintf(
			"architecture-cartographer campaign create %s --from-audit %s",
			scenario, artifactPath),
	}
}
