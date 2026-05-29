// Package couplingsmell is the boundary-health detector. It runs the
// domain-level coupling analysis (efferent/afferent coupling, instability,
// fan-out, stable-kernel detection) over the snapshot + derived domain map
// and emits one advisory conflict per smell.
//
// It is the detector form of the boundary-health score view; the same
// boundaries.Analyze powers `arch-cart signals boundaries`.
package couplingsmell

import (
	"context"
	"fmt"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/signals/boundaries"
)

// Detector is the production coupling-smell detector.
type Detector struct {
	cfg boundaries.Config
}

// New returns the production detector using default thresholds.
func New() *Detector { return &Detector{cfg: boundaries.DefaultConfig()} }

// NewWithConfig returns a detector with custom thresholds (control surface).
func NewWithConfig(cfg boundaries.Config) *Detector { return &Detector{cfg: cfg} }

func (Detector) Name() string { return "coupling_smell" }
func (Detector) Description() string {
	return "Scores domain boundary health (coupling, instability, fan-out) and flags god-domains and unstable dependencies. Advisory."
}

func (Detector) EmitsTypes() []string { return []string{"coupling_smell"} }

// Detect analyzes coupling and emits a conflict per smell. Healthy domains
// produce nothing.
func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	rep := boundaries.Analyze(in.Scenario, in.Snapshot, in.DomainMap, d.cfg)
	var out []conflicts.Conflict
	for _, dc := range rep.Domains {
		for _, smell := range dc.Smells {
			out = append(out, conflicts.Conflict{
				Scenario:  in.Scenario,
				Detector:  d.Name(),
				Type:      "coupling_smell",
				Subtype:   smell.Kind,
				Severity:  severityFor(smell.Severity),
				Locations: []string{dc.Domain},
				Domains:   []string{dc.Domain},
				Evidence: []conflicts.Evidence{
					{
						Kind:    smell.Kind,
						Summary: smell.Message,
						Locator: dc.Domain,
					},
					{
						Kind:    "coupling_metrics",
						Summary: fmt.Sprintf("Ce=%d Ca=%d instability=%.2f fan_out=%.2f health=%.2f", dc.Efferent, dc.Afferent, dc.Instability, dc.FanOut, dc.HealthScore),
						Locator: dc.Domain,
					},
				},
				SuggestedFixes: suggestedFixesFor(smell, dc),
			})
		}
	}
	return out, nil
}

// suggestedFixesFor renders 1–2 deterministic fix options per smell kind.
// Templates name the literal dependency edges or invariants involved so an
// operator can act without re-running the analysis.
func suggestedFixesFor(smell boundaries.Smell, dc boundaries.DomainCoupling) []conflicts.Fix {
	switch smell.Kind {
	case boundaries.SmellGodDomain:
		deps := joinUpTo(dc.DependsOn, 5)
		return []conflicts.Fix{
			{
				Kind:       conflicts.FixKindBreakCycle,
				Summary:    "Split " + dc.Domain + " into smaller domains, or push composition into a thinner composition-root archetype (currently depends on: " + deps + ").",
				Confidence: 0.5,
			},
			{
				Kind:       conflicts.FixKindAddDependency,
				Summary:    "If " + dc.Domain + " legitimately wires many domains together, mark its archetype as exempt in the coupling config.",
				Confidence: 0.5,
			},
		}
	case boundaries.SmellUnstableDependency:
		deps := joinUpTo(dc.DependsOn, 5)
		dependedBy := joinUpTo(dc.DependedBy, 5)
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindAddDependency,
			Summary:    "Reduce " + dc.Domain + "'s efferent coupling by injecting dependencies via interfaces (depends on: " + deps + "; depended on by: " + dependedBy + ").",
			Confidence: 0.5,
		}}
	default:
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindAddDependency,
			Summary:    "Review " + dc.Domain + " boundaries: " + smell.Message,
			Confidence: 0.5,
		}}
	}
}

func joinUpTo(items []string, n int) string {
	if len(items) == 0 {
		return "—"
	}
	if len(items) <= n {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:n], ", ") + fmt.Sprintf(" (+%d more)", len(items)-n)
}

func severityFor(s boundaries.Severity) conflicts.Severity {
	switch s {
	case boundaries.SeverityWarn:
		return conflicts.SeverityWarn
	default:
		return conflicts.SeverityInfo
	}
}
