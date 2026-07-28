package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type fakePayments struct {
	checkout func(string, string, string, string) (*lpbsv1.CheckoutSession, error)
	portal   func(context.Context, string, string) (*lpbsv1.BillingPortalResponse, error)
}

func (f fakePayments) CreateCheckoutSession(price, success, cancel, email string) (*lpbsv1.CheckoutSession, error) {
	return f.checkout(price, success, cancel, email)
}

func (fakePayments) VerifySubscription(string) (*shared.SubscriptionStatus, error) {
	return &shared.SubscriptionStatus{}, nil
}

func (fakePayments) CancelSubscription(string) (*lpbsv1.CancelSubscriptionResponse, error) {
	return &lpbsv1.CancelSubscriptionResponse{}, nil
}

func (f fakePayments) CreateBillingPortalSession(ctx context.Context, user, returnURL string) (*lpbsv1.BillingPortalResponse, error) {
	return f.portal(ctx, user, returnURL)
}

func testConnectDependencies(payments Payments) ConnectDependencies {
	return ConnectDependencies{
		Payments:            payments,
		ValidateEmail:       func(email string) (string, error) { return email, nil },
		NormalizeRedirect:   func(raw string) (string, error) { return raw, nil },
		ValidateOptionalURL: func(raw string) (string, error) { return raw, nil },
		UserEmail:           func(context.Context) string { return "user@example.test" },
	}
}

func TestCreateCheckoutRequiresEmailForCreditTopup(t *testing.T) {
	handler := NewConnectHandler(testConnectDependencies(fakePayments{}))
	_, err := handler.CreateCheckoutSession(context.Background(), connect.NewRequest(&lpbsv1.CreateCheckoutSessionRequest{
		PriceId: "price_credits", SessionKind: lpbsv1.SessionKind_SESSION_KIND_CREDITS_TOPUP,
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", got, connect.CodeInvalidArgument)
	}
}

func TestGetBillingPortalUsesAuthenticatedIdentityAndTypedReturnURL(t *testing.T) {
	var gotUser, gotReturnURL string
	handler := NewConnectHandler(testConnectDependencies(fakePayments{portal: func(_ context.Context, user, returnURL string) (*lpbsv1.BillingPortalResponse, error) {
		gotUser, gotReturnURL = user, returnURL
		return &lpbsv1.BillingPortalResponse{Url: "https://billing.example.test/session"}, nil
	}}))
	response, err := handler.GetBillingPortal(context.Background(), connect.NewRequest(&lpbsv1.GetBillingPortalRequest{ReturnUrl: "https://app.example.test/account"}))
	if err != nil || response.Msg.GetUrl() == "" || gotUser != "user@example.test" || gotReturnURL != "https://app.example.test/account" {
		t.Fatalf("response=%#v user=%q returnURL=%q err=%v", response, gotUser, gotReturnURL, err)
	}
}

func TestRegisterConnectRoutesServesGeneratedCheckoutProcedure(t *testing.T) {
	router := mux.NewRouter()
	RegisterConnectRoutes(router, testConnectDependencies(fakePayments{checkout: func(price, success, cancel, email string) (*lpbsv1.CheckoutSession, error) {
		if price != "price_1" || email != "buyer@example.test" || success != "/success" || cancel != "/cancel" {
			t.Fatalf("checkout inputs price=%q email=%q success=%q cancel=%q", price, email, success, cancel)
		}
		return &lpbsv1.CheckoutSession{SessionId: "cs_1", Url: "https://checkout.example.test/cs_1"}, nil
	}}), func(next http.HandlerFunc) http.HandlerFunc { return next }, func(next http.HandlerFunc) http.HandlerFunc { return next })
	server := httptest.NewServer(router)
	defer server.Close()
	client := lpbsconnect.NewLandingPagePaymentsServiceClient(server.Client(), server.URL)
	response, err := client.CreateCheckoutSession(context.Background(), connect.NewRequest(&lpbsv1.CreateCheckoutSessionRequest{
		PriceId: "price_1", CustomerEmail: "buyer@example.test", SuccessUrl: "/success", CancelUrl: "/cancel",
	}))
	if err != nil || response.Msg.GetSession().GetSessionId() != "cs_1" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}
