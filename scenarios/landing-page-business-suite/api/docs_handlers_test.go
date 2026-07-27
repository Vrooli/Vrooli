package main

import (
	"path/filepath"
	"testing"
)

func TestIsWithinDirectoryRejectsSiblingPrefixAndTraversal(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "docs")
	if !isWithinDirectory(root, filepath.Join(root, "guide.md")) {
		t.Fatal("expected file below docs root to be accepted")
	}
	for _, candidate := range []string{
		filepath.Join(string(filepath.Separator), "tmp", "docs-backup", "guide.md"),
		filepath.Join(root, "..", "secret.md"),
	} {
		if isWithinDirectory(root, candidate) {
			t.Errorf("isWithinDirectory(%q) accepted an escape", candidate)
		}
	}
}

func TestExtractTitleUsesFirstHeadingThenFilename(t *testing.T) {
	if got := extractTitle("intro\n# Operator Guide\n# Later", "guide.md"); got != "Operator Guide" {
		t.Fatalf("extractTitle heading = %q, want Operator Guide", got)
	}
	if got := extractTitle("no heading", "nested/runbook.md"); got != "runbook" {
		t.Fatalf("extractTitle fallback = %q, want runbook", got)
	}
}
