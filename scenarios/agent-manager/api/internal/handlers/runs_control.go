package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/rolepolicy"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func (h *Handler) GetRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	req := apipb.GetRunByTagRequest{Tag: tag}
	if !h.validateProto(w, r, &req) {
		return
	}

	run, err := h.svc.GetRunByTag(r.Context(), tag)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunByTagResponse{
		Run: protoconv.RunToProto(run),
	})
}

// StopRunByTag stops a run identified by its custom tag.
func (h *Handler) StopRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	req := apipb.StopRunByTagRequest{Tag: tag}
	if !h.validateProto(w, r, &req) {
		return
	}

	if err := h.svc.StopRunByTag(r.Context(), tag); err != nil {
		writeError(w, r, err)
		return
	}

	run, err := h.svc.GetRunByTag(r.Context(), tag)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopRunByTagResponse{
		Status: "stopped",
		Tag:    tag,
		Run:    protoconv.RunToProto(run),
	})
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
// POST /api/v1/runs/stop-all
// Body: {"tagPrefix": "ecosystem-", "force": true}
func (h *Handler) StopAllRuns(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "stop-all") {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.StopAllRunsRequest
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &req); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body")
			return
		}
		if !h.validateProto(w, r, &req) {
			return
		}
	}
	tagPrefix := ""
	if req.TagPrefix != nil {
		tagPrefix = *req.TagPrefix
	}

	result, err := h.svc.StopAllRuns(r.Context(), orchestration.StopAllOptions{
		TagPrefix: tagPrefix,
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopAllRunsResponse{
		Result: protoconv.StopAllResultToProto(&protoconv.StopAllResult{
			Stopped:   result.Stopped,
			Failed:    result.Failed,
			Skipped:   result.Skipped,
			FailedIDs: result.FailedIDs,
		}),
	})
}

// QuiesceScenario drains in-flight runs targeting a scenario so a Baseline Modes
// promote can re-point and restart its live instance without killing in-flight
// agent work (Baseline Modes P6).
func (h *Handler) QuiesceScenario(w http.ResponseWriter, r *http.Request) {
	if h.denyRunInitiatedLifecycleOperation(w, r, "quiesce") {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.QuiesceScenarioRequest
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &req); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body")
			return
		}
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.QuiesceOptions{
		Scenario:    req.GetScenario(),
		ScopePrefix: req.GetScopePrefix(),
		TagPrefix:   req.GetTagPrefix(),
		Force:       req.GetForce(),
	}
	if v := req.GetExcludeRunId(); v != "" {
		id, perr := uuid.Parse(v)
		if perr != nil {
			writeSimpleError(w, r, "exclude_run_id", "invalid UUID format for exclude_run_id")
			return
		}
		opts.ExcludeRunID = &id
	}
	if v := req.GetTimeout(); v != "" {
		d, perr := time.ParseDuration(v)
		if perr != nil {
			writeSimpleError(w, r, "timeout", "invalid duration; use a Go duration like \"5m\"")
			return
		}
		opts.Timeout = d
	}

	result, err := h.svc.QuiesceScenario(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.QuiesceScenarioResponse{
		Result: quiesceResultToProto(result),
	})
}

func quiesceResultToProto(res *orchestration.QuiesceResult) *apipb.QuiesceResult {
	if res == nil {
		return nil
	}
	return &apipb.QuiesceResult{
		Scenario:  res.Scenario,
		Drained:   res.Drained,
		Aborted:   res.Aborted,
		Initial:   int32(res.Initial),
		InFlight:  quiesceRefsToProto(res.InFlight),
		Cancelled: quiesceRefsToProto(res.Cancelled),
		WaitedMs:  res.WaitedMs,
		Reason:    res.Reason,
	}
}

func quiesceRefsToProto(refs []orchestration.QuiesceRunRef) []*apipb.QuiesceRunRef {
	if len(refs) == 0 {
		return nil
	}
	out := make([]*apipb.QuiesceRunRef, len(refs))
	for i, ref := range refs {
		out[i] = &apipb.QuiesceRunRef{
			Id:        ref.ID,
			Tag:       ref.Tag,
			Status:    ref.Status,
			ScopePath: ref.ScopePath,
		}
	}
	return out
}

