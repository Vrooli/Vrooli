package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	reservationflow "landing-page-business-suite-api/internal/commerce/reservation/flow"
	reservationstate "landing-page-business-suite-api/internal/commerce/reservation/flow/generated"
)

// BusinessAccountUsageService keeps the billed audience (the selected
// business account) separate from the human actor who initiated each request.
// It intentionally owns its tables instead of encoding account IDs in
// user_identity, which would make customer usage and audit attribution lie.
type BusinessAccountUsageService struct {
	db           UsageStore
	limits       LimitsServicer
	dialect      string
	now          func() time.Time
	newID        func() string
	insufficient error
	logf         func(string, map[string]interface{})
	liveMode     bool
}

type BusinessAccountUsageOptions struct {
	DB           UsageStore
	Limits       LimitsServicer
	Dialect      string
	Now          func() time.Time
	NewID        func() string
	Insufficient error
	Log          func(string, map[string]interface{})
	LiveMode     *bool
}

// BusinessAccountUsageSummary reports usage for a billing audience without
// presenting the business account ID as a human user identity.
type BusinessAccountUsageSummary struct {
	BusinessAccountID string             `json:"business_account_id"`
	BillingPeriod     string             `json:"billing_period"`
	Tier              string             `json:"tier"`
	Limits            map[string]int64   `json:"limits"`
	Usage             map[string]int64   `json:"usage"`
	Reserved          map[string]int64   `json:"reserved"`
	Remaining         map[string]int64   `json:"remaining"`
	DisplayCredits    map[string]float64 `json:"display_credits"`
	Wallet            *AccountWalletView `json:"wallet,omitempty"`
}

func NewBusinessAccountUsageService(opts BusinessAccountUsageOptions) *BusinessAccountUsageService {
	if opts.Dialect == "" {
		opts.Dialect = "postgres"
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	clock := opts.Now
	opts.Now = func() time.Time { return clock().UTC() }
	if opts.NewID == nil {
		opts.NewID = newReservationID
	}
	if opts.Insufficient == nil {
		opts.Insufficient = errors.New("insufficient credits")
	}
	if opts.Log == nil {
		opts.Log = func(string, map[string]interface{}) {}
	}
	// Unknown configuration must default to live mode: test funding must never
	// become spendable by a production route through a zero-value option.
	liveMode := true
	if opts.LiveMode != nil {
		liveMode = *opts.LiveMode
	}
	return &BusinessAccountUsageService{db: opts.DB, limits: opts.Limits, dialect: opts.Dialect, now: opts.Now, newID: opts.NewID, insufficient: opts.Insufficient, logf: opts.Log, liveMode: liveMode}
}

func (s *BusinessAccountUsageService) billingPeriod() string { return s.now().Format("2006-01") }

func normalizeAccountUsageInput(accountID, actor, tier, key string, amount int64) (string, string, string, string, error) {
	accountID, actor, tier, key = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(actor)), strings.ToLower(strings.TrimSpace(tier)), strings.ToLower(strings.TrimSpace(key))
	switch {
	case accountID == "":
		return "", "", "", "", errors.New("business_account_id is required")
	case actor == "":
		return "", "", "", "", errors.New("actor_identity is required")
	case tier == "":
		return "", "", "", "", errors.New("subscription tier is required")
	case key == "":
		return "", "", "", "", errors.New("limit_key is required")
	case amount <= 0:
		return "", "", "", "", errors.New("amount must be positive")
	default:
		return accountID, actor, tier, key, nil
	}
}

// ReserveCredits atomically reserves against the selected account's own
// monthly usage and retains the authenticated person as actor attribution.
func (s *BusinessAccountUsageService) ReserveCredits(ctx context.Context, account, actor, tier, key string, amount int64) (string, error) {
	id, _, err := s.ReserveCreditsWithID(ctx, account, actor, tier, key, amount, "")
	return id, err
}

// ReserveCreditsWithID returns the immutable original hold on an identical
// replay. Callers that dispatch a provider must use created to avoid dispatching twice.
func (s *BusinessAccountUsageService) ReserveCreditsWithID(ctx context.Context, account, actor, tier, key string, amount int64, operation string) (string, bool, error) {
	return s.ReserveScopedCreditsWithID(ctx, account, actor, tier, key, "", amount, operation)
}

// ReserveScopedCreditsWithID pins the authenticated app with the hold and checks
// global and exact-app caps under the same account serialization lock.
func (s *BusinessAccountUsageService) ReserveScopedCreditsWithID(ctx context.Context, account, actor, tier, key, app string, amount int64, operation string) (string, bool, error) {
	return s.ReserveScopedCreditsWithAdmission(ctx, account, actor, tier, key, app, amount, operation, nil)
}

