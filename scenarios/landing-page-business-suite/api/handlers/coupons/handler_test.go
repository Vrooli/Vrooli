package coupons

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
	"landing-page-business-suite-api/internal/commerce"
)

type fakeCouponService struct {
	list    []commerce.Coupon
	listErr error
	created commerce.CreateCouponInput
}

func (f *fakeCouponService) ListCoupons(context.Context) ([]commerce.Coupon, error) {
	return f.list, f.listErr
}

func (*fakeCouponService) GetCoupon(context.Context, string) (*commerce.Coupon, error) {
	return nil, errors.New("not found")
}

func (f *fakeCouponService) CreateCoupon(_ context.Context, input commerce.CreateCouponInput) (*commerce.Coupon, error) {
	f.created = input
	return &commerce.Coupon{ID: "coupon_created", Duration: input.Duration, Valid: true}, nil
}

func (*fakeCouponService) UpdateCoupon(context.Context, string, commerce.UpdateCouponInput) (*commerce.Coupon, error) {
	return nil, errors.New("not found")
}
func (*fakeCouponService) DeleteCoupon(context.Context, string) error { return errors.New("not found") }
func (*fakeCouponService) GetCouponImportPreview(context.Context) (*commerce.CouponImportPreview, error) {
	return &commerce.CouponImportPreview{}, nil
}

func (*fakeCouponService) GetIntroCouponMap() map[string]string {
	return map[string]string{"price_intro": "coupon_intro"}
}

func TestHandlerMapsListAndCreateRequests(t *testing.T) {
	stripe := &fakeCouponService{list: []commerce.Coupon{{ID: "coupon_intro", Duration: "forever", Valid: true}}}
	handler := NewHandler(stripe, nil, nil, nil, nil)
	listed, err := handler.ListCoupons(context.Background(), connect.NewRequest(&lpbsv1.ListCouponsRequest{}))
	if err != nil || listed.Msg.GetCoupons()[0].GetId() != "coupon_intro" {
		t.Fatalf("ListCoupons() = %#v, %v", listed, err)
	}
	name, currency := "Launch", "usd"
	created, err := handler.CreateCoupon(context.Background(), connect.NewRequest(&lpbsv1.CreateCouponRequest{Name: &name, Currency: &currency, Duration: lpbsv1.CouponDuration_COUPON_DURATION_FOREVER}))
	if err != nil || created.Msg.GetCoupon().GetId() != "coupon_created" {
		t.Fatalf("CreateCoupon() = %#v, %v", created, err)
	}
	if stripe.created.Name != name || stripe.created.Currency != currency || stripe.created.Duration != "forever" {
		t.Fatalf("created input = %#v", stripe.created)
	}
}

func TestHandlerRejectsInvalidInputAndMapsErrors(t *testing.T) {
	mapped := false
	handler := NewHandler(&fakeCouponService{listErr: errors.New("Stripe unavailable")}, nil, nil, func(err error) error {
		mapped = true
		return connect.NewError(connect.CodeUnavailable, err)
	}, nil)
	if _, err := handler.GetCoupon(context.Background(), connect.NewRequest(&lpbsv1.GetCouponRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty coupon ID code = %v", connect.CodeOf(err))
	}
	if _, err := handler.ListCoupons(context.Background(), connect.NewRequest(&lpbsv1.ListCouponsRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("provider error code = %v", connect.CodeOf(err))
	}
	if !mapped {
		t.Fatal("provider error mapper was not invoked")
	}
}

func TestConnectRoutesRequireAdmin(t *testing.T) {
	router := mux.NewRouter()
	RegisterConnectRoutes(router, &fakeCouponService{}, nil, nil, func(http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusUnauthorized) }
	}, nil, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, lpbsconnect.CouponAdminServiceListCouponsProcedure, nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
