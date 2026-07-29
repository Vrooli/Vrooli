// Package account owns authenticated account-read Connect transport.
package account

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	accountdomain "landing-page-business-suite-api/internal/account"
)

type Reader interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*accountdomain.CreditsEnvelope, error)
	GetEntitlementsContext(context.Context, string) (*accountdomain.EntitlementPayload, error)
}

type Handler struct {
	reader    Reader
	userEmail func(context.Context) string
}

func NewHandler(reader Reader, userEmail func(context.Context) string) *Handler {
	return &Handler{reader: reader, userEmail: userEmail}
}

func (h *Handler) user(ctx context.Context) (string, error) {
	user := h.userEmail(ctx)
	if user == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return user, nil
}

func (h *Handler) GetMySubscription(ctx context.Context, _ *connect.Request[lpbsv1.GetMySubscriptionRequest]) (*connect.Response[shared.VerifySubscriptionResponse], error) {
	user, err := h.user(ctx)
	if err != nil {
		return nil, err
	}
	status, err := h.reader.GetSubscriptionContext(ctx, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get subscription: %w", err))
	}
	return connect.NewResponse(&shared.VerifySubscriptionResponse{Status: status}), nil
}

func (h *Handler) GetMyCredits(ctx context.Context, _ *connect.Request[lpbsv1.GetMyCreditsRequest]) (*connect.Response[lpbsv1.GetMyCreditsResponse], error) {
	user, err := h.user(ctx)
	if err != nil {
		return nil, err
	}
	credits, err := h.reader.GetCreditsContext(ctx, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get credits: %w", err))
	}
	return connect.NewResponse(&lpbsv1.GetMyCreditsResponse{Balance: credits.Balance, DisplayCreditsLabel: credits.DisplayCreditsLabel, DisplayCreditsMultiplier: credits.DisplayCreditsMultiplier}), nil
}

func (h *Handler) GetEntitlements(ctx context.Context, _ *connect.Request[lpbsv1.GetEntitlementsRequest]) (*connect.Response[lpbsv1.GetEntitlementsResponse], error) {
	user, err := h.user(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := h.reader.GetEntitlementsContext(ctx, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get entitlements: %w", err))
	}
	if payload.BillingCycleStart < -1<<31 || payload.BillingCycleStart > 1<<31-1 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("billing cycle start is outside the protocol range"))
	}
	return connect.NewResponse(&lpbsv1.GetEntitlementsResponse{Status: payload.Status, PlanTier: payload.PlanTier, PriceId: payload.PriceID, Features: payload.Features, Credits: payload.Credits, Subscription: payload.Subscription, BillingCycleStart: int32(payload.BillingCycleStart)}), nil
}

// RegisterRoutes mounts all generated AccountService procedures behind the
// existing user-auth middleware. Identity remains server-derived from claims.
func RegisterRoutes(router *mux.Router, reader Reader, userEmail func(context.Context) string, requireUserAuth func(http.HandlerFunc) http.HandlerFunc) {
	path, handler := lpbsconnect.NewAccountServiceHandler(NewHandler(reader, userEmail))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: requireUserAuth(handler.ServeHTTP)})
}
