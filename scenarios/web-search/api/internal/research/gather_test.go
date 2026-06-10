package research_test

import (
	"context"
	"errors"
	"testing"

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
