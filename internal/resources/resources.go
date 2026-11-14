package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/capacity"
	internalcontrol "github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/operatorstate"
	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
	"github.com/vrooli/vrooli/internal/values"
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
	Root          string
	Home          string
	Environment   string
	operatorState *operatorstate.Service
	// NoReclaim requires manifest-derived starts to use only already-free capacity.
	NoReclaim bool
	// healthObservationOnly separates frequent process/health observations from
	// current on-disk artifact audits. Launch verification is always independent.
	healthObservationOnly bool
	// admitCapacity is an instance-scoped owner seam; nil uses canonical admission.
	admitCapacity func(context.Context, capacity.AdmitOptions, func(context.Context) error) error
	// The supervisor retains one controller so bounded recovery passes rotate
	// fairly across unresolved resources instead of retrying the first forever.
	openRGBPayloadEffects *openRGBPayloadEffects
	updateEffects         *managedUpdateEffects
	openRGBUpdateEffects  *openRGBPublicUpdateEffects
	updateResumeMu        sync.Mutex
	updateResumeAfter     string
	updateCleanupAfter    string
}

func (c *Controller) operatorStateService() *operatorstate.Service {
	if c.operatorState != nil {
		return c.operatorState
	}
	cfg := operatorstate.Config{RepoRoot: c.Root}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		cfg.LogicalOperatorHome = c.Home
	}
	return operatorstate.New(cfg)
}

