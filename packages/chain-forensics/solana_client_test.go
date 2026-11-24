package chainforensics

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSolanaRPCPathDecodesPinnedPumpCreateV2Capture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "cf6-pump-createv2-20261114", "getTransaction-response.json")
	responseBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	fixtureHash := sha256.Sum256(responseBody)
	if fmt.Sprintf("%x", fixtureHash) != "9970f16e3e88579f23de73b6aad6780e480bce9288577e2467300d0e50894da2" {
		t.Fatal("retained finalized transaction fixture changed")
	}
	const signature = "4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case "getSignaturesForAddress":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[{"signature":%q,"slot":451779205,"err":null,"blockTime":1756954400}]}`, signature)
		case "getTransaction":
			_, _ = w.Write(responseBody)
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.FetchAddressPage(context.Background(), "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Transactions) != 1 {
		t.Fatalf("expected one transaction, got %d", len(page.Transactions))
	}
	got := page.Transactions[0]
	if got.Failed || len(got.PumpInstructions) != 1 || len(got.UnresolvedPumpInstructions) != 1 {
		t.Fatalf("Pump decode status: failed=%v decoded=%+v unresolved=%+v", got.Failed, got.PumpInstructions, got.UnresolvedPumpInstructions)
	}
	ix := got.PumpInstructions[0]
	if ix.Kind != "create_v2" || ix.SourceID != got.Source.ID || ix.DecoderVersion != pumpCreateV2DecoderVersion || ix.Mint != "9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj" || ix.User != "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz" || ix.Creator != "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz" || ix.Name != "Super Intelligence" || ix.Symbol != "SI" || ix.MayhemMode || ix.CashbackEnabled || ix.CreatorFeeBPS != 0 || ix.HolderReward || !strings.HasPrefix(ix.MetadataURI, "https://") {
		t.Fatalf("unexpected IDL-decoded launch instruction: %+v", ix)
	}
	if got.Source.RawResponseSHA256 != fmt.Sprintf("%x", fixtureHash) || len(got.Source.RawResponseBody) != len(responseBody) {
		t.Fatal("exact RPC envelope evidence was not retained")
	}
	unknown := got.UnresolvedPumpInstructions[0]
	if unknown.Index != 4 || unknown.Discriminator != pumpBuyExactQuoteInV2Discriminator || unknown.CandidateKind != "buy_exact_quote_in_v2" || unknown.DecoderVersion != pumpBuyExactQuoteInV2DecoderVersion || unknown.ExpectedDataLength != 25 || unknown.ObservedDataLength != 24 || !strings.Contains(unknown.Reason, "Historical deployed layout is unverified") {
		t.Fatalf("recognized current IDL candidate with a mismatched historical layout should stay unresolved: %+v", unknown)
	}
	if err := Validate(Case{ID: "pump-fixture", Title: "Private Pump instruction capture", Private: true, Sources: []Source{page.Source, got.Source}, PumpInstructions: got.PumpInstructions, UnresolvedPumpInstructions: got.UnresolvedPumpInstructions}); err != nil {
		t.Fatalf("captured Pump records failed source-linked case validation: %v", err)
	}
}

