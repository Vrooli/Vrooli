package account

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestAccountProceduresRequireIdentityBeforeLookup(t *testing.T) {
	calls := 0
	handler := NewHandler(fakeReader{subscription: func(context.Context, string) (*shared.SubscriptionStatus, error) { calls++; return nil, nil }}, func(context.Context) string { return "" })
	_, err := handler.GetMySubscription(context.Background(), connect.NewRequest(&lpbsv1.GetMySubscriptionRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeUnauthenticated || calls != 0 {
		t.Fatalf("code=%v calls=%d", got, calls)
	}
}

func TestGetMyCreditsPreservesDisplayConfiguration(t *testing.T) {
	handler := NewHandler(fakeReader{credits: func(context.Context, string) (*Credits, error) {
		return &Credits{DisplayCreditsLabel: "tokens", DisplayCreditsMultiplier: 2}, nil
	}}, testUser)
	response, err := handler.GetMyCredits(context.Background(), connect.NewRequest(&lpbsv1.GetMyCreditsRequest{}))
	if err != nil || response.Msg.GetDisplayCreditsLabel() != "tokens" || response.Msg.GetDisplayCreditsMultiplier() != 2 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestGetEntitlementsPreservesBillingCycleStart(t *testing.T) {
	handler := NewHandler(fakeReader{entitlements: func(context.Context, string) (*Entitlements, error) {
		return &Entitlements{Status: "active", BillingCycleStart: 12}, nil
	}}, testUser)
	response, err := handler.GetEntitlements(context.Background(), connect.NewRequest(&lpbsv1.GetEntitlementsRequest{}))
	if err != nil || response.Msg.GetBillingCycleStart() != 12 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestGetEntitlementsMapsLookupFailureToInternal(t *testing.T) {
	handler := NewHandler(fakeReader{entitlements: func(context.Context, string) (*Entitlements, error) {
		return nil, errors.New("store")
	}}, testUser)
	_, err := handler.GetEntitlements(context.Background(), connect.NewRequest(&lpbsv1.GetEntitlementsRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("code=%v", got)
	}
}

func TestGetEntitlementsRejectsOutOfRangeBillingCycleStart(t *testing.T) {
	tooLarge := int(^uint(0) >> 1)
	handler := NewHandler(fakeReader{entitlements: func(context.Context, string) (*Entitlements, error) {
		return &Entitlements{BillingCycleStart: tooLarge}, nil
	}}, testUser)
	_, err := handler.GetEntitlements(context.Background(), connect.NewRequest(&lpbsv1.GetEntitlementsRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("code = %v, want %v", got, connect.CodeInternal)
	}
}

func TestRegisterRoutesServesGeneratedAccountProcedures(t *testing.T) {
	router := mux.NewRouter()
	RegisterRoutes(router, fakeReader{subscription: func(context.Context, string) (*shared.SubscriptionStatus, error) {
		return &shared.SubscriptionStatus{UserIdentity: "user@example.test"}, nil
	}}, testUser, func(next http.HandlerFunc) http.HandlerFunc { return next })
	server := httptest.NewServer(router)
	defer server.Close()
	client := lpbsconnect.NewAccountServiceClient(server.Client(), server.URL)
	response, err := client.GetMySubscription(context.Background(), connect.NewRequest(&lpbsv1.GetMySubscriptionRequest{}))
	if err != nil || response.Msg.GetStatus().GetUserIdentity() != "user@example.test" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

var testUser = func(context.Context) string { return "user@example.test" }

type fakeReader struct {
	subscription func(context.Context, string) (*shared.SubscriptionStatus, error)
	credits      func(context.Context, string) (*Credits, error)
	entitlements func(context.Context, string) (*Entitlements, error)
}

func (f fakeReader) GetSubscriptionContext(c context.Context, u string) (*shared.SubscriptionStatus, error) {
	if f.subscription != nil {
		return f.subscription(c, u)
	}
	return nil, nil
}

func (f fakeReader) GetCreditsContext(c context.Context, u string) (*Credits, error) {
	if f.credits != nil {
		return f.credits(c, u)
	}
	return nil, nil
}

func (f fakeReader) GetEntitlementsContext(c context.Context, u string) (*Entitlements, error) {
	if f.entitlements != nil {
		return f.entitlements(c, u)
	}
	return nil, nil
}
