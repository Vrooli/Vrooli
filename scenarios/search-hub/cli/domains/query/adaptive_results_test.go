package query

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/structpb"
)

type queryTextContext struct {
	showAllResultsContext
	text string
}

func (c queryTextContext) Positional(name string) string {
	if name == "text" {
		return c.text
	}
	return ""
}

func TestTaskViewShowsPerPartEvidenceAndCoverage(t *testing.T) {
	hit := &routingv1.SearchHit{
		ProviderId: "ui-health.surfaces",
		Type:       "doc",
		Id:         "health-card",
		Title:      "HealthCard surface",
		Path:       "scenarios/ui-health/ui/src/features/health/HealthCard.tsx",
		Snippet:    "HealthCard renders health information.",
		Confidence: &commonv1.Confidence{Weak: true, Regime: "fused"},
	}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "ui-health.surfaces",
		Count:      1,
		Hits:       []*routingv1.SearchHit{hit},
	}}, QueryParts: []*routingv1.QueryPartEvidence{
		{
			PartId: "surface", Label: "Find the health tracking surface",
			Source:              routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
			SearchedProviderIds: []string{"ui-health.surfaces"},
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: "ui-health.surfaces", GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: "health-card",
			}},
			Status:   routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Degraded: true, Partial: true,
		},
		{
			PartId: "ios-design", Label: "Locate women's-health iOS design guidance",
			Source:              routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
			SearchedProviderIds: []string{"prompt-manager.skills"},
			Status:              routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY,
			Reason:              "no relevant skill hit",
		},
		{
			PartId: "research", Label: "Find women's-health research",
			Source:              routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
			SearchedProviderIds: []string{"source-ledger.research"},
			Status:              routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED,
			Reason:              "provider timeout",
		},
		{
			PartId: "card-use", Label: "Use the health card in the tracker",
			Source:              routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
			SearchedProviderIds: []string{"ui-health.surfaces"},
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: "ui-health.surfaces", GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: "health-card",
			}},
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
		},
	}}

	report := (&handlers{}).queryReport(nil, msg)
	got := strings.Join(report.Results, "\n")
	if report.ResultsHeading != "Task evidence by request part" {
		t.Fatalf("results heading = %q", report.ResultsHeading)
	}
	for _, expected := range []string{
		"Find the health tracking surface — candidate hits returned (relevance not judged); degraded part coverage; partial part coverage; searched ui-health.surfaces",
		"degraded part coverage; partial part coverage",
		"all referenced candidates are weak",
		"HealthCard surface [confidence=weak/fused ui-health.surfaces/",
		"searched; no candidates returned (derived segment)",
		"provider degraded; coverage incomplete (derived segment)",
		"provider timeout",
		"same candidate shown above: HealthCard surface (ui-health.surfaces)",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("task view missing %q:\n%s", expected, got)
		}
	}
}

func TestTaskPartLocalizesMatchingProviderDegradation(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "ui-health.surfaces", Id: "health-card", Title: "HealthCard surface"}
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{
			ProviderId: hit.GetProviderId(), Degraded: true, Note: "semantic leg timed out", Hits: []*routingv1.SearchHit{hit},
		}},
		QueryParts: []*routingv1.QueryPartEvidence{{
			PartId: "surface", Label: "Find the health tracking surface",
			SearchedProviderIds: []string{hit.GetProviderId()},
			Status:              routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: hit.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: hit.GetId(),
			}},
		}},
	}
	got := strings.Join((&handlers{}).queryReport(nil, msg).Results, "\n")
	if !strings.Contains(got, "degraded provider coverage ui-health.surfaces: semantic leg timed out") {
		t.Fatalf("task part did not localize degradation from its searched provider:\n%s", got)
	}
}

