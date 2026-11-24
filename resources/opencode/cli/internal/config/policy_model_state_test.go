package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyModelStateRetainsAndSortsOwnership(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "opencode.json")
	if err := writePolicyModelState(context.Background(), configPath, []string{"model:b", "model:a", "model:a"}); err != nil {
		t.Fatal(err)
	}
	got, err := readPolicyModelState(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "model:a" || got[1] != "model:b" {
		t.Fatalf("policy model ownership state = %v, want sorted unique IDs", got)
	}
	if err := writePolicyModelState(context.Background(), configPath, mergeModelIDs(got, []string{"model:retired"})); err != nil {
		t.Fatal(err)
	}
	got, err = readPolicyModelState(configPath)
	if err != nil || len(got) != 3 {
		t.Fatalf("retained policy model state = %v err=%v", got, err)
	}
}

func TestPolicyModelStateRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencode.json")
	target := filepath.Join(dir, "state.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"models":["operator/model"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	statePath := configPath + ".ollama-policy-models.json"
	if err := os.Symlink(target, statePath); err != nil {
		t.Skipf("symlink fixture is unavailable: %v", err)
	}
	if _, err := readPolicyModelState(configPath); err == nil {
		t.Fatal("readPolicyModelState followed a symlink")
	}
	if err := writePolicyModelState(context.Background(), configPath, []string{"model:a"}); err == nil {
		t.Fatal("writePolicyModelState followed a symlink")
	}
}
