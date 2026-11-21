package depsapproved

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const runtimeManifest = `{
  "unrelated": {"preserve": [1, 2], "scenarios": {"decoy": true}},
  "dependencies": {
    "resources": {"keep": {"enabled": true}},
    "scenarios": {
      "peer": {"required": true, "startup_policy": "must_start", "unknown": 42}
    }
  },
  "lifecycle": {"start": "never execute this"}
}
`

func runtimeFixture(t *testing.T) (string, string, runtimeDeclaration) {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"consumer", "source"} {
		p := filepath.Join(root, "scenarios", name, ".vrooli")
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "service.json"), []byte(runtimeManifest), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, filepath.Join(root, "scenarios", "consumer", ".vrooli", "service.json"), runtimeDeclaration{Scenario: "consumer", Dependency: "source", Description: "Read current state", Rationale: "Runtime HTTP without source package imports", DegradedBehavior: "Unavailable source remains unverified"}
}
func TestDeclareRuntimePreviewApplyIsolationAndRepeat(t *testing.T) {
	root, path, req := runtimeFixture(t)
	before, _ := os.ReadFile(path)
	preview, err := declareRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if !bytes.Equal(before, actual) || preview.Applied || !preview.Changed {
		t.Fatal("preview wrote or did not propose change")
	}
	req.Apply = true
	req.ExpectedSHA256 = preview.BeforeSHA
	result, err := declareRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ = os.ReadFile(path)
	if !result.Applied || result.AfterSHA != preview.AfterSHA || sha256Hex(actual) != preview.AfterSHA {
		t.Fatal("apply differs from preview")
	}
	var beforeMap, afterMap map[string]any
	json.Unmarshal(before, &beforeMap)
	json.Unmarshal(actual, &afterMap)
	deps := afterMap["dependencies"].(map[string]any)["scenarios"].(map[string]any)
	entry := deps["source"].(map[string]any)
	if entry["required"] != false || entry["startup_policy"] != "ignore" || entry["runtime_only"] != true || entry["enabled"] != true {
		t.Fatalf("unsafe declaration: %v", entry)
	}
	delete(deps, "source")
	a, _ := json.Marshal(beforeMap)
	b, _ := json.Marshal(afterMap)
	if !bytes.Equal(a, b) {
		t.Fatal("changed unrelated manifest semantics")
	}
	if !bytes.Contains(actual, []byte(`"preserve": [1, 2]`)) {
		t.Fatal("reformatted unrelated bytes")
	}
	source, _ := os.ReadFile(filepath.Join(root, "scenarios", "source", ".vrooli", "service.json"))
	if string(source) != runtimeManifest {
		t.Fatal("source modified")
	}
	req.ExpectedSHA256 = result.AfterSHA
	repeat, err := declareRuntime(root, req)
	if err != nil || repeat.Applied || repeat.Changed {
		t.Fatalf("repeat not idempotent: %+v %v", repeat, err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("file mode changed")
	}
}
func TestDeclareRuntimeRejectsUnsafeInputs(t *testing.T) {
	for _, name := range []string{"traversal", "self", "missing-source", "missing-rationale", "missing-description", "missing-degraded", "apply-without-preview", "stale-digest", "symlink", "duplicate", "existing", "missing-object", "malformed"} {
		t.Run(name, func(t *testing.T) {
			root, path, req := runtimeFixture(t)
			switch name {
			case "traversal":
				req.Dependency = "../source"
			case "self":
				req.Dependency = req.Scenario
			case "missing-source":
				req.Dependency = "unknown"
			case "missing-rationale":
				req.Rationale = " "
			case "missing-description":
				req.Description = ""
			case "missing-degraded":
				req.DegradedBehavior = ""
			case "apply-without-preview":
				req.Apply = true
			case "stale-digest":
				req.Apply = true
				req.ExpectedSHA256 = sha256Hex([]byte("old"))
			case "symlink":
				os.Rename(path, path+".real")
				os.Symlink(path+".real", path)
			case "duplicate":
				os.WriteFile(path, []byte(`{"dependencies":{"scenarios":{},"scenarios":{}}}`), 0600)
			case "existing":
				os.WriteFile(path, []byte(`{"dependencies":{"scenarios":{"source":{"required":true}}}}`), 0600)
			case "missing-object":
				os.WriteFile(path, []byte(`{"dependencies":{}}`), 0600)
			case "malformed":
				os.WriteFile(path, []byte(`{"dependencies":{"scenarios":{}}} bad`), 0600)
			}
			before, _ := os.ReadFile(path)
			if _, err := declareRuntime(root, req); err == nil {
				t.Fatal("accepted unsafe input")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("failure modified manifest")
			}
		})
	}
}
func TestDeclareRuntimeDoesNotRequireAPI(t *testing.T) {
	group := RegisterInstall(nil)
	for _, command := range group.Subcommands {
		if command.Name == "declare-runtime" {
			if command.NeedsAPIOverride == nil || *command.NeedsAPIOverride {
				t.Fatal("runtime declaration can launch or require API")
			}
			return
		}
	}
	t.Fatal("command missing")
}
