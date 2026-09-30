package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	corestorage "github.com/vrooli/api-core/storage"
)

const (
	ConversationSearchProviderID = "agent-manager.runs"
	conversationCollectionDomain = "conversation-search"
	defaultAdmissionCapacity     = int64(8)
	semanticStatusCacheTTL       = time.Minute
	semanticStatusRefreshTimeout = 10 * time.Second
)

type SemanticRuntimeOptions struct {
	SearchFilePath    string
	Source            SourceRepository
	Projection        ProjectionRepository
	Admission         *aisearch.WeightedAdmission
	GenerationCatalog aisearch.GenerationCatalog
}

type SemanticRuntime struct {
	Retriever           SemanticRetriever
	Admission           *aisearch.WeightedAdmission
	Source              *SemanticSource
	Binding             aisearch.SourceBinding
	Engine              aisearch.TunedEngine
	Collection          string
	EmbeddingModel      string
	GenerationStore     aisearch.GenerationStore
	StreamingBinding    aisearch.StreamingBinding
	StreamingReconciler *aisearch.StreamingReconciler
	StagedRepository    *SQLiteRepository
	InitializationError error
	statusMu            sync.Mutex
	statusCached        uint64
	statusUntil         time.Time
	statusErr           error
	statusRefresh       chan struct{}
}

// BuildSemanticRuntime validates authoritative config before attempting any
// optional resource. Configuration errors are returned to the caller; resource
// failures are captured as a typed unavailable retriever so lexical service can
// still boot and report precise degradation on hybrid requests.
func BuildSemanticRuntime(ctx context.Context, options SemanticRuntimeOptions) (SemanticRuntime, error) {
	if strings.TrimSpace(options.SearchFilePath) == "" || options.Source == nil || options.Projection == nil {
		return SemanticRuntime{}, fmt.Errorf("search file, source, and projection are required")
	}
	file, err := aisearch.LoadSearchFile(options.SearchFilePath)
	if err != nil {
		return SemanticRuntime{}, err
	}
	provider, ok := file.Provider(ConversationSearchProviderID)
	if !ok {
		return SemanticRuntime{}, fmt.Errorf("provider %q is missing from %s", ConversationSearchProviderID, options.SearchFilePath)
	}
	tuning := provider.ResolvedTuning()
	if tuning.Engine != aisearch.EngineHybrid {
		return SemanticRuntime{}, fmt.Errorf("provider %q must use hybrid dense+sparse retrieval, got %q", ConversationSearchProviderID, tuning.Engine)
	}
	collection, err := corestorage.Collection(conversationCollectionDomain)
	if err != nil {
		return SemanticRuntime{}, fmt.Errorf("resolve variant-aware conversation collection: %w", err)
	}
	admission := options.Admission
	if admission == nil {
		admission = aisearch.NewWeightedAdmission(defaultAdmissionCapacity)
	}
	runtime := SemanticRuntime{Admission: admission, Source: NewSemanticSource(options.Source), Collection: collection}
	if repository, ok := options.Projection.(*SQLiteRepository); ok {
		runtime.StagedRepository = repository
	}
	operational := aisearch.LoadConfig("")
	// Keep the lifecycle-provided resource binding explicit at the adopter
	// boundary. Besides documenting ownership, this lets dependency analysis
	// prove that Agent Manager consumes the optional Qdrant resource even though
	// the HTTP client itself lives in the shared search package.
	if resourceURL := strings.TrimSpace(os.Getenv("QDRANT_URL")); resourceURL != "" {
		operational.QdrantURL = resourceURL
	}
	engine, err := aisearch.NewServiceForTuningResolved(ctx, tuning, aisearch.EngineDeps{
		QdrantURL: operational.QdrantURL, QdrantAPIKey: operational.QdrantAPIKey, Collection: collection,
		EmbedRole: operational.EmbedRole, RerankerURL: operational.RerankerURL,
		RerankerModel: operational.RerankerModel, RerankRole: operational.RerankRole,
	})
	if err != nil {
		runtime.InitializationError = errors.Join(ErrEmbeddingUnavailable, err)
		runtime.Retriever = unavailableSemanticRetriever{err: runtime.InitializationError}
		return runtime, nil
	}
	runtime.Engine = engine
	runtime.EmbeddingModel = engine.Spec.Model
	runtime.Binding = aisearch.NewHybridBinding("conversation", "", engine.VectorStore, runtime.Source, aisearch.NewIdentityChunker(), conversationEmbeddingComposer{}, engine.SparseEncoder)
	if !engine.VectorStore.Available(ctx) {
		runtime.InitializationError = ErrVectorStoreUnavailable
		runtime.Retriever = unavailableSemanticRetriever{err: runtime.InitializationError}
		return runtime, nil
	}
	generationStore, err := aisearch.NewQdrantGenerationStore(aisearch.QdrantGenerationOptions{BaseURL: operational.QdrantURL, APIKey: operational.QdrantAPIKey, Alias: collection, Namespace: conversationCollectionDomain, Owner: ConversationSearchProviderID, Catalog: options.GenerationCatalog, Spec: engine.Spec})
	if err != nil {
		return SemanticRuntime{}, err
	}
	runtime.GenerationStore = generationStore
	runtime.StreamingBinding = aisearch.StreamingBinding{
		Kind: "conversation", Store: generationStore, Source: runtime.Source,
		Chunker: aisearch.NewIdentityChunker(), Composer: conversationEmbeddingComposer{}, Sparse: engine.SparseEncoder,
		PageSize: defaultIndexPageSize, Admission: admission, EmbedWeight: 2, EmbedConcurrency: 4,
	}
	runtime.StreamingReconciler = aisearch.NewStreamingReconciler(engine.Embedder)
	serviceOptions := engine.ServiceOptions()
	serviceOptions.MaxLimit = maximumSearchPageSize
	serviceOptions.DefaultLimit = defaultSearchPageSize
	serviceOptions.Project = func(result aisearch.SearchResult) aisearch.SearchResult {
		if sourceID, ok := result.Payload["source_id"].(string); ok {
			result.SourceID = sourceID
		}
		return result
	}
	sharedService := aisearch.NewService(serviceOptions)
	retriever, err := NewSharedSemanticRetriever(sharedService, options.Projection, admission)
	if err != nil {
		return SemanticRuntime{}, err
	}
	runtime.Retriever = retriever
	return runtime, nil
}

