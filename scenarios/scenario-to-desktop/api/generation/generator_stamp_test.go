package generation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGeneratorStampFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	buildTools := filepath.Join(root, "build-tools")
	if err := os.MkdirAll(filepath.Join(buildTools, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{"template-generator.ts": "export const value = 1;\n", "package.json": "{}\n", "tsconfig.json": "{}\n"} {
		if err := os.WriteFile(filepath.Join(buildTools, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hash, err := templateGeneratorInputsHash(buildTools)
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := json.Marshal(templateGeneratorBuildStamp{Schema: "scenario-to-desktop-template-generator-build-v1", InputsHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildTools, "dist", ".build-stamp.json"), stamp, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestVerifyTemplateGeneratorStamp(t *testing.T) {
	root := writeGeneratorStampFixture(t)
	if err := verifyTemplateGeneratorStamp(root); err != nil {
		t.Fatalf("matching stamp: %v", err)
	}

	if err := os.Remove(filepath.Join(root, "build-tools", "dist", ".build-stamp.json")); err != nil {
		t.Fatal(err)
	}
	if err := verifyTemplateGeneratorStamp(root); err == nil || !strings.Contains(err.Error(), "vrooli scenario setup scenario-to-desktop") {
		t.Fatalf("missing stamp error = %v", err)
	}
}

func TestTemplateGeneratorStampTracksSourcesButNotTests(t *testing.T) {
	root := writeGeneratorStampFixture(t)
	buildTools := filepath.Join(root, "build-tools")
	original, err := templateGeneratorInputsHash(buildTools)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildTools, "template-generator.test.ts"), []byte("changed test"), 0o644); err != nil {
		t.Fatal(err)
	}
	unchanged, err := templateGeneratorInputsHash(buildTools)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != original {
		t.Fatalf("test file changed stamp hash: %s != %s", unchanged, original)
	}
	if err := os.WriteFile(filepath.Join(buildTools, "template-generator.ts"), []byte("export const value = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := templateGeneratorInputsHash(buildTools)
	if err != nil {
		t.Fatal(err)
	}
	if changed == original {
		t.Fatal("source edit did not change stamp hash")
	}
	if err := verifyTemplateGeneratorStamp(root); err == nil || !strings.Contains(err.Error(), "recorded hash") {
		t.Fatalf("stale stamp error = %v", err)
	}
}
