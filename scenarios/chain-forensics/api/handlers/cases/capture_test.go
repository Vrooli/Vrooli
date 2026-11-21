package cases

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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

type fakeSolanaSignatureReader struct {
	page            forensics.CapturedSolanaPage
	pages           []forensics.CapturedSolanaPage
	err             error
	calls           []string
	address, before string
	limit           uint32
}

type fakeSolanaTransactionReader struct {
	capture   forensics.CapturedSolanaTransaction
	signature string
}

type fakeSolanaTokenAccountReader struct {
	capture        forensics.CapturedSolanaTokenAccountSnapshot
	owner, program string
}

func (f *fakeSolanaTokenAccountReader) FetchTokenAccountsByOwner(_ context.Context, owner, program string) (forensics.CapturedSolanaTokenAccountSnapshot, error) {
	f.owner, f.program = owner, program
	return f.capture, nil
}

func TestCaptureSolanaTokenAccountsPersistsTypedOwnerInventory(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-owner-test", Title: "Private SPL owner inventory", Private: true}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	owner := "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz"
	program := forensics.SPLTokenProgramID
	source := forensics.Source{ID: "owner-inventory-source", Name: "Solana finalized token accounts", URL: "https://rpc.example", Retrieved: "2026-11-16T12:00:00Z", SHA256: strings.Repeat("a", 64)}
	snapshot := forensics.SolanaTokenAccountSnapshot{ID: "owner-snapshot", Owner: owner, ProgramID: program, ContextSlot: 42, SourceID: source.ID, Accounts: []forensics.SolanaTokenAccount{{Address: "DHFFk4GpLvsTbfZnY361PzAq7ddzuJtWvGxY8t4X9Fqb", Mint: "9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj", Owner: owner, ProgramID: program, AtomicQuantity: "123456789", Scale: 6, ScaleKnown: true}}}
	reader := &fakeSolanaTokenAccountReader{capture: forensics.CapturedSolanaTokenAccountSnapshot{Snapshot: snapshot, Source: source}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithSolanaTokenAccountReader(reader)
	response, err := h.CaptureSolanaTokenAccounts(context.Background(), connect.NewRequest(&p.CaptureSolanaTokenAccountsRequest{CaseId: doc.ID, Owner: owner, ProgramId: program}))
	require.NoError(t, err)
	require.Equal(t, owner, reader.owner)
	require.Equal(t, program, reader.program)
	require.Equal(t, uint32(1), response.Msg.GetCapturedAccountCount())
	require.Equal(t, uint64(42), response.Msg.GetContextSlot())
	require.Equal(t, owner, response.Msg.GetCase().GetSolanaTokenAccountSnapshots()[0].GetOwner())
	require.Equal(t, "DHFFk4GpLvsTbfZnY361PzAq7ddzuJtWvGxY8t4X9Fqb", response.Msg.GetCase().GetSolanaTokenAccountSnapshots()[0].GetAccounts()[0].GetAddress())
	readback, err := repo.Get(context.Background(), doc.ID)
	require.NoError(t, err)
	require.Equal(t, snapshot, readback.Case.SolanaTokenAccountSnapshots[0])
}

func (f *fakeSolanaTransactionReader) FetchTransaction(_ context.Context, signature string) (forensics.CapturedSolanaTransaction, error) {
	f.signature = signature
	return f.capture, nil
}

func (f *fakeSolanaAddressReader) FetchAddressPage(_ context.Context, address, before string) (forensics.CapturedSolanaPage, error) {
	f.address, f.before = address, before
	return f.page, nil
}

func (f *fakeSolanaSignatureReader) FetchAddressSignatures(_ context.Context, address, before string, limit uint32) (forensics.CapturedSolanaPage, error) {
	f.address, f.before, f.limit = address, before, limit
	f.calls = append(f.calls, before)
	if len(f.pages) > 0 {
		page := f.pages[0]
		f.pages = f.pages[1:]
		return page, nil
	}
	if f.err != nil {
		return forensics.CapturedSolanaPage{}, f.err
	}
	return f.page, nil
}

func signaturePage(address, signature string, slot uint64) forensics.CapturedSolanaPage {
	result := []byte(fmt.Sprintf(`[{"signature":%q,"slot":%d,"err":null}]`, signature, slot))
	digest := sha256.Sum256(result)
	return forensics.CapturedSolanaPage{
		Address:    address,
		Signatures: []forensics.SolanaSignature{{Signature: signature, Slot: slot}},
		NextBefore: signature,
		Source: forensics.Source{
			ID: "signature-page-" + signature, Name: "Solana finalized signature page", URL: "https://api.mainnet.solana.com",
			Retrieved: "2026-11-16T12:00:00Z", SHA256: fmt.Sprintf("%x", digest), ResultBody: result,
			Citation: "getSignaturesForAddress(commitment=finalized)", Limitations: "Signature discovery only; no transfer inferred.",
		},
	}
}

func TestCaptureSolanaSignatureOnlyPagePersistsWithoutFetchingTransactions(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-signatures-test", Title: "Private signature discovery", Private: true}
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	address := "Bo9jh3wsmcC2AjakLWzNmKJ3SgtZmXEcSaW7L2FAvUsU"
	sig := strings.Repeat("1", 64)
	result := []byte(`[{"signature":"` + sig + `","slot":123,"blockTime":1740000000,"err":null}]`)
	source := forensics.Source{ID: "signature-page-source", Name: "Solana finalized signature page", URL: "https://api.mainnet.solana.com", Retrieved: "2026-11-16T12:00:00Z", SHA256: "5a9dbaf25b8abf87d2896a692e45513e8316c2b6a245f822ff35642633770546", Citation: "getSignaturesForAddress(commitment=finalized,limit=1000)", Limitations: "Signature discovery only; no transaction transfers inferred.", ResultBody: result}
	page := forensics.CapturedSolanaPage{Address: address, Signatures: []forensics.SolanaSignature{{Signature: sig, Slot: 123}}, NextBefore: sig, Source: source}
	reader := &fakeSolanaSignatureReader{page: page}
	h := NewHandler(repo, log.New(io.Discard, "", 0)).WithSolanaSignatureReader(reader)
	response, err := h.CaptureSolanaAddressPage(ctx, connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, SignaturesOnly: true}))
	require.NoError(t, err)
	require.Equal(t, address, reader.address)
	require.Empty(t, reader.before)
	require.Equal(t, uint32(1000), reader.limit)
	require.True(t, response.Msg.GetSignaturesOnly())
	require.Equal(t, uint32(1), response.Msg.GetCapturedSignatureCount())
	require.Equal(t, sig, response.Msg.GetNextBeforeSignature())
	require.Empty(t, response.Msg.GetCase().GetOccurrences())
	require.Len(t, response.Msg.GetCase().GetSources(), 1)
	require.Equal(t, source.ID, response.Msg.GetCase().GetSources()[0].GetId())
	readback, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, result, readback.Case.Sources[0].ResultBody)
}

