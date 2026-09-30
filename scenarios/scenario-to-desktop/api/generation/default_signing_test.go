package generation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop-api/signing"
	signingtypes "scenario-to-desktop-api/signing/types"
)

type memorySigningStore struct {
	configs map[string]*signing.SigningConfig
	saved   []string
}

func newMemorySigningStore() *memorySigningStore {
	return &memorySigningStore{configs: map[string]*signing.SigningConfig{}}
}

func (m *memorySigningStore) Get(_ context.Context, scenario string) (*signing.SigningConfig, error) {
	return m.configs[scenario], nil
}

func (m *memorySigningStore) Save(_ context.Context, scenario string, config *signing.SigningConfig) error {
	m.configs[scenario] = config
	m.saved = append(m.saved, scenario)
	return nil
}

func sharedManagedConfig() *signing.SigningConfig {
	return &signing.SigningConfig{
		SchemaVersion: signingtypes.SchemaVersion,
		Enabled:       true,
		Linux: &signing.LinuxSigningConfig{
			GPGKeyID:         "FINGERPRINT123",
			GPGPassphraseEnv: signingtypes.DefaultPassphraseEnvVar,
			ManagedKey:       &signing.ManagedSigningKey{LogicalID: signing.DefaultSharedLogicalID},
		},
	}
}

func scenarioRootWithDemo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPrepareSigningConfigAppliesSharedKeyByDefault(t *testing.T) {
	builds := newMockBuildStore()
	builds.Create("b1")
	store := newMemorySigningStore()
	service := NewService(
		WithVrooliRoot(scenarioRootWithDemo(t)),
		WithBuildStore(builds),
		WithSigningStore(store),
		WithDefaultSigningResolver(func(context.Context, string) (*signing.SigningConfig, error) {
			return sharedManagedConfig(), nil
		}),
	)

	config := &DesktopConfig{ScenarioName: "demo", AppName: "demo", OutputPath: t.TempDir()}
	if err := service.prepareSigningConfig("b1", config); err != nil {
		t.Fatalf("prepareSigningConfig() error = %v", err)
	}
	if config.CodeSigning == nil || !config.CodeSigning.Enabled || config.CodeSigning.Linux == nil {
		t.Fatalf("shared signing key was not applied: %#v", config.CodeSigning)
	}
	if len(store.saved) != 1 || store.saved[0] != "demo" {
		t.Fatalf("signing config was not persisted: %#v", store.saved)
	}
	if _, err := os.Stat(filepath.Join(config.OutputPath, "scripts", "sign-linux-artifacts.js")); err != nil {
		t.Fatalf("Linux signer hook missing: %v", err)
	}
	status, _ := builds.Get("b1")
	if !strings.Contains(strings.Join(status.BuildLog, "\n"), "Applied shared desktop signing key") {
		t.Fatalf("build log did not record the applied key: %#v", status.BuildLog)
	}
}

func TestPrepareSigningConfigRespectsExplicitDisabledScenarioConfig(t *testing.T) {
	builds := newMockBuildStore()
	builds.Create("b1")
	store := newMemorySigningStore()
	store.configs["demo"] = &signing.SigningConfig{Enabled: false}
	service := NewService(
		WithVrooliRoot(scenarioRootWithDemo(t)),
		WithBuildStore(builds),
		WithSigningStore(store),
		WithDefaultSigningResolver(func(context.Context, string) (*signing.SigningConfig, error) {
			return sharedManagedConfig(), nil
		}),
	)

	config := &DesktopConfig{ScenarioName: "demo", AppName: "demo", OutputPath: t.TempDir()}
	if err := service.prepareSigningConfig("b1", config); err != nil {
		t.Fatalf("prepareSigningConfig() error = %v", err)
	}
	if config.CodeSigning != nil {
		t.Fatalf("an explicit opt-out must not be overridden: %#v", config.CodeSigning)
	}
	if len(store.saved) != 0 {
		t.Fatalf("opt-out must not persist a new config: %#v", store.saved)
	}
}

func TestPrepareSigningConfigLeavesUnsignedWhenNoSharedKey(t *testing.T) {
	builds := newMockBuildStore()
	builds.Create("b1")
	store := newMemorySigningStore()
	service := NewService(
		WithVrooliRoot(scenarioRootWithDemo(t)),
		WithBuildStore(builds),
		WithSigningStore(store),
		WithDefaultSigningResolver(func(context.Context, string) (*signing.SigningConfig, error) {
			return nil, nil
		}),
	)

	config := &DesktopConfig{ScenarioName: "demo", AppName: "demo", OutputPath: t.TempDir()}
	if err := service.prepareSigningConfig("b1", config); err != nil {
		t.Fatalf("prepareSigningConfig() error = %v", err)
	}
	if config.CodeSigning != nil || len(store.saved) != 0 {
		t.Fatalf("absence of the shared key must leave the build unsigned: %#v", config.CodeSigning)
	}
	status, _ := builds.Get("b1")
	if !strings.Contains(strings.Join(status.BuildLog, "\n"), "unsigned") {
		t.Fatalf("build log did not report the unsigned build: %#v", status.BuildLog)
	}
}

func TestPrepareSigningConfigSkipsDefaultForUnknownScenario(t *testing.T) {
	builds := newMockBuildStore()
	builds.Create("b1")
	store := newMemorySigningStore()
	service := NewService(
		WithVrooliRoot(t.TempDir()),
		WithBuildStore(builds),
		WithSigningStore(store),
		WithDefaultSigningResolver(func(context.Context, string) (*signing.SigningConfig, error) {
			return sharedManagedConfig(), nil
		}),
	)

	config := &DesktopConfig{ScenarioName: "ghost", AppName: "ghost", OutputPath: t.TempDir()}
	if err := service.prepareSigningConfig("b1", config); err != nil {
		t.Fatalf("prepareSigningConfig() error = %v", err)
	}
	if config.CodeSigning != nil || len(store.saved) != 0 {
		t.Fatalf("a name without a scenario directory must not be signed: %#v", config.CodeSigning)
	}
}
