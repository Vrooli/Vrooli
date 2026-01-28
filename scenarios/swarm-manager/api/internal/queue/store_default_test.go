package queue

import (
	"path/filepath"
	"testing"
)

func TestNewStore_DefaultPath(t *testing.T) {
	store := NewStore("")
	expected := filepath.Join("scenarios", "swarm-manager", ".vrooli", "queue.json")
	if store.path != expected {
		t.Fatalf("expected store path %q, got %q", expected, store.path)
	}
}
