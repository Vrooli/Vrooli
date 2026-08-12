// Package audit is the domain-scoped home for the append-only accountability
// trail (OT-P0-008): every job dispatch and (Phase 4) provisioning operation —
// who, which node, what verb/args, and the outcome — recorded immutably because
// remote code execution and remote provisioning must be reconstructable after
// the fact.
//
// The domain is intentionally split into two narrow seams:
//
//   - Sink (Append): the WRITE side. The dispatch/provision domains hold a Sink
//     and append a record as a side effect of the operation they perform. There
//     is no proto RPC that writes audit records, so there is no wire path to
//     forge or mutate one.
//   - Reader (List): the READ side. The AuditService handler holds a Reader and
//     serves the owner-gated ListAuditRecords query.
//
// The default substrate behind both seams is the local append-only SQLite store
// (sqlite.go) — only INSERT and SELECT exist; there is no update/delete. A
// workspace-sandbox-backed Sink is the documented alternative behind the same
// Sink seam (the "accountability substrate" of SECURITY.md), wired when that
// scenario is green without changing any caller.
package audit

import (
	"fmt"
	"time"
)

// Action is the kind of operation a record describes. Mirrors the audit.proto
// AuditAction; the handler translates at the boundary so the domain never
// imports proto.
type Action int

const (
	// ActionUnspecified is the zero value.
	ActionUnspecified Action = 0
	// ActionDispatch records a job dispatch.
	ActionDispatch Action = 1
	// ActionProvision records a provisioning operation (Phase 4).
	ActionProvision Action = 2
	// ActionBreakGlass records a positively verified offline owner capability.
	ActionBreakGlass     Action = 3
	ActionSessionOpen    Action = 4
	ActionSessionClose   Action = 5
	ActionSessionResize  Action = 6
	ActionSessionDataIn  Action = 7
	ActionSessionDataOut Action = 8
)

// Outcome is the result recorded for an audited operation.
type Outcome int

const (
	// OutcomeUnspecified is the zero value.
	OutcomeUnspecified Outcome = 0
	// OutcomeAccepted — authorized and accepted (job dispatched / op started).
	OutcomeAccepted Outcome = 1
	// OutcomeRejected — denied by the allowlist / scope check / precondition.
	OutcomeRejected Outcome = 2
	// OutcomeCompleted — completed successfully (terminal).
	OutcomeCompleted Outcome = 3
	// OutcomeFailed — failed (terminal).
	OutcomeFailed Outcome = 4
)

// Record is one immutable accountability entry.
type Record struct {
	ID         string
	Action     Action
	Actor      string
	NodeID     string
	Scenario   string
	Verb       string
	Args       []string
	Outcome    Outcome
	Detail     string
	RunID      string
	RecordedAt time.Time
}

// ListFilter narrows the read query. Zero-value fields are not applied.
type ListFilter struct {
	NodeID string
	RunID  string
	Limit  int
}

// ErrInvalidRecord is the typed sentinel returned when a record fails the
// minimal write-side validation (an actor and node are required).
type ErrInvalidRecord struct {
	Field  string
	Reason string
}

func (e ErrInvalidRecord) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
