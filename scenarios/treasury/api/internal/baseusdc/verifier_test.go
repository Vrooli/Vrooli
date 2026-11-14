package baseusdc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	walletpkg "treasury/internal/wallet"
)

const (
	testWalletID = "coinbase-inbound"
	testTo       = "0x1111111111111111111111111111111111111111"
	testFrom     = "0x2222222222222222222222222222222222222222"
	testTx       = "0x" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testBlock    = "0x" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type fakeRPC struct {
	chainID string
	receipt *receipt
	head    *block
	block   *block
	methods []string
	err     map[string]error
}

func (r *fakeRPC) Call(_ context.Context, method string, params []any, out any) error {
	r.methods = append(r.methods, method)
	if err := r.err[method]; err != nil {
		return err
	}
	switch value := out.(type) {
	case *string:
		*value = r.chainID
	case **receipt:
		*value = r.receipt
	case **block:
		if params[0] == "finalized" {
			*value = r.head
		} else {
			*value = r.block
		}
	default:
		return fmt.Errorf("unexpected result type %T", out)
	}
	return nil
}

func testVerifier(rpc *fakeRPC, at time.Time) *Verifier {
	v, err := NewVerifier(rpc, "fixture-base-rpc", func() time.Time { return at })
	if err != nil {
		panic(err)
	}
	return v
}

func testWallet() walletpkg.Wallet {
	return walletpkg.Wallet{ID: testWalletID, Network: Network, Address: testTo, CustodyReference: "vrooli/treasury/wallets/fixture/reference"}
}

func validRPC() *fakeRPC {
	blockAt := &block{Number: "0x64", Hash: testBlock, Timestamp: "0x6553f100"}
	return &fakeRPC{
		chainID: "0x2105",
		receipt: &receipt{Status: "0x1", TransactionHash: testTx, BlockNumber: "0x64", BlockHash: testBlock, Logs: []log{transferLog(0, testFrom, testTo, "64")}},
		head:    &block{Number: "0x64", Hash: testBlock, Timestamp: blockAt.Timestamp},
		block:   blockAt,
		err:     map[string]error{},
	}
}

func transferLog(index uint64, from, to, amount string) log {
	return log{
		Address:         USDCAddress,
		Topics:          []string{transferEventTopic, indexedAddress(from), indexedAddress(to)},
		Data:            "0x" + strings.Repeat("0", 64-len(amount)) + amount,
		TransactionHash: testTx, BlockNumber: "0x64", BlockHash: testBlock,
		LogIndex: fmt.Sprintf("0x%x", index),
	}
}

func indexedAddress(address string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}

func TestVerifyWalletTransactionRequiresBaseAndReturnsExactFinalizedTransfer(t *testing.T) {
	rpc := validRPC()
	at := time.Date(2026, 11, 2, 18, 0, 0, 0, time.FixedZone("EST", -5*60*60))
	observations, err := testVerifier(rpc, at).VerifyWalletTransaction(context.Background(), testWallet(), strings.ReplaceAll(testTx, "a", "A"))
	require.NoError(t, err)
	require.Len(t, observations, 1)
	got := observations[0]
	require.Equal(t, "100", got.AtomicAmount)
	require.Equal(t, uint32(6), got.Decimals)
	require.Equal(t, TransferAsset, got.AssetID)
	require.Equal(t, testTo, got.To)
	require.Equal(t, testFrom, got.From)
	require.Equal(t, "finalized", got.Finality)
	require.Equal(t, "0x64", fmt.Sprintf("0x%x", got.FinalizedHeadNumber))
	require.Equal(t, "fixture-base-rpc", got.ProviderID)
	require.True(t, got.ObservedAt.Equal(at.UTC()))
	require.True(t, got.BlockTime.Equal(time.Unix(0x6553f100, 0).UTC()))
	require.Contains(t, got.EconomicKey, "/log/0/")
	require.Contains(t, got.InclusionRevision, "@"+strings.ToLower(testBlock))
	require.Equal(t, []string{"eth_chainId", "eth_getTransactionReceipt", "eth_getBlockByNumber", "eth_getBlockByNumber"}, rpc.methods)
}

func TestVerifyWalletTransactionKeepsIncludedTransferUnfinalized(t *testing.T) {
	rpc := validRPC()
	rpc.head.Number = "0x63"
	observations, err := testVerifier(rpc, time.Now()).VerifyWalletTransaction(context.Background(), testWallet(), testTx)
	require.NoError(t, err)
	require.Len(t, observations, 1)
	require.Equal(t, "included", observations[0].Finality)
	require.Equal(t, uint64(99), observations[0].FinalizedHeadNumber)
}

func TestVerifyWalletTransactionFailsClosedOnNetworkAndReceiptFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*fakeRPC)
		want   error
	}{
		{name: "wrong network", mutate: func(r *fakeRPC) { r.chainID = "0x14a34" }, want: ErrWrongNetwork},
		{name: "pending receipt", mutate: func(r *fakeRPC) { r.receipt = nil }, want: ErrPending},
		{name: "failed transaction", mutate: func(r *fakeRPC) { r.receipt.Status = "0x0" }, want: ErrFailed},
		{name: "unexpected status", mutate: func(r *fakeRPC) { r.receipt.Status = "0x2" }, want: ErrFailed},
		{name: "rpc finality unavailable", mutate: func(r *fakeRPC) { r.err["eth_getBlockByNumber"] = errors.New("unavailable") }, want: ErrRPCFinality},
		{name: "provider block disagrees", mutate: func(r *fakeRPC) { r.block.Hash = "0x" + strings.Repeat("c", 64) }, want: ErrNotCanonical},
		{name: "removed transfer", mutate: func(r *fakeRPC) { r.receipt.Logs[0].Removed = true }, want: ErrNotCanonical},
		{name: "no transfer to wallet", mutate: func(r *fakeRPC) {
			r.receipt.Logs[0] = transferLog(0, testFrom, "0x3333333333333333333333333333333333333333", "64")
		}, want: ErrNoTransfer},
		{name: "wrong token contract", mutate: func(r *fakeRPC) { r.receipt.Logs[0].Address = "0x4444444444444444444444444444444444444444" }, want: ErrNoTransfer},
		{name: "zero transfer", mutate: func(r *fakeRPC) { r.receipt.Logs[0] = transferLog(0, testFrom, testTo, "0") }, want: ErrNoTransfer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpc := validRPC()
			tt.mutate(rpc)
			_, err := testVerifier(rpc, time.Now()).VerifyWalletTransaction(context.Background(), testWallet(), testTx)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestVerifyWalletTransactionRejectsMalformedOrAmbiguousLogs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*fakeRPC)
	}{
		{name: "wrong topic count", mutate: func(r *fakeRPC) { r.receipt.Logs[0].Topics = r.receipt.Logs[0].Topics[:2] }},
		{name: "nonzero topic padding", mutate: func(r *fakeRPC) { r.receipt.Logs[0].Topics[2] = "0x01" + r.receipt.Logs[0].Topics[2][4:] }},
		{name: "malformed amount", mutate: func(r *fakeRPC) { r.receipt.Logs[0].Data = "0x01" }},
		{name: "duplicate log index", mutate: func(r *fakeRPC) {
			duplicate := transferLog(0, testFrom, testTo, "1")
			r.receipt.Logs = append(r.receipt.Logs, duplicate)
		}},
		{name: "log block hash mismatch", mutate: func(r *fakeRPC) { r.receipt.Logs[0].BlockHash = "0x" + strings.Repeat("c", 64) }},
		{name: "log transaction mismatch", mutate: func(r *fakeRPC) { r.receipt.Logs[0].TransactionHash = "0x" + strings.Repeat("c", 64) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpc := validRPC()
			tt.mutate(rpc)
			_, err := testVerifier(rpc, time.Now()).VerifyWalletTransaction(context.Background(), testWallet(), testTx)
			require.Error(t, err)
		})
	}
}

