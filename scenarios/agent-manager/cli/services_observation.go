// Responsibility: retain services declarations within their original package.
package main

import (
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	"google.golang.org/protobuf/proto"
	"net/url"
	"strconv"
)

type WatchService struct{ api *cliutil.APIClient }

type InvestigationService struct{ api *cliutil.APIClient }

func (s *InvestigationService) call(path string, request, response proto.Message) ([]byte, error) {
	payload, err := marshalProtoRequest(request)
	if err != nil {
		return nil, err
	}
	body, err := s.api.Request("POST", path, nil, payload)
	if err != nil {
		return body, err
	}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, fmt.Errorf("decode investigation response: %w", err)
	}
	return body, nil
}

func (s *InvestigationService) Start(request *apipb.StartInvestigationRequest) ([]byte, *apipb.StartInvestigationResponse, error) {
	response := &apipb.StartInvestigationResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceStartInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Get(request *apipb.GetInvestigationRequest) ([]byte, *apipb.InvestigationRecord, error) {
	response := &apipb.InvestigationRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceGetInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) List(request *apipb.ListInvestigationsRequest) ([]byte, *apipb.ListInvestigationsResponse, error) {
	response := &apipb.ListInvestigationsResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceListInvestigationsProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Wait(request *apipb.WaitInvestigationRequest) ([]byte, *apipb.WaitInvestigationResponse, error) {
	response := &apipb.WaitInvestigationResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceWaitInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Cancel(request *apipb.CancelInvestigationRequest) ([]byte, *apipb.InvestigationRecord, error) {
	response := &apipb.InvestigationRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceCancelInvestigationProcedure, request, response)
	return body, response, err
}

func (s *WatchService) call(path string, request, response proto.Message) ([]byte, error) {
	payload, err := marshalProtoRequest(request)
	if err != nil {
		return nil, err
	}
	body, err := s.api.Request("POST", path, nil, payload)
	if err != nil {
		return body, err
	}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, fmt.Errorf("decode cohort watch response: %w", err)
	}
	return body, nil
}

func (s *WatchService) Create(request *domainpb.CreateCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := s.call(apiconnect.AgentManagerServiceCreateCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Get(request *domainpb.GetCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := s.call(apiconnect.AgentManagerServiceGetCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) List(request *domainpb.ListCohortWatchesRequest) ([]byte, *domainpb.ListCohortWatchesResponse, error) {
	response := &domainpb.ListCohortWatchesResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceListCohortWatchesProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Wait(request *domainpb.WaitCohortWatchRequest) ([]byte, *domainpb.WaitCohortWatchResponse, error) {
	response := &domainpb.WaitCohortWatchResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceWaitCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Cancel(request *domainpb.CancelCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := s.call(apiconnect.AgentManagerServiceCancelCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Inspect(request *domainpb.InspectCohortWatchRequest) ([]byte, *domainpb.InspectCohortWatchResponse, error) {
	response := &domainpb.InspectCohortWatchResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceInspectCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) RequestAction(request *domainpb.RequestCohortWatchActionRequest) ([]byte, *domainpb.RequestCohortWatchActionResponse, error) {
	response := &domainpb.RequestCohortWatchActionResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceRequestCohortWatchActionProcedure, request, response)
	return body, response, err
}

func (s *WatchService) ListActions(request *domainpb.ListCohortWatchActionsRequest) ([]byte, *domainpb.ListCohortWatchActionsResponse, error) {
	response := &domainpb.ListCohortWatchActionsResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceListCohortWatchActionsProcedure, request, response)
	return body, response, err
}

func (s *WatchService) GetPolicy(request *domainpb.GetSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := s.call(apiconnect.AgentManagerServiceGetSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) ListOutcomes(request *domainpb.ListSupervisionOutcomesRequest) ([]byte, *domainpb.ListSupervisionOutcomesResponse, error) {
	response := &domainpb.ListSupervisionOutcomesResponse{}
	body, err := s.call(apiconnect.AgentManagerServiceListSupervisionOutcomesProcedure, request, response)
	return body, response, err
}

// ConversationSearchService keeps the CLI on the generated protobuf contract
// while using the same scenario-aware HTTP transport as the older commands.
type ConversationSearchService struct{ api *cliutil.APIClient }

func (s *ConversationSearchService) call(path string, request, response proto.Message) ([]byte, error) {
	payload, err := marshalProtoRequest(request)
	if err != nil {
		return nil, err
	}
	body, err := s.api.Request("POST", path, nil, payload)
	if err != nil {
		return body, err
	}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, fmt.Errorf("decode conversation search response: %w", err)
	}
	return body, nil
}

func (s *ConversationSearchService) Search(request *domainpb.SearchConversationsRequest) (*domainpb.SearchConversationsResponse, error) {
	response := &domainpb.SearchConversationsResponse{}
	_, err := s.call(domainconnect.ConversationSearchServiceSearchConversationsProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Context(request *domainpb.GetConversationContextRequest) (*domainpb.GetConversationContextResponse, error) {
	response := &domainpb.GetConversationContextResponse{}
	_, err := s.call(domainconnect.ConversationSearchServiceGetConversationContextProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Status() (*domainpb.GetConversationIndexStatusResponse, error) {
	response := &domainpb.GetConversationIndexStatusResponse{}
	_, err := s.call(domainconnect.ConversationSearchServiceGetConversationIndexStatusProcedure, &domainpb.GetConversationIndexStatusRequest{}, response)
	return response, err
}

func (s *ConversationSearchService) PlanReindex(request *domainpb.PlanConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := s.call(domainconnect.ConversationSearchControlServicePlanConversationReindexProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Reindex(request *domainpb.ReindexConversationsRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := s.call(domainconnect.ConversationSearchControlServiceReindexConversationsProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) CancelReindex(request *domainpb.CancelConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := s.call(domainconnect.ConversationSearchControlServiceCancelConversationReindexProcedure, request, response)
	return response, err
}

type SubscriptionService struct{ api *cliutil.APIClient }

func (s *SubscriptionService) List(provider, planRef string) ([]byte, error) {
	query := url.Values{}
	if provider != "" {
		query.Set("provider", provider)
	}
	if planRef != "" {
		query.Set("plan_ref", planRef)
	}
	return s.api.Get("/api/v1/pricing/subscription-periods", query)
}

func (s *SubscriptionService) Create(payload []byte) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/pricing/subscription-periods", nil, payload)
}

func (s *SubscriptionService) Remove(id string) ([]byte, error) {
	return s.api.Request("DELETE", "/api/v1/pricing/subscription-periods/"+url.PathEscape(id), nil, nil)
}

func (s *SubscriptionService) QuotaObservations(provider, pool, window string, limit int) ([]byte, error) {
	query := url.Values{}
	if provider != "" {
		query.Set("provider", provider)
	}
	if pool != "" {
		query.Set("pool", pool)
	}
	if window != "" {
		query.Set("window", window)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	return s.api.Get("/api/v1/pricing/quota-observations", query)
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
	// This endpoint owns both finite and indefinite waits. An ordinary CLI
	// transport deadline must not detach before the requested server bound.
	body, err := s.api.WithoutTimeout().Request("POST", "/api/v1/workflow-executions/"+id+"/wait", nil, payload)
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
