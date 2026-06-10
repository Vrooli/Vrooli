package research_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"web-search/internal/findings"
	"web-search/internal/research"

	"github.com/stretchr/testify/require"
)

// fakeGatherer is a FindingsGatherer that records the limit it was asked for and
// returns a canned set, optionally OVER-returning to prove the service truncates
// to the cap defensively.
type fakeGatherer struct {
	gotQuery string
	gotLimit int
	returnN  int // how many findings to return regardless of limit
	err      error
}

func (f *fakeGatherer) Gather(_ context.Context, query string, limit int) ([]research.GatheredFinding, error) {
	f.gotQuery = query
	f.gotLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	out := make([]research.GatheredFinding, 0, f.returnN)
	for i := 0; i < f.returnN; i++ {
		out = append(out, research.GatheredFinding{FindingID: "f", Claim: "c", Status: findings.StatusActive})
	}
	return out, nil
}

// TestGatherRelatedFindingsEnforcesBound is the OT-P1-003 bounded-sweep contract:
// an unset max defaults to 20, a too-large max is clamped to 20, and a
// well-behaved smaller max is honored. The seam is always asked for the clamped
// limit, never the raw request.
func TestGatherRelatedFindingsEnforcesBound(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		reqMax    int
		wantLimit int
	}{
		{"unset defaults to cap", 0, research.MaxGatherFindings},
		{"negative defaults to cap", -5, research.MaxGatherFindings},
		{"too-large clamps to cap", 1000, research.MaxGatherFindings},
		{"exactly cap honored", research.MaxGatherFindings, research.MaxGatherFindings},
		{"smaller honored", 3, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &fakeGatherer{returnN: 0}
			svc := research.NewService(research.Deps{Gatherer: g})
			_, capApplied, err := svc.GatherRelatedFindings(ctx, "what is X", tc.reqMax)
			require.NoError(t, err)
			require.Equal(t, tc.wantLimit, g.gotLimit, "seam must be asked for the clamped limit")
			require.Equal(t, tc.wantLimit, capApplied, "applied cap must equal the clamped limit")
			require.Equal(t, "what is X", g.gotQuery)
		})
	}
}

// TestGatherRelatedFindingsTruncatesOverReturn proves the cap is enforced even
// if a misbehaving seam returns MORE than the requested limit: the service
// truncates the result to the bound so the sweep can never widen.
func TestGatherRelatedFindingsTruncatesOverReturn(t *testing.T) {
	g := &fakeGatherer{returnN: 100} // seam ignores the limit and floods
	svc := research.NewService(research.Deps{Gatherer: g})
	out, capApplied, err := svc.GatherRelatedFindings(context.Background(), "q", 5)
	require.NoError(t, err)
	require.Equal(t, 5, capApplied)
	require.Len(t, out, 5, "result is truncated to the bound regardless of seam over-return")
}

// TestGatherRelatedFindingsConfigurableCap asserts the sweep bound is
// configuration-driven (Deps.GatherCap, fed by WEB_SEARCH_MAX_GATHER_FINDINGS):
// a configured cap tightens both the default and the clamp, and an invalid cap
// falls back to the compiled default of 20.
func TestGatherRelatedFindingsConfigurableCap(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		cap       int
		reqMax    int
		wantLimit int
	}{
		{"configured cap bounds the default", 5, 0, 5},
		{"configured cap clamps a too-large request", 5, 100, 5},
		{"smaller request honored under configured cap", 5, 3, 3},
		{"zero cap falls back to default 20", 0, 1000, research.MaxGatherFindings},
		{"negative cap falls back to default 20", -3, 0, research.MaxGatherFindings},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &fakeGatherer{}
			svc := research.NewService(research.Deps{Gatherer: g, GatherCap: tc.cap})
			_, capApplied, err := svc.GatherRelatedFindings(ctx, "q", tc.reqMax)
			require.NoError(t, err)
			require.Equal(t, tc.wantLimit, g.gotLimit)
			require.Equal(t, tc.wantLimit, capApplied)
		})
	}
}

// TestGatherRelatedFindingsRequiresWiringAndQuery asserts the bounded gather
// needs a wired seam and a non-empty query.
func TestGatherRelatedFindingsRequiresWiringAndQuery(t *testing.T) {
	// No gatherer wired.
	svc := research.NewService(research.Deps{})
	_, _, err := svc.GatherRelatedFindings(context.Background(), "q", 0)
	require.Error(t, err)

	// Empty query.
	svc = research.NewService(research.Deps{Gatherer: &fakeGatherer{}})
	_, _, err = svc.GatherRelatedFindings(context.Background(), "   ", 0)
	require.Error(t, err)
}

