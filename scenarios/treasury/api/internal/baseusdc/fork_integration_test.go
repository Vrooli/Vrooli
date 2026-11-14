package baseusdc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"treasury/internal/testutil/basefork"
	walletpkg "treasury/internal/wallet"
)

func TestLocalBaseForkVerifiesActualUSDCTransfer(t *testing.T) {
	fork := basefork.Start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_, err := fork.Mint(ctx, fork.Accounts[0], "9000001")
	require.NoError(t, err)
	tx, err := fork.Transfer(ctx, fork.Accounts[0], fork.Accounts[1], "1000001")
	require.NoError(t, err)
	require.NoError(t, fork.Call(ctx, "anvil_mine", []any{"0x80"}, nil))
	verifier, err := NewVerifier(fork, "local-base-fork", time.Now)
	require.NoError(t, err)
	observations, err := verifier.VerifyWalletTransaction(ctx, walletpkg.Wallet{ID: "fork-budget", Network: Network, Address: fork.Accounts[1], CustodyReference: "test-only"}, tx)
	require.NoError(t, err)
	require.Len(t, observations, 1)
	require.Equal(t, "1000001", observations[0].AtomicAmount)
	require.Equal(t, "finalized", observations[0].Finality)
	require.Equal(t, fork.Accounts[0], observations[0].From)
	t.Logf("actual USDC transfer=%s atomic=%s finality=%s", tx, observations[0].AtomicAmount, observations[0].Finality)
}
