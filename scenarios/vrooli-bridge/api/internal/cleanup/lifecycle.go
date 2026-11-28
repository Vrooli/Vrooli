package cleanup

import "fmt"

// TransitionStatus is the single owner of cleanup operation state changes.
// Same-state writes are allowed because repository updates also persist
// receipts and metadata without changing the lifecycle.
func TransitionStatus(from, to Status) error {
	if !from.Valid() {
		return fmt.Errorf("invalid cleanup operation source status %d", from)
	}
	if !to.Valid() {
		return fmt.Errorf("invalid cleanup operation target status %d", to)
	}
	if from == to {
		return nil
	}
	allowed := map[Status]map[Status]bool{
		StatusQueued:    {StatusPlanning: true, StatusPlanned: true, StatusApplying: true, StatusFailed: true, StatusBlocked: true, StatusCanceled: true},
		StatusPlanning:  {StatusPlanned: true, StatusFailed: true, StatusBlocked: true, StatusCanceled: true},
		StatusPlanned:   {StatusConfirmed: true, StatusFailed: true, StatusBlocked: true, StatusCanceled: true},
		StatusConfirmed: {StatusApplying: true, StatusFailed: true, StatusBlocked: true, StatusCanceled: true},
		StatusApplying:  {StatusCompleted: true, StatusFailed: true, StatusBlocked: true, StatusCanceled: true},
		StatusFailed:    {StatusApplying: true},
		StatusBlocked:   {StatusApplying: true, StatusFailed: true},
	}
	if !allowed[from][to] {
		return fmt.Errorf("invalid cleanup operation transition %s -> %s", from, to)
	}
	return nil
}

func statusFromEvent(current Status, event Event) (Status, bool) {
	next := current
	changed := false
	if event.Status != "" {
		if status, ok := parseStatus(event.Status); ok {
			next, changed = status, true
		}
	}
	if event.Kind == EventExit && !next.Terminal() {
		// A successful plan exit only acknowledges the read-only inventory call.
		if !(next == StatusPlanned && event.ExitCode == 0) {
			next, changed = statusFromExit(event.ExitCode), true
		}
	}
	return next, changed
}

// Valid reports whether a status is a persisted operation state.
func (s Status) Valid() bool {
	return s >= StatusQueued && s <= StatusCanceled
}

func (op *Operation) Transition(to Status) error {
	if op == nil {
		return fmt.Errorf("cleanup operation is nil")
	}
	if err := TransitionStatus(op.Status, to); err != nil {
		return err
	}
	op.Status = to
	return nil
}