// TestRunResearchCycleAnswerFirstOrdering is the OT-P1-003 budget-order contract:
// the cycle GATHERS (bounded) first, produces the answer, and only THEN runs the
// bounded reconcile post-step.
func TestRunResearchCycleAnswerFirstOrdering(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	contested, err := fsvc.Add(ctx, findings.NewFinding{Claim: "outdated", Confidence: 0.5})
	require.NoError(t, err)

	g := &fakeGatherer{returnN: 2}
	svc := research.NewService(research.Deps{Gatherer: g, Findings: fsvc})

	var sawGatheredBeforeAnswer bool
	res, err := svc.RunResearchCycle(ctx, "what changed", func(_ context.Context, gathered []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		// The answerer runs AFTER gather: it must see the gathered set.
		sawGatheredBeforeAnswer = len(gathered) == 2
		return research.Brief{Query: "what changed", Level: research.LevelL3, Summary: "the answer"},
			[]research.ReconcileItem{{ExistingID: contested.ID, Confidence: 0.9, Contradicts: true}}, nil
	})
	require.NoError(t, err)
	require.True(t, sawGatheredBeforeAnswer, "gather runs before the answer step")
	require.Equal(t, "the answer", res.Brief.Summary)
	require.False(t, res.ReconcileSkipped)
	require.Equal(t, research.MaxGatherFindings, g.gotLimit, "the cycle gathers with the bounded cap")

	// Reconcile ran AFTER the answer: the contested finding was superseded.
	require.Len(t, res.Reconciled, 1)
	require.Equal(t, research.ActionSupersede, res.Reconciled[0].Action)
	got, err := fsvc.Get(ctx, contested.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)
}

// TestRunResearchCycleSkipsReconcileWhenAnswerErrors asserts the store is NOT
// curated when the run errors before answering — reconcile is a post-step that
// never fires off a failed answer.
func TestRunResearchCycleSkipsReconcileWhenAnswerErrors(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	contested, err := fsvc.Add(ctx, findings.NewFinding{Claim: "outdated", Confidence: 0.5})
	require.NoError(t, err)

	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{returnN: 1}, Findings: fsvc})

	answerErr := errors.New("synthesis failed")
	res, err := svc.RunResearchCycle(ctx, "q", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		return research.Brief{}, []research.ReconcileItem{{ExistingID: contested.ID, Confidence: 0.99, Contradicts: true}}, answerErr
	})
	require.ErrorIs(t, err, answerErr)
	require.True(t, res.ReconcileSkipped, "a failed answer skips reconcile")

	// The contested finding is untouched — no curation off a failed run.
	got, err := fsvc.Get(ctx, contested.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, got.Status)
}

// TestRunResearchCycleSkipsReconcileWhenNoAnswer asserts an empty summary (the
// run produced no answer) also skips the bounded reconcile post-step.
func TestRunResearchCycleSkipsReconcileWhenNoAnswer(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	contested, err := fsvc.Add(ctx, findings.NewFinding{Claim: "outdated", Confidence: 0.5})
	require.NoError(t, err)

	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{returnN: 1}, Findings: fsvc})

	res, err := svc.RunResearchCycle(ctx, "q", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		return research.Brief{Query: "q", Summary: "   "}, []research.ReconcileItem{{ExistingID: contested.ID, Confidence: 0.99, Contradicts: true}}, nil
	})
	require.NoError(t, err)
	require.True(t, res.ReconcileSkipped, "an empty answer skips reconcile")
	require.Empty(t, res.Reconciled)

	got, err := fsvc.Get(ctx, contested.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, got.Status)
}

// TestRunResearchCycleRequiresAnswerer asserts a nil answerer is rejected.
func TestRunResearchCycleRequiresAnswerer(t *testing.T) {
	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{}})
	_, err := svc.RunResearchCycle(context.Background(), "q", nil)
	require.Error(t, err)
}

// fakeSemanticIndex returns canned hits and records the limit it was asked for.
type fakeSemanticIndex struct {
	hits     []research.GatherHit
	gotLimit int
	err      error
}

func (f *fakeSemanticIndex) Search(_ context.Context, _ string, limit int) ([]research.GatherHit, error) {
	f.gotLimit = limit
	return f.hits, f.err
}

// fakeFindingsStore hydrates a fixed id->finding map.
type fakeFindingsStore struct {
	byID map[string]findings.Finding
}

