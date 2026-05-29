package orchestrator

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"test-genie/internal/orchestrator/phases"
)

// Migration-nudge defaults. When the architecture-audit battery surfaces
// more findings than one pass can responsibly fix, the suite output steers
// the agent to open a TRACKED migration in architecture-cartographer rather
// than fixing ad-hoc (the workflow that failed on swarm-manager because the
// surface area was too large to track by hand).
//
// Both thresholds are configurable via env; the nudge fires when EITHER is
// exceeded. Always logged when it fires — never silent.
const (
	defaultMigrationSevereThreshold = 5  // count(BLOCKER|ERROR) ≥ this
	defaultMigrationTotalThreshold  = 15 // count(all findings)   > this

	envMigrationSevereThreshold = "TESTGENIE_ARCH_MIGRATION_THRESHOLD_SEVERE"
	envMigrationTotalThreshold  = "TESTGENIE_ARCH_MIGRATION_THRESHOLD_TOTAL"
)

// MigrationNudge is the structured steer appended to a suite result when
// the architectural finding load exceeds the single-pass threshold.
type MigrationNudge struct {
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
	// Command is the literal command to open the tracked migration.
	Command string `json:"command"`
}

// migrationThreshold reads an int env override, falling back to def when
// unset or invalid (≤0 is rejected so a misconfig can't disable the nudge
// silently).
func migrationThreshold(env string, def int) int {
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

// architecturePhaseRan reports whether the cohesion axis was actually
// measured this run. The nudge only fires when the architecture phase ran
// (i.e. the architecture-audit preset or an explicit selection), so the
// existing presets never surface a surprise nudge.
func architecturePhaseRan(phaseResults []phases.ExecutionResult) bool {
	for _, p := range phaseResults {
		if p.Name == phases.Architecture.String() {
			return true
		}
	}
	return false
}

// computeMigrationNudge aggregates the normalized findings across all
// phases and returns a nudge when the load exceeds either threshold. It
// returns nil when the architecture phase did not run or the load is below
// both thresholds.
func computeMigrationNudge(scenario string, phaseResults []phases.ExecutionResult) *MigrationNudge {
	if !architecturePhaseRan(phaseResults) {
		return nil
	}

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

	severeThreshold := migrationThreshold(envMigrationSevereThreshold, defaultMigrationSevereThreshold)
	totalThreshold := migrationThreshold(envMigrationTotalThreshold, defaultMigrationTotalThreshold)

	if severe < severeThreshold && total <= totalThreshold {
		return nil
	}

	return &MigrationNudge{
		Triggered:  true,
		Total:      total,
		Severe:     severe,
		BySeverity: bySeverity,
		Reason: fmt.Sprintf(
			"%d architectural findings (%d blocker/error) exceed the single-pass threshold (severe≥%d or total>%d). Track this refactor instead of fixing ad-hoc.",
			total, severe, severeThreshold, totalThreshold),
		Command: fmt.Sprintf(
			"architecture-cartographer migration create %s --from-audit <audit-report.json>",
			scenario),
	}
}
