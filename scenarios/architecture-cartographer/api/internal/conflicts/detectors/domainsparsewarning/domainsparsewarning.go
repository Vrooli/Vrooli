// Package domainsparsewarning surfaces non-fatal extraction warnings —
// rows the structured DOMAINS.md parser silently skipped, or other
// per-rung anomalies the ladder forwards on DerivedDomainMap.Warnings.
//
// Without this detector, a DOMAINS.md row with the wrong column count
// (a common authoring slip) just disappears: the "10 declared, 3
// derived" mystery the L5-readiness investigation surfaced. With it, the
// operator sees an explicit warn-severity conflict per skipped row.
package domainsparsewarning

import (
	"context"
	"fmt"

	"architecture-cartographer/internal/conflicts"
)

// Detector is the production parse-warning detector.
type Detector struct{}

// New returns the production detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "domains_doc_parse_warning" }
func (Detector) Description() string {
	return "Surfaces silently-skipped rows in structured domain inventories (today: DOMAINS.md)."
}

func (Detector) EmitsTypes() []string { return []string{"domains_doc_parse_warning"} }

// Detect emits one warn-severity conflict per ladder warning. The
// underlying warning's Kind becomes the conflict subtype; its
// path:line becomes the Location; its Summary becomes the evidence.
// Severity is uniformly warn — these never block the audit, but they
// must not be silent.
func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	warns := in.DomainMap.Warnings
	if len(warns) == 0 {
		return nil, nil
	}
	out := make([]conflicts.Conflict, 0, len(warns))
	for _, w := range warns {
		loc := w.Path
		if w.Line > 0 {
			loc = fmt.Sprintf("%s:%d", w.Path, w.Line)
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "domains_doc_parse_warning",
			Subtype:   w.Kind,
			Severity:  conflicts.SeverityWarn,
			Locations: []string{loc},
			Evidence: []conflicts.Evidence{{
				Kind:    w.Kind,
				Summary: w.Summary,
				Locator: loc,
			}},
			Status: conflicts.ResolutionStatusDetected,
		})
	}
	return out, nil
}