func TestTaskHitBudgetPreservesStatusForEveryPart(t *testing.T) {
	hits := make([]*routingv1.SearchHit, 5)
	for index := range hits {
		hits[index] = &routingv1.SearchHit{
			ProviderId: "ui-health.surfaces",
			Id:         fmt.Sprintf("surface-%d", index+1),
			Title:      fmt.Sprintf("Health surface %d", index+1),
		}
	}
	group := &routingv1.ProviderResultGroup{ProviderId: "ui-health.surfaces", Hits: hits}
	parts := make([]*routingv1.QueryPartEvidence, 0, 6)
	for index := 0; index < 4; index++ {
		parts = append(parts, &routingv1.QueryPartEvidence{
			PartId: fmt.Sprintf("covered-%d", index+1), Label: fmt.Sprintf("Covered request part %d", index+1),
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: group.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(int32(index)), HitId: hits[index].GetId(),
			}},
		})
	}
	parts = append(parts,
		&routingv1.QueryPartEvidence{PartId: "missing", Label: "Later missing requirement", Status: routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, Reason: "ambiguous_provider_evidence"},
		&routingv1.QueryPartEvidence{
			PartId: "over-budget", Label: "Later returned requirement", Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: group.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(4), HitId: hits[4].GetId(),
			}},
		},
	)
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{group}, QueryParts: parts}
	got := strings.Join(renderTaskParts(msg, defaultVisibleHits), "\n")
	for _, expected := range []string{
		"Later missing requirement — not routed",
		"multiple owners have similar evidence; owner is ambiguous",
		"Later returned requirement — candidate hits returned (relevance not judged)",
		"omitted by shared display budget",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("four-hit budget hid part status or omitted-evidence marker %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "additional request part(s) omitted") {
		t.Fatalf("part coverage was incorrectly capped with result hits:\n%s", got)
	}
}

func TestTaskPartShowsAllDistinctReferencedHitsWithinSharedBudget(t *testing.T) {
	hits := make([]*routingv1.SearchHit, 6)
	refs := make([]*routingv1.SearchHitReference, 6)
	for index := range hits {
		hits[index] = &routingv1.SearchHit{
			ProviderId: "catalog.docs", Id: fmt.Sprintf("candidate-%d", index+1),
			Title: fmt.Sprintf("Candidate evidence %d", index+1), Path: fmt.Sprintf("scenarios/catalog/evidence-%d.md", index+1),
		}
		refs[index] = &routingv1.SearchHitReference{
			ProviderId: "catalog.docs", GroupIndex: int32Ref(0), HitIndex: int32Ref(int32(index)), HitId: hits[index].GetId(),
		}
	}
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{ProviderId: "catalog.docs", Hits: hits}},
		QueryParts: []*routingv1.QueryPartEvidence{{
			PartId: "alternatives", Label: "Compare returned evidence candidates",
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, Evidence: refs,
		}},
	}
	limited := strings.Join(renderTaskParts(msg, defaultVisibleHits), "\n")
	if got := strings.Count(limited, "Candidate evidence "); got != defaultVisibleHits {
		t.Fatalf("default task view rendered %d distinct candidates, want %d:\n%s", got, defaultVisibleHits, limited)
	}
	if !strings.Contains(limited, "2 unique evidence hit(s) omitted by shared display budget") {
		t.Fatalf("task view omitted candidates without an exact per-part count:\n%s", limited)
	}
	full := strings.Join(renderTaskParts(msg, 0), "\n")
	if got := strings.Count(full, "Candidate evidence "); got != len(hits) {
		t.Fatalf("unbounded task view rendered %d candidates, want %d:\n%s", got, len(hits), full)
	}
	if strings.Contains(full, "omitted by shared display budget") {
		t.Fatalf("unbounded task view still omits supported candidates:\n%s", full)
	}
	if got := len(visibleResponseHits(queryTextContext{text: "task"}, msg)); got != defaultVisibleHits {
		t.Fatalf("handoff visible-hit set has %d hits, want the displayed budget %d", got, defaultVisibleHits)
	}
	if got := len(visibleResponseHits(queryTextContext{showAllResultsContext: showAllResultsContext{enabled: true}, text: "task"}, msg)); got != len(hits) {
		t.Fatalf("--show-all-results handoff visible-hit set has %d hits, want %d", got, len(hits))
	}
}

