package treasuryadmin_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"treasury/handlers/treasuryadmin"
	"treasury/internal/instrument"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	instrumentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
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
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, registrar))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorizationv1.RegisterInstrumentRequest{Instrument: &instrumentv1.Instrument{Id: "instrument-1", MandateId: "mandate-1", Rail: "manual", CredentialReference: "vrooli/treasury/manual-1", Counterparty: "vendor.example", CapMinor: 999999}})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")

	response, err := client.RegisterInstrument(context.Background(), request)
	require.NoError(t, err)
	require.Empty(t, response.Msg.GetInstrument().GetCredentialReference())
	require.Equal(t, int64(500), response.Msg.GetInstrument().GetCapMinor(), "the domain-derived cap must replace caller input")
	require.Equal(t, "vrooli/treasury/manual-1", registrar.input.CredentialReference)
}
