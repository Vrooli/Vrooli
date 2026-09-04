package handlers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

func requireFieldNumbers(t *testing.T, descriptor protoreflect.MessageDescriptor, expected map[protoreflect.Name]protoreflect.FieldNumber) {
	t.Helper()
	require.Equal(t, len(expected), descriptor.Fields().Len())
	for name, number := range expected {
		field := descriptor.Fields().ByName(name)
		require.NotNilf(t, field, "missing field %s", name)
		require.Equal(t, number, field.Number())
	}
}
