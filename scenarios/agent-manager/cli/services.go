package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// Services Aggregator
// =============================================================================

// Services aggregates all domain-specific services.
type Services struct {
	Profiles    *ProfileService
	Tasks       *TaskService
	Runs        *RunService
	Runners     *RunnerService
	Settings    *SettingsService
	Maintenance *MaintenanceService
}

// NewServices creates a new Services instance with all domain services.
func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Profiles:    &ProfileService{api: api},
		Tasks:       &TaskService{api: api},
		Runs:        &RunService{api: api},
		Runners:     &RunnerService{api: api},
		Settings:    &SettingsService{api: api},
		Maintenance: &MaintenanceService{api: api},
	}
}

// =============================================================================
// Profile Service
// =============================================================================

// ProfileService handles profile-related API operations.
type ProfileService struct {
	api *cliutil.APIClient
}

// List retrieves all profiles.
func (s *ProfileService) List(limit, offset int) ([]byte, []*domainpb.AgentProfile, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	body, err := s.api.Get("/api/v1/profiles", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ListProfilesResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil // Return raw body for fallback
	}
	return body, resp.Profiles, nil
}

// Get retrieves a single profile by ID.
func (s *ProfileService) Get(id string) ([]byte, *domainpb.AgentProfile, error) {
	body, err := s.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetProfileResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Profile, nil
}

// Create creates a new profile.
func (s *ProfileService) Create(profile *domainpb.AgentProfile) ([]byte, *domainpb.AgentProfile, error) {
	payload, err := marshalProtoRequest(&apipb.CreateProfileRequest{Profile: profile})
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/profiles", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateProfileResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Profile, nil
}

// Update updates an existing profile.
func (s *ProfileService) Update(id string, profile *domainpb.AgentProfile) ([]byte, *domainpb.AgentProfile, error) {
	payload, err := marshalProtoRequest(&apipb.UpdateProfileRequest{ProfileId: id, Profile: profile})
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.UpdateProfileResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Profile, nil
}

// Delete removes a profile.
func (s *ProfileService) Delete(id string) error {
	_, err := s.api.Request("DELETE", "/api/v1/profiles/"+id, nil, nil)
	return err
}

