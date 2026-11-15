package funding_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	fundinghandler "treasury/handlers/funding"
	"treasury/internal/operatorauth"

	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
)

type intentServiceStub struct {
	operator string
	called   int
}

func (s *intentServiceStub) CreateFundingIntent(_ context.Context, operator string, _ *fundingv1.CreateFundingIntentRequest) (*fundingv1.FundingIntent, error) {
	s.operator, s.called = operator, s.called+1
	return &fundingv1.FundingIntent{Id: "intent-1"}, nil
}
func (s *intentServiceStub) GetFundingIntent(_ context.Context, operator, _ string) (*fundingv1.FundingIntent, error) {
	s.operator, s.called = operator, s.called+1
	return &fundingv1.FundingIntent{Id: "intent-1"}, nil
}
func (s *intentServiceStub) ConfirmFundingTransfer(_ context.Context, operator string, _ *fundingv1.ConfirmFundingTransferRequest) (*fundingv1.FundingIntent, error) {
	s.operator, s.called = operator, s.called+1
	return &fundingv1.FundingIntent{Id: "intent-1"}, nil
}
func (s *intentServiceStub) ListFundingIntents(_ context.Context, operator, _ string) ([]*fundingv1.FundingIntent, error) {
	s.operator, s.called = operator, s.called+1
	return []*fundingv1.FundingIntent{{Id: "intent-1"}}, nil
}

func TestFundingIntentServiceRequiresOperatorRealm(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	service := &intentServiceStub{}
	_, handler := fundingconnect.NewFundingAdminServiceHandler(fundinghandler.NewConnectHandler(authorizer, service, nil, nil))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := fundingconnect.NewFundingAdminServiceClient(server.Client(), server.URL)

	agentRequest := connect.NewRequest(&fundingv1.CreateFundingIntentRequest{Id: "intent-1"})
	agentRequest.Header().Set(operatorauth.HeaderAgentToken, "agent-token")
	_, err = client.CreateFundingIntent(context.Background(), agentRequest)
	require.Error(t, err)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Zero(t, service.called)

	operatorRequest := connect.NewRequest(&fundingv1.CreateFundingIntentRequest{Id: "intent-1"})
	operatorRequest.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	response, err := client.CreateFundingIntent(context.Background(), operatorRequest)
	require.NoError(t, err)
	require.Equal(t, "intent-1", response.Msg.GetIntent().GetId())
	require.Equal(t, "local-operator", service.operator)
	require.Equal(t, 1, service.called)
}