func TestSolanaSignatureOnlyPageIsBoundedCachedAndDoesNotFetchTransactions(t *testing.T) {
	const address = "Bo9jh3wsmcC2AjakLWzNmKJ3SgtZmXEcSaW7L2FAvUsU"
	const before = "4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk"
	const signature = "1111111111111111111111111111111111111111111111111111111111111111"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "getSignaturesForAddress" || len(request.Params) != 2 || request.Params[0] != address {
			t.Errorf("unexpected signature-page request: %+v", request)
		}
		options, _ := request.Params[1].(map[string]any)
		if options["commitment"] != "finalized" || options["limit"] != float64(1000) || options["before"] != before {
			t.Errorf("request did not preserve bounded finalized cursor parameters: %+v", options)
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[{"signature":%q,"slot":123,"err":null,"blockTime":1740000000}]}`, signature)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.FetchAddressSignatures(context.Background(), address, before, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Signatures) != 1 || page.Signatures[0].Signature != signature || page.NextBefore != signature || len(page.Transactions) != 0 {
		t.Fatalf("unexpected signature-only page: %+v", page)
	}
	if page.Source.URL != server.URL || page.Source.RawResponseSHA256 == "" || len(page.Source.RawResponseBody) == 0 || string(page.Source.ResultBody) != "[{\"signature\":\""+signature+"\",\"slot\":123,\"err\":null,\"blockTime\":1740000000}]" {
		t.Fatalf("exact source evidence was not retained: %+v", page.Source)
	}
	if !strings.Contains(page.Source.Citation, "limit=1000") || !strings.Contains(page.Source.Citation, "before="+before) || !strings.Contains(page.Source.Limitations, "no transaction-level transfer evidence") {
		t.Fatalf("signature-page scope/limitations are not explicit: %+v", page.Source)
	}
	if _, err := client.FetchAddressSignatures(context.Background(), address, before, 1000); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("identical bounded page was not served from shared capture cache; calls=%d", calls.Load())
	}
	if _, err := client.FetchAddressSignatures(context.Background(), address, before, 1001); err == nil {
		t.Fatal("signature-only page limit above the RPC maximum should fail")
	}
}

func TestSolanaTokenAccountsByOwnerRetainsTypedFinalizedInventory(t *testing.T) {
	const owner = "AV7PjXHL5JXZ1YoYRoN9Dsstg1x2UciBupMCXcJP8gUz"
	const tokenAccount = "DHFFk4GpLvsTbfZnY361PzAq7ddzuJtWvGxY8t4X9Fqb"
	const mint = "9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "getTokenAccountsByOwner" || len(request.Params) != 3 || request.Params[0] != owner {
			t.Errorf("unexpected owner inventory request: %+v", request)
		}
		filter, _ := request.Params[1].(map[string]any)
		config, _ := request.Params[2].(map[string]any)
		if filter["programId"] != SPLTokenProgramID || config["commitment"] != "finalized" || config["encoding"] != "jsonParsed" {
			t.Errorf("request did not pin the intended finalized token program: filter=%v config=%v", filter, config)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":470097167},"value":[{"pubkey":%q,"account":{"owner":%q,"data":{"program":"spl-token","parsed":{"info":{"mint":%q,"owner":%q,"tokenAmount":{"amount":"123456789","decimals":6}}}}}}]}}`, tokenAccount, SPLTokenProgramID, mint, owner)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := client.FetchTokenAccountsByOwner(context.Background(), owner, SPLTokenProgramID)
	if err != nil {
		t.Fatal(err)
	}
	if captured.Snapshot.ContextSlot != 470097167 || captured.Snapshot.Owner != owner || len(captured.Snapshot.Accounts) != 1 {
		t.Fatalf("unexpected snapshot identity: %+v", captured.Snapshot)
	}
	account := captured.Snapshot.Accounts[0]
	if account.Address != tokenAccount || account.Owner != owner || account.Mint != mint || account.ProgramID != SPLTokenProgramID || account.AtomicQuantity != "123456789" || account.Scale != 6 || !account.ScaleKnown {
		t.Fatalf("unexpected typed token account: %+v", account)
	}
	if captured.Source.SHA256 == "" || captured.Source.RawResponseSHA256 == "" || len(captured.Source.RawResponseBody) == 0 {
		t.Fatal("exact RPC result or HTTP envelope was not retained")
	}
	caseFile := Case{ID: "solana-owner-inventory", Title: "Private SPL owner inventory", Private: true, Sources: []Source{captured.Source}, SolanaTokenAccountSnapshots: []SolanaTokenAccountSnapshot{captured.Snapshot}}
	if err := Validate(caseFile); err != nil {
		t.Fatalf("typed owner inventory failed case validation: %v", err)
	}
}

