package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Services Aggregator
// =============================================================================

// Services aggregates all domain-specific services.
type Services struct {
	Profiles *ProfileService
	Tasks    *TaskService
	Runs     *RunService
}

// NewServices creates a new Services instance with all domain services.
func NewServices(api *cliutil.APIClient) *Services {
	return &Services{
		Profiles: &ProfileService{api: api},
		Tasks:    &TaskService{api: api},
		Runs:     &RunService{api: api},
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
func (s *ProfileService) List(limit, offset int) ([]byte, []Profile, error) {
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

	var profiles []Profile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return body, nil, nil // Return raw body for fallback
	}
	return body, profiles, nil
}

// Get retrieves a single profile by ID.
func (s *ProfileService) Get(id string) ([]byte, *Profile, error) {
	body, err := s.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return body, nil, nil
	}
	return body, &profile, nil
}

// Create creates a new profile.
func (s *ProfileService) Create(req CreateProfileRequest) ([]byte, *Profile, error) {
	body, err := s.api.Request("POST", "/api/v1/profiles", nil, req)
	if err != nil {
		return body, nil, err
	}

	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return body, nil, nil
	}
	return body, &profile, nil
}

// Update updates an existing profile.
func (s *ProfileService) Update(id string, req CreateProfileRequest) ([]byte, *Profile, error) {
	body, err := s.api.Request("PUT", "/api/v1/profiles/"+id, nil, req)
	if err != nil {
		return body, nil, err
	}

	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return body, nil, nil
	}
	return body, &profile, nil
}

// Delete removes a profile.
func (s *ProfileService) Delete(id string) error {
	_, err := s.api.Request("DELETE", "/api/v1/profiles/"+id, nil, nil)
	return err
}

// =============================================================================
// Task Service
// =============================================================================

// TaskService handles task-related API operations.
type TaskService struct {
	api *cliutil.APIClient
}

// List retrieves all tasks.
func (s *TaskService) List(limit, offset int, status string) ([]byte, []Task, error) {
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

	var tasks []Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return body, nil, nil
	}
	return body, tasks, nil
}

// Get retrieves a single task by ID.
func (s *TaskService) Get(id string) ([]byte, *Task, error) {
	body, err := s.api.Get("/api/v1/tasks/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return body, nil, nil
	}
	return body, &task, nil
}

// Create creates a new task.
func (s *TaskService) Create(req CreateTaskRequest) ([]byte, *Task, error) {
	body, err := s.api.Request("POST", "/api/v1/tasks", nil, req)
	if err != nil {
		return body, nil, err
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return body, nil, nil
	}
	return body, &task, nil
}

// Cancel cancels a task.
func (s *TaskService) Cancel(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/tasks/"+id+"/cancel", nil, nil)
}

// =============================================================================
// Run Service
// =============================================================================

// RunService handles run-related API operations.
type RunService struct {
	api *cliutil.APIClient
}

// List retrieves runs with optional filters.
func (s *RunService) List(limit, offset int, taskID, profileID, status, tagPrefix string) ([]byte, []Run, error) {
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

	var runs []Run
	if err := json.Unmarshal(body, &runs); err != nil {
		return body, nil, nil
	}
	return body, runs, nil
}

// Get retrieves a single run by ID.
func (s *RunService) Get(id string) ([]byte, *Run, error) {
	body, err := s.api.Get("/api/v1/runs/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var run Run
	if err := json.Unmarshal(body, &run); err != nil {
		return body, nil, nil
	}
	return body, &run, nil
}

// Create creates a new run.
func (s *RunService) Create(req CreateRunRequest) ([]byte, *Run, error) {
	body, err := s.api.Request("POST", "/api/v1/runs", nil, req)
	if err != nil {
		return body, nil, err
	}

	var run Run
	if err := json.Unmarshal(body, &run); err != nil {
		return body, nil, nil
	}
	return body, &run, nil
}

// Stop stops a running execution.
func (s *RunService) Stop(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/stop", nil, nil)
}

// GetByTag retrieves a run by its custom tag.
func (s *RunService) GetByTag(tag string) ([]byte, *Run, error) {
	body, err := s.api.Get("/api/v1/runs/tag/"+tag, nil)
	if err != nil {
		return body, nil, err
	}

	var run Run
	if err := json.Unmarshal(body, &run); err != nil {
		return body, nil, nil
	}
	return body, &run, nil
}

// StopByTag stops a run identified by its custom tag.
func (s *RunService) StopByTag(tag string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/tag/"+tag+"/stop", nil, nil)
}

// StopAllRequest is the request body for the stop-all endpoint.
type StopAllRequest struct {
	TagPrefix string `json:"tagPrefix,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

// StopAllResult is the result of a bulk stop operation.
type StopAllResult struct {
	Stopped   int      `json:"stopped"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	FailedIDs []string `json:"failedIds"`
}

// StopAll stops all running runs, optionally filtered by tag prefix.
func (s *RunService) StopAll(req StopAllRequest) ([]byte, *StopAllResult, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/stop-all", nil, req)
	if err != nil {
		return body, nil, err
	}

	var result StopAllResult
	if err := json.Unmarshal(body, &result); err != nil {
		return body, nil, nil
	}
	return body, &result, nil
}

// Approve approves a run.
func (s *RunService) Approve(id string, req ApproveRequest) ([]byte, *ApproveResult, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/approve", nil, req)
	if err != nil {
		return body, nil, err
	}

	var result ApproveResult
	if err := json.Unmarshal(body, &result); err != nil {
		return body, nil, nil
	}
	return body, &result, nil
}

// Reject rejects a run.
func (s *RunService) Reject(id string, req RejectRequest) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/reject", nil, req)
}

// GetDiff retrieves the diff for a run.
func (s *RunService) GetDiff(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/diff", nil)
}

// GetEvents retrieves events for a run.
func (s *RunService) GetEvents(id string, limit int) ([]byte, []RunEvent, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	body, err := s.api.Get("/api/v1/runs/"+id+"/events", query)
	if err != nil {
		return body, nil, err
	}

	var events []RunEvent
	if err := json.Unmarshal(body, &events); err != nil {
		return body, nil, nil
	}
	return body, events, nil
}
