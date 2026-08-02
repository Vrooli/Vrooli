package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// RUN HANDLERS
// =============================================================================

// CreateRun creates a new run.
func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "create-run") {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var protoReq apipb.CreateRunRequest
	if err := protoconv.UnmarshalJSON(body, &protoReq); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body: "+err.Error())
		return
	}
	if !h.validateProto(w, r, &protoReq) {
		return
	}
	if protoReq.TaskId == "" {
		writeSimpleError(w, r, "task_id", "task_id is required")
		return
	}

	taskID, err := uuid.Parse(protoReq.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	req := orchestration.CreateRunRequest{
		TaskID: taskID,
		Force:  protoReq.Force,
	}
	if protoReq.AgentProfileId != nil {
		agentProfileID, err := uuid.Parse(protoReq.GetAgentProfileId())
		if err != nil {
			writeSimpleError(w, r, "agent_profile_id", "invalid UUID format for agent profile ID")
			return
		}
		req.AgentProfileID = &agentProfileID
	}
	if protoReq.Tag != nil {
		req.Tag = protoReq.GetTag()
	}
	if protoReq.RunMode != nil {
		mode := protoconv.RunModeFromProto(*protoReq.RunMode)
		req.RunMode = &mode
	}
	if protoReq.ExecutionMode != nil {
		req.ExecutionMode = protoconv.ExecutionModeFromProto(protoReq.GetExecutionMode())
	}
	if protoReq.IdempotencyKey != nil {
		req.IdempotencyKey = protoReq.GetIdempotencyKey()
	}
	if protoReq.Prompt != nil {
		req.Prompt = protoReq.GetPrompt()
	}
	if protoReq.ExistingSandboxId != nil {
		existingSandboxID, err := uuid.Parse(protoReq.GetExistingSandboxId())
		if err != nil {
			writeSimpleError(w, r, "existing_sandbox_id", "invalid UUID format for existing sandbox ID")
			return
		}
		req.ExistingSandboxID = &existingSandboxID
	}
	if protoReq.ProfileRef != nil {
		req.ProfileRef = &orchestration.ProfileRef{
			ProfileKey:     protoReq.ProfileRef.ProfileKey,
			Defaults:       protoconv.AgentProfileFromProto(protoReq.ProfileRef.Defaults),
			UpdateExisting: protoReq.ProfileRef.UpdateExisting,
		}
	}
	if protoReq.InlineConfig != nil {
		inline := protoReq.InlineConfig
		if inline.ResultSpec != nil {
			req.ResultSpec = protoconv.ResultSpecFromProto(inline.ResultSpec)
		}
		if inline.RoleRef != nil {
			roleRef := inline.GetRoleRef()
			req.RoleRef = &roleRef
		}
		if inline.MaxTurns != nil {
			maxTurns := int(inline.GetMaxTurns())
			req.MaxTurns = &maxTurns
		}
		if inline.Timeout != nil {
			timeout := inline.Timeout.AsDuration()
			req.Timeout = &timeout
		}
		if inline.Effort != nil {
			effort := domain.Effort(inline.GetEffort())
			req.Effort = &effort
		}
		if inline.Model != nil {
			model := inline.GetModel()
			req.Model = &model
		}
		if len(inline.AllowedTools) > 0 || inline.ClearAllowedTools {
			req.AllowedTools = inline.AllowedTools
		}
		if len(inline.DeniedTools) > 0 || inline.ClearDeniedTools {
			req.DeniedTools = inline.DeniedTools
		}
		if inline.SkipPermissionPrompt != nil {
			skipPermissionPrompt := inline.GetSkipPermissionPrompt()
			req.SkipPermissionPrompt = &skipPermissionPrompt
		}
		if inline.Features != nil {
			f := protoconv.FeatureFlagsFromProto(inline.Features)
			req.EnableBrowser = &f.EnableBrowser
		}
		if len(inline.ExtraFlags) > 0 || inline.ClearExtraFlags {
			req.ExtraFlags = protoconv.RunnerExtraFlagsFromProto(inline.ExtraFlags)
		}
		if inline.NetworkAccess != nil {
			na := protoconv.NetworkAccessFromProto(inline.GetNetworkAccess())
			req.NetworkAccess = &na
		}
		if inline.SandboxConfig != nil {
			req.SandboxConfig = protoconv.SandboxConfigFromProto(inline.SandboxConfig)
		}
		if len(inline.AllowedPaths) > 0 || inline.ClearAllowedPaths {
			req.AllowedPaths = inline.AllowedPaths
		}
		if len(inline.DeniedPaths) > 0 || inline.ClearDeniedPaths {
			req.DeniedPaths = inline.DeniedPaths
		}
	}

	if len(protoReq.Environment) > 0 {
		if err := validateCustomEnvironment(protoReq.Environment); err != nil {
			writeSimpleError(w, r, "environment", err.Error())
			return
		}
		req.Environment = protoReq.Environment
	}
	if protoReq.ConversationId != nil {
		req.ConversationID = protoReq.GetConversationId()
	}
	if protoReq.ParentRunId != nil {
		parentRunID, err := uuid.Parse(protoReq.GetParentRunId())
		if err != nil {
			writeSimpleError(w, r, "parent_run_id", "invalid UUID format for parent run ID")
			return
		}
		req.ParentRunID = &parentRunID
	}

	run, err := h.svc.CreateRun(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, h.newCreateRunResponse(run))
}

