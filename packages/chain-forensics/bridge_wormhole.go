package chainforensics

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const (
	wormholeMessagePublishedTopic = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"
	wormholeTransferRedeemedTopic = "0xcaf280c8cfeba144da67230d9b009c8f868a75bac9a528fa0474be1ba317c169"
	wormholeEthereumCore          = "0x98f3c9e6e3face36baad05fe09d375ef1464288b"
	wormholeArbitrumCore          = "0xa5f208e072434bc67592e4c49c1b991ba79bca46"
	wormholeBaseCore              = "0xbebdb6c8ddc678ffa9f8748f85c815c556dd8ac6"
	wormholeEthereumTokenBridge   = "0x3ee18b2214aff97000d974cf647e7c347e8fa585"
	wormholeArbitrumTokenBridge   = "0x0b2402144bb366a632d14b83f244d2e0e21bd39c"
	wormholeBaseTokenBridge       = "0x8d2de8d2f73f1f4cab472ac9a881c9b123c79627"
)

// wormholeTokenBridgeEdges correlates an EVM Token Bridge redemption with
// its source Core message using Wormhole's emitter-chain/address/sequence
// tuple. It intentionally does not decode arbitrary payloads or infer token
// unit continuity. Only the documented Ethereum and Arbitrum deployments are
// admitted; adding networks requires a source-backed address and chain ID.
func wormholeTokenBridgeEdges(c Case) []Edge {
	type message struct {
		chain, emitter, sequence, tx string
		sourceID                     string
		amount, toChain, recipient   string
	}
	var messages []message
	for _, item := range receiptSources(c) {
		for _, log := range item.receipt.Logs {
			chainID, ok := wormholeCoreChain(log.Address)
			if !ok || len(log.Topics) != 2 || !strings.EqualFold(log.Topics[0], wormholeMessagePublishedTopic) {
				continue
			}
			words, ok := decodeWords(log.Data)
			if !ok || len(words) < 4 {
				continue
			}
			sequence, valid := parseWordDecimal(words[0])
			if !valid || len(words[0]) != 64 || strings.Trim(words[0][:48], "0") != "" {
				continue
			}
			emitter := strings.ToLower(log.Topics[1])
			if !evmHashPattern.MatchString(emitter) || strings.Trim(emitter[2:26], "0") != "" {
				continue
			}
			sender := topicAddress(log.Topics[1])
			if !strings.EqualFold(sender, wormholeTokenBridgeForCoreChain(chainID)) {
				continue
			}
			payload, ok := wormholePublishedPayload(log.Data)
			if !ok {
				continue
			}
			messages = append(messages, message{chain: chainID, emitter: emitter, sequence: sequence, tx: strings.ToLower(item.receipt.TransactionHash), sourceID: item.source.ID, amount: payload.amount, toChain: payload.toChain, recipient: payload.recipient})
		}
	}
	var edges []Edge
	for _, item := range receiptSources(c) {
		for _, log := range item.receipt.Logs {
			if !wormholeTokenBridge(log.Address) || len(log.Topics) != 4 || !strings.EqualFold(log.Topics[0], wormholeTransferRedeemedTopic) {
				continue
			}
			chainID, valid := parseWordDecimal(strings.TrimPrefix(strings.ToLower(log.Topics[1]), "0x"))
			if !valid || len(strings.TrimPrefix(log.Topics[1], "0x")) != 64 {
				continue
			}
			emitter, sequence := strings.ToLower(log.Topics[2]), ""
			sequence, valid = parseWordDecimal(strings.TrimPrefix(strings.ToLower(log.Topics[3]), "0x"))
			if !valid || !evmHashPattern.MatchString(emitter) {
				continue
			}
			destinationChain, ok := wormholeCoreChainForTokenBridge(log.Address)
			if !ok {
				continue
			}
			for _, msg := range messages {
				destinationChainID, known := wormholeChainID(destinationChain)
				if !known || msg.chain != chainID || msg.emitter != emitter || msg.sequence != sequence || msg.toChain != destinationChainID {
					continue
				}
				from, sourceMatches := uniqueBridgeOccurrence(c, chainFromWormholeID(msg.chain), msg.tx, msg.sourceID, wormholeTokenBridgeForCoreChain(msg.chain), msg.amount)
				to, destinationMatches := uniqueRecipientOccurrence(c, destinationChain, item.receipt.TransactionHash, item.source.ID, msg.recipient, msg.amount)
				if !sourceMatches || !destinationMatches || from.ID == to.ID {
					continue
				}
				id := fmt.Sprintf("bridge:wormhole:token-bridge:%s:%s:%s", msg.chain, msg.emitter, msg.sequence)
				edges = append(edges, Edge{ID: id, From: from.ID, To: to.ID, Grade: GradeProtocolCorrelated, Rule: "bridge.wormhole.token_bridge.transfer_redeemed_emitter_chain_address_sequence", EvidenceIDs: []string{from.ID, to.ID}, Caveats: []string{"Matches a Wormhole Core LogMessagePublished event to a Token Bridge TransferRedeemed event by emitter chain, emitter address, and sequence. This establishes protocol-message correlation only; it does not prove equal asset units, owner identity, control, intent, or exchange account."}})
			}
		}
	}
	return edges
}

