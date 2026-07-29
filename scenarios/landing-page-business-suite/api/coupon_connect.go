package main

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
	"landing-page-business-suite-api/handlers/coupons"
)

// couponConnectHandler is the administrator-only typed boundary for Stripe
// coupons and the local plan mappings that control introductory offers.
type couponConnectHandler struct {
	stripe StripeCouponService
	plans  *PlanService
	db     CouponUsageStore
}

func newCouponConnectHandler(stripe StripeCouponService, plans *PlanService, db CouponUsageStore) *couponConnectHandler {
	return &couponConnectHandler{stripe: stripe, plans: plans, db: db}
}

func (h *couponConnectHandler) ListCoupons(ctx context.Context, _ *connect.Request[lpbsv1.ListCouponsRequest]) (*connect.Response[lpbsv1.ListCouponsResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	coupons, err := h.stripe.ListCoupons(ctx)
	if err != nil {
		return nil, couponError(err)
	}
	result, err := couponsProto(coupons)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.ListCouponsResponse{Coupons: result, IntroCouponMap: h.stripe.GetIntroCouponMap()}), nil
}

func (h *couponConnectHandler) CreateCoupon(ctx context.Context, request *connect.Request[lpbsv1.CreateCouponRequest]) (*connect.Response[lpbsv1.CreateCouponResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	input, err := createCouponInput(request.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	coupon, err := h.stripe.CreateCoupon(ctx, input)
	if err != nil {
		return nil, couponError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	logStructured("admin_coupon_created", map[string]interface{}{"coupon_id": coupon.ID, "duration": coupon.Duration})
	return connect.NewResponse(&lpbsv1.CreateCouponResponse{Coupon: result}), nil
}

func (h *couponConnectHandler) GetCoupon(ctx context.Context, request *connect.Request[lpbsv1.GetCouponRequest]) (*connect.Response[lpbsv1.GetCouponResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	coupon, err := h.stripe.GetCoupon(ctx, id)
	if err != nil {
		return nil, couponError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.GetCouponResponse{Coupon: result}), nil
}

func (h *couponConnectHandler) UpdateCoupon(ctx context.Context, request *connect.Request[lpbsv1.UpdateCouponRequest]) (*connect.Response[lpbsv1.UpdateCouponResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	coupon, err := h.stripe.UpdateCoupon(ctx, id, UpdateCouponRequest{Name: request.Msg.GetName()})
	if err != nil {
		return nil, couponError(err)
	}
	result, err := couponProto(coupon)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	logStructured("admin_coupon_updated", map[string]interface{}{"coupon_id": id})
	return connect.NewResponse(&lpbsv1.UpdateCouponResponse{Coupon: result}), nil
}

func (h *couponConnectHandler) DeleteCoupon(ctx context.Context, request *connect.Request[lpbsv1.DeleteCouponRequest]) (*connect.Response[lpbsv1.DeleteCouponResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	id := strings.TrimSpace(request.Msg.GetCouponId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("coupon_id is required"))
	}
	if err := h.stripe.DeleteCoupon(ctx, id); err != nil {
		return nil, couponError(err)
	}
	logStructured("admin_coupon_deleted", map[string]interface{}{"coupon_id": id})
	return connect.NewResponse(&lpbsv1.DeleteCouponResponse{Deleted: true}), nil
}

func (h *couponConnectHandler) ListCouponUsage(ctx context.Context, _ *connect.Request[lpbsv1.ListCouponUsageRequest]) (*connect.Response[lpbsv1.ListCouponUsageResponse], error) {
	if h.db == nil {
		return nil, couponUnavailable("database")
	}
	usage, err := listCouponUsage(ctx, sqlCouponUsageStore{db: h.db})
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

func (h *couponConnectHandler) GetCouponMappings(context.Context, *connect.Request[lpbsv1.GetCouponMappingsRequest]) (*connect.Response[lpbsv1.GetCouponMappingsResponse], error) {
	if h.plans == nil {
		return nil, couponUnavailable("plan service")
	}
	return connect.NewResponse(&lpbsv1.GetCouponMappingsResponse{Mappings: h.plans.GetCouponMappings()}), nil
}

func (h *couponConnectHandler) SetCouponForPlan(_ context.Context, request *connect.Request[lpbsv1.SetCouponForPlanRequest]) (*connect.Response[lpbsv1.SetCouponForPlanResponse], error) {
	if h.plans == nil {
		return nil, couponUnavailable("plan service")
	}
	priceID, couponID := strings.TrimSpace(request.Msg.GetPriceId()), strings.TrimSpace(request.Msg.GetCouponId())
	if priceID == "" || couponID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id and coupon_id are required"))
	}
	if err := h.plans.SetCouponForPlan(priceID, couponID); err != nil {
		return nil, couponError(err)
	}
	logStructured("admin_coupon_assigned_to_plan", map[string]interface{}{"price_id": priceID, "coupon_id": couponID})
	return connect.NewResponse(&lpbsv1.SetCouponForPlanResponse{Assigned: true}), nil
}

func (h *couponConnectHandler) RemoveCouponFromPlan(_ context.Context, request *connect.Request[lpbsv1.RemoveCouponFromPlanRequest]) (*connect.Response[lpbsv1.RemoveCouponFromPlanResponse], error) {
	if h.plans == nil {
		return nil, couponUnavailable("plan service")
	}
	priceID := strings.TrimSpace(request.Msg.GetPriceId())
	if priceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id is required"))
	}
	if err := h.plans.RemoveCouponFromPlan(priceID); err != nil {
		return nil, couponError(err)
	}
	logStructured("admin_coupon_removed_from_plan", map[string]interface{}{"price_id": priceID})
	return connect.NewResponse(&lpbsv1.RemoveCouponFromPlanResponse{Removed: true}), nil
}

func (h *couponConnectHandler) GetCouponImportPreview(ctx context.Context, _ *connect.Request[lpbsv1.GetCouponImportPreviewRequest]) (*connect.Response[lpbsv1.GetCouponImportPreviewResponse], error) {
	if h.stripe == nil {
		return nil, couponUnavailable("Stripe service")
	}
	preview, err := h.stripe.GetCouponImportPreview(ctx)
	if err != nil {
		return nil, couponError(err)
	}
	items := make([]*lpbsv1.CouponImportPreviewItem, 0, len(preview.Coupons))
	for _, item := range preview.Coupons {
		duration, err := couponDurationProto(item.Duration)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		months, err := couponOptionalInt32(item.DurationInMonths)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		times, err := couponInt32(item.TimesRedeemed)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		items = append(items, &lpbsv1.CouponImportPreviewItem{Id: item.ID, Name: optionalCouponString(item.Name), AmountOff: item.AmountOff, PercentOff: item.PercentOff, Currency: optionalCouponString(item.Currency), Duration: duration, DurationInMonths: months, TimesRedeemed: times, Valid: item.Valid, ExistsLocally: item.ExistsLocally})
	}
	total, err := couponInt32(preview.TotalCoupons)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	existing, err := couponInt32(preview.ExistingCount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	newCount, err := couponInt32(preview.NewCount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.GetCouponImportPreviewResponse{Coupons: items, TotalCoupons: total, ExistingCount: existing, NewCount: newCount}), nil
}

func couponProto(coupon *StripeCoupon) (*lpbsv1.Coupon, error) {
	if coupon == nil {
		return nil, errors.New("nil coupon")
	}
	return couponValueProto(*coupon)
}

func couponsProto(coupons []StripeCoupon) ([]*lpbsv1.Coupon, error) {
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

func couponValueProto(coupon StripeCoupon) (*lpbsv1.Coupon, error) {
	duration, err := couponDurationProto(coupon.Duration)
	if err != nil {
		return nil, err
	}
	months, err := couponOptionalInt32(coupon.DurationInMonths)
	if err != nil {
		return nil, err
	}
	max, err := couponOptionalInt32(coupon.MaxRedemptions)
	if err != nil {
		return nil, err
	}
	times, err := couponInt32(coupon.TimesRedeemed)
	if err != nil {
		return nil, err
	}
	return &lpbsv1.Coupon{Id: coupon.ID, Name: optionalCouponString(coupon.Name), AmountOff: coupon.AmountOff, PercentOff: coupon.PercentOff, Currency: optionalCouponString(coupon.Currency), Duration: duration, DurationInMonths: months, MaxRedemptions: max, RedeemBy: coupon.RedeemBy, TimesRedeemed: times, Valid: coupon.Valid, Created: coupon.Created, IsIntroCoupon: coupon.IsIntroCoupon, IntroTier: optionalCouponString(coupon.IntroTier)}, nil
}

func createCouponInput(input *lpbsv1.CreateCouponRequest) (CreateCouponRequest, error) {
	duration, err := couponDurationString(input.GetDuration())
	if err != nil {
		return CreateCouponRequest{}, err
	}
	months, err := couponOptionalInt(input.DurationInMonths)
	if err != nil {
		return CreateCouponRequest{}, err
	}
	max, err := couponOptionalInt(input.MaxRedemptions)
	if err != nil {
		return CreateCouponRequest{}, err
	}
	return CreateCouponRequest{ID: input.GetId(), Name: input.GetName(), AmountOff: input.AmountOff, PercentOff: input.PercentOff, Currency: input.GetCurrency(), Duration: duration, DurationInMonths: months, MaxRedemptions: max, RedeemBy: input.RedeemBy}, nil
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

func optionalCouponString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func couponInt32(value int) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("coupon value %d outside protobuf int32 range", value)
	}
	return int32(value), nil
}

func couponOptionalInt32(value *int) (*int32, error) {
	if value == nil {
		return nil, nil
	}
	result, err := couponInt32(*value)
	return &result, err
}

func couponOptionalInt(value *int32) (*int, error) {
	if value == nil {
		return nil, nil
	}
	result := int(*value)
	return &result, nil
}

func couponUnavailable(name string) error {
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("%s unavailable", name))
}

func couponError(err error) error {
	if status, _, _, ok := classifyStripeError(err); ok {
		switch status {
		case http.StatusNotFound:
			return connect.NewError(connect.CodeNotFound, err)
		case http.StatusBadGateway, http.StatusServiceUnavailable:
			return connect.NewError(connect.CodeUnavailable, err)
		case http.StatusUnauthorized:
			return connect.NewError(connect.CodeUnauthenticated, err)
		case http.StatusForbidden:
			return connect.NewError(connect.CodePermissionDenied, err)
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInvalidArgument, err)
}

func registerCouponAdminConnectRoutes(router *mux.Router, stripe StripeCouponService, plans *PlanService, db CouponUsageStore, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewCouponAdminServiceHandler(newCouponConnectHandler(stripe, plans, db))
	coupons.Register(router, generated, requireAdmin)
}