func TestDirectAndRankedHitsLocalizeTheirProviderDegradation(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "catalog.docs", Id: "health-card", Title: "Health tracker card", Path: "scenarios/catalog/health.md"}
	group := &routingv1.ProviderResultGroup{
		ProviderId: hit.GetProviderId(), Degraded: true, Note: "lexical fallback active", Hits: []*routingv1.SearchHit{hit},
	}
	ranked := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{hit}, Groups: []*routingv1.ProviderResultGroup{group}}
	rankedResults := strings.Join((&handlers{}).queryReport(nil, ranked).Results, "\n")
	if !strings.Contains(rankedResults, "provider coverage degraded: lexical fallback active") {
		t.Fatalf("ranked hit did not localize its provider degradation:\n%s", rankedResults)
	}
	directCtx := queryTextContext{text: hit.GetTitle()}
	direct := (&handlers{}).queryReport(directCtx, ranked)
	if direct.ResultsHeading != "Direct entity match" || !strings.Contains(strings.Join(direct.Results, "\n"), "provider coverage degraded: lexical fallback active") {
		t.Fatalf("direct view lost selected-owner degradation: %#v", direct)
	}
}

func TestTaskHandoffOnlyUsesVisiblePartEvidence(t *testing.T) {
	command := &routingv1.SearchHit{
		ProviderId: "cli-health.commands",
		Id:         "draft-begin",
		Title:      "react-component-library components draft-begin",
		Path:       "react-component-library components draft-begin",
	}
	unrelated := &routingv1.SearchHit{
		ProviderId: "ui-health.surfaces",
		Id:         "health-card",
		Title:      "HealthCard surface",
		Path:       "scenarios/ui-health/ui/src/HealthCard.tsx",
	}
	msg := &routingv1.QueryResponse{
		Reranked: true,
		Ranked:   []*routingv1.SearchHit{command},
		Groups: []*routingv1.ProviderResultGroup{{
			ProviderId: unrelated.GetProviderId(),
			Hits:       []*routingv1.SearchHit{unrelated},
		}},
		QueryParts: []*routingv1.QueryPartEvidence{{
			PartId: "surface", Label: "Find the health tracking surface",
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: unrelated.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: unrelated.GetId(),
			}},
		}},
	}
	ctx := queryTextContext{text: "Find the React component library draft workflow for fixing a typo"}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("handoff referenced a command hidden by the task view: %v", got)
	}
	visible := visibleResponseHits(ctx, msg)
	if len(visible) != 1 || visible[0] != unrelated {
		t.Fatalf("visible task evidence = %#v, want only the referenced part hit", visible)
	}
}

func TestCorrectionViewKeepsAmbiguousCandidatesGrounded(t *testing.T) {
	hit := &routingv1.SearchHit{
		ProviderId: "react-component-library.components",
		Type:       "doc",
		Id:         "deps-readme",
		Title:      "Component dependency declarations",
		Path:       "scenarios/react-component-library/README.md",
		Confidence: &commonv1.Confidence{Weak: true, Regime: "fused"},
	}
	ref := &routingv1.SearchHitReference{
		ProviderId: "react-component-library.components",
		GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: "deps-readme",
	}
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{ProviderId: hit.GetProviderId(), Hits: []*routingv1.SearchHit{hit}}},
		CorrectionCandidates: []*routingv1.CorrectionCandidate{
			{SubmittedValue: "everythign", CandidateValue: "everything", ProviderId: hit.GetProviderId(), Strength: routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE, Evidence: ref},
			{SubmittedValue: "everythign", CandidateValue: "everything else", ProviderId: hit.GetProviderId(), Strength: routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE, Evidence: ref},
		},
	}
	got := strings.Join((&handlers{}).queryReport(nil, msg).Results, "\n")
	for _, expected := range []string{"Possible entity corrections (not verified):", `"everythign" → "everything"`, `"everythign" → "everything else"`, "possible candidate, not verified", "ambiguous: 2 alternatives", "weak supporting evidence", "Component dependency declarations"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("correction view missing %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "Direct entity match") || strings.Contains(got, "Verified correction") {
		t.Fatalf("possible correction was presented as verified:\n%s", got)
	}
}

