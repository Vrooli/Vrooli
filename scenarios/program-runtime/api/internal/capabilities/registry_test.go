package capabilities

import "testing"

func TestRegistryCanBeConstructed(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("nil registry")
	}
	if err := registry.Validate(); err != nil {
		t.Fatalf("capability registry should validate: %v", err)
	}
}
