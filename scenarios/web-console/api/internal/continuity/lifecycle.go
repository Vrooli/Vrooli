// Package continuity contains the provider-neutral Web Console conversation
// lifecycle.  Runtime packages report facts to this package; they do not
// choose whether durable evidence is retained.
package continuity

import "fmt"

type State string

const (
	StateProvisioning   State = "provisioning"
	StateLive           State = "live"
	StateArchivePending State = "archive_pending"
	StateArchived       State = "archived"
	StateExited         State = "exited"
	StateRecoverable    State = "recoverable"
	StateDeletePending  State = "delete_pending"
	StateDeleted        State = "deleted"
)

type Event string

const (
	EventProcessStarted            Event = "process_started"
	EventProcessStartFailedHistory Event = "process_start_failed_with_history"
	EventArchiveRequested          Event = "archive_requested"
	EventArchiveUndone             Event = "archive_undone"
	EventGraceElapsed              Event = "grace_elapsed_and_process_stopped"
	EventProcessExited             Event = "process_exited"
	EventResumableSourceFound      Event = "resumable_source_found"
	EventRecoverySucceeded         Event = "recovery_succeeded"
	EventDismissedToArchive        Event = "dismiss_to_archive"
	EventReopenSucceeded           Event = "reopen_succeeded"
	EventPermanentDeleteConfirmed  Event = "permanent_delete_confirmed"
	EventArtifactsPrunedReceipt    Event = "artifacts_pruned_and_receipt_committed"
	EventDeletionFailedPreserved   Event = "deletion_failed_preserved"
)

// Events returns the complete finite event set. Keeping this beside States
// gives callers and tests one authoritative vocabulary for exhaustive
// transition and invariant checks.
func Events() []Event {
	return []Event{
		EventProcessStarted,
		EventProcessStartFailedHistory,
		EventArchiveRequested,
		EventArchiveUndone,
		EventGraceElapsed,
		EventProcessExited,
		EventResumableSourceFound,
		EventRecoverySucceeded,
		EventDismissedToArchive,
		EventReopenSucceeded,
		EventPermanentDeleteConfirmed,
		EventArtifactsPrunedReceipt,
		EventDeletionFailedPreserved,
	}
}

// ValidateState is the lifecycle invariant boundary used by persistence and
// transport adapters before they interpret a state value. Unknown states must
// never be treated as live or as permission to perform a destructive action.
func ValidateState(state State) error {
	for _, known := range States() {
		if state == known {
			return nil
		}
	}
	return fmt.Errorf("unknown continuity state %q", state)
}

// ValidateTransition verifies both endpoints and the pure transition table.
// It is intentionally side-effect free so callers can check a proposed
// receipt before mutating any durable projection.
func ValidateTransition(from State, event Event, to State) error {
	if err := ValidateState(from); err != nil {
		return err
	}
	if expected, err := Transition(from, event); err != nil {
		return err
	} else if expected != to {
		return fmt.Errorf("continuity transition %s --%s--> %s does not match table (want %s)", from, event, to, expected)
	}
	return ValidateState(to)
}

// Transition is the pure lifecycle decision table. External side effects
// must happen only after this function accepts the event and must be
// represented by a receipt before the caller acknowledges the operation.
func Transition(from State, event Event) (State, error) {
	if err := ValidateState(from); err != nil {
		return "", err
	}
	var to State
	switch from {
	case StateProvisioning:
		switch event {
		case EventProcessStarted:
			to = StateLive
		case EventProcessStartFailedHistory:
			to = StateRecoverable
		}
	case StateLive:
		switch event {
		case EventArchiveRequested:
			to = StateArchivePending
		case EventProcessExited:
			to = StateExited
		}
	case StateArchivePending:
		switch event {
		case EventArchiveUndone:
			to = StateLive
		case EventGraceElapsed:
			to = StateArchived
		}
	case StateExited:
		switch event {
		case EventResumableSourceFound:
			to = StateRecoverable
		case EventRecoverySucceeded:
			to = StateLive
		case EventArchiveRequested:
			to = StateArchived
		}
	case StateRecoverable:
		switch event {
		case EventRecoverySucceeded:
			to = StateLive
		case EventDismissedToArchive:
			to = StateArchived
		}
	case StateArchived:
		switch event {
		case EventReopenSucceeded:
			to = StateLive
		case EventPermanentDeleteConfirmed:
			to = StateDeletePending
		}
	case StateDeletePending:
		switch event {
		case EventArtifactsPrunedReceipt:
			to = StateDeleted
		case EventDeletionFailedPreserved:
			to = StateArchived
		}
	case StateDeleted:
		// Deleted is terminal. In particular, retries cannot recreate or
		// reinterpret a destructive operation.
		to = ""
	}
	if to == "" {
		return "", fmt.Errorf("illegal continuity transition %s --%s-->", from, event)
	}
	return to, nil
}

// States returns the complete finite state set for exhaustive callers and
// documentation generators.
func States() []State {
	return []State{StateProvisioning, StateLive, StateArchivePending, StateArchived, StateExited, StateRecoverable, StateDeletePending, StateDeleted}
}