// GetRunEvents returns events for a run.
func (h *Handler) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	req := apipb.GetRunEventsRequest{RunId: id.String()}
	if after, provided, err := parseQueryInt64Strict(r, "after_sequence", "afterSequence"); err != nil {
		writeSimpleError(w, r, "after_sequence", "must be a number")
		return
	} else if provided {
		value := after
		req.AfterSequence = &value
	}
	if limit, provided, err := parseQueryIntStrict(r, "limit"); err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	} else if provided {
		value := int32(limit)
		req.Limit = &value
	}

	opts := event.GetOptions{AfterSequence: -1}
	eventTypesRaw := r.URL.Query()["event_types"]
	if len(eventTypesRaw) == 0 {
		eventTypesRaw = r.URL.Query()["eventTypes"]
	}
	if len(eventTypesRaw) > 0 {
		types, invalid := parseEventTypes(eventTypesRaw)
		if len(invalid) > 0 {
			writeSimpleError(w, r, "event_types", "invalid event type")
			return
		}
		req.EventTypes = make([]domainpb.RunEventType, len(types))
		for i, t := range types {
			req.EventTypes[i] = protoconv.RunEventTypeToProto(t)
		}
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	if req.AfterSequence != nil {
		opts.AfterSequence = req.GetAfterSequence()
	}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if len(req.EventTypes) > 0 {
		opts.EventTypes = make([]domain.RunEventType, len(req.EventTypes))
		for i, t := range req.EventTypes {
			opts.EventTypes[i] = protoconv.RunEventTypeFromProto(t)
		}
	}

	events, err := h.svc.GetRunEvents(r.Context(), id, opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunEventsResponse{
		Events: protoconv.RunEventsToProto(events),
	})
}

// GetRunDiff returns the diff for a run.
func (h *Handler) GetRunDiff(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetRunDiffRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	diff, err := h.svc.GetRunDiff(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert sandbox.FileChange to protoconv.FileChange
	files := make([]protoconv.FileChange, len(diff.Files))
	for i, f := range diff.Files {
		files[i] = protoconv.FileChange{
			ID:           f.ID,
			FilePath:     f.FilePath,
			ChangeType:   string(f.ChangeType),
			FileSize:     f.FileSize,
			LinesAdded:   f.LinesAdded,
			LinesRemoved: f.LinesRemoved,
		}
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunDiffResponse{
		Diff: protoconv.DiffResultToProto(id, &protoconv.DiffResult{
			SandboxID:   diff.SandboxID,
			Files:       files,
			UnifiedDiff: diff.UnifiedDiff,
			Generated:   diff.Generated,
		}),
	})
}

// ApproveRun approves a run's changes.
func (h *Handler) ApproveRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.ApproveRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.RunId != "" && req.RunId != id.String() {
		writeSimpleError(w, r, "run_id", "run_id does not match URL")
		return
	}

	actor := normalizeActor(req.GetActor())
	result, err := h.svc.ApproveRun(r.Context(), orchestration.ApproveRequest{
		RunID:     id,
		Actor:     actor,
		CommitMsg: req.GetCommitMsg(),
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ApproveRunResponse{
		Result: protoconv.ApproveResultToProto(&protoconv.ApproveResult{
			Success:    result.Success,
			Applied:    result.Applied,
			Remaining:  result.Remaining,
			IsPartial:  result.IsPartial,
			CommitHash: result.CommitHash,
			ErrorMsg:   result.ErrorMsg,
		}),
	})
}

// RejectRun rejects a run's changes.
func (h *Handler) RejectRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.RejectRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.RunId != "" && req.RunId != id.String() {
		writeSimpleError(w, r, "run_id", "run_id does not match URL")
		return
	}

	actor := normalizeActor(req.GetActor())
	reason := strings.TrimSpace(req.Reason)
	if err := h.svc.RejectRun(r.Context(), id, actor, reason); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.RejectRunResponse{Status: "rejected"})
}

// PartialApproveRun approves only selected files from a run's changes.
func (h *Handler) PartialApproveRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.PartialApproveRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.RunId != "" && req.RunId != id.String() {
		writeSimpleError(w, r, "run_id", "run_id does not match URL")
		return
	}

	if len(req.FileIds) == 0 {
		writeSimpleError(w, r, "file_ids", "at least one file ID is required")
		return
	}

	fileIDs := make([]uuid.UUID, len(req.FileIds))
	for i, fid := range req.FileIds {
		parsed, err := uuid.Parse(fid)
		if err != nil {
			writeSimpleError(w, r, "file_ids", "invalid UUID format for file ID: "+fid)
			return
		}
		fileIDs[i] = parsed
	}

	actor := normalizeActor(req.GetActor())
	result, err := h.svc.PartialApprove(r.Context(), orchestration.PartialApproveRequest{
		RunID:     id,
		FileIDs:   fileIDs,
		Actor:     actor,
		CommitMsg: req.GetCommitMsg(),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.PartialApproveRunResponse{
		Result: protoconv.ApproveResultToProto(&protoconv.ApproveResult{
			Success:    result.Success,
			Applied:    result.Applied,
			Remaining:  result.Remaining,
			IsPartial:  result.IsPartial,
			CommitHash: result.CommitHash,
			ErrorMsg:   result.ErrorMsg,
		}),
	})
}

