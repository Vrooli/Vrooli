package account

import (
	"context"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalaccount "landing-page-react-vite-api/internal/account"
)

// userHeader carries the caller identity for account-scoped RPCs. The frontend
// forwards the authenticated user's email here; the old REST surface read the
// same X-User-Email header.
const userHeader = "X-User-Email"

// Deps wires the Account Connect handler over the account service.
type Deps struct {
	Service *internalaccount.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the AccountService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetMySubscription(_ context.Context, req *connect.Request[landingv1.GetMySubscriptionRequest]) (*connect.Response[landingv1.VerifySubscriptionResponse], error) {
	user := req.Header().Get(userHeader)
	status, err := h.deps.Service.GetSubscription(user)
	if err != nil {
		h.deps.Logger.Printf("account.GetMySubscription: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.VerifySubscriptionResponse{Status: status}), nil
}

func (h *connectHandler) GetMyCredits(_ context.Context, req *connect.Request[landingv1.GetMyCreditsRequest]) (*connect.Response[landingv1.GetMyCreditsResponse], error) {
	user := req.Header().Get(userHeader)
	credits, err := h.deps.Service.GetCredits(user)
	if err != nil {
		h.deps.Logger.Printf("account.GetMyCredits: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.GetMyCreditsResponse{
		Balance:                  credits.Balance,
		DisplayCreditsLabel:      credits.DisplayCreditsLabel,
		DisplayCreditsMultiplier: credits.DisplayCreditsMultiplier,
	}), nil
}

func (h *connectHandler) GetEntitlements(_ context.Context, req *connect.Request[landingv1.GetEntitlementsRequest]) (*connect.Response[landingv1.GetEntitlementsResponse], error) {
	user := req.Header().Get(userHeader)
	ent, err := h.deps.Service.GetEntitlements(user)
	if err != nil {
		h.deps.Logger.Printf("account.GetEntitlements: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.GetEntitlementsResponse{
		Status:       ent.Status,
		PlanTier:     ent.PlanTier,
		PriceId:      ent.PriceID,
		Features:     ent.Features,
		Credits:      ent.Credits,
		Subscription: ent.Subscription,
	}), nil
}
