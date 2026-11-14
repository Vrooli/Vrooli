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

func loadActiveRecipeRecord(ctx context.Context, catalog aisearch.GenerationCatalog, collection string) (*aisearch.GenerationRecord, error) {
	if catalog == nil {
		return nil, nil
	}
	records, err := catalog.ListGenerationRecords(ctx, conversationCollectionDomain, collection)
	if err != nil {
		return nil, err
	}
	var active *aisearch.GenerationRecord
	for i := range records {
		record := records[i]
		if record.State != aisearch.GenerationStateActive {
			continue
		}
		if record.Metadata.Owner != ConversationSearchProviderID || record.Metadata.Namespace != conversationCollectionDomain || record.Metadata.Alias != collection {
			return nil, fmt.Errorf("active generation %q has mismatched owner identity", record.Metadata.ID)
		}
		if active != nil {
			return nil, fmt.Errorf("multiple active conversation generations are recorded for %q", collection)
		}
		active = &record
	}
	return active, nil
}

func completeGenerationRecipe(metadata aisearch.GenerationMetadata) bool {
	if strings.TrimSpace(metadata.Model) == "" || strings.TrimSpace(metadata.EmbeddingRole) == "" ||
		metadata.EmbeddingDimensions <= 0 || strings.TrimSpace(metadata.PolicySchemaVersion) == "" ||
		strings.TrimSpace(metadata.ChunkPolicy) == "" {
		return false
	}
	return verifyGenerationRecipe(metadata) == nil
}

func verifyGenerationRecipe(metadata aisearch.GenerationMetadata) error {
	if strings.TrimSpace(metadata.RecipeID) == "" || strings.TrimSpace(metadata.RecipeCanonical) == "" {
		return errors.New("durable embedding recipe id and canonical JSON are required")
	}
	recipe, err := aisearch.EmbeddingRecipeFromCanonical([]byte(metadata.RecipeCanonical), metadata.RecipeID)
	if err != nil {
		return err
	}
	input, err := recipe.Input()
	if err != nil {
		return err
	}
	if input.ModelID != metadata.Model || input.Dimensions != metadata.EmbeddingDimensions {
		return errors.New("canonical embedding recipe model/dimensions disagree with generation metadata")
	}
	return nil
}

func tuningForGenerationRecipe(tuning aisearch.TuningConfig, metadata aisearch.GenerationMetadata) aisearch.TuningConfig {
	tuning.EmbedModel = metadata.Model
	tuning.EmbedTaskPrefix = metadata.EmbeddingTaskPrefix
	return tuning
}

func engineFromGenerationRecipe(tuning aisearch.TuningConfig, deps aisearch.EngineDeps, metadata aisearch.GenerationMetadata) aisearch.TunedEngine {
	tuning = tuningForGenerationRecipe(tuning, metadata)
	deps.EmbedRole = metadata.EmbeddingRole
	deps.EmbedModel = metadata.Model
	deps.EmbedDimensions = metadata.EmbeddingDimensions
	deps.EmbedPolicySchemaVersion = metadata.PolicySchemaVersion
	engine := aisearch.NewServiceForTuning(tuning, deps)
	engine.Spec.RecipeID = metadata.RecipeID
	engine.Spec.RecipeCanonical = metadata.RecipeCanonical
	return engine
}

// EmbeddingRecipeRequest is the immutable identity required to construct a
// candidate embedding lane. Values are copied into each durable generation.
type EmbeddingRecipeRequest struct {
	Model               string `json:"model"`
	Role                string `json:"role"`
	Dimensions          int    `json:"dimensions"`
	PolicySchemaVersion string `json:"policySchemaVersion"`
	RecipeID            string `json:"recipe_id"`
	RecipeCanonical     string `json:"recipe_canonical"`
	ChunkPolicy         string `json:"chunkPolicy"`
	TaskPrefix          bool   `json:"taskPrefix"`
}

// GenerationMigrationRecord is the restart-readable catalog projection used
// to census, promote, or restore one exact semantic generation.
type GenerationMigrationRecord struct {
	Generation aisearch.GenerationRecord `json:"generation"`
	Phase      string                    `json:"phase"`
	Ready      bool                      `json:"ready"`
}

type SemanticRecipeMigration interface {
	StageRecipe(context.Context, string, EmbeddingRecipeRequest) (GenerationMigrationRecord, error)
	PromoteRecipe(context.Context, string) error
	RestoreRecipe(context.Context, string) error
	RestoreRecipeCollection(context.Context, string) error
	ListRecipeGenerations(context.Context) ([]GenerationMigrationRecord, error)
}

