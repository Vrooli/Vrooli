// Batch queue operations for backlog items: topologically sorted queuing
// that respects dependency ordering and reports per-item status.
package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
)

// ExecutionQueuer abstracts execution operations needed by batch queue,
// allowing tests to inject a mock without constructing a real execution service.
type ExecutionQueuer interface {
	ProcessPreflight(ctx context.Context, backlogKind, backlogName string) (execution.ProcessPreflight, error)
	QueueBacklog(ctx context.Context, req execution.CreateRequest) (execution.Record, error)
	// ManuallyAcceptLatestForBacklog flips the most recent failed/needs_fixup
	// execution for the given backlog item to Completed and marks it manually
	// accepted. Returns (executionID, true) on success.
	ManuallyAcceptLatestForBacklog(ctx context.Context, backlogKind, backlogName, acceptor, reason string) (string, bool, error)
	// RetryLatestForBacklog creates a new execution attempt parented to the
	// most recent terminal execution for the given backlog item. Returns
	// (Record{}, false, nil) when the item has no executions at all; the
	// boolean false signals "no prior execution to retry" so handlers can
	// map it to a 400 distinct from internal errors.
	RetryLatestForBacklog(ctx context.Context, backlogKind, backlogName, note string) (execution.Record, bool, error)
}

// SetExecutionQueuer injects the execution service the backlog handler queues
// through. It is required for the queue/batch-queue/process-preflight paths: the
// backlog domain package no longer constructs an execution service itself (that
// path threaded a spawn-capable agent service into the domain; autonomous
// launches now flow exclusively through the operation runner). An unset queuer
// makes those endpoints return an Unavailable error rather than falling back.
func (h *Handler) SetExecutionQueuer(eq ExecutionQueuer) {
	h.executionQueuer = eq
}

// batchQueueRequest is the JSON request body for batch-queuing backlog items.
type batchQueueRequest struct {
	Items   []string `json:"items"`           // "kind/name" references
	Mode    string   `json:"mode,omitempty"`  // execution mode (manual, yolo)
	Confirm bool     `json:"confirm"`         // if false, preview only
	Force   bool     `json:"force,omitempty"` // override forceable blocking reasons
}

// batchQueueResponse is the JSON response for a batch queue operation.
type batchQueueResponse struct {
	Results        []batchQueueItemResult `json:"results"`
	ExecutionOrder []string               `json:"execution_order"`
}

// batchQueueItemResult reports the outcome of queuing a single item.
type batchQueueItemResult struct {
	Item              string   `json:"item"`
	Queued            bool     `json:"queued"`
	Message           string   `json:"message"`
	ExecutionID       string   `json:"execution_id,omitempty"`
	UnmetDependencies []string `json:"unmet_dependencies,omitempty"`
}

// BatchQueue queues multiple backlog items respecting dependency order.
// Items are topologically sorted and queued in order. Items with unmet
// dependencies are reported but not queued.
func (h *Handler) BatchQueue(w http.ResponseWriter, r *http.Request) {
	var req batchQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[backlog] batch-queue", apierr.BadRequest("%s", "invalid request body: "+err.Error()))
		return
	}

	if len(req.Items) == 0 {
		apierr.MapError(w, "[backlog] batch-queue", apierr.BadRequest("at least one item is required"))
		return
	}

	// Deduplicate items, preserving first-seen order.
	seen := make(map[string]bool, len(req.Items))
	unique := make([]string, 0, len(req.Items))
	for _, ref := range req.Items {
		if !seen[ref] {
			seen[ref] = true
			unique = append(unique, ref)
		}
	}
	req.Items = unique

	mode := execution.ModeYOLO
	if strings.TrimSpace(req.Mode) != "" {
		mode = execution.Mode(strings.ToLower(strings.TrimSpace(req.Mode)))
		if !execution.ValidateMode(mode) {
			apierr.MapError(w, "[backlog] batch-queue", apierr.BadRequest("invalid execution mode %q: must be manual or yolo", mode))
			return
		}
	}

	// Phase 1: Load all referenced items and validate they exist.
	type loadedItem struct {
		ref  string
		item BacklogItem
	}
	loaded := make([]loadedItem, 0, len(req.Items))
	itemMap := make(map[string]BacklogItem, len(req.Items))

	for _, ref := range req.Items {
		kind, name, err := parseDependencyRef(ref)
		if err != nil {
			apierr.MapError(w, "[backlog] batch-queue", apierr.BadRequest("invalid item reference %q: %s", ref, err.Error()))
			return
		}
		item, loadErr := h.store.LoadItem(kind, name)
		if loadErr != nil {
			if errors.Is(loadErr, ErrNotFound) {
				apierr.MapError(w, "[backlog] batch-queue", apierr.NotFound("item %q not found", ref))
				return
			}
			apierr.MapError(w, "[backlog] batch-queue", apierr.Internal("failed to load %q", ref))
			return
		}
		loaded = append(loaded, loadedItem{ref: ref, item: item})
		itemMap[ref] = item
	}

	// Phase 2: Build dependency graph and topological sort.
	g := depgraph.New()
	for _, li := range loaded {
		g.AddNode(li.ref, li.item.DependsOn)
	}

	if cycle, found := g.DetectCycle(); found {
		apierr.MapError(w, "[backlog] batch-queue",
			apierr.BadRequest("dependency cycle detected: %s", strings.Join(cycle, " -> ")))
		return
	}
	sortedOrder, _ := g.TopologicalSort()

	// Phase 3: Process each item in topological order.
	if h.executionQueuer == nil {
		apierr.MapError(w, "[backlog] batch-queue", apierr.Unavailable("execution service is not available"))
		return
	}
	eq := h.executionQueuer

	results := make([]batchQueueItemResult, 0, len(loaded))

	// Track which items we've successfully queued (treat as "will be completed")
	// for dependency evaluation within the batch.
	queuedInBatch := make(map[string]bool, len(loaded))

	for _, ref := range sortedOrder {
		item, exists := itemMap[ref]
		if !exists {
			continue // not in our batch, skip
		}

		result, queued := h.processBatchQueueItem(r.Context(), eq, ref, item, mode, req.Confirm, req.Force, queuedInBatch)
		results = append(results, result)
		if queued {
			queuedInBatch[ref] = true
		}
	}

	resp := batchQueueResponse{
		Results:        results,
		ExecutionOrder: sortedOrder,
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] batch-queue", apierr.Internal("failed to encode response"))
	}
}

