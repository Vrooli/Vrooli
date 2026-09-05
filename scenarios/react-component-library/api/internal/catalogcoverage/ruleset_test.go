package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"

	"react-component-library/internal/gates"
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

func TestAnnotateFindingRecognizesCorpusAssetIDs(t *testing.T) {
	for _, assetID := range []string{"__corpus__.dependency-rank", "workbench.conformance", "supplemental.tokens"} {
		finding := gates.Finding{AssetID: assetID}
		if err := annotateFinding("/unused", "dependency-rank", &finding); err != nil {
			t.Fatalf("annotate corpus finding %q: %v", assetID, err)
		}
		if finding.RuleSource != gates.RuleSourceCorpus {
			t.Fatalf("%s rule source = %q, want corpus", assetID, finding.RuleSource)
		}
		if finding.RuleDeclaredIn == "" {
			t.Fatalf("%s corpus finding has no declaration source", assetID)
		}
	}
	corpusRunnerFinding := gates.Finding{AssetID: "foundations.base-styles"}
	if err := annotateFinding("/unused", "restyle-contract", &corpusRunnerFinding); err != nil {
		t.Fatalf("annotate corpus-runner finding: %v", err)
	}
	if corpusRunnerFinding.RuleSource != gates.RuleSourceCorpus {
		t.Fatalf("corpus-runner rule source = %q, want corpus", corpusRunnerFinding.RuleSource)
	}
}
