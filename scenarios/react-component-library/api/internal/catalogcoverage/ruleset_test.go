package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuleSetIncludesKindAndAssetDeclarations(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Clean(filepath.Join(root, "..", "..", "..", "..", ".."))

	bindings, err := ResolveRuleSet(root, "ai.conversation-shell")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]RuleBinding{}
	for _, binding := range bindings {
		seen[binding.GateID] = binding
	}
	if got, ok := seen["performance"]; !ok || got.Source != RuleSourceAsset || got.DeclaredIn == "" {
		t.Fatalf("performance binding = %+v, want asset provenance", got)
	}
	if got, ok := seen["composition"]; !ok || got.Source != RuleSourceKind {
		t.Fatalf("composition binding = %+v, want kind provenance", got)
	}
	if got, ok := seen["graph-reconciled"]; !ok || got.DeclaredIn != "scenarios/react-component-library/catalog/config.json" {
		t.Fatalf("graph-reconciled binding = %+v, want config declaration", got)
	}
}

func TestResolveRuleSetRejectsUnknownAsset(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Clean(filepath.Join(root, "..", "..", "..", "..", ".."))
	if _, err := ResolveRuleSet(root, "missing.asset"); err == nil {
		t.Fatal("expected unknown asset error")
	}
}