// validateCustomEnvironment validates that custom environment variables use
// the VROOLI_ prefix and don't exceed size limits.
func validateCustomEnvironment(env map[string]string) error {
	if len(env) > 20 {
		return fmt.Errorf("max 20 entries allowed, got %d", len(env))
	}
	totalSize := 0
	for k, v := range env {
		if !strings.HasPrefix(k, "VROOLI_") {
			return fmt.Errorf("key %q must start with VROOLI_ prefix", k)
		}
		totalSize += len(k) + len(v)
	}
	if totalSize > 4096 {
		return fmt.Errorf("total size %d exceeds 4096 byte limit", totalSize)
	}
	return nil
}

// CreateInvestigationRun creates a new investigation run for specified run IDs.
func (h *Handler) CreateInvestigationRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "investigate") {
		return
	}
	var req struct {
		RunIDs        []string          `json:"runIds"`
		CustomContext string            `json:"customContext,omitempty"`
		Depth         string            `json:"depth,omitempty"` // quick, standard, or deep
		ProjectRoot   string            `json:"projectRoot,omitempty"`
		ScopePaths    []string          `json:"scopePaths,omitempty"`
		AttachmentIDs []string          `json:"attachmentIds,omitempty"`
		RoleRef       string            `json:"roleRef,omitempty"`
		Environment   map[string]string `json:"environment,omitempty"`
		GoalID        string            `json:"goalId,omitempty"`
		Selector      *struct {
			Filter invocationreadmodel.Filter `json:"filter"`
			Limit  int                        `json:"limit,omitempty"`
		} `json:"selector,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON")
		return
	}
	if err := validateCustomEnvironment(req.Environment); err != nil {
		writeSimpleError(w, r, "environment", err.Error())
		return
	}

	selectionCount := 0
	if len(req.RunIDs) > 0 {
		selectionCount++
	}
	if req.Selector != nil {
		selectionCount++
	}
	if strings.TrimSpace(req.GoalID) != "" {
		selectionCount++
	}
	if selectionCount > 1 {
		writeSimpleError(w, r, "selection", "provide exactly one of runIds, selector, or goalId")
		return
	}
	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeSimpleError(w, r, "runIds", "invalid UUID format: "+idStr)
			return
		}
		runIDs = append(runIDs, id)
	}
	var selection *orchestration.InvestigationSelection
	if req.Selector != nil || strings.TrimSpace(req.GoalID) != "" {
		filter := invocationreadmodel.Filter{}
		kind := "goal"
		if req.Selector != nil {
			filter = req.Selector.Filter
			kind = "cohort"
		}
		if goalID := strings.TrimSpace(req.GoalID); goalID != "" {
			filter.GoalID = goalID
		}
		limit := 50
		if req.Selector != nil {
			limit = req.Selector.Limit
		}
		if limit <= 0 {
			limit = 50
		}
		if limit > 50 {
			writeSimpleError(w, r, "selector.limit", "investigations are capped at 50 runs")
			return
		}
		cohort, err := h.svc.SelectInvocationCohort(r.Context(), filter, limit)
		if err != nil {
			writeError(w, r, err)
			return
		}
		selection = &orchestration.InvestigationSelection{Kind: kind, Filter: filter, MatchedRuns: cohort.MatchedRuns, DroppedRuns: cohort.DroppedRuns}
		for _, idStr := range cohort.RunIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, r, fmt.Errorf("selector returned invalid run ID: %w", err))
				return
			}
			runIDs = append(runIDs, id)
		}
	}

	// Validate depth if provided
	depth := domain.InvestigationDepth(req.Depth)
	if !depth.IsValid() {
		writeSimpleError(w, r, "depth", "must be 'quick', 'standard', or 'deep'")
		return
	}

	run, err := h.svc.CreateInvestigationRun(r.Context(), orchestration.CreateInvestigationRequest{
		RunIDs:        runIDs,
		CustomContext: req.CustomContext,
		Depth:         depth,
		ProjectRoot:   req.ProjectRoot,
		ScopePaths:    req.ScopePaths,
		AttachmentIDs: req.AttachmentIDs,
		RoleRef:       optionalTrimmedString(req.RoleRef),
		Environment:   req.Environment,
		Selection:     selection,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, h.newCreateRunResponse(run))
}

// optionalTrimmedString returns nil for an omitted override.
func optionalTrimmedString(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// ResumeFromFailedRun creates a new run that resumes the work of a failed
// or cancelled run, inheriting its task + profile and seeding the prior
// attempt's transcript and diff so the agent can complete the remaining work.
func (h *Handler) ResumeFromFailedRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "resume-from-failed") {
		return
	}
	var req struct {
		RunID         string   `json:"runId"`
		CustomContext string   `json:"customContext,omitempty"`
		AttachmentIDs []string `json:"attachmentIds,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON")
		return
	}

	runID, err := uuid.Parse(req.RunID)
	if err != nil {
		writeSimpleError(w, r, "runId", "invalid UUID format")
		return
	}

	run, err := h.svc.ResumeFromFailedRun(r.Context(), orchestration.ResumeFromFailedRunRequest{
		RunID:         runID,
		CustomContext: req.CustomContext,
		AttachmentIDs: req.AttachmentIDs,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, h.newCreateRunResponse(run))
}