// Ensure resolves a profile by key, creating it with defaults if needed.
func (s *ProfileService) Ensure(req *apipb.EnsureProfileRequest) ([]byte, *apipb.EnsureProfileResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/profiles/ensure", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.EnsureProfileResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// =============================================================================
// Task Service
// =============================================================================

// TaskService handles task-related API operations.
type TaskService struct {
	api *cliutil.APIClient
}

// List retrieves all tasks.
func (s *TaskService) List(limit, offset int, status string) ([]byte, []*domainpb.Task, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}
	if status != "" {
		query.Set("status", status)
	}

	body, err := s.api.Get("/api/v1/tasks", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ListTasksResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Tasks, nil
}

// Get retrieves a single task by ID.
func (s *TaskService) Get(id string) ([]byte, *domainpb.Task, error) {
	body, err := s.api.Get("/api/v1/tasks/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetTaskResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Task, nil
}

// Create creates a new task.
func (s *TaskService) Create(task *domainpb.Task) ([]byte, *domainpb.Task, error) {
	payload, err := marshalProtoRequest(&apipb.CreateTaskRequest{Task: task})
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/tasks", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateTaskResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Task, nil
}

// Cancel cancels a task.
func (s *TaskService) Cancel(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/tasks/"+id+"/cancel", nil, nil)
}

// Update updates an existing task.
func (s *TaskService) Update(id string, task *domainpb.Task) ([]byte, *domainpb.Task, error) {
	payload, err := marshalProtoRequest(&apipb.UpdateTaskRequest{TaskId: id, Task: task})
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("PUT", "/api/v1/tasks/"+id, nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.UpdateTaskResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Task, nil
}

// Delete removes a task.
func (s *TaskService) Delete(id string) error {
	_, err := s.api.Request("DELETE", "/api/v1/tasks/"+id, nil, nil)
	return err
}

// =============================================================================
// Run Service
// =============================================================================

// RunService handles run-related API operations.
type RunService struct {
	api *cliutil.APIClient
}

// List retrieves runs with optional filters.
func (s *RunService) List(limit, offset int, taskID, profileID, status, tagPrefix string) ([]byte, []*domainpb.Run, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}
	if taskID != "" {
		query.Set("taskId", taskID)
	}
	if profileID != "" {
		query.Set("profileId", profileID)
	}
	if status != "" {
		query.Set("status", status)
	}
	if tagPrefix != "" {
		query.Set("tagPrefix", tagPrefix)
	}

	body, err := s.api.Get("/api/v1/runs", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ListRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Runs, nil
}

// Get retrieves a single run by ID.
func (s *RunService) Get(id string) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Get("/api/v1/runs/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Create creates a new run.
func (s *RunService) Create(req *apipb.CreateRunRequest) ([]byte, *domainpb.Run, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Stop stops a running execution.
func (s *RunService) Stop(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/stop", nil, nil)
}

// GetByTag retrieves a run by its custom tag.
func (s *RunService) GetByTag(tag string) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Get("/api/v1/runs/tag/"+tag, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunByTagResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// StopByTag stops a run identified by its custom tag.
func (s *RunService) StopByTag(tag string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/tag/"+tag+"/stop", nil, nil)
}

// StopAll stops all running runs, optionally filtered by tag prefix.
func (s *RunService) StopAll(req *apipb.StopAllRunsRequest) ([]byte, *domainpb.StopAllResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/stop-all", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopAllRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// Approve approves a run.
func (s *RunService) Approve(id string, req *apipb.ApproveRunRequest) ([]byte, *domainpb.ApproveResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/approve", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ApproveRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// Reject rejects a run.
func (s *RunService) Reject(id string, req *apipb.RejectRunRequest) ([]byte, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, err
	}
	return s.api.Request("POST", "/api/v1/runs/"+id+"/reject", nil, payload)
}

// GetDiff retrieves the diff for a run.
func (s *RunService) GetDiff(id string) ([]byte, *domainpb.RunDiff, error) {
	body, err := s.api.Get("/api/v1/runs/"+id+"/diff", nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunDiffResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Diff, nil
}

// GetEvents retrieves events for a run.
func (s *RunService) GetEvents(id string, limit int) ([]byte, []*domainpb.RunEvent, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	body, err := s.api.Get("/api/v1/runs/"+id+"/events", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunEventsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Events, nil
}

// Delete removes a run.
func (s *RunService) Delete(id string) error {
	_, err := s.api.Request("DELETE", "/api/v1/runs/"+id, nil, nil)
	return err
}

// Continue continues an existing run with a follow-up message.
func (s *RunService) Continue(id string, req *domainpb.ContinueRunRequest) ([]byte, *domainpb.Run, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/continue", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.ContinueRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Investigate creates an investigation run from one or more existing runs.
func (s *RunService) Investigate(req json.RawMessage) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/investigate", nil, req)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// InvestigationApply creates a run that applies investigation recommendations.
func (s *RunService) InvestigationApply(req json.RawMessage) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/investigation-apply", nil, req)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// SandboxSync syncs run state from a sandbox.
func (s *RunService) SandboxSync(id string, req json.RawMessage) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/sandbox-sync", nil, req)
}

// ExtractRecommendations extracts recommendations from an investigation run.
func (s *RunService) ExtractRecommendations(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/extract-recommendations", nil, nil)
}

// RegenerateRecommendations regenerates recommendations for an investigation run.
func (s *RunService) RegenerateRecommendations(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/regenerate-recommendations", nil, nil)
}

// =============================================================================
// Runner Service
// =============================================================================

// RunnerService handles runner-related API operations.
type RunnerService struct {
	api *cliutil.APIClient
}

// GetStatus retrieves the status of all runners.
func (s *RunnerService) GetStatus() ([]byte, []*domainpb.RunnerStatus, error) {
	body, err := s.api.Get("/api/v1/runners", nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunnerStatusResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Runners, nil
}

// Probe sends a test request to verify a runner can respond.
func (s *RunnerService) Probe(runnerType string) ([]byte, *domainpb.ProbeResult, error) {
	body, err := s.api.Request("POST", "/api/v1/runners/"+runnerType+"/probe", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ProbeRunnerResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// GetModels retrieves the model registry for all runners.
func (s *RunnerService) GetModels() ([]byte, error) {
	return s.api.Get("/api/v1/runner-models", nil)
}

// UpdateModels replaces the model registry with the provided payload.
func (s *RunnerService) UpdateModels(data json.RawMessage) ([]byte, error) {
	return s.api.Request("PUT", "/api/v1/runner-models", nil, data)
}

// =============================================================================
// Settings Service
// =============================================================================

// SettingsService handles settings-related API operations.
type SettingsService struct {
	api *cliutil.APIClient
}

// GetInvestigation retrieves investigation settings.
func (s *SettingsService) GetInvestigation() ([]byte, error) {
	return s.api.Get("/api/v1/investigation-settings", nil)
}

// UpdateInvestigation updates investigation settings.
func (s *SettingsService) UpdateInvestigation(data json.RawMessage) ([]byte, error) {
	return s.api.Request("PUT", "/api/v1/investigation-settings", nil, data)
}

// ResetInvestigation resets investigation settings to defaults.
func (s *SettingsService) ResetInvestigation() ([]byte, error) {
	return s.api.Request("POST", "/api/v1/investigation-settings/reset", nil, nil)
}

// =============================================================================
// Maintenance Service
// =============================================================================

// MaintenanceService handles maintenance-related API operations.
type MaintenanceService struct {
	api *cliutil.APIClient
}

// Purge deletes profiles, tasks, or runs matching a regex pattern.
func (s *MaintenanceService) Purge(req *apipb.PurgeDataRequest) ([]byte, *apipb.PurgeDataResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/maintenance/purge", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.PurgeDataResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// =============================================================================
// Proto Helpers
// =============================================================================

var protoMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

var protoUnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

func marshalProtoRequest(msg proto.Message) (json.RawMessage, error) {
	data, err := protoMarshalOptions.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func unmarshalProtoResponse(data []byte, msg proto.Message) error {
	return protoUnmarshalOptions.Unmarshal(data, msg)
}
