package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"agent-manager/internal/conversationsearch"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	"google.golang.org/protobuf/proto"
)

// ConversationSearchOperations is the transport-independent application seam
// for attributable conversation recall. The concrete search engine owns
// ranking, cursor integrity, visibility, and projection state.
type ConversationSearchOperations interface {
	SearchConversations(context.Context, *domainpb.SearchConversationsRequest) (*domainpb.SearchConversationsResponse, error)
	GetConversationContext(context.Context, *domainpb.GetConversationContextRequest) (*domainpb.GetConversationContextResponse, error)
	GetConversationIndexStatus(context.Context, *domainpb.GetConversationIndexStatusRequest) (*domainpb.GetConversationIndexStatusResponse, error)
	RecordConversationSearchInteraction(context.Context, *domainpb.RecordConversationSearchInteractionRequest) (*domainpb.RecordConversationSearchInteractionResponse, error)
}

// ConversationSearchControlOperations is deliberately separate from the read
// service so operator authorization can be applied to its mount as one unit.
type ConversationSearchControlOperations interface {
	PlanConversationReindex(context.Context, *domainpb.PlanConversationReindexRequest) (*domainpb.ConversationReindexResponse, error)
	ReindexConversations(context.Context, *domainpb.ReindexConversationsRequest) (*domainpb.ConversationReindexResponse, error)
	CancelConversationReindex(context.Context, *domainpb.CancelConversationReindexRequest) (*domainpb.ConversationReindexResponse, error)
	WriteConversationSearchConfig(context.Context, *domainpb.WriteConversationSearchConfigRequest) (*domainpb.WriteConversationSearchConfigResponse, error)
	WriteConversationSearchCorpus(context.Context, *domainpb.WriteConversationSearchCorpusRequest) (*domainpb.WriteConversationSearchCorpusResponse, error)
}

type ConversationSearchConnectHandler struct {
	domainconnect.UnimplementedConversationSearchServiceHandler
	operations ConversationSearchOperations
	validator  protovalidate.Validator
	validation sync.Mutex
}

func NewConversationSearchConnectHandler(operations ConversationSearchOperations) *ConversationSearchConnectHandler {
	validator, err := protovalidate.New()
	if err != nil {
		panic(fmt.Sprintf("initialize conversation-search validator: %v", err))
	}
	return &ConversationSearchConnectHandler{operations: operations, validator: validator}
}

