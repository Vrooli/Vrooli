// Batch queue operations for backlog items: topologically sorted queuing
// that respects dependency ordering and reports per-item status.
package backlog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
)

// batchQueueRequest is the JSON request body for batch-queuing backlog items.
type batchQueueRequest struct {
	Items   []string `json:"items"`           // "kind/name" references
	Mode    string   `json:"mode,omitempty"`  // execution mode (manual, yolo, scheduled)
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
		httputil.BadRequest(w, "[backlog] batch-queue", "invalid request body: "+err.Error())
		return
	}

	if len(req.Items) == 0 {
		httputil.BadRequest(w, "[backlog] batch-queue", "at least one item is required")
		return
	}

	mode := execution.ModeYOLO
	if strings.TrimSpace(req.Mode) != "" {
		mode = execution.Mode(strings.ToLower(strings.TrimSpace(req.Mode)))
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
			httputil.BadRequest(w, "[backlog] batch-queue", fmt.Sprintf("invalid item reference %q: %s", ref, err.Error()))
			return
		}
		item, loadErr := h.store.LoadItem(kind, name)
		if loadErr != nil {
			if os.IsNotExist(loadErr) {
				httputil.NotFound(w, "[backlog] batch-queue", fmt.Sprintf("item %q not found", ref))
				return
			}
			httputil.InternalError(w, "[backlog] batch-queue", fmt.Sprintf("failed to load %q", ref))
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

	sortedOrder, sortErr := g.TopologicalSort()
	if sortErr != nil {
		httputil.BadRequest(w, "[backlog] batch-queue", "dependency cycle detected among requested items")
		return
	}

	// Phase 3: Process each item in topological order.
	executionService := execution.NewService(execution.ServiceConfig{
		RootDir:      h.rootDir,
		StorePath:    filepath.Join(h.rootDir, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(h.rootDir, ".vrooli", "execution-policy.json"),
		AgentService: h.agentService,
	})

	results := make([]batchQueueItemResult, 0, len(loaded))

	// Track which items we've successfully queued (treat as "will be completed")
	// for dependency evaluation within the batch.
	queuedInBatch := make(map[string]bool, len(loaded))

	for _, ref := range sortedOrder {
		item, exists := itemMap[ref]
		if !exists {
			continue // not in our batch, skip
		}

		result := batchQueueItemResult{Item: ref}

		// Check if item kind is research (not queueable).
		if item.Kind == KindResearch {
			result.Message = "Research items must be converted before processing"
			results = append(results, result)
			continue
		}

		// Check if item status is queueable.
		if !isQueueableStatus(item.Kind, item.Status) {
			result.Message = fmt.Sprintf("Cannot queue from current status: %s", item.Status)
			results = append(results, result)
			continue
		}

		// Check dependencies: all must be completed or queued in this batch.
		unmet := computeUnmetDependencies(item.DependsOn, h.store, queuedInBatch)
		if len(unmet) > 0 {
			result.UnmetDependencies = unmet
			result.Message = "Unmet dependencies"
			results = append(results, result)
			continue
		}

		// Preview mode: report as ready but don't actually queue.
		if !req.Confirm {
			result.Message = "Ready to queue (preview mode)"
			results = append(results, result)
			queuedInBatch[ref] = true
			continue
		}

		// Run preflight check.
		preflight, preflightErr := executionService.ProcessPreflight(r.Context(), string(item.Kind), item.Name)
		if preflightErr != nil {
			log.Printf("[backlog] batch-queue: preflight failed for %s: %v", ref, preflightErr)
			result.Message = "Preflight check failed"
			results = append(results, result)
			continue
		}

		// Check workshop feedback for additional blocking.
		itemDir := h.store.ItemDir(item.Kind, item.Name)
		latestRound, _, _ := LoadLatestRound(itemDir)
		pendingDecisions := CountPendingDecisions(latestRound)
		blockingReasons := append([]string{}, preflight.BlockingReasons...)
		if pendingDecisions > 0 {
			blockingReasons = append(blockingReasons, fmt.Sprintf("%d workshop decision(s) still pending", pendingDecisions))
		}
		blockingReasons = dedupeQueueReasons(blockingReasons)

		if len(blockingReasons) > 0 {
			if !req.Force || hasNonForceableQueueReasons(blockingReasons) {
				result.Message = "Blocked: " + strings.Join(blockingReasons, "; ")
				results = append(results, result)
				continue
			}
		}

		// Queue the item.
		record, queueErr := executionService.QueueBacklog(r.Context(), execution.CreateRequest{
			BacklogKind: string(item.Kind),
			BacklogName: item.Name,
			Mode:        mode,
			StartedBy:   "swarm-manager",
			Operation:   "generator",
			Force:       req.Force,
		})
		if queueErr != nil {
			log.Printf("[backlog] batch-queue: failed to queue %s: %v", ref, queueErr)
			result.Message = "Queue failed: " + queueErr.Error()
			results = append(results, result)
			continue
		}

		result.Queued = true
		result.ExecutionID = record.ExecutionID
		result.Message = "Queued successfully"
		results = append(results, result)
		queuedInBatch[ref] = true

		log.Printf("[backlog] batch-queue: queued %s (execution_id=%s)", ref, record.ExecutionID)
	}

	resp := batchQueueResponse{
		Results:        results,
		ExecutionOrder: sortedOrder,
	}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] batch-queue", "failed to encode response")
	}
}

// computeUnmetDependencies returns dependencies that are neither completed on
// disk nor queued in the current batch.
func computeUnmetDependencies(dependsOn []string, store *Store, queuedInBatch map[string]bool) []string {
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
			unmet = append(unmet, ref)
			continue
		}
		item, loadErr := store.LoadItem(kind, name)
		if loadErr != nil {
			unmet = append(unmet, ref)
			continue
		}
		if item.Status != StatusCompleted {
			unmet = append(unmet, ref)
		}
	}
	return unmet
}

// checkDependencyReadiness verifies that all of an item's dependencies are
// completed. Returns unmet dependency references and an error if deps can't
// be loaded.
func checkDependencyReadiness(dependsOn []string, store *Store) ([]string, error) {
	return store.CheckDependencies(dependsOn)
}

// formatDependencyBlockingReason returns a human-readable blocking reason for
// unmet dependencies.
func formatDependencyBlockingReason(unmet []string) string {
	if len(unmet) == 0 {
		return ""
	}
	return fmt.Sprintf("unmet dependencies: %s", strings.Join(unmet, ", "))
}

// appendDependencyBlockingReasons checks an item's dependencies and appends
// a blocking reason if any are unmet. Used by the single-item Queue handler.
func appendDependencyBlockingReasons(item BacklogItem, store *Store, reasons []string) ([]string, error) {
	if len(item.DependsOn) == 0 {
		return reasons, nil
	}
	unmet, err := checkDependencyReadiness(item.DependsOn, store)
	if err != nil {
		return reasons, err
	}
	if msg := formatDependencyBlockingReason(unmet); msg != "" {
		reasons = append(reasons, msg)
	}
	return reasons, nil
}
