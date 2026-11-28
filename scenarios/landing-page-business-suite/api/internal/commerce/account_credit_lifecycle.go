package commerce

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var ErrCreditLifecycleState = errors.New("account-credit lifecycle transition refused")
var ErrInvalidCreditSettlement = errors.New("invalid measured account-credit settlement")
var ErrInvalidCreditLease = errors.New("invalid account-credit lease")

// AccountCreditLifecycleService adds a dispatch fence and bounded lease to the
// account wallet's existing durable reservation and settlement owner.
type AccountCreditLifecycleService struct {
	usage   *UsageService
	dialect string
	now     func() time.Time
}

type AccountCreditLifecycle struct {
	ReservationID     string    `json:"reservation_id"`
	Phase             string    `json:"phase"`
	LeaseExpires      time.Time `json:"lease_expires_at"`
	WalletStatus      string    `json:"wallet_status"`
	Reserved          int64     `json:"reserved_amount"`
	CostMultiplier    *int64    `json:"cost_multiplier,omitempty"`
	USDBudgetMicros   *int64    `json:"usd_budget_micros,omitempty"`
	Actual            *int64    `json:"actual_amount,omitempty"`
	ProviderReceiptID string    `json:"provider_receipt_id,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ReserveUSDBudgetV1 reserves a USD-denominated ceiling using the authoritative
// tier/app policy. The snapshot is immutable for the life of this operation.
func (s *AccountCreditLifecycleService) ReserveUSDBudgetV1(ctx context.Context, account, actor, tier, app, operationID string, budgetMicros int64) (AccountCreditLifecycle, bool, error) {
	if s == nil || s.usage == nil || budgetMicros <= 0 || strings.TrimSpace(app) == "" {
		return AccountCreditLifecycle{}, false, errors.New("positive USD budget and lifecycle scope are required")
	}
	var version, storedBudget, storedMultiplier int64
	var err error
	lookupErr := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT contract_version,usd_budget_micros,cost_multiplier FROM business_account_credit_lifecycle WHERE reservation_id=$1`), operationID).Scan(&version, &storedBudget, &storedMultiplier)
	var multiplier int64
	if lookupErr == nil {
		multiplier, err = s.waitUSDContractSnapshot(ctx, operationID, budgetMicros)
		if err != nil {
			return AccountCreditLifecycle{}, false, err
		}
	} else if errors.Is(lookupErr, sql.ErrNoRows) {
		limit, err := s.usage.limitsSvc.GetLimit(ctx, tier, "ai_credits", &app)
		if err != nil || limit == nil || limit.LimitType != "cost_based" || limit.CostMultiplier <= 0 {
			return AccountCreditLifecycle{}, false, errors.New("current tier/app cost conversion policy is unavailable")
		}
		multiplier = limit.CostMultiplier
	} else {
		return AccountCreditLifecycle{}, false, lookupErr
	}
	credits, err := costMicrosToCredits(budgetMicros, multiplier)
	if err != nil {
		return AccountCreditLifecycle{}, false, err
	}
	state, created, reserveErr := s.Reserve(ctx, account, actor, tier, app, operationID, credits)
	if reserveErr != nil {
		// A competing request may have won the wallet operation with another
		// live policy. The wallet's replay conflict is recoverable only when the
		// lifecycle owner has committed its authoritative v1 snapshot.
		if durableMultiplier, snapshotErr := s.waitUSDContractSnapshot(ctx, operationID, budgetMicros); snapshotErr == nil {
			state, snapshotErr = s.Recover(ctx, actor, app, operationID)
			if snapshotErr == nil {
				state.CostMultiplier, state.USDBudgetMicros = int64Ptr(durableMultiplier), int64Ptr(budgetMicros)
				return state, false, nil
			}
		}
		return state, created, reserveErr
	}
	if created {
		result, err := s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET contract_version=1,usd_budget_micros=$1,cost_multiplier=$2 WHERE reservation_id=$3 AND phase='reserved' AND contract_version=0`), budgetMicros, multiplier, operationID)
		ok, updateErr := oneRow(result, err)
		if updateErr != nil {
			return AccountCreditLifecycle{}, false, updateErr
		}
		if !ok {
			return AccountCreditLifecycle{}, false, ErrCreditLifecycleState
		}
	}
	durableMultiplier, err := s.waitUSDContractSnapshot(ctx, operationID, budgetMicros)
	if err != nil {
		return AccountCreditLifecycle{}, false, err
	}
	if !created {
		state, err = s.Recover(ctx, actor, app, operationID)
		if err != nil {
			return AccountCreditLifecycle{}, false, err
		}
	}
	state.CostMultiplier, state.USDBudgetMicros = int64Ptr(durableMultiplier), int64Ptr(budgetMicros)
	return state, created, nil
}

// waitUSDContractSnapshot fences concurrent same-operation requests. A caller
// that did not create the wallet hold may return only after the winner's v1
// multiplier and USD budget are durable.
func (s *AccountCreditLifecycleService) waitUSDContractSnapshot(ctx context.Context, operationID string, budgetMicros int64) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	deadline := time.NewTimer(750 * time.Millisecond)
	defer deadline.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var version, storedBudget, multiplier int64
		err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT contract_version,usd_budget_micros,cost_multiplier FROM business_account_credit_lifecycle WHERE reservation_id=$1`), operationID).Scan(&version, &storedBudget, &multiplier)
		if err == nil && version == 1 {
			if storedBudget != budgetMicros || multiplier <= 0 {
				return 0, ErrCreditLifecycleState
			}
			return multiplier, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-deadline.C:
			return 0, fmt.Errorf("%w: USD contract snapshot was not committed", ErrCreditLifecycleState)
		case <-ticker.C:
		}
	}
}

