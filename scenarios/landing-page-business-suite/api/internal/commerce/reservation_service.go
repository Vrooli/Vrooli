package commerce

import (
	"context"
	cryptoRand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReservationService owns atomic credit reservations, settlement, expiry, and
// adjustments. Its collaborators are explicit so monetization policy remains
// testable without the root API package.
type ReservationService struct {
	db                  UsageStore
	limitsSvc           LimitsServicer
	dialect             string
	insufficientCredits error
	logf                func(string, map[string]interface{})
	now                 func() time.Time
	newID               func() string
}

type ReservationRuntime struct {
	InsufficientCredits error
	Log                 func(string, map[string]interface{})
	Now                 func() time.Time
	NewID               func() string
}

func NewReservationService(db UsageStore, limits LimitsServicer, dialect string, runtime ReservationRuntime) *ReservationService {
	if dialect == "" {
		dialect = "postgres"
	}
	if runtime.Now == nil {
		runtime.Now = time.Now
	}
	if runtime.NewID == nil {
		runtime.NewID = newReservationID
	}
	if runtime.Log == nil {
		runtime.Log = func(string, map[string]interface{}) {}
	}
	if runtime.InsufficientCredits == nil {
		runtime.InsufficientCredits = errors.New("insufficient credits")
	}
	return &ReservationService{db: db, limitsSvc: limits, dialect: dialect, insufficientCredits: runtime.InsufficientCredits, logf: runtime.Log, now: runtime.Now, newID: runtime.NewID}
}

func (s *ReservationService) billingPeriod() string                           { return s.now().Format("2006-01") }
func (s *ReservationService) generateID() string                              { return s.newID() }
func (s *ReservationService) log(event string, fields map[string]interface{}) { s.logf(event, fields) }

func newReservationID() string {
	b := make([]byte, 16)
	if _, err := cryptoRand.Read(b); err != nil {
		return fmt.Sprintf("reservation-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ReserveAndCharge atomically checks the credit limit and records usage in a single transaction.
// This prevents TOCTOU (time-of-check to time-of-use) race conditions where a user could
// exceed their limit by making concurrent requests.
//
// The method uses SELECT FOR UPDATE to lock the user's usage records during the transaction,
// ensuring that concurrent requests are serialized.
func (s *ReservationService) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return fmt.Errorf("limit_key is required")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	billingPeriod := s.billingPeriod()

	// Start a serializable transaction to prevent concurrent limit bypass
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current usage with row locking (FOR UPDATE)
	var currentUsage int64
	var query string
	if s.dialect == "sqlite" {
		// SQLite doesn't support FOR UPDATE, but serializable isolation provides equivalent protection
		query = `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ?
		`
	} else {
		query = `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3
			FOR UPDATE
		`
	}

	err = tx.QueryRowContext(ctx, query, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("get current usage: %w", err)
	}

	// Check the limit if tier is specified
	if tier != "" && s.limitsSvc != nil {
		limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
		if err != nil {
			return fmt.Errorf("get limit: %w", err)
		}

		if limit != nil && limit.LimitValue >= 0 {
			// Check if the new usage would exceed the limit
			if currentUsage+amount > limit.LimitValue {
				return fmt.Errorf("%w: would use %d, limit is %d, remaining is %d",
					s.insufficientCredits, currentUsage+amount, limit.LimitValue, limit.LimitValue-currentUsage)
			}
		}
		// If limit is nil or negative (unlimited), allow the operation
	}

	// Record the usage within the same transaction
	var appKey interface{}
	if metadata.AppBundleKey != "" {
		appKey = strings.TrimSpace(strings.ToLower(metadata.AppBundleKey))
	}

	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, last_operation_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'))
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + excluded.usage_amount,
				last_operation_at = datetime('now'),
				updated_at = datetime('now')
		`
	} else {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, last_operation_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + $4,
				last_operation_at = NOW(),
				updated_at = NOW()
		`
	}

	_, err = tx.ExecContext(ctx, insertQuery, userIdentity, billingPeriod, limitKey, amount, appKey)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.log("usage_reserved_and_charged", map[string]interface{}{
		"level":          "debug",
		"user_identity":  userIdentity,
		"limit_key":      limitKey,
		"amount":         amount,
		"previous_usage": currentUsage,
		"new_usage":      currentUsage + amount,
		"operation":      metadata.Operation,
	})

	return nil
}

// ReserveCredits atomically checks if the user has enough credits and creates a reservation.
// This prevents TOCTOU race conditions in streaming requests where multiple concurrent
// requests could exceed the credit limit.
// Returns the reservation ID on success.
func (s *ReservationService) ReserveCredits(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return "", fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return "", fmt.Errorf("limit_key is required")
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	billingPeriod := s.billingPeriod()

	// Start a serializable transaction for atomic check-and-reserve
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current usage + pending reservations with row locking
	var currentUsage int64
	var pendingReservations int64

	if s.dialect == "sqlite" {
		// Get current usage
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ?
		`, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get current usage: %w", err)
		}

		// Get pending reservations
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(reserved_amount), 0)
			FROM credit_reservations
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ? AND status = 'pending'
		`, userIdentity, billingPeriod, limitKey).Scan(&pendingReservations)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get pending reservations: %w", err)
		}
	} else {
		// Get current usage with FOR UPDATE lock
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3
			FOR UPDATE
		`, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get current usage: %w", err)
		}

		// Get pending reservations with FOR UPDATE lock
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(reserved_amount), 0)
			FROM credit_reservations
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3 AND status = 'pending'
			FOR UPDATE
		`, userIdentity, billingPeriod, limitKey).Scan(&pendingReservations)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get pending reservations: %w", err)
		}
	}

	effectiveUsage := currentUsage + pendingReservations

	// Check the limit if tier is specified
	if tier != "" && s.limitsSvc != nil {
		limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
		if err != nil {
			return "", fmt.Errorf("get limit: %w", err)
		}

		if limit != nil && limit.LimitValue >= 0 {
			// Check if the new usage would exceed the limit
			if effectiveUsage+amount > limit.LimitValue {
				return "", fmt.Errorf("%w: would use %d, limit is %d, remaining is %d",
					s.insufficientCredits, effectiveUsage+amount, limit.LimitValue, limit.LimitValue-effectiveUsage)
			}
		}
	}

	// Create the reservation (expires in 10 minutes)
	reservationID := s.generateID()
	expiresAt := time.Now().Add(10 * time.Minute)

	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, 'pending', ?, datetime('now'))
		`
		_, err = tx.ExecContext(ctx, insertQuery, reservationID, userIdentity, billingPeriod, limitKey, amount, expiresAt)
	} else {
		insertQuery = `
			INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6, NOW())
		`
		_, err = tx.ExecContext(ctx, insertQuery, reservationID, userIdentity, billingPeriod, limitKey, amount, expiresAt)
	}
	if err != nil {
		return "", fmt.Errorf("create reservation: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	s.log("credits_reserved", map[string]interface{}{
		"level":          "debug",
		"user_identity":  userIdentity,
		"limit_key":      limitKey,
		"reserved":       amount,
		"reservation_id": reservationID,
		"current_usage":  currentUsage,
		"pending":        pendingReservations,
	})

	return reservationID, nil
}

// FinalizeReservation marks a reservation as finalized and records the actual usage.
// Call this after a streaming request completes successfully.
func (s *ReservationService) FinalizeReservation(ctx context.Context, reservationID string, actualAmount int64) error {
	if reservationID == "" {
		return fmt.Errorf("reservation_id is required")
	}
	if actualAmount < 0 {
		return fmt.Errorf("actual_amount must be non-negative")
	}

	// Start a transaction to finalize the reservation and record usage atomically
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get reservation details
	var userIdentity, billingPeriod, limitKey, status string
	var reservedAmount int64
	var getQuery string

	if s.dialect == "sqlite" {
		getQuery = `
			SELECT user_identity, billing_period, limit_key, reserved_amount, status
			FROM credit_reservations
			WHERE id = ?
		`
	} else {
		getQuery = `
			SELECT user_identity, billing_period, limit_key, reserved_amount, status
			FROM credit_reservations
			WHERE id = $1
			FOR UPDATE
		`
	}

	err = tx.QueryRowContext(ctx, getQuery, reservationID).Scan(&userIdentity, &billingPeriod, &limitKey, &reservedAmount, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("reservation not found: %s", reservationID)
		}
		return fmt.Errorf("get reservation: %w", err)
	}

	if status != "pending" {
		return fmt.Errorf("reservation already %s", status)
	}

	// Mark reservation as finalized
	var updateQuery string
	if s.dialect == "sqlite" {
		updateQuery = `
			UPDATE credit_reservations
			SET status = 'finalized', finalized_at = datetime('now')
			WHERE id = ?
		`
	} else {
		updateQuery = `
			UPDATE credit_reservations
			SET status = 'finalized', finalized_at = NOW()
			WHERE id = $1
		`
	}
	_, err = tx.ExecContext(ctx, updateQuery, reservationID)
	if err != nil {
		return fmt.Errorf("finalize reservation: %w", err)
	}

	// Record actual usage
	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, last_operation_at)
			VALUES (?, ?, ?, ?, datetime('now'))
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + excluded.usage_amount,
				last_operation_at = datetime('now'),
				updated_at = datetime('now')
		`
	} else {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, last_operation_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + $4,
				last_operation_at = NOW(),
				updated_at = NOW()
		`
	}
	_, err = tx.ExecContext(ctx, insertQuery, userIdentity, billingPeriod, limitKey, actualAmount)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.log("reservation_finalized", map[string]interface{}{
		"level":          "debug",
		"reservation_id": reservationID,
		"user_identity":  userIdentity,
		"reserved":       reservedAmount,
		"actual":         actualAmount,
	})

	return nil
}

// ReleaseReservation marks a reservation as released without recording usage.
// Call this when a streaming request is cancelled or fails before completing.
func (s *ReservationService) ReleaseReservation(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return fmt.Errorf("reservation_id is required")
	}

	var query string
	if s.dialect == "sqlite" {
		query = `
			UPDATE credit_reservations
			SET status = 'released'
			WHERE id = ? AND status = 'pending'
		`
	} else {
		query = `
			UPDATE credit_reservations
			SET status = 'released'
			WHERE id = $1 AND status = 'pending'
		`
	}

	result, err := s.db.ExecContext(ctx, query, reservationID)
	if err != nil {
		return fmt.Errorf("release reservation: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.log("reservation_release_noop", map[string]interface{}{
			"level":          "debug",
			"reservation_id": reservationID,
			"reason":         "already finalized/released/expired or not found",
		})
	} else {
		s.log("reservation_released", map[string]interface{}{
			"level":          "debug",
			"reservation_id": reservationID,
		})
	}

	return nil
}

// CleanupExpiredReservations marks expired pending reservations as expired.
// Returns the number of reservations that were expired.
func (s *ReservationService) CleanupExpiredReservations(ctx context.Context) (int, error) {
	var query string
	if s.dialect == "sqlite" {
		query = `
			UPDATE credit_reservations
			SET status = 'expired'
			WHERE status = 'pending' AND expires_at < datetime('now')
		`
	} else {
		query = `
			UPDATE credit_reservations
			SET status = 'expired'
			WHERE status = 'pending' AND expires_at < NOW()
		`
	}

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired reservations: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		s.log("reservations_expired", map[string]interface{}{
			"level": "info",
			"count": rowsAffected,
		})
	}

	return int(rowsAffected), nil
}

// StartReservationCleanup starts a background goroutine that periodically cleans up
// expired reservations. Returns a cancel function to stop the cleanup goroutine.
func (s *ReservationService) StartReservationCleanup(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_, _ = s.CleanupExpiredReservations(ctx)
				cancel()
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
	}
}

// AdjustUsage adjusts usage for a user, allowing both positive and negative amounts.
// Negative amounts are used for refunds when actual usage was less than estimated.
// Usage will not go below 0 (floor at 0).
func (s *ReservationService) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return fmt.Errorf("limit_key is required")
	}
	if adjustment == 0 {
		return nil // No-op
	}

	billingPeriod := s.billingPeriod()

	var query string
	if s.dialect == "sqlite" {
		// SQLite: use MAX to floor at 0
		query = `
			UPDATE usage_records
			SET usage_amount = MAX(0, usage_amount + ?),
			    last_operation_at = datetime('now'),
			    updated_at = datetime('now')
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ? AND app_bundle_key IS NULL
		`
		_, err := s.db.ExecContext(ctx, query, adjustment, userIdentity, billingPeriod, limitKey)
		if err != nil {
			return fmt.Errorf("adjust usage: %w", err)
		}
	} else {
		// PostgreSQL: use GREATEST to floor at 0
		query = `
			UPDATE usage_records
			SET usage_amount = GREATEST(0, usage_amount + $1),
			    last_operation_at = NOW(),
			    updated_at = NOW()
			WHERE user_identity = $2 AND billing_period = $3 AND limit_key = $4 AND app_bundle_key IS NULL
		`
		_, err := s.db.ExecContext(ctx, query, adjustment, userIdentity, billingPeriod, limitKey)
		if err != nil {
			return fmt.Errorf("adjust usage: %w", err)
		}
	}

	s.log("usage_adjusted", map[string]interface{}{
		"level":         "debug",
		"user_identity": userIdentity,
		"limit_key":     limitKey,
		"adjustment":    adjustment,
		"reason":        reason,
	})

	return nil
}
