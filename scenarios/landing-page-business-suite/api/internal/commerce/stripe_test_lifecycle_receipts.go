package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StripeTestLifecycleReceipt is a minimized receipt for renewal, failed-renewal,
// price-change, or cancellation events correlated to a PY1 test clock run.
type StripeTestLifecycleReceipt struct {
	EventID, RunID, Outcome, EventType, ObjectID, SubscriptionID                  string
	SubscriptionStatus, PaymentStatus, BillingReason, Currency, MoneyLedgerStatus string
	AmountMinor                                                                   int64
	MoneyLedgerExternalID, MoneyLedgerPostingID, MoneyLedgerReceiptID             string
}

func (s *StripeWebhookService) recordStripeTestLifecycleReceipt(ctx context.Context, eventID, eventType string, object map[string]interface{}, link *StripeTestOutcomeLink) error {
	if s == nil || s.db == nil || link == nil {
		return errors.New("PY1 lifecycle receipt store is unavailable")
	}
	if link.Outcome != "renewal" && link.Outcome != "past_due" && link.Outcome != "cancel" && link.Outcome != "upgrade" && link.Outcome != "downgrade" {
		return nil
	}
	objectID := stripeObjectID(object)
	subscriptionID := ""
	status := ""
	paymentStatus := "not_applicable"
	amount := int64(0)
	currency := ""
	ledgerStatus := "not_applicable"
	externalID := ""
	billingReason := ""
	switch eventType {
	case "invoice.paid":
		subscriptionID = stripeReference(object["subscription"])
		if subscriptionID == "" || !strings.HasPrefix(objectID, "in_") {
			return errors.New("paid PY1 lifecycle invoice lacks invoice or subscription identity")
		}
		value, ok := object["amount_paid"].(float64)
		if !ok || value <= 0 || value != float64(int64(value)) {
			return errors.New("paid PY1 lifecycle invoice has an invalid amount")
		}
		amount = int64(value)
		currency, _ = object["currency"].(string)
		currency = strings.ToLower(strings.TrimSpace(currency))
		if len(currency) != 3 {
			return errors.New("paid PY1 lifecycle invoice has an invalid currency")
		}
		paymentStatus, ledgerStatus = "paid", "pending"
		externalID = "lpbs-py1-invoice:" + objectID
		billingReason, _ = object["billing_reason"].(string)
	case "invoice.payment_failed":
		subscriptionID = stripeReference(object["subscription"])
		paymentStatus = "failed"
	case "customer.subscription.updated":
		subscriptionID = objectID
		status, _ = object["status"].(string)
		if status != "active" && status != "past_due" && status != "canceled" && status != "unpaid" && status != "incomplete" && status != "incomplete_expired" && status != "trialing" && status != "paused" {
			return errors.New("PY1 subscription event has an unsupported status")
		}
	case "customer.subscription.deleted":
		subscriptionID = objectID
		status, paymentStatus = "canceled", "not_applicable"
	default:
		return errors.New("event type is not a PY1 subscription lifecycle receipt")
	}
	if !strings.HasPrefix(subscriptionID, "sub_") {
		return errors.New("PY1 lifecycle event lacks subscription identity")
	}
	if (eventType == "customer.subscription.updated" || eventType == "customer.subscription.deleted") && strings.TrimSpace(status) == "" {
		return errors.New("PY1 subscription state event lacks status")
	}
	_, err := s.db.Exec(`
		INSERT INTO py1_test_lifecycle_receipts
			(stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,currency,money_ledger_external_id,money_ledger_status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13)
		ON CONFLICT (stripe_event_id) DO NOTHING
	`, eventID, link.RunID, link.Outcome, eventType, objectID, subscriptionID, status, paymentStatus, billingReason, amount, currency, externalID, ledgerStatus)
	if err != nil {
		return fmt.Errorf("persist PY1 subscription lifecycle receipt: %w", err)
	}
	var retained StripeTestLifecycleReceipt
	err = s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE stripe_event_id=$1
	`, eventID).Scan(&retained.EventID, &retained.RunID, &retained.Outcome, &retained.EventType, &retained.ObjectID, &retained.SubscriptionID, &retained.SubscriptionStatus, &retained.PaymentStatus, &retained.BillingReason, &retained.AmountMinor, &retained.Currency, &retained.MoneyLedgerStatus, &retained.MoneyLedgerExternalID, &retained.MoneyLedgerPostingID, &retained.MoneyLedgerReceiptID)
	if err != nil {
		return fmt.Errorf("read back PY1 subscription lifecycle receipt: %w", err)
	}
	if retained.RunID != link.RunID || retained.Outcome != link.Outcome || retained.EventType != eventType || retained.ObjectID != objectID || retained.SubscriptionID != subscriptionID || retained.SubscriptionStatus != status || retained.PaymentStatus != paymentStatus || retained.BillingReason != billingReason || retained.AmountMinor != amount || retained.Currency != currency || retained.MoneyLedgerExternalID != externalID {
		return errors.New("Stripe event ID was replayed with conflicting PY1 lifecycle data")
	}
	return nil
}

func (s *StripeWebhookService) LoadTestLifecycleReceipt(ctx context.Context, runID, outcome string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) {
		return nil, errors.New("invalid PY1 lifecycle receipt lookup")
	}
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE run_id=$1 AND outcome=$2 ORDER BY created_at DESC LIMIT 1
	`, runID, outcome).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 subscription lifecycle receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) LoadTestLifecycleReceiptByOutcomeAndEventType(ctx context.Context, runID, outcome, eventType string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) || (eventType != "invoice.payment_failed" && eventType != "customer.subscription.deleted" && eventType != "customer.subscription.updated") {
		return nil, errors.New("invalid PY1 lifecycle outcome/event lookup")
	}
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE run_id=$1 AND outcome=$2 AND event_type=$3 ORDER BY created_at DESC,stripe_event_id DESC LIMIT 1
	`, runID, outcome, eventType).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 lifecycle outcome event receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) LoadTestLifecycleReceiptByOutcomeEventTypeAndStatus(ctx context.Context, runID, outcome, eventType, status string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) || eventType != "customer.subscription.updated" || strings.TrimSpace(status) == "" {
		return nil, errors.New("invalid PY1 lifecycle status receipt lookup")
	}
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE run_id=$1 AND outcome=$2 AND event_type=$3 AND subscription_status=$4 ORDER BY created_at DESC,stripe_event_id DESC LIMIT 1
	`, runID, outcome, eventType, status).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 lifecycle status receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) LoadTestLifecycleRenewalReceipt(ctx context.Context, runID, outcome string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) {
		return nil, errors.New("invalid PY1 renewal receipt lookup")
	}
	return s.loadTestLifecycleRenewalReceipt(ctx, runID, outcome)
}

