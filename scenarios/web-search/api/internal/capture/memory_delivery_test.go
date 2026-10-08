package capture

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	learningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"google.golang.org/protobuf/proto"
)

func TestDeliverToMemoryRejectsCorruptOutboxPayload(t *testing.T) {
	attempt := Attempt{Payload: []byte(`{"attempt_id":"attempt-1"}`), PayloadSHA256: "wrong"}
	err := DeliverToMemory(context.Background(), "http://127.0.0.1:1", attempt)
	require.ErrorContains(t, err, "integrity check")
}

func TestDeliverToMemoryRejectsPayloadWithoutIdentity(t *testing.T) {
	payload := []byte(`{"task_id":"task-1"}`)
	err := DeliverToMemory(context.Background(), "http://127.0.0.1:1", Attempt{
		Payload:       payload,
		PayloadSHA256: PayloadHash(payload),
	})
	require.ErrorContains(t, err, "no identity")
}

func TestMemoryOutageReplaysExactlyOnceAfterRecovery(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)))
	clock := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repo := NewRepository(db, func() time.Time { return clock })
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "memory outage", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/proto")
		var response proto.Message = &learningv1.RecordAttemptResponse{EntryId: "entry-1"}
		if r.URL.Path != "/vrooli.memory.v1.LearningService/RecordAttempt" {
			response = &learningv1.RecordObservationResponse{EntryId: "entry-1"}
		}
		body, err := proto.Marshal(response)
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	payload := []byte(`{"attempt_id":"attempt-1","task_id":"task-1","outcome":"verified_success","observation_id":"observation-1","disposition":"supported","provenance":"test","observed_at":"2026-09-05T12:00:00Z"}`)
	_, _, err := repo.Enqueue(context.Background(), "attempt-1", "task-1", "delivery-1", payload)
	require.NoError(t, err)
	delivered, err := repo.Drain(context.Background(), 1, func(ctx context.Context, attempt Attempt) error { return DeliverToMemory(ctx, server.URL, attempt) })
	require.NoError(t, err)
	require.Zero(t, delivered)
	clock = clock.Add(2 * time.Minute)
	delivered, err = repo.Drain(context.Background(), 1, func(ctx context.Context, attempt Attempt) error { return DeliverToMemory(ctx, server.URL, attempt) })
	require.NoError(t, err)
	require.Equal(t, 1, delivered)
	got, err := repo.Get(context.Background(), "attempt-1")
	require.NoError(t, err)
	require.Equal(t, StateDelivered, got.State)
	require.Equal(t, int32(3), requests.Load(), "replay must submit attempt and observation once each after the outage")
}
