package depsapproved

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeclareResourcePreviewApplyAndDigestGuard(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "consumer", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "kale-tts"), 0700); err != nil {
		t.Fatal(err)
	}
	before := []byte("{\n  \"dependencies\": {\"scenarios\": {}},\n  \"keep\": true\n}\n")
	if err := os.WriteFile(manifest, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "kale-tts", "resource.json"), []byte(`{"name":"kale-tts"}`), 0600); err != nil {
		t.Fatal(err)
	}
	req := resourceDeclaration{Scenario: "consumer", Resource: "kale-tts", Description: "Optional TTS", DegradedBehavior: "Use Kokoro", StartupPolicy: "ignore"}
	preview, err := declareResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, _ := os.ReadFile(manifest)
	if !preview.Changed || preview.Applied || string(unchanged) != string(before) {
		t.Fatal("preview wrote or omitted proposal")
	}
	req.Apply, req.ExpectedSHA256 = true, preview.BeforeSHA
	result, err := declareResource(root, req)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(manifest)
	if !result.Applied || sha256Hex(after) != preview.AfterSHA {
		t.Fatal("apply did not match preview")
	}
	req.ExpectedSHA256 = result.BeforeSHA
	if _, err := declareResource(root, req); err == nil {
		t.Fatal("stale digest accepted")
	}
	req.ExpectedSHA256 = result.AfterSHA
	repeat, err := declareResource(root, req)
	if err != nil || repeat.Changed || repeat.Applied {
		t.Fatalf("identical declaration not idempotent: %+v %v", repeat, err)
	}
}
