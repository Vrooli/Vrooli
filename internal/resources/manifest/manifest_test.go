package manifest

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	err := Validate(ResourceManifest{})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidDriver(t *testing.T) {
	err := Validate(ResourceManifest{
		Name: "redis",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-redis",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		},
		Driver:          "legacy-adapter",
		PortabilityTier: "partial",
	})
	if err == nil || !strings.Contains(err.Error(), "driver") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExternalCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name: "redis",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-redis",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		},
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
		Name: "fixturecli",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-fixturecli",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		},
		Driver:          "native-cli",
		Binary:          "resource-fixturecli",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateAppliesDefaultCLIArtifacts(t *testing.T) {
	manifest := ResourceManifest{
		Name: "redis",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-redis",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		},
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
		Name: "litellm",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-litellm",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		},
		Driver:                "docker-service",
		PortabilityTier:       "full",
		LegacyRepoDataAllowed: true,
		Runtime: ResourceRuntime{
			Image: "ghcr.io/berriai/litellm:main-latest",
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
		Name: "ollama",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-ollama",
			Adapter: scenario.CLIAdapterConfig{Kind: "go_module", ModuleDir: "cli"},
		},
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

func TestValidateAcceptsShellScriptCLIBlock(t *testing.T) {
	err := Validate(ResourceManifest{
		Name: "fixture",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-fixture",
			Adapter: scenario.CLIAdapterConfig{
				Kind:          "shell_script",
				ScriptPath:    "cli/resource-fixture",
				InstallScript: "cli/install.sh",
			},
		},
		Driver:          "manual",
		PortabilityTier: "full",
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}
