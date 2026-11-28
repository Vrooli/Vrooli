package commerce

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	reservationstate "landing-page-business-suite-api/internal/commerce/reservation/flow/generated"
)

var ErrBrowserGatewayOperationReplay = errors.New("Gateway operation is already recorded for this browser session; reconcile its receipt")
var ErrBrowserSessionOperationsPending = errors.New("browser session has an uncertain or incomplete Gateway operation")

func validFingerprint(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type BrowserGatewayReceipt struct {
	RootOperationID    string
	RequestFingerprint string
	State              string
	Provider           string
	Model              string
	Credits            int64
	CostMicros         int64
	PromptTokens       int64
	CompletionTokens   int64
	ReceiptFingerprint string
}

type BrowserGatewayAggregate struct {
	Credits          int64
	CostMicros       int64
	PromptTokens     int64
	CompletionTokens int64
	Operations       int
}

// BindBrowserSessionIdentity binds a pending account hold to the authenticated
// run, grant fingerprint and exact BAS session before model dispatch. Replays
// are accepted only for the identical tuple; raw grant receipts are never stored.
func (s *BusinessAccountUsageService) BindBrowserSessionIdentity(ctx context.Context, accountID, actor, reservationID, tier, runID, sessionID, grantFingerprint string) error {
	accountID, actor = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor))
	reservationID, runID, sessionID = strings.TrimSpace(reservationID), strings.TrimSpace(runID), strings.TrimSpace(sessionID)
	grantFingerprint = strings.TrimSpace(grantFingerprint)
	tier = strings.TrimSpace(tier)
	if s.db == nil || accountID == "" || actor == "" || reservationID == "" || tier == "" || runID == "" || sessionID == "" || !validFingerprint(grantFingerprint) {
		return errors.New("account, actor, reservation, tier, run, session and grant fingerprint are required")
	}
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return fmt.Errorf("begin browser session binding: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	query := `SELECT actor_identity,status,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_evidence_fingerprint,browser_tier FROM business_account_credit_reservations WHERE id=? AND business_account_id=?`
	if s.dialect != "sqlite" {
		query = `SELECT actor_identity,status,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_evidence_fingerprint,browser_tier FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2 FOR UPDATE`
	}
	var savedActor, status, savedRun, savedSession, savedGrant, proof, savedTier string
	if err := tx.QueryRowContext(ctx, query, reservationID, accountID).Scan(&savedActor, &status, &savedRun, &savedSession, &savedGrant, &proof, &savedTier); err != nil {
		return fmt.Errorf("load browser session reservation: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(savedActor), actor) {
		return ErrReservationNotOwned
	}
	if status != string(reservationstate.ReservationPending) {
		return errors.New("browser session identity can bind only to a pending reservation")
	}
	if proof != "" {
		return errors.New("closed-session proof cannot be bound before session reconciliation")
	}
	if savedRun != "" || savedSession != "" || savedGrant != "" {
		if savedRun == runID && savedSession == sessionID && savedGrant == grantFingerprint && savedTier == tier {
			return tx.Commit()
		}
		return errors.New("browser session reservation is already bound to another tier, run, grant or session")
	}
	update := `UPDATE business_account_credit_reservations SET browser_run_id=?,browser_session_id=?,browser_grant_fingerprint=?,browser_tier=? WHERE id=? AND business_account_id=? AND status=? AND browser_run_id='' AND browser_session_id='' AND browser_grant_fingerprint='' AND browser_tier=''`
	if s.dialect != "sqlite" {
		update = `UPDATE business_account_credit_reservations SET browser_run_id=$1,browser_session_id=$2,browser_grant_fingerprint=$3,browser_tier=$4 WHERE id=$5 AND business_account_id=$6 AND status=$7 AND browser_run_id='' AND browser_session_id='' AND browser_grant_fingerprint='' AND browser_tier=''`
	}
	result, err := tx.ExecContext(ctx, update, runID, sessionID, grantFingerprint, tier, reservationID, accountID, string(reservationstate.ReservationPending))
	if err != nil {
		return fmt.Errorf("bind browser session identity: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return errors.New("browser session identity was concurrently changed")
	}
	return tx.Commit()
}

// BeginBrowserGatewayReceipt durably claims a unique Gateway operation before
// provider dispatch. A pending, settled or uncertain replay never dispatches a
// second time; exact request identity is required for reconciliation.
func (s *BusinessAccountUsageService) BeginBrowserGatewayReceipt(ctx context.Context, accountID, actor, tier, app, reservationID, runID, sessionID, grantFingerprint, rootOperationID, requestFingerprint string, estimatedCredits int64) (bool, error) {
	accountID, actor = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor))
	reservationID, rootOperationID = strings.TrimSpace(reservationID), strings.TrimSpace(rootOperationID)
	requestFingerprint = strings.TrimSpace(requestFingerprint)
	if s.db == nil || accountID == "" || actor == "" || tier == "" || app == "" || reservationID == "" || runID == "" || sessionID == "" || !validFingerprint(grantFingerprint) || rootOperationID == "" || len(rootOperationID) > 128 || !validFingerprint(requestFingerprint) || estimatedCredits <= 0 {
		return false, errors.New("account, actor, bound browser session, Gateway operation and request fingerprint are required")
	}
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("begin browser operation claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	reservationQuery := `SELECT actor_identity,status,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_tier,app_bundle_key,billing_period FROM business_account_credit_reservations WHERE id=? AND business_account_id=?`
	if s.dialect != "sqlite" {
		reservationQuery = `SELECT actor_identity,status,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_tier,app_bundle_key,billing_period FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2 FOR UPDATE`
	}
	var savedActor, status, savedRun, savedSession, savedGrant, savedTier, savedApp, savedPeriod string
	if err := tx.QueryRowContext(ctx, reservationQuery, reservationID, accountID).Scan(&savedActor, &status, &savedRun, &savedSession, &savedGrant, &savedTier, &savedApp, &savedPeriod); err != nil {
		return false, fmt.Errorf("load browser session reservation: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(savedActor), actor) {
		return false, ErrReservationNotOwned
	}
	if status != string(reservationstate.ReservationPending) || savedRun != runID || savedSession != sessionID || savedGrant != grantFingerprint {
		return false, errors.New("Gateway operation requires a bound pending browser session reservation")
	}
	if savedTier == "" || savedTier != tier || savedApp != app || savedPeriod == "" {
		return false, errors.New("Gateway operation tier and app must match the bound session reservation")
	}
	var previousCount int64
	countQuery := `SELECT COUNT(*) FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND business_account_id=?`
	if s.dialect != "sqlite" {
		countQuery = `SELECT COUNT(*) FROM business_account_browser_gateway_receipts WHERE reservation_id=$1 AND business_account_id=$2`
	}
	if err := tx.QueryRowContext(ctx, countQuery, reservationID, accountID).Scan(&previousCount); err != nil {
		return false, err
	}
	additionalHold := int64(0)
	if previousCount > 0 {
		additionalHold = estimatedCredits
		if err := s.extendBrowserSessionHold(ctx, tx, accountID, tier, app, savedPeriod, reservationID, additionalHold); err != nil {
			return false, err
		}
	}
	insert := `INSERT INTO business_account_browser_gateway_receipts(reservation_id,gateway_root_operation_id,business_account_id,actor_identity,request_fingerprint,reserved_amount,state) VALUES(?,?,?,?,?,?,'dispatching') ON CONFLICT DO NOTHING`
	if s.dialect != "sqlite" {
		insert = `INSERT INTO business_account_browser_gateway_receipts(reservation_id,gateway_root_operation_id,business_account_id,actor_identity,request_fingerprint,reserved_amount,state) VALUES($1,$2,$3,$4,$5,$6,'dispatching') ON CONFLICT DO NOTHING`
	}
	result, err := tx.ExecContext(ctx, insert, reservationID, rootOperationID, accountID, actor, requestFingerprint, additionalHold)
	if err != nil {
		return false, fmt.Errorf("claim Gateway browser operation: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return false, err
	} else if affected == 1 {
		return true, tx.Commit()
	}
	// A concurrent uniqueness winner must not leave an extra estimate on the
	// shared hold. Roll back this transaction's extension and resolve the owner.
	return false, ErrBrowserGatewayOperationReplay
	/*
		var oldReservation, oldAccount, oldActor, oldRequest, oldState string
		lookup := `SELECT reservation_id,business_account_id,actor_identity,request_fingerprint,state FROM business_account_browser_gateway_receipts WHERE gateway_root_operation_id=?`
		if s.dialect != "sqlite" {
			lookup = `SELECT reservation_id,business_account_id,actor_identity,request_fingerprint,state FROM business_account_browser_gateway_receipts WHERE gateway_root_operation_id=$1 FOR UPDATE`
		}
		if err := tx.QueryRowContext(ctx, lookup, rootOperationID).Scan(&oldReservation, &oldAccount, &oldActor, &oldRequest, &oldState); err != nil {
			return false, fmt.Errorf("read Gateway operation replay: %w", err)
		}
		if oldReservation != reservationID || oldAccount != accountID || !strings.EqualFold(oldActor, actor) || oldRequest != requestFingerprint {
			return false, errors.New("Gateway operation was replayed across account, actor, reservation or request")
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, ErrBrowserGatewayOperationReplay
	*/
}

func (s *BusinessAccountUsageService) extendBrowserSessionHold(ctx context.Context, tx *sql.Tx, accountID, tier, app, period, reservationID string, amount int64) error {
	if amount <= 0 {
		return errors.New("browser Gateway operation hold must be positive")
	}
	var reserved int64
	if err := tx.QueryRowContext(ctx, `SELECT reserved_amount FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2 AND status='pending'`, reservationID, accountID).Scan(&reserved); err != nil {
		return err
	}
	var used, pending int64
	usageQuery := `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=? AND billing_period=? AND limit_key='ai_credits'`
	reservationQuery := `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND billing_period=? AND limit_key='ai_credits' AND status='pending'`
	if s.dialect != "sqlite" {
		usageQuery = `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key='ai_credits'`
		reservationQuery = `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND limit_key='ai_credits' AND status='pending'`
	}
	if err := tx.QueryRowContext(ctx, usageQuery, accountID, period).Scan(&used); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, reservationQuery, accountID, period).Scan(&pending); err != nil {
		return err
	}
	scope, err := s.creditScope(ctx, tier, "ai_credits", app)
	if err != nil {
		return err
	}
	if scope.Global == nil {
		return errors.New("business-account AI credit limit is unavailable")
	}
	if amount > math.MaxInt64-used-pending || (scope.Global.LimitValue >= 0 && used+pending+amount > scope.Global.LimitValue) {
		return fmt.Errorf("%w: global credit cap exceeded", s.insufficient)
	}
	if scope.App != nil && scope.App.LimitValue >= 0 {
		var appUsed, appPending int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key='ai_credits' AND app_bundle_key=$3`, accountID, period, app).Scan(&appUsed); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND limit_key='ai_credits' AND app_bundle_key=$3 AND status='pending'`, accountID, period, app).Scan(&appPending); err != nil {
			return err
		}
		if amount > math.MaxInt64-appUsed-appPending || appUsed+appPending+amount > scope.App.LimitValue {
			return fmt.Errorf("%w: app credit cap exceeded", s.insufficient)
		}
	}
	spendable, err := spendableFundingBalance(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return err
	}
	reservedCredits, err := pendingFundingReservations(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return err
	}
	if spendable-reservedCredits < amount {
		return fmt.Errorf("%w: insufficient spendable credits", s.insufficient)
	}
	update := `UPDATE business_account_credit_reservations SET reserved_amount=reserved_amount+?,expires_at=? WHERE id=? AND business_account_id=? AND status='pending'`
	if s.dialect != "sqlite" {
		update = `UPDATE business_account_credit_reservations SET reserved_amount=reserved_amount+$1,expires_at=$2 WHERE id=$3 AND business_account_id=$4 AND status='pending'`
	}
	result, err := tx.ExecContext(ctx, update, amount, s.now().Add(10*time.Minute), reservationID, accountID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return errors.New("browser session hold could not be extended")
	}
	_ = reserved
	return nil
}

// RecordBrowserGatewayReceipt makes a provider accounting child immutable. An
// identical retry is idempotent; a changed payload or operation scope is denied.
func (s *BusinessAccountUsageService) RecordBrowserGatewayReceipt(ctx context.Context, accountID, actor, reservationID string, receipt BrowserGatewayReceipt) error {
	accountID, actor = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor))
	reservationID, receipt.RootOperationID = strings.TrimSpace(reservationID), strings.TrimSpace(receipt.RootOperationID)
	if accountID == "" || actor == "" || reservationID == "" || receipt.RootOperationID == "" || receipt.State != "settled" || receipt.Credits < 0 || receipt.CostMicros < 0 || receipt.PromptTokens < 0 || receipt.CompletionTokens < 0 || !validFingerprint(receipt.RequestFingerprint) || !validFingerprint(receipt.ReceiptFingerprint) {
		return errors.New("complete bounded Gateway settlement receipt is required")
	}
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return fmt.Errorf("begin Gateway receipt attachment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	query := `SELECT state,business_account_id,actor_identity,request_fingerprint,provider,model,credits,cost_micros,prompt_tokens,completion_tokens,receipt_fingerprint FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND gateway_root_operation_id=?`
	if s.dialect != "sqlite" {
		query = `SELECT state,business_account_id,actor_identity,request_fingerprint,provider,model,credits,cost_micros,prompt_tokens,completion_tokens,receipt_fingerprint FROM business_account_browser_gateway_receipts WHERE reservation_id=$1 AND gateway_root_operation_id=$2 FOR UPDATE`
	}
	var saved BrowserGatewayReceipt
	var account, savedActor string
	err = tx.QueryRowContext(ctx, query, reservationID, receipt.RootOperationID).Scan(&saved.State, &account, &savedActor, &saved.RequestFingerprint, &saved.Provider, &saved.Model, &saved.Credits, &saved.CostMicros, &saved.PromptTokens, &saved.CompletionTokens, &saved.ReceiptFingerprint)
	if err != nil {
		return fmt.Errorf("load claimed Gateway operation: %w", err)
	}
	if account != accountID || !strings.EqualFold(savedActor, actor) || saved.RequestFingerprint != receipt.RequestFingerprint {
		return errors.New("Gateway receipt crossed its account, actor or request binding")
	}
	if saved.State == "settled" {
		if saved.Provider == receipt.Provider && saved.Model == receipt.Model && saved.Credits == receipt.Credits && saved.CostMicros == receipt.CostMicros && saved.PromptTokens == receipt.PromptTokens && saved.CompletionTokens == receipt.CompletionTokens && saved.ReceiptFingerprint == receipt.ReceiptFingerprint {
			return tx.Commit()
		}
		return errors.New("Gateway operation receipt replay conflicts with immutable settlement")
	}
	if saved.State != "dispatching" {
		return ErrBrowserSessionOperationsPending
	}
	update := `UPDATE business_account_browser_gateway_receipts SET state='settled',provider=?,model=?,credits=?,cost_micros=?,prompt_tokens=?,completion_tokens=?,receipt_fingerprint=?,settled_at=? WHERE reservation_id=? AND gateway_root_operation_id=? AND state='dispatching'`
	if s.dialect != "sqlite" {
		update = `UPDATE business_account_browser_gateway_receipts SET state='settled',provider=$1,model=$2,credits=$3,cost_micros=$4,prompt_tokens=$5,completion_tokens=$6,receipt_fingerprint=$7,settled_at=NOW() WHERE reservation_id=$8 AND gateway_root_operation_id=$9 AND state='dispatching'`
	}
	args := []any{receipt.Provider, receipt.Model, receipt.Credits, receipt.CostMicros, receipt.PromptTokens, receipt.CompletionTokens, receipt.ReceiptFingerprint, s.now(), reservationID, receipt.RootOperationID}
	if s.dialect != "sqlite" {
		args = args[:7]
		args = append(args, reservationID, receipt.RootOperationID)
	}
	result, err := tx.ExecContext(ctx, update, args...)
	if err != nil {
		return fmt.Errorf("persist Gateway operation receipt: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return errors.New("Gateway receipt state changed concurrently")
	}
	return tx.Commit()
}

func (s *BusinessAccountUsageService) MarkBrowserGatewayOperationUncertain(ctx context.Context, accountID, actor, reservationID, rootOperationID string) error {
	query := `UPDATE business_account_browser_gateway_receipts SET state='uncertain' WHERE reservation_id=? AND gateway_root_operation_id=? AND business_account_id=? AND actor_identity=? AND state='dispatching'`
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_browser_gateway_receipts SET state='uncertain' WHERE reservation_id=$1 AND gateway_root_operation_id=$2 AND business_account_id=$3 AND actor_identity=$4 AND state='dispatching'`
	}
	result, err := s.db.ExecContext(ctx, query, strings.TrimSpace(reservationID), strings.TrimSpace(rootOperationID), strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor)))
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		var state string
		q := `SELECT state FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND gateway_root_operation_id=? AND business_account_id=? AND actor_identity=?`
		if s.dialect != "sqlite" {
			q = `SELECT state FROM business_account_browser_gateway_receipts WHERE reservation_id=$1 AND gateway_root_operation_id=$2 AND business_account_id=$3 AND actor_identity=$4`
		}
		if err := s.db.QueryRowContext(ctx, q, strings.TrimSpace(reservationID), strings.TrimSpace(rootOperationID), strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor))).Scan(&state); err != nil {
			return err
		}
		if state == "uncertain" {
			return nil
		}
		if state == "settled" {
			return errors.New("settled Gateway receipt cannot become uncertain")
		}
		return errors.New("Gateway operation state could not be retained")
	}
	return nil
}

// BrowserGatewayReceiptAggregate returns immutable child totals only when all
// claimed operations have settled. Uncertain or interrupted dispatches retain
// the parent hold and cannot be mistaken for zero usage.
func (s *BusinessAccountUsageService) BrowserGatewayReceiptAggregate(ctx context.Context, accountID, actor, reservationID string) (BrowserGatewayAggregate, error) {
	query := `SELECT state,credits,cost_micros,prompt_tokens,completion_tokens FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND business_account_id=? AND actor_identity=? ORDER BY gateway_root_operation_id`
	if s.dialect != "sqlite" {
		query = `SELECT state,credits,cost_micros,prompt_tokens,completion_tokens FROM business_account_browser_gateway_receipts WHERE reservation_id=$1 AND business_account_id=$2 AND actor_identity=$3 ORDER BY gateway_root_operation_id`
	}
	rows, err := s.db.QueryContext(ctx, query, strings.TrimSpace(reservationID), strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor)))
	if err != nil {
		return BrowserGatewayAggregate{}, err
	}
	defer rows.Close()
	var total BrowserGatewayAggregate
	for rows.Next() {
		var state string
		var credits, cost, prompt, completion int64
		if err := rows.Scan(&state, &credits, &cost, &prompt, &completion); err != nil {
			return BrowserGatewayAggregate{}, err
		}
		if state != "settled" {
			return BrowserGatewayAggregate{}, ErrBrowserSessionOperationsPending
		}
		if credits > int64(^uint64(0)>>1)-total.Credits || cost > int64(^uint64(0)>>1)-total.CostMicros || prompt > int64(^uint64(0)>>1)-total.PromptTokens || completion > int64(^uint64(0)>>1)-total.CompletionTokens {
			return BrowserGatewayAggregate{}, errors.New("browser Gateway receipt aggregate overflows")
		}
		total.Credits += credits
		total.CostMicros += cost
		total.PromptTokens += prompt
		total.CompletionTokens += completion
		total.Operations++
	}
	return total, rows.Err()
}
