package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"agent-manager/internal/conversationsearch"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	"github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConversationSearchControlOptions struct {
	Indexer        *conversationsearch.Indexer
	SearchFilePath string
	ControlToken   func() string
}

// ConversationSearchSharedControl implements Search Hub's single provider
// control contract. Every method fails closed until registration supplies a
// non-empty token; public search never passes through this handler.
type ConversationSearchSharedControl struct {
	control_v1connect.UnimplementedSearchControlServiceHandler
	options ConversationSearchControlOptions
}

func NewConversationSearchSharedControl(options ConversationSearchControlOptions) *ConversationSearchSharedControl {
	return &ConversationSearchSharedControl{options: options}
}

func (h *ConversationSearchSharedControl) authorize(presented string) error {
	want := ""
	if h != nil && h.options.ControlToken != nil {
		want = h.options.ControlToken()
	}
	if want == "" || presented == "" || len(want) != len(presented) || subtle.ConstantTimeCompare([]byte(want), []byte(presented)) != 1 {
		return connect.NewError(connect.CodePermissionDenied, errors.New("conversation search control is unavailable"))
	}
	return nil
}

// ConversationSearchDirectControl is Agent Manager's operator-facing control
// adapter. It deliberately shares the same options and token policy as the
// Search Hub provider adapter, but keeps the direct CLI on Agent Manager's
// generated domain contract.
type ConversationSearchDirectControl struct {
	options ConversationSearchControlOptions
}

func NewConversationSearchDirectControl(options ConversationSearchControlOptions) *ConversationSearchDirectControl {
	return &ConversationSearchDirectControl{options: options}
}

func (h *ConversationSearchDirectControl) authorize(presented string) error {
	return (&ConversationSearchSharedControl{options: h.options}).authorize(presented)
}

func (h *ConversationSearchDirectControl) PlanConversationReindex(ctx context.Context, request *domainpb.PlanConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	if err := h.authorize(request.GetControlToken()); err != nil {
		return nil, err
	}
	if h.options.Indexer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation indexer is unavailable"))
	}
	if !request.GetFull() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operator reindex currently requires full=true"))
	}
	job, err := h.options.Indexer.Plan(ctx, request.GetMaxDocuments())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return directReindexResponse(job), nil
}

func (h *ConversationSearchDirectControl) ReindexConversations(ctx context.Context, request *domainpb.ReindexConversationsRequest) (*domainpb.ConversationReindexResponse, error) {
	if err := h.authorize(request.GetControlToken()); err != nil {
		return nil, err
	}
	if h.options.Indexer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation indexer is unavailable"))
	}
	if !request.GetFull() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("operator reindex currently requires full=true"))
	}
	job, err := h.options.Indexer.Reindex(ctx, request.GetMaxDocuments(), request.GetIdempotencyKey(), false)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return directReindexResponse(job), nil
}

func (h *ConversationSearchDirectControl) CancelConversationReindex(_ context.Context, request *domainpb.CancelConversationReindexRequest) (*domainpb.ConversationReindexResponse, error) {
	if err := h.authorize(request.GetControlToken()); err != nil {
		return nil, err
	}
	if h.options.Indexer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation indexer is unavailable"))
	}
	if _, found := h.options.Indexer.Status(request.GetOperationId()); !found {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("reindex operation not found"))
	}
	h.options.Indexer.Cancel(request.GetOperationId())
	job, _ := h.options.Indexer.Status(request.GetOperationId())
	return directReindexResponse(job), nil
}

func (h *ConversationSearchDirectControl) WriteConversationSearchConfig(_ context.Context, request *domainpb.WriteConversationSearchConfigRequest) (*domainpb.WriteConversationSearchConfigResponse, error) {
	if err := h.authorize(request.GetControlToken()); err != nil {
		return nil, err
	}
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("direct configuration writes are not exposed; use the Search Hub provider control contract"))
}

func (h *ConversationSearchDirectControl) WriteConversationSearchCorpus(_ context.Context, request *domainpb.WriteConversationSearchCorpusRequest) (*domainpb.WriteConversationSearchCorpusResponse, error) {
	if err := h.authorize(request.GetControlToken()); err != nil {
		return nil, err
	}
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("direct corpus writes are not exposed; use the Search Hub provider control contract"))
}

