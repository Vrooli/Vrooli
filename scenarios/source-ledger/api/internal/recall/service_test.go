package recall

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type source []Node

func (s source) Nodes(context.Context) ([]Node, error) { return s, nil }

// The fake mirrors the real query's contract: wake sees pinned and frontier
// nodes only, and never receives vectors.
func (s source) AmbientNodes(context.Context, int) ([]Node, error) {
	out := make([]Node, 0, len(s))
	for _, n := range s {
		if !n.Pinned && !n.Frontier {
			continue
		}
		n.Vectors = nil
		out = append(out, n)
	}
	return out, nil
}

type embedder struct{ v []float64 }

func (e embedder) EmbedQuery(context.Context, string) ([]float64, error) { return e.v, nil }
func TestRecallKeepsAbsorbedLeafAndCollapsesDescendants(t *testing.T) { // [REQ:VMEM-P0-001] [REQ:VMEM-P0-003] [REQ:VMEM-P0-004]
	now := time.Now()
	svc := NewService(source{{ID: "summary", Text: "weak summary", Vectors: [][]float64{{.1, 1}}, Depth: 1}, {ID: "leaf", ParentID: "summary", EntryID: "leaf", Text: "exact leaf", Vectors: [][]float64{{1, 0}}, CreatedAt: now}}, embedder{[]float64{1, 0}}, Config{})
	hits, err := svc.Recall(context.Background(), "exact", 10)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "leaf", hits[0].Node.ID)
}

func TestWakePinsFirstAndNeverSilentlyTruncatesPins(t *testing.T) { // [REQ:VMEM-P0-006] [REQ:VMEM-P0-008]
	now := time.Now()
	svc := NewService(source{{ID: "p1", Pinned: true, Text: "one\ntwo", CreatedAt: now}, {ID: "p2", Pinned: true, Text: "three\nfour", CreatedAt: now}, {ID: "frontier", Frontier: true, Text: "later", CreatedAt: now}}, embedder{}, Config{WakeBudget: 3})
	wake, err := svc.Wake(context.Background(), 0)
	require.NoError(t, err)
	require.True(t, wake.Overflow)
	require.Len(t, wake.Hits, 2)
	require.Equal(t, "p1", wake.Hits[0].Node.ID)
}

func TestRecallCollapsesDescendantsWhenAncestorWins(t *testing.T) {
	svc := NewService(source{{ID: "summary", Text: "summary", Vectors: [][]float64{{1, 0}}, Depth: 1}, {ID: "leaf", ParentID: "summary", Text: "leaf", Vectors: [][]float64{{.8, 0}}}}, embedder{[]float64{1, 0}}, Config{})
	hits, err := svc.Recall(context.Background(), "summary", 10)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "summary", hits[0].Node.ID)
	require.Equal(t, []Node{{ID: "leaf", ParentID: "summary", Text: "leaf", Vectors: [][]float64{{.8, 0}}}}, hits[0].Descendants)
}

func TestWakeBudgetStableAcrossCorpusSize(t *testing.T) {
	now := time.Now()
	small := make(source, 100)
	large := make(source, 1000)
	for i := range large {
		n := Node{ID: string(rune(i + 1)), Frontier: true, Text: "one line", CreatedAt: now.Add(time.Duration(i) * time.Second)}
		large[i] = n
		if i < len(small) {
			small[i] = n
		}
	}
	for _, nodes := range []source{small, large} {
		wake, err := NewService(nodes, embedder{}, Config{WakeBudget: 20}).Wake(context.Background(), 0)
		require.NoError(t, err)
		require.Len(t, wake.Hits, 20)
	}
}

// The declared unit of a facet's resident budget is entries. Measuring it in
// lines made every facet whose newest memory was longer than its ceiling emit
// nothing at all, so ambient recall collapsed to whichever facet happened to
// hold short entries.
func TestFacetResidentBudgetCountsEntriesNotLines(t *testing.T) { // [REQ:VMEM-P0-005] [REQ:VMEM-P0-008]
	now := time.Now()
	nodes := source{
		{ID: "rule-1", FacetID: "standing-rule", Frontier: true, Text: "a rule\nwith\nmany\nlines\nindeed", CreatedAt: now},
		{ID: "rule-2", FacetID: "standing-rule", Frontier: true, Text: "another rule\nalso\nlong", CreatedAt: now.Add(-time.Minute)},
		{ID: "note-1", FacetID: "gotcha", Frontier: true, Text: "a gotcha\nspanning\nfour\nlines", CreatedAt: now},
	}
	svc := NewService(nodes, embedder{}, Config{WakeBudget: 100, MaxEntryLines: 2, FacetBudgets: map[string]int{"standing-rule": 2, "gotcha": 1}})
	wake, err := svc.Wake(context.Background(), 0)
	require.NoError(t, err)
	require.False(t, wake.Overflow)

	byFacet := map[string]int{}
	for _, h := range wake.Hits {
		byFacet[h.Node.FacetID]++
	}
	require.Equal(t, 2, byFacet["standing-rule"], "a facet fills to its entry ceiling regardless of how long its entries are")
	require.Equal(t, 1, byFacet["gotcha"])
}

