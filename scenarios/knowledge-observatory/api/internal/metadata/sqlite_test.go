package metadata_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/metadata"
)

func newRepo(t *testing.T) *metadata.SQLite {
	t.Helper()
	return metadata.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(metadata.Schema)))
}

func ptr(v float64) *float64 { return &v }

func TestEntryRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	accessed := time.Date(2026, 4, 5, 6, 7, 8, 0, time.UTC)
	in := metadata.Entry{
		VectorID:       "vec-1",
		CollectionName: "vrooli_knowledge",
		ContentHash:    "abc123",
		SourceScenario: "vrooli-autoheal",
		SourceType:     "doc",
		QualityScore:   ptr(0.77),
		AccessCount:    12,
		LastAccessed:   &accessed,
	}
	if err := repo.UpsertEntry(ctx, in); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := repo.GetEntry(ctx, "vec-1")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.ID == "" {
		t.Error("id was not generated")
	}
	if got.VectorID != in.VectorID || got.CollectionName != in.CollectionName {
		t.Errorf("identity = %q/%q", got.VectorID, got.CollectionName)
	}
	if got.ContentHash != in.ContentHash || got.SourceScenario != in.SourceScenario || got.SourceType != in.SourceType {
		t.Errorf("provenance = %q/%q/%q", got.ContentHash, got.SourceScenario, got.SourceType)
	}
	if got.QualityScore == nil || *got.QualityScore != 0.77 {
		t.Errorf("quality_score = %v, want 0.77", got.QualityScore)
	}
	if got.AccessCount != 12 {
		t.Errorf("access_count = %d, want 12", got.AccessCount)
	}
	if got.LastAccessed == nil || !got.LastAccessed.Equal(accessed) {
		t.Errorf("last_accessed = %v, want %v", got.LastAccessed, accessed)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("timestamps were not defaulted")
	}
}

// TestUpsertOnVectorIDPreservesQualityScore checks the COALESCE arm: a later
// upsert that carries no score must not erase one already recorded.
func TestUpsertOnVectorIDPreservesQualityScore(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if err := repo.UpsertEntry(ctx, metadata.Entry{
		VectorID: "vec-1", CollectionName: "a", QualityScore: ptr(0.5),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertEntry(ctx, metadata.Entry{
		VectorID: "vec-1", CollectionName: "b",
	}); err != nil {
		t.Fatal(err)
	}

	got, _, err := repo.GetEntry(ctx, "vec-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.CollectionName != "b" {
		t.Errorf("collection_name = %q, want the updated b", got.CollectionName)
	}
	if got.QualityScore == nil || *got.QualityScore != 0.5 {
		t.Errorf("quality_score = %v, want the preserved 0.5", got.QualityScore)
	}

	n, err := repo.CountByCollection(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 — upsert must not duplicate", n)
	}
}

// TestSourceTypeDefaultsToUnknown preserves the previous adapter's behaviour.
func TestSourceTypeDefaultsToUnknown(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if err := repo.UpsertEntry(ctx, metadata.Entry{VectorID: "v", CollectionName: "c"}); err != nil {
		t.Fatal(err)
	}
	got, _, _ := repo.GetEntry(ctx, "v")
	if got.SourceType != "unknown" {
		t.Errorf("source_type = %q, want unknown", got.SourceType)
	}
}

func TestLookupCollectionForVectorID(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, ok, err := repo.LookupCollectionForVectorID(ctx, "missing"); err != nil || ok {
		t.Fatalf("missing vector: ok=%v err=%v", ok, err)
	}
	if err := repo.UpsertEntry(ctx, metadata.Entry{VectorID: "v", CollectionName: "c"}); err != nil {
		t.Fatal(err)
	}
	collection, ok, err := repo.LookupCollectionForVectorID(ctx, "v")
	if err != nil || !ok || collection != "c" {
		t.Errorf("lookup = %q/%v/%v", collection, ok, err)
	}
}

func TestDeleteByCollection(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, v := range []string{"a1", "a2", "b1"} {
		collection := "a"
		if v == "b1" {
			collection = "b"
		}
		if err := repo.UpsertEntry(ctx, metadata.Entry{VectorID: v, CollectionName: collection}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := repo.DeleteByCollection(ctx, "a")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}
	if n, _ := repo.CountByCollection(ctx, "b"); n != 1 {
		t.Errorf("collection b count = %d, want 1 (untouched)", n)
	}
}

func TestExternalIDMappingRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	in := metadata.ExternalIDMapping{
		Namespace:   "vrooli",
		ExternalID:  "ext-9",
		Kind:        "record",
		RecordID:    "rec-9",
		ContentHash: "hash-9",
	}
	if err := repo.UpsertExternalIDMapping(ctx, in); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := repo.LookupExternalIDMapping(ctx, "vrooli", "ext-9", "record")
	if err != nil || !ok {
		t.Fatalf("lookup: ok=%v err=%v", ok, err)
	}
	if got.ID == "" {
		t.Error("id was not generated")
	}
	if got.Namespace != in.Namespace || got.ExternalID != in.ExternalID || got.Kind != in.Kind {
		t.Errorf("key = %q/%q/%q", got.Namespace, got.ExternalID, got.Kind)
	}
	if got.RecordID != in.RecordID || got.ContentHash != in.ContentHash {
		t.Errorf("payload = %q/%q", got.RecordID, got.ContentHash)
	}
	if got.DocumentID != "" {
		t.Errorf("document_id = %q, want empty", got.DocumentID)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("timestamps were not defaulted")
	}
}

// TestExternalIDMappingRejectsBadKind keeps the validation that the CHECK
// constraint also enforces, so callers get a clear error rather than a driver
// one.
func TestExternalIDMappingRejectsBadKind(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, m := range []metadata.ExternalIDMapping{
		{Namespace: "n", ExternalID: "e", Kind: "chunk", RecordID: "r"},
		{Namespace: "n", ExternalID: "e", Kind: "record"},               // record without record_id
		{Namespace: "n", ExternalID: "e", Kind: "document"},             // document without document_id
		{Namespace: "", ExternalID: "e", Kind: "record", RecordID: "r"}, // no namespace
	} {
		if err := repo.UpsertExternalIDMapping(ctx, m); err == nil {
			t.Errorf("expected rejection for %+v", m)
		}
	}
}

// TestExternalIDMappingUpsertIsIdempotent proves the composite unique key
// survived: a repeat ingest updates rather than duplicating.
func TestExternalIDMappingUpsertIsIdempotent(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	base := metadata.ExternalIDMapping{
		Namespace: "n", ExternalID: "e", Kind: "record", RecordID: "r1", ContentHash: "h1",
	}
	if err := repo.UpsertExternalIDMapping(ctx, base); err != nil {
		t.Fatal(err)
	}
	base.ContentHash = "h2"
	if err := repo.UpsertExternalIDMapping(ctx, base); err != nil {
		t.Fatal(err)
	}

	got, _, err := repo.LookupExternalIDMapping(ctx, "n", "e", "record")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentHash != "h2" {
		t.Errorf("content_hash = %q, want the updated h2", got.ContentHash)
	}
}