// ReservationAdmission joins another domain's admission to the wallet transaction.
// Callbacks perform transaction-local SQL only, never commit or roll back, and
// may run again after a transaction conflict. Reject records a terminal refusal
// without retaining capacity when the wallet declines an admitted operation.
type ReservationAdmission struct {
	Reserve func(context.Context, *sql.Tx) error
	Reject  func(context.Context, *sql.Tx) error
}

func (s *BusinessAccountUsageService) ReserveScopedCreditsWithAdmission(ctx context.Context, account, actor, tier, key, app string, amount int64, operation string, admission *ReservationAdmission) (string, bool, error) {
	if admission != nil && (admission.Reserve == nil || admission.Reject == nil) {
		return "", false, errors.New("reservation admission requires reserve and reject callbacks")
	}
	account, actor, tier, key, err := normalizeAccountUsageInput(account, actor, tier, key, amount)
	if err != nil {
		return "", false, err
	}
	if s.db == nil {
		return "", false, errors.New("business-account billing authority is unavailable")
	}
	if _, err := s.creditScope(ctx, tier, key, app); err != nil {
		return "", false, err
	}
	id := strings.TrimSpace(operation)
	if id == "" {
		id = strings.TrimSpace(s.newID())
	}
	if id == "" {
		return "", false, errors.New("reservation id is required")
	}
	for attempt := 0; attempt < 4; attempt++ {
		value, created, err := s.reserveCreditsOnce(ctx, account, actor, tier, key, app, amount, id, admission)
		if err == nil || !retryableCommerceTransaction(err) || attempt == 3 {
			return value, created, err
		}
		if err := waitCommerceRetry(ctx, attempt); err != nil {
			return "", false, err
		}
	}
	return "", false, errors.New("reservation retry budget exhausted")
}

