package metrics

import "testing"

func TestNewRepositoryStoresDatabaseHandle(t *testing.T) {
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("expected repository")
	}
	if repo.db != nil {
		t.Fatalf("expected nil database handle, got %v", repo.db)
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
