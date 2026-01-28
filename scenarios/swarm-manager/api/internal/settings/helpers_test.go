package settings

import (
	"path/filepath"
	"testing"
)

func TestNewStore_DefaultPath(t *testing.T) {
	store := NewStore("")
	expected := filepath.Join("scenarios", "swarm-manager", ".vrooli", "settings.json")
	if store.path != expected {
		t.Fatalf("expected store path %q, got %q", expected, store.path)
	}
}

func TestNormalizeRecommendationSources(t *testing.T) {
	if normalizeRecommendationSources(RecommendationSources{}) != DefaultRecommendationSources() {
		t.Fatalf("expected empty sources to default")
	}

	custom := RecommendationSources{Problems: false, Completeness: true, Tests: false, Coverage: true, CustomFocus: false, ScenarioNotes: true}
	if normalizeRecommendationSources(custom) != custom {
		t.Fatalf("expected non-empty sources to remain unchanged")
	}
}

func TestApplyRecommendationSources(t *testing.T) {
	current := RecommendationSources{Problems: true, Completeness: true, Tests: true, Coverage: true, CustomFocus: true, ScenarioNotes: true}
	patch := &RecommendationSourcesPatch{
		Problems:      boolPtr(false),
		Completeness:  boolPtr(false),
		Tests:         boolPtr(false),
		Coverage:      boolPtr(false),
		CustomFocus:   boolPtr(false),
		ScenarioNotes: boolPtr(false),
	}

	applyRecommendationSources(&current, patch)

	if current != (RecommendationSources{}) {
		t.Fatalf("expected all fields false, got %+v", current)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
