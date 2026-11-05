package baseusdc

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	walletpkg "treasury/internal/wallet"
)

type verificationStub struct {
	observations []Observation
	err          error
}

func (s verificationStub) VerifyWalletTransaction(context.Context, walletpkg.Wallet, string) ([]Observation, error) {
	return s.observations, s.err
}

func openReceiptStore(t *testing.T, path string) (*sql.DB, *SQLiteStore) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(50)")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, NewSQLiteStore(db)
}

func TestServiceRetainsExactReceiptObservationAcrossDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "receipts.sqlite")
	db, store := openReceiptStore(t, path)
	observedAt := time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	transfer := Observation{
		EconomicKey: "eip155:8453/erc20:" + USDCAddress + "/log/0", InclusionRevision: testTx + "/log/0@" + testBlock,
		WalletID: testWalletID, Network: Network, AssetID: TransferAsset, Decimals: 6,
		From: testFrom, To: testTo, AtomicAmount: "5901522", TransactionHash: testTx,
		LogIndex: 0, BlockNumber: 100, BlockHash: testBlock, BlockTime: observedAt.Add(-time.Minute),
		Finality: "included", FinalizedHeadNumber: 99, FinalizedHeadHash: "0x" + strings.Repeat("c", 64),
		ProviderID: "fixture-base-rpc", ObservedAt: observedAt,
	}
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	svc := NewService(store, verificationStub{observations: []Observation{transfer}}, "fixture-base-rpc", func() time.Time { return observedAt })
	check, err := svc.Verify(ctx, "operator-a", w, testTx)
	require.NoError(t, err)
	require.Equal(t, "verified", check.Outcome)
	require.NotEmpty(t, check.ID)
	require.NoError(t, db.Close())

	freshDB, freshStore := openReceiptStore(t, path)
	defer freshDB.Close()
	checks, err := freshStore.List(ctx, "operator-a", testWalletID, testTx, 10)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	require.Equal(t, check.ID, checks[0].ID)
	require.Equal(t, "5901522", checks[0].Transfers[0].AtomicAmount)
	require.Equal(t, "included", checks[0].Transfers[0].Finality)
	require.Equal(t, testBlock, checks[0].Transfers[0].BlockHash)

	checks, err = freshStore.List(ctx, "operator-b", testWalletID, testTx, 10)
	require.NoError(t, err)
	require.Empty(t, checks)
}

func TestServiceRetainsFailedCheckWithoutCreatingTransferEvidence(t *testing.T) {
	ctx := context.Background()
	_, store := openReceiptStore(t, filepath.Join(t.TempDir(), "receipts.sqlite"))
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	svc := NewService(store, verificationStub{err: ErrPending}, "fixture-base-rpc", func() time.Time { return time.Unix(10, 0).UTC() })
	_, err := svc.Verify(ctx, "operator-a", w, testTx)
	require.ErrorIs(t, err, ErrPending)

	checks, err := svc.List(ctx, "operator-a", w.ID, testTx, 10)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	require.Equal(t, "pending", checks[0].Outcome)
	require.Equal(t, "pending", checks[0].ErrorCode)
	require.Empty(t, checks[0].Transfers)
}

func TestServiceRetainsUnavailableProviderResult(t *testing.T) {
	ctx := context.Background()
	_, store := openReceiptStore(t, filepath.Join(t.TempDir(), "receipts.sqlite"))
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	svc := NewService(store, verificationStub{err: ErrRPCUnavailable}, "fixture-base-rpc", func() time.Time { return time.Unix(11, 0).UTC() })
	_, err := svc.Verify(ctx, "operator-a", w, testTx)
	require.ErrorIs(t, err, ErrRPCUnavailable)

	checks, err := svc.List(ctx, "operator-a", w.ID, testTx, 10)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	require.Equal(t, "provider_unavailable", checks[0].Outcome)
	require.Empty(t, checks[0].Transfers)
}

func TestServiceRefusesOtherOperatorAndInvalidHash(t *testing.T) {
	ctx := context.Background()
	_, store := openReceiptStore(t, filepath.Join(t.TempDir(), "receipts.sqlite"))
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	svc := NewService(store, verificationStub{}, "fixture-base-rpc", time.Now)
	_, err := svc.Verify(ctx, "operator-b", w, testTx)
	require.Error(t, err)
	_, err = svc.List(ctx, "operator-a", w.ID, "invalid", 10)
	require.Error(t, err)
}

func TestServiceReturnsRetentionFailureInsteadOfUnstoredSuccess(t *testing.T) {
	ctx := context.Background()
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	svc := NewService(failingStore{}, verificationStub{}, "fixture-base-rpc", time.Now)
	_, err := svc.Verify(ctx, "operator-a", w, testTx)
	require.ErrorIs(t, err, ErrPersist)
	require.False(t, errors.Is(err, ErrPending))
}

type failingStore struct{}

func (failingStore) Append(context.Context, Check) error { return errors.New("storage unavailable") }
func (failingStore) List(context.Context, string, string, string, int) ([]Check, error) {
	return nil, errors.New("storage unavailable")
}

func TestEveryRecheckAddsNewEvidenceInsteadOfReplacingPriorInclusion(t *testing.T) {
	ctx := context.Background()
	_, store := openReceiptStore(t, filepath.Join(t.TempDir(), "receipts.sqlite"))
	w := testWallet()
	w.OperatorIdentity = "operator-a"
	first := serviceObservation(testBlock, "included", 99)
	secondBlock := "0x" + "d" + strings.Repeat("0", 63)
	second := serviceObservation(secondBlock, "finalized", 100)
	verifier := &sequenceVerifier{results: [][]Observation{{first}, {second}}}
	checkedAt := time.Unix(0, 0).UTC()
	svc := NewService(store, verifier, "fixture-base-rpc", func() time.Time {
		checkedAt = checkedAt.Add(time.Second)
		return checkedAt
	})
	_, err := svc.Verify(ctx, "operator-a", w, testTx)
	require.NoError(t, err)
	_, err = svc.Verify(ctx, "operator-a", w, testTx)
	require.NoError(t, err)
	checks, err := svc.List(ctx, "operator-a", w.ID, testTx, 10)
	require.NoError(t, err)
	require.Len(t, checks, 2)
	require.Equal(t, second.BlockHash, checks[0].Transfers[0].BlockHash)
	require.Equal(t, first.BlockHash, checks[1].Transfers[0].BlockHash)
}

func serviceObservation(blockHash, finality string, head uint64) Observation {
	observedAt := time.Unix(1, 0).UTC()
	return Observation{
		EconomicKey: "stable-key", InclusionRevision: testTx + "/log/0@" + blockHash,
		WalletID: testWalletID, Network: Network, AssetID: TransferAsset, Decimals: USDCDecimals,
		From: testFrom, To: testTo, AtomicAmount: "1", TransactionHash: testTx,
		LogIndex: 0, BlockNumber: 100, BlockHash: blockHash, BlockTime: observedAt,
		Finality: finality, FinalizedHeadNumber: head, FinalizedHeadHash: "0x" + strings.Repeat("e", 64),
		ProviderID: "fixture-base-rpc", ObservedAt: observedAt,
	}
}

type sequenceVerifier struct {
	results [][]Observation
	index   int
}

func (s *sequenceVerifier) VerifyWalletTransaction(context.Context, walletpkg.Wallet, string) ([]Observation, error) {
	value := s.results[s.index]
	s.index++
	return value, nil
}
