package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadBehavior_MissingDirReturnsDefaults(t *testing.T) {
	got, err := LoadBehavior("")
	if err != nil {
		t.Fatalf("LoadBehavior(empty): %v", err)
	}
	if !reflect.DeepEqual(got, DefaultBehavior()) {
		t.Fatalf("empty scenarioDir should return defaults; got %+v", got)
	}
}

func TestLoadBehavior_MissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior(no file): %v", err)
	}
	if !reflect.DeepEqual(got, DefaultBehavior()) {
		t.Fatalf("missing config.json should return defaults; got %+v", got)
	}
}

func TestLoadBehavior_FileWithoutBehaviorKeyReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"environment": {"FOO": {"description": "x"}}}`)
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior: %v", err)
	}
	if !reflect.DeepEqual(got, DefaultBehavior()) {
		t.Fatalf("missing 'behavior' key should return defaults; got %+v", got)
	}
}

func TestLoadBehavior_LoadsAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		"behavior": {
			"protected": {
				"gitAllowlist": ["status", "diff", "log"]
			}
		}
	}`)
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior: %v", err)
	}
	want := []string{"status", "diff", "log"}
	if !reflect.DeepEqual(got.Protected.GitAllowlist, want) {
		t.Fatalf("allowlist = %v, want %v", got.Protected.GitAllowlist, want)
	}
}

func TestLoadBehavior_LoadsTemplates(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		"behavior": {
			"protected": {
				"gitDenyMessageTemplate": "blocked: {verb}",
				"gitNoVerbMessageTemplate": "no verb"
			}
		}
	}`)
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior: %v", err)
	}
	if got.Protected.GitDenyMessageTemplate != "blocked: {verb}" {
		t.Errorf("blocked template = %q", got.Protected.GitDenyMessageTemplate)
	}
	if got.Protected.GitNoVerbMessageTemplate != "no verb" {
		t.Errorf("no-verb template = %q", got.Protected.GitNoVerbMessageTemplate)
	}
}

func TestLoadBehavior_PartialOverrideLeavesOthersAtDefault(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		"behavior": {
			"protected": {
				"gitDenyMessageTemplate": "blocked"
			}
		}
	}`)
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior: %v", err)
	}
	if got.Protected.GitDenyMessageTemplate != "blocked" {
		t.Errorf("template did not override; got %q", got.Protected.GitDenyMessageTemplate)
	}
	// Allowlist not provided in JSON should remain default (nil/empty).
	if len(got.Protected.GitAllowlist) != 0 {
		t.Errorf("allowlist should still be empty; got %v", got.Protected.GitAllowlist)
	}
	if got.Protected.GitNoVerbMessageTemplate != "" {
		t.Errorf("no-verb template should still be empty; got %q", got.Protected.GitNoVerbMessageTemplate)
	}
}

func TestLoadBehavior_MalformedJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{not valid json`)
	_, err := LoadBehavior(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoadBehavior_EmptyFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, ``)
	got, err := LoadBehavior(dir)
	if err != nil {
		t.Fatalf("LoadBehavior(empty file): %v", err)
	}
	if !reflect.DeepEqual(got, DefaultBehavior()) {
		t.Fatalf("empty file should return defaults; got %+v", got)
	}
}

func writeConfig(t *testing.T, scenarioDir, body string) {
	t.Helper()
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(vrooliDir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
