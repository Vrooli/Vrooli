package treasuryadmin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/operatorauth"
)

// [REQ:TRS-P0-004] Every TreasuryAdmin method rejects the agent realm before
// its domain implementation is reached, while an operator credential crosses
// the realm boundary and reaches the owning domain (or its unavailable-dependency boundary).
func TestEveryMethodSeparatesAgentAndOperatorRealms(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)

	methods := []struct {
		name         string
		operatorCode connect.Code
		call         func(headers func(http.Header)) error
	}{
		{name: "CreateMandate", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.CreateMandateRequest{})
			set(request.Header())
			_, err := client.CreateMandate(context.Background(), request)
			return err
		}},
		{name: "RevokeMandate", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.RevokeMandateRequest{})
			set(request.Header())
			_, err := client.RevokeMandate(context.Background(), request)
			return err
		}},
		{name: "SetBudgetCaps", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.SetBudgetCapsRequest{})
			set(request.Header())
			_, err := client.SetBudgetCaps(context.Background(), request)
			return err
		}},
		{name: "SetGating", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.SetGatingRequest{})
			set(request.Header())
			_, err := client.SetGating(context.Background(), request)
			return err
		}},
		{name: "ResolveApproval", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.ResolveApprovalRequest{})
			set(request.Header())
			_, err := client.ResolveApproval(context.Background(), request)
			return err
		}},
		{name: "FreezeBudget", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.FreezeBudgetRequest{})
			set(request.Header())
			_, err := client.FreezeBudget(context.Background(), request)
			return err
		}},
		{name: "UnfreezeBudget", operatorCode: connect.CodeUnimplemented, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.UnfreezeBudgetRequest{})
			set(request.Header())
			_, err := client.UnfreezeBudget(context.Background(), request)
			return err
		}},
		{name: "RegisterInstrument", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.RegisterInstrumentRequest{})
			set(request.Header())
			_, err := client.RegisterInstrument(context.Background(), request)
			return err
		}},
	}

	for _, method := range methods {
		t.Run(method.name+" rejects agent", func(t *testing.T) {
			err := method.call(func(header http.Header) { header.Set(operatorauth.HeaderAgentToken, "valid-agent-token") })
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		})
		t.Run(method.name+" accepts operator realm", func(t *testing.T) {
			err := method.call(func(header http.Header) { header.Set(operatorauth.HeaderOperatorToken, "operator-secret") })
			require.Equal(t, method.operatorCode, connect.CodeOf(err), "operator must cross the realm boundary and reach the handler")
		})
	}
}
