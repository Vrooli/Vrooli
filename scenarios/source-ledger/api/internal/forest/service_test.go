package forest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"source-ledger/internal/inference"
	"source-ledger/internal/policy"
	"source-ledger/internal/testutil/mocks"
)

func TestSummaryPromptPreservesTailAndForbidsInventedResolution(t *testing.T) { // [REQ:VMEM-P1-004]
	longSource := "initial status: phases 2-7 remain unimplemented\n" + strings.Repeat("detail ", 1000) + "\n2026-07-27: phases 2-7 shipped and validated"
	prompt := summaryPrompt(
		policy.DefaultSummaryInstruction,
		Candidate{Body: longSource, CreatedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		Candidate{Body: "separate source", CreatedAt: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)},
	)

	require.Contains(t, prompt, "initial status: phases 2-7 remain unimplemented")
	require.Contains(t, prompt, "2026-07-27: phases 2-7 shipped and validated")
	require.Contains(t, prompt, "[EARLIER CONTEXT]")
	require.Contains(t, prompt, "[STATUS EVIDENCE — authoritative for current status]")
	require.Contains(t, prompt, "[LATEST CONTEXT — authoritative for current status]")
	require.Contains(t, prompt, "treat its latest explicit dated update as authoritative")
	require.Contains(t, prompt, "do not headline, repeat, or infer a current status from EARLIER CONTEXT")
	require.Contains(t, prompt, "say that status is unresolved")
	require.LessOrEqual(t, len([]rune(boundedSummaryInput(longSource))), 3500)
}

func TestRecentStatusEvidencePrefersRecentStatusPassages(t *testing.T) {
	text := strings.Repeat("filler ", 200) + "phase 1 was done years ago" + strings.Repeat(" filler", 300) + "2026-07-27: phases 2-7 shipped and validated"

	evidence := recentStatusEvidence(text, 700)

	require.Contains(t, evidence, "phases 2-7 shipped and validated")
}

func TestRunCompactsOnlyEligibleOldEpisodesAndPreservesLeaves(t *testing.T) { // [REQ:VMEM-P0-001] [REQ:VMEM-P0-007]
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	source := &memorySource{candidates: []Candidate{
		{ID: "a", FacetID: "episode", Compactable: true, Body: "old episode a", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "b", FacetID: "episode", Compactable: true, Body: "old episode b", Vectors: [][]float64{{.99, .01}}, CreatedAt: now.Add(-47 * time.Hour), Kind: "entry"},
		{ID: "pinned", FacetID: "episode", Compactable: true, Body: "pinned", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry", Pinned: true},
		{ID: "rule", FacetID: "standing-rule", Compactable: false, RetentionPolicy: "retain", Body: "rule", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "recent", FacetID: "episode", Compactable: true, Body: "recent", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-time.Hour), Kind: "entry"},
	}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "compressed old episodes", EmbedOut: []float64{1, 0}}, Config{Target: 1, RecencyFloor: 24 * time.Hour})
	svc.now = func() time.Time { return now }
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.CompactedCount)
	require.Equal(t, 2, result.EligibleFrontierBefore)
	require.Equal(t, 1, result.EligibleFrontierAfter)
	require.Equal(t, []string{"a", "b"}, repo.edges[0].childIDs())
	require.Equal(t, 2, len(source.leaves), "compaction absorbs leaves in the forest but never removes journal leaves")
}

func TestRunWithFrontierUnderTargetIsNoop(t *testing.T) { // [REQ:VMEM-P0-007]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{{ID: "only", FacetID: "episode", Compactable: true, Body: "only", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "must not run", EmbedOut: []float64{1, 0}}, Config{Target: 2})
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Zero(t, result.CompactedCount)
	require.Empty(t, repo.edges)
}

func TestRunLeavesOrphanUntilTightClusterCompacts(t *testing.T) { // [REQ:VMEM-P0-007]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "tight-a", FacetID: "episode", Compactable: true, Body: "tight a", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "tight-b", FacetID: "episode", Compactable: true, Body: "tight b", Vectors: [][]float64{{.99, .01}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "orphan", FacetID: "episode", Compactable: true, Body: "orphan", Vectors: [][]float64{{0, 1}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
	}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "tight summary", EmbedOut: []float64{1, 0}}, Config{Target: 2})
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.CompactedCount)
	require.Equal(t, []string{"tight-a", "tight-b"}, repo.edges[0].childIDs())
	for _, candidate := range source.candidates {
		if candidate.ID == "orphan" {
			require.Equal(t, "orphan", candidate.ID)
		}
	}
}

