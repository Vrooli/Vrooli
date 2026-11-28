package repairbudget

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Dispatch lifecycle states for one reserved operation. A reserved operation is
// charged once at Begin; the dispatch state only tracks which route owner holds
// it and whether replacement is currently safe.
const (
	DispatchReserved   = "reserved"
	DispatchActive     = "active"
	DispatchUncertain  = "uncertain"
	DispatchReconciled = "reconciled"
	DispatchFinished   = "finished"
)

// DispatchState is the derived, durable route-owner state for one operation
// identity. ReplacementAllowed is true only when no active or uncertain owner
// exists for the operation, so a fallback cannot duplicate live work.
type DispatchState struct {
	AttemptID          string `json:"attempt_id"`
	Fingerprint        string `json:"fingerprint"`
	Component          string `json:"component"`
	Kind               string `json:"kind"`
	Status             string `json:"status"`
	OwnerWork          string `json:"owner_work,omitempty"`
	OwnerRun           string `json:"owner_run,omitempty"`
	ReplacementAllowed bool   `json:"replacement_allowed"`
}

func (l *Ledger) appendEvent(current state, event Event) (Event, error) {
	current.Events = append(current.Events, event)
	if err := l.save(current); err != nil {
		return Event{}, err
	}
	return event, nil
}

