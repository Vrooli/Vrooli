// Package dispatch is the safety gate for remote execution (OT-P0-004). It
// validates every {scenario, verb, args} job against the scenario-CLI manifest
// allowlist AND the target node's granted verb-namespace scopes before anything
// runs, and it NEVER constructs a raw shell string — a valid job is carried to
// the node as the typed channel.JobPush envelope (translated at the handler
// boundary). On a valid, non-dry-run dispatch the service creates a durable run
// (runs domain), writes an append-only audit record (audit domain), and pushes
// the typed job down the node's held dial-out channel.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// dispatch-local DTOs, so the domain imports no sibling domain and no proto: the
// handler module is the single translation point to the registry/runs/audit
// services and the channel push. This keeps the allowlist logic — the
// highest-stakes surface in the scenario — pure and exhaustively unit-testable.
package dispatch

import (
	"fmt"
	"strings"
)

// Job is the typed unit of work. `Verb` is the space-joined vrooli command
// namespace that the allowlist matches (e.g. "scenario test"); `Scenario` is
// the target scenario the command operates on, inserted as the trailing
// positional by the node-agent; `Args` are additional typed tokens. None is
// ever joined into a shell string.
type Job struct {
	NodeID         string
	Scenario       string
	Verb           string
	Args           []string
	TimeoutSeconds int64
}

// trimmed returns a copy with surrounding whitespace stripped from the scalar
// fields and empty args dropped.
func (j Job) trimmed() Job {
	out := j
	out.NodeID = strings.TrimSpace(j.NodeID)
	out.Scenario = strings.TrimSpace(j.Scenario)
	out.Verb = strings.TrimSpace(j.Verb)
	out.Args = trimAll(j.Args)
	return out
}

// TargetNode is the minimal node shape the dispatch policy needs: its id, the
// granted verb-namespace scopes that authorize what it may run, and whether it
// is revoked. The handler adapter projects a registry node down to this.
type TargetNode struct {
	ID      string
	OS      string
	Arch    string
	Scopes  []string
	Revoked bool
}

// Decision is the result of a dispatch: the created run id (empty on a dry-run),
// whether it was a dry-run, and the validated job echoed back for confirmation.
type Decision struct {
	RunID  string
	DryRun bool
	Job    Job
}

// DispatchInput is what Service.Dispatch accepts: the owner actor (for audit),
// the typed job, and whether this is a dry-run (X-Dry-Run).
type DispatchInput struct {
	Actor  string
	Job    Job
	DryRun bool
}

// trimAll trims each element and drops empties.
func trimAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrNodeNotFound — the target node id is unknown.
type ErrNodeNotFound struct{ ID string }

func (e ErrNodeNotFound) Error() string { return fmt.Sprintf("node %q not found", e.ID) }

// ErrNodeRevoked — the target node has been revoked; it can run nothing.
type ErrNodeRevoked struct{ ID string }

func (e ErrNodeRevoked) Error() string { return fmt.Sprintf("node %q is revoked", e.ID) }

// ErrNodeOffline — the target node holds no dial-out channel right now.
type ErrNodeOffline struct{ ID string }

func (e ErrNodeOffline) Error() string {
	return fmt.Sprintf("node %q is offline (no dial-out channel)", e.ID)
}

// ErrNodeNeedsUpdate — the target node is online but its agent protocol version
// is flagged (needs-update / incompatible); it is excluded from work until the
// agent is updated (OT-P1-001 protocol-compatibility gating).
type ErrNodeNeedsUpdate struct{ ID string }

func (e ErrNodeNeedsUpdate) Error() string {
	return fmt.Sprintf("node %q needs an agent update (protocol incompatible); excluded from dispatch", e.ID)
}

// ErrVerbNotInManifest — the verb is not a recognised manifest verb namespace.
type ErrVerbNotInManifest struct{ Verb string }

func (e ErrVerbNotInManifest) Error() string {
	return fmt.Sprintf("verb %q is not an allowlisted manifest verb", e.Verb)
}

// ErrVerbOutOfScope — the verb is a known manifest verb but the node has not
// been granted a scope covering it.
type ErrVerbOutOfScope struct{ Verb string }

func (e ErrVerbOutOfScope) Error() string {
	return fmt.Sprintf("verb %q is outside this node's granted scopes", e.Verb)
}

// ErrUnsafeToken — a job token carries a shell metacharacter. The job is typed,
// never a shell string, so this is a hard reject (defence in depth).
type ErrUnsafeToken struct {
	Token  string
	Reason string
}

func (e ErrUnsafeToken) Error() string {
	return fmt.Sprintf("unsafe token %q: %s", e.Token, e.Reason)
}

// ErrInvalidJob — a structural validation failure (empty required field).
type ErrInvalidJob struct {
	Field  string
	Reason string
}

func (e ErrInvalidJob) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrDeliveryFailed — the job could not be pushed to the (briefly) reachable
// node; the run is aborted and the dispatch fails.
type ErrDeliveryFailed struct{ NodeID string }

func (e ErrDeliveryFailed) Error() string {
	return fmt.Sprintf("job could not be delivered to node %q", e.NodeID)
}