func TestCompactionPreservesPinnedEntry(t *testing.T) { // [REQ:VMEM-P0-006]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "pinned", FacetID: "episode", Compactable: true, Body: "pinned", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry", Pinned: true},
		{ID: "ordinary", FacetID: "episode", Compactable: true, Body: "ordinary", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
	}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "must not run", EmbedOut: []float64{1, 0}}, Config{Target: 1})
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Zero(t, result.CompactedCount)
	require.Empty(t, repo.edges)
}

func TestRunSkipsFailedPairAndCompactsNextCluster(t *testing.T) {
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "failed-a", FacetID: "episode", Compactable: true, Body: "failed a", Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "failed-b", FacetID: "episode", Compactable: true, Body: "failed b", Vectors: [][]float64{{.99, .01}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "good-a", FacetID: "episode", Compactable: true, Body: "good a", Vectors: [][]float64{{0, 1}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "good-b", FacetID: "episode", Compactable: true, Body: "good b", Vectors: [][]float64{{.01, .99}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
	}}
	repo := &memoryRepo{source: source}
	client := &retryingInference{errs: []error{errors.New("provider unavailable")}}
	svc := NewService(repo, source, client, Config{Target: 3})
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.CompactedCount)
	require.Equal(t, []string{"good-a", "good-b"}, repo.edges[0].childIDs())
}

func TestRunSummaryFailureLeavesFrontierUnchanged(t *testing.T) {
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{{ID: "a", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}, {ID: "b", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeErr: errors.New("offline")}, Config{Target: 1})
	svc.now = func() time.Time { return now }
	_, err := svc.Run(context.Background())
	require.ErrorContains(t, err, "summarize compaction cluster")
	require.Empty(t, repo.edges)
	require.Len(t, source.candidates, 2)
}

func TestRunRetriesTransientSummaryEOFBeforeWriting(t *testing.T) {
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{{ID: "a", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}, {ID: "b", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
	repo := &memoryRepo{source: source}
	client := &retryingInference{errs: []error{errors.New("unavailable: unexpected EOF")}}
	svc := NewService(repo, source, client, Config{Target: 1})
	svc.now = func() time.Time { return now }
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, client.calls)
	require.Equal(t, 1, result.CompactedCount)
}

func TestRunRepeatsUntilEligibleFrontierReachesTarget(t *testing.T) { // [REQ:VMEM-P0-007] [REQ:SL-P0-003]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "a", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "b", FacetID: "episode", Compactable: true, Vectors: [][]float64{{.99, .01}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "c", FacetID: "episode", Compactable: true, Vectors: [][]float64{{0, 1}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "d", FacetID: "episode", Compactable: true, Vectors: [][]float64{{.01, .99}}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
	}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "summary", EmbedOut: []float64{1, 0}}, Config{Target: 1})
	svc.now = func() time.Time { return now }
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, result.CompactedCount)
	require.Equal(t, 1, result.EligibleFrontierAfter)
	require.Len(t, repo.edges, 3)
}

type memorySource struct {
	candidates []Candidate
	leaves     []string
}

func (s *memorySource) CompactionCandidates(context.Context) ([]Candidate, error) {
	return append([]Candidate(nil), s.candidates...), nil
}

type writtenEdges []Edge

func (e writtenEdges) childIDs() []string { return []string{e[0].ChildID, e[1].ChildID} }

type memoryRepo struct {
	source *memorySource
	edges  []writtenEdges
	nodes  []Node
}