type SemanticRuntime struct {
	Retriever           SemanticRetriever
	Admission           *aisearch.WeightedAdmission
	Source              *SemanticSource
	Binding             aisearch.SourceBinding
	Engine              aisearch.TunedEngine
	Collection          string
	EmbeddingModel      string
	EmbeddingRecipeID   string
	EmbeddingRecipeJSON string
	GenerationStore     aisearch.GenerationStore
	GenerationCatalog   aisearch.GenerationCatalog
	QdrantBaseURL       string
	QdrantAPIKey        string
	AliasBoundary       aisearch.FleetAliasMutationBoundary
	RerankerURL         string
	RerankerModel       string
	RerankRole          string
	StreamingBinding    aisearch.StreamingBinding
	StreamingReconciler *aisearch.StreamingReconciler
	Tuning              aisearch.TuningConfig
	Projection          ProjectionRepository
	recipeMu            sync.RWMutex
	swapRetriever       *swappableSemanticRetriever
	StagedRepository    *SQLiteRepository
	InitializationError error
	statusMu            sync.Mutex
	statusCached        uint64
	statusUntil         time.Time
	statusErr           error
	statusRefresh       chan struct{}
}

// BuildSemanticRuntime validates authoritative config before wiring optional
// resource clients. Invalid embedding policy is captured as a typed unavailable
// retriever so lexical service can still boot and report precise degradation.
// Transient vector-store failures stay on the live client, allowing later
// requests and indexing retries to recover without an API restart.
func BuildSemanticRuntime(ctx context.Context, options SemanticRuntimeOptions) (*SemanticRuntime, error) {
	if strings.TrimSpace(options.SearchFilePath) == "" || options.Source == nil || options.Projection == nil {
		return nil, fmt.Errorf("search file, source, and projection are required")
	}
	file, err := aisearch.LoadSearchFile(options.SearchFilePath)
	if err != nil {
		return nil, err
	}
	provider, ok := file.Provider(ConversationSearchProviderID)
	if !ok {
		return nil, fmt.Errorf("provider %q is missing from %s", ConversationSearchProviderID, options.SearchFilePath)
	}
	tuning := provider.ResolvedTuning()
	if tuning.Engine != aisearch.EngineHybrid {
		return nil, fmt.Errorf("provider %q must use hybrid dense+sparse retrieval, got %q", ConversationSearchProviderID, tuning.Engine)
	}
	collection, err := corestorage.Collection(conversationCollectionDomain)
	if err != nil {
		return nil, fmt.Errorf("resolve variant-aware conversation collection: %w", err)
	}
	admission := options.Admission
	if admission == nil {
		admission = aisearch.NewWeightedAdmission(defaultAdmissionCapacity)
	}
	runtime := &SemanticRuntime{
		Admission: admission, Source: NewSemanticSource(options.Source), Collection: collection,
		GenerationCatalog: options.GenerationCatalog, Projection: options.Projection,
	}
	runtime.swapRetriever = &swappableSemanticRetriever{current: unavailableSemanticRetriever{err: ErrSemanticUnavailable}}
	runtime.Retriever = runtime.swapRetriever
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
	runtime.QdrantBaseURL, runtime.QdrantAPIKey = operational.QdrantURL, operational.QdrantAPIKey
	runtime.RerankerURL, runtime.RerankerModel, runtime.RerankRole = operational.RerankerURL, operational.RerankerModel, operational.RerankRole
	engineDeps := aisearch.EngineDeps{
		QdrantURL: operational.QdrantURL, QdrantAPIKey: operational.QdrantAPIKey, Collection: collection,
		EmbedRole: operational.EmbedRole, RerankerURL: operational.RerankerURL,
		RerankerModel: operational.RerankerModel, RerankRole: operational.RerankRole,
	}
	engine, engineErr := aisearch.NewServiceForTuningResolved(ctx, tuning, engineDeps)
	activeRecord, activeErr := loadActiveRecipeRecord(ctx, options.GenerationCatalog, collection)
	if activeErr == nil && activeRecord != nil {
		hasCanonicalRecipe := activeRecord.Metadata.RecipeID != "" || activeRecord.Metadata.RecipeCanonical != ""
		if hasCanonicalRecipe {
			if verifyErr := verifyGenerationRecipe(activeRecord.Metadata); verifyErr != nil {
				activeErr = fmt.Errorf("active conversation generation has invalid immutable recipe identity: %w", verifyErr)
			} else if !completeGenerationRecipe(activeRecord.Metadata) {
				activeErr = errors.New("active conversation generation has incomplete embedding recipe metadata")
			} else {
				engine, tuning = engineFromGenerationRecipe(tuning, engineDeps, activeRecord.Metadata), tuningForGenerationRecipe(tuning, activeRecord.Metadata)
				engineErr = nil
			}
		} else if engineErr == nil {
			// Existing active generations predate recipe identity columns. Their
			// serving recipe was the configured role, so persist that known identity
			// once instead of losing semantic search after upgrade.
			activeRecord.Metadata.Model = engine.Spec.Model
			activeRecord.Metadata.ChunkPolicy = DefaultRecipeVersion
			activeRecord.Metadata.EmbeddingRole = engine.Spec.Role
			activeRecord.Metadata.EmbeddingDimensions = engine.Spec.DenseSize
			activeRecord.Metadata.PolicySchemaVersion = engine.Spec.PolicySchemaVersion
			activeRecord.Metadata.EmbeddingTaskPrefix = tuning.EmbedTaskPrefix
			activeRecord.UpdatedAt = time.Now().UTC()
			if saveErr := options.GenerationCatalog.SaveGenerationRecord(ctx, *activeRecord); saveErr != nil {
				activeErr = fmt.Errorf("persist configured identity for active conversation generation: %w", saveErr)
			}
		} else {
			activeErr = errors.New("active conversation generation has no recipe identity and configured embedding policy is unavailable")
		}
	}
	if activeErr != nil {
		engineErr = errors.Join(engineErr, activeErr)
	}
	if engineErr != nil {
		runtime.InitializationError = errors.Join(ErrEmbeddingUnavailable, engineErr)
		runtime.swapRetriever.replace(unavailableSemanticRetriever{err: runtime.InitializationError})
		return runtime, nil
	}
	runtime.Engine = engine
	runtime.Tuning = tuning
	runtime.EmbeddingModel = engine.Spec.Model
	if activeRecord != nil && completeGenerationRecipe(activeRecord.Metadata) {
		runtime.EmbeddingRecipeID = activeRecord.Metadata.RecipeID
		runtime.EmbeddingRecipeJSON = activeRecord.Metadata.RecipeCanonical
	}
	runtime.Binding = aisearch.NewHybridBinding("conversation", "", engine.VectorStore, runtime.Source, aisearch.NewIdentityChunker(), conversationEmbeddingComposer{}, engine.SparseEncoder)
	// Do not freeze the runtime in lexical-only mode based on a startup probe.
	// Qdrant is an optional dependency and may become available after this API
	// starts. Keep the live retriever and indexing pipeline wired; their normal
	// operations report transient Qdrant failures and can recover on later calls.
	aliasBoundary, _ := options.GenerationCatalog.(aisearch.FleetAliasMutationBoundary)
	runtime.AliasBoundary = aliasBoundary
	generationStore, err := aisearch.NewQdrantGenerationStore(aisearch.QdrantGenerationOptions{
		BaseURL: operational.QdrantURL, APIKey: operational.QdrantAPIKey, Alias: collection,
		Namespace: conversationCollectionDomain, Owner: ConversationSearchProviderID,
		Catalog: options.GenerationCatalog, AliasBoundary: aliasBoundary, Spec: engine.Spec,
	})
	if err != nil {
		return nil, err
	}
	runtime.GenerationStore = generationStore
	runtime.StreamingBinding = aisearch.StreamingBinding{
		Kind: "conversation", Store: generationStore, Source: runtime.Source,
		Chunker: aisearch.NewIdentityChunker(), Composer: conversationEmbeddingComposer{}, Sparse: engine.SparseEncoder,
		PageSize: defaultIndexPageSize, Admission: admission, EmbedWeight: 2, EmbedConcurrency: 4,
	}
	runtime.StreamingReconciler = aisearch.NewStreamingReconciler(engine.Embedder)
	retriever, err := runtime.sharedRetrieverFor(engine)
	if err != nil {
		return nil, err
	}
	runtime.swapRetriever.replace(retriever)
	runtime.Retriever = runtime.swapRetriever
	return runtime, nil
}

