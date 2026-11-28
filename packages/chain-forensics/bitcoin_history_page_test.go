package chainforensics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestExpandBitcoinAddressHistoryPageReconstructsRetained1HQ3Pages(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	type retainedPage struct{ file, address, sourceID string }
	pages := []retainedPage{
		{"1BAD-page-1.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-f0917c773a1d8677db6dd747b4aa6fa0ea5fd7c47f268b4d81d8a95c601803fb-1c4b8828eeeda73743521aad35741955b0ac92f32a55eee3bfc5d1581268306e"},
		{"1BAD-page-2.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-b05e79e64de44f120ef3499cdf785fe43c0f008aecae617d07f22149a00167e1-728497c0a18c4202cc972bcf4e6f9c30cec07b66535c055bc1d6b7a11b357749"},
		{"1BBq-page-1.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-9e51108a9d5f58cba93fe169ef67b47578e8e07c512a41cf199369806c94109d-7a22a43df3021e28613d49eeec93ee458e4840833ce9c891e825545c97062f1c"},
		{"1BBq-page-2.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-65b69b81370bba082f5cb887148bb1328b3c2e95c280021d6651ecc4795be0a6-cb5506a5cfb0354dcdf1e61945b9492b40e6b404614be11acd89b4a87f57b2dc"},
	}
	sources := make(map[string]Source)
	for _, source := range c.Sources {
		sources[source.ID] = source
	}
	var all []Occurrence
	seenOccurrences := map[string]bool{}
	for _, p := range pages {
		source, ok := sources[p.sourceID]
		if !ok {
			t.Fatalf("missing retained source %s", p.sourceID)
		}
		body, err := exampleFS.ReadFile("examples/cases/us-v-1hq3/source-captures/" + p.file)
		if err != nil {
			t.Fatal(err)
		}
		source.ResultBody = body
		occurrences, err := ExpandBitcoinAddressHistoryPage(source, p.address)
		if err != nil {
			t.Fatalf("expand %s: %v", p.file, err)
		}
		if len(occurrences) == 0 {
			t.Fatalf("page %s yielded no exact transaction occurrences", p.file)
		}
		var rawTransactions []MempoolTransaction
		if err := json.Unmarshal(body, &rawTransactions); err != nil {
			t.Fatal(err)
		}
		wantOccurrences := 0
		for _, tx := range rawTransactions {
			wantOccurrences += len(tx.Inputs) + len(tx.Outputs)
		}
		if len(occurrences) != wantOccurrences {
			t.Fatalf("page %s expanded to %d occurrences; raw bodies contain %d inputs and outputs", p.file, len(occurrences), wantOccurrences)
		}
		for _, occurrence := range occurrences {
			key := occurrence.SourceID + ":" + occurrence.ID
			if seenOccurrences[key] {
				t.Fatalf("duplicate normalized occurrence %s", key)
			}
			seenOccurrences[key] = true
			all = append(all, occurrence)
		}
	}
	const first, second = "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN"
	var publishedOutputs int
	for _, o := range all {
		if o.Direction == "output" && (o.To == first || o.To == second) && strings.HasPrefix(o.Timestamp, "2012-05-06") {
			publishedOutputs++
		}
		if o.Kind != EvidenceObservation || o.Chain != ChainBitcoin || o.SourceID == "" {
			t.Fatalf("expanded event lost bounded observation provenance: %#v", o)
		}
	}
	if publishedOutputs != 54 {
		t.Fatalf("expanded pages contain %d published 1HQ3 transfer outputs; want 54", publishedOutputs)
	}
	// A normalized input retains the exact referenced outpoint; address/value
	// resemblance cannot create a join when the named transaction output is absent.
	var exactInput, exactOutput Occurrence
	for _, o := range all {
		if o.Direction == "input" && o.PreviousTx != "" {
			exactInput = o
			for _, candidate := range all {
				if candidate.Direction == "output" && candidate.Transaction == o.PreviousTx && candidate.EventIndex == o.PreviousIndex {
					exactOutput = candidate
					break
				}
			}
			if exactOutput.ID != "" {
				break
			}
		}
	}
	if exactInput.ID == "" || exactOutput.ID == "" {
		t.Fatal("retained pages did not replay a complete exact input/outpoint/output chain")
	}
	edges := BuildBitcoinOutpointEdges(all)
	matched := false
	for _, edge := range edges {
		if edge.From == exactOutput.ID && edge.To == exactInput.ID {
			matched = edge.Grade == GradeObserved && edge.Rule == "bitcoin.input.previous_outpoint_exact_match"
		}
	}
	if !matched {
		t.Fatalf("exact retained outpoint was not joined as observed: input=%#v output=%#v", exactInput, exactOutput)
	}
}

