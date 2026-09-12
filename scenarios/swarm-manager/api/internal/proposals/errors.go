package proposals

import "errors"

// ErrInvalidProposal signals that a proposal fails schema or semantic
// validation. The error message is safe to surface to the user — it was
// produced by Validate with the intent of being actionable feedback to the
// agent or user.
var ErrInvalidProposal = errors.New("invalid proposal")

// ErrUnknownOp signals that a mutation carries an op value the apply layer
// does not recognize. Distinguished from ErrInvalidProposal so callers can
// tell "agent is speaking a dialect we don't support" from "agent made a
// mistake inside a known op".
var ErrUnknownOp = errors.New("unknown mutation op")

// ErrTargetNotFound signals that a mutation references an item that is not
// present in the milestone's current graph.
var ErrTargetNotFound = errors.New("mutation target not found in milestone")

// ErrDuplicateItem signals that an add_item mutation collides with an item
// that already exists (either elsewhere in the batch or on disk).
var ErrDuplicateItem = errors.New("item already exists")

// ErrTerminalStatusWrite signals that a mutation attempted to set a terminal
// status (completed/failed/needs_followup) on an item. Terminal transitions
// are the user's sole authority, reachable only through the review-decide
// endpoint.
var ErrTerminalStatusWrite = errors.New("terminal status transitions require review-decide")

// ErrStandaloneOperation signals an operation which needs milestone graph
// context but was proposed for an unattached backlog item.
var ErrStandaloneOperation = errors.New("operation is not allowed for a standalone backlog item")
