package forest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/testutil/mocks"
)

func TestSummaryPromptPreservesTailAndForbidsInventedResolution(t *testing.T) { // [REQ:VMEM-P1-004]
	longSource := "initial status: phases 2-7 remain unimplemented\n" + strings.Repeat("detail ", 1000) + "\n2026-07-27: phases 2-7 shipped and validated"
	prompt := summaryPrompt(
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
		{ID: "a", FacetID: "episode", Body: "old episode a", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "b", FacetID: "episode", Body: "old episode b", Vector: []float64{.99, .01}, CreatedAt: now.Add(-47 * time.Hour), Kind: "entry"},
		{ID: "pinned", FacetID: "episode", Body: "pinned", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry", Pinned: true},
		{ID: "rule", FacetID: "standing-rule", RetentionPolicy: "retain", Body: "rule", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "recent", FacetID: "episode", Body: "recent", Vector: []float64{1, 0}, CreatedAt: now.Add(-time.Hour), Kind: "entry"},
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
	source := &memorySource{candidates: []Candidate{{ID: "only", FacetID: "episode", Body: "only", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
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
		{ID: "tight-a", FacetID: "episode", Body: "tight a", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "tight-b", FacetID: "episode", Body: "tight b", Vector: []float64{.99, .01}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "orphan", FacetID: "episode", Body: "orphan", Vector: []float64{0, 1}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
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
		{ID: "pinned", FacetID: "episode", Body: "pinned", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry", Pinned: true},
		{ID: "ordinary", FacetID: "episode", Body: "ordinary", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
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
		{ID: "failed-a", FacetID: "episode", Body: "failed a", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "failed-b", FacetID: "episode", Body: "failed b", Vector: []float64{.99, .01}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "good-a", FacetID: "episode", Body: "good a", Vector: []float64{0, 1}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "good-b", FacetID: "episode", Body: "good b", Vector: []float64{.01, .99}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
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
	source := &memorySource{candidates: []Candidate{{ID: "a", FacetID: "episode", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}, {ID: "b", FacetID: "episode", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
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
	source := &memorySource{candidates: []Candidate{{ID: "a", FacetID: "episode", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}, {ID: "b", FacetID: "episode", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"}}}
	repo := &memoryRepo{source: source}
	client := &retryingInference{errs: []error{errors.New("unavailable: unexpected EOF")}}
	svc := NewService(repo, source, client, Config{Target: 1})
	svc.now = func() time.Time { return now }
	result, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, client.calls)
	require.Equal(t, 1, result.CompactedCount)
}

func TestRunRepeatsUntilEligibleFrontierReachesTarget(t *testing.T) { // [REQ:VMEM-P0-007]
	now := time.Now().UTC()
	source := &memorySource{candidates: []Candidate{
		{ID: "a", FacetID: "episode", Vector: []float64{1, 0}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "b", FacetID: "episode", Vector: []float64{.99, .01}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "c", FacetID: "episode", Vector: []float64{0, 1}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
		{ID: "d", FacetID: "episode", Vector: []float64{.01, .99}, CreatedAt: now.Add(-48 * time.Hour), Kind: "entry"},
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
	next = append(next, Candidate{ID: s.ID, FacetID: s.FacetID, Body: s.Body, Vector: s.Vector, CreatedAt: time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC), Kind: "summary", Depth: s.Depth, Generation: s.Generation})
	r.source.candidates = next
	r.source.leaves = append(r.source.leaves, edges[0].ChildID, edges[1].ChildID)
	return s, nil
}
func (r *memoryRepo) Frontier(context.Context) ([]Summary, error) { return nil, nil }
func (r *memoryRepo) Nodes(context.Context, int) ([]Node, error)  { return nil, nil }
func (r *memoryRepo) Rebuild(context.Context) error               { return nil }

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
