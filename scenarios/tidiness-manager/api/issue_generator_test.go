package main

import (
	"strings"
	"testing"
)

func TestGenerateIssues_DoesNotEmitPercentageDuplicationLabels(t *testing.T) {
	dup := 100.0
	issues := GenerateIssuesFromMetrics("test-scenario", []DetailedFileMetrics{{FilePath: "api/handler.go", DuplicationPct: &dup}}, DefaultIssueGeneratorConfig())
	for _, issue := range issues {
		if issue.Category == "duplication" || strings.Contains(issue.Message, "duplicated code") {
			t.Fatalf("percentage duplication label survived: %#v", issue)
		}
	}
}
