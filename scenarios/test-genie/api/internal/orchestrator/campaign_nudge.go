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
// threshold, not only the architecture phase — a docs/standards/cli/ui load
// is just as worth tracking. Both thresholds are configurable via env; the
// nudge fires when EITHER is exceeded. Always logged when it fires — never
// silent. Zero-finding runs naturally no-op (total below threshold).
const (
	defaultCampaignSevereThreshold = 5  // count(BLOCKER|ERROR) ≥ this
	defaultCampaignTotalThreshold  = 15 // count(all findings)   > this

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
// and returns a nudge when the load exceeds either threshold. It returns nil
// when the load is below both thresholds (so a clean or light run never
// nudges). The trigger does NOT depend on which phases ran — any battery
// that crosses the threshold is worth tracking as a campaign.
func computeCampaignNudge(scenario string, phaseResults []phases.ExecutionResult) *CampaignNudge {
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

	severeThreshold := campaignThreshold(envCampaignSevereThreshold, defaultCampaignSevereThreshold)
	totalThreshold := campaignThreshold(envCampaignTotalThreshold, defaultCampaignTotalThreshold)

	if severe < severeThreshold && total <= totalThreshold {
		return nil
	}

	return &CampaignNudge{
		Triggered:  true,
		Total:      total,
		Severe:     severe,
		BySeverity: bySeverity,
		Reason: fmt.Sprintf(
			"%d findings (%d blocker/error) exceed the single-pass threshold (severe≥%d or total>%d). Track this as a campaign instead of fixing ad-hoc.",
			total, severe, severeThreshold, totalThreshold),
		Command: fmt.Sprintf(
			"architecture-cartographer campaign create %s --from-audit <audit-report.json>",
			scenario),
	}
}
