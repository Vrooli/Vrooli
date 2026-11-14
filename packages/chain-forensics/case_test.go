package chainforensics

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestExampleCasePreservesObservedFlowsAndSeparateLegalClaim(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	if !c.Private {
		t.Fatal("case fixture must default to private")
	}
	if len(c.Claims) != 1 || c.Claims[0].Kind != EvidenceLegalClaim {
		t.Fatalf("legal assertion was not retained as a separate claim: %#v", c.Claims)
	}
	if len(c.Occurrences) != 58 || len(c.Edges) != 57 {
		t.Fatalf("unexpected case size: occurrences=%d edges=%d", len(c.Occurrences), len(c.Edges))
	}
	var totalOutputFound bool
	for _, occurrence := range c.Occurrences {
		if occurrence.Transaction == "5d9ef693d41cb3bb4c6d98e70ea8b2cc91be29a804245a06ec8761d9cddc103c" && occurrence.Direction == "output" {
			totalOutputFound = occurrence.Atomic == "6947108220100" && occurrence.To == "1HQ3Go3ggs8pFnXuHVHRytPCq5fGG8Hbhx"
		}
	}
	if !totalOutputFound {
		t.Fatal("large exact BTC output to the published 1HQ3 address changed")
	}
	for _, edge := range c.Edges {
		if edge.Grade != GradeObserved {
			t.Fatalf("direct transaction edge has wrong grade %q", edge.Grade)
		}
	}
	counts := map[string]int{}
	inputTotal := uint64(0)
	for _, occurrence := range c.Occurrences {
		if occurrence.Transaction != "5d9ef693d41cb3bb4c6d98e70ea8b2cc91be29a804245a06ec8761d9cddc103c" || occurrence.Direction != "input" {
			continue
		}
		counts[occurrence.From]++
		value, err := strconv.ParseUint(occurrence.Atomic, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		inputTotal += value
	}
	if counts["1BADznNF3W1gi47R65MQs754KB7zTaGuYZ"] != 31 || counts["1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN"] != 23 || inputTotal != 6947108220100 {
		t.Fatalf("published 54-input flow not reproduced exactly: counts=%v input_sats=%d", counts, inputTotal)
	}
}

func TestSourceRejectsTamperedRetainedHTTPResponse(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	response := []byte(`{"jsonrpc":"2.0","id":1,"result":[]}`)
	digest := sha256.Sum256(response)
	source := c.Sources[0]
	source.RawResponseBody = append([]byte(nil), response...)
	source.RawResponseSHA256 = fmt.Sprintf("%x", digest)
	c.Sources[0] = source
	if err := Validate(c); err != nil {
		t.Fatalf("valid retained HTTP response was rejected: %v", err)
	}
	c.Sources[0].RawResponseBody[0] ^= 1
	if err := Validate(c); err == nil || !strings.Contains(err.Error(), "raw HTTP response does not match") {
		t.Fatalf("tampered retained HTTP response was accepted: %v", err)
	}
}

func TestBitcoinOutputIndexZeroSurvivesTypedCasePersistenceRoundTrip(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	c.Occurrences = append(c.Occurrences, Occurrence{
		ID: "zero-outpoint-input", Chain: ChainBitcoin, Transaction: "855cbf0000000000000000000000000000000000000000000000000000000000",
		EventIndex: 0, Direction: "input", PreviousTx: "855cbf0000000000000000000000000000000000000000000000000000000000", PreviousIndex: 0,
		Asset: "BTC", Atomic: "100", Kind: EvidenceObservation, SourceID: c.Sources[0].ID,
	})
	encoded, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"previous_output_index":0`) {
		t.Fatalf("typed case omitted zero outpoint index: %s", encoded)
	}
	decoded, err := DecodeCase(strings.NewReader(string(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Occurrences[len(decoded.Occurrences)-1].PreviousIndex; got != 0 {
		t.Fatalf("round-tripped output index = %d, want 0", got)
	}
}

func TestCaseValidatesEVMWalletSnapshotsAgainstSourcesAndBalances(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"wallet-block", "wallet-native", "wallet-token", "wallet-decimals", "wallet-block-check"} {
		c.Sources = append(c.Sources, Source{ID: id, Name: "public EVM RPC", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: strings.Repeat("a", 64)})
	}
	c.WalletSnapshots = []EVMWalletSnapshot{{ID: "wallet-snapshot", Chain: ChainBase, Address: fixtureEVMAddress, Block: 31, BlockHash: fixtureEVMBlock, Timestamp: "2026-11-12T12:00:00Z", NativeAsset: "ETH", NativeAtomic: "0", NativeScale: 18, Tokens: []WalletTokenBalance{{Contract: fixtureEVMContract, Atomic: "42", Scale: 6, ScaleKnown: true}}, SourceIDs: []string{"wallet-block", "wallet-native", "wallet-token", "wallet-decimals", "wallet-block-check"}}}
	if err := Validate(c); err != nil {
		t.Fatalf("valid EVM wallet snapshot rejected: %v", err)
	}
	missingSource := c
	missingSource.WalletSnapshots = append([]EVMWalletSnapshot(nil), c.WalletSnapshots...)
	missingSource.WalletSnapshots[0].SourceIDs = []string{"wallet-block", "missing-native"}
	if err := Validate(missingSource); err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("snapshot without source provenance was accepted: %v", err)
	}
	badBalance := c
	badBalance.WalletSnapshots = append([]EVMWalletSnapshot(nil), c.WalletSnapshots...)
	badBalance.WalletSnapshots[0].NativeAtomic = "-1"
	if err := Validate(badBalance); err == nil {
		t.Fatal("negative native balance was accepted")
	}
}

func TestNormalizeBitcoinTransactionAndMatchExactOutpoint(t *testing.T) {
	var tx MempoolTransaction
	raw := `{"txid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","status":{"confirmed":true,"block_height":100,"block_time":1700000000},"vin":[{"txid":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","vout":3,"prevout":{"scriptpubkey_address":"source","value":9000000000}}],"vout":[{"scriptpubkey_address":"destination","value":8999999000}]}`
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatal(err)
	}
	source := Source{ID: "capture", Name: "fixture", URL: "https://example.test/tx", Retrieved: "2026-11-09T00:00:00Z", SHA256: strings.Repeat("a", 64)}
	current, err := NormalizeBitcoinTransaction(source, tx)
	if err != nil {
		t.Fatal(err)
	}
	previous := Occurrence{ID: "bitcoin:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:out:3", Chain: ChainBitcoin, Transaction: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Direction: "output", EventIndex: 3}
	edges := BuildBitcoinOutpointEdges(append([]Occurrence{previous}, current...))
	if len(edges) != 1 || edges[0].To != current[0].ID || edges[0].Grade != GradeObserved {
		t.Fatalf("expected exact prevout match, got %#v", edges)
	}
	if current[0].Atomic != "9000000000" || current[0].PreviousIndex != 3 {
		t.Fatalf("input was not normalized exactly: %#v", current[0])
	}
}

func TestPublishedCaseCapturesReplayToTheCuratedOutpoint(t *testing.T) {
	const root = "examples/cases/us-v-1hq3/source-captures/"
	ids := []string{
		"5d9ef693d41cb3bb4c6d98e70ea8b2cc91be29a804245a06ec8761d9cddc103c",
		"97facbd7e07c4f3140d9ea167c2e54d85bc5f6fd04d9774d80b1284b706920ff",
	}
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	sourceByURL := make(map[string]Source, len(c.Sources))
	for _, source := range c.Sources {
		sourceByURL[source.URL] = source
	}
	var normalized []Occurrence
	for _, txid := range ids {
		path := root + txid + ".json"
		body, err := exampleFS.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var tx MempoolTransaction
		if err := json.Unmarshal(body, &tx); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if tx.TxID != txid {
			t.Fatalf("capture %s contains transaction %s", path, tx.TxID)
		}
		captureHash := fmt.Sprintf("%x", sha256.Sum256(body))
		source, ok := sourceByURL["https://mempool.space/api/tx/"+txid]
		if !ok || source.SHA256 != captureHash {
			t.Fatalf("capture hash mismatch for %s: source=%q actual=%q", txid, source.SHA256, captureHash)
		}
		events, err := NormalizeBitcoinTransaction(source, tx)
		if err != nil {
			t.Fatalf("normalize %s: %v", txid, err)
		}
		normalized = append(normalized, events...)
	}
	edges := BuildBitcoinOutpointEdges(normalized)
	var matched bool
	for _, edge := range edges {
		if strings.Contains(edge.ID, ids[0]+":0->"+ids[1]+":0") {
			matched = edge.Grade == GradeObserved && edge.Rule == "bitcoin.input.previous_outpoint_exact_match"
		}
	}
	if !matched {
		t.Fatalf("captured transactions did not reconstruct their exact outpoint edge; found %d edges", len(edges))
	}

	fixture, err := fs.ReadFile(exampleFS, "examples/cases/us-v-1hq3/case.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fixture), "complaint-1hq3-history") {
		t.Fatal("separate complaint claim disappeared from fixture")
	}
}

func TestPublishedCaseIndependentExplorerCapturesAndCounterexample(t *testing.T) {
	const mempoolRoot = "examples/cases/us-v-1hq3/source-captures/"
	const blockstreamRoot = "examples/cases/us-v-1hq3/counterexample-captures/"
	ids := []string{
		"5d9ef693d41cb3bb4c6d98e70ea8b2cc91be29a804245a06ec8761d9cddc103c",
		"97facbd7e07c4f3140d9ea167c2e54d85bc5f6fd04d9774d80b1284b706920ff",
	}
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Counterexamples) < 2 || len(c.Disagreements) == 0 {
		t.Fatalf("case must retain the attribution counterexample and provider disagreement: %#v %#v", c.Counterexamples, c.Disagreements)
	}
	sourceByURL := make(map[string]Source, len(c.Sources))
	for _, source := range c.Sources {
		sourceByURL[source.URL] = source
	}
	for _, txid := range ids {
		mempoolBody, err := exampleFS.ReadFile(mempoolRoot + txid + ".json")
		if err != nil {
			t.Fatal(err)
		}
		blockstreamBody, err := exampleFS.ReadFile(blockstreamRoot + txid + ".json")
		if err != nil {
			t.Fatal(err)
		}
		mempoolSource, ok := sourceByURL["https://mempool.space/api/tx/"+txid]
		if !ok || mempoolSource.SHA256 != fmt.Sprintf("%x", sha256.Sum256(mempoolBody)) {
			t.Fatalf("missing or corrupt mempool provenance for %s", txid)
		}
		blockstreamSource, ok := sourceByURL["https://blockstream.info/api/tx/"+txid]
		if !ok || blockstreamSource.SHA256 != fmt.Sprintf("%x", sha256.Sum256(blockstreamBody)) || blockstreamSource.Retrieved != "2026-11-12T09:59:57Z" {
			t.Fatalf("missing, corrupt, or untimed Blockstream provenance for %s", txid)
		}
		var mempoolTx, blockstreamTx MempoolTransaction
		if err := json.Unmarshal(mempoolBody, &mempoolTx); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(blockstreamBody, &blockstreamTx); err != nil {
			t.Fatal(err)
		}
		mempoolEvents, err := NormalizeBitcoinTransaction(mempoolSource, mempoolTx)
		if err != nil {
			t.Fatal(err)
		}
		blockstreamEvents, err := NormalizeBitcoinTransaction(blockstreamSource, blockstreamTx)
		if err != nil {
			t.Fatal(err)
		}
		for i := range mempoolEvents {
			mempoolEvents[i].SourceID = ""
			blockstreamEvents[i].SourceID = ""
		}
		if !reflect.DeepEqual(mempoolEvents, blockstreamEvents) {
			t.Fatalf("independent captures disagree on normalized chain facts for %s", txid)
		}
	}
	complaintBody, err := exampleFS.ReadFile("examples/cases/us-v-1hq3/complaint-captures/complaint_crypto_seizure.pdf")
	if err != nil {
		t.Fatal(err)
	}
	complaint, ok := sourceByURL["https://www.justice.gov/d9/press-releases/attachments/2020/11/05/complaint_crypto_seizure.pdf"]
	if !ok || complaint.Retrieved != "2026-11-12T10:05:12Z" || complaint.SHA256 != fmt.Sprintf("%x", sha256.Sum256(complaintBody)) {
		t.Fatal("bundled DOJ complaint capture is missing timestamped hash provenance")
	}

	// The case's strongest counterexample is the 2015 same-address return
	// output: input flow is not safely assignable to the 101 BTC payment.
	var change, payment bool
	for _, occurrence := range c.Occurrences {
		if occurrence.Transaction != ids[1] || occurrence.Direction != "output" {
			continue
		}
		if occurrence.To == "1HQ3Go3ggs8pFnXuHVHRytPCq5fGG8Hbhx" && occurrence.Atomic == "6937008220100" {
			change = true
		}
		if occurrence.To == "1PLbnu8oYYyyXftkCBW4so6KHNKkpnexCB" && occurrence.Atomic == "10100000000" {
			payment = true
		}
	}
	if !change || !payment {
		t.Fatalf("expected exact change/payment counterexample; change=%t payment=%t", change, payment)
	}
}

func TestValidateRejectsSourceWithoutResponseProvenance(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	c.Sources[1].SHA256 = ""
	if err := Validate(c); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("expected incomplete provenance error, got %v", err)
	}
}

func TestValidateRejectsEdgeWithUnknownEvidence(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	c.Edges[0].EvidenceIDs[0] = "missing"
	if err := Validate(c); err == nil || !strings.Contains(err.Error(), "unknown evidence") {
		t.Fatalf("expected unknown evidence error, got %v", err)
	}
}

func TestPacketExportIsExplicitVersionedAndPrivate(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	typed, err := BuildPrivatePacket(c)
	if err != nil {
		t.Fatal(err)
	}
	packet := typed.Bytes
	if len(typed.SHA256) != 64 || typed.SchemaVersion != "chain-forensics.packet/v1" || typed.Case.ID != c.ID {
		t.Fatalf("typed private packet omitted version, digest, or case identity: %+v", typed)
	}
	if !strings.Contains(string(packet), `"schema_version": "chain-forensics.packet/v1"`) || !strings.Contains(string(packet), `"private": true`) {
		t.Fatalf("packet omitted its version or privacy marker: %s", packet)
	}
	if !strings.Contains(string(packet), `"counterexamples"`) || !strings.Contains(string(packet), `"source_disagreements"`) {
		t.Fatalf("private packet omitted counterexample/disagreement assessment: %s", packet)
	}
	c.Private = false
	if _, err := MarshalPrivatePacket(c); err == nil || !strings.Contains(err.Error(), "not marked private") {
		t.Fatalf("expected non-private export to be refused, got %v", err)
	}
}
