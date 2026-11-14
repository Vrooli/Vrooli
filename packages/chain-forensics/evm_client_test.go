package chainforensics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEVMRPCClientCapturesReceiptAndBlockAndUsesBriefCache(t *testing.T) {
	txHash := "0x" + repeatHex("1", 64)
	blockHash := "0x" + repeatHex("2", 64)
	from := "0x" + repeatHex("a", 40)
	to := "0x" + repeatHex("b", 40)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_getTransactionReceipt":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"transactionHash":%q,"blockHash":%q,"blockNumber":"0x10","logs":[{"address":"0x%s","topics":[%q,"0x%s%s","0x%s%s"],"data":"0x%s","blockNumber":"0x10","transactionHash":%q,"transactionIndex":"0x0","logIndex":"0x0"}]}}`, req.ID, txHash, blockHash, repeatHex("c", 40), ERC20TransferTopic, repeatHex("0", 24), repeatHex("a", 40), repeatHex("0", 24), repeatHex("b", 40), repeatHex("0", 63)+"7", txHash)
		case "eth_getBlockByHash":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"hash":%q,"number":"0x10","timestamp":"0x65920000"}}`, req.ID, blockHash)
		case "eth_getTransactionByHash":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"hash":%q,"blockNumber":"0x10","from":%q,"to":%q,"value":"0x0"}}`, req.ID, txHash, from, to)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "bad method", 400)
		}
	}))
	defer server.Close()
	client, err := NewEVMRPCClient(server.URL, nil, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.FetchERC20Transfers(context.Background(), ChainBase, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Occurrences) != 1 || first.Occurrences[0].From != from || first.Occurrences[0].To != to || first.Occurrences[0].Atomic != "7" || first.Occurrences[0].Asset != "erc20:0x"+repeatHex("c", 40) {
		t.Fatalf("unexpected normalized capture: %+v", first.Occurrences)
	}
	if len(first.Sources) != 3 || first.Sources[0].SHA256 == "" || first.Sources[0].RawResponseSHA256 == "" || len(first.Sources[0].RawResponseBody) == 0 || first.BlockHash != blockHash {
		t.Fatalf("missing source/hash provenance: %+v", first)
	}
	if err := Validate(Case{ID: "evm-source-test", Title: "test", Private: true, Sources: first.Sources, Occurrences: first.Occurrences}); err != nil {
		t.Fatalf("captured EVM sources failed raw-body validation: %v", err)
	}
	second, err := client.FetchERC20Transfers(context.Background(), ChainBase, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Occurrences) != 1 || calls.Load() != 3 {
		t.Fatalf("expected one receipt, block and transaction call across cache hit, calls=%d", calls.Load())
	}
}

func TestEVMRPCClientCapturesNativeValueOccurrence(t *testing.T) {
	txHash := "0x" + repeatHex("1", 64)
	blockHash := "0x" + repeatHex("2", 64)
	from := "0x" + repeatHex("a", 40)
	to := "0x" + repeatHex("b", 40)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_getTransactionReceipt":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"transactionHash":%q,"blockHash":%q,"blockNumber":"0x10","logs":[]}}`, req.ID, txHash, blockHash)
		case "eth_getBlockByHash":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"hash":%q,"number":"0x10","timestamp":"0x65920000"}}`, req.ID, blockHash)
		case "eth_getTransactionByHash":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"hash":%q,"blockNumber":"0x10","from":%q,"to":%q,"value":"0x5"}}`, req.ID, txHash, from, to)
		default:
			t.Errorf("unexpected method %s", req.Method)
			http.Error(w, "bad method", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client, err := NewEVMRPCClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := client.FetchERC20Transfers(context.Background(), ChainBase, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(captured.Occurrences) != 1 {
		t.Fatalf("expected native-value occurrence, got %#v", captured.Occurrences)
	}
	got := captured.Occurrences[0]
	if got.ID != "base:"+txHash+":native" || got.From != from || got.To != to || got.Asset != "ETH" || got.Atomic != "5" || got.Scale != 18 || got.SourceID != "evm-transaction-base-"+txHash {
		t.Fatalf("unexpected native-value occurrence: %+v", got)
	}
}

func TestEVMRPCClientRejectsReceiptBlockMismatchAndEndpointCredential(t *testing.T) {
	if _, err := NewEVMRPCClient("https://user:secret@example.com", nil, nil); err == nil {
		t.Fatal("credentialed endpoint accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"transactionHash":"0x%s","blockHash":"0x%s","blockNumber":"0x10","logs":[]}}`, req.ID, repeatHex("1", 64), repeatHex("2", 64))
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{"hash":"0x%s","number":"0x10","timestamp":"0x65920000"}}`, req.ID, repeatHex("3", 64))
	}))
	defer server.Close()
	c, err := NewEVMRPCClient(server.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FetchERC20Transfers(context.Background(), ChainEthereum, "0x"+repeatHex("1", 64))
	if err == nil {
		t.Fatal("mismatched block accepted")
	}
}

func repeatHex(s string, n int) string {
	out := ""
	for len(out) < n {
		out += s
	}
	return out[:n]
}
