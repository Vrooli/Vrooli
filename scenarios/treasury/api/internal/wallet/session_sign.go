package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SessionSignAuthorization is the narrow server-to-server signing contract
// for a mandate-derived wallet. It carries the complete economic scope of one
// unsigned transaction; it never carries key material or an agent-provided
// verdict.
type SessionSignAuthorization struct {
	WalletID, Network, OperatorIdentity string
	MandateID, AuthorizationID          string
	SigningPurpose                      string
	Recipient, AtomicAmount             string
	Currency                            string
	ValueMinor, AuthorizationValueMinor int64
	TransactionDigest                   string
}

// SessionSignAuthorizer is implemented by Treasury's wallet-ops owner. It
// checks the live mandate, settled authorization, freeze state, active
// session generation and one-use replay fence before custody can sign.
type SessionSignAuthorizer interface {
	AuthorizeAndConsumeSessionSign(context.Context, SessionSignAuthorization) error
}

func (s *Service) SetSessionSignAuthorizer(authorizer SessionSignAuthorizer) {
	s.sessionSignAuthorizer = authorizer
}

// SignUSDCTransfer exposes only a mandate-authorized transfer signing seam.
// Raw signing remains private to this package and cannot be reached without a
// current, single-use Treasury session grant.
func (s *Service) SignUSDCTransfer(ctx context.Context, request SessionSignAuthorization, tx *types.Transaction) (*types.Transaction, error) {
	if s == nil || s.sessionSignAuthorizer == nil {
		return nil, errors.New("mandate-scoped wallet signing authority is unavailable")
	}
	if request.WalletID == "" || request.Network == "" || request.OperatorIdentity == "" || request.MandateID == "" || request.AuthorizationID == "" || request.Currency == "" || request.ValueMinor <= 0 {
		return nil, errors.New("mandate-scoped transfer authorization is incomplete")
	}
	amountAtoms, err := AtomicAmountForUSDCMinor(request.ValueMinor, request.Currency)
	if err != nil || amountAtoms != request.AtomicAmount {
		return nil, errors.New("USDC token amount does not match the approved USD valuation")
	}
	if err := validateUSDCTransfer(request.Network, tx); err != nil {
		return nil, err
	}
	data := tx.Data()
	recipient := common.BytesToAddress(data[16:36]).Hex()
	atomicAmount := new(big.Int).SetBytes(data[36:]).String()
	if !strings.EqualFold(recipient, request.Recipient) || atomicAmount != request.AtomicAmount {
		return nil, errors.New("unsigned transfer does not match its mandate session scope")
	}
	unsigned, err := tx.MarshalBinary()
	if err != nil {
		return nil, errors.New("unsigned transfer could not be fingerprinted")
	}
	digest := sha256.Sum256(unsigned)
	request.TransactionDigest = "sha256:" + hex.EncodeToString(digest[:])
	if err = s.sessionSignAuthorizer.AuthorizeAndConsumeSessionSign(ctx, request); err != nil {
		return nil, err
	}
	return s.signUSDCTransfer(ctx, request.WalletID, request.Network, request.OperatorIdentity, tx)
}

// AtomicAmountForUSDCMinor converts USD minor units to six-decimal USDC atoms
// without floating point. Native USDC signing accepts USD valuation only.
func AtomicAmountForUSDCMinor(valueMinor int64, currency string) (string, error) {
	if valueMinor <= 0 || strings.ToUpper(strings.TrimSpace(currency)) != "USD" {
		return "", errors.New("positive USD valuation is required for native USDC")
	}
	amount := new(big.Int).Mul(big.NewInt(valueMinor), big.NewInt(10_000))
	if amount.BitLen() > 256 {
		return "", errors.New("USDC atomic amount exceeds uint256")
	}
	return amount.String(), nil
}
