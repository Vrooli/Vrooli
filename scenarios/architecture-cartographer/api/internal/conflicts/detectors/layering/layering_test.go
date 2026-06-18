package layering_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/layering"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func testMap() domains.DerivedDomainMap {
	return domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**", "api/handlers/billing/**"}, Archetype: "validation"},
			{Name: "orders", Paths: []string{"api/internal/orders/**", "api/handlers/orders/**"}, Archetype: "service"},
		},
		SharedSubstrate: []string{"api/internal/database/**"},
	}
}

func detect(t *testing.T, snap graph.GraphSnapshot) []conflicts.Conflict {
	t.Helper()
	got, err := layering.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario:  "demo",
		Snapshot:  snap,
		DomainMap: testMap(),
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	return got
}

func TestDetect_AllowsHandlerToOwnDomain(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:handler", RepoPath: "api/handlers/billing"},
			{ID: "pkg:domain", RepoPath: "api/internal/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:handler", ToPackageID: "pkg:domain"}},
	})
	if len(got) != 0 {
		t.Fatalf("handler -> owning domain should be allowed, got %+v", got)
	}
}

func TestDetect_DomainImportingHandlerIsBlocker(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:domain", RepoPath: "api/internal/billing"},
			{ID: "pkg:handler", RepoPath: "api/handlers/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:domain", ToPackageID: "pkg:handler"}},
	})
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "domain-imports-transport" || got[0].Severity != conflicts.SeverityBlocker {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_SubstrateImportingDomainIsBlocker(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:database", RepoPath: "api/internal/database"},
			{ID: "pkg:domain", RepoPath: "api/internal/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:database", ToPackageID: "pkg:domain"}},
	})
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "substrate-imports-product" || got[0].Severity != conflicts.SeverityBlocker {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_CompositionRootMayImportHandlers(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:app", RepoPath: "api/internal/app"},
			{ID: "pkg:handler", RepoPath: "api/handlers/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:app", ToPackageID: "pkg:handler"}},
	})
	if len(got) != 0 {
		t.Fatalf("composition root importing handlers should be allowed, got %+v", got)
	}
}

func TestDetect_ModulesCompositionRootMayImportDomains(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:modules", RepoPath: "api/internal/modules"},
			{ID: "pkg:domain", RepoPath: "api/internal/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:modules", ToPackageID: "pkg:domain"}},
	})
	if len(got) != 0 {
		t.Fatalf("modules composition root importing domains should be allowed, got %+v", got)
	}
}

func TestDetect_NonCoordinatorDomainImportingSiblingDomain(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:billing", RepoPath: "api/internal/billing"},
			{ID: "pkg:orders", RepoPath: "api/internal/orders"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:billing", ToPackageID: "pkg:orders"}},
	})
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "domain-imports-sibling-domain" || got[0].Severity != conflicts.SeverityError {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_OrchestrationArchetypeMayCoordinateSiblingDomain(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:orders", RepoPath: "api/internal/orders"},
			{ID: "pkg:billing", RepoPath: "api/internal/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:orders", ToPackageID: "pkg:billing"}},
	})
	if len(got) != 0 {
		t.Fatalf("service/orchestration package should be allowed to coordinate, got %+v", got)
	}
}

func TestDetect_IgnoresTestOnlyEdges(t *testing.T) {
	got := detect(t, graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:domain", RepoPath: "api/internal/billing"},
			{ID: "pkg:handler", RepoPath: "api/handlers/billing"},
		},
		Imports: []graph.ImportEdge{{From: "pkg:domain", ToPackageID: "pkg:handler", TestOnly: true}},
	})
	if len(got) != 0 {
		t.Fatalf("test-only layering edge should be ignored, got %+v", got)
	}
}

func TestDetect_NonStrictUsesErrorForBlockerEligibleRule(t *testing.T) {
	got, err := layering.NewWithStrict(false).Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Packages: []graph.PackageNode{
				{ID: "pkg:domain", RepoPath: "api/internal/billing"},
				{ID: "pkg:handler", RepoPath: "api/handlers/billing"},
			},
			Imports: []graph.ImportEdge{{From: "pkg:domain", ToPackageID: "pkg:handler"}},
		},
		DomainMap: testMap(),
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 || got[0].Severity != conflicts.SeverityError {
		t.Fatalf("non-strict should emit error, got %+v", got)
	}
}
