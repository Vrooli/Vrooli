package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
	corestorage "github.com/vrooli/api-core/storage"
)

const (
	ConversationSearchProviderID = "agent-manager.runs"
	conversationCollectionDomain = "conversation-search"
	defaultAdmissionCapacity     = int64(8)
	semanticStatusCacheTTL       = time.Minute
	// A failed point-count probe is retried sooner than a good observation, so a
	// transient Qdrant/host stall does not hide coverage for a full minute.
	semanticStatusErrorTTL       = 15 * time.Second
	semanticStatusRefreshTimeout = 20 * time.Second
	// The eligible count is an indexed read; this only bounds a wedged writer.
	semanticEligibleTimeout = 5 * time.Second
)

type SemanticRuntimeOptions struct {
	SearchFilePath    string
	Source            SourceRepository
	Projection        ProjectionRepository
	Admission         *aisearch.WeightedAdmission
	GenerationCatalog aisearch.GenerationCatalog
	// Resolver resolves the embedding policy; the zero value runs the
	// resource-ollama CLI. Tests inject a fake runner.
	Resolver ollamapolicy.Resolver
	// Lifecycle ends background recovery of a start that could not finish
	// (default: the process lifetime).
	Lifecycle context.Context
	// RecoveryBackoff spaces those retries (default: defaultRecoveryBackoff).
	RecoveryBackoff func(attempt int) time.Duration
}

func loadActiveRecipeRecord(ctx context.Context, catalog aisearch.GenerationCatalog, collection string) (*aisearch.GenerationRecord, error) {
	if catalog == nil {
		return nil, nil
	}
	actives, err := listActiveGenerationRecords(ctx, catalog, collection)
	if err != nil {
		return nil, err
	}
	if len(actives) > 1 {
		// Legacy catalogs hold duplicate active rows because every promotion used
		// to retire only the alias target it could still resolve. Reconcile when
		// the catalog can, then read the single survivor; never fail the whole
		// semantic path just because history left stale rows behind.
		if reconciler, ok := catalog.(ActiveGenerationReconciler); ok {
			if _, reconcileErr := reconciler.ReconcileActiveGenerations(ctx, conversationCollectionDomain, collection, GenerationReconcileOptions{
				Reason: "load-active-recipe",
			}); reconcileErr != nil {
				return nil, reconcileErr
			}
			if actives, err = listActiveGenerationRecords(ctx, catalog, collection); err != nil {
				return nil, err
			}
		}
	}
	if len(actives) == 0 {
		return nil, nil
	}
	if len(actives) == 1 {
		// One active row is unambiguous by definition. Return it even when it
		// predates recipe identity: the runtime adopts the configured policy for
		// that legacy shape and persists it, instead of dropping a serving alias.
		selected := actives[0]
		return &selected, nil
	}
	choice, err := chooseActiveGeneration(actives, "", false)
	if err != nil {
		return nil, fmt.Errorf("%w for %q", err, collection)
	}
	if choice.Survivor == nil {
		return nil, nil
	}
	selected := *choice.Survivor
	return &selected, nil
}

