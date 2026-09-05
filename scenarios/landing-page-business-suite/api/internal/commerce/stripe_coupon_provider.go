package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// StripeRequester is the authenticated provider edge used by Stripe-backed
// commerce workflows. It intentionally exposes requests, not StripeService,
// so commerce policy cannot reach into API-root configuration or HTTP state.
//
// seam: StripeRequester permits deterministic provider tests without a live
// Stripe account.
type StripeRequester interface {
	Request(context.Context, string, string, io.Reader, string) ([]byte, error)
}

// CouponMappingReader supplies local plan-to-coupon assignments for import
// previews without coupling the provider to the file-backed plan store.
//
// seam: CouponMappingReader keeps import-preview policy testable with a small
// in-memory mapping rather than a catalog fixture.
type CouponMappingReader interface {
	GetCouponMappings() map[string]string
}

// StripeCouponProvider owns the Stripe-specific implementation of the
// commerce CouponService contract. Runtime authentication and configuration
// stay behind StripeCouponRequester and IntroCouponMap.
type StripeCouponProvider struct {
	requester StripeRequester
	mappings  CouponMappingReader
	introMap  func() map[string]string
}

func NewStripeCouponProvider(requester StripeRequester, mappings CouponMappingReader, introMap func() map[string]string) *StripeCouponProvider {
	return &StripeCouponProvider{requester: requester, mappings: mappings, introMap: introMap}
}

var _ CouponService = (*StripeCouponProvider)(nil)

type stripeCoupon struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	AmountOff        *int64   `json:"amount_off"`
	PercentOff       *float64 `json:"percent_off"`
	Currency         string   `json:"currency"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months"`
	MaxRedemptions   *int     `json:"max_redemptions"`
	RedeemBy         *int64   `json:"redeem_by"`
	TimesRedeemed    int      `json:"times_redeemed"`
	Valid            bool     `json:"valid"`
	Created          int64    `json:"created"`
}

