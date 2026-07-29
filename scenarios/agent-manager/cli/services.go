package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

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
	Profiles         *ProfileService
	Declarations     *DeclarationService
	Workflows        *WorkflowService
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
}

// NewServices creates a new Services instance with all domain services.
func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Profiles:         &ProfileService{api: api},
		Declarations:     &DeclarationService{api: api},
		Workflows:        &WorkflowService{api: api},
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
	}
}

type FindingsService struct{ api *cliutil.APIClient }

func (s *FindingsService) List(query url.Values) ([]byte, error) {
	return s.api.Get("/api/v1/findings", query)
}

type WorkflowService struct{ api *cliutil.APIClient }

func (s *WorkflowService) Validate(req *apipb.ValidateWorkflowRequest) ([]byte, *apipb.ValidateWorkflowResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflows/validate", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ValidateWorkflowResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Reconcile(path string, req *apipb.ReconcileScenarioWorkflowsRequest) ([]byte, *apipb.ReconcileScenarioWorkflowsResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", path, nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ReconcileScenarioWorkflowsResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) List(owner, key string, limit, offset int) ([]byte, *apipb.ListWorkflowRevisionsResponse, error) {
	query := url.Values{"owner": {owner}}
	if key != "" {
		query.Set("key", key)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprint(limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprint(offset))
	}
	body, err := s.api.Get("/api/v1/workflows", query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ListWorkflowRevisionsResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Get(path, owner, key, digest string) ([]byte, *apipb.GetWorkflowRevisionResponse, error) {
	query := url.Values{"owner": {owner}}
	if key != "" {
		query.Set("key", key)
	}
	if digest != "" {
		query.Set("digest", digest)
	}
	body, err := s.api.Get(path, query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.GetWorkflowRevisionResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) StartExecution(req *apipb.StartWorkflowExecutionRequest) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflow-executions", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WorkflowExecutionResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) ListExecutions(owner, key, status string, limit, offset int) ([]byte, *apipb.ListWorkflowExecutionsResponse, error) {
	query := url.Values{}
	if owner != "" {
		query.Set("owner", owner)
	}
	if key != "" {
		query.Set("workflow_key", key)
	}
	if status != "" {
		query.Set("status", status)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}
	body, err := s.api.Get("/api/v1/workflow-executions", query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ListWorkflowExecutionsResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) ExecutionResult(id string) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	query := url.Values{"explicitly_authorized": {"true"}}
	body, err := s.api.Get("/api/v1/workflow-executions/"+id+"/result", query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WorkflowExecutionResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Signal(req *apipb.SignalWorkflowExecutionRequest) ([]byte, *apipb.WorkflowExecutionOperationResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflow-executions/"+req.ExecutionId+"/signals", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WorkflowExecutionOperationResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Control(operation string, req *apipb.WorkflowExecutionOperationRequest) ([]byte, *apipb.WorkflowExecutionOperationResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflow-executions/"+req.ExecutionId+"/"+operation, nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WorkflowExecutionOperationResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Execution(id string, advance bool) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	path := "/api/v1/workflow-executions/" + id
	method := "GET"
	if advance {
		path += "/advance"
		method = "POST"
	}
	var body []byte
	var err error
	if method == "GET" {
		body, err = s.api.Get(path, nil)
	} else {
		body, err = s.api.Request(method, path, nil, []byte(`{}`))
	}
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WorkflowExecutionResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Wait(id string, timeoutSeconds int) ([]byte, *apipb.WaitWorkflowExecutionResponse, error) {
	payload, err := marshalProtoRequest(&apipb.WaitWorkflowExecutionRequest{ExecutionId: id, TimeoutSeconds: int32(timeoutSeconds)})
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflow-executions/"+id+"/wait", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.WaitWorkflowExecutionResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Trace(id string, after int64, limit int) ([]byte, *apipb.GetWorkflowExecutionTraceResponse, error) {
	query := url.Values{}
	if after > 0 {
		query.Set("after_sequence", strconv.FormatInt(after, 10))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	body, err := s.api.Get("/api/v1/workflow-executions/"+id+"/trace", query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.GetWorkflowExecutionTraceResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) ExecutionRuns(id string) ([]byte, *apipb.ListWorkflowExecutionRunsResponse, error) {
	body, err := s.api.Get("/api/v1/workflow-executions/"+id+"/runs", nil)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ListWorkflowExecutionRunsResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *WorkflowService) Simulate(req *apipb.SimulateWorkflowRequest) ([]byte, *apipb.SimulateWorkflowResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/workflows/simulate", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.SimulateWorkflowResponse
	if unmarshalProtoResponse(body, &resp) != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// =============================================================================
// Profile Service
// =============================================================================

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

// GetReport retrieves the bounded shared run-report projection.
func (s *RunService) GetReport(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/report", nil)
}

// Stats returns the existing filtered run summary projection.
func (s *RunService) Stats(query url.Values) ([]byte, error) {
	return s.api.Get("/api/v1/stats/summary", query)
}

// GetReceipts retrieves platform observations for one run.
func (s *RunService) GetReceipts(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/observed-receipts", nil)
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
func (s *RunService) Stop(id string) ([]byte, *apipb.StopRunResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/stop", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
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
func (s *RunService) StopByTag(tag string) ([]byte, *apipb.StopRunByTagResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/tag/"+tag+"/stop", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopRunByTagResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
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

// Quiesce drains in-flight runs targeting a scenario so a Baseline Modes promote
// can re-point and restart its live instance (Baseline Modes P6).
func (s *RunService) Quiesce(req *apipb.QuiesceScenarioRequest) ([]byte, *apipb.QuiesceResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/quiesce", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.QuiesceScenarioResponse
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
func (s *RunService) GetEvents(id string, limit int, afterSequence *int64) ([]byte, []*domainpb.RunEvent, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if afterSequence != nil {
		query.Set("after_sequence", fmt.Sprintf("%d", *afterSequence))
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

// Park parks a run on externally-owned async work (durable park/resume).
func (s *RunService) Park(id string, req *domainpb.ParkRunRequest) ([]byte, *domainpb.ParkRunResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/park", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.ParkRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// Wake wakes a parked run with a result (ops/manual recovery).
func (s *RunService) Wake(id string, req *domainpb.WakeRunRequest) ([]byte, *domainpb.WakeRunResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/wake", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.WakeRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// AwaitResult fetches a run's most recently resolved await result (the
// non-blocking re-fetch path). Pure read; never parks.
func (s *RunService) AwaitResult(id string) ([]byte, *domainpb.GetAwaitResultResponse, error) {
	body, err := s.api.Request("GET", "/api/v1/runs/"+id+"/await-result", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.GetAwaitResultResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *RunService) Recover(id string) ([]byte, *apipb.RecoverRunResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/recover", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.RecoverRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
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

// CohortReport reads the bounded multi-run projection. The service deliberately
// accepts only explicit run IDs; it never requests a bulk transcript endpoint.
func (s *RunService) CohortReport(runIDs string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/cohort-report", url.Values{"run_ids": []string{runIDs}}, nil)
}

func (s *RunService) InvocationFacts(id string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/"+id+"/invocation-facts", nil, nil)
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

// PolicyService exposes declared catalog inspection and controlled activation.
type PolicyService struct {
	api *cliutil.APIClient
}

func (s *PolicyService) Status() ([]byte, *apipb.GetRolePolicyStatusResponse, error) {
	body, err := s.api.Get("/api/v1/role-policy/status", nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.GetRolePolicyStatusResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PolicyService) Catalog() ([]byte, *apipb.GetRolePolicyCatalogResponse, error) {
	body, err := s.api.Get("/api/v1/role-policy/catalog", nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.GetRolePolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PolicyService) Validate() ([]byte, *apipb.ValidateRolePolicyCatalogResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/role-policy/validate", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ValidateRolePolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PolicyService) Reload() ([]byte, *apipb.ReloadRolePolicyCatalogResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/role-policy/reload", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ReloadRolePolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PolicyService) Explain(req *apipb.ExplainRolePolicyRequest) ([]byte, *apipb.ExplainRolePolicyResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/role-policy/explain", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ExplainRolePolicyResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

// PermissionPolicyService exposes the global desired-permission control plane.
// The server delegates all native projection to the owning resource CLIs.
type PermissionPolicyService struct {
	api *cliutil.APIClient
}

func (s *PermissionPolicyService) Status() ([]byte, *apipb.GetPermissionPolicyStatusResponse, error) {
	body, err := s.api.Get("/api/v1/permission-policy/status", nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.GetPermissionPolicyStatusResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Catalog() ([]byte, *apipb.GetPermissionPolicyCatalogResponse, error) {
	body, err := s.api.Get("/api/v1/permission-policy/catalog", nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.GetPermissionPolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Validate() ([]byte, *apipb.ValidatePermissionPolicyCatalogResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/permission-policy/validate", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ValidatePermissionPolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Reload() ([]byte, *apipb.ReloadPermissionPolicyCatalogResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/permission-policy/reload", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ReloadPermissionPolicyCatalogResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Plan() ([]byte, *apipb.PlanPermissionPolicyResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/permission-policy/plan", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.PlanPermissionPolicyResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Reconcile(req *apipb.ReconcilePermissionPolicyRequest) ([]byte, *apipb.ReconcilePermissionPolicyResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/permission-policy/reconcile", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var response apipb.ReconcilePermissionPolicyResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
}

func (s *PermissionPolicyService) Doctor() ([]byte, *apipb.DoctorPermissionPolicyResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/permission-policy/doctor", nil, nil)
	if err != nil {
		return body, nil, err
	}
	var response apipb.DoctorPermissionPolicyResponse
	if err := unmarshalProtoResponse(body, &response); err != nil {
		return body, nil, err
	}
	return body, &response, nil
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