func (s *BusinessAccountUsageService) reserveCreditsOnce(ctx context.Context, accountID, actor, tier, limitKey, app string, amount int64, id string, admission *ReservationAdmission) (string, bool, error) {
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return "", false, fmt.Errorf("begin business-account reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if admission != nil {
		if err := admission.Reserve(ctx, tx); err != nil {
			return "", false, err
		}
	}
	value, created, reservationErr := s.reserveCreditsInTransaction(ctx, tx, accountID, actor, tier, limitKey, app, amount, id)
	if reservationErr != nil {
		if admission == nil {
			return "", false, reservationErr
		}
		if err := admission.Reject(ctx, tx); err != nil {
			return "", false, errors.Join(reservationErr, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", false, fmt.Errorf("commit business-account reservation: %w", err)
	}
	return value, created, reservationErr
}

func (s *BusinessAccountUsageService) reserveCreditsInTransaction(ctx context.Context, tx *sql.Tx, accountID, actor, tier, limitKey, app string, amount int64, id string) (string, bool, error) {
	period := s.billingPeriod()
	var oldAccount, oldActor, oldKey string
	var oldApp sql.NullString
	var oldAmount int64
	var oldMode sql.NullBool
	err := tx.QueryRowContext(ctx, `SELECT business_account_id,actor_identity,limit_key,reserved_amount,app_bundle_key,livemode FROM business_account_credit_reservations WHERE id=$1`, id).Scan(&oldAccount, &oldActor, &oldKey, &oldAmount, &oldApp, &oldMode)
	if err == nil {
		if err := reservationFundingMode(oldMode, s.liveMode); err != nil {
			return "", false, err
		}
		if oldAccount != accountID || oldActor != actor || oldKey != limitKey || oldAmount != amount || oldApp.String != app {
			return "", false, errors.New("reservation operation was replayed with different account, actor, or amount")
		}
		return id, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, fmt.Errorf("read reservation operation: %w", err)
	}

	var used, pending int64
	usageQuery := `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=? AND billing_period=? AND limit_key=?`
	reservationQuery := `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND billing_period=? AND limit_key=? AND status='pending'`
	if s.dialect != "sqlite" {
		usageQuery = `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3`
		reservationQuery = `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3 AND status='pending'`
	}
	if err := tx.QueryRowContext(ctx, usageQuery, accountID, period, limitKey).Scan(&used); err != nil {
		return "", false, fmt.Errorf("read business-account usage: %w", err)
	}
	if err := tx.QueryRowContext(ctx, reservationQuery, accountID, period, limitKey).Scan(&pending); err != nil {
		return "", false, fmt.Errorf("read business-account reservations: %w", err)
	}
	scope, err := s.creditScope(ctx, tier, limitKey, app)
	if err != nil {
		return "", false, fmt.Errorf("read business-account tier limit: %w", err)
	}
	if scope.Global == nil {
		return "", false, fmt.Errorf("business-account tier limit %q is unavailable", limitKey)
	}
	if used < 0 || pending < 0 || used > math.MaxInt64-pending || amount > math.MaxInt64-used-pending {
		return "", false, errors.New("credit reservation amount overflows")
	}
	if scope.Global.LimitValue >= 0 && used+pending+amount > scope.Global.LimitValue {
		return "", false, fmt.Errorf("%w: global credit cap exceeded", s.insufficient)
	}
	if s.dialect != "sqlite" {
		if err := checkFundingAgeCap(ctx, tx, accountID, used+pending+amount, s.liveMode); err != nil {
			return "", false, err
		}
	}
	if scope.App != nil && scope.App.LimitValue >= 0 {
		var appUsed, appPending int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3 AND app_bundle_key=$4`, accountID, period, limitKey, app).Scan(&appUsed); err != nil {
			return "", false, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3 AND app_bundle_key=$4 AND status='pending'`, accountID, period, limitKey, app).Scan(&appPending); err != nil {
			return "", false, err
		}
		if appUsed < 0 || appPending < 0 || appUsed > math.MaxInt64-appPending || amount > math.MaxInt64-appUsed-appPending || appUsed+appPending+amount > scope.App.LimitValue {
			return "", false, fmt.Errorf("%w: app credit cap exceeded", s.insufficient)
		}
	}
	spendable, err := spendableFundingBalance(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return "", false, fmt.Errorf("read spendable funding lots: %w", err)
	}
	reservedCredits, err := pendingFundingReservations(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return "", false, fmt.Errorf("read pending funding reservations: %w", err)
	}
	if spendable-reservedCredits < amount {
		return "", false, fmt.Errorf("%w: spendable credits %d, already reserved %d, requested %d", s.insufficient, spendable, reservedCredits, amount)
	}
	expiresAt := s.now().Add(10 * time.Minute)
	insert := `INSERT INTO business_account_credit_reservations (id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,expires_at,app_bundle_key,livemode,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	// This timestamp fences funding admission, so record statement time after
	// the account lock rather than the transaction's potentially earlier start.
	if s.dialect != "sqlite" {
		insert = `INSERT INTO business_account_credit_reservations (id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,expires_at,app_bundle_key,livemode,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,clock_timestamp())`
	}
	insertArgs := []any{id, accountID, actor, period, limitKey, amount, string(reservationstate.ReservationPending), expiresAt, app, s.liveMode, s.now()}
	if s.dialect != "sqlite" {
		insertArgs = insertArgs[:len(insertArgs)-1]
	}
	if _, err := tx.ExecContext(ctx, insert, insertArgs...); err != nil {
		return "", false, fmt.Errorf("create business-account reservation: %w", err)
	}
	return id, true, nil
}

// FinalizeReservationWithCost settles exactly once. A replay with identical
// measured usage succeeds; conflicting settlement data is rejected.
func (s *BusinessAccountUsageService) FinalizeReservationWithCost(ctx context.Context, accountID, reservationID string, actualAmount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	return s.FinalizeFeatureReservationWithCost(ctx, accountID, reservationID, actualAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, "")
}

// FinalizeFeatureReservationWithCost retains the server-selected feature with
// the immutable settlement. A replay cannot change that attribution.
func (s *BusinessAccountUsageService) FinalizeFeatureReservationWithCost(ctx context.Context, accountID, reservationID string, actualAmount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider, feature string) error {
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		err = s.finalizeReservationOnce(ctx, accountID, reservationID, actualAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, feature)
		if err == nil || !retryableCommerceTransaction(err) || attempt == 3 {
			return err
		}
		if err := waitCommerceRetry(ctx, attempt); err != nil {
			return err
		}
	}
	return errors.New("settlement retry budget exhausted")
}

func (s *BusinessAccountUsageService) finalizeReservationOnce(ctx context.Context, accountID, reservationID string, actualAmount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider, feature string) error {
	accountID, reservationID = strings.TrimSpace(accountID), strings.TrimSpace(reservationID)
	if s.db == nil || accountID == "" || reservationID == "" || actualAmount < 0 || costMicros < 0 || promptTokens < 0 || completionTokens < 0 {
		return errors.New("business_account_id, reservation_id, and non-negative actual amount are required")
	}
	appBundleKey = strings.ToLower(strings.TrimSpace(appBundleKey))
	if appBundleKey == "" {
		appBundleKey = defaultUsageEventAppKey
	}
	model, provider = firstNonEmpty(model, defaultUsageEventModel), firstNonEmpty(provider, "openrouter")
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return fmt.Errorf("begin business-account settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := checkSettlementFeature(ctx, tx, s.dialect, accountID, reservationID, feature); err != nil {
		return err
	}
	var actor, period, limitKey, status, existingApp, existingModel, existingProvider string
	var reservedAmount int64
	var existing sql.NullInt64
	var existingCharged, existingOverrun sql.NullInt64
	var existingCost sql.NullInt64
	var existingPrompt, existingCompletion sql.NullInt64
	var existingMode sql.NullBool
	selectReservation := `SELECT actor_identity,billing_period,limit_key,status,reserved_amount,actual_amount,charged_amount,overrun_amount,cost_micros,prompt_tokens,completion_tokens,app_bundle_key,model,provider,livemode FROM business_account_credit_reservations WHERE id=? AND business_account_id=?`
	if s.dialect != "sqlite" {
		selectReservation = `SELECT actor_identity,billing_period,limit_key,status,reserved_amount,actual_amount,charged_amount,overrun_amount,cost_micros,prompt_tokens,completion_tokens,app_bundle_key,model,provider,livemode FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2 FOR UPDATE`
	}
	if err := tx.QueryRowContext(ctx, selectReservation, reservationID, accountID).Scan(&actor, &period, &limitKey, &status, &reservedAmount, &existing, &existingCharged, &existingOverrun, &existingCost, &existingPrompt, &existingCompletion, &existingApp, &existingModel, &existingProvider, &existingMode); err != nil {
		return fmt.Errorf("load business-account reservation: %w", err)
	}
	if err := reservationFundingMode(existingMode, s.liveMode); err != nil {
		return err
	}
	chargedAmount := actualAmount
	if chargedAmount > reservedAmount {
		chargedAmount = reservedAmount
	}
	overrunAmount := actualAmount - chargedAmount
	terminalStatus := reservationstate.ReservationFinalized
	event := reservationstate.ReservationFinalize
	if overrunAmount > 0 {
		terminalStatus = reservationstate.ReservationOverrun
		event = reservationstate.ReservationFinalizeOverrun
	}
	if status == string(reservationstate.ReservationFinalized) || status == string(reservationstate.ReservationOverrun) {
		if status == string(terminalStatus) && existing.Valid && existing.Int64 == actualAmount && existingCharged.Valid && existingCharged.Int64 == chargedAmount && existingOverrun.Valid && existingOverrun.Int64 == overrunAmount && existingCost.Valid && existingCost.Int64 == costMicros && existingPrompt.Valid && existingPrompt.Int64 == int64(promptTokens) && existingCompletion.Valid && existingCompletion.Int64 == int64(completionTokens) && existingApp == appBundleKey && existingModel == model && existingProvider == provider {
			return nil
		}
		return errors.New("business-account reservation was already settled with different usage")
	}
	nextStatus, err := reservationflow.TransitionReservation(reservationstate.ReservationStatus(status), event)
	if err != nil {
		return fmt.Errorf("transition business-account reservation: %w", err)
	}
	if chargedAmount > 0 {
		if err := allocateFundingLots(ctx, tx, s.dialect, accountID, reservationID, chargedAmount, costMicros, s.liveMode, s.now(), true); err != nil {
			return fmt.Errorf("allocate settled usage across funding lots: %w", err)
		}
	}
	upsert := `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?, ?,?) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=excluded.actor_identity,usage_amount=business_account_usage_records.usage_amount+excluded.usage_amount,last_operation_at=excluded.last_operation_at,updated_at=excluded.updated_at`
	if s.dialect != "sqlite" {
		upsert = `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW(),NOW()) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=EXCLUDED.actor_identity,usage_amount=business_account_usage_records.usage_amount+EXCLUDED.usage_amount,last_operation_at=NOW(),updated_at=NOW()`
	}
	upsertArgs := []any{accountID, actor, period, limitKey, actualAmount, appBundleKey, s.now(), s.now(), s.now()}
	if s.dialect != "sqlite" {
		upsertArgs = []any{accountID, actor, period, limitKey, actualAmount, appBundleKey}
	}
	if _, err := tx.ExecContext(ctx, upsert, upsertArgs...); err != nil {
		return fmt.Errorf("record business-account usage: %w", err)
	}
	if err := s.writeSettlementEvent(ctx, tx, reservationID, businessAccountUsageEvent{accountID: accountID, actor: actor, appKey: appBundleKey, model: model, credits: chargedAmount, costMicros: costMicros, promptTokens: int64(promptTokens), completionTokens: int64(completionTokens), provider: provider}, feature); err != nil {
		return fmt.Errorf("write business-account usage event: %w", err)
	}
	update := `UPDATE business_account_credit_reservations SET status=?,actual_amount=?,charged_amount=?,overrun_amount=?,settled_at=?,app_bundle_key=?,model=?,cost_micros=?,prompt_tokens=?,completion_tokens=?,provider=? WHERE id=? AND business_account_id=? AND status=?`
	if s.dialect != "sqlite" {
		update = `UPDATE business_account_credit_reservations SET status=$1,actual_amount=$2,charged_amount=$3,overrun_amount=$4,settled_at=NOW(),app_bundle_key=$5,model=$6,cost_micros=$7,prompt_tokens=$8,completion_tokens=$9,provider=$10 WHERE id=$11 AND business_account_id=$12 AND status=$13`
	}
	updateArgs := []any{string(nextStatus), actualAmount, chargedAmount, overrunAmount, s.now(), appBundleKey, model, costMicros, promptTokens, completionTokens, provider, reservationID, accountID, string(reservationstate.ReservationPending)}
	if s.dialect != "sqlite" {
		updateArgs = []any{string(nextStatus), actualAmount, chargedAmount, overrunAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, reservationID, accountID, string(reservationstate.ReservationPending)}
	}
	result, err := tx.ExecContext(ctx, update, updateArgs...)
	if err != nil {
		return fmt.Errorf("finalize business-account reservation: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return fmt.Errorf("verify business-account reservation settlement: %w", err)
		}
		return errors.New("business-account reservation was not finalized")
	}
	return tx.Commit()
}

func (s *BusinessAccountUsageService) ReleaseReservation(ctx context.Context, accountID, reservationID string) error {
	accountID, reservationID = strings.TrimSpace(accountID), strings.TrimSpace(reservationID)
	if s.db == nil || accountID == "" || reservationID == "" {
		return errors.New("business_account_id and reservation_id are required")
	}
	if err := s.checkReservationFundingMode(ctx, accountID, reservationID); err != nil {
		return err
	}
	nextStatus, err := reservationflow.TransitionReservation(reservationstate.ReservationPending, reservationstate.ReservationRelease)
	if err != nil {
		return fmt.Errorf("transition released reservation: %w", err)
	}
	query := `UPDATE business_account_credit_reservations SET status=? WHERE id=? AND business_account_id=? AND status=? AND livemode=?`
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_credit_reservations SET status=$1 WHERE id=$2 AND business_account_id=$3 AND status=$4 AND livemode=$5`
	}
	_, err = s.db.ExecContext(ctx, query, string(nextStatus), reservationID, accountID, string(reservationstate.ReservationPending), s.liveMode)
	return err
}

