package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	fundingflow "landing-page-business-suite-api/internal/commerce/fundinglot/flow"
	fundingstate "landing-page-business-suite-api/internal/commerce/fundinglot/flow/generated"
)

type spendableFundingLot struct {
	id                 string
	sourceKind         string
	remaining          int64
	expiresAt          sql.NullTime
	reversalWindowEnds sql.NullTime
	createdAt          time.Time
	status             string
	restricted         bool
	debitRemaining     int64
}

func spendableFundingBalance(ctx context.Context, tx *sql.Tx, dialect, accountID string, liveMode bool, now time.Time) (int64, error) {
	query := `SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id=? AND status='open' AND settlement_state='settled' AND (spendable_after IS NULL OR spendable_after<=?) AND (expires_at IS NULL OR expires_at>?) AND livemode=?`
	if dialect != "sqlite" {
		query = `SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots WHERE business_account_id=$1 AND status='open' AND settlement_state='settled' AND (spendable_after IS NULL OR spendable_after<=$2) AND (expires_at IS NULL OR expires_at>$2) AND livemode=$3`
		returnValue := int64(0)
		err := tx.QueryRowContext(ctx, query, accountID, now, liveMode).Scan(&returnValue)
		return returnValue, err
	}
	var available int64
	err := tx.QueryRowContext(ctx, query, accountID, now, now, liveMode).Scan(&available)
	return available, err
}

// pendingFundingReservations is the shared funding-availability read used by
// admission and display. Subscription caps remain account-wide; funding holds
// belong to the immutable mode recorded when their reservation was created.
func pendingFundingReservations(ctx context.Context, tx *sql.Tx, dialect, accountID string, liveMode bool, now time.Time) (int64, error) {
	query := `SELECT COALESCE(SUM(CASE WHEN livemode=$3 THEN reserved_amount ELSE 0 END),0),COUNT(CASE WHEN livemode IS NULL THEN 1 END) FROM business_account_credit_reservations WHERE business_account_id=$1 AND status='pending' AND expires_at>$2`
	args := []any{accountID, now, liveMode}
	if dialect == "sqlite" {
		query = `SELECT COALESCE(SUM(CASE WHEN livemode=? THEN reserved_amount ELSE 0 END),0),COUNT(CASE WHEN livemode IS NULL THEN 1 END) FROM business_account_credit_reservations WHERE business_account_id=? AND status='pending' AND julianday(expires_at)>julianday(?)`
		args = []any{liveMode, accountID, now.Format(time.RFC3339Nano)}
	}
	var reserved, unknown int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&reserved, &unknown); err != nil {
		return 0, err
	}
	if unknown > 0 {
		return 0, errors.New("pending reservation funding mode is unavailable")
	}
	return reserved, nil
}

func reservationFundingMode(stored sql.NullBool, expected bool) error {
	if !stored.Valid {
		return errors.New("reservation funding mode is unavailable")
	}
	if stored.Bool != expected {
		return errors.New("reservation funding mode does not match this operation")
	}
	return nil
}

