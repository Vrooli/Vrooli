package conversationsearch

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
)

const guardTestAlias = "agent-manager_conversation-search"

// legacyGenerationRow is the exact shape a pre-fix catalog left behind: an
// active row with no recipe identity. It is inserted with raw SQL because the
// adapter now refuses to create a second active row.
type legacyGenerationRow struct {
	ID        string
	CreatedAt time.Time
	Model     string
	RecipeID  string
	Canonical string
}

func insertLegacyActiveGeneration(t *testing.T, db *sqlx.DB, row legacyGenerationRow) {
	t.Helper()
	dimensions := 0
	chunkPolicy, role, schema := "", "", ""
	if row.Model != "" {
		dimensions, chunkPolicy, role, schema = 384, DefaultRecipeVersion, "embedding.default", "schema-v7"
	}
	_, err := db.Exec(`INSERT INTO conversation_search_qdrant_generations
        (generation_id, owner, namespace, alias_name, collection_name, model, recipe_id, recipe_canonical,
         chunk_policy, embedding_role, embedding_dimensions, policy_schema_version, state, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		row.ID, ConversationSearchProviderID, conversationCollectionDomain, guardTestAlias,
		guardTestAlias+"--generation--"+row.ID, row.Model, row.RecipeID, row.Canonical,
		chunkPolicy, role, dimensions, schema, formatTime(row.CreatedAt), formatTime(row.CreatedAt))
	require.NoError(t, err)
}

func legacyRow(t *testing.T, id string, createdAt time.Time, model string) legacyGenerationRow {
	t.Helper()
	row := legacyGenerationRow{ID: id, CreatedAt: createdAt}
	if model != "" {
		row.RecipeID, row.Canonical = catalogTestRecipe(t, model, 384)
		row.Model = model
	}
	return row
}

func activeStateReasons(t *testing.T, db *sqlx.DB) map[string]string {
	t.Helper()
	var rows []struct {
		GenerationID string `db:"generation_id"`
		State        string `db:"state"`
		Reason       string `db:"state_reason"`
		Superseded   string `db:"superseded_by"`
	}
	require.NoError(t, db.Select(&rows, `SELECT generation_id, state, state_reason, superseded_by
        FROM conversation_search_qdrant_generations WHERE namespace = ? AND alias_name = ?
        ORDER BY generation_id`, conversationCollectionDomain, guardTestAlias))
	out := map[string]string{}
	for _, row := range rows {
		out[row.GenerationID] = row.State + "|" + row.Reason + "|" + row.Superseded
	}
	return out
}

func TestReconcileRetiresDuplicateActiveGenerationsAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-old-complete", base, "nomic-embed-text"))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-new-recorded", base.Add(time.Hour), "nomic-embed-text"))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-newest-incomplete", base.Add(2*time.Hour), ""))

	report, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{Reason: "test"})
	require.NoError(t, err)
	require.Equal(t, 3, report.ActiveBefore)
	require.Equal(t, "gen-new-recorded", report.SurvivorID)
	require.True(t, report.RecipeValid)
	require.Len(t, report.Retired, 2)
	require.True(t, report.InvariantPresent)

	states := activeStateReasons(t, db)
	require.Equal(t, "active||", states["gen-new-recorded"])
	require.Contains(t, states["gen-old-complete"], "retired|duplicate-active: same recipe; superseded by gen-new-recorded|gen-new-recorded")
	require.Contains(t, states["gen-newest-incomplete"], "retired|duplicate-active: incomplete identity; superseded by gen-new-recorded|gen-new-recorded")

	// Idempotent: a second pass observes one active row, retires nothing and
	// takes the read-only fast path (it must not rewrite the healthy row).
	var updatedBefore string
	require.NoError(t, db.Get(&updatedBefore, `SELECT updated_at FROM conversation_search_qdrant_generations WHERE generation_id = ?`, "gen-new-recorded"))
	second, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{Reason: "test"})
	require.NoError(t, err)
	require.Equal(t, 1, second.ActiveBefore)
	require.Empty(t, second.Retired)
	require.Equal(t, "gen-new-recorded", second.SurvivorID)
	require.True(t, second.InvariantPresent)
	var updatedAfter string
	require.NoError(t, db.Get(&updatedAfter, `SELECT updated_at FROM conversation_search_qdrant_generations WHERE generation_id = ?`, "gen-new-recorded"))
	require.Equal(t, updatedBefore, updatedAfter, "a healthy status read must not write the generation row")

	// The invariant is now real: a direct duplicate active insert is rejected.
	_, err = db.Exec(`INSERT INTO conversation_search_qdrant_generations
        (generation_id, owner, namespace, alias_name, collection_name, state, created_at, updated_at)
        VALUES ('gen-rogue', ?, ?, ?, 'rogue-collection', 'active', '2026-11-11T20:00:00Z', '2026-11-11T20:00:00Z')`,
		ConversationSearchProviderID, conversationCollectionDomain, guardTestAlias)
	require.Error(t, err, "the partial unique index must reject a second active generation")

	// The survivor is what the semantic path adopts.
	active, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, "gen-new-recorded", active.Metadata.ID)
	require.NoError(t, verifyGenerationRecipe(active.Metadata))
}

func TestReconcileRefusesDivergentCompleteRecipesWithoutEvidence(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-recipe-a", base, "nomic-embed-text"))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-recipe-b", base.Add(time.Hour), "another-embed-model"))

	_, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{Reason: "test"})
	require.ErrorIs(t, err, ErrAmbiguousActiveGenerations)
	states := activeStateReasons(t, db)
	require.Equal(t, "active||", states["gen-recipe-a"], "an ambiguous catalog must not be mutated")
	require.Equal(t, "active||", states["gen-recipe-b"])

	// Explicit operator reconciliation resolves it and records the reason.
	forced, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{
		Reason: "test", AllowRecipeDivergence: true,
	})
	require.NoError(t, err)
	require.Equal(t, "gen-recipe-b", forced.SurvivorID)
	require.Len(t, forced.Retired, 1)
	require.Contains(t, activeStateReasons(t, db)["gen-recipe-a"], "retired|duplicate-active: divergent recipe; superseded by gen-recipe-b")
}

func TestReconcileUsesServingAliasToSettleDivergentRecipes(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-recipe-a", base, "nomic-embed-text"))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-recipe-b", base.Add(time.Hour), "another-embed-model"))

	report, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{
		Reason:                  "test",
		AuthoritativeCollection: guardTestAlias + "--generation--gen-recipe-a",
	})
	require.NoError(t, err)
	require.Equal(t, "gen-recipe-a", report.SurvivorID, "the Qdrant alias target is the serving truth")
	require.Len(t, report.Retired, 1)
	require.Equal(t, "gen-recipe-b", report.Retired[0].GenerationID)
}

func TestReconcileRetiresUnverifiedGenerationsSoFreshBuildCanStart(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-incomplete-a", base, ""))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-incomplete-b", base.Add(time.Hour), ""))

	report, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{Reason: "test"})
	require.NoError(t, err)
	require.Empty(t, report.SurvivorID)
	require.False(t, report.RecipeValid)
	require.Len(t, report.Retired, 2)
	require.Contains(t, activeStateReasons(t, db)["gen-incomplete-a"], "duplicate-active: no usable identity")

	active, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.Nil(t, active, "with no verified recipe the runtime must adopt the configured policy and rebuild")

	// A rebuild promotion mints exactly one new active generation.
	promoted := legacyRow(t, "conversation-fresh-1", base.Add(2*time.Hour), "nomic-embed-text")
	insertLegacyActiveGenerationReplacing(t, db, promoted)
	active, err = loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, "conversation-fresh-1", active.Metadata.ID)
}

// insertLegacyActiveGenerationReplacing inserts the fresh generation through the
// adapter, which is the promotion path a rebuild uses.
func insertLegacyActiveGenerationReplacing(t *testing.T, db *sqlx.DB, row legacyGenerationRow) {
	t.Helper()
	catalog := NewSQLiteQdrantCatalog(db)
	record := aisearch.GenerationRecord{
		Metadata: aisearch.GenerationMetadata{
			ID: row.ID, CreatedAt: row.CreatedAt, Owner: ConversationSearchProviderID,
			Namespace: conversationCollectionDomain, Alias: guardTestAlias,
			Model: row.Model, RecipeID: row.RecipeID, RecipeCanonical: row.Canonical,
			ChunkPolicy: DefaultRecipeVersion, EmbeddingRole: "embedding.default",
			EmbeddingDimensions: 384, PolicySchemaVersion: "schema-v7",
		},
		CollectionName: guardTestAlias + "--generation--" + row.ID,
		State:          aisearch.GenerationStateActive,
		UpdatedAt:      row.CreatedAt,
	}
	require.NoError(t, catalog.SaveGenerationRecord(context.Background(), record))
}

func TestReconcileRetainsServingAliasTargetWithoutRecipe(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-serving", base, ""))
	insertLegacyActiveGeneration(t, db, legacyRow(t, "gen-stale", base.Add(time.Hour), ""))

	report, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{
		Reason:                  "test",
		AuthoritativeCollection: guardTestAlias + "--generation--gen-serving",
	})
	require.NoError(t, err)
	require.Equal(t, "gen-serving", report.SurvivorID)
	require.False(t, report.RecipeValid)
	require.Len(t, report.Retired, 1)
	require.Contains(t, report.Findings[len(report.Findings)-1], "retired 1 duplicate active generation")
}

func TestSaveGenerationRecordRetiresThePreviousActiveGeneration(t *testing.T) {
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	first := legacyRow(t, "gen-promote-1", base, "nomic-embed-text")
	second := legacyRow(t, "gen-promote-2", base.Add(time.Minute), "nomic-embed-text")
	insertLegacyActiveGenerationReplacing(t, db, first)
	insertLegacyActiveGenerationReplacing(t, db, second)

	states := activeStateReasons(t, db)
	require.Equal(t, "active||", states["gen-promote-2"])
	require.Contains(t, states["gen-promote-1"], "retired|lifecycle: superseded by gen-promote-2|gen-promote-2")

	var active int
	require.NoError(t, db.Get(&active, `SELECT COUNT(*) FROM conversation_search_qdrant_generations
        WHERE namespace = ? AND alias_name = ? AND state = 'active'`, conversationCollectionDomain, guardTestAlias))
	require.Equal(t, 1, active)

	// Rollback replays an active transition for the previous generation; the
	// adapter must retire the currently active row instead of failing the unique
	// index.
	insertLegacyActiveGenerationReplacing(t, db, first)
	states = activeStateReasons(t, db)
	require.Equal(t, "active||", states["gen-promote-1"])
	require.Contains(t, states["gen-promote-2"], "retired|lifecycle: superseded by gen-promote-1")
}

// nonReconcilingCatalog exposes the read-only GenerationCatalog contract, as an
// older or third-party adapter would.
type nonReconcilingCatalog struct {
	records []aisearch.GenerationRecord
}

func (c nonReconcilingCatalog) ListGenerationRecords(context.Context, string, string) ([]aisearch.GenerationRecord, error) {
	return c.records, nil
}

func (c nonReconcilingCatalog) SaveGenerationRecord(context.Context, aisearch.GenerationRecord) error {
	return nil
}

func guardRecord(t *testing.T, id string, createdAt time.Time, model string) aisearch.GenerationRecord {
	t.Helper()
	metadata := aisearch.GenerationMetadata{
		ID: id, CreatedAt: createdAt, Owner: ConversationSearchProviderID,
		Namespace: conversationCollectionDomain, Alias: guardTestAlias,
	}
	if model != "" {
		metadata.Model = model
		metadata.RecipeID, metadata.RecipeCanonical = catalogTestRecipe(t, model, 384)
		metadata.ChunkPolicy, metadata.EmbeddingRole, metadata.EmbeddingDimensions = DefaultRecipeVersion, "embedding.default", 384
		metadata.PolicySchemaVersion = "schema-v7"
	}
	return aisearch.GenerationRecord{
		Metadata: metadata, CollectionName: guardTestAlias + "--generation--" + id,
		State: aisearch.GenerationStateActive, UpdatedAt: createdAt,
	}
}

func TestLoadActiveRecipeRecordToleratesLegacyDuplicatesInAnyCatalog(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	catalog := nonReconcilingCatalog{records: []aisearch.GenerationRecord{
		guardRecord(t, "gen-old", base, "nomic-embed-text"),
		guardRecord(t, "gen-incomplete", base.Add(time.Hour), ""),
		guardRecord(t, "gen-newest-complete", base.Add(2*time.Hour), "nomic-embed-text"),
	}}
	active, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, "gen-newest-complete", active.Metadata.ID)

	// One active row is unambiguous even when it predates recipe identity: the
	// runtime must be able to adopt the configured policy for it.
	single := nonReconcilingCatalog{records: []aisearch.GenerationRecord{
		guardRecord(t, "gen-legacy", base, ""),
	}}
	active, err = loadActiveRecipeRecord(ctx, single, guardTestAlias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, "gen-legacy", active.Metadata.ID)

	divergent := nonReconcilingCatalog{records: []aisearch.GenerationRecord{
		guardRecord(t, "gen-a", base, "nomic-embed-text"),
		guardRecord(t, "gen-b", base.Add(time.Hour), "another-embed-model"),
	}}
	_, err = loadActiveRecipeRecord(ctx, divergent, guardTestAlias)
	require.ErrorIs(t, err, ErrAmbiguousActiveGenerations)
}

// insertIdentityOnlyActiveGeneration seeds the legacy pre-recipe shape: a
// complete serving identity (model, role, dimensions, policy, chunk policy)
// with no immutable recipe identity.
func insertIdentityOnlyActiveGeneration(t *testing.T, db *sqlx.DB, id string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO conversation_search_qdrant_generations
        (generation_id, owner, namespace, alias_name, collection_name, model,
         chunk_policy, embedding_role, embedding_dimensions, policy_schema_version, state, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, ConversationSearchProviderID, conversationCollectionDomain, guardTestAlias,
		guardTestAlias+"--generation--"+id, "nomic-embed-text:latest",
		DefaultRecipeVersion, "embedding.default", 384, "2026-06-10", formatTime(createdAt), formatTime(createdAt))
	require.NoError(t, err)
}

func TestReconcileKeepsTheServingLegacyIdentityOnlyGeneration(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	insertIdentityOnlyActiveGeneration(t, db, "gen-legacy-serving", base)
	insertIdentityOnlyActiveGeneration(t, db, "gen-legacy-newer", base.Add(time.Hour))

	report, err := catalog.ReconcileActiveGenerations(ctx, conversationCollectionDomain, guardTestAlias, GenerationReconcileOptions{
		Reason:                  "test",
		AuthoritativeCollection: guardTestAlias + "--generation--gen-legacy-serving",
	})
	require.NoError(t, err)
	require.Equal(t, "gen-legacy-serving", report.SurvivorID,
		"the alias target serves real vectors and must keep serving")
	require.False(t, report.RecipeValid, "legacy generations cannot claim an immutable recipe they never recorded")
	require.Len(t, report.Retired, 1)

	active, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, 1, generationServingRank(active.Metadata))
	require.False(t, completeGenerationRecipe(active.Metadata))
}

func TestSemanticStatusNeverReportsAnEmptyIndexAsHealthy(t *testing.T) {
	store := searchtest.NewVectorStore()
	runtime := &SemanticRuntime{
		Engine: aisearch.TunedEngine{VectorStore: store}, Collection: guardTestAlias,
		EmbeddingModel: "nomic-embed-text:latest",
	}
	count, _, _, _, err := runtime.SemanticStatus(context.Background())
	require.Equal(t, uint64(0), count)
	require.ErrorIs(t, err, ErrSemanticUnavailable, "an empty semantic index must be an explicit degradation")
	require.Contains(t, err.Error(), "holds no semantic vectors")
}

func TestSemanticGenerationHealthNeverReportsAnEmptyPathHealthy(t *testing.T) {
	missing := SemanticGenerationHealth{State: GenerationHealthMissingRecipe, Detail: "no active conversation generation has a verified embedding recipe"}
	require.NotEmpty(t, missing.Degradation())
	require.Contains(t, missing.Degradation(), GenerationHealthMissingRecipe)
	require.Empty(t, SemanticGenerationHealth{State: GenerationHealthHealthy}.Degradation())

	runtime := &SemanticRuntime{Collection: guardTestAlias}
	runtime.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 0}, true, nil, nil)
	health := runtime.GenerationHealthSnapshot()
	require.Equal(t, GenerationHealthMissingRecipe, health.State)
	require.NotEmpty(t, health.Degradation())

	// A catalog without the reconciliation seam is reported as unknown.
	unmanaged := &SemanticRuntime{Collection: guardTestAlias}
	unmanaged.recordGenerationHealth(ActiveGenerationReconciliation{}, false, nil, nil)
	require.Equal(t, GenerationHealthUnknown, unmanaged.GenerationHealthSnapshot().State)

	// A repaired catalog stays visibly recovered for the process lifetime even
	// after later passes observe a clean generation.
	recovered := &SemanticRuntime{Collection: guardTestAlias}
	active := guardRecord(t, "gen-served", time.Now().UTC(), "nomic-embed-text")
	recovered.recordGenerationHealth(ActiveGenerationReconciliation{
		ActiveBefore: 3, Retired: []RetiredActiveGeneration{{GenerationID: "stale"}}, RecipeValid: true,
	}, true, nil, &active)
	require.Equal(t, GenerationHealthRecovered, recovered.GenerationHealthSnapshot().State)
	require.Contains(t, recovered.GenerationHealthSnapshot().Degradation(), "reconciled 1 duplicate")
	recovered.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 1, RecipeValid: true}, true, nil, &active)
	require.Equal(t, GenerationHealthRecovered, recovered.GenerationHealthSnapshot().State,
		"a repaired catalog must keep reporting the repair until a promotion replaces it")

	ambiguous := &SemanticRuntime{Collection: guardTestAlias}
	ambiguous.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 2, Ambiguous: true}, true, ErrAmbiguousActiveGenerations, nil)
	require.Equal(t, GenerationHealthAmbiguous, ambiguous.GenerationHealthSnapshot().State)
	require.Contains(t, ambiguous.GenerationHealthSnapshot().Degradation(), GenerationHealthAmbiguous)
}

// inertGenerationStore satisfies the lifecycle contract so the resume probe can
// be exercised without any real vector store (the production panic was a nil
// client inside one).
type inertGenerationStore struct{}

func (inertGenerationStore) BeginGeneration(context.Context, aisearch.GenerationMetadata) error {
	return nil
}

func (inertGenerationStore) LookupActiveSources(context.Context, []string) (map[string]aisearch.StoredSourceState, error) {
	return nil, nil
}

func (inertGenerationStore) StageSource(context.Context, string, aisearch.GenerationSourceWrite) error {
	return nil
}

func (inertGenerationStore) StageDelete(context.Context, string, string) error { return nil }

func (inertGenerationStore) ValidateGeneration(context.Context, string) (aisearch.GenerationValidation, error) {
	return aisearch.GenerationValidation{}, nil
}

func (inertGenerationStore) PromoteGeneration(context.Context, string) error { return nil }

func (inertGenerationStore) RollbackGeneration(context.Context, string) error { return nil }

func (inertGenerationStore) CleanupGenerations(context.Context, int) error { return nil }

// catalogOnlyGenerationStore is enough for the resume probe: it must decide
// from catalog metadata alone and never reach into a vector store it cannot
// safely construct.
type catalogOnlyGenerationStore struct {
	records []aisearch.GenerationRecord
}

func (s catalogOnlyGenerationStore) ListGenerationRecords(context.Context, string, string) ([]aisearch.GenerationRecord, error) {
	return s.records, nil
}

func (s catalogOnlyGenerationStore) SaveGenerationRecord(context.Context, aisearch.GenerationRecord) error {
	return nil
}

// TestResumeProbeNeverPanicsWithoutAVectorStore is the regression for the
// 2026-11-19 startup panic: a nil HTTP client inside the vector store crashed
// agent-manager on boot while probing a resumable generation.
func TestResumeProbeNeverPanicsWithoutAVectorStore(t *testing.T) {
	records := []aisearch.GenerationRecord{{
		Metadata: aisearch.GenerationMetadata{
			ID: "conversation-published", CreatedAt: time.Now().UTC(), Owner: ConversationSearchProviderID,
			Namespace: conversationCollectionDomain, Alias: guardTestAlias,
		},
		CollectionName: guardTestAlias + "--generation--conversation-published",
		State:          aisearch.GenerationStateCandidate,
	}}
	catalog := catalogOnlyGenerationStore{records: records}
	store := inertGenerationStore{}

	var nilRuntime *SemanticRuntime
	require.False(t, nilRuntime.CanResumeSemanticGeneration(context.Background(), "conversation-published"))

	unconfigured := &SemanticRuntime{GenerationCatalog: catalog, GenerationStore: store}
	require.NotPanics(t, func() {
		require.False(t, unconfigured.CanResumeSemanticGeneration(context.Background(), "conversation-published"),
			"an empty Qdrant base URL is unavailable, not resumable")
	})

	// An unreachable endpoint must return false, not panic or block boot.
	unreachable := &SemanticRuntime{
		GenerationCatalog: catalog, GenerationStore: store, Collection: guardTestAlias,
		QdrantBaseURL: "http://127.0.0.1:1",
	}
	require.NotPanics(t, func() {
		require.False(t, unreachable.CanResumeSemanticGeneration(context.Background(), "conversation-published"))
	})

	// The whole resume entry point is safe with no staged repository.
	require.NotPanics(t, func() { nilRuntime.resumePublishedSemanticGeneration(context.Background()) })
	require.NotPanics(t, func() { (&SemanticRuntime{}).resumePublishedSemanticGeneration(context.Background()) })
}

func TestAcquireAliasMutationReleasesAProvablyDeadHolder(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	operation := aisearch.FleetAliasOperation{
		OwnerID: ConversationSearchProviderID, EndpointID: "http://qdrant:6333", Alias: guardTestAlias,
		OperationID: "legacy:rollback:conversation-dead:collection", Kind: aisearch.FleetRollback,
		ArmID: "conversation-dead", Target: "collection",
	}
	// A fence left by a process that can never be alive again (pid 1 with a start
	// time that is not its own).
	_, err := db.Exec(`INSERT INTO qdrant_fleet_alias_fences
        (endpoint_id, alias_name, owner_id, operation_id, holder_pid, holder_start, record_json)
        VALUES (?, ?, ?, ?, 1, 'not-this-process', '{}')`,
		operation.EndpointID, operation.Alias, operation.OwnerID, operation.OperationID)
	require.NoError(t, err)

	promote := operation
	promote.OperationID = "legacy:promote:conversation-new:new-collection"
	promote.Kind = aisearch.FleetPromote
	promote.ArmID = "conversation-new"
	promote.Target = "new-collection"
	fence, err := catalog.AcquireFleetAliasMutation(ctx, promote)
	require.NoError(t, err, "a dead holder must not fence the alias forever")
	require.NoError(t, fence.Release(ctx))

	// A live holder is never displaced.
	live := operation
	live.OperationID = "legacy:rollback:live:collection"
	fence, err = catalog.AcquireFleetAliasMutation(ctx, live)
	require.NoError(t, err)
	_, err = catalog.AcquireFleetAliasMutation(ctx, promote)
	require.ErrorIs(t, err, aisearch.ErrFleetStateConflict)
	require.NoError(t, fence.Release(ctx))
}