func (s *BusinessAccountUsageService) VerifyActor(ctx context.Context, account, actor, id string) error {
	if strings.TrimSpace(account) == "" || strings.TrimSpace(actor) == "" || strings.TrimSpace(id) == "" {
		return ErrBusinessAccountRequired
	}
	var owner string
	if err := s.db.QueryRowContext(ctx, `SELECT actor_identity FROM business_account_credit_reservations WHERE business_account_id=$1 AND id=$2`, account, id).Scan(&owner); err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(actor), owner) {
		return ErrReservationNotOwned
	}
	return nil
}

func (s *BusinessAccountUsageService) RetainUncertainReservation(ctx context.Context, accountID, reservationID string) error {
	if s.db == nil || strings.TrimSpace(reservationID) == "" {
		return errors.New("reservation_id is required")
	}
	if err := s.checkReservationFundingMode(ctx, accountID, reservationID); err != nil {
		return err
	}
	nextStatus, err := reservationflow.TransitionReservation(reservationstate.ReservationPending, reservationstate.ReservationRetain)
	if err != nil {
		return fmt.Errorf("transition retained reservation: %w", err)
	}
	query := `UPDATE business_account_credit_reservations SET expires_at=? WHERE business_account_id=? AND id=? AND status=? AND livemode=?`
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_credit_reservations SET expires_at=$1 WHERE business_account_id=$2 AND id=$3 AND status=$4 AND livemode=$5`
	}
	_, err = s.db.ExecContext(ctx, query, time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), strings.TrimSpace(accountID), strings.TrimSpace(reservationID), string(nextStatus), s.liveMode)
	return err
}

func (s *BusinessAccountUsageService) RecordUsage(ctx context.Context, accountID, actor, tier, limitKey string, amount int64, operationID, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	if s.db == nil {
		return errors.New("business-account billing authority is unavailable")
	}
	accountID, actor, tier, limitKey, err := normalizeAccountUsageInput(accountID, actor, tier, limitKey, amount)
	if err != nil {
		return err
	}
	limit, err := requireMeterKind(ctx, s.limits, tier, limitKey, "cost_based")
	if err != nil {
		return err
	}
	if costMicros < 0 || promptTokens < 0 || completionTokens < 0 {
		return errors.New("cost and token counts must be non-negative")
	}
	appBundleKey, model, provider = firstNonEmpty(strings.ToLower(strings.TrimSpace(appBundleKey)), defaultUsageEventAppKey), firstNonEmpty(model, defaultUsageEventModel), firstNonEmpty(provider, "openrouter")
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		operationID = strings.TrimSpace(s.newID())
	}
	if operationID == "" {
		return errors.New("usage operation id generator returned an empty value")
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		lastErr = s.recordUsageOnce(ctx, accountID, actor, limitKey, amount, limit.LimitValue, operationID, appBundleKey, model, costMicros, promptTokens, completionTokens, provider)
		if lastErr == nil || (!retryableCommerceTransaction(lastErr) && !retryableUsageOperationConflict(lastErr)) || attempt == 3 {
			return lastErr
		}
		if err := waitCommerceRetry(ctx, attempt); err != nil {
			return err
		}
	}
	return lastErr
}

func (s *BusinessAccountUsageService) recordUsageOnce(ctx context.Context, accountID, actor, limitKey string, amount, limitValue int64, operationID, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	period := s.billingPeriod()
	tx, err := s.beginAccountTransaction(ctx, accountID)
	if err != nil {
		return fmt.Errorf("begin business-account usage record: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	existingEvent, found, err := loadBusinessAccountUsageEvent(ctx, tx, s.dialect, operationID)
	if err != nil {
		return fmt.Errorf("check business-account usage replay: %w", err)
	}
	if found {
		if existingEvent.matches(accountID, actor, appBundleKey, model, amount, costMicros, promptTokens, completionTokens, provider) {
			return nil
		}
		return errors.New("business-account usage operation was replayed with different usage")
	}
	var used, pending int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3`, accountID, period, limitKey).Scan(&used); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND status='pending'`, accountID).Scan(&pending); err != nil {
		return err
	}
	if used < 0 || pending < 0 || used > math.MaxInt64-pending || amount > math.MaxInt64-used-pending {
		return errors.New("credit usage amount overflows")
	}
	if s.dialect != "sqlite" {
		if err := checkFundingAgeCap(ctx, tx, accountID, used+pending+amount, s.liveMode); err != nil {
			return err
		}
	}
	if limitValue >= 0 && used+pending+amount > limitValue {
		return fmt.Errorf("%w: account tier credit limit exceeded", s.insufficient)
	}
	spendable, err := spendableFundingBalance(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return err
	}
	fundingPending, err := pendingFundingReservations(ctx, tx, s.dialect, accountID, s.liveMode, s.now())
	if err != nil {
		return err
	}
	if spendable-fundingPending < amount {
		return fmt.Errorf("%w: existing holds reserve the available funding", s.insufficient)
	}
	upsert := `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?, ?,?) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=excluded.actor_identity,usage_amount=business_account_usage_records.usage_amount+excluded.usage_amount,last_operation_at=excluded.last_operation_at,updated_at=excluded.updated_at`
	eventInsert := `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	args := []any{accountID, actor, period, limitKey, amount, appBundleKey, s.now(), s.now(), s.now()}
	eventArgs := []any{operationID, accountID, actor, appBundleKey, model, amount, costMicros, promptTokens, completionTokens, provider, s.now()}
	if s.dialect != "sqlite" {
		upsert = `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW(),NOW()) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=EXCLUDED.actor_identity,usage_amount=business_account_usage_records.usage_amount+EXCLUDED.usage_amount,last_operation_at=NOW(),updated_at=NOW()`
		eventInsert = `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW()) ON CONFLICT (operation_id) DO NOTHING`
		args = args[:6]
		eventArgs = eventArgs[:len(eventArgs)-1]
	}
	eventResult, err := tx.ExecContext(ctx, eventInsert, eventArgs...)
	if err != nil {
		return fmt.Errorf("write business-account usage event: %w", err)
	}
	inserted, err := eventResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("verify business-account usage claim: %w", err)
	}
	if inserted != 1 {
		existing, found, err := loadBusinessAccountUsageEvent(ctx, tx, s.dialect, operationID)
		if err != nil {
			return fmt.Errorf("load conflicting business-account usage event: %w", err)
		}
		if found && existing.matches(accountID, actor, appBundleKey, model, amount, costMicros, promptTokens, completionTokens, provider) {
			return nil
		}
		return errors.New("business-account usage operation conflicts with an existing event")
	}
	if _, err := tx.ExecContext(ctx, upsert, args...); err != nil {
		return fmt.Errorf("update business-account usage: %w", err)
	}
	if err := allocateFundingLots(ctx, tx, s.dialect, accountID, operationID, amount, costMicros, s.liveMode, s.now(), false); err != nil {
		return fmt.Errorf("allocate direct usage across funding lots: %w", err)
	}
	return tx.Commit()
}

