package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

type authProxyResolverStub struct{ url string }

func (s authProxyResolverStub) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return s.url, nil
}

type authProxyAccountsStub struct {
	accountsconnect.UnimplementedAccountsServiceHandler
}

func (authProxyAccountsStub) Login(_ context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	if req.Msg.GetEmail() != "owner@example.com" || req.Msg.GetPassword() != "correct horse battery staple" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return connect.NewResponse(&accountsv1.LoginResponse{
		Account: &accountsv1.Account{Id: "owner-1", Email: req.Msg.GetEmail()},
		Tokens:  &accountsv1.TokenPair{AccessToken: "access-token", RefreshToken: "refresh-token"},
	}), nil
}

func TestAuthProxyLoginForwardsSameOriginRequest(t *testing.T) {
	connectMux := http.NewServeMux()
	path, handler := accountsconnect.NewAccountsServiceHandler(authProxyAccountsStub{})
	connectMux.Handle(path, handler)
	authenticator := httptest.NewServer(connectMux)
	defer authenticator.Close()

	handlers := &authProxyHandlers{resolver: authProxyResolverStub{url: authenticator.URL}, doer: authenticator.Client()}
	router := mux.NewRouter()
	handlers.RegisterRoutes(router.PathPrefix("/api/v1").Subrouter())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"owner@example.com","password":"correct horse battery staple"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body authOwnerResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AccessToken != "access-token" || body.UserID != "owner-1" {
		t.Fatalf("response = %#v", body)
	}
}

func TestOwnerAuthBoundaryScopesDeploymentToken(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_DEPLOYMENT_TOKEN", "deployment-token")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := ownerAuthBoundary(next, nil, nil)

	t.Run("deployment manifest accepts the scoped token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/deployment/secrets/demo", nil)
		req.Header.Set("Authorization", "Bearer deployment-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d", response.Code)
		}
	})

	t.Run("vault route rejects the scoped token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/vaults/personal/status", nil)
		req.Header.Set("Authorization", "Bearer deployment-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", response.Code)
		}
	})
}
