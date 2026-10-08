// Package gate is the domain-scoped home for the cross-OS deployment gate
// (OT-P1-002): the missing primitive behind cross-platform deployment
// validation. deployment-manager asks bridge "is this scenario production-ready
// on Ubuntu + macOS + Windows?"; gate fans the same project revision out to one
// eligible node per target OS, runs the native validation suite on each as a
// durable remote run — delegating to the dispatch + runs domains, gate NEVER
// reimplements dispatch or run management — and aggregates the per-OS verdicts
// into one cross-OS result. Bridge SUPPLIES the capability; deployment-manager
// OWNS the promotion verdict.
//
// Aggregation rule: a gate PASSES only when every target OS validated green.
// ANY failing OS — a non-zero/aborted validation run OR a target OS with no
// eligible node — fails the gate, with the offending OS surfaced. While targets
// are still running and none has failed yet, the gate is PENDING.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs, so the domain imports no sibling domain and no proto: the
// handler module is the single translation point to the registry (enumerate +
// OS/revocation), the presence hub (online + protocol compatibility), and the
// dispatch + runs services (dispatch a validation run, read/await its verdict).
package gate

import (
	"fmt"
	"strings"
	"time"
)

// DefaultWaitTimeout bounds a WaitGate that passes timeout_seconds <= 0, so a
// block-once wait can never hang forever even if a node vanishes without a
// terminal event. The CLI surfaces a timed-out wait so the operator re-issues
// it (the gate is durable).
const DefaultWaitTimeout = 60 * time.Minute

// DefaultVerb is the cross-OS validation verb a gate dispatches when the request
// names none: run the scenario's native test suite on each target OS.
const DefaultVerb = "scenario test"

// GateVerdict is a gate's aggregate cross-OS disposition, derived from its
// per-OS results.
type GateVerdict int

const (
	// VerdictUnspecified is the zero value; a returned gate never holds it.
	VerdictUnspecified GateVerdict = 0
	// VerdictPending — at least one target OS is still running and none failed.
	VerdictPending GateVerdict = 1
	// VerdictPassed — every target OS validated green.
	VerdictPassed GateVerdict = 2
	// VerdictFailed — at least one target OS failed (run or no-node).
	VerdictFailed GateVerdict = 3
)

// String renders the verdict as a short lowercase label.
func (v GateVerdict) String() string {
	switch v {
	case VerdictPending:
		return "pending"
	case VerdictPassed:
		return "passed"
	case VerdictFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// OSDisposition is one target OS's current outcome in a gate.
type OSDisposition int

const (
	// OSDispositionUnspecified is the zero value.
	OSDispositionUnspecified OSDisposition = 0
	// OSDispositionPending — a validation run is dispatched and not yet terminal.
	OSDispositionPending OSDisposition = 1
	// OSDispositionPassed — the validation run exited 0.
	OSDispositionPassed OSDisposition = 2
	// OSDispositionFailed — the validation run exited non-zero.
	OSDispositionFailed OSDisposition = 3
	// OSDispositionAborted — the validation run was aborted.
	OSDispositionAborted OSDisposition = 4
	// OSDispositionNoNode — no eligible node runs this OS.
	OSDispositionNoNode OSDisposition = 5
	// OSDispositionDispatchFailed — the validation job was rejected at dispatch.
	OSDispositionDispatchFailed OSDisposition = 6
)

// String renders the disposition as a short lowercase label.
func (d OSDisposition) String() string {
	switch d {
	case OSDispositionPending:
		return "pending"
	case OSDispositionPassed:
		return "passed"
	case OSDispositionFailed:
		return "failed"
	case OSDispositionAborted:
		return "aborted"
	case OSDispositionNoNode:
		return "no_node"
	case OSDispositionDispatchFailed:
		return "dispatch_failed"
	default:
		return "unspecified"
	}
}

// failing reports whether the disposition counts against the gate (it can never
// pass once any OS is failing). A no-node / dispatch-failed / aborted / non-zero
// run all mean "not validated green on this OS".
func (d OSDisposition) failing() bool {
	switch d {
	case OSDispositionFailed, OSDispositionAborted, OSDispositionNoNode, OSDispositionDispatchFailed:
		return true
	default:
		return false
	}
}

// OSResult is one target OS's line in a gate's ledger.
type OSResult struct {
	OS          string
	NodeID      string
	RunID       string
	Disposition OSDisposition
	ExitCode    int32
	Detail      string
}

// Gate is the durable, server-owned record of one cross-OS deployment gate.
type Gate struct {
	ID             string
	Scenario       string
	TargetRevision string
	Verb           string
	Args           []string
	Verdict        GateVerdict
	TotalTargets   int
	Passed         int
	Failed         int
	Pending        int
	CreatedAt      time.Time
}

// RunInput is what Service.Run accepts.
type RunInput struct {
	Actor          string
	Scenario       string
	TargetRevision string
	TargetOSes     []string
	Verb           string
	Args           []string
	TimeoutSeconds int64
	DryRun         bool
}

// RunDecision is the result of a Run: the persisted gate id (empty on a
// dry-run), whether it was a dry-run, the aggregate verdict, and the per-OS
// ledger.
type RunDecision struct {
	GateID  string
	DryRun  bool
	Verdict GateVerdict
	Results []OSResult
}

// ListFilter narrows ListGates.
type ListFilter struct {
	Limit int
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrInvalidGate — a structural validation failure (empty required field).
type ErrInvalidGate struct {
	Field  string
	Reason string
}

func (e ErrInvalidGate) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrGateNotFound — no gate matches the id.
type ErrGateNotFound struct{ ID string }

func (e ErrGateNotFound) Error() string { return fmt.Sprintf("gate %q not found", e.ID) }

// aggregateVerdict derives a gate's verdict from its per-OS results: ANY failing
// OS fails the gate (it can never become production-ready on that OS); else any
// still-pending target keeps it PENDING; else every OS passed.
func aggregateVerdict(results []OSResult) GateVerdict {
	failed, pending, passed := 0, 0, 0
	for _, r := range results {
		switch {
		case r.Disposition.failing():
			failed++
		case r.Disposition == OSDispositionPending:
			pending++
		case r.Disposition == OSDispositionPassed:
			passed++
		}
	}
	switch {
	case failed > 0:
		return VerdictFailed
	case pending > 0:
		return VerdictPending
	case passed > 0:
		return VerdictPassed
	default:
		// No classifiable results — treat as failed (nothing was validated).
		return VerdictFailed
	}
}

// counts tallies the gate summary fields from the per-OS results.
func counts(results []OSResult) (passed, failed, pending int) {
	for _, r := range results {
		switch {
		case r.Disposition.failing():
			failed++
		case r.Disposition == OSDispositionPending:
			pending++
		case r.Disposition == OSDispositionPassed:
			passed++
		}
	}
	return passed, failed, pending
}

// normaliseOSes trims, lowercases, and dedupes the requested target OSes while
// preserving first-seen order, so a gate validates each OS exactly once.
func normaliseOSes(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		os := strings.ToLower(strings.TrimSpace(raw))
		if os == "" {
			continue
		}
		if _, dup := seen[os]; dup {
			continue
		}
		seen[os] = struct{}{}
		out = append(out, os)
	}
	return out
}
