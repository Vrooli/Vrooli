package instrument_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	handler "treasury/handlers/instrument"
	"treasury/internal/instrument"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	instrumentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	instrumentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument/instrument_v1connect"
)

type instrumentRegistrar struct{ input instrument.RegisterInput }

func (r *instrumentRegistrar) Register(_ context.Context, input instrument.RegisterInput) (instrument.Instrument, error) {
	r.input = input
	return instrument.Instrument{ID: input.ID, BookID: "book-1", MandateID: input.MandateID, Rail: input.Rail, CredentialReference: input.CredentialReference, CapMinor: 500, Currency: "USD", Counterparty: input.Counterparty, ExpiresAt: time.Unix(100, 0).UTC()}, nil
}

func TestRegisterInstrumentNeverReturnsCredentialReference(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	registrar := &instrumentRegistrar{}
	_, transport := instrumentconnect.NewInstrumentAdminServiceHandler(handler.NewConnectHandler(authorizer, registrar))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := instrumentconnect.NewInstrumentAdminServiceClient(server.Client(), server.URL)
	agentRequest := connect.NewRequest(&instrumentv1.RegisterInstrumentRequest{Instrument: &instrumentv1.Instrument{Id: "agent-instrument", MandateId: "mandate-1", Rail: "manual"}})
	agentRequest.Header().Set(operatorauth.HeaderAgentToken, "valid-agent-token")
	_, err = client.RegisterInstrument(context.Background(), agentRequest)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Empty(t, registrar.input.ID, "agent realm is denied before instrument registration")
	request := connect.NewRequest(&instrumentv1.RegisterInstrumentRequest{Instrument: &instrumentv1.Instrument{Id: "instrument-1", MandateId: "mandate-1", Rail: "manual", CredentialReference: "vrooli/treasury/manual-1", Counterparty: "vendor.example", CapMinor: 999999}})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")

	response, err := client.RegisterInstrument(context.Background(), request)
	require.NoError(t, err)
	require.Empty(t, response.Msg.GetInstrument().GetCredentialReference())
	require.Equal(t, int64(500), response.Msg.GetInstrument().GetCapMinor(), "the domain-derived cap must replace caller input")
	require.Equal(t, "vrooli/treasury/manual-1", registrar.input.CredentialReference)
}
