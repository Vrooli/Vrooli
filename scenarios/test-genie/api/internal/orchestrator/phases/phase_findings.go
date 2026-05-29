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

// newFinding builds a normalized ArchitectureFinding for one surface and
// stamps its deterministic `afid:` stable ID. Producers (the delegating
// phases) call this so every finding carries a consistent source,
// severity, and stable ID without re-implementing the contract.
func newFinding(
	scenario string,
	source architecturev1.FindingSource,
	code, severity, message, suggestion string,
	locations, domains []string,
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
	}
	findingid.Stamp(f)
	return f
}