// SettleMeasuredCostV1 derives credits from measured provider cost and the
// multiplier persisted at reservation time; caller supplied credits are absent.
func (s *AccountCreditLifecycleService) SettleMeasuredCostV1(ctx context.Context, actor, app, id string, input AccountCreditSettlement) error {
	var version, multiplier int64
	if err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT contract_version,cost_multiplier FROM business_account_credit_lifecycle WHERE reservation_id=$1 AND actor_identity=$2 AND app_bundle_key=$3`), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app)).Scan(&version, &multiplier); err != nil {
		return err
	}
	if version != 1 || multiplier <= 0 {
		return ErrCreditLifecycleState
	}
	credits, err := costMicrosToCredits(input.CostMicros, multiplier)
	if err != nil {
		return err
	}
	input.ActualAmount = credits
	return s.Settle(ctx, actor, app, id, input)
}

func costMicrosToCredits(costMicros, multiplier int64) (int64, error) {
	if costMicros <= 0 || multiplier <= 0 {
		return 0, errors.New("cost and conversion multiplier must be positive")
	}
	if costMicros > (math.MaxInt64-9999)/multiplier {
		return 0, errors.New("USD cost conversion overflows")
	}
	credits := (costMicros*multiplier + 9999) / 10000
	if credits <= 0 {
		return 0, errors.New("USD cost conversion is invalid")
	}
	return credits, nil
}

func int64Ptr(value int64) *int64 { return &value }

type AccountCreditSettlement struct {
	ActualAmount      int64  `json:"actual_amount"`
	CostMicros        int64  `json:"cost_micros"`
	ActiveSeconds     *int64 `json:"active_seconds"`
	PausedSeconds     *int64 `json:"paused_seconds"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	Model             string `json:"model"`
	Provider          string `json:"provider"`
	ProviderReceiptID string `json:"provider_receipt_id"`
}

func NewAccountCreditLifecycleService(usage *UsageService) *AccountCreditLifecycleService {
	dialect := "postgres"
	if usage != nil && usage.dialect != "" {
		dialect = usage.dialect
	}
	return &AccountCreditLifecycleService{usage: usage, dialect: dialect, now: time.Now}
}

