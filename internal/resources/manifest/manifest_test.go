package manifest

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func validCLI(command string) *scenario.CLIConfig {
	return &scenario.CLIConfig{
		Enabled:     true,
		Command:     command,
		Adapter:     scenario.CLIAdapterConfig{Kind: "go_module", ModuleDir: "cli"},
		SourceBuild: &scenario.CLISourceBuildConfig{Kind: "go_module"},
		Invoke:      scenario.CLIInvokeConfig{Kind: "installed_command", Command: command},
		Freshness:   &scenario.CLIFreshnessCheck{Inputs: []string{"cli/**", "resource.json"}},
	}
}

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	err := Validate(ResourceManifest{})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidDriver(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "redis",
		CLI:             validCLI("resource-redis"),
		Driver:          "legacy-adapter",
		PortabilityTier: "partial",
	})
	if err == nil || !strings.Contains(err.Error(), "driver") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExternalCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "redis",
		CLI:             validCLI("resource-redis"),
		Driver:          "external-cli",
		Binary:          "redis-server",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateAcceptsNativeCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "fixturecli",
		CLI:             validCLI("resource-fixturecli"),
		Driver:          "native-cli",
		Binary:          "resource-fixturecli",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateRejectsDeploymentModeOutsideArchetypeBaseline(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:   "fixture",
		CLI:    validCLI("resource-fixture"),
		Driver: "external-cli", Binary: "fixture", PortabilityTier: "full",
		Deployment: ResourceDeployment{Profiles: map[string]ResourceDeploymentProfile{
			"desktop": {
				Linux:   &ResourceDeploymentTarget{Support: "conditional", Mode: "remote-service", Architectures: []string{"amd64"}, Evidence: []string{"test"}},
				MacOS:   &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "test"},
				Windows: &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "test"},
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "not permitted") {
		t.Fatalf("Validate() error = %v, want archetype baseline rejection", err)
	}
}

func TestValidateAppliesDefaultCLIArtifacts(t *testing.T) {
	manifest := ResourceManifest{
		Name:            "redis",
		CLI:             validCLI("resource-redis"),
		Driver:          "external-cli",
		Binary:          "redis-server",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"},
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if manifest.CLI.Artifacts.Manifest.Location != scenario.CLIArtifactLocationSibling {
		t.Fatalf("manifest artifact location = %q", manifest.CLI.Artifacts.Manifest.Location)
	}
	if manifest.CLI.Artifacts.BuildMetadata.Location != scenario.CLIArtifactLocationSibling {
		t.Fatalf("build metadata artifact location = %q", manifest.CLI.Artifacts.BuildMetadata.Location)
	}
}

func TestSupportForCurrentPlatformUsesMappedOSNames(t *testing.T) {
	value := ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"}.SupportForCurrentPlatform()
	if value == "" {
		t.Fatal("expected support state for current platform")
	}
}

func TestValidateAcceptsLegacyRepoDataMarker(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:                  "legacy-proxy",
		CLI:                   validCLI("resource-legacy-proxy"),
		Driver:                "docker-service",
		PortabilityTier:       "full",
		LegacyRepoDataAllowed: true,
		Runtime: ResourceRuntime{
			Image: "ghcr.io/example/legacy-proxy:latest",
		},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestResourceCredentialsUnmarshalAcceptsSecretDescriptors(t *testing.T) {
	var manifest ResourceManifest
	err := json.Unmarshal([]byte(`{
		"name": "openrouter",
		"cli": {
			"enabled": true,
			"command": "resource-openrouter",
			"adapter": {"kind": "go_module", "module_dir": "cli"}
		},
		"driver": "cloud-api",
		"endpoint": "https://openrouter.ai/api/v1/models",
		"portability_tier": "full",
		"credentials": {
			"env": [
				{
					"env": "OPENROUTER_API_KEY",
					"label": "OpenRouter API Key",
					"description": "OpenRouter unified API gateway.",
					"classification": "user",
					"required": true,
					"obtain_url": "https://openrouter.ai/keys"
				},
				"OPENROUTER_SITE_URL"
			],
			"secret_ref": "secret/openrouter"
		}
	}`), &manifest)
	if err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if got, want := manifest.Credentials.Env, []string{"OPENROUTER_API_KEY", "OPENROUTER_SITE_URL"}; !slices.Equal(got, want) {
		t.Fatalf("Credentials.Env = %#v, want %#v", got, want)
	}
	if manifest.Credentials.SecretRef != "secret/openrouter" {
		t.Fatalf("Credentials.SecretRef = %q", manifest.Credentials.SecretRef)
	}
}

func TestValidateRejectsMissingCLIBlock(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "fixture",
		Driver:          "manual",
		PortabilityTier: "full",
	})
	if err == nil || !strings.Contains(err.Error(), "cli is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExplicitDisabledCLIBlock(t *testing.T) {
	err := Validate(ResourceManifest{
		Name: "fixture",
		CLI: &scenario.CLIConfig{
			Enabled: false,
		},
		Driver:          "manual",
		PortabilityTier: "full",
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateMemoryLimit(t *testing.T) {
	base := ResourceManifest{
		Name:            "ollama",
		CLI:             validCLI("resource-ollama"),
		Driver:          "docker-service",
		PortabilityTier: "full",
		Runtime:         ResourceRuntime{Image: "ollama/ollama:latest"},
	}

	for _, ok := range []string{"", "12g", "8192m", "536870912", "1B", "4G", "  2g  "} {
		m := base
		m.Runtime.MemoryLimit = ok
		if err := Validate(m); err != nil {
			t.Fatalf("Validate(memory_limit=%q) returned error: %v", ok, err)
		}
	}

	for _, bad := range []string{"0g", "-1g", "12gb", "abc", "12 g", "g12"} {
		m := base
		m.Runtime.MemoryLimit = bad
		if err := Validate(m); err == nil {
			t.Fatalf("Validate(memory_limit=%q) expected error", bad)
		}
	}
}

func TestValidateRejectsUnsupportedCLIAdapter(t *testing.T) {
	err := Validate(ResourceManifest{
		Name: "fixture",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-fixture",
			Adapter: scenario.CLIAdapterConfig{
				Kind: "script",
			},
		},
		Driver:          "manual",
		PortabilityTier: "full",
	})
	if err == nil {
		t.Fatal("Validate() accepted unsupported CLI adapter")
	}
}
