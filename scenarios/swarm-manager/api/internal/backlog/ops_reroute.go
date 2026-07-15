// Declarative-operation reroute seam for the pre-execution backlog flows.
//
// The research, workshop-refinement, clarification, and deferred auto-advance
// entrypoints used to spawn agents directly through agentmanager.SpawnBacklog /
// ContinueRun. They now START an operation through the generic operation runner
// (opsRunner.Invoke), which resolves the bound operating mode, spawns the agent
// through the operating-mode engine's chokepoint, and returns a run handle the
// legacy response fields carry unchanged. The terminal domain writes (round
// files, plan binding, clarification thread turns) land later through the closed
// action handlers in opshandlers.go when the operation's round completes.
package backlog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/workshop"
)

// advanceIntentName is the stable, per-item name of the deferred auto-advance
// scheduled intent. One item has at most one pending advance, so scheduling
// replaces (cancel-then-schedule) and canceling targets this one name.
const advanceIntentName = "advance-next-round"

// pinnedOperationVersion is the exact operation-contract version every backlog
// Invoke pins. An empty version resolves the contract-latest but NOT the pinned
// system binding (which pins 1.0.0), so the reroute must always pass a concrete
// version or binding resolution fails closed.
const pinnedOperationVersion = "1.0.0"

// invokeItemOperation starts one operation against a backlog item through the
// runner and returns the live run association the legacy response fields carry.
// It fails closed when the runner is unavailable, mapping onto the same typed
// error the legacy SpawnBacklog path returned. Errors from the live start (e.g.
// the per-item busy guard) propagate wrapped and remain errors.Is-matchable.
//
// callerInputs are the operator's typed request context, keyed by the OPERATION
// contract's caller-input names (USER_PROMPT, CONTEXT_PATHS, USER_QUESTION, …).
// They are validated against that contract (unknown key / missing required /
// non-replayable retention fail closed before anything is pinned), their canonical
// digest lands in provenance, and the run-starter routes them into the bound mode's
// structured caller-context generic providers — WITHOUT any mode caller-input, so
// the empty-set engine invariant still holds. Auto-triggered flows (create,
// auto-advance, workshop-round) carry no operator steering and pass nil; their
// modes derive all context from the item folder.
func (h *Handler) invokeItemOperation(ctx context.Context, kind BacklogKind, name string, op agentops.OperationID, idempotencyKey string, callerInputs map[string]any) (opsrunner.StartHandle, error) {
	if h.opsRunner == nil {
		return opsrunner.StartHandle{}, agentmanager.ErrNotAvailable
	}
	res, err := h.opsRunner.Invoke(ctx, opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: string(kind) + "/" + name},
		Operation:        op,
		OperationVersion: pinnedOperationVersion,
		CallerInputs:     callerInputs,
		IdempotencyKey:   idempotencyKey,
		RequestedBy:      "swarm-manager",
	})
	if err != nil {
		return opsrunner.StartHandle{}, err
	}
	if res.StartHandle != nil {
		return *res.StartHandle, nil
	}
	// Synchronous (simulation/test) path: no live handle, the operation already
	// drove to a terminal outcome. Surface the execution id so callers still have
	// a correlation handle.
	return opsrunner.StartHandle{RunID: res.ExecutionID}, nil
}

// putCallerString adds a trimmed non-empty operator-context string to a caller-
// input map under an operation-contract input name. Empty values are omitted so
// the pinned snapshot carries exactly the context the operator supplied and an
// absent optional input never appears; a nil map is returned by the builders when
// nothing applied so invokeItemOperation forwards no inputs.
func putCallerString(inputs map[string]any, key, value string) {
	if v := strings.TrimSpace(value); v != "" {
		inputs[key] = v
	}
}

// ResolveAdvance implements opsbridge.AdvanceResolver: it turns the deferred
// auto-advance intent into the concrete Invoke that advances the item, deciding
// workshop-round vs workshop-finalize from the item's CURRENT readiness (not the
// stale schedule-time decision) and deriving a round-count idempotency key so a
// crash re-fire replays instead of starting a second round.
func (h *Handler) ResolveAdvance(_ context.Context, w agentops.WorkflowInstance, _ agentops.ScheduledIntent) (opsrunner.InvokeRequest, bool, error) {
	kind, name, err := splitItemRef(w.Domain.ID)
	if err != nil {
		return opsrunner.InvokeRequest{}, false, err
	}
	if _, err := h.store.LoadItem(kind, name); err != nil {
		// The item vanished; the advance is no longer warranted.
		return opsrunner.InvokeRequest{}, false, nil
	}
	itemDir := h.store.ItemDir(kind, name)
	latestRound, roundCount, err := workshop.LoadLatestRound(itemDir)
	if err != nil {
		return opsrunner.InvokeRequest{}, false, err
	}
	op := deriveAdvanceOperation(latestRound, roundCount, kind)
	req := opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: w.Domain.ID},
		Operation:        op,
		OperationVersion: pinnedOperationVersion,
		IdempotencyKey:   fmt.Sprintf("advance-r%d-%s", roundCount, op),
		RequestedBy:      "swarm-manager-auto-advance",
	}
	return req, true, nil
}

