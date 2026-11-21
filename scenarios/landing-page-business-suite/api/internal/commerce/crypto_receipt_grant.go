package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

// CryptoReceiptGrantService turns an immutable Money Ledger acceptance into
// an LPBS funding lot. The receipt ID is the replay key; callers never choose
// an LPBS lot ID or a credit amount.
type CryptoReceiptGrantService struct {
	DB interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Plans  BundleProductReader
	Wallet interface {
		AddCreditsForBusinessAccountContext(context.Context, string, string, int64, string, string, string, map[string]interface{}) error
	}
	LiveMode bool
}

type CryptoReceiptLot struct {
	ID, ReceiptID, BusinessAccountID, CustomerEmail string
	Credits                                         int64
}

func (s *CryptoReceiptGrantService) GrantAcceptedReceipt(ctx context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CryptoReceiptLot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.DB == nil || s.Plans == nil || s.Wallet == nil {
		return CryptoReceiptLot{}, errors.New("crypto receipt grant owners are unavailable")
	}
	if receipt == nil || strings.TrimSpace(receipt.Id) == "" || strings.TrimSpace(receipt.PostingId) == "" || receipt.Source == nil || receipt.Source.GetOperatorIdentity() == "" || receipt.GetAcceptedBy() != "treasury" || receipt.AcceptedAt == nil || receipt.AcceptedAt.CheckValid() != nil || receipt.AcceptedAt.AsTime().IsZero() || receipt.Action != ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL || !receipt.NoChargeback || receipt.Observation == nil || receipt.Attribution == nil || receipt.Valuation == nil {
		return CryptoReceiptLot{}, errors.New("an accepted original receipt with payer, valuation and no-chargeback evidence is required")
	}
	observation, valuation, attribution := receipt.Observation, receipt.Valuation, receipt.Attribution
	if observation.GetNetwork() != "eip155:8453" || observation.GetAssetId() != "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" || observation.GetDecimals() != 6 || observation.GetState() != ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED {
		return CryptoReceiptLot{}, errors.New("receipt must be a finalized native Base USDC transfer")
	}
	atomic, ok := new(big.Int).SetString(receipt.GetAtomicDelta(), 10)
	if !ok || atomic.Sign() <= 0 || atomic.String() != receipt.GetAtomicDelta() || atomic.String() != observation.GetAtomicAmount() {
		return CryptoReceiptLot{}, errors.New("accepted receipt must contain a canonical positive atomic quantity")
	}
	payerAddress := strings.ToLower(strings.TrimSpace(attribution.GetPayerAddress()))
	payerIdentity := strings.TrimSpace(attribution.GetPayerIdentity())
	if payerAddress == "" || payerIdentity == "" || valuation.GetAmountMinor() <= 0 || !strings.EqualFold(valuation.GetCurrency(), "USD") || valuation.GetBasis() == sharedpb.Basis_BASIS_UNSPECIFIED || valuation.GetBasisReference() == "" {
		return CryptoReceiptLot{}, errors.New("verified payer and explicit positive USD valuation basis are required")
	}
	var accountID, customerEmail string
	err := s.DB.QueryRowContext(ctx, `SELECT business_account_id,customer_email FROM crypto_payer_customer_bindings WHERE payer_identity=$1 AND lower(payer_address)=lower($2) AND enabled=TRUE`, payerIdentity, payerAddress).Scan(&accountID, &customerEmail)
	if err != nil {
		return CryptoReceiptLot{}, fmt.Errorf("resolve verified crypto payer to LPBS customer: %w", err)
	}
	bundle, err := s.Plans.GetBundleProduct()
	if err != nil {
		return CryptoReceiptLot{}, err
	}
	if bundle == nil || bundle.CreditsPerUsd <= 0 || valuation.GetAmountMinor() > math.MaxInt64/bundle.CreditsPerUsd {
		return CryptoReceiptLot{}, errors.New("LPBS credit conversion policy is missing or overflows")
	}
	credits := bundle.CreditsPerUsd * valuation.GetAmountMinor() / 100
	if credits <= 0 {
		return CryptoReceiptLot{}, errors.New("accepted receipt valuation does not produce a positive LPBS grant")
	}
	metadata := map[string]interface{}{
		"livemode":                      s.LiveMode,
		"paid_amount_minor":             valuation.GetAmountMinor(),
		"paid_currency":                 strings.ToUpper(valuation.GetCurrency()),
		"money_ledger_receipt_id":       receipt.GetId(),
		"crypto_network":                observation.GetNetwork(),
		"crypto_asset_id":               observation.GetAssetId(),
		"crypto_atomic_amount":          atomic.String(),
		"crypto_payer_address":          payerAddress,
		"crypto_payer_identity":         payerIdentity,
		"crypto_no_chargeback":          receipt.GetNoChargeback(),
		"crypto_valuation_basis":        valuation.GetBasis().String(),
		"crypto_valuation_basis_ref":    valuation.GetBasisReference(),
		"crypto_valuation_amount_minor": valuation.GetAmountMinor(),
	}
	if err := s.Wallet.AddCreditsForBusinessAccountContext(ctx, accountID, customerEmail, credits, "crypto_topup", "money-ledger", receipt.GetId(), metadata); err != nil {
		return CryptoReceiptLot{}, err
	}
	var lotID string
	if err := s.DB.QueryRowContext(ctx, `SELECT id::text FROM credit_funding_lots WHERE provider='money-ledger' AND provider_ref=$1 AND business_account_id=$2 AND money_ledger_receipt_id=$1`, receipt.GetId(), accountID).Scan(&lotID); err != nil {
		return CryptoReceiptLot{}, fmt.Errorf("read created LPBS credit funding lot: %w", err)
	}
	return CryptoReceiptLot{ID: lotID, ReceiptID: receipt.GetId(), BusinessAccountID: accountID, CustomerEmail: customerEmail, Credits: credits}, nil
}
