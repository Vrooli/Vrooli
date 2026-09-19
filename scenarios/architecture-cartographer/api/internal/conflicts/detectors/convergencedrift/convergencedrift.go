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

func (Detector) Class() conflicts.FindingClass {
	return conflicts.FindingClassDeterministic
}

// Detect maps each domains.ConvergenceFinding to a Conflict. Advisory
// (info) findings become info-severity conflicts; real disagreements
// (warn) become warn-severity. Each finding carries a templated
// SuggestedFix naming the literal edit required to reconcile the
// surfaces (no LLM, no resolver execution — deterministic strings).
func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	findings := domains.Convergence(in.DomainMap)
	if len(findings) == 0 {
		return nil, nil
	}
	out := make([]conflicts.Conflict, 0, len(findings))
	for _, f := range findings {
		affected := domainsForFinding(f)
		locator := strings.Join(affected, ",")
		if locator == "" {
			locator = "—"
		}
		out = append(out, conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "convergence_drift",
			Subtype:   f.Kind,
			Severity:  severityFor(f.Severity),
			Locations: affected,
			Domains:   affected,
			Evidence: []conflicts.Evidence{{
				Kind:    f.Kind,
				Summary: f.Message,
				Locator: fmt.Sprintf("%s [%s]", locator, joinSources(f.Sources)),
			}},
			SuggestedFixes: suggestedFixesFor(f, affected),
		})
	}
	return out, nil
}

// suggestedFixesFor returns 1–2 templated fix options for a convergence
// finding. Templates are deterministic (no LLM); Confidence is fixed at
// 0.5 until usage validates it. Resolver is left empty — these are
// editor-facing suggestions, not auto-apply.
func suggestedFixesFor(f domains.ConvergenceFinding, affected []string) []conflicts.Fix {
	name := strings.Join(affected, ",")
	if name == "" {
		name = "<domain>"
	}
	switch f.Kind {
	case domains.FindingUndeclaredFolder:
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Add a row to docs/concepts/DOMAINS.md for " + name + " (columns: name | purpose | archetype | owns_data | surfaces | requirements | source_paths | glossary) — or remove api/internal/" + name + "/ if it is not a domain.",
			Confidence: 0.5,
		}}
	case domains.FindingMissingImplementation:
		return []conflicts.Fix{
			{
				Kind:       conflicts.FixKindReassignDomain,
				Summary:    "Create api/internal/" + name + "/ implementing the domain declared in docs/concepts/DOMAINS.md.",
				Confidence: 0.5,
			},
			{
				Kind:       conflicts.FixKindReassignDomain,
				Summary:    "Or remove the " + name + " row from docs/concepts/DOMAINS.md if it is no longer planned.",
				Confidence: 0.5,
			},
		}
	case domains.FindingMissingCLIGroup:
		return []conflicts.Fix{
			{
				Kind:       conflicts.FixKindReassignDomain,
				Summary:    "Add a CLI group at cli/domains/" + name + "/ and register it in cli/manifest.json.",
				Confidence: 0.5,
			},
			{
				Kind:       conflicts.FixKindReassignDomain,
				Summary:    "Or document " + name + " as API-only in the DOMAINS.md Surfaces column.",
				Confidence: 0.5,
			},
		}
	case domains.FindingUIFeatureNoDomain:
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Add ui/src/features/" + name + "/ to the closest domain's Source Paths in docs/concepts/DOMAINS.md, or declare a new domain row for " + name + ".",
			Confidence: 0.5,
		}}
	case domains.FindingAuthorityFallback:
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Write docs/concepts/DOMAINS.md (template at docs/concepts/DOMAINS.template.md) to promote the inferred authority to a curated one.",
			Confidence: 0.5,
		}}
	default:
		return []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    "Reconcile the disagreeing surfaces for " + name + " in docs/concepts/DOMAINS.md.",
			Confidence: 0.5,
		}}
	}
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
