package docschema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateScenarioDocumentation(t *testing.T) {
	scenario := t.TempDir()

	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "PRD.md"), "# PRD")

	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), "{}")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")
	writeFile(t, filepath.Join(scenario, "docs", "PROGRESS.md"), "# Progress")
	writeFile(t, filepath.Join(scenario, "docs", "QUICKSTART.md"), "# Quickstart")
	writeFile(t, filepath.Join(scenario, "docs", "concepts", "ARCHITECTURE.md"), "# Architecture")
	writeFile(t, filepath.Join(scenario, "docs", "misc", "NOTE.md"), "# Note")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "IMPLEMENTATION_PLAN.md"), "# Temp Plan")

	result, err := ValidateScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation returned error: %v", err)
	}

	if !hasMisplacedDoc(result.MisplacedDocs, DocTypeProgress, "docs/PROGRESS.md", "docs/internal/PROGRESS.md") {
		t.Fatalf("expected misplaced progress doc, got: %#v", result.MisplacedDocs)
	}

	if !containsDocType(result.MissingDocs, DocTypeProgress) {
		t.Fatalf("expected missing progress doc")
	}

	if !containsString(result.ExtraDocs, "docs/misc/NOTE.md") {
		t.Fatalf("expected extra docs to include note, got: %#v", result.ExtraDocs)
	}
	if !containsString(result.TemporaryDocs, "docs/internal/IMPLEMENTATION_PLAN.md") {
		t.Fatalf("expected temporary docs to include implementation plan, got: %#v", result.TemporaryDocs)
	}

	if result.HealthScore < 0 || result.HealthScore > 1 {
		t.Fatalf("expected health score between 0 and 1, got %f", result.HealthScore)
	}
}

func TestValidateScenarioDocumentation_ExtraDocsDoNotAffectHealth(t *testing.T) {
	base := t.TempDir()
	withExtra := t.TempDir()

	setup := func(root string) {
		writeFile(t, filepath.Join(root, "README.md"), "# Scenario")
		writeFile(t, filepath.Join(root, "PRD.md"), "# PRD")
		writeFile(t, filepath.Join(root, "docs", "manifest.json"), "{}")
		writeFile(t, filepath.Join(root, "docs", "internal", "PROBLEMS.md"), "# Problems")
		writeFile(t, filepath.Join(root, "docs", "internal", "PROGRESS.md"), "# Progress")
		writeFile(t, filepath.Join(root, "docs", "internal", "SEAMS.md"), "# Seams")
		writeFile(t, filepath.Join(root, "docs", "internal", "INVARIANTS.md"), "# Invariants")
		writeFile(t, filepath.Join(root, "docs", "internal", "ASSUMPTIONS.md"), "# Assumptions")
		writeFile(t, filepath.Join(root, "docs", "internal", "ERROR-SEMANTICS.md"), "# Error Semantics")
		writeFile(t, filepath.Join(root, "docs", "internal", "SECURITY-POSTURE.md"), "# Security Posture")
		writeFile(t, filepath.Join(root, "docs", "internal", "TEMPORAL-FLOWS.md"), "# Temporal Flows")
		writeFile(t, filepath.Join(root, "docs", "internal", "COHERENCE-NOTES.md"), "# Coherence Notes")
		writeFile(t, filepath.Join(root, "docs", "internal", "EXPERIENCE-AUDIT.md"), "# Experience Audit")
		writeFile(t, filepath.Join(root, "docs", "QUICKSTART.md"), "# Quickstart")
		writeFile(t, filepath.Join(root, "docs", "concepts", "ARCHITECTURE.md"), "# Architecture")
		writeFile(t, filepath.Join(root, "docs", "concepts", "GLOSSARY.md"), "# Glossary")
	}

	setup(base)
	setup(withExtra)
	writeFile(t, filepath.Join(withExtra, "docs", "misc", "NOTE.md"), "# Extra note")

	baseResult, err := ValidateScenarioDocumentation(base)
	if err != nil {
		t.Fatalf("base validation error: %v", err)
	}
	extraResult, err := ValidateScenarioDocumentation(withExtra)
	if err != nil {
		t.Fatalf("extra validation error: %v", err)
	}

	if baseResult.HealthScore != extraResult.HealthScore {
		t.Fatalf("expected extra docs to be non-penalizing: base=%f extra=%f", baseResult.HealthScore, extraResult.HealthScore)
	}
}