func (r *SemanticRuntime) generationOwner() (aisearch.GenerationLifecycleOwner, error) {
	if r == nil {
		return nil, errors.New("conversation search generation owner is unavailable")
	}
	r.recipeMu.RLock()
	store := r.GenerationStore
	r.recipeMu.RUnlock()
	if store == nil {
		return nil, errors.New("conversation search generation owner is unavailable")
	}
	owner, ok := store.(aisearch.GenerationLifecycleOwner)
	if !ok {
		return nil, errors.New("conversation search generation store does not expose owner cleanup")
	}
	return owner, nil
}

func (r *SemanticRuntime) sharedRetrieverFor(engine aisearch.TunedEngine) (SemanticRetriever, error) {
	serviceOptions := engine.ServiceOptions()
	serviceOptions.MaxLimit = maximumSearchPageSize
	serviceOptions.DefaultLimit = defaultSearchPageSize
	serviceOptions.Project = func(result aisearch.SearchResult) aisearch.SearchResult {
		if sourceID, ok := result.Payload["source_id"].(string); ok {
			result.SourceID = sourceID
		}
		return result
	}
	return NewSharedSemanticRetriever(aisearch.NewService(serviceOptions), r.Projection, r.Admission)
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
	if r == nil {
		return nil
	}
	r.recipeMu.RLock()
	defer r.recipeMu.RUnlock()
	if r.StreamingReconciler == nil || r.GenerationStore == nil {
		return nil
	}
	binding := r.StreamingBinding
	binding.BeforePromote = beforePromote
	if r.StagedRepository != nil {
		binding.Source = NewStagedSemanticSource(r.StagedRepository, generationID)
	}
	_, err := r.StreamingReconciler.RunFull(ctx, binding, aisearch.GenerationMetadata{
		ID: generationID, CreatedAt: time.Now().UTC(), Model: r.Engine.Spec.Model, ChunkPolicy: DefaultRecipeVersion,
		EmbeddingRole: r.Engine.Spec.Role, EmbeddingDimensions: r.Engine.Spec.DenseSize,
		PolicySchemaVersion: r.Engine.Spec.PolicySchemaVersion, RecipeID: r.EmbeddingRecipeID,
		RecipeCanonical: r.EmbeddingRecipeJSON, EmbeddingTaskPrefix: r.Tuning.EmbedTaskPrefix, Full: true,
	})
	return err
}

