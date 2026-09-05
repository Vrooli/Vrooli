package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/orchestration/obs"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConversationSearchAdapter translates the generated transport contract to
// the engine-independent conversation-search service.
type ConversationSearchAdapter struct {
	service *conversationsearch.Service
	indexer *conversationsearch.Indexer
}

func NewConversationSearchAdapter(service *conversationsearch.Service, indexer ...*conversationsearch.Indexer) *ConversationSearchAdapter {
	adapter := &ConversationSearchAdapter{service: service}
	if len(indexer) > 0 {
		adapter.indexer = indexer[0]
	}
	return adapter
}

func (a *ConversationSearchAdapter) SearchConversations(ctx context.Context, request *domainpb.SearchConversationsRequest) (*domainpb.SearchConversationsResponse, error) {
	startedAt := time.Now()
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("conversation search is unavailable")
	}
	mode := request.GetMode()
	if mode == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_UNSPECIFIED {
		mode = domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_HYBRID
	}
	sortOrder, err := searchSortFromProto(request.GetSort())
	if err != nil {
		requestID, _ := a.service.RecordSearchTelemetry(ctx, conversationTelemetry(request, nil, err, time.Since(startedAt)))
		logConversationSearch(requestID, request.GetMode().String(), nil, nil, err)
		return nil, errors.Join(conversationsearch.ErrInvalidRequest, err)
	}
	filters := filtersFromProto(request.GetFilters())
	var response conversationsearch.TextSearchResponse
	switch mode {
	case domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX:
		response, err = a.service.SearchRegex(ctx, conversationsearch.RegexSearchRequest{
			Pattern: request.GetQuery(), Sort: sortOrder, PageSize: int(request.GetPageSize()), Cursor: request.GetPageCursor(), Filters: filters,
		})
	case domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_SEMANTIC:
		response, err = a.service.SearchSemantic(ctx, conversationsearch.TextSearchRequest{
			Query: request.GetQuery(), Sort: sortOrder, PageSize: int(request.GetPageSize()), Cursor: request.GetPageCursor(), Filters: filters,
		})
	case domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_HYBRID:
		response, err = a.service.SearchHybrid(ctx, conversationsearch.TextSearchRequest{
			Query: request.GetQuery(), Sort: sortOrder, PageSize: int(request.GetPageSize()), Cursor: request.GetPageCursor(), Filters: filters,
		})
	default:
		response, err = a.service.SearchText(ctx, conversationsearch.TextSearchRequest{
			Query: request.GetQuery(), Sort: sortOrder, PageSize: int(request.GetPageSize()), Cursor: request.GetPageCursor(), Filters: filters,
		})
	}
	if err != nil {
		requestID, _ := a.service.RecordSearchTelemetry(ctx, conversationTelemetry(request, nil, err, time.Since(startedAt)))
		logConversationSearch(requestID, mode.String(), nil, nil, err)
		return nil, err
	}
	output := &domainpb.SearchConversationsResponse{
		NextPageCursor: response.NextCursor,
		ModeUsed:       mode,
		SortUsed:       searchSortToProto(sortOrder),
		Coverage: &domainpb.ConversationSearchCoverage{
			CanonicalVisibleMessages: response.CanonicalVisibleMessages,
			CatalogDocuments:         response.CatalogDocuments,
			LexicalDocuments:         response.LexicalDocuments,
			LexicalRatio:             coverageRatio(response.LexicalDocuments, response.CatalogDocuments),
		},
	}
	for _, degradation := range response.Degradations {
		output.Degradations = append(output.Degradations, searchDegradationToProto(degradation))
	}
	if response.PartialReason != conversationsearch.RegexLimitNone {
		output.Degradations = append(output.Degradations, regexPartialDegradation(response))
	}
	for _, hit := range response.Hits {
		output.Hits = append(output.Hits, searchHitToProto(hit))
	}
	if a.indexer != nil {
		if status, statusErr := a.indexer.StatusSnapshot(ctx); statusErr == nil {
			output.Coverage.SemanticDocuments = status.SemanticDocuments
			output.Coverage.PendingDocuments = status.PendingChanges
			output.Coverage.DeletedDocuments = status.DeletedDocuments
			output.Coverage.OrphanDocuments = status.OrphanDocuments
			output.Coverage.SemanticRatio = coverageRatio(status.SemanticDocuments, status.CatalogDocuments)
			if !status.LastSuccessAt.IsZero() {
				output.Coverage.LastReconciledAt = timestamppb.New(status.LastSuccessAt)
				if lag := time.Since(status.LastSuccessAt); lag > 0 {
					output.Coverage.FreshnessLagMs = uint64(lag.Milliseconds())
				}
			}
		}
	}
	output.TookMs = uint64(time.Since(startedAt).Milliseconds())
	requestID, _ := a.service.RecordSearchTelemetry(ctx, conversationTelemetry(request, output, nil, time.Since(startedAt)))
	output.RequestId = requestID
	resultIDs := make([]string, 0, len(output.GetHits()))
	for _, hit := range output.GetHits() {
		resultIDs = append(resultIDs, hit.GetStableHitId())
	}
	logConversationSearch(requestID, output.GetModeUsed().String(), resultIDs, output.GetDegradations(), nil)
	return output, nil
}

