package commerce

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	fundingstate "landing-page-business-suite-api/internal/commerce/fundinglot/flow/generated"
)

// CreditWallet owns attributed account funding grants and their displayed balance.
// seam: commerce funding authority, backed by the account lot ledger.
// Credit spending is owned by BusinessAccountUsageService, never an email wallet.
type CreditWallet interface {
	AddCreditsForBusinessAccount(businessAccountID, customerEmail string, amount int64, txnType, source, externalEventID string, metadata map[string]interface{}) error
	BalanceForBusinessAccount(businessAccountID, customerEmail string) (int64, error)
}

type creditWalletStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type CreditWalletService struct{ db creditWalletStore }

func NewCreditWalletService(db creditWalletStore) *CreditWalletService {
	return &CreditWalletService{db: db}
}

var _ CreditWallet = (*CreditWalletService)(nil)

func knownGrantMode(metadata map[string]interface{}) (bool, error) {
	switch value := metadata["livemode"].(type) {
	case bool:
		return value, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
	}
	return false, errors.New("funding livemode attribution is required")
}

func sameGrantMetadata(stored []byte, expected []byte) bool {
	var value any
	if json.Unmarshal(stored, &value) != nil {
		return false
	}
	canonical, err := json.Marshal(value)
	return err == nil && bytes.Equal(canonical, expected)
}

func (s *CreditWalletService) AddCreditsForBusinessAccount(businessAccountID, customerEmail string, amount int64, txnType, source, externalEventID string, metadata map[string]interface{}) error {
	return s.AddCreditsForBusinessAccountContext(context.Background(), businessAccountID, customerEmail, amount, txnType, source, externalEventID, metadata)
}

