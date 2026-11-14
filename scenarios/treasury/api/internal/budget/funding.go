package budget

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Funding is authorization evidence derived only from accepted ledger receipts.
// Valuation limits spending in the budget's currency. AtomicAmount is separate
// token inventory evidence and is never implicitly treated as a dollar amount.
type Funding struct {
	BudgetID, Network, AssetID, AtomicAmount string
	Decimals                                 uint32
	ReceiptCount, CreditedMinor              int64
	Required                                 bool
}

type FundingCredit struct {
	LedgerReceiptID, PostingID, SourceCheckID string
	BudgetID, AtomicDelta, ValuationCurrency  string
	ValuationDeltaMinor                       int64
	AcceptedAt                                time.Time
}

type FundingReader interface {
	Funding(context.Context, string) (Funding, error)
}

// BindFunding is an operator-side configuration operation. Once a budget is
// receipt-funded it cannot silently return to policy-only authorization.
func (r *SQLiteRepository) BindFunding(ctx context.Context, budgetID, network, assetID string, decimals uint32, now time.Time) error {
	if strings.TrimSpace(budgetID) == "" || strings.TrimSpace(network) == "" || strings.TrimSpace(assetID) == "" || decimals > 255 || now.IsZero() {
		return fmt.Errorf("%w: budget and exact asset identity are required", ErrInvalid)
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO budget_funding_bindings(budget_id,network,asset_id,decimals,created_at) VALUES(?,?,?,?,?) ON CONFLICT(budget_id) DO NOTHING`, budgetID, network, assetID, decimals, now.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	var storedNetwork, storedAsset string
	var storedDecimals uint32
	if err := tx.QueryRowContext(ctx, `SELECT network,asset_id,decimals FROM budget_funding_bindings WHERE budget_id=?`, budgetID).Scan(&storedNetwork, &storedAsset, &storedDecimals); err != nil {
		return err
	}
	if storedNetwork != network || storedAsset != assetID || storedDecimals != decimals {
		return fmt.Errorf("%w: budget already has a different funding asset", ErrInvalid)
	}
	return tx.Commit()
}

// CreditAccepted records the acceptance response from the authenticated Money
// Ledger client. It is not exposed by AgentSpend or an operator mutation RPC.
// A lost response or restart can replay this call without granting credit twice.
func (r *SQLiteRepository) CreditAccepted(ctx context.Context, value FundingCredit) error {
	if value.LedgerReceiptID == "" || value.PostingID == "" || value.SourceCheckID == "" || value.BudgetID == "" || value.AcceptedAt.IsZero() || value.ValuationCurrency == "" {
		return fmt.Errorf("%w: an accepted ledger receipt and posting are required", ErrInvalid)
	}
	atomic, ok := new(big.Int).SetString(value.AtomicDelta, 10)
	if !ok || atomic.String() != value.AtomicDelta {
		return fmt.Errorf("%w: atomic delta must be a canonical signed integer", ErrInvalid)
	}
	value.AcceptedAt = value.AcceptedAt.UTC()
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(digest[:])
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var currency string
	if err := tx.QueryRowContext(ctx, `SELECT b.currency FROM budgets b JOIN budget_funding_bindings f ON f.budget_id=b.id WHERE b.id=?`, value.BudgetID).Scan(&currency); err != nil {
		return fmt.Errorf("receipt funding binding: %w", err)
	}
	if currency != value.ValuationCurrency {
		return fmt.Errorf("%w: receipt valuation currency does not match budget", ErrInvalid)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO budget_funding_credits(ledger_receipt_id,budget_id,posting_id,source_check_id,atomic_delta,valuation_delta_minor,valuation_currency,accepted_at,payload_sha256) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(ledger_receipt_id) DO NOTHING`, value.LedgerReceiptID, value.BudgetID, value.PostingID, value.SourceCheckID, value.AtomicDelta, value.ValuationDeltaMinor, value.ValuationCurrency, value.AcceptedAt.Format(time.RFC3339Nano), fingerprint); err != nil {
		return err
	}
	var existing string
	if err := tx.QueryRowContext(ctx, `SELECT payload_sha256 FROM budget_funding_credits WHERE ledger_receipt_id=?`, value.LedgerReceiptID).Scan(&existing); err != nil {
		return err
	}
	if existing != fingerprint {
		return fmt.Errorf("%w: accepted receipt was already credited with different content", ErrInvalid)
	}
	return tx.Commit()
}

func (r *SQLiteRepository) Funding(ctx context.Context, budgetID string) (Funding, error) {
	value := Funding{BudgetID: budgetID, AtomicAmount: "0"}
	err := r.db.QueryRowContext(ctx, `SELECT network,asset_id,decimals FROM budget_funding_bindings WHERE budget_id=?`, budgetID).Scan(&value.Network, &value.AssetID, &value.Decimals)
	if errors.Is(err, sql.ErrNoRows) {
		return value, nil
	}
	if err != nil {
		return Funding{}, err
	}
	value.Required = true
	rows, err := r.db.QueryContext(ctx, `SELECT atomic_delta,valuation_delta_minor FROM budget_funding_credits WHERE budget_id=? ORDER BY accepted_at,ledger_receipt_id`, budgetID)
	if err != nil {
		return Funding{}, err
	}
	defer rows.Close()
	tokens, valuation := new(big.Int), new(big.Int)
	for rows.Next() {
		var atomic string
		var minor int64
		if err := rows.Scan(&atomic, &minor); err != nil {
			return Funding{}, err
		}
		quantity, ok := new(big.Int).SetString(atomic, 10)
		if !ok || quantity.String() != atomic {
			return Funding{}, errors.New("stored receipt funding quantity is invalid")
		}
		tokens.Add(tokens, quantity)
		valuation.Add(valuation, big.NewInt(minor))
		value.ReceiptCount++
	}
	if err := rows.Err(); err != nil {
		return Funding{}, err
	}
	if !valuation.IsInt64() {
		return Funding{}, errors.New("receipt funding valuation exceeds budget range")
	}
	value.AtomicAmount, value.CreditedMinor = tokens.String(), valuation.Int64()
	return value, nil
}

func (s *service) Funding(ctx context.Context, budgetID string) (Funding, error) {
	if _, err := s.Get(ctx, budgetID); err != nil {
		return Funding{}, err
	}
	reader, ok := s.repository.(FundingReader)
	if !ok {
		return Funding{BudgetID: budgetID, AtomicAmount: "0"}, nil
	}
	return reader.Funding(ctx, budgetID)
}
