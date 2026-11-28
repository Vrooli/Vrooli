package conversationsearch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
)

// scriptedPolicyRunner stands in for the resource-ollama CLI: it fails the
// first failFirst invocations the way a CLI killed under host load does.
type scriptedPolicyRunner struct {
	failFirst int32
	calls     atomic.Int32
	payload   []byte
}

func (s *scriptedPolicyRunner) Run(context.Context, string, ...string) ([]byte, error) {
	if s.calls.Add(1) <= s.failFirst {
		return []byte("signal: killed"), errors.New("signal: killed")
	}
	return s.payload, nil
}

func recoveryRuntimeOptions(t *testing.T, runner *scriptedPolicyRunner) SemanticRuntimeOptions {
	t.Helper()
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/aliases":
			_, _ = w.Write([]byte(`{"result":{"aliases":[]}}`))
		case "/collections":
			_, _ = w.Write([]byte(`{"result":{"collections":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(qdrant.Close)
	t.Setenv("QDRANT_URL", qdrant.URL)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "agent-manager")
	_, repository := searchFixtureService(t)
	payload, err := json.Marshal(policyForTest())
	require.NoError(t, err)
	runner.payload = payload
	return SemanticRuntimeOptions{
		SearchFilePath: filepath.Join("..", "..", "..", ".vrooli", "search.json"),
		Source:         sourceRepositoryStub{}, Projection: repository,
		GenerationCatalog: NewSQLiteQdrantCatalog(repository.db),
		Resolver:          ollamapolicy.Resolver{Bin: "resource-ollama", Run: runner},
		RecoveryBackoff:   func(int) time.Duration { return 10 * time.Millisecond },
	}
}

// A start whose policy resolver was killed must not freeze the semantic leg for
// the life of the process: it recovers in place once the resolver answers.
func TestSemanticRuntimeRecoversInPlaceWhenThePolicyResolverFailsAtStart(t *testing.T) {
	runner := &scriptedPolicyRunner{failFirst: 4} // the whole first start (3 tries) plus one retry
	options := recoveryRuntimeOptions(t, runner)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options.Lifecycle = ctx

	runtime, err := BuildSemanticRuntime(ctx, options)
	require.NoError(t, err)
	runtime.recipeMu.RLock()
	failed := runtime.InitializationError
	runtime.recipeMu.RUnlock()
	require.ErrorIs(t, failed, ErrEmbeddingUnavailable, "the start itself must report the unavailable leg")
	_, err = runtime.Retriever.SearchSemantic(ctx, SemanticSearchRequest{Query: "x", Limit: 1})
	require.ErrorIs(t, err, ErrEmbeddingUnavailable, "the retriever must say why while the leg is down")
	require.NotNil(t, runtime.Retriever, "callers hold this retriever, so recovery must swap behind it")

	require.Eventually(t, func() bool {
		runtime.recipeMu.RLock()
		defer runtime.recipeMu.RUnlock()
		return runtime.InitializationError == nil && runtime.StreamingReconciler != nil && runtime.GenerationStore != nil
	}, 5*time.Second, 10*time.Millisecond, "the semantic leg must wire itself once the resolver answers")
	require.Equal(t, testEmbeddingModel, runtime.EmbeddingModel)
	require.Eventually(t, func() bool { return !runtime.recovering.Load() }, time.Second, 10*time.Millisecond, "recovery stops once it has succeeded")
	require.GreaterOrEqual(t, runner.calls.Load(), int32(5))
	// The retriever is wired now: it no longer answers with the stored failure.
	_, err = runtime.Retriever.SearchSemantic(ctx, SemanticSearchRequest{Query: "x", Limit: 1})
	require.NotErrorIs(t, err, ErrEmbeddingUnavailable)
}

func TestSemanticRuntimeThatStartsCleanSchedulesNoRecovery(t *testing.T) {
	runner := &scriptedPolicyRunner{}
	options := recoveryRuntimeOptions(t, runner)
	runtime, err := BuildSemanticRuntime(t.Context(), options)
	require.NoError(t, err)
	require.NoError(t, runtime.InitializationError)
	require.False(t, runtime.recovering.Load())
	require.Equal(t, int32(1), runner.calls.Load(), "a healthy start resolves the policy exactly once")
}

func TestSemanticRuntimeRecoveryStopsWithItsLifecycle(t *testing.T) {
	runner := &scriptedPolicyRunner{failFirst: 1 << 20}
	options := recoveryRuntimeOptions(t, runner)
	ctx, cancel := context.WithCancel(context.Background())
	options.Lifecycle = ctx
	runtime, err := BuildSemanticRuntime(context.Background(), options)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return runtime.recovering.Load() }, time.Second, time.Millisecond)
	cancel()
	require.Eventually(t, func() bool { return !runtime.recovering.Load() }, 3*time.Second, 10*time.Millisecond, "an ended lifecycle must stop the retry loop")
}

// While the embedding engine is not wired, the vectors Qdrant already holds are
// still countable: status reports them instead of a zero it never measured.
func TestSemanticStatusCountsVectorsEvenWhileTheEngineIsDown(t *testing.T) {
	qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections/"+guardTestAlias+"/points/count" {
			_, _ = w.Write([]byte(`{"result":{"count":59593}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer qdrant.Close()
	runtime := &SemanticRuntime{
		Collection: guardTestAlias, QdrantBaseURL: qdrant.URL,
		InitializationError: errors.Join(ErrEmbeddingUnavailable, errors.New("policy resolve killed")),
	}
	count, _, _, _, err := runtime.SemanticStatus(t.Context())
	require.ErrorIs(t, err, ErrEmbeddingUnavailable, "the unwired engine is still a reported degradation")
	require.NotErrorIs(t, err, ErrSemanticCoverageUnknown, "the count was observed")
	require.Equal(t, uint64(59593), count)
}

func TestSemanticStatusWithNoVectorStoreAtAllIsUnknown(t *testing.T) {
	runtime := &SemanticRuntime{Collection: guardTestAlias, InitializationError: ErrEmbeddingUnavailable}
	count, _, _, _, err := runtime.SemanticStatus(t.Context())
	require.ErrorIs(t, err, ErrSemanticCoverageUnknown)
	require.ErrorIs(t, err, ErrEmbeddingUnavailable)
	require.Zero(t, count)
}
