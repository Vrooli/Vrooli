// Responsibility: retain services declarations within their original package.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"strings"
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
	return typedRPC[apipb.ListExecutionOptionsResponse](s.api, apiconnect.AgentManagerServiceListExecutionOptionsProcedure, &apipb.ListExecutionOptionsRequest{RoleRef: role})
}

// Probe sends a test request to verify a runner can respond. The runner is
// named by its slug ("claude-code").
func (s *RunnerService) Probe(runnerType string) ([]byte, *domainpb.ProbeResult, error) {
	value, ok := domainpb.RunnerType_value["RUNNER_TYPE_"+strings.ToUpper(strings.ReplaceAll(runnerType, "-", "_"))]
	if !ok || value == 0 {
		return nil, nil, fmt.Errorf("unknown runner type %q", runnerType)
	}
	body, response, err := lenientRPC[apipb.ProbeRunnerResponse](s.api, apiconnect.AgentManagerServiceProbeRunnerProcedure, &apipb.ProbeRunnerRequest{RunnerType: domainpb.RunnerType(value)})
	if response == nil {
		return body, nil, err
	}
	return body, response.Result, nil
}

// PolicyService exposes declared catalog inspection and controlled activation.
type PolicyService struct {
	api *cliutil.APIClient
}

func (s *PolicyService) Status() ([]byte, *apipb.GetRolePolicyStatusResponse, error) {
	return typedRPC[apipb.GetRolePolicyStatusResponse](s.api, apiconnect.AgentManagerServiceGetRolePolicyStatusProcedure, &apipb.GetRolePolicyStatusRequest{})
}

func (s *PolicyService) Catalog() ([]byte, *apipb.GetRolePolicyCatalogResponse, error) {
	return typedRPC[apipb.GetRolePolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceGetRolePolicyCatalogProcedure, &apipb.GetRolePolicyCatalogRequest{})
}

func (s *PolicyService) Validate() ([]byte, *apipb.ValidateRolePolicyCatalogResponse, error) {
	return typedRPC[apipb.ValidateRolePolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceValidateRolePolicyCatalogProcedure, &apipb.ValidateRolePolicyCatalogRequest{})
}

func (s *PolicyService) Reload() ([]byte, *apipb.ReloadRolePolicyCatalogResponse, error) {
	return typedRPC[apipb.ReloadRolePolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceReloadRolePolicyCatalogProcedure, &apipb.ReloadRolePolicyCatalogRequest{})
}

func (s *PolicyService) Explain(req *apipb.ExplainRolePolicyRequest) ([]byte, *apipb.ExplainRolePolicyResponse, error) {
	return typedRPC[apipb.ExplainRolePolicyResponse](s.api, apiconnect.AgentManagerServiceExplainRolePolicyProcedure, req)
}

// PermissionPolicyService exposes the global desired-permission control plane.
// The server delegates all native projection to the owning resource CLIs.
type PermissionPolicyService struct {
	api *cliutil.APIClient
}

func (s *PermissionPolicyService) Status() ([]byte, *apipb.GetPermissionPolicyStatusResponse, error) {
	return typedRPC[apipb.GetPermissionPolicyStatusResponse](s.api, apiconnect.AgentManagerServiceGetPermissionPolicyStatusProcedure, &apipb.GetPermissionPolicyStatusRequest{})
}

func (s *PermissionPolicyService) Catalog() ([]byte, *apipb.GetPermissionPolicyCatalogResponse, error) {
	return typedRPC[apipb.GetPermissionPolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceGetPermissionPolicyCatalogProcedure, &apipb.GetPermissionPolicyCatalogRequest{})
}

func (s *PermissionPolicyService) Validate() ([]byte, *apipb.ValidatePermissionPolicyCatalogResponse, error) {
	return typedRPC[apipb.ValidatePermissionPolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceValidatePermissionPolicyCatalogProcedure, &apipb.ValidatePermissionPolicyCatalogRequest{})
}

func (s *PermissionPolicyService) Reload() ([]byte, *apipb.ReloadPermissionPolicyCatalogResponse, error) {
	return typedRPC[apipb.ReloadPermissionPolicyCatalogResponse](s.api, apiconnect.AgentManagerServiceReloadPermissionPolicyCatalogProcedure, &apipb.ReloadPermissionPolicyCatalogRequest{})
}

func (s *PermissionPolicyService) Plan() ([]byte, *apipb.PlanPermissionPolicyResponse, error) {
	return typedRPC[apipb.PlanPermissionPolicyResponse](s.api, apiconnect.AgentManagerServicePlanPermissionPolicyProcedure, &apipb.PlanPermissionPolicyRequest{})
}

func (s *PermissionPolicyService) Reconcile(req *apipb.ReconcilePermissionPolicyRequest) ([]byte, *apipb.ReconcilePermissionPolicyResponse, error) {
	return typedRPC[apipb.ReconcilePermissionPolicyResponse](s.api, apiconnect.AgentManagerServiceReconcilePermissionPolicyProcedure, req)
}

func (s *PermissionPolicyService) Doctor() ([]byte, *apipb.DoctorPermissionPolicyResponse, error) {
	return typedRPC[apipb.DoctorPermissionPolicyResponse](s.api, apiconnect.AgentManagerServiceDoctorPermissionPolicyProcedure, &apipb.DoctorPermissionPolicyRequest{})
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
	return lenientRPC[apipb.PurgeDataResponse](s.api, apiconnect.AgentManagerServicePurgeDataProcedure, req)
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
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceCreateSupervisionPolicyCandidateProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyEvaluate(request *domainpb.EvaluateSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionReplayReport, error) {
	response := &domainpb.SupervisionReplayReport{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceEvaluateSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyAssess(request *domainpb.RecordSupervisionOutcomeRequest) ([]byte, *domainpb.RecordSupervisionOutcomeResponse, error) {
	response := &domainpb.RecordSupervisionOutcomeResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceRecordSupervisionOutcomeProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyPromote(request *domainpb.PromoteSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServicePromoteSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyReject(request *domainpb.RejectSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceRejectSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyRollback(request *domainpb.RollbackSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceRollbackSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) PolicyDisable(request *domainpb.SetSupervisionPolicyDisabledRequest) ([]byte, *domainpb.SupervisionPolicyControl, error) {
	response := &domainpb.SupervisionPolicyControl{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceSetSupervisionPolicyDisabledProcedure, request, response)
	return body, response, err
}