// deriveAdvanceOperation mirrors resolveNextMode's readiness gate: a ready,
// fully-answered, not-yet-synthesized latest round advances to workshop-finalize;
// otherwise another workshop-round is warranted.
func deriveAdvanceOperation(latestRound *workshop.Round, roundCount int, kind BacklogKind) agentops.OperationID {
	if latestRound == nil || workshop.CountPendingDecisions(latestRound) > 0 || !workshop.NeedsSynthesis(latestRound) {
		return agentops.OpWorkshopRound
	}
	effective := workshop.ComputeEffectiveScores(latestRound.Readiness, roundCount, string(kind))
	if workshop.IsReady(effective) {
		return agentops.OpWorkshopFinalize
	}
	return agentops.OpWorkshopRound
}

// scheduleDeferredAdvanceIntent replaces the legacy pending-advance file + ticker
// with a durable scheduler intent carrying the advance OPERATION. Replace
// semantics (cancel-then-schedule) mirror the legacy delete+write so duplicate
// saves converge on one intent whose not_before reflects the latest delay.
func (h *Handler) scheduleDeferredAdvanceIntent(kind BacklogKind, name string, op agentops.OperationID, notBefore string) error {
	if h.opsScheduler == nil {
		return agentmanager.ErrNotAvailable
	}
	id := string(kind) + "/" + name
	if err := h.opsScheduler.CancelIntent(agentops.TargetBacklogItem, id, advanceIntentName); err != nil {
		return err
	}
	return h.opsScheduler.ScheduleIntent(agentops.TargetBacklogItem, id, agentops.ScheduledIntent{
		Intent:    advanceIntentName,
		Operation: op,
		NotBefore: notBefore,
	})
}

// hasScheduledAdvance reports whether a deferred auto-advance intent is pending
// for an item.
func (h *Handler) hasScheduledAdvance(kind BacklogKind, name string) bool {
	if h.opsScheduler == nil {
		return false
	}
	has, err := h.opsScheduler.HasIntent(agentops.TargetBacklogItem, string(kind)+"/"+name, advanceIntentName)
	if err != nil {
		return false
	}
	return has
}

// cancelDeferredAdvanceIntent cancels a pending auto-advance for an item. It is a
// no-op when nothing is scheduled.
func (h *Handler) cancelDeferredAdvanceIntent(kind BacklogKind, name string) error {
	if h.opsScheduler == nil {
		return nil
	}
	return h.opsScheduler.CancelIntent(agentops.TargetBacklogItem, string(kind)+"/"+name, advanceIntentName)
}

// operationForResearchMode maps a legacy research mode onto the operation the
// runner starts: workshop/initialize refine the spec (research-refine); finalize
// authors and binds the plan (workshop-finalize).
func operationForResearchMode(mode ResearchMode) agentops.OperationID {
	switch mode {
	case ResearchModeFinalize:
		return agentops.OpWorkshopFinalize
	default:
		return agentops.OpResearchRefine
	}
}

// advanceOperationForMode maps an auto-advance run mode onto the operation the
// advance starts: finalize authors the plan (workshop-finalize); anything else
// is another synthesis round (workshop-round).
func advanceOperationForMode(mode ResearchMode) agentops.OperationID {
	if mode == ResearchModeFinalize {
		return agentops.OpWorkshopFinalize
	}
	return agentops.OpWorkshopRound
}

// mapInvokeError classifies a runner Invoke/start error into the same API error
// the legacy spawn path returned.
func mapInvokeError(err error) *invokeErr {
	switch {
	case errors.Is(err, agentmanager.ErrNotAvailable):
		return &invokeErr{kind: invokeUnavailable, err: err}
	case errors.Is(err, agentactivity.ErrBacklogItemBusy):
		return &invokeErr{kind: invokeBusy, err: err}
	default:
		return &invokeErr{kind: invokeInternal, err: err}
	}
}

type invokeErrKind int

const (
	invokeInternal invokeErrKind = iota
	invokeUnavailable
	invokeBusy
)

type invokeErr struct {
	kind invokeErrKind
	err  error
}
