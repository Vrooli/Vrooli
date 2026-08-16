package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	internalcontrol "github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/operatorstate"
	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	StatusCodeOK                   = "ok"
	StatusCodeUnavailable          = "unavailable"
	StatusCodeTimeout              = "timeout"
	StatusCodeCommandError         = "command_error"
	StatusCodeInvalidStatusPayload = "invalid_status_payload"
)

const (
	ErrorCodeCommandUnavailable = "command_unavailable"
	ErrorCodeOperationFailed    = "operation_failed"
	resourceConfigPath          = catalogpkg.ResourceConfigPath
)

type Controller struct {
	Root        string
	Home        string
	Environment string
}

type (
	ConfigEntry     = catalogpkg.ConfigEntry
	Error           = vroolierr.Error
	Resource        = catalogpkg.Resource
	Status          = resourcecontrol.Status
	StatusReport    = resourcecontrol.StatusReport
	DiscoveryReport = discovery.Report[Resource]
)

type commandResult struct {
	output []byte
	err    error
}

var (
	lookPathResourceFn = exec.LookPath
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		output, err := cmd.CombinedOutput()
		return commandResult{output: output, err: err}
	}
)

func NewController(root, home string) *Controller {
	return &Controller{
		Root:        filepath.Clean(root),
		Home:        filepath.Clean(home),
		Environment: normalizeEnvironment(""),
	}
}

func normalizeEnvironment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "development"
	}
	return value
}

func (c *Controller) LoadManifest(path string) (ResourceManifest, error) {
	return manifestpkg.Load(path)
}

func (c *Controller) Discover() ([]Resource, error) {
	report, err := c.DiscoverReport()
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("discover resource %s: %s", failure.Name, failure.Error)
	}
	return report.Items, nil
}

func (c *Controller) DiscoverReport() (DiscoveryReport, error) {
	deprecated, err := c.deprecatedNameSet()
	if err != nil {
		return DiscoveryReport{}, err
	}
	return catalogpkg.New(c.Root).DiscoverReport(catalogpkg.DiscoverOptions{
		DeprecatedNames: deprecated,
		ResolveCLIPath:  c.resolveCLIPath,
	})
}

func (c *Controller) discoverResource(name string) (*Resource, error) {
	deprecated, err := c.deprecatedNameSet()
	if err != nil {
		return nil, err
	}
	return catalogpkg.New(c.Root).DiscoverOne(name, catalogpkg.DiscoverOptions{
		DeprecatedNames: deprecated,
		ResolveCLIPath:  c.resolveCLIPath,
	})
}

func (c *Controller) runResourceCommand(name, operation string, args []string, stdout, stderr io.Writer) error {
	cmd, err := c.commandForResource(name, append([]string{operation}, args...)...)
	if err != nil {
		return err
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return &Error{
			Code:      ErrorCodeOperationFailed,
			Resource:  name,
			Operation: operation,
			Category:  "Runtime",
			Err:       err,
		}
	}
	return nil
}

// RunResourceCLI invokes the resource's installed CLI binary directly,
// bypassing the driver action whitelist. Use this for resource-local verbs
// (e.g. `ensure`) that aren't part of the shared lifecycle surface.
// args[0] is treated as the operation name for error attribution; pass the
// full argv including subcommand flags.
func (c *Controller) RunResourceCLI(name string, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("run resource CLI %s: args must include the operation name", name)
	}
	return c.runResourceCommand(name, args[0], args[1:], stdout, stderr)
}

func (c *Controller) SetEnabled(name string, enabled bool) error {
	patch := map[string]any{"resources": map[string]any{name: map[string]any{"enabled": enabled}}}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	service := operatorstate.New(operatorstate.Config{RepoRoot: c.Root})
	_, err = service.Apply(context.Background(), data)
	return err
}

