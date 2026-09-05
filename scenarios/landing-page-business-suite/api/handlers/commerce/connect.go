// Package billing owns checkout and customer-portal transport.
package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// Payments is the billing-domain capability exposed through the generated
// LandingPagePaymentsService contract.
type Payments interface {
	CreateCheckoutSession(string, string, string, string) (*lpbsv1.CheckoutSession, error)
	VerifySubscription(string) (*shared.SubscriptionStatus, error)
	CancelSubscription(string) (*lpbsv1.CancelSubscriptionResponse, error)
	CreateBillingPortalSession(context.Context, string, string) (*lpbsv1.BillingPortalResponse, error)
}

type AttributedPayments interface {
	CreateCheckoutSessionWithAttribution(string, string, string, string, commerce.Attribution) (*lpbsv1.CheckoutSession, error)
}

type ConnectDependencies struct {
	Payments            Payments
	ValidateEmail       func(string) (string, error)
	NormalizeRedirect   func(string) (string, error)
	ValidateOptionalURL func(string) (string, error)
	UserEmail           func(context.Context) string
}

type ConnectHandler struct{ deps ConnectDependencies }

func NewConnectHandler(deps ConnectDependencies) *ConnectHandler { return &ConnectHandler{deps: deps} }

func invalidArgument(format string, args ...any) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(format, args...))
}

func internal(operation string, err error) error {
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", operation, err))
}

func (h *ConnectHandler) CreateCheckoutSession(_ context.Context, request *connect.Request[lpbsv1.CreateCheckoutSessionRequest]) (*connect.Response[lpbsv1.CreateCheckoutSessionResponse], error) {
	input := request.Msg
	priceID := strings.TrimSpace(input.GetPriceId())
	if priceID == "" {
		return nil, invalidArgument("price_id is required")
	}
	email := strings.TrimSpace(input.GetCustomerEmail())
	if input.GetSessionKind() == lpbsv1.SessionKind_SESSION_KIND_CREDITS_TOPUP && email == "" {
		return nil, invalidArgument("customer_email is required for credit purchases")
	}
	if email != "" {
		var err error
		email, err = h.deps.ValidateEmail(email)
		if err != nil {
			return nil, invalidArgument("invalid customer_email: %v", err)
		}
	}
	successURL, err := h.deps.NormalizeRedirect(input.GetSuccessUrl())
	if err != nil {
		return nil, invalidArgument("invalid success_url: %v", err)
	}
	cancelURL, err := h.deps.NormalizeRedirect(input.GetCancelUrl())
	if err != nil {
		return nil, invalidArgument("invalid cancel_url: %v", err)
	}
	var session *lpbsv1.CheckoutSession
	if attributed, ok := h.deps.Payments.(AttributedPayments); ok {
		session, err = attributed.CreateCheckoutSessionWithAttribution(priceID, successURL, cancelURL, email, commerce.Attribution{VisitorID: input.GetVisitorId(), UTMSource: input.GetUtmSource(), UTMMedium: input.GetUtmMedium(), UTMCampaign: input.GetUtmCampaign(), ReferrerKind: input.GetReferrerKind(), CountryCode: input.GetCountryCode()})
	} else {
		session, err = h.deps.Payments.CreateCheckoutSession(priceID, successURL, cancelURL, email)
	}
	if err != nil {
		return nil, internal("create checkout session", err)
	}
	return connect.NewResponse(&lpbsv1.CreateCheckoutSessionResponse{Session: session}), nil
}

func (h *ConnectHandler) VerifySubscription(_ context.Context, request *connect.Request[lpbsv1.VerifySubscriptionRequest]) (*connect.Response[shared.VerifySubscriptionResponse], error) {
	identity := strings.TrimSpace(request.Msg.GetUserIdentity())
	if identity == "" {
		return nil, invalidArgument("user_identity is required")
	}
	status, err := h.deps.Payments.VerifySubscription(identity)
	if err != nil {
		return nil, internal("verify subscription", err)
	}
	return connect.NewResponse(&shared.VerifySubscriptionResponse{Status: status}), nil
}

func (h *ConnectHandler) CancelSubscription(_ context.Context, request *connect.Request[lpbsv1.CancelSubscriptionRequest]) (*connect.Response[lpbsv1.CancelSubscriptionResponse], error) {
	identity := strings.TrimSpace(request.Msg.GetUserIdentity())
	if identity == "" {
		return nil, invalidArgument("user_identity is required")
	}
	response, err := h.deps.Payments.CancelSubscription(identity)
	if err != nil {
		return nil, internal("cancel subscription", err)
	}
	return connect.NewResponse(response), nil
}

func (h *ConnectHandler) GetBillingPortal(ctx context.Context, request *connect.Request[lpbsv1.GetBillingPortalRequest]) (*connect.Response[lpbsv1.BillingPortalResponse], error) {
	user := h.deps.UserEmail(ctx)
	if user == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	returnURL, err := h.deps.ValidateOptionalURL(request.Msg.GetReturnUrl())
	if err != nil {
		return nil, invalidArgument("invalid return URL: %v", err)
	}
	response, err := h.deps.Payments.CreateBillingPortalSession(ctx, user, returnURL)
	if err != nil {
		return nil, internal("create billing portal", err)
	}
	return connect.NewResponse(response), nil
}

// RegisterConnectRoutes mounts each generated procedure independently so its
// authorization matches the pre-existing operation, rather than applying one
// blanket policy to a mixed public/authenticated service.
func RegisterConnectRoutes(router *mux.Router, deps ConnectDependencies, requireUserAuth, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewLandingPagePaymentsServiceHandler(NewConnectHandler(deps))
	mount := func(path string, middleware func(http.HandlerFunc) http.HandlerFunc) {
		handler := http.HandlerFunc(service.ServeHTTP)
		if middleware != nil {
			handler = middleware(handler)
		}
		router.Handle(path, handler).Methods(http.MethodPost)
	}
	mount(lpbsconnect.LandingPagePaymentsServiceCreateCheckoutSessionProcedure, nil)
	mount(lpbsconnect.LandingPagePaymentsServiceVerifySubscriptionProcedure, nil)
	mount(lpbsconnect.LandingPagePaymentsServiceCancelSubscriptionProcedure, requireAdmin)
	mount(lpbsconnect.LandingPagePaymentsServiceGetBillingPortalProcedure, requireUserAuth)
}
