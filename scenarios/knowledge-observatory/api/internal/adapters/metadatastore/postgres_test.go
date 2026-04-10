package metadatastore

import (
	"context"
	"testing"

	"knowledge-observatory/internal/ports"
)

func TestMetadataStoreNoopWithoutDB(t *testing.T) {
	store := &Postgres{}
	if err := store.UpsertKnowledgeMetadata(context.Background(), "id", "collection", "hash", "scenario", "type"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.InsertIngestHistory(context.Background(), ports.IngestHistoryRow{RecordID: "r"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.InsertSearchHistory(context.Background(), ports.SearchHistoryRow{Query: ""}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if _, _, err := store.LookupCollectionForVectorID(context.Background(), "id"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.UpsertExternalIDMapping(context.Background(), ports.ExternalIDMapping{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if _, _, err := store.LookupExternalIDMapping(context.Background(), "", "", ""); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.UpsertQualityMetrics(context.Background(), ports.QualityMetricsRow{CollectionName: ""}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.UpsertCollectionStats(context.Background(), ports.CollectionStatsRow{CollectionName: ""}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := store.UpsertRelationshipEdges(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
