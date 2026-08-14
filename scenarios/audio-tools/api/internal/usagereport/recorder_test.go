package usagereport

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/logx"
	"audio-tools/internal/store"
	db "github.com/vrooli/api-core/databasetest"
)

func newSchemaDB(t *testing.T) *apidb.RoutedDB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	return apidb.NewFromPrimary(d)
}

func TestAsyncRecorder_EnqueueAndDrain(t *testing.T) {
	d := newSchemaDB(t)
	repo := store.NewUsageStore(d)
	var buf bytes.Buffer
	r := New(repo, logx.Std{L: log.New(&buf, "", 0)})
	t.Cleanup(r.Close)

	row := store.UsageRow{
		OperationID: "op-async-1", EmittedAt: time.Now().UTC(),
		Capability: "stt", Operation: "transcribe",
		ProviderTier: "local", ProviderID: "whisper",
	}
	r.Enqueue(row)

	// Polling-via-Eventually is the event-driven substitute for an
	// explicit sleep loop: testify drives the cadence and fails fast.
	require.Eventually(t, func() bool {
		rows, err := repo.ListRecent(context.Background(), time.Now().Add(-1*time.Hour), 10, "", "")
		if err != nil {
			return false
		}
		return len(rows) == 1 && rows[0].OperationID == "op-async-1"
	}, 2*time.Second, 10*time.Millisecond, "row did not appear in store within deadline")
}

func TestAsyncRecorder_RecordSync(t *testing.T) {
	d := newSchemaDB(t)
	repo := store.NewUsageStore(d)
	r := New(repo, logx.Std{L: log.New(&bytes.Buffer{}, "", 0)})
	t.Cleanup(r.Close)

	row := store.UsageRow{
		OperationID: "op-sync-1", EmittedAt: time.Now().UTC(),
		Capability: "tts", Operation: "synthesize",
		ProviderTier: "local", ProviderID: "kokoro",
	}
	require.NoError(t, r.Record(context.Background(), row))
	rows, err := repo.ListRecent(context.Background(), time.Now().Add(-1*time.Hour), 10, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "op-sync-1", rows[0].OperationID)
}

// TestAsyncRecorder_BoundedDropNewest locks the backpressure policy:
// when the queue is full, Enqueue is non-blocking and drops the new
// row. To make the test deterministic, we construct an AsyncRecorder
// without starting the drain goroutine so the channel stays full.
func TestAsyncRecorder_BoundedDropNewest(t *testing.T) {
	var buf bytes.Buffer
	r := &AsyncRecorder{
		repo:    nil, // unused — drain never runs in this test
		logger:  logx.Std{L: log.New(&buf, "", 0)},
		queue:   make(chan store.UsageRow, QueueCapacity),
		retries: []time.Duration{},
	}

	// Fill the queue to exactly capacity.
	for i := 0; i < QueueCapacity; i++ {
		r.Enqueue(store.UsageRow{OperationID: "op", Capability: "stt"})
	}
	preStats := r.Stats()
	require.Equal(t, uint64(QueueCapacity), preStats.EnqueuedTotal)
	require.Equal(t, uint64(0), preStats.DroppedTotal)
	require.Equal(t, QueueCapacity, preStats.QueueDepth)
	require.Equal(t, QueueCapacity, preStats.QueueCapacity)

	// Next 7 enqueues must drop, not block.
	for i := 0; i < 7; i++ {
		r.Enqueue(store.UsageRow{OperationID: "drop", Capability: "stt"})
	}
	post := r.Stats()
	require.Equal(t, uint64(7), post.DroppedTotal, "drop-newest policy: extra enqueues increment DroppedTotal")
	require.Equal(t, uint64(QueueCapacity), post.EnqueuedTotal, "EnqueuedTotal must not advance on drop")
	require.Contains(t, buf.String(), "queue full")
}

func TestAsyncRecorder_CloseDrainsPending(t *testing.T) {
	d := newSchemaDB(t)
	repo := store.NewUsageStore(d)
	r := New(repo, logx.Std{L: log.New(&bytes.Buffer{}, "", 0)})

	for i := 0; i < 10; i++ {
		r.Enqueue(store.UsageRow{
			OperationID: time.Now().Format("op-150405.000000000-") + string(rune('a'+i)),
			Capability:  "stt", Operation: "transcribe",
			ProviderTier: "local", ProviderID: "whisper",
		})
	}
	r.Close() // must block until drain goroutine returns

	rows, err := repo.ListRecent(context.Background(), time.Now().Add(-1*time.Hour), 100, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 10)
}
