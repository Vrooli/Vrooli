package focus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"meta-optimization-manager/internal/trials"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/spacedoc"
	agentapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	agentdomain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// fakeSource is an in-memory GapSource.
type fakeSource struct {
	gaps []Gap
	err  error
}

func (f *fakeSource) DerivedGaps(_ context.Context) ([]Gap, error) { return f.gaps, f.err }

// fakeRepo is an in-memory Repository.
type fakeRepo struct {
	rows map[string]Gap
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]Gap{}} }

func (r *fakeRepo) List(_ context.Context) ([]Gap, error) {
	out := make([]Gap, 0, len(r.rows))
	for _, g := range r.rows {
		out = append(out, g)
	}
	return out, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (Gap, bool, error) {
	g, ok := r.rows[id]
	return g, ok, nil
}

func (r *fakeRepo) Upsert(_ context.Context, g Gap) error {
	r.rows[g.ID] = g
	return nil
}

type fakeProviderInsights struct {
	signals []ProviderInsight
	err     error
}

type fakeMaturityReader struct {
	observations []MaturityObservation
	err          error
}

func (f fakeMaturityReader) Maturity(context.Context) ([]MaturityObservation, error) {
	return f.observations, f.err
}

func (f fakeProviderInsights) Insights(context.Context) ([]ProviderInsight, error) {
	return f.signals, f.err
}

func derivedFixture() []Gap {
	return []Gap{
		{ID: "answer/1", Projection: ProjectionAnswer, Title: "explain domain map", Status: spacedoc.StatusMissing, SourceCellID: "1"},
		{ID: "validate/2", Projection: ProjectionValidate, Title: "verify perf", Status: spacedoc.StatusInReach, SourceCellID: "2"},
		{ID: "guide/3", Projection: ProjectionGuide, Title: "rename skill", Status: spacedoc.StatusInReach, SourceCellID: "3"},
	}
}

func newSvc(src GapSource, repo Repository) Service {
	return NewService(Deps{Source: src, Repo: repo})
}

func TestGetFocusRanksMissingAnswerHighest(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	result, err := svc.GetFocus(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("GetFocus: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("want 3 items, got %d", len(result.Items))
	}
	// answer/1 is MISSING (impact 1.0) × answer (importance 1.0) = 1.0 — top.
	if result.Items[0].Gap.ID != "answer/1" {
		t.Fatalf("want answer/1 ranked first, got %s (priority=%.3f)", result.Items[0].Gap.ID, result.Items[0].Priority)
	}
	if result.Items[0].Priority <= result.Items[1].Priority {
		t.Fatalf("expected descending priority, got %.3f then %.3f", result.Items[0].Priority, result.Items[1].Priority)
	}
	if result.Items[0].Rationale == "" {
		t.Fatalf("expected a rationale on the top item")
	}
}

func TestGetFocusLimitAndProjectionFilter(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	result, err := svc.GetFocus(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("GetFocus: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("limit=1 should cap to 1, got %d", len(result.Items))
	}

	guideResult, err := svc.GetFocus(context.Background(), 0, ProjectionGuide)
	if err != nil {
		t.Fatalf("GetFocus(guide): %v", err)
	}
	if len(guideResult.Items) != 1 || guideResult.Items[0].Gap.Projection != ProjectionGuide {
		t.Fatalf("projection filter failed: %+v", guideResult.Items)
	}
}

func TestGetFocusUnknownProjectionErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	if _, err := svc.GetFocus(context.Background(), 0, Projection("bogus")); err == nil {
		t.Fatalf("expected error on unknown projection")
	}
}

func TestListGapsFilters(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil || len(all) != 3 {
		t.Fatalf("ListGaps all: len=%d err=%v", len(all), err)
	}
	missing, err := svc.ListGaps(context.Background(), GapFilter{Status: spacedoc.StatusMissing})
	if err != nil || len(missing) != 1 || missing[0].ID != "answer/1" {
		t.Fatalf("status filter failed: %+v err=%v", missing, err)
	}
	byCell, err := svc.ListGaps(context.Background(), GapFilter{CellID: "2"})
	if err != nil || len(byCell) != 1 || byCell[0].ID != "validate/2" {
		t.Fatalf("cell filter failed: %+v err=%v", byCell, err)
	}
}

func TestAddGapNoteMaterializesAndAppends(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, repo)

	// First note materializes a registry row from the derived gap.
	g, err := svc.AddGapNote(context.Background(), "answer/1", "try a cartographer provider")
	if err != nil {
		t.Fatalf("AddGapNote: %v", err)
	}
	if len(g.Approaches) != 1 || g.Approaches[0] != "try a cartographer provider" {
		t.Fatalf("approach not appended: %+v", g.Approaches)
	}
	if _, ok := repo.rows["answer/1"]; !ok {
		t.Fatalf("expected registry row materialized for answer/1")
	}

	// Second note appends without clobbering; duplicates are de-duped.
	g, err = svc.AddGapNote(context.Background(), "answer/1", "or extend code-facts")
	if err != nil {
		t.Fatalf("AddGapNote 2: %v", err)
	}
	if len(g.Approaches) != 2 {
		t.Fatalf("want 2 approaches, got %+v", g.Approaches)
	}
	g, err = svc.AddGapNote(context.Background(), "answer/1", "or extend code-facts")
	if err != nil {
		t.Fatalf("AddGapNote dup: %v", err)
	}
	if len(g.Approaches) != 2 {
		t.Fatalf("duplicate approach should be de-duped, got %+v", g.Approaches)
	}
}

func TestAddGapNoteUnknownGapErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	if _, err := svc.AddGapNote(context.Background(), "answer/999", "x"); err == nil {
		t.Fatalf("expected error adding note to unknown gap")
	}
}

func TestAddGapNoteNilRepoErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, nil)
	if _, err := svc.AddGapNote(context.Background(), "answer/1", "x"); err == nil {
		t.Fatalf("expected error when registry unavailable")
	}
}

func TestRegistryOnlyGlobalGapSurfaces(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["global-typed-contracts"] = Gap{
		ID:         "global-typed-contracts",
		Title:      "typed contracts everywhere",
		Global:     true,
		Status:     spacedoc.StatusMissing,
		Approaches: []string{"ratchet"},
	}
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, repo)
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil {
		t.Fatalf("ListGaps: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("want 4 (3 derived + 1 global), got %d", len(all))
	}
	g, err := svc.GetGap(context.Background(), "global-typed-contracts")
	if err != nil || !g.Global {
		t.Fatalf("expected global gap, got %+v err=%v", g, err)
	}
}

func TestDegradesWhenSourceErrors(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["global-x"] = Gap{ID: "global-x", Title: "x", Global: true}
	svc := newSvc(&fakeSource{err: context.DeadlineExceeded}, repo)
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil {
		t.Fatalf("ListGaps should degrade, got err=%v", err)
	}
	if len(all) != 1 || all[0].ID != "global-x" {
		t.Fatalf("expected registry-only gap to survive a down source, got %+v", all)
	}
}

func TestMultiGapSourceKeepsHealthySourceWhenOneFails(t *testing.T) {
	healthy := &fakeSource{gaps: []Gap{{ID: "healthy", Axis: AxisEmpirical, Title: "healthy evidence"}}}
	failing := &fakeSource{err: errors.New("history unavailable")}
	source := NewMultiGapSource([]NamedGapSource{
		{Name: "trials", Source: failing},
		{Name: "coverage", Source: healthy},
	})
	gaps, err := source.DerivedGaps(context.Background())
	if err == nil {
		t.Fatal("multi source should propagate degradation so focus can mark the response")
	}
	if len(gaps) != 2 || gaps[0].ID != "source/trials/availability" || gaps[1].ID != "healthy" {
		t.Fatalf("expected failure status plus healthy gap, got %+v", gaps)
	}
	if gaps[0].AvailabilityReason == "" {
		t.Fatal("expected a stated source availability reason")
	}
}

func TestListConditionReturnsConditionEvidenceWhenSiblingSourceFails(t *testing.T) {
	condition := &fakeSource{gaps: []Gap{{ID: "condition/provider-a", Axis: AxisEmpirical, Title: "degraded"}}}
	svc := newSvc(NewMultiGapSource([]NamedGapSource{
		{Name: "agent-manager", Source: &fakeSource{err: errors.New("owner unavailable")}},
		{Name: "condition", Source: condition},
	}), newFakeRepo())
	gaps, err := svc.ListCondition(context.Background())
	if err != nil {
		t.Fatalf("ListCondition should preserve condition evidence: %v", err)
	}
	if len(gaps) != 1 || gaps[0].ID != "condition/provider-a" {
		t.Fatalf("condition gaps = %+v", gaps)
	}
}

type fakeTrialHistory struct {
	runs []trials.TrialRun
	err  error
}

func (f *fakeTrialHistory) Runs(_ context.Context, _ trials.RunFilter, _ int, _ bool) ([]trials.TrialRun, error) {
	return f.runs, f.err
}

