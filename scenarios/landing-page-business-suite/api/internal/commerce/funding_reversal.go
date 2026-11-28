package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	fundingflow "landing-page-business-suite-api/internal/commerce/fundinglot/flow"
	fundingstate "landing-page-business-suite-api/internal/commerce/fundinglot/flow/generated"
)

// Stripe references can be expanded objects or their stable IDs.
func stripeReference(value any) string {
	if id, ok := value.(string); ok {
		return id
	}
	if object, ok := value.(map[string]interface{}); ok {
		id, _ := object["id"].(string)
		return id
	}
	return ""
}

func (s *StripeWebhookService) handleFundingReversal(ctx context.Context, eventID, kind string, object map[string]interface{}, live bool) error {
	if s.db == nil {
		return errors.New("funding authority unavailable")
	}
	payment, charge := stripeReference(object["payment_intent"]), stripeReference(object["charge"])
	if kind == "charge.refunded" {
		charge = stripeReference(object["id"])
	}
	if payment == "" && charge == "" {
		return errors.New("funding event lacks payment or charge reference")
	}
	account, lot, err := s.matchFundingLot(ctx, payment, charge, live)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, account); err != nil {
		return err
	}
	if charge != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE credit_funding_lots SET provider_ref2=$2 WHERE id=$1`, lot, charge); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(map[string]any{"type": kind, "object": object})
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO funding_provider_events(event_id,livemode,payload) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, eventID, live, string(payload))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var equal bool
		err = tx.QueryRowContext(ctx, `SELECT payload=$3::jsonb FROM funding_provider_events WHERE event_id=$1 AND livemode=$2`, eventID, live, string(payload)).Scan(&equal)
		if err != nil {
			return err
		}
		if !equal {
			return errors.New("funding event replay changed payload")
		}
		return nil
	}
	if err = applyFundingReversal(ctx, tx, lot, account, kind, object); err != nil {
		return err
	}
	return tx.Commit()
}

func applyFundingReversal(ctx context.Context, tx *sql.Tx, lot, account, kind string, object map[string]interface{}) error {
	var status string
	var granted, remaining int64
	var paid sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT status,granted,remaining,paid_amount_minor FROM credit_funding_lots WHERE id=$1 FOR UPDATE`, lot).Scan(&status, &granted, &remaining, &paid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO funding_payment_states(lot_id) VALUES($1) ON CONFLICT DO NOTHING`, lot); err != nil {
		return err
	}
	var fraudWarning bool
	var refunded int64
	if err := tx.QueryRowContext(ctx, `SELECT refunded_minor,fraud_warning FROM funding_payment_states WHERE lot_id=$1`, lot).Scan(&refunded, &fraudWarning); err != nil {
		return err
	}
	ref := stripeReference(object["id"])
	if ref == "" {
		return errors.New("funding provider object ID required")
	}
	if kind == "charge.refunded" {
		amount, err := validStripeMinorAmount(object["amount_refunded"])
		if err != nil {
			return err
		}
		if !paid.Valid || paid.Int64 <= 0 || amount < 0 || amount > paid.Int64 {
			return errors.New("refund amount has no valid payment basis")
		}
		target := amount
		if target <= refunded {
			return nil
		}
		// Cumulative amount_refunded makes reordered partial-refund events safe.
		// Integer division rounds removed credits upward, so a refunded fraction
		// cannot leave an extra spendable credit.
		removed := proportionalCredits(granted, target, paid.Int64) - proportionalCredits(granted, refunded, paid.Int64)
		remaining -= minInt64(remaining, removed)
		if target == paid.Int64 {
			remaining = 0
			if status == "open" || status == "frozen" {
				next, err := fundingflow.TransitionFundingLot(fundingstate.FundingLotStatus(status), fundingstate.FundingLotReverse)
				if err != nil {
					return err
				}
				status = string(next)
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE credit_funding_lots SET remaining=$2,status=$3 WHERE id=$1`, lot, remaining, status); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE funding_payment_states SET refunded_minor=$2,restricted_at=clock_timestamp() WHERE lot_id=$1`, lot, target); err != nil {
			return err
		}
		return refreshFundingLoss(ctx, tx, lot, account)
	}
	if strings.HasPrefix(kind, "radar.") {
		if status == "open" {
			if err := transitionFundingStatus(ctx, tx, lot, status, fundingstate.FundingLotFreeze); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE funding_payment_states SET fraud_warning=TRUE WHERE lot_id=$1`, lot)
		return err
	}
	incoming, _ := object["status"].(string)
	var current string
	err := tx.QueryRowContext(ctx, `SELECT status FROM funding_disputes WHERE lot_id=$1 AND dispute_id=$2`, lot, ref).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	next, err := nextDisputeStatus(current, incoming)
	if err != nil {
		return err
	}
	if next != incoming {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO funding_disputes(lot_id,dispute_id,status,amount_minor) VALUES($1,$2,$3,$4) ON CONFLICT(lot_id,dispute_id) DO UPDATE SET status=EXCLUDED.status,amount_minor=GREATEST(funding_disputes.amount_minor,EXCLUDED.amount_minor)`, lot, ref, next, stripeAmount(object["amount"])); err != nil {
		return err
	}
	if err := recordStripeDisputeTransactions(ctx, tx, lot, ref, object); err != nil {
		return err
	}

	var active bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM funding_disputes WHERE lot_id=$1 AND status NOT IN ('won','warning_closed','prevented'))`, lot).Scan(&active); err != nil {
		return err
	}
	if status == "open" && active {
		if err := transitionFundingStatus(ctx, tx, lot, status, fundingstate.FundingLotFreeze); err != nil {
			return err
		}
	}
	if status == "frozen" && !active && !fraudWarning {
		if err := transitionFundingStatus(ctx, tx, lot, status, fundingstate.FundingLotUnfreeze); err != nil {
			return err
		}
	}
	if next == "lost" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO funding_purchase_blocks(business_account_id,reason,event_ref) VALUES($1,'lost_dispute',$2) ON CONFLICT DO NOTHING`, account, ref); err != nil {
			return err
		}
	}

	return refreshFundingLoss(ctx, tx, lot, account)
}

func proportionalCredits(credits, part, total int64) int64 {
	var value, divisor big.Int
	value.Mul(big.NewInt(credits), big.NewInt(part))
	divisor.SetInt64(total)
	value.Add(&value, new(big.Int).Sub(&divisor, big.NewInt(1)))
	value.Div(&value, &divisor)
	return value.Int64()
}
func transitionFundingStatus(ctx context.Context, tx *sql.Tx, lot, status string, event fundingstate.FundingLotEvent) error {
	next, err := fundingflow.TransitionFundingLot(fundingstate.FundingLotStatus(status), event)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE credit_funding_lots SET status=$2 WHERE id=$1`, lot, string(next))
	if err != nil {
		return err
	}
	if event == fundingstate.FundingLotFreeze {
		// Each actual restriction cycle must include holds admitted since a
		// previous recovery/refund. A duplicate event on an already-frozen lot
		// does not enter this transition and cannot admit post-freeze holds.
		_, err = tx.ExecContext(ctx, `UPDATE funding_payment_states SET restricted_at=clock_timestamp() WHERE lot_id=$1`, lot)
	}
	return err
}

