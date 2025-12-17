// Package types provides shared types for workspace sandboxes.
// This file contains the status state machine and transition logic.
package types

import "fmt"

// Status represents the lifecycle state of a sandbox.
type Status string

const (
	StatusCreating Status = "creating"
	StatusActive   Status = "active"
	StatusStopped  Status = "stopped"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusDeleted  Status = "deleted"
	StatusError    Status = "error"
)

// --- Status Classification ---
// These methods classify statuses into meaningful categories.

// IsActive returns true if the sandbox is currently usable.
func (s Status) IsActive() bool {
	return s == StatusCreating || s == StatusActive
}

// IsTerminal returns true if the sandbox has reached a final state.
// Terminal states cannot transition to other states.
func (s Status) IsTerminal() bool {
	return s == StatusApproved || s == StatusRejected || s == StatusDeleted
}

// IsMounted returns true if the sandbox currently has a mounted overlay.
func (s Status) IsMounted() bool {
	return s == StatusActive
}

// RequiresCleanup returns true if the sandbox may have filesystem resources
// that need cleanup.
func (s Status) RequiresCleanup() bool {
	return s == StatusActive || s == StatusStopped || s == StatusError || s == StatusCreating
}

// --- State Transition Decisions ---
// These functions make state transition decisions explicit and testable.
// Each function checks if a specific operation is allowed given the current status.

// CanStop checks if a sandbox can be stopped from its current status.
// Stop means unmounting the overlay but preserving the data for later review.
func CanStop(status Status) error {
	if status == StatusActive {
		return nil
	}
	return &InvalidTransitionError{
		Current:   status,
		Attempted: StatusStopped,
		Reason:    "only active sandboxes can be stopped",
	}
}

// CanApprove checks if a sandbox's changes can be approved.
// Approval applies the changes to the canonical repo.
func CanApprove(status Status) error {
	switch status {
	case StatusActive, StatusStopped:
		return nil
	case StatusApproved:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusApproved,
			Reason:    "sandbox is already approved",
		}
	case StatusRejected:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusApproved,
			Reason:    "cannot approve a rejected sandbox",
		}
	case StatusDeleted:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusApproved,
			Reason:    "cannot approve a deleted sandbox",
		}
	default:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusApproved,
			Reason:    fmt.Sprintf("sandbox in %s state cannot be approved", status),
		}
	}
}

// CanReject checks if a sandbox's changes can be rejected.
// Rejection discards all changes without applying them.
func CanReject(status Status) error {
	switch status {
	case StatusActive, StatusStopped:
		return nil
	case StatusRejected:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusRejected,
			Reason:    "sandbox is already rejected",
		}
	case StatusApproved:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusRejected,
			Reason:    "cannot reject an approved sandbox",
		}
	case StatusDeleted:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusRejected,
			Reason:    "cannot reject a deleted sandbox",
		}
	default:
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusRejected,
			Reason:    fmt.Sprintf("sandbox in %s state cannot be rejected", status),
		}
	}
}

// CanDelete checks if a sandbox can be deleted.
// All sandboxes can be deleted, but this function documents that decision.
func CanDelete(status Status) error {
	if status == StatusDeleted {
		return &InvalidTransitionError{
			Current:   status,
			Attempted: StatusDeleted,
			Reason:    "sandbox is already deleted",
		}
	}
	return nil // Can always delete
}

// CanGenerateDiff checks if a diff can be generated for a sandbox.
// Diffs require an active or stopped sandbox with upper directory data.
func CanGenerateDiff(status Status) error {
	switch status {
	case StatusActive, StatusStopped:
		return nil
	case StatusCreating:
		return &InvalidTransitionError{
			Current: status,
			Reason:  "sandbox is still being created, diff not available yet",
		}
	case StatusDeleted:
		return &InvalidTransitionError{
			Current: status,
			Reason:  "sandbox is deleted, diff data no longer available",
		}
	default:
		return nil // Allow diff on approved/rejected for historical view
	}
}

// CanGetWorkspacePath checks if a workspace path can be returned.
// Only mounted sandboxes have a valid workspace path.
func CanGetWorkspacePath(status Status) error {
	if !status.IsMounted() {
		return fmt.Errorf("sandbox is not mounted (status: %s)", status)
	}
	return nil
}

// --- State Transition Matrix ---
// ValidTransitions documents which state transitions are allowed.
// This is the authoritative source for the state machine.
var ValidTransitions = map[Status][]Status{
	StatusCreating: {StatusActive, StatusError, StatusDeleted},
	StatusActive:   {StatusStopped, StatusApproved, StatusRejected, StatusError, StatusDeleted},
	StatusStopped:  {StatusActive, StatusApproved, StatusRejected, StatusDeleted},
	StatusApproved: {StatusDeleted},
	StatusRejected: {StatusDeleted},
	StatusError:    {StatusDeleted},
	StatusDeleted:  {}, // No transitions from deleted
}

// CanTransitionTo checks if a transition from the current status to the target is valid.
func CanTransitionTo(from, to Status) bool {
	allowed, exists := ValidTransitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// InvalidTransitionError indicates an invalid state transition was attempted.
type InvalidTransitionError struct {
	Current   Status
	Attempted Status
	Reason    string
}

func (e *InvalidTransitionError) Error() string {
	if e.Attempted != "" {
		return fmt.Sprintf("cannot transition from %s to %s: %s", e.Current, e.Attempted, e.Reason)
	}
	return fmt.Sprintf("invalid operation for status %s: %s", e.Current, e.Reason)
}
