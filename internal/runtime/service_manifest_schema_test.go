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
	schema := compileRepoSchema(t, "service.schema.json")
	matches, err := filepath.Glob(filepath.Join(repoRoot(t), "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no scenario service manifests found")
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
			t.Errorf("%s: %v", path, err)
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