func TestWakeExcerptsLongEntriesAndMarksTheTruncation(t *testing.T) { // [REQ:VMEM-P0-008]
	now := time.Now()
	svc := NewService(source{{ID: "long", FacetID: "episode", Frontier: true, Text: "headline\ndetail\ndropped\nalso dropped", CreatedAt: now}}, embedder{}, Config{WakeBudget: 100, MaxEntryLines: 2, FacetBudgets: map[string]int{"episode": 4}})
	wake, err := svc.Wake(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, wake.Hits, 1)
	require.Equal(t, "headline\ndetail "+truncationMarker, wake.Hits[0].Node.Text)
	require.Equal(t, 2, lines(wake.Hits[0].Node.Text), "an excerpt costs at most MaxEntryLines against the wake budget")
}

// One oversized memory must not cost its facet the rest of its residency.
func TestWakeSkipsAnOversizedEntryInsteadOfAbandoningTheFacet(t *testing.T) { // [REQ:VMEM-P0-008]
	now := time.Now()
	nodes := source{
		{ID: "huge", FacetID: "episode", Frontier: true, Text: "aaa\nbbb", CreatedAt: now},
		{ID: "small", FacetID: "episode", Frontier: true, Text: "fits", CreatedAt: now.Add(-time.Minute)},
	}
	// Budget 1 admits the one-line entry only; the two-line excerpt must be
	// skipped rather than ending the facet's turn.
	svc := NewService(nodes, embedder{}, Config{WakeBudget: 1, MaxEntryLines: 2, FacetBudgets: map[string]int{"episode": 4}})
	wake, err := svc.Wake(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, wake.Hits, 1)
	require.Equal(t, "small", wake.Hits[0].Node.ID)
}

// Every facet with a declared residency must be reachable within the default
// budget, or the last facets in sort order silently never appear.
func TestDefaultWakeBudgetSatisfiesEveryDeclaredResidency(t *testing.T) { // [REQ:VMEM-P0-008]
	now := time.Now()
	budgets := map[string]int{"standing-rule": 8, "environment-fact": 4, "gotcha": 4, "episode": 12, "thread": 4, "entity-record": 4}
	var nodes source
	total := 0
	for facet, ceiling := range budgets {
		for i := 0; i < ceiling; i++ {
			nodes = append(nodes, Node{ID: facet + string(rune('a'+i)), FacetID: facet, Frontier: true, Text: "first line\nsecond line\nthird line", CreatedAt: now.Add(-time.Duration(i) * time.Minute)})
			total++
		}
	}
	svc := NewService(nodes, embedder{}, Config{FacetBudgets: budgets})
	wake, err := svc.Wake(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, wake.Hits, total, "the default wake budget must hold every declared residency at DefaultMaxEntryLines apiece")
}

// Imported memory files open with YAML frontmatter. Rendering that verbatim
// spent an entry's whole excerpt on a delimiter and a slug.
func TestExcerptLeadsWithFrontmatterDescriptionNotTheDelimiter(t *testing.T) { // [REQ:VMEM-P0-008]
	body := "---\nname: never-git-stash-in-vrooli\ndescription: a failed stash push let a trailing pop target an unrelated stash\nmetadata:\n  type: feedback\n---\n\nUse baselines or a worktree instead.\nMore detail here."
	got := excerptText(body, 2)
	require.Equal(t, "a failed stash push let a trailing pop target an unrelated stash\nUse baselines or a worktree instead. "+truncationMarker, got)
}

func TestExcerptFallsBackToNameWhenNoDescription(t *testing.T) {
	require.Equal(t, "some-slug\nprose", excerptText("---\nname: some-slug\n---\nprose", 2))
}

func TestExcerptLeavesNonFrontmatterTextAlone(t *testing.T) {
	require.Equal(t, "plain memory", excerptText("plain memory", 2))
	require.Equal(t, "--- not frontmatter", excerptText("--- not frontmatter", 2))
}

// A query that matches a memory's entities framing must surface it even when the
// topic framing is unrelated. Scoring only the first space made two thirds of the
// embedding corpus dead weight.
func TestRecallScoresAcrossEveryFacetSpace(t *testing.T) { // [REQ:VMEM-P1-005] [REQ:VMEM-P0-003]
	nodes := source{
		{ID: "topic-match", Text: "topic match", Vectors: [][]float64{{.6, .8}}},
		{ID: "entities-match", Text: "entities match", Vectors: [][]float64{{0, 1}, {1, 0}}},
	}
	hits, err := NewService(nodes, embedder{[]float64{1, 0}}, Config{}).Recall(context.Background(), "q", 10)
	require.NoError(t, err)
	require.Equal(t, "entities-match", hits[0].Node.ID,
		"a perfect match in a secondary space must outrank a partial match in the primary space")
}

// Wake can never emit more than a facet's residency, so the query must not
// carry the rest of the frontier's bodies into memory to reach that answer.
func TestPerFacetLimitIsTheLargestDeclaredResidency(t *testing.T) { // [REQ:VMEM-P0-008]
	svc := NewService(source{}, embedder{}, Config{FacetBudgets: map[string]int{"episode": 12, "gotcha": 4}})
	require.Equal(t, 12, svc.perFacetLimit())

	// With no declared residencies wake walks one recency list, so the line
	// budget is the only bound that can apply.
	require.Equal(t, 40, NewService(source{}, embedder{}, Config{WakeBudget: 40}).perFacetLimit())
}
