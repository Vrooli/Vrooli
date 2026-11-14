package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const conformantPRD = `# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Validate things.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | First target | Does the first thing.
- [x] OT-P0-002 | Second target | Does the second thing.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Later target | Does a later thing.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Future target | Does a future thing.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go.

## 🤝 Dependencies & Launch Plan
- None.

## 🎨 UX & Branding
- Accessibility: WCAG AA.

## 📎 Appendix
Nothing.
`

func writePRD(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "PRD.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func codesOf(findings []Finding) map[string]int {
	out := map[string]int{}
	for _, f := range findings {
		out[f.Code]++
	}
	return out
}

func TestExtractPRDDocumentMissing(t *testing.T) {
	doc, err := ExtractPRDDocument(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Present {
		t.Fatal("expected Present=false for missing PRD")
	}
}

func TestExtractPRDDocumentConformant(t *testing.T) {
	doc, err := ExtractPRDDocument(writePRD(t, conformantPRD))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Present {
		t.Fatal("expected Present")
	}
	if len(doc.Targets) != 4 {
		t.Fatalf("targets = %d, want 4", len(doc.Targets))
	}
	first := doc.Targets[0]
	if first.ID != "OT-P0-001" || first.Tier != "P0" || first.Checked {
		t.Fatalf("unexpected first target: %+v", first)
	}
	if first.Title != "First target" || first.Description != "Does the first thing." {
		t.Fatalf("pipe fields not parsed: %+v", first)
	}
	if !doc.Targets[1].Checked {
		t.Fatal("expected OT-P0-002 checked")
	}
	if doc.Targets[2].Tier != "P1" || doc.Targets[3].Tier != "P2" {
		t.Fatalf("tier attribution wrong: %+v %+v", doc.Targets[2], doc.Targets[3])
	}

	// A conformant PRD passes every template check.
	tmpl := DefaultPRDTemplate()
	for name, findings := range map[string][]Finding{
		"sections":   CheckTemplateSections(doc, tmpl),
		"unexpected": CheckUnexpectedSections(doc, tmpl),
		"content":    CheckTemplateContent(doc, tmpl),
		"ot-format":  CheckOTIDFormat(doc),
	} {
		if len(findings) != 0 {
			t.Fatalf("%s: expected clean, got %+v", name, findings)
		}
	}
}

func TestCheckTemplateSectionsMissing(t *testing.T) {
	content := strings.Replace(conformantPRD, "## 🧱 Tech Direction Snapshot\n- Preferred stacks: Go.\n\n", "", 1)
	content = strings.Replace(content, "### 🟢 P2 – Future / expansion\n- [ ] OT-P2-001 | Future target | Does a future thing.\n\n", "", 1)
	doc, err := ExtractPRDDocument(writePRD(t, content))
	if err != nil {
		t.Fatal(err)
	}
	findings := CheckTemplateSections(doc, DefaultPRDTemplate())
	if codesOf(findings)[CodeTemplateSections] != 2 {
		t.Fatalf("expected 2 missing-section findings, got %+v", findings)
	}
	var sawSub bool
	for _, f := range findings {
		if f.Severity != "error" {
			t.Fatalf("severity = %q, want error", f.Severity)
		}
		if strings.Contains(f.Message, "Operational Targets > ") {
			sawSub = true
		}
	}
	if !sawSub {
		t.Fatal("expected the missing P2 subsection to carry its parent label")
	}
}

func TestCheckTemplateSectionsMatchesWithoutEmoji(t *testing.T) {
	// Emoji-stripped headings still match (textual identity), per the
	// legacy validator's normalization.
	content := strings.ReplaceAll(conformantPRD, "## 🧱 Tech Direction Snapshot", "## Tech Direction Snapshot")
	doc, err := ExtractPRDDocument(writePRD(t, content))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(CheckTemplateSections(doc, DefaultPRDTemplate())); n != 0 {
		t.Fatalf("expected emoji-less heading to match, got %d findings", n)
	}
}

func TestCheckUnexpectedSections(t *testing.T) {
	content := conformantPRD + "\n## Roadmap\nStuff.\n"
	doc, err := ExtractPRDDocument(writePRD(t, content))
	if err != nil {
		t.Fatal(err)
	}
	findings := CheckUnexpectedSections(doc, DefaultPRDTemplate())
	if len(findings) != 1 || findings[0].Code != CodeTemplateUnexpectedSections {
		t.Fatalf("expected one unexpected-section finding, got %+v", findings)
	}
	if findings[0].Severity != "info" {
		t.Fatalf("severity = %q, want info", findings[0].Severity)
	}
}

func TestCheckTemplateContent(t *testing.T) {
	t.Run("missing required content is an error", func(t *testing.T) {
		content := strings.Replace(conformantPRD, "- **Purpose**: Validate things.", "- Something else entirely.", 1)
		doc, err := ExtractPRDDocument(writePRD(t, content))
		if err != nil {
			t.Fatal(err)
		}
		findings := CheckTemplateContent(doc, DefaultPRDTemplate())
		if len(findings) != 1 || findings[0].Severity != "error" || !strings.Contains(findings[0].Message, "missing_content") {
			t.Fatalf("unexpected findings: %+v", findings)
		}
	})
	t.Run("empty section is a warning", func(t *testing.T) {
		content := strings.Replace(conformantPRD, "## 🤝 Dependencies & Launch Plan\n- None.", "## 🤝 Dependencies & Launch Plan", 1)
		doc, err := ExtractPRDDocument(writePRD(t, content))
		if err != nil {
			t.Fatal(err)
		}
		findings := CheckTemplateContent(doc, DefaultPRDTemplate())
		if len(findings) != 1 || findings[0].Severity != "warning" || !strings.Contains(findings[0].Message, "empty_section") {
			t.Fatalf("unexpected findings: %+v", findings)
		}
	})
	t.Run("P0 tier without checklist lines is an error", func(t *testing.T) {
		content := strings.Replace(conformantPRD,
			"- [ ] OT-P0-001 | First target | Does the first thing.\n- [x] OT-P0-002 | Second target | Does the second thing.",
			"Plain prose, no checklist.", 1)
		doc, err := ExtractPRDDocument(writePRD(t, content))
		if err != nil {
			t.Fatal(err)
		}
		var sawChecklist bool
		for _, f := range CheckTemplateContent(doc, DefaultPRDTemplate()) {
			if strings.Contains(f.Message, "invalid_checklist") && f.Severity == "error" {
				sawChecklist = true
			}
		}
		if !sawChecklist {
			t.Fatal("expected an invalid_checklist error for the P0 tier")
		}
	})
}

func TestCheckOTIDFormat(t *testing.T) {
	content := strings.Replace(conformantPRD, "OT-P0-001", "OT-P0-1", 1)
	content = strings.Replace(content, "OT-P1-001", "OT-P0-005", 1)
	doc, err := ExtractPRDDocument(writePRD(t, content))
	if err != nil {
		t.Fatal(err)
	}
	findings := CheckOTIDFormat(doc)
	if len(findings) != 2 {
		t.Fatalf("expected 2 format findings, got %+v", findings)
	}
	if findings[0].ClaimID != "OT-P0-001" {
		t.Fatalf("expected canonicalized claim id OT-P0-001, got %q", findings[0].ClaimID)
	}
	if !strings.Contains(findings[1].Message, "tier") {
		t.Fatalf("expected tier mismatch message, got %q", findings[1].Message)
	}
}

func TestCanonicalOTID(t *testing.T) {
	for raw, want := range map[string]string{
		"OT-P0-1":    "OT-P0-001",
		"ot-p2-042":  "OT-P2-042",
		"OT-P1-100":  "OT-P1-100",
		"OT-BAD-001": "OT-BAD-001",
	} {
		if got := CanonicalOTID(raw); got != want {
			t.Fatalf("CanonicalOTID(%q) = %q, want %q", raw, got, want)
		}
	}
}
