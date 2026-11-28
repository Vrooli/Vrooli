package search

import (
	"context"
	"testing"
	"time"

	"content-desk/internal/aisearch"
	internalartifacts "content-desk/internal/artifacts"
	internalledger "content-desk/internal/ledger"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/search"
)

type draftsStub struct {
	internalartifacts.Repository
	drafts      []internalartifacts.Draft
	revisions   map[string]internalartifacts.CurrentRevision
	listErr     error
	revisionErr error
}

func (s draftsStub) List(context.Context) ([]internalartifacts.Draft, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.drafts, nil
}

func (s draftsStub) GetCurrentRevision(_ context.Context, id string) (internalartifacts.CurrentRevision, error) {
	if s.revisionErr != nil {
		return internalartifacts.CurrentRevision{}, s.revisionErr
	}
	return s.revisions[id], nil
}

type ledgerStub struct {
	internalledger.Repository
	records []internalledger.PublishRecord
	err     error
}

func (s ledgerStub) ListPublishHistory(context.Context, int) ([]internalledger.PublishRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

func newHandler(drafts draftsStub, ledger ledgerStub) handler {
	return handler{service: aisearch.NewService(aisearch.NewStoreSource(drafts, ledger))}
}

// [REQ:CONTENTD-P1-008] Search Hub receives a read-only live projection of
// both draft and publish history, with the editorial corpus remaining local.
func TestSearchMatchesDraftAndPublishHistory(t *testing.T) {
	drafts := draftsStub{
		drafts: []internalartifacts.Draft{{ID: "draft-product", Body: "Launch evidence for product team", Channel: "linkedin"}},
		revisions: map[string]internalartifacts.CurrentRevision{
			"draft-product": {DraftID: "draft-product", RevisionID: "rev-1", ApprovalActorKind: "operator", ApprovalCapacity: "operator"},
		},
	}
	ledger := ledgerStub{records: []internalledger.PublishRecord{{ID: "publish-product", DraftID: "draft-product", PublishedURL: "https://example.test/product", PlatformPostID: "post-product", Channel: "linkedin"}}}
	h := newHandler(drafts, ledger)

	response, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "product", Limit: 10}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Results, 2)
	// Both records must match; the relative order is not a contract. The draft
	// title must carry a human identity, never the opaque id alone.
	byID := make(map[string]int, len(response.Msg.Results))
	for i, r := range response.Msg.Results {
		byID[r.Id] = i
	}
	require.Contains(t, byID, "draft:draft-product")
	require.Contains(t, byID, "publish:publish-product")
	draft := response.Msg.Results[byID["draft:draft-product"]]
	require.Equal(t, "draft", draft.Kind)
	require.NotEqual(t, "draft-product", draft.Title)
	require.Contains(t, draft.Title, "draft")
	require.Equal(t, "publish-record", response.Msg.Results[byID["publish:publish-product"]].Kind)
}

func TestSearchRejectsGibberish(t *testing.T) {
	drafts := draftsStub{
		drafts:    []internalartifacts.Draft{{ID: "draft-product", Body: "Launch evidence for product team"}},
		revisions: map[string]internalartifacts.CurrentRevision{"draft-product": {DraftID: "draft-product"}},
	}
	h := newHandler(drafts, ledgerStub{})

	response, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "zzzxq", Limit: 10}))
	require.NoError(t, err)
	require.Empty(t, response.Msg.Results)
}

func TestSourceErrorFailsSearch(t *testing.T) {
	drafts := draftsStub{drafts: []internalartifacts.Draft{{ID: "draft-1"}}, revisionErr: context.DeadlineExceeded}
	h := newHandler(drafts, ledgerStub{})

	_, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "product", Limit: 10}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestStatusReportsNewestPublishTimeAndIsStable(t *testing.T) {
	older := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	newest := time.Date(2026, time.February, 2, 11, 30, 0, 0, time.UTC)
	ledger := ledgerStub{records: []internalledger.PublishRecord{
		{ID: "publish-old", DraftID: "draft-a", PublishedAt: older},
		{ID: "publish-new", DraftID: "draft-b", PublishedAt: newest},
	}}
	h := newHandler(draftsStub{}, ledger)

	first, err := h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	require.NoError(t, err)
	require.True(t, first.Msg.Available)
	require.Equal(t, int32(2), first.Msg.IndexedCount)
	require.Equal(t, newest.Format(time.RFC3339Nano), first.Msg.LastIndexedAt)
	require.NotEqual(t, time.Now().UTC().Format(time.RFC3339Nano), first.Msg.LastIndexedAt)

	second, err := h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, first.Msg.LastIndexedAt, second.Msg.LastIndexedAt, "status reads must not fabricate freshness")
}

func TestStatusWithNoSourcesReportsZeroMaterialization(t *testing.T) {
	h := newHandler(draftsStub{}, ledgerStub{})

	response, err := h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	require.NoError(t, err)
	require.True(t, response.Msg.Available)
	require.Equal(t, int32(0), response.Msg.IndexedCount)
	require.Empty(t, response.Msg.LastIndexedAt, "no observed mutation must not claim an index time")
}
