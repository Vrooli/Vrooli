package commerce

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type couponUsageStoreFake struct {
	rows CouponUsageRows
	err  error
}

func (s couponUsageStoreFake) queryCouponUsage(context.Context) (CouponUsageRows, error) {
	return s.rows, s.err
}

type couponUsageRowsFake struct {
	entries  []couponUsageRow
	index    int
	scanErr  error
	closeErr error
	err      error
}

type couponUsageRow struct {
	couponID      string
	totalUses     int64
	lastUsedAt    time.Time
	hasLastUsedAt bool
}

func (r *couponUsageRowsFake) Close() error { return r.closeErr }
func (r *couponUsageRowsFake) Next() bool   { r.index++; return r.index <= len(r.entries) }
func (r *couponUsageRowsFake) Err() error   { return r.err }
func (r *couponUsageRowsFake) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	entry := r.entries[r.index-1]
	*(dest[0].(*string)) = entry.couponID
	*(dest[1].(*int64)) = entry.totalUses
	nullTime := dest[2].(*sql.NullTime)
	nullTime.Time, nullTime.Valid = entry.lastUsedAt, entry.hasLastUsedAt
	return nil
}

func TestListCouponUsage_MapsNullableUsageHistory(t *testing.T) {
	lastUsed := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	usage, err := listCouponUsage(context.Background(), couponUsageStoreFake{rows: &couponUsageRowsFake{entries: []couponUsageRow{{couponID: "coupon-active", totalUses: 3, lastUsedAt: lastUsed, hasLastUsedAt: true}, {couponID: "coupon-never", totalUses: 0}}}})
	if err != nil {
		t.Fatalf("listCouponUsage() error = %v", err)
	}
	if len(usage) != 2 || usage[0].CouponID != "coupon-active" || usage[0].LastUsedAt == nil {
		t.Fatalf("usage = %#v", usage)
	}
	if got, want := usage[0].LastUsedAt.Location(), time.UTC; got != want {
		t.Fatalf("LastUsedAt location = %s, want UTC", got)
	}
	if usage[1].LastUsedAt != nil {
		t.Fatalf("nullable last use = %v, want nil", usage[1].LastUsedAt)
	}
}

func TestListCouponUsage_ReportsStoreAndCursorFailures(t *testing.T) {
	queryErr := errors.New("query failed")
	if _, err := listCouponUsage(context.Background(), couponUsageStoreFake{err: queryErr}); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v", err)
	}
	scanErr := errors.New("scan failed")
	if _, err := listCouponUsage(context.Background(), couponUsageStoreFake{rows: &couponUsageRowsFake{entries: []couponUsageRow{{}}, scanErr: scanErr}}); !errors.Is(err, scanErr) {
		t.Fatalf("scan error = %v", err)
	}
	cursorErr := errors.New("cursor failed")
	if _, err := listCouponUsage(context.Background(), couponUsageStoreFake{rows: &couponUsageRowsFake{err: cursorErr}}); !errors.Is(err, cursorErr) {
		t.Fatalf("cursor error = %v", err)
	}
}