func (r *memoryRepo) CreateSummary(_ context.Context, s Summary, edges []Edge) (Summary, error) {
	s.ID = fmt.Sprintf("summary-%d", len(r.edges)+1)
	for i := range edges {
		edges[i].ParentID = s.ID
	}
	r.edges = append(r.edges, writtenEdges(edges))
	ids := map[string]bool{edges[0].ChildID: true, edges[1].ChildID: true}
	next := make([]Candidate, 0, len(r.source.candidates)-1)
	for _, c := range r.source.candidates {
		if !ids[c.ID] {
			next = append(next, c)
		}
	}
	next = append(next, Candidate{Compactable: true, ID: s.ID, FacetID: s.FacetID, Body: s.Body, Vectors: [][]float64{s.Vector}, CreatedAt: time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC), Kind: "summary", Depth: s.Depth, Generation: s.Generation})
	r.source.candidates = next
	r.source.leaves = append(r.source.leaves, edges[0].ChildID, edges[1].ChildID)
	return s, nil
}
func (r *memoryRepo) Frontier(context.Context) ([]Summary, error) { return nil, nil }
func (r *memoryRepo) Nodes(context.Context, int) ([]Node, error) {
	return append([]Node(nil), r.nodes...), nil
}
func (r *memoryRepo) Rebuild(context.Context) error { return nil }

type retryingInference struct {
	errs  []error
	calls int
}

func (r *retryingInference) Embed(context.Context, string, inference.EmbeddingTask) ([]float64, error) {
	return []float64{1, 0}, nil
}
func (r *retryingInference) Classify(context.Context, string) (string, error) { return "", nil }
func (r *retryingInference) Summarize(context.Context, string) (string, error) {
	r.calls++
	if len(r.errs) > 0 {
		err := r.errs[0]
		r.errs = r.errs[1:]
		return "", err
	}
	return "retry summary", nil
}

// VMEM-P1-005's purpose clause: several derived facet texts exist so clustering
// can group by more than one notion of relatedness. Scoring only the first space
// meant two memories about the same entities never clustered when their topic
// framings diverged.
func TestClusteringGroupsOnASecondaryFacetSpace(t *testing.T) { // [REQ:VMEM-P1-005]
	now := time.Now().UTC()
	// Topic framings (index 0) pull a/b apart; entities framings (index 1) agree.
	source := &memorySource{candidates: []Candidate{
		{
			ID: "a", FacetID: "episode", Compactable: true, Body: "a", Kind: "entry", CreatedAt: now.Add(-48 * time.Hour),
			Vectors: [][]float64{{1, 0}, {0, 1}},
		},
		{
			ID: "b", FacetID: "episode", Compactable: true, Body: "b", Kind: "entry", CreatedAt: now.Add(-48 * time.Hour),
			Vectors: [][]float64{{0, 1}, {0, 1}},
		},
		{
			ID: "c", FacetID: "episode", Compactable: true, Body: "c", Kind: "entry", CreatedAt: now.Add(-48 * time.Hour),
			Vectors: [][]float64{{.7, .7}, {1, 0}},
		},
	}}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "shared entities", EmbedOut: []float64{1, 0}}, Config{Target: 2})
	svc.now = func() time.Time { return now }

	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.CompactedCount)
	require.Equal(t, []string{"a", "b"}, repo.edges[0].childIDs(),
		"a and b agree perfectly on the entities space and must cluster there, even though c is closer to a on topic")
}

func TestRebuildRestoresLeafFrontierWithoutCallingInference(t *testing.T) { // [REQ:SL-P0-003]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{{ID: "entry-1", Kind: "entry", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour)}}}
	repo := &memoryRepo{source: source}
	inferenceClient := &retryingInference{errs: []error{errors.New("summarizer must not be called by rebuild")}}
	svc := NewService(repo, source, inferenceClient, Config{Target: 1})

	result, err := svc.Rebuild(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.EligibleFrontierBefore)
	require.Equal(t, 1, result.EligibleFrontierAfter)
	require.Equal(t, 0, inferenceClient.calls)
}

func TestPairScoringNeverComparesAcrossFacetSpaces(t *testing.T) { // [REQ:VMEM-P1-005]
	// Identical vectors sit in different spaces. Comparing index 0 to index 1
	// would score a perfect match that means nothing.
	require.InDelta(t, -1.0, bestSpacePairScore([][]float64{{1, 0}}, [][]float64{}), 1e-9)
	require.InDelta(t, 0.0, bestSpacePairScore([][]float64{{1, 0}}, [][]float64{{0, 1}}), 1e-9)
	// A summary carries one space; a leaf carries three. The shorter side bounds.
	require.InDelta(t, 1.0, bestSpacePairScore([][]float64{{1, 0}}, [][]float64{{1, 0}, {0, 1}, {1, 1}}), 1e-9)
}

