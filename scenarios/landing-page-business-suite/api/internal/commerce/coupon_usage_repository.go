package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CouponUsageStore is the narrow persistence seam for introductory coupon
// redemption statistics. Transport packages depend on this domain contract,
// not on database/sql directly.
type CouponUsageStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// CouponUsageRows is the smallest cursor contract needed by this repository.
// Keeping it interface-based makes scan and cursor-error paths testable without
// a database driver or testcontainer.
type CouponUsageRows interface {
	Close() error
	Next() bool
	Scan(...any) error
	Err() error
}

type couponUsageRepositoryStore interface {
	queryCouponUsage(context.Context) (CouponUsageRows, error)
}

type sqlCouponUsageStore struct{ db CouponUsageStore }

func (s sqlCouponUsageStore) queryCouponUsage(ctx context.Context) (CouponUsageRows, error) {
	return s.db.QueryContext(ctx, listCouponUsageQuery)
}

// CouponUsageSummary is the domain representation of one coupon's redemption
// history.
type CouponUsageSummary struct {
	CouponID   string
	TotalUses  int64
	LastUsedAt *time.Time
}

const listCouponUsageQuery = `SELECT coupon_id, COUNT(*) AS total_uses, MAX(created_at) AS last_used_at FROM intro_coupon_usage GROUP BY coupon_id ORDER BY total_uses DESC`

// ListCouponUsage returns coupon redemption history ordered by use count.
func ListCouponUsage(ctx context.Context, db CouponUsageStore) ([]CouponUsageSummary, error) {
	return listCouponUsage(ctx, sqlCouponUsageStore{db: db})
}

func listCouponUsage(ctx context.Context, store couponUsageRepositoryStore) ([]CouponUsageSummary, error) {
	rows, err := store.queryCouponUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("query coupon usage: %w", err)
	}
	defer rows.Close()

	usage := make([]CouponUsageSummary, 0)
	for rows.Next() {
		var summary CouponUsageSummary
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&summary.CouponID, &summary.TotalUses, &lastUsedAt); err != nil {
			return nil, fmt.Errorf("scan coupon usage: %w", err)
		}
		if lastUsedAt.Valid {
			value := lastUsedAt.Time.UTC()
			summary.LastUsedAt = &value
		}
		usage = append(usage, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read coupon usage: %w", err)
	}
	return usage, nil
}