func (s *AccountCreditLifecycleService) bind(query string) string {
	if s.dialect == "sqlite" {
		return strings.ReplaceAll(query, "$", "?")
	}
	return query
}

func (s *AccountCreditLifecycleService) Reserve(ctx context.Context, account, actor, tier, app, operationID string, amount int64) (AccountCreditLifecycle, bool, error) {
	if s == nil || s.usage == nil || strings.TrimSpace(account) == "" || strings.TrimSpace(actor) == "" || strings.TrimSpace(app) == "" || strings.TrimSpace(operationID) == "" || amount <= 0 {
		return AccountCreditLifecycle{}, false, errors.New("account-credit reservation inputs are incomplete")
	}
	id, created, err := s.usage.ReserveBusinessAccountInferenceCreditsWithID(ctx, account, actor, tier, app, amount, operationID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "replayed with different") {
			return AccountCreditLifecycle{}, false, fmt.Errorf("%w: operation ID scope or amount differs", ErrCreditLifecycleState)
		}
		return AccountCreditLifecycle{}, false, err
	}
	if id != operationID {
		return AccountCreditLifecycle{}, false, errors.New("LPBS returned a different reservation operation id")
	}
	lease := s.now().UTC().Add(10 * time.Minute)
	query := `INSERT INTO business_account_credit_lifecycle(reservation_id,business_account_id,actor_identity,app_bundle_key,phase,lease_expires_at,updated_at) VALUES($1,$2,$3,$4,'reserved',$5,$6) ON CONFLICT(reservation_id) DO NOTHING`
	if _, err := s.usage.db.ExecContext(ctx, s.bind(query), id, account, actor, app, lease, s.now().UTC()); err != nil {
		// The wallet hold remains pending and expires through its ordinary owner
		// cleanup; no response is returned, so the caller cannot dispatch it.
		return AccountCreditLifecycle{}, false, fmt.Errorf("record account-credit lifecycle: %w", err)
	}
	var storedAccount, storedActor, storedApp string
	if err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT business_account_id,actor_identity,app_bundle_key FROM business_account_credit_lifecycle WHERE reservation_id=$1`), id).Scan(&storedAccount, &storedActor, &storedApp); err != nil {
		return AccountCreditLifecycle{}, false, err
	}
	if storedAccount != account || !strings.EqualFold(storedActor, actor) || storedApp != app {
		return AccountCreditLifecycle{}, false, fmt.Errorf("%w: operation ID has different authenticated scope", ErrCreditLifecycleState)
	}
	state, err := s.Recover(ctx, actor, app, id)
	return state, created, err
}

// BeginDispatch is the release proof: the provider may be contacted only after
// this one-time owner transition succeeds. Release and dispatch race through
// one conditional update, so only one can win.
func (s *AccountCreditLifecycleService) BeginDispatch(ctx context.Context, actor, app, id string) (bool, error) {
	tx, err := s.usage.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC()
	result, err := tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='dispatched',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase='reserved' AND lease_expires_at>$1`), now, id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
	started, err := oneRow(result, err)
	if err != nil || !started {
		return started, err
	}
	result, err = tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_reservations SET expires_at=$1 WHERE id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND status='pending' AND livemode=$5`), time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app), s.usage.liveMode)
	updated, err := oneRow(result, err)
	if err != nil {
		return false, err
	}
	if !updated {
		return false, ErrCreditLifecycleState
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *AccountCreditLifecycleService) Heartbeat(ctx context.Context, actor, app, id string, lease time.Duration) (time.Time, error) {
	if lease < 30*time.Second || lease > 5*time.Minute {
		return time.Time{}, fmt.Errorf("%w: lease must be between 30 seconds and 5 minutes", ErrInvalidCreditLease)
	}
	until := s.now().UTC().Add(lease)
	tx, err := s.usage.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET lease_expires_at=$1,updated_at=$2 WHERE reservation_id=$3 AND actor_identity=$4 AND app_bundle_key=$5 AND phase='reserved'`), until, s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
	if err != nil {
		return time.Time{}, err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return time.Time{}, err
		}
		return time.Time{}, ErrCreditLifecycleState
	}
	result, err = tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_reservations SET expires_at=$1 WHERE id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND status='pending' AND livemode=$5`), until, id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app), s.usage.liveMode)
	if err != nil {
		return time.Time{}, err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return time.Time{}, err
		}
		return time.Time{}, ErrCreditLifecycleState
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, err
	}
	return until, nil
}

func (s *AccountCreditLifecycleService) Release(ctx context.Context, actor, app, id string) error {
	tx, err := s.usage.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='released',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase='reserved'`), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return err
		}
		return ErrCreditLifecycleState
	}
	var account string
	if err := tx.QueryRowContext(ctx, s.bind(`SELECT business_account_id FROM business_account_credit_lifecycle WHERE reservation_id=$1`), id).Scan(&account); err != nil {
		return err
	}
	result, err = tx.ExecContext(ctx, s.bind(`UPDATE business_account_credit_reservations SET status='released' WHERE id=$1 AND business_account_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND status='pending' AND livemode=$5`), id, account, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app), s.usage.liveMode)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return err
		}
		return ErrCreditLifecycleState
	}
	return tx.Commit()
}

