package conversationsearch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
)

// countingStore is a vector store whose point count the test scripts: it can
// hold the first probe open, fail it, or report a number.
type countingStore struct {
	*searchtest.VectorStore
	release chan struct{}
	count   atomic.Int64
	fail    atomic.Pointer[error]
	calls   atomic.Int64
}

func newCountingStore() *countingStore {
	return &countingStore{VectorStore: searchtest.NewVectorStore()}
}

func (s *countingStore) CountPoints(ctx context.Context) (int, error) {
	s.calls.Add(1)
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	if failure := s.fail.Load(); failure != nil {
		return 0, *failure
	}
	return int(s.count.Load()), nil
}

func statusRuntime(store aisearch.VectorStore) *SemanticRuntime {
	return &SemanticRuntime{Engine: aisearch.TunedEngine{VectorStore: store}, Collection: "conversation-test", EmbeddingModel: "test-model"}
}

func expireProbeCache(runtime *SemanticRuntime) {
	runtime.statusMu.Lock()
	runtime.statusUntil = time.Time{}
	runtime.statusMu.Unlock()
}

func TestSemanticStatusProbeInFlightIsUnknownNotZero(t *testing.T) {
	store := newCountingStore()
	store.release = make(chan struct{})
	store.count.Store(59_000)
	runtime := statusRuntime(store)
	runtime.health.EmbeddedDocuments = 58_000

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	count, _, _, _, err := runtime.SemanticStatus(ctx)
	require.ErrorIs(t, err, ErrSemanticCoverageUnknown, "an unfinished first probe is unknown, never a measured zero")
	require.Zero(t, count)
	require.Equal(t, uint64(58_000), runtime.health.EmbeddedDocuments, "unknown coverage must not overwrite a published count")

	close(store.release)
	count, _, _, _, err = runtime.SemanticStatus(context.Background())
	require.NoError(t, err, "the in-flight probe is awaited, not restarted")
	require.Equal(t, uint64(59_000), count)
	require.Equal(t, int64(1), store.calls.Load(), "one probe serves both reads")
}

func TestSemanticStatusFirstProbeFailureIsUnknownThenRecovers(t *testing.T) {
	store := newCountingStore()
	store.count.Store(59_000)
	failure := errors.New("qdrant stalled")
	store.fail.Store(&failure)
	runtime := statusRuntime(store)

	count, _, _, _, err := runtime.SemanticStatus(context.Background())
	require.ErrorIs(t, err, ErrSemanticCoverageUnknown)
	require.ErrorContains(t, err, "qdrant stalled", "the probe failure must travel with the unknown state")
	require.Zero(t, count)
	require.False(t, runtime.statusKnown, "a failed probe is not an observation")

	store.fail.Store(nil)
	expireProbeCache(runtime)
	count, _, _, _, err = runtime.SemanticStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(59_000), count)
	require.True(t, runtime.statusKnown)
}

func TestSemanticStatusLaterFailureKeepsTheLastObservedCount(t *testing.T) {
	store := newCountingStore()
	store.count.Store(59_000)
	runtime := statusRuntime(store)
	count, _, _, _, err := runtime.SemanticStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(59_000), count)

	failure := errors.New("qdrant stalled")
	store.fail.Store(&failure)
	expireProbeCache(runtime)
	count, _, _, _, err = runtime.SemanticStatus(context.Background())
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrSemanticCoverageUnknown, "a count was observed, so coverage is known")
	require.Equal(t, uint64(59_000), count)
	require.Equal(t, uint64(59_000), runtime.health.EmbeddedDocuments)
}

func TestSemanticStatusGenuinelyEmptyCollectionIsZeroCoverage(t *testing.T) {
	store := newCountingStore()
	runtime := statusRuntime(store)
	runtime.health.EmbeddedDocuments = 12

	count, _, _, _, err := runtime.SemanticStatus(context.Background())
	require.ErrorIs(t, err, ErrSemanticUnavailable)
	require.NotErrorIs(t, err, ErrSemanticCoverageUnknown, "an observed empty collection is a measurement")
	require.Zero(t, count)
	require.Zero(t, runtime.health.EmbeddedDocuments, "a measured empty collection is recorded as zero coverage")
}

func TestSemanticStatusReplacedStoreDropsTheOldObservation(t *testing.T) {
	old := newCountingStore()
	old.count.Store(10)
	runtime := statusRuntime(old)
	_, _, _, _, err := runtime.SemanticStatus(context.Background())
	require.NoError(t, err)

	next := newCountingStore()
	next.release = make(chan struct{})
	runtime.recipeMu.Lock()
	runtime.Engine = aisearch.TunedEngine{VectorStore: next}
	runtime.recipeMu.Unlock()
	runtime.statusMu.Lock()
	runtime.statusUntil, runtime.statusKnown, runtime.statusCached = time.Time{}, false, 0
	runtime.statusMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	count, _, _, _, err := runtime.SemanticStatus(ctx)
	require.ErrorIs(t, err, ErrSemanticCoverageUnknown)
	require.Zero(t, count, "the previous collection's count must not describe the new one")
	close(next.release)
}

type unknownCoverageReporter struct {
	count uint64
	err   error
}

func (unknownCoverageReporter) Rebuild(context.Context, string) error { return nil }
func (r unknownCoverageReporter) SemanticStatus(context.Context) (uint64, string, string, string, error) {
	return r.count, "collection", "dense+sparse", "model", r.err
}

func TestIndexerStatusMarksUnobservedSemanticCoverageUnknown(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		count       uint64
		err         error
		wantUnknown bool
		wantCount   uint64
		wantLine    string
	}{
		{"never observed", 0, errors.Join(ErrSemanticCoverageUnknown, errors.New("deadline")), true, 0, SemanticCoverageUnknownPrefix},
		{"observed then failed", 59_000, errors.New("probe failed"), false, 59_000, "semantic: probe failed"},
		{"observed and healthy", 59_000, nil, false, 59_000, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			db := openProjectionTestDB(t)
			applyProjectionSchema(t, db)
			indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{}, Repository: NewSQLiteRepository(db), Semantic: unknownCoverageReporter{count: test.count, err: test.err}})
			require.NoError(t, err)
			status, err := indexer.readStatusSnapshot(t.Context())
			require.NoError(t, err)
			require.Equal(t, test.wantUnknown, status.SemanticCoverageUnknown)
			require.Equal(t, test.wantCount, status.SemanticDocuments)
			joined := ""
			for _, line := range status.DegradedDependencies {
				joined += line + "\n"
			}
			if test.wantLine == "" {
				require.Empty(t, status.DegradedDependencies)
			} else {
				require.Contains(t, joined, test.wantLine)
			}
		})
	}
}
