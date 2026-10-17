package journal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	localdb "source-ledger/internal/database"
	"source-ledger/internal/facets"
	"source-ledger/internal/inference"
	"source-ledger/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func journalDB(t *testing.T) *SQLiteRepository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	return NewSQLiteRepository(d)
}

func TestAppendPreservesWriteOrderAndFacetEmbeddings(t *testing.T) { // [REQ:VMEM-P0-001] [REQ:VMEM-P0-002] [REQ:VMEM-P1-005]
	repo := journalDB(t)
	client := &mocks.FakeInference{ClassifyOut: "project", EmbedOut: []float64{0.1, 0.2}}
	svc := NewService(repo, client)
	for _, body := range []string{"first", "second", "third"} {
		_, err := svc.Append(context.Background(), Entry{Body: body, Kind: "memory"})
		require.NoError(t, err)
	}
	entries, err := repo.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	for i, body := range []string{"first", "second", "third"} {
		require.Equal(t, body, entries[i].Body)
		require.NotEmpty(t, entries[i].ID)
		require.Len(t, entries[i].FacetTexts, 3)
	}
}

func TestFacetTextsKeepDistinctClusteringSpaces(t *testing.T) { // [REQ:VMEM-P1-005]
	repo := journalDB(t)
	client := &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{0.1, 0.2}}
	entry, err := NewService(repo, client).Append(context.Background(), Entry{Body: "one memory", Kind: "memory"})
	require.NoError(t, err)
	require.Equal(t, []string{"topic", "rule", "entities"}, []string{entry.FacetTexts[0].Kind, entry.FacetTexts[1].Kind, entry.FacetTexts[2].Kind})
	require.Equal(t, []string{"one memory", "Implication: one memory", "Entities: one memory"}, []string{entry.FacetTexts[0].Text, entry.FacetTexts[1].Text, entry.FacetTexts[2].Text})
	for _, facetText := range entry.FacetTexts {
		require.Equal(t, []float64{0.1, 0.2}, facetText.Vector)
	}
}

func TestClassifierFailureStillAppendsUnclassifiedEntry(t *testing.T) { // [REQ:VMEM-P0-002]
	repo := journalDB(t)
	client := &mocks.FakeInference{ClassifyErr: errors.New("gateway unavailable"), EmbedOut: []float64{1}}
	entry, err := NewService(repo, client).Append(context.Background(), Entry{Body: "never lose this", Kind: "memory"})
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, entry.FacetID)
	stored, err := repo.Get(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, stored.FacetID)
	var queuedEntryID, reason string
	require.NoError(t, repo.db.QueryRowContext(context.Background(), `SELECT entry_id, reason FROM journal_retry_queue`).Scan(&queuedEntryID, &reason))
	require.Equal(t, entry.ID, queuedEntryID)
	require.Equal(t, "classify", reason)
}

type contextualTestInference struct{ kind string }

func (f *contextualTestInference) Classify(context.Context, string) (string, error) {
	return "thread", nil
}

func (f *contextualTestInference) ClassifyEntry(_ context.Context, _ string, kind string) (string, error) {
	f.kind = kind
	return "episode", nil
}

func (*contextualTestInference) Embed(context.Context, string, inference.EmbeddingTask) ([]float64, error) {
	return []float64{1}, nil
}
func (*contextualTestInference) Summarize(context.Context, string) (string, error) { return "", nil }

func TestAppendPassesValidatedEntryKindToContextualClassifier(t *testing.T) {
	repo := journalDB(t)
	client := &contextualTestInference{}
	entry, err := NewService(repo, client).Append(context.Background(), Entry{Body: "Trigger: request\nApproach: work\nEvidence: tests\nOutcome: done", Kind: "work-record"})
	require.NoError(t, err)
	require.Equal(t, "work-record", client.kind)
	require.Equal(t, "episode", entry.FacetID)
}

