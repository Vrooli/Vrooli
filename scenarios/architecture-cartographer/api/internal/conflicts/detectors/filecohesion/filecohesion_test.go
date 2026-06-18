package filecohesion_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/filecohesion"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func TestDetect_FlagsLargeFile(t *testing.T) {
	got, err := filecohesion.NewWithConfig(filecohesion.Config{MaxLines: 100, MaxSymbols: 2}).Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:billing", Path: "api/internal/billing/service.go", Lines: 150}},
			Symbols: []graph.SymbolNode{
				{ID: "sym:1", FileID: "file:billing"},
				{ID: "sym:2", FileID: "file:billing"},
				{ID: "sym:3", FileID: "file:billing"},
			},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Type != "file_cohesion" || got[0].Subtype != "oversized_file" || got[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
	if len(got[0].Evidence) != 2 || len(got[0].Domains) != 1 || got[0].Domains[0] != "billing" {
		t.Fatalf("expected line + symbol evidence in billing domain, got %+v", got[0])
	}
}

func TestDetect_IgnoresFilesWithinThresholds(t *testing.T) {
	got, err := filecohesion.NewWithConfig(filecohesion.Config{MaxLines: 100, MaxSymbols: 2}).Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:billing", Path: "api/internal/billing/service.go", Lines: 100}},
			Symbols: []graph.SymbolNode{
				{ID: "sym:1", FileID: "file:billing"},
				{ID: "sym:2", FileID: "file:billing"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}

func TestDetect_IgnoresTestFiles(t *testing.T) {
	got, err := filecohesion.NewWithConfig(filecohesion.Config{MaxLines: 10, MaxSymbols: 1}).Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:test", Path: "api/internal/billing/service_test.go", Lines: 500, IsTest: true}},
			Symbols: []graph.SymbolNode{
				{ID: "sym:1", FileID: "file:test"},
				{ID: "sym:2", FileID: "file:test"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no test-file conflicts, got %+v", got)
	}
}

func TestDetect_StableIDDeterministic(t *testing.T) {
	in := conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:billing", Path: "api/internal/billing/service.go", Lines: 150}},
		},
	}
	first, err := filecohesion.NewWithConfig(filecohesion.Config{MaxLines: 100}).Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect first: %v", err)
	}
	second, err := filecohesion.NewWithConfig(filecohesion.Config{MaxLines: 100}).Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect second: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one conflict in each run, got %+v / %+v", first, second)
	}
	if conflicts.StableID(first[0]) != conflicts.StableID(second[0]) {
		t.Fatalf("stable id drift: %s vs %s", conflicts.StableID(first[0]), conflicts.StableID(second[0]))
	}
}