func logConversationSearch(requestID, mode string, resultIDs []string, degradations []*domainpb.ConversationSearchDegradation, searchErr error) {
	attrs := []any{
		obs.KeySearchRequestID, requestID,
		obs.KeySearchMode, mode,
		obs.KeySearchResultCount, len(resultIDs),
		obs.KeySearchResultIDs, resultIDs,
		obs.KeySearchDegraded, len(degradations) > 0,
	}
	logger := obs.Component("conversation-search")
	if searchErr != nil {
		attrs = append(attrs, obs.KeySearchErrorClass, conversationSearchErrorCategory(searchErr))
		logger.Warn("conversation_search_completed", attrs...)
		return
	}
	logger.Info("conversation_search_completed", attrs...)
}

func (a *ConversationSearchAdapter) RecordConversationSearchInteraction(ctx context.Context, request *domainpb.RecordConversationSearchInteractionRequest) (*domainpb.RecordConversationSearchInteractionResponse, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("conversation search is unavailable")
	}
	interaction := conversationsearch.SearchInteraction{
		RequestID: request.GetRequestId(), SessionToken: request.GetTelemetrySessionToken(),
		StableHitID: request.GetStableHitId(), SelectedRank: int(request.GetSelectedRank()),
		Reformulated: request.GetKind() == domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_REFORMULATED,
	}
	accepted, err := a.service.RecordSearchInteraction(ctx, interaction)
	if err != nil {
		if errors.Is(err, conversationsearch.ErrInvalidRequest) {
			return nil, errors.Join(conversationsearch.ErrInvalidRequest, err)
		}
		return nil, err
	}
	return &domainpb.RecordConversationSearchInteractionResponse{Accepted: accepted}, nil
}

func conversationTelemetry(request *domainpb.SearchConversationsRequest, response *domainpb.SearchConversationsResponse, searchErr error, duration time.Duration) conversationsearch.SearchTelemetry {
	record := conversationsearch.SearchTelemetry{
		SessionToken: request.GetTelemetrySessionToken(), Mode: request.GetMode().String(), Sort: request.GetSort().String(),
		FilterFamilies: conversationFilterFamilies(request.GetFilters()), Duration: duration,
		FreshnessBand: "unknown", ErrorCategory: conversationSearchErrorCategory(searchErr),
	}
	if response == nil {
		return record
	}
	record.Mode = response.GetModeUsed().String()
	record.Sort = response.GetSortUsed().String()
	record.ResultCount = len(response.GetHits())
	record.CandidateCount = record.ResultCount
	record.WeakOnly = record.ResultCount > 0
	for _, hit := range response.GetHits() {
		record.ResultStableHitIDs = append(record.ResultStableHitIDs, hit.GetStableHitId())
		if !hit.GetWeak() {
			record.WeakOnly = false
		}
		for _, evidence := range hit.GetRankEvidence() {
			switch evidence.GetLeg() {
			case domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_LEXICAL, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_REGEX:
				record.LexicalContributed = true
			case domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_DENSE, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_SPARSE, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_RERANK:
				record.SemanticContributed = true
			}
		}
	}
	for _, degradation := range response.GetDegradations() {
		record.DegradationReasons = append(record.DegradationReasons, degradation.GetReason().String())
	}
	if coverage := response.GetCoverage(); coverage != nil && coverage.GetLastReconciledAt() != nil {
		lag := time.Since(coverage.GetLastReconciledAt().AsTime())
		switch {
		case lag < 0:
			record.FreshnessBand = "clock_skew"
		case lag <= time.Minute:
			record.FreshnessBand = "fresh"
		case lag <= 24*time.Hour:
			record.FreshnessBand = "aging"
		default:
			record.FreshnessBand = "stale"
		}
	}
	return record
}

