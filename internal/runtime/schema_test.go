package runtime

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/vrooli/internal/tools"
)

// loadToolSchema compiles the on-disk tool.schema.json (plus its common.schema
// dependency) so tests validate against the same contract the docs declare.
func loadToolSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine schema dir")
	}
	schemaDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".vrooli", "schemas"))

	compiler := jsonschema.NewCompiler()
	for _, name := range []string{"common.schema.json", "tool.schema.json"} {
		data, err := os.ReadFile(filepath.Join(schemaDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := compiler.AddResource(name, bytes.NewReader(data)); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}
	schema, err := compiler.Compile("tool.schema.json")
	if err != nil {
		t.Fatalf("compile tool schema: %v", err)
	}
	return schema
}

func validateAgainst(t *testing.T, schema *jsonschema.Schema, raw string) error {
	t.Helper()
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return schema.Validate(payload)
}

// TestToolSchemaAcceptsSourceTypes confirms the additive source/requires schema
// accepts a package (implicit), url, and release manifest.
func TestToolSchemaAcceptsSourceTypes(t *testing.T) {
	schema := loadToolSchema(t)
	sha := "1111111111111111111111111111111111111111111111111111111111111111"

	cases := map[string]string{
		"package-implicit": `{"name":"jq","description":"x","commands":["jq"],"versionArgs":["--version"],"defaultPackage":"jq","bundling":"host-required"}`,
		"url":              `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"bundling":"vendorable","source":{"type":"url","targets":{"linux/amd64":{"url":"https://e/t","sha256":"` + sha + `"}}}}`,
		"release":          `{"name":"sd","description":"x","commands":["sd"],"versionArgs":["--help"],"bundling":"vendorable","source":{"type":"release","targets":{"linux/amd64":{"url":"https://e/sd.tar.gz","sha256":"` + sha + `","archive":"tar.gz","binPath":"bin/sd","mode":"0755"}}},"requires":{"gpu":true,"minVramGb":4,"arch":["amd64"],"minRamGb":8}}`,
		"hybrid":           `{"name":"yq","description":"x","commands":["yq"],"versionArgs":["--version"],"bundling":"vendorable","packages":{"brew":"yq","winget":"MikeF.yq"},"source":{"type":"release","targets":{"linux/amd64":{"url":"https://e/yq","sha256":"` + sha + `"}}}}`,
	}
	for name, raw := range cases {
		if err := validateAgainst(t, schema, raw); err != nil {
			t.Errorf("%s: unexpected validation error: %v", name, err)
		}
	}
}

// TestToolSchemaRejectsBadSource confirms the schema enforces the security-
// sensitive invariants: mandatory 64-hex sha256, known archive enum, known
// source type.
func TestToolSchemaRejectsBadSource(t *testing.T) {
	schema := loadToolSchema(t)
	sha := "1111111111111111111111111111111111111111111111111111111111111111"

	bad := map[string]string{
		"missing-sha256":   `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"source":{"type":"url","targets":{"linux/amd64":{"url":"https://e/t"}}}}`,
		"short-sha256":     `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"source":{"type":"url","targets":{"linux/amd64":{"url":"https://e/t","sha256":"abc"}}}}`,
		"unknown-type":     `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"source":{"type":"git","targets":{"linux/amd64":{"url":"https://e/t","sha256":"` + sha + `"}}}}`,
		"bad-archive":      `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"source":{"type":"url","targets":{"linux/amd64":{"url":"https://e/t","sha256":"` + sha + `","archive":"rar"}}}}`,
		"bad-target-key":   `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"source":{"type":"url","targets":{"linux":{"url":"https://e/t","sha256":"` + sha + `"}}}}`,
		"unknown-property": `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"requires":{"cpu":true}}`,
	}
	for name, raw := range bad {
		if err := validateAgainst(t, schema, raw); err == nil {
			t.Errorf("%s: expected validation error, got none", name)
		}
	}
}

// TestEmbeddedToolManifestsValidateAgainstSchema validates every shipped
// tool.json against the schema, so a malformed source/requires block in a real
// manifest is a red build.
func TestEmbeddedToolManifestsValidateAgainstSchema(t *testing.T) {
	schema := loadToolSchema(t)
	err := fs.WalkDir(tools.Manifests, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "tool.json" {
			return nil
		}
		data, readErr := fs.ReadFile(tools.Manifests, path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		var payload any
		if jsonErr := json.Unmarshal(data, &payload); jsonErr != nil {
			t.Fatalf("parse %s: %v", path, jsonErr)
		}
		if validErr := schema.Validate(payload); validErr != nil {
			t.Errorf("%s fails tool.schema.json: %v", path, validErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk tool manifests: %v", err)
	}
}
