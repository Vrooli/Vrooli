package resources

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

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

func (c *Controller) compatService() *compatpkg.Service {
	return compatpkg.New(c.Root, c.Home, lookPathResourceFn)
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
	return c.resourceControl().StatusForResource(item, fast)
}

func (c *Controller) resolveCLIPath(name string) (string, bool) {
	return c.compatService().ResolveCLIPath(name)
}

func (c *Controller) commandForResource(name string, args ...string) (*exec.Cmd, error) {
	return c.compatService().CommandForResource(name, args...)
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
