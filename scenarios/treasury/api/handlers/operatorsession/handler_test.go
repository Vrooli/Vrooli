package operatorsession

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/browsersession"
	coreidentity "github.com/vrooli/api-core/identity"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/budget"
	"treasury/internal/operatorauth"
)

type accountsFixture struct{ tokens *accountsv1.TokenPair }

func (a accountsFixture) Login(_ context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	if req.Msg.GetResource() != "treasury" || req.Msg.GetEmail() != "person@example.test" {
		return nil, connect.NewError(connect.CodeUnauthenticated, context.Canceled)
	}
	return connect.NewResponse(&accountsv1.LoginResponse{Tokens: a.tokens}), nil
}
func (a accountsFixture) Refresh(context.Context, *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	return connect.NewResponse(&accountsv1.RefreshResponse{Tokens: a.tokens}), nil
}
func (a accountsFixture) Logout(context.Context, *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
	return connect.NewResponse(&accountsv1.LogoutResponse{}), nil
}

type verifierFixture struct{}

func (verifierFixture) Verify(_ context.Context, token string) (coreidentity.Principal, error) {
	if token != "opaque-access" {
		return coreidentity.Principal{}, context.Canceled
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "person-1", Verified: true}, nil
}

type headroomFixture struct{ calls int }

func (f *headroomFixture) Headroom(context.Context, string) (budget.Headroom, error) {
	f.calls++
	return budget.Headroom{BudgetID: "budget-1", Currency: "USD", AvailableMinor: 42}, nil
}

type operatorFixture struct{}

func (operatorFixture) Authorize(context.Context, http.Header) (operatorauth.Identity, error) {
	return operatorauth.Identity{Subject: "person-1"}, nil
}

func TestSignInSetsHttpOnlySessionAndNeverReturnsBearer(t *testing.T) {
	edge := &browsersession.Edge{Resource: "treasury", Accounts: func(context.Context) (browsersession.Accounts, error) {
		return accountsFixture{tokens: &accountsv1.TokenPair{
			AccessToken: "opaque-access", RefreshToken: "opaque-refresh",
			AccessTokenExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
		}}, nil
	}}
	handler := Handler{Edge: edge, Verifier: verifierFixture{}}
	request := httptest.NewRequest(http.MethodPost, path+"/sign-in", strings.NewReader(`{"email":"person@example.test","password":"secret"}`))
	request.Header.Set("Origin", "http://localhost")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), "opaque-access")
	require.NotContains(t, response.Body.String(), "opaque-refresh")
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, true, body["authenticated"])
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, cookie := range cookies {
		require.True(t, cookie.HttpOnly)
		require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		require.Equal(t, "/", cookie.Path)
	}

	statusRequest := httptest.NewRequest(http.MethodGet, path, nil)
	statusRequest.AddCookie(cookies[0])
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	require.Equal(t, http.StatusOK, statusResponse.Code)
	require.Contains(t, statusResponse.Body.String(), `"subject":"person-1"`)
}

func TestHeadroomRequiresOperatorOwnershipAndReturnsCurrentBudgetView(t *testing.T) {
	headroom := &headroomFixture{}
	handler := Handler{
		Edge: &browsersession.Edge{Resource: "treasury"}, Verifier: verifierFixture{},
		Operator: operatorFixture{}, Budgets: headroom,
		OwnsBudget: func(_ context.Context, operator, budgetID string) bool {
			return operator == "person-1" && budgetID == "budget-1"
		},
	}
	request := httptest.NewRequest(http.MethodGet, path+"/budget-headroom?budget_id=budget-1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"available_minor":42`)
	require.Equal(t, 1, headroom.calls)

	handler.OwnsBudget = func(context.Context, string, string) bool { return false }
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, request)
	require.Equal(t, http.StatusNotFound, denied.Code)
	require.Equal(t, 1, headroom.calls, "foreign budget must not be read")
}
