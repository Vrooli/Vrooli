package conversationsearch

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConversationSearchTelemetrySchemaCannotStoreRawContent(t *testing.T) {
	t.Parallel()
	typeOfRecord := reflect.TypeOf(SearchTelemetry{})
	for _, forbidden := range []string{"query", "snippet", "content", "pattern", "path"} {
		for index := 0; index < typeOfRecord.NumField(); index++ {
			require.NotContains(t, strings.ToLower(typeOfRecord.Field(index).Name), forbidden)
		}
	}
	lowerSchema := strings.ToLower(Schema())
	table := lowerSchema[strings.Index(lowerSchema, "create table if not exists conversation_search_telemetry"):]
	for _, forbidden := range []string{"query_text", "snippet", "message_content", "regex_pattern", "transcript_path"} {
		require.NotContains(t, table, forbidden)
	}
}

func TestSQLiteConversationSearchTelemetryAggregatesAndCorrelatesWithoutContent(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	start := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	key := []byte("01234567890123456789012345678901")
	sessionHash := hashTelemetrySession(key, "ephemeral-browser-token")
	for index, latency := range []int{10, 20, 30, 40, 100} {
		record := SearchTelemetry{
			RequestID: "request-" + string(rune('a'+index)), SessionToken: sessionHash,
			Mode: "hybrid", Sort: "relevance", FilterFamilies: []string{"project", "role", "project"},
			Duration: time.Duration(latency) * time.Millisecond, CandidateCount: 5, ResultCount: 2,
			ResultStableHitIDs: []string{"stable-hit-2", "stable-hit-3"},
			WeakOnly:           index == 0, FreshnessBand: "fresh", LexicalContributed: true,
			SemanticContributed: index > 0, CreatedAt: start.Add(time.Duration(index) * time.Minute),
		}
		if index == 4 {
			record.ResultCount = 0
			record.DegradationReasons = []string{"embedding_unavailable"}
			record.ErrorCategory = "deadline"
		}
		require.NoError(t, repository.AppendSearchTelemetry(ctx, record))
	}

	accepted, err := repository.RecordSearchInteraction(ctx, SearchInteraction{RequestID: "request-b", SessionToken: "wrong", Reformulated: true})
	require.NoError(t, err)
	require.False(t, accepted)
	accepted, err = repository.RecordSearchInteraction(ctx, SearchInteraction{RequestID: "request-b", SessionToken: sessionHash, Reformulated: true})
	require.NoError(t, err)
	require.True(t, accepted)
	accepted, err = repository.RecordSearchInteraction(ctx, SearchInteraction{RequestID: "request-c", SessionToken: sessionHash, SelectedRank: 3, StableHitID: "stable-hit-3"})
	require.NoError(t, err)
	require.True(t, accepted)
	accepted, err = repository.RecordSearchInteraction(ctx, SearchInteraction{RequestID: "request-d", SessionToken: sessionHash, SelectedRank: 2, StableHitID: "not-returned"})
	require.NoError(t, err)
	require.False(t, accepted)

	aggregate, err := repository.AggregateSearchTelemetry(ctx, start.Add(-time.Minute), start.Add(time.Hour), 100)
	require.NoError(t, err)
	require.EqualValues(t, 5, aggregate.Queries)
	require.EqualValues(t, 1, aggregate.NoResult)
	require.EqualValues(t, 1, aggregate.WeakOnly)
	require.EqualValues(t, 1, aggregate.Reformulated)
	require.EqualValues(t, 1, aggregate.Selected)
	require.EqualValues(t, 5, aggregate.LexicalContributed)
	require.EqualValues(t, 4, aggregate.SemanticContributed)
	require.EqualValues(t, 1, aggregate.Degraded)
	require.EqualValues(t, 1, aggregate.Errors)
	require.Equal(t, 30.0, aggregate.P50LatencyMS)
	require.Equal(t, 100.0, aggregate.P95LatencyMS)
	require.Equal(t, 3.0, aggregate.P50SelectedRank)

	var storedColumns []string
	require.NoError(t, db.Select(&storedColumns, `SELECT name FROM pragma_table_info('conversation_search_telemetry')`))
	joined := strings.Join(storedColumns, " ")
	require.NotContains(t, joined, "query")
	require.NotContains(t, joined, "content")
	require.Contains(t, joined, "result_stable_hit_ids_json")
}

func TestSQLiteConversationSearchTelemetryReclaimAppliesAgeAndCountBounds(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 6; index++ {
		require.NoError(t, repository.AppendSearchTelemetry(ctx, SearchTelemetry{
			RequestID: "reclaim-" + string(rune('a'+index)), Mode: "text", Sort: "relevance",
			CreatedAt: now.Add(time.Duration(index-3) * time.Hour),
		}))
	}
	removed, err := repository.ReclaimSearchTelemetry(ctx, now.Add(-2*time.Hour), 2)
	require.NoError(t, err)
	require.EqualValues(t, 4, removed)
	var retained int
	require.NoError(t, db.Get(&retained, `SELECT COUNT(*) FROM conversation_search_telemetry`))
	require.Equal(t, 2, retained)
}
