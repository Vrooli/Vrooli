package chainforensics

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

const (
	opMessagePassedTopic       = "0x02a52367d10742d8032712c1bb8e0144ff1ec5ffda1ed7d70bb05a2744955054"
	opWithdrawalFinalizedTopic = "0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b"
	opMessagePasserAddress     = "0x4200000000000000000000000000000000000016"
	opPortalAddress            = "0x49048044d57e1c92a77f79988d21fa8faf74e97e"

	arbitrumL2ToL1TxTopic = "0x3e7aafa77dbf186b7fd488006beff893744caa3c4f6f299e8a709fa2087374fc"
	arbitrumArbSysAddress = "0x0000000000000000000000000000000000000064"
)

// MergeBridgeCorrelationEdges returns c with deterministic protocol-level
// correlation edges for bridge messages whose source-chain event and
// destination-chain execution are both present. It only consumes retained
// source bodies and exact occurrences; shared addresses or similar amounts
// alone never create an edge.
func MergeBridgeCorrelationEdges(c Case) Case {
	out := c
	out.Edges = append([]Edge(nil), c.Edges...)
	for _, edge := range append(opStackWithdrawalEdges(c), arbitrumWithdrawalEdges(c)...) {
		found := false
		for _, old := range out.Edges {
			if old.ID == edge.ID {
				found = true
				break
			}
		}
		if !found {
			out.Edges = append(out.Edges, edge)
		}
	}
	return out
}

type receiptEnvelope struct {
	TransactionHash string   `json:"transactionHash"`
	Logs            []EVMLog `json:"logs"`
}

func receiptSources(c Case) []struct {
	source  Source
	receipt receiptEnvelope
} {
	out := make([]struct {
		source  Source
		receipt receiptEnvelope
	}, 0)
	for _, source := range c.Sources {
		var r receiptEnvelope
		if len(source.ResultBody) == 0 || json.Unmarshal(source.ResultBody, &r) != nil || !evmHashPattern.MatchString(r.TransactionHash) || len(r.Logs) == 0 {
			continue
		}
		out = append(out, struct {
			source  Source
			receipt receiptEnvelope
		}{source: source, receipt: r})
	}
	return out
}

func occurrenceByTx(c Case, chain Chain, tx string, from string, to string, atomic string) (Occurrence, bool) {
	for _, occurrence := range c.Occurrences {
		if occurrence.Chain == chain && strings.EqualFold(occurrence.Transaction, tx) && strings.EqualFold(occurrence.From, from) && strings.EqualFold(occurrence.To, to) && occurrence.Atomic == atomic && occurrence.Asset == "ETH" {
			return occurrence, true
		}
	}
	return Occurrence{}, false
}

func sourceByID(c Case, id string) (Source, bool) {
	for _, source := range c.Sources {
		if source.ID == id {
			return source, true
		}
	}
	return Source{}, false
}

func nativeTxSourceMatches(c Case, occurrence Occurrence) bool {
	source, ok := sourceByID(c, occurrence.SourceID)
	if !ok {
		return false
	}
	var tx struct {
		Hash  string `json:"hash"`
		From  string `json:"from"`
		To    string `json:"to"`
		Value string `json:"value"`
	}
	if json.Unmarshal(source.ResultBody, &tx) != nil {
		return false
	}
	value, valid := decimalHex(tx.Value)
	return valid && strings.EqualFold(tx.Hash, occurrence.Transaction) && strings.EqualFold(tx.From, occurrence.From) && strings.EqualFold(tx.To, occurrence.To) && value == occurrence.Atomic
}

func internalTransferSourceMatches(c Case, occurrence Occurrence) bool {
	source, ok := sourceByID(c, occurrence.SourceID)
	if !ok {
		return false
	}
	var response struct {
		Items []struct {
			From  any    `json:"from"`
			To    any    `json:"to"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"items"`
	}
	if json.Unmarshal(source.ResultBody, &response) != nil {
		return false
	}
	for _, item := range response.Items {
		value, valid := decimalUint(item.Value)
		from, fromOK := addressString(item.From)
		to, toOK := addressString(item.To)
		if valid && fromOK && toOK && strings.EqualFold(from, occurrence.From) && strings.EqualFold(to, occurrence.To) && value == occurrence.Atomic && (item.Type == "call" || item.Type == "create" || item.Type == "selfdestruct") {
			return true
		}
	}
	return false
}

func addressString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case map[string]any:
		address, ok := typed["hash"].(string)
		return address, ok
	default:
		return "", false
	}
}

