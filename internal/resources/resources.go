package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
	compatpkg "github.com/vrooli/vrooli/internal/resources/compat"
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
)

type Controller struct {
	Root string
	Home string
}

type Error = vroolierr.Error

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
		Root: filepath.Clean(root),
		Home: filepath.Clean(home),
	}
}

func (c *Controller) Discover() ([]Resource, error) {
	deprecated, err := c.deprecatedNameSet()
	if err != nil {
		return nil, err
	}
	return c.catalogService().Discover(catalogpkg.DiscoverOptions{
		DeprecatedNames: deprecated,
		ResolveCLIPath:  c.resolveCLIPath,
	})
}

func (c *Controller) discoverResource(name string) (*Resource, error) {
	deprecated, err := c.deprecatedNameSet()
	if err != nil {
		return nil, err
	}
	return c.catalogService().DiscoverOne(name, catalogpkg.DiscoverOptions{
		DeprecatedNames: deprecated,
		ResolveCLIPath:  c.resolveCLIPath,
	})
}

func (c *Controller) runLegacyResourceCommand(name, operation string, args []string, stdout, stderr io.Writer) error {
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

func (c *Controller) SetEnabled(name string, enabled bool) error {
	configPath := filepath.Join(c.Root, filepath.FromSlash(resourceConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	dependencies := ensureObject(payload, "dependencies")
	resources := ensureObject(dependencies, "resources")
	entryValue, ok := resources[name]
	var entry map[string]any
	if ok {
		if typed, typedOK := entryValue.(map[string]any); typedOK {
			entry = typed
		}
	}
	if entry == nil {
		entry = map[string]any{}
	}
	entry["enabled"] = enabled
	resources[name] = entry

	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	updated = append(updated, '\n')
	return os.WriteFile(configPath, updated, 0o644)
}

func (c *Controller) readConfigEntries() (map[string]ConfigEntry, error) {
	return c.catalogService().ReadConfigEntries()
}

func (c *Controller) filesystemNames() ([]string, error) {
	return c.catalogService().FilesystemNames()
}

func (c *Controller) manifestNames() ([]string, error) {
	return c.catalogService().ManifestNames()
}

func (c *Controller) statusForResource(item Resource, fast bool) (Status, error) {
	if item.ManifestPath != "" && item.ControlMode == "manifest-native" {
		manifest, err := c.loadResourceManifest(item.ManifestPath)
		if err != nil {
			return Status{}, err
		}
		driver, err := driverForManifest(manifest)
		if err != nil {
			return Status{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return driver.Status(ctx, c, item, manifest, fast)
	}
	status := Status{
		Resource:   item,
		Installed:  item.Exists || item.HasCLI || item.HasScript,
		Running:    false,
		StatusCode: StatusCodeOK,
		Message:    "not running",
	}

	cmd, err := c.commandForResource(item.Name, append([]string{"status", "--format", "json"}, fastArgs(fast)...)...)
	if err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "resource status command is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmd.Dir = cmd.Dir
	cmd.Env = cmd.Env

	result := runCommandResource(ctx, cmd)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(result.err, context.DeadlineExceeded) {
		status.StatusCode = StatusCodeTimeout
		status.Message = "resource status command timed out"
		if ctx.Err() != nil {
			status.ProbeError = ctx.Err().Error()
		} else if result.err != nil {
			status.ProbeError = result.err.Error()
		}
		return status, nil
	}

	rawJSON, ok := extractJSONPayload(result.output)
	if !ok {
		if result.err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "resource status command failed"
			status.ProbeError = result.err.Error()
			return status, nil
		}
		status.StatusCode = StatusCodeInvalidStatusPayload
		status.Message = "resource status command did not emit valid JSON"
		status.ProbeError = strings.TrimSpace(string(result.output))
		return status, nil
	}

	status.Raw = rawJSON
	var payload map[string]any
	if err := json.Unmarshal(rawJSON, &payload); err != nil {
		status.StatusCode = StatusCodeInvalidStatusPayload
		status.Message = "resource status command emitted invalid JSON"
		status.ProbeError = err.Error()
		return status, nil
	}

	status.Installed = boolValue(payload["installed"], status.Installed)
	status.Running = boolValue(payload["running"], false)
	status.Message = stringValue(payload["message"])
	if status.Message == "" {
		status.Message = stringValue(payload["status"])
	}
	status.Health = stringValue(payload["health"])
	if healthy, ok := payload["healthy"]; ok {
		value := boolValue(healthy, false)
		status.Healthy = &value
	} else if status.Health != "" {
		value := strings.EqualFold(status.Health, "healthy")
		status.Healthy = &value
	}
	if status.Message == "" {
		switch {
		case status.Running && status.Healthy != nil && *status.Healthy:
			status.Message = "healthy"
		case status.Running:
			status.Message = "running"
		default:
			status.Message = "stopped"
		}
	}
	if result.err != nil {
		status.ProbeError = result.err.Error()
	}

	return status, nil
}

func (c *Controller) resolveCLIPath(name string) (string, bool) {
	return compatpkg.New(c.Root, c.Home, lookPathResourceFn).ResolveCLIPath(name)
}

func (c *Controller) commandForResource(name string, args ...string) (*exec.Cmd, error) {
	return compatpkg.New(c.Root, c.Home, lookPathResourceFn).CommandForResource(name, args...)
}

func resourceEnv(root, home string) []string {
	return compatpkg.ResourceEnv(root, home)
}

func setEnvValue(env []string, key, value string) []string {
	return compatpkg.SetEnvValue(env, key, value)
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

func fastArgs(fast bool) []string {
	if fast {
		return []string{"--fast"}
	}
	return nil
}

func extractJSONPayload(output []byte) (json.RawMessage, bool) {
	return compatpkg.ExtractJSONPayload(output)
}

func boolValue(value any, fallback bool) bool {
	return compatpkg.BoolValue(value, fallback)
}

func stringValue(value any) string {
	return compatpkg.StringValue(value)
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
