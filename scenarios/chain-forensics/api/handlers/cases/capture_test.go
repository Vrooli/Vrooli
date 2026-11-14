package cases

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	internalcases "chain-forensics/internal/cases"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	forensics "github.com/vrooli/chain-forensics"
	p "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
)

type fakeEVMReader struct {
	result forensics.CapturedEVMTransaction
	chain  forensics.Chain
	hash   string
}

type fakeBitcoinReader struct {
	result forensics.CapturedTransaction
	txid   string
}

type fakeBitcoinAddressReader struct {
	page    forensics.CapturedBitcoinAddressPage
	address string
	cursor  string
}

type fakeEVMWalletReader struct {
	result  forensics.CapturedEVMWalletSnapshot
	chain   forensics.Chain
	address string
	tokens  []string
}

type fakeSolanaAddressReader struct {
	page            forensics.CapturedSolanaPage
	address, before string
}

func (f *fakeSolanaAddressReader) FetchAddressPage(_ context.Context, address, before string) (forensics.CapturedSolanaPage, error) {
	f.address, f.before = address, before
	return f.page, nil
}

func TestCaptureSolanaAddressPagePersistsFinalizedSourcesAndTokenDeltas(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-page-test", Title: "Private Solana case", Private: true, Gaps: []string{"Only the first page is captured."}}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	address, before, sig := "11111111111111111111111111111111", strings.Repeat("1", 64), strings.Repeat("1", 64)
	pageSource := forensics.Source{ID: "sol-page", Name: "Solana finalized signatures", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("a", 64)}
	txSource := forensics.Source{ID: "sol-tx", Name: "Solana finalized transaction", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("b", 64)}
	occ := forensics.Occurrence{ID: "solana:" + sig + ":spl:1", Chain: forensics.ChainSolana, Transaction: sig, Block: 99, Timestamp: "2026-11-12T11:00:00Z", EventIndex: 1, Direction: "input", From: address, Asset: "SPL:" + address, Atomic: "42", Scale: 6, Kind: forensics.EvidenceObservation, SourceID: txSource.ID}
	page := forensics.CapturedSolanaPage{Address: address, Before: before, NextBefore: sig, Signatures: []forensics.SolanaSignature{{Signature: sig, Slot: 99}}, Source: pageSource, Transactions: []forensics.CapturedSolanaTransaction{{Source: txSource, Occurrences: []forensics.Occurrence{occ}}}}
	reader := &fakeSolanaAddressReader{page: page}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithSolanaReader(reader)
	response, err := h.CaptureSolanaAddressPage(context.Background(), connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, BeforeSignature: before}))
	require.NoError(t, err)
	require.Equal(t, address, reader.address)
	require.Equal(t, before, reader.before)
	require.Equal(t, uint32(1), response.Msg.CapturedTransactionCount)
	require.Equal(t, uint32(1), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, sig, response.Msg.NextBeforeSignature)
	require.Len(t, response.Msg.Case.Sources, 2)
	require.Equal(t, occ.ID, response.Msg.Case.Occurrences[0].Id)
}

func (f *fakeEVMWalletReader) FetchWalletSnapshot(_ context.Context, chain forensics.Chain, address string, tokens []string) (forensics.CapturedEVMWalletSnapshot, error) {
	f.chain, f.address, f.tokens = chain, address, append([]string(nil), tokens...)
	return f.result, nil
}

