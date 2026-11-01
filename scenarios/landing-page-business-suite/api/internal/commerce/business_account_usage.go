package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
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
}

type BusinessAccountUsageOptions struct {
	DB           UsageStore
	Limits       LimitsServicer
	Dialect      string
	Now          func() time.Time
	NewID        func() string
	Insufficient error
	Log          func(string, map[string]interface{})
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
}

func NewBusinessAccountUsageService(opts BusinessAccountUsageOptions) *BusinessAccountUsageService {
	if opts.Dialect == "" {
		opts.Dialect = "postgres"
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.NewID == nil {
		opts.NewID = newReservationID
	}
	if opts.Insufficient == nil {
		opts.Insufficient = errors.New("insufficient credits")
	}
	if opts.Log == nil {
		opts.Log = func(string, map[string]interface{}) {}
	}
	return &BusinessAccountUsageService{db: opts.DB, limits: opts.Limits, dialect: opts.Dialect, now: opts.Now, newID: opts.NewID, insufficient: opts.Insufficient, logf: opts.Log}
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
func (s *BusinessAccountUsageService) ReserveCredits(ctx context.Context, accountID, actor, tier, limitKey string, amount int64) (string, error) {
	accountID, actor, tier, limitKey, err := normalizeAccountUsageInput(accountID, actor, tier, limitKey, amount)
	if err != nil {
		return "", err
	}
	if s.db == nil || s.limits == nil {
		return "", errors.New("business-account billing authority is unavailable")
	}
	period := s.billingPeriod()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", fmt.Errorf("begin business-account reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var used, pending int64
	usageQuery := `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=? AND billing_period=? AND limit_key=?`
	reservationQuery := `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND billing_period=? AND limit_key=? AND status='pending'`
	if s.dialect != "sqlite" {
		usageQuery = `SELECT COALESCE(SUM(usage_amount),0) FROM business_account_usage_records WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3`
		reservationQuery = `SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=$1 AND billing_period=$2 AND limit_key=$3 AND status='pending'`
	}
	if err := tx.QueryRowContext(ctx, usageQuery, accountID, period, limitKey).Scan(&used); err != nil {
		return "", fmt.Errorf("read business-account usage: %w", err)
	}
	if err := tx.QueryRowContext(ctx, reservationQuery, accountID, period, limitKey).Scan(&pending); err != nil {
		return "", fmt.Errorf("read business-account reservations: %w", err)
	}
	limit, err := s.limits.GetLimit(ctx, tier, limitKey, nil)
	if err != nil {
		return "", fmt.Errorf("read business-account tier limit: %w", err)
	}
	if limit == nil {
		return "", fmt.Errorf("business-account tier limit %q is unavailable", limitKey)
	}
	if limit.LimitValue >= 0 && used+pending+amount > limit.LimitValue {
		return "", fmt.Errorf("%w: would use %d, limit is %d, remaining is %d", s.insufficient, used+pending+amount, limit.LimitValue, limit.LimitValue-used-pending)
	}
	id := strings.TrimSpace(s.newID())
	if id == "" {
		return "", errors.New("reservation id generator returned an empty value")
	}
	expiresAt := s.now().Add(10 * time.Minute)
	insert := `INSERT INTO business_account_credit_reservations (id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,expires_at,created_at) VALUES (?,?,?,?,?,?,'pending',?,?)`
	if s.dialect != "sqlite" {
		insert = `INSERT INTO business_account_credit_reservations (id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,expires_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,'pending',$7,NOW())`
	}
	insertArgs := []any{id, accountID, actor, period, limitKey, amount, expiresAt, s.now()}
	if s.dialect != "sqlite" {
		insertArgs = insertArgs[:len(insertArgs)-1]
	}
	if _, err := tx.ExecContext(ctx, insert, insertArgs...); err != nil {
		return "", fmt.Errorf("create business-account reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit business-account reservation: %w", err)
	}
	return id, nil
}

// FinalizeReservationWithCost settles exactly once. A replay with identical
// measured usage succeeds; conflicting settlement data is rejected.
func (s *BusinessAccountUsageService) FinalizeReservationWithCost(ctx context.Context, accountID, reservationID string, actualAmount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	accountID, reservationID = strings.TrimSpace(accountID), strings.TrimSpace(reservationID)
	if s.db == nil || accountID == "" || reservationID == "" || actualAmount < 0 || costMicros < 0 || promptTokens < 0 || completionTokens < 0 {
		return errors.New("business_account_id, reservation_id, and non-negative actual amount are required")
	}
	appBundleKey = strings.ToLower(strings.TrimSpace(appBundleKey))
	if appBundleKey == "" {
		appBundleKey = defaultUsageEventAppKey
	}
	model, provider = firstNonEmpty(model, defaultUsageEventModel), firstNonEmpty(provider, "openrouter")
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin business-account settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var actor, period, limitKey, status, existingApp, existingModel, existingProvider string
	var existing sql.NullInt64
	var existingCost sql.NullInt64
	var existingPrompt, existingCompletion sql.NullInt64
	selectReservation := `SELECT actor_identity,billing_period,limit_key,status,actual_amount,cost_micros,prompt_tokens,completion_tokens,app_bundle_key,model,provider FROM business_account_credit_reservations WHERE id=? AND business_account_id=?`
	if s.dialect != "sqlite" {
		selectReservation = `SELECT actor_identity,billing_period,limit_key,status,actual_amount,cost_micros,prompt_tokens,completion_tokens,app_bundle_key,model,provider FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2 FOR UPDATE`
	}
	if err := tx.QueryRowContext(ctx, selectReservation, reservationID, accountID).Scan(&actor, &period, &limitKey, &status, &existing, &existingCost, &existingPrompt, &existingCompletion, &existingApp, &existingModel, &existingProvider); err != nil {
		return fmt.Errorf("load business-account reservation: %w", err)
	}
	if status == "finalized" {
		if existing.Valid && existing.Int64 == actualAmount && existingCost.Valid && existingCost.Int64 == costMicros && existingPrompt.Valid && existingPrompt.Int64 == int64(promptTokens) && existingCompletion.Valid && existingCompletion.Int64 == int64(completionTokens) && existingApp == appBundleKey && existingModel == model && existingProvider == provider {
			return nil
		}
		return errors.New("business-account reservation was already settled with different usage")
	}
	if status != "pending" {
		return fmt.Errorf("business-account reservation is %s", status)
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
	eventInsert := `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	if s.dialect != "sqlite" {
		eventInsert = `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())`
	}
	eventArgs := []any{reservationID, accountID, actor, appBundleKey, model, actualAmount, costMicros, promptTokens, completionTokens, provider, s.now()}
	if s.dialect != "sqlite" {
		eventArgs = eventArgs[:len(eventArgs)-1]
	}
	if _, err := tx.ExecContext(ctx, eventInsert, eventArgs...); err != nil {
		return fmt.Errorf("write business-account usage event: %w", err)
	}
	update := `UPDATE business_account_credit_reservations SET status='finalized',actual_amount=?,settled_at=?,app_bundle_key=?,model=?,cost_micros=?,prompt_tokens=?,completion_tokens=?,provider=? WHERE id=? AND business_account_id=? AND status='pending'`
	if s.dialect != "sqlite" {
		update = `UPDATE business_account_credit_reservations SET status='finalized',actual_amount=$1,settled_at=NOW(),app_bundle_key=$2,model=$3,cost_micros=$4,prompt_tokens=$5,completion_tokens=$6,provider=$7 WHERE id=$8 AND business_account_id=$9 AND status='pending'`
	}
	updateArgs := []any{actualAmount, s.now(), appBundleKey, model, costMicros, promptTokens, completionTokens, provider, reservationID, accountID}
	if s.dialect != "sqlite" {
		updateArgs = []any{actualAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, reservationID, accountID}
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
	query := `UPDATE business_account_credit_reservations SET status='released' WHERE id=? AND business_account_id=? AND status='pending'`
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_credit_reservations SET status='released' WHERE id=$1 AND business_account_id=$2 AND status='pending'`
	}
	_, err := s.db.ExecContext(ctx, query, reservationID, accountID)
	return err
}

func (s *BusinessAccountUsageService) RetainUncertainReservation(ctx context.Context, reservationID string) error {
	if s.db == nil || strings.TrimSpace(reservationID) == "" {
		return errors.New("reservation_id is required")
	}
	query := `UPDATE business_account_credit_reservations SET expires_at=? WHERE id=? AND status='pending'`
	if s.dialect != "sqlite" {
		query = `UPDATE business_account_credit_reservations SET expires_at=$1 WHERE id=$2 AND status='pending'`
	}
	_, err := s.db.ExecContext(ctx, query, time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), strings.TrimSpace(reservationID))
	return err
}