func TestAppendPreservesExplicitFacetForDeterministicImports(t *testing.T) {
	repo := journalDB(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), repo.db, apidb.SchemaProviderFunc(facets.Schema)))
	facetRepo := facets.NewSQLiteRepository(repo.db)
	require.NoError(t, facetRepo.Seed(context.Background()))
	svc := NewService(repo, &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, facets.NewService(facetRepo))

	entry, err := svc.Append(context.Background(), Entry{Body: `{"decision":"keep"}`, Scope: "agent-memory", FacetID: "standing-rule"})
	require.NoError(t, err)
	require.Equal(t, "standing-rule", entry.FacetID)
	stored, err := repo.Get(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, "standing-rule", stored.FacetID)
}

func TestRepositoryExposesNoMutationMethods(t *testing.T) { // [REQ:VMEM-P0-001]
	typ := reflect.TypeOf((*Repository)(nil)).Elem()
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		require.NotContains(t, name, "Update")
		require.NotContains(t, name, "Delete")
	}
}

func TestProcessClassificationRetriesAppendsFacetAssignmentAndAcknowledgesQueue(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(facets.Schema)))
	repo := NewSQLiteRepository(d)
	fr := facets.NewSQLiteRepository(d)
	require.NoError(t, fr.Seed(context.Background()))
	failing := &mocks.FakeInference{ClassifyErr: errors.New("unavailable"), EmbedOut: []float64{1}}
	entry, err := NewService(repo, failing, facets.NewService(fr)).Append(context.Background(), Entry{Body: "keep this rule", Kind: "memory"})
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, entry.FacetID)

	result, err := NewService(repo, &mocks.FakeInference{ClassifyOut: "standing-rule"}, facets.NewService(fr)).ProcessClassificationRetries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	assignments, err := fr.Assignments(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, "standing-rule", assignments[0].FacetID)
	var queued int
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM journal_retry_queue`).Scan(&queued))
	require.Zero(t, queued)
	stored, err := repo.Get(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, stored.FacetID, "entry rows remain immutable; facet history owns retry correction")
}

func TestProcessEmbeddingRetriesRestoresAllFacetVectorsAndAcknowledgesQueue(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)
	failing := &mocks.FakeInference{ClassifyOut: "episode", EmbedErr: errors.New("unavailable")}
	entry, err := NewService(repo, failing).Append(context.Background(), Entry{Body: "recover vectors", Kind: "memory"})
	require.NoError(t, err)
	require.Len(t, entry.FacetTexts, 3)
	result, err := NewService(repo, &mocks.FakeInference{EmbedOut: []float64{0.1, 0.2}}).ProcessEmbeddingRetries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	var vectors, queued int
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM embeddings`).Scan(&vectors))
	require.Equal(t, 3, vectors)
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM journal_retry_queue WHERE reason='embed'`).Scan(&queued))
	require.Zero(t, queued)
}

type ruleTestInference struct{ classifyCalls int }

func (f *ruleTestInference) Classify(context.Context, string) (string, error) {
	f.classifyCalls++
	return "gotcha", nil
}

func (*ruleTestInference) Embed(context.Context, string, inference.EmbeddingTask) ([]float64, error) {
	return []float64{1, 0}, nil
}
func (*ruleTestInference) Summarize(context.Context, string) (string, error) { return "", nil }

func TestRuleMatchAssignsWithoutCallingClassifierAndRecordsProvenance(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(facets.Schema)))
	repo := NewSQLiteRepository(d)
	fr := facets.NewSQLiteRepository(d)
	require.NoError(t, fr.Seed(context.Background()))
	rule, err := fr.CreateRule(context.Background(), facets.Rule{ID: "source-episode", Priority: 1, FacetID: "episode", SourceRuntime: "swarm-manager"})
	require.NoError(t, err)
	_, err = fr.DryRunRule(context.Background(), rule.ID)
	require.NoError(t, err)
	require.NoError(t, fr.EnableRule(context.Background(), rule.ID))
	client := &ruleTestInference{}
	entry, err := NewService(repo, client, facets.NewService(fr)).Append(context.Background(), Entry{Body: "imported work", Kind: "work-record", Attribution: Attribution{SourceRuntime: "swarm-manager"}})
	require.NoError(t, err)
	require.Equal(t, "episode", entry.FacetID)
	require.Zero(t, client.classifyCalls)
	assignments, err := fr.Assignments(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, "rule:source-episode", assignments[0].ActorID)
}
