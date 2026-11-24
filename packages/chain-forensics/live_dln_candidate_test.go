package chainforensics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetainedDLNLiveCandidatePinsRawSourceAndDestination(t *testing.T) {
	root := filepath.Join("testdata", "cf6-dln-live-order-candidate-20261116")
	source := readDLNCapture(t, filepath.Join(root, "ethereum-source-capture.json"), ChainEthereum,
		"0x4ae8877d82d382fdf855b5106f7a43eaa71ca2369ffcb7c76711b9e8b642b9e6", "0xbc4d47a3e98c319b039a2850fe812047dba9ac168659257142184c365a4827e4", "e60703bdbded4ec3765b161b43048a11b5aac718000c6424f30edc5c701b770c")
	destination := readDLNCapture(t, filepath.Join(root, "arbitrum-destination-capture.json"), ChainArbitrum,
		"0x1af40575cebdc4c2bdcc1ba416ccf9c8ca0c8069c8c97086355954a5a08029f6", "0x23a9316c94cc102c7903b103bc3175529fed6f20cf7126fe97a39713c3126856", "25a67c17ebe18c18bf0f962d35f48432430119e101b39d30660be8aa5abd5687")

	var sourceLog, destinationLog EVMLog
	for _, log := range source.Logs {
		if strings.EqualFold(log.Address, debridgeSourceAddress) && len(log.Topics) == 1 && strings.EqualFold(log.Topics[0], debridgeCreatedOrderTopic) {
			sourceLog = log
		}
	}
	if sourceLog.Address == "" {
		t.Fatal("pinned source receipt lacks the expected CreatedOrder event")
	}
	order, sourceOrderID, percentFee, ok := decodeDLNOrderEvent(sourceLog.Data, dlnSourceEvent)
	if !ok {
		t.Fatal("pinned source CreatedOrder tuple did not decode")
	}
	b, err := hex.DecodeString(strings.TrimPrefix(sourceLog.Data, "0x"))
	if err != nil || len(b) < 64 {
		t.Fatal("malformed source event data")
	}
	orderID := "0x" + hex.EncodeToString(b[32:64])
	if orderID != "0x3fc4925ab91392988a8d6109d28bcc6a7a73c79a8497ec428dbeb08a28961683" || sourceOrderID != orderID || percentFee != "4794275" || order.giveChain != "1" || order.giveAmount != "11980893267" || order.takeChain != "42161" || order.takeAmount != "11975620286" || !strings.EqualFold(order.giveToken, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48") || !strings.EqualFold(order.takeToken, "0xaf88d065e77c8cc2239327c5edb3a432268e5831") || !strings.EqualFold(order.receiver, "0xc013551d940e387d84d71a08f2a0539f0d0620fd") {
		t.Fatalf("unexpected exact source event identity/terms: id=%s order=%+v", orderID, order)
	}
	for _, log := range destination.Logs {
		if strings.EqualFold(log.Address, debridgeDestinationAddress) {
			destinationLog = log
		}
	}
	if destinationLog.Address == "" || len(destinationLog.Topics) != 1 || !strings.EqualFold(destinationLog.Topics[0], debridgeFulfilledOrderAmountTopic) {
		t.Fatalf("expected retained destination DLN event with the verified amount-bearing selector: %+v", destinationLog)
	}
	destinationOrder, destinationOrderID, actualFill, ok := decodeDLNDestinationEvent(destinationLog.Data, dlnDestinationEventWithActualAmount)
	if !ok || destinationOrderID != orderID || !sameDLNOrder(order, destinationOrder) || actualFill != "11975620286" {
		t.Fatalf("destination ABI did not reconcile exact order identity/terms/fill: orderId=%s fill=%s terms=%+v", destinationOrderID, actualFill, destinationOrder)
	}
	var matchingTransfer bool
	for _, occurrence := range destination.Occurrences {
		if strings.EqualFold(occurrence.Asset, "erc20:"+order.takeToken) && occurrence.Atomic == order.takeAmount && strings.EqualFold(occurrence.To, order.receiver) && strings.EqualFold(occurrence.Transaction, destination.TransactionHash) {
			matchingTransfer = true
		}
	}
	if !matchingTransfer {
		t.Fatal("destination receipt lacks the exact requested-token transfer to the source order receiver")
	}
	caseDoc := Case{ID: "dln-live-candidate", Private: true, Sources: append(source.Sources, destination.Sources...), Occurrences: append(source.Occurrences, destination.Occurrences...)}
	for _, s := range caseDoc.Sources {
		if strings.HasPrefix(s.ID, "evm-receipt-") {
			var receipt receiptEnvelope
			if json.Unmarshal(s.ResultBody, &receipt) != nil || !canonicalDLNReceipt(s, receipt) {
				t.Fatalf("captured receipt source %s failed exact-body/hash validation", s.ID)
			}
		}
	}
	edges := debridgeDLNEdges(caseDoc)
	if len(edges) != 1 || edges[0].Grade != GradeProtocolCorrelated || edges[0].ID != "bridge:debridge-dln:"+orderID {
		t.Fatalf("verified event identity and exact transfers did not create the expected DLN pair: %+v", edges)
	}
	if edges[0].From == "" || edges[0].To == "" {
		t.Fatalf("DLN correlation did not bind exact transfer occurrences: %+v", edges[0])
	}
	byID := make(map[string]Occurrence, len(caseDoc.Occurrences))
	for _, occurrence := range caseDoc.Occurrences {
		byID[occurrence.ID] = occurrence
	}
	from, to := byID[edges[0].From], byID[edges[0].To]
	if from.Atomic != "11985687542" || to.Atomic != actualFill || from.To != debridgeSourceAddress || !strings.EqualFold(to.To, order.receiver) {
		t.Fatalf("DLN pair failed exact source fee / destination fill binding: from=%+v to=%+v", from, to)
	}
	// Pin the independent explorer's deployed ABI decode of this exact tx.
	explorer, err := os.ReadFile(filepath.Join(root, "arbiscan-destination-verified-event.html"))
	if err != nil {
		t.Fatal(err)
	}
	explorerHash := sha256.Sum256(explorer)
	if hex.EncodeToString(explorerHash[:]) != "ac119e21d3dc31d43567eb2d55eda89c57761d7cf58c0979d2bb0ce3e3039c66" || !strings.Contains(string(explorer), destination.TransactionHash) || !strings.Contains(string(explorer), debridgeFulfilledOrderAmountTopic) || !strings.Contains(string(explorer), "actualFulfillAmount") || !strings.Contains(string(explorer), strings.TrimPrefix(orderID, "0x")) {
		t.Fatal("independent verified explorer artifact does not match the exact destination tx/event/order ABI")
	}
	// A pooled destination can receive an unrelated same-token/amount flow.
	// Transaction identity prevents it from becoming a second continuation.
	var target Occurrence
	for _, occurrence := range caseDoc.Occurrences {
		if occurrence.Transaction == destination.TransactionHash && strings.EqualFold(occurrence.Asset, "erc20:"+order.takeToken) && occurrence.Atomic == order.takeAmount && strings.EqualFold(occurrence.To, order.receiver) {
			target = occurrence
		}
	}
	unrelated := target
	unrelated.ID += ":unrelated-flow"
	unrelated.Transaction = "0x" + strings.Repeat("9", 64)
	caseDoc.Occurrences = append(caseDoc.Occurrences, unrelated)
	edges = debridgeDLNEdges(caseDoc)
	if len(edges) != 1 || edges[0].To != target.ID {
		t.Fatalf("unrelated same-token/amount flow was misattributed as bridge destination: %+v", edges)
	}
	// Two indistinguishable transfers inside the destination transaction make
	// the receipt-to-occurrence join ambiguous; the adapter must fail closed.
	duplicate := target
	duplicate.ID += ":duplicate-log"
	duplicate.EventIndex++
	caseDoc.Occurrences = append(caseDoc.Occurrences, duplicate)
	if edges := debridgeDLNEdges(caseDoc); len(edges) != 0 {
		t.Fatalf("ambiguous duplicate destination transfers created a protocol edge: %+v", edges)
	}
}

func readDLNCapture(t *testing.T, path string, chain Chain, tx, blockHash, expectedCaptureHash string) CapturedEVMTransaction {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var capture CapturedEVMTransaction
	if err := json.Unmarshal(b, &capture); err != nil {
		t.Fatal(err)
	}
	if capture.Chain != chain || !strings.EqualFold(capture.TransactionHash, tx) || !strings.EqualFold(capture.BlockHash, blockHash) || len(capture.Sources) < 3 {
		t.Fatalf("unexpected capture identity in %s: chain=%s tx=%s block=%s sources=%d", path, capture.Chain, capture.TransactionHash, capture.BlockHash, len(capture.Sources))
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != expectedCaptureHash {
		t.Fatalf("capture body hash changed for %s: %s", path, hex.EncodeToString(h[:]))
	}
	for _, source := range capture.Sources {
		resultHash := sha256.Sum256(source.ResultBody)
		rawHash := sha256.Sum256(source.RawResponseBody)
		if hex.EncodeToString(resultHash[:]) != source.SHA256 || hex.EncodeToString(rawHash[:]) != source.RawResponseSHA256 {
			t.Fatalf("source bytes/hash mismatch for %s", source.ID)
		}
	}
	return capture
}
