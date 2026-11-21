package resources

import (
	"os"
	"path/filepath"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedUpdateContractFreezesDeclaredConfigFile(t *testing.T) {
	m, err := manifestpkg.Load(filepath.Join(projectRoot(t), "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.ManagedService.UpdatePolicy = &resourcedeployment.ServiceUpdatePolicy{Strategy: resourcedeployment.UpdateStrategyExclusiveStopStart, CandidatePort: "sdk"}
	for i := range m.HealthChecks {
		m.HealthChecks[i].Target = "127.0.0.1:${RESOURCE_PORT_SDK}"
	}
	root := t.TempDir()
	asset := filepath.Join(root, filepath.FromSlash(m.ManagedService.Config.ContentFile))
	if err := os.MkdirAll(filepath.Dir(asset), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{\"Server\":{\"enabled\":false}}\n")
	if err := os.WriteFile(asset, original, 0o600); err != nil {
		t.Fatal(err)
	}
	contract, err := newManagedUpdateContractForResource(m, ManagedServiceArtifact{Resource: m.Name, Path: filepath.Join(root, "openrgb.AppImage"), Version: m.ManagedService.Artifact.Version, SHA256: m.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"]}, root)
	if err != nil {
		t.Fatal(err)
	}
	if string(contract.ConfigBytes) != string(original) || contract.ConfigFile != m.ManagedService.Config.ContentFile || contract.ConfigSHA256 == "" {
		t.Fatalf("frozen config identity = file:%q digest:%q bytes:%q", contract.ConfigFile, contract.ConfigSHA256, contract.ConfigBytes)
	}
	if err := os.WriteFile(asset, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := string(contract.ConfigBytes); got != string(original) {
		t.Fatalf("contract config changed with source asset: %q", got)
	}
	contract.ConfigBytes[0] ^= 1
	if err := contract.validate(); err == nil {
		t.Fatal("mutated config bytes passed the frozen digest check")
	}
}

func TestManagedUpdateConfigFileRejectsEscapeAndOversize(t *testing.T) {
	root := t.TempDir()
	if _, err := readManagedUpdateConfigFile(root, "../outside.json"); err == nil {
		t.Fatal("escaping config asset was accepted")
	}
	asset := filepath.Join(root, "large.json")
	if err := os.WriteFile(asset, make([]byte, managedUpdateConfigMaxBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readManagedUpdateConfigFile(root, "large.json"); err == nil {
		t.Fatal("oversized config asset was accepted")
	}
}