func directReindexResponse(job *conversationsearch.ReindexJob) *domainpb.ConversationReindexResponse {
	if job == nil {
		return &domainpb.ConversationReindexResponse{}
	}
	return &domainpb.ConversationReindexResponse{
		OperationId: job.ID, State: directReindexState(job.State), DryRun: job.DryRun,
		PlannedDocuments: job.PlannedDocuments, ProcessedDocuments: job.ProcessedDocuments,
		UpsertedDocuments: job.UpsertedDocuments, DeletedDocuments: job.DeletedDocuments,
		FailedDocuments: job.FailedDocuments, SourceCheckpoint: job.SourceCheckpoint,
		ShadowGeneration: job.ShadowGeneration, ActiveGeneration: job.ActiveGeneration,
		StartedAt: timestamppb.New(job.StartedAt), UpdatedAt: timestamppb.New(job.UpdatedAt),
	}
}

func directReindexState(state conversationsearch.ReindexState) domainpb.ConversationReindexState {
	switch state {
	case conversationsearch.ReindexPlanned:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_PLANNED
	case conversationsearch.ReindexQueued:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_QUEUED
	case conversationsearch.ReindexRunning:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_RUNNING
	case conversationsearch.ReindexCancelled:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_CANCELLED
	case conversationsearch.ReindexFailed:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_FAILED
	case conversationsearch.ReindexComplete:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_COMPLETE
	default:
		return domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_UNSPECIFIED
	}
}

func (h *ConversationSearchSharedControl) Reindex(ctx context.Context, req *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error) {
	if err := h.authorize(req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	if h.options.Indexer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation indexer is unavailable"))
	}
	if strings.TrimSpace(req.Msg.GetScope()) != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("conversation reindex does not support a partial scope"))
	}
	job, err := h.options.Indexer.Reindex(ctx, 0, "", req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.ReindexResponse{JobId: job.ID, PlannedUpserts: boundedInt32(job.PlannedDocuments), PlannedDeletes: boundedInt32(job.DeletedDocuments), DryRun: job.DryRun}), nil
}

func (h *ConversationSearchSharedControl) ReindexStatus(_ context.Context, req *connect.Request[controlv1.ReindexStatusRequest]) (*connect.Response[controlv1.ReindexStatusResponse], error) {
	if err := h.authorize(req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	job, ok := h.options.Indexer.Status(req.Msg.GetJobId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("reindex job not found"))
	}
	return connect.NewResponse(&controlv1.ReindexStatusResponse{JobId: job.ID, State: sharedJobState(job.State), Processed: boundedInt32(job.ProcessedDocuments), Total: boundedInt32(job.PlannedDocuments), Error: job.ErrorCode}), nil
}

func (h *ConversationSearchSharedControl) ReindexCancel(_ context.Context, req *connect.Request[controlv1.ReindexCancelRequest]) (*connect.Response[controlv1.ReindexCancelResponse], error) {
	if err := h.authorize(req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.ReindexCancelResponse{JobId: req.Msg.GetJobId(), Cancelled: h.options.Indexer != nil && h.options.Indexer.Cancel(req.Msg.GetJobId())}), nil
}

func (h *ConversationSearchSharedControl) WriteConfig(ctx context.Context, req *connect.Request[controlv1.WriteConfigRequest]) (*connect.Response[controlv1.WriteConfigResponse], error) {
	if err := h.authorize(req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	if req.Msg.GetProviderId() != conversationsearch.ConversationSearchProviderID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("provider is not owned by agent-manager"))
	}
	tuning := tuningFromProto(req.Msg.GetTuning()).WithDefaults()
	if err := tuning.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	effective, indexChanged, written, err := aisearch.WriteProviderTuning(h.options.SearchFilePath, req.Msg.GetProviderId(), tuning, req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &controlv1.WriteConfigResponse{Written: written, Effective: tuningToProto(effective)}
	if written && indexChanged && h.options.Indexer != nil {
		job, reindexErr := h.options.Indexer.Reindex(ctx, 0, "config-write", false)
		if reindexErr != nil {
			return nil, connect.NewError(connect.CodeInternal, reindexErr)
		}
		response.ReindexTriggered, response.ReindexJobId = true, job.ID
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchSharedControl) WriteCorpus(_ context.Context, req *connect.Request[controlv1.WriteCorpusRequest]) (*connect.Response[controlv1.WriteCorpusResponse], error) {
	if err := h.authorize(req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	if req.Msg.GetProviderId() != conversationsearch.ConversationSearchProviderID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("provider is not owned by agent-manager"))
	}
	suite := suiteFromProto(req.Msg.GetCorpus())
	if err := suite.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	effective, written, err := aisearch.WriteProviderCorpus(h.options.SearchFilePath, req.Msg.GetProviderId(), suite, req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.WriteCorpusResponse{Written: written, Effective: suiteToProto(req.Msg.GetProviderId(), effective)}), nil
}