func (f *fakeFindingsStore) GetMany(_ context.Context, ids []string) (map[string]findings.Finding, error) {
	out := make(map[string]findings.Finding, len(ids))
	for _, id := range ids {
		if v, ok := f.byID[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

// TestIndexGathererPreservesOrderAndSkipsMissing asserts the production gatherer
// hydrates the semantic hits in relevance order and silently skips an id the
// store no longer has (e.g. a raced supersede), projecting score from the hit.
func TestIndexGathererPreservesOrderAndSkipsMissing(t *testing.T) {
	idx := &fakeSemanticIndex{hits: []research.GatherHit{
		{FindingID: "b", Score: 0.9},
		{FindingID: "gone", Score: 0.8}, // not in the store
		{FindingID: "a", Score: 0.7},
	}}
	store := &fakeFindingsStore{byID: map[string]findings.Finding{
		"a": {ID: "a", Claim: "claim a", Confidence: 0.6, Status: findings.StatusActive},
		"b": {ID: "b", Claim: "claim b", Confidence: 0.8, Status: findings.StatusDisputed},
	}}
	g := research.IndexGatherer{Index: idx, Store: store}

	out, err := g.Gather(context.Background(), "q", 10)
	require.NoError(t, err)
	require.Equal(t, 10, idx.gotLimit)
	require.Len(t, out, 2, "the id missing from the store is skipped")
	require.Equal(t, "b", out[0].FindingID, "relevance order preserved")
	require.Equal(t, 0.9, out[0].Score, "score carried from the hit")
	require.Equal(t, findings.StatusDisputed, out[0].Status)
	require.Equal(t, "a", out[1].FindingID)
}

// recordingFindings is a research.FindingsService that records the ORDER of
// store mutations (and can fail supersedes), so tests can assert the answer
// precedes any curation.
type recordingFindings struct {
	events       *[]string
	supersedeErr error
}

func (r *recordingFindings) Add(_ context.Context, _ findings.NewFinding) (findings.Finding, error) {
	*r.events = append(*r.events, "add")
	return findings.Finding{ID: "new"}, nil
}

func (r *recordingFindings) Supersede(_ context.Context, id, _, _ string) (findings.Finding, error) {
	if r.supersedeErr != nil {
		return findings.Finding{}, r.supersedeErr
	}
	*r.events = append(*r.events, "supersede:"+id)
	return findings.Finding{ID: id}, nil
}

func (r *recordingFindings) Flag(_ context.Context, id, _ string) (findings.Finding, error) {
	*r.events = append(*r.events, "flag:"+id)
	return findings.Finding{ID: id}, nil
}

// TestRunResearchCycleAnswerBeforeReconcileMutation pins the non-blocking
// budget-order contract at the mutation level: the answer is produced BEFORE
// any reconcile mutation touches the findings store, so curation can never
// delay (or gate) the user-facing answer.
func TestRunResearchCycleAnswerBeforeReconcileMutation(t *testing.T) {
	var events []string
	rec := &recordingFindings{events: &events}
	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{}, Findings: rec})

	_, err := svc.RunResearchCycle(context.Background(), "q", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		events = append(events, "answer")
		return research.Brief{Query: "q", Level: research.LevelL3, Summary: "the answer"},
			[]research.ReconcileItem{{ExistingID: "old", Confidence: 0.9, Contradicts: true}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"answer", "supersede:old"}, events,
		"the answer must exist before the first reconcile mutation appears")
}

// TestRunResearchCycleAnswerSurvivesReconcileFailure asserts the user still
// receives the answer when the bounded curation post-step fails: the brief is
// carried on the result even though the cycle reports the reconcile error.
func TestRunResearchCycleAnswerSurvivesReconcileFailure(t *testing.T) {
	var events []string
	rec := &recordingFindings{events: &events, supersedeErr: errors.New("store locked")}
	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{}, Findings: rec})

	res, err := svc.RunResearchCycle(context.Background(), "q", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		return research.Brief{Query: "q", Level: research.LevelL3, Summary: "the answer"},
			[]research.ReconcileItem{{ExistingID: "old", Confidence: 0.9, Contradicts: true}}, nil
	})
	require.Error(t, err, "the reconcile failure is reported")
	require.Equal(t, "the answer", res.Brief.Summary, "the answer is delivered despite the failed curation pass")
}