func (s *BusinessAccountUsageService) RecordUsage(ctx context.Context, accountID, actor, tier, limitKey string, amount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	accountID, actor, tier, limitKey, err := normalizeAccountUsageInput(accountID, actor, tier, limitKey, amount)
	if err != nil {
		return err
	}
	_ = tier // The caller already reserved; direct records settle an observed overage.
	appBundleKey, model, provider = firstNonEmpty(strings.ToLower(strings.TrimSpace(appBundleKey)), defaultUsageEventAppKey), firstNonEmpty(model, defaultUsageEventModel), firstNonEmpty(provider, "openrouter")
	operationID := strings.TrimSpace(s.newID())
	if operationID == "" {
		return errors.New("usage operation id generator returned an empty value")
	}
	period := s.billingPeriod()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin business-account usage record: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	upsert := `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?, ?,?) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=excluded.actor_identity,usage_amount=business_account_usage_records.usage_amount+excluded.usage_amount,last_operation_at=excluded.last_operation_at,updated_at=excluded.updated_at`
	eventInsert := `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	args := []any{accountID, actor, period, limitKey, amount, appBundleKey, s.now(), s.now(), s.now()}
	eventArgs := []any{operationID, accountID, actor, appBundleKey, model, amount, costMicros, promptTokens, completionTokens, provider, s.now()}
	if s.dialect != "sqlite" {
		upsert = `INSERT INTO business_account_usage_records (business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW(),NOW()) ON CONFLICT (business_account_id,billing_period,limit_key,app_bundle_key) DO UPDATE SET actor_identity=EXCLUDED.actor_identity,usage_amount=business_account_usage_records.usage_amount+EXCLUDED.usage_amount,last_operation_at=NOW(),updated_at=NOW()`
		eventInsert = `INSERT INTO business_account_usage_events (operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())`
		args = args[:6]
		eventArgs = eventArgs[:len(eventArgs)-1]
	}
	if _, err := tx.ExecContext(ctx, upsert, args...); err != nil {
		return fmt.Errorf("update business-account usage: %w", err)
	}
	if _, err := tx.ExecContext(ctx, eventInsert, eventArgs...); err != nil {
		return fmt.Errorf("write business-account usage event: %w", err)
	}
	return tx.Commit()
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
	query := `UPDATE business_account_credit_reservations SET status='expired' WHERE status='pending' AND expires_at < ?`
	args := []any{s.now()}
	if s.dialect != "sqlite" {
		query, args = `UPDATE business_account_credit_reservations SET status='expired' WHERE status='pending' AND expires_at < NOW()`, nil
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("expire business-account reservations: %w", err)
	}
	return result.RowsAffected()
}
