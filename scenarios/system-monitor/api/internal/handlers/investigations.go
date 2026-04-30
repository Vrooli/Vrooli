package handlers
// DOC: docs/reference/api-endpoints.md#investigations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"system-monitor-api/internal/apierrors"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/convert"
	"system-monitor-api/internal/httputil"
	"system-monitor-api/internal/models"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
)

// InvestigationHandler handles investigation-related requests
type InvestigationHandler struct {
	log              *slog.Logger
	config           *config.Config
	investigationSvc InvestigationManager
	scriptSvc        ScriptRunner
}

// NewInvestigationHandler creates a new investigation handler
func NewInvestigationHandler(cfg *config.Config, investigationSvc InvestigationManager, scriptSvc ScriptRunner, log *slog.Logger) *InvestigationHandler {
	return &InvestigationHandler{
		log:              log,
		config:           cfg,
		investigationSvc: investigationSvc,
		scriptSvc:        scriptSvc,
	}
}

// ListInvestigations handles GET /api/v1/investigations
func (h *InvestigationHandler) ListInvestigations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 20
	if queryLimit := r.URL.Query().Get("limit"); queryLimit != "" {
		if parsed, err := strconv.Atoi(queryLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	investigations, err := h.investigationSvc.ListInvestigations(ctx, limit)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.ListInvestigationsResponse{
		Investigations: convert.InvestigationsToProto(investigations),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// GetLatestInvestigation handles GET /api/v1/investigations/latest
func (h *InvestigationHandler) GetLatestInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// TriggerInvestigation handles POST /api/v1/investigations/trigger
func (h *InvestigationHandler) TriggerInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var pbReq apipb.TriggerInvestigationRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		if r.ContentLength <= 0 || errors.Is(err, io.EOF) {
			// Allow empty body for backwards compatibility
			pbReq.AutoFix = false
			pbReq.Note = ""
		} else {
			httputil.HandleError(w, h.log, r, apierrors.Validation("body", "invalid JSON"))
			return
		}
	}

	investigation, err := h.investigationSvc.TriggerInvestigation(ctx, pbReq.AutoFix, pbReq.Note)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	// Return immediate response with API info
	resp := &apipb.TriggerInvestigationResponse{
		Status:          models.StatusQueued,
		InvestigationId: investigation.ID,
		ApiBaseUrl:      h.resolveAPIBaseURL(r),
		Message:         "Investigation queued for processing",
		AutoFix:         pbReq.AutoFix,
		Note:            pbReq.Note,
	}
	httputil.SafeProtoJSONWithStatus(w, h.log, r, http.StatusAccepted, resp)
}

// GetInvestigation handles GET /api/v1/investigations/{id}
func (h *InvestigationHandler) GetInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetInvestigation(ctx, id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// UpdateInvestigationStatus handles PUT /api/v1/investigations/{id}/status
func (h *InvestigationHandler) UpdateInvestigationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.UpdateInvestigationStatusRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	// Convert proto enum to string for the service layer
	statusStr := strings.ToLower(strings.TrimPrefix(pbReq.Status.String(), "INVESTIGATION_STATUS_"))

	if err := h.investigationSvc.UpdateInvestigationStatus(ctx, id, statusStr); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.UpdateInvestigationStatusResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateInvestigationFindings handles PUT /api/v1/investigations/{id}/findings
func (h *InvestigationHandler) UpdateInvestigationFindings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.UpdateInvestigationFindingsRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	// Convert proto Struct to map[string]interface{}
	var details map[string]interface{}
	if pbReq.Details != nil {
		details = pbReq.Details.AsMap()
	}

	if err := h.investigationSvc.UpdateInvestigationFindings(ctx, id, pbReq.Findings, details); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.UpdateInvestigationFindingsResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateInvestigationProgress handles PUT /api/v1/investigations/{id}/progress
func (h *InvestigationHandler) UpdateInvestigationProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.UpdateInvestigationProgressRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	if err := h.investigationSvc.UpdateInvestigationProgress(ctx, id, int(pbReq.Progress)); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.UpdateInvestigationProgressResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// AddInvestigationStep handles POST /api/v1/investigations/{id}/step
func (h *InvestigationHandler) AddInvestigationStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.AddInvestigationStepRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	step := protoStepToModel(pbReq.Step)
	if err := h.investigationSvc.AddInvestigationStep(ctx, id, step); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.AddInvestigationStepResponse{Status: "step_added"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// GetCooldownStatus handles GET /api/v1/investigations/cooldown
func (h *InvestigationHandler) GetCooldownStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.GetCooldownStatusResponse{
		Cooldown: convert.CooldownStatusToProto(status),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// ResetCooldown handles POST /api/v1/investigations/cooldown/reset
func (h *InvestigationHandler) ResetCooldown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.investigationSvc.ResetCooldown(ctx); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": "cooldown_reset"}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// UpdateCooldownPeriod handles PUT /api/v1/investigations/cooldown/period
func (h *InvestigationHandler) UpdateCooldownPeriod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		CooldownPeriodSeconds int `json:"cooldown_period_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	if err := h.investigationSvc.UpdateCooldownPeriod(ctx, req.CooldownPeriodSeconds); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": "updated"}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// GetTriggers handles GET /api/v1/investigations/triggers
func (h *InvestigationHandler) GetTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	triggers, err := h.investigationSvc.GetTriggers(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.GetTriggersResponse{
		Triggers: convert.TriggerConfigsMapToProto(triggers),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateTrigger handles PUT /api/v1/investigations/triggers/{id}
func (h *InvestigationHandler) UpdateTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.UpdateTriggerRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	var enabled, autoFix *bool
	var threshold *float64
	if pbReq.Enabled != nil {
		e := *pbReq.Enabled
		enabled = &e
	}
	if pbReq.AutoFix != nil {
		a := *pbReq.AutoFix
		autoFix = &a
	}
	if pbReq.Threshold != nil {
		t := *pbReq.Threshold
		threshold = &t
	}

	if err := h.investigationSvc.UpdateTrigger(ctx, id, enabled, autoFix, threshold); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.UpdateTriggerResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateTriggerThreshold handles PUT /api/v1/investigations/triggers/{id}/threshold
func (h *InvestigationHandler) UpdateTriggerThreshold(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var req struct {
		Threshold float64 `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	if err := h.investigationSvc.UpdateTrigger(ctx, id, nil, nil, &req.Threshold); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": "updated"}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

func (h *InvestigationHandler) resolveAPIBaseURL(r *http.Request) string {
	// Prefer forwarded headers for correct addressing behind proxies
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		scheme := r.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	if r.Host != "" {
		return fmt.Sprintf("http://%s", r.Host)
	}
	port := strings.TrimSpace(h.config.Server.APIPort)
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// =============================================================================
// Agent Configuration Endpoints
// =============================================================================

// GetAgentConfig handles GET /api/agent/config
func (h *InvestigationHandler) GetAgentConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := h.investigationSvc.GetAgentConfig(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, config); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// GetAvailableRunners handles GET /api/agent/runners
func (h *InvestigationHandler) GetAvailableRunners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runners, err := h.investigationSvc.GetAvailableRunners(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, runners); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// UpdateAgentConfig handles PUT /api/agent/config
func (h *InvestigationHandler) UpdateAgentConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		RunnerType      string   `json:"runner_type,omitempty"`
		Model           string   `json:"model,omitempty"`
		MaxTurns        int32    `json:"max_turns,omitempty"`
		TimeoutSeconds  int32    `json:"timeout_seconds,omitempty"`
		AllowedTools    []string `json:"allowed_tools,omitempty"`
		SkipPermissions bool     `json:"skip_permissions,omitempty"`
		// SandboxMode replaces the (requires_sandbox, requires_approval)
		// pair removed in agent-manager Phase 1. Accepted: "off",
		// "tracking", "protected", or empty for the agent-manager default.
		SandboxMode string `json:"sandbox_mode,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	config, err := h.investigationSvc.UpdateAgentConfig(ctx, req.RunnerType, req.Model, req.MaxTurns, req.TimeoutSeconds, req.AllowedTools, req.SkipPermissions, req.SandboxMode)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, config); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// GetAgentStatus handles GET /api/agent/status
func (h *InvestigationHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetAgentStatus(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, status); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// =============================================================================
// Agent Current & Scripts Endpoints
// =============================================================================

// GetCurrentAgent handles GET /api/v1/investigations/agent/current
// Returns current running investigation agent status (if any)
func (h *InvestigationHandler) GetCurrentAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the latest in-progress investigation
	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	// Only return if there's an active investigation
	if investigation != nil && !models.IsTerminalStatus(investigation.Status) {
		if err := httputil.ProtoJSON(w, convert.InvestigationToProto(investigation)); err != nil {
			h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
		}
		return
	}

	// No active agent
	if err := httputil.JSON(w, nil); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// GetAgentStatusByID handles GET /api/v1/investigations/agent/{id}/status
// Returns the latest investigation payload for a specific agent.
func (h *InvestigationHandler) GetAgentStatusByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetInvestigationAgentStatus(ctx, id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// StopAgent handles POST /api/v1/investigations/agent/{id}/stop
func (h *InvestigationHandler) StopAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	if err := h.investigationSvc.StopInvestigationAgent(ctx, id); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": models.StatusStopped, "id": id}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// ListScripts handles GET /api/v1/investigations/scripts
func (h *InvestigationHandler) ListScripts(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.scriptSvc.ListScripts()
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.ListScriptsResponse{
		Scripts: convert.ScriptMetasToProto(scripts),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// GetScript handles GET /api/v1/investigations/scripts/{id}
func (h *InvestigationHandler) GetScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	meta, content, err := h.scriptSvc.GetScript(id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.GetScriptResponse{
		Script:  convert.ScriptMetaToProto(meta),
		Content: content,
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// ExecuteScript handles POST /api/v1/investigations/scripts/{id}/execute
func (h *InvestigationHandler) ExecuteScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	var pbReq apipb.ExecuteScriptRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		// Allow empty body — execute script as-is
		pbReq.Content = nil
	}

	var contentOverride string
	if pbReq.Content != nil {
		contentOverride = *pbReq.Content
	}

	execution, err := h.scriptSvc.ExecuteScript(ctx, id, contentOverride)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &apipb.ExecuteScriptResponse{
		Execution: convert.ScriptExecutionToProto(execution),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// protoStepToModel converts a proto InvestigationStep to the internal model.
func protoStepToModel(step *domainpb.InvestigationStep) models.InvestigationStep {
	if step == nil {
		return models.InvestigationStep{}
	}
	m := models.InvestigationStep{
		Name:     step.Name,
		Status:   strings.ToLower(strings.TrimPrefix(step.Status.String(), "INVESTIGATION_STEP_STATUS_")),
		Findings: step.Findings,
	}
	if step.StartTime != nil {
		m.StartTime = step.StartTime.AsTime()
	}
	if step.EndTime != nil {
		t := step.EndTime.AsTime()
		m.EndTime = &t
	}
	return m
}
