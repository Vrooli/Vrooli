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
				Status: conflicts.ResolutionStatusDetected,
			})
		}
	}
	return out, nil
}

func severityFor(s boundaries.Severity) conflicts.Severity {
	switch s {
	case boundaries.SeverityWarn:
		return conflicts.SeverityWarn
	default:
		return conflicts.SeverityInfo
	}
}
