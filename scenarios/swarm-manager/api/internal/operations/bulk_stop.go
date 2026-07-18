package operations

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// maxBulkStopBodyBytes caps the bulk-stop request body. Operators bulk-stop
// at most a few dozen runs at a time; a 64KiB limit is generous and stops a
// runaway client from forcing the server to materialize an unbounded
// runID list.
const maxBulkStopBodyBytes = 64 * 1024

// maxBulkStopRunIDs caps the number of run IDs in a single bulk-stop call.
// Keeps the cancellation loop bounded and the response payload small. The
// ceiling is well above any realistic operator action — the Operations
// Center tops out at ~50 active runs per the lane caps.
const maxBulkStopRunIDs = 200

// Stopper is the seam Bulk uses to cancel a single run. agentactivity.Service
// satisfies it; tests pass a recording fake that asserts serial invocation.
type Stopper interface {
	StopRun(ctx context.Context, runID string) error
}

// BulkStopRequest is the wire shape POSTed to /api/v1/operations/bulk-stop.
//
// Exactly one of RunIDs or Filter must be set:
//   - RunIDs targets specific runs by ID (the common case driven by row
//     selection in the UI).
//   - Filter targets every active run matching all set predicates. Used by
//     the "Stop all running" affordance and by future scripted operators.
//
// Both modes return the same per-run outcome list so callers don't need to
// branch on shape — successful stops carry Success=true; failures carry
// Success=false and a short Error message. The endpoint always returns 200
// when at least one outcome was produced; per-run errors are not promoted
// to HTTP status codes because partial success is the common case (e.g. one
// run already finished while the operator was reading the page).
type BulkStopRequest struct {
	RunIDs []string        `json:"run_ids,omitempty"`
	Filter *BulkStopFilter `json:"filter,omitempty"`
}

// BulkStopFilter narrows a "stop all running" call by lane and/or status.
// Empty fields mean "any". Lane validates against the four canonical lanes;
// invalid lanes return 400 instead of being silently ignored, matching the
// /api/v1/operations endpoint's parseFilters semantics.
type BulkStopFilter struct {
	Lane   string `json:"lane,omitempty"`
	Status string `json:"status,omitempty"`
}

