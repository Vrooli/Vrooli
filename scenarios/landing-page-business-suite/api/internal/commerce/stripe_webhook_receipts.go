package commerce

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const stripeWebhookProcessingLeaseSeconds = int64(300)

var ErrStripeWebhookReceiptClaimLost = errors.New("Stripe webhook receipt claim was lost")

// StripeTestOutcomeLink is an optional, strictly test-mode correlation carried
// by a Stripe object's metadata. It contains no customer identity or raw data.
type StripeTestOutcomeLink struct {
	RunID   string
	Outcome string
}

// StripeWebhookReceipt is the immutable, minimized event receipt persisted by
// LPBS. Stripe event IDs are the causal identity; payment/object IDs remain
// separate and can be added by the handler that owns the economic transition.
type StripeWebhookReceipt struct {
	EventID   string
	EventType string
	ObjectID  string
	RunID     string
	Outcome   string
	Status    string
	Livemode  bool
}

// StripeTestEconomicReceipt is a minimized paid Checkout receipt and the
// stable Money Ledger outbox identity. It contains no email or raw payload.
type StripeTestEconomicReceipt struct {
	EventID, CheckoutSessionID, PaymentIntentID, SubscriptionID string
	RunID, Outcome, Currency, PaymentStatus, MoneyLedgerStatus  string
	AmountMinor                                                 int64
	MoneyLedgerExternalID                                       string
	MoneyLedgerPostingID, MoneyLedgerReceiptID                  string
}

func stripeTestOutcomeLink(metadata map[string]interface{}, livemode bool) (*StripeTestOutcomeLink, error) {
	runID, _ := metadata["lpbs_py1_test_run_id"].(string)
	outcome, _ := metadata["lpbs_py1_test_outcome"].(string)
	runID = strings.TrimSpace(runID)
	outcome = strings.TrimSpace(outcome)
	if runID == "" && outcome == "" {
		return nil, nil
	}
	if livemode {
		// Metadata is application-owned but can still be copied across a product
		// boundary. Never let it create a test receipt or block a valid live event.
		return nil, nil
	}
	if runID == "" || len(runID) > 100 || outcome == "" || !stripeTestOutcomeNameAllowed(outcome) {
		return nil, errors.New("Stripe test outcome metadata is incomplete or unsupported")
	}
	return &StripeTestOutcomeLink{RunID: runID, Outcome: outcome}, nil
}

func stripeTestOutcomeNameAllowed(name string) bool {
	switch name {
	case "real_webhook_delivery", "checkout_success", "decline", "3ds", "cancel", "upgrade", "downgrade", "renewal", "past_due", "credit_topup", "donation", "audio_tools_to_bundle_upgrade":
		return true
	default:
		return false
	}
}

