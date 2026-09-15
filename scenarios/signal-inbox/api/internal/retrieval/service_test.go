package retrieval

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"signal-inbox/internal/categories"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/inference"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/triage"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"
)

type memoryVectorStore struct {
	points   map[string]aisearch.Point
	resultID string
}

func (m *memoryVectorStore) EnsureCollection(context.Context, aisearch.CollectionSpec) error {
	return nil
}

func (m *memoryVectorStore) Upsert(_ context.Context, point aisearch.Point) error {
	if m.points == nil {
		m.points = map[string]aisearch.Point{}
	}
	m.points[point.ID] = point
	return nil
}
func (m *memoryVectorStore) SetPayload(context.Context, string, map[string]any) error { return nil }
func (m *memoryVectorStore) BatchDelete(context.Context, []string) error              { return nil }
func (m *memoryVectorStore) Query(_ context.Context, _ aisearch.HybridQuery) ([]aisearch.SearchResult, error) {
	return []aisearch.SearchResult{{ID: m.resultID, Score: 0.91, Payload: map[string]any{"signal_id": m.resultID}}}, nil
}
func (m *memoryVectorStore) CountPoints(context.Context) (int, error) { return len(m.points), nil }
func (m *memoryVectorStore) ScrollIDs(context.Context) (map[string]aisearch.ScrollItem, error) {
	return map[string]aisearch.ScrollItem{}, nil
}
func (m *memoryVectorStore) Available(context.Context) bool { return true }

func newService(t *testing.T) (*Service, signals.Service, *sql.DB) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(categories.Schema), apidb.SchemaProviderFunc(triage.Schema), apidb.SchemaProviderFunc(Schema)))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	journal := signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
	return NewService(NewSQLiteRepository(database), clk), journal, database
}

func TestSearchIndexesEverySignalRegardlessOfDisposition(t *testing.T) {
	t.Log("[REQ:SIG-P0-009] [REQ:SIG-P0-010]")
	service, journal, database := newService(t)
	active, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "A plan for durable capture and retrieval"})
	require.NoError(t, err)
	dropped, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "A discontinued platform source remains searchable"})
	require.NoError(t, err)
	triageService := triage.NewService(triage.NewSQLiteRepository(database), schedule.System())
	_, err = triageService.Set(context.Background(), dropped.Signal.ID, triage.Dropped, nil)
	require.NoError(t, err)
	indexed, total, err := service.Coverage(context.Background())
	require.NoError(t, err)
	require.Equal(t, total, indexed)
	results, err := service.Search(context.Background(), Filter{Text: "discontinued", Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, dropped.Signal.ID, results[0].Signal.ID)
	_, err = service.Search(context.Background(), Filter{Text: "durable", Disposition: "dropped", Limit: 10})
	require.NoError(t, err)
	_ = active
}

func TestAmbientExcludesDoneAndDroppedWithoutAffectingSearch(t *testing.T) {
	t.Log("[REQ:SIG-P0-012] [REQ:SIG-P0-010]")
	service, journal, database := newService(t)
	newSignal, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "new work"})
	require.NoError(t, err)
	done, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "completed work remains searchable"})
	require.NoError(t, err)
	triageService := triage.NewService(triage.NewSQLiteRepository(database), schedule.System())
	_, err = triageService.Set(context.Background(), done.Signal.ID, triage.Triaged, nil)
	require.NoError(t, err)
	_, err = triageService.Set(context.Background(), done.Signal.ID, triage.Done, nil)
	require.NoError(t, err)
	ambient, err := service.Ambient(context.Background(), "", 10)
	require.NoError(t, err)
	require.Len(t, ambient, 1)
	require.Equal(t, newSignal.Signal.ID, ambient[0].Signal.ID)
	search, err := service.Search(context.Background(), Filter{Text: "completed", Limit: 10})
	require.NoError(t, err)
	require.Len(t, search, 1)
	require.Equal(t, done.Signal.ID, search[0].Signal.ID)
}

