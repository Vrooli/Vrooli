// Responsibility: retain services declarations within their original package.
package main

import (
	"errors"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	"google.golang.org/protobuf/proto"
	"net/url"
	"strconv"
	"strings"
)

type WatchService struct{ api *cliutil.APIClient }

// callRPC invokes one AgentManagerService (or sibling) procedure as a JSON
// Connect unary call and decodes its typed response. Every CLI operation that
// has an RPC goes through it; the generated procedure constant carries the path.
func callRPC(api *cliutil.APIClient, procedure string, request, response proto.Message) ([]byte, error) {
	payload, err := marshalProtoRequest(request)
	if err != nil {
		return nil, err
	}
	body, err := api.Request("POST", procedure, nil, payload)
	if err != nil {
		return body, err
	}
	if err := unmarshalProtoResponse(body, response); err != nil {
		return body, fmt.Errorf("%w %s: %v", errDecodeResponse, procedure, err)
	}
	return body, nil
}

// errDecodeResponse marks a call that succeeded but returned a body the typed
// response cannot decode; callers that can print raw JSON fall back to it.
var errDecodeResponse = errors.New("decode response")

type InvestigationService struct{ api *cliutil.APIClient }

func (s *InvestigationService) Start(request *apipb.StartInvestigationRequest) ([]byte, *apipb.StartInvestigationResponse, error) {
	response := &apipb.StartInvestigationResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceStartInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Get(request *apipb.GetInvestigationRequest) ([]byte, *apipb.InvestigationRecord, error) {
	response := &apipb.InvestigationRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceGetInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) List(request *apipb.ListInvestigationsRequest) ([]byte, *apipb.ListInvestigationsResponse, error) {
	response := &apipb.ListInvestigationsResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceListInvestigationsProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Wait(request *apipb.WaitInvestigationRequest) ([]byte, *apipb.WaitInvestigationResponse, error) {
	response := &apipb.WaitInvestigationResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceWaitInvestigationProcedure, request, response)
	return body, response, err
}

func (s *InvestigationService) Cancel(request *apipb.CancelInvestigationRequest) ([]byte, *apipb.InvestigationRecord, error) {
	response := &apipb.InvestigationRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceCancelInvestigationProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Create(request *domainpb.CreateCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceCreateCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Get(request *domainpb.GetCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceGetCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) List(request *domainpb.ListCohortWatchesRequest) ([]byte, *domainpb.ListCohortWatchesResponse, error) {
	response := &domainpb.ListCohortWatchesResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceListCohortWatchesProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Wait(request *domainpb.WaitCohortWatchRequest) ([]byte, *domainpb.WaitCohortWatchResponse, error) {
	response := &domainpb.WaitCohortWatchResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceWaitCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Cancel(request *domainpb.CancelCohortWatchRequest) ([]byte, *domainpb.CohortWatch, error) {
	response := &domainpb.CohortWatch{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceCancelCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) Inspect(request *domainpb.InspectCohortWatchRequest) ([]byte, *domainpb.InspectCohortWatchResponse, error) {
	response := &domainpb.InspectCohortWatchResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceInspectCohortWatchProcedure, request, response)
	return body, response, err
}

func (s *WatchService) RequestAction(request *domainpb.RequestCohortWatchActionRequest) ([]byte, *domainpb.RequestCohortWatchActionResponse, error) {
	response := &domainpb.RequestCohortWatchActionResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceRequestCohortWatchActionProcedure, request, response)
	return body, response, err
}

func (s *WatchService) ListActions(request *domainpb.ListCohortWatchActionsRequest) ([]byte, *domainpb.ListCohortWatchActionsResponse, error) {
	response := &domainpb.ListCohortWatchActionsResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceListCohortWatchActionsProcedure, request, response)
	return body, response, err
}

func (s *WatchService) GetPolicy(request *domainpb.GetSupervisionPolicyRequest) ([]byte, *domainpb.SupervisionPolicyRecord, error) {
	response := &domainpb.SupervisionPolicyRecord{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceGetSupervisionPolicyProcedure, request, response)
	return body, response, err
}

func (s *WatchService) ListOutcomes(request *domainpb.ListSupervisionOutcomesRequest) ([]byte, *domainpb.ListSupervisionOutcomesResponse, error) {
	response := &domainpb.ListSupervisionOutcomesResponse{}
	body, err := callRPC(s.api, apiconnect.AgentManagerServiceListSupervisionOutcomesProcedure, request, response)
	return body, response, err
}

// ConversationSearchService keeps the CLI on the generated protobuf contract
// while using the same scenario-aware HTTP transport as the older commands.
type ConversationSearchService struct{ api *cliutil.APIClient }

func (s *ConversationSearchService) Search(request *domainpb.SearchConversationsRequest) (*domainpb.SearchConversationsResponse, error) {
	response := &domainpb.SearchConversationsResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchServiceSearchConversationsProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Context(request *domainpb.GetConversationContextRequest) (*domainpb.GetConversationContextResponse, error) {
	response := &domainpb.GetConversationContextResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchServiceGetConversationContextProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Status() (*domainpb.GetConversationIndexStatusResponse, error) {
	response := &domainpb.GetConversationIndexStatusResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchServiceGetConversationIndexStatusProcedure, &domainpb.GetConversationIndexStatusRequest{}, response)
	return response, err
}

func (s *ConversationSearchService) PlanReindex(request *domainpb.PlanConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchControlServicePlanConversationReindexProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) Reindex(request *domainpb.ReindexConversationsRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchControlServiceReindexConversationsProcedure, request, response)
	return response, err
}

func (s *ConversationSearchService) CancelReindex(request *domainpb.CancelConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	response := &domainpb.ConversationReindexResponse{}
	_, err := callRPC(s.api, domainconnect.ConversationSearchControlServiceCancelConversationReindexProcedure, request, response)
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

// typedRPC calls one procedure and returns the typed response with its JSON in
// proto field names, the shape these operations printed over REST.
func typedRPC[R any, PR interface {
	*R
	proto.Message
}](api *cliutil.APIClient, procedure string, request proto.Message) ([]byte, PR, error) {
	response := PR(new(R))
	body, err := callRPC(api, procedure, request, response)
	if err != nil {
		return body, nil, err
	}
	body, err = marshalProtoRequest(response)
	return body, response, err
}

// lenientRPC is typedRPC for commands that print the raw JSON body when it
// does not decode: a decode failure yields a nil response and no error.
func lenientRPC[R any, PR interface {
	*R
	proto.Message
}](api *cliutil.APIClient, procedure string, request proto.Message) ([]byte, PR, error) {
	body, response, err := typedRPC[R, PR](api, procedure, request)
	if errors.Is(err, errDecodeResponse) {
		return body, nil, nil
	}
	if err != nil {
		return body, nil, err
	}
	return body, response, nil
}

func (s *WorkflowService) Validate(req *apipb.ValidateWorkflowRequest) ([]byte, *apipb.ValidateWorkflowResponse, error) {
	return lenientRPC[apipb.ValidateWorkflowResponse](s.api, apiconnect.AgentManagerServiceValidateWorkflowProcedure, req)
}

func (s *WorkflowService) Reconcile(procedure string, req *apipb.ReconcileScenarioWorkflowsRequest) ([]byte, *apipb.ReconcileScenarioWorkflowsResponse, error) {
	return lenientRPC[apipb.ReconcileScenarioWorkflowsResponse](s.api, procedure, req)
}

func (s *WorkflowService) List(owner, key string, limit, offset int) ([]byte, *apipb.ListWorkflowRevisionsResponse, error) {
	return lenientRPC[apipb.ListWorkflowRevisionsResponse](s.api, apiconnect.AgentManagerServiceListWorkflowRevisionsProcedure, &apipb.ListWorkflowRevisionsRequest{Owner: owner, Key: key, Limit: int32(limit), Offset: int32(offset)})
}

func (s *WorkflowService) Get(procedure, owner, key, digest string) ([]byte, *apipb.GetWorkflowRevisionResponse, error) {
	return lenientRPC[apipb.GetWorkflowRevisionResponse](s.api, procedure, &apipb.GetWorkflowRevisionRequest{Owner: owner, Key: key, Digest: digest})
}

func (s *WorkflowService) StartExecution(req *apipb.StartWorkflowExecutionRequest) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	return lenientRPC[apipb.WorkflowExecutionResponse](s.api, apiconnect.AgentManagerServiceStartWorkflowExecutionProcedure, req)
}

func (s *WorkflowService) ListExecutions(owner, key, status string, limit, offset int) ([]byte, *apipb.ListWorkflowExecutionsResponse, error) {
	return lenientRPC[apipb.ListWorkflowExecutionsResponse](s.api, apiconnect.AgentManagerServiceListWorkflowExecutionsProcedure, &apipb.ListWorkflowExecutionsRequest{
		Owner: owner, WorkflowKey: key, Status: domainpb.WorkflowExecutionStatus(domainpb.WorkflowExecutionStatus_value["WORKFLOW_EXECUTION_STATUS_"+strings.ToUpper(status)]),
		Limit: int32(limit), Offset: int32(offset),
	})
}

func (s *WorkflowService) ExecutionResult(id string) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	return lenientRPC[apipb.WorkflowExecutionResponse](s.api, apiconnect.AgentManagerServiceGetWorkflowExecutionResultProcedure, &apipb.GetWorkflowExecutionResultRequest{ExecutionId: id, ExplicitlyAuthorized: true})
}

func (s *WorkflowService) Signal(req *apipb.SignalWorkflowExecutionRequest) ([]byte, *apipb.WorkflowExecutionOperationResponse, error) {
	return lenientRPC[apipb.WorkflowExecutionOperationResponse](s.api, apiconnect.AgentManagerServiceSignalWorkflowExecutionProcedure, req)
}

// Control sends cancel, retry or resume for one execution.
func (s *WorkflowService) Control(operation string, req *apipb.WorkflowExecutionOperationRequest) ([]byte, *apipb.WorkflowExecutionOperationResponse, error) {
	procedures := map[string]string{
		"cancel": apiconnect.AgentManagerServiceCancelWorkflowExecutionProcedure,
		"retry":  apiconnect.AgentManagerServiceRetryWorkflowExecutionProcedure,
		"resume": apiconnect.AgentManagerServiceResumeWorkflowExecutionProcedure,
	}
	return lenientRPC[apipb.WorkflowExecutionOperationResponse](s.api, procedures[operation], req)
}

func (s *WorkflowService) Execution(id string, advance bool) ([]byte, *apipb.WorkflowExecutionResponse, error) {
	procedure := apiconnect.AgentManagerServiceGetWorkflowExecutionProcedure
	if advance {
		procedure = apiconnect.AgentManagerServiceAdvanceWorkflowExecutionProcedure
	}
	return lenientRPC[apipb.WorkflowExecutionResponse](s.api, procedure, &apipb.GetWorkflowExecutionRequest{ExecutionId: id})
}

func (s *WorkflowService) Wait(id string, timeoutSeconds int) ([]byte, *apipb.WaitWorkflowExecutionResponse, error) {
	// This endpoint owns both finite and indefinite waits. An ordinary CLI
	// transport deadline must not detach before the requested server bound.
	return lenientRPC[apipb.WaitWorkflowExecutionResponse](s.api.WithoutTimeout(), apiconnect.AgentManagerServiceWaitWorkflowExecutionProcedure, &apipb.WaitWorkflowExecutionRequest{ExecutionId: id, TimeoutSeconds: int32(timeoutSeconds)})
}

func (s *WorkflowService) Trace(id string, after int64, limit int) ([]byte, *apipb.GetWorkflowExecutionTraceResponse, error) {
	return lenientRPC[apipb.GetWorkflowExecutionTraceResponse](s.api, apiconnect.AgentManagerServiceGetWorkflowExecutionTraceProcedure, &apipb.GetWorkflowExecutionTraceRequest{ExecutionId: id, AfterSequence: after, Limit: int32(limit)})
}

func (s *WorkflowService) ExecutionRuns(id string) ([]byte, *apipb.ListWorkflowExecutionRunsResponse, error) {
	return lenientRPC[apipb.ListWorkflowExecutionRunsResponse](s.api, apiconnect.AgentManagerServiceListWorkflowExecutionRunsProcedure, &apipb.ListWorkflowExecutionRunsRequest{ExecutionId: id})
}

func (s *WorkflowService) Simulate(req *apipb.SimulateWorkflowRequest) ([]byte, *apipb.SimulateWorkflowResponse, error) {
	return lenientRPC[apipb.SimulateWorkflowResponse](s.api, apiconnect.AgentManagerServiceSimulateWorkflowProcedure, req)
}

// =============================================================================
// Profile Service
// =============================================================================