// One derived loss per lot prevents a refund plus dispute from counting the
// same inference cost twice. Immutable provider events preserve the audit trail.
func refreshFundingLoss(ctx context.Context, tx *sql.Tx, lot, account string) error {
	var granted, spent, cost, refunded, disputed, fee int64
	var paid sql.NullInt64
	var active bool
	err := tx.QueryRowContext(ctx, `SELECT l.granted,l.paid_amount_minor,p.refunded_minor,
 EXISTS(SELECT 1 FROM funding_disputes WHERE lot_id=l.id AND status IN ('needs_response','under_review','appeal_needs_response','appeal_under_review','lost')),
 COALESCE((SELECT SUM(COALESCE((SELECT -SUM(b.amount_minor) FROM funding_dispute_balance_transactions b WHERE b.lot_id=d.lot_id AND b.dispute_id=d.dispute_id),d.amount_minor)) FROM funding_disputes d WHERE d.lot_id=l.id AND d.status IN ('needs_response','under_review','appeal_needs_response','appeal_under_review','lost')),0),
 COALESCE((SELECT SUM(fee_minor) FROM funding_dispute_balance_transactions WHERE lot_id=l.id),0),
 COALESCE((SELECT SUM(credits) FROM credit_lot_allocations WHERE lot_id=l.id),0),COALESCE((SELECT SUM(cost_micros) FROM credit_lot_allocations WHERE lot_id=l.id),0)
 FROM credit_funding_lots l JOIN funding_payment_states p ON p.lot_id=l.id WHERE l.id=$1`, lot).Scan(&granted, &paid, &refunded, &active, &disputed, &fee, &spent, &cost)
	if err != nil {
		return err
	}
	loss := int64(0)
	if active {
		loss = cost
	} else if paid.Valid && paid.Int64 > 0 && spent > 0 {
		lostCredits := proportionalCredits(granted, refunded, paid.Int64) - (granted - spent)
		if lostCredits > 0 {
			loss = proportionalCredits(cost, minInt64(lostCredits, spent), spent)
		}
	}

	revenue := refunded + disputed
	var oldCost, oldRefund, oldDispute, oldFee int64
	err = tx.QueryRowContext(ctx, `SELECT consumed_cost_micros,refunded_revenue_minor,disputed_revenue_minor,fee_minor FROM funding_losses WHERE lot_id=$1 AND kind='funding_reversal'`, lot).Scan(&oldCost, &oldRefund, &oldDispute, &oldFee)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO funding_losses(lot_id,kind,consumed_cost_micros,reversed_revenue_minor,refunded_revenue_minor,disputed_revenue_minor,fee_minor,recorded_to_account,event_ref) VALUES($1::uuid,'funding_reversal',$3,$4,$5,$6,$7,$2,$1::uuid::text) ON CONFLICT(lot_id,event_ref,kind) DO UPDATE SET consumed_cost_micros=EXCLUDED.consumed_cost_micros,reversed_revenue_minor=EXCLUDED.reversed_revenue_minor,refunded_revenue_minor=EXCLUDED.refunded_revenue_minor,disputed_revenue_minor=EXCLUDED.disputed_revenue_minor,fee_minor=EXCLUDED.fee_minor`, lot, account, loss, revenue, refunded, disputed, fee)
	if err != nil {
		return err
	}
	// Money Ledger speaks currency minor units. Round cumulative USD cost once,
	// then send its delta; sub-cent changes cannot accumulate rounding errors.
	deltas := []struct {
		category string
		amount   int64
	}{{"loss", roundedCostMinor(loss) - roundedCostMinor(oldCost)}, {"refund", refunded - oldRefund}, {"dispute", disputed - oldDispute}, {"fee", fee - oldFee}}
	for _, delta := range deltas {
		if delta.amount != 0 {
			if _, err = tx.ExecContext(ctx, `INSERT INTO funding_ledger_outbox(lot_id,category,amount_minor,currency) SELECT $1,$2,$3,CASE WHEN $2='loss' THEN 'USD' ELSE UPPER(paid_currency) END FROM credit_funding_lots WHERE id=$1`, lot, delta.category, delta.amount); err != nil {
				return err
			}
		}
	}

	return err
}

func checkFundingPurchaseBlock(ctx context.Context, tx *sql.Tx, account string) error {
	var blocked bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM funding_purchase_blocks WHERE business_account_id=$1)`, account).Scan(&blocked); err != nil {
		return err
	}
	if blocked {
		return errors.New("purchases blocked after lost dispute")
	}
	return nil
}
func applyFundingPolicy(ctx context.Context, tx *sql.Tx, lot, account, provider, source string) error {
	// No application default sets a trust bar. Policy values remain data.
	_, err := tx.ExecContext(ctx, `UPDATE credit_funding_lots l SET policy_id=p.id,spendable_after=CASE WHEN a.created_at+make_interval(secs=>p.young_account_seconds::double precision)>NOW() THEN NOW()+make_interval(secs=>p.young_hold_seconds::double precision) ELSE NULL END FROM funding_source_policies p,business_accounts a WHERE l.id=$1 AND a.id=$2 AND p.provider=$3 AND p.source_kind=$4`, lot, account, provider, source)
	return err
}

