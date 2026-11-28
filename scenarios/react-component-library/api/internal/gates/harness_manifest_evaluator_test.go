package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHarnessManifestAcceptsRegisteredPreviewHarness(t *testing.T) {
	root := t.TempDir()
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/manifest.json", `{"schemaVersion":1,"kind":"preview-harness-registry","ownership":"preview-only","referenceStories":"reference-stories.json","families":[{"id":"showcase","version":"1.0.0","export":"Preview","archetypes":["form"],"subjectKinds":["component"]}]}`)
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/reference-stories.json", `{"stories":[{"id":"button-default","family":"showcase","subjectKind":"component"}]}`)
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/showcase/versions/1.0.0/Preview.tsx", `export function Preview() { return null }`)
	writeHarnessFile(t, root, "scenarios/react-component-library/library/components/button/versions/1.0.0/story.json", `{"stories":[{"id":"default","composition":{"harness":{"asset":"preview.showcase","version":"1.0.0","export":"Preview","config":{}}}}]}`)

	result, err := ValidateHarnessManifest(Scope{Root: root})
	if err != nil {
		t.Fatalf("ValidateHarnessManifest() error = %v", err)
	}
	if result.Inspected != 2 {
		t.Fatalf("Inspected = %d, want manifest plus one story", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("Findings = %#v, want valid registered harness to pass", result.Findings)
	}
}

func TestValidateHarnessManifestReportsRegistrationAndIsolationViolations(t *testing.T) {
	root := t.TempDir()
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/manifest.json", `{"schemaVersion":1,"kind":"preview-harness-registry","ownership":"preview-only","referenceStories":"reference-stories.json","families":[{"id":"showcase","version":"1.0.0","export":"Preview","archetypes":["form"],"subjectKinds":["component"]}]}`)
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/reference-stories.json", `{"stories":[{"id":"wrong-subject","family":"showcase","subjectKind":"page"},{"id":"unknown-family","family":"missing","subjectKind":"component"}]}`)
	writeHarnessFile(t, root, "scenarios/react-component-library/harnesses/showcase/versions/1.0.0/Preview.tsx", `import Button from "../../components/Button"; const token = localStorage.getItem("x"); export function Preview() { return token }`)
	writeHarnessFile(t, root, "scenarios/react-component-library/library/components/button/versions/1.0.0/story.json", `{"stories":[{"id":"bad-ref","composition":{"harness":{"asset":"preview.missing","version":"1.0.0","export":"Preview","config":{}},"specimen":{}}}]}`)

	result, err := ValidateHarnessManifest(Scope{Root: root})
	if err != nil {
		t.Fatalf("ValidateHarnessManifest() error = %v", err)
	}
	joined := make([]string, 0, len(result.Findings))
	for _, finding := range result.Findings {
		joined = append(joined, finding.Message)
	}
	for _, want := range []string{
		"forbidden side effect or import",
		"subject kind is not allowed",
		"reference story family is not registered",
		"composition harness is not registered",
		"story cannot declare both specimen and composition harness",
	} {
		if !containsFragment(joined, want) {
			t.Errorf("Findings %#v do not include %q", joined, want)
		}
	}
}

func writeHarnessFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func containsFragment(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
