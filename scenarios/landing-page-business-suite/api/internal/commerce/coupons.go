package commerce

import "context"

// Coupon is the provider-independent representation used by coupon transport
// and provider implementations. Keeping it in commerce prevents either layer
// from depending on API composition types.
type Coupon struct {
	ID               string
	Name             string
	AmountOff        *int64
	PercentOff       *float64
	Currency         string
	Duration         string
	DurationInMonths *int
	MaxRedemptions   *int
	RedeemBy         *int64
	TimesRedeemed    int
	Valid            bool
	Created          int64
	IsIntroCoupon    bool
	IntroTier        string
}

type CreateCouponInput struct {
	ID               string
	Name             string
	AmountOff        *int64
	PercentOff       *float64
	Currency         string
	Duration         string
	DurationInMonths *int
	MaxRedemptions   *int
	RedeemBy         *int64
}

type UpdateCouponInput struct{ Name string }

type CouponImportPreviewItem struct {
	ID               string
	Name             string
	AmountOff        *int64
	PercentOff       *float64
	Currency         string
	Duration         string
	DurationInMonths *int
	TimesRedeemed    int
	Valid            bool
	ExistsLocally    bool
}

type CouponImportPreview struct {
	Coupons       []CouponImportPreviewItem
	TotalCoupons  int
	ExistingCount int
	NewCount      int
}

// seam: CouponService keeps generated coupon administration transport independent
// of the Stripe provider implementation and its API composition error policy.
type CouponService interface {
	ListCoupons(context.Context) ([]Coupon, error)
	GetCoupon(context.Context, string) (*Coupon, error)
	CreateCoupon(context.Context, CreateCouponInput) (*Coupon, error)
	UpdateCoupon(context.Context, string, UpdateCouponInput) (*Coupon, error)
	DeleteCoupon(context.Context, string) error
	GetCouponImportPreview(context.Context) (*CouponImportPreview, error)
	GetIntroCouponMap() map[string]string
}
