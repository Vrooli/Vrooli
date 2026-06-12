package phases

import (
	"strings"

	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// normalizeSeverity maps the heterogeneous severity vocabularies of the
// per-surface validators into the shared FindingSeverity ladder. This is
// the ONE normalization table; every producer routes through it so
// severity semantics stay consistent across CLI/UI/docs/standards/
// structure/architecture. (R3 in the plan: drift across four producers is
// guarded by a table-driven test over every enum value.)
//
// Mapping:
//   - blocker                          → BLOCKER (e.g. cartographer cycles)
//   - error / failure / critical / high → ERROR
//   - warn / warning / medium           → WARNING
//   - info / notice / low               → INFO
//   - anything else / empty             → UNSPECIFIED
//
// Inputs may be bare ("error") or proto-style ("SEVERITY_ERROR"); both
// normalize identically.
func normalizeFindingSeverity(raw string) architecturev1.FindingSeverity {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "SEVERITY_")
	switch s {
	case "BLOCKER":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
	case "ERROR", "FAILURE", "CRITICAL", "HIGH":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR
	case "WARN", "WARNING", "MEDIUM":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING
	case "INFO", "NOTICE", "LOW":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO
	default:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED
	}
}

// nonEmptyLocations drops blank locations and trims each. findingid does
// its own normalization for the hash; this keeps the displayed/stored
// Locations tidy.
func nonEmptyLocations(locs ...string) []string {
	out := make([]string, 0, len(locs))
	for _, l := range locs {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// defaultEffortForSource is the documented per-source effort heuristic.
// Effort is advisory ranking input (the campaign tracker's FAST/LONG_TERM
// profiles consume it) and is EXCLUDED from the stable-ID hash, so a coarse
// per-source default is a safe starting point — every phase maps to exactly
// one source, so this is equivalently a per-phase default. When a validator
// begins emitting a real per-finding estimate, route it through
// newFindingWithEffort instead of this table.
//
//	DOCS         → TRIVIAL  (broken link / missing manifest entry)
//	CLI / UI     → SMALL    (one contract/slot binding gap)
//	STANDARDS    → SMALL    (one rule violation, usually one file)
//	TIDINESS     → SMALL    (file/function quality nit)
//	PROTO        → SMALL    (one proto contract organization or sync gap)
//	COVERAGE     → MEDIUM   (write tests for an under-covered target)
//	SECURITY     → MEDIUM   (rotate a secret / bump a vulnerable dep)
//	STRUCTURE    → LARGE    (package mislocation — a structural move)
//	ARCHITECTURE → LARGE    (import cycle / coupling — structural)
//	(anything else) → UNSPECIFIED
func defaultEffortForSource(source architecturev1.FindingSource) architecturev1.EffortHint {
	switch source {
	case architecturev1.FindingSource_FINDING_SOURCE_DOCS:
		return architecturev1.EffortHint_EFFORT_HINT_TRIVIAL
	case architecturev1.FindingSource_FINDING_SOURCE_CLI,
		architecturev1.FindingSource_FINDING_SOURCE_UI,
		architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		architecturev1.FindingSource_FINDING_SOURCE_PROTO:
		return architecturev1.EffortHint_EFFORT_HINT_SMALL
	case architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		architecturev1.FindingSource_FINDING_SOURCE_SECURITY:
		return architecturev1.EffortHint_EFFORT_HINT_MEDIUM
	case architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE:
		return architecturev1.EffortHint_EFFORT_HINT_LARGE
	default:
		return architecturev1.EffortHint_EFFORT_HINT_UNSPECIFIED
	}
}

// newFinding builds a normalized ArchitectureFinding for one surface and
// stamps its deterministic `afid:` stable ID. Producers (the delegating
// phases) call this so every finding carries a consistent source,
// severity, effort, and stable ID without re-implementing the contract.
// Effort defaults to the per-source heuristic (defaultEffortForSource).
func newFinding(
	scenario string,
	source architecturev1.FindingSource,
	code, severity, message, suggestion string,
	locations, domains []string,
) *architecturev1.ArchitectureFinding {
	return newFindingWithEffort(scenario, source, code, severity, message, suggestion,
		locations, domains, defaultEffortForSource(source))
}

// newFindingWithEffort is newFinding with an explicit effort estimate, for
// producers whose validator supplies one rather than relying on the
// per-source default.
func newFindingWithEffort(
	scenario string,
	source architecturev1.FindingSource,
	code, severity, message, suggestion string,
	locations, domains []string,
	effort architecturev1.EffortHint,
) *architecturev1.ArchitectureFinding {
	f := &architecturev1.ArchitectureFinding{
		Scenario:   scenario,
		Source:     source,
		Code:       strings.TrimSpace(code),
		Severity:   normalizeFindingSeverity(severity),
		Locations:  locations,
		Domains:    domains,
		Message:    strings.TrimSpace(message),
		Suggestion: strings.TrimSpace(suggestion),
		Effort:     effort,
	}
	findingid.Stamp(f)
	return f
}