func (p *StripeCouponProvider) ListCoupons(ctx context.Context) ([]Coupon, error) {
	values := url.Values{"limit": {"100"}}
	body, err := p.request(ctx, http.MethodGet, "/v1/coupons?"+values.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []stripeCoupon `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode stripe coupons: %w", err)
	}
	coupons := make([]Coupon, 0, len(response.Data))
	for _, item := range response.Data {
		coupons = append(coupons, p.toCoupon(item))
	}
	return coupons, nil
}

func (p *StripeCouponProvider) GetCoupon(ctx context.Context, couponID string) (*Coupon, error) {
	if strings.TrimSpace(couponID) == "" {
		return nil, errors.New("coupon ID is required")
	}
	body, err := p.request(ctx, http.MethodGet, "/v1/coupons/"+url.PathEscape(couponID), nil, "")
	if err != nil {
		return nil, err
	}
	var decoded stripeCoupon
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}
	coupon := p.toCoupon(decoded)
	return &coupon, nil
}

func (p *StripeCouponProvider) CreateCoupon(ctx context.Context, input CreateCouponInput) (*Coupon, error) {
	values, err := couponCreateForm(input)
	if err != nil {
		return nil, err
	}
	body, err := p.request(ctx, http.MethodPost, "/v1/coupons", strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	var decoded stripeCoupon
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}
	coupon := p.toCoupon(decoded)
	return &coupon, nil
}

func (p *StripeCouponProvider) UpdateCoupon(ctx context.Context, couponID string, input UpdateCouponInput) (*Coupon, error) {
	if strings.TrimSpace(couponID) == "" {
		return nil, errors.New("coupon ID is required")
	}
	values := url.Values{}
	if input.Name != "" {
		values.Set("name", input.Name)
	}
	body, err := p.request(ctx, http.MethodPost, "/v1/coupons/"+url.PathEscape(couponID), strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("update coupon: %w", err)
	}
	var decoded stripeCoupon
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("parse coupon response: %w", err)
	}
	coupon := p.toCoupon(decoded)
	return &coupon, nil
}

func (p *StripeCouponProvider) DeleteCoupon(ctx context.Context, couponID string) error {
	if strings.TrimSpace(couponID) == "" {
		return errors.New("coupon ID is required")
	}
	_, err := p.request(ctx, http.MethodDelete, "/v1/coupons/"+url.PathEscape(couponID), nil, "")
	return err
}

func (p *StripeCouponProvider) GetCouponImportPreview(ctx context.Context) (*CouponImportPreview, error) {
	coupons, err := p.ListCoupons(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stripe coupons: %w", err)
	}
	used := make(map[string]struct{})
	if p.mappings != nil {
		for _, couponID := range p.mappings.GetCouponMappings() {
			used[couponID] = struct{}{}
		}
	}
	for _, couponID := range p.GetIntroCouponMap() {
		used[couponID] = struct{}{}
	}
	preview := &CouponImportPreview{Coupons: make([]CouponImportPreviewItem, 0, len(coupons))}
	for _, coupon := range coupons {
		_, exists := used[coupon.ID]
		preview.Coupons = append(preview.Coupons, CouponImportPreviewItem{
			ID: coupon.ID, Name: coupon.Name, AmountOff: coupon.AmountOff, PercentOff: coupon.PercentOff,
			Currency: coupon.Currency, Duration: coupon.Duration, DurationInMonths: coupon.DurationInMonths,
			TimesRedeemed: coupon.TimesRedeemed, Valid: coupon.Valid, ExistsLocally: exists,
		})
		if exists {
			preview.ExistingCount++
		} else {
			preview.NewCount++
		}
	}
	preview.TotalCoupons = len(coupons)
	return preview, nil
}

func (p *StripeCouponProvider) GetIntroCouponMap() map[string]string {
	if p.introMap == nil {
		return nil
	}
	mapping := p.introMap()
	if mapping == nil {
		return nil
	}
	copy := make(map[string]string, len(mapping))
	for tier, couponID := range mapping {
		copy[tier] = couponID
	}
	return copy
}

func (p *StripeCouponProvider) request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	if p.requester == nil {
		return nil, errors.New("stripe coupon requester unavailable")
	}
	return p.requester.Request(ctx, method, path, body, contentType)
}

func (p *StripeCouponProvider) toCoupon(value stripeCoupon) Coupon {
	result := Coupon{
		ID: value.ID, Name: value.Name, AmountOff: value.AmountOff, PercentOff: value.PercentOff,
		Currency: value.Currency, Duration: value.Duration, DurationInMonths: value.DurationInMonths,
		MaxRedemptions: value.MaxRedemptions, RedeemBy: value.RedeemBy, TimesRedeemed: value.TimesRedeemed,
		Valid: value.Valid, Created: value.Created,
	}
	for tier, configuredID := range p.GetIntroCouponMap() {
		if configuredID == value.ID {
			result.IsIntroCoupon, result.IntroTier = true, tier
			break
		}
	}
	return result
}

func couponCreateForm(input CreateCouponInput) (url.Values, error) {
	if input.AmountOff == nil && input.PercentOff == nil {
		return nil, errors.New("either amount_off or percent_off is required")
	}
	if input.AmountOff != nil && input.PercentOff != nil {
		return nil, errors.New("cannot specify both amount_off and percent_off")
	}
	duration := strings.TrimSpace(input.Duration)
	if duration != "once" && duration != "repeating" && duration != "forever" {
		return nil, errors.New("duration must be once, repeating, or forever")
	}
	if duration == "repeating" && (input.DurationInMonths == nil || *input.DurationInMonths <= 0) {
		return nil, errors.New("duration_in_months is required for repeating coupons")
	}
	values := url.Values{"duration": {duration}}
	if id := strings.TrimSpace(input.ID); id != "" {
		values.Set("id", id)
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		values.Set("name", name)
	}
	if input.AmountOff != nil {
		values.Set("amount_off", strconv.FormatInt(*input.AmountOff, 10))
		currency := strings.ToLower(strings.TrimSpace(input.Currency))
		if currency == "" {
			currency = "usd"
		}
		values.Set("currency", currency)
	}
	if input.PercentOff != nil {
		values.Set("percent_off", strconv.FormatFloat(*input.PercentOff, 'f', 2, 64))
	}
	if input.DurationInMonths != nil && duration == "repeating" {
		values.Set("duration_in_months", strconv.Itoa(*input.DurationInMonths))
	}
	if input.MaxRedemptions != nil && *input.MaxRedemptions > 0 {
		values.Set("max_redemptions", strconv.Itoa(*input.MaxRedemptions))
	}
	if input.RedeemBy != nil && *input.RedeemBy > 0 {
		values.Set("redeem_by", strconv.FormatInt(*input.RedeemBy, 10))
	}
	return values, nil
}