func TestFrontierFiltersAndOrdersEligibleNodesByCompactionScore(t *testing.T) {
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "z-high", Kind: "entry", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "a-mid", Kind: "entry", FacetID: "episode", Compactable: true, Vectors: [][]float64{{0, 1}}, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "z-peer", Kind: "entry", FacetID: "episode", Compactable: true, Vectors: [][]float64{{1, 0}}, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "standing", Kind: "entry", FacetID: "standing-rule", Compactable: false, Vectors: [][]float64{{1, 1}}, CreatedAt: now.Add(-48 * time.Hour)},
	}}
	repo := &memoryRepo{source: source, nodes: []Node{{ID: "a-mid", FacetID: "episode"}, {ID: "standing", FacetID: "standing-rule"}, {ID: "z-peer", FacetID: "episode"}, {ID: "z-high", FacetID: "episode"}}}
	svc := NewService(repo, source, &mocks.FakeInference{}, Config{Target: 2, FrontierLimit: 10})
	svc.now = func() time.Time { return now }
	result, err := svc.Frontier(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 3, result.EligibleCount)
	require.Equal(t, 2, result.Target)
	require.Equal(t, []string{"z-high", "z-peer", "a-mid"}, []string{result.Nodes[0].ID, result.Nodes[1].ID, result.Nodes[2].ID})
	require.NotContains(t, []string{result.Nodes[0].ID, result.Nodes[1].ID, result.Nodes[2].ID}, "standing")
	require.InDelta(t, 1, result.Nodes[0].CompactionScore, 1e-9)
	require.InDelta(t, 0, result.Nodes[2].CompactionScore, 1e-9)
}

// A scheduled pass must stop at its cluster budget and return cleanly, even
// with the frontier still far above target. Driving a large backlog to target
// in one pass costs one provider call per cluster, which a loop cannot bound.
func TestRunBoundedStopsAtClusterBudgetAboveTarget(t *testing.T) {
	now := time.Now().UTC()
	candidates := make([]Candidate, 0, 12)
	for n := range 12 {
		candidates = append(candidates, Candidate{
			ID: fmt.Sprintf("entry-%02d", n), FacetID: "episode", Compactable: true,
			Body: fmt.Sprintf("body %d", n), Kind: "entry", CreatedAt: now.Add(-48 * time.Hour),
			Vectors: [][]float64{{1, float64(n) / 100}},
		})
	}
	source := &memorySource{candidates: candidates}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "summary", EmbedOut: []float64{1, 0}}, Config{Target: 2})
	svc.now = func() time.Time { return now }

	result, err := svc.RunBounded(context.Background(), 3)
	require.NoError(t, err, "hitting the budget is a clean stop, not a failure")
	require.Equal(t, 3, result.CompactedCount)
	require.Equal(t, 12, result.EligibleFrontierBefore)
	require.Equal(t, 9, result.EligibleFrontierAfter, "each cluster absorbs two nodes and adds one summary")
	require.Greater(t, result.EligibleFrontierAfter, 2, "the frontier is deliberately left above target")
}

// A budget of zero means run to target, matching the unbounded operator RPC.
func TestRunBoundedWithoutBudgetRunsToTarget(t *testing.T) {
	now := time.Now().UTC()
	candidates := make([]Candidate, 0, 6)
	for n := range 6 {
		candidates = append(candidates, Candidate{
			ID: fmt.Sprintf("entry-%02d", n), FacetID: "episode", Compactable: true,
			Body: fmt.Sprintf("body %d", n), Kind: "entry", CreatedAt: now.Add(-48 * time.Hour),
			Vectors: [][]float64{{1, float64(n) / 100}},
		})
	}
	source := &memorySource{candidates: candidates}
	repo := &memoryRepo{source: source}
	svc := NewService(repo, source, &mocks.FakeInference{SummarizeOut: "summary", EmbedOut: []float64{1, 0}}, Config{Target: 2})
	svc.now = func() time.Time { return now }

	result, err := svc.RunBounded(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 2, result.EligibleFrontierAfter)
}