func TestCaptureSolanaSignaturePagesBatchAndAppendOnce(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-signature-batch-test", Title: "Private batched signature discovery", Private: true}
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	address := "Bo9jh3wsmcC2AjakLWzNmKJ3SgtZmXEcSaW7L2FAvUsU"
	sig1, sig2 := strings.Repeat("1", 64), strings.Repeat("2", 64)
	reader := &fakeSolanaSignatureReader{pages: []forensics.CapturedSolanaPage{signaturePage(address, sig1, 123), signaturePage(address, sig2, 122)}}
	h := NewHandler(repo, log.New(io.Discard, "", 0)).WithSolanaSignatureReader(reader)
	response, err := h.CaptureSolanaAddressPage(ctx, connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, SignaturesOnly: true, SignatureLimit: 1, SignaturePageCount: 2}))
	require.NoError(t, err)
	require.Equal(t, []string{"", sig1}, reader.calls)
	require.Equal(t, uint32(2), response.Msg.GetCapturedPageCount())
	require.Equal(t, uint32(2), response.Msg.GetCapturedSignatureCount())
	require.Equal(t, sig2, response.Msg.GetNextBeforeSignature())
	require.Equal(t, doc.ID, response.Msg.GetCaseId())
	require.Nil(t, response.Msg.GetCase(), "multi-page mutation omits the full packet payload")
	readback, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Len(t, readback.Case.Sources, 2)
	require.Empty(t, readback.Case.Occurrences)
}

