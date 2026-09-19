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
	"context"
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

// EventLogger records view events for analytics.
type EventLogger interface {
	EmitCaptureViewed(captureID string)
}

// CaptureIndexer is an optional write-through semantic projection. The
// capture domain remains independent of aisearch; composition supplies the
// implementation when the embedding stack is configured.
type CaptureIndexer interface {
	IndexCapture(context.Context, string, string, string) error
	DeleteCapture(context.Context, string) error
}

// GroundingProvider builds read-only context for the classify workflow. A
// provider failure is deliberately non-fatal: the input still carries an
// explicit degraded marker so the contract skill can choose the conservative
// research landing.
type GroundingProvider interface {
	BuildCaptureGrounding(context.Context, string) (map[string]any, error)
}

// ProposalRecorder is the durable handoff from the disposable capture event
// to the agent-session decision rail. Composition supplies the recorder so
// capture intake does not own session storage.
type ProposalRecorder interface {
	RecordCaptureProposals(context.Context, string, string, string, string, string, string, []byte) error
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
	eventDispatcher        dispatch.Invalidator
	eventLogger            EventLogger
	aiIndexer              CaptureIndexer
	groundingProvider      GroundingProvider
	proposalRecorder       ProposalRecorder
}

func (h *Handler) SetAIIndexer(indexer CaptureIndexer) { h.aiIndexer = indexer }

func (h *Handler) SetGroundingProvider(provider GroundingProvider) { h.groundingProvider = provider }

func (h *Handler) SetProposalRecorder(recorder ProposalRecorder) { h.proposalRecorder = recorder }

func (h *Handler) indexCapture(cap *capture) {
	if h.aiIndexer == nil {
		return
	}
	go func(id, text, note string) {
		if err := h.aiIndexer.IndexCapture(context.Background(), id, text, note); err != nil {
			slog.Debug("captures: semantic index upsert failed", "capture_id", id, "err", err)
		}
	}(cap.ID, cap.Text, cap.Note)
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
	h.indexCapture(cap)
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
	if h.aiIndexer != nil {
		if err := h.aiIndexer.DeleteCapture(r.Context(), id); err != nil {
			slog.Debug("captures: semantic index delete failed", "capture_id", id, "err", err)
		}
	}
	h.invalidateTopologyGraph()
	w.WriteHeader(http.StatusNoContent)
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
