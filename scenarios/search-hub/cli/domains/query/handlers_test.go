package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/structpb"
)

type showAllResultsContext struct {
	cliapp.OperationContext
	enabled bool
}

func (c showAllResultsContext) BoolFlag(name string) bool {
	return name == "show-all-results" && c.enabled
}

func (showAllResultsContext) Positional(string) string { return "" }

func TestEmptyQueryReportDistinguishesCoverageFromNoMatch(t *testing.T) {
	h := &handlers{}
	clean := h.queryReport(nil, &routingv1.QueryResponse{})
	degraded := h.queryReport(nil, &routingv1.QueryResponse{Degraded: true})
	if !strings.Contains(strings.Join(clean.RetrievalHints, " "), "broader capability phrase") {
		t.Fatalf("clean no-match guidance = %v", clean.RetrievalHints)
	}
	if !strings.Contains(strings.Join(degraded.RetrievalHints, " "), "Coverage was incomplete") {
		t.Fatalf("degraded no-match guidance = %v", degraded.RetrievalHints)
	}
}

func TestWeakEvidenceHintUsesTheVisibleSelectedView(t *testing.T) {
	weakRanked := &routingv1.SearchHit{
		ProviderId: "catalog.docs", Id: "weak-ranked", Title: "Weak ranked hit",
		Confidence: &commonv1.Confidence{Weak: true, Regime: "fused"},
	}
	strongPartHit := &routingv1.SearchHit{
		ProviderId: "ui-health.surfaces", Id: "strong-part", Title: "Health tracker surface",
		Confidence: &commonv1.Confidence{Regime: "fused"},
	}
	weakPartHit := &routingv1.SearchHit{
		ProviderId: "ui-health.surfaces", Id: "weak-part", Title: "Weak health tracker surface",
		Confidence: &commonv1.Confidence{Weak: true, Regime: "fused"},
	}
	response := func(partHit, rankedHit *routingv1.SearchHit) *routingv1.QueryResponse {
		return &routingv1.QueryResponse{
			Reranked: true,
			Ranked:   []*routingv1.SearchHit{rankedHit},
			Groups: []*routingv1.ProviderResultGroup{{
				ProviderId: partHit.GetProviderId(), Hits: []*routingv1.SearchHit{partHit},
			}},
			QueryParts: []*routingv1.QueryPartEvidence{{
				PartId: "surface", Label: "Find the health tracker surface",
				Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
				Evidence: []*routingv1.SearchHitReference{{
					ProviderId: partHit.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: partHit.GetId(),
				}},
			}},
		}
	}
	weakHint := "The visible matches are weak."
	strongVisible := (&handlers{}).queryReport(nil, response(strongPartHit, weakRanked)).RetrievalHints
	if strings.Contains(strings.Join(strongVisible, " "), weakHint) {
		t.Fatalf("hidden weak ranked hit caused a warning despite strong visible task evidence: %v", strongVisible)
	}
	weakVisible := (&handlers{}).queryReport(nil, response(weakPartHit, &routingv1.SearchHit{
		ProviderId: "catalog.docs", Id: "strong-ranked", Title: "Strong ranked hit",
		Confidence: &commonv1.Confidence{Regime: "fused"},
	})).RetrievalHints
	if !strings.Contains(strings.Join(weakVisible, " "), weakHint) {
		t.Fatalf("weak visible task evidence was not called out: %v", weakVisible)
	}
}

func TestInferredIntentDoesNotMasqueradeAsTaskCoverage(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{
		"search_hub_intent_match": "Diagnose and recover an Agent Manager runner transcript after a crash",
	})
	if err != nil {
		t.Fatal(err)
	}
	hit := &routingv1.SearchHit{ProviderId: "content-desk.editorial-history", Id: "draft-1", Title: "marketing eval essay draft", Metadata: metadata}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{hit}}
	report := (&handlers{}).queryReport(nil, msg)
	if strings.Contains(report.ResultsHeading, "Task evidence") {
		t.Fatalf("inferred routing metadata was presented as task coverage: %q", report.ResultsHeading)
	}
	if report.ResultsHeading != "Unified ranking (reranked across providers; ordered by rerank_score)" {
		t.Fatalf("heading = %q, want honest ranked view", report.ResultsHeading)
	}
}

func TestMachineJSONConnectFieldNamesAreOptInAndConsumerSpecific(t *testing.T) {
	call := func(cliapp.OperationContext) (*routingv1.QueryResponse, error) {
		return &routingv1.QueryResponse{CorporaSearched: []string{"docs"}}, nil
	}
	report := func(cliapp.OperationContext, *routingv1.QueryResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"human report"}}
	}
	handler := cliapp.ProtoListWithConnectJSONFlag(call, report, "json-connect")
	render := func(connectNames bool) string {
		var out strings.Builder
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema:    cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "json-connect", Bool: true}}},
			JSON:      true,
			BoolFlags: map[string]bool{"json-connect": connectNames},
			Stdout:    &out,
		})
		if err := handler.Run(ctx); err != nil {
			t.Fatalf("render connectNames=%t: %v", connectNames, err)
		}
		return out.String()
	}
	legacy := render(false)
	if !strings.Contains(legacy, `"corpora_searched"`) || strings.Contains(legacy, `"corporaSearched"`) {
		t.Fatalf("plain --json changed its legacy field-name contract: %s", legacy)
	}
	connect := render(true)
	if !strings.Contains(connect, `"corporaSearched"`) || strings.Contains(connect, `"corpora_searched"`) {
		t.Fatalf("--json-connect did not match Connect ProtoJSON field names: %s", connect)
	}
}

