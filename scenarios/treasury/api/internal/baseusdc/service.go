package baseusdc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	walletpkg "treasury/internal/wallet"
)

var ErrPersist = errors.New("Base receipt evidence could not be retained")

var atomicAmountPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

type Verification interface {
	VerifyWalletTransaction(context.Context, walletpkg.Wallet, string) ([]Observation, error)
}

type Service struct {
	store    Store
	verifier Verification
	provider string
	now      func() time.Time
}

func NewService(store Store, verifier Verification, providerID string, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, verifier: verifier, provider: strings.TrimSpace(providerID), now: now}
}

func (s *Service) Verify(ctx context.Context, operator string, wallet walletpkg.Wallet, transactionHash string) (Check, error) {
	if s == nil || s.store == nil || s.verifier == nil || s.provider == "" || operator == "" || wallet.OperatorIdentity != operator || wallet.ID == "" || wallet.Network != Network || !evHash.MatchString(transactionHash) {
		return Check{}, errors.New("operator-bound Base wallet, transaction hash, and receipt service are required")
	}
	canonicalHash := strings.ToLower(transactionHash)
	transfers, verifyErr := s.verifier.VerifyWalletTransaction(ctx, wallet, canonicalHash)
	if verifyErr == nil && !validObservations(wallet, canonicalHash, s.provider, transfers) {
		verifyErr = errors.New("Base receipt verifier returned inconsistent transfer evidence")
	}
	checkID, err := newCheckID()
	if err != nil {
		return Check{}, err
	}
	checkedAt := s.now().UTC()
	check := Check{
		ID: checkID, OperatorIdentity: operator, WalletID: wallet.ID, Network: Network,
		TransactionHash: canonicalHash, ProviderID: s.provider, CheckedAt: checkedAt,
		Transfers: transfers,
	}
	if verifyErr == nil {
		check.Outcome = "verified"
	} else {
		check.ErrorCode = errorCode(verifyErr)
		check.Outcome = check.ErrorCode
		check.Transfers = nil
	}
	if err := s.store.Append(ctx, check); err != nil {
		return Check{}, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return check, verifyErr
}

func (s *Service) List(ctx context.Context, operator, walletID, transactionHash string, limit int) ([]Check, error) {
	if s == nil || s.store == nil || operator == "" || walletID == "" || !evHash.MatchString(transactionHash) {
		return nil, errors.New("operator, wallet, and valid transaction hash are required")
	}
	return s.store.List(ctx, operator, walletID, strings.ToLower(transactionHash), limit)
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, ErrPending):
		return "pending"
	case errors.Is(err, ErrFailed):
		return "transaction_failed"
	case errors.Is(err, ErrWrongNetwork):
		return "wrong_network"
	case errors.Is(err, ErrNoTransfer):
		return "no_matching_transfer"
	case errors.Is(err, ErrNotCanonical):
		return "not_canonical"
	case errors.Is(err, ErrRPCFinality), errors.Is(err, ErrRPCUnavailable):
		return "provider_unavailable"
	default:
		return "invalid_provider_evidence"
	}
}

func validObservations(wallet walletpkg.Wallet, transactionHash, provider string, observations []Observation) bool {
	if len(observations) == 0 {
		return false
	}
	for _, value := range observations {
		if value.WalletID != wallet.ID || value.Network != Network || value.AssetID != TransferAsset || value.Decimals != USDCDecimals ||
			!strings.EqualFold(value.To, wallet.Address) || !evAddress.MatchString(value.From) || !atomicAmountPattern.MatchString(value.AtomicAmount) ||
			!strings.EqualFold(value.TransactionHash, transactionHash) || !evHash.MatchString(value.BlockHash) ||
			!evHash.MatchString(value.FinalizedHeadHash) || value.ProviderID != provider || value.ObservedAt.IsZero() || value.BlockTime.IsZero() ||
			(value.Finality != "included" && value.Finality != "finalized") {
			return false
		}
		if value.InclusionRevision != strings.ToLower(transactionHash)+"/log/"+fmt.Sprint(value.LogIndex)+"@"+strings.ToLower(value.BlockHash) {
			return false
		}
		if value.Finality == "finalized" && value.FinalizedHeadNumber < value.BlockNumber {
			return false
		}
	}
	return true
}

func newCheckID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "base-receipt-" + hex.EncodeToString(value[:]), nil
}
