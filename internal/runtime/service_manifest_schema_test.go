package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Every scenario service.json must validate against the service schema. The
// existing schema coverage stops at the first manifest that happens to
// validate, so a later scenario could carry an invalid platform_capabilities
// declaration unnoticed.
func TestEveryScenarioServiceManifestValidates(t *testing.T) {
	validateEveryManifest(t, "service.schema.json", filepath.Join(repoRoot(t), "scenarios", "*", ".vrooli", "service.json"))
}

// Every resource manifest must validate against the resource schema. The Go
// loader decodes into typed structs and ignores enum vocabulary, so without
// this sweep a manifest could carry a value the schema forbids (for example an
// undeclared requirements.source) and nothing would notice.
func TestEveryResourceManifestValidates(t *testing.T) {
	validateEveryManifest(t, "resource.schema.json", filepath.Join(repoRoot(t), "resources", "*", "resource.json"))
}

func validateEveryManifest(t *testing.T, schemaName, pattern string) {
	t.Helper()
	schema := compileRepoSchema(t, schemaName)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no manifests match %s", pattern)
	}
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if err := schema.Validate(doc); err != nil {
			t.Errorf("%s: %#v", path, err)
		}
	}
}

// mechanism is what separates "deliberately absent" from "a real mechanism
// nobody wired": the capability resolver reads a named mechanism on an
// unsupported platform as unwired and an absent one as ineligible. The schema
// enforces the distinction so a placeholder mechanism cannot park a genuine
// no-equivalent case in the wire-me-up queue.
func TestPlatformCapabilityRequiresMechanismExceptWhenUnsupported(t *testing.T) {
	schema := compileRepoSchema(t, "service.schema.json#/definitions/platformCapability")

	cases := []struct {
		name     string
		document string
		valid    bool
	}{
		{"supported names its mechanism", `{"status":"supported","mechanism":"procfs","evidence":"linux contract tests"}`, true},
		{"supported without a mechanism", `{"status":"supported","evidence":"linux contract tests"}`, false},
		{"build-verified without a mechanism", `{"status":"build-verified","evidence":"cross-build only"}`, false},
		{"unsupported omits its mechanism", `{"status":"unsupported","evidence":"no equivalent interface exists on this platform"}`, true},
		{"unsupported may still name an unwired mechanism", `{"status":"unsupported","mechanism":"powermetrics","evidence":"exists but nobody wired it"}`, true},
		{"evidence is always required", `{"status":"unsupported"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc any
			if err := json.Unmarshal([]byte(tc.document), &doc); err != nil {
				t.Fatalf("decode: %v", err)
			}
			err := schema.Validate(doc)
			if tc.valid && err != nil {
				t.Errorf("expected accepted, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected rejected, got none")
			}
		})
	}
}

func TestBuildIdentityExclusionsAreExactRepositoryPaths(t *testing.T) {
	schema := compileRepoSchema(t, "service.schema.json#/properties/build_identity")
	cases := []struct {
		name     string
		document string
		valid    bool
	}{
		{"explicit operational document", `{"exclude_paths":["docs/internal/REFRACTOR_PROGRESS.md"]}`, true},
		{"wildcard is not an exact path", `{"exclude_paths":["docs/**"]}`, false},
		{"absolute path", `{"exclude_paths":["/outside"]}`, false},
		{"duplicate path", `{"exclude_paths":["docs/a.md","docs/a.md"]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var document any
			if err := json.Unmarshal([]byte(tc.document), &document); err != nil {
				t.Fatalf("decode: %v", err)
			}
			err := schema.Validate(document)
			if tc.valid && err != nil {
				t.Errorf("expected accepted, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected rejected, got none")
			}
		})
	}
}

func TestBuildIdentityRuntimeDocsAreDeclaredDocumentationPaths(t *testing.T) {
	schema := compileRepoSchema(t, "service.schema.json#/properties/build_identity")
	cases := []struct {
		name     string
		document string
		valid    bool
	}{
		{"runtime-served document", `{"runtime_docs":["docs/guides/setup.md"]}`, true},
		{"docs root is not a selected path", `{"runtime_docs":["docs"]}`, false},
		{"path traversal", `{"runtime_docs":["docs/../README.md"]}`, false},
		{"outside docs tree", `{"runtime_docs":["README.md"]}`, false},
		{"duplicate path", `{"runtime_docs":["docs/guide.md","docs/guide.md"]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var document any
			if err := json.Unmarshal([]byte(tc.document), &document); err != nil {
				t.Fatalf("decode: %v", err)
			}
			err := schema.Validate(document)
			if tc.valid && err != nil {
				t.Errorf("expected accepted, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected rejected, got none")
			}
		})
	}
}

// The managed-service environment file is read beneath RESOURCE_DATA_DIR, so
// the schema refuses the same escapes the Go loader refuses.
func TestManagedServiceEnvironmentFileStaysUnderTheDataDirectory(t *testing.T) {
	schema := compileRepoSchema(t, "resource.schema.json#/definitions/managedService/properties/environment_file")
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{`"active.env"`, true},
		{`"state/active.env"`, true},
		{`"..hidden.env"`, true},
		{`""`, false},
		{`"/etc/active.env"`, false},
		{`"../active.env"`, false},
		{`"state/../../active.env"`, false},
	} {
		var doc any
		if err := json.Unmarshal([]byte(tc.value), &doc); err != nil {
			t.Fatalf("decode %s: %v", tc.value, err)
		}
		if err := schema.Validate(doc); (err == nil) != tc.valid {
			t.Errorf("environment_file %s: valid=%t, err=%v", tc.value, tc.valid, err)
		}
	}
}

// An asset component is build-only (for example a Vite bundle a Go UI server
// serves): it must not declare run, and every other role must.
func TestAssetComponentsAreBuildOnly(t *testing.T) {
	schema := compileRepoSchema(t, "service.schema.json#/definitions/component")
	cases := []struct {
		name  string
		doc   string
		valid bool
	}{
		{"asset without run", `{"role":"asset","build":{"kind":"pnpm_vite","dir":"ui"}}`, true},
		{"asset with run", `{"role":"asset","build":{"kind":"pnpm_vite","dir":"ui"},"run":{"argv":["node","x.js"]}}`, false},
		{"ui without run", `{"role":"ui","build":{"kind":"pnpm_vite","dir":"ui"}}`, false},
		{"ui with run", `{"role":"ui","build":{"kind":"pnpm_vite","dir":"ui"},"run":{"argv":["node","x.js"]}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc any
			if err := json.Unmarshal([]byte(tc.doc), &doc); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(doc); (err == nil) != tc.valid {
				t.Fatalf("valid=%t, want %t: %v", err == nil, tc.valid, err)
			}
		})
	}
}
