package chainforensics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	fixtureEVMAddress  = "0x1111111111111111111111111111111111111111"
	fixtureEVMContract = "0x2222222222222222222222222222222222222222"
	fixtureEVMBlock    = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestEVMWalletSnapshotPinsNativeAndTokenBalancesToStableBlock(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     uint64            `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		calls++
		var result any
		switch calls {
		case 1:
			if request.Method != "eth_chainId" {
				t.Errorf("call 1 method=%s", request.Method)
			}
			result = "0x2105"
		case 2, 6:
			wantBlock := "latest"
			if calls == 6 {
				wantBlock = "0x1f"
			}
			var gotBlock string
			if err := json.Unmarshal(request.Params[0], &gotBlock); err != nil || gotBlock != wantBlock {
				t.Errorf("block parameter=%s err=%v want=%s", request.Params[0], err, wantBlock)
			}
			result = map[string]string{"hash": fixtureEVMBlock, "number": "0x1f", "timestamp": "0x6553f100"}
		case 3:
			if request.Method != "eth_getBalance" || string(request.Params[1]) != `"0x1f"` {
				t.Errorf("native balance was not pinned: method=%s params=%s", request.Method, request.Params)
			}
			result = "0x123456789"
		case 4:
			var call struct {
				To   string `json:"to"`
				Data string `json:"data"`
			}
			if err := json.Unmarshal(request.Params[0], &call); err != nil {
				t.Error(err)
			}
			if request.Method != "eth_call" || call.To != fixtureEVMContract || !strings.HasPrefix(call.Data, "0x70a08231") || string(request.Params[1]) != `"0x1f"` {
				t.Errorf("bad balanceOf call: method=%s call=%#v params=%s", request.Method, call, request.Params)
			}
			result = "0x" + strings.Repeat("0", 62) + "2a"
		case 5:
			var call struct{ To, Data string }
			if err := json.Unmarshal(request.Params[0], &call); err != nil {
				t.Error(err)
			}
			if request.Method != "eth_call" || call.To != fixtureEVMContract || call.Data != "0x313ce567" {
				t.Errorf("bad decimals call: method=%s call=%#v", request.Method, call)
			}
			result = "0x" + strings.Repeat("0", 62) + "06"
		default:
			t.Fatalf("unexpected extra RPC call %d: %s", calls, request.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	client, err := NewEVMRPCClient(server.URL, server.Client(), func() time.Time { return time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	client.interval = 0
	result, err := client.FetchWalletSnapshot(context.Background(), ChainBase, fixtureEVMAddress, []string{fixtureEVMContract})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 6 || result.Snapshot.Block != 31 || result.Snapshot.BlockHash != fixtureEVMBlock || result.Snapshot.NativeAtomic != "4886718345" || len(result.Snapshot.Tokens) != 1 {
		t.Fatalf("incomplete snapshot: calls=%d result=%#v", calls, result)
	}
	token := result.Snapshot.Tokens[0]
	if token.Atomic != "42" || !token.ScaleKnown || token.Scale != 6 {
		t.Fatalf("token quantity/scale mismatch: %#v", token)
	}
	if len(result.Sources) != 5 || len(result.Snapshot.SourceIDs) != 5 {
		t.Fatalf("snapshot source provenance is incomplete: sources=%d ids=%d", len(result.Sources), len(result.Snapshot.SourceIDs))
	}
	for _, source := range result.Sources {
		if len(source.RawResponseBody) == 0 || source.RawResponseSHA256 == "" {
			t.Fatalf("wallet source does not retain the exact HTTP response: %+v", source)
		}
	}
}

func TestEVMWalletSnapshotRejectsReorgDuringRead(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		calls++
		var result any
		switch calls {
		case 1:
			result = "0x2105"
		case 2:
			result = map[string]string{"hash": fixtureEVMBlock, "number": "0x1f", "timestamp": "0x6553f100"}
		case 3:
			result = "0x0"
		case 4:
			result = map[string]string{"hash": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
		default:
			t.Fatalf("unexpected call %d (%s)", calls, request.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	client, err := NewEVMRPCClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client.interval = 0
	_, err = client.FetchWalletSnapshot(context.Background(), ChainBase, fixtureEVMAddress, nil)
	if err == nil || !strings.Contains(err.Error(), "block changed") {
		t.Fatalf("expected pinned-block reorg rejection, got %v", err)
	}
	if calls != 4 {
		t.Fatal(fmt.Sprintf("unexpected RPC call count %d", calls))
	}
}
