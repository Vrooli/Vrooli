package surfacecoherence_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/surfacecoherence"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func TestDetect_FlagsDeclaredMissingUISurface(t *testing.T) {
	got := detect(t, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{
			Name:       "billing",
			Archetypes: domains.DeclaredArchetypes("service"),
			Paths: []string{
				"api/internal/billing/**",
				"ui/src/features/billing/**",
			},
		}},
		Declarations: []domains.DomainDeclaration{
			{Source: domains.SourceAPIFolders, DomainNames: []string{"billing"}},
			{Source: domains.SourceUIFeatures, DomainNames: []string{"orders"}},
		},
	})
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "missing_ui" || got[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_NoConflictWhenDeclaredSurfaceHasEvidence(t *testing.T) {
	got := detect(t, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{
			Name:  "billing",
			Paths: []string{"api/internal/billing/**", "ui/src/features/billing/**"},
		}},
		Declarations: []domains.DomainDeclaration{
			{Source: domains.SourceAPIFolders, DomainNames: []string{"billing"}},
			{Source: domains.SourceUIFeatures, DomainNames: []string{"billing"}},
		},
	})
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}

func TestDetect_ArchetypeImpliedAPIRequiresEvidenceWhenAPISurfaceExists(t *testing.T) {
	got := detect(t, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "billing", Archetypes: domains.DeclaredArchetypes("service"), Paths: []string{"cli/domains/billing/**"}},
			{Name: "orders", Archetypes: domains.DeclaredArchetypes("service"), Paths: []string{"api/internal/orders/**"}},
		},
		Declarations: []domains.DomainDeclaration{
			{Source: domains.SourceAPIFolders, DomainNames: []string{"orders"}},
			{Source: domains.SourceCLIGroups, DomainNames: []string{"billing"}},
		},
	})
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Domains[0] != "billing" || got[0].Subtype != "missing_api" {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_IgnoresInactiveSurfaceExpectations(t *testing.T) {
	got := detect(t, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{
			Name:  "billing",
			Paths: []string{"api/internal/billing/**", "ui/src/features/billing/**"},
		}},
		Declarations: []domains.DomainDeclaration{
			{Source: domains.SourceAPIFolders, DomainNames: []string{"billing"}},
		},
	})
	if len(got) != 0 {
		t.Fatalf("inactive UI surface must not emit, got %+v", got)
	}
}

func detect(t *testing.T, m domains.DerivedDomainMap) []conflicts.Conflict {
	t.Helper()
	got, err := surfacecoherence.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario:  "demo",
		Snapshot:  graph.GraphSnapshot{},
		DomainMap: m,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	return got
}
