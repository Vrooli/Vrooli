package cliapp

import (
	"strings"
	"testing"
)

func TestRenderListReportRejectsMissingRowsForNonEmptyList(t *testing.T) {
	var out strings.Builder
	err := RenderListReport(&out, ListReport{
		Summary:     []string{"3 items."},
		ListShaped:  true,
		ResultCount: 3,
	})
	if err == nil {
		t.Fatal("expected list-shaped report with omitted rows to fail")
	}
}

func TestRenderListReportAllowsSummaryOnlyReport(t *testing.T) {
	var out strings.Builder
	if err := RenderListReport(&out, ListReport{Summary: []string{"Session read."}}); err != nil {
		t.Fatalf("summary-only report should render: %v", err)
	}
}