func tuningFromProto(value *registryv1.Tuning) aisearch.TuningConfig {
	if value == nil {
		return aisearch.TuningConfig{}
	}
	tuning := aisearch.TuningConfig{Engine: value.GetEngine(), EmbedModel: value.GetEmbedModel(), EmbedTaskPrefix: value.GetEmbedTaskPrefix(), RerankEnabled: value.GetRerankEnabled(), RerankBlend: value.GetRerankBlend(), RerankShortlist: int(value.GetRerankShortlist()), RerankPreference: value.GetRerankPreference(), HybridFusion: value.GetHybridFusion()}
	if floor := value.GetFloor(); floor != nil {
		tuning.Floor = aisearch.FloorTuning{MaxGap: floor.GetMaxGap(), HardFloor: floor.GetHardFloor()}
	}
	return tuning
}

func tuningToProto(value aisearch.TuningConfig) *registryv1.Tuning {
	return &registryv1.Tuning{Engine: value.Engine, EmbedModel: value.EmbedModel, EmbedTaskPrefix: value.EmbedTaskPrefix, RerankEnabled: value.RerankEnabled, RerankBlend: value.RerankBlend, RerankShortlist: int32(value.RerankShortlist), RerankPreference: value.RerankPreference, HybridFusion: value.HybridFusion, Floor: &registryv1.FloorConfig{MaxGap: value.Floor.MaxGap, HardFloor: value.Floor.HardFloor}}
}

func suiteFromProto(value *evalv1.EvalSuite) aisearch.TestSuite {
	if value == nil {
		return aisearch.TestSuite{}
	}
	suite := aisearch.TestSuite{SuiteID: value.GetSuiteId(), Name: value.GetName(), Description: value.GetDescription()}
	if suite.SuiteID == value.GetProviderId()+".primary" {
		suite.SuiteID = ""
	}
	for _, c := range value.GetCases() {
		suite.Cases = append(suite.Cases, aisearch.TestCase{ID: c.GetCaseId(), Query: c.GetQuery(), Scope: c.GetScope(), Status: c.GetStatus(), Tags: append([]string(nil), c.GetTags()...), ExpectIDs: append([]string(nil), c.GetExpectIds()...), ExpectWithinTopK: int(c.GetExpectWithinTopK()), ExpectMinScore: c.GetExpectMinScore(), ExpectMaxScore: c.GetExpectMaxScore(), ExpectNoStrongHit: c.GetExpectNoStrongHit(), Note: c.GetNote()})
	}
	return suite
}

func suiteToProto(providerID string, value aisearch.TestSuite) *evalv1.EvalSuite {
	suite := &evalv1.EvalSuite{SuiteId: value.ResolvedSuiteID(providerID), ProviderId: providerID, Name: value.Name, Description: value.Description, State: "active"}
	for _, c := range value.Cases {
		suite.Cases = append(suite.Cases, &evalv1.EvalCase{CaseId: c.ID, Query: c.Query, Scope: c.Scope, Status: c.Status, Tags: append([]string(nil), c.Tags...), ExpectIds: append([]string(nil), c.ExpectIDs...), ExpectWithinTopK: int32(c.ExpectWithinTopK), ExpectMinScore: c.ExpectMinScore, ExpectMaxScore: c.ExpectMaxScore, ExpectNoStrongHit: c.ExpectNoStrongHit, Note: c.Note})
	}
	return suite
}

func sharedJobState(state conversationsearch.ReindexState) string {
	switch state {
	case conversationsearch.ReindexQueued:
		return "pending"
	case conversationsearch.ReindexRunning:
		return "running"
	case conversationsearch.ReindexComplete:
		return "succeeded"
	case conversationsearch.ReindexCancelled:
		return "cancelled"
	case conversationsearch.ReindexFailed:
		return "failed"
	default:
		return string(state)
	}
}

func boundedInt32(value uint64) int32 {
	if value > uint64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(value)
}

var (
	_ control_v1connect.SearchControlServiceHandler = (*ConversationSearchSharedControl)(nil)
	_ ConversationSearchControlOperations           = (*ConversationSearchDirectControl)(nil)
)
