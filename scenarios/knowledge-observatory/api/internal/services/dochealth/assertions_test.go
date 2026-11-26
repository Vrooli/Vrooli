package dochealth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAssertionFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAssertionFindingsFailDuplicateAndOppositeClaimsWithinOneDocument(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/policy.md", "# Policy\n\nThe owner must record each source revision.\nThe owner must record each source revision.\nThe owner must not record each source revision.\n")

	findings := assertionFindings([]string{path}, root)
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.Code]++
		if finding.Severity != SeverityFailure {
			t.Errorf("same-document finding %q severity = %v, want failure", finding.Code, finding.Severity)
		}
	}
	if counts["duplicate_assertion"] != 1 {
		t.Errorf("duplicate_assertion findings = %d, want 1: %+v", counts["duplicate_assertion"], findings)
	}
	if counts["contradictory_assertion"] != 2 {
		t.Errorf("contradictory_assertion findings = %d, want both sides of the conflict: %+v", counts["contradictory_assertion"], findings)
	}
}

func TestAssertionFindingsWarnWhenIdenticalStatementsSpanFiles(t *testing.T) {
	root := t.TempDir()
	left := writeAssertionFixture(t, root, "docs/operations.md", "Every request must include its owner identity.\n")
	right := writeAssertionFixture(t, root, "docs/reference.md", "Every request must include its owner identity.\n")

	findings := assertionFindings([]string{left, right}, root)
	if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityWarning {
		t.Fatalf("cross-file duplicate should be advisory pending scope review: %+v", findings)
	}
}

func TestAssertionFindingsKeepRepeatedClaimsInDistinctSectionsAdvisory(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/decisions.md", "## Decision A\nThe operator must record publication revocation.\n\n## Decision B\nThe operator must record publication revocation.\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityWarning {
		t.Fatalf("same wording in distinct section scopes should remain advisory: %+v", findings)
	}
}

func TestAssertionFindingsJoinWrappedProseAndIgnoreNormativeHeadings(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/policy.md", "You must:\n\n- The editor must check each source revision before editing.\n- The editor must check each source revision before editing.\n\nYou must NOT:\n\nObserve the operation once using the supported wait mechanism. Do not\npoll repeatedly. Verify the final result.\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityFailure {
		t.Fatalf("wrapped duplicate should fail once while headings and line breaks stay structural: %+v", findings)
	}
	if findings[0].Target != "The editor must check each source revision before editing." {
		t.Fatalf("unexpected duplicate target: %+v", findings[0])
	}
}

func TestAssertionFindingsKeepSupersededHistoricalRepeatsAdvisory(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/ledger.md", "> **SUPERSEDED — historical provenance only (2026-10-15).**\n\nThe owner must preserve this receipt.\n\n## Checkpoint 1\nThe owner must preserve this receipt.\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityWarning {
		t.Fatalf("retained historical repeat should remain visible without blocking current docs: %+v", findings)
	}
}

func TestAssertionFindingsKeepRetainedEvidenceRepeatsAdvisory(t *testing.T) {
	for _, rel := range []string{
		"docs/internal/goal/evidence/E1/checkpoint.md",
		"docs/internal/delivery-reliability-research/sources/run-1/prewrite/E1.md",
	} {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			path := writeAssertionFixture(t, root, rel, "The owner must preserve this receipt.\nThe owner must preserve this receipt.\n")

			findings := assertionFindings([]string{path}, root)
			if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityWarning {
				t.Fatalf("retained evidence repeat should stay visible as advisory: %+v", findings)
			}
		})
	}
}

func TestAssertionFindingsKeepBoilerplateInRetainedEvidenceAdvisory(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/internal/goal/evidence/E1/baseline.md", "# Baseline\n\n## Purpose Of This Document\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "boilerplate_purpose_heading" || findings[0].Severity != SeverityWarning {
		t.Fatalf("retained evidence heading should stay visible as advisory: %+v", findings)
	}
}

func TestAssertionFindingsIgnoreRepeatedTableHeadersButCheckDataRows(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/internal/SEAMS.md", "# Seams\n\n| File | Boundary | Can Change | Cannot Change |\n|---|---|---|---|\n| api/a.go | a | yes | no |\n| File | Boundary | Can Change | Cannot Change |\n|---|---|---|---|\n\n| Rule | Statement |\n|---|---|\n| storage | The owner must preserve every row. |\n| storage | The owner must preserve every row. |\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "duplicate_assertion" || findings[0].Severity != SeverityFailure || findings[0].Target != "storage | The owner must preserve every row." {
		t.Fatalf("table headers should be ignored while repeated normative data rows still fail: %+v", findings)
	}
}

func TestAssertionFindingsDoNotTreatDataCatalogRowsAsPolicies(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/errors.md", "# API errors\n\n| Status | Code | Description |\n|---|---|---|\n| 400 | INVALID_DATE_FORMAT | Date must be YYYY-MM-DD format |\n| 400 | INVALID_DATE_RANGE | end_date must be after start_date |\n")

	if findings := assertionFindings([]string{path}, root); len(findings) != 0 {
		t.Fatalf("descriptive error catalog rows should not be treated as repeated normative claims: %+v", findings)
	}
}

