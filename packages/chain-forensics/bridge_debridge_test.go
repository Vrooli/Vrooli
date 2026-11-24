package chainforensics

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestDebridgeDLNExactOrderCorrelationAndMismatchControls(t *testing.T) {
	c := dlnFixture()
	edges := debridgeDLNEdges(c)
	if len(edges) != 1 || edges[0].Grade != GradeProtocolCorrelated || edges[0].ID != "bridge:debridge-dln:"+dlnFixtureOrderID {
		t.Fatalf("unexpected DLN edge: %+v", edges)
	}
	var fulfilledReceipt receiptEnvelope
	for _, source := range c.Sources {
		if source.ID == "dst" {
			if err := json.Unmarshal(source.ResultBody, &fulfilledReceipt); err != nil {
				t.Fatal(err)
			}
		}
	}
	order, _, actual, ok := decodeDLNDestinationEvent(fulfilledReceipt.Logs[0].Data, dlnDestinationEventWithActualAmount)
	if !ok || order.takeAmount != "9" || actual != "8" || c.Occurrences[1].Atomic != actual {
		t.Fatalf("positive correlation did not bind the partial fill amount: order=%+v actual=%q occurrences=%+v", order, actual, c.Occurrences)
	}
	cases := []struct {
		name   string
		mutate func(*Case)
	}{
		{"wrong contract", func(c *Case) {
			editDLNLog(c, "src", func(l *EVMLog) { l.Address = "0x0000000000000000000000000000000000000001" })
		}},
		{"wrong event topic", func(c *Case) { editDLNLog(c, "dst", func(l *EVMLog) { l.Topics[0] = strings.Repeat("0", 66) }) }},
		{"wrong order id", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) { l.Data = replaceDLNDataWord(l.Data, 1, abiWord("99")) })
		}},
		{"wrong destination chain", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) {
				l.Data = dlnTestFulfilledData(dlnFixtureOrderID, "1", "42161", "0x0000000000000000000000000000000000000011", "0x0000000000000000000000000000000000000022", "0x0000000000000000000000000000000000000033")
			})
		}},
		{"different give amount", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) {
				l.Data = dlnTestFulfilledData(dlnFixtureOrderID, "2", "1", "0x0000000000000000000000000000000000000011", "0x0000000000000000000000000000000000000022", "0x0000000000000000000000000000000000000033")
			})
		}},
		{"malformed offsets", func(c *Case) { editDLNLog(c, "dst", func(l *EVMLog) { l.Data = "0x" + strings.Repeat("f", 64) }) }},
		{"duplicate candidate occurrence", func(c *Case) { c.Occurrences = append(c.Occurrences, c.Occurrences[1]) }},
		{"same amount and token sent elsewhere", func(c *Case) { c.Occurrences[0].To = "0x0000000000000000000000000000000000000099" }},
		{"net balance delta is not a transfer", func(c *Case) {
			c.Occurrences[0].Measurement, c.Occurrences[0].Account = MeasurementNetBalanceDelta, "11111111111111111111111111111111"
		}},
		{"destination net balance delta is not a transfer", func(c *Case) {
			c.Occurrences[1].Measurement, c.Occurrences[1].Account = MeasurementNetBalanceDelta, "11111111111111111111111111111111"
		}},
		{"partial-fill occurrence must equal actual not requested", func(c *Case) {
			c.Occurrences[1].Atomic = "9"
		}},
		{"actual fill cannot exceed requested amount", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) { l.Data = replaceDLNDataWord(l.Data, 2, abiWord("10")) })
		}},
		{"receipt transaction hash must match each log", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) { l.TransactionHash = "0x" + strings.Repeat("99", 32) })
		}},
		{"receipt block hash must match each log", func(c *Case) {
			editDLNLog(c, "src", func(l *EVMLog) { l.BlockHash = "0x" + strings.Repeat("99", 32) })
		}},
		{"receipt block number must match each log", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) { l.BlockNumber = "0x999" })
		}},
		{"receipt transaction index must match each log", func(c *Case) {
			editDLNLog(c, "dst", func(l *EVMLog) { l.TransactionIndex = "0x99" })
		}},
		{"receipt requires a log index", func(c *Case) {
			editDLNLog(c, "src", func(l *EVMLog) { l.LogIndex = "" })
		}},
		{"reverted receipt cannot emit bridge correlation", func(c *Case) {
			editDLNReceipt(c, "dst", func(r *receiptEnvelope) { r.Status = "0x0" })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := dlnFixture()
			tc.mutate(&bad)
			if got := debridgeDLNEdges(bad); len(got) != 0 {
				t.Fatalf("false positive: %+v", got)
			}
		})
	}
	bad := dlnFixture()
	bad.Sources[0].SHA256 = strings.Repeat("0", 64)
	if got := debridgeDLNEdges(bad); len(got) != 0 {
		t.Fatalf("hash-mismatched receipt result created edge: %+v", got)
	}
	bad = dlnFixture()
	bad.Sources[0].RawResponseSHA256 = strings.Repeat("0", 64)
	if got := debridgeDLNEdges(bad); len(got) != 0 {
		t.Fatalf("hash-mismatched raw RPC envelope created edge: %+v", got)
	}
	bad = dlnFixture()
	bad.Sources[0].RawResponseBody = append([]byte(nil), bad.Sources[0].RawResponseBody...)
	bad.Sources[0].RawResponseBody = bytes.Replace(bad.Sources[0].RawResponseBody, []byte(`"jsonrpc":"2.0"`), []byte(`"jsonrpc":"1.0"`), 1)
	digest := sha256.Sum256(bad.Sources[0].RawResponseBody)
	bad.Sources[0].RawResponseSHA256 = hex.EncodeToString(digest[:])
	if got := debridgeDLNEdges(bad); len(got) != 0 {
		t.Fatalf("noncanonical JSON-RPC envelope created edge: %+v", got)
	}
	// The same order ID cannot bridge a chain-ID/deployment mismatch.
	bad = dlnFixture()
	bad.Occurrences[1].Chain = ChainArbitrum
	if got := debridgeDLNEdges(bad); len(got) != 0 {
		t.Fatalf("chain mismatch created edge: %+v", got)
	}
}

