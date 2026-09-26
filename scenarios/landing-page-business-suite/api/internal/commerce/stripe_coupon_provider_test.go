package commerce

import (
	"context"
	"io"
	"strings"
	"testing"
)

type stripeCouponRequesterFake struct {
	method, path, contentType, body string
	response                        string
	err                             error
}

func (f *stripeCouponRequesterFake) Request(_ context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	f.method, f.path, f.contentType = method, path, contentType
	if body != nil {
		encoded, _ := io.ReadAll(body)
		f.body = string(encoded)
	}
	return []byte(f.response), f.err
}

type couponMappingsFake map[string]string

func (m couponMappingsFake) GetCouponMappings() map[string]string { return m }

func TestStripeCouponProviderCreatesTypedCouponAndPreservesIntroMapping(t *testing.T) {
	requester := &stripeCouponRequesterFake{response: `{"id":"coupon_intro","name":"Intro","percent_off":25,"duration":"once","valid":true}`}
	provider := NewStripeCouponProvider(requester, nil, func() map[string]string { return map[string]string{"pro": "coupon_intro"} })
	percent := 25.0
	coupon, err := provider.CreateCoupon(context.Background(), CreateCouponInput{ID: "coupon_intro", PercentOff: &percent, Duration: "once"})
	if err != nil {
		t.Fatalf("CreateCoupon() error = %v", err)
	}
	if requester.method != "POST" || requester.path != "/v1/coupons" || requester.contentType != "application/x-www-form-urlencoded" {
		t.Fatalf("request = %#v, want Stripe coupon form POST", requester)
	}
	if !strings.Contains(requester.body, "percent_off=25.00") || !strings.Contains(requester.body, "duration=once") {
		t.Fatalf("form = %q, want encoded coupon input", requester.body)
	}
	if !coupon.IsIntroCoupon || coupon.IntroTier != "pro" {
		t.Fatalf("coupon intro mapping = %#v, want pro", coupon)
	}
}

func TestStripeCouponProviderImportPreviewCombinesPlanAndIntroAssignments(t *testing.T) {
	requester := &stripeCouponRequesterFake{response: `{"data":[{"id":"coupon_plan","name":"Plan","duration":"once","valid":true},{"id":"coupon_intro","name":"Intro","duration":"once","valid":true},{"id":"coupon_new","name":"New","duration":"once","valid":true}]}`}
	provider := NewStripeCouponProvider(requester, couponMappingsFake{"price_1": "coupon_plan"}, func() map[string]string { return map[string]string{"pro": "coupon_intro"} })
	preview, err := provider.GetCouponImportPreview(context.Background())
	if err != nil {
		t.Fatalf("GetCouponImportPreview() error = %v", err)
	}
	if preview.TotalCoupons != 3 || preview.ExistingCount != 2 || preview.NewCount != 1 {
		t.Fatalf("preview counts = %#v, want 3/2/1", preview)
	}
	if !preview.Coupons[0].ExistsLocally || !preview.Coupons[1].ExistsLocally || preview.Coupons[2].ExistsLocally {
		t.Fatalf("preview assignments = %#v", preview.Coupons)
	}
}

func TestStripeCouponProviderRejectsInvalidCreateInputBeforeRequest(t *testing.T) {
	requester := &stripeCouponRequesterFake{}
	provider := NewStripeCouponProvider(requester, nil, nil)
	if _, err := provider.CreateCoupon(context.Background(), CreateCouponInput{Duration: "once"}); err == nil {
		t.Fatal("CreateCoupon() error = nil, want discount validation")
	}
	if requester.method != "" {
		t.Fatalf("requester invoked for invalid input: %#v", requester)
	}
}