func wormholeCoreChain(address string) (string, bool) {
	switch strings.ToLower(address) {
	case wormholeEthereumCore:
		return "2", true
	case wormholeArbitrumCore:
		return "23", true
	case wormholeBaseCore:
		return "30", true
	default:
		return "", false
	}
}

func wormholeTokenBridge(address string) bool {
	return strings.EqualFold(address, wormholeEthereumTokenBridge) || strings.EqualFold(address, wormholeArbitrumTokenBridge) || strings.EqualFold(address, wormholeBaseTokenBridge)
}

func wormholeCoreChainForTokenBridge(address string) (Chain, bool) {
	switch strings.ToLower(address) {
	case wormholeEthereumTokenBridge:
		return ChainEthereum, true
	case wormholeArbitrumTokenBridge:
		return ChainArbitrum, true
	case wormholeBaseTokenBridge:
		return ChainBase, true
	default:
		return "", false
	}
}

func wormholeTokenBridgeForCoreChain(id string) string {
	switch id {
	case "2":
		return wormholeEthereumTokenBridge
	case "23":
		return wormholeArbitrumTokenBridge
	case "30":
		return wormholeBaseTokenBridge
	default:
		return ""
	}
}

func wormholeChainID(chain Chain) (string, bool) {
	switch chain {
	case ChainEthereum:
		return "2", true
	case ChainArbitrum:
		return "23", true
	case ChainBase:
		return "30", true
	default:
		return "", false
	}
}

func chainFromWormholeID(id string) Chain {
	switch id {
	case "2":
		return ChainEthereum
	case "23":
		return ChainArbitrum
	case "30":
		return ChainBase
	default:
		return ""
	}
}

func uniqueBridgeOccurrence(c Case, chain Chain, tx, sourceID, bridge, amount string) (Occurrence, bool) {
	var found Occurrence
	count := 0
	for _, occurrence := range c.Occurrences {
		if occurrence.Chain == chain && strings.EqualFold(occurrence.Transaction, tx) && occurrence.SourceID == sourceID && strings.EqualFold(occurrence.To, bridge) && occurrence.Atomic == amount && occurrence.Kind == EvidenceObservation && occurrence.Measurement != MeasurementNetBalanceDelta {
			found = occurrence
			count++
		}
	}
	return found, count == 1
}

func uniqueRecipientOccurrence(c Case, chain Chain, tx, sourceID, recipient, amount string) (Occurrence, bool) {
	var found Occurrence
	count := 0
	for _, occurrence := range c.Occurrences {
		if occurrence.Chain == chain && strings.EqualFold(occurrence.Transaction, tx) && occurrence.SourceID == sourceID && strings.EqualFold(occurrence.To, recipient) && occurrence.Atomic == amount && occurrence.Kind == EvidenceObservation && occurrence.Measurement != MeasurementNetBalanceDelta {
			found = occurrence
			count++
		}
	}
	return found, count == 1
}

type wormholeTokenPayload struct {
	amount, toChain, recipient string
}

// wormholePublishedPayload reads only the common prefix of Wormhole Token
// Bridge Transfer and TransferWithPayload messages. Unknown payload types,
// malformed ABI offsets, and non-EVM recipients remain unresolved.
func wormholePublishedPayload(data string) (wormholeTokenPayload, bool) {
	encoded, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(data), "0x"))
	if err != nil || len(encoded) < 160 {
		return wormholeTokenPayload{}, false
	}
	offsetBig := new(big.Int).SetBytes(encoded[64:96])
	if !offsetBig.IsUint64() {
		return wormholeTokenPayload{}, false
	}
	offset := offsetBig.Uint64()
	if offset%32 != 0 || offset > uint64(len(encoded)) || uint64(len(encoded))-offset < 32 {
		return wormholeTokenPayload{}, false
	}
	lengthBig := new(big.Int).SetBytes(encoded[offset : offset+32])
	if !lengthBig.IsUint64() {
		return wormholeTokenPayload{}, false
	}
	length := lengthBig.Uint64()
	start := offset + 32
	if length > uint64(len(encoded))-start || length < 133 {
		return wormholeTokenPayload{}, false
	}
	payload := encoded[start : start+length]
	if payload[0] != 1 && payload[0] != 3 {
		return wormholeTokenPayload{}, false
	}
	for _, b := range payload[67:79] {
		if b != 0 {
			return wormholeTokenPayload{}, false
		}
	}
	amount := new(big.Int).SetBytes(payload[1:33]).String()
	toChain := new(big.Int).SetBytes(payload[99:101]).String()
	if amount == "0" || toChain == "0" {
		return wormholeTokenPayload{}, false
	}
	return wormholeTokenPayload{amount: amount, toChain: toChain, recipient: "0x" + hex.EncodeToString(payload[79:99])}, true
}