var dlnFixtureOrderID = "0x" + "ab" + strings.Repeat("cd", 31)

func dlnFixture() Case {
	srcTx := "0x" + strings.Repeat("11", 32)
	dstTx := "0x" + strings.Repeat("22", 32)
	src := dlnReceiptSource("src", srcTx, debridgeSourceAddress, debridgeCreatedOrderTopic, dlnFixtureOrderID, dlnTestEventData(dlnFixtureOrderID, "1", "8453", "0x0000000000000000000000000000000000000011", "0x0000000000000000000000000000000000000022", "0x0000000000000000000000000000000000000033"))
	dst := dlnReceiptSource("dst", dstTx, debridgeDestinationAddress, debridgeFulfilledOrderAmountTopic, dlnFixtureOrderID, dlnTestFulfilledData(dlnFixtureOrderID, "1", "8453", "0x0000000000000000000000000000000000000011", "0x0000000000000000000000000000000000000022", "0x0000000000000000000000000000000000000033"))
	return Case{ID: "dln-fixture", Private: true, Sources: []Source{src, dst}, Occurrences: []Occurrence{
		{ID: "src-occ", Chain: ChainEthereum, Transaction: srcTx, To: debridgeSourceAddress, Asset: "erc20:0x0000000000000000000000000000000000000011", Atomic: "1", Kind: EvidenceObservation, SourceID: "src"},
		{ID: "dst-occ", Chain: ChainBase, Transaction: dstTx, To: "0x0000000000000000000000000000000000000033", Asset: "erc20:0x0000000000000000000000000000000000000022", Atomic: "8", Kind: EvidenceObservation, SourceID: "dst"},
	}}
}

func dlnReceiptSource(id, tx, address, topic, orderID, data string) Source {
	blockNumber, blockHash, txIndex, logIndex := "0x100", "0x"+strings.Repeat("aa", 32), "0x1", "0x2"
	if id == "dst" {
		blockNumber, blockHash, txIndex, logIndex = "0x200", "0x"+strings.Repeat("bb", 32), "0x3", "0x4"
	}
	log := EVMLog{Address: address, Topics: []string{topic}, Data: data, BlockHash: blockHash, BlockNumber: blockNumber, TransactionHash: tx, TransactionIndex: txIndex, LogIndex: logIndex}
	b, _ := json.Marshal(receiptEnvelope{BlockHash: blockHash, BlockNumber: blockNumber, Status: "0x1", TransactionHash: tx, TransactionIndex: txIndex, Logs: []EVMLog{log}})
	h := sha256.Sum256(b)
	raw, _ := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{JSONRPC: "2.0", ID: 1, Result: b})
	rawHash := sha256.Sum256(raw)
	return Source{ID: id, Name: "Synthetic canonical RPC receipt fixture", URL: "https://rpc.example.invalid", Retrieved: "2026-11-15T00:00:00Z", Citation: "Synthetic ABI and receipt-identity fixture; not a chain record.", SHA256: hex.EncodeToString(h[:]), RawResponseSHA256: hex.EncodeToString(rawHash[:]), ResultBody: b, RawResponseBody: raw}
}
func editDLNLog(c *Case, id string, fn func(*EVMLog)) {
	for i := range c.Sources {
		if c.Sources[i].ID != id {
			continue
		}
		var r receiptEnvelope
		_ = json.Unmarshal(c.Sources[i].ResultBody, &r)
		fn(&r.Logs[0])
		refreshDLNReceiptSource(&c.Sources[i], r)
	}
}

