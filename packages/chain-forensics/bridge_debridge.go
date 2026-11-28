package chainforensics

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
)

// DLN EVM event identities and production proxy addresses are taken from the
// deBridge dln-contracts repository. The same proxies are deployed on several
// EVM networks; the chain field and the order's give/take chain IDs are checked
// separately, so a matching address alone is never sufficient.
const (
	debridgeSourceAddress             = "0xef4fb24ad0916217251f553c0596f8edc630eb66"
	debridgeDestinationAddress        = "0xe7351fd770a37282b91d153ee690b63579d6dd7f"
	debridgeCreatedOrderTopic         = "0xfc8703fd57380f9dd234a89dce51333782d49c5902f307b02f03e014d18fe471"
	debridgeFulfilledOrderTopic       = "0xd281ee92bab1446041582480d2c0a9dc91f855386bb27ea295faac1e992f7fe4"
	debridgeFulfilledOrderAmountTopic = "0xc164aca37b9805a1c9027b6f32260a069723a82926f6e9ece4926e4dd3ea8ecf"
)

type dlnOrder struct {
	giveChain, giveAmount, takeChain, takeAmount string
	giveToken, takeToken, receiver               string
}

type dlnLog struct {
	tx, sourceID, orderID string
	order                 dlnOrder
	fulfillAmount         string
	sourceTransferAmount  string
	chain                 Chain
}

// debridgeDLNEdges correlates CreatedOrder and FulfilledOrder by the
// non-indexed orderId and fully decoded order tuple from canonical receipt
// bodies. Asset quantities on the give and take sides remain independent;
// this is message continuity, not proof of equal value or economic identity.
func debridgeDLNEdges(c Case) []Edge {
	var created, fulfilled []dlnLog
	for _, item := range receiptSources(c) {
		if !canonicalDLNReceipt(item.source, item.receipt) {
			continue
		}
		for _, log := range item.receipt.Logs {
			if !canonicalDLNLog(item.receipt, log) || len(log.Topics) == 0 {
				continue
			}
			switch {
			case strings.EqualFold(log.Address, debridgeSourceAddress) && strings.EqualFold(log.Topics[0], debridgeCreatedOrderTopic):
				o, orderID, percentFee, ok := decodeDLNOrderEvent(log.Data, dlnSourceEvent)
				if !ok {
					continue
				}
				giveAmount, amountOK := new(big.Int).SetString(o.giveAmount, 10)
				feeAmount, feeOK := new(big.Int).SetString(percentFee, 10)
				if !amountOK || !feeOK {
					continue
				}
				created = append(created, dlnLog{tx: item.receipt.TransactionHash, sourceID: item.source.ID, orderID: orderID, order: o, sourceTransferAmount: new(big.Int).Add(giveAmount, feeAmount).String()})
			case strings.EqualFold(log.Address, debridgeDestinationAddress) && (strings.EqualFold(log.Topics[0], debridgeFulfilledOrderTopic) || strings.EqualFold(log.Topics[0], debridgeFulfilledOrderAmountTopic)):
				if len(log.Topics) != 1 {
					continue
				}
				eventKind := dlnDestinationEvent
				if strings.EqualFold(log.Topics[0], debridgeFulfilledOrderAmountTopic) {
					eventKind = dlnDestinationEventWithActualAmount
				}
				o, orderID, actual, ok := decodeDLNDestinationEvent(log.Data, eventKind)
				if !ok {
					continue
				}
				fulfilled = append(fulfilled, dlnLog{tx: item.receipt.TransactionHash, sourceID: item.source.ID, orderID: orderID, order: o, fulfillAmount: actual})
			}
		}
	}
	var edges []Edge
	for _, src := range created {
		for _, dst := range fulfilled {
			src.chain = dlnEventChain(c, src)
			dst.chain = dlnEventChain(c, dst)
			if src.chain == "" || dst.chain == "" || src.chain == dst.chain || src.orderID != dst.orderID || src.order.giveChain != evmChainIDFor(src.chain) || dst.order.giveChain != src.order.giveChain || src.order.takeChain != evmChainIDFor(dst.chain) || src.order.takeChain != dst.order.takeChain || !sameDLNOrder(src.order, dst.order) {
				continue
			}
			from, okFrom := dlnOccurrence(c, src, true)
			to, okTo := dlnOccurrence(c, dst, false)
			if !okFrom || !okTo || from.ID == to.ID {
				continue
			}
			edges = append(edges, Edge{ID: "bridge:debridge-dln:" + src.orderID, From: from.ID, To: to.ID, Grade: GradeProtocolCorrelated, Rule: "bridge.debridge_dln.created_order_fulfilled_order_order_id", EvidenceIDs: []string{from.ID, to.ID}, Caveats: []string{"Exact DLN orderId and order terms match across the documented source and destination deployments. Give and take token denominations and atomic quantities are distinct; fulfillment does not prove equal value, address ownership, intent, or exchange account control."}})
		}
	}
	return edges
}

