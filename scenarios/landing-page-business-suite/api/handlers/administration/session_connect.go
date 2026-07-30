package administration

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

// SessionConnectHandler adapts the browser cookie workflow to the generated
// AdminAuthService contract. Cookies remain HTTP response headers and are
// never copied into protobuf messages.
type SessionConnectHandler struct{ deps Dependencies }

func NewSessionConnectHandler(deps Dependencies) *SessionConnectHandler {
	return &SessionConnectHandler{deps: deps}
}

func (h *SessionConnectHandler) Login(ctx context.Context, request *connect.Request[lpbsv1.LoginRequest]) (*connect.Response[lpbsv1.AdminSessionResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("login request is required"))
	}
	r, w := connectHTTP(ctx, request.Header())
	result, err := LoginSession(r, w, LoginRequest{Email: request.Msg.GetEmail(), Password: request.Msg.GetPassword()}, h.deps)
	if err != nil {
		return nil, connect.NewError(connectCode(err.Status), errors.New(err.Message))
	}
	return connectSessionResponse(result, w), nil
}

func (h *SessionConnectHandler) Logout(ctx context.Context, request *connect.Request[lpbsv1.LogoutRequest]) (*connect.Response[lpbsv1.LogoutResponse], error) {
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("logout request is required"))
	}
	r, w := connectHTTP(ctx, request.Header())
	LogoutSession(r, w, h.deps)
	response := connect.NewResponse(&lpbsv1.LogoutResponse{Success: true})
	copyHeaders(response.Header(), w.Header())
	return response, nil
}

func (h *SessionConnectHandler) Session(ctx context.Context, request *connect.Request[lpbsv1.SessionRequest]) (*connect.Response[lpbsv1.AdminSessionResponse], error) {
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session request is required"))
	}
	r, w := connectHTTP(ctx, request.Header())
	result, authenticated := ReadSession(r, w, h.deps)
	if !authenticated {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	return connectSessionResponse(result, w), nil
}

type ResetConnectHandler struct{ deps ResetDependencies }

func NewResetConnectHandler(deps ResetDependencies) *ResetConnectHandler {
	return &ResetConnectHandler{deps: deps}
}

func (h *ResetConnectHandler) ResetDemoData(ctx context.Context, _ *connect.Request[lpbsv1.ResetDemoDataRequest]) (*connect.Response[lpbsv1.ResetDemoDataResponse], error) {
	if err := h.deps.Reset(ctx); err != nil {
		h.deps.LogError("admin_reset_failed", map[string]any{"error": err.Error()})
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to reset demo data"))
	}
	return connect.NewResponse(&lpbsv1.ResetDemoDataResponse{Reset_: true, Timestamp: h.deps.Now().UTC().Format(time.RFC3339)}), nil
}

func connectHTTP(ctx context.Context, headers http.Header) (*http.Request, *headerRecorder) {
	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://connect.local/", nil)
	r.Header = headers.Clone()
	return r, &headerRecorder{header: make(http.Header)}
}

type headerRecorder struct{ header http.Header }

func (w *headerRecorder) Header() http.Header     { return w.header }
func (*headerRecorder) Write([]byte) (int, error) { return 0, nil }
func (*headerRecorder) WriteHeader(int)           {}

func connectSessionResponse(session SessionResponse, writer *headerRecorder) *connect.Response[lpbsv1.AdminSessionResponse] {
	message := &lpbsv1.AdminSessionResponse{Email: session.Email, Authenticated: session.Authenticated, ResetEnabled: session.ResetEnabled}
	if session.SessionID != "" {
		message.SessionId = &session.SessionID
	}
	response := connect.NewResponse(message)
	copyHeaders(response.Header(), writer.Header())
	return response
}

func copyHeaders(destination, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func connectCode(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	default:
		return connect.CodeInternal
	}
}

// RegisterSessionConnectRoutes mounts each generated auth/reset procedure with
// the same authorization policy as the former REST routes.
func RegisterSessionConnectRoutes(router *mux.Router, sessions Dependencies, reset ResetDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, auth := lpbsconnect.NewAdminAuthServiceHandler(NewSessionConnectHandler(sessions))
	router.Handle(lpbsconnect.AdminAuthServiceLoginProcedure, auth).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdminAuthServiceLogoutProcedure, requireAdmin(auth.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdminAuthServiceSessionProcedure, auth).Methods(http.MethodPost)
	_, resetHandler := lpbsconnect.NewAdminResetServiceHandler(NewResetConnectHandler(reset))
	router.Handle(lpbsconnect.AdminResetServiceResetDemoDataProcedure, requireAdmin(resetHandler.ServeHTTP)).Methods(http.MethodPost)
}

var (
	_ lpbsconnect.AdminAuthServiceHandler  = (*SessionConnectHandler)(nil)
	_ lpbsconnect.AdminResetServiceHandler = (*ResetConnectHandler)(nil)
)
