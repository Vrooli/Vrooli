package queue

import (
	"swarm-manager/internal/runtimepaths"
	"testing"
)

func TestNewStore_DefaultPath(t *testing.T) {
	store := NewStore("")
	expected, err := runtimepaths.StatePath("queue.json")
	if err != nil {
		t.Fatalf("resolve queue path: %v", err)
	}
	if store.path != expected {
		t.Fatalf("expected store path %q, got %q", expected, store.path)
	}
}
