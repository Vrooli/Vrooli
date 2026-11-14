package chainforensics

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOPStackAdapterCorrelatesRetainedBALDWithdrawal(t *testing.T) {
	c := readBridgeFixture(t, "examples/cases/bald-base-rug-bridge/case.json")
	merged := MergeBridgeCorrelationEdges(c)
	if len(merged.Edges) != len(c.Edges)+1 {
		t.Fatalf("expected one protocol edge in addition to the two hypothesis edges, got %d edges", len(merged.Edges))
	}
	var found bool
	for _, edge := range merged.Edges {
		if edge.Rule == "bridge.op_stack.message_passed_and_successful_withdrawal_finalized" {
			found = true
			if edge.Grade != GradeProtocolCorrelated || edge.From != "base:0xb3b7282b333b4568dda371e9bb8807d0d0a669ccff0316ad2c06a8d5afdb9def:native" || edge.To != "ethereum:0x716be54e820732d7f3813c23f461dd463e01f05019665c60ce3d59e68b041f6a:internal:9" {
				t.Fatalf("unexpected OP Stack edge: %+v", edge)
			}
		}
	}
	if !found {
		t.Fatal("OP Stack withdrawal event and successful finalization did not correlate")
	}
	if again := MergeBridgeCorrelationEdges(merged); len(again.Edges) != len(merged.Edges) {
		t.Fatalf("bridge correlation is not idempotent: %d -> %d", len(merged.Edges), len(again.Edges))
	}
}

func TestArbitrumAdapterCorrelatesRealNitroOutboxWithdrawal(t *testing.T) {
	c := readBridgeFixture(t, "examples/cases/arbitrum-bridge-withdrawal/case.json")
	merged := MergeBridgeCorrelationEdges(c)
	if len(merged.Edges) != 1 {
		t.Fatalf("expected one Arbitrum protocol edge, got %d", len(merged.Edges))
	}
	edge := merged.Edges[0]
	if edge.Grade != GradeProtocolCorrelated || edge.Rule != "bridge.arbitrum.l2_to_l1_position_sender_target_value_match" || edge.From != "arbitrum:0x1d236964eb1a471838edd322cb7c9d2182c50948b09c41a374e97e65ed1359a5:native" || edge.To != "ethereum:0xa2cd09b80a2ed232e018294a3356c872a87094ee4f28b37868059e287591fbf0:internal:3" {
		t.Fatalf("unexpected Arbitrum edge: %+v", edge)
	}
	if err := Validate(merged); err != nil {
		t.Fatalf("correlated real-chain case failed validation: %v", err)
	}
}

func TestBridgeAdaptersRejectProtocolIdentityMismatches(t *testing.T) {
	t.Run("OP Stack mismatched withdrawal hash", func(t *testing.T) {
		c := readBridgeFixture(t, "examples/cases/bald-base-rug-bridge/case.json")
		for i := range c.Sources {
			var body map[string]any
			if json.Unmarshal(c.Sources[i].ResultBody, &body) != nil {
				continue
			}
			logs, _ := body["logs"].([]any)
			for _, raw := range logs {
				log, _ := raw.(map[string]any)
				topics, _ := log["topics"].([]any)
				if len(topics) == 2 && topics[0] == opWithdrawalFinalizedTopic {
					topics[1] = "0x" + strings.Repeat("1", 64)
					c.Sources[i].ResultBody, _ = json.Marshal(body)
					break
				}
			}
		}
		if got := len(MergeBridgeCorrelationEdges(c).Edges) - len(c.Edges); got != 0 {
			t.Fatalf("mismatched protocol hash produced %d edge(s)", got)
		}
	})

	t.Run("Arbitrum mismatched outbox position", func(t *testing.T) {
		c := readBridgeFixture(t, "examples/cases/arbitrum-bridge-withdrawal/case.json")
		for i := range c.Sources {
			if c.Sources[i].ID != "ethereum-arbitrum-outbox-transaction-decoded" {
				continue
			}
			var body map[string]any
			if err := json.Unmarshal(c.Sources[i].ResultBody, &body); err != nil {
				t.Fatal(err)
			}
			decoded, _ := body["decoded_input"].(map[string]any)
			params, _ := decoded["parameters"].([]any)
			for _, raw := range params {
				parameter, _ := raw.(map[string]any)
				if parameter["name"] == "index" {
					parameter["value"] = "163303"
				}
			}
			c.Sources[i].ResultBody, _ = json.Marshal(body)
		}
		if got := len(MergeBridgeCorrelationEdges(c).Edges); got != 0 {
			t.Fatalf("mismatched outbox position produced %d edge(s)", got)
		}
	})
}

func readBridgeFixture(t *testing.T, path string) Case {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := DecodeCase(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("decode bridge fixture %s: %v", path, err)
	}
	return c
}
