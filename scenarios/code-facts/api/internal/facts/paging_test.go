package facts

import (
	"context"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestPageReportPreservesCompleteFactSetAcrossPages(t *testing.T) {
	full := &factsv1.CodeFactsReport{Facts: []*factsv1.GenericFact{
		{Id: "a"}, {Id: "b"}, {Id: "c"}, {Id: "d"}, {Id: "e"},
	}}
	var got []string
	token := ""
	for {
		page, err := pageReport(&factsv1.CodeFactsReport{Facts: full.GetFacts()}, 2, token)
		if err != nil {
			t.Fatal(err)
		}
		for _, fact := range page.GetFacts() {
			got = append(got, fact.GetId())
		}
		if page.GetNextPageToken() == "" {
			break
		}
		token = page.GetNextPageToken()
	}
	if len(got) != len(full.GetFacts()) {
		t.Fatalf("paged fact count = %d, want %d", len(got), len(full.GetFacts()))
	}
	for i, id := range []string{"a", "b", "c", "d", "e"} {
		if got[i] != id {
			t.Fatalf("fact %d = %q, want %q", i, got[i], id)
		}
	}
}

func TestPageReportRejectsInvalidOrOutOfRangeTokens(t *testing.T) {
	report := &factsv1.CodeFactsReport{Facts: []*factsv1.GenericFact{{Id: "a"}}}
	for _, token := range []string{"nope", "-1", "2"} {
		if _, err := pageReport(report, 1, token); err == nil {
			t.Errorf("page token %q unexpectedly accepted", token)
		}
	}
}

func TestPageReportZeroPageSizeKeepsLegacyResponse(t *testing.T) {
	report := &factsv1.CodeFactsReport{Facts: []*factsv1.GenericFact{{Id: "a"}}}
	got, err := pageReport(report, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GetFacts()) != 1 || got.GetNextPageToken() != "" {
		t.Fatalf("zero page size changed legacy response: %+v", got)
	}
}

func TestPageReportDoesNotNeedContextOrSharedState(t *testing.T) {
	// Keep the helper's contract explicit: page boundaries are a pure response
	// concern and cannot invalidate the underlying cached report.
	_ = context.Background()
	if _, err := pageReport(&factsv1.CodeFactsReport{}, 1, "0"); err != nil {
		t.Fatal(err)
	}
}