func TestCorrectionViewOmitsCandidatesWithoutReturnedHitEvidence(t *testing.T) {
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{ProviderId: "catalog.docs", Hits: []*routingv1.SearchHit{{ProviderId: "catalog.docs", Id: "unrelated", Title: "Unrelated result"}}}},
		CorrectionCandidates: []*routingv1.CorrectionCandidate{{
			SubmittedValue: "everythign", CandidateValue: "everything", ProviderId: "catalog.docs",
			Strength: routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE,
			Evidence: &routingv1.SearchHitReference{ProviderId: "catalog.docs", GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: "missing-id"},
		}},
	}
	got := strings.Join((&handlers{}).queryReport(nil, msg).Results, "\n")
	if strings.Contains(got, `"everythign" → "everything"`) || !strings.Contains(got, "1 correction candidate(s) omitted (unusable evidence or over display budget)") {
		t.Fatalf("unreferenced possibility was rendered or omission was unclear:\n%s", got)
	}
}

func TestCorrectionBudgetDoesNotHintFromAnOmittedSuggestion(t *testing.T) {
	first := &routingv1.SearchHit{ProviderId: "catalog.docs", Id: "first", Title: "Unrelated first evidence"}
	omitted := &routingv1.SearchHit{
		ProviderId: "cli-health.commands",
		Id:         "draft-begin",
		Title:      "react-component-library components draft-begin",
		Path:       "react-component-library components draft-begin",
	}
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{
			{ProviderId: first.GetProviderId(), Hits: []*routingv1.SearchHit{first}},
			{ProviderId: omitted.GetProviderId(), Hits: []*routingv1.SearchHit{omitted}},
		},
		CorrectionCandidates: []*routingv1.CorrectionCandidate{
			{SubmittedValue: "unrelated typo", CandidateValue: "unrelated term", Evidence: &routingv1.SearchHitReference{ProviderId: first.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: first.GetId()}},
			{SubmittedValue: "unrelated typo", CandidateValue: "another unrelated term", Evidence: &routingv1.SearchHitReference{ProviderId: first.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: first.GetId()}},
			{SubmittedValue: "unrelated typo", CandidateValue: "third unrelated term", Evidence: &routingv1.SearchHitReference{ProviderId: first.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: first.GetId()}},
			{SubmittedValue: "unrelated typo", CandidateValue: "fourth unrelated term", Evidence: &routingv1.SearchHitReference{ProviderId: first.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: first.GetId()}},
			{SubmittedValue: "React component library draft workflow typo", CandidateValue: "React component library draft workflow", ProviderId: omitted.GetProviderId(), Evidence: &routingv1.SearchHitReference{ProviderId: omitted.GetProviderId(), GroupIndex: int32Ref(1), HitIndex: int32Ref(0), HitId: omitted.GetId()}},
		},
	}
	ctx := queryTextContext{text: "Find React component library draft workflow typo"}
	report := (&handlers{}).queryReport(ctx, msg)
	results := strings.Join(report.Results, "\n")
	if strings.Contains(results, omitted.GetTitle()) || !strings.Contains(results, "1 correction candidate(s) omitted") {
		t.Fatalf("fixed-hit correction budget did not omit the fifth candidate: %s", results)
	}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("handoff depended on the omitted fifth correction candidate: %v", got)
	}
	visible := visibleResponseHits(ctx, msg)
	if len(visible) != 1 || visible[0] != first {
		t.Fatalf("visible correction evidence = %#v, want only the displayed first hit", visible)
	}
	showAll := queryTextContext{
		showAllResultsContext: showAllResultsContext{enabled: true},
		text:                  ctx.text,
	}
	fullReport := (&handlers{}).queryReport(showAll, msg)
	fullResults := strings.Join(fullReport.Results, "\n")
	if !strings.Contains(fullResults, omitted.GetTitle()) {
		t.Fatalf("--show-all-results did not expose the fifth correction candidate: %s", fullResults)
	}
	if hints := strings.Join(nextActionHints(showAll, msg), "\n"); !strings.Contains(hints, "react-component-library components draft-begin --help") {
		t.Fatalf("--show-all-results did not ground the visible fifth-candidate action: %s", hints)
	}
}

