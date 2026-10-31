package payments

import (
	"context"
	"errors"
	"landing-page-react-vite-api/internal/paymentsettings"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/stripe"
	"log"
	"strings"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Deps wires the LandingPagePayments Connect handler over the plan, stripe, and
// payment-settings application services.
type Deps struct {
	Stripe          *stripe.Service
	Plan            *plan.Service
	PaymentSettings *paymentsettings.Service
	Logger          *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the LandingPagePaymentsService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) CreateCheckoutSession(_ context.Context, req *connect.Request[landingv1.CreateCheckoutSessionRequest]) (*connect.Response[landingv1.CreateCheckoutSessionResponse], error) {
	m := req.Msg
	if strings.TrimSpace(m.PriceId) == "" || strings.TrimSpace(m.CustomerEmail) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id and customer_email are required"))
	}
	successURL := m.SuccessUrl
	if successURL == "" {
		successURL = "/success"
	}
	cancelURL := m.CancelUrl
	if cancelURL == "" {
		cancelURL = "/cancel"
	}

	session, err := h.deps.Stripe.CreateCheckoutSession(m.PriceId, successURL, cancelURL, m.CustomerEmail)
	if err != nil {
		h.deps.Logger.Printf("payments.CreateCheckoutSession: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.CreateCheckoutSessionResponse{Session: session}), nil
}

func (h *connectHandler) VerifySubscription(_ context.Context, req *connect.Request[landingv1.VerifySubscriptionRequest]) (*connect.Response[landingv1.VerifySubscriptionResponse], error) {
	if strings.TrimSpace(req.Msg.UserIdentity) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_identity is required"))
	}
	status, err := h.deps.Stripe.VerifySubscription(req.Msg.UserIdentity)
	if err != nil {
		h.deps.Logger.Printf("payments.VerifySubscription: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.VerifySubscriptionResponse{Status: status}), nil
}

func (h *connectHandler) CancelSubscription(_ context.Context, req *connect.Request[landingv1.CancelSubscriptionRequest]) (*connect.Response[landingv1.CancelSubscriptionResponse], error) {
	if strings.TrimSpace(req.Msg.UserIdentity) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_identity is required"))
	}
	resp, err := h.deps.Stripe.CancelSubscription(req.Msg.UserIdentity)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetPricing(_ context.Context, _ *connect.Request[landingv1.GetPricingRequest]) (*connect.Response[landingv1.GetPricingResponse], error) {
	overview, err := h.deps.Plan.GetPricingOverview()
	if err != nil {
		h.deps.Logger.Printf("payments.GetPricing: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.GetPricingResponse{Pricing: overview}), nil
}

func (h *connectHandler) GetStripeSettings(ctx context.Context, _ *connect.Request[landingv1.GetStripeSettingsRequest]) (*connect.Response[landingv1.GetStripeSettingsResponse], error) {
	record, err := h.deps.PaymentSettings.GetStripeSettings(ctx)
	if err != nil {
		h.deps.Logger.Printf("payments.GetStripeSettings: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.GetStripeSettingsResponse{
		Settings: record,
		Snapshot: h.deps.Stripe.ConfigSnapshot(),
	}), nil
}

func (h *connectHandler) UpdateStripeSettings(ctx context.Context, req *connect.Request[landingv1.UpdateStripeSettingsRequest]) (*connect.Response[landingv1.UpdateStripeSettingsResponse], error) {
	m := req.Msg
	pub := normalize(m.PublishableKey)
	secret := normalize(m.SecretKey)

	current, err := h.deps.PaymentSettings.GetStripeSettings(ctx)
	if err != nil {
		h.deps.Logger.Printf("payments.UpdateStripeSettings read current: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	needsPublishable := current == nil || strings.TrimSpace(current.PublishableKey) == ""
	needsSecret := current == nil || strings.TrimSpace(current.SecretKey) == ""
	if needsPublishable && (pub == nil || *pub == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("publishable_key is required"))
	}
	if needsSecret && (secret == nil || *secret == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("secret_key is required"))
	}

	record, err := h.deps.PaymentSettings.SaveStripeSettings(ctx, paymentsettings.Input{
		PublishableKey: pub,
		SecretKey:      secret,
		WebhookSecret:  normalize(m.WebhookSecret),
		DashboardURL:   normalize(m.DashboardUrl),
	})
	if err != nil {
		h.deps.Logger.Printf("payments.UpdateStripeSettings save: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := h.deps.Stripe.RefreshConfig(ctx); err != nil {
		h.deps.Logger.Printf("payments.UpdateStripeSettings refresh: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&landingv1.UpdateStripeSettingsResponse{
		Settings: record,
		Snapshot: h.deps.Stripe.ConfigSnapshot(),
	}), nil
}

func (h *connectHandler) GetBillingPortal(_ context.Context, _ *connect.Request[landingv1.GetBillingPortalRequest]) (*connect.Response[landingv1.BillingPortalResponse], error) {
	return connect.NewResponse(&landingv1.BillingPortalResponse{
		Url: "https://dashboard.stripe.com/test/customers",
	}), nil
}

func normalize(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