type businessAccountUsageEvent struct {
	accountID, actor, appKey, model, provider           string
	credits, costMicros, promptTokens, completionTokens int64
}

func loadBusinessAccountUsageEvent(ctx context.Context, tx *sql.Tx, dialect, operationID string) (businessAccountUsageEvent, bool, error) {
	query := `SELECT business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider FROM business_account_usage_events WHERE operation_id=?`
	if dialect != "sqlite" {
		query = `SELECT business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider FROM business_account_usage_events WHERE operation_id=$1`
	}
	var event businessAccountUsageEvent
	err := tx.QueryRowContext(ctx, query, operationID).Scan(&event.accountID, &event.actor, &event.appKey, &event.model, &event.credits, &event.costMicros, &event.promptTokens, &event.completionTokens, &event.provider)
	if errors.Is(err, sql.ErrNoRows) {
		return businessAccountUsageEvent{}, false, nil
	}
	if err != nil {
		return businessAccountUsageEvent{}, false, err
	}
	return event, true, nil
}

func (event businessAccountUsageEvent) matches(accountID, actor, appKey, model string, credits, costMicros int64, promptTokens, completionTokens int, provider string) bool {
	return event.accountID == accountID && event.actor == actor && event.appKey == appKey && event.model == model && event.credits == credits && event.costMicros == costMicros && event.promptTokens == int64(promptTokens) && event.completionTokens == int64(completionTokens) && event.provider == provider
}

