package conflicts

import (
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/suppressions"
)

func domainMap() domains.DerivedDomainMap {
	return domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"api/internal/graph/"}},
			{Name: "conflicts", Paths: []string{"api/internal/conflicts/"}},
		},
	}
}

func TestApplySuppressions_MatchByDetectorAndFile(t *testing.T) {
	conflicts := []Conflict{
		{Detector: "cycle", Type: "cycle", Locations: []string{"api/internal/graph"}},
	}
	markers := []suppressions.Marker{
		{ID: "cycle", Reason: "known legacy hub", File: "api/internal/graph/service.go"},
	}
	out := applySuppressions(conflicts, markers, domainMap())
	if !out[0].Suppressed || out[0].SuppressionReason != "known legacy hub" {
		t.Fatalf("expected suppressed-with-reason, got %+v", out[0])
	}
}

func TestApplySuppressions_MatchBySubtypeAndDomain(t *testing.T) {
	// coupling_smell uses the domain name as its location; match via the
	// marker file's owning domain.
	conflicts := []Conflict{
		{Detector: "coupling_smell", Type: "coupling_smell", Subtype: "god_domain", Domains: []string{"conflicts"}, Locations: []string{"conflicts"}},
	}
	markers := []suppressions.Marker{
		{ID: "god_domain", Reason: "orchestration root by design", File: "api/internal/conflicts/service.go"},
	}
	out := applySuppressions(conflicts, markers, domainMap())
	if !out[0].Suppressed {
		t.Fatalf("expected suppression via subtype+domain, got %+v", out[0])
	}
}

func TestApplySuppressions_NoMatchWrongId(t *testing.T) {
	conflicts := []Conflict{
		{Detector: "cycle", Type: "cycle", Locations: []string{"api/internal/graph"}},
	}
	markers := []suppressions.Marker{
		{ID: "mislocated_file", Reason: "x", File: "api/internal/graph/service.go"},
	}
	out := applySuppressions(conflicts, markers, domainMap())
	if out[0].Suppressed {
		t.Fatal("id mismatch must not suppress")
	}
}

func TestApplySuppressions_NoMatchWrongLocation(t *testing.T) {
	conflicts := []Conflict{
		{Detector: "cycle", Type: "cycle", Locations: []string{"api/internal/graph"}},
	}
	markers := []suppressions.Marker{
		{ID: "cycle", Reason: "x", File: "api/internal/conflicts/service.go"},
	}
	out := applySuppressions(conflicts, markers, domainMap())
	if out[0].Suppressed {
		t.Fatal("location mismatch must not suppress")
	}
}