func TestTaskPartEvidencePrecedesOptionalCorrectionHintsUnderSharedBudget(t *testing.T) {
	hits := []*routingv1.SearchHit{
		{ProviderId: "docs", Id: "task-1", Title: "Primary evidence one"},
		{ProviderId: "docs", Id: "task-2", Title: "Primary evidence two"},
		{ProviderId: "docs", Id: "task-3", Title: "Primary evidence three"},
		{ProviderId: "docs", Id: "task-4", Title: "Primary evidence four"},
		{ProviderId: "cli-health.commands", Id: "draft-begin", Title: "react-component-library components draft-begin", Path: "react-component-library components draft-begin"},
	}
	parts := make([]*routingv1.QueryPartEvidence, 0, 4)
	for i := 0; i < 4; i++ {
		parts = append(parts, &routingv1.QueryPartEvidence{
			PartId: fmt.Sprintf("part-%d", i+1), Label: fmt.Sprintf("Primary request part %d", i+1),
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: "docs", GroupIndex: int32Ref(0), HitIndex: int32Ref(int32(i)), HitId: hits[i].GetId(),
			}},
		})
	}
	msg := &routingv1.QueryResponse{
		Groups:     []*routingv1.ProviderResultGroup{{ProviderId: "docs", Hits: hits[:4]}, {ProviderId: "cli-health.commands", Hits: hits[4:]}},
		QueryParts: parts,
		CorrectionCandidates: []*routingv1.CorrectionCandidate{{
			SubmittedValue: "React component library draft workflow typo",
			CandidateValue: "React component library draft workflow",
			ProviderId:     "cli-health.commands",
			Evidence:       &routingv1.SearchHitReference{ProviderId: "cli-health.commands", GroupIndex: int32Ref(1), HitIndex: int32Ref(0), HitId: "draft-begin"},
		}},
	}
	ctx := queryTextContext{text: "Find React component library draft workflow typo"}
	report := (&handlers{}).queryReport(ctx, msg)
	results := strings.Join(report.Results, "\n")
	for _, title := range []string{"Primary evidence one", "Primary evidence two", "Primary evidence three", "Primary evidence four"} {
		if !strings.Contains(results, title) {
			t.Errorf("task evidence was displaced by optional correction: %q\n%s", title, results)
		}
	}
	if strings.Contains(results, hits[4].GetTitle()) || !strings.Contains(results, "1 correction candidate(s) omitted") {
		t.Fatalf("correction did not yield the shared budget to task evidence: %s", results)
	}
	visible := visibleResponseHits(ctx, msg)
	if len(visible) != defaultVisibleHits {
		t.Fatalf("visible evidence count = %d, want %d", len(visible), defaultVisibleHits)
	}
	for _, hit := range visible {
		if hit == hits[4] {
			t.Fatal("optional correction candidate was treated as visible after task evidence filled the budget")
		}
	}
	if hints := nextActionHints(ctx, msg); len(hints) > 0 {
		t.Fatalf("handoff depended on a correction whose evidence was omitted: %v", hints)
	}
}

func TestCorrectionViewWithNoCandidatesUsesOrdinaryResults(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "catalog.docs", Id: "component-guide", Title: "Component guide"}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{ProviderId: hit.GetProviderId(), Hits: []*routingv1.SearchHit{hit}}}}
	report := (&handlers{}).queryReport(nil, msg)
	if report.ResultsHeading != "Results by provider" || !strings.Contains(strings.Join(report.Results, "\n"), "Component guide") {
		t.Fatalf("response without correction candidates changed its ordinary result view: %#v", report)
	}
}

func TestDerivedPartIdentifiesCandidateAlsoFoundForFullQuery(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "ui-health.surfaces", Id: "health-card", Title: "Health tracker card", Path: "scenarios/ui-health/ui/src/HealthCard.tsx"}
	msg := &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{ProviderId: hit.GetProviderId(), Hits: []*routingv1.SearchHit{hit}}},
		QueryParts: []*routingv1.QueryPartEvidence{{
			PartId: "tracker-card", Label: "Find the tracker card", Source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
			Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{ProviderId: hit.GetProviderId(), GroupIndex: int32Ref(0), HitIndex: int32Ref(0), HitId: hit.GetId(), RetrievalSources: []routingv1.SearchHitReference_RetrievalSource{
				routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT,
				routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY,
			}}},
		}},
	}
	got := strings.Join(renderTaskParts(msg, defaultVisibleHits), "\n")
	if !strings.Contains(got, "also found for full query") {
		t.Fatalf("task view omitted useful retrieval attribution:\n%s", got)
	}
}