// AddCreditsForBusinessAccountContext binds the entire funding transaction to
// the request-selected database pool. In particular, test-mode requests must
// never fall through to the primary customer ledger.
func (s *CreditWalletService) AddCreditsForBusinessAccountContext(ctx context.Context, businessAccountID, customerEmail string, amount int64, txnType, source, externalEventID string, metadata map[string]interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	businessAccountID, customerEmail = strings.TrimSpace(businessAccountID), normalizeEmail(customerEmail)
	txnType, source, externalEventID = strings.TrimSpace(txnType), strings.TrimSpace(source), strings.TrimSpace(externalEventID)
	if businessAccountID == "" || customerEmail == "" || amount <= 0 || txnType == "" || source == "" || externalEventID == "" {
		return errors.New("business_account_id, customer, positive amount, funding kind, source, and external event attribution are required")
	}
	liveMode, err := knownGrantMode(metadata)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode immutable grant metadata: %w", err)
	}
	if s.db == nil {
		return errors.New("account funding authority is unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account credit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, businessAccountID); err != nil {
		return err
	}
	if err := checkFundingPurchaseBlock(ctx, tx, businessAccountID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO business_account_credit_transactions (business_account_id, customer_email, amount_credits, transaction_type, source, external_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (source, external_event_id) DO NOTHING
	`, strings.TrimSpace(businessAccountID), normalizeEmail(customerEmail), amount, txnType, source, externalEventID, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("insert account credit transaction: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check account credit transaction: %w", err)
	}
	if rowsAffected == 0 {
		var account, email, kind string
		var previousAmount int64
		var originalMetadata []byte
		if err := tx.QueryRowContext(ctx, `SELECT business_account_id,customer_email,amount_credits,transaction_type,metadata FROM business_account_credit_transactions WHERE source=$1 AND external_event_id=$2`, source, externalEventID).Scan(&account, &email, &previousAmount, &kind, &originalMetadata); err != nil {
			return fmt.Errorf("load existing account grant: %w", err)
		}
		if account != businessAccountID || email != customerEmail || previousAmount != amount || kind != txnType || !sameGrantMetadata(originalMetadata, metadataJSON) {
			return errors.New("funding event was replayed with different account or grant data")
		}
		return nil
	}
	lotID := newReservationID()
	providerRef := firstNonEmpty(metadataString(metadata, "payment_intent_id"), externalEventID)
	providerRef2 := firstNonEmpty(metadataString(metadata, "charge_id"), metadataString(metadata, "session_id"))
	if _, err := tx.ExecContext(ctx, `INSERT INTO credit_funding_lots (id,business_account_id,source_kind,provider,provider_ref,provider_ref2,external_event_id,livemode,settlement_state,granted,remaining,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'settled',$9,$9,'open',NOW())`, lotID, businessAccountID, txnType, source, providerRef, providerRef2, externalEventID, liveMode, amount); err != nil {
		return fmt.Errorf("create account funding lot: %w", err)
	}
	if source == "money-ledger" {
		atomicAmount := metadataString(metadata, "crypto_atomic_amount")
		payerAddress := strings.ToLower(strings.TrimSpace(metadataString(metadata, "crypto_payer_address")))
		payerIdentity := strings.TrimSpace(metadataString(metadata, "crypto_payer_identity"))
		receiptID := strings.TrimSpace(metadataString(metadata, "money_ledger_receipt_id"))
		network, assetID := strings.TrimSpace(metadataString(metadata, "crypto_network")), strings.TrimSpace(metadataString(metadata, "crypto_asset_id"))
		noChargeback, _ := metadata["crypto_no_chargeback"].(bool)
		if atomicAmount == "" || payerAddress == "" || payerIdentity == "" || receiptID == "" || network == "" || assetID == "" || !noChargeback {
			return errors.New("accepted crypto lot requires exact receipt, asset and payer attribution")
		}
		atomic, ok := new(big.Int).SetString(atomicAmount, 10)
		if !ok || atomic.Sign() <= 0 || atomic.String() != atomicAmount {
			return errors.New("accepted crypto lot has invalid atomic amount")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE credit_funding_lots SET crypto_network=$2,crypto_asset_id=$3,crypto_atomic_amount=$4,crypto_payer_address=$5,crypto_payer_identity=$6,money_ledger_receipt_id=$7,crypto_no_chargeback=$8 WHERE id=$1`, lotID, network, assetID, atomicAmount, payerAddress, payerIdentity, receiptID, noChargeback); err != nil {
			return fmt.Errorf("persist accepted crypto attribution on lot: %w", err)
		}
	}
	if paid, ok := metadata["paid_amount_minor"].(int64); ok {
		if _, err := tx.ExecContext(ctx, `UPDATE credit_funding_lots SET paid_amount_minor=$2,paid_currency=$3 WHERE id=$1`, lotID, paid, metadataString(metadata, "paid_currency")); err != nil {
			return err
		}
	}
	if err := applyFundingPolicy(ctx, tx, lotID, businessAccountID, source, txnType); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE business_account_credit_transactions SET lot_id=$1 WHERE source=$2 AND external_event_id=$3`, lotID, source, externalEventID); err != nil {
		return fmt.Errorf("link account credit transaction to funding lot: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO business_account_credit_wallets (business_account_id, customer_email, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (business_account_id) DO UPDATE SET customer_email = EXCLUDED.customer_email, updated_at = NOW()
	`, strings.TrimSpace(businessAccountID), normalizeEmail(customerEmail)); err != nil {
		return fmt.Errorf("update account credit wallet: %w", err)
	}
	return tx.Commit()
}

func (s *CreditWalletService) BalanceForBusinessAccount(businessAccountID, customerEmail string) (int64, error) {
	if strings.TrimSpace(businessAccountID) == "" || strings.TrimSpace(customerEmail) == "" {
		return 0, ErrBusinessAccountRequired
	}
	var balance int64
	// The wallet is an identity row. Only attributed, usable funding lots
	// establish the displayed balance; an old cache amount has no provenance.
	err := s.db.QueryRowContext(context.Background(), `SELECT COALESCE(SUM(l.remaining),0) FROM credit_funding_lots l JOIN business_account_credit_wallets w ON w.business_account_id=l.business_account_id WHERE w.business_account_id=$1 AND w.customer_email=$2 AND l.status=$3 AND l.settlement_state='settled' AND l.livemode IS NOT NULL AND (l.spendable_after IS NULL OR l.spendable_after<=CURRENT_TIMESTAMP) AND (l.expires_at IS NULL OR l.expires_at>CURRENT_TIMESTAMP)`, strings.TrimSpace(businessAccountID), normalizeEmail(customerEmail), string(fundingstate.FundingLotOpen)).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get account credit balance: %w", err)
	}
	return balance, nil
}
