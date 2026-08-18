package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards"
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
	for _, name := range []string{"common.schema.json", "acquisition.schema.json", "tool.schema.json"} {
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

func TestReleaseManifestsDeclareLinuxArm64RouteOrUnsupportedReason(t *testing.T) {
	for path, data := range embeddedToolManifests(t) {
		var manifest hostreqkit.ToolManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if manifest.SourceType() != "url" || manifest.Acquisition == nil || len(manifest.Acquisition.Targets) == 0 {
			continue
		}
		if _, ok := hostreqkit.TargetFor(manifest.Acquisition, "linux", "arm64"); ok {
			continue
		}
		if reason, ok := hostreqkit.UnsupportedFor(manifest.Acquisition, "linux", "arm64"); !ok || strings.TrimSpace(reason) == "" {
			t.Errorf("%s declares URL targets but neither linux/arm64 nor an unsupported reason", path)
		}
	}
}

func embeddedToolManifests(t *testing.T) map[string][]byte {
	t.Helper()
	manifests := map[string][]byte{}
	err := fs.WalkDir(tools.Manifests, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "tool.json" {
			return nil
		}
		data, err := fs.ReadFile(tools.Manifests, path)
		if err != nil {
			return err
		}
		manifests[path] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk tool manifests: %v", err)
	}
	return manifests
}

func validateAgainst(t *testing.T, schema *jsonschema.Schema, raw string) error {
	t.Helper()
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return schema.Validate(payload)
}

// TestToolSchemaAcceptsAcquisitionTypes confirms the acquisition/requires schema
// accepts a package (implicit), URL, and directory-layout manifest.
func TestToolSchemaAcceptsAcquisitionTypes(t *testing.T) {
	schema := loadToolSchema(t)
	sha := "1111111111111111111111111111111111111111111111111111111111111111"

	cases := map[string]string{
		"package-implicit": `{"name":"jq","description":"x","commands":["jq"],"versionArgs":["--version"],"defaultPackage":"jq","bundling":"host-required","capability":"developer-utility","capability_role":"primary"}`,
		"url":              `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"bundling":"vendorable","capability":"developer-utility","capability_role":"peer","acquisition":{"kind":"url","targets":[{"when":{"os":"linux","arch":"amd64"},"url":"https://e/t","sha256":"` + sha + `"}]}}`,
		"dir-layout":       `{"name":"sd","description":"x","commands":["sd"],"versionArgs":["--help"],"bundling":"vendorable","capability":"developer-utility","capability_role":"primary","acquisition":{"kind":"url","targets":[{"when":{"os":"linux","arch":"amd64"},"url":"https://e/sd.tar.gz","sha256":"` + sha + `","archive":"tar.gz","layout":"dir","bin_path":"bin/sd","mode":"0755"}]},"requires":{"gpu":true,"minVramGb":4,"arch":["amd64"],"minRamGb":8}}`,
		"hybrid":           `{"name":"yq","description":"x","commands":["yq"],"versionArgs":["--version"],"bundling":"vendorable","capability":"developer-utility","capability_role":"primary","packages":{"brew":"yq","winget":"MikeF.yq"},"acquisition":{"kind":"url","targets":[{"when":{"os":"linux","arch":"amd64"},"url":"https://e/yq","sha256":"` + sha + `"}]}}`,
	}
	for name, raw := range cases {
		if err := validateAgainst(t, schema, raw); err != nil {
			t.Errorf("%s: unexpected validation error: %v", name, err)
		}
	}
}

// TestToolSchemaRejectsBadAcquisition confirms the schema enforces the security-
// sensitive invariants: mandatory 64-hex sha256, known archive enum, known
// source type.
func TestToolSchemaRejectsBadAcquisition(t *testing.T) {
	schema := loadToolSchema(t)
	sha := "1111111111111111111111111111111111111111111111111111111111111111"

	bad := map[string]string{
		"missing-sha256":   `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"acquisition":{"kind":"url","targets":[{"url":"https://e/t"}]}}`,
		"short-sha256":     `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"acquisition":{"kind":"url","targets":[{"url":"https://e/t","sha256":"abc"}]}}`,
		"unknown-type":     `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"acquisition":{"kind":"git","targets":[{"url":"https://e/t","sha256":"` + sha + `"}]}}`,
		"bad-archive":      `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"acquisition":{"kind":"url","targets":[{"url":"https://e/t","sha256":"` + sha + `","archive":"rar"}]}}`,
		"bad-target-key":   `{"name":"t","description":"x","commands":["t"],"versionArgs":["-v"],"acquisition":{"kind":"url","targets":[{"when":"linux","url":"https://e/t","sha256":"` + sha + `"}]}}`,
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

// TestEmbeddedSafeguardManifestsValidateAgainstSchema is intentionally strict:
// every shipped safeguard must satisfy the complete manifest contract. Do not
// filter known errors here; this test is the guard that prevents schema drift
// from becoming another runtime-only failure.
func TestEmbeddedSafeguardManifestsValidateAgainstSchema(t *testing.T) {
	schema := compileRepoSchema(t, "safeguard.schema.json")
	err := fs.WalkDir(safeguards.Manifests, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "safeguard.json" {
			return nil
		}
		data, readErr := fs.ReadFile(safeguards.Manifests, path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		var payload any
		if jsonErr := json.Unmarshal(data, &payload); jsonErr != nil {
			t.Fatalf("parse %s: %v", path, jsonErr)
		}
		if validErr := schema.Validate(payload); validErr != nil {
			t.Errorf("%s fails safeguard.schema.json: %v", path, validErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk safeguard manifests: %v", err)
	}
}

func TestOperatorStateFileValidatesAgainstSchema(t *testing.T) {
	schema := compileRepoSchema(t, "operator-state.schema.json")
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".vrooli", "operator-state.json"))
	if err != nil {
		t.Fatalf("read operator state: %v", err)
	}
	if err := schema.Validate(mustJSONPayload(t, data)); err != nil {
		t.Fatalf("operator-state.json fails schema: %v", err)
	}

	for name, raw := range map[string]string{
		"missing-version":   `{"updated_at":"2026-08-05T17:24:03Z"}`,
		"unknown-field":     `{"version":"1.0.0","updated_at":"2026-08-05T17:24:03Z","host_safeguards":{"x":{"opted_in":true,"unexpected":true}}}`,
		"wrong-choice-type": `{"version":"1.0.0","updated_at":"2026-08-05T17:24:03Z","host_safeguards":{"x":{"opted_in":"yes"}}}`,
	} {
		if err := schema.Validate(mustJSONPayload(t, []byte(raw))); err == nil {
			t.Errorf("%s: malformed operator state unexpectedly validated", name)
		}
	}
}

func mustJSONPayload(t *testing.T, data []byte) any {
	t.Helper()
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return payload
}

// TestManifestPlatformTokensHaveOneVocabulary keeps manifest declarations on
// the canonical platform vocabulary. Runtime resolver compatibility for legacy
// declarations is covered in internal/hostreq; published manifests must not
// continue adding the legacy token.
func TestManifestPlatformTokensHaveOneVocabulary(t *testing.T) {
	var violations []string
	check := func(source string, data []byte) {
		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatalf("parse %s: %v", source, err)
		}
		findLegacyPlatformToken(source, payload, &violations)
	}

	for _, sourceFS := range []struct {
		name string
		fsys fs.FS
		file string
	}{
		{name: "safeguard", fsys: safeguards.Manifests, file: "safeguard.json"},
		{name: "tool", fsys: tools.Manifests, file: "tool.json"},
	} {
		err := fs.WalkDir(sourceFS.fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || d.Name() != sourceFS.file {
				return nil
			}
			data, err := fs.ReadFile(sourceFS.fsys, path)
			if err != nil {
				return err
			}
			check(sourceFS.name+":"+path, data)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s manifests: %v", sourceFS.name, err)
		}
	}

	root := repoRoot(t)
	checkFile := func(path string) {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		check(path, data)
	}
	checkFile(filepath.Join(".vrooli", "service.json"))
	for _, base := range []string{"resources", "scenarios"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if d.Name() == "node_modules" || d.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() != "resource.json" && !strings.HasSuffix(path, filepath.Join(".vrooli", "service.json")) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			check(path, data)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s manifests: %v", base, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("legacy darwin platform token found in platforms arrays:\n%s", strings.Join(violations, "\n"))
	}
}

func TestSafeguardsDoNotReadVrooliEnvironmentConfiguration(t *testing.T) {
	root := filepath.Join(repoRoot(t), "internal", "safeguards")
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if strings.Contains(text, "os.Getenv(") && strings.Contains(text, "VROOLI_") {
			violations = append(violations, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk safeguard sources: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("safeguards read undeclared VROOLI_* environment configuration: %s", strings.Join(violations, ", "))
	}
}

func findLegacyPlatformToken(source string, value any, violations *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "platforms" {
				if values, ok := child.([]any); ok {
					for index, item := range values {
						if platform, ok := item.(string); ok && strings.EqualFold(strings.TrimSpace(platform), "darwin") {
							*violations = append(*violations, fmt.Sprintf("%s platforms[%d] = %q", source, index, platform))
						}
					}
				}
			}
			findLegacyPlatformToken(source, child, violations)
		}
	case []any:
		for _, child := range typed {
			findLegacyPlatformToken(source, child, violations)
		}
	}
}
