// Package captures provides HTTP handlers for quick-capture operations.
//
// Captures are raw, unclassified thoughts entered by the user. They are
// stored as folders under {rootDir}/captures/{id}/ with a capture.json file.
// An AI classification agent automatically analyzes the text and suggests
// one or more backlog items for user confirmation.
//
// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md
package captures

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	"github.com/gorilla/mux"
)

// BacklogItemDraft is the complete capture-derived item contract handed to the
// backlog adapter. Keeping the draft here prevents the capture endpoint from
// silently discarding classification fields before the domain adapter sees
// them.
type BacklogItemDraft struct {
	Kind        string
	Name        string
	Title       string
	Description string
	Priority    int
	Tags        []string
	SpawnedFrom string
}

// BacklogItemCreator abstracts backlog item creation for capture-derived items.
type BacklogItemCreator interface {
	SaveItem(draft BacklogItemDraft) error
}

// EventLogger records view events for analytics.
type EventLogger interface {
	EmitCaptureViewed(captureID string)
}

// Handler provides HTTP handlers for capture operations.
//
// cacheRoot is the runtime-home cache directory (captures are disposable
// per the cache-class invariant: paths_test.go T-R2). Captures live at
// `<cacheRoot>/captures/<id>/`.
type Handler struct {
	cacheRoot              string
	transitionRoot         string
	classificationWorkflow agentmanager.WorkflowInvoker
	transitionRegistry     transitions.Registry
	transitionRunner       *transitionrunner.Runner
	classificationMu       sync.Mutex
	backlogCreator         BacklogItemCreator
	eventDispatcher        dispatch.Invalidator
	eventLogger            EventLogger
}

// NewHandler creates a new captures handler. transitionRoot is where the
// correlation journal lives; it is deliberately separate from cacheRoot because
// captures themselves are disposable cache content while a pending transition
// result is durable state that must survive a cache eviction.
//
// The runner built here is a standalone fallback for tests. Production
// composition replaces it through SetTransitionRunner so every subject shares
// one runner and one journal.
func NewHandler(cacheRoot, transitionRoot string, registry transitions.Registry) *Handler {
	h := &Handler{
		cacheRoot:              cacheRoot,
		transitionRoot:         transitionRoot,
		classificationWorkflow: agentmanager.NewWorkflowService(),
		transitionRegistry:     registry,
	}
	h.configureTransitionRunner()
	return h
}

// SetClassificationWorkflow injects the Agent Manager workflow transport.
func (h *Handler) SetClassificationWorkflow(workflow agentmanager.WorkflowInvoker) {
	h.classificationWorkflow = workflow
	h.configureTransitionRunner()
}

func (h *Handler) configureTransitionRunner() {
	if h.classificationWorkflow == nil {
		h.transitionRunner = nil
		return
	}
	h.transitionRunner = transitionrunner.New(h.transitionRegistry, h.classificationWorkflow, transitionrun.NewFileStore(filepath.Join(h.transitionRoot, "transition-runs")), nil)
	h.transitionRunner.RegisterInput("capture.classify", h.buildClassificationInput)
	h.transitionRunner.RegisterApply("apply_capture_classification", h.applyClassificationOutcome)
}

// RegisterTransitionAdapter contributes capture's two domain functions to a
// server-owned runner. Composition owns the runner and its durable store;
// captures only owns its immutable snapshot and mutation implementation.
func (h *Handler) RegisterTransitionAdapter(registrar transitionrunner.Registrar) {
	registrar.RegisterInput("capture.classify", h.buildClassificationInput)
	registrar.RegisterApply("apply_capture_classification", h.applyClassificationOutcome)
}

// SetTransitionRunner replaces the handler-local test runner with the shared
// server-owned runner used in production.
func (h *Handler) SetTransitionRunner(runner *transitionrunner.Runner) {
	h.transitionRunner = runner
}

// SetWorkflowStartGuard applies transition-registry preflight policy at the
// composition edge without leaking it into capture domain code.
func (h *Handler) SetWorkflowStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := h.classificationWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

// SetWorkflowActivityRecorder makes capture classification launches visible in
// the common activity ledger at the WorkflowService transport boundary.
func (h *Handler) SetWorkflowActivityRecorder(recorder agentmanager.WorkflowActivityRecorder) {
	if workflow, ok := h.classificationWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetWorkflowActivityRecorder(recorder)
	}
}