func editDLNReceipt(c *Case, id string, fn func(*receiptEnvelope)) {
	for i := range c.Sources {
		if c.Sources[i].ID != id {
			continue
		}
		var r receiptEnvelope
		_ = json.Unmarshal(c.Sources[i].ResultBody, &r)
		fn(&r)
		refreshDLNReceiptSource(&c.Sources[i], r)
	}
}

func refreshDLNReceiptSource(source *Source, receipt receiptEnvelope) {
	b, _ := json.Marshal(receipt)
	h := sha256.Sum256(b)
	raw, _ := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{JSONRPC: "2.0", ID: 1, Result: b})
	rawHash := sha256.Sum256(raw)
	source.ResultBody, source.SHA256 = b, hex.EncodeToString(h[:])
	source.RawResponseBody, source.RawResponseSHA256 = raw, hex.EncodeToString(rawHash[:])
}

// ABI encoding of the documented 14-field DLN Order tuple; dynamic byte
// members use tuple-relative offsets. Source and take quantities intentionally differ.
func dlnTestEventData(orderID, giveAmount, takeChain, giveToken, takeToken, receiver string) string {
	words := make([]string, 14)
	words[0] = abiWord("1")
	words[2] = abiWord("1")
	words[4] = abiWord(giveAmount)
	words[5] = abiWord(takeChain)
	words[7] = abiWord("9")
	for _, i := range []int{1, 3, 6, 8, 9, 10, 11, 12, 13} {
		words[i] = abiWord("0")
	}
	enc := func(s string) string {
		v := strings.TrimPrefix(s, "0x")
		return abiWord(fmt.Sprint(len(v)/2)) + v + strings.Repeat("0", (64-len(v)%64)%64)
	}
	payloads := map[int]string{1: "0x00000000000000000000000000000000000000aa", 3: giveToken, 6: takeToken, 8: receiver, 9: "0x0000000000000000000000000000000000000044", 10: "0x0000000000000000000000000000000000000055", 11: "0x0000000000000000000000000000000000000066", 12: "0x0000000000000000000000000000000000000077", 13: "0x"}
	var dynamic strings.Builder
	offset := 14 * 32
	for _, i := range []int{1, 3, 6, 8, 9, 10, 11, 12, 13} {
		words[i] = abiWord(fmt.Sprint(offset))
		data := enc(payloads[i])
		dynamic.WriteString(data)
		offset += len(data) / 2
	}
	tuple := strings.Join(words, "") + dynamic.String()
	tupleBody := tuple
	affiliateOffset := 7*32 + len(tupleBody)/2
	metadataOffset := affiliateOffset + 32
	head := abiWord(fmt.Sprint(7*32)) + strings.TrimPrefix(orderID, "0x") + abiWord(fmt.Sprint(affiliateOffset)) + abiWord("0") + abiWord("0") + abiWord("0") + abiWord(fmt.Sprint(metadataOffset))
	return "0x" + head + tupleBody + abiWord("0") + abiWord("0")
}
func dlnTestFulfilledData(orderID, giveAmount, takeChain, giveToken, takeToken, receiver string) string {
	orderData := strings.TrimPrefix(dlnTestEventData(orderID, giveAmount, takeChain, giveToken, takeToken, receiver), "0x")
	// Current deployed event carries Order, orderId, actualFulfillAmount, sender,
	// and unlockAuthority. Eight demonstrates a partial fill against takeAmount 9.
	return "0x" + abiWord(fmt.Sprint(5*32)) + strings.TrimPrefix(orderID, "0x") + abiWord("8") + abiWord("1") + abiWord("2") + orderData[7*64:]
}
func replaceDLNDataWord(data string, index int, word string) string {
	b := strings.TrimPrefix(data, "0x")
	start := index * 64
	return "0x" + b[:start] + word + b[start+64:]
}
func abiWord(v string) string     { n := newBigInt(v); return fmt.Sprintf("%064x", n) }
func newBigInt(v string) *big.Int { n := new(big.Int); n.SetString(v, 10); return n }
