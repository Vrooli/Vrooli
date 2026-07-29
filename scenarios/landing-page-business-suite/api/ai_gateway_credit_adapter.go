package main

import (
	"context"

	"landing-page-business-suite-api/internal/intelligence"
)

// aiGatewayUsageSource is the commerce behavior required at the composition
// boundary. Keeping it narrow makes the adapter independently testable.
type aiGatewayUsageSource interface {
	ReserveAndCharge(context.Context, string, string, string, int64, UsageReportRequest) error
	ReserveCredits(context.Context, string, string, string, int64) (string, error)
	FinalizeReservation(context.Context, string, int64) error
	ReleaseReservation(context.Context, string) error
	AdjustUsage(context.Context, string, string, int64, string) error
	RecordUsage(context.Context, UsageReportRequest) error
}

// aiGatewayCreditAdapter is the sole composition boundary between AI provider
// orchestration and commerce persistence. It preserves the domain rule that
// internal/intelligence must not import its sibling commerce package.
type aiGatewayCreditAdapter struct{ usage aiGatewayUsageSource }

func (a aiGatewayCreditAdapter) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, report intelligence.UsageReport) error {
	return a.usage.ReserveAndCharge(ctx, userIdentity, tier, limitKey, amount, toCommerceUsageReport(report))
}

func (a aiGatewayCreditAdapter) ReserveCredits(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
	return a.usage.ReserveCredits(ctx, userIdentity, tier, limitKey, amount)
}

func (a aiGatewayCreditAdapter) FinalizeReservation(ctx context.Context, reservationID string, actualAmount int64) error {
	return a.usage.FinalizeReservation(ctx, reservationID, actualAmount)
}

func (a aiGatewayCreditAdapter) ReleaseReservation(ctx context.Context, reservationID string) error {
	return a.usage.ReleaseReservation(ctx, reservationID)
}

func (a aiGatewayCreditAdapter) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	return a.usage.AdjustUsage(ctx, userIdentity, limitKey, adjustment, reason)
}

func (a aiGatewayCreditAdapter) RecordUsage(ctx context.Context, report intelligence.UsageReport) error {
	return a.usage.RecordUsage(ctx, toCommerceUsageReport(report))
}

func toCommerceUsageReport(report intelligence.UsageReport) UsageReportRequest {
	return UsageReportRequest{UserIdentity: report.UserIdentity, LimitKey: report.LimitKey, Amount: report.Amount, AppBundleKey: report.AppBundleKey, Operation: report.Operation, Metadata: report.Metadata}
}

var _ intelligence.UsageServicer = aiGatewayCreditAdapter{}
var _ aiGatewayUsageSource = (*UsageService)(nil)