func retryableUsageOperationConflict(err error) bool {
	return err != nil && (strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") || strings.Contains(strings.ToLower(err.Error()), "duplicate key value"))
}

func (s *BusinessAccountUsageService) AdjustUsage(ctx context.Context, accountID, limitKey string, adjustment int64) error {
	accountID, limitKey = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(limitKey))
	if accountID == "" || limitKey == "" {
		return errors.New("business_account_id and limit_key are required")
	}
	query := `UPDATE business_account_usage_records SET usage_amount=MAX(0,usage_amount+?),last_operation_at=?,updated_at=? WHERE business_account_id=? AND billing_period=? AND limit_key=?`
	args := []any{adjustment, s.now(), s.now(), accountID, s.billingPeriod(), limitKey}
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_usage_records SET usage_amount=GREATEST(0,usage_amount+$1),last_operation_at=NOW(),updated_at=NOW() WHERE business_account_id=$2 AND billing_period=$3 AND limit_key=$4`
		args = []any{adjustment, accountID, s.billingPeriod(), limitKey}
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetUsageSummary returns the current account audience's aggregate usage and
// authoritative tier limits. Membership authorization remains at the HTTP edge.
func (s *BusinessAccountUsageService) GetUsageSummary(ctx context.Context, accountID, tier string) (BusinessAccountUsageSummary, error) {
	accountID, tier = strings.TrimSpace(accountID), strings.ToLower(strings.TrimSpace(tier))
	if s.db == nil || accountID == "" || tier == "" || s.limits == nil {
		return BusinessAccountUsageSummary{}, errors.New("business-account usage authority is unavailable")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT limit_key,SUM(usage_amount) FROM business_account_usage_records WHERE business_account_id=? AND billing_period=? GROUP BY limit_key`, accountID, s.billingPeriod())
	if s.dialect != "sqlite" {
		rows, err = s.db.QueryContext(ctx, `SELECT limit_key,SUM(usage_amount) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 GROUP BY limit_key`, accountID, s.billingPeriod())
	}
	if err != nil {
		return BusinessAccountUsageSummary{}, fmt.Errorf("read business-account usage summary: %w", err)
	}
	defer rows.Close()
	summary := BusinessAccountUsageSummary{BusinessAccountID: accountID, BillingPeriod: s.billingPeriod(), Tier: tier, Limits: map[string]int64{}, Usage: map[string]int64{}, Reserved: map[string]int64{}, Remaining: map[string]int64{}, DisplayCredits: map[string]float64{}}
	for rows.Next() {
		var key string
		var amount int64
		if err := rows.Scan(&key, &amount); err != nil {
			return BusinessAccountUsageSummary{}, err
		}
		summary.Usage[key] = amount
	}
	if err := rows.Err(); err != nil {
		return BusinessAccountUsageSummary{}, err
	}
	reservationQuery := `SELECT limit_key,COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND billing_period=? AND status='pending' GROUP BY limit_key`
	reservationArgs := []any{accountID, s.billingPeriod()}
	if s.dialect != "sqlite" {
		reservationQuery = `SELECT limit_key,COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND status='pending' GROUP BY limit_key`
	}
	reservations, err := s.db.QueryContext(ctx, reservationQuery, reservationArgs...)
	if err != nil {
		return BusinessAccountUsageSummary{}, fmt.Errorf("read business-account pending reservations: %w", err)
	}
	defer reservations.Close()
	for reservations.Next() {
		var key string
		var amount int64
		if err := reservations.Scan(&key, &amount); err != nil {
			return BusinessAccountUsageSummary{}, err
		}
		summary.Reserved[key] = amount
	}
	if err := reservations.Err(); err != nil {
		return BusinessAccountUsageSummary{}, err
	}
	limits, err := s.limits.GetTierLimits(ctx, tier)
	if err != nil {
		return BusinessAccountUsageSummary{}, fmt.Errorf("read business-account tier limits: %w", err)
	}
	for _, limit := range limits {
		if limit.LimitType != "cost_based" {
			continue
		}
		used := summary.Usage[limit.LimitKey]
		reserved := summary.Reserved[limit.LimitKey]
		summary.Limits[limit.LimitKey] = limit.LimitValue
		if limit.LimitValue < 0 {
			summary.Remaining[limit.LimitKey] = -1
		} else {
			summary.Remaining[limit.LimitKey] = limit.LimitValue - used - reserved
		}
		if limit.CostMultiplier > 0 {
			summary.DisplayCredits[limit.LimitKey] = float64(used) / float64(limit.CostMultiplier)
		}
	}
	return summary, nil
}