func TestCaptureSolanaSignaturePagesRetainsPartialBatchOnRateLimit(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-signature-partial-test", Title: "Private partial signature discovery", Private: true}
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	address := "Bo9jh3wsmcC2AjakLWzNmKJ3SgtZmXEcSaW7L2FAvUsU"
	sig := strings.Repeat("1", 64)
	reader := &fakeSolanaSignatureReader{pages: []forensics.CapturedSolanaPage{signaturePage(address, sig, 123)}, err: errors.New("Solana RPC returned HTTP 429; retry after 10s")}
	h := NewHandler(repo, log.New(io.Discard, "", 0)).WithSolanaSignatureReader(reader)
	response, err := h.CaptureSolanaAddressPage(ctx, connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, SignaturesOnly: true, SignatureLimit: 1, SignaturePageCount: 2}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.Msg.GetCapturedPageCount())
	require.Equal(t, sig, response.Msg.GetNextBeforeSignature())
	require.Contains(t, response.Msg.GetStoppedReason(), "429")
	readback, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Len(t, readback.Case.Sources, 1)
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
	occ := forensics.Occurrence{ID: "solana:" + sig + ":spl:1", Chain: forensics.ChainSolana, Transaction: sig, Block: 99, Timestamp: "2026-11-12T11:00:00Z", EventIndex: 1, Direction: "input", From: address, Account: address, Asset: "SPL:" + address, Atomic: "42", Scale: 6, Measurement: forensics.MeasurementNetBalanceDelta, Kind: forensics.EvidenceObservation, SourceID: txSource.ID}
	pump := forensics.PumpInstruction{ID: "solana:" + sig + ":pump:2", Transaction: sig, Slot: 99, Index: 2, ProgramID: "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", SourceID: txSource.ID, Kind: "create_v2", DecoderVersion: "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#create_v2", Mint: "11111111111111111111111111111112", User: address, Creator: address, Name: "Test Token", Symbol: "TST", MetadataURI: "https://metadata.example/token.json"}
	unresolved := forensics.UnresolvedPumpInstruction{ID: "solana:" + sig + ":pump:4", Transaction: sig, Slot: 99, Index: 4, ProgramID: pump.ProgramID, SourceID: txSource.ID, Discriminator: "c2ab1c46684d5b2f", Reason: "Pump discriminator is not in this decoder version"}
	page := forensics.CapturedSolanaPage{Address: address, Before: before, NextBefore: sig, Signatures: []forensics.SolanaSignature{{Signature: sig, Slot: 99}}, Source: pageSource, Transactions: []forensics.CapturedSolanaTransaction{{Source: txSource, Occurrences: []forensics.Occurrence{occ}, PumpInstructions: []forensics.PumpInstruction{pump}, UnresolvedPumpInstructions: []forensics.UnresolvedPumpInstruction{unresolved}}}}
	reader := &fakeSolanaAddressReader{page: page}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithSolanaReader(reader)
	response, err := h.CaptureSolanaAddressPage(context.Background(), connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, BeforeSignature: before}))
	require.NoError(t, err)
	require.Equal(t, address, reader.address)
	require.Equal(t, before, reader.before)
	require.Equal(t, uint32(1), response.Msg.CapturedTransactionCount)
	require.Equal(t, uint32(1), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, uint32(1), response.Msg.CapturedPumpInstructionCount)
	require.Equal(t, uint32(1), response.Msg.UnresolvedPumpInstructionCount)
	require.Equal(t, sig, response.Msg.NextBeforeSignature)
	require.Len(t, response.Msg.Case.Sources, 2)
	require.Equal(t, occ.ID, response.Msg.Case.Occurrences[0].Id)
	require.Equal(t, forensics.MeasurementNetBalanceDelta, response.Msg.Case.Occurrences[0].Measurement)
	require.Equal(t, occ.Account, response.Msg.Case.Occurrences[0].Account)
	require.Equal(t, pump.ID, response.Msg.Case.PumpInstructions[0].Id)
	require.Equal(t, pump.DecoderVersion, response.Msg.Case.PumpInstructions[0].DecoderVersion)
	require.Equal(t, unresolved.Discriminator, response.Msg.Case.UnresolvedPumpInstructions[0].Discriminator)
	// The SQLite document must preserve the typed records and replay must be idempotent.
	replay, err := h.CaptureSolanaAddressPage(context.Background(), connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, BeforeSignature: before}))
	require.NoError(t, err)
	require.Len(t, replay.Msg.Case.PumpInstructions, 1)
	require.Len(t, replay.Msg.Case.UnresolvedPumpInstructions, 1)
	packet, err := h.ExportPrivatePacket(context.Background(), connect.NewRequest(&p.ExportPrivatePacketRequest{Id: doc.ID}))
	require.NoError(t, err)
	var packetEnvelope struct {
		Case forensics.Case `json:"case"`
	}
	require.NoError(t, json.Unmarshal(packet.Msg.GetPacketJson(), &packetEnvelope))
	require.Equal(t, []forensics.PumpInstruction{pump}, packetEnvelope.Case.PumpInstructions)
	require.Equal(t, []forensics.UnresolvedPumpInstruction{unresolved}, packetEnvelope.Case.UnresolvedPumpInstructions)
}