type (
	ConfigEntry     = catalogpkg.ConfigEntry
	CensusRow       = catalogpkg.CensusRow
	Error           = vroolierr.Error
	Resource        = catalogpkg.Resource
	Status          = resourcecontrol.Status
	StatusReport    = resourcecontrol.StatusReport
	DiscoveryReport = scenario.Report[Resource]
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

// ForHealthObservation returns an isolated read configuration. Its status
// records explicitly leave artifact verification unevaluated; it changes no
// acquisition or launch gate and never mutates the original controller.
func (c *Controller) ForHealthObservation() *Controller {
	return &Controller{
		Root: c.Root, Home: c.Home, Environment: c.Environment, operatorState: c.operatorState,
		NoReclaim: c.NoReclaim, healthObservationOnly: true, admitCapacity: c.admitCapacity,
		openRGBPayloadEffects: c.openRGBPayloadEffects, updateEffects: c.updateEffects,
		openRGBUpdateEffects: c.openRGBUpdateEffects,
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

func (c *Controller) Census() ([]CensusRow, error) {
	deprecated, err := c.deprecatedNameSet()
	if err != nil {
		return nil, err
	}
	_ = deprecated
	return catalogpkg.New(c.Root).Census(catalogpkg.DiscoverOptions{ResolveCLIPath: c.resolveCLIPath})
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

func (c *Controller) runResourceCommand(ctx context.Context, name, operation string, args []string, stdout, stderr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd, err := c.commandForResourceContext(ctx, name, append([]string{operation}, args...)...)
	if err != nil {
		return err
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			err = errors.Join(ctx.Err(), err)
		}
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
	return c.runResourceCommand(context.Background(), name, args[0], args[1:], stdout, stderr)
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

// SetManagedServiceSource persists an operator's explicit source selection.
// Host-tool adoption is preflighted before state changes and never modifies the
// selected executable.
func (c *Controller) SetManagedServiceSource(ctx context.Context, name, mode string) (operatorstate.ResourceSourceChoice, error) {
	manifest, err := c.ResourceManifest(name)
	if err != nil {
		return operatorstate.ResourceSourceChoice{}, err
	}
	service := manifest.ManagedService
	if manifest.Driver != "managed-service" || service == nil || service.HostToolExecutable == "" || service.HostToolVersion == "" {
		return operatorstate.ResourceSourceChoice{}, fmt.Errorf("resource %s does not declare selectable managed-service sources", name)
	}
	switch mode {
	case "managed":
		choice := operatorstate.ResourceSourceChoice{Mode: "managed"}
		if err := c.persistManagedServiceSource(ctx, name, choice); err != nil {
			return choice, err
		}
		return choice, nil
	case "host":

		var path string
		var err error
		if name == openRGBResourceName {
			path, err = resolveOpenRGBAdoptionPath(c.Home, service.HostToolExecutable)
		} else {
			path, err = exec.LookPath(service.HostToolExecutable)
		}
		if err != nil {
			return operatorstate.ResourceSourceChoice{}, fmt.Errorf("compatible host OpenRGB not found on PATH: %w", err)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return operatorstate.ResourceSourceChoice{}, err
		}
		if name == openRGBResourceName {
			digest, err := inspectOpenRGBAdoptedSource(ctx, path)
			if err != nil {
				return operatorstate.ResourceSourceChoice{}, err
			}
			choice := operatorstate.ResourceSourceChoice{Mode: "host", Executable: path, Version: openRGBAppImageVersion, SHA256: digest}
			if err := c.persistManagedServiceSource(ctx, name, choice); err != nil {
				return choice, err
			}
			return choice, nil
		}

		cmd := exec.CommandContext(ctx, path, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return operatorstate.ResourceSourceChoice{}, fmt.Errorf("read host OpenRGB version: %w: %s", err, strings.TrimSpace(string(output)))
		}
		version := strings.TrimSpace(string(output))
		if !compatibleHostToolVersion(version, service.HostToolVersion) {
			return operatorstate.ResourceSourceChoice{}, fmt.Errorf("host OpenRGB version is incompatible: expected %s, found %q", service.HostToolVersion, version)
		}
		digest, err := managedFileSHA256(path)
		if err != nil {
			return operatorstate.ResourceSourceChoice{}, fmt.Errorf("fingerprint adopted OpenRGB executable: %w", err)
		}
		choice := operatorstate.ResourceSourceChoice{Mode: "host", Executable: path, Version: service.HostToolVersion, SHA256: digest}
		if err := c.persistManagedServiceSource(ctx, name, choice); err != nil {
			return choice, err
		}
		return choice, nil
	default:
		return operatorstate.ResourceSourceChoice{}, fmt.Errorf("source must be managed or host")
	}
}

func compatibleHostToolVersion(output, expected string) bool {
	expected = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(expected), "v"))
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !strings.EqualFold(strings.TrimSpace(fields[0]), "OpenRGB") {
			continue
		}
		version := strings.Trim(strings.TrimPrefix(strings.ToLower(fields[1]), "v"), ",;()[]")
		if version == expected {
			return true
		}
	}
	return false
}

func (c *Controller) persistManagedServiceSource(ctx context.Context, name string, choice operatorstate.ResourceSourceChoice) error {
	patch, err := json.Marshal(map[string]any{"resources": map[string]any{name: map[string]any{"source": choice}}})
	if err != nil {
		return err
	}
	_, err = c.operatorStateService().Apply(ctx, patch)
	return err
}

func (c *Controller) ManagedServiceSource(ctx context.Context, name string) (operatorstate.ResourceSourceChoice, bool, error) {
	doc, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return operatorstate.ResourceSourceChoice{}, false, err
	}
	resource, ok := doc.Resources[name]
	if !ok || resource.Source == nil {
		return operatorstate.ResourceSourceChoice{}, false, nil
	}
	return *resource.Source, true, nil
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
	slices.Sort(names)
	return names, nil
}

// ResourceRequired reports whether the project configuration marks a resource
// as semantically required. This is distinct from the operator's enabled
// choice: an enabled resource may still be an optional capability whose
// absence has a documented fallback.
func (c *Controller) ResourceRequired(name string) (bool, error) {
	entries, err := c.readConfigEntries()
	if err != nil {
		return false, err
	}
	return entries[name].Required, nil
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
		DiscoverReportFn: func() (scenario.Report[catalogpkg.Resource], error) {
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
		StartupTimeoutFn: func(manifest ResourceManifest) time.Duration {
			seconds := managedServiceStartTimeoutSeconds(c, manifest)
			if seconds <= 0 {
				seconds = 60
			}
			if uint64(seconds) > uint64((1<<63-1)/time.Second) {
				return 0
			}
			return tuning.ManagedServiceConfiguredTimeout(time.Duration(seconds) * time.Second)
		},
		DriverRunFn: func(ctx context.Context, item catalogpkg.Resource, manifest ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
			driver, err := driverForManifest(manifest)
			if err != nil {
				return err
			}
			run := func(ctx context.Context) error {
				return driver.Run(ctx, c, item, manifest, operation, args, stdout, stderr)
			}
			if manifest.Driver == "managed-service" {
				return run(ctx)
			} // admission follows that driver's lifecycle lock
			return c.withResourceCapacity(ctx, item.Name, operation, run)
		},
		RunResourceCommandFn: func(ctx context.Context, name, operation string, args []string, stdout, stderr io.Writer) error {
			return c.withResourceCapacity(ctx, name, operation, func(ctx context.Context) error {
				return c.runResourceCommand(ctx, name, operation, args, stdout, stderr)
			})
		},
		CommandForResourceFn: c.commandForResource,
		RunCommandFn: func(ctx context.Context, cmd *exec.Cmd) resourcecontrol.CommandResult {
			result := runCommandResource(ctx, cmd)
			return resourcecontrol.CommandResult{Output: result.output, Err: result.err}
		},
	}
}

func (c *Controller) Status(name string, fast bool) (Status, error) {
	status, err := c.resourceControl().Status(name, fast)
	if err != nil {
		return status, err
	}
	return c.withEffectiveCapacity(status)
}

func (c *Controller) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	report, err := c.ListStatusesReport(fast, onlyEnabled)
	return report.Items, err
}

func (c *Controller) ListStatusesReport(fast bool, onlyEnabled bool) (StatusReport, error) {
	report, err := c.resourceControl().ListStatusesReport(fast, onlyEnabled)
	if err != nil {
		return report, err
	}
	for index := range report.Items {
		report.Items[index], err = c.withEffectiveCapacity(report.Items[index])
		if err != nil {
			return report, err
		}
	}
	return report, nil
}

