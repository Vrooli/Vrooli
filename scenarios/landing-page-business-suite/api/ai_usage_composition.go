package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
	intelligencecore "landing-page-business-suite-api/internal/intelligence"
)

// CommerceUsageSource is the narrow cross-domain accounting seam for AI
// orchestration. It lives in composition because it translates commerce DTOs
// into intelligence DTOs; neither HTTP handler owns that policy.
type CommerceUsageSource interface {
	ReserveAndCharge(context.Context, string, string, string, int64, commerce.UsageReportRequest) error
	ReserveCredits(context.Context, string, string, string, int64) (string, error)
	FinalizeReservation(context.Context, string, int64) error
	ReleaseReservation(context.Context, string) error
	AdjustUsage(context.Context, string, string, int64, string) error
	RecordUsage(context.Context, commerce.UsageReportRequest) error
}

type commerceUsageServicer struct{ usage CommerceUsageSource }

// newCommerceUsageServicer adapts commerce accounting to AI orchestration.
func newCommerceUsageServicer(usage CommerceUsageSource) intelligencecore.UsageServicer {
	return commerceUsageServicer{usage: usage}
}

func (s commerceUsageServicer) ReserveAndCharge(ctx context.Context, user, tier, key string, amount int64, report intelligencecore.UsageReport) error {
	return s.usage.ReserveAndCharge(ctx, user, tier, key, amount, commerceReport(report))
}

func (s commerceUsageServicer) ReserveCredits(ctx context.Context, user, tier, key string, amount int64) (string, error) {
	return s.usage.ReserveCredits(ctx, user, tier, key, amount)
}

func (s commerceUsageServicer) FinalizeReservation(ctx context.Context, id string, amount int64) error {
	return s.usage.FinalizeReservation(ctx, id, amount)
}

func (s commerceUsageServicer) ReleaseReservation(ctx context.Context, id string) error {
	return s.usage.ReleaseReservation(ctx, id)
}

func (s commerceUsageServicer) AdjustUsage(ctx context.Context, user, key string, amount int64, reason string) error {
	return s.usage.AdjustUsage(ctx, user, key, amount, reason)
}

func (s commerceUsageServicer) RecordUsage(ctx context.Context, report intelligencecore.UsageReport) error {
	return s.usage.RecordUsage(ctx, commerceReport(report))
}

func commerceReport(report intelligencecore.UsageReport) commerce.UsageReportRequest {
	return commerce.UsageReportRequest{UserIdentity: report.UserIdentity, LimitKey: report.LimitKey, Amount: report.Amount, AppBundleKey: report.AppBundleKey, Operation: report.Operation, Metadata: report.Metadata}
}

var (
	_ intelligencecore.UsageServicer = commerceUsageServicer{}
	_ CommerceUsageSource            = (*commerce.UsageService)(nil)
)