func (h *ConversationSearchConnectHandler) SearchConversations(ctx context.Context, req *connect.Request[domainpb.SearchConversationsRequest]) (*connect.Response[domainpb.SearchConversationsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := ValidateConversationSearchRequest(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
	}
	if h.operations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation search is unavailable"))
	}
	response, err := h.operations.SearchConversations(ctx, req.Msg)
	if err != nil {
		return nil, conversationSearchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchConnectHandler) GetConversationContext(ctx context.Context, req *connect.Request[domainpb.GetConversationContextRequest]) (*connect.Response[domainpb.GetConversationContextResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
	}
	if h.operations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation search is unavailable"))
	}
	response, err := h.operations.GetConversationContext(ctx, req.Msg)
	if err != nil {
		return nil, conversationSearchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchConnectHandler) validate(message proto.Message) error {
	// Protovalidate builds descriptor caches lazily. Serialize validation so the
	// handler remains race-free even on the first concurrent requests.
	h.validation.Lock()
	defer h.validation.Unlock()
	return h.validator.Validate(message)
}

func (h *ConversationSearchConnectHandler) GetConversationIndexStatus(ctx context.Context, req *connect.Request[domainpb.GetConversationIndexStatusRequest]) (*connect.Response[domainpb.GetConversationIndexStatusResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation search is unavailable"))
	}
	response, err := h.operations.GetConversationIndexStatus(ctx, req.Msg)
	if err != nil {
		return nil, conversationSearchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchConnectHandler) RecordConversationSearchInteraction(ctx context.Context, req *connect.Request[domainpb.RecordConversationSearchInteractionRequest]) (*connect.Response[domainpb.RecordConversationSearchInteractionResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
	}
	if err := ValidateConversationSearchInteraction(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h == nil || h.operations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("conversation search is unavailable"))
	}
	response, err := h.operations.RecordConversationSearchInteraction(ctx, req.Msg)
	if err != nil {
		return nil, conversationSearchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

// ValidateConversationSearchInteraction keeps outcome telemetry attributable
// to the ephemeral session that issued the search and rejects ambiguous event
// shapes before they reach persistence.
func ValidateConversationSearchInteraction(request *domainpb.RecordConversationSearchInteractionRequest) error {
	if request == nil {
		return errors.New("request is required")
	}
	if strings.TrimSpace(request.GetTelemetrySessionToken()) == "" {
		return errors.New("telemetry_session_token: required")
	}
	switch request.GetKind() {
	case domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_SELECTED:
		if strings.TrimSpace(request.GetStableHitId()) == "" {
			return errors.New("stable_hit_id: required for selected interaction")
		}
		if request.GetSelectedRank() < 1 || request.GetSelectedRank() > 100 {
			return errors.New("selected_rank: must be between 1 and 100 for selected interaction")
		}
	case domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_REFORMULATED:
		if request.GetStableHitId() != "" || request.GetSelectedRank() != 0 {
			return errors.New("stable_hit_id and selected_rank: must be empty for reformulated interaction")
		}
	}
	return nil
}

func conversationSearchConnectError(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr
	}
	switch {
	case errors.Is(err, conversationsearch.ErrInvalidRequest):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, conversationsearch.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// ValidateConversationSearchRequest enforces semantic constraints that cannot
// be expressed as independent protobuf field rules. Projection-specific limits
// such as regex scan bytes and execution time remain server-owned.
func ValidateConversationSearchRequest(request *domainpb.SearchConversationsRequest) error {
	if request == nil {
		return errors.New("request is required")
	}
	queryEmpty := strings.TrimSpace(request.GetQuery()) == ""
	if !queryEmpty {
		return validateConversationTimeRange(request.GetFilters())
	}
	if request.GetMode() == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX {
		return errors.New("query: required for regex mode")
	}
	if request.GetMode() == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_SEMANTIC {
		return errors.New("query: required for semantic mode")
	}
	if request.GetSort() != domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST && request.GetSort() != domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_OLDEST {
		return errors.New("query: may be empty only when sort is newest or oldest")
	}
	if !hasConversationSearchFilter(request.GetFilters()) {
		return errors.New("filters: at least one structured filter is required when query is empty")
	}
	return validateConversationTimeRange(request.GetFilters())
}

func hasConversationSearchFilter(filters *domainpb.ConversationSearchFilters) bool {
	if filters == nil {
		return false
	}
	return len(filters.GetRoles()) > 0 || len(filters.GetHarnesses()) > 0 || len(filters.GetProviderOrigins()) > 0 ||
		len(filters.GetProjectScopes()) > 0 || len(filters.GetCwdScopes()) > 0 || len(filters.GetRunners()) > 0 ||
		len(filters.GetModels()) > 0 || len(filters.GetProfiles()) > 0 || len(filters.GetRunStatuses()) > 0 ||
		len(filters.GetTags()) > 0 || len(filters.GetWorkloads()) > 0 || filters.GetOccurredAfter() != nil ||
		filters.GetOccurredBefore() != nil || len(filters.GetContentClasses()) > 0 || filters.GetIncludeToolEvents()
}

func validateConversationTimeRange(filters *domainpb.ConversationSearchFilters) error {
	if filters == nil || filters.GetOccurredAfter() == nil || filters.GetOccurredBefore() == nil {
		return nil
	}
	if filters.GetOccurredAfter().AsTime().After(filters.GetOccurredBefore().AsTime()) {
		return fmt.Errorf("filters.occurred_after: must not be after filters.occurred_before")
	}
	return nil
}

type ConversationSearchControlConnectHandler struct {
	domainconnect.UnimplementedConversationSearchControlServiceHandler
	operations ConversationSearchControlOperations
}

func NewConversationSearchControlConnectHandler(operations ConversationSearchControlOperations) *ConversationSearchControlConnectHandler {
	return &ConversationSearchControlConnectHandler{operations: operations}
}

func (h *ConversationSearchControlConnectHandler) PlanConversationReindex(ctx context.Context, req *connect.Request[domainpb.PlanConversationReindexRequest]) (*connect.Response[domainpb.ConversationReindexResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, conversationSearchControlUnavailable()
	}
	response, err := h.operations.PlanConversationReindex(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchControlConnectHandler) ReindexConversations(ctx context.Context, req *connect.Request[domainpb.ReindexConversationsRequest]) (*connect.Response[domainpb.ConversationReindexResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, conversationSearchControlUnavailable()
	}
	response, err := h.operations.ReindexConversations(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchControlConnectHandler) CancelConversationReindex(ctx context.Context, req *connect.Request[domainpb.CancelConversationReindexRequest]) (*connect.Response[domainpb.ConversationReindexResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, conversationSearchControlUnavailable()
	}
	response, err := h.operations.CancelConversationReindex(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchControlConnectHandler) WriteConversationSearchConfig(ctx context.Context, req *connect.Request[domainpb.WriteConversationSearchConfigRequest]) (*connect.Response[domainpb.WriteConversationSearchConfigResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, conversationSearchControlUnavailable()
	}
	response, err := h.operations.WriteConversationSearchConfig(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *ConversationSearchControlConnectHandler) WriteConversationSearchCorpus(ctx context.Context, req *connect.Request[domainpb.WriteConversationSearchCorpusRequest]) (*connect.Response[domainpb.WriteConversationSearchCorpusResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.operations == nil {
		return nil, conversationSearchControlUnavailable()
	}
	response, err := h.operations.WriteConversationSearchCorpus(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func conversationSearchControlUnavailable() error {
	return connect.NewError(connect.CodeUnavailable, errors.New("conversation search control is unavailable"))
}

var (
	_ domainconnect.ConversationSearchServiceHandler        = (*ConversationSearchConnectHandler)(nil)
	_ domainconnect.ConversationSearchControlServiceHandler = (*ConversationSearchControlConnectHandler)(nil)
)
