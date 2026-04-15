package resourcefixture

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	"github.com/vrooli/vrooli/packages/testkit-go/internal/displayname"
)

type ResourceManifestOption func(*manifestpkg.ResourceManifest)

type ResourceTemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type ResourceTemplateManifest struct {
	Name                 string                         `json:"name,omitempty"`
	DisplayName          string                         `json:"displayName,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Driver               string                         `json:"driver,omitempty"`
	RequiredVars         map[string]ResourceTemplateVar `json:"requiredVars,omitempty"`
	OptionalVars         map[string]ResourceTemplateVar `json:"optionalVars,omitempty"`
	Docs                 map[string]string              `json:"docs,omitempty"`
	PlatformExpectations []string                       `json:"platformExpectations,omitempty"`
	Transitional         bool                           `json:"transitional,omitempty"`
}

type ResourceTemplateOption func(*ResourceTemplateManifest)

func ResourceManifest(name string, opts ...ResourceManifestOption) manifestpkg.ResourceManifest {
	manifest := manifestpkg.ResourceManifest{
		Name:            name,
		DisplayName:     displayname.Default(name),
		Description:     fmt.Sprintf("%s fixture", displayname.Default(name)),
		Driver:          "external-cli",
		Template:        "external-cli",
		Binary:          "bash",
		PortabilityTier: "full",
		Platforms: manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "supported",
			Windows: "supported",
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func ResourceTemplate(name string, opts ...ResourceTemplateOption) ResourceTemplateManifest {
	manifest := ResourceTemplateManifest{
		Name:        name,
		DisplayName: displayname.Default(name),
		Description: fmt.Sprintf("%s fixture template", displayname.Default(name)),
		Driver:      "docker-service",
		RequiredVars: map[string]ResourceTemplateVar{
			"RESOURCE_NAME": {Flag: "name", Description: "Fixture resource name"},
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func WithResourceDriver(driver string) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.Driver = driver
	}
}

func WithTemplateDisplayName(displayName string) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.DisplayName = displayName
	}
}

func WithTemplateDescription(description string) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.Description = description
	}
}

func WithTemplateDriver(driver string) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.Driver = driver
	}
}

func WithTemplateRequiredVars(vars map[string]ResourceTemplateVar) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.RequiredVars = cloneTemplateVarMap(vars)
	}
}

func WithTemplateOptionalVars(vars map[string]ResourceTemplateVar) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.OptionalVars = cloneTemplateVarMap(vars)
	}
}

func WithTemplateDocs(docs map[string]string) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.Docs = cloneTemplateDocs(docs)
	}
}

func WithTemplatePlatformExpectations(expectations ...string) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.PlatformExpectations = append([]string(nil), expectations...)
	}
}

func WithTemplateTransitional(enabled bool) ResourceTemplateOption {
	return func(manifest *ResourceTemplateManifest) {
		manifest.Transitional = enabled
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

func WithResourceDependencySchema(raw json.RawMessage) ResourceManifestOption {
	return func(manifest *manifestpkg.ResourceManifest) {
		manifest.DependencySchema = append(json.RawMessage(nil), raw...)
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

func WriteResourceManifest(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = name
	}
	testkitgo.WriteJSON(t, manifestpkg.DefaultPath(root, name), manifest)
}

func WriteResourceTemplateManifest(t *testing.T, root, name string, manifest ResourceTemplateManifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = name
	}
	testkitgo.WriteJSON(t, filepath.Join(root, "templates", "resources", name, "template.json"), manifest)
}

func WriteResourceCLIGoMod(t *testing.T, root, name, module string) {
	t.Helper()
	if strings.TrimSpace(module) == "" {
		module = "resource-" + name + "/cli"
	}
	testkitgo.WriteFile(t, filepath.Join(root, "resources", name, "cli", "go.mod"), "module "+module+"\n")
}

func WriteMalformedResourceManifest(t *testing.T, root, name, raw string) {
	t.Helper()
	testkitgo.WriteMalformedJSON(t, manifestpkg.DefaultPath(root, name), raw, 0o644)
}

func ReadResourceManifest(t *testing.T, root, name string) manifestpkg.ResourceManifest {
	t.Helper()
	return testkitgo.ReadJSONFileInto[manifestpkg.ResourceManifest](t, manifestpkg.DefaultPath(root, name))
}

func WriteResourceCLI(t *testing.T, root, name, contents string) string {
	t.Helper()
	return testkitgo.WriteRelativeExecutable(t, root, filepath.Join("resources", name, "cli.sh"), contents)
}

func WriteExternalCLIResourceFixture(t *testing.T, root, name, script string, opts ...ResourceManifestOption) string {
	t.Helper()
	binary := "resource-" + strings.TrimSpace(name)
	manifestOpts := append([]ResourceManifestOption{
		WithResourceDriver("external-cli"),
		WithResourceTemplate("external-cli"),
		WithResourceBinary(binary),
	}, opts...)
	WriteResourceManifest(t, root, name, ResourceManifest(name, manifestOpts...))
	WriteResourceCLI(t, root, name, script)
	return testkitgo.WriteExecutableOnPath(t, binary, script)
}

func WritePortRegistry(t *testing.T, root string, ports map[string]int) {
	t.Helper()
	WritePortRegistryState(t, root, resourceenv.PortRegistry{
		ResourcePorts:  ports,
		ReservedRanges: map[string]string{},
	})
}

func WritePortRegistryState(t *testing.T, root string, registry resourceenv.PortRegistry) {
	t.Helper()

	ports := cloneIntMap(registry.ResourcePorts)
	ranges := cloneStringMap(registry.ReservedRanges)
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
			lines = append(lines, fmt.Sprintf("  [\"%s\"]=\"%d\"", name, ports[name]))
		}
		lines = append(lines, ")")
	}
	testkitgo.WriteFile(t, scriptPath, strings.Join(lines, "\n")+"\n")

	testkitgo.WriteJSON(t, filepath.Join(root, "scripts", "resources", "port_registry.json"), resourceenv.PortRegistry{
		ResourcePorts:  ports,
		ReservedRanges: ranges,
	})
}

func WriteResourceRegistryEntry(t *testing.T, root, name string) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, ".vrooli", "resource-registry", name+".json"), map[string]any{
		"name": name,
	})
}

func WriteResourceDefinitionsMetadata(t *testing.T, root string) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, ".vrooli", "schemas", "resource-definitions.json"), map[string]any{
		"definitions": map[string]any{
			"resourceSchemas": map[string]any{},
		},
	})
}

func WriteFakeDocker(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "docker-state.txt")
	scriptPath := filepath.Join(dir, "docker")
	script := `#!/usr/bin/env bash
set -e
state_file="${FAKE_DOCKER_STATE}"
cmd="${1:-}"
shift || true

case "$cmd" in
  compose)
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -f|--project-name)
          shift 2
          ;;
        *)
          break
          ;;
      esac
    done
    subcmd="${1:-}"
    shift || true
    case "$subcmd" in
      ps)
        if [[ "${1:-}" == "-a" ]]; then
          shift
        fi
        if [[ "${1:-}" == "--format" ]]; then
          shift 2
        fi
        if [[ -f "$state_file" ]]; then
          state="$(tr -d '\n' < "$state_file")"
          if [[ "$state" == "running" ]]; then
            printf '[{"Service":"app","State":"running","Health":"healthy"}]'
          else
            printf '[{"Service":"app","State":"exited","Health":""}]'
          fi
        else
          printf '[]'
        fi
        exit 0
        ;;
      pull|up)
        printf 'running\n' > "$state_file"
        exit 0
        ;;
      stop)
        printf 'stopped\n' > "$state_file"
        exit 0
        ;;
      down)
        rm -f "$state_file"
        exit 0
        ;;
      logs)
        echo "fixture logs"
        exit 0
        ;;
    esac
    ;;
  image)
    if [[ "${1:-}" == "inspect" ]]; then
      exit 0
    fi
    ;;
  inspect)
    if [[ -f "$state_file" ]]; then
      state="$(tr -d '\n' < "$state_file")"
      if [[ "$state" == "running" ]]; then
        printf '{"Running":true}'
      else
        printf '{"Running":false}'
      fi
      exit 0
    fi
    echo "Error: No such object" >&2
    exit 1
    ;;
  run)
    printf 'running\n' > "$state_file"
    echo "container-id"
    exit 0
    ;;
  start)
    printf 'running\n' > "$state_file"
    exit 0
    ;;
  stop)
    printf 'stopped\n' > "$state_file"
    exit 0
    ;;
  restart)
    printf 'running\n' > "$state_file"
    exit 0
    ;;
  rm)
    rm -f "$state_file"
    exit 0
    ;;
  logs)
    echo "fixture logs"
    exit 0
    ;;
esac

echo "unexpected docker invocation: $cmd $*" >&2
exit 1
`
	testkitgo.WriteExecutable(t, scriptPath, script)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DOCKER_STATE", stateFile)
	return stateFile
}

func UseSystemLookPath(t *testing.T, target *func(file string) (string, error)) {
	t.Helper()
	original := *target
	*target = exec.LookPath
	t.Cleanup(func() {
		*target = original
	})
}

func cloneIntMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return map[string]int{}
	}
	cloned := make(map[string]int, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneTemplateVarMap(vars map[string]ResourceTemplateVar) map[string]ResourceTemplateVar {
	if len(vars) == 0 {
		return nil
	}
	cloned := make(map[string]ResourceTemplateVar, len(vars))
	for name, spec := range vars {
		cloned[name] = spec
	}
	return cloned
}

func cloneTemplateDocs(docs map[string]string) map[string]string {
	if len(docs) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(docs))
	for name, path := range docs {
		cloned[name] = path
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
