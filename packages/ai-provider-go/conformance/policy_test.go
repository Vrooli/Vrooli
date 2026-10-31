package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestCheckPolicyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model-policy.json")
	body := `{
      "schema_version": "1",
      "roles": {"chat.default": {"model": "m", "endpoint": "chat", "required_capabilities": ["chat"], "preference": 1, "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}},
      "models": {"m": {"provider": "p", "family": "f", "capabilities": ["chat"], "modalities": {"input": ["text"], "output": ["text"]}, "endpoints": ["chat"], "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}}},
      "constraints": {"endpoints": ["chat"], "capability_vocabulary": ["chat"], "role_preference_order": ["chat.default"], "provenance_source_kinds": ["manual_policy"], "modality_vocabulary": ["text"]},
      "provenance": {"policy": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}
    }`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckPolicyFile(path, policy.CloudOptions()); err != nil {
		t.Fatalf("CheckPolicyFile: %v", err)
	}
}