func dlnEventChain(c Case, event dlnLog) Chain {
	var found Chain
	for _, o := range c.Occurrences {
		if strings.EqualFold(o.Transaction, event.tx) && o.SourceID == event.sourceID {
			if found != "" && found != o.Chain {
				return ""
			}
			found = o.Chain
		}
	}
	return found
}

func sameDLNOrder(a, b dlnOrder) bool {
	return a.giveChain == b.giveChain && a.giveAmount == b.giveAmount && a.takeChain == b.takeChain && a.takeAmount == b.takeAmount && a.giveToken == b.giveToken && a.takeToken == b.takeToken && a.receiver == b.receiver
}
func evmChainIDFor(c Chain) string {
	switch c {
	case ChainEthereum:
		return "1"
	case ChainBase:
		return "8453"
	case ChainArbitrum:
		return "42161"
	}
	return ""
}

func dlnOccurrence(c Case, event dlnLog, source bool) (Occurrence, bool) {
	chain := event.chain
	if evmChainIDFor(chain) == "" {
		return Occurrence{}, false
	}
	var found Occurrence
	n := 0
	for _, o := range c.Occurrences {
		if o.Chain != chain || !strings.EqualFold(o.Transaction, event.tx) || o.SourceID != event.sourceID || o.Kind != EvidenceObservation || o.Measurement == MeasurementNetBalanceDelta {
			continue
		}
		wantAsset := event.order.giveToken
		wantAmount := event.sourceTransferAmount
		if !source {
			wantAsset = event.order.takeToken
			wantAmount = event.fulfillAmount
		}
		if !strings.EqualFold(o.Asset, "erc20:"+wantAsset) && !(wantAsset == "0x0000000000000000000000000000000000000000" && o.Asset == "ETH") {
			continue
		}
		if o.Atomic != wantAmount {
			continue
		}
		if source && !strings.EqualFold(o.To, debridgeSourceAddress) {
			continue
		}
		if !source && !strings.EqualFold(o.To, event.order.receiver) {
			continue
		}
		found = o
		n++
	}
	return found, n == 1
}

