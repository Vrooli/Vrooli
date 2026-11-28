package chainforensics

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWormholeAdapterCorrelatesRetainedBaseToArbitrumPair(t *testing.T) {
	root := filepath.Join("testdata", "cf6-wormhole-live-pair-20261114")
	baseSource, baseBlockSource, baseReceipt, baseTimestamp := retainedRPCReceipt(t, root, "base-source-receipt.json", "base-source-block.json", "live-base-source", ChainBase, "https://mainnet.base.org")
	arbSource, arbBlockSource, arbReceipt, arbTimestamp := retainedRPCReceipt(t, root, "arbitrum-destination-receipt.json", "arbitrum-destination-block.json", "live-arbitrum-destination", ChainArbitrum, "https://arb1.arbitrum.io/rpc")
	operationBytes, err := os.ReadFile(filepath.Join(root, "wormholescan-operation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var operation struct {
		SourceChain struct {
			Transaction struct {
				Hash string `json:"txHash"`
			} `json:"transaction"`
			Status string `json:"status"`
		} `json:"sourceChain"`
		TargetChain struct {
			Transaction struct {
				Hash string `json:"txHash"`
			} `json:"transaction"`
			Status string `json:"status"`
		} `json:"targetChain"`
		Content struct {
			StandardizedProperties struct {
				AppIDs []string `json:"appIds"`
				From   uint64   `json:"fromChain"`
				To     uint64   `json:"toChain"`
			} `json:"standarizedProperties"`
		} `json:"content"`
	}
	if err := json.Unmarshal(operationBytes, &operation); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(operation.SourceChain.Transaction.Hash, baseReceipt.TransactionHash) || !strings.EqualFold(operation.TargetChain.Transaction.Hash, arbReceipt.TransactionHash) || operation.SourceChain.Status != "confirmed" || operation.TargetChain.Status != "completed" || operation.Content.StandardizedProperties.From != 30 || operation.Content.StandardizedProperties.To != 23 || !containsString(operation.Content.StandardizedProperties.AppIDs, "PORTAL_TOKEN_BRIDGE") {
		t.Fatalf("retained Wormholescan tuple does not match the paired RPC receipts: %+v", operation)
	}
	operationSource := sourceForBody("wormholescan-live-pair", "Wormholescan indexed operation", "https://api.wormholescan.io/api/v1/operations/30/0000000000000000000000008d2de8d2f73f1f4cab472ac9a881c9b123c79627/86175", "GET", time.Date(2026, 11, 14, 6, 6, 16, 0, time.UTC), operationBytes, "Public indexer metadata identifies a completed source/destination pair; it is secondary and not a substitute for chain RPC receipts.", operationBytes)

	occurrences := append(transferOccurrences(t, ChainBase, baseSource, baseReceipt, baseTimestamp), transferOccurrences(t, ChainArbitrum, arbSource, arbReceipt, arbTimestamp)...)
	caseDoc := Case{
		ID: "wormhole-live-pair-20240911", Title: "Private Wormhole Base to Arbitrum protocol-pair fixture", Private: true,
		Sources: []Source{operationSource, baseSource, baseBlockSource, arbSource, arbBlockSource}, Occurrences: occurrences,
		Gaps: []string{
			"Wormholescan's retained operation response identifies the transaction pair and reports source confirmed / target completed; this is an aggregator record, not a substitute for either raw chain receipt.",
			"The correlation matches the Wormhole message tuple and exact amount/recipient observations but does not prove unit continuity, token equivalence, sender identity, ownership, control, intent, or exchange custody.",
		},
	}
	merged := MergeBridgeCorrelationEdges(caseDoc)
	var wormholeEdges []Edge
	for _, edge := range merged.Edges {
		if strings.HasPrefix(edge.ID, "bridge:wormhole:") {
			wormholeEdges = append(wormholeEdges, edge)
		}
	}
	if len(wormholeEdges) != 1 {
		t.Fatalf("expected one live protocol correlation, got %d edges from %d transfer observations", len(wormholeEdges), len(occurrences))
	}
	if wormholeEdges[0].Grade != GradeProtocolCorrelated || wormholeEdges[0].From == "" || wormholeEdges[0].To == "" {
		t.Fatalf("unexpected live Wormhole correlation: %+v", wormholeEdges[0])
	}
	byID := make(map[string]Occurrence, len(occurrences))
	for _, occurrence := range occurrences {
		byID[occurrence.ID] = occurrence
	}
	from, to := byID[wormholeEdges[0].From], byID[wormholeEdges[0].To]
	if from.Chain != ChainBase || from.Asset != "erc20:0xff0c62a4979400841efaa6faadb07ac7d5c98b27" || !strings.EqualFold(from.To, wormholeBaseTokenBridge) || from.Atomic != "75966" || to.Chain != ChainArbitrum || to.Asset != "erc20:0xe4728f3e48e94c6da2b53610e677cc241dafb134" || !strings.EqualFold(to.To, "0xe70946692e2e56ae47bfae2d93d31bd60952b090") || to.Atomic != "75966" {
		t.Fatalf("live edge did not bind the expected exact transfer observations: from=%+v to=%+v", from, to)
	}
	if err := Validate(merged); err != nil {
		t.Fatalf("live paired case failed validation: %v", err)
	}

	// A pooled recipient can receive unrelated transfers with the same atomic
	// quantity. They must not become bridge continuations without appearing in
	// the exact redemption transaction and retained receipt.
	unrelated := to
	unrelated.ID += ":unrelated-same-amount"
	unrelated.Transaction = "0x" + strings.Repeat("9", 64)
	caseDoc.Occurrences = append(caseDoc.Occurrences, unrelated)
	withUnrelated := MergeBridgeCorrelationEdges(caseDoc)
	var correlated []Edge
	for _, edge := range withUnrelated.Edges {
		if strings.HasPrefix(edge.ID, "bridge:wormhole:") {
			correlated = append(correlated, edge)
		}
	}
	if len(correlated) != 1 || correlated[0].To != to.ID {
		t.Fatalf("same-amount transfer outside the redemption transaction created a false bridge edge: %+v", correlated)
	}
}

func retainedRPCReceipt(t *testing.T, root, receiptFile, blockFile, sourceID string, chain Chain, endpoint string) (Source, Source, receiptEnvelope, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, receiptFile))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil || len(response.Result) == 0 {
		t.Fatalf("decode retained receipt %s: %v", receiptFile, err)
	}
	var receipt receiptEnvelope
	if err := json.Unmarshal(response.Result, &receipt); err != nil {
		t.Fatalf("decode receipt result %s: %v", receiptFile, err)
	}
	blockRaw, err := os.ReadFile(filepath.Join(root, blockFile))
	if err != nil {
		t.Fatal(err)
	}
	var blockResponse struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(blockRaw, &blockResponse); err != nil {
		t.Fatalf("decode retained block %s: %v", blockFile, err)
	}
	var block struct {
		Hash      string `json:"hash"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(blockResponse.Result, &block); err != nil {
		t.Fatalf("decode block result %s: %v", blockFile, err)
	}
	var receiptDetails struct {
		BlockHash string `json:"blockHash"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(response.Result, &receiptDetails); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(receiptDetails.BlockHash, block.Hash) || receiptDetails.Status != "0x1" {
		t.Fatalf("receipt is not successful/canonical against retained block: %s", receiptFile)
	}
	blockSource := sourceForBody(sourceID+"-block", "Public EVM JSON-RPC block", endpoint, "eth_getBlockByNumber", time.Date(2026, 11, 14, 6, 9, 1, 0, time.UTC), blockResponse.Result, "Anchors the receipt's block hash and on-chain timestamp; exact HTTP response bytes are retained.", blockRaw)
	blockSeconds, ok := new(big.Int).SetString(strings.TrimPrefix(block.Timestamp, "0x"), 16)
	if !ok || !blockSeconds.IsInt64() {
		t.Fatalf("invalid block timestamp in %s", blockFile)
	}
	blockTimestamp := time.Unix(blockSeconds.Int64(), 0).UTC().Format(time.RFC3339)
	resultHash := time.Date(2026, 11, 14, 6, 9, 2, 0, time.UTC)
	source := sourceForBody(sourceID, "Public EVM JSON-RPC receipt", endpoint, "eth_getTransactionReceipt", resultHash, response.Result, "Successful receipt retained as a bounded read-only capture; exact HTTP response bytes retained.", raw)
	_ = chain // The caller uses this to normalize transfer logs after receipt validation.
	return source, blockSource, receipt, blockTimestamp
}

func transferOccurrences(t *testing.T, chain Chain, source Source, receipt receiptEnvelope, timestamp string) []Occurrence {
	t.Helper()
	var occurrences []Occurrence
	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 || !strings.EqualFold(log.Topics[0], ERC20TransferTopic) {
			continue
		}
		occurrence, err := NormalizeERC20TransferLog(chain, source, log, timestamp)
		if err != nil {
			t.Fatalf("normalize retained ERC-20 transfer log %s: %v", log.LogIndex, err)
		}
		occurrences = append(occurrences, occurrence)
	}
	return occurrences
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