func TestValidateScenarioDocumentation_TemporalFlowsContentHeadings(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "PRD.md"), "# PRD")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), "{}")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROGRESS.md"), "# Progress")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "SEAMS.md"), "# Seams")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "TEMPORAL-FLOWS.md"), `# Temporal Flows

## Flow Index

## Unmodeled Candidates
`)

	result, err := ValidateScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation returned error: %v", err)
	}

	if !hasContentIssue(result.ContentIssues, DocTypeTemporalFlows, "missing heading: Declarative & Formal Spec Status") {
		t.Fatalf("expected missing spec-status heading issue, got: %#v", result.ContentIssues)
	}
	if !hasContentIssue(result.ContentIssues, DocTypeTemporalFlows, "missing heading: Audit Notes") {
		t.Fatalf("expected missing audit-notes heading issue, got: %#v", result.ContentIssues)
	}
}

func TestValidateScenarioDocumentation_TemporalFlowsTemplateHasRequiredHeadings(t *testing.T) {
	template, err := TemplateForDocType(DocTypeTemporalFlows)
	if err != nil {
		t.Fatalf("TemplateForDocType returned error: %v", err)
	}
	for _, heading := range requiredTemporalFlowsHeadings() {
		if !hasMarkdownHeading(template, heading) {
			t.Fatalf("temporal-flows template missing heading %q", heading)
		}
	}
}

func TestValidateScenarioDocumentation_ArchitectureContentHeadings(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "docs", "concepts", "ARCHITECTURE.md"), `# Architecture

## Surfaces

## Domain Map
`)

	result, err := ValidateScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation returned error: %v", err)
	}

	if !hasContentIssue(result.ContentIssues, DocTypeArchitecture, "missing heading: Shared Infrastructure") {
		t.Fatalf("expected missing shared-infrastructure heading issue, got: %#v", result.ContentIssues)
	}
	if !hasContentIssue(result.ContentIssues, DocTypeArchitecture, "missing heading: Architecture Maturity") {
		t.Fatalf("expected missing maturity heading issue, got: %#v", result.ContentIssues)
	}
}

func TestValidateScenarioDocumentation_ArchitectureTemplatesHaveRequiredHeadings(t *testing.T) {
	tests := []struct {
		docType  DocType
		headings []string
	}{
		{DocTypeArchitecture, requiredArchitectureHeadings()},
		{DocTypeSeams, requiredSeamsHeadings()},
		{DocTypeProblems, requiredProblemsHeadings()},
	}
	for _, tc := range tests {
		template, err := TemplateForDocType(tc.docType)
		if err != nil {
			t.Fatalf("TemplateForDocType(%s) returned error: %v", tc.docType, err)
		}
		for _, heading := range tc.headings {
			if !hasMarkdownHeading(template, heading) {
				t.Fatalf("%s template missing heading %q", tc.docType, heading)
			}
		}
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func containsDocType(values []DocType, target DocType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasMisplacedDoc(docs []MisplacedDoc, docType DocType, actual string, expected string) bool {
	for _, doc := range docs {
		if doc.DocType == docType && doc.ActualPath == actual && doc.ExpectedPath == expected {
			return true
		}
	}
	return false
}

func hasContentIssue(issues []DocContentIssue, docType DocType, message string) bool {
	for _, issue := range issues {
		if issue.DocType == docType && issue.Message == message {
			return true
		}
	}
	return false
}
