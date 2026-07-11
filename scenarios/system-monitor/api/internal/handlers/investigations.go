package handlers

// DOC: docs/reference/api-endpoints.md#investigations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
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

// ListScripts handles the typed Connect-RPC script listing contract.
func (h *InvestigationHandler) ListScripts(context.Context, *connect.Request[scriptspb.ListScriptsRequest]) (*connect.Response[scriptspb.ListScriptsResponse], error) {
	scripts, err := h.scriptSvc.ListScripts()
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&scriptspb.ListScriptsResponse{
		Scripts: convert.ScriptMetasToProto(scripts),
	}), nil
}

// GetScript handles the typed Connect-RPC script retrieval contract.
func (h *InvestigationHandler) GetScript(_ context.Context, req *connect.Request[scriptspb.GetScriptRequest]) (*connect.Response[scriptspb.GetScriptResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connectError(apierrors.Validation("id", "Script ID is required"))
	}

	meta, content, err := h.scriptSvc.GetScript(id)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&scriptspb.GetScriptResponse{
		Script:  convert.ScriptMetaToProto(meta),
		Content: content,
	}), nil
}

// ExecuteScript handles the typed Connect-RPC script execution contract.
func (h *InvestigationHandler) ExecuteScript(ctx context.Context, req *connect.Request[scriptspb.ExecuteScriptRequest]) (*connect.Response[scriptspb.ExecuteScriptResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connectError(apierrors.Validation("id", "Script ID is required"))
	}

	contentOverride := ""
	if req.Msg.Content != nil {
		contentOverride = req.Msg.GetContent()
	}

	execution, err := h.scriptSvc.ExecuteScript(ctx, id, contentOverride)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&scriptspb.ExecuteScriptResponse{
		Execution: convert.ScriptExecutionToProto(execution),
	}), nil
}

// TriggerInvestigation starts a new anomaly investigation via Connect-RPC.
func (h *InvestigationHandler) TriggerInvestigation(ctx context.Context, req *connect.Request[investigationspb.TriggerInvestigationRequest]) (*connect.Response[investigationspb.TriggerInvestigationResponse], error) {
	investigation, err := h.investigationSvc.TriggerInvestigation(ctx, req.Msg.GetAutoFix(), req.Msg.GetNote())
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.TriggerInvestigationResponse{
		Status:          models.StatusQueued,
		InvestigationId: investigation.ID,
		ApiBaseUrl:      h.configuredAPIBaseURL(),
		Message:         "Investigation queued for processing",
		AutoFix:         req.Msg.GetAutoFix(),
		Note:            req.Msg.GetNote(),
	}), nil
}

// GetInvestigation retrieves an investigation by ID via Connect-RPC.
func (h *InvestigationHandler) GetInvestigation(ctx context.Context, req *connect.Request[investigationspb.GetInvestigationRequest]) (*connect.Response[investigationspb.GetInvestigationResponse], error) {
	investigation, err := h.investigationSvc.GetInvestigation(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.GetInvestigationResponse{
		Investigation: convert.InvestigationToProto(investigation),
	}), nil
}

// GetLatestInvestigation retrieves the most recent investigation via Connect-RPC.
func (h *InvestigationHandler) GetLatestInvestigation(ctx context.Context, _ *connect.Request[investigationspb.GetLatestInvestigationRequest]) (*connect.Response[investigationspb.GetLatestInvestigationResponse], error) {
	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.GetLatestInvestigationResponse{
		Investigation: convert.InvestigationToProto(investigation),
	}), nil
}

// ListInvestigations returns recent investigations via Connect-RPC.
func (h *InvestigationHandler) ListInvestigations(ctx context.Context, req *connect.Request[investigationspb.ListInvestigationsRequest]) (*connect.Response[investigationspb.ListInvestigationsResponse], error) {
	limit := 20
	if req.Msg.Limit != nil && req.Msg.GetLimit() > 0 {
		limit = int(req.Msg.GetLimit())
	}

	investigations, err := h.investigationSvc.ListInvestigations(ctx, limit)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.ListInvestigationsResponse{
		Investigations: convert.InvestigationsToProto(investigations),
	}), nil
}