// canonicalDLNReceipt pins the event scan to a hash-verified successful RPC
// receipt and its exact result object. A copied or internally inconsistent
// log is not eligible for cross-chain correlation.
func canonicalDLNReceipt(source Source, receipt receiptEnvelope) bool {
	if !evmHashPattern.MatchString(receipt.TransactionHash) || !evmHashPattern.MatchString(receipt.BlockHash) || !strings.EqualFold(receipt.Status, "0x1") {
		return false
	}
	blockNumber, blockErr := parseEVMIndex(receipt.BlockNumber)
	_, txIndexErr := parseEVMIndex(receipt.TransactionIndex)
	if blockErr != nil || blockNumber == 0 || txIndexErr != nil || len(receipt.Logs) == 0 {
		return false
	}
	if len(source.ResultBody) == 0 || !sha256Pattern.MatchString(source.SHA256) || len(source.RawResponseBody) == 0 || !sha256Pattern.MatchString(source.RawResponseSHA256) {
		return false
	}
	resultHash := sha256.Sum256(source.ResultBody)
	rawHash := sha256.Sum256(source.RawResponseBody)
	if hex.EncodeToString(resultHash[:]) != source.SHA256 || hex.EncodeToString(rawHash[:]) != source.RawResponseSHA256 {
		return false
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(source.RawResponseBody, &envelope) != nil || envelope.JSONRPC != "2.0" || envelope.Error != nil || !bytes.Equal(bytes.TrimSpace(envelope.Result), bytes.TrimSpace(source.ResultBody)) {
		return false
	}
	return true
}

func canonicalDLNLog(receipt receiptEnvelope, log EVMLog) bool {
	if !strings.EqualFold(log.TransactionHash, receipt.TransactionHash) || !strings.EqualFold(log.BlockHash, receipt.BlockHash) {
		return false
	}
	blockNumber, blockErr := parseEVMIndex(log.BlockNumber)
	receiptBlock, receiptBlockErr := parseEVMIndex(receipt.BlockNumber)
	transactionIndex, txIndexErr := parseEVMIndex(log.TransactionIndex)
	receiptTxIndex, receiptTxIndexErr := parseEVMIndex(receipt.TransactionIndex)
	_, logIndexErr := parseEVMIndex(log.LogIndex)
	return blockErr == nil && receiptBlockErr == nil && blockNumber == receiptBlock && txIndexErr == nil && receiptTxIndexErr == nil && transactionIndex == receiptTxIndex && logIndexErr == nil
}

type dlnEventKind uint8

const (
	dlnSourceEvent dlnEventKind = iota
	dlnDestinationEvent
	dlnDestinationEventWithActualAmount
)

// DLN event arguments are non-indexed. CreatedOrder is (Order, orderId, ...);
// older FulfilledOrder is (Order, orderId, sender, unlockAuthority), while the
// deployed ABI variant additionally carries actualFulfillAmount after orderId.
// Decode only these exact declared layouts; source terms and transfer
// occurrences are still validated separately before an edge can be emitted.
func decodeDLNOrderEvent(data string, kind dlnEventKind) (dlnOrder, string, string, bool) {
	b, err := decodeDLNEventBytes(data)
	if err != nil || len(b) < 64 {
		return dlnOrder{}, "", "", false
	}
	o, ok := decodeDLNOrder(data)
	if !ok {
		return dlnOrder{}, "", "", false
	}
	orderID := "0x" + hex.EncodeToString(b[32:64])
	if kind == dlnSourceEvent {
		// Verify the top-level tuple offset is after CreatedOrder's 7-word head.
		offset, valid := abiOffset(b, 0)
		if !valid || offset != 7*32 || len(b) < int(offset+14*32) {
			return dlnOrder{}, "", "", false
		}
		// The source event emits percentFee separately; the pinned live source
		// transfer equals giveAmount + percentFee. nativeFixFee is separate.
		return o, orderID, new(big.Int).SetBytes(b[4*32 : 5*32]).String(), true
	} else {
		return dlnOrder{}, "", "", false
	}
}

func decodeDLNDestinationEvent(data string, kind dlnEventKind) (dlnOrder, string, string, bool) {
	b, err := decodeDLNEventBytes(data)
	if err != nil || len(b) < 5*32 {
		return dlnOrder{}, "", "", false
	}
	offset, valid := abiOffset(b, 0)
	if !valid || offset != 5*32 && kind == dlnDestinationEventWithActualAmount || offset != 4*32 && kind == dlnDestinationEvent {
		return dlnOrder{}, "", "", false
	}
	o, ok := decodeDLNOrder(data)
	if !ok {
		return dlnOrder{}, "", "", false
	}
	orderID := "0x" + hex.EncodeToString(b[32:64])
	actual := o.takeAmount
	if kind == dlnDestinationEventWithActualAmount {
		actualNumber := new(big.Int).SetBytes(b[64:96])
		requested, parsed := new(big.Int).SetString(o.takeAmount, 10)
		actual = actualNumber.String()
		if !parsed || actualNumber.Sign() <= 0 || actualNumber.Cmp(requested) > 0 {
			return dlnOrder{}, "", "", false
		}
	}
	return o, orderID, actual, true
}

func decodeDLNEventBytes(data string) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(data), "0x"))
	if err != nil || len(b) < 32 || len(b) > 64*1024 || len(b)%32 != 0 {
		return nil, errors.New("invalid DLN event data")
	}
	return b, nil
}

