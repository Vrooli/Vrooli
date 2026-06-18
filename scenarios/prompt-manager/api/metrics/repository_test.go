package metrics

import (
	"context"
	"testing"

	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

func TestNewRepositoryStoresDatabaseHandle(t *testing.T) {
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("expected repository")
	}
	if repo.db != nil {
		t.Fatalf("expected nil database handle, got %v", repo.db)
	}
}

func TestRepositoryRecordsUsageAndRatingInSQLite(t *testing.T) {
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("apply metrics schema: %v", err)
	}
	repo := NewRepository(db.Primary())

	count, lastUsed, err := repo.RecordUsage("skill.core/storage-steer")
	if err != nil {
		t.Fatalf("record usage: %v", err)
	}
	if count != 1 || lastUsed.IsZero() {
		t.Fatalf("unexpected first usage count/time: %d %v", count, lastUsed)
	}
	count, _, err = repo.RecordUsage("skill.core/storage-steer")
	if err != nil {
		t.Fatalf("record second usage: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected second usage count 2, got %d", count)
	}

	notes := "portable storage"
	if err := repo.SetRating("skill.core/storage-steer", 5, &notes); err != nil {
		t.Fatalf("set rating: %v", err)
	}
	got, err := repo.Get("skill.core/storage-steer")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if got == nil || got.UsageCount != 2 || got.EffectivenessRating == nil || *got.EffectivenessRating != 5 {
		t.Fatalf("unexpected metrics: %+v", got)
	}
	if got.LastUsed == nil || got.LastUsed.IsZero() {
		t.Fatalf("expected parsed last_used, got %+v", got)
	}
	if got.Notes == nil || *got.Notes != notes {
		t.Fatalf("unexpected notes: %+v", got.Notes)
	}
}

func TestMetricsResponseModelsPreserveFields(t *testing.T) {
	rating := 4
	notes := "useful"
	metrics := SkillMetrics{
		SkillID:             "skill-1",
		UsageCount:          3,
		EffectivenessRating: &rating,
		Notes:               &notes,
	}

	if metrics.SkillID != "skill-1" || metrics.UsageCount != 3 {
		t.Fatalf("unexpected metrics identity: %+v", metrics)
	}
	if metrics.EffectivenessRating == nil || *metrics.EffectivenessRating != 4 {
		t.Fatalf("unexpected rating: %+v", metrics.EffectivenessRating)
	}
	if metrics.Notes == nil || *metrics.Notes != "useful" {
		t.Fatalf("unexpected notes: %+v", metrics.Notes)
	}
}