func TestSolanaRPCFetchesKnownFinalizedSignatureWithoutAddressEnumeration(t *testing.T) {
	const signature = "4orDLftSdVP3Y1AmygWzTxkEZgTU9M5iSfJskmbp2H8JkvXFgdhYtbfup9iCQj2WajGbheNdTGftegEf782Jwxwk"
	responseBody, err := os.ReadFile(filepath.Join("testdata", "cf6-pump-createv2-20261114", "getTransaction-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests.Add(1)
		method, _ := request["method"].(string)
		if method != "getTransaction" {
			t.Errorf("known-signature capture must not enumerate address history; got %s", method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := client.FetchTransaction(context.Background(), signature)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || captured.Transaction.Transaction.Signatures[0] != signature {
		t.Fatalf("expected exactly one exact finalized-signature read: requests=%d signature=%q", requests.Load(), captured.Transaction.Transaction.Signatures[0])
	}
	if len(captured.PumpInstructions) != 1 || len(captured.UnresolvedPumpInstructions) != 1 {
		t.Fatalf("known-signature capture lost supported and unresolved Pump records: decoded=%d unresolved=%d", len(captured.PumpInstructions), len(captured.UnresolvedPumpInstructions))
	}
	var native, spl, exactSPL, exactSOL int
	for _, occurrence := range captured.Occurrences {
		if occurrence.Measurement == MeasurementInstructionTransfer {
			if occurrence.Direction != "transfer" || occurrence.From == "" || occurrence.To == "" || occurrence.Account != occurrence.From {
				t.Fatalf("real retained response has malformed exact instruction transfer: %+v", occurrence)
			}
			switch occurrence.Asset {
			case "SPL:9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj":
				exactSPL++
				if occurrence.Atomic != "176989690673615" || occurrence.Scale != 6 || occurrence.To != "DHFFk4GpLvsTbfZnY361PzAq7ddzuJtWvGxY8t4X9Fqb" {
					t.Fatalf("unexpected real checked SPL transfer: %+v", occurrence)
				}
			case "SOL":
				exactSOL++
			}
			continue
		}
		if occurrence.Measurement != MeasurementNetBalanceDelta || occurrence.Account == "" {
			t.Fatalf("real retained response lacks typed account-delta semantics: %+v", occurrence)
		}
		switch occurrence.Asset {
		case "SOL":
			native++
			if occurrence.From != "" || occurrence.To != "" {
				t.Fatalf("raw native balance delta was misrepresented as a transfer: %+v", occurrence)
			}
		case "SPL:9aqmJjCnnMQv42TXLk921ceUkN35nea2QP969n1caqjj":
			spl++
		}
	}
	if native == 0 || spl == 0 || exactSPL != 1 || exactSOL != 6 {
		t.Fatalf("retained real Pump tx did not distinguish balance deltas from instruction transfers: native=%d spl=%d exact SPL=%d exact SOL=%d", native, spl, exactSPL, exactSOL)
	}
	if !strings.HasPrefix(captured.Source.ID, "solana-transaction-"+signature+"-") || captured.Source.RawResponseSHA256 == "" {
		t.Fatalf("exact provider source identity/envelope not retained: %+v", captured.Source)
	}
	if !strings.Contains(captured.Source.Limitations, "does not include the address-history page") {
		t.Fatalf("caller-supplied signature provenance boundary was lost: %s", captured.Source.Limitations)
	}
	if captured.PumpInstructions[0].SourceID != captured.Source.ID || captured.UnresolvedPumpInstructions[0].SourceID != captured.Source.ID {
		t.Fatalf("decoded Pump records do not bind the exact transaction source: %+v %+v", captured.PumpInstructions[0], captured.UnresolvedPumpInstructions[0])
	}
	if err := Validate(Case{ID: "known-signature-capture", Title: "Private exact-signature capture", Private: true, Sources: []Source{captured.Source}, PumpInstructions: captured.PumpInstructions, UnresolvedPumpInstructions: captured.UnresolvedPumpInstructions}); err != nil {
		t.Fatalf("known-signature capture is not a valid source-bound case: %v", err)
	}
}

func TestPumpDecoderLeavesUnknownAndWrongProgramUnresolvedOrIgnored(t *testing.T) {
	unknown := SolanaInstruction{ProgramID: pumpProgramID, Accounts: []string{"11111111111111111111111111111111", "11111111111111111111111111111111", "11111111111111111111111111111111", "11111111111111111111111111111111", "11111111111111111111111111111111", "11111111111111111111111111111111"}, Data: "111111111111"}
	decoded, unresolved := decodePumpInstructions([]SolanaInstruction{unknown})
	if len(decoded) != 0 || len(unresolved) != 1 || unresolved[0].Discriminator == "" {
		t.Fatalf("unknown Pump instruction was not retained unresolved: %+v %+v", decoded, unresolved)
	}
	unknown.ProgramID = "11111111111111111111111111111111"
	decoded, unresolved = decodePumpInstructions([]SolanaInstruction{unknown})
	if len(decoded) != 0 || len(unresolved) != 0 {
		t.Fatalf("non-Pump program was classified as Pump: %+v %+v", decoded, unresolved)
	}
}

func TestSolanaAddressPageCapturesFinalizedSPLDeltasAndCaches(t *testing.T) {
	signature, address, owner, mint := strings.Repeat("1", 64), strings.Repeat("1", 32), "11111111111111111111111111111112", "11111111111111111111111111111113"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case "getSignaturesForAddress":
			var cfg map[string]any
			_ = json.Unmarshal(request.Params[1], &cfg)
			if cfg["commitment"] != "finalized" || cfg["limit"] != float64(25) {
				t.Errorf("unexpected bounded finalized request: %#v", cfg)
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[{"signature":%q,"slot":99,"err":null,"blockTime":1790000000}]}`, signature)
		case "getTransaction":
			var config map[string]any
			if err := json.Unmarshal(request.Params[1], &config); err != nil {
				t.Error(err)
			}
			if config["encoding"] != "json" || config["commitment"] != "finalized" || config["maxSupportedTransactionVersion"] != float64(1) {
				t.Errorf("unexpected finalized transaction config: %#v", config)
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":2,"result":{"slot":99,"blockTime":1790000000,"transaction":{"signatures":[%q],"message":{"accountKeys":[%q,%q]}},"meta":{"err":null,"fee":5000,"preBalances":[10000,2000],"postBalances":[4500,2000],"preTokenBalances":[{"accountIndex":1,"mint":%q,"owner":%q,"uiTokenAmount":{"amount":"10","decimals":6}}],"postTokenBalances":[{"accountIndex":1,"mint":%q,"owner":%q,"uiTokenAmount":{"amount":"4","decimals":6}}]}}}`, signature, address, owner, mint, owner, mint, owner)
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Signatures) != 1 || page.NextBefore != signature || len(page.Transactions) != 1 || page.Source.SHA256 == "" || page.Source.RawResponseSHA256 == "" || len(page.Source.RawResponseBody) == 0 || page.Transactions[0].Source.RawResponseSHA256 == "" || len(page.Transactions[0].Source.RawResponseBody) == 0 {
		t.Fatalf("incomplete page: %+v", page)
	}
	captured := page.Transactions[0]
	if len(captured.Occurrences) != 2 {
		t.Fatalf("expected one SOL and one SPL delta, got %d", len(captured.Occurrences))
	}
	occ := captured.Occurrences[1]
	if occ.Direction != "input" || occ.From != owner || occ.Asset != "SPL:"+mint || occ.Atomic != "6" || occ.Scale != 6 || occ.Block != 99 {
		t.Fatalf("unexpected SPL balance delta: %+v", occ)
	}
	if occ.Measurement != MeasurementNetBalanceDelta || occ.Account != owner {
		t.Fatalf("SPL delta lacks its measured-account semantics: %+v", occ)
	}
	native := captured.Occurrences[0]
	if native.Asset != "SOL" || native.Direction != "balance_debit" || native.Account != address || native.Atomic != "5500" || native.Scale != 9 || native.Measurement != MeasurementNetBalanceDelta || native.From != "" || native.To != "" {
		t.Fatalf("native balance delta conflates account change with transfer: %+v", native)
	}
	if err := Validate(Case{ID: "solana-test", Title: "test", Private: true, Sources: []Source{page.Source, captured.Source}, Occurrences: captured.Occurrences}); err != nil {
		t.Fatalf("captured Solana page failed case validation: %v", err)
	}
	otherClient, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := otherClient.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || replay.Transactions[0].Occurrences[1].Atomic != "6" {
		t.Fatalf("page cache did not return an isolated repeat: calls=%d page=%+v", requests.Load(), replay)
	}
	page.Transactions[0].Occurrences[1].Atomic = "999"
	replay, err = client.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if replay.Transactions[0].Occurrences[1].Atomic != "6" {
		t.Fatal("caller mutation corrupted the cached Solana result")
	}
}

func TestSolanaAddressAndSignatureValidation(t *testing.T) {
	if !validSolanaPubkey(strings.Repeat("1", 32)) || validSolanaPubkey(strings.Repeat("1", 31)) {
		t.Fatal("Solana public key length validation is incorrect")
	}
	if !validSolanaSignature(strings.Repeat("1", 64)) || validSolanaSignature(strings.Repeat("1", 63)) {
		t.Fatal("Solana signature length validation is incorrect")
	}
	if _, err := NewSolanaRPCClient("http://example.com", nil, time.Now); err == nil {
		t.Fatal("non-loopback HTTP RPC endpoint was accepted")
	}
}

func TestSolanaProviderRetryAfterIsReturnedWithoutAutomaticRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), func() time.Time { return time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FetchAddressPage(context.Background(), strings.Repeat("1", 32), "")
	var providerErr *SolanaProviderError
	if !errors.Is(err, ErrSolanaProviderRateLimited) || !errors.As(err, &providerErr) || providerErr.RetryAfter != 7*time.Second || calls.Load() != 1 {
		t.Fatalf("rate limit was not preserved without retry: err=%v calls=%d", err, calls.Load())
	}
}
