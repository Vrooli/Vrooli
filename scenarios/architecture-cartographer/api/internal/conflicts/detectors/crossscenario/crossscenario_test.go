package crossscenario_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/crossscenario"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func TestDetect_FlagsOtherScenarioInternalImport(t *testing.T) {
	got, err := crossscenario.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "billing",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:service", PackageID: "pkg:billing", Path: "api/internal/invoices/service.go"}},
			Packages: []graph.PackageNode{
				{ID: "pkg:billing", RepoPath: "api/internal/invoices", ImportPath: "billing/internal/invoices"},
				{ID: "pkg:private", ImportPath: "github.com/vrooli/vrooli/scenarios/payments/api/internal/ledger"},
			},
			Imports: []graph.ImportEdge{{From: "file:service", ToPackageID: "pkg:private"}},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{{
			Name: "invoices", Paths: []string{"api/internal/invoices/**"},
		}}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Type != "cross_scenario" || got[0].Subtype != "internal_import" || got[0].Severity != conflicts.SeverityError {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
	if len(got[0].Domains) != 1 || got[0].Domains[0] != "invoices" {
		t.Fatalf("expected importing domain, got %+v", got[0].Domains)
	}
}

func TestDetect_AllowsPublicScenarioImport(t *testing.T) {
	got, err := crossscenario.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "billing",
		Snapshot: graph.GraphSnapshot{
			Packages: []graph.PackageNode{
				{ID: "pkg:billing", RepoPath: "api/internal/invoices"},
				{ID: "pkg:client", ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/payments/v1/ledger"},
			},
			Imports: []graph.ImportEdge{{From: "pkg:billing", ToPackageID: "pkg:client"}},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected public import to pass, got %+v", got)
	}
}

func TestDetect_AllowsSameScenarioInternalImport(t *testing.T) {
	got, err := crossscenario.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "billing",
		Snapshot: graph.GraphSnapshot{
			Packages: []graph.PackageNode{
				{ID: "pkg:billing", RepoPath: "api/internal/invoices"},
				{ID: "pkg:same", ImportPath: "github.com/vrooli/vrooli/scenarios/billing/api/internal/ledger"},
			},
			Imports: []graph.ImportEdge{{From: "pkg:billing", ToPackageID: "pkg:same"}},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected same-scenario import to pass, got %+v", got)
	}
}

func TestDetect_IgnoresTestOnlyImport(t *testing.T) {
	got, err := crossscenario.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "billing",
		Snapshot: graph.GraphSnapshot{
			Packages: []graph.PackageNode{
				{ID: "pkg:billing", RepoPath: "api/internal/invoices"},
				{ID: "pkg:private", ImportPath: "github.com/vrooli/vrooli/scenarios/payments/api/internal/ledger"},
			},
			Imports: []graph.ImportEdge{{From: "pkg:billing", ToPackageID: "pkg:private", TestOnly: true}},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected test-only import to pass, got %+v", got)
	}
}
