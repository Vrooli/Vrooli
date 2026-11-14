package docschema

import (
	"path/filepath"
	"testing"
)

const validPerfAuditDoc = validPerfAuditFrontmatter

const perfAuditMissingTable = `---
date: 2026-05-03
scenario: swarm-manager
interactions:
  - command-post-sidebar-resize
status: open
---

# Audit body

No per-component table here at all.
`

const perfAuditMismatchedDate = `---
date: 2026-05-03
scenario: swarm-manager
interactions:
  - command-post-sidebar-resize
status: fixed
---

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| BacklogTab | 60 | 114 | 1900 | 8000 |
`

func TestAuditPerfDocs_ValidDocPasses(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")
	writeFile(t, filepath.Join(scenario, "docs", "perf", "2026-05-03-sidebar-resize.md"), validPerfAuditDoc)

	issues := auditPerfDocs(scenario)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %+v", len(issues), issues)
	}
}

func TestAuditPerfDocs_MissingTable(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")
	writeFile(t, filepath.Join(scenario, "docs", "perf", "2026-05-03-no-table.md"), perfAuditMissingTable)

	issues := auditPerfDocs(scenario)
	foundTable := false
	foundHeading := false
	for _, i := range issues {
		if i.Code == "perf-audit:missing-component-table" {
			foundTable = true
		}
		if i.Code == "perf-audit:missing-component-table-heading" {
			foundHeading = true
		}
	}
	if !foundTable {
		t.Errorf("expected missing-component-table issue, got %+v", issues)
	}
	if !foundHeading {
		t.Errorf("expected missing-component-table-heading issue, got %+v", issues)
	}
}

func TestAuditPerfDocs_FilenameDateMismatch(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")
	// File named 2026-04-01 but frontmatter says 2026-05-03.
	writeFile(t, filepath.Join(scenario, "docs", "perf", "2026-04-01-mismatch.md"), perfAuditMismatchedDate)

	issues := auditPerfDocs(scenario)
	found := false
	for _, i := range issues {
		if i.Code == "perf-audit:date-mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected date-mismatch issue, got %+v", issues)
	}
}

func TestAuditPerfDocs_IgnoresReadme(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")
	writeFile(t, filepath.Join(scenario, "docs", "perf", "README.md"), "# Index of perf audits")
	writeFile(t, filepath.Join(scenario, "docs", "perf", "2026-05-03-sidebar-resize.md"), validPerfAuditDoc)

	issues := auditPerfDocs(scenario)
	if len(issues) != 0 {
		t.Fatalf("README.md inside docs/perf should be ignored; got %+v", issues)
	}
}

func TestAuditPerfDocs_NoDirectoryReturnsNil(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")

	if got := auditPerfDocs(scenario); got != nil {
		t.Fatalf("expected nil when docs/perf doesn't exist, got %+v", got)
	}
}

func TestAuditScenarioDocumentation_IncludesPerfAuditIssues(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# x")
	writeFile(t, filepath.Join(scenario, "docs", "perf", "2026-05-03-broken.md"), perfAuditMissingTable)

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("AuditScenarioDocumentation: %v", err)
	}
	if len(result.PerfAuditIssues) == 0 {
		t.Fatal("expected PerfAuditIssues populated, got empty")
	}
}
