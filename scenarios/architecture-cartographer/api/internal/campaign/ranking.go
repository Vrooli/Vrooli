package campaign

import (
	"sort"
	"strings"

	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
)

// ranking.go is the profile-strategy seam: the ONE place that decides how an
// open worklist is ordered. Detection (test-genie) and reconciliation
// (service.go) are profile-blind; only this file knows about FAST / BALANCED
// / LONG_TERM. Adding a profile is a new case here plus a proto enum value —
// nothing else in the engine changes.

// severityRank orders severities for the worklist (higher = more urgent).
func severityRank(sev string) int {
	switch sev {
	case "blocker":
		return 4
	case "error":
		return 3
	case "warn":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// effortRank orders effort tokens cheapest-first (lower = cheaper). Unknown
// effort sorts as medium so an un-estimated item never jumps ahead of a
// known-trivial one in the FAST profile.
func effortRank(effort string) int {
	switch effort {
	case EffortTrivial:
		return 1
	case EffortSmall:
		return 2
	case EffortMedium:
		return 3
	case EffortLarge:
		return 4
	default: // unspecified / unknown
		return 3
	}
}

// isCycle is the legacy structural-blocker predicate used by BALANCED to
// preserve the historical "cycles block dependent moves — fix first" order.
func isCycle(f Finding) bool { return strings.HasPrefix(f.Code, "cycle") }

// gatesSuite reports whether a finding's source can fail a scenario's test
// suite. Architecture (structural cohesion) and tidiness findings are
// ADVISORY — they never hard-fail the suite — so the FAST profile sinks them
// below gating findings. Every other surface (standards, structure, cli, ui,
// docs) gates.
func gatesSuite(source string) bool {
	switch source {
	case "architecture", "tidiness":
		return false
	default:
		return true
	}
}

// structuralRootCause reports whether a finding is a structural root cause —
// the kind of defect whose fix removes whole classes of downstream symptoms
// (import cycles, package mislocation, structural-cohesion findings). The
// LONG_TERM profile elevates these so root causes are fixed before symptoms.
func structuralRootCause(f Finding) bool {
	switch f.Source {
	case "architecture", "structure":
		return true
	}
	return strings.HasPrefix(f.Code, "cycle") || strings.HasPrefix(f.Code, "mislocat")
}

// normalizeProfile collapses the zero value (UNSPECIFIED) to BALANCED.
func normalizeProfile(p campaignv1.RankProfile) campaignv1.RankProfile {
	if p == campaignv1.RankProfile_RANK_PROFILE_UNSPECIFIED {
		return campaignv1.RankProfile_RANK_PROFILE_BALANCED
	}
	return p
}

// Order sorts the open worklist in place under the given profile and returns
// it. Ordering is stable so equal-rank items keep their incoming order.
func Order(findings []Finding, profile campaignv1.RankProfile) []Finding {
	switch normalizeProfile(profile) {
	case campaignv1.RankProfile_RANK_PROFILE_FAST:
		orderFast(findings)
	case campaignv1.RankProfile_RANK_PROFILE_LONG_TERM:
		orderLongTerm(findings)
	default:
		orderBalanced(findings)
	}
	return findings
}

// orderBalanced preserves the historical sortWorklist order: regressions
// first (the work broke something), then import cycles (they block dependent
// moves), then severity desc, then code, then stable id for determinism.
func orderBalanced(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Regressed != b.Regressed {
			return a.Regressed
		}
		if ac, bc := isCycle(a), isCycle(b); ac != bc {
			return ac
		}
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra > rb
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.StableID < b.StableID
	})
}

// orderFast is "make it work now": gating sources first (advisory
// architecture/tidiness sink to the bottom), then severity desc, then
// cheapest effort, then fewest locations — the shortest path to a green
// suite.
func orderFast(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if ga, gb := gatesSuite(a.Source), gatesSuite(b.Source); ga != gb {
			return ga
		}
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra > rb
		}
		if ea, eb := effortRank(a.Effort), effortRank(b.Effort); ea != eb {
			return ea < eb
		}
		if la, lb := len(a.Locations), len(b.Locations); la != lb {
			return la < lb
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.StableID < b.StableID
	})
}

// orderLongTerm is "best solution": regressions first, then structural root
// causes (fix the cause before the symptom), then severity desc. Effort is
// NOT a deprioritizer — a large, high-value structural fix stays at the top.
func orderLongTerm(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Regressed != b.Regressed {
			return a.Regressed
		}
		if sa, sb := structuralRootCause(a), structuralRootCause(b); sa != sb {
			return sa
		}
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra > rb
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.StableID < b.StableID
	})
}