func listActiveGenerationRecords(ctx context.Context, catalog aisearch.GenerationCatalog, collection string) ([]aisearch.GenerationRecord, error) {
	records, err := catalog.ListGenerationRecords(ctx, conversationCollectionDomain, collection)
	if err != nil {
		return nil, err
	}
	actives := make([]aisearch.GenerationRecord, 0, 1)
	for _, record := range records {
		if record.State != aisearch.GenerationStateActive {
			continue
		}
		if record.Metadata.Owner != ConversationSearchProviderID || record.Metadata.Namespace != conversationCollectionDomain || record.Metadata.Alias != collection {
			return nil, fmt.Errorf("active generation %q has mismatched owner identity", record.Metadata.ID)
		}
		actives = append(actives, record)
	}
	return actives, nil
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

// resolveSemanticEngine builds the serving engine from the authoritative
// embedding policy and returns the policy it resolved, whose runtime artifact
// identity is what a recipe is minted from. Policy resolution shells out to the
// ollama resource and has been killed under host contention, which used to
// disable semantic search for the whole process lifetime; a short bounded retry
// keeps a transient resolver failure from taking the semantic leg down with it.
func resolveSemanticEngine(ctx context.Context, resolver ollamapolicy.Resolver, tuning aisearch.TuningConfig, deps aisearch.EngineDeps) (aisearch.TunedEngine, ollamapolicy.ResolvedRole, error) {
	role := strings.TrimSpace(deps.EmbedRole)
	if role == "" {
		role = aisearch.DefaultEmbedRole
	}
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		var resolved ollamapolicy.ResolvedRole
		resolved, err = resolver.ResolveRole(ctx, role)
		if err == nil {
			var cfg aisearch.Config
			if cfg, err = aisearch.ConfigWithResolvedEmbedding(aisearch.Config{EmbedRole: role}, resolved); err == nil {
				deps.EmbedRole, deps.EmbedModel = cfg.EmbedRole, cfg.EmbedModel
				deps.EmbedDimensions, deps.EmbedPolicySchemaVersion = cfg.EmbedDimensions, cfg.EmbedPolicySchemaVersion
				return aisearch.NewServiceForTuning(tuning, deps), resolved, nil
			}
		}
		if ctx.Err() != nil {
			return aisearch.TunedEngine{}, ollamapolicy.ResolvedRole{}, err
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return aisearch.TunedEngine{}, ollamapolicy.ResolvedRole{}, err
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
		}
	}
	return aisearch.TunedEngine{}, ollamapolicy.ResolvedRole{}, err
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
	// buildMu serializes semantic builds. recipeMu is a read lock for a running
	// build (so a promotion waits for it), which by itself would let two builds
	// write Qdrant concurrently.
	buildMu             sync.Mutex
	swapRetriever       *swappableSemanticRetriever
	StagedRepository    *SQLiteRepository
	InitializationError error
	// servingCollection is the Qdrant alias target observed at startup. It is
	// the only external evidence of which physical collection serves, so it
	// settles a catalog with divergent complete recipes instead of guessing.
	servingCollection string
	// recipeAdoptionNote says why a recipe-less serving generation was not
	// stamped with an immutable recipe at startup; empty once it carries one.
	recipeAdoptionNote string
	healthMu           sync.Mutex
	health             SemanticGenerationHealth
	// start is what a bind attempt needs; recovering keeps one retry loop.
	start            semanticStart
	recovering       atomic.Bool
	healthObservedAt time.Time
	statusMu         sync.Mutex
	statusCached     uint64
	statusUntil      time.Time
	statusErr        error
	// statusKnown is true once one point-count probe has succeeded for the
	// current vector store. A failed or in-flight first probe leaves it false,
	// and coverage is then unknown rather than zero.
	statusKnown   bool
	statusRefresh chan struct{}
	// countStore counts vectors while the engine is not wired.
	countStore aisearch.VectorStore
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
	lifecycle := options.Lifecycle
	if lifecycle == nil {
		lifecycle = context.Background()
	}
	backoff := options.RecoveryBackoff
	if backoff == nil {
		backoff = defaultRecoveryBackoff
	}
	runtime.start = semanticStart{
		tuning: tuning, deps: engineDeps, admission: admission, resolver: options.Resolver,
		lifecycle: lifecycle, backoff: backoff,
	}
	// A start that cannot finish (the policy resolver was killed under host
	// load, Qdrant was not up yet) used to freeze the semantic leg for the whole
	// process lifetime, so every later generation failed with "embedding
	// unavailable" until someone restarted the API. The failure is recorded and
	// the bind is retried in the background instead.
	if err := runtime.bind(ctx, "runtime-startup"); err != nil {
		runtime.scheduleRecovery()
	}
	return runtime, nil
}

