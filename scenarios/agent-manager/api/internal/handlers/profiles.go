package handlers

import (
	"io"
	"net/http"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// =============================================================================
// PROFILE HANDLERS
// =============================================================================

// CreateProfile creates a new agent profile.
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.CreateProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Profile == nil {
		writeSimpleError(w, r, "profile", "profile is required")
		return
	}

	profile := protoconv.AgentProfileFromProto(req.Profile)

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.profiles.CreateProfile(r.Context(), profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateProfileResponse{
		Profile: protoconv.AgentProfileToProto(result),
	})
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (h *Handler) EnsureProfile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.EnsureProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	result, err := h.profiles.EnsureProfile(r.Context(), orchestration.EnsureProfileRequest{
		ProfileKey:     req.ProfileKey,
		Defaults:       protoconv.AgentProfileFromProto(req.Defaults),
		UpdateExisting: req.UpdateExisting,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.EnsureProfileResponse{
		Profile: protoconv.AgentProfileToProto(result.Profile),
		Created: result.Created,
		Updated: result.Updated,
	})
}

// ReconcileScenarioProfiles reconciles all profile sources declared by a scenario.
func (h *Handler) ReconcileScenarioProfiles(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.ReconcileScenarioProfilesRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	result, err := h.profiles.ReconcileScenarioProfiles(r.Context(), orchestration.ReconcileScenarioProfilesRequest{
		Scenario: req.Scenario,
		DryRun:   req.DryRun,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, reconcileScenarioProfilesToProto(result))
}

func reconcileScenarioProfilesToProto(result *orchestration.ReconcileScenarioProfilesResult) *apipb.ReconcileScenarioProfilesResponse {
	if result == nil {
		return &apipb.ReconcileScenarioProfilesResponse{}
	}
	items := make([]*apipb.ProfileReconcileResult, 0, len(result.Results))
	for _, item := range result.Results {
		items = append(items, &apipb.ProfileReconcileResult{
			ProfileKey:  item.ProfileKey,
			SourcePath:  item.SourcePath,
			SourceHash:  item.SourceHash,
			ProfileId:   item.ProfileID,
			Status:      profileReconcileStatusToProto(item.Status),
			Message:     item.Message,
			Diagnostics: workflowDiagnosticsToProto(item.Diagnostics),
		})
	}
	return &apipb.ReconcileScenarioProfilesResponse{
		Scenario:   result.Scenario,
		Results:    items,
		Created:    int32(result.Created),
		Updated:    int32(result.Updated),
		Unchanged:  int32(result.Unchanged),
		Skipped:    int32(result.Skipped),
		Conflicted: int32(result.Conflicted),
		Failed:     int32(result.Failed),
		DryRun:     result.DryRun,
	}
}

func profileReconcileStatusToProto(status orchestration.ProfileReconcileStatus) apipb.ProfileReconcileStatus {
	switch status {
	case orchestration.ProfileReconcileStatusCreated:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CREATED
	case orchestration.ProfileReconcileStatusUpdated:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UPDATED
	case orchestration.ProfileReconcileStatusUnchanged:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UNCHANGED
	case orchestration.ProfileReconcileStatusSkipped:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_SKIPPED
	case orchestration.ProfileReconcileStatusConflictedLocalOverride:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CONFLICTED_LOCAL_OVERRIDE
	case orchestration.ProfileReconcileStatusFailedValidation:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_FAILED_VALIDATION
	default:
		return apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UNSPECIFIED
	}
}

// ReconcileScenarioDeclarations reconciles a scenario's unified declaration
// block (profiles and workflows) in one call.
func (h *Handler) ReconcileScenarioDeclarations(w http.ResponseWriter, r *http.Request) {
	h.reconcileScenarioDeclarations(w, r, false)
}

// PlanScenarioDeclarations validates every declaration source without writes.
func (h *Handler) PlanScenarioDeclarations(w http.ResponseWriter, r *http.Request) {
	h.reconcileScenarioDeclarations(w, r, true)
}

func (h *Handler) reconcileScenarioDeclarations(w http.ResponseWriter, r *http.Request, forceDryRun bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}
	var req apipb.ReconcileScenarioDeclarationsRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	result, err := h.profiles.ReconcileScenarioDeclarations(r.Context(), orchestration.ReconcileScenarioDeclarationsRequest{
		Scenario:     req.Scenario,
		DryRun:       req.DryRun || forceDryRun,
		ValidateOnly: req.ValidateOnly,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, reconcileScenarioDeclarationsToProto(result))
}

func reconcileScenarioDeclarationsToProto(result *orchestration.ReconcileScenarioDeclarationsResult) *apipb.ReconcileScenarioDeclarationsResponse {
	if result == nil {
		return &apipb.ReconcileScenarioDeclarationsResponse{}
	}
	profiles := make([]*apipb.ProfileReconcileResult, 0, len(result.ProfileResults))
	for _, item := range result.ProfileResults {
		profiles = append(profiles, &apipb.ProfileReconcileResult{
			ProfileKey: item.ProfileKey,
			SourcePath: item.SourcePath,
			SourceHash: item.SourceHash,
			ProfileId:  item.ProfileID,
			Status:     profileReconcileStatusToProto(item.Status),
			Message:    item.Message,
		})
	}
	workflows := make([]*apipb.WorkflowReconcileResult, 0, len(result.WorkflowResults))
	for _, item := range result.WorkflowResults {
		workflows = append(workflows, workflowReconcileResultToProto(item))
	}
	return &apipb.ReconcileScenarioDeclarationsResponse{
		Scenario:           result.Scenario,
		ProfileResults:     profiles,
		WorkflowResults:    workflows,
		ProfilesCreated:    int32(result.ProfilesCreated),
		ProfilesUpdated:    int32(result.ProfilesUpdated),
		ProfilesUnchanged:  int32(result.ProfilesUnchanged),
		ProfilesSkipped:    int32(result.ProfilesSkipped),
		ProfilesConflicted: int32(result.ProfilesConflicted),
		ProfilesFailed:     int32(result.ProfilesFailed),
		WorkflowsCreated:   int32(result.WorkflowsCreated),
		WorkflowsActivated: int32(result.WorkflowsActivated),
		WorkflowsUnchanged: int32(result.WorkflowsUnchanged),
		WorkflowsSkipped:   int32(result.WorkflowsSkipped),
		WorkflowsFailed:    int32(result.WorkflowsFailed),
		DryRun:             result.DryRun,
		ValidateOnly:       result.ValidateOnly,
	}
}

// GetProfile retrieves a profile by ID.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetProfileRequest{ProfileId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.ProfileId)
	if err != nil {
		writeSimpleError(w, r, "profile_id", "invalid UUID format for profile ID")
		return
	}

	profile, err := h.profiles.GetProfile(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetProfileResponse{
		Profile: protoconv.AgentProfileToProto(profile),
	})
}

// ListProfiles returns all agent profiles.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	limit, limitProvided, err := parseQueryIntStrict(r, "limit")
	if err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	}
	offset, offsetProvided, err := parseQueryIntStrict(r, "offset")
	if err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	}
	req := apipb.ListProfilesRequest{}
	if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.ListOptions{}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}
	profiles, err := h.profiles.ListProfiles(r.Context(), orchestration.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListProfilesResponse{
		Profiles: protoconv.AgentProfilesToProto(profiles),
		Total:    int32(len(profiles)),
	})
}

// UpdateProfile updates an existing profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.UpdateProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Profile == nil {
		writeSimpleError(w, r, "profile", "profile is required")
		return
	}
	if req.ProfileId != "" {
		if req.ProfileId != id.String() {
			writeSimpleError(w, r, "profile_id", "profile_id does not match URL")
			return
		}
	}

	profile := protoconv.AgentProfileFromProto(req.Profile)
	profile.ID = id

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.profiles.UpdateProfile(r.Context(), profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.UpdateProfileResponse{
		Profile: protoconv.AgentProfileToProto(result),
	})
}

// DeleteProfile removes a profile.
func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteProfileRequest{ProfileId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.ProfileId)
	if err != nil {
		writeSimpleError(w, r, "profile_id", "invalid UUID format for profile ID")
		return
	}

	if err := h.profiles.DeleteProfile(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteProfileResponse{Success: true})
}
