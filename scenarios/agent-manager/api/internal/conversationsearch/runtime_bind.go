package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
)

// semanticStart is everything one bind attempt needs, captured once so a failed
// start can be retried without the caller.
type semanticStart struct {
	tuning    aisearch.TuningConfig
	deps      aisearch.EngineDeps
	admission *aisearch.WeightedAdmission
	resolver  ollamapolicy.Resolver
	lifecycle context.Context
	backoff   func(attempt int) time.Duration
}

const recoveryAttemptTimeout = 60 * time.Second

// defaultRecoveryBackoff spaces background re-binds: quick enough that a
// resolver that was merely busy at start recovers within seconds, capped so a
// resource that stays down costs one cheap attempt every five minutes.
func defaultRecoveryBackoff(attempt int) time.Duration {
	schedule := []time.Duration{3 * time.Second, 10 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute}
	if attempt < len(schedule) {
		return schedule[attempt]
	}
	return 5 * time.Minute
}

// bind reconciles the generation catalog, resolves the embedding policy and
// wires the semantic leg. On failure it records why (InitializationError,
// health, an unavailable retriever) and returns the cause; on success it
// replaces any earlier failure in place, because the retriever, indexer and
// status all reach the runtime through its fields.
func (r *SemanticRuntime) bind(ctx context.Context, reason string) error {
	start := r.start
	// Repair the generation catalog before anything reads it. The Qdrant alias
	// target is authoritative evidence of what actually serves, so resolve it
	// first and pass it into reconciliation; an unreachable Qdrant simply leaves
	// the choice to the durable catalog.
	serving := resolveServingAliasCollection(ctx, r.QdrantBaseURL, r.QdrantAPIKey, r.Collection)
	r.recipeMu.Lock()
	r.servingCollection = serving
	r.recipeMu.Unlock()
	reconcileReport, reconcileSupported, reconcileErr := reconcileGenerationCatalog(ctx, r.GenerationCatalog, r.Collection, GenerationReconcileOptions{
		AuthoritativeCollection: serving,
		Reason:                  reason,
	})
	r.recordGenerationHealth(reconcileReport, reconcileSupported, reconcileErr, nil)
	tuning := start.tuning
	engine, resolvedPolicy, engineErr := resolveSemanticEngine(ctx, start.resolver, tuning, start.deps)
	activeRecord, activeErr := loadActiveRecipeRecord(ctx, r.GenerationCatalog, r.Collection)
	adoptionNote := ""
	if activeErr == nil && activeRecord != nil {
		hasCanonicalRecipe := activeRecord.Metadata.RecipeID != "" || activeRecord.Metadata.RecipeCanonical != ""
		if hasCanonicalRecipe {
			if verifyErr := verifyGenerationRecipe(activeRecord.Metadata); verifyErr != nil {
				activeErr = fmt.Errorf("active conversation generation has invalid immutable recipe identity: %w", verifyErr)
			} else if !completeGenerationRecipe(activeRecord.Metadata) {
				activeErr = errors.New("active conversation generation has incomplete embedding recipe metadata")
			} else {
				engine, tuning = engineFromGenerationRecipe(tuning, start.deps, activeRecord.Metadata), tuningForGenerationRecipe(tuning, activeRecord.Metadata)
				engineErr = nil
				r.noteRecipeDrift(activeRecord.Metadata, resolvedPolicy, engine.Spec, tuning.EmbedTaskPrefix)
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
			if saveErr := r.GenerationCatalog.SaveGenerationRecord(ctx, *activeRecord); saveErr != nil {
				activeErr = fmt.Errorf("persist configured identity for active conversation generation: %w", saveErr)
			} else {
				// Give the serving generation the immutable recipe it lacks when
				// its own collection confirms the vectors, so every later
				// generation is created with a recipe instead of inheriting none.
				adoptionNote = r.adoptServingRecipe(ctx, r.GenerationCatalog, activeRecord, resolvedPolicy, &engine, tuning)
			}
		} else {
			activeErr = errors.New("active conversation generation has no recipe identity and configured embedding policy is unavailable")
		}
	}
	r.healthMu.Lock()
	r.recipeAdoptionNote = adoptionNote
	r.healthMu.Unlock()
	if activeErr != nil {
		engineErr = errors.Join(engineErr, activeErr)
	}
	if reconcileErr != nil {
		engineErr = errors.Join(engineErr, reconcileErr)
	}
	if engineErr != nil {
		return r.failBind(errors.Join(ErrEmbeddingUnavailable, engineErr), reconcileReport, reconcileSupported, reconcileErr, activeRecord)
	}
	// Do not freeze the runtime in lexical-only mode based on a startup probe.
	// Qdrant is an optional dependency and may become available after this API
	// starts. Keep the live retriever and indexing pipeline wired; their normal
	// operations report transient Qdrant failures and can recover on later calls.
	aliasBoundary, _ := r.GenerationCatalog.(aisearch.FleetAliasMutationBoundary)
	generationStore, err := aisearch.NewQdrantGenerationStore(aisearch.QdrantGenerationOptions{
		BaseURL: r.QdrantBaseURL, APIKey: r.QdrantAPIKey, Alias: r.Collection,
		Namespace: conversationCollectionDomain, Owner: ConversationSearchProviderID,
		Catalog: r.GenerationCatalog, AliasBoundary: aliasBoundary, Spec: engine.Spec,
	})
	if err != nil {
		return r.failBind(err, reconcileReport, reconcileSupported, reconcileErr, activeRecord)
	}
	retriever, err := r.sharedRetrieverFor(engine)
	if err != nil {
		return r.failBind(err, reconcileReport, reconcileSupported, reconcileErr, activeRecord)
	}
	r.recipeMu.Lock()
	r.Engine, r.Tuning, r.EmbeddingModel = engine, tuning, engine.Spec.Model
	r.EmbeddingRecipeID, r.EmbeddingRecipeJSON = "", ""
	if activeRecord != nil && completeGenerationRecipe(activeRecord.Metadata) {
		r.EmbeddingRecipeID = activeRecord.Metadata.RecipeID
		r.EmbeddingRecipeJSON = activeRecord.Metadata.RecipeCanonical
	}
	r.Binding = aisearch.NewHybridBinding("conversation", "", engine.VectorStore, r.Source, aisearch.NewIdentityChunker(), conversationEmbeddingComposer{}, engine.SparseEncoder)
	r.AliasBoundary = aliasBoundary
	r.GenerationStore = generationStore
	r.StreamingBinding = aisearch.StreamingBinding{
		Kind: "conversation", Store: generationStore, Source: r.Source,
		Chunker: aisearch.NewIdentityChunker(), Composer: conversationEmbeddingComposer{}, Sparse: engine.SparseEncoder,
		PageSize: defaultIndexPageSize, Admission: start.admission, EmbedWeight: 2, EmbedConcurrency: 4,
	}
	r.StreamingReconciler = aisearch.NewStreamingReconciler(engine.Embedder)
	r.InitializationError = nil
	r.recipeMu.Unlock()
	r.recordGenerationHealth(reconcileReport, reconcileSupported, reconcileErr, activeRecord)
	r.swapRetriever.replace(retriever)
	r.statusMu.Lock()
	r.statusUntil = time.Time{}
	r.statusMu.Unlock()
	r.resumePublishedSemanticGeneration(ctx)
	return nil
}

// failBind records an unusable semantic leg and returns the cause.
func (r *SemanticRuntime) failBind(cause error, report ActiveGenerationReconciliation, supported bool, reconcileErr error, activeRecord *aisearch.GenerationRecord) error {
	r.recipeMu.Lock()
	r.InitializationError = cause
	r.recipeMu.Unlock()
	r.recordGenerationHealth(report, supported, reconcileErr, activeRecord)
	r.swapRetriever.replace(unavailableSemanticRetriever{err: cause})
	return cause
}

// scheduleRecovery retries a failed bind in the background until it succeeds
// or the runtime's lifecycle ends. At most one retry loop runs.
func (r *SemanticRuntime) scheduleRecovery() {
	if !r.recovering.CompareAndSwap(false, true) {
		return
	}
	start := r.start
	go func() {
		defer r.recovering.Store(false)
		lastFailure := ""
		for attempt := 0; ; attempt++ {
			timer := time.NewTimer(start.backoff(attempt))
			select {
			case <-start.lifecycle.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			attemptCtx, cancel := context.WithTimeout(start.lifecycle, recoveryAttemptTimeout)
			err := r.bind(attemptCtx, "runtime-recovery")
			cancel()
			if err == nil {
				log.Printf("conversation search: semantic leg recovered after %d retry attempt(s)", attempt+1)
				return
			}
			if message := err.Error(); message != lastFailure {
				log.Printf("conversation search: semantic leg still unavailable (retry %d): %v", attempt+1, err)
				lastFailure = message
			}
		}
	}()
}

func (r *SemanticRuntime) adoptionNote() string {
	r.healthMu.Lock()
	defer r.healthMu.Unlock()
	return r.recipeAdoptionNote
}

// adoptServingRecipe stamps a recipe-less serving generation with the immutable
// recipe minted from the owner policy, once the serving collection's own meta
// sentinels confirm its vectors. It returns why it did not (empty when it did)
// so status can explain the legacy note instead of repeating it. Nothing is
// re-embedded: the catalog row gains an identity and later generations inherit
// it.
func (r *SemanticRuntime) adoptServingRecipe(ctx context.Context, catalog aisearch.GenerationCatalog, record *aisearch.GenerationRecord, resolved ollamapolicy.ResolvedRole, engine *aisearch.TunedEngine, tuning aisearch.TuningConfig) string {
	recipe, err := mintConversationRecipe(resolved, engine.Spec, tuning.EmbedTaskPrefix)
	if err != nil {
		return "recipe adoption declined: " + err.Error()
	}
	collection := r.servingCollection
	if collection == "" {
		collection = r.Collection
	}
	sentinels, err := readServingSentinels(ctx, r.QdrantBaseURL, r.QdrantAPIKey, collection)
	if err != nil {
		return "recipe adoption deferred to the next start: " + err.Error()
	}
	reason, err := adoptLegacyRecipe(ctx, catalog, record, recipe, sentinels)
	switch {
	case err != nil:
		return "recipe adoption failed: " + err.Error()
	case reason != "":
		return "recipe adoption declined: " + reason
	}
	engine.Spec.RecipeID, engine.Spec.RecipeCanonical = recipe.ID(), string(recipe.Canonical())
	log.Printf("conversation search: generation %s adopted immutable recipe %s (model %s, %d dimensions) after its collection confirmed the vectors", record.Metadata.ID, recipe.ID(), engine.Spec.Model, engine.Spec.DenseSize)
	return ""
}

// noteRecipeDrift logs when the owner policy now resolves to a different
// recipe than the serving generation recorded (a model artifact changed under
// the same name). Serving continues on the recorded recipe; replacing it is an
// explicit migration, never a restart side effect.
func (r *SemanticRuntime) noteRecipeDrift(recorded aisearch.GenerationMetadata, resolved ollamapolicy.ResolvedRole, spec aisearch.CollectionSpec, taskPrefix bool) {
	current, err := mintConversationRecipe(resolved, spec, taskPrefix)
	if err != nil || current.ID() == recorded.RecipeID {
		return
	}
	log.Printf("conversation search: the embedding policy now resolves to recipe %s but generation %s serves %s; keeping the recorded recipe until an explicit migration", current.ID(), recorded.ID, recorded.RecipeID)
}
