package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

type authProxyResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type authProxyHandlers struct {
	resolver authProxyResolver
	doer     *http.Client
}

type authCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authRefresh struct {
	RefreshToken string `json:"refresh_token"`
}

type authRegistration struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
}

type authOwnerResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Email        string `json:"email,omitempty"`
	UserID       string `json:"user_id,omitempty"`
}

func newAuthProxyHandlers() *authProxyHandlers {
	return &authProxyHandlers{
		resolver: discovery.DefaultResolver(),
		doer:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *authProxyHandlers) RegisterRoutes(router *mux.Router) {
	// The identity provider remains Connect-only; browsers call this same-origin
	// edge and never discover or contact the authenticator directly.
	router.HandleFunc("/auth/login", h.login).Methods(http.MethodPost)
	router.HandleFunc("/auth/register", h.register).Methods(http.MethodPost)
	router.HandleFunc("/auth/refresh", h.refresh).Methods(http.MethodPost)
}

func (h *authProxyHandlers) login(w http.ResponseWriter, r *http.Request) {
	var input authCredentials
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil {
		writeAuthProxyError(w, http.StatusBadRequest, "invalid login request")
		return
	}
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		writeAuthProxyError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	owner, err := h.forwardLogin(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeAuthProxyJSON(w, http.StatusOK, owner)
}

func (h *authProxyHandlers) register(w http.ResponseWriter, r *http.Request) {
	var input authRegistration
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil {
		writeAuthProxyError(w, http.StatusBadRequest, "invalid registration request")
		return
	}
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		writeAuthProxyError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	owner, err := h.forwardRegister(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeAuthProxyJSON(w, http.StatusOK, owner)
}

func (h *authProxyHandlers) refresh(w http.ResponseWriter, r *http.Request) {
	var input authRefresh
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil || strings.TrimSpace(input.RefreshToken) == "" {
		writeAuthProxyError(w, http.StatusBadRequest, "refresh token is required")
		return
	}
	client, err := h.client(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	response, err := client.Refresh(r.Context(), connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: input.RefreshToken}))
	if err != nil {
		h.writeError(w, err)
		return
	}
	owner, err := projectAuthResponse(nil, response.Msg.GetTokens())
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeAuthProxyJSON(w, http.StatusOK, owner)
}

func (h *authProxyHandlers) forwardLogin(ctx context.Context, input authCredentials) (authOwnerResponse, error) {
	client, err := h.client(ctx)
	if err != nil {
		return authOwnerResponse{}, err
	}
	response, err := client.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: strings.TrimSpace(input.Email), Password: input.Password}))
	if err != nil {
		return authOwnerResponse{}, err
	}
	return projectAuthResponse(response.Msg.GetAccount(), response.Msg.GetTokens())
}

func (h *authProxyHandlers) forwardRegister(ctx context.Context, input authRegistration) (authOwnerResponse, error) {
	client, err := h.client(ctx)
	if err != nil {
		return authOwnerResponse{}, err
	}
	response, err := client.Register(ctx, connect.NewRequest(&accountsv1.RegisterRequest{
		Email: strings.TrimSpace(input.Email), Password: input.Password, Username: strings.TrimSpace(input.Username),
	}))
	if err != nil {
		return authOwnerResponse{}, err
	}
	return projectAuthResponse(response.Msg.GetAccount(), response.Msg.GetTokens())
}

func (h *authProxyHandlers) client(ctx context.Context) (accountsconnect.AccountsServiceClient, error) {
	if h.resolver == nil {
		return nil, errors.New("scenario-authenticator is unavailable")
	}
	base, err := h.resolver.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("scenario-authenticator is unavailable")
	}
	return accountsconnect.NewAccountsServiceClient(h.doer, strings.TrimRight(base, "/")), nil
}

func projectAuthResponse(account *accountsv1.Account, tokens *accountsv1.TokenPair) (authOwnerResponse, error) {
	if tokens == nil || strings.TrimSpace(tokens.GetAccessToken()) == "" {
		return authOwnerResponse{}, errors.New("scenario-authenticator returned no access token")
	}
	response := authOwnerResponse{AccessToken: tokens.GetAccessToken(), RefreshToken: tokens.GetRefreshToken()}
	if account != nil {
		response.Email = account.GetEmail()
		response.UserID = account.GetId()
	}
	return response, nil
}

func (h *authProxyHandlers) writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	message := "scenario-authenticator is unavailable"
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		status, message = http.StatusUnauthorized, "invalid credentials"
	case connect.CodeAlreadyExists:
		status, message = http.StatusConflict, "account already exists"
	case connect.CodeInvalidArgument:
		status, message = http.StatusBadRequest, err.Error()
	}
	writeAuthProxyError(w, status, message)
}

func writeAuthProxyJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAuthProxyError(w http.ResponseWriter, status int, message string) {
	writeAuthProxyJSON(w, status, map[string]string{"error": "authenticator_request_failed", "message": message})
}
