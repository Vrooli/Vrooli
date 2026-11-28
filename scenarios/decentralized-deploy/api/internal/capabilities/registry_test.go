package capabilities

import "testing"

// TestNewRegistryCoversEveryDeclaredCapability validates the real registry
// wiring, including each declared scenario target.
func TestNewRegistryCoversEveryDeclaredCapability(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if err := registry.Validate(); err != nil {
		t.Fatalf("NewRegistry wiring is invalid: %v", err)
	}
}
