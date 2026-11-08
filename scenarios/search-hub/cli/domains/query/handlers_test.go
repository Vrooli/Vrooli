package query

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestOperationHintReadsBundleWithoutExecutingRetrievedCode(t *testing.T) {
	msg := &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{
		{ProviderId: "program-runtime.library", Id: "device-control.do-task"},
	}}
	hint := strings.Join(operationHints(msg), "\n")
	if !strings.Contains(hint, "library run program-runtime.prepare-operation --input name=device-control.do-task") {
		t.Fatal(hint)
	}
	for _, unsafe := range []string{"thing;echo secret", "thing.$(whoami)", "thing.name\nnext"} {
		msg.Ranked[0].Id = unsafe
		if len(operationHints(msg)) != 0 {
			t.Fatal("unsafe command suggestion", unsafe)
		}
	}
}

func TestOperationHintPrintsLibraryUsage(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{"usage": float64(4)})
	if err != nil {
		t.Fatal(err)
	}
	msg := &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
		ProviderId: "program-runtime.library",
		Id:         "program-runtime.setpoint-read",
		Metadata:   metadata,
	}}}
	hint := strings.Join(operationHints(msg), "\n")
	if !strings.Contains(hint, "recorded usage: 4") {
		t.Fatalf("operation hint = %q", hint)
	}
}

func TestFormatHitConfidenceAndLocations(t *testing.T) {
	hit := &routingv1.SearchHit{
		Id:            "plan-manager/authoring",
		Title:         "plan-manager/authoring",
		Snippet:       "Turns intent into plans.",
		ProviderGroup: "architecture-cartographer",
		Confidence:    &commonv1.Confidence{Weak: true, Regime: "fused"},
		Locations: []string{
			"scenarios/plan-manager/api/internal/authoring/",
			"scenarios/plan-manager/api/handlers/authoring/",
			"packages/proto/schemas/plan-manager/v1/authoring/",
		},
	}

	got := formatHit(1, hit)
	for _, want := range []string{
		"confidence=weak/fused",
		"architecture-cartographer/scenarios/plan-manager/api/internal/authoring/",
		"locations: scenarios/plan-manager/api/internal/authoring/, scenarios/plan-manager/api/handlers/authoring/ (+1 more)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatHit() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "score=") {
		t.Fatalf("formatHit() should not render raw score: %q", got)
	}
}

func TestRenderGroupsAllWeakNotice(t *testing.T) {
	lines := renderGroups([]*routingv1.ProviderResultGroup{{
		ProviderId: "architecture-cartographer.domain-map",
		Count:      1,
		Hits: []*routingv1.SearchHit{{
			Id:            "x",
			ProviderGroup: "architecture-cartographer",
			Confidence:    &commonv1.Confidence{Weak: true, Regime: "fused"},
		}},
	}})

	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "no confident match") {
		t.Fatalf("renderGroups() = %q, want no confident match notice", got)
	}
}

func TestDegradedSummaryNamesProvidersAndReasons(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{
		{ProviderId: "prompt-manager.skills", Degraded: true, Note: "service unavailable"},
		{ProviderId: "search-hub.records", Degraded: true, Note: "provider timeout"},
	}
	got := degradedSummary(groups, true)
	for _, want := range []string{"prompt-manager.skills: service unavailable", "search-hub.records: provider timeout"} {
		if !strings.Contains(got, want) {
			t.Fatalf("degradedSummary() = %q, missing %q", got, want)
		}
	}
	if got := degradedSummary(nil, true); !strings.Contains(got, "provider details unavailable") {
		t.Fatalf("degradedSummary() without details = %q", got)
	}
	if got := degradedSummary(groups, false); got != "" {
		t.Fatalf("degradedSummary() for healthy response = %q", got)
	}
}

func TestQueryReportPutsDegradationInFirstSummaryLine(t *testing.T) {
	msg := &routingv1.QueryResponse{Degraded: true, Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "prompt-manager.skills", Degraded: true, Note: "service unavailable",
	}}}
	report := (&handlers{}).queryReport(nil, msg)
	if len(report.Summary) == 0 || !strings.Contains(report.Summary[0], "prompt-manager.skills: service unavailable") {
		t.Fatalf("first summary line = %q", report.Summary)
	}
}

func TestRerankedQueryReportRendersHitPayloadOnceAndKeepsProviderSummary(t *testing.T) {
	hit := &routingv1.SearchHit{
		ProviderId: "agent-manager.runs",
		Id:         "run-1",
		Title:      "Agent run",
		Snippet:    "A distinctive result snippet",
	}
	report := (&handlers{}).queryReport(nil, &routingv1.QueryResponse{
		Reranked:  true,
		OrderedBy: "rerank_score",
		Ranked:    []*routingv1.SearchHit{hit},
		Groups: []*routingv1.ProviderResultGroup{{
			ProviderId: "agent-manager.runs",
			Count:      1,
			Hits:       []*routingv1.SearchHit{hit}, // Simulate a pre-projection response.
		}},
	})
	joined := strings.Join(report.Results, "\n")
	if strings.Count(joined, "A distinctive result snippet") != 1 {
		t.Fatalf("result snippet appears more than once in report:\n%s", joined)
	}
	if !strings.Contains(joined, "agent-manager.runs (1)") {
		t.Fatalf("provider summary is missing:\n%s", joined)
	}
	if !strings.Contains(strings.Join(report.Summary, "\n"), "1 total hit(s)") {
		t.Fatalf("provider hit count is missing from summary: %v", report.Summary)
	}
}

func TestGroupedQueryReportKeepsProviderHitDetails(t *testing.T) {
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "docs.provider",
		Count:      1,
		Hits: []*routingv1.SearchHit{{
			Id: "doc-1", Title: "Useful document", Snippet: "Grouped evidence remains visible",
		}},
	}}}
	report := (&handlers{}).queryReport(nil, msg)
	joined := strings.Join(report.Results, "\n")
	if !strings.Contains(joined, "Grouped evidence remains visible") {
		t.Fatalf("grouped hit evidence is missing:\n%s", joined)
	}
	if !strings.Contains(strings.Join(report.Summary, "\n"), "1 total hit(s)") {
		t.Fatalf("provider count is missing from summary: %v", report.Summary)
	}
}

func TestParseLimitUsesFastDefaultAndHonorsExplicitDepth(t *testing.T) {
	for _, raw := range []string{"", "  ", "invalid", "0", "-2"} {
		if got := parseLimit(raw); got != defaultQueryLimit {
			t.Errorf("parseLimit(%q) = %d, want default %d", raw, got, defaultQueryLimit)
		}
	}
	if got := parseLimit("10"); got != 10 {
		t.Fatalf("parseLimit(10) = %d, want explicit limit 10", got)
	}
}