func TestAmbientDefersRevisitUntilItsScheduledTime(t *testing.T) {
	t.Log("[REQ:SIG-P0-007] [REQ:SIG-P0-012]")
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(categories.Schema), apidb.SchemaProviderFunc(triage.Schema), apidb.SchemaProviderFunc(Schema)))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	journal := signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
	deferred, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "Review this at the next scheduled revisit"})
	require.NoError(t, err)
	revisitAt := clk.Now().Add(2 * time.Hour)
	triageService := triage.NewService(triage.NewSQLiteRepository(database), clk)
	_, err = triageService.Set(context.Background(), deferred.Signal.ID, triage.Triaged, &revisitAt)
	require.NoError(t, err)
	service := NewService(NewSQLiteRepository(database), clk)
	before, err := service.Ambient(context.Background(), "", 10)
	require.NoError(t, err)
	require.Empty(t, before, "a scheduled revisit is deferred from ambient surfacing")
	clk.Advance(2 * time.Hour)
	after, err := service.Ambient(context.Background(), "", 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, deferred.Signal.ID, after[0].Signal.ID)
	search, err := service.Search(context.Background(), Filter{Text: "scheduled revisit", Limit: 10})
	require.NoError(t, err)
	require.Len(t, search, 1, "deferral affects ambient surfacing, never the journal index")
}

func TestStructuredSearchComposesImmutableTagFilters(t *testing.T) {
	t.Log("[REQ:SIG-P0-009]")
	service, journal, _ := newService(t)
	meal, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "Preserve this recipe", Tags: []string{"Meals", "Reference"}})
	require.NoError(t, err)
	_, err = journal.Capture(context.Background(), signals.CaptureInput{Text: "Preserve this unrelated note", Tags: []string{"reference"}})
	require.NoError(t, err)
	results, err := service.Search(context.Background(), Filter{Tags: []string{"meals", "reference"}, Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, meal.Signal.ID, results[0].Signal.ID)
	require.Equal(t, []string{"meals", "reference"}, results[0].Signal.Tags)
}

func TestStructuredSearchUsesStableOpaqueCursorPagination(t *testing.T) {
	t.Log("[REQ:SIG-P0-009]")
	service, journal, _ := newService(t)
	for _, text := range []string{"first stable page", "second stable page", "third stable page"} {
		_, err := journal.Capture(context.Background(), signals.CaptureInput{Text: text})
		require.NoError(t, err)
	}
	first, err := service.SearchPage(context.Background(), Filter{Text: "stable page", Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Results, 2)
	require.NotEmpty(t, first.NextPageAfter)
	second, err := service.SearchPage(context.Background(), Filter{Text: "stable page", Limit: 2, PageAfter: first.NextPageAfter})
	require.NoError(t, err)
	require.Len(t, second.Results, 1)
	require.Empty(t, second.NextPageAfter)
	seen := map[string]bool{}
	for _, result := range append(first.Results, second.Results...) {
		seen[result.Signal.ID] = true
	}
	require.Len(t, seen, 3, "cursor pages neither duplicate nor omit immutable journal rows")
	_, err = service.SearchPage(context.Background(), Filter{PageAfter: "not-a-cursor", Limit: 2})
	require.Error(t, err)
}

func TestSemanticSearchIndexesWholeJournalWithDocumentAndQueryTasks(t *testing.T) {
	t.Log("[REQ:SIG-P0-010]")
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(categories.Schema), apidb.SchemaProviderFunc(triage.Schema), apidb.SchemaProviderFunc(Schema)))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	journal := signals.NewService(signals.NewSQLiteRepository(database, clk), clk)
	semantic, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "The blue orchard protocol protects rare fruit trees."})
	require.NoError(t, err)
	dropped, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "A discarded platform account safety note."})
	require.NoError(t, err)
	triageService := triage.NewService(triage.NewSQLiteRepository(database), clk)
	_, err = triageService.Set(context.Background(), dropped.Signal.ID, triage.Dropped, nil)
	require.NoError(t, err)
	fake := &recordingInference{}
	store := &memoryVectorStore{resultID: semantic.Signal.ID}
	service := NewService(NewSQLiteRepository(database), clk, NewSemanticSearchForStore(fake, store))
	// Use a query result containing the real signal id, which has no word in
	// common with the query. This proves the semantic path, not FTS fallback.
	results, err := service.Search(context.Background(), Filter{Text: "how can I care for unusual trees", Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, semantic.Signal.ID, results[0].Signal.ID)
	require.Equal(t, 0.91, results[0].Score)
	require.Len(t, store.points, 2, "dropped signals remain in the semantic index")
	require.Equal(t, []inference.EmbeddingTask{inference.EmbeddingDocument, inference.EmbeddingDocument, inference.EmbeddingQuery}, fake.tasks)
}

type recordingInference struct{ tasks []inference.EmbeddingTask }

func (f *recordingInference) Embed(_ context.Context, _ string, task inference.EmbeddingTask) ([]float64, error) {
	f.tasks = append(f.tasks, task)
	return []float64{0.1, 0.2}, nil
}
func (*recordingInference) Classify(context.Context, string) (string, error) { return "", nil }
