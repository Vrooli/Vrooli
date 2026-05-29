package domains

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

const goldenDoc = `# Domains — Example

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths (planned) | Glossary |
|---|---|---|---|---|---|---|---|
| graph | Build the graph. | service / orchestration | snapshots | API, CLI | OT-1 | ` + "`api/internal/graph/`, `api/handlers/graph/`" + ` | GraphSnapshot, ImportEdge |
| analytics | Event log. | reporting / query | events | API | OT-9 | ` + "`api/internal/analytics/`" + ` | Event, Placement |
| conflicts | Detect drift. | service / classification | conflicts | API, CLI, UI | OT-3 | ` + "`api/internal/conflicts/`" + ` | |

## Non-Domains

These are infrastructure, not product domains:

- ` + "`api/internal/server/`" + ` — HTTP composition substrate.
- ` + "`api/internal/database/`" + ` — cross-cutting database infrastructure.
- ` + "`api/main.go`" + ` — composition root.

## Cross-References

- Something else.
`

func TestDomainsDocExtractor_Golden(t *testing.T) {
	e := NewDomainsDocExtractor()
	ext, err := e.parse(goldenDoc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ext.Source != SourceDomainsDoc {
		t.Fatalf("source = %q, want %q", ext.Source, SourceDomainsDoc)
	}
	// Domains sorted by name: analytics, conflicts, graph.
	wantNames := []string{"analytics", "conflicts", "graph"}
	gotNames := make([]string, 0, len(ext.Domains))
	for _, d := range ext.Domains {
		gotNames = append(gotNames, d.Name)
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("domain names = %v, want %v", gotNames, wantNames)
	}

	byName := map[string]ExtractedDomain{}
	for _, d := range ext.Domains {
		byName[d.Name] = d
	}

	// graph: paths (backtick-stripped, comma-split), archetype first token, glossary.
	graph := byName["graph"]
	if !reflect.DeepEqual(graph.Paths, []string{"api/internal/graph/", "api/handlers/graph/"}) {
		t.Fatalf("graph paths = %v", graph.Paths)
	}
	if graph.Archetype != "service" {
		t.Fatalf("graph archetype = %q, want service", graph.Archetype)
	}
	if !reflect.DeepEqual(graph.Glossary, []string{"GraphSnapshot", "ImportEdge"}) {
		t.Fatalf("graph glossary = %v", graph.Glossary)
	}

	// analytics archetype "reporting / query" -> reporting.
	if byName["analytics"].Archetype != "reporting" {
		t.Fatalf("analytics archetype = %q", byName["analytics"].Archetype)
	}

	// conflicts has an empty glossary cell.
	if len(byName["conflicts"].Glossary) != 0 {
		t.Fatalf("conflicts glossary = %v, want empty", byName["conflicts"].Glossary)
	}

	// Non-Domains -> shared substrate + names.
	wantShared := []string{"api/internal/database/", "api/internal/server/", "api/main.go"}
	if !reflect.DeepEqual(ext.SharedSubstrate, wantShared) {
		t.Fatalf("shared substrate = %v, want %v", ext.SharedSubstrate, wantShared)
	}
	wantNonDomNames := []string{"database", "main.go", "server"}
	if !reflect.DeepEqual(ext.NonDomains, wantNonDomNames) {
		t.Fatalf("non-domains = %v, want %v", ext.NonDomains, wantNonDomNames)
	}
}

func TestDomainsDocExtractor_MissingFile(t *testing.T) {
	e := NewDomainsDocExtractor()
	ext, err := e.Extract(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("missing DOMAINS.md should not error, got %v", err)
	}
	if len(ext.Domains) != 0 {
		t.Fatalf("expected empty extraction, got %d domains", len(ext.Domains))
	}
}

func TestDomainsDocExtractor_Errors(t *testing.T) {
	e := NewDomainsDocExtractor()
	t.Run("missing inventory section", func(t *testing.T) {
		_, err := e.parse("# Title\n\n## Other\n\ntext\n")
		if err == nil || !strings.Contains(err.Error(), "Domain Inventory") {
			t.Fatalf("want missing-inventory error, got %v", err)
		}
	})
	t.Run("missing required column", func(t *testing.T) {
		doc := "## Domain Inventory\n\n| Domain | Purpose |\n|---|---|\n| graph | x |\n"
		_, err := e.parse(doc)
		if err == nil || !strings.Contains(err.Error(), "Source Paths") {
			t.Fatalf("want missing-column error, got %v", err)
		}
	})
	t.Run("domain with no paths", func(t *testing.T) {
		// Per L5-readiness Phase 3: a row with no source paths is now a
		// non-fatal warning (skipped + reported), not a hard error.
		doc := "## Domain Inventory\n\n| Domain | Source Paths |\n|---|---|\n| graph |  |\n"
		ex, err := e.parse(doc)
		if err != nil {
			t.Fatalf("want nil err with warning, got %v", err)
		}
		if len(ex.Domains) != 0 {
			t.Fatalf("want empty domains, got %d", len(ex.Domains))
		}
		if len(ex.Warnings) != 1 || ex.Warnings[0].Kind != "domains_doc.no_paths" {
			t.Fatalf("want one no_paths warning, got %+v", ex.Warnings)
		}
	})
}

func TestNormalizeArchetype(t *testing.T) {
	cases := map[string]string{
		"service / orchestration": "service",
		"reporting / query":       "reporting",
		"validation / contract":   "validation",
		"Composition-Root":        "composition-root",
		"weird thing":             "weird thing",
		"":                        "",
	}
	for in, want := range cases {
		if got := normalizeArchetype(in); got != want {
			t.Fatalf("normalizeArchetype(%q) = %q, want %q", in, got, want)
		}
	}
}
