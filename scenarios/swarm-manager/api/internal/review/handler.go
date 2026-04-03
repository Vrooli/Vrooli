package review

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// Handler provides HTTP endpoints for review evidence management.
type Handler struct {
	service *Service
}

// NewHandler creates a new review handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all review evidence endpoints on the router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review", h.ListRounds).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review/captures/{filepath:.*}", h.GetCapture).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review/{round}/verify/{evidence_id}", h.VerifyEvidence).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review/{round}/request", h.RequestMoreEvidence).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review/{round}/request/{thread_id}", h.ContinueRequest).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review/{round}/request/{thread_id}/dismiss", h.DismissRequest).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/trigger-review-agent", h.TriggerReviewAgent).Methods("POST")
}

// ListRounds returns all review evidence rounds for a backlog item.
func (h *Handler) ListRounds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]

	rounds, err := h.service.ListRounds(kind, name)
	if err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("list review rounds: %v", err))
		return
	}
	if rounds == nil {
		rounds = []Round{}
	}

	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"rounds": rounds}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

// GetCapture serves a binary capture file from the review/captures/ directory.
func (h *Handler) GetCapture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]
	capturePath := vars["filepath"]

	itemDir := h.service.resolveItemDir(kind, name)
	data, err := LoadCapture(itemDir, "captures/"+capturePath)
	if err != nil {
		apierr.MapError(w, "[review]", apierr.NotFound("capture not found: %v", err))
		return
	}

	// Detect content type from extension.
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// VerifyEvidence marks an evidence item as verified or unverified.
func (h *Handler) VerifyEvidence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]
	evidenceID := vars["evidence_id"]

	roundNum, err := strconv.Atoi(vars["round"])
	if err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid round number"))
		return
	}

	var body struct {
		Verified    bool   `json:"verified"`
		ExecutionID string `json:"execution_id"`
	}
	if err := readJSON(r, &body); err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid request body: %v", err))
		return
	}

	if err := h.service.VerifyEvidence(kind, name, roundNum, evidenceID, body.Verified, body.ExecutionID); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("verify evidence: %v", err))
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"ok": true}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

// RequestMoreEvidence creates a new evidence request thread.
func (h *Handler) RequestMoreEvidence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]

	roundNum, err := strconv.Atoi(vars["round"])
	if err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid round number"))
		return
	}

	var body struct {
		Message    string `json:"message"`
		EvidenceID string `json:"evidence_id"`
	}
	if err := readJSON(r, &body); err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid request body: %v", err))
		return
	}
	if body.Message == "" {
		apierr.MapError(w, "[review]", apierr.BadRequest("message is required"))
		return
	}

	threadID, err := h.service.RequestMoreEvidence(r.Context(), kind, name, roundNum, body.Message, body.EvidenceID)
	if err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("request more evidence: %v", err))
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{"thread_id": threadID}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

// ContinueRequest appends a message to an existing request thread.
func (h *Handler) ContinueRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]
	threadID := vars["thread_id"]

	roundNum, err := strconv.Atoi(vars["round"])
	if err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid round number"))
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := readJSON(r, &body); err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid request body: %v", err))
		return
	}
	if body.Message == "" {
		apierr.MapError(w, "[review]", apierr.BadRequest("message is required"))
		return
	}

	if err := h.service.ContinueRequest(kind, name, roundNum, threadID, body.Message); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("continue request: %v", err))
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"ok": true}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

// DismissRequest marks a request thread as dismissed.
func (h *Handler) DismissRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kind, name := vars["kind"], vars["name"]
	threadID := vars["thread_id"]

	roundNum, err := strconv.Atoi(vars["round"])
	if err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid round number"))
		return
	}

	if err := h.service.DismissRequest(kind, name, roundNum, threadID); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("dismiss request: %v", err))
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"ok": true}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

// TriggerReviewAgent manually triggers a review agent for an execution.
func (h *Handler) TriggerReviewAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["execution_id"]

	var body struct {
		BacklogKind       string   `json:"backlog_kind"`
		BacklogName       string   `json:"backlog_name"`
		AffectedScenarios []string `json:"affected_scenarios"`
	}
	if err := readJSON(r, &body); err != nil {
		apierr.MapError(w, "[review]", apierr.BadRequest("invalid request body: %v", err))
		return
	}
	if body.BacklogKind == "" || body.BacklogName == "" {
		apierr.MapError(w, "[review]", apierr.BadRequest("backlog_kind and backlog_name are required"))
		return
	}

	if err := h.service.TriggerReviewAgent(r.Context(), body.BacklogKind, body.BacklogName, executionID, body.AffectedScenarios); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("trigger review agent: %v", err))
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusAccepted, map[string]any{"ok": true}); err != nil {
		apierr.MapError(w, "[review]", apierr.Internal("failed to encode response"))
	}
}

func readJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		return nil // Allow empty body for optional fields.
	}
	return json.Unmarshal(body, v)
}
