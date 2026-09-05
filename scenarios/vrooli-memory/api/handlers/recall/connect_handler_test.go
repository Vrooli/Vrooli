package recall

import (
	"context"
	"errors"
	"testing"

	"vrooli-memory/internal/ledgerclient"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
)

type unavailableRecall struct{}

func (unavailableRecall) Recall(context.Context, *connect.Request[sourcev1.RecallRequest]) (*connect.Response[sourcev1.RecallResponse], error) {
	return nil, &ledgerclient.UnavailableError{Operation: "recall", Err: errors.New("source-ledger stopped")}
}

func (unavailableRecall) Wake(context.Context, *connect.Request[sourcev1.WakeRequest]) (*connect.Response[sourcev1.WakeResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

func (unavailableRecall) Zoom(context.Context, *connect.Request[sourcev1.ZoomRequest]) (*connect.Response[sourcev1.ZoomResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

func (unavailableRecall) ListSiblingEvents(context.Context, *connect.Request[sourcev1.ListSiblingEventsRequest]) (*connect.Response[sourcev1.ListSiblingEventsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unused"))
}

var _ sourceconnect.RecallServiceClient = unavailableRecall{}

func TestRecallReturnsTypedUnavailableWhenSourceLedgerIsDown(t *testing.T) {
	h := NewConnectHandler(unavailableRecall{}, nil)
	_, err := h.Recall(context.Background(), connect.NewRequest(&memoryv1.RecallRequest{Query: "durable memory", Scope: "agent-memory"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