// decodeDLNOrder accepts the current event ABI's outer tuple offset and the
// tuple's 14-word head. Dynamic byte values are capped at 4 KiB; total input is
// capped at 64 KiB. Only identity/amount fields needed by this adapter are
// interpreted, while all offsets and word boundaries are strictly checked.
func decodeDLNOrder(data string) (dlnOrder, bool) {
	b, err := decodeDLNEventBytes(data)
	if err != nil {
		return dlnOrder{}, false
	}
	outer, ok := abiOffset(b, 0)
	if !ok || outer+14*32 > uint64(len(b)) {
		return dlnOrder{}, false
	}
	base := outer
	w := func(i int) []byte { return b[base+uint64(i*32) : base+uint64((i+1)*32)] }
	readBytes := func(i int) ([]byte, bool) {
		off, ok := abiOffset(b, base+uint64(i*32))
		if !ok || off < 14*32 || base+off+32 > uint64(len(b)) {
			return nil, false
		}
		p := base + off
		n := new(big.Int).SetBytes(b[p : p+32])
		if !n.IsUint64() || n.Uint64() > 4096 {
			return nil, false
		}
		l := n.Uint64()
		if p+32+l > uint64(len(b)) {
			return nil, false
		}
		return b[p+32 : p+32+l], true
	}
	// Validate every dynamic bytes field in the tuple, including terms this
	// correlation rule does not interpret. Offsets must follow ABI declaration
	// order and their encoded spans may not overlap.
	previousEnd := uint64(14 * 32)
	for _, i := range []int{1, 3, 6, 8, 9, 10, 11, 12, 13} {
		off, valid := abiOffset(b, base+uint64(i*32))
		if !valid || off < previousEnd || base+off+32 > uint64(len(b)) {
			return dlnOrder{}, false
		}
		length := new(big.Int).SetBytes(b[base+off : base+off+32])
		if !length.IsUint64() || length.Uint64() > 4096 {
			return dlnOrder{}, false
		}
		padded := (length.Uint64() + 31) / 32 * 32
		if base+off+32+padded > uint64(len(b)) {
			return dlnOrder{}, false
		}
		previousEnd = off + 32 + padded
	}
	uint := func(i int) string { return new(big.Int).SetBytes(w(i)).String() }
	giveToken, ok1 := readBytes(3)
	takeToken, ok2 := readBytes(6)
	receiver, ok3 := readBytes(8)
	if !ok1 || !ok2 || !ok3 || len(giveToken) == 0 || len(takeToken) == 0 || len(receiver) == 0 {
		return dlnOrder{}, false
	}
	// EVM address byte strings must be exactly 20 bytes. Other chain engines
	// are intentionally unresolved by this EVM adapter.
	if len(giveToken) != 20 || len(takeToken) != 20 || len(receiver) != 20 {
		return dlnOrder{}, false
	}
	return dlnOrder{giveChain: uint(2), giveAmount: uint(4), takeChain: uint(5), takeAmount: uint(7), giveToken: "0x" + hex.EncodeToString(giveToken), takeToken: "0x" + hex.EncodeToString(takeToken), receiver: "0x" + hex.EncodeToString(receiver)}, true
}
func abiOffset(b []byte, at uint64) (uint64, bool) {
	if at+32 > uint64(len(b)) {
		return 0, false
	}
	n := new(big.Int).SetBytes(b[at : at+32])
	if !n.IsUint64() {
		return 0, false
	}
	v := n.Uint64()
	return v, v%32 == 0 && v <= uint64(len(b))
}
