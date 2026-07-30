package recall

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	internaljournal "vrooli-memory/internal/journal"
	"vrooli-memory/internal/journal/mocks"
	internalrecall "vrooli-memory/internal/recall"
)

type handlerSource []internalrecall.Node

func (s handlerSource) Nodes(context.Context) ([]internalrecall.Node, error) { return s, nil }

type handlerEmbedder struct{ vector []float64 }

func (e handlerEmbedder) EmbedQuery(context.Context, string) ([]float64, error) { return e.vector, nil }

func TestWakeReportsOverflowAndRecallExposesZoomableNodeID(t *testing.T) {
	now := time.Now()
	svc := internalrecall.NewService(handlerSource{
		{ID: "pinned", EntryID: "entry-pinned", Pinned: true, Text: "one\ntwo", CreatedAt: now, Vector: []float64{0, 1}},
		{ID: "summary-1", EntryID: "summary-1", FacetID: "episode", Frontier: true, Text: "summary", CreatedAt: now.Add(time.Second), Vector: []float64{1, 0}, Depth: 1, Span: 2, Summary: true},
	}, handlerEmbedder{vector: []float64{1, 0}}, internalrecall.Config{WakeBudget: 1})
	h := NewConnectHandler(svc, nil)

	wake, err := h.Wake(context.Background(), connect.NewRequest(&recallv1.WakeRequest{}))
	require.NoError(t, err)
	require.True(t, wake.Msg.GetOverflow())
	require.Len(t, wake.Msg.GetHits(), 1)
	require.Equal(t, "pinned", wake.Msg.GetHits()[0].GetNodeId())

	recall, err := h.Recall(context.Background(), connect.NewRequest(&recallv1.RecallRequest{Query: "summary", Limit: 1}))
	require.NoError(t, err)
	require.Len(t, recall.Msg.GetHits(), 1)
	require.Equal(t, "summary-1", recall.Msg.GetHits()[0].GetNodeId())
	require.True(t, recall.Msg.GetHits()[0].GetSummary())
	require.EqualValues(t, 2, recall.Msg.GetHits()[0].GetSpan())
}

func TestListSiblingEventsUsesOnlyTheStoredRunCorrelation(t *testing.T) { // [REQ:VMEM-P1-002]
	entry := internaljournal.Entry{ID: "entry-1", Body: "first", Correlation: internaljournal.Correlation{RunID: "run-1"}}
	sibling := internaljournal.Entry{ID: "entry-2", Body: "second", Correlation: internaljournal.Correlation{RunID: "run-1"}}
	repo := &mocks.Repository{GetOut: entry, ListOut: []internaljournal.Entry{entry, sibling}}
	journal := internaljournal.NewService(repo, nil)
	h := NewConnectHandler(internalrecall.NewService(handlerSource{}, handlerEmbedder{}, internalrecall.Config{}), nil, journal)
	resp, err := h.ListSiblingEvents(context.Background(), connect.NewRequest(&recallv1.ListSiblingEventsRequest{EntryId: entry.ID}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEntries(), 1)
	require.Equal(t, sibling.ID, resp.Msg.GetEntries()[0].GetEntryId())
}