func TestVerifyWalletTransactionRequiresTreasuryBaseWallet(t *testing.T) {
	rpc := validRPC()
	v := testVerifier(rpc, time.Now())
	wallet := testWallet()
	wallet.Network = "eip155:84532"
	_, err := v.VerifyWalletTransaction(context.Background(), wallet, testTx)
	require.Error(t, err)
	require.Empty(t, rpc.methods)
}

func TestHTTPRPCAllowsOnlyLocalPlaintextAndReceiptMethods(t *testing.T) {
	_, err := NewHTTPRPC("http://example.com/rpc", nil)
	require.Error(t, err)
	_, err = NewHTTPRPC("https://user:pass@example.com/rpc", nil)
	require.Error(t, err)
	rpc, err := NewHTTPRPC("http://127.0.0.1:8545", nil)
	require.NoError(t, err)
	require.Error(t, rpc.Call(context.Background(), "eth_sendTransaction", []any{}, nil))
	require.Error(t, rpc.Call(context.Background(), "personal_sign", []any{}, nil))
}

func TestHTTPRPCRefusesRedirects(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x2105"}`))
	}))
	defer redirectTarget.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, nil, redirectTarget.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	rpc, err := NewHTTPRPC(redirector.URL, redirector.Client())
	require.NoError(t, err)
	var chainID string
	require.Error(t, rpc.Call(context.Background(), "eth_chainId", []any{}, &chainID))
	require.Empty(t, chainID)
}