func (r *SemanticRuntime) RebuildChanges(ctx context.Context, generationID string, documents []Document, deletedSourceIDs []string, beforePromote func(context.Context) error) error {
	if r == nil {
		return nil
	}
	r.recipeMu.RLock()
	defer r.recipeMu.RUnlock()
	if r.StreamingReconciler == nil || r.GenerationStore == nil {
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
		EmbeddingRole: r.Engine.Spec.Role, EmbeddingDimensions: r.Engine.Spec.DenseSize,
		PolicySchemaVersion: r.Engine.Spec.PolicySchemaVersion, RecipeID: r.EmbeddingRecipeID,
		RecipeCanonical: r.EmbeddingRecipeJSON, EmbeddingTaskPrefix: r.Tuning.EmbedTaskPrefix,
	}, aisearch.ChangeSet{Changes: changes})
	return err
}

func (r *SemanticRuntime) SemanticStatus(ctx context.Context) (uint64, string, string, string, error) {
	if r == nil {
		return 0, "", "", "", ErrSemanticUnavailable
	}
	r.recipeMu.RLock()
	layout := "dense"
	if r.Engine.Spec.Sparse {
		layout = "dense+sparse"
	}
	collection, model, initErr, vectorStore := r.Collection, r.EmbeddingModel, r.InitializationError, r.Engine.VectorStore
	r.recipeMu.RUnlock()
	if initErr != nil {
		return 0, collection, layout, model, initErr
	}
	if vectorStore == nil {
		return 0, collection, layout, model, ErrSemanticUnavailable
	}
	count, err := r.semanticPointCount(ctx, vectorStore)
	return count, collection, layout, model, err
}