func (s *StripeWebhookService) LoadTestLifecycleReceiptByEvent(ctx context.Context, eventID string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(eventID) == "" {
		return nil, errors.New("invalid PY1 lifecycle event receipt lookup")
	}
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE stripe_event_id=$1
	`, eventID).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 lifecycle event receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) loadNextPendingTestLifecycleReceipt(ctx context.Context, runID string) (*StripeTestLifecycleReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" {
		return nil, errors.New("invalid PY1 pending lifecycle receipt lookup")
	}
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE run_id=$1 AND money_ledger_status IN ('pending','retryable_failed') ORDER BY created_at,stripe_event_id LIMIT 1
	`, runID).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load next PY1 pending lifecycle receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) loadTestLifecycleRenewalReceipt(ctx context.Context, runID, outcome string) (*StripeTestLifecycleReceipt, error) {
	var receipt StripeTestLifecycleReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,run_id,outcome,event_type,stripe_object_id,subscription_id,subscription_status,payment_status,billing_reason,amount_minor,COALESCE(currency,''),money_ledger_status,COALESCE(money_ledger_external_id,''),COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_lifecycle_receipts WHERE run_id=$1 AND outcome=$2 AND event_type='invoice.paid' AND billing_reason='subscription_cycle' ORDER BY created_at DESC LIMIT 1
	`, runID, outcome).Scan(&receipt.EventID, &receipt.RunID, &receipt.Outcome, &receipt.EventType, &receipt.ObjectID, &receipt.SubscriptionID, &receipt.SubscriptionStatus, &receipt.PaymentStatus, &receipt.BillingReason, &receipt.AmountMinor, &receipt.Currency, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 subscription renewal receipt: %w", err)
	}
	return &receipt, nil
}

// DispatchPY1TestLifecycleReceipt posts only positively paid renewal invoices.
// Failed-renewal and cancellation receipts are durable state transitions with
// no fictitious money movement.
func DispatchPY1TestLifecycleReceipt(ctx context.Context, db StripeStore, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID, outcome string) (*StripeTestLifecycleReceipt, error) {
	if db == nil || writer == nil || strings.TrimSpace(target.AdapterID) == "" || strings.TrimSpace(target.BookID) == "" || strings.TrimSpace(target.RevenueAccountID) == "" {
		return nil, errors.New("PY1 lifecycle Money Ledger destination is incomplete")
	}
	service := &StripeWebhookService{db: db}
	receipt, err := service.loadTestLifecycleRenewalReceipt(ctx, runID, outcome)
	if err != nil || receipt == nil || receipt.MoneyLedgerStatus != "pending" {
		return receipt, err
	}
	return DispatchPY1TestLifecycleReceiptByEvent(ctx, db, writer, target, receipt.EventID)
}

func DispatchPY1TestLifecycleReceiptByEvent(ctx context.Context, db StripeStore, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, eventID string) (*StripeTestLifecycleReceipt, error) {
	if db == nil || writer == nil || strings.TrimSpace(target.AdapterID) == "" || strings.TrimSpace(target.BookID) == "" || strings.TrimSpace(target.RevenueAccountID) == "" {
		return nil, errors.New("PY1 lifecycle Money Ledger destination is incomplete")
	}
	service := &StripeWebhookService{db: db}
	receipt, err := service.LoadTestLifecycleReceiptByEvent(ctx, eventID)
	if err != nil || receipt == nil || receipt.MoneyLedgerStatus == "posted" {
		return receipt, err
	}
	if receipt.EventType != "invoice.paid" || receipt.PaymentStatus != "paid" || receipt.AmountMinor <= 0 || receipt.MoneyLedgerExternalID == "" {
		return receipt, errors.New("PY1 lifecycle receipt is not a positive paid invoice")
	}
	if _, err := db.Exec(`UPDATE py1_test_lifecycle_receipts SET money_ledger_attempts=money_ledger_attempts+1 WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID); err != nil {
		return receipt, fmt.Errorf("record PY1 lifecycle ledger attempt: %w", err)
	}
	now := time.Now().UTC()
	event := &sharedpb.MoneyEvent{
		ExternalId: receipt.MoneyLedgerExternalID, AdapterId: target.AdapterID,
		BookId: target.BookID, AccountId: target.RevenueAccountID,
		AmountMinor: receipt.AmountMinor, Currency: strings.ToUpper(receipt.Currency),
		OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now),
		Basis: sharedpb.Basis_BASIS_DERIVED, Category: "test.py1.stripe_subscription_invoice",
		Description: fmt.Sprintf("PY1 test subscription invoice %s (%s; %s)", receipt.ObjectID, receipt.Outcome, receipt.BillingReason),
	}
	accepted, err := writer.IngestSandboxCapture(ctx, event)
	if err != nil || strings.TrimSpace(accepted.PostingID) == "" || strings.TrimSpace(accepted.ReceiptID) == "" || accepted.ExternalID != receipt.MoneyLedgerExternalID {
		_, _ = db.Exec(`UPDATE py1_test_lifecycle_receipts SET money_ledger_status='retryable_failed' WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID)
		return receipt, errors.New("Money Ledger did not accept the PY1 test subscription invoice identity")
	}
	if _, err := db.Exec(`UPDATE py1_test_lifecycle_receipts SET money_ledger_status='posted',money_ledger_posting_id=$2,money_ledger_receipt_id=$3,posted_at=CURRENT_TIMESTAMP WHERE stripe_event_id=$1 AND money_ledger_status<>'posted'`, receipt.EventID, accepted.PostingID, accepted.ReceiptID); err != nil {
		return receipt, fmt.Errorf("persist accepted PY1 invoice ledger receipt: %w", err)
	}
	return service.LoadTestLifecycleReceiptByEvent(ctx, receipt.EventID)
}

func DispatchPendingPY1TestLifecycleReceipts(ctx context.Context, db StripeStore, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID string) error {
	service := &StripeWebhookService{db: db}
	for {
		receipt, err := service.loadNextPendingTestLifecycleReceipt(ctx, runID)
		if err != nil || receipt == nil {
			return err
		}
		if _, err := DispatchPY1TestLifecycleReceiptByEvent(ctx, db, writer, target, receipt.EventID); err != nil {
			return err
		}
	}
}

func (s *StripeWebhookService) DispatchTestLifecycleReceipt(ctx context.Context, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID, outcome string) (*StripeTestLifecycleReceipt, error) {
	if s == nil {
		return nil, errors.New("PY1 test Stripe webhook service is unavailable")
	}
	return DispatchPY1TestLifecycleReceipt(ctx, s.db, writer, target, runID, outcome)
}

func (s *StripeWebhookService) DispatchTestLifecycleEventReceipt(ctx context.Context, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, eventID string) (*StripeTestLifecycleReceipt, error) {
	if s == nil {
		return nil, errors.New("PY1 test Stripe webhook service is unavailable")
	}
	return DispatchPY1TestLifecycleReceiptByEvent(ctx, s.db, writer, target, eventID)
}

func (s *StripeWebhookService) DispatchPendingTestLifecycleReceipts(ctx context.Context, writer ServiceOrderLedgerWriter, target PY1TestLedgerTarget, runID string) error {
	if s == nil {
		return errors.New("PY1 test Stripe webhook service is unavailable")
	}
	return DispatchPendingPY1TestLifecycleReceipts(ctx, s.db, writer, target, runID)
}