// resumePublishedSemanticGeneration continues the semantic leg of a published
// generation whose embed was interrupted. A full embed of the conversation
// corpus takes hours on this host, so the corpus is never re-embedded from zero
// after a restart: the candidate collection keeps its vectors and the streaming
// reconciler reuses every already-staged source. The lexical activation only
// happens after the semantic leg succeeds.
func (r *SemanticRuntime) resumePublishedSemanticGeneration(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("conversation search: semantic resume skipped after panic: %v", recovered)
		}
	}()
	if r == nil || r.StagedRepository == nil || r.GenerationStore == nil {
		return
	}
	published, ok := r.publishedSemanticCandidate(ctx)
	if !ok {
		return
	}
	changeWatermark, err := r.StagedRepository.MaxPendingChangeSequence(ctx)
	if err != nil {
		return
	}
	go func() {
		// A background resume must never take the API down: it is an optimization,
		// and every failure leaves the current serving generation untouched.
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("conversation search: resumed generation %s panicked: %v", published.GenerationID, recovered)
			}
		}()
		buildCtx, cancel := buildContext(ctx, published.PlannedDocuments)
		defer cancel()
		validate := func(checkCtx context.Context) error {
			deleted, err := r.StagedRepository.HasPendingDeletionAfter(checkCtx, changeWatermark)
			if err != nil {
				return err
			}
			if deleted {
				return errCanonicalDeletionDuringSemanticBuild
			}
			return nil
		}
		if err := r.RebuildValidated(buildCtx, published.GenerationID, validate); err != nil {
			log.Printf("conversation search: resuming published generation %s failed: %v", published.GenerationID, err)
			return
		}
		if err := r.StagedRepository.ActivateGeneration(context.WithoutCancel(ctx), published.GenerationID, time.Now().UTC()); err != nil {
			log.Printf("conversation search: activate resumed generation %s: %v", published.GenerationID, err)
			return
		}
		log.Printf("conversation search: resumed and activated generation %s", published.GenerationID)
	}()
}

// publishedSemanticCandidate returns the newest published generation that
// already owns embedded vectors and therefore can be resumed.
func (r *SemanticRuntime) publishedSemanticCandidate(ctx context.Context) (Generation, bool) {
	generations, err := r.StagedRepository.ReadyGenerations(ctx)
	if err != nil {
		return Generation{}, false
	}
	for _, generation := range generations {
		if r.CanResumeSemanticGeneration(ctx, generation.GenerationID) {
			return generation, true
		}
	}
	return Generation{}, false
}

