package depsapproved

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestDeclareRuntimeCreatesMissingScenariosObject(t *testing.T) {
	root, path, req := runtimeFixture(t)
	manifest := `{
  "dependencies": {
    "resources": {"keep": {"enabled": true}}
  },
  "other": {"preserve": true}
}
`
	if err := os.WriteFile(path, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	preview, err := declareRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, _ := os.ReadFile(path)
	if !preview.Changed || preview.Applied || string(unchanged) != manifest {
		t.Fatal("preview modified manifest or omitted the proposal")
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	result, err := declareRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(path)
	if !result.Applied || sha256Hex(updated) != preview.AfterSHA || !json.Valid(updated) {
		t.Fatalf("apply did not match preview: %+v", result)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(updated, &value); err != nil {
		t.Fatal(err)
	}
	var dependencies map[string]json.RawMessage
	if err := json.Unmarshal(value["dependencies"], &dependencies); err != nil {
		t.Fatal(err)
	}
	if _, ok := dependencies["resources"]; !ok {
		t.Fatal("existing resource dependencies were lost")
	}
	var scenarios map[string]json.RawMessage
	if err := json.Unmarshal(dependencies["scenarios"], &scenarios); err != nil {
		t.Fatal(err)
	}
	if _, ok := scenarios[req.Dependency]; !ok {
		t.Fatal("new runtime scenario dependency was not inserted")
	}
	if !bytes.Contains(updated, []byte(`"other": {"preserve": true}`)) {
		t.Fatal("unrelated manifest bytes changed")
	}
}

func TestDeclareRuntimeRejectsUnsafeInputs(t *testing.T) {
	for _, name := range []string{"traversal", "self", "missing-source", "missing-rationale", "missing-description", "missing-degraded", "apply-without-preview", "stale-digest", "symlink", "duplicate", "existing", "missing-dependencies", "malformed"} {
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
			case "missing-dependencies":
				os.WriteFile(path, []byte(`{"metadata":{"keep":true}}`), 0600)
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

const existingRuntimeManifest = `{
  "metadata": {"keep": 7},
  "dependencies": {"scenarios": {"source": {"type":"scenario", "enabled":true, "required":false, "startup_policy":"ignore", "runtime_only":true, "description":"old", "runtime_only_rationale":"old rationale", "degraded_behavior":"old behavior", "future_field":{"x":1}}}},
  "tail": [1, 2]
}
`

func updateRuntimeFixture(t *testing.T) (string, string, runtimeDeclaration) {
	t.Helper()
	root, path, req := runtimeFixture(t)
	if err := os.WriteFile(path, []byte(existingRuntimeManifest), 0600); err != nil {
		t.Fatal(err)
	}
	req.Description = "new description"
	req.Rationale = "new rationale"
	req.DegradedBehavior = "new behavior"
	return root, path, req
}

func TestUpdateRuntimePreviewApplyPreservesUnknownAndMode(t *testing.T) {
	root, path, req := updateRuntimeFixture(t)
	before, _ := os.ReadFile(path)
	preview, err := updateRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	afterPreview, _ := os.ReadFile(path)
	if !bytes.Equal(before, afterPreview) || !preview.Changed || preview.Applied {
		t.Fatal("preview wrote or did not propose update")
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	result, err := updateRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !result.Applied || sha256Hex(after) != preview.AfterSHA || result.AfterSHA != preview.AfterSHA {
		t.Fatal("apply differs from preview")
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(after, &manifest); err != nil {
		t.Fatal(err)
	}
	var all map[string]any
	_ = json.Unmarshal(after, &all)
	entry := all["dependencies"].(map[string]any)["scenarios"].(map[string]any)["source"].(map[string]any)
	if entry["description"] != req.Description || entry["runtime_only_rationale"] != req.Rationale || entry["degraded_behavior"] != req.DegradedBehavior || entry["future_field"].(map[string]any)["x"] != float64(1) {
		t.Fatalf("fields not preserved/updated: %v", entry)
	}
	if !bytes.Contains(after, []byte(`"tail": [1, 2]`)) {
		t.Fatal("unrelated manifest bytes changed")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("mode changed")
	}
}

func TestUpdateRuntimeSemanticNoOpPreservesCompactReorderedManifest(t *testing.T) {
	root, path, req := updateRuntimeFixture(t)
	content := `{"dependencies":{"scenarios":{"source":{"degraded_behavior":"new behavior","runtime_only":true,"startup_policy":"ignore","required":false,"runtime_only_rationale":"new rationale","enabled":true,"type":"scenario","description":"new description","future":1}}},"other":{"keep":true}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := updateRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	afterPreview, _ := os.ReadFile(path)
	if !bytes.Equal(before, afterPreview) || preview.Changed || preview.Applied || preview.BeforeSHA != preview.AfterSHA {
		t.Fatalf("semantic no-op preview changed bytes or receipt: %+v", preview)
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	result, err := updateRuntime(root, req)
	if err != nil {
		t.Fatal(err)
	}
	afterApply, _ := os.ReadFile(path)
	if !bytes.Equal(before, afterApply) || result.Changed || result.Applied || result.BeforeSHA != result.AfterSHA {
		t.Fatalf("semantic no-op apply changed bytes or receipt: %+v", result)
	}
}

func TestUpdateRuntimeRejectsUnsafeAndStaleInputs(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"missing", `{"dependencies":{"scenarios":{}}}`},
		{"structural", `{"dependencies":{"scenarios":{"source":{"type":"scenario","enabled":true,"required":true,"runtime_only":true,"startup_policy":"ignore"}}}}`},
		{"duplicate", `{"dependencies":{"scenarios":{"source":{"type":"scenario","enabled":true,"required":false,"runtime_only":true,"startup_policy":"ignore","x":1,"x":2}}}}`},
		{"malformed", `{"dependencies":{"scenarios":{}}} trailing`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, path, req := updateRuntimeFixture(t)
			_ = os.WriteFile(path, []byte(tc.content), 0600)
			before, _ := os.ReadFile(path)
			if _, err := updateRuntime(root, req); err == nil {
				t.Fatal("accepted unsafe manifest")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("failure wrote manifest")
			}
		})
	}
	t.Run("symlinked-consumer-path", func(t *testing.T) {
		root, path, req := updateRuntimeFixture(t)
		real := path + ".real"
		if err := os.Rename(path, real); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, path); err != nil {
			t.Fatal(err)
		}
		if _, err := updateRuntime(root, req); err == nil {
			t.Fatal("accepted symlinked manifest path")
		} else if !strings.Contains(err.Error(), "symlinked scenario path") {
			t.Fatalf("unexpected file-safety error: %v", err)
		}
	})
	root, path, req := updateRuntimeFixture(t)
	req.Apply = true
	req.ExpectedSHA256 = sha256Hex([]byte("stale"))
	before, _ := os.ReadFile(path)
	if _, err := updateRuntime(root, req); err == nil {
		t.Fatal("accepted stale digest")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("stale apply wrote manifest")
	}
}

func TestUpdateRuntimeRequiresExplicitStructuralJSONTypes(t *testing.T) {
	for _, tc := range []struct{ name, contract string }{
		{"required-null", `"type":"scenario","enabled":true,"required":null,"runtime_only":true,"startup_policy":"ignore"`},
		{"enabled-null", `"type":"scenario","enabled":null,"required":false,"runtime_only":true,"startup_policy":"ignore"`},
		{"type-null", `"type":null,"enabled":true,"required":false,"runtime_only":true,"startup_policy":"ignore"`},
		{"runtime-only-wrong-type", `"type":"scenario","enabled":true,"required":false,"runtime_only":"true","startup_policy":"ignore"`},
		{"startup-policy-null", `"type":"scenario","enabled":true,"required":false,"runtime_only":true,"startup_policy":null`},
		{"missing-required", `"type":"scenario","enabled":true,"runtime_only":true,"startup_policy":"ignore"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, path, req := updateRuntimeFixture(t)
			content := `{"dependencies":{"scenarios":{"source":{` + tc.contract + `,"description":"original","runtime_only_rationale":"original","degraded_behavior":"original"}}}}`
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := updateRuntime(root, req); err == nil {
				t.Fatal("preview accepted structural field with absent/wrong JSON type")
			}
			afterPreview, _ := os.ReadFile(path)
			if !bytes.Equal(before, afterPreview) {
				t.Fatal("rejected preview wrote manifest")
			}
			req.Apply, req.ExpectedSHA256 = true, sha256Hex(before)
			if _, err := updateRuntime(root, req); err == nil {
				t.Fatal("apply accepted structural field with absent/wrong JSON type")
			}
			afterApply, _ := os.ReadFile(path)
			if !bytes.Equal(before, afterApply) {
				t.Fatal("rejected apply wrote manifest")
			}
		})
	}
}

func TestAtomicFileWriteRejectsChangedServiceManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "service.json")
	original := []byte(`{"state":"previewed"}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := []byte(`{"state":"changed-after-preview"}`)
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicallyIfUnchanged(dir, path, []byte(`{"state":"apply"}`), info, sha256Hex(original)); err == nil || !strings.Contains(err.Error(), "target changed before apply") {
		t.Fatalf("expected generic stale-target error, got %v", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, changed) {
		t.Fatal("stale atomic write modified target")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "service.json" {
		t.Fatalf("temporary file was not cleaned up: %v", entries)
	}
}

func TestReadRegularFileNoSymlinkUsesGenericSafetyError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "service.json")
	link := filepath.Join(dir, "service-link.json")
	if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRegularFileNoSymlink(link); err == nil || !strings.Contains(err.Error(), "target must be a regular non-symlink file") {
		t.Fatalf("expected generic symlink safety error, got %v", err)
	}
}
