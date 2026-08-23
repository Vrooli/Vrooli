package adapterregistry

import (
	"testing"

	"unit-health/internal/adapters/reactvitest"
)

func TestDefaultResolvesOnlyDeclaredTypeScriptAdapter(t *testing.T) {
	r := Default()
	if analyzer, ok := r.Resolve("", "typescript", "vite"); !ok || analyzer.Identity().ID != "react-vitest" {
		t.Fatalf("vite analyzer = %v, %v", analyzer, ok)
	}
	if _, ok := r.Resolve("", "typescript", "jest"); ok {
		t.Fatal("Jest unexpectedly inherited the React/Vite analyzer")
	}
	if _, ok := r.Resolve("", "python", "pytest"); ok {
		t.Fatal("Python unexpectedly inherited a TypeScript analyzer")
	}
	if _, ok := r.Resolve("react-vitest", "node", "vitest"); ok {
		t.Fatal("generic Node/Vitest unexpectedly inherited the React/Vitest analyzer")
	}
}

func TestRegistryRejectsDuplicateAnalyzerIdentity(t *testing.T) {
	r := New()
	if err := r.Register(reactvitest.Analyzer{}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(reactvitest.Analyzer{}); err == nil {
		t.Fatal("duplicate analyzer identity was accepted")
	}
}
