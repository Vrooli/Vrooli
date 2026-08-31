package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateVersionShapeReportsExactMissingAndForbiddenFiles(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "version-shape.json"), []byte(`{"enforcement":"new-only","cutoff":"2020-01-01T00:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	versionDir := filepath.Join(root, "library", "components", "Button", "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"Button.tsx": "export {}", "story.tsx": "export {}", "story.json": "{}", "unexpected.txt": "bad"} {
		if err := os.WriteFile(filepath.Join(versionDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	problems, err := ValidateVersionShape(root, versionDir, "Button", false)
	if err != nil {
		t.Fatal(err)
	}
	joined := FormatVersionShapeFindings(problems)
	for _, expected := range []string{"./unexpected.txt", "./dependencies.json: required generated dependency lock is missing"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("shape findings %q do not contain %q", joined, expected)
		}
	}
}