func TestTaskViewExplainsRoutingAmbiguityReasons(t *testing.T) {
	msg := &routingv1.QueryResponse{QueryParts: []*routingv1.QueryPartEvidence{
		{PartId: "owner-tie", Label: "Choose the owning provider", Status: routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, Reason: "ambiguous_provider_evidence"},
		{PartId: "no-owner", Label: "Find a positive owner", Status: routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, Reason: "no_positive_provider_evidence"},
		{PartId: "boundary", Label: "Route at the top-k boundary", Status: routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, Reason: "ambiguous_lexical_boundary"},
	}}
	got := strings.Join(renderTaskParts(msg, defaultVisibleHits), "\n")
	for _, expected := range []string{
		"multiple owners have similar evidence; owner is ambiguous",
		"no provider returned positive evidence for this part",
		"routing stopped at a tied owner boundary; owner is ambiguous",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("task-part view missing %q:\n%s", expected, got)
		}
	}
}

func TestGroupedViewDeduplicatesSameSourceAcrossProviders(t *testing.T) {
	hit := &routingv1.SearchHit{Id: "same", Title: "Same source", Path: "scenarios/sample/README.md", Snippet: "one payload"}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{
		{ProviderId: "provider-a.docs", Count: 1, Hits: []*routingv1.SearchHit{hit}},
		{ProviderId: "provider-b.docs", Count: 1, Hits: []*routingv1.SearchHit{{Id: "same-copy", Title: "Same source", Path: "scenarios/sample/README.md", Snippet: "one payload"}}},
	}}
	got := strings.Join(renderGroups(msg.GetGroups()), "\n")
	if strings.Count(got, "one payload") != 1 {
		t.Fatalf("duplicate source payload was rendered more than once:\n%s", got)
	}
	if !strings.Contains(got, "provider-a.docs (1)") || !strings.Contains(got, "provider-b.docs (1)") {
		t.Fatalf("provider counts/provenance were lost:\n%s", got)
	}
}

func TestFixedHitRenderingUsesOneCompactGlobalBudget(t *testing.T) {
	const hitCount = 12
	hits := make([]*routingv1.SearchHit, hitCount)
	for index := range hits {
		hits[index] = &routingv1.SearchHit{
			ProviderId: "react-component-library.docs",
			Type:       "doc",
			Id:         fmt.Sprintf("dependency-check-%02d", index),
			Title:      fmt.Sprintf("Component dependency check and supported adopt workflow %02d", index),
			Snippet:    "The component dependency declarations are checked against the target scenario before adoption.",
			Path:       fmt.Sprintf("scenarios/react-component-library/requirements/dependency-%02d/module.json#L10-L32", index),
		}
	}
	groups := []*routingv1.ProviderResultGroup{
		{ProviderId: "react-component-library.docs", Count: 6, Hits: hits[:6]},
		{ProviderId: "business-health.intent", Count: 7, Hits: append([]*routingv1.SearchHit{hits[0]}, hits[6:]...)},
	}
	before := strings.Join(renderGroupsLimit(groups, 10), "\n")
	after := strings.Join(renderGroupsLimit(groups, defaultVisibleHits), "\n")
	if got := strings.Count(after, "The component dependency declarations"); got != defaultVisibleHits {
		t.Fatalf("candidate rendered %d hit snippets, want the shared budget %d:\n%s", got, defaultVisibleHits, after)
	}
	if strings.Count(after, "requirements/dependency-00/module.json") != 1 {
		t.Fatalf("candidate duplicated a cross-provider hit:\n%s", after)
	}
	if len(after)*4 > len(before)*3 {
		t.Fatalf("fixed-hit output shrank less than 25%%: baseline=%d bytes candidate=%d bytes", len(before), len(after))
	}
	t.Logf("fixed-hit result bytes: 10-row baseline=%d, 4-row candidate=%d, reduction=%.1f%%", len(before), len(after), 100*(1-float64(len(after))/float64(len(before))))
}

func TestTruncateKeepsUntrustedProviderTextValidUTF8(t *testing.T) {
	got := truncate(string([]byte{'a', 0xff, 'b', 'c'}), 3)
	if !utf8.ValidString(got) {
		t.Fatalf("truncated output is invalid UTF-8: %q", got)
	}
	if got != "a�b…" {
		t.Fatalf("truncated output = %q", got)
	}
}

