package coupons

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"landing-page-business-suite-api/internal/commerce"
)

// ErrorMapper turns a provider error into the public Connect error contract.
// It is injected because Stripe error classification belongs to API composition.
type (
	ErrorMapper func(error) error
	Logger      func(string, map[string]interface{})
)

type Handler struct {
	stripe   commerce.CouponService
	plans    *commerce.PlanService
	db       commerce.CouponUsageStore
	mapError ErrorMapper
	log      Logger
}

func NewHandler(stripe commerce.CouponService, plans *commerce.PlanService, db commerce.CouponUsageStore, mapError ErrorMapper, log Logger) *Handler {
	return &Handler{stripe: stripe, plans: plans, db: db, mapError: mapError, log: log}
}

func (h *Handler) providerError(err error) error {
	if h.mapError != nil {
		return h.mapError(err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInvalidArgument, err)
}
func unavailable(name string) error {
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("%s unavailable", name))
}
func (h *Handler) event(name string, fields map[string]interface{}) {
	if h.log != nil {
		h.log(name, fields)
	}
}

func (h *Handler) ListCoupons(ctx context.Context, _ *connect.Request[lpbsv1.ListCouponsRequest]) (*connect.Response[lpbsv1.ListCouponsResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	items, err := h.stripe.ListCoupons(ctx)
	if err != nil {
		return nil, h.providerError(err)
	}
	result, err := couponsProto(items)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.ListCouponsResponse{Coupons: result, IntroCouponMap: h.stripe.GetIntroCouponMap()}), nil
}

