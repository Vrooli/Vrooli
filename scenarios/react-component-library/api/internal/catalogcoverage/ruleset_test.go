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

func TestResolveRuleSetRequiresCountableValidOptOutReasons(t *testing.T) {
	catalogRoot := filepath.Join(t.TempDir(), "catalog")
	assetDir := filepath.Join(catalogRoot, "assets", "controls")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(assetDir, "button.json")
	definitions := []GateDefinition{{ID: "responsive", Tier: "asset-quality", Rung: RungVerified, Blocking: true, AppliesTo: []string{"component"}}}
	asset := Asset{ID: "controls.button", Kind: "component", Maturity: "implemented"}
	write := func(optOuts string) {
		t.Helper()
		data := `{"asset":{"id":"controls.button"},"qualityGateOptOuts":` + optOuts + `}`
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("valid opt-out resolves out of active checks", func(t *testing.T) {
		write(`[{"gate":"responsive","reason":"This source-only asset has no rendered surface."}]`)
		bindings, err := ResolveRuleSetFor(catalogRoot, asset, definitions)
		if err != nil {
			t.Fatal(err)
		}
		for _, binding := range bindings {
			if binding.GateID == "responsive" {
				t.Fatal("opted-out gate remained active")
			}
		}
	})
	for name, value := range map[string]string{
		"missing reason":    `[{"gate":"responsive"}]`,
		"short reason":      `[{"gate":"responsive","reason":"not applicable"}]`,
		"unknown gate":      `[{"gate":"missing","reason":"This gate does not exist in this catalog."}]`,
		"not a default":     `[{"gate":"other","reason":"This source-only asset has no rendered surface."}]`,
		"duplicate":         `[{"gate":"responsive","reason":"This source-only asset has no rendered surface."},{"gate":"responsive","reason":"The second declaration is a duplicate."}]`,
		"blocking maturity": `[{"gate":"responsive","reason":"This source-only asset has no rendered surface."}]`,
	} {
		t.Run(name, func(t *testing.T) {
			current := definitions
			currentAsset := asset
			if name == "not a default" {
				current = append(current, GateDefinition{ID: "other", Tier: "asset-quality", Rung: RungVerified, AppliesTo: []string{"primitive"}})
			}
			if name == "blocking maturity" {
				currentAsset.Maturity = "verified"
			}
			write(value)
			if _, err := ResolveRuleSetFor(catalogRoot, currentAsset, current); err == nil {
				t.Fatal("invalid opt-out was accepted")
			}
		})
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