func (s *BusinessAccountUsageService) CleanupExpiredReservations(ctx context.Context) (int64, error) {
	nextStatus, err := reservationflow.TransitionReservation(reservationstate.ReservationPending, reservationstate.ReservationExpire)
	if err != nil {
		return 0, fmt.Errorf("transition expired reservation: %w", err)
	}
	query := `UPDATE business_account_credit_reservations SET status=? WHERE status=? AND expires_at < ?`
	args := []any{string(nextStatus), string(reservationstate.ReservationPending), s.now()}
	if s.dialect != "sqlite" {
		query, args = `UPDATE business_account_credit_reservations SET status=$1 WHERE status=$2 AND expires_at < NOW()`, args[:2]
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("expire business-account reservations: %w", err)
	}
	return result.RowsAffected()
}

// Account money mutations share one lock before reading balances and holds.
// PostgreSQL's serializable snapshots alone can exhaust retries under a burst
// of independent requests; an account lock makes admission queue predictably.
func (s *BusinessAccountUsageService) beginAccountTransaction(ctx context.Context, accountID string) (*sql.Tx, error) {
	isolation := sql.LevelSerializable
	if s.dialect != "sqlite" {
		isolation = sql.LevelReadCommitted
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: isolation})
	if err != nil {
		return nil, err
	}
	if s.dialect != "sqlite" {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, accountID); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	return tx, nil
}