func (c *Controller) readConfigEntries() (map[string]ConfigEntry, error) {
	entries, err := catalogpkg.New(c.Root).ReadConfigEntries()
	if err != nil {
		return nil, err
	}
	if err := c.seedOperatorState(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// EnabledResourceNames returns the effective resource choices after the
// one-time repository-to-operator-state adoption. Operator state is the
// mutable authority; callers must not read service.json directly after this
// boundary.
func (c *Controller) EnabledResourceNames() ([]string, error) {
	if _, err := c.readConfigEntries(); err != nil {
		return nil, err
	}
	doc, err := operatorstate.New(operatorstate.Config{RepoRoot: c.Root}).Load(context.Background())
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(doc.Resources))
	for name, choice := range doc.Resources {
		if choice.Enabled != nil && *choice.Enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// seedOperatorState adopts legacy repository enabled values once. The
// repository manifest remains readable for migration, but after adoption the
// operator-state document is the only mutable authority.
func (c *Controller) seedOperatorState(entries map[string]ConfigEntry) error {
	if len(entries) == 0 {
		return nil
	}
	service := operatorstate.New(operatorstate.Config{RepoRoot: c.Root})
	doc, err := service.Load(context.Background())
	if err != nil {
		return err
	}
	patchResources := make(map[string]any)
	for name, entry := range entries {
		if _, exists := doc.Resources[name]; exists {
			continue
		}
		patchResources[name] = map[string]any{"enabled": entry.Enabled}
	}
	if len(patchResources) == 0 {
		return nil
	}
	patch, err := json.Marshal(map[string]any{"resources": patchResources})
	if err != nil {
		return err
	}
	if _, err := service.Apply(context.Background(), patch); err != nil {
		return fmt.Errorf("seed resource choices into operator state: %w", err)
	}
	for name := range patchResources {
		log.Printf("adopted resources.%s.enabled from .vrooli/service.json into operator state", name)
	}
	return nil
}

func (c *Controller) statusForResource(item Resource, fast bool) (Status, error) {
	return c.resourceControl().StatusForResource(item, fast)
}

func (c *Controller) resourceControl() *resourcecontrol.Service {
	return &resourcecontrol.Service{
		DiscoverFn: func() ([]catalogpkg.Resource, error) {
			return c.Discover()
		},
		DiscoverReportFn: func() (discovery.Report[catalogpkg.Resource], error) {
			return c.DiscoverReport()
		},
		DiscoverOneFn: func(name string) (*catalogpkg.Resource, error) {
			return c.discoverResource(name)
		},
		IsDeprecatedFn:    c.IsDeprecated,
		IsBlueprintArchFn: c.IsBlueprintArchived,
		LoadManifestFn: func(path string) (ResourceManifest, error) {
			return c.LoadManifest(path)
		},
		DriverStatusFn: func(ctx context.Context, item catalogpkg.Resource, manifest ResourceManifest, fast bool) (resourcecontrol.Status, error) {
			driver, err := driverForManifest(manifest)
			if err != nil {
				return Status{}, err
			}
			return driver.Status(ctx, c, item, manifest, fast)
		},
		DriverRunFn: func(ctx context.Context, item catalogpkg.Resource, manifest ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
			driver, err := driverForManifest(manifest)
			if err != nil {
				return err
			}
			return driver.Run(ctx, c, item, manifest, operation, args, stdout, stderr)
		},
		RunResourceCommandFn: func(name, operation string, args []string, stdout, stderr io.Writer) error {
			return c.runResourceCommand(name, operation, args, stdout, stderr)
		},
		CommandForResourceFn: c.commandForResource,
		RunCommandFn: func(ctx context.Context, cmd *exec.Cmd) resourcecontrol.CommandResult {
			result := runCommandResource(ctx, cmd)
			return resourcecontrol.CommandResult{Output: result.output, Err: result.err}
		},
	}
}

func (c *Controller) Status(name string, fast bool) (Status, error) {
	return c.resourceControl().Status(name, fast)
}

func (c *Controller) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	return c.resourceControl().ListStatuses(fast, onlyEnabled)
}

func (c *Controller) ListStatusesReport(fast bool, onlyEnabled bool) (StatusReport, error) {
	return c.resourceControl().ListStatusesReport(fast, onlyEnabled)
}

func (c *Controller) Run(name string, args []string, stdout, stderr io.Writer) error {
	return c.resourceControl().Run(name, args, stdout, stderr)
}

func (c *Controller) StartAll(stdout, stderr io.Writer) (internalcontrol.StartReport, error) {
	return c.resourceControl().StartAll(stdout, stderr)
}

func (c *Controller) StopAll(stdout, stderr io.Writer) (internalcontrol.StopReport, error) {
	return c.resourceControl().StopAll(stdout, stderr)
}

func (c *Controller) resolveCLIPath(name string) (string, bool) {
	if lookPathResourceFn == nil {
		return "", false
	}
	if path, err := lookPathResourceFn("resource-" + name); err == nil {
		return path, true
	}
	return "", false
}

func (c *Controller) commandForResource(name string, args ...string) (*exec.Cmd, error) {
	if path, ok := c.resolveCLIPath(name); ok {
		env := resourceEnvForResource(c.Root, c.Home, name)
		// Resource CLIs need the same provider-selected, non-secret runtime
		// context as lifecycle operations. In particular, a managed-service
		// client must not fall back to a legacy Docker adapter simply because it
		// was invoked through `vrooli resource` rather than by the supervisor.
		if manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json")); err == nil && manifest.ManagedService != nil {
			for _, port := range manifest.Ports {
				if port.Host > 0 {
					env = setEnvValue(env, managedServicePortEnvName(port.Name), fmt.Sprintf("%d", port.Host))
				}
			}
			for key, value := range manifest.ManagedService.Environment {
				env = setEnvValue(env, key, value)
			}
			env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
			if artifact, err := managedServiceArtifactPath(c, manifest); err == nil {
				env = setEnvValue(env, "VROOLI_MANAGED_SERVICE_ARTIFACT", artifact)
			}
		}
		return shell.Command(shell.Spec{
			Name: path,
			Args: args,
			Dir:  c.Root,
			Env:  env,
		}), nil
	}

	return nil, &Error{
		Code:      ErrorCodeCommandUnavailable,
		Resource:  name,
		Operation: "invoke",
		Category:  "Environment",
		Err:       fmt.Errorf("no installed CLI"),
	}
}

func resourceEnv(root, home string) []string {
	env := os.Environ()
	env = setEnvValue(env, "VROOLI_ROOT", root)
	if strings.TrimSpace(home) != "" {
		env = setEnvValue(env, "HOME", home)
	}
	return env
}

func resourceEnvForResource(root, home, resourceName string) []string {
	env := resourceEnv(root, home)
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return env
	}
	paths, err := resourceStoragePaths(resourceName)
	if err != nil {
		return env
	}
	env = setEnvValue(env, "RESOURCE_ROOT", filepath.Join(root, "resources", resourceName))
	env = setEnvValue(env, "RESOURCE_CONFIG_DIR", paths.ConfigDir)
	env = setEnvValue(env, "RESOURCE_DATA_DIR", paths.DataDir)
	env = setEnvValue(env, "RESOURCE_CACHE_DIR", paths.CacheDir)
	env = setEnvValue(env, "RESOURCE_LOGS_DIR", paths.LogsDir)
	env = setEnvValue(env, "RESOURCE_STATE_DIR", paths.StateDir)
	return env
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if current, ok := parent[key]; ok {
		if typed, typedOK := current.(map[string]any); typedOK {
			return typed
		}
	}
	created := map[string]any{}
	parent[key] = created
	return created
}

func (c *Controller) deprecatedNameSet() (map[string]struct{}, error) {
	items, err := c.loadDeprecatedResources()
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item.Name] = struct{}{}
	}
	return result, nil
}