func opStackWithdrawalEdges(c Case) []Edge {
	type passed struct {
		tx, hash, sender, target, value string
	}
	var messages []passed
	future := receiptSources(c)
	for _, item := range future {
		for _, log := range item.receipt.Logs {
			if !strings.EqualFold(log.Address, opMessagePasserAddress) || len(log.Topics) != 4 || !strings.EqualFold(log.Topics[0], opMessagePassedTopic) {
				continue
			}
			data, ok := decodeWords(log.Data)
			if !ok || len(data) < 4 {
				continue
			}
			// MessagePassed has four non-indexed head words before its dynamic
			// bytes payload: value, gasLimit, data offset, withdrawalHash.
			hash := "0x" + data[3]
			value, valid := parseWordDecimal(data[0])
			if !valid || !evmHashPattern.MatchString(hash) {
				continue
			}
			messages = append(messages, passed{tx: strings.ToLower(item.receipt.TransactionHash), hash: hash, sender: topicAddress(log.Topics[2]), target: topicAddress(log.Topics[3]), value: value})
		}
	}
	finalized := make(map[string]struct {
		tx      string
		success bool
	})
	for _, item := range future {
		for _, log := range item.receipt.Logs {
			if !strings.EqualFold(log.Address, opPortalAddress) || len(log.Topics) != 2 || !strings.EqualFold(log.Topics[0], opWithdrawalFinalizedTopic) {
				continue
			}
			words, ok := decodeWords(log.Data)
			if !ok || len(words) != 1 || words[0] != strings.Repeat("0", 63)+"1" {
				continue
			}
			finalized[strings.ToLower(log.Topics[1])] = struct {
				tx      string
				success bool
			}{tx: strings.ToLower(item.receipt.TransactionHash), success: true}
		}
	}
	var edges []Edge
	for _, message := range messages {
		final, ok := finalized[message.hash]
		if !ok || !final.success {
			continue
		}
		from, foundFrom := occurrenceByTx(c, ChainBase, message.tx, message.sender, opMessagePasserAddress, message.value)
		to, foundTo := occurrenceByTx(c, ChainEthereum, final.tx, opPortalAddress, message.target, message.value)
		if !foundFrom || !foundTo || !nativeTxSourceMatches(c, from) || !internalTransferSourceMatches(c, to) {
			continue
		}
		edges = append(edges, Edge{ID: "bridge:op-stack:withdrawal:" + message.hash, From: from.ID, To: to.ID, Grade: GradeProtocolCorrelated, Rule: "bridge.op_stack.message_passed_and_successful_withdrawal_finalized", EvidenceIDs: []string{from.ID, to.ID}, Caveats: []string{"Correlates the OP Stack withdrawal message and successful finalization; it does not establish real-world identity, ownership, intent, or an exchange account."}})
	}
	return edges
}

type arbitrumDecoded struct {
	Hash         string `json:"hash"`
	Status       string `json:"status"`
	Method       string `json:"method"`
	DecodedInput struct {
		MethodCall string `json:"method_call"`
		Parameters []struct {
			Name  string `json:"name"`
			Value any    `json:"value"`
		} `json:"parameters"`
	} `json:"decoded_input"`
}

type arbExecution struct {
	tx, caller, target, value, position string
}

