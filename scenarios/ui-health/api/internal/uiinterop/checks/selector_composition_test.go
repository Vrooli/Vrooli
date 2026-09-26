package checks

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestSelectorCompositionUsesDeclaredPaths(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"ui/package.json":                `{"dependencies":{"@vrooli/react-component-library":"file:library"}}`,
		"ui/manifest.json":               `{"files":{"selectorRegistry":{"path":"ui/app/selectors.ts"},"librarySelectors":{"path":"ui/generated/selectors.ts"},"appEntry":{"path":"ui/app/main.tsx"}}}`,
		"ui/app/main.tsx":                "LibraryStringsProvider i18n.t",
		"ui/generated/selectors.ts":      "export const librarySelectors = { account: { status: 'account-status' } };",
		"ui/app/selectors.ts":            "createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors)",
		"ui/app/selectors.manifest.json": `{"selectors":{"library.account.status":{"testId":"account-status","selector":"[data-testid=\"account-status\"]"}}}`,
	}
	var manifest map[string]any
	json.Unmarshal([]byte(files["ui/app/selectors.manifest.json"]), &manifest)
	manifest["sources"] = map[string]string{"ui/app/selectors.ts": fmt.Sprintf("%x", sha256.Sum256([]byte(files["ui/app/selectors.ts"]))), "ui/generated/selectors.ts": fmt.Sprintf("%x", sha256.Sum256([]byte(files["ui/generated/selectors.ts"])))}
	raw, _ := json.Marshal(manifest)
	files["ui/app/selectors.manifest.json"] = string(raw)
	for p, body := range files {
		path := filepath.Join(root, p)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(body), 0o644)
	}
	result := checkComponentAdoptionContracts(uiinterop.CheckContext{ScenarioRoot: root})
	if !result.Passed {
		t.Fatalf("%+v", result)
	}
	os.WriteFile(filepath.Join(root, "ui/app/selectors.ts"), []byte("import { librarySelectors } from '../generated/selectors'; createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);"), 0o644)
	if checkComponentAdoptionContracts(uiinterop.CheckContext{ScenarioRoot: root}).Passed {
		t.Fatal("uncomposed library import passed")
	}
	os.WriteFile(filepath.Join(root, "ui/app/selectors.ts"), []byte(files["ui/app/selectors.ts"]), 0o644)
	os.WriteFile(filepath.Join(root, "ui/generated/selectors.ts"), []byte("export const librarySelectors = {};"), 0o644)
	if checkComponentAdoptionContracts(uiinterop.CheckContext{ScenarioRoot: root}).Passed {
		t.Fatal("stale library composition passed")
	}
	os.Remove(filepath.Join(root, "ui/app/selectors.manifest.json"))
	result = checkComponentAdoptionContracts(uiinterop.CheckContext{ScenarioRoot: root})
	if result.Passed {
		t.Fatal("unexported composition passed")
	}
}
