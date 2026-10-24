// Responsibility: retain services declarations within their original package.
package main

import (
	"encoding/json"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"net/url"
)

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

func (s *RunnerService) ExecutionOptions(role string) ([]byte, *apipb.ListExecutionOptionsResponse, error) {
	path := "/api/v1/execution-options"
	if role != "" {
		path += "?role=" + url.QueryEscape(role)
	}
	body, err := s.api.Get(path, nil)
	if err != nil {
		return body, nil, err
	}
	response := &apipb.ListExecutionOptionsResponse{}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, nil, err
	}
	return body, response, nil
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

func (s *WatchService) PolicyCandidate(request *domainpb.CreateSupervisionPolicyCandidateRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceCreateSupervisionPolicyCandidateProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyEvaluate(request *domainpb.EvaluateSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionReplayReport, error) {
	response := &domainpb.SupervisionReplayReport{}
	body, err := s.call(apiconnect.AgentManagerServiceEvaluateSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyAssess(request *domainpb.RecordSupervisionOutcomeRequest) ([]byte, *domainpb.RecordSupervisionOutcomeResponse, error) {
	response := &domainpb.RecordSupervisionOutcomeResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceRecordSupervisionOutcomeProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyPromote(request *domainpb.PromoteSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := s.call(apiconnect.AgentManagerServicePromoteSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyReject(request *domainpb.RejectSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceRejectSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyRollback(request *domainpb.RollbackSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceRollbackSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyDisable(request *domainpb.SetSupervisionPolicyDisabledRequest) ([]byte, *domainpb.SupervisionPolicyControl, error) {
	response := &domainpb.SupervisionPolicyControl{}
	body, err := s.call(apiconnect.AgentManagerServiceSetSupervisionPolicyDisabledProcedure, request, response)
	return body, response, err
}