// CreateInvestigationApplyRun creates a new run to apply investigation recommendations.
func (h *Handler) CreateInvestigationApplyRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "investigation-apply") {
		return
	}
	var req struct {
		InvestigationRunID string            `json:"investigationRunId"`
		Decision           string            `json:"decision,omitempty"`
		Selected           []string          `json:"selected,omitempty"`
		CustomContext      string            `json:"customContext,omitempty"`
		AttachmentIDs      []string          `json:"attachmentIds,omitempty"`
		RoleRef            string            `json:"roleRef,omitempty"`
		Environment        map[string]string `json:"environment,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON")
		return
	}
	if err := validateCustomEnvironment(req.Environment); err != nil {
		writeSimpleError(w, r, "environment", err.Error())
		return
	}

	runID, err := uuid.Parse(req.InvestigationRunID)
	if err != nil {
		writeSimpleError(w, r, "investigationRunId", "invalid UUID format")
		return
	}

	run, err := h.svc.CreateInvestigationApplyRun(r.Context(), orchestration.CreateInvestigationApplyRequest{
		InvestigationRunID: runID,
		Decision:           req.Decision,
		Selected:           req.Selected,
		CustomContext:      req.CustomContext,
		AttachmentIDs:      req.AttachmentIDs,
		RoleRef:            optionalTrimmedString(req.RoleRef),
		Environment:        req.Environment,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, h.newCreateRunResponse(run))
}

// GetRun retrieves a run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	pbRun := protoconv.RunToProto(run)
	h.attachObservedReceipts(r.Context(), id.String(), pbRun.Result)
	writeProtoJSON(w, http.StatusOK, &apipb.GetRunResponse{Run: pbRun})
}

// GetAuditTranscript returns a deliberately bounded excerpt for an audit
// sample. It never accepts a file path, emits a stable content hash, and caps
// output at 64KiB so audit evidence cannot become an unbounded transcript API.
func (h *Handler) GetAuditTranscript(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if run.TranscriptPath == "" {
		writeSimpleError(w, r, "transcript_unavailable", "run has no persisted transcript")
		return
	}
	limit := 16 * 1024
	if raw := r.URL.Query().Get("maxBytes"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 64*1024 {
			limit = parsed
		} else {
			writeSimpleError(w, r, "max_bytes", "maxBytes must be between 1 and 65536")
			return
		}
	}
	f, err := os.Open(run.TranscriptPath)
	if err != nil {
		writeSimpleError(w, r, "transcript_unavailable", "persisted transcript cannot be read")
		return
	}
	defer f.Close()
	buf := make([]byte, limit+1)
	n, readErr := io.ReadFull(f, buf)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		writeSimpleError(w, r, "transcript_read", "unable to read transcript excerpt")
		return
	}
	truncated := n > limit
	if truncated {
		n = limit
	}
	data := buf[:n]
	w.Header().Set("Content-Type", "application/json")
	digest := sha256.Sum256(data)
	_ = json.NewEncoder(w).Encode(map[string]any{"runId": id.String(), "maxBytes": limit, "truncated": truncated, "contentHash": fmt.Sprintf("sha256:%x", digest[:]), "content": string(data)})
}

// ListRuns returns all runs, with optional filtering.
// Query parameters:
//   - status: Filter by run status (e.g., "running", "pending", "complete")
//   - taskId: Filter by task ID
//   - profileId: Filter by agent profile ID
//   - tagPrefix: Filter by tag prefix (e.g., "ecosystem-" to get all swarm-manager runs)
//   - investigates_run_id: Filter investigation runs linked to a source run ID
//   - applies_investigation_run_id: Filter apply runs linked to an investigation run ID
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	req := apipb.ListRunsRequest{}
	var investigatesRunID *uuid.UUID
	var appliesInvestigationRunID *uuid.UUID

	// Parse status filter
	if statusStr := queryFirst(r, "status"); statusStr != "" {
		if status, ok := parseRunStatus(statusStr); ok {
			converted := protoconv.RunStatusToProto(status)
			req.Status = &converted
		} else {
			writeSimpleError(w, r, "status", "invalid run status")
			return
		}
	}

	// Parse task ID filter
	if taskIDStr := queryFirst(r, "task_id", "taskId"); taskIDStr != "" {
		if _, err := uuid.Parse(taskIDStr); err == nil {
			req.TaskId = &taskIDStr
		} else {
			writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
			return
		}
	}

	// Parse profile ID filter
	if profileIDStr := queryFirst(r, "agent_profile_id", "profileId", "agentProfileId"); profileIDStr != "" {
		if _, err := uuid.Parse(profileIDStr); err == nil {
			req.AgentProfileId = &profileIDStr
		} else {
			writeSimpleError(w, r, "agent_profile_id", "invalid UUID format for agent profile ID")
			return
		}
	}

	// Parse tag prefix filter
	if tagPrefix := queryFirst(r, "tag_prefix", "tagPrefix"); tagPrefix != "" {
		req.TagPrefix = &tagPrefix
	}
	if investigatesRunIDStr := queryFirst(r, "investigates_run_id", "investigatesRunId"); investigatesRunIDStr != "" {
		parsed, err := uuid.Parse(investigatesRunIDStr)
		if err != nil {
			writeSimpleError(w, r, "investigates_run_id", "invalid UUID format")
			return
		}
		investigatesRunID = &parsed
	}
	if appliesInvestigationRunIDStr := queryFirst(r, "applies_investigation_run_id", "appliesInvestigationRunId"); appliesInvestigationRunIDStr != "" {
		parsed, err := uuid.Parse(appliesInvestigationRunIDStr)
		if err != nil {
			writeSimpleError(w, r, "applies_investigation_run_id", "invalid UUID format")
			return
		}
		appliesInvestigationRunID = &parsed
	}

	if limit, limitProvided, err := parseQueryIntStrict(r, "limit"); err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	} else if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offset, offsetProvided, err := parseQueryIntStrict(r, "offset"); err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	} else if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}

	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.RunListOptions{}
	if req.Status != nil {
		status := protoconv.RunStatusFromProto(req.GetStatus())
		if status != "" {
			opts.Status = &status
		}
	}
	if req.TaskId != nil {
		taskID, _ := uuid.Parse(req.GetTaskId())
		opts.TaskID = &taskID
	}
	if req.AgentProfileId != nil {
		profileID, _ := uuid.Parse(req.GetAgentProfileId())
		opts.AgentProfileID = &profileID
	}
	if req.TagPrefix != nil {
		opts.TagPrefix = req.GetTagPrefix()
	}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}
	opts.InvestigatesRunID = investigatesRunID
	opts.AppliesInvestigationRunID = appliesInvestigationRunID

	runs, err := h.svc.ListRuns(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListRunsResponse{
		Runs:  protoconv.RunsToProto(runs),
		Total: int32(len(runs)),
	})
}

// DeleteRun permanently removes a run.
func (h *Handler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "delete-run") {
		return
	}
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.DeleteRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteRunResponse{Success: true})
}

// StopRun stops a running run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "stop-run") {
		return
	}
	idStr := mux.Vars(r)["id"]
	req := apipb.StopRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.StopRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopRunResponse{
		Status: "stopped",
		Run:    protoconv.RunToProto(run),
	})
}

// ContinueRun continues an existing run's conversation with a follow-up message.
// POST /api/v1/runs/{id}/continue
// Body: {"message": "Please also update the tests"}
func (h *Handler) ContinueRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "continue-run") {
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req domainpb.ContinueRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Override run_id from path
	req.RunId = idStr
	if !h.validateProto(w, r, &req) {
		return
	}

	run, err := h.svc.ContinueRun(r.Context(), orchestration.ContinueRunRequest{
		RunID:          id,
		Message:        req.Message,
		AttachmentIDs:  req.AttachmentIds,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &domainpb.ContinueRunResponse{
		Success: true,
		Run:     protoconv.RunToProto(run),
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ParkRun suspends a running run on externally-owned async work (durable
// park/resume). Called from inside an agent-manager-controlled run; the caller
// authenticates with its VROOLI_AGENT_IDENTITY_TOKEN and may only park itself.
// On success the run transitions running→parked, agent-manager begins waiting on
// the producer, and the agent process group is terminated after a short grace so
// the suspended run burns zero tokens. The response carries the clean
// tool-result text the in-run command prints before its turn ends.
// POST /api/v1/runs/{id}/park
func (h *Handler) ParkRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req domainpb.ParkRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	req.RunId = idStr
	if !h.validateProto(w, r, &req) {
		return
	}

	parkReq := orchestration.ParkRunFromAgentRequest{
		RunID:         id,
		Producer:      req.Producer,
		Key:           req.Key,
		IdentityToken: req.IdentityToken,
	}
	if req.DeadlineUnix > 0 {
		d := time.Unix(req.DeadlineUnix, 0)
		parkReq.Deadline = &d
	}

	result, err := h.svc.ParkRunFromAgent(r.Context(), parkReq)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &domainpb.ParkRunResponse{
		Success: !result.Refused,
		Message: result.Message,
		Refused: result.Refused,
		Result:  result.Result,
	}
	if result.Run != nil {
		resp.Run = protoconv.RunToProto(result.Run)
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// GetAwaitResult returns a run's most recently resolved await result — the
// non-blocking re-fetch path that lets a woken agent re-read what it parked on
// without re-running the blocking producer. Pure read.
// GET /api/v1/runs/{id}/await-result
func (h *Handler) GetAwaitResult(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	res, err := h.svc.GetAwaitResult(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &domainpb.GetAwaitResultResponse{
		Found:  res.Found,
		Key:    res.Key,
		Result: res.Result,
	}
	if res.ResolvedAt != nil {
		resp.ResolvedAt = res.ResolvedAt.Format(time.RFC3339)
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// WakeRun resumes a parked run with the awaited result injected as the next
// turn. Wake is normally orchestrator-internal (the waiter goroutine drives it);
// this endpoint is for manual/ops recovery. It is idempotent: a run that is no
// longer parked is returned unchanged with success=false (not an error).
// POST /api/v1/runs/{id}/wake
func (h *Handler) WakeRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "wake-run") {
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req domainpb.WakeRunRequest
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &req); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body")
			return
		}
	}
	req.RunId = idStr
	if !h.validateProto(w, r, &req) {
		return
	}

	// Capture the pre-wake status so we can report whether this call actually
	// woke the run (parked→running) or was an idempotent no-op.
	before, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	wasParked := before != nil && before.Status == domain.RunStatusParked

	run, err := h.svc.WakeRun(r.Context(), orchestration.WakeRunInput{
		RunID:    id,
		Result:   req.Result,
		TimedOut: req.TimedOut,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &domainpb.WakeRunResponse{
		Success: wasParked,
	}
	if run != nil {
		resp.Run = protoconv.RunToProto(run)
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *Handler) RecoverRun(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "recover-run") {
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	result, err := h.svc.RecoverRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &apipb.RecoverRunResponse{}
	if result != nil {
		resp.Recovered = result.Recovered
		resp.Idempotent = result.Idempotent
		resp.Message = result.Message
		if result.Run != nil {
			resp.Run = protoconv.RunToProto(result.Run)
		}
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// DeleteRunMessage marks a message event as deleted.
// POST /api/v1/runs/{id}/messages/{event_id}/delete
func (h *Handler) DeleteRunMessage(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "delete-run-message") {
		return
	}
	runIDStr := mux.Vars(r)["id"]
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	eventIDStr := mux.Vars(r)["event_id"]
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		writeSimpleError(w, r, "event_id", "invalid UUID format for event ID")
		return
	}

	req := domainpb.DeleteRunMessageRequest{
		RunId:   runIDStr,
		EventId: eventIDStr,
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	if _, err := h.svc.DeleteRunMessage(r.Context(), runID, eventID); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &domainpb.DeleteRunMessageResponse{
		Success: true,
	})
}
