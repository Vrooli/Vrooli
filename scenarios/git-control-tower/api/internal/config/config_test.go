package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_MissingDirReturnsDefaults(t *testing.T) {
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load(empty): %v", err)
	}
	if !reflect.DeepEqual(got, DefaultConfig()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load(no file): %v", err)
	}
	if !reflect.DeepEqual(got, DefaultConfig()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestLoad_NoPolicyKeyReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"environment": {"FOO": {"description": "x"}}}`)
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, DefaultConfig()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestLoad_PolicyOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		"policy": {
			"agentAccess": "deny",
			"agentOverrideFlag": "--really-yes",
			"callerDetection": "strict",
			"agentDenyMessageTemplate": "denied: {command}"
		}
	}`)
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Policy.AgentAccess != AgentAccessDeny {
		t.Errorf("AgentAccess = %q, want deny", got.Policy.AgentAccess)
	}
	if got.Policy.AgentOverrideFlag != "--really-yes" {
		t.Errorf("AgentOverrideFlag = %q", got.Policy.AgentOverrideFlag)
	}
	if got.Policy.CallerDetection != CallerDetectionStrict {
		t.Errorf("CallerDetection = %q", got.Policy.CallerDetection)
	}
	if got.Policy.AgentDenyMessageTemplate != "denied: {command}" {
		t.Errorf("AgentDenyMessageTemplate = %q", got.Policy.AgentDenyMessageTemplate)
	}
}

func TestLoad_PartialOverride(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		"policy": {
			"agentAccess": "warn"
		}
	}`)
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Policy.AgentAccess != AgentAccessWarn {
		t.Errorf("AgentAccess = %q, want warn", got.Policy.AgentAccess)
	}
	if got.Policy.AgentOverrideFlag != DefaultConfig().Policy.AgentOverrideFlag {
		t.Error("agentOverrideFlag should fall back to default")
	}
	if got.Policy.CallerDetection != DefaultConfig().Policy.CallerDetection {
		t.Error("callerDetection should fall back to default")
	}
}

func TestLoad_MalformedJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{ not json`)
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoad_InvalidAgentAccessRejected(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"policy": {"agentAccess": "yolo"}}`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected validation error for unknown agentAccess")
	}
}

func TestLoad_InvalidDetectionRejected(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"policy": {"callerDetection": "x"}}`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected validation error for unknown callerDetection")
	}
}

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	vd := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vd, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
