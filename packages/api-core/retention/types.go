package retention

import (
	"context"
	"fmt"
	"time"
)

// Budget is one declared storage ceiling. A zero MaxAge means no age bound and a
// zero MaxBytes means no size bound; a Budget with neither is not enforceable
// and is rejected at parse time.
type Budget struct {
	// Name is the manifest key that declared this budget. It names the budget
	// in logs and findings, and is the registry name a custom pruner registers
	// under.
	Name string
	// MaxAge bounds how old retained items may be. Zero means no age bound.
	MaxAge time.Duration
	// MaxBytes bounds how much space the target may occupy. Zero means no size
	// bound, which makes the budget unbounded in size no matter what MaxAge
	// says.
	MaxBytes int64
}

// HasAgeBound reports whether an age horizon is declared.
func (b Budget) HasAgeBound() bool { return b.MaxAge > 0 }

// HasByteBound reports whether a size ceiling is declared. A budget without one
// makes no promise about disk, however short its age horizon.
func (b Budget) HasByteBound() bool { return b.MaxBytes > 0 }

// Usage is one measurement of what a target currently occupies.
type Usage struct {
	// Bytes is the measured on-disk size attributable to the target.
	Bytes int64
	// Items is the measured item count: rows for a table, top-level entries for
	// a directory.
	Items int64
}

// Candidate is one selected filesystem entry ready for deletion. Selection
// records the observed size, while Delete re-stats it before acting.
type Candidate struct {
	Path    string
	Bytes   int64
	ModTime time.Time
}

// Candidates is an ordered deletion set, normally oldest first.
type Candidates []Candidate

// Batch bounds one deletion transaction. Lock, when supplied, returns an
// unlock function and is acquired before the first deletion.
type Batch struct {
	MaxItems int
	MaxBytes int64
	Lock     func(context.Context) (func(), error)
}

// Receipt is the durable-friendly result of a filesystem deletion batch.
type Receipt struct {
	BytesBefore int64
	BytesAfter  int64
	Files       int64
	Duration    time.Duration
	Partial     bool
}

// Bound names which declared bound determined the retained set.
type Bound int

const (
	// BoundNone means no declared bound was reached, so nothing constrained
	// what was kept.
	BoundNone Bound = iota
	// BoundAge means the age horizon determined the retained set and the size
	// ceiling was never reached.
	BoundAge
	// BoundBytes means the size ceiling determined the retained set. This is a
	// signal about the producer: it is generating data faster than its declared
	// age horizon allows, and the ceiling is doing work the horizon cannot.
	BoundBytes
)

// String renders the bound for logs and findings.
func (b Bound) String() string {
	switch b {
	case BoundNone:
		return "none"
	case BoundAge:
		return "age"
	case BoundBytes:
		return "bytes"
	default:
		return fmt.Sprintf("bound(%d)", int(b))
	}
}

// Result reports one budget's enforcement cycle.
type Result struct {
	// Budget is the budget name this result describes.
	Budget string
	// Deleted counts items removed: rows for a table, top-level entries for a
	// directory.
	Deleted int64
	// FreedBytes is the reduction in measured target size across the cycle.
	FreedBytes int64
	// BoundBy names which bound determined the retained set.
	BoundBy Bound
	// Incomplete is true when the cycle stopped before reaching the bound,
	// because its context was cancelled or its deadline passed. The target may
	// still be over budget.
	Incomplete bool
	// Before and After are the measurements bracketing the cycle.
	Before Usage
	After  Usage
	// Refused is true when the pruner measured the target, found it over
	// budget, and deliberately deleted nothing. A refusal is a governed
	// outcome, not a failure: the budget keeps working as an alarm, it just
	// does not work as a deleter this cycle.
	Refused bool
	// RefusedReason states why, empty when Refused is false.
	RefusedReason string
	// CompactSkipped is true when reclaimable space was left in the file
	// because compacting it would have needed more free space than the
	// filesystem had.
	CompactSkipped bool
	// CompactSkipReason states why compaction was skipped, empty when it was
	// not.
	CompactSkipReason string
}

// OverBudget reports whether the target still exceeds its size ceiling after the
// cycle. A cycle can be complete and still over budget when the pruner has
// nothing left it is permitted to delete.
func (r Result) OverBudget(b Budget) bool {
	return b.HasByteBound() && r.After.Bytes > b.MaxBytes
}

// Pruner is the seam a component implements when it owns its own selection rule.
// Builtin implementations cover sqlite_table, directory, and file targets, so a
// component with no domain rule implements nothing.
//
// Implementations must sample ctx while working and return whatever progress
// they made with Incomplete set when it is cancelled, rather than discarding it.
type Pruner interface {
	// Measure reports what the target currently occupies without changing it.
	Measure(ctx context.Context) (Usage, error)
	// Prune reduces the target to b and reports which bound determined the
	// retained set.
	Prune(ctx context.Context, b Budget) (Result, error)
}
