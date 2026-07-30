package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedServiceTemplateGeneratesTargetAwareFixture(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(workingDir, "..", "..", "..", "templates", "resources", "managed-service", "resource.json"))
	if err != nil {
		t.Fatalf("read managed-service template: %v", err)
	}
	generated := strings.NewReplacer(
		"{{RESOURCE_NAME}}", "fixture-service",
		"{{RESOURCE_DISPLAY_NAME}}", "Fixture Service",
		"{{RESOURCE_DESCRIPTION}}", "Fixture managed service",
		"{{RESOURCE_CLI_COMMAND}}", "resource-fixture-service",
		"{{SERVICE_VERSION}}", "1.0.0",
		"{{SERVICE_SHA256}}", strings.Repeat("a", 64),
		"{{RESOURCE_PORTABILITY_TIER}}", "full",
		"{{RESOURCE_CATEGORY}}", "storage",
	).Replace(string(data))
	var manifest ResourceManifest
	if err := json.Unmarshal([]byte(generated), &manifest); err != nil {
		t.Fatalf("parse generated fixture: %v", err)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("validate generated fixture: %v", err)
	}
	policy := manifest.ManagedService.ProviderPolicy
	if got, err := policy.ResolveProvider(resourcedeployment.ProviderRequest{Target: resourcedeployment.ProviderTargetControlPlane}); err != nil || got != resourcedeployment.ProviderManagedShared {
		t.Fatalf("generated control-plane default = %q, %v", got, err)
	}
	if got, err := policy.ResolveProvider(resourcedeployment.ProviderRequest{Target: resourcedeployment.ProviderTargetDesktopBundle}); err != nil || got != resourcedeployment.ProviderManagedPrivate {
		t.Fatalf("generated desktop default = %q, %v", got, err)
	}
}

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

func TestValidateManagedServiceRequiresFailClosedProviderPolicy(t *testing.T) {
	base := ResourceManifest{
		Name:            "fixture-service",
		CLI:             validCLI("resource-fixture-service"),
		Driver:          "managed-service",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
	}
	if err := Validate(base); err == nil || !strings.Contains(err.Error(), "managed_service is required") {
		t.Fatalf("missing policy error = %v", err)
	}
	base.ManagedService = &ResourceManagedService{ProviderPolicy: resourcedeployment.ProviderPolicy{
		TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{
			resourcedeployment.ProviderTargetControlPlane:  resourcedeployment.ProviderManagedPrivate,
			resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate,
		},
		AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderAttachOnly},
		ExternalAccessCapabilities: []resourcedeployment.AccessCapability{resourcedeployment.AccessReadOnly},
		SharedReuseRequiresConsent: true,
		ExternalManagement:         "forbidden",
	}, AttachHealthPath: "/health", Artifact: resourcedeployment.ServiceArtifact{
		Path:    "bin/fixture-service",
		Version: "1.0.0",
		SHA256:  strings.Repeat("a", 64),
	}}
	if err := Validate(base); err != nil {
		t.Fatalf("managed-service manifest validation: %v", err)
	}
}

func TestValidateManagedServiceRejectsInvalidEnvironmentKey(t *testing.T) {
	manifest := ResourceManifest{
		Name:            "fixture-service",
		CLI:             validCLI("resource-fixture-service"),
		Driver:          "managed-service",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
		ManagedService: &ResourceManagedService{ProviderPolicy: resourcedeployment.ProviderPolicy{
			TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{
				resourcedeployment.ProviderTargetControlPlane:  resourcedeployment.ProviderManagedPrivate,
				resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate,
			},
			AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate},
			SharedReuseRequiresConsent: true,
			ExternalManagement:         "forbidden",
		}, Artifact: resourcedeployment.ServiceArtifact{Path: "bin/fixture", Version: "1.0.0", SHA256: strings.Repeat("a", 64)}, Environment: map[string]string{"INVALID-KEY": "value"}},
	}
	if err := Validate(manifest); err == nil || !strings.Contains(err.Error(), "environment key") {
		t.Fatalf("Validate() error = %v, want environment key error", err)
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

func TestValidateRejectsUnknownDeploymentTargetRequirement(t *testing.T) {
	manifest := ResourceManifest{
		Name:            "fixture",
		CLI:             validCLI("resource-fixture"),
		Driver:          "external-cli",
		Binary:          "fixture",
		PortabilityTier: "full",
		Privilege:       "user",
		Bundling:        "host-required",
		Deployment: ResourceDeployment{Profiles: map[string]ResourceDeploymentProfile{
			"desktop": {
				Linux: &ResourceDeploymentTarget{
					Support: "conditional", Mode: "native-host-tool", Architectures: []string{"amd64"},
					Requires: []string{"secret-toool"}, Limitations: []string{"fixture host dependency"}, Evidence: []string{"fixture-test"},
				},
				MacOS:   &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "fixture"},
				Windows: &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "fixture"},
			},
		}},
	}
	if err := Validate(manifest); err == nil || !strings.Contains(err.Error(), "unknown registered tool or safeguard") || !strings.Contains(err.Error(), "secret-toool") {
		t.Fatalf("Validate() error = %v, want unknown target requirement", err)
	}
}