func settlementFingerprint(input AccountCreditSettlement) (string, error) {
	input.Model, input.Provider = strings.TrimSpace(input.Model), strings.TrimSpace(input.Provider)
	input.ProviderReceiptID = strings.TrimSpace(input.ProviderReceiptID)
	if input.ActualAmount < 0 || input.CostMicros < 0 || input.ActiveSeconds == nil || input.PausedSeconds == nil || *input.ActiveSeconds < 0 || *input.PausedSeconds < 0 || input.PromptTokens < 0 || input.CompletionTokens < 0 || input.Model == "" || input.Provider == "" || input.ProviderReceiptID == "" {
		return "", fmt.Errorf("%w: measured non-negative usage, model, provider, and provider receipt ID are required", ErrInvalidCreditSettlement)
	}
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (s *AccountCreditLifecycleService) Settle(ctx context.Context, actor, app, id string, input AccountCreditSettlement) error {
	fingerprint, err := settlementFingerprint(input)
	if err != nil {
		return err
	}
	var oldPhase, oldFingerprint string
	if err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT phase,settlement_fingerprint FROM business_account_credit_lifecycle WHERE reservation_id=$1 AND actor_identity=$2 AND app_bundle_key=$3`), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app)).Scan(&oldPhase, &oldFingerprint); err != nil {
		return err
	}
	if oldPhase == "settled" && oldFingerprint == fingerprint {
		return nil
	}
	result, err := s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='settling',settlement_fingerprint=CASE WHEN settlement_fingerprint='' THEN $1 ELSE settlement_fingerprint END,provider_receipt_id=CASE WHEN provider_receipt_id='' THEN $2 ELSE provider_receipt_id END,updated_at=$3 WHERE reservation_id=$4 AND actor_identity=$5 AND app_bundle_key=$6 AND ((phase IN ('dispatched','uncertain') AND settlement_fingerprint='') OR (phase='settling' AND settlement_fingerprint=$1))`), fingerprint, strings.TrimSpace(input.ProviderReceiptID), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
	ok, err := oneRow(result, err)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCreditLifecycleState
	}
	var account string
	if err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT business_account_id FROM business_account_credit_lifecycle WHERE reservation_id=$1`), id).Scan(&account); err != nil {
		return err
	}
	if err := s.usage.FinalizeBusinessAccountReservationWithCost(ctx, account, actor, id, input.ActualAmount, app, input.Model, input.CostMicros, input.PromptTokens, input.CompletionTokens, input.Provider); err != nil {
		return err
	}
	result, err = s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='settled',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase='settling' AND settlement_fingerprint=$5`), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app), fingerprint)
	ok, err = oneRow(result, err)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCreditLifecycleState
	}
	return nil
}

