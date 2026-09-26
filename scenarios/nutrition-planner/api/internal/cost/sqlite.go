package cost

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/money"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, v Observation) (Observation, error) {
	if err := ValidateObservation(v); err != nil {
		return Observation{}, err
	}
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	validThrough := ""
	if !v.ValidThrough.IsZero() {
		validThrough = v.ValidThrough.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO price_observations(id,workspace_id,item_id,product_id,package_label,package_amount,package_unit,price_minor,currency,currency_exponent,retailer,observed_at,valid_through,available,membership_required,coupon_required,minimum_buy,source) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.WorkspaceID, v.ItemID, v.ProductID, v.PackageLabel, v.PackageAmount.String(), v.PackageUnit, v.Price.Minor, v.Price.Currency, v.Price.Exponent, v.Retailer, v.ObservedAt.UTC().Format(time.RFC3339Nano), validThrough, boolInt(v.Available), boolInt(v.MembershipRequired), boolInt(v.CouponRequired), v.MinimumBuy, v.Source)
	if err != nil {
		return Observation{}, fmt.Errorf("insert price observation: %w", err)
	}
	return v, nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID, itemID string) ([]Observation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,item_id,product_id,package_label,package_amount,package_unit,price_minor,currency,currency_exponent,retailer,observed_at,valid_through,available,membership_required,coupon_required,minimum_buy,source FROM price_observations WHERE workspace_id=? AND (?='' OR item_id=?) ORDER BY observed_at DESC,id`, workspaceID, itemID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Observation
	for rows.Next() {
		v, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanObservation(s interface{ Scan(...any) error }) (Observation, error) {
	var v Observation
	var amount, observed, valid string
	var minor, exponent, available, membership, coupon int
	if err := s.Scan(&v.ID, &v.WorkspaceID, &v.ItemID, &v.ProductID, &v.PackageLabel, &amount, &v.PackageUnit, &minor, &v.Price.Currency, &exponent, &v.Retailer, &observed, &valid, &available, &membership, &coupon, &v.MinimumBuy, &v.Source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Observation{}, err
		}
		return Observation{}, err
	}
	var err error
	v.PackageAmount, err = decimalx.Parse(amount)
	if err != nil {
		return Observation{}, err
	}
	v.Price, err = money.New(int64(minor), v.Price.Currency, exponent)
	if err != nil {
		return Observation{}, err
	}
	v.ObservedAt, err = time.Parse(time.RFC3339Nano, observed)
	if err != nil {
		return Observation{}, err
	}
	if valid != "" {
		v.ValidThrough, err = time.Parse(time.RFC3339Nano, valid)
		if err != nil {
			return Observation{}, err
		}
	}
	v.Available, v.MembershipRequired, v.CouponRequired = available != 0, membership != 0, coupon != 0
	return v, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