func TestCaptureSolanaSignatureOnlyPagePersistsRawSourceAndNoTransfers(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-signature-only-test", Title: "Private history discovery", Private: true}
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	address := "Bo9jh3wsmcC2AjakLWzNmKJ3SgtZmXEcSaW7L2FAvUsU"
	sig := strings.Repeat("1", 64)
	resultBody := []byte(`[{"signature":"` + sig + `","slot":123,"blockTime":1740000000,"err":null}]`)
	source := forensics.Source{ID: "signature-source", Name: "Solana finalized signature page", URL: "https://api.mainnet.solana.com", Retrieved: "2026-11-16T12:00:00Z", SHA256: "5a9dbaf25b8abf87d2896a692e45513e8316c2b6a245f822ff35642633770546", Citation: "getSignaturesForAddress(finalized,limit=1000)", Limitations: "Signature discovery only; no transaction-level transfer evidence.", ResultBody: resultBody}
	reader := &fakeSolanaSignatureReader{page: forensics.CapturedSolanaPage{Address: address, Signatures: []forensics.SolanaSignature{{Signature: sig, Slot: 123}}, NextBefore: sig, Source: source}}
	h := NewHandler(repo, log.New(io.Discard, "", 0)).WithSolanaSignatureReader(reader)
	response, err := h.CaptureSolanaAddressPage(ctx, connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: doc.ID, Address: address, SignaturesOnly: true}))
	require.NoError(t, err)
	require.Equal(t, address, reader.address)
	require.Empty(t, reader.before)
	require.Equal(t, uint32(1000), reader.limit)
	require.True(t, response.Msg.GetSignaturesOnly())
	require.Equal(t, uint32(1), response.Msg.GetCapturedSignatureCount())
	require.Equal(t, sig, response.Msg.GetNextBeforeSignature())
	require.Empty(t, response.Msg.GetCase().GetOccurrences())
	require.Len(t, response.Msg.GetCase().GetSources(), 1)
	require.Equal(t, source.ID, response.Msg.GetCase().GetSources()[0].GetId())
	readback, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, resultBody, readback.Case.Sources[0].ResultBody)
}

