package generation

import (
	"encoding/json"
	"os"
	"os/exec"
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
	for name, contents := range map[string]string{"template-generator.ts": "export const value = 1;\n", "package.json": "{}\n", "tsconfig.json": "{}\n", "write-build-stamp.js": "// producer fixture\n"} {
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

func TestTemplateGeneratorStampNodeReadOnlyParity(t *testing.T) {
	root := writeGeneratorStampFixture(t)
	buildTools := filepath.Join(root, "build-tools")
	producer, err := os.ReadFile(filepath.Join("..", "..", "templates", "build-tools", "write-build-stamp.js"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildTools, "write-build-stamp.js"), producer, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Z.ts", "a.ts", "é.ts", "𐀀.ts"} {
		if err := os.WriteFile(filepath.Join(buildTools, name), []byte("export {};\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(buildTools, "dist", "template-generator.js"), []byte("// compiled fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", filepath.Join(buildTools, "write-build-stamp.js"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture stamp producer: %v: %s", err, output)
	}
	if err := verifyTemplateGeneratorStamp(root); err != nil {
		t.Fatalf("Node producer/Go verifier disagree: %v", err)
	}
	stampPath := filepath.Join(buildTools, "dist", ".build-stamp.json")
	before, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("node", filepath.Join(buildTools, "write-build-stamp.js"), "--check")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("read-only owner checker: %v: %s", err, output)
	}
	after, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("read-only check rewrote the stamp")
	}
	if err := os.WriteFile(filepath.Join(buildTools, "write-build-stamp.js"), append(producer, []byte("\n// changed producer\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyTemplateGeneratorStamp(root); err == nil {
		t.Fatal("Go verifier accepted changed producer")
	}
	cmd = exec.Command("node", filepath.Join(buildTools, "write-build-stamp.js"), "--check")
	if output, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(output), "stale") {
		t.Fatalf("Node checker accepted changed producer: %v: %s", err, output)
	}
}
