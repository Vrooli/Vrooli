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

var (
	ErrPersist            = errors.New("Base receipt evidence could not be retained")
	ErrPriorCheckNotFound = errors.New("verified Base receipt check not found in this operator, wallet, and transaction scope")
)

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
	return s.verify(ctx, operator, wallet, transactionHash, nil)
}

func (s *Service) Recheck(ctx context.Context, operator string, wallet walletpkg.Wallet, transactionHash, priorCheckID string) (Check, error) {
	if s == nil || s.store == nil || operator == "" || wallet.OperatorIdentity != operator || wallet.ID == "" || wallet.Network != Network || !evHash.MatchString(transactionHash) || strings.TrimSpace(priorCheckID) == "" {
		return Check{}, errors.New("operator-bound Base wallet, transaction hash, and prior check are required")
	}
	canonicalHash := strings.ToLower(transactionHash)
	prior, err := s.store.GetPriorVerified(ctx, operator, wallet.ID, wallet.Network, canonicalHash, priorCheckID)
	if err != nil {
		return Check{}, err
	}
	if prior.OperatorIdentity != operator || prior.WalletID != wallet.ID || prior.Network != wallet.Network || prior.TransactionHash != canonicalHash || prior.Outcome != "verified" || prior.ProviderID == "" || !validObservations(wallet, canonicalHash, prior.ProviderID, prior.Transfers) {
		return Check{}, ErrPriorCheckNotFound
	}
	return s.verify(ctx, operator, wallet, canonicalHash, &prior)
}

func (s *Service) verify(ctx context.Context, operator string, wallet walletpkg.Wallet, transactionHash string, prior *Check) (Check, error) {
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
	if prior != nil {
		check.PriorCheckID = prior.ID
	}
	if verifyErr == nil {
		check.Outcome = "verified"
		if prior != nil {
			check.Relation = recheckRelation(*prior, transfers)
			if check.Relation == "provider_observation_changed" {
				check.Outcome = check.Relation
			}
		}
	} else {
		check.ErrorCode = errorCode(verifyErr)
		check.Outcome = check.ErrorCode
		check.Transfers = nil
		if prior != nil {
			switch {
			case errors.Is(verifyErr, ErrNotCanonical):
				check.Relation = "provider_reports_not_canonical"
				check.Outcome = check.Relation
			case errors.Is(verifyErr, ErrPending):
				check.Relation = "provider_reports_pending"
			default:
				check.Relation = "provider_check_failed"
			}
		}
	}
	if err := s.store.Append(ctx, check); err != nil {
		return Check{}, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return check, verifyErr
}

func recheckRelation(prior Check, current []Observation) string {
	priorByKey := make(map[string]Observation, len(prior.Transfers))
	for _, value := range prior.Transfers {
		priorByKey[value.EconomicKey] = value
	}
	if len(priorByKey) == 0 || len(priorByKey) != len(current) {
		return "provider_observation_changed"
	}
	reincluded := false
	for _, value := range current {
		old, ok := priorByKey[value.EconomicKey]
		if !ok || old.AssetID != value.AssetID || old.Decimals != value.Decimals || old.From != value.From || old.To != value.To || old.AtomicAmount != value.AtomicAmount || old.TransactionHash != value.TransactionHash || old.LogIndex != value.LogIndex {
			return "provider_observation_changed"
		}
		if old.InclusionRevision != value.InclusionRevision {
			reincluded = true
		}
	}
	if reincluded {
		return "reincluded_new_revision"
	}
	return "same_inclusion"
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
