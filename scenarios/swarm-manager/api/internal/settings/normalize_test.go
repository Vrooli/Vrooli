package settings

import "testing"

func TestNormalizeAutoSyncDefaults(t *testing.T) {
	sync := normalizeAutoSync(RecommendationAutoSync{})
	if sync.Interval != "1h" {
		t.Fatalf("expected default interval 1h, got %q", sync.Interval)
	}
	if sync.RefreshScope != "manual" {
		t.Fatalf("expected default refresh scope manual, got %q", sync.RefreshScope)
	}
}
