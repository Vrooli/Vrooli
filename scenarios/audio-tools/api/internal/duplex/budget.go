package duplex

import (
	"errors"
	"sync"
)

var (
	ErrBudgetExceeded = errors.New("audio-tools/duplex: budget limit exceeded")
	ErrInvalidBudget  = errors.New("audio-tools/duplex: reservation must be positive")
	ErrOverSettlement = errors.New("audio-tools/duplex: settlement exceeds reservation")
	ErrFinalized      = errors.New("audio-tools/duplex: reservation already finalized")
	ErrNilReservation = errors.New("audio-tools/duplex: admission returned a nil reservation")
)

// Budget atomically bounds committed spend plus outstanding reservations.
// It is deliberately in-memory and provider-neutral; callers supply measured
// usage in integer micros and persist/audit snapshots at their policy boundary.
type Budget struct {
	mu                     sync.Mutex
	limit, reserved, spent uint64
}

type BudgetSnapshot struct{ LimitMicros, ReservedMicros, SpentMicros uint64 }

func NewBudget(limitMicros uint64) *Budget { return &Budget{limit: limitMicros} }

func (b *Budget) Snapshot() BudgetSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return BudgetSnapshot{b.limit, b.reserved, b.spent}
}

func (b *Budget) Reserve(maxMicros uint64) (Reservation, error) {
	if maxMicros == 0 {
		return nil, ErrInvalidBudget
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if maxMicros > b.limit-b.spent-b.reserved {
		return nil, ErrBudgetExceeded
	}
	b.reserved += maxMicros
	return &budgetReservation{budget: b, amount: maxMicros}, nil
}

type budgetReservation struct {
	mu     sync.Mutex
	budget *Budget
	amount uint64
	done   bool
}

func (r *budgetReservation) Settle(actual uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return ErrFinalized
	}
	if actual > r.amount {
		return ErrOverSettlement
	}
	r.budget.mu.Lock()
	r.budget.reserved -= r.amount
	r.budget.spent += actual
	r.budget.mu.Unlock()
	r.done = true
	return nil
}

func (r *budgetReservation) Release() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return
	}
	r.budget.mu.Lock()
	r.budget.reserved -= r.amount
	r.budget.mu.Unlock()
	r.done = true
}
