package chainforensics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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

// MaterializeBitcoinPrevoutOutputs creates output observations for the exact
// prevout records embedded in confirmed transaction inputs. A mempool.space
// transaction response includes the referenced txid/vout, script address and
// value alongside each input; the resulting output node is therefore
// source-backed at the spending transaction, even when the parent transaction
// body has not also been captured. It deliberately leaves the parent's block
// and time unknown. Existing full output observations are retained after
// their address/value are checked against the embedded prevout data.
//
// The returned edges are exact outpoint joins only. They do not establish the
// parent transaction's own inputs or the origin of the output's value.
func MaterializeBitcoinPrevoutOutputs(inputs, existing []Occurrence) ([]Occurrence, []Edge, error) {
	outputs := make(map[string]Occurrence, len(existing))
	for _, occurrence := range existing {
		if occurrence.Chain == ChainBitcoin && occurrence.Direction == "output" {
			outputs[fmt.Sprintf("%s:%d", occurrence.Transaction, occurrence.EventIndex)] = occurrence
		}
	}
	materialized := make([]Occurrence, 0, len(inputs))
	seen := make(map[string]Occurrence, len(inputs))
	for _, input := range inputs {
		if input.Chain != ChainBitcoin || input.Direction != "input" {
			continue
		}
		if !validBitcoinTxID(strings.ToLower(input.PreviousTx)) || input.SourceID == "" || input.Atomic == "" || input.Asset != "BTC" || input.Scale != 8 {
			return nil, nil, fmt.Errorf("Bitcoin input %q has incomplete exact prevout provenance", input.ID)
		}
		id := fmt.Sprintf("bitcoin:%s:out:%d", strings.ToLower(input.PreviousTx), input.PreviousIndex)
		candidate := Occurrence{
			ID: id, Chain: ChainBitcoin, Transaction: strings.ToLower(input.PreviousTx),
			EventIndex: input.PreviousIndex, Direction: "output", To: input.From,
			Asset: "BTC", Atomic: input.Atomic, Scale: 8,
			Kind: EvidenceObservation, SourceID: input.SourceID,
		}
		if prior, ok := outputs[fmt.Sprintf("%s:%d", candidate.Transaction, candidate.EventIndex)]; ok {
			if prior.To != candidate.To || prior.Atomic != candidate.Atomic || prior.Asset != candidate.Asset || prior.Scale != candidate.Scale {
				return nil, nil, fmt.Errorf("Bitcoin prevout %s conflicts with retained output observation", id)
			}
			continue
		}
		if prior, ok := seen[id]; ok {
			if prior.To != candidate.To || prior.Atomic != candidate.Atomic {
				return nil, nil, fmt.Errorf("Bitcoin prevout %s has conflicting embedded source observations", id)
			}
			continue
		}
		seen[id] = candidate
		outputs[fmt.Sprintf("%s:%d", candidate.Transaction, candidate.EventIndex)] = candidate
		materialized = append(materialized, candidate)
	}
	all := append(append([]Occurrence(nil), existing...), materialized...)
	byID := make(map[string]Occurrence, len(all))
	for _, occurrence := range all {
		byID[occurrence.ID] = occurrence
	}
	edges := make([]Edge, 0, len(inputs))
	for _, input := range inputs {
		if input.Chain != ChainBitcoin || input.Direction != "input" {
			continue
		}
		previousID := fmt.Sprintf("bitcoin:%s:out:%d", strings.ToLower(input.PreviousTx), input.PreviousIndex)
		previous, ok := byID[previousID]
		if !ok {
			return nil, nil, fmt.Errorf("Bitcoin input %q lacks a materialized exact prevout", input.ID)
		}
		id := fmt.Sprintf("outpoint:%s:%d->%s:%d", strings.ToLower(input.PreviousTx), input.PreviousIndex, input.Transaction, input.EventIndex)
		edges = append(edges, Edge{
			ID: id, From: previous.ID, To: input.ID, Grade: GradeObserved,
			Rule:        "bitcoin.input.previous_outpoint_exact_match",
			EvidenceIDs: []string{previous.ID, input.ID},
		})
	}
	return materialized, edges, nil
}

// BuildBitcoinTransactionMembershipEdgesForTransactions records that an input
// and output belong to the same confirmed transaction without claiming that a
// particular input funded a particular output. This is useful for multi-input
// or multi-output transactions whose exact UTXOs are known but whose satoshi
// allocation is not. Callers choose the transaction set explicitly so a
// bounded case does not accidentally grow a whole-network graph.
func BuildBitcoinTransactionMembershipEdgesForTransactions(occurrences []Occurrence, transactionIDs []string) []Edge {
	wanted := make(map[string]struct{}, len(transactionIDs))
	for _, txid := range transactionIDs {
		wanted[strings.ToLower(txid)] = struct{}{}
	}
	type transactionEvents struct {
		inputs  []Occurrence
		outputs []Occurrence
	}
	byTransaction := make(map[string]*transactionEvents, len(wanted))
	for _, occurrence := range occurrences {
		if occurrence.Chain != ChainBitcoin {
			continue
		}
		txid := strings.ToLower(occurrence.Transaction)
		if _, ok := wanted[txid]; !ok {
			continue
		}
		events := byTransaction[txid]
		if events == nil {
			events = &transactionEvents{}
			byTransaction[txid] = events
		}
		switch occurrence.Direction {
		case "input":
			events.inputs = append(events.inputs, occurrence)
		case "output":
			events.outputs = append(events.outputs, occurrence)
		}
	}
	var edges []Edge
	for txid, events := range byTransaction {
		if len(events.outputs) < 2 {
			continue
		}
		for _, input := range events.inputs {
			for _, output := range events.outputs {
				id := fmt.Sprintf("transaction-membership:%s:in:%d->out:%d", txid, input.EventIndex, output.EventIndex)
				edges = append(edges, Edge{
					ID: id, From: input.ID, To: output.ID, Grade: GradeObserved,
					Rule:        "bitcoin.transaction.event_membership_without_value_allocation",
					EvidenceIDs: []string{input.ID, output.ID},
					Caveats:     []string{"These input and output observations belong to the same confirmed Bitcoin transaction. The transaction does not assign a particular input's value to this output; no input-specific satoshi allocation is inferred."},
				})
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	return edges
}