func TestMaterializeBitcoinPrevoutsClosesPublished1HQ3InputOutpoints(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	type retainedPage struct{ file, address, sourceID string }
	pages := []retainedPage{
		{"1BAD-page-1.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-f0917c773a1d8677db6dd747b4aa6fa0ea5fd7c47f268b4d81d8a95c601803fb-1c4b8828eeeda73743521aad35741955b0ac92f32a55eee3bfc5d1581268306e"},
		{"1BAD-page-2.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-b05e79e64de44f120ef3499cdf785fe43c0f008aecae617d07f22149a00167e1-728497c0a18c4202cc972bcf4e6f9c30cec07b66535c055bc1d6b7a11b357749"},
		{"1BBq-page-1.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-9e51108a9d5f58cba93fe169ef67b47578e8e07c512a41cf199369806c94109d-7a22a43df3021e28613d49eeec93ee458e4840833ce9c891e825545c97062f1c"},
		{"1BBq-page-2.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-65b69b81370bba082f5cb887148bb1328b3c2e95c280021d6651ecc4795be0a6-cb5506a5cfb0354dcdf1e61945b9492b40e6b404614be11acd89b4a87f57b2dc"},
	}
	sources := make(map[string]Source)
	for _, source := range c.Sources {
		sources[source.ID] = source
	}
	var all []Occurrence
	for _, page := range pages {
		source, ok := sources[page.sourceID]
		if !ok {
			t.Fatalf("missing source %s", page.sourceID)
		}
		body, err := exampleFS.ReadFile("examples/cases/us-v-1hq3/source-captures/" + page.file)
		if err != nil {
			t.Fatal(err)
		}
		source.ResultBody = body
		occurrences, err := ExpandBitcoinAddressHistoryPage(source, page.address)
		if err != nil {
			t.Fatalf("expand %s: %v", page.file, err)
		}
		all = append(all, occurrences...)
	}
	const first, second = "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN"
	targetTx := make(map[string]bool)
	for _, event := range all {
		if event.Direction == "output" && (event.To == first || event.To == second) && strings.HasPrefix(event.Timestamp, "2012-05-06") {
			targetTx[event.Transaction] = true
		}
	}
	if len(targetTx) != 54 {
		t.Fatalf("target transaction count = %d; want 54", len(targetTx))
	}
	var inputs []Occurrence
	var existingOutputs []Occurrence
	for _, event := range all {
		if event.Direction == "input" && targetTx[event.Transaction] {
			inputs = append(inputs, event)
		}
		if event.Direction == "output" {
			existingOutputs = append(existingOutputs, event)
		}
	}
	if len(inputs) != 3350 {
		t.Fatalf("published transfer input count = %d; want 3350", len(inputs))
	}
	added, edges, err := MaterializeBitcoinPrevoutOutputs(inputs, existingOutputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != len(inputs) {
		t.Fatalf("exact prevout edges = %d; want %d", len(edges), len(inputs))
	}
	ids := make(map[string]struct{}, len(added))
	for _, output := range added {
		if output.Block != 0 || output.Timestamp != "" {
			t.Fatalf("embedded prevout metadata was given an unsupported block/time: %#v", output)
		}
		ids[output.ID] = struct{}{}
	}
	if len(added) != 3322 {
		t.Fatalf("materialized predecessor output count = %d; want 3,322 after 28 in-page outputs", len(added))
	}
	for _, edge := range edges {
		if edge.Grade != GradeObserved || edge.Rule != "bitcoin.input.previous_outpoint_exact_match" || len(edge.EvidenceIDs) != 2 {
			t.Fatalf("invalid source-backed exact outpoint edge: %#v", edge)
		}
	}
	// A conflicting claimed prevout cannot be joined, even when its outpoint
	// identity matches an existing output.
	for _, input := range inputs {
		for _, output := range existingOutputs {
			if output.Transaction == input.PreviousTx && output.EventIndex == input.PreviousIndex {
				input.Atomic = "1"
				if _, _, err := MaterializeBitcoinPrevoutOutputs([]Occurrence{input}, existingOutputs); err == nil {
					t.Fatal("conflicting exact outpoint metadata was accepted")
				}
				return
			}
		}
	}
	t.Fatal("fixture lacks an existing exact predecessor output for conflict validation")
}

func TestExpandBitcoinAddressHistoryPageRejectsUnboundOrUnconfirmedEvidence(t *testing.T) {
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	const address = "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN"
	var source Source
	for _, candidate := range c.Sources {
		if strings.Contains(candidate.URL, "/address/"+address+"/") {
			source = candidate
			break
		}
	}
	if source.ID == "" {
		t.Fatal("missing retained address page source")
	}
	source.ResultBody, err = exampleFS.ReadFile("examples/cases/us-v-1hq3/source-captures/1BBq-page-1.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExpandBitcoinAddressHistoryPage(source, "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ"); err == nil {
		t.Fatal("accepted an endpoint bound to a different address")
	}
	badHash := source
	badHash.ResultBody = append(append([]byte(nil), source.ResultBody...), ' ')
	if _, err := ExpandBitcoinAddressHistoryPage(badHash, address); err == nil {
		t.Fatal("accepted response bytes that do not match the retained source digest")
	}
	var txs []MempoolTransaction
	if err := json.Unmarshal(source.ResultBody, &txs); err != nil {
		t.Fatal(err)
	}
	txs[0].Status.Confirmed = false
	mutated, err := json.Marshal(txs)
	if err != nil {
		t.Fatal(err)
	}
	unconfirmed := source
	unconfirmed.ResultBody = mutated
	digest := sha256.Sum256(mutated)
	unconfirmed.SHA256 = hex.EncodeToString(digest[:])
	if _, err := ExpandBitcoinAddressHistoryPage(unconfirmed, address); err == nil {
		t.Fatal("accepted transaction without a confirmed chain position")
	}
}

func TestBuildBitcoinOutpointEdgesDoesNotInferApproximateMatches(t *testing.T) {
	output := Occurrence{ID: "bitcoin:prev:out:0", Chain: ChainBitcoin, Direction: "output", Transaction: "prev", EventIndex: 0, To: "same-address", Atomic: "100", Kind: EvidenceObservation}
	input := Occurrence{ID: "bitcoin:spend:in:0", Chain: ChainBitcoin, Direction: "input", Transaction: "spend", EventIndex: 0, PreviousTx: "other", PreviousIndex: 0, From: "same-address", Atomic: "100", Kind: EvidenceObservation}
	if edges := BuildBitcoinOutpointEdges([]Occurrence{output, input}); len(edges) != 0 {
		t.Fatalf("address/value resemblance produced an outpoint edge: %#v", edges)
	}
}

func TestBitcoinTransactionMembershipEdgesDoNotAssignInputValueToOutput(t *testing.T) {
	input := Occurrence{ID: "bitcoin:tx:in:0", Chain: ChainBitcoin, Transaction: "tx", EventIndex: 0, Direction: "input", PreviousTx: strings.Repeat("a", 64), From: "input-address", Atomic: "300", Asset: "BTC", Scale: 8, Kind: EvidenceObservation, SourceID: "source"}
	outputs := []Occurrence{
		{ID: "bitcoin:tx:out:0", Chain: ChainBitcoin, Transaction: "tx", EventIndex: 0, Direction: "output", To: "recipient", Atomic: "100", Asset: "BTC", Scale: 8, Kind: EvidenceObservation, SourceID: "source"},
		{ID: "bitcoin:tx:out:1", Chain: ChainBitcoin, Transaction: "tx", EventIndex: 1, Direction: "output", To: "change", Atomic: "199", Asset: "BTC", Scale: 8, Kind: EvidenceObservation, SourceID: "source"},
	}
	foreign := Occurrence{ID: "bitcoin:other:out:0", Chain: ChainBitcoin, Transaction: "other", EventIndex: 0, Direction: "output"}
	edges := BuildBitcoinTransactionMembershipEdgesForTransactions(append([]Occurrence{input}, append(outputs, foreign)...), []string{"tx"})
	if len(edges) != 2 {
		t.Fatalf("membership edge count = %d; want one for each output", len(edges))
	}
	for i, edge := range edges {
		if edge.Grade != GradeObserved || edge.Rule != "bitcoin.transaction.event_membership_without_value_allocation" || edge.From != input.ID || edge.To != outputs[i].ID || len(edge.Caveats) == 0 || !strings.Contains(edge.Caveats[0], "no input-specific satoshi allocation") {
			t.Fatalf("membership edge overstates a multi-output transaction: %#v", edge)
		}
	}
}
