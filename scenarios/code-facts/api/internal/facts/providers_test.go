package facts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestGraphProjectPathUsesProjectConfigForTypeScriptAndRootForJavaScript(t *testing.T) {
	unit := &factsv1.ParseUnit{RootPath: "/repo/ui", ConfigPath: "/repo/ui/package.json"}
	if got := graphProjectPath("typescript", unit); got != unit.GetConfigPath() {
		t.Fatalf("TypeScript project path = %q, want config %q", got, unit.GetConfigPath())
	}
	if got := graphProjectPath("javascript", unit); got != unit.GetRootPath() {
		t.Fatalf("JavaScript project path = %q, want root %q", got, unit.GetRootPath())
	}
}

func TestTypeScriptImportProfileIsSelectedOnlyForImportFamily(t *testing.T) {
	if !importsOnlyFamilies([]factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}) {
		t.Fatal("imports-only family was not recognized")
	}
	if importsOnlyFamilies(nil) {
		t.Fatal("empty family set must keep the full compatibility profile")
	}
	if importsOnlyFamilies([]factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_IMPORTS,
		factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
	}) {
		t.Fatal("mixed family set must keep the full compatibility profile")
	}
	if got := providerExtractionProfile(&tsGraphProvider{language: "typescript"}, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}); got != "typescript-imports-only-v1" {
		t.Fatalf("import profile cache identity = %q", got)
	}
	if got := providerExtractionProfile(&tsGraphProvider{language: "typescript"}, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SYMBOLS}); got != "typescript-owned-source-body-identity-js-v1" {
		t.Fatalf("full profile cache identity = %q", got)
	}
}

func TestGoProviderExtractsFromSharedSubstrate(t *testing.T) {
	root := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/sample\n\ngo 1.25.0\n")
	write("sample.go", "package sample\n\nfunc Answer() int { return 42 }\n")

	provider := NewGoGraphProvider().(*goGraphProvider)
	result, err := provider.ExtractForFamilies(context.Background(), &factsv1.ParseUnit{
		RootPath: root, PackagePatterns: []string{"."},
	}, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SYMBOLS})
	if err != nil {
		t.Fatal(err)
	}
	if result.GetExtractionProfile() != "structural" {
		t.Fatalf("profile = %q, want structural", result.GetExtractionProfile())
	}
	if len(result.GetOmittedInformation()) == 0 || result.GetOmittedInformation()[0].GetCapability() == "" {
		t.Fatalf("structural profile omissions missing: %+v", result.GetOmittedInformation())
	}
	if result.GetGraphHash() == "" || len(result.GetGraph().GetNodes()) == 0 {
		t.Fatalf("shared extraction returned incomplete graph: %+v", result)
	}
	found := false
	for _, node := range result.GetGraph().GetNodes() {
		if strings.Contains(node.GetId(), "go_func:") && node.GetAttributes()["kind"] == "go_func" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Go symbol kind projection missing from graph: %+v", result.GetGraph().GetNodes())
	}
}

func TestGoProviderPreservesUnsupportedWorkspaceClassification(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewGoGraphProvider()
	_, err := provider.ExtractForFamilies(context.Background(), &factsv1.ParseUnit{RootPath: root}, nil)
	var unsupported ProviderUnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Analyzer != goProviderScenario {
		t.Fatalf("workspace error = %#v, want unsupported from %s", err, goProviderScenario)
	}
}