// BulkStopOutcome reports the result of a single StopRun call. Error is
// populated when Success is false; both fields are stable across the wire
// so clients can render per-row status without a second lookup.
type BulkStopOutcome struct {
	RunID   string `json:"run_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// BulkStopResponse is the top-level wire shape. Outcomes is always
// populated (possibly empty when a Filter matched nothing). Counts make
// the success / failure split trivially renderable in the UI without a
// second pass over Outcomes.
type BulkStopResponse struct {
	Outcomes []BulkStopOutcome `json:"outcomes"`
	Total    int               `json:"total"`
	Stopped  int               `json:"stopped"`
	Failed   int               `json:"failed"`
}

// BulkStopHandler handles POST /api/v1/operations/bulk-stop.
//
// Cancellation is serialized server-side: the handler iterates the resolved
// run IDs in order and calls Stopper.StopRun once per ID. Two reasons for
// serial iteration over fan-out:
//
//  1. Initiative locks serialize
//     access to a given initiative; concurrent StopRun calls against runs
//     belonging to the same initiative would queue inside the lock and any
//     additional fan-out from the UI under load could deadlock against
//     governance.
//  2. Operator audit. Serial cancellation gives a deterministic per-run
//     outcome ordering that operators can read top-to-bottom in the UI
//     bulk-action toast.
//
// Filter resolution reads the live activity ledger via ActivityLister so the
// "stop all running" path has the same source-of-truth as the Operations
// Center page itself.
type BulkStopHandler struct {
	stopper    Stopper
	activities ActivityLister
}

// NewBulkStopHandler returns a handler bound to a Stopper and the same
// ActivityLister the aggregator uses. Both are required.
func NewBulkStopHandler(stopper Stopper, activities ActivityLister) (*BulkStopHandler, error) {
	if stopper == nil {
		return nil, fmt.Errorf("operations: NewBulkStopHandler: stopper is required")
	}
	if activities == nil {
		return nil, fmt.Errorf("operations: NewBulkStopHandler: activities is required")
	}
	return &BulkStopHandler{stopper: stopper, activities: activities}, nil
}

// RegisterRoutes wires POST /api/v1/operations/bulk-stop onto the router.
func (h *BulkStopHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/operations/bulk-stop", h.Post).Methods("POST")
}

// Post decodes the request, resolves the target run-ID set, then iterates
// serially. The response is always 200 when input parses; per-run errors
// flow through the Outcomes slice rather than the HTTP status.
func (h *BulkStopHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req BulkStopRequest
	if err := httputil.DecodeJSONStrictBounded(r, &req, maxBulkStopBodyBytes); err != nil {
		apierr.MapError(w, "[operations] bulk-stop", apierr.BadRequest("invalid request body: %v", err))
		return
	}

	runIDs, derr := h.resolveTargets(r.Context(), req)
	if derr != nil {
		apierr.MapError(w, "[operations] bulk-stop", derr)
		return
	}

	outcomes := make([]BulkStopOutcome, 0, len(runIDs))
	stopped, failed := 0, 0
	for _, id := range runIDs {
		err := h.stopper.StopRun(r.Context(), id)
		if err != nil {
			failed++
			outcomes = append(outcomes, BulkStopOutcome{
				RunID:   id,
				Success: false,
				Error:   httputil.TruncateErrorMessage(err, 240),
			})
			continue
		}
		stopped++
		outcomes = append(outcomes, BulkStopOutcome{RunID: id, Success: true})
	}

	resp := BulkStopResponse{
		Outcomes: outcomes,
		Total:    len(outcomes),
		Stopped:  stopped,
		Failed:   failed,
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[operations] bulk-stop", apierr.Internal("failed to encode bulk-stop response"))
	}
}

// resolveTargets normalizes the incoming request into a deduplicated,
// ordered slice of run IDs. RunIDs and Filter are mutually exclusive — the
// operator picks one or the other in the UI, and accepting both would make
// "did the filter shrink my selection?" semantics ambiguous.
//
// RunIDs path: trim, drop empties, dedupe (preserving first-seen order).
// Filter path: list active records, apply lane/status predicates, sort by
// RequestedAt descending so cancellation hits the freshest runs first
// (matching the row order the operator sees).
func (h *BulkStopHandler) resolveTargets(ctx context.Context, req BulkStopRequest) ([]string, *apierr.DomainError) {
	hasIDs := len(req.RunIDs) > 0
	hasFilter := req.Filter != nil

	if !hasIDs && !hasFilter {
		return nil, apierr.BadRequest("either run_ids or filter is required")
	}
	if hasIDs && hasFilter {
		return nil, apierr.BadRequest("run_ids and filter are mutually exclusive")
	}

	if hasIDs {
		return dedupeRunIDs(req.RunIDs)
	}
	return h.resolveFilterTargets(ctx, *req.Filter)
}

func dedupeRunIDs(in []string) ([]string, *apierr.DomainError) {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, apierr.BadRequest("run_ids must contain at least one non-empty id")
	}
	if len(out) > maxBulkStopRunIDs {
		return nil, apierr.BadRequest("run_ids exceeds maximum of %d", maxBulkStopRunIDs)
	}
	return out, nil
}

// resolveFilterTargets reads the active activity ledger and returns the
// run IDs of records matching the (optional) lane and status predicates.
// Records without a RunID are skipped — StopRun has no meaningful action
// for an activity that never reached the agent-manager.
func (h *BulkStopHandler) resolveFilterTargets(ctx context.Context, filter BulkStopFilter) ([]string, *apierr.DomainError) {
	lane := strings.ToLower(strings.TrimSpace(filter.Lane))
	status := strings.ToLower(strings.TrimSpace(filter.Status))

	if lane != "" && !agentactivity.IsValidLane(agentactivity.Lane(lane)) {
		return nil, apierr.BadRequest("invalid lane %q (expected one of investigate|execute|review|reconcile)", filter.Lane)
	}
	if status != "" && !IsActiveStatus(status) {
		// Bulk-stop only makes sense over active records — stopping a
		// finished run is a no-op at best, and the UI never targets
		// them. Reject any non-active status so a typo surfaces as 400
		// rather than silently producing zero outcomes.
		return nil, apierr.BadRequest("status %q is not a stoppable active status", filter.Status)
	}

	records, err := h.activities.List(ctx, agentactivity.ListFilters{ActiveOnly: true})
	if err != nil {
		return nil, apierr.Internal("failed to list active activities: %v", err)
	}

	type candidate struct {
		runID       string
		requestedAt string
	}
	candidates := make([]candidate, 0, len(records))
	seen := make(map[string]struct{}, len(records))

	for _, rec := range records {
		runID := strings.TrimSpace(rec.RunID)
		if runID == "" {
			continue
		}
		if _, ok := seen[runID]; ok {
			continue
		}
		if status != "" && string(rec.Status) != status {
			continue
		}
		if lane != "" {
			recLane, lerr := agentactivity.LaneOf(rec.Purpose, rec.PhaseKind)
			if lerr != nil || string(recLane) != lane {
				continue
			}
		}
		seen[runID] = struct{}{}
		candidates = append(candidates, candidate{runID: runID, requestedAt: rec.RequestedAt})
	}

	// Newest-first cancellation. Mirrors Aggregate's ActivityRow ordering so
	// the operator's mental model ("stop the things at the top of my list
	// first") matches what the server actually does.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].requestedAt > candidates[j].requestedAt
	})

	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.runID)
	}
	return out, nil
}
