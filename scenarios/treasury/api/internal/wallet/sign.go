package wallet

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// signUSDCTransfer is the custody-only primitive. Callers must pass through
// SignUSDCTransfer's mandate-scoped authorizer before this method can run.
func (s *Service) signUSDCTransfer(ctx context.Context, id, network, operator string, tx *types.Transaction) (*types.Transaction, error) {
	w, err := s.Get(ctx, id, network, operator)
	if err != nil {
		return nil, err
	}
	if err = validateUSDCTransfer(network, tx); err != nil {
		return nil, err
	}
	chain := int64(8453)
	if network == "eip155:84532" {
		chain = 84532
	}
	key, err := s.key(w)
	if err != nil {
		return nil, err
	}
	defer key.Zero()
	raw := key.Serialize()
	defer clear(raw)
	private, err := gethcrypto.ToECDSA(raw)
	if err != nil {
		return nil, err
	}
	defer private.D.SetInt64(0)
	return types.SignTx(tx, types.NewLondonSigner(big.NewInt(chain)), private)
}

func validateUSDCTransfer(network string, tx *types.Transaction) error {
	chain := int64(8453)
	asset := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	if network == "eip155:84532" {
		chain = 84532
		asset = common.HexToAddress("0x036cbd53842c5426634e7929541ec2318f3dcf7e")
	} else if network != "eip155:8453" {
		return errors.New("custody refuses unsupported native-USDC network")
	}
	if tx == nil || tx.Type() != types.DynamicFeeTxType || tx.ChainId().Cmp(big.NewInt(chain)) != 0 || tx.To() == nil || *tx.To() != asset || tx.Value().Sign() != 0 || tx.Gas() == 0 || tx.Gas() > 1_000_000 || tx.GasFeeCap().Sign() <= 0 || tx.GasTipCap().Sign() < 0 || tx.GasTipCap().Cmp(tx.GasFeeCap()) > 0 {
		return errors.New("custody refuses a transaction outside the bounded native-USDC transfer contract")
	}
	data := tx.Data()
	if len(data) != 68 || !bytes.Equal(data[:4], []byte{0xa9, 0x05, 0x9c, 0xbb}) || !bytes.Equal(data[4:16], make([]byte, 12)) || new(big.Int).SetBytes(data[36:]).Sign() <= 0 || bytes.Equal(data[16:36], make([]byte, 20)) {
		return errors.New("custody requires one positive ERC20 transfer to a nonzero address")
	}
	return nil
}