func conversationFilterFamilies(filters *domainpb.ConversationSearchFilters) []string {
	if filters == nil {
		return nil
	}
	var families []string
	checks := []struct {
		name string
		set  bool
	}{
		{"role", len(filters.GetRoles()) > 0},
		{"harness", len(filters.GetHarnesses()) > 0},
		{"provider_origin", len(filters.GetProviderOrigins()) > 0},
		{"project", len(filters.GetProjectScopes()) > 0},
		{"cwd", len(filters.GetCwdScopes()) > 0},
		{"runner", len(filters.GetRunners()) > 0},
		{"model", len(filters.GetModels()) > 0},
		{"profile", len(filters.GetProfiles()) > 0},
		{"run_status", len(filters.GetRunStatuses()) > 0},
		{"tag", len(filters.GetTags()) > 0},
		{"workload", len(filters.GetWorkloads()) > 0},
		{"time", filters.GetOccurredAfter() != nil || filters.GetOccurredBefore() != nil},
		{"content_class", len(filters.GetContentClasses()) > 0},
		{"tool_events", filters.GetIncludeToolEvents()},
	}
	for _, check := range checks {
		if check.set {
			families = append(families, check.name)
		}
	}
	return families
}

func conversationSearchErrorCategory(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, conversationsearch.ErrInvalidRequest) {
		return "invalid_request"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline"
	}
	return "internal"
}

func searchDegradationToProto(degradation conversationsearch.Degradation) *domainpb.ConversationSearchDegradation {
	reason := domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE
	switch degradation.Reason {
	case conversationsearch.DegradationEmbeddingUnavailable:
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_EMBEDDING_UNAVAILABLE
	case conversationsearch.DegradationVectorStoreUnavailable:
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_VECTOR_STORE_UNAVAILABLE
	case conversationsearch.DegradationIndexLayoutMismatch:
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_LAYOUT_MISMATCH
	case conversationsearch.DegradationRerankUnavailable:
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_RERANK_UNAVAILABLE
	case conversationsearch.DegradationDeadline:
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE
	}
	return &domainpb.ConversationSearchDegradation{Reason: reason, Leg: searchLegToProto(degradation.Leg), Detail: degradation.Detail, Retryable: degradation.Retryable}
}

func regexPartialDegradation(response conversationsearch.TextSearchResponse) *domainpb.ConversationSearchDegradation {
	reason := domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_CANDIDATE_LIMIT
	if response.PartialReason == conversationsearch.RegexLimitDeadline {
		reason = domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE
	}
	return &domainpb.ConversationSearchDegradation{
		Reason: reason, Leg: domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_REGEX,
		Detail: fmt.Sprintf("bounded regex search stopped at %s after %d candidates and %d bytes", response.PartialReason, response.ScannedCandidates, response.ScannedBytes),
	}
}

func (a *ConversationSearchAdapter) GetConversationContext(ctx context.Context, request *domainpb.GetConversationContextRequest) (*domainpb.GetConversationContextResponse, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("conversation search is unavailable")
	}
	hit, documents, err := a.service.Context(ctx, request.GetStableHitId(), int(request.GetBeforeEvents()), int(request.GetAfterEvents()))
	if err != nil {
		return nil, err
	}
	output := &domainpb.GetConversationContextResponse{Hit: searchHitToProto(conversationsearch.SearchHit{Document: hit, Snippet: hit.Content, DeepLink: "/runs/" + hit.SourceRunID + "?event=" + hit.SourceEventID})}
	for _, document := range documents {
		output.Events = append(output.Events, &domainpb.ConversationContextEvent{
			EventId: document.SourceEventID, EventSequence: document.EventSequence, Role: document.Role,
			OccurredAt: timestamppb.New(document.OccurredAt), BoundedContent: boundedContextContent(document.Content),
			ContentClass: contentClassToProto(document.ContentClass), Matched: document.SourceEventID == hit.SourceEventID,
		})
	}
	return output, nil
}

