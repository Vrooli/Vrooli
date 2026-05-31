// Package findings is the controller's DIAGNOSE stage: it runs a test-genie
// audit of a target and parses the result into the dimension-bucketed findings
// vector that selection and termination operate on.
//
// The parse boundary is deliberately EM-local (its own structs) rather than
// importing test-genie's internal SuiteExecutionResult: EM consumes test-genie
// only across the `execute --json` contract, so the shape it depends on is
// pinned here and exercised by a committed fixture (interoperability boundary).
package findings

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Audit is the projection of test-genie's `execute --json` output that the
// controller depends on. Fields test-genie may add are ignored.
type Audit struct {
	ScenarioName  string       `json:"scenarioName"`
	Verdict       string       `json:"verdict"`
	PresetUsed    string       `json:"preset"`
	PlannedPhases []string     `json:"plannedPhases"`
	Phases        []AuditPhase `json:"phases"`
}

// AuditPhase is one executed phase and its findings.
type AuditPhase struct {
	Name     string         `json:"name"`
	Status   string         `json:"status"`
	Findings []AuditFinding `json:"findings"`
}

// AuditFinding mirrors the fields of architecture.v1.ArchitectureFinding that
// the controller reads. Source/Severity arrive as their proto enum integers.
type AuditFinding struct {
	Source    int32    `json:"source"`
	Severity  int32    `json:"severity"`
	Code      string   `json:"code"`
	StableID  string   `json:"stable_id"`
	Locations []string `json:"locations"`
	Message   string   `json:"message"`
}

// ParseAudit decodes test-genie's `execute --json` stdout into an Audit.
func ParseAudit(raw []byte) (*Audit, error) {
	var a Audit
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("parse test-genie audit json: %w", err)
	}
	return &a, nil
}

// phaseFailed reports whether a phase status counts as an open problem. A
// findingless failing phase still contributes a synthetic finding so dimensions
// without structured-finding producers (tests, performance, …) are visible.
func phaseFailed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "fail", "failed", "error", "errored":
		return true
	default: // pass, skip, skipped, partial, "" → not an open finding
		return false
	}
}

// ToFindings flattens an Audit into the controller's finding list. A structured
// finding's dimension is resolved by its FindingSource, falling back to the
// producing phase when the source is unspecified/unmapped. A failing phase that
// emitted no structured findings yields one synthetic finding in the phase's
// dimension.
func ToFindings(a *Audit) []Finding {
	if a == nil {
		return nil
	}
	out := make([]Finding, 0)
	for _, ph := range a.Phases {
		if len(ph.Findings) > 0 {
			for _, raw := range ph.Findings {
				dim, ok := dimensions.ForSource(architecturev1.FindingSource(raw.Source))
				if !ok {
					dim, ok = dimensions.ForPhase(ph.Name)
				}
				if !ok {
					// Neither source nor phase maps; skip rather than invent a
					// dimension. The P1 anti-drift guard exists to prevent this.
					continue
				}
				out = append(out, Finding{
					ID:        findingID(raw, ph.Name),
					Dimension: dim,
					Severity:  architecturev1.FindingSeverity(raw.Severity),
					Location:  firstLocation(raw.Locations),
					Code:      raw.Code,
					Message:   raw.Message,
					Phase:     ph.Name,
				})
			}
			continue
		}
		if phaseFailed(ph.Status) {
			dim, ok := dimensions.ForPhase(ph.Name)
			if !ok {
				continue
			}
			out = append(out, Finding{
				ID:        "phase:" + ph.Name,
				Dimension: dim,
				Severity:  architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
				Code:      "PHASE_FAILED",
				Message:   "phase " + ph.Name + " failed without structured findings",
				Phase:     ph.Name,
				Synthetic: true,
			})
		}
	}
	return out
}

func firstLocation(locs []string) string {
	for _, l := range locs {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// findingID prefers test-genie's deterministic stable id; when absent it derives
// a stable key from phase+code+location so fingerprints stay reproducible.
func findingID(raw AuditFinding, phase string) string {
	if id := strings.TrimSpace(raw.StableID); id != "" {
		return id
	}
	return strings.Join([]string{"derived", phase, raw.Code, firstLocation(raw.Locations)}, ":")
}