// SetBacklogCreator sets the backlog item creator for capture-derived items.
func (h *Handler) SetBacklogCreator(creator BacklogItemCreator) {
	h.backlogCreator = creator
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (h *Handler) SetEventLogger(l EventLogger) {
	h.eventLogger = l
}

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (h *Handler) SetEventDispatcher(d dispatch.Invalidator) {
	h.eventDispatcher = d
}

func (h *Handler) invalidateTopologyGraph() {
	if h.eventDispatcher == nil {
		return
	}
	h.eventDispatcher.DispatchInvalidate("topology", "plan")
}

func (h *Handler) invalidateAllGraphLenses() {
	if h.eventDispatcher == nil {
		return
	}
	h.eventDispatcher.DispatchInvalidate("topology", "plan")
}

// RegisterRoutes registers capture endpoints on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/captures", h.List).Methods("GET")
	r.HandleFunc("/api/v1/captures", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/captures/{id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/captures/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/captures/{id}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/captures/{id}/create-item", h.CreateItem).Methods("POST")
}

// List returns all captures, newest first.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	capturesRoot := h.capturesDir()
	entries, err := os.ReadDir(capturesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			_ = httputil.JSON(w, map[string]any{"captures": []any{}})
			return
		}
		apierr.MapError(w, "[captures] list", apierr.Internal("failed to read captures directory"))
		return
	}

	var caps []capture
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cap, err := h.loadCapture(entry.Name())
		if err != nil {
			slog.Warn("skipping capture", "id", entry.Name(), "error", err)
			continue
		}
		caps = append(caps, *cap)
	}

	// Sort by created time descending (newest first).
	sort.Slice(caps, func(i, j int) bool {
		return caps[i].Created > caps[j].Created
	})

	if caps == nil {
		caps = []capture{}
	}
	_ = httputil.JSON(w, map[string]any{"captures": caps})
}

// Get returns a single capture by ID.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] get", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] get", apierr.Internal("failed to load capture"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"capture": cap})
	if h.eventLogger != nil {
		h.eventLogger.EmitCaptureViewed(id)
	}
}

// Update modifies mutable fields on a capture (currently only note).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req struct {
		Note *string `json:"note"`
	}
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[captures] update", apierr.BadRequest("invalid request body: %s", err))
		return
	}
	if req.Note == nil {
		apierr.MapError(w, "[captures] update", apierr.BadRequest("at least one field must be provided"))
		return
	}

	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] update", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] update", apierr.Internal("failed to load capture"))
		return
	}

	cap.Note = strings.TrimSpace(*req.Note)
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] update", apierr.Internal("failed to save capture"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"capture": cap})
}

// Delete removes a capture and its folder.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dir := h.captureDir(id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Idempotent: already deleted.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		apierr.MapError(w, "[captures] delete", apierr.Internal("failed to delete capture"))
		return
	}
	h.invalidateTopologyGraph()
	w.WriteHeader(http.StatusNoContent)
}

// CreateItem creates a backlog item from a classified capture.
// Each classified item becomes one backlog item and carries the capture ID as
// provenance. The endpoint remains temporarily available for compatibility;
// the proposal-rail migration removes it in the later intake phase.
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	if h.backlogCreator == nil {
		apierr.MapError(w, "[captures] create-item", apierr.Internal("backlog creator not configured"))
		return
	}

	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] create-item", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] create-item", apierr.Internal("failed to load capture"))
		return
	}

	if cap.Classification == nil || len(cap.Classification.Items) == 0 {
		apierr.MapError(w, "[captures] create-item", apierr.Conflict("capture has no classified items"))
		return
	}

	created := make([]map[string]any, 0, len(cap.Classification.Items))
	for index, classificationItem := range cap.Classification.Items {
		kind := strings.ToLower(strings.TrimSpace(classificationItem.Kind))
		title := strings.TrimSpace(classificationItem.Title)
		if title == "" {
			title = truncate(cap.Text, 80)
		}
		name := sanitizeCaptureItemName(title)
		if name == "" {
			name = "capture-item-" + id
		}
		if index > 0 {
			name = fmt.Sprintf("%s-%d", name, index+1)
		}
		priority := classificationItem.Priority
		if priority < 1 || priority > 10 {
			priority = 5
		}
		if err := h.backlogCreator.SaveItem(BacklogItemDraft{
			Kind:        kind,
			Name:        name,
			Title:       title,
			Description: classificationItem.Description,
			Priority:    priority,
			Tags:        deduplicateTags(classificationItem.Tags),
			SpawnedFrom: id,
		}); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				apierr.MapError(w, "[captures] create-item", apierr.Conflict("backlog item already exists"))
				return
			}
			slog.Error("failed to create backlog item from capture", "error", err, "capture_id", id, "item_index", index)
			apierr.MapError(w, "[captures] create-item", apierr.Internal("failed to create backlog item"))
			return
		}
		created = append(created, map[string]any{"kind": kind, "name": name, "priority": priority, "spawned_from": id})
	}

	// Mark capture as classified.
	cap.Status = "classified"
	_ = h.writeCapture(cap)

	slog.Info("created backlog items from capture", "count", len(created), "capture_id", id)
	h.invalidateAllGraphLenses()
	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{
		"items": created,
		"count": len(created),
	})
}

func deduplicateTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

// sanitizeCaptureItemName converts a title to a folder-safe name.
func sanitizeCaptureItemName(title string) string {
	name := strings.ToLower(title)
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	s := result.String()
	// Truncate to a reasonable length for folder names.
	if len(s) > 60 {
		s = s[:60]
	}
	return strings.TrimRight(s, "-")
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
