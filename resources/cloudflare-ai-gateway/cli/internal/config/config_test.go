package config

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	resourceenv "resource-cloudflare-ai-gateway/cli/internal/env"
)

func TestStoreEnsureInitializedCreatesDefaults(t *testing.T) {
	t.Parallel()

	store := NewStore(testRuntime(t))
	if err := store.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}

	cfg, err := store.LoadGatewayConfig()
	if err != nil {
		t.Fatalf("LoadGatewayConfig() error = %v", err)
	}
	if cfg.Active {
		t.Fatalf("LoadGatewayConfig().Active = true, want false")
	}

	state, err := store.LoadGatewayState()
	if err != nil {
		t.Fatalf("LoadGatewayState() error = %v", err)
	}
	if state.Status != "inactive" {
		t.Fatalf("LoadGatewayState().Status = %q, want inactive", state.Status)
	}
}

func TestStoreNamedConfigLifecycle(t *testing.T) {
	t.Parallel()

	store := NewStore(testRuntime(t))
	if err := store.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}

	payload := json.RawMessage(`{"provider":"openrouter"}`)
	if err := store.SaveNamedConfig("primary", payload); err != nil {
		t.Fatalf("SaveNamedConfig() error = %v", err)
	}

	names, err := store.ListNamedConfigs()
	if err != nil {
		t.Fatalf("ListNamedConfigs() error = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"primary"}) {
		t.Fatalf("ListNamedConfigs() = %v, want [primary]", names)
	}

	got, err := store.LoadNamedConfig("primary")
	if err != nil {
		t.Fatalf("LoadNamedConfig() error = %v", err)
	}
	if string(got) != "{\"provider\":\"openrouter\"}\n" {
		t.Fatalf("LoadNamedConfig() = %s", got)
	}

	if err := store.DeleteNamedConfig("primary"); err != nil {
		t.Fatalf("DeleteNamedConfig() error = %v", err)
	}
	if _, err := store.LoadNamedConfig("primary"); !errors.Is(err, ErrNamedConfigNotFound) {
		t.Fatalf("LoadNamedConfig() after delete error = %v, want %v", err, ErrNamedConfigNotFound)
	}
}

func TestMarkStatus(t *testing.T) {
	t.Parallel()

	store := NewStore(testRuntime(t))
	if err := store.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}

	now := time.Unix(1700000000, 0).UTC()
	if err := store.MarkStatus("reachable", now); err != nil {
		t.Fatalf("MarkStatus() error = %v", err)
	}

	state, err := store.LoadGatewayState()
	if err != nil {
		t.Fatalf("LoadGatewayState() error = %v", err)
	}
	if state.Status != "reachable" {
		t.Fatalf("LoadGatewayState().Status = %q, want reachable", state.Status)
	}
	if state.LastCheck == nil || !state.LastCheck.Equal(now) {
		t.Fatalf("LoadGatewayState().LastCheck = %v, want %v", state.LastCheck, now)
	}
}

func TestGatewayAPIBaseURL(t *testing.T) {
	t.Parallel()

	got := GatewayAPIBaseURL("https://api.cloudflare.com/client/v4/accounts", "acct-123")
	want := "https://api.cloudflare.com/client/v4/accounts/acct-123/ai-gateway"
	if got != want {
		t.Fatalf("GatewayAPIBaseURL() = %q, want %q", got, want)
	}
}

func testRuntime(t *testing.T) resourceenv.Runtime {
	t.Helper()

	root := t.TempDir()
	return resourceenv.Runtime{
		DataRoot:   root,
		ConfigsDir: filepath.Join(root, "configs"),
		LogsDir:    filepath.Join(root, "logs"),
		ConfigFile: filepath.Join(root, "config.json"),
		StateFile:  filepath.Join(root, "state.json"),
		APIBaseURL: "https://api.cloudflare.com/client/v4/accounts",
	}
}