func TestOperationHintReadsBundleWithoutExecutingRetrievedCode(t *testing.T) {
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{
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
	msg.Ranked = []*routingv1.SearchHit{
		{ProviderId: "cli-health.commands", Id: "device-control device screenshot"},
		{ProviderId: "program-runtime.library", Id: "device-control.do-task"},
	}
	if hint := operationHints(msg); len(hint) != 0 {
		t.Fatalf("lower-ranked program received a default suggestion: %v", hint)
	}
}

func TestOperationHintPrintsLibraryUsage(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{"usage": float64(4)})
	if err != nil {
		t.Fatal(err)
	}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{{
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
	msg := &routingv1.QueryResponse{
		Reranked:  true,
		OrderedBy: "rerank_score",
		Ranked:    []*routingv1.SearchHit{hit},
		Groups: []*routingv1.ProviderResultGroup{{
			ProviderId: "agent-manager.runs",
			Count:      1,
			Hits:       []*routingv1.SearchHit{hit}, // Simulate a pre-projection response.
		}},
	}
	report := (&handlers{}).queryReport(nil, msg)
	joined := strings.Join(report.Results, "\n")
	if strings.Count(joined, "A distinctive result snippet") != 1 {
		t.Fatalf("result snippet appears more than once in report:\n%s", joined)
	}
	if strings.Contains(joined, "agent-manager.runs (1)") {
		t.Fatalf("default output includes the provider inventory:\n%s", joined)
	}
	detailed := (&handlers{}).queryReport(showAllResultsContext{enabled: true}, msg)
	if !strings.Contains(strings.Join(detailed.Results, "\n"), "agent-manager.runs (1)") {
		t.Fatalf("detailed output lost provider provenance: %v", detailed.Results)
	}
	if !strings.Contains(strings.Join(report.Summary, "\n"), "1 provider-reported hit(s)") {
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
	if !strings.Contains(strings.Join(report.Summary, "\n"), "1 provider-reported hit(s)") {
		t.Fatalf("provider count is missing from summary: %v", report.Summary)
	}
}

func TestQueryReportCapsHumanOutputButPreservesOptInAndProvenance(t *testing.T) {
	hits := make([]*routingv1.SearchHit, 12)
	for i := range hits {
		hits[i] = &routingv1.SearchHit{
			Id: fmt.Sprintf("result-%d", i), Title: "Result", Snippet: "unique snippet",
			ProviderGroup: "docs", Path: fmt.Sprintf("docs/result-%d.md", i),
		}
	}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "docs.provider", Count: int32(len(hits)), Hits: hits,
	}}}

	defaultReport := (&handlers{}).queryReport(nil, msg)
	defaultOutput := strings.Join(defaultReport.Results, "\n")
	if strings.Count(defaultOutput, "unique snippet") != defaultVisibleHits {
		t.Fatalf("default output showed %d hits, want %d", strings.Count(defaultOutput, "unique snippet"), defaultVisibleHits)
	}
	if !strings.Contains(defaultOutput, "8 additional unique returned hit(s) omitted") || !strings.Contains(defaultOutput, "docs.provider (12)") || !strings.Contains(defaultOutput, "docs/result") {
		t.Fatalf("default output lost omission count or provenance:\n%s", defaultOutput)
	}

	fullReport := (&handlers{}).queryReport(showAllResultsContext{enabled: true}, msg)
	if got := strings.Count(strings.Join(fullReport.Results, "\n"), "unique snippet"); got != len(hits) {
		t.Fatalf("--show-all-results rendered %d hits, want %d", got, len(hits))
	}
}

func TestQueryReportCapsUntrustedHitFieldsAndRankedRows(t *testing.T) {
	longValue := strings.Repeat("x", 10000)
	hits := make([]*routingv1.SearchHit, defaultVisibleHits+1)
	for i := range hits {
		hits[i] = &routingv1.SearchHit{Id: longValue + fmt.Sprint(i), Title: longValue}
	}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: hits}
	report := (&handlers{}).queryReport(nil, msg)
	output := strings.Join(report.Results, "\n")
	if len(output) > defaultVisibleHits*500 {
		t.Fatalf("bounded ranked output unexpectedly large: %d bytes", len(output))
	}
	if !strings.Contains(output, "1 additional unique returned hit(s) omitted") {
		t.Fatalf("ranked output did not report omitted hits:\n%s", output)
	}
}

// A provider that lost one ranking leg still answered; its hits are shown
// under the degradation note instead of being hidden.
func TestDegradedProviderWithHitsStillShowsThem(t *testing.T) {
	lines := renderGroups([]*routingv1.ProviderResultGroup{
		{ProviderId: "source-ledger.research", Count: 1, Degraded: true, Note: "lexical ranking: vectors warming", Hits: []*routingv1.SearchHit{{Id: "MAP.md", Title: "MAP.md#L3-L5", Snippet: "planner owns week view"}}},
		{ProviderId: "prompt-manager.skills", Degraded: true, Note: "service unavailable"},
		{ProviderId: "ui-health.surfaces", Degraded: true, Degradations: []*routingv1.ProviderDegradation{{Leg: "semantic", Reason: "unavailable", Detail: "vector store down"}}},
	})
	got := strings.Join(lines, "\n")
	for _, want := range []string{"source-ledger.research (1) — degraded: lexical ranking", "planner owns week view", "prompt-manager.skills (0) — degraded: service unavailable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderGroups() = %q, missing %q", got, want)
		}
	}
	if !strings.Contains(got, "ui-health.surfaces (0) — degraded: semantic: unavailable: vector store down") {
		t.Fatalf("structured provider degradation was not shown beside its group:\n%s", got)
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
