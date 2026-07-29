package main

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
)

type couponConnectFakeStripe struct {
	list      []StripeCoupon
	listErr   error
	created   CreateCouponRequest
	createErr error
}

func (f *couponConnectFakeStripe) ListCoupons(context.Context) ([]StripeCoupon, error) {
	return f.list, f.listErr
}

func (f *couponConnectFakeStripe) GetCoupon(context.Context, string) (*StripeCoupon, error) {
	return nil, errors.New("not found")
}

func (f *couponConnectFakeStripe) CreateCoupon(_ context.Context, input CreateCouponRequest) (*StripeCoupon, error) {
	f.created = input
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &StripeCoupon{ID: "coupon_created", Duration: input.Duration, Valid: true}, nil
}

func (f *couponConnectFakeStripe) UpdateCoupon(context.Context, string, UpdateCouponRequest) (*StripeCoupon, error) {
	return nil, errors.New("not found")
}

func (f *couponConnectFakeStripe) DeleteCoupon(context.Context, string) error {
	return errors.New("not found")
}

func (f *couponConnectFakeStripe) GetCouponImportPreview(context.Context) (*CouponImportPreview, error) {
	return &CouponImportPreview{}, nil
}

func (f *couponConnectFakeStripe) GetIntroCouponMap() map[string]string {
	return map[string]string{"price_intro": "coupon_intro"}
}

func TestCouponConnectMapsListAndCreateRequests(t *testing.T) {
	stripe := &couponConnectFakeStripe{list: []StripeCoupon{{ID: "coupon_intro", Duration: "forever", Valid: true}}}
	handler := newCouponConnectHandler(stripe, nil, nil)
	listed, err := handler.ListCoupons(context.Background(), connect.NewRequest(&lpbsv1.ListCouponsRequest{}))
	if err != nil {
		t.Fatalf("ListCoupons() error = %v", err)
	}
	if got := listed.Msg.GetCoupons()[0].GetId(); got != "coupon_intro" {
		t.Fatalf("coupon ID = %q", got)
	}
	if got := listed.Msg.GetIntroCouponMap()["price_intro"]; got != "coupon_intro" {
		t.Fatalf("intro mapping = %q", got)
	}

	name, currency := "Launch", "usd"
	created, err := handler.CreateCoupon(context.Background(), connect.NewRequest(&lpbsv1.CreateCouponRequest{
		Name: &name, Currency: &currency, Duration: lpbsv1.CouponDuration_COUPON_DURATION_FOREVER,
	}))
	if err != nil {
		t.Fatalf("CreateCoupon() error = %v", err)
	}
	if created.Msg.GetCoupon().GetId() != "coupon_created" {
		t.Fatalf("created coupon = %#v", created.Msg.GetCoupon())
	}
	if stripe.created.Name != name || stripe.created.Currency != currency || stripe.created.Duration != "forever" {
		t.Fatalf("created input = %#v", stripe.created)
	}
}

func TestCouponConnectRejectsInvalidInputAndMapsStripeErrors(t *testing.T) {
	stripe := &couponConnectFakeStripe{listErr: errors.New("Stripe unavailable")}
	handler := newCouponConnectHandler(stripe, nil, nil)
	_, err := handler.GetCoupon(context.Background(), connect.NewRequest(&lpbsv1.GetCouponRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("empty coupon ID code = %v", got)
	}
	_, err = handler.ListCoupons(context.Background(), connect.NewRequest(&lpbsv1.ListCouponsRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("Stripe error code = %v", got)
	}
	_, err = handler.CreateCoupon(context.Background(), connect.NewRequest(&lpbsv1.CreateCouponRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("duration code = %v", got)
	}
}

func TestCouponConnectRoutesRequireAdminAuthentication(t *testing.T) {
	router := mux.NewRouter()
	registerCouponAdminConnectRoutes(router, &couponConnectFakeStripe{}, nil, nil, func(http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusUnauthorized) }
	})
	request := httptest.NewRequest(http.MethodPost, lpbsconnect.CouponAdminServiceListCouponsProcedure, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
