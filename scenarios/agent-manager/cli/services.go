package main

import (
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"net/url"
)

// Services aggregates all domain-specific services.
type Services struct {
	Profiles         *ProfileService
	Declarations     *DeclarationService
	Workflows        *WorkflowService
	Watches          *WatchService
	Investigations   *InvestigationService
	Tasks            *TaskService
	Runs             *RunService
	Runners          *RunnerService
	Policy           *PolicyService
	PermissionPolicy *PermissionPolicyService
	Settings         *SettingsService
	Maintenance      *MaintenanceService
	Operational      *OperationalService
	HealthAudit      *HealthAuditService
	Events           *EventsService
	Findings         *FindingsService
	Subscriptions    *SubscriptionService
	Conversation     *ConversationSearchService
}

// NewServices creates a new Services instance with all domain services.
func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Profiles:         &ProfileService{api: api},
		Declarations:     &DeclarationService{api: api},
		Workflows:        &WorkflowService{api: api},
		Watches:          &WatchService{api: api},
		Investigations:   &InvestigationService{api: api},
		Tasks:            &TaskService{api: api},
		Runs:             &RunService{api: api},
		Runners:          &RunnerService{api: api},
		Policy:           &PolicyService{api: api},
		PermissionPolicy: &PermissionPolicyService{api: api},
		Settings:         &SettingsService{api: api},
		Maintenance:      &MaintenanceService{api: api},
		Operational:      &OperationalService{api: api},
		HealthAudit:      &HealthAuditService{api: api},
		Events:           &EventsService{api: api},
		Findings:         &FindingsService{api: api},
		Subscriptions:    &SubscriptionService{api: api},
		Conversation:     &ConversationSearchService{api: api},
	}
}

// ProfileService handles profile-related API operations.
type ProfileService struct {
	api *cliutil.APIClient
}

// DeclarationService drives the unified scenario declaration reconcile.
type DeclarationService struct{ api *cliutil.APIClient }

// Reconcile reconciles (or, at the plan path, validates) a scenario's unified
// declaration block.
func (s *DeclarationService) Reconcile(path string, req *apipb.ReconcileScenarioDeclarationsRequest) ([]byte, *apipb.ReconcileScenarioDeclarationsResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", path, nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ReconcileScenarioDeclarationsResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
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

// ReconcileScenario reconciles profiles declared by a scenario manifest.
func (s *ProfileService) ReconcileScenario(req *apipb.ReconcileScenarioProfilesRequest) ([]byte, *apipb.ReconcileScenarioProfilesResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/profiles/reconcile-scenario", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ReconcileScenarioProfilesResponse
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

var protoMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

var protoUnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}
