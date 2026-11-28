package journal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
)

type rememberJournal struct {
	sourceconnect.JournalServiceClient
	requests []*connect.Request[sourcev1.AppendEntryRequest]
	fail     int
}

func (s *rememberJournal) AppendEntry(_ context.Context, req *connect.Request[sourcev1.AppendEntryRequest]) (*connect.Response[sourcev1.AppendEntryResponse], error) {
	s.requests = append(s.requests, req)
	if len(s.requests) == s.fail {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("append unavailable"))
	}
	return connect.NewResponse(&sourcev1.AppendEntryResponse{Entry: &sourcev1.Entry{
		Id: fmt.Sprint(len(s.requests)), Body: req.Msg.Body, Kind: req.Msg.Kind,
		Evidence: req.Msg.Evidence, RoutingHint: req.Msg.RoutingHint,
	}}), nil
}

func TestRememberPreservesEachObservationAndForwardsProvenance(t *testing.T) {
	j := &rememberJournal{}
	h := NewConnectHandler(j, nil)
	req := connect.NewRequest(&memoryv1.RememberRequest{
		Observations: []string{" Browser capture failed.\n", "Local fix verified."},
		Evidence:     "test run 18",
		RoutingHint:  "browser",
		RequestKey:   "capture-1",
	})
	req.Header().Set("X-Correlation-Id", "correlation")
	resp, err := h.Remember(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.Entries, 2)
	for i, actual := range j.requests {
		require.Equal(t, req.Msg.Observations[i], actual.Msg.Body)
		require.Equal(t, req.Msg.Observations[i], resp.Msg.Entries[i].Body)
		require.Equal(t, req.Msg.Evidence, actual.Msg.Evidence)
		require.Equal(t, req.Msg.RoutingHint, actual.Msg.RoutingHint)
		require.Equal(t, "agent-memory", actual.Msg.Scope)
		require.Equal(t, fmt.Sprintf("remember/capture-1/%d", i), actual.Msg.RequestKey)
		require.Equal(t, "correlation", actual.Header().Get("X-Correlation-Id"))
	}
}

func TestRememberValidatesBatchBeforeWriteAndReturnsPartialReceipt(t *testing.T) {
	j := &rememberJournal{fail: 2}
	h := NewConnectHandler(j, nil)
	_, err := h.Remember(t.Context(), connect.NewRequest(&memoryv1.RememberRequest{Observations: []string{"observed", "\t"}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Empty(t, j.requests)

	_, err = h.Remember(t.Context(), connect.NewRequest(&memoryv1.RememberRequest{Observations: []string{"first", "second", "third"}}))
	var failure *connect.Error
	require.ErrorAs(t, err, &failure)
	require.Equal(t, connect.CodeUnavailable, failure.Code())
	require.Len(t, j.requests, 2)
	require.Len(t, failure.Details(), 1)
	receipt, err := failure.Details()[0].Value()
	require.NoError(t, err)
	require.Len(t, receipt.(*memoryv1.RememberResponse).Entries, 1)
}