// UpdateInvestigationStatus updates an investigation's status via Connect-RPC.
func (h *InvestigationHandler) UpdateInvestigationStatus(ctx context.Context, req *connect.Request[investigationspb.UpdateInvestigationStatusRequest]) (*connect.Response[investigationspb.UpdateInvestigationStatusResponse], error) {
	statusStr := strings.ToLower(strings.TrimPrefix(req.Msg.Status.String(), "INVESTIGATION_STATUS_"))
	if err := h.investigationSvc.UpdateInvestigationStatus(ctx, req.Msg.GetId(), statusStr); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.UpdateInvestigationStatusResponse{Status: "updated"}), nil
}

// UpdateInvestigationFindings updates an investigation's findings via Connect-RPC.
func (h *InvestigationHandler) UpdateInvestigationFindings(ctx context.Context, req *connect.Request[investigationspb.UpdateInvestigationFindingsRequest]) (*connect.Response[investigationspb.UpdateInvestigationFindingsResponse], error) {
	var details map[string]interface{}
	if req.Msg.Details != nil {
		details = req.Msg.Details.AsMap()
	}

	if err := h.investigationSvc.UpdateInvestigationFindings(ctx, req.Msg.GetId(), req.Msg.GetFindings(), details); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.UpdateInvestigationFindingsResponse{Status: "updated"}), nil
}

// UpdateInvestigationProgress updates investigation progress via Connect-RPC.
func (h *InvestigationHandler) UpdateInvestigationProgress(ctx context.Context, req *connect.Request[investigationspb.UpdateInvestigationProgressRequest]) (*connect.Response[investigationspb.UpdateInvestigationProgressResponse], error) {
	if err := h.investigationSvc.UpdateInvestigationProgress(ctx, req.Msg.GetId(), int(req.Msg.GetProgress())); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.UpdateInvestigationProgressResponse{Status: "updated"}), nil
}

// AddInvestigationStep adds a step to an investigation via Connect-RPC.
func (h *InvestigationHandler) AddInvestigationStep(ctx context.Context, req *connect.Request[investigationspb.AddInvestigationStepRequest]) (*connect.Response[investigationspb.AddInvestigationStepResponse], error) {
	step := protoStepToModel(req.Msg.GetStep())
	if err := h.investigationSvc.AddInvestigationStep(ctx, req.Msg.GetId(), step); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.AddInvestigationStepResponse{Status: "step_added"}), nil
}

// GetCooldownStatus returns the current investigation cooldown via Connect-RPC.
func (h *InvestigationHandler) GetCooldownStatus(ctx context.Context, _ *connect.Request[investigationspb.GetCooldownStatusRequest]) (*connect.Response[investigationspb.GetCooldownStatusResponse], error) {
	status, err := h.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.GetCooldownStatusResponse{
		Cooldown: convert.CooldownStatusToProto(status),
	}), nil
}

// ResetCooldown resets the investigation cooldown via Connect-RPC.
func (h *InvestigationHandler) ResetCooldown(ctx context.Context, _ *connect.Request[investigationspb.ResetCooldownRequest]) (*connect.Response[investigationspb.ResetCooldownResponse], error) {
	if err := h.investigationSvc.ResetCooldown(ctx); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.ResetCooldownResponse{Status: "cooldown_reset"}), nil
}

// UpdateCooldownPeriod updates the investigation cooldown period via Connect-RPC.
func (h *InvestigationHandler) UpdateCooldownPeriod(ctx context.Context, req *connect.Request[investigationspb.UpdateCooldownPeriodRequest]) (*connect.Response[investigationspb.UpdateCooldownPeriodResponse], error) {
	if err := h.investigationSvc.UpdateCooldownPeriod(ctx, int(req.Msg.GetCooldownPeriodSeconds())); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.UpdateCooldownPeriodResponse{Status: "updated"}), nil
}

