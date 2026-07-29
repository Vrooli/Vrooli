package search

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	internalartifacts "content-desk/internal/artifacts"
	internalledger "content-desk/internal/ledger"
	"github.com/stretchr/testify/require"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/search"
)

type draftsStub struct {
	internalartifacts.Repository
	drafts []internalartifacts.Draft
}

func (s draftsStub) List(context.Context) ([]internalartifacts.Draft, error) { return s.drafts, nil }

type ledgerStub struct {
	internalledger.Repository
	records []internalledger.PublishRecord
}

func (s ledgerStub) ListPublishHistory(context.Context, int) ([]internalledger.PublishRecord, error) {
	return s.records, nil
}

// [REQ:CONTENTD-P1-008] Search Hub receives a read-only live projection of
// both draft and publish history, with the editorial corpus remaining local.
func TestSearchMatchesDraftAndPublishHistory(t *testing.T) {
	h := handler{drafts: draftsStub{drafts: []internalartifacts.Draft{{ID: "draft-product", Body: "Launch evidence for product team", Channel: "linkedin"}}}, ledger: ledgerStub{records: []internalledger.PublishRecord{{ID: "publish-product", DraftID: "draft-product", PublishedURL: "https://example.test/product", PlatformPostID: "post-product"}}}}
	response, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "product", Limit: 10}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Results, 2)
	require.Equal(t, "draft-product", response.Msg.Results[0].Id)
	require.Equal(t, "publish-record", response.Msg.Results[1].Kind)
}