// buildContext bounds one semantic build by the corpus size. It is detached
// from the request context (the API must keep serving) but cancelled on
// shutdown through the runtime's parent context when available.
func buildContext(ctx context.Context, planned uint64) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), semanticRebuildBudget(planned))
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
	r.buildMu.Lock()
	defer r.buildMu.Unlock()
	if r.StreamingReconciler == nil || r.GenerationStore == nil {
		// Never report a successful semantic rebuild that did nothing. The
		// optional leg is genuinely unavailable, and the caller must record the
		// degradation instead of promoting a "complete" generation with no
		// vectors.
		return r.semanticLegUnavailable("conversation semantic rebuild")
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
	// Snapshot the recipe and release the read lock before the incremental
	// pass. buildMu still serializes builds; a cutover that lands mid-pass
	// replaces the serving fields without invalidating this snapshot.
	r.recipeMu.RLock()
	if r.StreamingReconciler == nil || r.GenerationStore == nil {
		r.recipeMu.RUnlock()
		return r.semanticLegUnavailable("conversation semantic change rebuild")
	}
	binding := r.StreamingBinding
	reconciler := r.StreamingReconciler
	model, role, dimensions := r.Engine.Spec.Model, r.Engine.Spec.Role, r.Engine.Spec.DenseSize
	policySchema, recipeID, recipeJSON := r.Engine.Spec.PolicySchemaVersion, r.EmbeddingRecipeID, r.EmbeddingRecipeJSON
	taskPrefix := r.Tuning.EmbedTaskPrefix
	r.recipeMu.RUnlock()

	r.buildMu.Lock()
	defer r.buildMu.Unlock()
	binding.BeforePromote = beforePromote
	// One generation per incremental run. The reconciler pages the change set
	// internally; splitting it into per-page generations here is what minted a
	// full copy of the serving collection for every 250 changes (2026-09-09).
	_, err := reconciler.RunChanges(ctx, binding, aisearch.GenerationMetadata{
		ID: generationID, CreatedAt: time.Now().UTC(), Model: model, ChunkPolicy: DefaultRecipeVersion,
		EmbeddingRole: role, EmbeddingDimensions: dimensions,
		PolicySchemaVersion: policySchema, RecipeID: recipeID,
		RecipeCanonical: recipeJSON, EmbeddingTaskPrefix: taskPrefix,
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
	baseURL, apiKey := r.QdrantBaseURL, r.QdrantAPIKey
	r.recipeMu.RUnlock()
	// A semantic leg that is not wired (embedding policy unresolved) is a
	// degradation to report, but it does not make the vectors Qdrant already
	// holds unknowable: count them through a bare client on the alias.
	var configErr error
	switch {
	case initErr != nil:
		configErr = initErr
	case vectorStore == nil:
		configErr = ErrSemanticUnavailable
	}
	if vectorStore == nil {
		vectorStore = r.fallbackCountStore(baseURL, apiKey, collection)
	}
	if vectorStore == nil {
		healthErr := r.maintainGenerationHealth(ctx)
		return 0, collection, layout, model, errors.Join(ErrSemanticCoverageUnknown, configErr, healthErr)
	}
	// Read the cheap vector count before the generation reconciliation. A
	// healthy catalog still has to prove itself, and that proof can outlast the
	// status budget under writer contention; it must not turn a serving 98.8%
	// index into a reported zero.
	count, known, probeErr := r.semanticPointCount(ctx, vectorStore)
	healthErr := r.maintainGenerationHealth(ctx)
	var err error
	switch {
	case !known:
		// No probe has ever completed for this vector store: coverage is
		// unknown, which is neither zero nor healthy. The cause (in flight,
		// timed out, store down) rides along for the operator.
		cause := probeErr
		if cause == nil {
			cause = errors.New("the first vector count has not completed")
		}
		err = fmt.Errorf("%w: collection %s: %w", ErrSemanticCoverageUnknown, collection, cause)
	case probeErr != nil:
		// A later probe failed; the last observed count is still reported.
		err = probeErr
	case count == 0:
		// Empty is never healthy. A generation can be structurally valid and still
		// hold no vectors (an interrupted embed, a fresh promotion); say so instead
		// of reporting a clean status with 0% semantic coverage.
		err = fmt.Errorf("%w: collection %s holds no semantic vectors; the active generation has not been embedded", ErrSemanticUnavailable, collection)
	}
	r.recordSemanticCoverage(ctx, count, known)
	return count, collection, layout, model, errors.Join(configErr, err, healthErr)
}

// fallbackCountStore is a bare vector-store client on the serving alias, used
// only to count vectors while the embedding engine is not wired.
func (r *SemanticRuntime) fallbackCountStore(baseURL, apiKey, collection string) aisearch.VectorStore {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(collection) == "" {
		return nil
	}
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	if r.countStore == nil {
		r.countStore = aisearch.NewVectorStore(baseURL, apiKey, collection)
	}
	return r.countStore
}

// recordSemanticCoverage publishes what the semantic leg actually covers
// against the chunks it is supposed to embed, so status never reports a
// catalog-wide ratio as if tool payloads were embeddable. An unknown count
// (no probe has completed) leaves the published value untouched; a known zero
// is a genuinely empty collection and is recorded as such.
func (r *SemanticRuntime) recordSemanticCoverage(ctx context.Context, embedded uint64, known bool) {
	if r == nil {
		return
	}
	eligible := r.semanticEligibleDocuments(ctx)
	r.healthMu.Lock()
	if known {
		r.health.EmbeddedDocuments, r.health.EmbeddedObserved = embedded, true
	}
	if eligible > 0 {
		r.health.EligibleDocuments = eligible
	}
	r.healthMu.Unlock()
}

// semanticEligibleDocuments counts the chunks the semantic leg is supposed to
// embed. The count rides the content-class covering index (about 20ms over the
// 1.2M-row catalog); it used to be cached behind a detached refresh because it
// once scanned every row. A failed or timed-out count returns 0 and the caller
// keeps the last denominator.
func (r *SemanticRuntime) semanticEligibleDocuments(ctx context.Context) uint64 {
	repository, ok := r.Projection.(*SQLiteRepository)
	if !ok {
		return 0
	}
	countCtx, cancel := context.WithTimeout(ctx, semanticEligibleTimeout)
	defer cancel()
	eligible, err := repository.CountSemanticEligible(countCtx)
	if err != nil {
		return 0
	}
	return eligible
}

// semanticLegUnavailable is the typed reason an un-wired semantic leg refuses
// to take part in a rebuild. A silent nil here used to make the indexer mark a
// generation "complete" with zero vectors.
func (r *SemanticRuntime) semanticLegUnavailable(operation string) error {
	err := errors.New(operation + " is unavailable")
	if r != nil && r.InitializationError != nil {
		return errors.Join(ErrEmbeddingUnavailable, err, r.InitializationError)
	}
	return errors.Join(ErrSemanticUnavailable, err)
}

// GenerationHealthSnapshot reports the last observed generation health.
func (r *SemanticRuntime) GenerationHealthSnapshot() SemanticGenerationHealth {
	if r == nil {
		return SemanticGenerationHealth{State: GenerationHealthUnavailable, Detail: "conversation semantic runtime is unavailable"}
	}
	r.healthMu.Lock()
	defer r.healthMu.Unlock()
	health := r.health
	health.Findings = append([]string(nil), health.Findings...)
	if health.State == "" {
		health.State = GenerationHealthUnknown
		health.Detail = "conversation generation catalog has not been reconciled in this process"
	}
	return health
}

// recordGenerationHealth folds a reconciliation report and the adopted active
// record into the runtime's health. It never upgrades an ambiguous or
// unavailable state to healthy.
func (r *SemanticRuntime) recordGenerationHealth(report ActiveGenerationReconciliation, supported bool, reconcileErr error, activeRecord *aisearch.GenerationRecord) {
	if r == nil {
		return
	}
	now := report.ObservedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	health := SemanticGenerationHealth{
		ObservedAt:        now,
		ActiveGenerations: report.ActiveBefore,
		RetiredDuplicates: len(report.Retired),
		SurvivorID:        report.SurvivorID,
		Findings:          append([]string(nil), report.Findings...),
		RecipeValid:       report.RecipeValid,
	}
	if !supported {
		health.State = GenerationHealthUnknown
		health.Detail = "generation catalog does not expose reconciliation, so the one-active invariant cannot be verified"
	} else if reconcileErr != nil {
		if errors.Is(reconcileErr, ErrAmbiguousActiveGenerations) {
			health.State = GenerationHealthAmbiguous
		} else {
			health.State = GenerationHealthUnavailable
		}
		health.Detail = reconcileErr.Error()
	} else if activeRecord != nil && completeGenerationRecipe(activeRecord.Metadata) {
		health.SurvivorID = activeRecord.Metadata.ID
		health.SurvivorCollection = activeRecord.CollectionName
		health.RecipeValid = true
		if len(report.Retired) > 0 {
			health.State = GenerationHealthRecovered
			health.Detail = fmt.Sprintf("reconciled %d duplicate active generation(s); serving %s", len(report.Retired), activeRecord.Metadata.ID)
		} else {
			health.State = GenerationHealthHealthy
		}
	} else if activeRecord != nil && generationServingIdentity(activeRecord.Metadata) {
		health.SurvivorID = activeRecord.Metadata.ID
		health.SurvivorCollection = activeRecord.CollectionName
		health.RecipeValid = false
		health.State = GenerationHealthRecovered
		health.Detail = fmt.Sprintf("serving %s by its recorded identity; the generation predates immutable recipe identity and none was invented", activeRecord.Metadata.ID)
		if note := r.adoptionNote(); note != "" {
			health.Detail += "; " + note
		}
	} else if activeRecord != nil {
		health.State = GenerationHealthMissingRecipe
		health.Detail = fmt.Sprintf("active generation %s has no usable embedding identity", activeRecord.Metadata.ID)
	} else if len(report.Retired) > 0 {
		health.State = GenerationHealthMissingRecipe
		health.Detail = fmt.Sprintf("no active generation has a usable embedding identity; retired %d stale row(s) so a rebuild can mint a fresh generation", len(report.Retired))
	} else {
		health.State = GenerationHealthMissingRecipe
		health.Detail = "no active conversation generation has a usable embedding identity"
	}
	r.healthMu.Lock()
	// Retirements stay visible for as long as the same generation serves: it is
	// the in-process evidence that the catalog needed repair. A promotion to a
	// different generation clears it.
	if r.health.State == GenerationHealthRecovered && health.State == GenerationHealthHealthy &&
		health.SurvivorID == r.health.SurvivorID {
		health.State = GenerationHealthRecovered
		health.Detail = r.health.Detail
		health.RetiredDuplicates = r.health.RetiredDuplicates
	}
	// Coverage is measured separately from catalog health; a reconciliation pass
	// must not erase the last observed counts.
	health.EmbeddedDocuments, health.EmbeddedObserved, health.EligibleDocuments = r.health.EmbeddedDocuments, r.health.EmbeddedObserved, r.health.EligibleDocuments
	r.health = health
	r.healthObservedAt = time.Now().UTC()
	r.healthMu.Unlock()
}

// maintainGenerationHealth re-reconciles the catalog on the status path so a
// regression is repaired and reported. It writes only when the catalog is
// already broken (or once, to install the invariant).
func (r *SemanticRuntime) maintainGenerationHealth(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.recipeMu.RLock()
	catalog, collection, serving := r.GenerationCatalog, r.Collection, r.servingCollection
	r.recipeMu.RUnlock()
	if catalog == nil {
		return nil
	}
	r.healthMu.Lock()
	observedAt, state := r.healthObservedAt, r.health.State
	r.healthMu.Unlock()
	if !observedAt.IsZero() && time.Since(observedAt) < generationHealthRefreshInterval &&
		state != GenerationHealthAmbiguous && state != GenerationHealthMissingRecipe {
		return nil
	}
	r.recipeMu.RLock()
	baseURL, apiKey := r.QdrantBaseURL, r.QdrantAPIKey
	r.recipeMu.RUnlock()
	// Refresh serving truth; a promotion may have happened since startup, and
	// the alias is what settles a divergent catalog.
	if resolved := resolveServingAliasCollection(ctx, baseURL, apiKey, collection); resolved != "" {
		serving = resolved
		r.recipeMu.Lock()
		r.servingCollection = resolved
		r.recipeMu.Unlock()
	}
	reconcileCtx, cancel := context.WithTimeout(ctx, generationReconcileTimeout)
	defer cancel()
	report, supported, err := reconcileGenerationCatalog(reconcileCtx, catalog, collection, GenerationReconcileOptions{
		AuthoritativeCollection: serving,
		Reason:                  "status",
	})
	if !supported {
		r.recordGenerationHealth(report, supported, nil, nil)
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// Inconclusive probe (the status budget is short and the host may be busy).
		// Keep the last observation and retry on the next status call instead of
		// reporting a catalog fault that did not happen.
		return nil
	}
	var activeRecord *aisearch.GenerationRecord
	var activeErr error
	if err == nil {
		activeRecord, activeErr = loadActiveRecipeRecord(reconcileCtx, catalog, collection)
	}
	r.recordGenerationHealth(report, supported, err, activeRecord)
	switch {
	case err != nil:
		return fmt.Errorf("conversation generation reconciliation failed: %w", err)
	case activeErr != nil:
		return fmt.Errorf("conversation generation catalog is unreadable: %w", activeErr)
	}
	health := r.GenerationHealthSnapshot()
	if health.State == GenerationHealthMissingRecipe {
		return errors.New(health.Detail)
	}
	return nil
}

// semanticPointCount returns the cached vector count. known reports whether a
// probe has ever succeeded for this vector store; while it is false the count
// is meaningless and callers must treat coverage as unknown. After a first
// success, a failed refresh keeps the last good count and reports its error.
func (r *SemanticRuntime) semanticPointCount(ctx context.Context, vectorStore aisearch.VectorStore) (count uint64, known bool, err error) {
	for {
		r.statusMu.Lock()
		if !r.statusUntil.IsZero() && time.Now().Before(r.statusUntil) {
			count, known, err = r.statusCached, r.statusKnown, r.statusErr
			r.statusMu.Unlock()
			return count, known, err
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
			// A status budget that expires mid-probe is unknown, not zero. A
			// previous success is reported as the last good value with the
			// error so coverage never flips a serving index to 0%.
			r.statusMu.Lock()
			count, known = r.statusCached, r.statusKnown
			r.statusMu.Unlock()
			return count, known, ctx.Err()
		}
	}
}

func (r *SemanticRuntime) refreshSemanticPointCount(parent context.Context, done chan struct{}, vectorStore aisearch.VectorStore) {
	refreshCtx, cancel := context.WithTimeout(parent, semanticStatusRefreshTimeout)
	defer cancel()
	count, err := vectorStore.CountPoints(refreshCtx)
	if err == nil && count < 0 {
		count = 0
	}
	r.recipeMu.RLock()
	engineStore := r.Engine.VectorStore
	r.recipeMu.RUnlock()
	r.statusMu.Lock()
	// The probed store still serves if it is the wired engine's, or the bare
	// fallback client while no engine is wired.
	currentStore := vectorStore == engineStore || (engineStore == nil && vectorStore == r.countStore)
	if currentStore {
		if err == nil {
			r.statusCached, r.statusKnown, r.statusErr = uint64(count), true, nil
			r.statusUntil = time.Now().Add(semanticStatusCacheTTL)
		} else {
			// A failed probe never overwrites an observed count with a fake
			// zero; statusKnown stays as it was and the error is retried
			// sooner than a good observation.
			r.statusErr = err
			r.statusUntil = time.Now().Add(semanticStatusErrorTTL)
		}
	} else {
		// The serving store was replaced mid-probe; the result describes a
		// collection that no longer serves.
		r.statusUntil = time.Time{}
		r.statusKnown = false
		r.statusCached, r.statusErr = 0, nil
	}
	r.statusRefresh = nil
	close(done)
	r.statusMu.Unlock()
}

// CanResumeSemanticGeneration reports whether a published generation already
// owns a physical candidate collection with vectors. A full rebuild is
// resumable: the store keeps its candidate collection and the reconciler reuses
// already-staged sources, so an interrupted embed does not have to start over.
//
// This runs on the startup path, so it is strictly best-effort: missing
// configuration, an unreachable Qdrant, or any unexpected client panic means
// "cannot resume", never a failed boot.
func (r *SemanticRuntime) CanResumeSemanticGeneration(ctx context.Context, generationID string) (resumable bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("conversation search: semantic resume probe for %q skipped after panic: %v", generationID, recovered)
			resumable = false
		}
	}()
	if r == nil || r.GenerationCatalog == nil || r.GenerationStore == nil {
		return false
	}
	if strings.TrimSpace(r.QdrantBaseURL) == "" {
		return false
	}
	generationID = strings.TrimSpace(generationID)
	if generationID == "" {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, semanticStatusRefreshTimeout)
	defer cancel()
	records, err := r.GenerationCatalog.ListGenerationRecords(probeCtx, conversationCollectionDomain, r.Collection)
	if err != nil {
		return false
	}
	collection := ""
	for _, record := range records {
		if record.Metadata.ID != generationID {
			continue
		}
		if record.State != aisearch.GenerationStateCandidate || strings.TrimSpace(record.CollectionName) == "" {
			return false
		}
		collection = record.CollectionName
		break
	}
	if collection == "" {
		return false
	}
	// NewVectorStore supplies its own bounded HTTP client. Passing nil to
	// NewVectorStoreWithClient leaves the client nil and panics on the first
	// request, which is exactly how this probe took the API down once.
	store := aisearch.NewVectorStore(r.QdrantBaseURL, r.QdrantAPIKey, collection)
	if store == nil {
		return false
	}
	count, err := store.CountPoints(probeCtx)
	return err == nil && count > 0
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

	// Snapshot the serving recipe under a read lock, then release it before any
	// slow I/O. Holding the write lock across retriever construction and the
	// Qdrant cutover blocked status and rebuild readers for as long as the
	// promotion took (observed 2026-11-19). The lock is reacquired only to swap
	// the in-memory fields.
	r.recipeMu.RLock()
	if !restore {
		if err := r.ensureActiveRecipeIdentity(ctx); err != nil {
			r.recipeMu.RUnlock()
			return fmt.Errorf("preserve current serving recipe before cutover: %w", err)
		}
	}
	tuning := r.Tuning
	tuning.EmbedModel = metadata.Model
	tuning.EmbedTaskPrefix = metadata.EmbeddingTaskPrefix
	collection := r.Collection
	qdrantURL, qdrantKey := r.QdrantBaseURL, r.QdrantAPIKey
	rerankerURL, rerankerModel, rerankerRole := r.RerankerURL, r.RerankerModel, r.RerankRole
	projection, admission := r.Projection, r.Admission
	source, swapRetriever := r.Source, r.swapRetriever
	r.recipeMu.RUnlock()

	engine := aisearch.NewServiceForTuning(tuning, aisearch.EngineDeps{
		QdrantURL: qdrantURL, QdrantAPIKey: qdrantKey, Collection: collection,
		EmbedRole: metadata.EmbeddingRole, EmbedModel: metadata.Model,
		EmbedDimensions: metadata.EmbeddingDimensions, EmbedPolicySchemaVersion: metadata.PolicySchemaVersion,
		RerankerURL: rerankerURL, RerankerModel: rerankerModel, RerankRole: rerankerRole,
	})
	engine.Spec.RecipeID = metadata.RecipeID
	engine.Spec.RecipeCanonical = metadata.RecipeCanonical
	serviceOptions := engine.ServiceOptions()
	serviceOptions.MaxLimit = maximumSearchPageSize
	serviceOptions.DefaultLimit = defaultSearchPageSize
	serviceOptions.Project = func(result aisearch.SearchResult) aisearch.SearchResult {
		if sourceID, ok := result.Payload["source_id"].(string); ok {
			result.SourceID = sourceID
		}
		return result
	}
	retriever, err := NewSharedSemanticRetriever(aisearch.NewService(serviceOptions), projection, admission)
	if err != nil {
		return err
	}
	cutover := func() error { return store.PromoteGeneration(ctx, metadata.ID) }
	if swapRetriever != nil {
		if err := swapRetriever.replaceAfter(cutover, retriever); err != nil {
			return err
		}
	} else if err := cutover(); err != nil {
		return err
	}

	r.recipeMu.Lock()
	defer r.recipeMu.Unlock()
	r.Engine = engine
	r.Retriever = swapRetriever
	r.EmbeddingModel = metadata.Model
	r.EmbeddingRecipeID = metadata.RecipeID
	r.EmbeddingRecipeJSON = metadata.RecipeCanonical
	r.InitializationError = nil
	r.Binding = aisearch.NewHybridBinding("conversation", "", engine.VectorStore, source, aisearch.NewIdentityChunker(), conversationEmbeddingComposer{}, engine.SparseEncoder)
	r.GenerationStore = store
	r.StreamingBinding.Store = store
	r.StreamingBinding.Sparse = engine.SparseEncoder
	r.StreamingReconciler = aisearch.NewStreamingReconciler(engine.Embedder)
	r.Tuning = tuning
	r.statusMu.Lock()
	r.statusUntil = time.Time{}
	r.statusMu.Unlock()
	r.healthMu.Lock()
	r.health = SemanticGenerationHealth{
		State: GenerationHealthHealthy, SurvivorID: metadata.ID, SurvivorCollection: record.CollectionName,
		RecipeValid: true, ActiveGenerations: 1, ObservedAt: time.Now().UTC(),
	}
	r.healthObservedAt = time.Now().UTC()
	r.healthMu.Unlock()
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

	_ SemanticGenerationHealthReporter = (*SemanticRuntime)(nil)
)