func (c *Controller) Run(name string, args []string, stdout, stderr io.Writer) error {
	return c.RunContext(context.Background(), name, args, stdout, stderr)
}

// RunContext preserves the caller's cancellation through single-resource control.
func (c *Controller) RunContext(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.resourceControl().RunContext(ctx, name, args, stdout, stderr)
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
	return c.commandForResourceContext(context.Background(), name, args...)
}

func (c *Controller) commandForResourceContext(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if path, ok := c.resolveCLIPath(name); ok {
		env := resourceEnvForResource(c.Root, c.Home, name)
		var runAsUser string
		// Resource CLIs need the same provider-selected, non-secret runtime
		// context as lifecycle operations. In particular, a managed-service
		// client must not fall back to a legacy Docker adapter simply because it
		// was invoked through `vrooli resource` rather than by the supervisor.
		if manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json")); err == nil {
			env, err = resourceEnvForManifest(c.Root, c.Home, manifest)
			if err != nil {
				return nil, err
			}
			if manifest.ManagedService == nil {
				return shell.Command(shell.Spec{
					Context: ctx,
					Name:    path,
					Args:    args,
					Dir:     c.Root,
					Env:     env,
				}), nil
			}
			for _, port := range manifest.Ports {
				if port.Host > 0 {
					env = values.SetEnv(env, resourcedeployment.ServicePortEnvironmentName(port.Name), fmt.Sprintf("%d", port.Host))
				}
			}
			for key, value := range manifest.ManagedService.Environment {
				env = values.SetEnv(env, key, value)
			}
			env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
			if artifact, err := managedServiceArtifactPath(c, manifest); err == nil {
				env = values.SetEnv(env, "VROOLI_MANAGED_SERVICE_ARTIFACT", artifact)
			}
			if userName := strings.TrimSpace(manifest.ManagedService.RunAsUser); userName != "" && os.Geteuid() == 0 {
				runAsUser = userName
			}
		}
		// Keep privileged target execution safe even when a sealed release has
		// an older or temporarily unreadable declaration. PostgreSQL is the
		// current root-target managed service and has an explicit OS requirement.
		if runAsUser == "" && os.Geteuid() == 0 && os.Getenv("VROOLI_CLI_ARTIFACT_MODE") == "1" && strings.TrimSpace(name) == "postgres" {
			runAsUser = "postgres"
		}
		if runAsUser != "" && os.Geteuid() == 0 {
			return shell.Command(shell.Spec{Context: ctx, Name: "runuser", Args: append([]string{"--preserve-environment", "--user", runAsUser, "--", path}, args...), Dir: c.Root, Env: env}), nil
		}
		return shell.Command(shell.Spec{
			Context: ctx,
			Name:    path,
			Args:    args,
			Dir:     c.Root,
			Env:     env,
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
	env = values.SetEnv(env, buildinfo.SourceRootFallbackEnvVar, root)
	if strings.TrimSpace(home) != "" {
		env = values.SetEnv(env, "HOME", home)
	}
	return env
}

func resourceEnvForManifest(root, home string, manifest ResourceManifest) ([]string, error) {
	env := resourceEnvForResource(root, home, manifest.Name)
	report, err := resourceenv.ResolveResource(root, home, manifest.Name, resourceenv.ResolveOptions{})
	if err != nil {
		return nil, err
	}
	for key, value := range report.Values {
		env = values.SetEnv(env, key, value)
	}
	return env, nil
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
	env = values.SetEnv(env, "RESOURCE_ROOT", filepath.Join(root, "resources", resourceName))
	env = values.SetEnv(env, "RESOURCE_CONFIG_DIR", paths.ConfigDir)
	env = values.SetEnv(env, "RESOURCE_DATA_DIR", paths.DataDir)
	env = values.SetEnv(env, "RESOURCE_CACHE_DIR", paths.CacheDir)
	env = values.SetEnv(env, "RESOURCE_LOGS_DIR", paths.LogsDir)
	env = values.SetEnv(env, "RESOURCE_STATE_DIR", paths.StateDir)
	if lockPath, err := artifactlock.LockPath("resource-data:" + resourceName); err == nil {
		env = values.SetEnv(env, "RESOURCE_DATA_LOCK_PATH", lockPath)
	}
	return env
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

// withResourceCapacity is shared by standalone, batch and scenario callers.
// Managed services call it only after acquiring their lifecycle owner lock.
func (c *Controller) withResourceCapacity(ctx context.Context, name, operation string, run func(context.Context) error) error {
	if operation != "start" && operation != "restart" {
		return run(ctx)
	}
	admit := c.admitCapacity
	if admit == nil {
		admit = capacity.WithResourceAdmission
	}
	return admit(ctx, capacity.AdmitOptions{Root: c.Root, ResourceName: name, NoReclaim: c.NoReclaim, Log: os.Stderr}, run)
}
