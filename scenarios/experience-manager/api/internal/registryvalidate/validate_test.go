package registryvalidate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReportsBrokenRegistryDocument(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	registryDir := filepath.Join(root, "scenarios", "experience-manager", "capabilities", "capabilities")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", ".vrooli", "schemas", "experience-capability-registry.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "experience-capability-registry.schema.json"), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Dir(registryDir)
	for name, body := range map[string]string{
		"index.json":               `{"kind":"capability-registry-index","contract":{"kind":"experience-capability-registry","schema":"experience-capability-registry/v1"},"schemaVersion":"1.0.0","documents":[]}`,
		"states.json":              `{"kind":"state-vocabulary","schemaVersion":"1.0.0","canonical":[{"id":"default","group":"resting","description":"A settled usable state."}],"groups":[{"id":"resting","description":"Settled states."}],"views":{"region-lifecycle":{"description":"Region states.","states":["default"]}}}`,
		"axes.json":                `{"kind":"axis-registry","schemaVersion":"1.0.0","axes":[{"id":"viewport","title":"Viewport","description":"Viewport size axis.","values":[{"id":"desktop","description":"Desktop."}],"default":"desktop","mechanism":{"kind":"test"}}]}`,
		"evidence.json":            `{"kind":"evidence-registry","schemaVersion":"1.0.0","evidence":[{"id":"ax-tree","title":"AX tree","description":"Accessibility tree evidence.","producer":{"kind":"test"}}]}`,
		"capabilities/broken.json": `{"kind":"capability-group","schemaVersion":"1.0.0","group":{"id":"broken","description":"A broken group."},"capabilities":[{"id":"broken-cap","title":"Broken","facets":[],"description":"This capability has no declared facet."}]}`,
	} {
		path := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := New(root).Validate()
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Code == "registry.schema_error" || strings.Contains(finding.Message, "no facet") {
			return
		}
	}
	t.Fatalf("findings = %+v", findings)
}
