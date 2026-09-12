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
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		t.Fatal("generated managed-service fixture must declare acquisition")
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

func validManagedServiceHealthChecks() []ResourceHealthCheck {
	return []ResourceHealthCheck{
		{Type: "http", Target: "http://127.0.0.1:1/health", Kind: "readiness", IntervalSeconds: 10, TimeoutSeconds: 10},
		{Type: "http", Target: "http://127.0.0.1:1/health", Kind: "liveness", IntervalSeconds: 30, TimeoutSeconds: 10},
	}
}

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	err := Validate(ResourceManifest{})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadRejectsDuplicateCredentialDescriptors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.json")
	data, err := json.Marshal(ResourceManifest{
		Name:        "fixture",
		CLI:         validCLI("resource-fixture"),
		Driver:      "external-cli",
		Credentials: ResourceCredentials{Descriptors: []CredentialDescriptor{{LogicalID: "vrooli/demo"}, {LogicalID: "vrooli/demo"}}},
	})
	if err != nil {
		t.Fatalf("marshal resource manifest: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}

	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "declares credential vrooli/demo:value more than once") {
		t.Fatalf("Load error = %v", err)
	}
}

func TestValidateRejectsInvalidDriver(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:   "redis",
		CLI:    validCLI("resource-redis"),
		Driver: "legacy-adapter",
	})
	if err == nil || !strings.Contains(err.Error(), "driver") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExternalCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:      "redis",
		CLI:       validCLI("resource-redis"),
		Driver:    "external-cli",
		Binary:    "redis-server",
		Platforms: ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateAcceptsNativeCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:      "fixturecli",
		CLI:       validCLI("resource-fixturecli"),
		Driver:    "native-cli",
		Binary:    "resource-fixturecli",
		Platforms: ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateManagedServiceRequiresFailClosedProviderPolicy(t *testing.T) {
	base := ResourceManifest{
		Name:         "fixture-service",
		CLI:          validCLI("resource-fixture-service"),
		Driver:       "managed-service",
		Platforms:    ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
		HealthChecks: validManagedServiceHealthChecks(),
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
		Name:         "fixture-service",
		CLI:          validCLI("resource-fixture-service"),
		Driver:       "managed-service",
		Platforms:    ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
		HealthChecks: validManagedServiceHealthChecks(),
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

func TestValidateManagedServiceRequiresReadinessAndLivenessCadence(t *testing.T) {
	valid := ResourceManifest{Name: "fixture", CLI: validCLI("resource-fixture"), Driver: "managed-service", HealthChecks: validManagedServiceHealthChecks()}
	cases := []struct {
		name   string
		checks []ResourceHealthCheck
		want   string
	}{
		{name: "readiness", checks: nil, want: "readiness"},
		{name: "liveness", checks: []ResourceHealthCheck{valid.HealthChecks[0]}, want: "liveness"},
		{name: "cadence", checks: []ResourceHealthCheck{{Type: "http", Target: "http://127.0.0.1:1/health", Kind: "readiness", IntervalSeconds: 0, TimeoutSeconds: 10}, valid.HealthChecks[1]}, want: "positive interval_seconds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid.HealthChecks = tc.checks
			err := Validate(valid)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsDeploymentModeOutsideArchetypeBaseline(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:   "fixture",
		CLI:    validCLI("resource-fixture"),
		Driver: "external-cli", Binary: "fixture",
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
		Name:      "fixture",
		CLI:       validCLI("resource-fixture"),
		Driver:    "external-cli",
		Binary:    "fixture",
		Privilege: "user",
		Bundling:  "host-required",
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
		Name:      "fixture",
		CLI:       validCLI("resource-fixture"),
		Driver:    "external-cli",
		Binary:    "fixture",
		Privilege: "user",
		Bundling:  "host-required",
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
		Name:      "redis",
		CLI:       validCLI("resource-redis"),
		Driver:    "external-cli",
		Binary:    "redis-server",
		Platforms: ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"},
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
		Name:   "fixture",
		Driver: "external-cli",
	})
	if err == nil || !strings.Contains(err.Error(), "cli is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExplicitDisabledCLIBlock(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:   "fixture",
		Binary: "resource-fixture",
		CLI: &scenario.CLIConfig{
			Enabled: false,
		},
		Driver: "external-cli",
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestValidateMemoryLimit(t *testing.T) {
	base := ResourceManifest{
		Name:   "ollama",
		CLI:    validCLI("resource-ollama"),
		Driver: "external-cli",
		Binary: "resource-ollama",
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
		Driver: "external-cli",
	})
	if err == nil {
		t.Fatal("Validate() accepted unsupported CLI adapter")
	}
}
