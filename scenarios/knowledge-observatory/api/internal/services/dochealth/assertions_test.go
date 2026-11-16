package dochealth

import (
	"os"
	"path/filepath"
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

func TestAssertionFindingsRejectGenericPurposeHeadingAndIgnoreCodeExamples(t *testing.T) {
	root := t.TempDir()
	path := writeAssertionFixture(t, root, "docs/guide.md", "## Purpose Of This Document\n\n```text\nThe source MUST be verified.\n```\n")

	findings := assertionFindings([]string{path}, root)
	if len(findings) != 1 || findings[0].Code != "boilerplate_purpose_heading" || findings[0].Severity != SeverityFailure {
		t.Fatalf("expected only the generic heading finding; code sample is not an assertion: %+v", findings)
	}
}