// Acknowledge records the route owner's start acknowledgement for a reserved
// operation. A repeated acknowledgement of the same owner is idempotent; a
// second, different owner is refused until the first is reconciled terminal.
func (l *Ledger) Acknowledge(attemptID, ownerWork, ownerRun string) (Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	ownerWork, ownerRun = strings.TrimSpace(ownerWork), strings.TrimSpace(ownerRun)
	current, err := l.load()
	if err != nil {
		return Event{}, err
	}
	state, found := deriveDispatchState(current.Events, attemptID)
	if !found {
		return Event{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}
	event := Event{
		AttemptID:   attemptID,
		Fingerprint: state.Fingerprint,
		Component:   state.Component,
		Kind:        state.Kind,
		Event:       EventAcknowledged,
		OwnerWork:   ownerWork,
		OwnerRun:    ownerRun,
		At:          l.now().UTC().Format(time.RFC3339),
	}
	switch {
	case state.Status == DispatchFinished:
		return Event{}, fmt.Errorf("%w: %q", ErrAttemptTerminal, attemptID)
	case ownerWork == "" && ownerRun == "":
		return Event{}, fmt.Errorf("owner_work or owner_run is required")
	case state.Status == DispatchActive && state.OwnerWork == ownerWork && state.OwnerRun == ownerRun:
		return event, nil
	case (state.Status == DispatchActive || state.Status == DispatchUncertain) && state.OwnerRun != "" && state.OwnerRun != ownerRun:
		// A different run already holds this operation: a second executor is
		// a duplicate dispatch, not the original acknowledgement.
		return Event{}, fmt.Errorf("%w: %q", ErrExecutorActive, attemptID)
	}
	return l.appendEvent(current, event)
}

// MarkUncertain records that the start response for a dispatched operation was
// lost. Repeated marks are idempotent; a terminal attempt refuses the mark.
func (l *Ledger) MarkUncertain(attemptID, reason string) (Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	current, err := l.load()
	if err != nil {
		return Event{}, err
	}
	state, found := deriveDispatchState(current.Events, attemptID)
	if !found {
		return Event{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}
	if state.Status == DispatchFinished {
		return Event{}, fmt.Errorf("%w: %q", ErrAttemptTerminal, attemptID)
	}
	if state.Status == DispatchUncertain {
		return Event{AttemptID: attemptID, Event: EventUncertain, Reason: strings.TrimSpace(reason)}, nil
	}
	return l.appendEvent(current, Event{
		AttemptID:   attemptID,
		Fingerprint: state.Fingerprint,
		Component:   state.Component,
		Kind:        state.Kind,
		Event:       EventUncertain,
		Reason:      strings.TrimSpace(reason),
		At:          l.now().UTC().Format(time.RFC3339),
	})
}

// Reconcile records an owner observation for a possibly active operation and
// returns the durable dispatch state. ownerActive true retains the original
// executor; false marks the operation safe for a same-operation fallback.
func (l *Ledger) Reconcile(attemptID string, ownerActive bool, ownerRun, reason string) (DispatchState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	ownerRun = strings.TrimSpace(ownerRun)
	current, err := l.load()
	if err != nil {
		return DispatchState{}, err
	}
	state, found := deriveDispatchState(current.Events, attemptID)
	if !found {
		return DispatchState{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}
	if state.Status == DispatchFinished {
		return DispatchState{}, fmt.Errorf("%w: %q", ErrAttemptTerminal, attemptID)
	}
	if state.Status == DispatchReconciled && state.OwnerRun == ownerRun {
		return state, nil
	}
	event := Event{
		AttemptID:      attemptID,
		Fingerprint:    state.Fingerprint,
		Component:      state.Component,
		Kind:           state.Kind,
		Event:          EventReconciled,
		Reason:         strings.TrimSpace(reason),
		ObservedActive: ownerActive,
		At:             l.now().UTC().Format(time.RFC3339),
	}
	if ownerActive {
		event.OwnerRun = ownerRun
	}
	if _, err := l.appendEvent(current, event); err != nil {
		return DispatchState{}, err
	}
	reloaded, err := l.load()
	if err != nil {
		return DispatchState{}, err
	}
	reconciled, _ := deriveDispatchState(reloaded.Events, attemptID)
	return reconciled, nil
}

// Fallback re-dispatches a reserved or reconciled operation under the same
// charged identity. It never charges again, preserving the original remaining
// allowance. An uncertain or active owner refuses the fallback.
func (l *Ledger) Fallback(attemptID, ownerWork, ownerRun string) (Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	ownerWork, ownerRun = strings.TrimSpace(ownerWork), strings.TrimSpace(ownerRun)
	if ownerWork == "" && ownerRun == "" {
		return Event{}, fmt.Errorf("owner_work or owner_run is required")
	}
	current, err := l.load()
	if err != nil {
		return Event{}, err
	}
	state, found := deriveDispatchState(current.Events, attemptID)
	if !found {
		return Event{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}
	switch state.Status {
	case DispatchFinished:
		return Event{}, fmt.Errorf("%w: %q", ErrAttemptTerminal, attemptID)
	case DispatchUncertain:
		return Event{}, fmt.Errorf("%w: %q", ErrDispatchUncertain, attemptID)
	case DispatchActive:
		return Event{}, fmt.Errorf("%w: %q", ErrExecutorActive, attemptID)
	}
	return l.appendEvent(current, Event{
		AttemptID:   attemptID,
		Fingerprint: state.Fingerprint,
		Component:   state.Component,
		Kind:        state.Kind,
		Event:       EventFallback,
		OwnerWork:   ownerWork,
		OwnerRun:    ownerRun,
		At:          l.now().UTC().Format(time.RFC3339),
	})
}

// Cancel fences a reserved, active or uncertain operation and records its
// terminal disposition. A later acknowledgement or fallback is refused, so a
// delayed owner start cannot revive cancelled work.
func (l *Ledger) Cancel(attemptID, reason string) (Event, error) {
	event, err := l.Finish(attemptID, "cancelled", reason)
	if err != nil {
		return Event{}, err
	}
	return event, nil
}

// Dispatch returns the derived route-owner state for one operation identity.
func (l *Ledger) Dispatch(attemptID string) (DispatchState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	current, err := l.load()
	if err != nil {
		return DispatchState{}, err
	}
	state, found := deriveDispatchState(current.Events, attemptID)
	if !found {
		return DispatchState{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}
	return state, nil
}

// Outstanding returns every reserved, active or uncertain operation for this
// effort, sorted by attempt identity. Terminal attempts are excluded.
func (l *Ledger) Outstanding() ([]DispatchState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, err := l.load()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]DispatchState, 0)
	for _, event := range current.Events {
		if event.Event != EventStarted {
			continue
		}
		if _, ok := seen[event.AttemptID]; ok {
			continue
		}
		seen[event.AttemptID] = struct{}{}
		state, found := deriveDispatchState(current.Events, event.AttemptID)
		if !found || state.Status == DispatchFinished {
			continue
		}
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttemptID < out[j].AttemptID })
	return out, nil
}

// deriveDispatchState folds an attempt's events into its route-owner state. A
// terminal event is sticky: a late acknowledgement cannot downgrade it.
func deriveDispatchState(events []Event, attemptID string) (DispatchState, bool) {
	state := DispatchState{AttemptID: attemptID, Status: DispatchReserved, ReplacementAllowed: true}
	found := false
	for _, event := range events {
		if event.AttemptID != attemptID {
			continue
		}
		found = true
		if state.Status == DispatchFinished {
			continue
		}
		switch event.Event {
		case EventStarted:
			state.Fingerprint, state.Component, state.Kind = event.Fingerprint, event.Component, event.Kind
			state.Status, state.ReplacementAllowed = DispatchReserved, true
		case EventAcknowledged, EventFallback:
			if event.OwnerWork != "" {
				state.OwnerWork = event.OwnerWork
			}
			if event.OwnerRun != "" {
				state.OwnerRun = event.OwnerRun
			}
			state.Status, state.ReplacementAllowed = DispatchActive, false
		case EventUncertain:
			state.Status, state.ReplacementAllowed = DispatchUncertain, false
		case EventReconciled:
			if event.ObservedActive {
				if event.OwnerRun != "" {
					state.OwnerRun = event.OwnerRun
				}
				state.Status, state.ReplacementAllowed = DispatchActive, false
			} else {
				state.Status, state.ReplacementAllowed = DispatchReconciled, true
			}
		case EventFinished:
			state.Status, state.ReplacementAllowed = DispatchFinished, false
		}
	}
	return state, found
}
