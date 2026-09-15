package heartbeat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/internal/teamconfig"
)

func TestDefaultProfileKeyForRuntimeMode(t *testing.T) {
	keys, err := LoadDeclaredProfileKeys()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		runtimeMode string
		want        string
	}{
		{teamconfig.RuntimeModeSingleProcess, keys.SingleProcess},
		{teamconfig.RuntimeModeMultiProcess, keys.MultiProcess},
		{"", keys.MultiProcess},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("runtimeMode=%q", tt.runtimeMode), func(t *testing.T) {
			got, err := DefaultProfileKeyForRuntimeMode(tt.runtimeMode)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("DefaultProfileKeyForRuntimeMode(%q) = %q, want %q", tt.runtimeMode, got, tt.want)
			}
		})
	}
}

func TestLoadDeclaredProfileKeysRejectsMissingAndEmptyDeclarations(t *testing.T) {
	repoRoot := t.TempDir()
	dir := filepath.Join(repoRoot, "scenarios", "prompt-manager", ".vrooli", "agent-manager")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDeclaredProfileKeys(repoRoot); err == nil || !strings.Contains(err.Error(), "heartbeat.json") {
		t.Fatalf("missing declaration error = %v", err)
	}
	payload, err := json.Marshal(declaredProfile{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "heartbeat.json"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDeclaredProfileKeys(repoRoot); err == nil || !strings.Contains(err.Error(), "profileKey is empty") {
		t.Fatalf("empty declaration error = %v", err)
	}
}

func declaredKeysForTest(t *testing.T) DeclaredProfileKeys {
	t.Helper()
	keys, err := LoadDeclaredProfileKeys()
	if err != nil {
		t.Fatal(err)
	}
	return keys
}
