package ledgeradmin

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger"
	"treasury/internal/ledger"
	"treasury/internal/operatorauth"
)

type testAuthorizer struct{ err error }

func (a testAuthorizer) Authorize(context.Context, http.Header) (operatorauth.Identity, error) {
	return operatorauth.Identity{Subject: "operator"}, a.err
}

type testReader struct {
	externalID, status string
	limit              int
	values             []ledger.Emission
}

func (r *testReader) List(_ context.Context, externalID, status string, limit int) ([]ledger.Emission, error) {
	r.externalID, r.status, r.limit = externalID, status, limit
	return r.values, nil
}

func TestListEmissionsRequiresOperatorIdentity(t *testing.T) {
	handler := NewHandler(testAuthorizer{err: operatorauth.ErrRequired}, &testReader{})
	_, err := handler.ListEmissions(context.Background(), connect.NewRequest(&ledgerpb.ListEmissionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestListEmissionsReturnsOccurrenceDeliveryEvidence(t *testing.T) {
	created := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	accepted := created.Add(time.Second)
	reader := &testReader{values: []ledger.Emission{{
		ID: "occurrence:ledger", SettlementID: "occurrence", ExternalID: "base:0xabc:4", AdapterID: "treasury-base",
		AccountID: "account", BookID: "book", AmountMinor: 1250, Currency: "USD", Basis: "authoritative",
		Status: ledger.StatusAccepted, Attempts: 2, OccurredAt: created, FetchedAt: created,
		CreatedAt: created, AcceptedAt: accepted,
	}}}
	request := connect.NewRequest(&ledgerpb.ListEmissionsRequest{ExternalId: "base:0xabc:4", Status: "accepted", Limit: 25})
	response, err := NewHandler(testAuthorizer{}, reader).ListEmissions(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "base:0xabc:4", reader.externalID)
	require.Equal(t, "accepted", reader.status)
	require.Equal(t, 25, reader.limit)
	require.Len(t, response.Msg.Emissions, 1)
	emission := response.Msg.Emissions[0]
	require.Equal(t, "occurrence:ledger", emission.Id)
	require.Equal(t, "base:0xabc:4", emission.ExternalId)
	require.Equal(t, "accepted", emission.Status)
	require.Equal(t, int32(2), emission.Attempts)
	require.Equal(t, "account", emission.AccountId)
	require.Equal(t, "book", emission.BookId)
	require.EqualValues(t, 1250, emission.AmountMinor)
}

func TestListEmissionsDoesNotMaskAuthorizationOutage(t *testing.T) {
	handler := NewHandler(testAuthorizer{err: errors.Join(operatorauth.ErrUnavailable, errors.New("token provider down"))}, &testReader{})
	_, err := handler.ListEmissions(context.Background(), connect.NewRequest(&ledgerpb.ListEmissionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
