package chainforensics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The retained complaint-date outputs are a material part of the published
// 1HQ3 flow set. Keep their raw address pages, exact output joins and source
// boundary in the reusable example, not only in a research-run audit script.
func TestPublished1HQ3TransferSetCoverageReplaysFromBundledPages(t *testing.T) {
	const rawRoot = "examples/cases/us-v-1hq3/source-captures/"
	type page struct {
		file, address, sourceID string
	}
	pages := []page{
		{"1BAD-page-1.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-f0917c773a1d8677db6dd747b4aa6fa0ea5fd7c47f268b4d81d8a95c601803fb-1c4b8828eeeda73743521aad35741955b0ac92f32a55eee3bfc5d1581268306e"},
		{"1BAD-page-2.json", "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "mempool-address-page-b05e79e64de44f120ef3499cdf785fe43c0f008aecae617d07f22149a00167e1-728497c0a18c4202cc972bcf4e6f9c30cec07b66535c055bc1d6b7a11b357749"},
		{"1BBq-page-1.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-9e51108a9d5f58cba93fe169ef67b47578e8e07c512a41cf199369806c94109d-7a22a43df3021e28613d49eeec93ee458e4840833ce9c891e825545c97062f1c"},
		{"1BBq-page-2.json", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN", "mempool-address-page-65b69b81370bba082f5cb887148bb1328b3c2e95c280021d6651ecc4795be0a6-cb5506a5cfb0354dcdf1e61945b9492b40e6b404614be11acd89b4a87f57b2dc"},
	}
	c, err := ExampleCase()
	if err != nil {
		t.Fatal(err)
	}
	const first, second = "1BADznNF3W1gi47R65MQs754KB7zTaGuYZ", "1BBqjKsYuLEUE9Y5WzdbzCtYzCiQgHqtPN"
	pageBySource := map[string]page{}
	for _, p := range pages {
		pageBySource[p.sourceID] = p
	}
	sourceByID := make(map[string]Source, len(c.Sources))
	for _, source := range c.Sources {
		sourceByID[source.ID] = source
	}
	type output struct {
		TxID   string `json:"txid"`
		Status struct {
			Confirmed   bool   `json:"confirmed"`
			BlockTime   int64  `json:"block_time"`
			BlockHeight uint64 `json:"block_height"`
		} `json:"status"`
		Vout []struct {
			Address string `json:"scriptpubkey_address"`
			Value   uint64 `json:"value"`
		} `json:"vout"`
	}
	type expected struct {
		address, atomic, sourceID, timestamp string
		block                                uint64
	}
	want := map[string]expected{}
	for _, p := range pages {
		body, err := exampleFS.ReadFile(rawRoot + p.file)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(body)
		source, ok := sourceByID[p.sourceID]
		if !ok || source.SHA256 != hex.EncodeToString(digest[:]) || !strings.Contains(source.URL, "/txs/chain") {
			t.Fatalf("page %s lacks exact source hash and endpoint provenance", p.file)
		}
		var txs []output
		if err := json.Unmarshal(body, &txs); err != nil {
			t.Fatalf("decode %s: %v", p.file, err)
		}
		for _, tx := range txs {
			if !tx.Status.Confirmed || tx.Status.BlockTime == 0 {
				continue
			}
			stamp := time.Unix(tx.Status.BlockTime, 0).UTC().Format(time.RFC3339)
			if !strings.HasPrefix(stamp, "2012-05-06") {
				continue
			}
			for i, vout := range tx.Vout {
				if vout.Address != first && vout.Address != second {
					continue
				}
				id := fmt.Sprintf("bitcoin:%s:out:%d", tx.TxID, i)
				if _, duplicate := want[id]; duplicate {
					t.Fatalf("duplicate published outpoint %s across captured pages", id)
				}
				want[id] = expected{vout.Address, fmt.Sprint(vout.Value), p.sourceID, stamp, tx.Status.BlockHeight}
			}
		}
	}
	got := map[string]Occurrence{}
	for _, o := range c.Occurrences {
		if o.Chain == ChainBitcoin && o.Direction == "output" && (o.To == first || o.To == second) && strings.HasPrefix(o.Timestamp, "2012-05-06") {
			got[o.ID] = o
		}
	}
	if len(want) != 54 || len(got) != len(want) {
		t.Fatalf("published transfer set count mismatch: raw pages=%d case observations=%d; want 54", len(want), len(got))
	}
	var total uint64
	for id, raw := range want {
		o, ok := got[id]
		if !ok || o.To != raw.address || o.Atomic != raw.atomic || o.SourceID != raw.sourceID || o.Timestamp != raw.timestamp || o.Block != raw.block || o.Kind != EvidenceObservation || o.Asset != "BTC" || o.Scale != 8 {
			t.Fatalf("case does not exactly match raw output %s: %#v want %#v", id, o, raw)
		}
		var value uint64
		if _, err := fmt.Sscan(o.Atomic, &value); err != nil {
			t.Fatal(err)
		}
		total += value
	}
	if total != 7_041_146_000_000 {
		t.Fatalf("published transfer amount = %d sat, want 7041146000000", total)
	}
	if len(c.Claims) != 1 || c.Claims[0].Kind != EvidenceLegalClaim || !strings.Contains(c.Claims[0].Text, "54 Silk Road transfers") {
		t.Fatalf("complaint attribution must remain a separate legal claim: %#v", c.Claims)
	}
	if len(c.Counterexamples) < 2 || len(c.Gaps) == 0 {
		t.Fatalf("published case lost its counterexamples or explicit limits: counterexamples=%d gaps=%v", len(c.Counterexamples), c.Gaps)
	}
}