func stripeObjectOutcomeMetadata(eventType string, object map[string]interface{}) map[string]interface{} {
	metadata := map[string]interface{}{}
	if top, ok := object["metadata"].(map[string]interface{}); ok {
		for key, value := range top {
			metadata[key] = value
		}
	}
	if eventType == "invoice.paid" || eventType == "invoice.payment_failed" {
		if details, ok := object["subscription_details"].(map[string]interface{}); ok {
			if nested, ok := details["metadata"].(map[string]interface{}); ok {
				for key, value := range nested {
					metadata[key] = value
				}
			}
		}
		// Stripe's newer invoice shape nests subscription metadata under
		// parent.subscription_details; retain compatibility with both API shapes.
		if parent, ok := object["parent"].(map[string]interface{}); ok {
			if details, ok := parent["subscription_details"].(map[string]interface{}); ok {
				if nested, ok := details["metadata"].(map[string]interface{}); ok {
					for key, value := range nested {
						metadata[key] = value
					}
				}
			}
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func stripeObjectID(object map[string]interface{}) string {
	value, _ := object["id"].(string)
	return strings.TrimSpace(value)
}

func (s *StripeWebhookService) beginWebhookReceipt(ctx context.Context, eventID, eventType, objectID string, livemode bool, body []byte, link *StripeTestOutcomeLink) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, errors.New("Stripe webhook receipt store is unavailable")
	}
	var claim [16]byte
	if _, err := rand.Read(claim[:]); err != nil {
		return "", false, fmt.Errorf("create Stripe webhook receipt claim: %w", err)
	}
	claimToken := hex.EncodeToString(claim[:])
	digest := sha256.Sum256(body)
	digestHex := hex.EncodeToString(digest[:])
	runID, outcome := "", ""
	if link != nil {
		runID, outcome = link.RunID, link.Outcome
	}
	result, err := s.db.Exec(`
		INSERT INTO stripe_webhook_events
			(stripe_event_id,livemode,event_type,stripe_object_id,payload_sha256,py1_test_run_id,py1_test_outcome,status,attempts)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),'received',0)
		ON CONFLICT (stripe_event_id) DO NOTHING
	`, eventID, livemode, eventType, objectID, digestHex, runID, outcome)
	if err != nil {
		return "", false, fmt.Errorf("persist Stripe webhook receipt: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("inspect Stripe webhook receipt insert: %w", err)
	}
	if inserted == 0 {
		var oldDigest, oldStatus string
		if err := s.db.QueryRowContext(ctx, `SELECT payload_sha256,status FROM stripe_webhook_events WHERE stripe_event_id=$1`, eventID).Scan(&oldDigest, &oldStatus); err != nil {
			return "", false, fmt.Errorf("load Stripe webhook receipt: %w", err)
		}
		if !strings.EqualFold(strings.TrimSpace(oldDigest), digestHex) {
			return "", false, errors.New("Stripe event ID was replayed with a different payload")
		}
		if oldStatus == "processed" {
			return "", false, nil
		}
		if oldStatus == "processing" {
			stale, err := s.db.ExecContext(ctx, `UPDATE stripe_webhook_events SET status='retryable_failed' WHERE stripe_event_id=$1 AND payload_sha256=$2 AND status='processing' AND last_seen_at < CURRENT_TIMESTAMP - ($3 * INTERVAL '1 second')`, eventID, digestHex, stripeWebhookProcessingLeaseSeconds)
			if err != nil {
				return "", false, fmt.Errorf("recover stale Stripe webhook receipt: %w", err)
			}
			rows, err := stale.RowsAffected()
			if err != nil {
				return "", false, fmt.Errorf("inspect stale Stripe webhook receipt: %w", err)
			}
			if rows != 1 {
				return "", false, nil
			}
		}
	}
	claimed, err := s.db.Exec(`
		UPDATE stripe_webhook_events
		SET status='processing', claim_token=$3, attempts=attempts+1, last_seen_at=CURRENT_TIMESTAMP, failure_code=NULL
		WHERE stripe_event_id=$1 AND payload_sha256=$2 AND status IN ('received','retryable_failed')
	`, eventID, digestHex, claimToken)
	if err != nil {
		return "", false, fmt.Errorf("claim Stripe webhook receipt: %w", err)
	}
	rows, err := claimed.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("inspect Stripe webhook receipt claim: %w", err)
	}
	if rows != 1 {
		return "", false, nil
	}
	return claimToken, true, nil
}

func (s *StripeWebhookService) finishWebhookReceipt(ctx context.Context, eventID, claimToken string, processErr error) error {
	if s == nil || s.db == nil || eventID == "" || claimToken == "" {
		return ErrStripeWebhookReceiptClaimLost
	}
	var err error
	var result sql.Result
	if processErr == nil {
		result, err = s.db.ExecContext(ctx, `UPDATE stripe_webhook_events SET status='processed',claim_token='',processed_at=CURRENT_TIMESTAMP,failure_code=NULL WHERE stripe_event_id=$1 AND claim_token=$2 AND status='processing'`, eventID, claimToken)
	} else {
		result, err = s.db.ExecContext(ctx, `UPDATE stripe_webhook_events SET status='retryable_failed',claim_token='',failure_code='handler_error' WHERE stripe_event_id=$1 AND claim_token=$2 AND status='processing'`, eventID, claimToken)
	}
	if err != nil {
		return fmt.Errorf("finish Stripe webhook receipt: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect Stripe webhook receipt finish: %w", err)
	}
	if rows != 1 {
		return ErrStripeWebhookReceiptClaimLost
	}
	return nil
}

func (s *StripeWebhookService) recordStripeTestEconomicReceipt(ctx context.Context, eventID string, object map[string]interface{}, link *StripeTestOutcomeLink) error {
	if s == nil || s.db == nil || link == nil {
		return errors.New("PY1 economic receipt store is unavailable")
	}
	sessionID := stripeObjectID(object)
	if !strings.HasPrefix(sessionID, "cs_test_") {
		return errors.New("test checkout receipt has no test Checkout Session ID")
	}
	if object["payment_status"] != "paid" {
		return nil
	}
	amount, ok := object["amount_total"].(float64)
	if !ok || amount <= 0 || amount != float64(int64(amount)) {
		return errors.New("paid test checkout has an invalid amount")
	}
	currency, _ := object["currency"].(string)
	currency = strings.ToLower(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return errors.New("paid test checkout has an invalid currency")
	}
	paymentIntentID := stripeReference(object["payment_intent"])
	subscriptionID := stripeReference(object["subscription"])
	if paymentIntentID == "" && subscriptionID == "" {
		return errors.New("paid test checkout has no payment intent or subscription identity")
	}
	externalID := "lpbs-py1-checkout:" + sessionID
	_, err := s.db.Exec(`
		INSERT INTO py1_test_economic_receipts
			(stripe_event_id,checkout_session_id,payment_intent_id,subscription_id,run_id,outcome,amount_minor,currency,payment_status,money_ledger_external_id)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,'paid',$9)
		ON CONFLICT (checkout_session_id) DO NOTHING
	`, eventID, sessionID, paymentIntentID, subscriptionID, link.RunID, link.Outcome, int64(amount), currency, externalID)
	if err != nil {
		return fmt.Errorf("persist PY1 paid Checkout receipt: %w", err)
	}
	var retained StripeTestEconomicReceipt
	err = s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,checkout_session_id,COALESCE(payment_intent_id,''),COALESCE(subscription_id,''),run_id,outcome,amount_minor,currency,payment_status,money_ledger_external_id,money_ledger_status,COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_economic_receipts WHERE checkout_session_id=$1
	`, sessionID).Scan(&retained.EventID, &retained.CheckoutSessionID, &retained.PaymentIntentID, &retained.SubscriptionID, &retained.RunID, &retained.Outcome, &retained.AmountMinor, &retained.Currency, &retained.PaymentStatus, &retained.MoneyLedgerExternalID, &retained.MoneyLedgerStatus, &retained.MoneyLedgerPostingID, &retained.MoneyLedgerReceiptID)
	if err != nil {
		return fmt.Errorf("read back PY1 paid Checkout receipt: %w", err)
	}
	if retained.RunID != link.RunID || retained.Outcome != link.Outcome || retained.AmountMinor != int64(amount) || retained.Currency != currency || retained.PaymentIntentID != paymentIntentID || retained.SubscriptionID != subscriptionID || retained.MoneyLedgerExternalID != externalID {
		return errors.New("Checkout Session ID was replayed with conflicting PY1 economic data")
	}
	return nil
}

// LoadTestEconomicReceipt reads back a verified paid test Checkout and its
// durable ledger-outbox state by run/outcome correlation.
func (s *StripeWebhookService) LoadTestEconomicReceipt(ctx context.Context, runID, outcome string) (*StripeTestEconomicReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) {
		return nil, errors.New("invalid PY1 economic receipt lookup")
	}
	var receipt StripeTestEconomicReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,checkout_session_id,COALESCE(payment_intent_id,''),COALESCE(subscription_id,''),run_id,outcome,amount_minor,currency,payment_status,money_ledger_external_id,money_ledger_status,COALESCE(money_ledger_posting_id,''),COALESCE(money_ledger_receipt_id,'')
		FROM py1_test_economic_receipts WHERE run_id=$1 AND outcome=$2 ORDER BY created_at DESC LIMIT 1
	`, runID, outcome).Scan(&receipt.EventID, &receipt.CheckoutSessionID, &receipt.PaymentIntentID, &receipt.SubscriptionID, &receipt.RunID, &receipt.Outcome, &receipt.AmountMinor, &receipt.Currency, &receipt.PaymentStatus, &receipt.MoneyLedgerExternalID, &receipt.MoneyLedgerStatus, &receipt.MoneyLedgerPostingID, &receipt.MoneyLedgerReceiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load PY1 paid Checkout receipt: %w", err)
	}
	return &receipt, nil
}