func (s *BusinessAccountUsageService) checkReservationFundingMode(ctx context.Context, account, id string) error {
	var mode sql.NullBool
	err := s.db.QueryRowContext(ctx, `SELECT livemode FROM business_account_credit_reservations WHERE business_account_id=$1 AND id=$2`, account, id).Scan(&mode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return reservationFundingMode(mode, s.liveMode)
}

func loadSpendableLots(ctx context.Context, tx *sql.Tx, dialect, accountID string, liveMode bool, now time.Time) ([]spendableFundingLot, error) {
	query := `SELECT id,source_kind,remaining,expires_at,reversal_window_ends,created_at FROM credit_funding_lots WHERE business_account_id=? AND status='open' AND settlement_state='settled' AND remaining>0 AND (spendable_after IS NULL OR spendable_after<=?) AND (expires_at IS NULL OR expires_at>?) AND livemode=?`
	args := []any{accountID, now, now, liveMode}
	if dialect != "sqlite" {
		query = `SELECT id,source_kind,remaining,expires_at,reversal_window_ends,created_at FROM credit_funding_lots WHERE business_account_id=$1 AND status='open' AND settlement_state='settled' AND remaining>0 AND (spendable_after IS NULL OR spendable_after<=$2) AND (expires_at IS NULL OR expires_at>$2) AND livemode=$3 FOR UPDATE`
		args = []any{accountID, now, liveMode}
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lots := make([]spendableFundingLot, 0)
	for rows.Next() {
		var lot spendableFundingLot
		if err := rows.Scan(&lot.id, &lot.sourceKind, &lot.remaining, &lot.expiresAt, &lot.reversalWindowEnds, &lot.createdAt); err != nil {
			return nil, err
		}
		lot.status = "open"
		lot.debitRemaining = lot.remaining
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortFundingLots(lots, now)
	return lots, nil
}

func allocateFundingLots(ctx context.Context, tx *sql.Tx, dialect, accountID, operationID string, amount, costMicros int64, liveMode bool, now time.Time, reserved bool) error {
	var lots []spendableFundingLot
	var err error
	if reserved && dialect != "sqlite" {
		lots, err = loadReservedSettlementLots(ctx, tx, accountID, operationID, liveMode, now)
	} else {
		lots, err = loadSpendableLots(ctx, tx, dialect, accountID, liveMode, now)
	}
	if err != nil {
		return fmt.Errorf("load spendable funding lots: %w", err)
	}
	var available int64
	for _, lot := range lots {
		available += lot.remaining
	}
	if amount > available {
		return errors.New("spendable funding lots changed before settlement")
	}
	leftCredits, leftCost := amount, costMicros
	for i, lot := range lots {
		if leftCredits == 0 {
			break
		}
		if lot.remaining <= 0 {
			continue
		}
		credits := minInt64(leftCredits, lot.remaining)
		cost := int64(0)
		if credits > 0 {
			if i == len(lots)-1 || credits == leftCredits {
				cost = leftCost
			} else if amount > 0 {
				cost = (costMicros * credits) / amount
			}
		}
		debit := credits
		if lot.restricted {
			debit = minInt64(credits, lot.debitRemaining)
		}
		var event fundingstate.FundingLotEvent
		nextStatus := fundingstate.FundingLotStatus(lot.status)
		var err error
		if lot.status == "open" && debit > 0 {
			event = fundingstate.FundingLotDebit
			if debit == lot.debitRemaining {
				event = fundingstate.FundingLotExhaust
			}
			nextStatus, err = fundingflow.TransitionFundingLot(fundingstate.FundingLotOpen, event)
		}
		if err != nil {
			return err
		}
		oldStatus := lot.status
		update := `UPDATE credit_funding_lots SET remaining=remaining-?,status=? WHERE id=? AND business_account_id=? AND status=? AND remaining>=?`
		insert := `INSERT INTO credit_lot_allocations (id,lot_id,operation_id,credits,cost_micros,created_at) VALUES (?,?,?,?,?,?) ON CONFLICT (lot_id,operation_id) DO NOTHING`
		if dialect != "sqlite" {
			update = `UPDATE credit_funding_lots SET remaining=remaining-$1,status=$2 WHERE id=$3 AND business_account_id=$4 AND status=$5 AND remaining>=$1`
			insert = `INSERT INTO credit_lot_allocations (id,lot_id,operation_id,credits,cost_micros,created_at) VALUES ($1,$2,$3,$4,$5,NOW()) ON CONFLICT (lot_id,operation_id) DO NOTHING`
		}
		var result sql.Result
		if dialect == "sqlite" {
			result, err = tx.ExecContext(ctx, update, debit, string(nextStatus), lot.id, accountID, oldStatus, debit)
		} else {
			result, err = tx.ExecContext(ctx, update, debit, string(nextStatus), lot.id, accountID, oldStatus)
		}
		if err != nil {
			return fmt.Errorf("decrement funding lot %s: %w", lot.id, err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return errors.New("funding lot changed during allocation")
		}
		digest := sha256.Sum256([]byte(operationID + ":" + lot.id))
		allocationID := fmt.Sprintf("%x-%x-%x-%x-%x", digest[:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
		if dialect == "sqlite" {
			if _, err := tx.ExecContext(ctx, insert, allocationID, lot.id, operationID, credits, cost, now); err != nil {
				return fmt.Errorf("insert funding allocation: %w", err)
			}
		} else if _, err := tx.ExecContext(ctx, insert, allocationID, lot.id, operationID, credits, cost); err != nil {
			return fmt.Errorf("insert funding allocation: %w", err)
		}
		if lot.restricted {
			if err := refreshFundingLoss(ctx, tx, lot.id, accountID); err != nil {
				return err
			}
		}
		leftCredits -= credits
		leftCost -= cost
	}
	if leftCredits != 0 || leftCost != 0 {
		return fmt.Errorf("funding allocation incomplete: credits=%d cost_micros=%d", leftCredits, leftCost)
	}
	return nil
}

func retryableCommerceTransaction(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"serialization", "deadlock", "database is locked", "busy", "sqlstate 40001", "sqlstate 40p01"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func waitCommerceRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(5+rand.Intn(5)) * time.Millisecond * time.Duration(1<<attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Prior holds settle even if their funding is reversed while a supplier runs.
// The immutable reservation timestamp fences out calls admitted after a freeze;
// original grant minus allocations bounds every such settlement. Unfunded
// provider cost is attributed to the same lot instead of making a wallet negative.
func loadReservedSettlementLots(ctx context.Context, tx *sql.Tx, account, operation string, live bool, now time.Time) ([]spendableFundingLot, error) {
	rows, err := tx.QueryContext(ctx, `SELECT l.id,l.source_kind,l.remaining,l.expires_at,l.reversal_window_ends,l.created_at,l.status,
 (p.restricted_at IS NOT NULL AND p.restricted_at>=r.created_at),
 l.granted-COALESCE((SELECT SUM(credits) FROM credit_lot_allocations WHERE lot_id=l.id),0)
 FROM credit_funding_lots l JOIN business_account_credit_reservations r ON r.id=$2 AND r.business_account_id=l.business_account_id
 LEFT JOIN funding_payment_states p ON p.lot_id=l.id
 WHERE l.business_account_id=$1 AND l.livemode=$3 AND ((l.status='open' AND l.settlement_state='settled' AND (l.spendable_after IS NULL OR l.spendable_after<=$4) AND (l.expires_at IS NULL OR l.expires_at>$4)) OR (p.restricted_at>=r.created_at))
 ORDER BY l.expires_at NULLS LAST,l.reversal_window_ends NULLS FIRST,l.created_at,l.id FOR UPDATE OF l`, account, operation, live, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lots []spendableFundingLot
	for rows.Next() {
		var lot spendableFundingLot
		var capacity int64
		var restricted sql.NullBool
		if err := rows.Scan(&lot.id, &lot.sourceKind, &lot.remaining, &lot.expiresAt, &lot.reversalWindowEnds, &lot.createdAt, &lot.status, &restricted, &capacity); err != nil {
			return nil, err
		}
		lot.restricted = restricted.Bool
		lot.debitRemaining = lot.remaining
		if lot.restricted {
			lot.remaining = capacity
		}
		lots = append(lots, lot)
	}
	sortFundingLots(lots, now)
	return lots, rows.Err()
}

func sortFundingLots(lots []spendableFundingLot, now time.Time) {
	sort.Slice(lots, func(i, j int) bool {
		left, right := lots[i], lots[j]
		if left.expiresAt.Valid != right.expiresAt.Valid {
			return left.expiresAt.Valid
		}
		if left.expiresAt.Valid && !left.expiresAt.Time.Equal(right.expiresAt.Time) {
			return left.expiresAt.Time.Before(right.expiresAt.Time)
		}
		leftIrreversible := !left.reversalWindowEnds.Valid || !left.reversalWindowEnds.Time.After(now)
		rightIrreversible := !right.reversalWindowEnds.Valid || !right.reversalWindowEnds.Time.After(now)
		if leftIrreversible != rightIrreversible {
			return leftIrreversible
		}
		if !leftIrreversible && !left.reversalWindowEnds.Time.Equal(right.reversalWindowEnds.Time) {
			return left.reversalWindowEnds.Time.Before(right.reversalWindowEnds.Time)
		}
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.Before(right.createdAt)
		}
		return left.id < right.id
	})
}
