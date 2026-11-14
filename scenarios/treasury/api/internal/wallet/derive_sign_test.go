package wallet

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type allowSessionSign struct{}

func (allowSessionSign) AuthorizeAndConsumeSessionSign(_ context.Context, request SessionSignAuthorization) error {
	if request.MandateID == "" || request.AuthorizationID == "" || request.TransactionDigest == "" {
		return errors.New("incomplete test session authorization")
	}
	return nil
}

func TestDerivedBudgetWalletsAreDistinctStableAndCustodySignaturesRecover(t *testing.T) {
	s, _, in := setup(t)
	s.SetSessionSignAuthorizer(allowSessionSign{})
	ctx := context.Background()
	parent, err := s.Create(ctx, in)
	require.NoError(t, err)
	childInput := in
	childInput.ID = "budget-one"
	child, err := s.CreateDerived(ctx, parent.ID, "budget-one", childInput)
	require.NoError(t, err)
	again, err := s.CreateDerived(ctx, parent.ID, "budget-one", childInput)
	require.NoError(t, err)
	require.Equal(t, child.Address, again.Address)
	childInput.ID = "budget-two"
	other, err := s.CreateDerived(ctx, parent.ID, "budget-two", childInput)
	require.NoError(t, err)
	require.NotEqual(t, child.Address, other.Address)
	require.NotEqual(t, parent.Address, child.Address)
	asset := common.HexToAddress("0x036cbd53842c5426634e7929541ec2318f3dcf7e")
	data := make([]byte, 68)
	copy(data, []byte{0xa9, 0x05, 0x9c, 0xbb})
	copy(data[16:36], common.HexToAddress(other.Address).Bytes())
	big.NewInt(1000000).FillBytes(data[36:])
	unsigned := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(84532), Nonce: 3, To: &asset, Gas: 100000, GasFeeCap: big.NewInt(100000000), GasTipCap: big.NewInt(1000000), Data: data})
	signRequest := SessionSignAuthorization{
		WalletID: child.ID, Network: child.Network, OperatorIdentity: child.OperatorIdentity,
		MandateID: "mandate-test", AuthorizationID: "authorization-test", Recipient: other.Address,
		AtomicAmount: "1000000", Currency: "USD", ValueMinor: 100,
	}
	signed, err := s.SignUSDCTransfer(ctx, signRequest, unsigned)
	require.NoError(t, err)
	sender, err := types.Sender(types.NewLondonSigner(big.NewInt(84532)), signed)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(child.Address), sender)
	require.Equal(t, data, signed.Data())
	require.Equal(t, uint64(3), signed.Nonce())
	signRequest.OperatorIdentity = "another-operator"
	_, err = s.SignUSDCTransfer(ctx, signRequest, unsigned)
	require.Error(t, err)
	data[0] = 0x09
	signRequest.OperatorIdentity = child.OperatorIdentity
	_, err = s.SignUSDCTransfer(ctx, signRequest, types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(84532), To: &asset, Gas: 100000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), Data: data}))
	require.Error(t, err, "arbitrary calls and approvals cannot use the transfer signing seam")
}
