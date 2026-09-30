package scenarioruntime

import (
	"path/filepath"
	"strings"
)

// PathOverlap says how two claimed paths relate. Claims are advisory in this
// release: the launcher prints the holder and continues; refusal is a later
// flag on the same comparison.
type PathOverlap string

const (
	// OverlapNone means the paths are independent.
	OverlapNone PathOverlap = ""
	// OverlapExact means the same path.
	OverlapExact PathOverlap = "exact"
	// OverlapExistingContainsNew means the existing claim is a parent of the new one.
	OverlapExistingContainsNew PathOverlap = "existing_contains_new"
	// OverlapNewContainsExisting means the new claim is a parent of the existing one.
	OverlapNewContainsExisting PathOverlap = "new_contains_existing"
)

// CheckPathOverlap compares two claimed paths after cleaning them. It is the
// comparison workspace-sandbox uses for sandbox scopes, lifted here so editor
// claims and sandbox scopes answer "do these overlap" the same way.
func CheckPathOverlap(existing, proposed string) PathOverlap {
	existing = filepath.Clean(existing)
	proposed = filepath.Clean(proposed)
	switch {
	case existing == proposed:
		return OverlapExact
	case strings.HasPrefix(proposed, existing+string(filepath.Separator)):
		return OverlapExistingContainsNew
	case strings.HasPrefix(existing, proposed+string(filepath.Separator)):
		return OverlapNewContainsExisting
	}
	return OverlapNone
}

// ClaimOverlaps lists the active leases whose claims overlap any of the
// proposed paths, so a launcher can name the holder before it continues.
func ClaimOverlaps(leases []EditorLease, proposed []string) []ClaimOverlap {
	var out []ClaimOverlap
	for _, lease := range leases {
		for _, held := range lease.Claims {
			for _, path := range proposed {
				if overlap := CheckPathOverlap(held, path); overlap != OverlapNone {
					out = append(out, ClaimOverlap{Holder: lease, HeldPath: held, ProposedPath: path, Overlap: overlap})
				}
			}
		}
	}
	return out
}

// ClaimOverlap names one overlapping claim and its holder.
type ClaimOverlap struct {
	Holder       EditorLease
	HeldPath     string
	ProposedPath string
	Overlap      PathOverlap
}