func TestEmpiricalGapSourceEmitsSustainedTrialFailures(t *testing.T) {
	reader := &fakeTrialHistory{runs: []trials.TrialRun{
		{ID: "run-new", TaskID: "task-a", Verdict: trials.VerdictError, At: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "run-old", TaskID: "task-a", Verdict: trials.VerdictFail, At: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)},
		{ID: "run-pass", TaskID: "task-b", Verdict: trials.VerdictPass, At: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "run-fail", TaskID: "task-b", Verdict: trials.VerdictFail, At: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)},
	}}
	gaps, err := NewEmpiricalGapSource(reader).DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("want one sustained empirical gap, got %+v", gaps)
	}
	if gaps[0].Axis != AxisEmpirical || gaps[0].Recurrence != 2 || gaps[0].EvidenceLocator != "trial-task:task-a/run:run-new" {
		t.Fatalf("unexpected empirical provenance: %+v", gaps[0])
	}
}

func TestEmpiricalGapSourceEmptyHistoryIsHealthyAndEmpty(t *testing.T) {
	gaps, err := NewEmpiricalGapSource(&fakeTrialHistory{}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("empty history should not error: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("empty history should produce no empirical gaps, got %+v", gaps)
	}
}

func TestEmpiricalRankingUsesRecurrenceAndTraceability(t *testing.T) {
	coverageMissing := Gap{ID: "coverage-missing", Axis: AxisCoverage, Projection: ProjectionAnswer, Status: spacedoc.StatusMissing}
	coverageInReach := Gap{ID: "coverage-in-reach", Axis: AxisCoverage, Projection: ProjectionAnswer, Status: spacedoc.StatusInReach}
	single := Gap{ID: "empirical-single", Axis: AxisEmpirical, Recurrence: 1, EvidenceSource: "trials", EvidenceLocator: "trial-task:x/run:1"}
	high := Gap{ID: "empirical-high", Axis: AxisEmpirical, Recurrence: 5, EvidenceSource: "trials", EvidenceLocator: "trial-task:x/run:5"}
	untraceable := Gap{ID: "empirical-untraceable", Axis: AxisEmpirical, Recurrence: 5, EvidenceSource: "trials"}

	result, err := newSvc(&fakeSource{gaps: []Gap{coverageMissing, coverageInReach, single, high, untraceable}}, newFakeRepo()).GetFocus(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("GetFocus: %v", err)
	}
	position := make(map[string]int, len(result.Items))
	for i, item := range result.Items {
		position[item.Gap.ID] = i
	}
	if position["empirical-single"] <= position["coverage-missing"] {
		t.Fatalf("single empirical observation outranked coverage missing: %+v", position)
	}
	if position["empirical-high"] >= position["coverage-in-reach"] {
		t.Fatalf("high recurrence did not outrank coverage in-reach: %+v", position)
	}
	if position["empirical-untraceable"] <= position["empirical-single"] {
		t.Fatalf("untraceable empirical signal outranked traceable signal: %+v", position)
	}
}

func TestSpaceGapSourceLiveJoinDrivesStatuses(t *testing.T) {
	reader := fakeSpaceReader{defs: map[Projection]*spacedoc.SpaceDefinition{
		ProjectionAnswer: {Projection: ProjectionAnswer, Cells: []spacedoc.Cell{
			{ID: "1", Question: "covered", Owner: "hot", Status: spacedoc.StatusMissing},
			{ID: "2", Question: "still missing", Owner: "cold", Status: spacedoc.StatusMissing},
		}},
		ProjectionValidate: {Projection: ProjectionValidate},
		ProjectionGuide:    {Projection: ProjectionGuide},
		ProjectionAct:      {Projection: ProjectionAct},
	}}
	source := NewSpaceGapSourceWithLiveJoin(reader, func(context.Context, Projection, *spacedoc.SpaceDefinition) (map[string]spacedoc.CellStatus, error) {
		return map[string]spacedoc.CellStatus{"1": spacedoc.StatusNow}, nil
	})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 1 || gaps[0].SourceCellID != "2" || len(gaps[0].ProviderIDs) != 1 || gaps[0].ProviderIDs[0] != "cold" {
		t.Fatalf("live statuses/providers not applied: %+v", gaps)
	}
}

func TestFocusDegradedLiveJoinNeverLooksComputed(t *testing.T) {
	source := &fakeSource{gaps: []Gap{{ID: "answer/9", Projection: ProjectionAnswer, Status: spacedoc.StatusInReach, ProviderIDs: []string{"hot"}}}, err: errors.New("search-hub timeout")}
	result, err := NewService(Deps{Source: source, Repo: newFakeRepo(), Insights: fakeProviderInsights{signals: []ProviderInsight{{ProviderID: "hot", TimesRouted: 5}}}}).GetFocus(context.Background(), 10, ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Degraded || !strings.Contains(result.DegradedReason, "search-hub timeout") {
		t.Fatalf("missing honest degradation: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Gap.Status != spacedoc.StatusInReach {
		t.Fatalf("authored fallback was not retained as explicitly degraded data: %+v", result.Items)
	}
}

func TestFocusRankingUsesProviderTelemetry(t *testing.T) {
	source := &fakeSource{gaps: []Gap{
		{ID: "answer/cold", Projection: ProjectionAnswer, Status: spacedoc.StatusMissing, ProviderIDs: []string{"cold"}},
		{ID: "answer/hot", Projection: ProjectionAnswer, Status: spacedoc.StatusMissing, ProviderIDs: []string{"hot"}},
	}}
	result, err := NewService(Deps{Source: source, Repo: newFakeRepo(), Insights: fakeProviderInsights{signals: []ProviderInsight{
		{ProviderID: "cold"}, {ProviderID: "hot", TimesRouted: 10, DegradationRate: 0.5},
	}}}).GetFocus(context.Background(), 10, ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if result.Degraded || result.Items[0].Gap.ID != "answer/hot" || result.Items[0].Priority <= result.Items[1].Priority {
		t.Fatalf("provider telemetry did not affect ranking: %+v", result)
	}
}

func TestFocusRanksConditionDegradationAlongsideCoverage(t *testing.T) {
	condition := fakeProviderInsights{signals: []ProviderInsight{{
		ProviderID: "answer.provider", TimesRouted: 100, DegradationRate: 1,
	}}}
	coverage := &fakeSource{gaps: []Gap{{ID: "answer/missing", Axis: AxisCoverage, Projection: ProjectionAnswer, Status: spacedoc.StatusMissing, Title: "one missing cell"}}}
	svc := NewService(Deps{
		Source: NewMultiGapSource([]NamedGapSource{
			{Name: "coverage", Source: coverage},
			{Name: "condition", Source: NewConditionGapSource(condition)},
		}),
		Insights: condition,
	})
	result, err := svc.GetFocus(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) < 2 {
		t.Fatalf("items = %+v, want coverage and condition findings", result.Items)
	}
	if result.Items[0].Gap.Axis != AxisEmpirical {
		t.Fatalf("top item axis = %q, want condition empirical finding", result.Items[0].Gap.Axis)
	}
}

func TestMaturityGapSourceSurfacesBlockingEvidence(t *testing.T) {
	gaps, err := NewMaturityGapSource(fakeMaturityReader{observations: []MaturityObservation{
		{Scenario: "search-provider", BlockingCodes: []string{"SEARCH_EVAL_CORPUS_MISSING", "SEARCH_CONFIG_INVALID"}},
		{Scenario: "clean-provider"},
	}}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 || gaps[0].ID != "condition/maturity/search-provider" {
		t.Fatalf("gaps = %#v, want one maturity condition gap", gaps)
	}
	if got := gaps[0].Notes; len(got) != 1 || got[0] != "blocking=SEARCH_CONFIG_INVALID,SEARCH_EVAL_CORPUS_MISSING" {
		t.Fatalf("notes = %#v, want sorted blocking evidence", got)
	}
}

func TestFocusRankingFallsBackToDeterministicIDsWhenTelemetryUnavailable(t *testing.T) {
	source := &fakeSource{gaps: []Gap{
		{ID: "answer/b", Projection: ProjectionAnswer, Status: spacedoc.StatusMissing},
		{ID: "answer/a", Projection: ProjectionAnswer, Status: spacedoc.StatusMissing},
	}}
	deps := Deps{Source: source, Repo: newFakeRepo(), Insights: fakeProviderInsights{err: errors.New("search-hub down")}}
	first, err := NewService(deps).GetFocus(context.Background(), 10, ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewService(deps).GetFocus(context.Background(), 10, ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Degraded || first.DegradedReason == "" || len(first.Items) != 2 || first.Items[0].Gap.ID != "answer/a" || first.Items[1].Gap.ID != "answer/b" {
		t.Fatalf("unexpected deterministic fallback: %+v", first)
	}
	if first.Items[0].Gap.ID != second.Items[0].Gap.ID || first.Items[1].Gap.ID != second.Items[1].Gap.ID {
		t.Fatalf("identical calls changed order: first=%+v second=%+v", first.Items, second.Items)
	}
}

type fakeSpaceReader struct {
	defs map[Projection]*spacedoc.SpaceDefinition
}

func (f fakeSpaceReader) Read(_ context.Context, p Projection) (*spacedoc.SpaceDefinition, error) {
	def, ok := f.defs[p]
	if !ok {
		return nil, errors.New("projection unavailable")
	}
	return def, nil
}

type fakeAgentFindingReader struct {
	report AgentFindingReport
	err    error
}

func (f *fakeAgentFindingReader) ReadFindings(_ context.Context) (AgentFindingReport, error) {
	return f.report, f.err
}

func TestAgentManagerGapSourceClustersAndReportsDroppedEvidence(t *testing.T) {
	gaps, err := NewAgentManagerGapSource(&fakeAgentFindingReader{report: AgentFindingReport{
		Findings: []AgentFindingObservation{
			{RunID: "run-2", Fingerprint: "fp-1", Pattern: "retry", OwnerScenario: "meta-optimization-manager", OwnerConfidence: "manifest-derived", EvidenceLocator: "agent-manager://runs/run-2/episodes/e-2"},
			{RunID: "run-1", Fingerprint: "fp-1", Pattern: "retry", OwnerScenario: "meta-optimization-manager", OwnerConfidence: "manifest-derived", EvidenceLocator: "agent-manager://runs/run-1/episodes/e-1"},
		},
		DroppedUnattributed: 3,
	}}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 2 {
		t.Fatalf("want clustered finding plus dropped-count status, got %+v", gaps)
	}
	if gaps[0].Axis != AxisEmpirical || gaps[0].Recurrence != 2 || gaps[0].EvidenceSource != "agent-manager" {
		t.Fatalf("unexpected finding gap: %+v", gaps[0])
	}
	if gaps[1].AvailabilityReason == "" || !strings.Contains(gaps[1].AvailabilityReason, "dropped 3") {
		t.Fatalf("dropped attribution count not visible: %+v", gaps[1])
	}
}

type fakeScenarioResolver struct {
	url string
	err error
}

func (f fakeScenarioResolver) ResolveScenarioURLDefault(_ context.Context, _ string) (string, error) {
	return f.url, f.err
}

type fakeAgentRunsClient struct {
	runs []*agentdomain.Run
	err  error
}

func (f fakeAgentRunsClient) ListRuns(_ context.Context, _ *connect.Request[agentapi.ListRunsRequest]) (*connect.Response[agentapi.ListRunsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&agentapi.ListRunsResponse{Runs: f.runs}), nil
}

type fakeAgentEpisodesClient struct {
	episodes map[string][]*agentdomain.FrictionEpisode
	err      error
}

func (f fakeAgentEpisodesClient) GetEpisodes(_ context.Context, req *connect.Request[agentdomain.GetEpisodesRequest]) (*connect.Response[agentdomain.GetEpisodesResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&agentdomain.GetEpisodesResponse{Episodes: f.episodes[req.Msg.GetRunId()]}), nil
}

type fakeAgentClientFactory struct {
	runs     agentManagerRunClient
	episodes agentManagerEpisodeClient
}

func (f fakeAgentClientFactory) New(_ connect.HTTPClient, _ string) (agentManagerRunClient, agentManagerEpisodeClient) {
	return f.runs, f.episodes
}

func TestAgentManagerFindingReaderUsesTypedClientsAndDropsUnknownOwners(t *testing.T) {
	runs := fakeAgentRunsClient{runs: []*agentdomain.Run{
		{Id: "run-1"},
		{Id: "run-2"},
	}}
	episodes := fakeAgentEpisodesClient{episodes: map[string][]*agentdomain.FrictionEpisode{
		"run-1": {{EpisodeId: "e-1", Fingerprint: "fp", SuspectedOwnerScenario: "meta-optimization-manager", OwnerConfidence: "manifest-derived"}},
		"run-2": {{EpisodeId: "e-2", Fingerprint: "fp", SuspectedOwnerScenario: "", OwnerConfidence: "unknown"}},
	}}
	reader := newAgentManagerFindingReader(fakeScenarioResolver{url: "http://agent-manager"}, nil, fakeAgentClientFactory{runs: runs, episodes: episodes}, time.Second)
	report, err := reader.ReadFindings(context.Background())
	if err != nil {
		t.Fatalf("ReadFindings: %v", err)
	}
	if len(report.Findings) != 1 || report.DroppedUnattributed != 1 {
		t.Fatalf("unexpected typed reader report: %+v", report)
	}
}
