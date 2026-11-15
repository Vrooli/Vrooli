package mandate_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	mandateHandler "treasury/handlers/mandate"
	domain "treasury/internal/mandate"
	"treasury/internal/operatorauth"

	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	mandateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate/mandate_v1connect"
)

type mandateServiceStub struct{}

func (mandateServiceStub) Issue(context.Context, domain.IssueInput) (domain.Mandate, error) {
	return domain.Mandate{}, nil
}
func (mandateServiceStub) Revoke(context.Context, string) (domain.Mandate, error) {
	return domain.Mandate{}, nil
}
func (mandateServiceStub) CancelStanding(context.Context, string) (domain.Mandate, error) {
	return domain.Mandate{}, nil
}
func (mandateServiceStub) List(context.Context) ([]domain.Mandate, error) {
	return nil, nil
}

type sessionRevokerStub struct {
	calls    int
	operator string
	mandate  string
}

func (s *sessionRevokerStub) RevokeMandateSession(_ context.Context, operator, mandate string) error {
	s.calls++
	s.operator = operator
	s.mandate = mandate
	return nil
}

func TestRevokeMandateSessionIsOperatorOwnedAndRetrySafe(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	sessions := &sessionRevokerStub{}
	_, handler := mandateconnect.NewMandateAdminServiceHandler(mandateHandler.NewConnectHandler(authorizer, mandateServiceStub{}, sessions))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := mandateconnect.NewMandateAdminServiceClient(server.Client(), server.URL)

	request := connect.NewRequest(&mandatev1.RevokeMandateSessionRequest{MandateId: "mandate-1"})
	_, err = client.RevokeMandateSession(context.Background(), request)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, sessions.calls)

	for range 2 {
		authorized := connect.NewRequest(&mandatev1.RevokeMandateSessionRequest{MandateId: "mandate-1"})
		authorized.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
		response, callErr := client.RevokeMandateSession(context.Background(), authorized)
		require.NoError(t, callErr)
		require.True(t, response.Msg.GetRevoked())
	}
	require.Equal(t, 2, sessions.calls, "replaying an operator revocation is safe and returns the same terminal result")
	require.Equal(t, "local-operator", sessions.operator)
	require.Equal(t, "mandate-1", sessions.mandate)

	missingID := connect.NewRequest(&mandatev1.RevokeMandateSessionRequest{})
	missingID.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	_, err = client.RevokeMandateSession(context.Background(), missingID)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Equal(t, 2, sessions.calls)
}
