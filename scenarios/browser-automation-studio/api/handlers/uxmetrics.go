package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
)

// UXMetricsHandler handles UX metrics HTTP endpoints.
type UXMetricsHandler struct {
	service uxmetrics.Service
	log     *logrus.Logger
}

// NewUXMetricsHandler creates a new UX metrics handler.
func NewUXMetricsHandler(service uxmetrics.Service, log *logrus.Logger) *UXMetricsHandler {
	return &UXMetricsHandler{
		service: service,
		log:     log,
	}
}

// GetExecutionMetrics retrieves computed UX metrics for an execution.
// GET /api/v1/executions/{id}/ux-metrics
func (h *UXMetricsHandler) GetExecutionMetrics(w http.ResponseWriter, r *http.Request) {
	// Check entitlement (Pro tier required)
	ent := entitlement.FromContext(r.Context())
	if ent != nil && !ent.Tier.AtLeast(entitlement.TierPro) {
		h.jsonError(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
		return
	}

	executionID, err := h.extractUUID(r, "id")
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid execution id")
		return
	}

	metrics, err := h.service.GetExecutionMetrics(r.Context(), executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution metrics")
		h.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if metrics == nil {
		h.jsonError(w, http.StatusNotFound, "metrics not found for this execution")
		return
	}

	h.jsonOK(w, metrics)
}

// GetStepMetrics retrieves UX metrics for a specific step.
// GET /api/v1/executions/{id}/ux-metrics/steps/{stepIndex}
func (h *UXMetricsHandler) GetStepMetrics(w http.ResponseWriter, r *http.Request) {
	ent := entitlement.FromContext(r.Context())
	if ent != nil && !ent.Tier.AtLeast(entitlement.TierPro) {
		h.jsonError(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
		return
	}

	executionID, err := h.extractUUID(r, "id")
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid execution id")
		return
	}

	stepIndexStr := chi.URLParam(r, "stepIndex")
	stepIndex, err := strconv.Atoi(stepIndexStr)
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid step index")
		return
	}

	metrics, err := h.service.Analyzer().AnalyzeStep(r.Context(), executionID, stepIndex)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"execution_id": executionID,
			"step_index":   stepIndex,
		}).Error("Failed to analyze step")
		h.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if metrics == nil {
		h.jsonError(w, http.StatusNotFound, "step metrics not found")
		return
	}

	h.jsonOK(w, metrics)
}

// ComputeMetrics triggers metric computation for a completed execution.
// POST /api/v1/executions/{id}/ux-metrics/compute
func (h *UXMetricsHandler) ComputeMetrics(w http.ResponseWriter, r *http.Request) {
	ent := entitlement.FromContext(r.Context())
	if ent != nil && !ent.Tier.AtLeast(entitlement.TierPro) {
		h.jsonError(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
		return
	}

	executionID, err := h.extractUUID(r, "id")
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid execution id")
		return
	}

	metrics, err := h.service.ComputeAndSaveMetrics(r.Context(), executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to compute metrics")
		h.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonOK(w, metrics)
}

// GetWorkflowMetricsAggregate retrieves aggregated metrics across workflow executions.
// GET /api/v1/workflows/{id}/ux-metrics/aggregate
func (h *UXMetricsHandler) GetWorkflowMetricsAggregate(w http.ResponseWriter, r *http.Request) {
	ent := entitlement.FromContext(r.Context())
	if ent != nil && !ent.Tier.AtLeast(entitlement.TierPro) {
		h.jsonError(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
		return
	}

	workflowID, err := h.extractUUID(r, "id")
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid workflow id")
		return
	}

	// Get limit from query params, default to 10
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	aggregate, err := h.service.GetWorkflowAggregate(r.Context(), workflowID, limit)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to get workflow metrics aggregate")
		h.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonOK(w, aggregate)
}

// extractUUID extracts a UUID from the request path.
func (h *UXMetricsHandler) extractUUID(r *http.Request, param string) (uuid.UUID, error) {
	idStr := chi.URLParam(r, param)
	return uuid.Parse(idStr)
}

// jsonOK sends a JSON response with HTTP 200 OK.
func (h *UXMetricsHandler) jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.WithError(err).Error("Failed to encode JSON response")
	}
}

// jsonError sends a JSON error response.
func (h *UXMetricsHandler) jsonError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
