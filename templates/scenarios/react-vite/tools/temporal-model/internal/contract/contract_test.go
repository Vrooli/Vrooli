package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/testkit"
)

func TestLoadRawRejectsSchemaViolations(t *testing.T) {
	root := t.TempDir()
	for name, mutate := range map[string]func(map[string]any){
		"unknown property": func(body map[string]any) { body["unexpected"] = true },
		"missing required": func(body map[string]any) { delete(body, "flowId") },
		"legacy outputs block": func(body map[string]any) {
			body["outputs"] = map[string]any{"modelPath": "model.qnt"}
		},
		"legacy replay kind": func(body map[string]any) {
			replay := body["replay"].(map[string]any)
			replay["kind"] = "go-test"
		},
	} {
		t.Run(name, func(t *testing.T) {
			body := testkit.MustJSONMap(t, testkit.ValidRawContract())
			mutate(body)
			path := filepath.Join(root, strings.ReplaceAll(name, " ", "_")+".flow.json")
			testkit.WriteJSONMap(t, path, body)
			_, err := contract.LoadRaw(path, filepath.Base(path))
			testkit.RequireErrorContains(t, err, "schema validation failed")
		})
	}
}

func TestLoadRawDoesNotCompileOrMutateContract(t *testing.T) {
	root := t.TempDir()
	raw := testkit.ValidRawContract()
	raw.Traces[0].Steps[0].Want = "done"
	rel := "api/internal/example/flow/flow.json"
	path := filepath.Join(root, filepath.FromSlash(rel))
	testkit.WriteJSONMap(t, path, testkit.MustJSONMap(t, raw))

	loaded, err := contract.LoadRaw(path, rel)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if got := loaded.Traces[0].Steps[0].Want; got != "done" {
		t.Fatalf("raw trace was compiled or mutated, got want=%s", got)
	}
	if loaded.Layout.BaseDir == "" {
		t.Fatalf("Layout.BaseDir should be derived")
	}
	if !strings.HasSuffix(loaded.Layout.BaseDir, "/flow") {
		t.Fatalf("Layout.BaseDir should end in /flow, got %s", loaded.Layout.BaseDir)
	}
}

func TestValidateConventionalFilesRequiresHandAuthoredFiles(t *testing.T) {
	root := t.TempDir()
	raw := testkit.ValidTypeScriptRawContract()
	err := contract.ValidateConventionalFiles(root, raw)
	testkit.RequireErrorContains(t, err, "transition.ts is missing")

	writeBindingFile(t, root, raw.Layout.TransitionPath, "export const transitionExample = () => null\n")
	writeBindingFile(t, root, raw.Layout.FixturesPath, "export const exampleFormalFixtures = {}\n")
	writeBindingFile(t, root, raw.Layout.TestPath, "// test\n")
	if err := contract.ValidateConventionalFiles(root, raw); err != nil {
		t.Fatalf("ValidateConventionalFiles() error = %v", err)
	}
}

func writeBindingFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
