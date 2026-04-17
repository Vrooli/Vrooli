package settings

import (
	"path/filepath"
	"swarm-manager/internal/pathutil"
	"testing"
)

func TestNewStore_DefaultPath(t *testing.T) {
	store := NewStore("")
	expected := filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), "config", "settings.json")
	if store.path != expected {
		t.Fatalf("expected store path %q, got %q", expected, store.path)
	}
}