func (a *ConversationSearchAdapter) GetConversationIndexStatus(ctx context.Context, _ *domainpb.GetConversationIndexStatusRequest) (*domainpb.GetConversationIndexStatusResponse, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("conversation search is unavailable")
	}
	visibleMessages, catalogDocuments, lexicalDocuments, err := a.service.Status(ctx)
	if err != nil {
		return nil, err
	}
	response := &domainpb.GetConversationIndexStatusResponse{
		State: domainpb.ConversationIndexState_CONVERSATION_INDEX_STATE_READY,
		Coverage: &domainpb.ConversationSearchCoverage{
			CanonicalVisibleMessages: visibleMessages, CatalogDocuments: catalogDocuments,
			LexicalDocuments: lexicalDocuments, LexicalRatio: coverageRatio(lexicalDocuments, catalogDocuments),
		},
		RecipeVersion: conversationsearch.DefaultRecipeVersion,
	}
	if a.indexer != nil {
		status, statusErr := a.indexer.StatusSnapshot(ctx)
		if statusErr != nil {
			return nil, statusErr
		}
		response.Coverage.CanonicalVisibleMessages = status.CanonicalMessages
		response.Coverage.CatalogDocuments = status.CatalogDocuments
		response.Coverage.LexicalDocuments = status.LexicalDocuments
		response.Coverage.SemanticDocuments = status.SemanticDocuments
		response.Coverage.PendingDocuments = status.PendingChanges
		response.Coverage.DeletedDocuments = status.DeletedDocuments
		response.Coverage.OrphanDocuments = status.OrphanDocuments
		response.Coverage.LexicalRatio = coverageRatio(status.LexicalDocuments, status.CatalogDocuments)
		response.Coverage.SemanticRatio = coverageRatio(status.SemanticDocuments, status.CatalogDocuments)
		response.ActiveGeneration = status.ActiveGeneration
		response.CandidateGeneration = status.CandidateGeneration
		response.CollectionName = status.CollectionName
		response.EmbeddingModel = status.EmbeddingModel
		response.CollectionLayout = status.CollectionLayout
		response.DegradedDependencies = append([]string(nil), status.DegradedDependencies...)
		response.LastErrorCode = status.LastErrorCode
		if !status.LastIndexedAt.IsZero() {
			response.LastIndexedAt = timestamppb.New(status.LastIndexedAt)
		}
		if !status.LastSuccessAt.IsZero() {
			response.LastSuccessAt = timestamppb.New(status.LastSuccessAt)
			response.Coverage.LastReconciledAt = timestamppb.New(status.LastSuccessAt)
			if lag := time.Since(status.LastSuccessAt); lag > 0 {
				response.Coverage.FreshnessLagMs = uint64(lag.Milliseconds())
			}
		}
		if status.LastErrorCode != "" || len(status.DegradedDependencies) > 0 {
			response.State = domainpb.ConversationIndexState_CONVERSATION_INDEX_STATE_DEGRADED
		} else if status.PendingChanges > 0 || status.OrphanDocuments > 0 {
			response.State = domainpb.ConversationIndexState_CONVERSATION_INDEX_STATE_STALE
		}
	}
	return response, nil
}

func filtersFromProto(filters *domainpb.ConversationSearchFilters) conversationsearch.SearchFilters {
	if filters == nil {
		return conversationsearch.SearchFilters{}
	}
	classes := make([]conversationsearch.ContentClass, 0, len(filters.GetContentClasses())+2)
	for _, class := range filters.GetContentClasses() {
		classes = append(classes, conversationsearch.ContentClass(class))
	}
	if filters.GetIncludeToolEvents() && len(classes) == 0 {
		classes = []conversationsearch.ContentClass{
			conversationsearch.ContentClassProse, conversationsearch.ContentClassQuotedProse,
			conversationsearch.ContentClassToolCall, conversationsearch.ContentClassToolResult,
		}
	}
	return conversationsearch.SearchFilters{
		OccurredAfter: timestampPointer(filters.GetOccurredAfter()), OccurredBefore: timestampPointer(filters.GetOccurredBefore()),
		Roles: filters.GetRoles(), Harnesses: filters.GetHarnesses(), ProviderOrigins: filters.GetProviderOrigins(),
		ProjectScopes: filters.GetProjectScopes(), CWDScopes: filters.GetCwdScopes(), Runners: filters.GetRunners(),
		Models: filters.GetModels(), Profiles: filters.GetProfiles(), RunStatuses: filters.GetRunStatuses(),
		Tags: filters.GetTags(), Workloads: filters.GetWorkloads(), ContentClasses: classes,
	}
}

