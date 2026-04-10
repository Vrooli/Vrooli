package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderDocsAuditReport_NoFindings(t *testing.T) {
	report := renderDocsAuditReport(docsAuditResponse{
		ScenarioName: "alpha",
		HealthScore:  1,
		TotalDocs:    12,
		Infrastructure: &docsAuditInfrastructure{
			MisplacedDocs: nil,
			MissingDocs:   nil,
			ExtraDocs:     nil,
			TemporaryDocs: nil,
		},
	}, "")

	assertContainsAll(t, report,
		"Documentation Audit: alpha",
		"Status: OK",
		"Health: 100% (12 docs)",
		"Findings: 0 total",
		"Triage",
		"- No findings",
		"Next steps",
		"1. No action required. To verify again later, run:",
		"knowledge-observatory docs audit alpha",
	)
	if strings.Contains(report, "--json") {
		t.Fatalf("expected no --json step for zero manual findings, got:\n%s", report)
	}
}

func TestRenderDocsAuditReport_MixedWithOverflowAndFail(t *testing.T) {
	report := renderDocsAuditReport(docsAuditResponse{
		ScenarioName: "beta",
		HealthScore:  0.82,
		TotalDocs:    23,
		Infrastructure: &docsAuditInfrastructure{
			MisplacedDocs: []docsAuditMisplacedDoc{
				{ActualPath: "ARCHITECTURE.md", ExpectedPath: "docs/concepts/ARCHITECTURE.md"},
				{ActualPath: "PROGRESS.md", ExpectedPath: "docs/internal/PROGRESS.md"},
				{ActualPath: "SEAMS.md", ExpectedPath: "docs/internal/SEAMS.md"},
				{ActualPath: "GLOSSARY.md", ExpectedPath: "docs/concepts/GLOSSARY.md"},
			},
			MissingDocs: []string{"invariants", "temporal-flows"},
			ExtraDocs:   []string{"docs/misc/NOTE.md"},
			TemporaryDocs: []string{
				"IMPLEMENTATION_PLAN.md",
			},
		},
		CodeWithoutDocRefs: []docsAuditUndocumentedFile{
			{Path: "api/server.go", ExportedSymbols: 4},
			{Path: "ui/src/App.tsx", ExportedSymbols: 2},
		},
		BrokenCodeRefs: []docsAuditBrokenRef{
			{DocPath: "docs/reference/api-endpoints.md", Line: 88, Target: "api/missing.go"},
		},
		OrphanedDocs: []string{"docs/internal/UNTRACKED.md"},
		DuplicateTitles: []docsAuditDuplicateTitle{
			{Title: "Overview", Files: []string{"docs/a.md", "docs/b.md"}},
		},
		UndocumentedTargets: []string{"OT-P01-04"},
	}, "")

	assertContainsAll(t, report,
		"Status: FAIL (drivers: 1 broken [CODE:] refs, 1 undocumented operational targets)",
		"Health: 82% (23 docs; 4 misplaced, 2 missing, 1 extra, 1 temporary)",
		"Findings: 14 total",
		"- Auto-fix now (4)",
		"- Agent repair (3)",
		"- Manual review (7)",
		"- No DOC refs (2)",
		"- Broken [CODE:] refs (1)",
		"- Temporary docs (1)",
		"docs/reference/api-endpoints.md:88 -> api/missing.go",
		"1. To apply deterministic quick fixes for misplaced docs, run:",
		"knowledge-observatory docs autofix beta",
		"2. To run agent-driven repair for missing/extra docs, run:",
		"knowledge-observatory docs heal beta --wait",
		"3. To inspect full findings in machine-readable form, run:",
		"knowledge-observatory docs audit beta --json",
		"4. To see a detailed documentation-health breakdown and penalties, run:",
		"knowledge-observatory docs health beta",
	)
}