func TestCaptureEVMWalletSnapshotPersistsPinnedBalancesAndProvenance(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "wallet-snapshot-test", Title: "Private Treasury fork wallets", Private: true, Gaps: []string{"Snapshot is pinned to one confirmed block."}}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	address, contract, blockHash := "0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222", "0x"+strings.Repeat("a", 64)
	sources := []forensics.Source{}
	for _, id := range []string{"wallet-block", "wallet-native", "wallet-token", "wallet-decimals", "wallet-check"} {
		sources = append(sources, forensics.Source{ID: id, Name: "Pinned EVM JSON-RPC", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("a", 64)})
	}
	snapshot := forensics.EVMWalletSnapshot{ID: "snapshot", Chain: forensics.ChainBase, Address: address, Block: 31, BlockHash: blockHash, Timestamp: "2026-11-12T12:00:00Z", NativeAsset: "ETH", NativeAtomic: "0", NativeScale: 18, Tokens: []forensics.WalletTokenBalance{{Contract: contract, Atomic: "42", Scale: 6, ScaleKnown: true}}, SourceIDs: []string{"wallet-block", "wallet-native", "wallet-token", "wallet-decimals", "wallet-check"}}
	reader := &fakeEVMWalletReader{result: forensics.CapturedEVMWalletSnapshot{Snapshot: snapshot, Sources: sources}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithWalletReaders(map[forensics.Chain]EVMWalletReader{forensics.ChainBase: reader})
	response, err := h.CaptureEVMWalletSnapshot(context.Background(), connect.NewRequest(&p.CaptureEVMWalletSnapshotRequest{CaseId: doc.ID, Chain: p.Chain_CHAIN_BASE, Address: address, TokenContracts: []string{contract}}))
	require.NoError(t, err)
	require.Equal(t, forensics.ChainBase, reader.chain)
	require.Equal(t, address, reader.address)
	require.Equal(t, []string{contract}, reader.tokens)
	require.Len(t, response.Msg.Case.WalletSnapshots, 1)
	require.Equal(t, "42", response.Msg.Case.WalletSnapshots[0].Tokens[0].AtomicQuantity)
	require.Equal(t, []string{"wallet-block", "wallet-native", "wallet-token", "wallet-decimals", "wallet-check"}, response.Msg.Case.WalletSnapshots[0].SourceIds)
}

func (f *fakeBitcoinAddressReader) FetchAddressPage(_ context.Context, address, cursor string) (forensics.CapturedBitcoinAddressPage, error) {
	f.address, f.cursor = address, cursor
	return f.page, nil
}

func (f *fakeBitcoinReader) FetchTransaction(_ context.Context, txid string) (forensics.CapturedTransaction, error) {
	f.txid = txid
	return f.result, nil
}

func TestCaptureBitcoinAddressPagePersistsPageAndReturnsCursor(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := internalcases.NewSQLiteRepository(handle, clock)
	doc := forensics.Case{ID: "address-page-test", Title: "Private Bitcoin page", Private: true, Gaps: []string{"Only one confirmed page was captured."}}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	txid := strings.Repeat("c", 64)
	tx := forensics.MempoolTransaction{TxID: txid}
	tx.Status.Confirmed = true
	tx.Status.BlockHeight = 42
	tx.Status.BlockTime = 1700000000
	tx.Outputs = []struct {
		Address string `json:"scriptpubkey_address"`
		Value   uint64 `json:"value"`
	}{{Address: "1BoatSLRHtKNngkdXEeobR76b53LETtpyT", Value: 100}}
	source := forensics.Source{ID: "address-page-source", Name: "mempool.space REST", URL: "https://mempool.space/api/address/1BoatSLRHtKNngkdXEeobR76b53LETtpyT/txs/chain", Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("a", 64)}
	reader := &fakeBitcoinAddressReader{page: forensics.CapturedBitcoinAddressPage{Address: "1BoatSLRHtKNngkdXEeobR76b53LETtpyT", AfterTxID: strings.Repeat("b", 64), NextAfterTxID: txid, Transactions: []forensics.MempoolTransaction{tx}, Source: source}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil, reader)
	response, err := h.CaptureBitcoinAddressPage(context.Background(), connect.NewRequest(&p.CaptureBitcoinAddressPageRequest{CaseId: doc.ID, Address: reader.page.Address, AfterTxid: reader.page.AfterTxID}))
	require.NoError(t, err)
	require.Equal(t, reader.page.Address, reader.address)
	require.Equal(t, reader.page.AfterTxID, reader.cursor)
	require.Equal(t, uint32(1), response.Msg.CapturedTransactionCount)
	require.Equal(t, uint32(1), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, txid, response.Msg.NextAfterTxid)
	require.Equal(t, source.ID, response.Msg.Case.Sources[0].Id)
}

func (f *fakeEVMReader) FetchERC20Transfers(_ context.Context, chain forensics.Chain, hash string) (forensics.CapturedEVMTransaction, error) {
	f.chain = chain
	f.hash = hash
	return f.result, nil
}

