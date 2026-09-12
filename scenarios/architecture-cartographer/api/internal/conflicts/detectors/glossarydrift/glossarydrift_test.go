package glossarydrift_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/glossarydrift"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func TestDetect_ForeignGlossaryToken(t *testing.T) {
	got, err := glossarydrift.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:ledger", Path: "api/internal/billing/ledger.go"}},
			Symbols: []graph.SymbolNode{{
				ID:       "sym:invoice",
				Name:     "InvoiceLedger",
				FileID:   "file:ledger",
				Exported: true,
			}},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}, Glossary: []string{"Payment"}},
			{Name: "invoices", Paths: []string{"api/internal/invoices/**"}, Glossary: []string{"Invoice"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Type != "glossary_drift" || got[0].Subtype != "foreign_domain_vocabulary" || got[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
	if len(got[0].Domains) != 2 || got[0].Domains[0] != "billing" || got[0].Domains[1] != "invoices" {
		t.Fatalf("unexpected domains: %+v", got[0].Domains)
	}
}

func TestDetect_AllowsCurrentDomainGlossary(t *testing.T) {
	got, err := glossarydrift.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files:   []graph.FileNode{{ID: "file:invoice", Path: "api/internal/invoices/service.go"}},
			Symbols: []graph.SymbolNode{{ID: "sym:invoice", Name: "InvoiceService", FileID: "file:invoice", Exported: true}},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "invoices", Paths: []string{"api/internal/invoices/**"}, Glossary: []string{"Invoice"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected current-domain vocabulary to pass, got %+v", got)
	}
}

func TestDetect_IgnoresTransportHandlers(t *testing.T) {
	got, err := glossarydrift.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:handler", Path: "api/handlers/billing/handler.go"}},
			Symbols: []graph.SymbolNode{{
				ID:       "sym:invoice-handler",
				Name:     "InvoiceLedgerHandler",
				FileID:   "file:handler",
				Exported: true,
			}},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/handlers/billing/**", "api/internal/billing/**"}, Glossary: []string{"Payment"}},
			{Name: "invoices", Paths: []string{"api/internal/invoices/**"}, Glossary: []string{"Invoice"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("transport handlers translate foreign domain vocabulary and must not be flagged, got %+v", got)
	}
}

func TestDetect_IgnoresUnexportedSymbolsAndMissingGlossary(t *testing.T) {
	got, err := glossarydrift.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files: []graph.FileNode{{ID: "file:billing", Path: "api/internal/billing/service.go"}},
			Symbols: []graph.SymbolNode{
				{ID: "sym:private", Name: "invoiceLedger", FileID: "file:billing"},
				{ID: "sym:unknown", Name: "CustomerLedger", FileID: "file:billing", Exported: true},
			},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no glossary-backed drift, got %+v", got)
	}
}

func TestDetect_StableIDDeterministic(t *testing.T) {
	in := conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{
			Files:   []graph.FileNode{{ID: "file:ledger", Path: "api/internal/billing/ledger.go"}},
			Symbols: []graph.SymbolNode{{ID: "sym:invoice", Name: "InvoiceLedger", FileID: "file:ledger", Exported: true}},
		},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
			{Name: "invoices", Paths: []string{"api/internal/invoices/**"}, Glossary: []string{"Invoice"}},
		}},
	}
	first, err := glossarydrift.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect first: %v", err)
	}
	second, err := glossarydrift.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect second: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one conflict in both runs, got %+v / %+v", first, second)
	}
	if conflicts.StableID(first[0]) != conflicts.StableID(second[0]) {
		t.Fatalf("stable id drift: %s vs %s", conflicts.StableID(first[0]), conflicts.StableID(second[0]))
	}
}
