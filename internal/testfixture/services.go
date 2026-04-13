package testfixture

import (
	"testing"

	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/process"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

type ScenarioServiceOption = testkitvrooli.ScenarioServiceOption
type ResourceManifestOption = testkitvrooli.ResourceManifestOption

func DefaultDisplayName(name string) string {
	return testkitvrooli.DefaultDisplayName(name)
}

func ProjectServiceManifest(opts ...ScenarioServiceOption) scenario.ServiceManifest {
	return testkitvrooli.ProjectServiceManifest(opts...)
}

func ScenarioServiceManifest(name string, opts ...ScenarioServiceOption) scenario.ServiceManifest {
	return testkitvrooli.ScenarioServiceManifest(name, opts...)
}

func ResourceManifest(name string, opts ...ResourceManifestOption) manifestpkg.ResourceManifest {
	return testkitvrooli.ResourceManifest(name, opts...)
}

func WithDescription(description string) ScenarioServiceOption {
	return testkitvrooli.WithDescription(description)
}

func WithDisplayName(displayName string) ScenarioServiceOption {
	return testkitvrooli.WithDisplayName(displayName)
}

func WithLifecycle(lifecycle scenario.Lifecycle) ScenarioServiceOption {
	return testkitvrooli.WithLifecycle(lifecycle)
}

func WithDependencies(dependencies scenario.Dependencies) ScenarioServiceOption {
	return testkitvrooli.WithDependencies(dependencies)
}

func WithPorts(ports map[string]scenario.Port) ScenarioServiceOption {
	return testkitvrooli.WithPorts(ports)
}

func WithResourceDriver(driver string) ResourceManifestOption {
	return testkitvrooli.WithResourceDriver(driver)
}

func WithResourceTemplate(template string) ResourceManifestOption {
	return testkitvrooli.WithResourceTemplate(template)
}

func WithResourceDescription(description string) ResourceManifestOption {
	return testkitvrooli.WithResourceDescription(description)
}

func WithResourceDisplayName(displayName string) ResourceManifestOption {
	return testkitvrooli.WithResourceDisplayName(displayName)
}

func WithResourcePlatforms(platforms manifestpkg.ResourcePlatforms) ResourceManifestOption {
	return testkitvrooli.WithResourcePlatforms(platforms)
}

func WithResourceRuntime(runtime manifestpkg.ResourceRuntime) ResourceManifestOption {
	return testkitvrooli.WithResourceRuntime(runtime)
}

func WithResourceComposeFile(path string) ResourceManifestOption {
	return testkitvrooli.WithResourceComposeFile(path)
}

func WithResourceBinary(binary string) ResourceManifestOption {
	return testkitvrooli.WithResourceBinary(binary)
}

func WithResourceVersionArgs(args ...string) ResourceManifestOption {
	return testkitvrooli.WithResourceVersionArgs(args...)
}

func WithResourceEndpoint(endpoint string) ResourceManifestOption {
	return testkitvrooli.WithResourceEndpoint(endpoint)
}

func WithResourceCredentialsEnv(env ...string) ResourceManifestOption {
	return testkitvrooli.WithResourceCredentialsEnv(env...)
}

func WithResourceHealthChecks(checks ...manifestpkg.ResourceHealthCheck) ResourceManifestOption {
	return testkitvrooli.WithResourceHealthChecks(checks...)
}

func WithResourceInstall(install manifestpkg.ResourceInstall) ResourceManifestOption {
	return testkitvrooli.WithResourceInstall(install)
}

func WithResourceHostTools(tools ...hostreqspec.Declaration) ResourceManifestOption {
	return testkitvrooli.WithResourceHostTools(tools...)
}

func WithResourceHostSafeguards(safeguards ...hostreqspec.Declaration) ResourceManifestOption {
	return testkitvrooli.WithResourceHostSafeguards(safeguards...)
}

func WithLegacyCLIPath(path string) ResourceManifestOption {
	return testkitvrooli.WithLegacyCLIPath(path)
}

func WriteProjectService(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()
	testkitvrooli.WriteProjectService(t, root, manifest)
}

func WriteScenarioService(t *testing.T, root, name string, manifest scenario.ServiceManifest) {
	t.Helper()
	testkitvrooli.WriteScenarioService(t, root, name, manifest)
}

func WriteScenarioServiceAtPath(t *testing.T, path string, manifest scenario.ServiceManifest) {
	t.Helper()
	testkitvrooli.WriteScenarioServiceAtPath(t, path, manifest)
}

func WriteResourceManifest(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()
	testkitvrooli.WriteResourceManifest(t, root, name, manifest)
}

func WriteRelativeFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	testkitgo.WriteRelativeFile(t, root, relPath, contents)
}

func WriteRelativeExecutable(t *testing.T, root, relPath, contents string) string {
	t.Helper()
	return testkitgo.WriteRelativeExecutable(t, root, relPath, contents)
}

func WriteResourceCLI(t *testing.T, root, name, contents string) string {
	t.Helper()
	return testkitvrooli.WriteResourceCLI(t, root, name, contents)
}

func WritePortRegistry(t *testing.T, root string, ports map[string]int) {
	t.Helper()
	testkitvrooli.WritePortRegistry(t, root, ports)
}

func WriteScenarioProcessRecord(t *testing.T, home, name, step string, record process.Record) {
	t.Helper()
	testkitvrooli.WriteScenarioProcessRecord(t, home, name, step, record)
}
