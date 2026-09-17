package administration

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"golang.org/x/crypto/bcrypt"
	admin "landing-page-business-suite-api/internal/administration"
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
	r, w := connectHTTP(ctx, request.Header(), request.Peer().Addr)
	result, err := LoginSession(r, w, LoginRequest{Email: request.Msg.GetEmail(), Password: request.Msg.GetPassword(), TOTPCode: request.Msg.GetTotpCode(), PasskeyAssertion: request.Msg.GetPasskeyAssertion(), PasskeyCeremonyID: request.Msg.GetPasskeyCeremonyId()}, h.deps)
	if err != nil {
		connectErr := connect.NewError(connectCode(err.Status), errors.New(err.Message))
		// The kind travels as a header so clients can branch without parsing
		// human-readable messages.
		connectErr.Meta().Set("X-Lpbs-Auth-Reason", err.Kind)
		return nil, connectErr
	}
	return connectSessionResponse(result, w), nil
}

func (h *SessionConnectHandler) Logout(ctx context.Context, request *connect.Request[lpbsv1.LogoutRequest]) (*connect.Response[lpbsv1.LogoutResponse], error) {
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("logout request is required"))
	}
	r, w := connectHTTP(ctx, request.Header(), request.Peer().Addr)
	LogoutSession(r, w, h.deps)
	response := connect.NewResponse(&lpbsv1.LogoutResponse{Success: true})
	copyHeaders(response.Header(), w.Header())
	return response, nil
}

func (h *SessionConnectHandler) Session(ctx context.Context, request *connect.Request[lpbsv1.SessionRequest]) (*connect.Response[lpbsv1.AdminSessionResponse], error) {
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session request is required"))
	}
	r, w := connectHTTP(ctx, request.Header(), request.Peer().Addr)
	result, authenticated := ReadSession(r, w, h.deps)
	if !authenticated {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	return connectSessionResponse(result, w), nil
}

func (h *SessionConnectHandler) Reauthenticate(ctx context.Context, request *connect.Request[lpbsv1.ReauthenticateRequest]) (*connect.Response[lpbsv1.ReauthenticateResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetPassword()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("password is required"))
	}
	r, _ := connectHTTP(ctx, request.Header(), request.Peer().Addr)
	session, err := h.deps.Sessions.GetSession(r, sessionName)
	if err != nil || session == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	email, _ := session.Values["email"].(string)
	sessionID, _ := session.Values["session_id"].(string)
	if strings.TrimSpace(email) == "" || strings.TrimSpace(sessionID) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	if h.deps.Throttle != nil {
		allowed, throttleErr := h.deps.Throttle.Exceeded(ctx, "admin-reauth:"+email, admin.ThrottleRule{Limit: 5, Window: 15 * time.Minute})
		if throttleErr != nil || allowed {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("reauthentication temporarily unavailable"))
		}
	}
	hash, err := h.deps.Auth.PasswordHash(ctx, email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(request.Msg.GetPassword())) != nil {
		if h.deps.Throttle != nil {
			_ = h.deps.Throttle.Record(ctx, "admin-reauth:"+email)
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("reauthentication failed"))
	}
	if h.deps.MFA == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("two-factor authentication is required"))
	}
	enabled, mfaErr := h.deps.MFA.Enabled(ctx, email)
	if mfaErr != nil || !enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("two-factor authentication is required"))
	}
	code := strings.TrimSpace(request.Msg.GetTotpCode())
	if code == "" {
		code = strings.TrimSpace(request.Msg.GetRecoveryCode())
	}
	var factorErr error
	if len(request.Msg.GetPasskeyAssertion()) > 0 {
		if h.deps.Passkeys == nil || strings.TrimSpace(request.Msg.GetPasskeyCeremonyId()) == "" {
			factorErr = errors.New("passkey reauthentication unavailable")
		} else {
			factorErr = h.deps.Passkeys.VerifyLogin(ctx, email, request.Msg.GetPasskeyCeremonyId(), request.Msg.GetPasskeyAssertion(), request.Header().Get("X-Lpbs-Browser-Binding"), r.UserAgent())
		}
	} else {
		factorErr = h.deps.MFA.Verify(ctx, email, code)
	}
	if factorErr != nil {
		if h.deps.Throttle != nil {
			_ = h.deps.Throttle.Record(ctx, "admin-reauth:"+email)
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("reauthentication failed"))
	}
	if len(request.Msg.GetPasskeyAssertion()) == 0 && len(code) != 6 && h.deps.SecurityEvents != nil {
		_ = h.deps.SecurityEvents.Record(ctx, "recovery_code_used", email, sessionID, r.RemoteAddr, r.UserAgent(), nil)
	}
	marker, ok := h.deps.Auth.(ReauthenticationMarker)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("reauthentication unavailable"))
	}
	if err := marker.MarkReauthenticated(ctx, sessionID, email); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("reauthentication unavailable"))
	}
	if h.deps.Throttle != nil {
		_ = h.deps.Throttle.Reset(ctx, "admin-reauth:"+email)
	}
	if h.deps.SecurityEvents != nil {
		_ = h.deps.SecurityEvents.Record(ctx, "admin_reauthenticated", email, sessionID, r.RemoteAddr, r.UserAgent(), nil)
	}
	return connect.NewResponse(&lpbsv1.ReauthenticateResponse{Reauthenticated: true}), nil
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

func connectHTTP(ctx context.Context, headers http.Header, peerAddr string) (*http.Request, *headerRecorder) {
	// Session helpers require an HTTP request only for its context, headers,
	// and peer address. The peer address keeps per-client throttling and
	// session audit fields real; without it every caller shares one bucket.
	// Constructing the synthetic URL structurally keeps this transport adapter
	// independent of any deployment endpoint.
	r := &http.Request{
		Method:     http.MethodPost,
		URL:        &url.URL{Scheme: "http", Host: "connect.local", Path: "/"},
		Header:     headers.Clone(),
		RemoteAddr: peerAddr,
	}
	r = r.WithContext(ctx)
	return r, &headerRecorder{header: make(http.Header)}
}

type headerRecorder struct{ header http.Header }

func (w *headerRecorder) Header() http.Header     { return w.header }
func (*headerRecorder) Write([]byte) (int, error) { return 0, nil }
func (*headerRecorder) WriteHeader(int)           {}

func connectSessionResponse(session SessionResponse, writer *headerRecorder) *connect.Response[lpbsv1.AdminSessionResponse] {
	message := &lpbsv1.AdminSessionResponse{Email: session.Email, Authenticated: session.Authenticated, ResetEnabled: session.ResetEnabled, Assurance: session.Assurance}
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
	case http.StatusPreconditionRequired:
		return connect.CodeFailedPrecondition
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
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
	router.Handle(lpbsconnect.AdminAuthServiceReauthenticateProcedure, requireAdmin(auth.ServeHTTP)).Methods(http.MethodPost)
	_, resetHandler := lpbsconnect.NewAdminResetServiceHandler(NewResetConnectHandler(reset))
	router.Handle(lpbsconnect.AdminResetServiceResetDemoDataProcedure, requireAdmin(resetHandler.ServeHTTP)).Methods(http.MethodPost)
}

var (
	_ lpbsconnect.AdminAuthServiceHandler  = (*SessionConnectHandler)(nil)
	_ lpbsconnect.AdminResetServiceHandler = (*ResetConnectHandler)(nil)
)
