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

func TestOptionalString(t *testing.T) {
	if optionalString("   ") != nil {
		t.Fatal("expected empty optional string to return nil")
	}
	result := optionalString(" value ")
	if result == nil || *result != "value" {
		t.Fatalf("expected trimmed value, got %v", result)
	}
}