func TestDirectEntityMatchRequiresExactIdentityAndNextActionIsReadable(t *testing.T) {
	hit := &routingv1.SearchHit{
		ProviderId: "react-component-library.components",
		Id:         "react-component-library/deps",
		Title:      "Component dependencies",
		Path:       "scenarios/react-component-library/requirements/10-dependency-compatibility-check/module.json#OT-P1-004",
	}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{ProviderId: hit.GetProviderId(), Hits: []*routingv1.SearchHit{hit}}}}
	ctx := queryTextContext{text: "react-component-library/deps"}
	report := (&handlers{}).queryReport(ctx, msg)
	if report.ResultsHeading != "Direct entity match" {
		t.Fatalf("results heading = %q", report.ResultsHeading)
	}
	if !strings.Contains(strings.Join(report.RetrievalHints, "\n"), "sed -n '1,140p' 'scenarios/react-component-library/requirements/10-dependency-compatibility-check/module.json'") {
		t.Fatalf("missing supported read command: %v", report.RetrievalHints)
	}
	if sourceReadCommand("../../etc/passwd") != "" || sourceReadCommand("scenarios/sample/evil'file.md") != "" || sourceReadCommand("/home/private.md") != "" {
		t.Fatal("read command accepted an unsafe or out-of-workspace path")
	}
}

func TestNextActionOpensSupportedCLIHelpWithoutExecutingMutation(t *testing.T) {
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "cli-health.commands",
		Hits: []*routingv1.SearchHit{{
			ProviderId: "cli-health.commands",
			Id:         "draft-begin",
			Title:      "react-component-library components draft-begin",
			Path:       "react-component-library components draft-begin",
		}},
	}}}
	ctx := queryTextContext{text: "fix the React component library typo and find the draft workflow"}
	got := strings.Join(nextActionHints(ctx, msg), "\n")
	if !strings.Contains(got, "react-component-library components draft-begin --help") {
		t.Fatalf("next action = %q", got)
	}
	msg.Groups[0].Hits[0].Path = "react-component-library components draft-begin; rm -rf /"
	if got := commandHelpSuggestion(msg.Groups[0].Hits[0]); got != "" {
		t.Fatalf("unsafe command suggestion = %q", got)
	}
}

func TestNextActionSkipsUnrelatedOwnerCommands(t *testing.T) {
	ctx := queryTextContext{text: "fix the component library typo and find the draft workflow"}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "business-health.intent",
		Hits: []*routingv1.SearchHit{{
			ProviderId: "business-health.intent",
			Id:         "workflow-migration",
			Title:      "TestGenie Workflow Migration",
			Path:       "scenarios/workflow-health/requirements/04-test-genie-workflow-migration/module.json",
		}},
	}}}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("unrelated result produced a next action: %v", got)
	}
}

func TestIOSOwnerCommandIsSkippedWithoutTaskSpecificOverlap(t *testing.T) {
	ctx := queryTextContext{text: "Build an iOS women's health tracking app and find the SwiftUI owner"}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "cli-health.commands",
		Hits: []*routingv1.SearchHit{{
			ProviderId: "cli-health.commands",
			Id:         "scenario-to-ios.status",
			Title:      "scenario-to-ios status",
			Path:       "scenario-to-ios status",
		}},
	}}}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("unrelated owner command produced a next action: %v", got)
	}
}

func TestInferredIntentDoesNotGroundOperationHint(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{"search_hub_intent_match": "Agent Manager runner transcript recovery after crash"})
	if err != nil {
		t.Fatal(err)
	}
	hit := &routingv1.SearchHit{
		ProviderId: "program-runtime.library",
		Id:         "agent-manager.efficiency-report",
		Title:      "agent-manager.efficiency-report",
		Snippet:    "Create an efficiency report.",
		Metadata:   metadata,
	}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{hit}}
	ctx := queryTextContext{text: "Diagnose an Agent Manager runner transcript failure after a crash; find the recovery procedure and next safe command to inspect status"}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("inferred intent grounded an unrelated operation: %v", got)
	}
}