func (s *AccountCreditLifecycleService) RetainUncertain(ctx context.Context, actor, app, id string) error {
	var account, phase string
	if err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT business_account_id,phase FROM business_account_credit_lifecycle WHERE reservation_id=$1 AND actor_identity=$2 AND app_bundle_key=$3`), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app)).Scan(&account, &phase); err != nil {
		return err
	}
	if phase != "dispatched" && phase != "settling" && phase != "uncertain" {
		return ErrCreditLifecycleState
	}
	if phase != "uncertain" {
		if err := s.usage.RetainUncertainBusinessAccountReservation(ctx, account, actor, id); err != nil {
			return err
		}
	}
	result, err := s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='uncertain',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase IN ('dispatched','settling','uncertain')`), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
	ok, err := oneRow(result, err)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCreditLifecycleState
	}
	return nil
}

func (s *AccountCreditLifecycleService) Recover(ctx context.Context, actor, app, id string) (AccountCreditLifecycle, error) {
	var result AccountCreditLifecycle
	err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT l.reservation_id,l.phase,l.lease_expires_at,r.status,r.reserved_amount,r.actual_amount,l.provider_receipt_id,l.updated_at FROM business_account_credit_lifecycle l JOIN business_account_credit_reservations r ON r.id=l.reservation_id AND r.business_account_id=l.business_account_id WHERE l.reservation_id=$1 AND l.actor_identity=$2 AND l.app_bundle_key=$3`), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app)).Scan(&result.ReservationID, &result.Phase, &result.LeaseExpires, &result.WalletStatus, &result.Reserved, &result.Actual, &result.ProviderReceiptID, &result.UpdatedAt)
	if err != nil {
		return AccountCreditLifecycle{}, err
	}
	// Resolve crashes between the wallet transaction and the lifecycle receipt
	// from durable owner state. A reserved lease can be released only because
	// dispatch was never authorized by the state machine.
	switch {
	case result.Phase == "reserved" && !result.LeaseExpires.After(s.now()):
		if result.WalletStatus == "pending" {
			if err := s.Release(ctx, actor, app, id); err != nil && !errors.Is(err, ErrCreditLifecycleState) {
				return AccountCreditLifecycle{}, err
			}
		} else if result.WalletStatus == "expired" {
			_, err := s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='expired',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase='reserved'`), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
			if err != nil {
				return AccountCreditLifecycle{}, err
			}
		}
	case (result.Phase == "settling" || result.Phase == "uncertain") && (result.WalletStatus == "finalized" || result.WalletStatus == "overrun"):
		_, err := s.usage.db.ExecContext(ctx, s.bind(`UPDATE business_account_credit_lifecycle SET phase='settled',updated_at=$1 WHERE reservation_id=$2 AND actor_identity=$3 AND app_bundle_key=$4 AND phase IN ('settling','uncertain')`), s.now().UTC(), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app))
		if err != nil {
			return AccountCreditLifecycle{}, err
		}
	}
	if result.Phase == "reserved" && !result.LeaseExpires.After(s.now()) || ((result.Phase == "settling" || result.Phase == "uncertain") && (result.WalletStatus == "finalized" || result.WalletStatus == "overrun")) {
		return s.Recover(ctx, actor, app, id)
	}
	return result, nil
}

// ReservationAccount returns the account from owner storage only after the
// immutable authenticated actor and service app both match the reservation.
func (s *AccountCreditLifecycleService) ReservationAccount(ctx context.Context, actor, app, id string) (string, error) {
	var account string
	err := s.usage.db.QueryRowContext(ctx, s.bind(`SELECT business_account_id FROM business_account_credit_lifecycle WHERE reservation_id=$1 AND actor_identity=$2 AND app_bundle_key=$3`), id, strings.ToLower(strings.TrimSpace(actor)), strings.TrimSpace(app)).Scan(&account)
	return account, err
}

func oneRow(result sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, fmt.Errorf("account-credit state update returned no result")
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}