func (h *Handler) CreateCoupon(ctx context.Context, request *connect.Request[lpbsv1.CreateCouponRequest]) (*connect.Response[lpbsv1.CreateCouponResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	input, err := createCouponInput(request.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	coupon, err := h.stripe.CreateCoupon(ctx, input)
	if err != nil {
		return nil, h.providerError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.event("admin_coupon_created", map[string]interface{}{"coupon_id": coupon.ID, "duration": coupon.Duration})
	return connect.NewResponse(&lpbsv1.CreateCouponResponse{Coupon: result}), nil
}

func (h *Handler) GetCoupon(ctx context.Context, request *connect.Request[lpbsv1.GetCouponRequest]) (*connect.Response[lpbsv1.GetCouponResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	coupon, err := h.stripe.GetCoupon(ctx, id)
	if err != nil {
		return nil, h.providerError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.GetCouponResponse{Coupon: result}), nil
}

func (h *Handler) UpdateCoupon(ctx context.Context, request *connect.Request[lpbsv1.UpdateCouponRequest]) (*connect.Response[lpbsv1.UpdateCouponResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	coupon, err := h.stripe.UpdateCoupon(ctx, id, commerce.UpdateCouponInput{Name: request.Msg.GetName()})
	if err != nil {
		return nil, h.providerError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.event("admin_coupon_updated", map[string]interface{}{"coupon_id": id})
	return connect.NewResponse(&lpbsv1.UpdateCouponResponse{Coupon: result}), nil
}

func (h *Handler) DeleteCoupon(ctx context.Context, request *connect.Request[lpbsv1.DeleteCouponRequest]) (*connect.Response[lpbsv1.DeleteCouponResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	if err := h.stripe.DeleteCoupon(ctx, id); err != nil {
		return nil, h.providerError(err)
	}
	h.event("admin_coupon_deleted", map[string]interface{}{"coupon_id": id})
	return connect.NewResponse(&lpbsv1.DeleteCouponResponse{Deleted: true}), nil
}

func (h *Handler) ListCouponUsage(ctx context.Context, _ *connect.Request[lpbsv1.ListCouponUsageRequest]) (*connect.Response[lpbsv1.ListCouponUsageResponse], error) {
	if h.db == nil {
		return nil, unavailable("database")
	}
	usage, err := commerce.ListCouponUsage(ctx, h.db)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := make([]*lpbsv1.CouponUsageStat, 0, len(usage))
	for _, item := range usage {
		entry := &lpbsv1.CouponUsageStat{CouponId: item.CouponID, TotalUses: item.TotalUses}
		if item.LastUsedAt != nil {
			value := item.LastUsedAt.Format("2006-01-02T15:04:05Z")
			entry.LastUsedAt = &value
		}
		result = append(result, entry)
	}
	return connect.NewResponse(&lpbsv1.ListCouponUsageResponse{Usage: result}), nil
}
func (h *Handler) GetCouponMappings(context.Context, *connect.Request[lpbsv1.GetCouponMappingsRequest]) (*connect.Response[lpbsv1.GetCouponMappingsResponse], error) {
	if h.plans == nil {
		return nil, unavailable("plan service")
	}
	return connect.NewResponse(&lpbsv1.GetCouponMappingsResponse{Mappings: h.plans.GetCouponMappings()}), nil
}
func (h *Handler) SetCouponForPlan(_ context.Context, request *connect.Request[lpbsv1.SetCouponForPlanRequest]) (*connect.Response[lpbsv1.SetCouponForPlanResponse], error) {
	if h.plans == nil {
		return nil, unavailable("plan service")
	}
	priceID, couponID := strings.TrimSpace(request.Msg.GetPriceId()), strings.TrimSpace(request.Msg.GetCouponId())
	if priceID == "" || couponID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id and coupon_id are required"))
	}
	if err := h.plans.SetCouponForPlan(priceID, couponID); err != nil {
		return nil, h.providerError(err)
	}
	h.event("admin_coupon_assigned_to_plan", map[string]interface{}{"price_id": priceID, "coupon_id": couponID})
	return connect.NewResponse(&lpbsv1.SetCouponForPlanResponse{Assigned: true}), nil
}
func (h *Handler) RemoveCouponFromPlan(_ context.Context, request *connect.Request[lpbsv1.RemoveCouponFromPlanRequest]) (*connect.Response[lpbsv1.RemoveCouponFromPlanResponse], error) {
	if h.plans == nil {
		return nil, unavailable("plan service")
	}
	priceID := strings.TrimSpace(request.Msg.GetPriceId())
	if priceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id is required"))
	}
	if err := h.plans.RemoveCouponFromPlan(priceID); err != nil {
		return nil, h.providerError(err)
	}
	h.event("admin_coupon_removed_from_plan", map[string]interface{}{"price_id": priceID})
	return connect.NewResponse(&lpbsv1.RemoveCouponFromPlanResponse{Removed: true}), nil
}
func (h *Handler) GetCouponImportPreview(ctx context.Context, _ *connect.Request[lpbsv1.GetCouponImportPreviewRequest]) (*connect.Response[lpbsv1.GetCouponImportPreviewResponse], error) {
	if h.stripe == nil {
		return nil, unavailable("Stripe service")
	}
	preview, err := h.stripe.GetCouponImportPreview(ctx)
	if err != nil {
		return nil, h.providerError(err)
	}
	items := make([]*lpbsv1.CouponImportPreviewItem, 0, len(preview.Coupons))
	for _, item := range preview.Coupons {
		duration, err := couponDurationProto(item.Duration)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		months, err := optionalInt32(item.DurationInMonths)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		times, err := int32Value(item.TimesRedeemed)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		items = append(items, &lpbsv1.CouponImportPreviewItem{Id: item.ID, Name: optionalString(item.Name), AmountOff: item.AmountOff, PercentOff: item.PercentOff, Currency: optionalString(item.Currency), Duration: duration, DurationInMonths: months, TimesRedeemed: times, Valid: item.Valid, ExistsLocally: item.ExistsLocally})
	}
	total, err := int32Value(preview.TotalCoupons)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	existing, err := int32Value(preview.ExistingCount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	fresh, err := int32Value(preview.NewCount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.GetCouponImportPreviewResponse{Coupons: items, TotalCoupons: total, ExistingCount: existing, NewCount: fresh}), nil
}

func couponProto(coupon *commerce.Coupon) (*lpbsv1.Coupon, error) {
	if coupon == nil {
		return nil, errors.New("nil coupon")
	}
	return couponValueProto(*coupon)
}
func couponsProto(coupons []commerce.Coupon) ([]*lpbsv1.Coupon, error) {
	result := make([]*lpbsv1.Coupon, 0, len(coupons))
	for _, coupon := range coupons {
		value, err := couponValueProto(coupon)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func couponValueProto(coupon commerce.Coupon) (*lpbsv1.Coupon, error) {
	duration, err := couponDurationProto(coupon.Duration)
	if err != nil {
		return nil, err
	}
	months, err := optionalInt32(coupon.DurationInMonths)
	if err != nil {
		return nil, err
	}
	max, err := optionalInt32(coupon.MaxRedemptions)
	if err != nil {
		return nil, err
	}
	times, err := int32Value(coupon.TimesRedeemed)
	if err != nil {
		return nil, err
	}
	return &lpbsv1.Coupon{Id: coupon.ID, Name: optionalString(coupon.Name), AmountOff: coupon.AmountOff, PercentOff: coupon.PercentOff, Currency: optionalString(coupon.Currency), Duration: duration, DurationInMonths: months, MaxRedemptions: max, RedeemBy: coupon.RedeemBy, TimesRedeemed: times, Valid: coupon.Valid, Created: coupon.Created, IsIntroCoupon: coupon.IsIntroCoupon, IntroTier: optionalString(coupon.IntroTier)}, nil
}
func createCouponInput(input *lpbsv1.CreateCouponRequest) (commerce.CreateCouponInput, error) {
	duration, err := couponDurationString(input.GetDuration())
	if err != nil {
		return commerce.CreateCouponInput{}, err
	}
	months, err := optionalInt(input.DurationInMonths)
	if err != nil {
		return commerce.CreateCouponInput{}, err
	}
	max, err := optionalInt(input.MaxRedemptions)
	if err != nil {
		return commerce.CreateCouponInput{}, err
	}
	return commerce.CreateCouponInput{ID: input.GetId(), Name: input.GetName(), AmountOff: input.AmountOff, PercentOff: input.PercentOff, Currency: input.GetCurrency(), Duration: duration, DurationInMonths: months, MaxRedemptions: max, RedeemBy: input.RedeemBy}, nil
}
func couponDurationProto(value string) (lpbsv1.CouponDuration, error) {
	switch value {
	case "once":
		return lpbsv1.CouponDuration_COUPON_DURATION_ONCE, nil
	case "repeating":
		return lpbsv1.CouponDuration_COUPON_DURATION_REPEATING, nil
	case "forever":
		return lpbsv1.CouponDuration_COUPON_DURATION_FOREVER, nil
	default:
		return lpbsv1.CouponDuration_COUPON_DURATION_UNSPECIFIED, fmt.Errorf("unknown coupon duration %q", value)
	}
}
func couponDurationString(value lpbsv1.CouponDuration) (string, error) {
	switch value {
	case lpbsv1.CouponDuration_COUPON_DURATION_ONCE:
		return "once", nil
	case lpbsv1.CouponDuration_COUPON_DURATION_REPEATING:
		return "repeating", nil
	case lpbsv1.CouponDuration_COUPON_DURATION_FOREVER:
		return "forever", nil
	default:
		return "", errors.New("duration must be once, repeating, or forever")
	}
}
func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
func int32Value(value int) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("coupon value %d outside protobuf int32 range", value)
	}
	return int32(value), nil
}
func optionalInt32(value *int) (*int32, error) {
	if value == nil {
		return nil, nil
	}
	result, err := int32Value(*value)
	return &result, err
}
func optionalInt(value *int32) (*int, error) {
	if value == nil {
		return nil, nil
	}
	result := int(*value)
	return &result, nil
}

func RegisterConnectRoutes(router *mux.Router, stripe commerce.CouponService, plans *commerce.PlanService, db commerce.CouponUsageStore, requireAdmin func(http.HandlerFunc) http.HandlerFunc, mapError ErrorMapper, log Logger) {
	_, generated := lpbsconnect.NewCouponAdminServiceHandler(NewHandler(stripe, plans, db, mapError, log))
	Register(router, generated, requireAdmin)
}
