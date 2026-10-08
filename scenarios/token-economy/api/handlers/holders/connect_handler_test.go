package holders_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	accessH "token-economy/handlers/access"
	holdersH "token-economy/handlers/holders"
	internalaccess "token-economy/internal/access"
	domain "token-economy/internal/holders"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

type serviceStub struct {
	subject string
	view    domain.View
	calls   int
}

func (*serviceStub) Add(_ context.Context, input domain.AddInput) (domain.Holder, error) {
	return domain.Holder{DisplayName: input.DisplayName, AuthenticatorSubject: input.AuthenticatorSubject}, nil
}

func (*serviceStub) Get(context.Context, string) (domain.Holder, error) {
	return domain.Holder{}, nil
}

func (*serviceStub) List(context.Context) ([]domain.Holder, error) {
	return nil, nil
}

func (s *serviceStub) View(_ context.Context, subject string) (domain.View, error) {
	s.calls++
	s.subject = subject
	return s.view, nil
}

// [REQ:TKE-P0-006] Holder access fails closed before the holder domain runs
// when scenario-authenticator validation is unavailable.
func TestViewEconomyFailsClosedWhenAuthenticatorUnavailable(t *testing.T) {
	service := &serviceStub{}
	module := accessH.Module(nil, nil, holdersH.NewConnectHandler(service, nil), nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := accessconnect.NewHolderServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.ViewEconomyRequest{})
	request.Header().Set("Authorization", "Bearer otherwise-valid")
	_, err := client.ViewEconomy(context.Background(), request)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.Zero(t, service.calls)
}

type validatorStub struct{}

func (validatorStub) Validate(context.Context, string) (internalaccess.Identity, error) {
	return internalaccess.Identity{Subject: "auth:sam", Scopes: []string{internalaccess.ScopeHolder}}, nil
}

// [REQ:TKE-P0-012] The authenticated holder transport returns each ordered
// event with its reason and the event-derived balance.
func TestViewEconomyReturnsCompleteExplainableHistory(t *testing.T) {
	createdAt := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC)
	service := &serviceStub{view: domain.View{
		Holder: domain.Holder{ID: "holder-sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam", CreatedAt: createdAt},
		History: domain.History{
			Events: []domain.HistoryEvent{
				{ID: "grant-1", TokenTypeID: "chores", Amount: 7, Kind: "credit", Reason: "weekly chores", CreatedAt: createdAt.Add(time.Minute)},
				{ID: "redeem-1", TokenTypeID: "chores", Amount: 2, Kind: "debit", Reason: "movie reward", CreatedAt: createdAt.Add(2 * time.Minute)},
			},
			Balances: []domain.Balance{{TokenTypeID: "chores", Amount: 5}},
		},
	}}
	module := accessH.Module(nil, nil, holdersH.NewConnectHandler(service, nil), nil, nil, nil, nil, validatorStub{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := accessconnect.NewHolderServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.ViewEconomyRequest{})
	request.Header().Set("Authorization", "Bearer valid-holder")
	response, err := client.ViewEconomy(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "auth:sam", service.subject)
	require.Equal(t, 1, service.calls)
	require.Equal(t, "holder-sam", response.Msg.Holder.Id)
	require.Equal(t, "Sam", response.Msg.Holder.DisplayName)
	require.Len(t, response.Msg.Events, 2)
	require.Equal(t, "weekly chores", response.Msg.Events[0].Reason)
	require.Equal(t, "movie reward", response.Msg.Events[1].Reason)
	require.Equal(t, int64(5), response.Msg.Balances[0].Amount)
}
