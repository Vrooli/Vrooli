package recall

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type source []Node

func (s source) Nodes(context.Context) ([]Node, error) { return s, nil }

type embedder struct{ v []float64 }

func (e embedder) EmbedQuery(context.Context, string) ([]float64, error) { return e.v, nil }
func TestRecallKeepsAbsorbedLeafAndCollapsesDescendants(t *testing.T) { // [REQ:VMEM-P0-003] [REQ:VMEM-P0-004]
	now := time.Now()
	svc := NewService(source{{ID: "summary", Text: "weak summary", Vector: []float64{.1, 1}, Depth: 1}, {ID: "leaf", ParentID: "summary", EntryID: "leaf", Text: "exact leaf", Vector: []float64{1, 0}, CreatedAt: now}}, embedder{[]float64{1, 0}}, Config{})
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
	svc := NewService(source{{ID: "summary", Text: "summary", Vector: []float64{1, 0}, Depth: 1}, {ID: "leaf", ParentID: "summary", Text: "leaf", Vector: []float64{.8, 0}}}, embedder{[]float64{1, 0}}, Config{})
	hits, err := svc.Recall(context.Background(), "summary", 10)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "summary", hits[0].Node.ID)
	require.Equal(t, []Node{{ID: "leaf", ParentID: "summary", Text: "leaf", Vector: []float64{.8, 0}}}, hits[0].Descendants)
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