// AccountCreditScope derives the account tier and its canonical meter conversion.
// Every member spends under the account subscription, never a caller's tier.
func (s *BusinessAccountUsageService) AccountCreditScope(ctx context.Context, account, app string) (string, int64, error) {
	var tier sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT plan_tier FROM subscriptions WHERE business_account_id=$1 AND status IN ('active','trialing') ORDER BY updated_at DESC LIMIT 1`, account).Scan(&tier)
	if errors.Is(err, sql.ErrNoRows) {
		tier = sql.NullString{String: "free", Valid: true}
	} else if err != nil {
		return "", 0, err
	}
	if !tier.Valid || strings.TrimSpace(tier.String) == "" {
		return "", 0, errors.New("account subscription tier unavailable")
	}
	value := strings.TrimSpace(tier.String)
	scope, err := s.creditScope(ctx, value, "ai_credits", app)
	if err != nil {
		return "", 0, err
	}
	return value, scope.CostMultiplier, nil
}

func (s *BusinessAccountUsageService) creditScope(ctx context.Context, tier, key, app string) (MeterLimits, error) {
	scope, err := ResolveMeterLimits(ctx, s.limits, tier, key, app)
	if err != nil {
		return scope, err
	}
	if scope.Kind != "cost_based" || scope.Global == nil {
		return scope, errors.New("global credit meter unavailable")
	}
	if app == "" {
		rows, err := s.limits.GetTierLimits(ctx, tier)
		if err != nil {
			return scope, err
		}
		for _, row := range rows {
			if row.LimitKey == key && row.AppBundleKey != nil {
				return scope, errors.New("verified app billing scope unavailable")
			}
		}
	}
	return scope, nil
}
