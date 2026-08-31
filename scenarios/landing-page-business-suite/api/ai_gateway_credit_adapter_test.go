package main

import (
	"context"
	"errors"
	"testing"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/intelligence"
)

func TestMeteredInferenceCreditAdapterMapsAndDelegatesUsage(t *testing.T) {
	source := &fakeMeteredInferenceUsageSource{reservationID: "reservation-1"}
	adapter := newCommerceUsageServicer(source)
	report := intelligence.UsageReport{UserIdentity: "customer@example.test", LimitKey: "ai_credits", Amount: 99, AppBundleKey: "suite", Operation: "chat", Metadata: map[string]string{"request": "1"}}

	if err := adapter.ReserveAndCharge(context.Background(), "customer@example.test", "pro", "ai_credits", 99, report); err != nil {
		t.Fatalf("reserve and charge: %v", err)
	}
	if source.reserveReport.UserIdentity != report.UserIdentity || source.reserveReport.LimitKey != report.LimitKey || source.reserveReport.Amount != report.Amount || source.reserveReport.AppBundleKey != report.AppBundleKey || source.reserveReport.Operation != report.Operation || source.reserveReport.Metadata["request"] != "1" {
		t.Fatalf("mapped report=%#v", source.reserveReport)
	}
	reservationID, err := adapter.ReserveCredits(context.Background(), "customer@example.test", "pro", "ai_credits", 99)
	if err != nil || reservationID != "reservation-1" {
		t.Fatalf("reservation=%q err=%v", reservationID, err)
	}
	if err := adapter.FinalizeReservation(context.Background(), reservationID, 70); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if err := adapter.ReleaseReservation(context.Background(), reservationID); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := adapter.AdjustUsage(context.Background(), "customer@example.test", "ai_credits", -29, "refund"); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if err := adapter.RecordUsage(context.Background(), report); err != nil {
		t.Fatalf("record: %v", err)
	}
	if source.recordReport.Operation != "chat" || source.finalizedAmount != 70 || source.adjustment != -29 {
		t.Fatalf("source=%#v", source)
	}
}

func TestMeteredInferenceCreditAdapterPropagatesErrors(t *testing.T) {
	want := errors.New("usage store unavailable")
	adapter := newCommerceUsageServicer(&fakeMeteredInferenceUsageSource{err: want})
	if err := adapter.RecordUsage(context.Background(), intelligence.UsageReport{}); !errors.Is(err, want) {
		t.Fatalf("record error=%v, want %v", err, want)
	}
}

type fakeMeteredInferenceUsageSource struct {
	err             error
	reservationID   string
	reserveReport   commerce.UsageReportRequest
	recordReport    commerce.UsageReportRequest
	finalizedAmount int64
	adjustment      int64
}

func (f *fakeMeteredInferenceUsageSource) ReserveAndCharge(_ context.Context, _ string, _ string, _ string, _ int64, report commerce.UsageReportRequest) error {
	f.reserveReport = report
	return f.err
}

func (f *fakeMeteredInferenceUsageSource) ReserveCredits(context.Context, string, string, string, int64) (string, error) {
	return f.reservationID, f.err
}

func (f *fakeMeteredInferenceUsageSource) FinalizeReservation(_ context.Context, _ string, amount int64) error {
	f.finalizedAmount = amount
	return f.err
}

func (f *fakeMeteredInferenceUsageSource) ReleaseReservation(context.Context, string) error {
	return f.err
}

func (f *fakeMeteredInferenceUsageSource) AdjustUsage(_ context.Context, _ string, _ string, adjustment int64, _ string) error {
	f.adjustment = adjustment
	return f.err
}

func (f *fakeMeteredInferenceUsageSource) RecordUsage(_ context.Context, report commerce.UsageReportRequest) error {
	f.recordReport = report
	return f.err
}
