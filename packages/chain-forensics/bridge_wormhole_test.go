package chainforensics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestWormholeAdapterCorrelatesTokenBridgeByVAAIdentity(t *testing.T) {
	c := wormholeFixture()
	var sourceReceipt receiptEnvelope
	if err := json.Unmarshal(c.Sources[0].ResultBody, &sourceReceipt); err != nil {
		t.Fatal(err)
	}
	payload, ok := wormholePublishedPayload(sourceReceipt.Logs[0].Data)
	if !ok || payload.amount != "100" || payload.toChain != "2" || payload.recipient != "0x0000000000000000000000000000000000000002" {
		t.Fatalf("unexpected decoded token payload: %+v, ok=%v data=%s", payload, ok, sourceReceipt.Logs[0].Data)
	}
	merged := MergeBridgeCorrelationEdges(c)
	if len(merged.Edges) != 1 {
		t.Fatalf("expected one Wormhole protocol edge, got %d", len(merged.Edges))
	}
	edge := merged.Edges[0]
	if edge.Grade != GradeProtocolCorrelated || edge.Rule != "bridge.wormhole.token_bridge.transfer_redeemed_emitter_chain_address_sequence" || edge.From != "arbitrum:wormhole-source" || edge.To != "ethereum:wormhole-destination" {
		t.Fatalf("unexpected Wormhole edge: %+v", edge)
	}
	if again := MergeBridgeCorrelationEdges(merged); len(again.Edges) != len(merged.Edges) {
		t.Fatalf("Wormhole merge is not idempotent: %d -> %d", len(merged.Edges), len(again.Edges))
	}
	if err := Validate(merged); err != nil {
		t.Fatalf("correlated Wormhole case failed validation: %v", err)
	}
}

func TestWormholeAdapterRejectsTupleAndDeploymentMismatches(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Case)
	}{
		{"sequence mismatch", func(c *Case) {
			editReceiptLog(c, "wormhole-redeemed", func(log *EVMLog) { log.Topics[3] = topicWord("556") })
		}},
		{"emitter chain mismatch", func(c *Case) {
			editReceiptLog(c, "wormhole-redeemed", func(log *EVMLog) { log.Topics[1] = topicWord("2") })
		}},
		{"emitter address mismatch", func(c *Case) {
			editReceiptLog(c, "wormhole-redeemed", func(log *EVMLog) {
				log.Topics[2] = wormholeTestTopicAddress("0x0000000000000000000000000000000000000001")
			})
		}},
		{"unrecognized core deployment", func(c *Case) {
			editReceiptLog(c, "wormhole-published", func(log *EVMLog) { log.Address = "0x0000000000000000000000000000000000000001" })
		}},
		{"non-Token-Bridge Core message sender", func(c *Case) {
			editReceiptLog(c, "wormhole-published", func(log *EVMLog) {
				log.Topics[1] = wormholeTestTopicAddress("0x0000000000000000000000000000000000000001")
			})
		}},
		{"unrecognized token bridge deployment", func(c *Case) {
			editReceiptLog(c, "wormhole-redeemed", func(log *EVMLog) { log.Address = "0x0000000000000000000000000000000000000001" })
		}},
		{"recipient mismatch", func(c *Case) {
			editReceiptLog(c, "wormhole-published", func(log *EVMLog) {
				log.Data = wormholePublishedEventData("555", "100", "2", "0x0000000000000000000000000000000000000003")
			})
		}},
		{"source bridge transfer amount mismatch", func(c *Case) {
			c.Occurrences[0].Atomic = "101"
		}},
		{"source occurrence uses another receipt", func(c *Case) {
			c.Occurrences[0].SourceID = "wormhole-redeemed"
		}},
		{"native net balance change is not a transfer", func(c *Case) {
			c.Occurrences[0].Measurement = MeasurementNetBalanceDelta
			c.Occurrences[0].Account = "11111111111111111111111111111111"
		}},
		{"recipient net balance change is not a transfer", func(c *Case) {
			c.Occurrences[1].Measurement = MeasurementNetBalanceDelta
			c.Occurrences[1].Account = "11111111111111111111111111111111"
		}},
		{"destination occurrence uses another receipt", func(c *Case) {
			c.Occurrences[1].SourceID = "wormhole-published"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := wormholeFixture()
			tt.edit(&c)
			if got := len(MergeBridgeCorrelationEdges(c).Edges); got != 0 {
				t.Fatalf("mismatched Wormhole evidence produced %d edge(s)", got)
			}
		})
	}
}

