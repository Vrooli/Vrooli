// Package account owns authenticated AccountService Connect transport.
//
// The generated AccountService is the protocol boundary; its business logic is
// implemented by the commerce domain.
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
)

// seam: Reader keeps Connect account transport independent from commerce.
type Reader interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*Credits, error)
	GetEntitlementsContext(context.Context, string) (*Entitlements, error)
}

// Credits and Entitlements are Connect transport DTOs. The API composition
// layer adapts commerce values to them so generated transport stays independent
// of the commerce domain implementation.
type Credits struct {
	Balance                  *shared.CreditsBalance
	DisplayCreditsLabel      string
	DisplayCreditsMultiplier float64
}

type Entitlements struct {
	Status            string
	PlanTier          string
	PriceID           string
	Features          []string
	BillingCycleStart int
	Credits           *shared.CreditsBalance
	Subscription      *shared.SubscriptionStatus
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