func TestValidateRejectsDuplicateDeploymentTargetRequirement(t *testing.T) {
	manifest := ResourceManifest{
		Name:            "fixture",
		CLI:             validCLI("resource-fixture"),
		Driver:          "external-cli",
		Binary:          "fixture",
		PortabilityTier: "full",
		Privilege:       "user",
		Bundling:        "host-required",
		Deployment: ResourceDeployment{Profiles: map[string]ResourceDeploymentProfile{
			"desktop": {
				Linux: &ResourceDeploymentTarget{
					Support: "conditional", Mode: "native-host-tool", Architectures: []string{"amd64"},
					Requires: []string{"secret-tool", "secret-tool"}, Limitations: []string{"fixture host dependency"}, Evidence: []string{"fixture-test"},
				},
				MacOS:   &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "fixture"},
				Windows: &ResourceDeploymentTarget{Support: "unsupported", Mode: "native-host-tool", Reason: "fixture"},
			},
		}},
	}
	if err := Validate(manifest); err == nil || !strings.Contains(err.Error(), "duplicate") || !strings.Contains(err.Error(), "secret-tool") {
		t.Fatalf("Validate() error = %v, want duplicate target requirement", err)
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
			Image: "ghcr.io/example/legacy-proxy:1.2.3",
		},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateEnforcesPinnedDockerImage(t *testing.T) {
	base := ResourceManifest{
		Name:            "pinned",
		CLI:             validCLI("resource-pinned"),
		Driver:          "docker-service",
		PortabilityTier: "full",
	}

	for _, ok := range []string{
		"postgres:16-alpine",
		"ghcr.io/example/app:v1.2.3",
		"registry.example.com:5000/team/app:2026.07",
		"minio/minio@sha256:d249d1fb6966de4d8ad26c04754b545205ff15a62e4fd19ebd0f26fa5baacbc0",
	} {
		m := base
		m.Runtime = ResourceRuntime{Image: ok}
		if err := Validate(m); err != nil {
			t.Fatalf("Validate(image=%q) returned error: %v", ok, err)
		}
	}

	for _, bad := range []string{
		"minio/minio",
		"minio/minio:latest",
		"homeassistant/home-assistant:stable",
		"registry.example.com:5000/team/app",
		"example/app@",
	} {
		m := base
		m.Runtime = ResourceRuntime{Image: bad}
		if err := Validate(m); err == nil {
			t.Fatalf("Validate(image=%q) expected error, got nil", bad)
		}
	}
}

func TestValidateRejectsProfileContradictingUnsupportedPlatform(t *testing.T) {
	m := ResourceManifest{
		Name:            "contradiction",
		CLI:             validCLI("resource-contradiction"),
		Driver:          "compose-service",
		ComposeFile:     "docker/docker-compose.yml",
		PortabilityTier: "platform-specific",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "unsupported", Windows: "unsupported"},
		Deployment: ResourceDeployment{
			Profiles: map[string]ResourceDeploymentProfile{
				"desktop": {
					Linux:   &ResourceDeploymentTarget{Support: "conditional", Mode: "docker-desktop", Architectures: []string{"amd64"}, Evidence: []string{"manifest-validation"}, Limitations: []string{"requires docker"}},
					MacOS:   &ResourceDeploymentTarget{Support: "conditional", Mode: "docker-desktop", Architectures: []string{"amd64"}, Evidence: []string{"manifest-validation"}, Limitations: []string{"requires docker"}},
					Windows: &ResourceDeploymentTarget{Support: "unsupported", Mode: "docker-desktop", Reason: "no gpu passthrough"},
				},
			},
		},
	}
	err := Validate(m)
	if err == nil {
		t.Fatal("expected contradiction error, got nil")
	}
	if !strings.Contains(err.Error(), "contradicts platforms.macos") {
		t.Fatalf("expected macos contradiction error, got: %v", err)
	}

	m.Deployment.Profiles["desktop"].MacOS.Support = "unsupported"
	m.Deployment.Profiles["desktop"].MacOS.Reason = "no gpu passthrough"
	if err := Validate(m); err != nil {
		t.Fatalf("Validate() after aligning profile: %v", err)
	}
}

func TestValidateHealthCheckKind(t *testing.T) {
	base := ResourceManifest{
		Name:            "healthy",
		CLI:             validCLI("resource-healthy"),
		Driver:          "compose-service",
		ComposeFile:     "docker/docker-compose.yml",
		PortabilityTier: "full",
	}

	for _, kind := range []string{"", "readiness", "liveness"} {
		m := base
		m.HealthChecks = []ResourceHealthCheck{{Type: "http", Target: "http://127.0.0.1:8080/ready", Kind: kind}}
		if err := Validate(m); err != nil {
			t.Fatalf("Validate(kind=%q) returned error: %v", kind, err)
		}
	}

	m := base
	m.HealthChecks = []ResourceHealthCheck{{Type: "http", Target: "http://127.0.0.1:8080/ready", Kind: "alive"}}
	if err := Validate(m); err == nil {
		t.Fatal("expected invalid health check kind error, got nil")
	}
}

func TestResourceCredentialsUnmarshalAcceptsCanonicalDescriptors(t *testing.T) {
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
		"credentials": {"descriptors": [{
			"logical_id": "vrooli/openrouter", "field": "api-key",
			"env": "OPENROUTER_API_KEY", "label": "OpenRouter API Key",
			"description": "OpenRouter unified API gateway.", "required": true,
			"obtain_url": "https://openrouter.ai/keys"
		}]}
	}`), &manifest)
	if err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if got := manifest.Credentials.All(); len(got) != 1 || got[0].LogicalID != "vrooli/openrouter" {
		t.Fatalf("Credentials.All() = %#v", got)
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
		Runtime:         ResourceRuntime{Image: "ollama/ollama:0.30.10"},
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