func TestRenderDocsAuditReport_WarnAutoFixOnly(t *testing.T) {
	report := renderDocsAuditReport(docsAuditResponse{
		ScenarioName: "gamma",
		HealthScore:  0.91,
		TotalDocs:    14,
		Infrastructure: &docsAuditInfrastructure{
			MisplacedDocs: []docsAuditMisplacedDoc{
				{ActualPath: "ARCHITECTURE.md", ExpectedPath: "docs/concepts/ARCHITECTURE.md"},
			},
		},
	}, "")

	assertContainsAll(t, report,
		"Status: WARN",
		"Health: 91% (14 docs; 1 misplaced, 0 missing, 0 extra, 0 temporary)",
		"Findings: 1 total",
		"- Auto-fix now (1)",
		"1. To apply deterministic quick fixes for misplaced docs, run:",
		"knowledge-observatory docs autofix gamma",
		"2. To see a detailed documentation-health breakdown and penalties, run:",
		"knowledge-observatory docs health gamma",
	)
	if strings.Contains(report, "docs heal gamma --wait") {
		t.Fatalf("did not expect agent-heal step for auto-only findings, got:\n%s", report)
	}
	if strings.Contains(report, "docs audit gamma --json") {
		t.Fatalf("did not expect --json step for zero manual findings, got:\n%s", report)
	}
}

func TestRenderDocsAuditReport_ManualOverflowShowsPlusMore(t *testing.T) {
	manual := make([]docsAuditUndocumentedFile, 0, 12)
	for i := 0; i < 12; i++ {
		manual = append(manual, docsAuditUndocumentedFile{
			Path:            "api/file_" + string(rune('a'+i)) + ".go",
			ExportedSymbols: i + 1,
		})
	}
	report := renderDocsAuditReport(docsAuditResponse{
		ScenarioName:       "overflow",
		HealthScore:        0.9,
		TotalDocs:          10,
		CodeWithoutDocRefs: manual,
	}, "")

	assertContainsAll(t, report,
		"- Manual review (12)",
		"- No DOC refs (12)",
		"... +2 more",
	)
}

func TestRenderDocsHealthReport_Default(t *testing.T) {
	report := renderDocsHealthReport(docsHealthResponse{
		ScenarioName: "healthy",
		HealthScore:  0.87,
		TotalDocs:    9,
		MisplacedDocs: []docsAuditMisplacedDoc{
			{ActualPath: "ARCHITECTURE.md", ExpectedPath: "docs/concepts/ARCHITECTURE.md"},
		},
		MissingDocs: []string{"seams"},
		ExtraDocs:   []string{"docs/misc/NOTE.md", "docs/misc/IDEA.md"},
		TemporaryDocs: []string{
			"IMPLEMENTATION_PLAN.md",
		},
		CanAutoFix:  true,
		FixCategory: "mixed",
	}, "")

	assertContainsAll(t, report,
		"Documentation Health: healthy",
		"Score: 87% (9 docs)",
		"Issues: 1 misplaced, 1 missing, 2 extra, 1 temporary",
		"Score breakdown",
		"Required docs baseline: 100% (1/1 present)",
		"Misplaced penalty: -5% (1 x 5%)",
		"Temporary-docs penalty: -1% (1 x 1%)",
		"Extra docs are informational only (2)",
		"Fixability",
		"Fix category: mixed",
		"Quick-fixable files: 1",
		"Auto-fix available: yes",
	)
}

func TestLegacyInfrastructureJSON_Unmarshal(t *testing.T) {
	payload := `{
		"infrastructure": {
			"MisplacedDocs": [
				{
					"ActualPath": "docs/PROGRESS.md",
					"ExpectedPath": "docs/internal/PROGRESS.md",
					"DocType": "progress",
					"Severity": "warning"
				}
			],
			"MissingDocs": ["manifest"],
			"ExtraDocs": ["docs/misc/NOTE.md"],
			"TemporaryDocs": ["IMPLEMENTATION_PLAN.md"]
		}
	}`

	var decoded struct {
		Infrastructure docsAuditInfrastructure `json:"infrastructure"`
	}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Infrastructure.MisplacedDocs) != 1 {
		t.Fatalf("expected 1 misplaced doc, got %d", len(decoded.Infrastructure.MisplacedDocs))
	}
	if decoded.Infrastructure.MisplacedDocs[0].ActualPath != "docs/PROGRESS.md" {
		t.Fatalf("expected legacy ActualPath to be parsed, got %q", decoded.Infrastructure.MisplacedDocs[0].ActualPath)
	}
	if len(decoded.Infrastructure.TemporaryDocs) != 1 {
		t.Fatalf("expected 1 temporary doc, got %d", len(decoded.Infrastructure.TemporaryDocs))
	}
}

func assertContainsAll(t *testing.T, got string, expected ...string) {
	t.Helper()
	for _, want := range expected {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q\n\nOutput:\n%s", want, got)
		}
	}
}
