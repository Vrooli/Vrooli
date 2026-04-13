package testfixture

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/process"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/testutil"
)

type ScenarioServiceOption func(*scenario.ServiceManifest)
type ResourceManifestOption func(*manifestpkg.ResourceManifest)

func DefaultDisplayName(name string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func ProjectServiceManifest(opts ...ScenarioServiceOption) scenario.ServiceManifest {
	manifest := scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        "project-alpha",
			DisplayName: "Project Alpha",
			Description: "Project-level fixture",
			Version:     "0.1.0",
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func ScenarioServiceManifest(name string, opts ...ScenarioServiceOption) scenario.ServiceManifest {
	manifest := scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: DefaultDisplayName(name),
			Description: fmt.Sprintf("%s fixture", DefaultDisplayName(name)),
			Version:     "0.1.0",
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func ResourceManifest(name string, opts ...ResourceManifestOption) manifestpkg.ResourceManifest {
	manifest := manifestpkg.ResourceManifest{
		Name:            name,
		DisplayName:     DefaultDisplayName(name),
		Description:     fmt.Sprintf("%s fixture", DefaultDisplayName(name)),
		Driver:          "legacy-adapter",
		Template:        "legacy-adapter",
		PortabilityTier: "partial",
		Platforms: manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "partial",
			Windows: "unsupported",
		},
		LegacyAdapter: manifestpkg.ResourceLegacyAdapter{
			Owner:            "CLI tests",
			DecisionDeadline: "2026-12-31",
			FinalDisposition: "migrate",
			LegacyCLIPath:    filepath.ToSlash(filepath.Join("resources", name, "cli.sh")),
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func WithDescription(description string) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Service.Description = description
	}
}

func WithDisplayName(displayName string) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Service.DisplayName = displayName
	}
}

func WithLifecycle(lifecycle scenario.Lifecycle) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Lifecycle = lifecycle
	}
}

func WithDependencies(dependencies scenario.Dependencies) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Dependencies = dependencies
	}
}

func WithPorts(ports map[string]scenario.Port) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Ports = ports
	}
}

func WithResourceDriver(driver string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Driver = driver
	}
}

func WithResourceTemplate(template string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Template = template
	}
}

func WithResourceDescription(description string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Description = description
	}
}

func WithResourceDisplayName(displayName string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.DisplayName = displayName
	}
}

func WithResourcePlatforms(platforms manifestpkg.ResourcePlatforms) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Platforms = platforms
	}
}

func WithResourceRuntime(runtime manifestpkg.ResourceRuntime) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Runtime = runtime
	}
}

func WithResourceComposeFile(path string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.ComposeFile = filepath.ToSlash(path)
	}
}

func WithResourceBinary(binary string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Binary = binary
	}
}

func WithResourceVersionArgs(args ...string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.VersionArgs = append([]string(nil), args...)
	}
}

func WithResourceEndpoint(endpoint string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Endpoint = endpoint
	}
}

func WithResourceCredentialsEnv(env ...string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Credentials.Env = append([]string(nil), env...)
	}
}

func WithResourceHealthChecks(checks ...manifestpkg.ResourceHealthCheck) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.HealthChecks = append([]manifestpkg.ResourceHealthCheck(nil), checks...)
	}
}

func WithResourceInstall(install manifestpkg.ResourceInstall) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Install = install
	}
}

func WithResourceHostTools(tools ...hostreqspec.Declaration) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.HostTools = append([]hostreqspec.Declaration(nil), tools...)
	}
}

func WithResourceHostSafeguards(safeguards ...hostreqspec.Declaration) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.HostSafeguards = append([]hostreqspec.Declaration(nil), safeguards...)
	}
}

func WithLegacyCLIPath(path string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.LegacyAdapter.LegacyCLIPath = filepath.ToSlash(path)
	}
}

func WriteProjectService(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()
	testutil.WriteJSON(t, scenario.ProjectServicePath(root), manifest)
}

func WriteScenarioService(t *testing.T, root, name string, manifest scenario.ServiceManifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Service.Name) == "" {
		manifest.Service.Name = name
	}
	testutil.WriteJSON(t, scenario.ServicePath(root, name), manifest)
}

func WriteScenarioServiceAtPath(t *testing.T, path string, manifest scenario.ServiceManifest) {
	t.Helper()
	testutil.WriteJSON(t, filepath.Join(path, ".vrooli", "service.json"), manifest)
}

func WriteResourceManifest(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = name
	}
	testutil.WriteJSON(t, manifestpkg.DefaultPath(root, name), manifest)
}

func WriteRelativeFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(relPath)), contents)
}

func WriteRelativeExecutable(t *testing.T, root, relPath, contents string) string {
	t.Helper()
	return testutil.WriteExecutable(t, filepath.Join(root, filepath.FromSlash(relPath)), contents)
}

func WriteResourceCLI(t *testing.T, root, name, contents string) string {
	t.Helper()
	return WriteRelativeExecutable(t, root, filepath.Join("resources", name, "cli.sh"), contents)
}

func WritePortRegistry(t *testing.T, root string, ports map[string]int) {
	t.Helper()

	scriptPath := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	lines := []string{"#!/usr/bin/env bash"}
	if len(ports) == 0 {
		lines = append(lines, "declare -g -A RESOURCE_PORTS=()")
	} else {
		names := make([]string, 0, len(ports))
		for name := range ports {
			names = append(names, name)
		}
		sort.Strings(names)
		lines = append(lines, "RESOURCE_PORTS=(")
		for _, name := range names {
			port := ports[name]
			lines = append(lines, fmt.Sprintf("  [\"%s\"]=\"%d\"", name, port))
		}
		lines = append(lines, ")")
	}
	testutil.WriteFile(t, scriptPath, strings.Join(lines, "\n")+"\n")

	resourcePorts := make(map[string]any, len(ports))
	for name, port := range ports {
		resourcePorts[name] = port
	}
	testutil.WriteJSON(t, filepath.Join(root, "scripts", "resources", "port_registry.json"), map[string]any{
		"resource_ports":  resourcePorts,
		"reserved_ranges": map[string]any{},
	})
}

func WriteScenarioProcessRecord(t *testing.T, home, name, step string, record process.Record) {
	t.Helper()
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}
	if err := process.WriteScenarioRecord(home, name, step, record); err != nil {
		t.Fatalf("write scenario process record: %v", err)
	}
}