// GetTriggers returns trigger configurations via Connect-RPC.
func (h *InvestigationHandler) GetTriggers(ctx context.Context, _ *connect.Request[investigationspb.GetTriggersRequest]) (*connect.Response[investigationspb.GetTriggersResponse], error) {
	triggers, err := h.investigationSvc.GetTriggers(ctx)
	if err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.GetTriggersResponse{
		Triggers: convert.TriggerConfigsMapToProto(triggers),
	}), nil
}

// UpdateTrigger updates a trigger configuration via Connect-RPC.
func (h *InvestigationHandler) UpdateTrigger(ctx context.Context, req *connect.Request[investigationspb.UpdateTriggerRequest]) (*connect.Response[investigationspb.UpdateTriggerResponse], error) {
	var enabled, autoFix *bool
	var threshold *float64
	if req.Msg.Enabled != nil {
		e := req.Msg.GetEnabled()
		enabled = &e
	}
	if req.Msg.AutoFix != nil {
		a := req.Msg.GetAutoFix()
		autoFix = &a
	}
	if req.Msg.Threshold != nil {
		t := req.Msg.GetThreshold()
		threshold = &t
	}

	if err := h.investigationSvc.UpdateTrigger(ctx, req.Msg.GetId(), enabled, autoFix, threshold); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.UpdateTriggerResponse{Status: "updated"}), nil
}

// StopAgent stops a running investigation agent via Connect-RPC.
func (h *InvestigationHandler) StopAgent(ctx context.Context, req *connect.Request[investigationspb.StopAgentRequest]) (*connect.Response[investigationspb.StopAgentResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connectError(apierrors.Validation("id", "Agent ID is required"))
	}

	if err := h.investigationSvc.StopInvestigationAgent(ctx, id); err != nil {
		return nil, connectError(err)
	}

	return connect.NewResponse(&investigationspb.StopAgentResponse{Status: models.StatusStopped, Id: id}), nil
}