func (r *SemanticRuntime) semanticPointCount(ctx context.Context, vectorStore aisearch.VectorStore) (uint64, error) {
	for {
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
			go r.refreshSemanticPointCount(context.WithoutCancel(ctx), refresh, vectorStore)
		}
		r.statusMu.Unlock()
		select {
		case <-refresh:
			continue
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

func (r *SemanticRuntime) refreshSemanticPointCount(parent context.Context, done chan struct{}, vectorStore aisearch.VectorStore) {
	refreshCtx, cancel := context.WithTimeout(parent, semanticStatusRefreshTimeout)
	defer cancel()
	count, err := vectorStore.CountPoints(refreshCtx)
	if count < 0 {
		count = 0
	}
	r.recipeMu.RLock()
	currentStore := vectorStore == r.Engine.VectorStore
	r.recipeMu.RUnlock()
	r.statusMu.Lock()
	if currentStore {
		r.statusCached = uint64(count)
		r.statusErr = err
		r.statusUntil = time.Now().Add(semanticStatusCacheTTL)
	} else {
		r.statusUntil = time.Time{}
	}
	r.statusRefresh = nil
	close(done)
	r.statusMu.Unlock()
}

func (r *SemanticRuntime) Rollback(ctx context.Context, generationID string) error {
	if r == nil {
		return nil
	}
	r.recipeMu.RLock()
	store := r.GenerationStore
	r.recipeMu.RUnlock()
	if store == nil {
		return nil
	}
	return store.RollbackGeneration(ctx, generationID)
}

var _ SemanticRecipeMigration = (*SemanticRuntime)(nil)

func (r *SemanticRuntime) StageRecipe(ctx context.Context, generationID string, recipe EmbeddingRecipeRequest) (GenerationMigrationRecord, error) {
	if r == nil || r.GenerationCatalog == nil || r.Source == nil || r.Admission == nil {
		return GenerationMigrationRecord{}, errors.New("conversation search recipe migration is unavailable")
	}
	generationID = strings.TrimSpace(generationID)
	if generationID == "" {
		return GenerationMigrationRecord{}, errors.New("candidate generation id is required")
	}
	if err := validateEmbeddingRecipe(ctx, recipe); err != nil {
		return GenerationMigrationRecord{}, err
	}
	r.recipeMu.RLock()
	tuning := r.Tuning
	streamingBinding := r.StreamingBinding
	qdrantURL, qdrantKey := r.QdrantBaseURL, r.QdrantAPIKey
	rerankerURL, rerankerModel, rerankRole := r.RerankerURL, r.RerankerModel, r.RerankRole
	r.recipeMu.RUnlock()
	records, err := r.GenerationCatalog.ListGenerationRecords(ctx, conversationCollectionDomain, r.Collection)
	if err != nil {
		return GenerationMigrationRecord{}, fmt.Errorf("list existing conversation generations: %w", err)
	}
	for _, existing := range records {
		if existing.Metadata.ID == generationID {
			return GenerationMigrationRecord{}, fmt.Errorf("generation id %q already exists", generationID)
		}
	}
	tuning.EmbedModel = recipe.Model
	tuning.EmbedTaskPrefix = recipe.TaskPrefix
	deps := aisearch.EngineDeps{
		QdrantURL: qdrantURL, QdrantAPIKey: qdrantKey, Collection: r.Collection,
		EmbedRole: recipe.Role, EmbedModel: recipe.Model, EmbedDimensions: recipe.Dimensions,
		EmbedPolicySchemaVersion: recipe.PolicySchemaVersion,
		RerankerURL:              rerankerURL, RerankerModel: rerankerModel, RerankRole: rerankRole,
	}
	// Check that the target role belongs to the current authoritative policy
	// schema, while allowing its candidate model and dimensionality to differ
	// from today's serving default. The shared engine still creates the embedder
	// that calls resource-ollama's gateway.
	resolved, err := aisearch.ResolveEmbeddingPolicy(ctx, recipe.Role)
	if err != nil {
		return GenerationMigrationRecord{}, fmt.Errorf("resolve candidate embedding role: %w", err)
	}
	if resolved.PolicySchemaVersion != recipe.PolicySchemaVersion {
		return GenerationMigrationRecord{}, errors.New("requested recipe policy schema does not match the current owner policy")
	}
	engine := aisearch.NewServiceForTuning(tuning, deps)
	engine.Spec.RecipeID = recipe.RecipeID
	engine.Spec.RecipeCanonical = recipe.RecipeCanonical
	store, err := r.newRecipeStore(engine.Spec)
	if err != nil {
		return GenerationMigrationRecord{}, err
	}
	binding := streamingBinding
	binding.Store = store
	// Candidate recipes always read the canonical paged source. The staged
	// projection is reserved for normal indexing transactions and can be partial.
	binding.Source = r.Source
	reconciler := aisearch.NewStreamingReconciler(engine.Embedder)
	result, err := reconciler.RunFullStaged(ctx, binding, aisearch.GenerationMetadata{
		ID: generationID, CreatedAt: time.Now().UTC(), Model: recipe.Model,
		ChunkPolicy: recipe.ChunkPolicy, EmbeddingRole: recipe.Role,
		EmbeddingDimensions: recipe.Dimensions, PolicySchemaVersion: recipe.PolicySchemaVersion,
		RecipeID: recipe.RecipeID, RecipeCanonical: recipe.RecipeCanonical,
		EmbeddingTaskPrefix: recipe.TaskPrefix,
		MigrationPhase:      "staging",
		Full:                true,
	})
	if err != nil {
		_ = r.markRecipePhase(context.WithoutCancel(ctx), generationID, "failed", 0, 0)
		return GenerationMigrationRecord{}, err
	}
	if err := r.markRecipePhase(ctx, generationID, "ready", result.Validation.SourceCount, result.Validation.PointCount); err != nil {
		return GenerationMigrationRecord{}, fmt.Errorf("persist candidate readiness: %w", err)
	}
	return r.recipeGeneration(ctx, generationID)
}

func validateEmbeddingRecipe(ctx context.Context, recipe EmbeddingRecipeRequest) error {
	if strings.TrimSpace(recipe.Model) == "" || strings.TrimSpace(recipe.Role) == "" || recipe.Dimensions <= 0 || strings.TrimSpace(recipe.PolicySchemaVersion) == "" || strings.TrimSpace(recipe.ChunkPolicy) == "" {
		return errors.New("embedding recipe requires model, role, positive dimensions, policy schema, and chunk policy")
	}
	if recipe.ChunkPolicy != DefaultRecipeVersion {
		return fmt.Errorf("conversation search supports chunk policy %q, got %q", DefaultRecipeVersion, recipe.ChunkPolicy)
	}
	canonicalRecipe, err := aisearch.EmbeddingRecipeFromCanonical([]byte(recipe.RecipeCanonical), recipe.RecipeID)
	if err != nil {
		return fmt.Errorf("validate immutable embedding recipe: %w", err)
	}
	input, err := canonicalRecipe.Input()
	if err != nil {
		return err
	}
	if input.ModelID != recipe.Model || input.Dimensions != recipe.Dimensions {
		return errors.New("canonical embedding recipe model/dimensions disagree with request fields")
	}
	resolved, err := aisearch.ResolveEmbeddingPolicy(ctx, recipe.Role)
	if err != nil {
		return fmt.Errorf("resolve embedding role %q: %w", recipe.Role, err)
	}
	if resolved.PolicySchemaVersion != recipe.PolicySchemaVersion {
		return fmt.Errorf("requested embedding recipe does not match the current role policy schema")
	}
	return nil
}

func (r *SemanticRuntime) newRecipeStore(spec aisearch.CollectionSpec) (aisearch.GenerationStore, error) {
	return aisearch.NewQdrantGenerationStore(aisearch.QdrantGenerationOptions{
		BaseURL: r.QdrantBaseURL, APIKey: r.QdrantAPIKey, Alias: r.Collection,
		Namespace: conversationCollectionDomain, Owner: ConversationSearchProviderID,
		Catalog: r.GenerationCatalog, AliasBoundary: r.AliasBoundary, Spec: spec,
	})
}

func (r *SemanticRuntime) recipeGeneration(ctx context.Context, generationID string) (GenerationMigrationRecord, error) {
	records, err := r.GenerationCatalog.ListGenerationRecords(ctx, conversationCollectionDomain, r.Collection)
	if err != nil {
		return GenerationMigrationRecord{}, err
	}
	for _, record := range records {
		if record.Metadata.ID == generationID {
			return generationMigrationRecord(record), nil
		}
	}
	return GenerationMigrationRecord{}, fmt.Errorf("generation %q is absent from the durable catalog", generationID)
}

func (r *SemanticRuntime) markRecipePhase(ctx context.Context, generationID, phase string, sources, points int) error {
	entry, err := r.recipeGeneration(ctx, generationID)
	if err != nil {
		return err
	}
	record := entry.Generation
	record.Metadata.MigrationPhase = phase
	record.Metadata.MigrationSources = sources
	record.Metadata.MigrationPoints = points
	record.UpdatedAt = time.Now().UTC()
	return r.GenerationCatalog.SaveGenerationRecord(ctx, record)
}

func (r *SemanticRuntime) PromoteRecipe(ctx context.Context, generationID string) error {
	return r.activateRecipeGeneration(ctx, generationID, false)
}

func (r *SemanticRuntime) RestoreRecipe(ctx context.Context, generationID string) error {
	return r.activateRecipeGeneration(ctx, generationID, true)
}

// RestoreRecipeCollection resolves one exact physical collection from durable
// owner records, then restores that generation through the shared alias owner.
func (r *SemanticRuntime) RestoreRecipeCollection(ctx context.Context, collectionName string) error {
	collectionName = strings.TrimSpace(collectionName)
	if collectionName == "" {
		return errors.New("retained collection name is required")
	}
	records, err := r.ListRecipeGenerations(ctx)
	if err != nil {
		return err
	}
	for _, entry := range records {
		if entry.Generation.CollectionName != collectionName {
			continue
		}
		if entry.Generation.State != aisearch.GenerationStateRetired && entry.Generation.State != aisearch.GenerationStateActive {
			return fmt.Errorf("collection %q is not retained and cannot be restored", collectionName)
		}
		return r.RestoreRecipe(ctx, entry.Generation.Metadata.ID)
	}
	return fmt.Errorf("retained collection %q is not in the durable conversation-search catalog", collectionName)
}

func (r *SemanticRuntime) activateRecipeGeneration(ctx context.Context, generationID string, restore bool) error {
	entry, err := r.recipeGeneration(ctx, strings.TrimSpace(generationID))
	if err != nil {
		return err
	}
	record := entry.Generation
	if record.Metadata.Owner != ConversationSearchProviderID || record.Metadata.Alias != r.Collection || record.Metadata.Namespace != conversationCollectionDomain {
		return fmt.Errorf("generation %q is outside the conversation-search owner boundary", generationID)
	}
	if restore {
		if record.State != aisearch.GenerationStateRetired && record.State != aisearch.GenerationStateActive {
			return fmt.Errorf("generation %q is not an exact retained generation", generationID)
		}
	} else {
		if record.State != aisearch.GenerationStateCandidate {
			return fmt.Errorf("generation %q is not a staged candidate", generationID)
		}
		if record.Metadata.MigrationPhase != "ready" {
			return fmt.Errorf("generation %q is not a validated ready candidate", generationID)
		}
	}
	metadata := record.Metadata
	if metadata.EmbeddingDimensions <= 0 || strings.TrimSpace(metadata.Model) == "" || strings.TrimSpace(metadata.EmbeddingRole) == "" || strings.TrimSpace(metadata.PolicySchemaVersion) == "" {
		return fmt.Errorf("generation %q lacks complete embedding recipe identity", generationID)
	}
	if metadata.RecipeID != "" || metadata.RecipeCanonical != "" || !restore || metadata.MigrationPhase == "ready" {
		if err := verifyGenerationRecipe(metadata); err != nil {
			return fmt.Errorf("generation %q has invalid immutable recipe identity: %w", generationID, err)
		}
	}
	store, err := r.newRecipeStore(aisearch.CollectionSpec{
		Name: r.Collection, DenseSize: metadata.EmbeddingDimensions, DenseDistance: aisearch.DefaultDenseDistance,
		Sparse: true, SparseModifier: aisearch.DefaultSparseModifier, Model: metadata.Model,
		Role: metadata.EmbeddingRole, PolicySchemaVersion: metadata.PolicySchemaVersion,
		RecipeID: metadata.RecipeID, RecipeCanonical: metadata.RecipeCanonical,
	})
	if err != nil {
		return err
	}
	r.recipeMu.Lock()
	defer r.recipeMu.Unlock()
	if !restore {
		if err := r.ensureActiveRecipeIdentity(ctx); err != nil {
			return fmt.Errorf("preserve current serving recipe before cutover: %w", err)
		}
	}
	tuning := r.Tuning
	tuning.EmbedModel = metadata.Model
	tuning.EmbedTaskPrefix = metadata.EmbeddingTaskPrefix
	engine := aisearch.NewServiceForTuning(tuning, aisearch.EngineDeps{
		QdrantURL: r.QdrantBaseURL, QdrantAPIKey: r.QdrantAPIKey, Collection: r.Collection,
		EmbedRole: metadata.EmbeddingRole, EmbedModel: metadata.Model,
		EmbedDimensions: metadata.EmbeddingDimensions, EmbedPolicySchemaVersion: metadata.PolicySchemaVersion,
		RerankerURL: r.RerankerURL, RerankerModel: r.RerankerModel, RerankRole: r.RerankRole,
	})
	engine.Spec.RecipeID = metadata.RecipeID
	engine.Spec.RecipeCanonical = metadata.RecipeCanonical
	retriever, err := r.sharedRetrieverFor(engine)
	if err != nil {
		return err
	}
	cutover := func() error { return store.PromoteGeneration(ctx, metadata.ID) }
	if r.swapRetriever != nil {
		if err := r.swapRetriever.replaceAfter(cutover, retriever); err != nil {
			return err
		}
	} else if err := cutover(); err != nil {
		return err
	}
	r.Engine = engine
	r.Retriever = r.swapRetriever
	r.EmbeddingModel = metadata.Model
	r.EmbeddingRecipeID = metadata.RecipeID
	r.EmbeddingRecipeJSON = metadata.RecipeCanonical
	r.InitializationError = nil
	r.Binding = aisearch.NewHybridBinding("conversation", "", engine.VectorStore, r.Source, aisearch.NewIdentityChunker(), conversationEmbeddingComposer{}, engine.SparseEncoder)
	r.GenerationStore = store
	r.StreamingBinding.Store = store
	r.StreamingBinding.Sparse = engine.SparseEncoder
	r.StreamingReconciler = aisearch.NewStreamingReconciler(engine.Embedder)
	r.Tuning = tuning
	r.statusMu.Lock()
	r.statusUntil = time.Time{}
	r.statusMu.Unlock()
	return nil
}

func (r *SemanticRuntime) ensureActiveRecipeIdentity(ctx context.Context) error {
	records, err := r.GenerationCatalog.ListGenerationRecords(ctx, conversationCollectionDomain, r.Collection)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.State != aisearch.GenerationStateActive || record.Metadata.EmbeddingDimensions > 0 {
			continue
		}
		record.Metadata.Model = r.Engine.Spec.Model
		record.Metadata.ChunkPolicy = DefaultRecipeVersion
		record.Metadata.EmbeddingRole = r.Engine.Spec.Role
		record.Metadata.EmbeddingDimensions = r.Engine.Spec.DenseSize
		record.Metadata.PolicySchemaVersion = r.Engine.Spec.PolicySchemaVersion
		record.Metadata.RecipeID = r.EmbeddingRecipeID
		record.Metadata.RecipeCanonical = r.EmbeddingRecipeJSON
		record.Metadata.EmbeddingTaskPrefix = r.Tuning.EmbedTaskPrefix
		record.UpdatedAt = time.Now().UTC()
		return r.GenerationCatalog.SaveGenerationRecord(ctx, record)
	}
	return nil
}

func (r *SemanticRuntime) ListRecipeGenerations(ctx context.Context) ([]GenerationMigrationRecord, error) {
	if r == nil || r.GenerationCatalog == nil {
		return nil, errors.New("conversation search generation catalog is unavailable")
	}
	records, err := r.GenerationCatalog.ListGenerationRecords(ctx, conversationCollectionDomain, r.Collection)
	if err != nil {
		return nil, err
	}
	result := make([]GenerationMigrationRecord, 0, len(records))
	for _, record := range records {
		result = append(result, generationMigrationRecord(record))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Generation.Metadata.CreatedAt.After(result[j].Generation.Metadata.CreatedAt)
	})
	return result, nil
}