type sandboxSyncRequest struct {
	RunID      string `json:"runId,omitempty"`
	SandboxID  string `json:"sandboxId,omitempty"`
	Status     string `json:"status"`
	Actor      string `json:"actor,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Applied    int    `json:"applied,omitempty"`
	Remaining  int    `json:"remaining,omitempty"`
	IsPartial  bool   `json:"isPartial,omitempty"`
	CommitHash string `json:"commitHash,omitempty"`
}

// SyncRunFromSandbox updates a run based on workspace-sandbox approval state.
func (h *Handler) SyncRunFromSandbox(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	var req sandboxSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if req.RunID != "" && req.RunID != id.String() {
		writeSimpleError(w, r, "runId", "runId does not match URL")
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		writeSimpleError(w, r, "status", "status is required")
		return
	}

	var sandboxID *uuid.UUID
	if strings.TrimSpace(req.SandboxID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(req.SandboxID))
		if err != nil {
			writeSimpleError(w, r, "sandboxId", "invalid UUID format for sandboxId")
			return
		}
		sandboxID = &parsed
	}

	run, err := h.svc.SyncRunFromSandbox(r.Context(), orchestration.SandboxSyncRequest{
		RunID:      id,
		SandboxID:  sandboxID,
		Status:     req.Status,
		Actor:      req.Actor,
		Reason:     req.Reason,
		Applied:    req.Applied,
		Remaining:  req.Remaining,
		IsPartial:  req.IsPartial,
		CommitHash: req.CommitHash,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "synced",
		"runId":         run.ID.String(),
		"runStatus":     run.Status,
		"approvalState": run.ApprovalState,
	})
}

// GetRunnerStatus returns status of all runners.
func (h *Handler) GetRunnerStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.GetRunnerStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert orchestration.RunnerStatus to protoconv types
	protoStatuses := make([]*protoconv.OrchestratorRunnerStatus, len(statuses))
	for i, s := range statuses {
		protoStatuses[i] = &protoconv.OrchestratorRunnerStatus{
			Type:      s.Type,
			Available: s.Available,
			Message:   s.Message,
			Capabilities: protoconv.RunnerCapabilities{
				SupportsMessages:        s.Capabilities.SupportsMessages,
				SupportsToolEvents:      s.Capabilities.SupportsToolEvents,
				SupportsCostTracking:    s.Capabilities.SupportsCostTracking,
				SupportsStreaming:       s.Capabilities.SupportsStreaming,
				SupportsCancellation:    s.Capabilities.SupportsCancellation,
				MaxTurns:                s.Capabilities.MaxTurns,
				SupportedModels:         s.Capabilities.SupportedModels,
				SupportsToolRestriction: s.Capabilities.SupportsToolRestriction,
				ToolRestrictionMappings: s.Capabilities.ToolRestrictionMappings,
			},
		}
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunnerStatusResponse{
		Runners: protoconv.OrchestratorRunnerStatusesToProto(protoStatuses),
	})
}

// ProbeRunner sends a test request to verify a runner can respond.
func (h *Handler) ProbeRunner(w http.ResponseWriter, r *http.Request) {
	runnerType := mux.Vars(r)["runner_type"]
	if runnerType == "" {
		writeSimpleError(w, r, "runner_type", "runner type is required")
		return
	}
	parsed, ok := parseRunnerType(runnerType)
	if !ok {
		writeSimpleError(w, r, "runner_type", "invalid runner type")
		return
	}

	result, err := h.svc.ProbeRunner(r.Context(), parsed)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ProbeRunnerResponse{
		Result: protoconv.ProbeResultToProto(&protoconv.ProbeResult{
			RunnerType: result.RunnerType,
			Success:    result.Success,
			Message:    result.Message,
			Response:   result.Response,
			DurationMs: result.DurationMs,
		}),
	})
}

// GetRolePolicyStatus returns truthful portable catalog activation state.
func (h *Handler) GetRolePolicyStatus(w http.ResponseWriter, r *http.Request) {
	if h.rolePolicy == nil {
		writeSimpleError(w, r, "role_policy", "role policy state is not configured")
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.GetRolePolicyStatusResponse{
		Status: protoconv.RolePolicyStatusToProto(h.rolePolicy.Status()),
	})
}

// GetRolePolicyCatalog exposes only portable roles and resource-role candidates.
func (h *Handler) GetRolePolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if h.rolePolicy == nil {
		writeSimpleError(w, r, "role_policy", "role policy state is not configured")
		return
	}
	var catalog *rolepolicy.Catalog
	if active := h.rolePolicy.Active(); active != nil {
		catalog = active.Catalog()
	}
	writeProtoJSON(w, http.StatusOK, &apipb.GetRolePolicyCatalogResponse{
		Status:  protoconv.RolePolicyStatusToProto(h.rolePolicy.Status()),
		Catalog: protoconv.RolePolicyCatalogToProto(catalog),
	})
}

// ValidateRolePolicyCatalog validates declared state without activation.
func (h *Handler) ValidateRolePolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if h.rolePolicy == nil {
		writeSimpleError(w, r, "role_policy", "role policy state is not configured")
		return
	}
	status := h.rolePolicy.Status()
	revision, err := h.rolePolicy.Validate()
	response := &apipb.ValidateRolePolicyCatalogResponse{ActiveDigest: status.ActiveDigest}
	if err != nil {
		response.Diagnostic = protoconv.RolePolicyDiagnosticToProto(&rolepolicy.Diagnostic{Code: rolepolicy.DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: err.Error()})
	} else {
		response.Valid = true
		response.CandidateDigest = revision.Digest()
	}
	writeProtoJSON(w, http.StatusOK, response)
}

// ReloadRolePolicyCatalog validates before one atomic activation swap.
func (h *Handler) ReloadRolePolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if h.rolePolicy == nil {
		writeSimpleError(w, r, "role_policy", "role policy state is not configured")
		return
	}
	_, err := h.rolePolicy.Reload()
	response := &apipb.ReloadRolePolicyCatalogResponse{
		Activated: err == nil,
		Status:    protoconv.RolePolicyStatusToProto(h.rolePolicy.Status()),
	}
	if err != nil {
		response.Diagnostic = protoconv.RolePolicyDiagnosticToProto(&rolepolicy.Diagnostic{Code: rolepolicy.DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: err.Error()})
	}
	writeProtoJSON(w, http.StatusOK, response)
}

// ExplainRolePolicy resolves a profile now or returns a run's persisted
// snapshot. Historical runs never borrow provenance from the current catalog.
func (h *Handler) ExplainRolePolicy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}
	var req apipb.ExplainRolePolicyRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	response := &apipb.ExplainRolePolicyResponse{}
	var snapshot *domain.ExecutionPolicySnapshot
	switch {
	case req.GetProfileId() != "":
		id, parseErr := uuid.Parse(req.GetProfileId())
		if parseErr != nil {
			writeSimpleError(w, r, "profile_id", "invalid UUID format")
			return
		}
		response.TargetType, response.TargetId = "profile", id.String()
		snapshot, err = h.svc.ExplainProfilePolicy(r.Context(), id)
	case req.GetRunId() != "":
		id, parseErr := uuid.Parse(req.GetRunId())
		if parseErr != nil {
			writeSimpleError(w, r, "run_id", "invalid UUID format")
			return
		}
		response.TargetType, response.TargetId = "run", id.String()
		snapshot, err = h.svc.ExplainRunPolicy(r.Context(), id)
		response.HistoricalWithoutSnapshot = err == nil && snapshot == nil
	default:
		writeSimpleError(w, r, "target", "exactly one of profile_id or run_id is required")
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.Snapshot = protoconv.ExecutionPolicySnapshotToProto(snapshot)
	if snapshot != nil {
		response.Summary = snapshot.Explanation.Summary
	} else if response.HistoricalWithoutSnapshot {
		response.Summary = "historical run has no persisted policy snapshot; current catalog provenance was not fabricated"
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) permissionPolicyConfigured(w http.ResponseWriter, r *http.Request) bool {
	if h.permissionPolicyState == nil || h.permissionPolicy == nil {
		writeSimpleError(w, r, "permission_policy", "permission policy service is not configured")
		return false
	}
	return true
}

// GetPermissionPolicyStatus returns activation state and the last persisted
// reconcile evidence. It never triggers a native resource operation.
func (h *Handler) GetPermissionPolicyStatus(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	last, err := h.permissionPolicy.LastReconcile(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.GetPermissionPolicyStatusResponse{
		Status:        protoconv.PermissionPolicyStatusToProto(h.permissionPolicyState.Status()),
		LastReconcile: protoconv.PermissionPolicyReconcileResultToProto(last),
	})
}

// GetPermissionPolicyCatalog exposes the global portable desired state, not
// native resource configuration or runner-specific syntax.
func (h *Handler) GetPermissionPolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	var catalog *permissionpolicy.Catalog
	if active := h.permissionPolicyState.Active(); active != nil {
		catalog = active.Catalog()
	}
	writeProtoJSON(w, http.StatusOK, &apipb.GetPermissionPolicyCatalogResponse{
		Status:  protoconv.PermissionPolicyStatusToProto(h.permissionPolicyState.Status()),
		Catalog: protoconv.PermissionPolicyCatalogToProto(catalog),
	})
}

func (h *Handler) ValidatePermissionPolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	status := h.permissionPolicyState.Status()
	revision, err := h.permissionPolicyState.Validate()
	response := &apipb.ValidatePermissionPolicyCatalogResponse{ActiveDigest: status.ActiveDigest}
	if err != nil {
		response.Diagnostic = protoconv.PermissionPolicyDiagnosticToProto(&permissionpolicy.Diagnostic{Code: permissionpolicy.DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: err.Error()})
	} else {
		response.Valid = true
		response.CandidateDigest = revision.Digest()
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) ReloadPermissionPolicyCatalog(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	_, err := h.permissionPolicyState.Reload()
	response := &apipb.ReloadPermissionPolicyCatalogResponse{
		Activated: err == nil,
		Status:    protoconv.PermissionPolicyStatusToProto(h.permissionPolicyState.Status()),
	}
	if err != nil {
		response.Diagnostic = protoconv.PermissionPolicyDiagnosticToProto(&permissionpolicy.Diagnostic{Code: permissionpolicy.DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: err.Error()})
		obs.Component("permission-policy").Error("permission_policy_reload_failed", obs.KeyError, err.Error(), obs.KeyPermissionPolicyDigest, response.Status.GetActiveDigest())
	} else {
		obs.Component("permission-policy").Info("permission_policy_reloaded", obs.KeyPermissionPolicyDigest, response.Status.GetActiveDigest())
	}
	writeProtoJSON(w, http.StatusOK, response)
}

func (h *Handler) PlanPermissionPolicy(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	plan, err := h.permissionPolicy.Plan(r.Context())
	if err != nil {
		writeSimpleError(w, r, "permission_policy", err.Error())
		return
	}
	logPermissionPolicyPlan("permission_policy_planned", plan)
	writeProtoJSON(w, http.StatusOK, &apipb.PlanPermissionPolicyResponse{Plan: protoconv.PermissionPolicyPlanToProto(plan)})
}

func (h *Handler) ReconcilePermissionPolicy(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}
	var request apipb.ReconcilePermissionPolicyRequest
	if err := protoconv.UnmarshalJSON(body, &request); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !request.ExplicitlyAuthorized {
		writeSimpleError(w, r, "explicitly_authorized", "explicit human authorization is required")
		return
	}
	obs.Component("permission-policy").Info("permission_policy_reconcile_started", obs.KeyPermissionPolicyDigest, h.permissionPolicyState.Status().ActiveDigest)
	result, err := h.permissionPolicy.Reconcile(r.Context(), true)
	if err != nil {
		writeError(w, r, err)
		return
	}
	logPermissionPolicyReconcile(result)
	writeProtoJSON(w, http.StatusOK, &apipb.ReconcilePermissionPolicyResponse{Result: protoconv.PermissionPolicyReconcileResultToProto(&result)})
}

func (h *Handler) DoctorPermissionPolicy(w http.ResponseWriter, r *http.Request) {
	if !h.permissionPolicyConfigured(w, r) {
		return
	}
	plan, err := h.permissionPolicy.Plan(r.Context())
	if err != nil {
		writeSimpleError(w, r, "permission_policy", err.Error())
		return
	}
	logPermissionPolicyPlan("permission_policy_doctor_planned", plan)
	status := h.permissionPolicyState.Status()
	healthy := status.Ready && plan.HardEnforcementSatisfied
	summary := "permission policy is ready; no required hard-enforcement gap was detected"
	if !status.Ready {
		summary = "permission policy catalog is not ready"
	} else if !plan.HardEnforcementSatisfied {
		summary = "one or more required permission rules lack native or hook-backed enforcement"
	}
	writeProtoJSON(w, http.StatusOK, &apipb.DoctorPermissionPolicyResponse{
		Status:  protoconv.PermissionPolicyStatusToProto(status),
		Plan:    protoconv.PermissionPolicyPlanToProto(plan),
		Healthy: healthy,
		Summary: summary,
	})
}

func logPermissionPolicyPlan(event string, plan permissionpolicy.AggregatePlan) {
	driftCount, unsupportedCount := permissionPolicyObservationCounts(plan.Resources)
	attrs := []any{
		obs.KeyPermissionPolicyDigest, plan.CatalogDigest,
		obs.KeyPermissionPolicyResourceCount, len(plan.Resources),
		obs.KeyPermissionPolicyDriftCount, driftCount,
		obs.KeyPermissionPolicyUnsupportedCount, unsupportedCount,
		obs.KeyHardEnforcementSatisfied, plan.HardEnforcementSatisfied,
		obs.KeyMissingHardEnforcementRuleIDs, plan.MissingHardEnforcementRuleIDs,
	}
	logger := obs.Component("permission-policy")
	if !plan.HardEnforcementSatisfied || unsupportedCount > 0 {
		logger.Warn(event, attrs...)
		return
	}
	logger.Info(event, attrs...)
}

func logPermissionPolicyReconcile(result permissionpolicy.ReconcileResult) {
	driftCount, unsupportedCount := permissionPolicyObservationCounts(result.Resources)
	attrs := []any{
		obs.KeyPermissionPolicyDigest, result.CatalogDigest,
		obs.KeyPermissionPolicyResourceCount, len(result.Resources),
		obs.KeyPermissionPolicyDriftCount, driftCount,
		obs.KeyPermissionPolicyUnsupportedCount, unsupportedCount,
		obs.KeyHardEnforcementSatisfied, result.HardEnforcementSatisfied,
		obs.KeyMissingHardEnforcementRuleIDs, result.MissingHardEnforcementRuleIDs,
		obs.KeyPermissionPolicyPartialFailure, !result.Success,
	}
	logger := obs.Component("permission-policy")
	if !result.Success || unsupportedCount > 0 {
		logger.Warn("permission_policy_reconcile_completed", attrs...)
		return
	}
	logger.Info("permission_policy_reconcile_completed", attrs...)
}

func permissionPolicyObservationCounts(resources []permissionpolicy.ResourcePlan) (driftCount, unsupportedCount int) {
	for _, resource := range resources {
		if resource.Drift {
			driftCount++
		}
		unsupportedCount += len(resource.UnsupportedMatchers)
	}
	return driftCount, unsupportedCount
}

// PurgeData deletes profiles, tasks, or runs matching a regex pattern.
func (h *Handler) PurgeData(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.PurgeDataRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Pattern == "" {
		writeSimpleError(w, r, "pattern", "pattern is required")
		return
	}
	if len(req.Targets) == 0 {
		writeSimpleError(w, r, "targets", "targets are required")
		return
	}

	targets := make([]orchestration.PurgeTarget, 0, len(req.Targets))
	for _, target := range req.Targets {
		switch target {
		case apipb.PurgeTarget_PURGE_TARGET_PROFILES:
			targets = append(targets, orchestration.PurgeTargetProfiles)
		case apipb.PurgeTarget_PURGE_TARGET_TASKS:
			targets = append(targets, orchestration.PurgeTargetTasks)
		case apipb.PurgeTarget_PURGE_TARGET_RUNS:
			targets = append(targets, orchestration.PurgeTargetRuns)
		default:
			writeSimpleError(w, r, "targets", "invalid purge target")
			return
		}
	}

	result, err := h.svc.PurgeData(r.Context(), orchestration.PurgeRequest{
		Pattern: req.Pattern,
		Targets: targets,
		DryRun:  req.DryRun,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.PurgeDataResponse{
		Matched: purgeCountsToProto(result.Matched),
		Deleted: purgeCountsToProto(result.Deleted),
		DryRun:  result.DryRun,
	})
}

func purgeCountsToProto(counts orchestration.PurgeCounts) *apipb.PurgeCounts {
	return &apipb.PurgeCounts{
		Profiles: int32(counts.Profiles),
		Tasks:    int32(counts.Tasks),
		Runs:     int32(counts.Runs),
	}
}

func normalizeActor(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}
