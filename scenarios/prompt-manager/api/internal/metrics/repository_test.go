package metrics

import (
	"context"
	"database/sql"
	"testing"

	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

type contextRecordingDB struct{ ctx context.Context }

const metricsTestSchema = `
CREATE TABLE IF NOT EXISTS skill_metrics (
  id TEXT PRIMARY KEY, skill_id TEXT NOT NULL UNIQUE,
  usage_count INTEGER NOT NULL DEFAULT 0, last_used TEXT,
  effectiveness_rating INTEGER, notes TEXT,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);`

func (d *contextRecordingDB) QueryContext(ctx context.Context, _ string, _ ...any) (*sql.Rows, error) {
	d.ctx = ctx
	return nil, nil
}

func (d *contextRecordingDB) QueryRowContext(ctx context.Context, _ string, _ ...any) *sql.Row {
	d.ctx = ctx
	return nil
}

func (d *contextRecordingDB) ExecContext(ctx context.Context, _ string, _ ...any) (sql.Result, error) {
	d.ctx = ctx
	return nil, nil
}

func TestNewRepositoryStoresDatabaseHandle(t *testing.T) {
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("expected repository")
	}
	if repo.db != nil {
		t.Fatalf("expected nil database handle, got %v", repo.db)
	}
}

func TestRepositoryWithContextForwardsTestModeToMutatingQuery(t *testing.T) {
	db := &contextRecordingDB{}
	repo := NewRepository(db).WithContext(database.WithTestMode(context.Background()))
	if err := repo.SetRating("skill-1", 5, nil); err != nil {
		t.Fatalf("set rating: %v", err)
	}
	if !database.IsTestMode(db.ctx) {
		t.Fatal("mutating query did not receive the request test-mode context")
	}
}

func TestRepositoryRecordsUsageAndRatingInSQLite(t *testing.T) {
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(func() string { return metricsTestSchema })); err != nil {
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
