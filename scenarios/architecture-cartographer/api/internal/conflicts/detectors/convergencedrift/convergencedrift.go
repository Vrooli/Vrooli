// Package convergencedrift is the cross-surface convergence detector. It
// asks the domains domain whether the scenario's surfaces (DOMAINS.md, the
// api/internal folders, the cli groups, the ui features) agree on the same
// domain set, and emits one conflict per disagreement.
//
// Unlike cycle/mislocated_file, this detector reads no graph edges — it
// operates purely on the DerivedDomainMap's per-source declarations. It is
// the detector form of `arch-cart domains convergence`.
package convergencedrift

import (
	"context"
	"fmt"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
)

// Detector is the production convergence-drift detector.
type Detector struct{}

// New returns the production detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "convergence_drift" }
func (Detector) Description() string {
	return "Flags domains where the scenario's surfaces (DOMAINS.md, api folders, cli groups, ui features) disagree on the domain set."
}

func (Detector) EmitsTypes() []string { return []string{"convergence_drift"} }

// Detect maps each domains.ConvergenceFinding to a Conflict. Advisory
// (info) findings become info-severity conflicts; real disagreements
// (warn) become warn-severity. No suggested fixes are emitted — the
// resolution is editorial (reconcile the declaring surfaces).
func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	findings := domains.Convergence(in.DomainMap)
	if len(findings) == 0 {
		return nil, nil
	}
	out := make([]conflicts.Conflict, 0, len(findings))
	for _, f := range findings {
		domains := domainsForFinding(f)
		locator := strings.Join(domains, ",")
		if locator == "" {
			locator = "—"
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "convergence_drift",
			Subtype:   f.Kind,
			Severity:  severityFor(f.Severity),
			Locations: domains,
			Domains:   domains,
			Evidence: []conflicts.Evidence{{
				Kind:    f.Kind,
				Summary: f.Message,
				Locator: fmt.Sprintf("%s [%s]", locator, joinSources(f.Sources)),
			}},
			Status: conflicts.ResolutionStatusDetected,
		})
	}
	return out, nil
}

// domainsForFinding returns the affected domain names for a finding,
// expanding RolledUpDomains when present. A finding without any domain
// (e.g., authority_fallback) returns nil — callers tolerate empty
// Locations / Domains.
func domainsForFinding(f domains.ConvergenceFinding) []string {
	if len(f.RolledUpDomains) > 0 {
		return append([]string(nil), f.RolledUpDomains...)
	}
	if f.Domain == "" {
		return nil
	}
	return []string{f.Domain}
}

func severityFor(s domains.ConvergenceSeverity) conflicts.Severity {
	switch s {
	case domains.ConvergenceWarn:
		return conflicts.SeverityWarn
	default:
		return conflicts.SeverityInfo
	}
}

func joinSources(sources []domains.Source) string {
	parts := make([]string, 0, len(sources))
	for _, s := range sources {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, " vs ")
}