func generationMigrationRecord(record aisearch.GenerationRecord) GenerationMigrationRecord {
	phase := record.Metadata.MigrationPhase
	return GenerationMigrationRecord{Generation: record, Phase: phase, Ready: phase == "ready"}
}

// swappableSemanticRetriever serializes an alias cutover with in-flight local
// searches: no request observes the new Qdrant target with the old embedder.
type swappableSemanticRetriever struct {
	mu      sync.RWMutex
	current SemanticRetriever
}

func (r *swappableSemanticRetriever) SearchSemantic(ctx context.Context, request SemanticSearchRequest) ([]SemanticCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.current == nil {
		return nil, ErrSemanticUnavailable
	}
	return r.current.SearchSemantic(ctx, request)
}

func (r *swappableSemanticRetriever) replace(next SemanticRetriever) {
	r.mu.Lock()
	r.current = next
	r.mu.Unlock()
}

func (r *swappableSemanticRetriever) replaceAfter(cutover func() error, next SemanticRetriever) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := cutover(); err != nil {
		return err
	}
	r.current = next
	return nil
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
	_ SemanticRetriever      = (*swappableSemanticRetriever)(nil)
	_ SemanticRebuilder      = (*SemanticRuntime)(nil)
	_ SemanticStatusReporter = (*SemanticRuntime)(nil)
	_ SemanticRollback       = (*SemanticRuntime)(nil)
)
