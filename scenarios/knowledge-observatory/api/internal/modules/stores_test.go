package modules_test

import (
	"context"
	"testing"

	docaccessmocks "knowledge-observatory/internal/docaccess/mocks"
	graphmocks "knowledge-observatory/internal/graph/mocks"
	ingestmocks "knowledge-observatory/internal/ingest/mocks"
	metadatamocks "knowledge-observatory/internal/metadata/mocks"
	"knowledge-observatory/internal/modules"
	"knowledge-observatory/internal/ports"
	qualitymocks "knowledge-observatory/internal/quality/mocks"
	searchmocks "knowledge-observatory/internal/search/mocks"
)

type fakes struct {
	quality   *qualitymocks.Repository
	search    *searchmocks.Repository
	ingest    *ingestmocks.Repository
	metadata  *metadatamocks.Repository
	graph     *graphmocks.Repository
	docaccess *docaccessmocks.Repository
}

func newStores(t *testing.T) (*modules.Stores, *fakes) {
	t.Helper()
	f := &fakes{
		quality:   qualitymocks.New(),
		search:    searchmocks.New(),
		ingest:    ingestmocks.New(),
		metadata:  metadatamocks.New(),
		graph:     graphmocks.New(),
		docaccess: docaccessmocks.New(),
	}
	return &modules.Stores{
		Quality:   f.quality,
		Search:    f.search,
		Ingest:    f.ingest,
		Metadata:  f.metadata,
		Graph:     f.graph,
		DocAccess: f.docaccess,
	}, f
}

// TestMetadataStoreFansOutToTheOwningDomain is the point of the composite:
// ports.MetadataStore spans five domains, and each write has to land in the
// domain that owns the table rather than in one catch-all store.
func TestMetadataStoreFansOutToTheOwningDomain(t *testing.T) {
	stores, f := newStores(t)
	store := modules.NewMetadataStore(stores)
	ctx := context.Background()

	if err := store.UpsertKnowledgeMetadata(ctx, "vec-1", "coll", "hash", "scenario", "doc"); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if len(f.metadata.Entries) != 1 {
		t.Errorf("knowledge metadata landed in %d entries, want 1 in the metadata domain", len(f.metadata.Entries))
	}

	if err := store.InsertIngestHistory(ctx, ports.IngestHistoryRow{
		RecordID: "r", Namespace: "n", Collection: "coll", Visibility: "shared", Status: "success",
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if len(f.ingest.History) != 1 {
		t.Errorf("ingest history landed in %d rows, want 1 in the ingest domain", len(f.ingest.History))
	}

	if err := store.InsertSearchHistory(ctx, ports.SearchHistoryRow{Query: "q", ResultCount: 3}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(f.search.History) != 1 {
		t.Errorf("search history landed in %d rows, want 1 in the search domain", len(f.search.History))
	}

	if err := store.UpsertQualityMetrics(ctx, ports.QualityMetricsRow{
		CollectionName: "coll", TotalEntries: 9,
	}); err != nil {
		t.Fatalf("quality metrics: %v", err)
	}
	if len(f.quality.Metrics) != 1 {
		t.Errorf("quality metrics landed in %d rows, want 1 in the quality domain", len(f.quality.Metrics))
	}

	if err := store.UpsertCollectionStats(ctx, ports.CollectionStatsRow{
		CollectionName: "coll", TotalEntries: 9,
	}); err != nil {
		t.Fatalf("collection stats: %v", err)
	}
	if len(f.quality.Stats) != 1 {
		t.Errorf("collection stats landed in %d rows, want 1 in the quality domain", len(f.quality.Stats))
	}

	if err := store.UpsertRelationshipEdges(ctx, []ports.RelationshipEdgeRow{
		{SourceID: "a", TargetID: "b", RelationshipType: "semantic_similarity", Weight: 0.7},
	}); err != nil {
		t.Fatalf("edges: %v", err)
	}
	if len(f.graph.Edges) != 1 {
		t.Errorf("edges landed in %d rows, want 1 in the graph domain", len(f.graph.Edges))
	}
}

func TestMetadataStoreReadsRoundTrip(t *testing.T) {
	stores, _ := newStores(t)
	store := modules.NewMetadataStore(stores)
	ctx := context.Background()

	if err := store.UpsertKnowledgeMetadata(ctx, "vec-1", "coll", "", "", ""); err != nil {
		t.Fatal(err)
	}
	collection, ok, err := store.LookupCollectionForVectorID(ctx, "vec-1")
	if err != nil || !ok || collection != "coll" {
		t.Errorf("lookup = %q/%v/%v", collection, ok, err)
	}

	if err := store.UpsertExternalIDMapping(ctx, ports.ExternalIDMapping{
		Namespace: "n", ExternalID: "e", Kind: "record", RecordID: "r", ContentHash: "h",
	}); err != nil {
		t.Fatal(err)
	}
	mapping, ok, err := store.LookupExternalIDMapping(ctx, "n", "e", "record")
	if err != nil || !ok {
		t.Fatalf("mapping lookup: ok=%v err=%v", ok, err)
	}
	if mapping.RecordID != "r" || mapping.ContentHash != "h" {
		t.Errorf("mapping = %+v", mapping)
	}
}

// TestMetadataStoreOnNilStoresIsInert protects the boot path: when the database
// is skipped, the composite must no-op rather than panic.
func TestMetadataStoreOnNilStoresIsInert(t *testing.T) {
	if store := modules.NewMetadataStore(nil); store != nil {
		t.Fatal("NewMetadataStore(nil) should return nil so callers can test for absence")
	}

	var store *modules.MetadataStore
	ctx := context.Background()
	if err := store.UpsertKnowledgeMetadata(ctx, "v", "c", "", "", ""); err != nil {
		t.Errorf("nil composite should be inert, got %v", err)
	}
	if _, ok, err := store.LookupCollectionForVectorID(ctx, "v"); ok || err != nil {
		t.Errorf("nil composite lookup = %v/%v", ok, err)
	}
}

func TestDocAccessLoggerDelegates(t *testing.T) {
	stores, f := newStores(t)
	logger := modules.NewDocAccessLogger(stores.DocAccess)
	ctx := context.Background()

	for _, op := range []string{"read", "read", "write"} {
		if err := logger.LogAccess(ctx, ports.DocAccessRow{
			ScenarioName: "alpha", DocType: "guide", Operation: op,
		}); err != nil {
			t.Fatalf("log: %v", err)
		}
	}
	if len(f.docaccess.Entries) != 3 {
		t.Fatalf("logged %d entries, want 3", len(f.docaccess.Entries))
	}

	stats, err := logger.QueryStats(ctx, ports.DocAccessFilter{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("got %d stat groups, want 1", len(stats))
	}
	if stats[0].ReadCount != 2 || stats[0].WriteCount != 1 {
		t.Errorf("tallies = %d/%d, want 2/1", stats[0].ReadCount, stats[0].WriteCount)
	}
}

func TestDocAccessLoggerOnNilRepoIsInert(t *testing.T) {
	if logger := modules.NewDocAccessLogger(nil); logger != nil {
		t.Fatal("NewDocAccessLogger(nil) should return nil")
	}
	var logger *modules.DocAccessLogger
	if err := logger.LogAccess(context.Background(), ports.DocAccessRow{}); err != nil {
		t.Errorf("nil logger should be inert, got %v", err)
	}
}
