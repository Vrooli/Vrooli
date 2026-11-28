package resources

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestOpenRGBFirstLaunchSeedsOnlyResourceOwnedConfig(t *testing.T) {
	root := projectRootForResourcesTest(t)
	manifest, err := manifestpkg.Load(filepath.Join(root, "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatalf("load OpenRGB manifest: %v", err)
	}

	home := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(home, ".vrooli"))
	ambientPath := filepath.Join(home, ".config", "OpenRGB", "OpenRGB.json")
	ambientBefore := []byte("{\n    \"UserInterface\": {\n        \"language\": \"preserve-me\"\n    }\n}")
	if err := os.MkdirAll(filepath.Dir(ambientPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ambientPath, ambientBefore, 0o600); err != nil {
		t.Fatal(err)
	}

	env := resourceEnvForResource(root, home, "openrgb")
	env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
	values := managedServiceEnvValues(env)
	configPath := filepath.Join(values["RESOURCE_CONFIG_DIR"], filepath.FromSlash(manifest.ManagedService.Config.Path))
	constructorPath := filepath.Join(values["XDG_CONFIG_HOME"], "OpenRGB", "OpenRGB.json")
	if filepath.Clean(constructorPath) != filepath.Clean(configPath) {
		t.Fatalf("constructor config path = %q, CLI config path = %q", constructorPath, configPath)
	}

	if err := writeManagedServiceConfig(&Controller{Root: root, Home: home}, manifest, env); err != nil {
		t.Fatalf("seed resource-owned config: %v", err)
	}
	seeded, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read seeded resource config: %v", err)
	}
	if len(seeded) == 0 {
		t.Fatal("resource-owned first-launch config is empty")
	}
	ambientAfter, err := os.ReadFile(ambientPath)
	if err != nil {
		t.Fatalf("read ambient config after seed: %v", err)
	}
	if !bytes.Equal(ambientAfter, ambientBefore) {
		t.Fatal("first-launch config seed changed ambient OpenRGB settings")
	}
}
