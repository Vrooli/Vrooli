package accountsecurity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	security "landing-page-business-suite-api/internal/accountsecurity"
)

type Dependencies struct {
	Service          *security.Service
	UserID           func(context.Context) string
	UserEmail        func(context.Context) string
	SessionID        func(context.Context) string
	ClientIP         func(*http.Request) string
	UserAgent        func(*http.Request) string
	SecureCookies    func() bool
	Now              func() time.Time
	PublicMiddleware func(http.HandlerFunc) http.HandlerFunc
	StartPasskey     func(context.Context, string, string) (string, string, time.Time, error)
	VerifyPasskey    func(context.Context, string, string, string, string, []byte, string) (time.Time, error)
}

type Handler struct{ deps Dependencies }

func NewHandler(deps Dependencies) *Handler { return &Handler{deps: deps} }

func (h *Handler) identity(ctx context.Context) (string, string, string, error) {
	userID, email, sessionID := h.deps.UserID(ctx), h.deps.UserEmail(ctx), h.deps.SessionID(ctx)
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(sessionID) == "" {
		return "", "", "", connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	return userID, email, sessionID, nil
}

func (h *Handler) ListSessions(ctx context.Context, _ *connect.Request[lpbsv1.ListSessionsRequest]) (*connect.Response[lpbsv1.ListSessionsResponse], error) {
	userID, _, sessionID, err := h.identity(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := h.deps.Service.ListSessions(ctx, userID, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.ListSessionsResponse{Sessions: sessions}), nil
}

func (h *Handler) RevokeSession(ctx context.Context, req *connect.Request[lpbsv1.RevokeSessionRequest]) (*connect.Response[lpbsv1.RevokeSessionResponse], error) {
	userID, _, currentSessionID, err := h.identity(ctx)
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	if sessionID == currentSessionID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("use sign out to revoke the current session"))
	}
	revoked, err := h.deps.Service.RevokeSession(ctx, userID, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !revoked {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("session not found"))
	}
	return connect.NewResponse(&lpbsv1.RevokeSessionResponse{Revoked: true}), nil
}

func (h *Handler) RevokeOtherSessions(ctx context.Context, _ *connect.Request[lpbsv1.RevokeOtherSessionsRequest]) (*connect.Response[lpbsv1.RevokeOtherSessionsResponse], error) {
	userID, _, sessionID, err := h.identity(ctx)
	if err != nil {
		return nil, err
	}
	count, err := h.deps.Service.RevokeOtherSessions(ctx, userID, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.RevokeOtherSessionsResponse{RevokedCount: int32(count)}), nil
}

func (h *Handler) StartReauthentication(ctx context.Context, req *connect.Request[lpbsv1.StartReauthenticationRequest]) (*connect.Response[lpbsv1.StartReauthenticationResponse], error) {
	userID, _, sessionID, err := h.identity(ctx)
	if err != nil {
		return nil, err
	}
	expiresAt, err := h.deps.Service.StartReauthentication(ctx, userID, sessionID, req.Header().Get("X-Lpbs-Browser-Binding"), "", "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &lpbsv1.StartReauthenticationResponse{ExpiresAt: timestamppb.New(expiresAt)}
	if h.deps.StartPasskey != nil {
		options, ceremony, passkeyExpires, passkeyErr := h.deps.StartPasskey(ctx, userID, req.Header().Get("X-Lpbs-Browser-Binding"))
		if passkeyErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("unable to start passkey reauthentication"))
		}
		response.PasskeyOptionsJson, response.PasskeyCeremonyId = options, ceremony
		if !passkeyExpires.IsZero() && passkeyExpires.Before(expiresAt) {
			response.ExpiresAt = timestamppb.New(passkeyExpires)
		}
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) Reauthenticate(ctx context.Context, req *connect.Request[lpbsv1.ReauthenticateRequest]) (*connect.Response[lpbsv1.ReauthenticateResponse], error) {
	userID, _, sessionID, err := h.identity(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetPasskeyAssertion()) > 0 {
		if h.deps.VerifyPasskey == nil || strings.TrimSpace(req.Msg.GetPasskeyCeremonyId()) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey reauthentication ceremony is not valid"))
		}
		now, err := h.deps.VerifyPasskey(ctx, userID, sessionID, req.Header().Get("X-Lpbs-Browser-Binding"), req.Msg.GetPasskeyCeremonyId(), req.Msg.GetPasskeyAssertion(), req.Header().Get("User-Agent"))
		if err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("reauthentication failed"))
		}
		accessToken, _, tokenErr := h.deps.Service.IssueAccessToken(ctx, userID, sessionID)
		if tokenErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("issue reauthenticated access token"))
		}
		response := connect.NewResponse(&lpbsv1.ReauthenticateResponse{AuthenticatedAt: timestamppb.New(now), Reauthenticated: true})
		secure := h.deps.SecureCookies != nil && h.deps.SecureCookies()
		name := "access_token"
		if secure {
			name = "__Host-access_token"
		}
		response.Header().Add("Set-Cookie", (&http.Cookie{Name: name, Value: accessToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
		return response, nil
	}
	now, err := h.deps.Service.Reauthenticate(ctx, userID, sessionID, req.Msg.GetEmail(), req.Msg.GetCode(), req.Header().Get("X-Lpbs-Browser-Binding"), "", "")
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("reauthentication failed: %w", err))
	}
	accessToken, _, tokenErr := h.deps.Service.IssueAccessToken(ctx, userID, sessionID)
	if tokenErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("issue reauthenticated access token: %w", tokenErr))
	}
	response := connect.NewResponse(&lpbsv1.ReauthenticateResponse{AuthenticatedAt: timestamppb.New(now)})
	secure := h.deps.SecureCookies != nil && h.deps.SecureCookies()
	name := "access_token"
	if secure {
		name = "__Host-access_token"
	}
	response.Header().Add("Set-Cookie", (&http.Cookie{Name: name, Value: accessToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	return response, nil
}

func (h *Handler) GetSignInDeliveryStatus(ctx context.Context, req *connect.Request[lpbsv1.GetSignInDeliveryStatusRequest]) (*connect.Response[lpbsv1.GetSignInDeliveryStatusResponse], error) {
	status, reasonClass, expiresAt, err := h.deps.Service.SignInDeliveryStatus(ctx, req.Msg.GetEmail(), req.Msg.GetBrowserBinding())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &lpbsv1.GetSignInDeliveryStatusResponse{Status: status, ReasonClass: reasonClass}
	if !expiresAt.IsZero() {
		response.ExpiresAt = timestamppb.New(expiresAt)
	}
	return connect.NewResponse(response), nil
}

func RegisterRoutes(router *mux.Router, deps Dependencies, requireUserAuth func(http.HandlerFunc) http.HandlerFunc) {
	path, handler := lpbsconnect.NewAccountSecurityServiceHandler(NewHandler(deps))
	// The delivery-status procedure is deliberately public and binding-scoped;
	// every other procedure requires a customer session.
	public := http.HandlerFunc(handler.ServeHTTP)
	if deps.PublicMiddleware != nil {
		public = deps.PublicMiddleware(public)
	}
	protected := requireUserAuth(handler.ServeHTTP)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/GetSignInDeliveryStatus") {
			public.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})})
}

var _ lpbsconnect.AccountSecurityServiceHandler = (*Handler)(nil)