func searchHitToProto(hit conversationsearch.SearchHit) *domainpb.ConversationSearchHit {
	document := hit.Document
	output := &domainpb.ConversationSearchHit{
		StableHitId: document.DocumentID, RunId: document.SourceRunID, EventId: document.SourceEventID,
		MessageId: document.SourceMessageID, ChunkId: document.DocumentID, ChunkIndex: uint32(document.ChunkIndex),
		EventSequence: document.EventSequence, Role: document.Role, OccurredAt: timestamppb.New(document.OccurredAt),
		Snippet: hit.Snippet, ContentClass: contentClassToProto(document.ContentClass),
		Provenance: &domainpb.ConversationSourceProvenance{
			Harness: document.Harness, SourceSessionId: document.SourceSessionID, ProviderOrigin: document.ProviderOrigin,
			Importer: document.Importer, ProjectScope: document.ProjectScope, CwdScope: document.CWDScope, EvidenceRef: document.EvidenceRef,
		},
		Run: &domainpb.ConversationRunSummary{
			RunId: document.SourceRunID, Label: document.RunLabel, Status: document.RunStatus,
			Runner: document.Runner, Model: document.Model, Profile: document.Profile,
			Tags: document.Tags, Workloads: document.Workloads,
		},
		DeepLink: hit.DeepLink,
		Weak:     hit.Weak,
	}
	for _, evidence := range hit.Evidence {
		output.RankEvidence = append(output.RankEvidence, &domainpb.ConversationRankEvidence{
			Leg: searchLegToProto(evidence.Leg), Rank: uint32(evidence.Rank), Score: evidence.Score, Explanation: evidence.Explanation,
		})
	}
	if len(output.RankEvidence) == 0 && hit.Rank > 0 {
		output.RankEvidence = []*domainpb.ConversationRankEvidence{{Leg: searchLegToProto(hit.Leg), Rank: uint32(hit.Rank), Score: hit.Score}}
	}
	for _, highlight := range hit.Highlights {
		output.Highlights = append(output.Highlights, &domainpb.ConversationHighlight{StartGrapheme: uint32(highlight.StartRune), EndGrapheme: uint32(highlight.EndRune), Field: "snippet"})
	}
	return output
}

func searchLegToProto(leg conversationsearch.SearchLeg) domainpb.ConversationSearchLeg {
	switch leg {
	case conversationsearch.SearchLegRegex:
		return domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_REGEX
	case conversationsearch.SearchLegDense:
		return domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_DENSE
	case conversationsearch.SearchLegSparse:
		return domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_SPARSE
	case conversationsearch.SearchLegRerank:
		return domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_RERANK
	default:
		return domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_LEXICAL
	}
}

func searchSortFromProto(value domainpb.ConversationSearchSort) (conversationsearch.SearchSort, error) {
	switch value {
	case domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_UNSPECIFIED, domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE:
		return conversationsearch.SearchSortRelevance, nil
	case domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST:
		return conversationsearch.SearchSortNewest, nil
	case domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_OLDEST:
		return conversationsearch.SearchSortOldest, nil
	default:
		return 0, fmt.Errorf("unsupported search sort %s", value)
	}
}

func searchSortToProto(value conversationsearch.SearchSort) domainpb.ConversationSearchSort {
	switch value {
	case conversationsearch.SearchSortNewest:
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST
	case conversationsearch.SearchSortOldest:
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_OLDEST
	default:
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE
	}
}

func contentClassToProto(value conversationsearch.ContentClass) domainpb.ConversationContentClass {
	return domainpb.ConversationContentClass(value)
}

func timestampPointer(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	parsed := value.AsTime()
	return &parsed
}

func coverageRatio(indexed, visible uint64) float64 {
	if visible == 0 {
		return 1
	}
	return float64(indexed) / float64(visible)
}

func boundedContextContent(content string) string {
	snippet, _ := conversationsearch.BoundedContext(content, maximumContextBytes)
	return snippet
}

const maximumContextBytes = 2048

var _ ConversationSearchOperations = (*ConversationSearchAdapter)(nil)
