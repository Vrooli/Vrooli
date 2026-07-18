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
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// BacklogItemCreator abstracts backlog item creation for the create-item endpoint.
type BacklogItemCreator interface {
	ItemDir(kind string, name string) string
	SaveItem(kind, name, title, description string, tags []string) error
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
	classificationWorkflow agentmanager.WorkflowInvoker
	classificationMu       sync.Mutex
	backlogCreator         BacklogItemCreator
	eventDispatcher        dispatch.Invalidator
	eventLogger            EventLogger
}

// NewHandler creates a new captures handler.
func NewHandler(cacheRoot string) *Handler {
	return &Handler{cacheRoot: cacheRoot, classificationWorkflow: agentmanager.NewWorkflowService()}
}

// SetClassificationWorkflow injects the Agent Manager workflow transport.
func (h *Handler) SetClassificationWorkflow(workflow agentmanager.WorkflowInvoker) {
	h.classificationWorkflow = workflow
}

// SetWorkflowStartGuard applies transition-registry preflight policy at the
// composition edge without leaking it into capture domain code.
func (h *Handler) SetWorkflowStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := h.classificationWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

// SetBacklogCreator sets the backlog item creator for the create-item endpoint.
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
	r.HandleFunc("/api/v1/captures/{id}/classify", h.Classify).Methods("POST")
	r.HandleFunc("/api/v1/captures/{id}/classify/{executionID}/apply", h.ApplyClassification).Methods("POST")
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

// createItemRequest is the JSON body for CreateItem.
type createItemRequest struct {
	Kind string `json:"kind"`
}

// CreateItem creates a backlog item from a classified capture.
// It pre-fills the item with text from the capture and tags from the classification.
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

	// Parse optional kind override from request body.
	kind := "execute"
	var req createItemRequest
	if r.Body != nil {
		if decErr := json.NewDecoder(r.Body).Decode(&req); decErr == nil && req.Kind != "" {
			kind = strings.ToLower(strings.TrimSpace(req.Kind))
		}
	}

	// Build item title from capture text (truncated).
	title := truncate(cap.Text, 80)

	// Collect tags from classification items.
	var tags []string
	if cap.Classification != nil {
		for _, ci := range cap.Classification.Items {
			tags = append(tags, ci.Tags...)
		}
	}
	// Deduplicate tags.
	seen := make(map[string]bool, len(tags))
	dedupTags := make([]string, 0, len(tags))
	for _, t := range tags {
		if !seen[t] {
			seen[t] = true
			dedupTags = append(dedupTags, t)
		}
	}

	// Generate a name from the title.
	name := sanitizeCaptureItemName(title)
	if name == "" {
		name = "capture-item-" + id
	}

	if err := h.backlogCreator.SaveItem(kind, name, title, cap.Text, dedupTags); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			apierr.MapError(w, "[captures] create-item", apierr.Conflict("backlog item already exists"))
			return
		}
		slog.Error("failed to create backlog item from capture", "error", err)
		apierr.MapError(w, "[captures] create-item", apierr.Internal("failed to create backlog item"))
		return
	}

	// Mark capture as classified.
	cap.Status = "classified"
	_ = h.writeCapture(cap)

	slog.Info("created backlog item from capture", "kind", kind, "name", name, "capture_id", id)
	h.invalidateAllGraphLenses()
	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{
		"kind": kind,
		"name": name,
	})
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