func TestAssertionFindingsRejectGeneratedBlockSourceOrContentDrift(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, ".vrooli", "program-runtime")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "sample.json")
	if err := os.WriteFile(source, []byte(`{"name":"sample"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	block := "| Program | purpose |\n|---|---|\n| sample | derived |"
	marker := generatedTableMarker(t, "sample.json", block)
	path := writeAssertionFixture(t, root, "docs/generated.md", marker+"\n"+block+"\n"+generatedBlockEnd+"\n")

	if findings := assertionFindings([]string{path}, root); len(findings) != 0 {
		t.Fatalf("current generated source and content should pass: %+v", findings)
	}
	if err := os.WriteFile(source, []byte(`{"name":"changed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "stale_generated_block" || findings[0].Severity != SeverityFailure {
		t.Fatalf("changed owner input should fail generated-block health: %+v", findings)
	}

	if err := os.WriteFile(source, []byte(`{"name":"sample"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(marker+"\n"+block+" changed\n"+generatedBlockEnd+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings = assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "stale_generated_block" || findings[0].Severity != SeverityFailure {
		t.Fatalf("hand-edited generated content should fail generated-block health: %+v", findings)
	}
}

func TestAssertionFindingsIgnoreGeneratedMarkerExamplesInCodeFences(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/markers.md", "```html\n<!-- BEGIN GENERATED PROGRAM TABLE source=../../outside source_sha256=bad block_sha256=bad -->\n<!-- END GENERATED PROGRAM TABLE -->\n```\n")
	if findings := assertionFindings([]string{path}, root); len(findings) != 0 {
		t.Fatalf("code-fenced marker examples are not generated content: %+v", findings)
	}
}

func TestAssertionFindingsIgnoreOtherOwnersGeneratedBlocks(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/ledger.md", "<!-- BEGIN GENERATED LEDGER -->\nowner content\n<!-- END GENERATED LEDGER -->\n")
	if findings := assertionFindings([]string{path}, root); len(findings) != 0 {
		t.Fatalf("unrelated owner marker formats stay under their current validators: %+v", findings)
	}
}

func TestAssertionFindingsRejectGeneratedSourceEscapeAndSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "owner.json")
	if err := os.WriteFile(outsideFile, []byte(`{"name":"owner"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	linkedDir := filepath.Join(root, "linked")
	if err := os.MkdirAll(linkedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(linkedDir, "owner.json")); err != nil {
		t.Fatal(err)
	}
	block := "| owner |\n|---|\n| sample |"
	blockHash := sha256.Sum256([]byte(block))
	for _, source := range []string{"../outside/*.json", "linked/*.json"} {
		marker := fmt.Sprintf("<!-- BEGIN GENERATED PROGRAM TABLE source=%s source_sha256=%s block_sha256=%x -->", source, strings.Repeat("0", 64), blockHash)
		path := writeAssertionFixture(t, root, "docs/generated.md", marker+"\n"+block+"\n"+generatedBlockEnd+"\n")
		findings := assertionFindings([]string{path}, root)
		if len(findings) != 1 || findings[0].Code != "stale_generated_block" || findings[0].Severity != SeverityFailure {
			t.Fatalf("unsafe generated source %q must fail closed: %+v", source, findings)
		}
	}
}

func generatedTableMarker(t *testing.T, sourceName, block string) string {
	t.Helper()
	source := []byte(`{"name":"sample"}`)
	inputDigest := sha256.New()
	_, _ = inputDigest.Write([]byte(sourceName))
	_, _ = inputDigest.Write([]byte{0})
	_, _ = inputDigest.Write(source)
	_, _ = inputDigest.Write([]byte{0})
	blockDigest := sha256.Sum256([]byte(block))
	return fmt.Sprintf("<!-- BEGIN GENERATED PROGRAM TABLE source=.vrooli/program-runtime/*.json source_sha256=%x block_sha256=%x -->", inputDigest.Sum(nil), blockDigest)
}

func TestAssertionFindingsIgnoreBoldPolicySectionLabels(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/review.md", "# Review\n\n**Must not produce:**\n\n- A workaround that hides the failure.\n\n**Must NOT produce:**\n\n- An assertion without evidence.\n")

	if findings := assertionFindings([]string{path}, root); len(findings) != 0 {
		t.Fatalf("bold policy section labels are not assertions: %+v", findings)
	}
}

func TestAssertionFindingsRejectGenericPurposeHeadingAndIgnoreCodeExamples(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/guide.md", "## Purpose Of This Document\n\n```text\nThe source MUST be verified.\n```\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "boilerplate_purpose_heading" || findings[0].Severity != SeverityFailure {
		t.Fatalf("expected only the generic heading finding; code sample is not an assertion: %+v", findings)
	}
}
