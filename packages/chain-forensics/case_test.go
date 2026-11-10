package chainforensics

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
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
	if len(c.Occurrences) != 4 || len(c.Edges) != 3 {
		t.Fatalf("unexpected bounded case excerpt: occurrences=%d edges=%d", len(c.Occurrences), len(c.Edges))
	}
	if c.Occurrences[0].Atomic != "6947108220100" {
		t.Fatalf("large exact BTC quantity changed: %s", c.Occurrences[0].Atomic)
	}
	for _, edge := range c.Edges {
		if edge.Grade != GradeObserved {
			t.Fatalf("direct transaction edge has wrong grade %q", edge.Grade)
		}
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
	packet, err := MarshalPrivatePacket(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(packet), `"schema_version": "chain-forensics.packet/v1"`) || !strings.Contains(string(packet), `"private": true`) {
		t.Fatalf("packet omitted its version or privacy marker: %s", packet)
	}
	c.Private = false
	if _, err := MarshalPrivatePacket(c); err == nil || !strings.Contains(err.Error(), "not marked private") {
		t.Fatalf("expected non-private export to be refused, got %v", err)
	}
}
