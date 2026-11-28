package depsapproved

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resourceUpdateFixture(t *testing.T, manifest string) (string, string, resourceUpdate) {
	t.Helper()
	root := t.TempDir()
	consumer := filepath.Join(root, "scenarios", "consumer", ".vrooli")
	resource := filepath.Join(root, "resources", "kyutai-stt")
	if err := os.MkdirAll(consumer, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resource, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(consumer, "service.json")
	if err := os.WriteFile(path, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resource, "resource.json"), []byte(`{"name":"kyutai-stt"}`), 0600); err != nil {
		t.Fatal(err)
	}
	req := resourceUpdate{Scenario: "consumer", Resource: "kyutai-stt", Description: "Optional streaming STT accelerator", DegradedBehavior: "Use Sherpa streaming recognition when stopped", StartupPolicy: "ignore"}
	return root, path, req
}

const resourceUpdateManifest = `{
  "before": {"keep": true},
  "dependencies": {"resources": {
    "kyutai-stt": {"required": false, "description": "Old purpose", "startup_policy": "try_start", "future": {"keep": 7}},
    "other": {"required": true}
  }},
  "after": [1, 2]
}
`

func TestUpdateResourcePreviewApplyDigestAndIsolation(t *testing.T) {
	root, path, req := resourceUpdateFixture(t, resourceUpdateManifest)
	before, _ := os.ReadFile(path)
	preview, err := updateResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if !bytes.Equal(before, actual) || !preview.Changed || preview.Applied {
		t.Fatal("preview wrote or did not propose change")
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	result, err := updateResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ = os.ReadFile(path)
	if !result.Applied || result.AfterSHA != preview.AfterSHA || sha256Hex(actual) != preview.AfterSHA {
		t.Fatal("apply differs from preview")
	}
	for _, keep := range []string{`"future": {`, `"other": {"required": true}`, `"before": {"keep": true}`, `"after": [1, 2]`} {
		if !bytes.Contains(actual, []byte(keep)) {
			t.Fatalf("lost unrelated field %s", keep)
		}
	}
	if !bytes.Contains(actual, []byte(`"startup_policy": "ignore"`)) {
		t.Fatal("startup policy was not updated")
	}
	resource, _ := os.ReadFile(filepath.Join(root, "resources", "kyutai-stt", "resource.json"))
	if string(resource) != `{"name":"kyutai-stt"}` {
		t.Fatal("resource manifest changed")
	}
	// Once the policy is current, the operation is a semantic no-op and retains bytes.
	req.Apply, req.ExpectedSHA256 = false, ""
	noOp, err := updateResource(root, req)
	if err != nil || noOp.Changed || noOp.Applied {
		t.Fatalf("identical update not a no-op: %+v %v", noOp, err)
	}
}

func TestUpdateResourceRefusesUnsafeTargetsAndContracts(t *testing.T) {
	tests := []struct {
		name, manifest string
		symlink        bool
	}{
		{"missing-required", `{"dependencies":{"resources":{"kyutai-stt":{"description":"x","startup_policy":"try_start"}}}}`, false},
		{"required", `{"dependencies":{"resources":{"kyutai-stt":{"required":true,"startup_policy":"try_start"}}}}`, false},
		{"bad-policy", `{"dependencies":{"resources":{"kyutai-stt":{"required":false,"startup_policy":"must_start"}}}}`, false},
		{"null-enabled", `{"dependencies":{"resources":{"kyutai-stt":{"enabled":null,"required":false,"startup_policy":"try_start"}}}}`, false},
		{"duplicate", `{"dependencies":{"resources":{"kyutai-stt":{"required":false,"required":false,"startup_policy":"try_start"}}}}`, false},
		{"symlink-resource", resourceUpdateManifest, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _, req := resourceUpdateFixture(t, tt.manifest)
			if tt.symlink {
				resource := filepath.Join(root, "resources", "kyutai-stt")
				if err := os.RemoveAll(resource); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), resource); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := updateResource(root, req); err == nil {
				t.Fatal("unsafe update accepted")
			}
		})
	}
}

func TestUpdateResourceRejectsStaleDigest(t *testing.T) {
	root, path, req := resourceUpdateFixture(t, resourceUpdateManifest)
	preview, err := updateResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(resourceUpdateManifest, `"Old purpose"`, `"concurrent edit"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	if _, err := updateResource(root, req); err == nil {
		t.Fatal("stale digest accepted")
	}
}

func TestUpdateResourceCanChangeOnlyStartupPolicy(t *testing.T) {
	root, path, req := resourceUpdateFixture(t, `{"dependencies":{"resources":{"kyutai-stt":{"required":false,"description":"Keep this text","startup_policy":"try_start"}}}}`)
	req.Description = ""
	req.DegradedBehavior = ""
	preview, err := updateResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Changed {
		t.Fatal("startup policy change was not proposed")
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	if _, err := updateResource(root, req); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(`"description": "Keep this text"`)) || !bytes.Contains(data, []byte(`"startup_policy": "ignore"`)) {
		t.Fatalf("unexpected row after policy-only update: %s", data)
	}
}