func TestOperationActionSelectsBestRelevantVisibleProgramCandidate(t *testing.T) {
	ctx := queryTextContext{text: "Inspect cleanup manager program after removal"}
	unrelated := &routingv1.SearchHit{
		ProviderId: "program-runtime.library", Id: "health-manager.daily-summary", Title: "Health manager daily summary",
	}
	relevant := &routingv1.SearchHit{
		ProviderId: "program-runtime.library", Id: "cleanup-manager.inspect-after-removal", Title: "Inspect cleanup manager after removal",
	}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: []*routingv1.SearchHit{unrelated, relevant}}
	got := strings.Join(nextActionHints(ctx, msg), "\n")
	if !strings.Contains(got, "name=cleanup-manager.inspect-after-removal") {
		t.Fatalf("operation handoff did not select the best relevant visible candidate: %q", got)
	}
	if strings.Contains(got, unrelated.GetId()) {
		t.Fatalf("operation handoff selected the less-relevant first-ranked candidate: %q", got)
	}
}

func TestOperationActionNeverUsesCandidateOutsideVisibleBudget(t *testing.T) {
	ctx := queryTextContext{text: "Inspect cleanup manager program after removal"}
	hits := []*routingv1.SearchHit{
		{ProviderId: "catalog.docs", Id: "doc-1", Title: "Unrelated guide one"},
		{ProviderId: "catalog.docs", Id: "doc-2", Title: "Unrelated guide two"},
		{ProviderId: "catalog.docs", Id: "doc-3", Title: "Unrelated guide three"},
		{ProviderId: "catalog.docs", Id: "doc-4", Title: "Unrelated guide four"},
		{ProviderId: "program-runtime.library", Id: "cleanup-manager.inspect-after-removal", Title: "Inspect cleanup manager after removal"},
	}
	msg := &routingv1.QueryResponse{Reranked: true, Ranked: hits}
	if got := nextActionHints(ctx, msg); len(got) != 0 {
		t.Fatalf("hidden fifth hit grounded an action outside the four-hit view: %v", got)
	}
}

func TestCommandActionSelectsTheWorkflowSpecificOwnerCommand(t *testing.T) {
	ctx := queryTextContext{text: "Find the React component library draft workflow for fixing a typo"}
	msg := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "cli-health.commands",
		Hits: []*routingv1.SearchHit{
			{ProviderId: "cli-health.commands", Id: "ingest", Title: "react-component-library components ingest", Path: "react-component-library components ingest"},
			{ProviderId: "cli-health.commands", Id: "check", Title: "react-component-library components check", Path: "react-component-library components check"},
			{ProviderId: "cli-health.commands", Id: "draft-begin", Title: "react-component-library components draft-begin", Path: "react-component-library components draft-begin"},
		},
	}}}
	got := strings.Join(nextActionHints(ctx, msg), "\n")
	if !strings.Contains(got, "react-component-library components draft-begin --help") {
		t.Fatalf("next action did not select the task-specific workflow command: %q", got)
	}
}

func TestGlobalDegradationAndPartialStatusStayVisible(t *testing.T) {
	msg := &routingv1.QueryResponse{
		Groups:   []*routingv1.ProviderResultGroup{{ProviderId: "docs.provider", Count: 2}},
		Degraded: true, SelectorLeg: "lexical", OrderedBy: "score",
		Degradations: []*routingv1.SearchDegradation{{Component: "selector", Reason: "reranker_unavailable", FallbackActive: true}},
		Partial:      true, PendingProviders: 2,
	}
	got := strings.Join((&handlers{}).queryReport(nil, msg).Summary, " ")
	for _, expected := range []string{"selector: reranker_unavailable (fallback active)", "partial; 2 provider(s) pending"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("summary missing %q: %s", expected, got)
		}
	}
}

func TestStructuredDegradationIsRenderedWithoutLegacyBoolean(t *testing.T) {
	msg := &routingv1.QueryResponse{Degradations: []*routingv1.SearchDegradation{{
		Component: "selector", Reason: "selector_fallback", Detail: "used lexical fallback", FallbackActive: true,
	}}}
	got := responseDegradationSummary(msg)
	if !strings.Contains(got, "selector: selector_fallback: used lexical fallback (fallback active)") {
		t.Fatalf("structured degradation was omitted without `degraded=true`: %q", got)
	}
}

func int32Ref(value int32) *int32 { return &value }