func (r *SemanticRuntime) generationOwner() (aisearch.GenerationLifecycleOwner, error) {
	if r == nil || r.GenerationStore == nil {
		return nil, errors.New("conversation search generation owner is unavailable")
	}
	owner, ok := r.GenerationStore.(aisearch.GenerationLifecycleOwner)
	if !ok {
		return nil, errors.New("conversation search generation store does not expose owner cleanup")
	}
	return owner, nil
}

func (r *SemanticRuntime) InspectGenerationLifecycle(ctx context.Context, policy aisearch.GenerationRetentionPolicy) (aisearch.GenerationInspection, error) {
	owner, err := r.generationOwner()
	if err != nil {
		return aisearch.GenerationInspection{}, err
	}
	return owner.InspectGenerationLifecycle(ctx, policy)
}

func (r *SemanticRuntime) PreviewGenerationCleanup(ctx context.Context, policy aisearch.GenerationRetentionPolicy, planIdentity string) (aisearch.GenerationCleanupPlan, error) {
	owner, err := r.generationOwner()
	if err != nil {
		return aisearch.GenerationCleanupPlan{}, err
	}
	return owner.PreviewGenerationCleanup(ctx, policy, planIdentity)
}

func (r *SemanticRuntime) ApplyGenerationCleanup(ctx context.Context, plan aisearch.GenerationCleanupPlan) (aisearch.GenerationCleanupReceipt, error) {
	owner, err := r.generationOwner()
	if err != nil {
		return aisearch.GenerationCleanupReceipt{}, err
	}
	return owner.ApplyGenerationCleanup(ctx, plan)
}

func (r *SemanticRuntime) Rebuild(ctx context.Context, generationID string) error {
	return r.RebuildValidated(ctx, generationID, nil)
}

func (r *SemanticRuntime) RebuildValidated(ctx context.Context, generationID string, beforePromote func(context.Context) error) error {
	if r == nil || r.StreamingReconciler == nil || r.GenerationStore == nil {
		return nil
	}
	binding := r.StreamingBinding
	binding.BeforePromote = beforePromote
	if r.StagedRepository != nil {
		binding.Source = NewStagedSemanticSource(r.StagedRepository, generationID)
	}
	_, err := r.StreamingReconciler.RunFull(ctx, binding, aisearch.GenerationMetadata{
		ID: generationID, CreatedAt: time.Now().UTC(), Model: r.Engine.Spec.Model, ChunkPolicy: DefaultRecipeVersion, Full: true,
	})
	return err
}

