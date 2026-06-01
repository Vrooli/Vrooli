// Package validation is security-health's producer core. It detects which
// substrates a target scenario actually contains (Go, pnpm UI, …), runs the
// applicable security scanners (secrets, Go SAST, Go vuln-DB, JS deps), and
// returns a normalized Finding list whose severities gate the
// ecosystem-manager R1 ladder rung.
//
// The package is intentionally flat: detector, the Scanner interface, the
// per-tool runners, and the orchestrating Service all live here so the
// severity-normalization contract (the load-bearing invariant) is defined
// once and shared without an import cycle.
package validation

import "strings"

// Severity is the normalized severity of a Finding. The normalization from
// each scanner's native vocabulary is the load-bearing contract of this
// scenario: only SeverityError causes passed=false, and only SeverityError
// gates the ecosystem-manager R1 ("Safe") ladder rung.
type Severity int

const (
	// SeverityUnspecified is the zero value; never emitted by a scanner.
	SeverityUnspecified Severity = iota
	// SeverityError maps from critical/high. Causes passed=false and gates R1.
	SeverityError
	// SeverityWarning maps from moderate/medium. Advisory only.
	SeverityWarning
	// SeverityInfo maps from low/info/unknown and degraded observations.
	SeverityInfo
)

// String renders the severity as the lowercase token the CLI --json surface
// emits and test-genie parses back.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unspecified"
	}
}

// NormalizeSeverity maps a scanner's native severity word onto the normalized
// scale. This is the single place the contract lives:
//
//	critical, high            -> ERROR   (gates R1)
//	moderate, medium          -> WARNING (advisory)
//	low, info, anything else  -> INFO
//
// The match is case-insensitive and tolerant of surrounding whitespace so
// each scanner's normalizer can pass its raw string straight through.
func NormalizeSeverity(raw string) Severity {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical", "crit", "high":
		return SeverityError
	case "moderate", "medium", "med":
		return SeverityWarning
	default:
		// low, info, informational, unknown, "" — advisory context.
		return SeverityInfo
	}
}

// Finding is a single normalized security result. The field set is the exact
// shape the test-genie `security` phase parses from `validate … --json`; keep
// it in lockstep with the proto Finding message and the CLI JSON encoder.
type Finding struct {
	// RuleID is a stable machine code identifying the rule/check
	// (e.g. "gosec.G101", "gitleaks.generic-api-key", "osv.GO-2024-1234").
	RuleID string
	// Severity is the normalized severity (see NormalizeSeverity).
	Severity Severity
	// Title is a short human-meaningful summary.
	Title string
	// Description explains the issue.
	Description string
	// Remediation is actionable guidance. Never empty — a finding the user
	// can't act on is noise.
	Remediation string
	// FilePath anchors the finding to a file (optionally file:line) relative
	// to the scenario root. Secret findings carry file:line only, never the
	// raw value.
	FilePath string
	// Scanner names the tool that produced the finding ("gitleaks", "gosec",
	// "govulncheck", "pnpm-audit", "osv-scanner"). Lets consumers group by tool.
	Scanner string
}

// Summary is a rollup of Finding counts by normalized severity.
type Summary struct {
	Errors   int
	Warnings int
	Infos    int
}

// Report is the full result of validating one scenario.
type Report struct {
	Scenario string
	Passed   bool
	Findings []Finding
	Summary  Summary
	// SkippedScanners lists scanners that applied to a detected substrate but
	// whose binary was not installed. Surfaced as degraded context, never a
	// failure.
	SkippedScanners []string
}

// finalize sorts nothing (callers control ordering) but computes the summary
// and the passed flag from the findings. passed is true iff no SeverityError
// finding is present.
func finalize(scenario string, findings []Finding, skipped []string) Report {
	if findings == nil {
		findings = []Finding{}
	}
	if skipped == nil {
		skipped = []string{}
	}
	var sum Summary
	passed := true
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			sum.Errors++
			passed = false
		case SeverityWarning:
			sum.Warnings++
		default:
			sum.Infos++
		}
	}
	return Report{
		Scenario:        scenario,
		Passed:          passed,
		Findings:        findings,
		Summary:         sum,
		SkippedScanners: skipped,
	}
}
