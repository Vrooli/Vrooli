package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PY1TestLedgerTarget struct {
	AdapterID, BookID, RevenueAccountID string
}

// DispatchPY1TestEconomicReceipt ingests one retained, paid test Checkout into
// an explicitly test-only Money Ledger destination. Stable external IDs make
// retries safe after uncertain network failures.
func DispatchPY1TestEconomicReceipt(ctx context.Context, db StripeStore, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID, outcome string) (*StripeTestEconomicReceipt, error) {
	if db == nil || writer == nil || strings.TrimSpace(target.AdapterID) == "" || strings.TrimSpace(target.BookID) == "" || strings.TrimSpace(target.RevenueAccountID) == "" {
		return nil, errors.New("PY1 test Money Ledger destination is incomplete")
	}
	service := &StripeWebhookService{db: db}
	receipt, err := service.LoadTestEconomicReceipt(ctx, runID, outcome)
	if err != nil || receipt == nil {
		return receipt, err
	}
	if receipt.MoneyLedgerStatus == "posted" {
		return receipt, nil
	}
	_, err = db.Exec(`UPDATE py1_test_economic_receipts SET money_ledger_attempts=money_ledger_attempts+1 WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID)
	if err != nil {
		return receipt, fmt.Errorf("record PY1 Money Ledger attempt: %w", err)
	}
	now := time.Now().UTC()
	event := &sharedpb.MoneyEvent{
		ExternalId: receipt.MoneyLedgerExternalID, AdapterId: target.AdapterID,
		BookId: target.BookID, AccountId: target.RevenueAccountID,
		AmountMinor: receipt.AmountMinor, Currency: strings.ToUpper(receipt.Currency),
		OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now),
		Basis:       sharedpb.Basis_BASIS_DERIVED,
		Category:    "test.py1.stripe_checkout",
		Description: fmt.Sprintf("PY1 test Checkout %s (%s)", receipt.CheckoutSessionID, receipt.Outcome),
	}
	accepted, err := writer.IngestSandboxCapture(ctx, event)
	if err != nil {
		_, _ = db.Exec(`UPDATE py1_test_economic_receipts SET money_ledger_status='retryable_failed' WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID)
		return receipt, errors.New("Money Ledger did not accept PY1 test Checkout posting")
	}
	if strings.TrimSpace(accepted.PostingID) == "" || strings.TrimSpace(accepted.ReceiptID) == "" || accepted.ExternalID != receipt.MoneyLedgerExternalID {
		_, _ = db.Exec(`UPDATE py1_test_economic_receipts SET money_ledger_status='retryable_failed' WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID)
		return receipt, errors.New("Money Ledger acceptance did not match the PY1 test Checkout identity")
	}
	_, err = db.Exec(`UPDATE py1_test_economic_receipts SET money_ledger_status='posted',money_ledger_posting_id=$2,money_ledger_receipt_id=$3,posted_at=CURRENT_TIMESTAMP WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID, accepted.PostingID, accepted.ReceiptID)
	if err != nil {
		return receipt, fmt.Errorf("persist accepted PY1 Money Ledger receipt: %w", err)
	}
	return service.LoadTestEconomicReceipt(ctx, runID, outcome)
}

func (s *StripeWebhookService) DispatchTestEconomicReceipt(ctx context.Context, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID, outcome string) (*StripeTestEconomicReceipt, error) {
	if s == nil {
		return nil, errors.New("PY1 test Stripe webhook service is unavailable")
	}
	return DispatchPY1TestEconomicReceipt(ctx, s.db, writer, target, runID, outcome)
}
