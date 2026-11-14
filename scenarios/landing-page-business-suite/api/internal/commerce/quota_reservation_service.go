package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var ErrReservationNotOwned = errors.New("reservation does not belong to authenticated user")

const (
	defaultUsageEventAppKey = "unattributed"
	defaultUsageEventModel  = "unknown"
)

// QuotaReservationService owns count-based capacity. It never grants, reads,
// or spends Vrooli credits. The historical table name is storage identity only.
type QuotaReservationService struct {
	db           UsageStore
	limits       LimitsServicer
	dialect      string
	insufficient error
	now          func() time.Time
}

func NewQuotaReservationService(db UsageStore, limits LimitsServicer, dialect string, insufficient error) *QuotaReservationService {
	if insufficient == nil {
		insufficient = errors.New("insufficient credits")
	}
	return &QuotaReservationService{db: db, limits: limits, dialect: dialect, insufficient: insufficient, now: time.Now}
}

func quotaTransition(state, event string) (string, error) {
	if state == "pending" {
		switch event {
		case "finalize":
			return "finalized", nil
		case "release":
			return "released", nil
		case "expire":
			return "expired", nil
		}
	}
	return "", fmt.Errorf("quota reservation cannot %s from %s", event, state)
}

func (s *QuotaReservationService) Reserve(ctx context.Context, actor, tier, key string, amount int64, window time.Duration) (string, error) {
	actor, key = strings.ToLower(strings.TrimSpace(actor)), strings.ToLower(strings.TrimSpace(key))
	if actor == "" || key == "" || amount <= 0 {
		return "", errors.New("user_identity, limit_key, and positive amount are required")
	}
	limit, err := requireMeterKind(ctx, s.limits, tier, key, "count_based")
	if err != nil {
		return "", err
	}
	if window < 0 || window > 24*time.Hour {
		return "", errors.New("reservation window must be between zero and 24 hours")
	}
	if window == 0 {
		window = 10 * time.Minute
	}
	for attempt := 0; attempt < 4; attempt++ {
		id, err := s.reserveOnce(ctx, actor, key, amount, window, limit.LimitValue)
		if err == nil || !retryableCommerceTransaction(err) || attempt == 3 {
			return id, err
		}
		if err := waitCommerceRetry(ctx, attempt); err != nil {
			return "", err
		}
	}
	return "", errors.New("quota reservation retry budget exhausted")
}

func (s *QuotaReservationService) reserveOnce(ctx context.Context, actor, key string, amount int64, window time.Duration, limit int64) (string, error) {
	if s.db == nil {
		return "", errors.New("quota authority is unavailable")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	period := s.now().Format("2006-01")
	var used, pending int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(usage_amount),0) FROM usage_records WHERE user_identity=$1 AND billing_period=$2 AND limit_key=$3`, actor, period, key).Scan(&used); err != nil {
		return "", err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(reserved_amount),0) FROM credit_reservations WHERE user_identity=$1 AND billing_period=$2 AND limit_key=$3 AND status='pending'`, actor, period, key).Scan(&pending); err != nil {
		return "", err
	}
	if used < 0 || pending < 0 || used > math.MaxInt64-pending || amount > math.MaxInt64-used-pending {
		return "", errors.New("quota amount overflows")
	}
	if limit >= 0 && used+pending+amount > limit {
		return "", fmt.Errorf("%w: quota limit exceeded", s.insufficient)
	}
	id := newReservationID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO credit_reservations(id,user_identity,billing_period,limit_key,reserved_amount,status,expires_at,created_at) VALUES($1,$2,$3,$4,$5,'pending',$6,CURRENT_TIMESTAMP)`, id, actor, period, key, amount, s.now().Add(window)); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *QuotaReservationService) binding(ctx context.Context, id string) (string, string, error) {
	if s.db == nil || strings.TrimSpace(id) == "" {
		return "", "", errors.New("reservation_id is required")
	}
	var actor, key string
	err := s.db.QueryRowContext(ctx, `SELECT user_identity,limit_key FROM credit_reservations WHERE id=$1`, id).Scan(&actor, &key)
	return actor, key, err
}

func (s *QuotaReservationService) verify(ctx context.Context, actor, tier, id string) error {
	owner, key, err := s.binding(ctx, id)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(actor), owner) {
		return ErrReservationNotOwned
	}
	_, err = requireMeterKind(ctx, s.limits, tier, key, "count_based")
	return err
}

func (s *QuotaReservationService) Finalize(ctx context.Context, actor, tier, id string, amount int64) error {
	if amount < 0 {
		return errors.New("actual_amount must be non-negative")
	}
	if err := s.verify(ctx, actor, tier, id); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	query := `SELECT user_identity,billing_period,limit_key,status FROM credit_reservations WHERE id=$1`
	if s.dialect != "sqlite" {
		query += ` FOR UPDATE`
	}
	var owner, period, key, status string
	if err := tx.QueryRowContext(ctx, query, id).Scan(&owner, &period, &key, &status); err != nil {
		return err
	}
	if !strings.EqualFold(owner, actor) {
		return ErrReservationNotOwned
	}
	next, err := quotaTransition(status, "finalize")
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE credit_reservations SET status=$1,finalized_at=CURRENT_TIMESTAMP WHERE id=$2 AND status='pending'`, next, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO usage_records(user_identity,billing_period,limit_key,usage_amount,app_bundle_key,last_operation_at) VALUES($1,$2,$3,$4,'',CURRENT_TIMESTAMP) ON CONFLICT(user_identity,billing_period,limit_key,app_bundle_key) DO UPDATE SET usage_amount=usage_records.usage_amount+excluded.usage_amount,last_operation_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP`, owner, period, key, amount); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *QuotaReservationService) Release(ctx context.Context, actor, tier, id string) error {
	if err := s.verify(ctx, actor, tier, id); err != nil {
		return err
	}
	next, err := quotaTransition("pending", "release")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE credit_reservations SET status=$1 WHERE id=$2 AND user_identity=$3 AND status='pending'`, next, id, strings.ToLower(strings.TrimSpace(actor)))
	return err
}

func (s *QuotaReservationService) Cleanup(ctx context.Context) (int64, error) {
	next, err := quotaTransition("pending", "expire")
	if err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE credit_reservations SET status=$1 WHERE status='pending' AND expires_at<CURRENT_TIMESTAMP AND limit_key IN (SELECT limit_key FROM subscription_tier_limits WHERE limit_type='count_based') AND limit_key NOT IN (SELECT limit_key FROM subscription_tier_limits WHERE limit_type<>'count_based')`, next)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