func checkFundingAgeCap(ctx context.Context, tx *sql.Tx, account string, amount int64, live bool) error {
	var cap sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT MIN(CASE WHEN a.created_at+make_interval(secs=>p.young_account_seconds::double precision)>NOW() THEN p.young_cap_credits ELSE p.mature_cap_credits END) FROM credit_funding_lots l JOIN funding_source_policies p ON p.id=l.policy_id JOIN business_accounts a ON a.id=l.business_account_id WHERE l.business_account_id=$1 AND l.status='open' AND l.livemode=$2`, account, live).Scan(&cap)
	if err != nil {
		return err
	}
	if cap.Valid && amount > cap.Int64 {
		return errors.New("insufficient credits: funding account-age cap exceeded")
	}
	return nil
}

func stripeAmount(value any) int64 { number, _ := value.(float64); return int64(number) }

func roundedCostMinor(cost int64) int64 {
	result := cost / 10000
	if cost%10000 >= 5000 {
		result++
	}
	return result
}

func nextDisputeStatus(current, incoming string) (string, error) {
	switch incoming {
	case "warning_needs_response", "warning_under_review", "warning_closed", "needs_response", "under_review", "appeal_needs_response", "appeal_under_review", "won", "lost", "prevented":
	default:
		return current, errors.New("unknown Stripe dispute state")
	}
	switch current {
	case "won", "warning_closed", "prevented":
		return current, nil
	case "lost":
		// Stripe can return funds after a late win. This correction does not
		// erase a separate purchase block or another dispute's freeze.
		if incoming == "won" {
			return incoming, nil
		}
		return current, nil
	case "needs_response", "under_review", "appeal_needs_response", "appeal_under_review":
		// Inquiry snapshots predate formal chargeback states. They cannot undo
		// its principal/loss or thaw its lot when delivered out of order.
		if strings.HasPrefix(incoming, "warning_") {
			return current, nil
		}
	}
	return incoming, nil
}
func validStripeMinorAmount(value any) (int64, error) {
	number, err := signedStripeMinorAmount(value)
	if err != nil {
		return 0, err
	}
	if number < 0 {
		return 0, errors.New("negative Stripe minor-unit amount")
	}
	return number, nil
}

func (s *StripeWebhookService) matchFundingLot(ctx context.Context, payment, charge string, live bool) (string, string, error) {
	var account, lot string
	var matches int
	match := func() error {
		return s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(MIN(business_account_id),''),COALESCE(MIN(id::text),'') FROM credit_funding_lots WHERE provider='stripe' AND livemode=$1 AND (($2<>'' AND provider_ref=$2) OR ($3<>'' AND provider_ref2=$3))`, live, payment, charge).Scan(&matches, &account, &lot)
	}
	if err := match(); err != nil {
		return "", "", err
	}
	// Stripe permits a nullable payment_intent on disputes/warnings. Resolve
	// the signed charge reference through the existing authenticated Stripe edge;
	// never infer an account from customer email or untrusted metadata.
	if matches == 0 && payment == "" && charge != "" && s.provider != nil {
		raw, err := s.provider.Request(ctx, http.MethodGet, "/charges/"+url.PathEscape(charge), nil, "")
		if err != nil {
			return "", "", err
		}
		var object map[string]interface{}
		if err := json.Unmarshal(raw, &object); err != nil {
			return "", "", err
		}
		mode, ok := object["livemode"].(bool)
		if !ok || mode != live || stripeReference(object["id"]) != charge {
			return "", "", errors.New("Stripe charge attribution mismatched mode or reference")
		}
		payment = stripeReference(object["payment_intent"])
		if payment == "" {
			return "", "", errors.New("Stripe charge lacks payment attribution")
		}
		if err := match(); err != nil {
			return "", "", err
		}
	}
	if matches != 1 {
		return "", "", errors.New("funding event payment attribution missing or ambiguous")
	}
	return account, lot, nil
}