func (r *SemanticRuntime) RebuildChanges(ctx context.Context, generationID string, documents []Document, deletedSourceIDs []string, beforePromote func(context.Context) error) error {
	if r == nil || r.StreamingReconciler == nil || r.GenerationStore == nil {
		return nil
	}
	upserts := make(map[string]aisearch.SourceDoc)
	for _, document := range documents {
		if semanticIndexableContent(document.ContentClass) {
			upserts[document.DocumentID] = semanticSourceDoc(document)
		}
	}
	changes := make([]aisearch.SourceChange, 0, len(deletedSourceIDs)+len(upserts))
	for _, sourceID := range compactStrings(deletedSourceIDs) {
		if _, replaced := upserts[sourceID]; !replaced {
			changes = append(changes, aisearch.SourceChange{Operation: aisearch.ChangeDelete, SourceID: sourceID})
		}
	}
	ids := make([]string, 0, len(upserts))
	for id := range upserts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		changes = append(changes, aisearch.SourceChange{Operation: aisearch.ChangeUpsert, Document: upserts[id]})
	}
	binding := r.StreamingBinding
	binding.BeforePromote = beforePromote
	// One generation per incremental run. The reconciler pages the change set
	// internally; splitting it into per-page generations here is what minted a
	// full copy of the serving collection for every 250 changes (2026-09-09).
	_, err := r.StreamingReconciler.RunChanges(ctx, binding, aisearch.GenerationMetadata{
		ID: generationID, CreatedAt: time.Now().UTC(), Model: r.Engine.Spec.Model, ChunkPolicy: DefaultRecipeVersion,
	}, aisearch.ChangeSet{Changes: changes})
	return err
}

func (r *SemanticRuntime) SemanticStatus(ctx context.Context) (uint64, string, string, string, error) {
	layout := "dense"
	if r != nil && r.Engine.Spec.Sparse {
		layout = "dense+sparse"
	}
	if r == nil {
		return 0, "", "", "", ErrSemanticUnavailable
	}
	if r.InitializationError != nil {
		return 0, r.Collection, layout, r.EmbeddingModel, r.InitializationError
	}
	if r.Engine.VectorStore == nil {
		return 0, r.Collection, layout, r.EmbeddingModel, ErrSemanticUnavailable
	}
	count, err := r.semanticPointCount(ctx)
	return count, r.Collection, layout, r.EmbeddingModel, err
}

func (r *SemanticRuntime) semanticPointCount(ctx context.Context) (uint64, error) {
	r.statusMu.Lock()
	if !r.statusUntil.IsZero() && time.Now().Before(r.statusUntil) {
		count, err := r.statusCached, r.statusErr
		r.statusMu.Unlock()
		return count, err
	}
	refresh := r.statusRefresh
	if refresh == nil {
		refresh = make(chan struct{})
		r.statusRefresh = refresh
		go r.refreshSemanticPointCount(refresh)
	}
	r.statusMu.Unlock()

	select {
	case <-refresh:
		r.statusMu.Lock()
		count, err := r.statusCached, r.statusErr
		r.statusMu.Unlock()
		return count, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (r *SemanticRuntime) refreshSemanticPointCount(done chan struct{}) {
	refreshCtx, cancel := context.WithTimeout(context.Background(), semanticStatusRefreshTimeout)
	defer cancel()
	count, err := r.Engine.VectorStore.CountPoints(refreshCtx)
	if count < 0 {
		count = 0
	}
	r.statusMu.Lock()
	r.statusCached = uint64(count)
	r.statusErr = err
	r.statusUntil = time.Now().Add(semanticStatusCacheTTL)
	r.statusRefresh = nil
	close(done)
	r.statusMu.Unlock()
}

func (r *SemanticRuntime) Rollback(ctx context.Context, generationID string) error {
	if r == nil || r.GenerationStore == nil {
		return nil
	}
	return r.GenerationStore.RollbackGeneration(ctx, generationID)
}

type unavailableSemanticRetriever struct{ err error }

func (r unavailableSemanticRetriever) SearchSemantic(context.Context, SemanticSearchRequest) ([]SemanticCandidate, error) {
	if r.err == nil {
		return nil, ErrSemanticUnavailable
	}
	return nil, r.err
}

var (
	_ SemanticRetriever      = unavailableSemanticRetriever{}
	_ SemanticRebuilder      = (*SemanticRuntime)(nil)
	_ SemanticStatusReporter = (*SemanticRuntime)(nil)
	_ SemanticRollback       = (*SemanticRuntime)(nil)
)