// processBatchQueueItem evaluates and (unless preview) queues a single batch
// item. The returned bool reports whether the item should be treated as
// queued-in-batch for downstream dependency evaluation (true for both preview
// "ready" and a successful queue).
func (h *Handler) processBatchQueueItem(ctx context.Context, eq ExecutionQueuer, ref string, item BacklogItem, mode execution.Mode, confirm, force bool, queuedInBatch map[string]bool) (batchQueueItemResult, bool) {
	result := batchQueueItemResult{Item: ref}

	// Check if item status is queueable.
	if !isQueueableItem(item) {
		result.Message = fmt.Sprintf("Cannot queue from current status: %s", item.Status)
		return result, false
	}

	// Check dependencies: all must be completed or queued in this batch.
	unmet := computeUnmetDependencies(item.DependsOn, h.store, queuedInBatch)
	if len(unmet) > 0 {
		result.UnmetDependencies = unmet
		result.Message = "Unmet dependencies"
		return result, false
	}

	// Preview mode: report as ready but don't actually queue.
	if !confirm {
		result.Message = "Ready to queue (preview mode)"
		return result, true
	}

	// Run preflight check.
	preflight, preflightErr := eq.ProcessPreflight(ctx, string(item.Kind), item.Name)
	if preflightErr != nil {
		slog.Error("batch-queue preflight failed", "item", ref, "err", preflightErr)
		result.Message = "Preflight check failed: " + httputil.TruncateErrorMessage(preflightErr, 240)
		return result, false
	}

	if msg, blocked := h.batchQueueBlockingMessage(item, preflight, force); blocked {
		result.Message = msg
		return result, false
	}

	// Queue the item.
	record, queueErr := eq.QueueBacklog(ctx, execution.CreateRequest{
		BacklogKind: string(item.Kind),
		BacklogName: item.Name,
		Mode:        mode,
		StartedBy:   "swarm-manager",
		Operation:   "generator",
		Force:       force,
	})
	if queueErr != nil {
		slog.Error("batch-queue failed to queue item", "item", ref, "err", queueErr)
		result.Message = "Queue failed: " + httputil.TruncateErrorMessage(queueErr, 240)
		return result, false
	}

	result.Queued = true
	result.ExecutionID = record.ExecutionID
	result.Message = "Queued successfully"
	slog.Info("batch-queue item queued", "item", ref, "execution_id", record.ExecutionID)
	return result, true
}

// batchQueueBlockingMessage evaluates preflight + workshop blocking reasons for
// a batch item. It returns a "Blocked: ..." message and true when the item must
// not be queued (respecting the force override for forceable-only reasons).
func (h *Handler) batchQueueBlockingMessage(item BacklogItem, preflight execution.ProcessPreflight, force bool) (string, bool) {
	var blockingReasons []BlockingReason
	for _, reason := range preflight.BlockingReasons {
		blockingReasons = append(blockingReasons, BlockingReason{Message: reason, Forceable: false})
	}
	blockingReasons = DedupeReasons(blockingReasons)

	if len(blockingReasons) == 0 {
		return "", false
	}
	if force && !HasNonForceableReasons(blockingReasons) {
		return "", false
	}
	msgs := make([]string, len(blockingReasons))
	for i, r := range blockingReasons {
		msgs[i] = r.Message
	}
	return "Blocked: " + strings.Join(msgs, "; "), true
}

// computeUnmetDependencies returns dependencies that are neither completed on
// disk nor queued in the current batch. Dependencies whose specs no longer
// exist are presumed completed & archived, so they never block execution.
func computeUnmetDependencies(dependsOn []string, store Store, queuedInBatch map[string]bool) []string {
	if len(dependsOn) == 0 {
		return nil
	}
	var unmet []string
	for _, ref := range dependsOn {
		if queuedInBatch[ref] {
			continue // will be processed before this item in topological order
		}
		kind, name, err := parseDependencyRef(ref)
		if err != nil {
			// Unparseable refs are skipped — cannot be validated.
			continue
		}
		item, loadErr := store.LoadItem(kind, name)
		if loadErr != nil {
			// Missing/unloadable specs are presumed completed & archived.
			continue
		}
		if blockingDepStatuses[item.Status] {
			unmet = append(unmet, ref)
		}
	}
	return unmet
}
