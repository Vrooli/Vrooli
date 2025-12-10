package security

import "strings"

// PathOverlapReason describes why two paths are considered overlapping or not.
type PathOverlapReason string

const (
	// OverlapReasonExactMatch indicates the paths are identical.
	OverlapReasonExactMatch PathOverlapReason = "exact_match"

	// OverlapReasonAContainsB indicates path A contains path B (A is parent of B).
	OverlapReasonAContainsB PathOverlapReason = "a_contains_b"

	// OverlapReasonBContainsA indicates path B contains path A (B is parent of A).
	OverlapReasonBContainsA PathOverlapReason = "b_contains_a"

	// OverlapReasonNoOverlap indicates the paths do not overlap.
	OverlapReasonNoOverlap PathOverlapReason = "no_overlap"
)

// PathOverlapResult describes the relationship between two paths.
type PathOverlapResult struct {
	Overlaps bool
	Reason   PathOverlapReason
}

// ClassifyPathOverlap determines the relationship between two file paths.
// This is the core decision for "do these paths conflict?"
//
// Decision criteria:
//   - Exact match: "api/handlers" == "api/handlers" → overlaps
//   - Parent-child: "api" contains "api/handlers" → overlaps
//   - Child-parent: "api/handlers" is inside "api" → overlaps
//   - Disjoint: "api/handlers" and "ui/components" → no overlap
func ClassifyPathOverlap(a, b string) PathOverlapResult {
	a = strings.TrimSuffix(a, "/")
	b = strings.TrimSuffix(b, "/")

	// Exact match
	if a == b {
		return PathOverlapResult{
			Overlaps: true,
			Reason:   OverlapReasonExactMatch,
		}
	}

	// Check if A is a parent of B
	if strings.HasPrefix(b, a+"/") {
		return PathOverlapResult{
			Overlaps: true,
			Reason:   OverlapReasonAContainsB,
		}
	}

	// Check if B is a parent of A
	if strings.HasPrefix(a, b+"/") {
		return PathOverlapResult{
			Overlaps: true,
			Reason:   OverlapReasonBContainsA,
		}
	}

	return PathOverlapResult{
		Overlaps: false,
		Reason:   OverlapReasonNoOverlap,
	}
}

// PathsOverlap checks if two paths overlap (one is a prefix of the other or they're equal).
// This is the simple boolean decision wrapper around ClassifyPathOverlap.
func PathsOverlap(a, b string) bool {
	return ClassifyPathOverlap(a, b).Overlaps
}

// ScopeConflictResult describes whether two scopes conflict and why.
type ScopeConflictResult struct {
	HasConflict      bool
	ConflictingPaths []struct{ PathA, PathB string }
	Reason           string
}

// ClassifyScopeConflict determines if two sets of paths have any overlapping entries.
// This is the decision point for "can these agents work simultaneously?"
//
// Decision criteria:
//   - Empty scope = "entire scenario" → always conflicts with anything
//   - Non-empty scopes → conflict if any path in A overlaps with any path in B
func ClassifyScopeConflict(scopeA, scopeB []string) ScopeConflictResult {
	result := ScopeConflictResult{
		HasConflict:      false,
		ConflictingPaths: nil,
	}

	// Empty scope means "entire scenario" which conflicts with everything
	if len(scopeA) == 0 || len(scopeB) == 0 {
		result.HasConflict = true
		result.Reason = "empty scope (entire scenario) conflicts with any other scope"
		return result
	}

	// Check all path pairs for overlap
	for _, pathA := range scopeA {
		for _, pathB := range scopeB {
			if PathsOverlap(pathA, pathB) {
				result.HasConflict = true
				result.ConflictingPaths = append(result.ConflictingPaths, struct{ PathA, PathB string }{pathA, pathB})
			}
		}
	}

	if result.HasConflict {
		result.Reason = "paths overlap between scopes"
	}

	return result
}

// PathSetsOverlap checks if two sets of paths have any overlap.
// Paths overlap if one is a prefix of another or they are identical.
// Empty scope means "entire scenario" and conflicts with everything.
//
// This is the simple boolean wrapper around ClassifyScopeConflict.
func PathSetsOverlap(a, b []string) bool {
	return ClassifyScopeConflict(a, b).HasConflict
}
