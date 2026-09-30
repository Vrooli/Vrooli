// Package inventory contains pure stock and batch accounting.
package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nutrition-planner/internal/decimalx"
)

type EventKind string

const (
	Purchase     EventKind = "purchase"
	Assertion    EventKind = "assertion"
	Correction   EventKind = "correction"
	Waste        EventKind = "waste"
	Preparation  EventKind = "preparation"
	BatchPortion EventKind = "batch_portion"
	PortionUndo  EventKind = "batch_portion_undo"
)

type Event struct {
	ID        string
	Kind      EventKind
	ItemID    string
	BatchID   string
	Amount    decimalx.Decimal
	Unit      string
	RecipeID  string
	CreatedAt time.Time
}

type ReceiptProposalStatus string

const (
	ReceiptProposalPending ReceiptProposalStatus = "pending"
	ReceiptProposalApplied ReceiptProposalStatus = "applied"
)

// ReceiptProposal is a reviewable purchase candidate. Staging a proposal does
// not affect stock; ApplyReceiptProposal must be called explicitly.
type ReceiptProposal struct {
	ID            string
	SourceID      string
	TransactionID string
	LineKey       string
	Description   string
	ItemID        string
	Amount        decimalx.Decimal
	Unit          string
	Price         string
	Status        ReceiptProposalStatus
	EventID       string
	CreatedAt     time.Time
	AppliedAt     time.Time
}

type ReceiptRepository interface {
	StageReceiptProposal(context.Context, string, ReceiptProposal) (ReceiptProposal, error)
	ListReceiptProposals(context.Context, string) ([]ReceiptProposal, error)
	ApplyReceiptProposal(context.Context, string, string) (ReceiptProposal, error)
}

type Repository interface {
	Append(context.Context, string, Event) error
	List(context.Context, string) ([]Event, error)
}

type BatchRepository interface {
	Repository
	PrepareBatch(context.Context, string, string, string, string, int64, decimalx.Decimal, string, []Event) (Batch, error)
	ConsumeBatchPortion(context.Context, string, string, string, decimalx.Decimal, string, string, bool) (Batch, error)
}

type Batch struct {
	ID             string
	RecipeID       string
	RecipeRevision int64
	Yield          decimalx.Decimal
	Available      decimalx.Decimal
	Unit           string
}

type State struct {
	OnHand  map[string]decimalx.Decimal
	Batches map[string]Batch
	Events  []Event
}

func NewState() State {
	return State{OnHand: map[string]decimalx.Decimal{}, Batches: map[string]Batch{}}
}

// Apply is idempotent by event ID. Reusing an ID with different content is a
// conflict, because silently applying the second payload would corrupt stock.
func Apply(state State, event Event) (State, error) {
	if event.ID == "" || event.Amount.IsUnknown() || event.Amount.IsZero() || event.Unit == "" {
		return state, errors.New("inventory event requires id, known non-zero amount, and unit")
	}
	for _, existing := range state.Events {
		if existing.ID == event.ID {
			if existing.Kind != event.Kind || existing.ItemID != event.ItemID || existing.BatchID != event.BatchID || existing.Amount.String() != event.Amount.String() || existing.Unit != event.Unit {
				return state, fmt.Errorf("idempotency key %q reused with different payload", event.ID)
			}
			return state, nil
		}
	}
	next := clone(state)
	sign := decimalx.KnownInt(1)
	switch event.Kind {
	case Purchase, Assertion, Correction:
		if event.ItemID == "" {
			return state, errors.New("stock event requires item")
		}
		current, ok := next.OnHand[event.ItemID]
		if !ok {
			current = decimalx.KnownInt(0)
		}
		next.OnHand[event.ItemID], _ = decimalx.Add(current, event.Amount)
	case Waste, Preparation:
		if event.ItemID == "" {
			return state, errors.New("consumption event requires item")
		}
		sign = decimalx.KnownInt(-1)
		decrement, _ := decimalx.Mul(event.Amount, sign)
		current, ok := next.OnHand[event.ItemID]
		if !ok {
			current = decimalx.KnownInt(0)
		}
		next.OnHand[event.ItemID], _ = decimalx.Add(current, decrement)
	case BatchPortion:
		batch, ok := next.Batches[event.BatchID]
		if !ok {
			return state, fmt.Errorf("batch %q not found", event.BatchID)
		}
		if batch.Available.IsUnknown() {
			return state, errors.New("batch availability is unknown")
		}
		if comparison, _ := decimalx.Compare(event.Amount, batch.Available); comparison > 0 {
			return state, errors.New("batch portion exceeds available yield")
		}
		batch.Available, _ = decimalx.Sub(batch.Available, event.Amount)
		next.Batches[event.BatchID] = batch
	case PortionUndo:
		batch, ok := next.Batches[event.BatchID]
		if !ok {
			return state, fmt.Errorf("batch %q not found", event.BatchID)
		}
		batch.Available, _ = decimalx.Add(batch.Available, event.Amount)
		next.Batches[event.BatchID] = batch
	default:
		return state, fmt.Errorf("unsupported inventory event kind %q", event.Kind)
	}
	next.Events = append(next.Events, event)
	return next, nil
}

// Prepare consumes raw requirements once and creates a pinned batch. Later
// serving events operate on the batch and never consume raw stock again.
func Prepare(state State, eventID, batchID, recipeID string, revision int64, requirements []Event, yield decimalx.Decimal, unit string) (State, error) {
	if eventID == "" || batchID == "" || yield.IsUnknown() || yield.IsZero() || unit == "" {
		return state, errors.New("preparation requires event, batch, known yield, and unit")
	}
	if _, exists := state.Batches[batchID]; exists {
		return state, nil
	}
	next := state
	for _, requirement := range requirements {
		if requirement.Kind == "" {
			requirement.Kind = Preparation
		}
		if requirement.ID == "" {
			requirement.ID = eventID + ":" + requirement.ItemID
		}
		var err error
		next, err = Apply(next, requirement)
		if err != nil {
			return state, err
		}
	}
	next.Batches[batchID] = Batch{ID: batchID, RecipeID: recipeID, RecipeRevision: revision, Yield: yield, Available: yield, Unit: unit}
	return next, nil
}

func clone(state State) State {
	next := NewState()
	for key, value := range state.OnHand {
		next.OnHand[key] = value
	}
	for key, value := range state.Batches {
		next.Batches[key] = value
	}
	next.Events = append([]Event(nil), state.Events...)
	return next
}
