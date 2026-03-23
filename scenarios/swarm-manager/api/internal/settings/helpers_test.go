package settings

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/pathutil"
)

func TestNewStore_DefaultPath(t *testing.T) {
	store := NewStore("")
	expected := filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "settings.json")
	if store.path != expected {
		t.Fatalf("expected store path %q, got %q", expected, store.path)
	}
}
