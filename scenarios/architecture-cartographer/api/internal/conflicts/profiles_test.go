package conflicts

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

type namedDetector string

func (d namedDetector) Name() string         { return string(d) }
func (d namedDetector) Description() string  { return string(d) }
func (d namedDetector) EmitsTypes() []string { return []string{string(d)} }
func (d namedDetector) Detect(context.Context, DetectInput) ([]Conflict, error) {
	return []Conflict{{Detector: string(d), Type: string(d)}}, nil
}

func TestProfiledRegistrySelectsAPIAndUniversalDetectors(t *testing.T) {
	reg := NewRegistryWithProfiles(DefaultSurfaceProfiles(),
		namedDetector("cross_scenario"),
		namedDetector("cycle"),
		namedDetector("file_cohesion"),
		namedDetector("glossary_drift"),
		namedDetector("layering"),
		namedDetector("surface_coherence"),
		namedDetector("unused_ui_only"),
	)
	got, err := reg.DetectAll(context.Background(), DetectInput{
		Snapshot: graph.GraphSnapshot{Files: []graph.FileNode{{
			ID: "file:api", Path: "api/internal/billing/service.go", Language: graph.LanguageGo,
		}}},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{{
			Name: "billing", Paths: []string{"api/internal/billing/**"},
		}}},
	})
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if detectors(got)["cross_scenario"] == false || detectors(got)["cycle"] == false || detectors(got)["file_cohesion"] == false || detectors(got)["glossary_drift"] == false || detectors(got)["layering"] == false || detectors(got)["surface_coherence"] == false {
		t.Fatalf("expected api + universal detectors, got %+v", got)
	}
	if detectors(got)["unused_ui_only"] {
		t.Fatalf("unexpected detector selected: %+v", got)
	}
}

func TestProfiledRegistryUsesDomainDeclarationsWhenGraphIsEmpty(t *testing.T) {
	reg := NewRegistryWithProfiles(DefaultSurfaceProfiles(),
		namedDetector("convergence_drift"),
		namedDetector("cross_scenario"),
		namedDetector("cycle"),
		namedDetector("layering"),
	)
	got, err := reg.DetectAll(context.Background(), DetectInput{
		DomainMap: domains.DerivedDomainMap{Declarations: []domains.DomainDeclaration{{
			Source: domains.SourceCLIGroups, DomainNames: []string{"billing"},
		}}},
	})
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	ds := detectors(got)
	if !ds["cycle"] || !ds["convergence_drift"] || !ds["cross_scenario"] || !ds["layering"] {
		t.Fatalf("expected cli profile + universal detectors, got %+v", got)
	}
}

func detectors(cs []Conflict) map[string]bool {
	out := map[string]bool{}
	for _, c := range cs {
		out[c.Detector] = true
	}
	return out
}
