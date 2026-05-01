package modelchain

import (
	"testing"

	"agent-manager/internal/modelregistry"
)

func TestFakeResolverReturnsCopy(t *testing.T) {
	resolver := NewFakeResolver(modelregistry.PresetChain{"primary", "fallback"})

	got, ok := resolver.ResolvePreset("codex", "SMART")
	if !ok {
		t.Fatal("expected preset to resolve")
	}
	got[0] = "mutated"

	again, _ := resolver.ResolvePreset("codex", "SMART")
	if again[0] != "primary" {
		t.Fatalf("resolver returned mutable chain, got %v", again)
	}
}

func TestFakeResolverEmptyChainMisses(t *testing.T) {
	resolver := NewFakeResolver(nil)

	if got, ok := resolver.ResolvePreset("codex", "SMART"); ok || got != nil {
		t.Fatalf("expected empty resolver miss, got ok=%v chain=%v", ok, got)
	}
}