// Reordered snapshots contribute immutable transaction IDs, never overwrite a
// cumulative counter. Principal is the actual gross balance movement; fees are
// separate. A partially won dispute can remain lost while positive transactions
// return part of its original principal.
func recordStripeDisputeTransactions(ctx context.Context, tx *sql.Tx, lot, dispute string, object map[string]interface{}) error {
	transactions, _ := object["balance_transactions"].([]interface{})
	for _, raw := range transactions {
		row, expanded := raw.(map[string]interface{})
		if !expanded {
			continue
		}
		id := stripeReference(row["id"])
		if id == "" {
			return errors.New("expanded Stripe balance transaction lacks ID")
		}
		amount, err := signedStripeMinorAmount(row["amount"])
		if err != nil {
			return err
		}
		fee, err := signedStripeMinorAmount(row["fee"])
		if err != nil {
			return err
		}
		currency, _ := row["currency"].(string)
		currency = strings.ToUpper(strings.TrimSpace(currency))
		var paidCurrency sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT paid_currency FROM credit_funding_lots WHERE id=$1`, lot).Scan(&paidCurrency); err != nil {
			return err
		}
		if !paidCurrency.Valid || currency == "" || !strings.EqualFold(currency, paidCurrency.String) {
			return errors.New("Stripe balance currency has no matching funding basis")
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO funding_dispute_balance_transactions(lot_id,dispute_id,transaction_id,amount_minor,fee_minor,currency) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(lot_id,dispute_id,transaction_id) DO UPDATE SET transaction_id=EXCLUDED.transaction_id WHERE funding_dispute_balance_transactions.amount_minor=EXCLUDED.amount_minor AND funding_dispute_balance_transactions.fee_minor=EXCLUDED.fee_minor AND funding_dispute_balance_transactions.currency=EXCLUDED.currency`, lot, dispute, id, amount, fee, currency)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return errors.New("Stripe balance transaction changed immutable money facts")
		}
	}
	return nil
}
func signedStripeMinorAmount(value any) (int64, error) {
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || math.Abs(number) > 9007199254740991 || number != math.Trunc(number) {
		return 0, errors.New("invalid signed Stripe minor-unit amount")
	}
	return int64(number), nil
}
