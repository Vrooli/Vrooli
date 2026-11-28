package models

import (
	"os"
	"path/filepath"
	"testing"
)

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model-policy.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	return path
}

const validPolicy = `{
  "schema_version": "2026-10-21",
  "roles": {
    "pii.small": {"model": "small", "fallbacks": [], "required_capabilities": ["detect_pii"], "preference": 10},
    "pii.default": {"model": "mid", "fallbacks": ["small"], "required_capabilities": ["detect_pii"], "preference": 20}
  },
  "models": {
    "small": {"family": "bert", "capabilities": ["detect_pii"], "disk_bytes": 30000000, "resident_bytes": 134217728, "default_eligible": true, "measurement": {"suite_id": "pii.detection"}},
    "mid": {"family": "gliner", "capabilities": ["detect_pii", "zero_shot"], "disk_bytes": 190000000, "resident_bytes": 402653184, "measurement": {"suite_id": "pii.detection"}}
  },
  "constraints": {"preferred_bytes": 134217728, "floor_bytes": 67108864, "role": "pii.small", "measurement_required": false}
}`

func TestLoadValidPolicy(t *testing.T) {
	p, err := Load(writePolicy(t, validPolicy))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(p.Roles) != 2 || len(p.Models) != 2 {
		t.Fatalf("unexpected policy shape: %d roles, %d models", len(p.Roles), len(p.Models))
	}
	if !p.Allowed("pii.default", "small") {
		t.Error("fallback 'small' should be allowed for pii.default")
	}
	if p.Allowed("pii.default", "mid") != true {
		t.Error("primary 'mid' should be allowed for pii.default")
	}
	if p.Allowed("pii.small", "mid") {
		t.Error("'mid' is not a member of pii.small")
	}
	if m, ok := p.DefaultModelForRole("pii.default"); !ok || m != "mid" {
		t.Errorf("DefaultModelForRole(pii.default) = %q,%v", m, ok)
	}
	if m, ok := p.DefaultModelForRole("nope"); !ok || m != "small" {
		t.Errorf("unknown role should fall to default: %q,%v", m, ok)
	}
}

func TestLoadRejectsMissingDefaultRole(t *testing.T) {
	body := `{"schema_version":"1","roles":{"other":{"model":"small"}},"models":{"small":{"capabilities":["detect_pii"],"measurement":{"suite_id":"pii.detection"}}},"constraints":{"preferred_bytes":2,"floor_bytes":1}}`
	if _, err := Load(writePolicy(t, body)); err == nil {
		t.Error("expected error when default role pii.small is missing")
	}
}

func TestLoadRejectsModelLackingDetectCapability(t *testing.T) {
	body := `{"schema_version":"1","roles":{"pii.small":{"model":"small"}},"models":{"small":{"capabilities":["other"],"measurement":{"suite_id":"pii.detection"}}},"constraints":{"preferred_bytes":2,"floor_bytes":1}}`
	if _, err := Load(writePolicy(t, body)); err == nil {
		t.Error("expected error when a model lacks detect_pii capability")
	}
}

func TestLoadRejectsFloorAbovePreferred(t *testing.T) {
	body := `{"schema_version":"1","roles":{"pii.small":{"model":"small"}},"models":{"small":{"capabilities":["detect_pii"],"measurement":{"suite_id":"pii.detection"}}},"constraints":{"preferred_bytes":1,"floor_bytes":2}}`
	if _, err := Load(writePolicy(t, body)); err == nil {
		t.Error("expected error when floor_bytes >= preferred_bytes")
	}
}

func TestLoadRejectsDanglingFallback(t *testing.T) {
	body := `{"schema_version":"1","roles":{"pii.small":{"model":"small","fallbacks":["ghost"]}},"models":{"small":{"capabilities":["detect_pii"],"measurement":{"suite_id":"pii.detection"}}},"constraints":{"preferred_bytes":2,"floor_bytes":1}}`
	if _, err := Load(writePolicy(t, body)); err == nil {
		t.Error("expected error when a fallback references a missing model")
	}
}

func TestMeasurementRequiredGate(t *testing.T) {
	// With measurement_required true and no completed measurement, Load must fail.
	body := `{"schema_version":"1","roles":{"pii.small":{"model":"small"}},"models":{"small":{"capabilities":["detect_pii"],"measurement":{"suite_id":"pii.detection","sample_count":0}}},"constraints":{"preferred_bytes":2,"floor_bytes":1,"measurement_required":true}}`
	if _, err := Load(writePolicy(t, body)); err == nil {
		t.Error("expected error when measurement_required but no sample_count/evidence")
	}
}

// TestLoadRealPolicy loads the committed model-policy.json so the shipped file
// stays valid against the loader.
func TestLoadRealPolicy(t *testing.T) {
	path := "../../../model-policy.json"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("model-policy.json not found at %s", path)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatalf("committed model-policy.json failed to load: %v", err)
	}
	for _, role := range []string{"pii.small", "pii.default", "pii.large"} {
		if _, ok := p.Roles[role]; !ok {
			t.Errorf("committed policy missing role %q", role)
		}
	}
}