// TestResearchCycleEndToEndSeededFindings is the OT-P1-003 integration path
// over a REAL SQLite findings store and the PRODUCTION IndexGatherer: a topic
// with pre-seeded related findings is gathered first (the answer step sees the
// priors), the answer is produced, and the reconcile post-step then updates
// finding statuses in the database.
func TestResearchCycleEndToEndSeededFindings(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)

	// Pre-seed related findings.
	outdated, err := fsvc.Add(ctx, findings.NewFinding{Claim: "the old fact", Confidence: 0.5})
	require.NoError(t, err)
	related, err := fsvc.Add(ctx, findings.NewFinding{Claim: "a related fact", Confidence: 0.7})
	require.NoError(t, err)

	idx := &fakeSemanticIndex{hits: []research.GatherHit{
		{FindingID: outdated.ID, Score: 0.9},
		{FindingID: related.ID, Score: 0.8},
	}}
	gatherer := research.IndexGatherer{Index: idx, Store: fsvc}
	svc := research.NewService(research.Deps{Gatherer: gatherer, Findings: fsvc})

	var gatheredClaims []string
	res, err := svc.RunResearchCycle(ctx, "what is the fact now", func(_ context.Context, gathered []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		for _, gf := range gathered {
			gatheredClaims = append(gatheredClaims, gf.Claim)
		}
		return research.Brief{Query: "what is the fact now", Level: research.LevelL3, Summary: "the updated answer"},
			[]research.ReconcileItem{{ExistingID: outdated.ID, Confidence: 0.9, Contradicts: true, Reason: "fresher evidence"}}, nil
	})
	require.NoError(t, err)

	// Gather read the pre-seeded priors (relevance order) before the answer.
	require.Equal(t, []string{"the old fact", "a related fact"}, gatheredClaims)
	require.Equal(t, "the updated answer", res.Brief.Summary)
	require.False(t, res.ReconcileSkipped)

	// The reconcile post-step updated statuses in the database.
	gotOutdated, err := fsvc.Get(ctx, outdated.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, gotOutdated.Status)
	gotRelated, err := fsvc.Get(ctx, related.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, gotRelated.Status, "the non-contradicted prior is untouched")
}

// TestSuccessiveResearchCyclesCurateStore asserts the librarian loop improves
// the store INCREMENTALLY: run 1 supersedes an outdated finding, run 2 flags a
// contested one, and the curation accumulates across runs.
func TestSuccessiveResearchCyclesCurateStore(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	outdated, err := fsvc.Add(ctx, findings.NewFinding{Claim: "outdated claim", Confidence: 0.5})
	require.NoError(t, err)
	contested, err := fsvc.Add(ctx, findings.NewFinding{Claim: "contested claim", Confidence: 0.5})
	require.NoError(t, err)

	svc := research.NewService(research.Deps{Gatherer: &fakeGatherer{returnN: 1}, Findings: fsvc})

	// Run 1: strong new evidence retires the outdated finding.
	_, err = svc.RunResearchCycle(ctx, "first question", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		return research.Brief{Query: "first question", Level: research.LevelL3, Summary: "first answer"},
			[]research.ReconcileItem{{ExistingID: outdated.ID, Confidence: 0.9, Contradicts: true}}, nil
	})
	require.NoError(t, err)
	got, err := fsvc.Get(ctx, outdated.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)

	// Run 2: a weak contradiction flags the contested finding for review.
	_, err = svc.RunResearchCycle(ctx, "second question", func(_ context.Context, _ []research.GatheredFinding) (research.Brief, []research.ReconcileItem, error) {
		return research.Brief{Query: "second question", Level: research.LevelL3, Summary: "second answer"},
			[]research.ReconcileItem{{ExistingID: contested.ID, Confidence: 0.4, Contradicts: true, Reason: "conflicting source"}}, nil
	})
	require.NoError(t, err)

	// Curation accumulated: run 1's supersede persists, run 2 added the flag.
	got, err = fsvc.Get(ctx, outdated.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, got.Status)
	got, err = fsvc.Get(ctx, contested.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusDisputed, got.Status)
}

// TestGatherBoundedSweepWithinBudget is the OT-P1-003 performance gate for the
// GATHER step: a full bounded sweep — 20 findings hydrated from a real SQLite
// store via the production IndexGatherer — must complete within 500ms.
func TestGatherBoundedSweepWithinBudget(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)

	hits := make([]research.GatherHit, 0, research.MaxGatherFindings)
	for i := 0; i < research.MaxGatherFindings; i++ {
		f, err := fsvc.Add(ctx, findings.NewFinding{Claim: fmt.Sprintf("seeded claim %d", i), Confidence: 0.6})
		require.NoError(t, err)
		hits = append(hits, research.GatherHit{FindingID: f.ID, Score: 1 - float64(i)/100})
	}
	gatherer := research.IndexGatherer{Index: &fakeSemanticIndex{hits: hits}, Store: fsvc}
	svc := research.NewService(research.Deps{Gatherer: gatherer})

	start := time.Now()
	out, capApplied, err := svc.GatherRelatedFindings(ctx, "seeded claim", research.MaxGatherFindings)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, research.MaxGatherFindings, capApplied)
	require.Len(t, out, research.MaxGatherFindings)
	require.Less(t, elapsed, 500*time.Millisecond, "bounded 20-finding sweep must complete within 500ms")
}