func arbitrumWithdrawalEdges(c Case) []Edge {
	var executions []arbExecution
	for _, source := range c.Sources {
		var tx arbitrumDecoded
		if len(source.ResultBody) == 0 || json.Unmarshal(source.ResultBody, &tx) != nil || !strings.Contains(tx.DecodedInput.MethodCall, "executeTransaction(") {
			continue
		}
		params := make(map[string]string, len(tx.DecodedInput.Parameters))
		for _, p := range tx.DecodedInput.Parameters {
			if v, ok := p.Value.(string); ok {
				params[p.Name] = v
			}
		}
		index, iok := params["index"]
		caller, cok := params["l2Sender"]
		target, tok := params["to"]
		value, vok := decimalUint(params["value"])
		if iok && cok && tok && vok && strings.EqualFold(tx.Status, "ok") && evmHashPattern.MatchString(tx.Hash) {
			executions = append(executions, arbExecution{tx: strings.ToLower(tx.Hash), caller: strings.ToLower(caller), target: strings.ToLower(target), value: value, position: index})
		}
	}
	var edges []Edge
	for _, item := range receiptSources(c) {
		for _, log := range item.receipt.Logs {
			if !strings.EqualFold(log.Address, arbitrumArbSysAddress) || len(log.Topics) != 4 || !strings.EqualFold(log.Topics[0], arbitrumL2ToL1TxTopic) {
				continue
			}
			words, ok := decodeWords(log.Data)
			if !ok || len(words) < 5 {
				continue
			}
			caller := "0x" + strings.ToLower(words[0][24:])
			value, valid := parseWordDecimal(words[4])
			if !valid {
				continue
			}
			target := topicAddress(log.Topics[1])
			position, valid := parseWordDecimal(strings.TrimPrefix(strings.ToLower(log.Topics[3]), "0x"))
			if !valid {
				continue
			}
			l2Tx := strings.ToLower(item.receipt.TransactionHash)
			fromOccurrence, foundFrom := occurrenceByTx(c, ChainArbitrum, l2Tx, caller, arbitrumArbSysAddress, value)
			if !foundFrom {
				continue
			}
			for _, execution := range executions {
				if execution.position != position || !strings.EqualFold(execution.caller, caller) || !strings.EqualFold(execution.target, target) || execution.value != value {
					continue
				}
				// The L1 outbox transfers value in an internal call; require the
				// occurrence to match the retained internal-transaction provider body.
				toOccurrence, foundTo := occurrenceByTxAnyFrom(c, ChainEthereum, execution.tx, target, value)
				if !foundTo || !internalTransferSourceMatches(c, toOccurrence) {
					continue
				}
				id := fmt.Sprintf("bridge:arbitrum:outbox:%s:%s", l2Tx, position)
				edges = append(edges, Edge{ID: id, From: fromOccurrence.ID, To: toOccurrence.ID, Grade: GradeProtocolCorrelated, Rule: "bridge.arbitrum.l2_to_l1_position_sender_target_value_match", EvidenceIDs: []string{fromOccurrence.ID, toOccurrence.ID}, Caveats: []string{"Correlates an ArbSys L2ToL1Tx event to a successful L1 Outbox executeTransaction by position, sender, target, and value; it does not establish real-world identity, ownership, intent, or an exchange account."}})
			}
		}
	}
	return edges
}

func occurrenceByTxAnyFrom(c Case, chain Chain, tx string, to string, atomic string) (Occurrence, bool) {
	for _, occurrence := range c.Occurrences {
		if occurrence.Chain == chain && strings.EqualFold(occurrence.Transaction, tx) && strings.EqualFold(occurrence.To, to) && occurrence.Atomic == atomic && occurrence.Asset == "ETH" {
			return occurrence, true
		}
	}
	return Occurrence{}, false
}

func decodeWords(value string) ([]string, bool) {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if len(value)%64 != 0 {
		return nil, false
	}
	words := make([]string, 0, len(value)/64)
	for len(value) > 0 {
		words = append(words, value[:64])
		value = value[64:]
	}
	return words, true
}

func topicAddress(topic string) string {
	value := strings.TrimPrefix(strings.ToLower(topic), "0x")
	if len(value) != 64 {
		return ""
	}
	return "0x" + value[24:]
}

func parseWordDecimal(word string) (string, bool) {
	value, ok := new(big.Int).SetString(word, 16)
	if !ok || value.Sign() < 0 {
		return "", false
	}
	return value.String(), true
}

func decimalUint(value string) (string, bool) {
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.Sign() < 0 {
		return "", false
	}
	return parsed.String(), true
}

func decimalHex(value string) (string, bool) {
	parsed, ok := new(big.Int).SetString(strings.TrimPrefix(strings.ToLower(value), "0x"), 16)
	if !ok || parsed.Sign() < 0 {
		return "", false
	}
	return parsed.String(), true
}
