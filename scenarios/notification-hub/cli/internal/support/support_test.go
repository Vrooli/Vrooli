package support

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func TestExtractLegacyGlobals(t *testing.T) {
	remaining, profileID, apiKey, err := ExtractLegacyGlobals([]string{
		"--profile-id", "prof-1",
		"--api-key=secret",
		"--api-url", "http://localhost:8080",
		"notifications", "send",
	})
	if err != nil {
		t.Fatalf("ExtractLegacyGlobals: %v", err)
	}
	if profileID != "prof-1" {
		t.Fatalf("profileID = %q", profileID)
	}
	if apiKey != "secret" {
		t.Fatalf("apiKey = %q", apiKey)
	}
	if len(remaining) != 4 || remaining[0] != "--api-base" || remaining[1] != "http://localhost:8080" {
		t.Fatalf("remaining = %#v", remaining)
	}
}

func TestDefaultsStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	core := &cliapp.ScenarioApp{
		ConfigFile: mustConfigFile(t, filepath.Join(dir, "config.json")),
	}

	store, err := NewDefaultsStore(core)
	if err != nil {
		t.Fatalf("NewDefaultsStore: %v", err)
	}
	if err := store.Save(DefaultsConfig{ProfileID: "prof-123"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ProfileID != "prof-123" {
		t.Fatalf("ProfileID = %q", got.ProfileID)
	}
}

func TestDependenciesResolveProfileID(t *testing.T) {
	t.Setenv("NOTIFICATION_HUB_PROFILE_ID", "")
	store := &DefaultsStore{file: mustConfigFile(t, filepath.Join(t.TempDir(), "defaults.json"))}
	if err := store.Save(DefaultsConfig{ProfileID: "default-prof"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	deps := Dependencies{
		Defaults: func() *DefaultsStore { return store },
	}

	got, err := deps.ResolveProfileID("")
	if err != nil {
		t.Fatalf("ResolveProfileID: %v", err)
	}
	if got != "default-prof" {
		t.Fatalf("ResolveProfileID = %q", got)
	}
}

func mustConfigFile(t *testing.T, path string) *cliutil.ConfigFile {
	t.Helper()
	cfg, err := cliutil.NewConfigFile(path)
	if err != nil {
		t.Fatalf("NewConfigFile: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	return cfg
}