// HandleListInvestigations handles GET /api/v1/investigations
func (h *InvestigationHandler) HandleListInvestigations(w http.ResponseWriter, r *http.Request) {
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

	resp := &investigationspb.ListInvestigationsResponse{
		Investigations: convert.InvestigationsToProto(investigations),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleGetLatestInvestigation handles GET /api/v1/investigations/latest
func (h *InvestigationHandler) HandleGetLatestInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetLatestInvestigation(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// HandleTriggerInvestigation handles POST /api/v1/investigations/trigger
func (h *InvestigationHandler) HandleTriggerInvestigation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var pbReq investigationspb.TriggerInvestigationRequest
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
	resp := &investigationspb.TriggerInvestigationResponse{
		Status:          models.StatusQueued,
		InvestigationId: investigation.ID,
		ApiBaseUrl:      h.resolveAPIBaseURL(r),
		Message:         "Investigation queued for processing",
		AutoFix:         pbReq.AutoFix,
		Note:            pbReq.Note,
	}
	httputil.SafeProtoJSONWithStatus(w, h.log, r, http.StatusAccepted, resp)
}

// HandleGetInvestigation handles GET /api/v1/investigations/{id}
func (h *InvestigationHandler) HandleGetInvestigation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetInvestigation(ctx, id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// HandleUpdateInvestigationStatus handles PUT /api/v1/investigations/{id}/status
func (h *InvestigationHandler) HandleUpdateInvestigationStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq investigationspb.UpdateInvestigationStatusRequest
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

	resp := &investigationspb.UpdateInvestigationStatusResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleUpdateInvestigationFindings handles PUT /api/v1/investigations/{id}/findings
func (h *InvestigationHandler) HandleUpdateInvestigationFindings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq investigationspb.UpdateInvestigationFindingsRequest
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

	resp := &investigationspb.UpdateInvestigationFindingsResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleUpdateInvestigationProgress handles PUT /api/v1/investigations/{id}/progress
func (h *InvestigationHandler) HandleUpdateInvestigationProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq investigationspb.UpdateInvestigationProgressRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	if err := h.investigationSvc.UpdateInvestigationProgress(ctx, id, int(pbReq.Progress)); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &investigationspb.UpdateInvestigationProgressResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleAddInvestigationStep handles POST /api/v1/investigations/{id}/step
func (h *InvestigationHandler) HandleAddInvestigationStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq investigationspb.AddInvestigationStepRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", err.Error()))
		return
	}

	step := protoStepToModel(pbReq.Step)
	if err := h.investigationSvc.AddInvestigationStep(ctx, id, step); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &investigationspb.AddInvestigationStepResponse{Status: "step_added"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleGetCooldownStatus handles GET /api/v1/investigations/cooldown
func (h *InvestigationHandler) HandleGetCooldownStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.investigationSvc.GetCooldownStatus(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &investigationspb.GetCooldownStatusResponse{
		Cooldown: convert.CooldownStatusToProto(status),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleResetCooldown handles POST /api/v1/investigations/cooldown/reset
func (h *InvestigationHandler) HandleResetCooldown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.investigationSvc.ResetCooldown(ctx); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": "cooldown_reset"}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// HandleUpdateCooldownPeriod handles PUT /api/v1/investigations/cooldown/period
func (h *InvestigationHandler) HandleUpdateCooldownPeriod(w http.ResponseWriter, r *http.Request) {
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

// HandleGetTriggers handles GET /api/v1/investigations/triggers
func (h *InvestigationHandler) HandleGetTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	triggers, err := h.investigationSvc.GetTriggers(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &investigationspb.GetTriggersResponse{
		Triggers: convert.TriggerConfigsMapToProto(triggers),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleUpdateTrigger handles PUT /api/v1/investigations/triggers/{id}
func (h *InvestigationHandler) HandleUpdateTrigger(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq investigationspb.UpdateTriggerRequest
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

	resp := &investigationspb.UpdateTriggerResponse{Status: "updated"}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateTriggerThreshold handles PUT /api/v1/investigations/triggers/{id}/threshold
func (h *InvestigationHandler) UpdateTriggerThreshold(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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

func (h *InvestigationHandler) configuredAPIBaseURL() string {
	port := "8080"
	if h.config != nil && strings.TrimSpace(h.config.Server.APIPort) != "" {
		port = strings.TrimSpace(h.config.Server.APIPort)
	}
	return fmt.Sprintf("http://localhost:%s", port)
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
	id := r.PathValue("id")
	ctx := r.Context()

	investigation, err := h.investigationSvc.GetInvestigationAgentStatus(ctx, id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InvestigationToProto(investigation))
}

// HandleStopAgent handles POST /api/v1/investigations/agent/{id}/stop.
func (h *InvestigationHandler) HandleStopAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	if err := h.investigationSvc.StopInvestigationAgent(ctx, id); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	if err := httputil.JSON(w, map[string]string{"status": models.StatusStopped, "id": id}); err != nil {
		h.log.Warn("response write failed", "error", err, "path", r.URL.Path)
	}
}

// HandleListScripts handles GET /api/v1/investigations/scripts.
func (h *InvestigationHandler) HandleListScripts(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.scriptSvc.ListScripts()
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &scriptspb.ListScriptsResponse{
		Scripts: convert.ScriptMetasToProto(scripts),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleGetScript handles GET /api/v1/investigations/scripts/{id}.
func (h *InvestigationHandler) HandleGetScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	meta, content, err := h.scriptSvc.GetScript(id)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	resp := &scriptspb.GetScriptResponse{
		Script:  convert.ScriptMetaToProto(meta),
		Content: content,
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// HandleExecuteScript handles POST /api/v1/investigations/scripts/{id}/execute.
func (h *InvestigationHandler) HandleExecuteScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	var pbReq scriptspb.ExecuteScriptRequest
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

	resp := &scriptspb.ExecuteScriptResponse{
		Execution: convert.ScriptExecutionToProto(execution),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// protoStepToModel converts a proto InvestigationStep to the internal model.
func protoStepToModel(step *investigationspb.InvestigationStep) models.InvestigationStep {
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