func TestCaptureBitcoinTransactionPersistsConfirmedInputsAndOutputs(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := internalcases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	previousSource := forensics.Source{ID: "previous-outpoint-source", Name: "test previous output", URL: "https://mempool.space/api/tx/" + strings.Repeat("b", 64), Retrieved: "2026-11-12T11:59:00Z", SHA256: strings.Repeat("b", 64)}
	doc.Sources = append(doc.Sources, previousSource)
	doc.Occurrences = append(doc.Occurrences, forensics.Occurrence{ID: "bitcoin:" + strings.Repeat("b", 64) + ":out:1", Chain: forensics.ChainBitcoin, Transaction: strings.Repeat("b", 64), Block: 41, Timestamp: "2026-11-12T11:59:00Z", EventIndex: 1, Direction: "output", To: "bc1qsource", Asset: "BTC", Atomic: "123456789", Scale: 8, Kind: forensics.EvidenceObservation, SourceID: previousSource.ID})
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	txid := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tx := forensics.MempoolTransaction{TxID: txid}
	tx.Status.Confirmed = true
	tx.Status.BlockHeight = 42
	tx.Status.BlockTime = 1700000000
	tx.Inputs = []struct {
		Prevout struct {
			Address string `json:"scriptpubkey_address"`
			Value   uint64 `json:"value"`
		} `json:"prevout"`
		PreviousTx    string `json:"txid"`
		PreviousIndex uint32 `json:"vout"`
	}{{PreviousTx: strings.Repeat("b", 64), PreviousIndex: 1}}
	tx.Inputs[0].Prevout.Address = "bc1qsource"
	tx.Inputs[0].Prevout.Value = 123456789
	tx.Outputs = []struct {
		Address string `json:"scriptpubkey_address"`
		Value   uint64 `json:"value"`
	}{{Address: "bc1qdestination", Value: 123456000}}
	source := forensics.Source{ID: "bitcoin-capture-test", Name: "test public mempool", URL: "https://mempool.space/api/tx/" + txid, Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("a", 64)}
	reader := &fakeBitcoinReader{result: forensics.CapturedTransaction{Transaction: tx, Source: source}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, reader)
	response, err := h.CaptureBitcoinTransaction(context.Background(), connect.NewRequest(&p.CaptureBitcoinTransactionRequest{CaseId: doc.ID, TransactionId: txid}))
	require.NoError(t, err)
	require.Equal(t, uint32(2), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, txid, reader.txid)
	require.Equal(t, "123456789", response.Msg.Case.Occurrences[len(response.Msg.Case.Occurrences)-2].AtomicQuantity)
	require.Equal(t, "123456000", response.Msg.Case.Occurrences[len(response.Msg.Case.Occurrences)-1].AtomicQuantity)
	require.Contains(t, response.Msg.Case.Edges, &p.FlowEdge{Id: "outpoint:" + strings.Repeat("b", 64) + ":1->" + txid + ":0", FromOccurrence: "bitcoin:" + strings.Repeat("b", 64) + ":out:1", ToOccurrence: "bitcoin:" + txid + ":in:0", Grade: p.EvidenceGrade_EVIDENCE_GRADE_OBSERVED, Rule: "bitcoin.input.previous_outpoint_exact_match", EvidenceIds: []string{"bitcoin:" + strings.Repeat("b", 64) + ":out:1", "bitcoin:" + txid + ":in:0"}})
}

func TestCaptureEVMTransactionPersistsOnlyValidatedSourceAndOccurrence(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := internalcases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	source := forensics.Source{ID: "capture-test", Name: "test public RPC", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	occ := forensics.Occurrence{ID: "base:" + hash + ":log:1", Chain: forensics.ChainBase, Transaction: hash, Block: 1, Timestamp: "2026-11-12T12:00:00Z", EventIndex: 1, Direction: "transfer", From: "0x1111111111111111111111111111111111111111", To: "0x2222222222222222222222222222222222222222", Asset: "erc20:0x3333333333333333333333333333333333333333", Atomic: "5", Kind: forensics.EvidenceObservation, SourceID: source.ID}
	reader := &fakeEVMReader{result: forensics.CapturedEVMTransaction{Chain: forensics.ChainBase, TransactionHash: hash, Sources: []forensics.Source{source}, Occurrences: []forensics.Occurrence{occ}}}
	h := NewHandler(repo, log.New(io.Discard, "", 0), map[forensics.Chain]EVMReader{forensics.ChainBase: reader})
	response, err := h.CaptureEVMTransaction(context.Background(), connect.NewRequest(&p.CaptureEVMTransactionRequest{CaseId: doc.ID, Chain: p.Chain_CHAIN_BASE, TransactionHash: hash}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, "capture-test", response.Msg.Case.Sources[len(response.Msg.Case.Sources)-1].Id)
	require.Equal(t, hash, response.Msg.Case.Occurrences[len(response.Msg.Case.Occurrences)-1].TransactionId)
	require.Equal(t, forensics.ChainBase, reader.chain)
	require.Equal(t, hash, reader.hash)
}
