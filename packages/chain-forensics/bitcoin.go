package chainforensics

import (
	"fmt"
	"strconv"
	"time"
)

// MempoolTransaction is the subset of mempool.space's public REST transaction
// response needed for deterministic Bitcoin event normalization.
type MempoolTransaction struct {
	TxID   string `json:"txid"`
	Status struct {
		Confirmed   bool   `json:"confirmed"`
		BlockHeight uint64 `json:"block_height"`
		BlockTime   int64  `json:"block_time"`
	} `json:"status"`
	Inputs []struct {
		Prevout struct {
			Address string `json:"scriptpubkey_address"`
			Value   uint64 `json:"value"`
		} `json:"prevout"`
		PreviousTx    string `json:"txid"`
		PreviousIndex uint32 `json:"vout"`
	} `json:"vin"`
	Outputs []struct {
		Address string `json:"scriptpubkey_address"`
		Value   uint64 `json:"value"`
	} `json:"vout"`
}

// NormalizeBitcoinTransaction expands a mempool.space transaction into
// exact input and output occurrences. Input/output event identities include
// their transaction-local index; a quantity or timestamp never identifies an
// occurrence by itself.
func NormalizeBitcoinTransaction(source Source, tx MempoolTransaction) ([]Occurrence, error) {
	if tx.TxID == "" || tx.Status.BlockHeight == 0 || tx.Status.BlockTime <= 0 {
		return nil, fmt.Errorf("transaction %q is unconfirmed or missing its chain position", tx.TxID)
	}
	if source.ID == "" || source.Name == "" || source.URL == "" || source.Retrieved == "" || !sha256Pattern.MatchString(source.SHA256) {
		return nil, fmt.Errorf("transaction %s has incomplete source provenance", tx.TxID)
	}
	timestamp := time.Unix(tx.Status.BlockTime, 0).UTC().Format(time.RFC3339)
	out := make([]Occurrence, 0, len(tx.Inputs)+len(tx.Outputs))
	for i, input := range tx.Inputs {
		id := fmt.Sprintf("bitcoin:%s:in:%d", tx.TxID, i)
		o := Occurrence{
			ID: id, Chain: ChainBitcoin, Transaction: tx.TxID,
			Block: tx.Status.BlockHeight, Timestamp: timestamp, EventIndex: uint32(i),
			Direction: "input", From: input.Prevout.Address, Asset: "BTC",
			Atomic: strconv.FormatUint(input.Prevout.Value, 10), Scale: 8,
			Kind: EvidenceObservation, SourceID: source.ID,
			PreviousTx: input.PreviousTx, PreviousIndex: input.PreviousIndex,
		}
		out = append(out, o)
	}
	for i, output := range tx.Outputs {
		id := fmt.Sprintf("bitcoin:%s:out:%d", tx.TxID, i)
		o := Occurrence{
			ID: id, Chain: ChainBitcoin, Transaction: tx.TxID,
			Block: tx.Status.BlockHeight, Timestamp: timestamp, EventIndex: uint32(i),
			Direction: "output", To: output.Address, Asset: "BTC",
			Atomic: strconv.FormatUint(output.Value, 10), Scale: 8,
			Kind: EvidenceObservation, SourceID: source.ID,
		}
		out = append(out, o)
	}
	return out, nil
}

// BuildBitcoinOutpointEdges links a spending input only to the exact previous
// transaction output it names. Missing previous outputs are left unresolved;
// callers must not substitute amount/time/address similarity.
func BuildBitcoinOutpointEdges(occurrences []Occurrence) []Edge {
	outputs := make(map[string]Occurrence)
	for _, o := range occurrences {
		if o.Chain == ChainBitcoin && o.Direction == "output" {
			outputs[fmt.Sprintf("%s:%d", o.Transaction, o.EventIndex)] = o
		}
	}
	var edges []Edge
	for _, in := range occurrences {
		if in.Chain != ChainBitcoin || in.Direction != "input" || in.PreviousTx == "" {
			continue
		}
		prev, ok := outputs[fmt.Sprintf("%s:%d", in.PreviousTx, in.PreviousIndex)]
		if !ok {
			continue
		}
		id := fmt.Sprintf("outpoint:%s:%d->%s:%d", in.PreviousTx, in.PreviousIndex, in.Transaction, in.EventIndex)
		edges = append(edges, Edge{
			ID: id, From: prev.ID, To: in.ID, Grade: GradeObserved,
			Rule:        "bitcoin.input.previous_outpoint_exact_match",
			EvidenceIDs: []string{prev.ID, in.ID},
		})
	}
	return edges
}