func TestWormholeAdapterRequiresUnambiguousTransactionOccurrences(t *testing.T) {
	c := wormholeFixture()
	duplicate := c.Occurrences[0]
	duplicate.ID += ":second"
	c.Occurrences = append(c.Occurrences, duplicate)
	if got := len(MergeBridgeCorrelationEdges(c).Edges); got != 0 {
		t.Fatalf("ambiguous source transaction produced %d edge(s)", got)
	}
}

func wormholeFixture() Case {
	sourceTx := "0x" + "1" + strings.Repeat("0", 63)
	destTx := "0x" + "2" + strings.Repeat("0", 63)
	recipient := "0x0000000000000000000000000000000000000002"
	sourceReceipt := receiptEnvelope{TransactionHash: sourceTx, Logs: []EVMLog{{Address: wormholeArbitrumCore, Topics: []string{wormholeMessagePublishedTopic, wormholeTestTopicAddress(wormholeArbitrumTokenBridge)}, Data: wormholePublishedEventData("555", "100", "2", recipient)}}}
	destReceipt := receiptEnvelope{TransactionHash: destTx, Logs: []EVMLog{{Address: wormholeEthereumTokenBridge, Topics: []string{wormholeTransferRedeemedTopic, topicWord("23"), wormholeTestTopicAddress(wormholeArbitrumTokenBridge), topicWord("555")}, Data: "0x"}}}
	sources := []Source{wormholeReceiptSource("wormhole-published", sourceReceipt), wormholeReceiptSource("wormhole-redeemed", destReceipt)}
	return Case{ID: "wormhole-protocol-fixture", Title: "Wormhole protocol correlation fixture", Private: true, Sources: sources, Occurrences: []Occurrence{
		{ID: "arbitrum:wormhole-source", Chain: ChainArbitrum, Transaction: sourceTx, To: wormholeArbitrumTokenBridge, Asset: "erc20:fixture-source", Atomic: "100", Kind: EvidenceObservation, SourceID: "wormhole-published"},
		{ID: "ethereum:wormhole-destination", Chain: ChainEthereum, Transaction: destTx, To: recipient, Asset: "erc20:fixture-destination", Atomic: "100", Kind: EvidenceObservation, SourceID: "wormhole-redeemed"},
	}, Gaps: []string{"Fixture uses protocol-shaped receipt logs; it is not a live bridge transfer capture."}}
}

func wormholePublishedEventData(sequence, amount, toChain, recipient string) string {
	address := strings.TrimPrefix(strings.ToLower(recipient), "0x")
	payload := "01" + wordBody(amount) + strings.Repeat("0", 24) + "1111111111111111111111111111111111111111" + "0002" + strings.Repeat("0", 24) + address + fmt.Sprintf("%04x", mustUint16(toChain)) + wordBody("0")
	length := len(payload) / 2
	padded := payload + strings.Repeat("0", ((32-length%32)%32)*2)
	return "0x" + wordBody(sequence) + wordBody("7") + wordBody("128") + wordBody("1") + fmt.Sprintf("%064x", length) + padded
}

func mustUint16(value string) uint64 {
	parsed, _ := new(big.Int).SetString(value, 10)
	return parsed.Uint64()
}

func wormholeReceiptSource(id string, receipt receiptEnvelope) Source {
	body, _ := json.Marshal(receipt)
	digest := sha256.Sum256(body)
	return Source{ID: id, Name: "Wormhole protocol event layout fixture", URL: "https://wormhole.com/docs/products/messaging/reference/core-contract-evm/", Retrieved: "2026-11-14T00:00:00Z", SHA256: hex.EncodeToString(digest[:]), ResultBody: body}
}

func editReceiptLog(c *Case, sourceID string, edit func(*EVMLog)) {
	for i := range c.Sources {
		if c.Sources[i].ID != sourceID {
			continue
		}
		var receipt receiptEnvelope
		_ = json.Unmarshal(c.Sources[i].ResultBody, &receipt)
		edit(&receipt.Logs[0])
		c.Sources[i] = wormholeReceiptSource(sourceID, receipt)
	}
}

func topicWord(decimal string) string {
	return "0x" + wordBody(decimal)
}

func wordBody(decimal string) string {
	n, _ := new(big.Int).SetString(decimal, 10)
	return fmt.Sprintf("%064x", n)
}

func wormholeTestTopicAddress(address string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}
