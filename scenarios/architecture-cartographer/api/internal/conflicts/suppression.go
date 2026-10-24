package conflicts

import (
	"strings"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/suppressions"
)

// applySuppressions marks each conflict that an active marker sanctions as
// Suppressed (with the marker's reason). Suppressed conflicts are reported,
// not dropped. Matching: the marker's id must equal the conflict's detector,
// type, or subtype, AND the marker must be location-relevant — its file lies
// under one of the conflict's locations, or the file's domain is one of the
// conflict's domains (or appears in its locations).
func applySuppressions(conflicts []Conflict, markers []suppressions.Marker, m domains.DerivedDomainMap) []Conflict {
	if len(markers) == 0 {
		return conflicts
	}
	for i := range conflicts {
		for _, mk := range markers {
			if !idMatches(mk.ID, conflicts[i]) {
				continue
			}
			if !locationRelevant(mk, conflicts[i], m) {
				continue
			}
			conflicts[i].Suppressed = true
			conflicts[i].SuppressionReason = mk.Reason
			break
		}
	}
	return conflicts
}

func idMatches(id string, c Conflict) bool {
	return id == c.Detector || id == c.Type || id == c.Subtype
}

func locationRelevant(mk suppressions.Marker, c Conflict, m domains.DerivedDomainMap) bool {
	markerDomain := m.DomainFor(mk.File)
	for _, loc := range c.Locations {
		if loc == mk.File || strings.HasPrefix(mk.File, strings.TrimSuffix(loc, "/")+"/") {
			return true
		}
		if markerDomain != "" && loc == markerDomain {
			return true
		}
	}
	if markerDomain != "" {
		for _, d := range c.Domains {
			if d == markerDomain {
				return true
			}
		}
	}
	return false
}