func (s *StripeWebhookService) loadTestOutcomeWebhookReceipt(ctx context.Context, runID, outcome string) (*StripeWebhookReceipt, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || !stripeTestOutcomeNameAllowed(outcome) {
		return nil, errors.New("invalid Stripe test outcome receipt lookup")
	}
	var receipt StripeWebhookReceipt
	err := s.db.QueryRowContext(ctx, `
		SELECT stripe_event_id,event_type,stripe_object_id,py1_test_run_id,py1_test_outcome,status,livemode
		FROM stripe_webhook_events
		WHERE py1_test_run_id=$1 AND py1_test_outcome=$2 AND status='processed' AND livemode=FALSE
		ORDER BY processed_at DESC LIMIT 1
	`, runID, outcome).Scan(&receipt.EventID, &receipt.EventType, &receipt.ObjectID, &receipt.RunID, &receipt.Outcome, &receipt.Status, &receipt.Livemode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load Stripe test outcome webhook receipt: %w", err)
	}
	return &receipt, nil
}

// LoadTestOutcomeWebhookReceipt returns the accepted, processed event receipt
// associated with a PY1 test outcome. It exposes only safe identifiers.
func (s *StripeWebhookService) LoadTestOutcomeWebhookReceipt(ctx context.Context, runID, outcome string) (*StripeWebhookReceipt, error) {
	return s.loadTestOutcomeWebhookReceipt(ctx, runID, outcome)
}