func TestCaptureSolanaTransactionPersistsExactSignatureEvidence(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-signature-test", Title: "Private signature capture", Private: true, Gaps: []string{"Address-history selector is not retained."}}
	selectorBody := []byte(`[ {"signature":"4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk","slot":451779205} ]`)
	doc.Sources = []forensics.Source{{ID: "history-page-1", Name: "Solana finalized address signature page", URL: "https://rpc.example", Retrieved: "2026-11-16T04:00:00Z", SHA256: "88fa8c5c7a4b213af7b2e18fb85d40c43d481e9fa4a242948e65884f942eaec8", Citation: "getSignaturesForAddress(commitment=finalized,limit=1000)", ResultBody: selectorBody}}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	signature := "4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk"
	source := forensics.Source{ID: "solana-transaction-" + signature + "-digest", Name: "Solana finalized transaction", URL: "https://rpc.example", Retrieved: "2026-11-16T04:00:00Z", SHA256: strings.Repeat("a", 64), Limitations: "Caller-supplied signature; selector-page lineage is absent."}
	occ := forensics.Occurrence{ID: "solana:" + signature + ":spl:1", Chain: forensics.ChainSolana, Transaction: signature, Block: 451779205, Timestamp: "2026-11-14T00:00:00Z", EventIndex: 1, Direction: "input", From: "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz", Asset: "SPL:9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj", Atomic: "42", Scale: 6, Measurement: forensics.MeasurementNetBalanceDelta, Account: "4hQUbddZUnAWxPs8R2BQdEBvo7HoeyouEdSZRQQAkZPq", Kind: forensics.EvidenceObservation, SourceID: source.ID}
	pump := forensics.PumpInstruction{ID: "solana:" + signature + ":pump:2", Transaction: signature, Slot: 451779205, Index: 2, ProgramID: "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", SourceID: source.ID, Kind: "create_v2", DecoderVersion: "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#create_v2", Mint: "9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj", User: "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz", Creator: "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz", Name: "Super Intelligence", Symbol: "SI"}
	unresolved := forensics.UnresolvedPumpInstruction{ID: "solana:" + signature + ":pump:4:candidate:buy_exact_quote_in_v2", Transaction: signature, Slot: 451779205, Index: 4, ProgramID: pump.ProgramID, SourceID: source.ID, Discriminator: "c2ab1c46684d5b2f", Reason: "Pinned IDL candidate has a historical data-length mismatch", CandidateKind: "buy_exact_quote_in_v2", DecoderVersion: "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#buy_exact_quote_in_v2", ExpectedDataLength: 25, ObservedDataLength: 24}
	reader := &fakeSolanaTransactionReader{capture: forensics.CapturedSolanaTransaction{Source: source, Occurrences: []forensics.Occurrence{occ}, PumpInstructions: []forensics.PumpInstruction{pump}, UnresolvedPumpInstructions: []forensics.UnresolvedPumpInstruction{unresolved}}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithSolanaTransactionReader(reader)
	response, err := h.CaptureSolanaTransaction(context.Background(), connect.NewRequest(&p.CaptureSolanaTransactionRequest{CaseId: doc.ID, Signature: signature, SelectionSourceId: "history-page-1"}))
	require.NoError(t, err)
	require.Equal(t, signature, reader.signature)
	require.Equal(t, "history-page-1", response.Msg.SelectionSourceId)
	require.Equal(t, uint32(1), response.Msg.CapturedOccurrenceCount)
	require.Equal(t, uint32(1), response.Msg.CapturedPumpInstructionCount)
	require.Equal(t, uint32(1), response.Msg.UnresolvedPumpInstructionCount)
	require.False(t, response.Msg.TransactionFailed)
	require.Equal(t, occ.ID, response.Msg.Case.Occurrences[0].Id)
	require.Equal(t, forensics.MeasurementNetBalanceDelta, response.Msg.Case.Occurrences[0].Measurement)
	require.Equal(t, occ.Account, response.Msg.Case.Occurrences[0].Account)
	require.Equal(t, pump.ID, response.Msg.Case.PumpInstructions[0].Id)
	require.Equal(t, unresolved.Discriminator, response.Msg.Case.UnresolvedPumpInstructions[0].Discriminator)
	require.Equal(t, unresolved.CandidateKind, response.Msg.Case.UnresolvedPumpInstructions[0].CandidateKind)
	require.Equal(t, unresolved.ExpectedDataLength, response.Msg.Case.UnresolvedPumpInstructions[0].ExpectedDataLength)
	require.Equal(t, unresolved.ObservedDataLength, response.Msg.Case.UnresolvedPumpInstructions[0].ObservedDataLength)
	selected := response.Msg.Case.Sources[1]
	require.Equal(t, "history-page-1", selected.SelectedBySourceId)
	require.Equal(t, source.SHA256, selected.Sha256)
	require.Equal(t, source.ResultBody, selected.ResultBody)
	// Replaying the same caller-supplied signature is idempotent at the owner repository.
	replay, err := h.CaptureSolanaTransaction(context.Background(), connect.NewRequest(&p.CaptureSolanaTransactionRequest{CaseId: doc.ID, Signature: signature}))
	require.NoError(t, err)
	require.Len(t, replay.Msg.Case.Sources, 3)
	require.Len(t, replay.Msg.Case.Occurrences, 1)
}

func TestCaptureSolanaTransactionRejectsSignatureNotInSelectedPage(t *testing.T) {
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(context.Background(), internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)))
	doc := forensics.Case{ID: "solana-selection-mismatch", Title: "Private selector validation", Private: true, Sources: []forensics.Source{{ID: "history", Name: "Solana finalized address signature page", URL: "https://rpc.example", Retrieved: "2026-11-16T04:00:00Z", SHA256: "d00ebf13ac4e14f3ae294726c5eb6ea6e8dc6e97f1cb32009becf343ecf8b7a7", Citation: "getSignaturesForAddress(commitment=finalized,limit=1000)", ResultBody: []byte(`[{"signature":"different-signature"}]`)}}}
	_, err = repo.Import(context.Background(), doc)
	require.NoError(t, err)
	reader := &fakeSolanaTransactionReader{capture: forensics.CapturedSolanaTransaction{Source: forensics.Source{ID: "tx", Name: "tx"}}}
	h := NewHandlerWithReaders(repo, log.New(io.Discard, "", 0), nil, nil).WithSolanaTransactionReader(reader)
	_, err = h.CaptureSolanaTransaction(context.Background(), connect.NewRequest(&p.CaptureSolanaTransactionRequest{CaseId: doc.ID, Signature: "requested-signature", SelectionSourceId: "history"}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not present in selected retained history source")
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
