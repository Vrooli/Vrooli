package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/conversationsearch"

	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

func TestConversationSearchSharedControlFailsClosedOnTokens(t *testing.T) {
	t.Parallel()
	handler := NewConversationSearchSharedControl(ConversationSearchControlOptions{ControlToken: func() string { return "minted-token" }})
	for _, token := range []string{"", "wrong-token"} {
		requests := []func() error{
			func() error {
				_, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: token, DryRun: true}))
				return err
			},
			func() error {
				_, err := handler.ReindexStatus(context.Background(), connect.NewRequest(&controlv1.ReindexStatusRequest{ControlToken: token, JobId: "job"}))
				return err
			},
			func() error {
				_, err := handler.ReindexCancel(context.Background(), connect.NewRequest(&controlv1.ReindexCancelRequest{ControlToken: token, JobId: "job"}))
				return err
			},
			func() error {
				_, err := handler.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{ControlToken: token, ProviderId: conversationsearch.ConversationSearchProviderID, Tuning: &registryv1.Tuning{Engine: "hybrid"}}))
				return err
			},
			func() error {
				_, err := handler.WriteCorpus(context.Background(), connect.NewRequest(&controlv1.WriteCorpusRequest{ControlToken: token, ProviderId: conversationsearch.ConversationSearchProviderID, Corpus: &evalv1.EvalSuite{SuiteId: "agent-manager.runs.primary"}}))
				return err
			},
		}
		for _, request := range requests {
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(request()))
		}
	}
	_, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "minted-token", DryRun: true}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestConversationSearchDirectControlFailsClosedAndMapsJobs(t *testing.T) {
	t.Parallel()
	handler := NewConversationSearchDirectControl(ConversationSearchControlOptions{ControlToken: func() string { return "minted-token" }})
	for _, token := range []string{"", "wrong-token"} {
		_, err := handler.PlanConversationReindex(context.Background(), &domainpb.PlanConversationReindexRequest{Full: true, ControlToken: token})
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	}
	_, err := handler.PlanConversationReindex(context.Background(), &domainpb.PlanConversationReindexRequest{Full: true, ControlToken: "minted-token"})
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	db, err := sqlx.Connect("sqlite", "file:handler-direct-conversation-control?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(conversationsearch.Schema)))
	indexer, err := conversationsearch.NewIndexer(conversationsearch.IndexerOptions{Source: emptyConversationSource{}, Repository: conversationsearch.NewSQLiteRepository(db)})
	require.NoError(t, err)
	handler = NewConversationSearchDirectControl(ConversationSearchControlOptions{Indexer: indexer, ControlToken: func() string { return "minted-token" }})
	plan, err := handler.PlanConversationReindex(context.Background(), &domainpb.PlanConversationReindexRequest{Full: true, MaxDocuments: 10, ControlToken: "minted-token"})
	require.NoError(t, err)
	require.True(t, plan.GetDryRun())
	require.Equal(t, domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_PLANNED, plan.GetState())
	_, err = handler.PlanConversationReindex(context.Background(), &domainpb.PlanConversationReindexRequest{ControlToken: "minted-token"})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	response := directReindexResponse(&conversationsearch.ReindexJob{
		ID: "job-1", State: conversationsearch.ReindexRunning, PlannedDocuments: 10,
		ProcessedDocuments: 4, UpsertedDocuments: 3, DeletedDocuments: 1,
		ShadowGeneration: "candidate-1", ActiveGeneration: "active-1", StartedAt: now, UpdatedAt: now,
	})
	require.Equal(t, domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_RUNNING, response.GetState())
	require.Equal(t, uint64(4), response.GetProcessedDocuments())
	require.Equal(t, "candidate-1", response.GetShadowGeneration())
	require.True(t, response.GetStartedAt().IsValid())
}

type emptyConversationSource struct{}

func (emptyConversationSource) LoadSourcePage(context.Context, *conversationsearch.SourceCursor, int) (conversationsearch.SourcePage, error) {
	return conversationsearch.SourcePage{}, nil
}

func TestConversationSearchContractJSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := &domainpb.SearchConversationsResponse{
		Hits: []*domainpb.ConversationSearchHit{{
			StableHitId:  "msg-42:0",
			RunId:        "run-7",
			Snippet:      "the corrected reasoning",
			ContentClass: domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_PROSE,
			Provenance: &domainpb.ConversationSourceProvenance{
				Harness:         "claude-code",
				SourceSessionId: "session-public-id",
				ProviderOrigin:  "agent-manager.runs",
				EvidenceRef:     "agent-manager://runs/run-7/events/msg-42",
			},
			RankEvidence: []*domainpb.ConversationRankEvidence{{
				Leg:   domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_LEXICAL,
				Rank:  1,
				Score: 0.75,
			}},
			DeepLink: "/runs/run-7?event=msg-42",
		}},
		ModeUsed: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_HYBRID,
		SortUsed: domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE,
		Coverage: &domainpb.ConversationSearchCoverage{CanonicalVisibleMessages: 10, LexicalDocuments: 9},
		Degradations: []*domainpb.ConversationSearchDegradation{{
			Reason: domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE,
			Leg:    domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_DENSE,
		}},
	}

	encoded, err := protojson.Marshal(input)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"stableHitId":"msg-42:0"`)
	require.Contains(t, string(encoded), `"CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE"`)
	require.NotContains(t, string(encoded), "transcriptPath")

	var output domainpb.SearchConversationsResponse
	require.NoError(t, protojson.Unmarshal(encoded, &output))
	require.Equal(t, input, &output)
}

func TestValidateConversationSearchRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *domainpb.SearchConversationsRequest
		wantErr string
	}{
		{name: "query search", request: &domainpb.SearchConversationsRequest{Query: "corrected reasoning"}},
		{name: "filtered browse", request: &domainpb.SearchConversationsRequest{Sort: domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST, Filters: &domainpb.ConversationSearchFilters{Harnesses: []string{"claude-code"}}}},
		{name: "regex needs query", request: &domainpb.SearchConversationsRequest{Mode: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX, Sort: domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST, Filters: &domainpb.ConversationSearchFilters{Harnesses: []string{"claude-code"}}}, wantErr: "query"},
		{name: "browse needs filter", request: &domainpb.SearchConversationsRequest{Sort: domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_OLDEST}, wantErr: "filters"},
		{name: "empty relevance rejected", request: &domainpb.SearchConversationsRequest{Filters: &domainpb.ConversationSearchFilters{Roles: []string{"assistant"}}}, wantErr: "sort"},
		{name: "invalid range", request: &domainpb.SearchConversationsRequest{Query: "reasoning", Filters: &domainpb.ConversationSearchFilters{OccurredAfter: timestamppb.New(timestamppb.Now().AsTime().AddDate(0, 0, 1)), OccurredBefore: timestamppb.Now()}}, wantErr: "occurred_after"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConversationSearchRequest(test.request)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateConversationSearchInteraction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		request *domainpb.RecordConversationSearchInteractionRequest
		wantErr string
	}{
		{name: "selected", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", TelemetrySessionToken: "session", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_SELECTED, StableHitId: "hit-1", SelectedRank: 1}},
		{name: "reformulated", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", TelemetrySessionToken: "session", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_REFORMULATED}},
		{name: "missing token", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_REFORMULATED}, wantErr: "telemetry_session_token"},
		{name: "selected missing hit", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", TelemetrySessionToken: "session", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_SELECTED, SelectedRank: 1}, wantErr: "stable_hit_id"},
		{name: "selected zero rank", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", TelemetrySessionToken: "session", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_SELECTED, StableHitId: "hit-1"}, wantErr: "selected_rank"},
		{name: "ambiguous reformulation", request: &domainpb.RecordConversationSearchInteractionRequest{RequestId: "request-1", TelemetrySessionToken: "session", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_REFORMULATED, StableHitId: "hit-1"}, wantErr: "must be empty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConversationSearchInteraction(test.request)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConversationSearchInteractionConnectHandlerRejectsInvalidShape(t *testing.T) {
	t.Parallel()
	handler := NewConversationSearchConnectHandler(conversationSearchOperationsStub{})
	_, err := handler.RecordConversationSearchInteraction(context.Background(), connect.NewRequest(&domainpb.RecordConversationSearchInteractionRequest{
		RequestId: "request-1", Kind: domainpb.ConversationSearchInteractionKind_CONVERSATION_SEARCH_INTERACTION_KIND_SELECTED,
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConversationSearchCursorContractIsVersionedAndStable(t *testing.T) {
	t.Parallel()

	descriptor := (&domainpb.ConversationSearchCursor{}).ProtoReflect().Descriptor()
	requireFieldNumbers(t, descriptor, map[protoreflect.Name]protoreflect.FieldNumber{
		"version":             1,
		"request_fingerprint": 2,
		"sort":                3,
		"relevance_score":     4,
		"occurred_at":         5,
		"stable_hit_id":       6,
	})
}

func TestListRunsRemainsMetadataOnly(t *testing.T) {
	t.Parallel()

	descriptor := (&apipb.ListRunsRequest{}).ProtoReflect().Descriptor()
	requireFieldNumbers(t, descriptor, map[protoreflect.Name]protoreflect.FieldNumber{
		"status":           1,
		"task_id":          2,
		"agent_profile_id": 3,
		"tag_prefix":       4,
		"limit":            5,
		"offset":           6,
	})
	for i := 0; i < descriptor.Fields().Len(); i++ {
		name := string(descriptor.Fields().Get(i).Name())
		require.False(t, strings.Contains(name, "query"), "ListRuns must remain metadata-only")
	}
}

func TestConversationSearchAdapterMapsLexicalHitAndCoverage(t *testing.T) {
	t.Parallel()
	db, err := sqlx.Connect("sqlite", "file:handler-conversation-search?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(conversationsearch.Schema)))
	repository := conversationsearch.NewSQLiteRepository(db)
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repository.UpsertDocument(context.Background(), conversationsearch.Document{
		DocumentID: "hit-1", SourceRunID: "run-1", SourceEventID: "event-1", SourceMessageID: "message-1",
		ChunkIndex: 0, ChunkTotal: 1, StartByte: 0, EndByte: 16, EventSequence: 1,
		Role: "assistant", OccurredAt: now, Content: "phase correction", ContentClass: conversationsearch.ContentClassProse,
		SourceHash: "source", ContentHash: "content", RecipeVersion: "v1", Harness: "claude-code",
		SourceSessionID: "session-public", ProviderOrigin: "claude", Importer: "agent-manager.transcript-import",
		ProjectScope: "/workspace/project", RunStatus: "complete", RunLabel: "Fixture", EvidenceRef: "agent-manager://runs/run-1/events/event-1",
		Visible: true, IndexedAt: now,
	}))

	adapter := NewConversationSearchAdapter(service)
	response, err := adapter.SearchConversations(context.Background(), &domainpb.SearchConversationsRequest{
		Query: "phase", Mode: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT, PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, response.GetHits(), 1)
	require.Equal(t, "run-1", response.GetHits()[0].GetRunId())
	require.Equal(t, "session-public", response.GetHits()[0].GetProvenance().GetSourceSessionId())
	require.Equal(t, "/runs/run-1?event=event-1", response.GetHits()[0].GetDeepLink())
	require.Equal(t, uint64(1), response.GetCoverage().GetLexicalDocuments())
	require.Equal(t, uint64(1), response.GetCoverage().GetCanonicalVisibleMessages())

	regexResponse, err := adapter.SearchConversations(context.Background(), &domainpb.SearchConversationsRequest{
		Query: `phase.*correction`, Mode: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX, PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, regexResponse.GetHits(), 1)
	require.Equal(t, domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX, regexResponse.GetModeUsed())
	require.Equal(t, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_REGEX, regexResponse.GetHits()[0].GetRankEvidence()[0].GetLeg())

	connectHandler := NewConversationSearchConnectHandler(adapter)
	_, err = connectHandler.SearchConversations(context.Background(), connect.NewRequest(&domainpb.SearchConversationsRequest{
		Query: `(`, Mode: domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX,
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConversationSearchConnectHandlerValidatesFieldsAndMapsDomainErrors(t *testing.T) {
	t.Parallel()
	handler := NewConversationSearchConnectHandler(conversationSearchOperationsStub{
		searchErr:  conversationsearch.ErrInvalidRequest,
		contextErr: conversationsearch.ErrNotFound,
	})

	_, err := handler.SearchConversations(context.Background(), connect.NewRequest(&domainpb.SearchConversationsRequest{
		Query: strings.Repeat("x", 4097),
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = handler.SearchConversations(context.Background(), connect.NewRequest(&domainpb.SearchConversationsRequest{Query: "valid"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = handler.GetConversationContext(context.Background(), connect.NewRequest(&domainpb.GetConversationContextRequest{StableHitId: "missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestRegexPartialDegradationIsMachineReadable(t *testing.T) {
	t.Parallel()
	degradation := regexPartialDegradation(conversationsearch.TextSearchResponse{
		PartialReason: conversationsearch.RegexLimitDeadline, ScannedCandidates: 17, ScannedBytes: 4096,
	})
	require.Equal(t, domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE, degradation.GetReason())
	require.Equal(t, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_REGEX, degradation.GetLeg())
	require.Contains(t, degradation.GetDetail(), "17 candidates")
	require.NotContains(t, degradation.GetDetail(), "query")
}

func TestSemanticRankEvidenceAndDegradationMapToWireContract(t *testing.T) {
	t.Parallel()
	hit := searchHitToProto(conversationsearch.SearchHit{
		Document: conversationsearch.Document{DocumentID: "stable-doc", SourceRunID: "run-1", SourceEventID: "event-1", Role: "assistant", OccurredAt: time.Now().UTC(), Visible: true},
		Score:    0.75, Rank: 1, Weak: true,
		Evidence: []conversationsearch.RankEvidence{
			{Leg: conversationsearch.SearchLegDense, Rank: 1, Score: 0.75, Explanation: "dense"},
			{Leg: conversationsearch.SearchLegSparse, Rank: 2, Score: 0.5, Explanation: "sparse"},
		},
	})
	require.True(t, hit.GetWeak())
	require.Equal(t, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_DENSE, hit.GetRankEvidence()[0].GetLeg())
	require.Equal(t, domainpb.ConversationSearchLeg_CONVERSATION_SEARCH_LEG_SPARSE, hit.GetRankEvidence()[1].GetLeg())

	degradation := searchDegradationToProto(conversationsearch.Degradation{
		Reason: conversationsearch.DegradationIndexLayoutMismatch, Leg: conversationsearch.SearchLegDense, Retryable: false,
	})
	require.Equal(t, domainpb.ConversationSearchDegradationReason_CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_LAYOUT_MISMATCH, degradation.GetReason())
	require.False(t, degradation.GetRetryable())
}

type conversationSearchOperationsStub struct {
	searchErr  error
	contextErr error
}

func (s conversationSearchOperationsStub) SearchConversations(context.Context, *domainpb.SearchConversationsRequest) (*domainpb.SearchConversationsResponse, error) {
	return nil, s.searchErr
}

func (s conversationSearchOperationsStub) GetConversationContext(context.Context, *domainpb.GetConversationContextRequest) (*domainpb.GetConversationContextResponse, error) {
	return nil, s.contextErr
}

func (conversationSearchOperationsStub) GetConversationIndexStatus(context.Context, *domainpb.GetConversationIndexStatusRequest) (*domainpb.GetConversationIndexStatusResponse, error) {
	return nil, errors.New("unimplemented")
}

func (conversationSearchOperationsStub) RecordConversationSearchInteraction(context.Context, *domainpb.RecordConversationSearchInteractionRequest) (*domainpb.RecordConversationSearchInteractionResponse, error) {
	return &domainpb.RecordConversationSearchInteractionResponse{Accepted: true}, nil
}

func requireFieldNumbers(t *testing.T, descriptor protoreflect.MessageDescriptor, expected map[protoreflect.Name]protoreflect.FieldNumber) {
	t.Helper()
	require.Equal(t, len(expected), descriptor.Fields().Len())
	for name, number := range expected {
		field := descriptor.Fields().ByName(name)
		require.NotNilf(t, field, "missing field %s", name)
		require.Equal(t, number, field.Number())
	}
}
