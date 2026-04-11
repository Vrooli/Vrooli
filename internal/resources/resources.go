package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/control"
)

const resourceConfigPath = ".vrooli/service.json"

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

type ConfigEntry struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type Resource struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Exists     bool        `json:"exists"`
	Registered bool        `json:"registered"`
	Enabled    bool        `json:"enabled"`
	Required   bool        `json:"required"`
	HasCLI     bool        `json:"has_cli"`
	HasScript  bool        `json:"has_script"`
	Config     ConfigEntry `json:"config"`
}

type Status struct {
	Resource   Resource        `json:"resource"`
	Installed  bool            `json:"installed"`
	Running    bool            `json:"running"`
	Healthy    *bool           `json:"healthy,omitempty"`
	Health     string          `json:"health,omitempty"`
	StatusCode string          `json:"status_code,omitempty"`
	Message    string          `json:"message,omitempty"`
	ProbeError string          `json:"probe_error,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

type Controller struct {
	Root string
	Home string
}

type Error struct {
	Code      string
	Resource  string
	Operation string
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	target := strings.TrimSpace(e.Resource)
	if target == "" {
		target = "resource"
	}
	action := strings.TrimSpace(e.Operation)
	switch {
	case e.Err == nil && action == "":
		return target
	case e.Err == nil:
		return fmt.Sprintf("%s %s", action, target)
	case action == "":
		return fmt.Sprintf("%s: %v", target, e.Err)
	default:
		return fmt.Sprintf("%s %s: %v", action, target, e.Err)
	}
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

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
	configEntries, err := c.readConfigEntries()
	if err != nil {
		return nil, err
	}

	filesystemNames, err := c.filesystemNames()
	if err != nil {
		return nil, err
	}

	namesMap := make(map[string]struct{}, len(configEntries)+len(filesystemNames))
	for name := range configEntries {
		namesMap[name] = struct{}{}
	}
	for _, name := range filesystemNames {
		namesMap[name] = struct{}{}
	}

	names := make([]string, 0, len(namesMap))
	for name := range namesMap {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]Resource, 0, len(names))
	for _, name := range names {
		configEntry, registered := configEntries[name]
		path := filepath.Join(c.Root, "resources", name)
		_, statErr := os.Stat(path)
		exists := statErr == nil
		cliPath, hasCLI := c.resolveCLIPath(name)
		scriptPath := filepath.Join(path, "cli.sh")
		_, hasScriptErr := os.Stat(scriptPath)

		item := Resource{
			Name:       name,
			Path:       path,
			Exists:     exists,
			Registered: registered,
			Enabled:    configEntry.Enabled,
			Required:   configEntry.Required,
			HasCLI:     hasCLI && cliPath != "",
			HasScript:  hasScriptErr == nil,
			Config:     configEntry,
		}
		items = append(items, item)
	}

	return items, nil
}

func (c *Controller) Status(name string, fast bool) (Status, error) {
	resources, err := c.Discover()
	if err != nil {
		return Status{}, err
	}

	var item *Resource
	for i := range resources {
		if resources[i].Name == name {
			item = &resources[i]
			break
		}
	}
	if item == nil {
		return Status{}, fmt.Errorf("resource %q not found", name)
	}

	return c.statusForResource(*item, fast)
}

func (c *Controller) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	items, err := c.Discover()
	if err != nil {
		return nil, err
	}

	statuses := make([]Status, 0, len(items))
	for _, item := range items {
		if onlyEnabled && !item.Enabled {
			continue
		}
		status, statusErr := c.statusForResource(item, fast)
		if statusErr != nil {
			status = Status{
				Resource:  item,
				Installed: item.Exists || item.HasCLI || item.HasScript,
				Message:   statusErr.Error(),
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (c *Controller) Run(name string, args []string, stdout, stderr io.Writer) error {
	operation := "invoke"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		operation = args[0]
	}
	cmd, err := c.commandForResource(name, args...)
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
			Err:       err,
		}
	}
	return nil
}

func (c *Controller) StartAll(stdout, stderr io.Writer) (control.StartReport, error) {
	statuses, err := c.ListStatuses(true, true)
	if err != nil {
		return control.StartReport{}, err
	}

	started := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	for _, status := range statuses {
		if !status.Resource.Enabled {
			continue
		}
		bestEffort := status.StatusCode != "" && status.StatusCode != StatusCodeOK
		if status.Running {
			started = append(started, control.Started(status.Resource.Name, "Already running"))
			continue
		}
		if err := c.Run(status.Resource.Name, []string{"start"}, stdout, stderr); err != nil {
			failed = append(failed, control.Failed(status.Resource.Name, err))
			continue
		}
		message := "Started successfully"
		if bestEffort {
			message = "Started successfully after degraded status probe"
		}
		started = append(started, control.Started(status.Resource.Name, message))
	}
	return control.StartReport{
		Started: started,
		Failed:  failed,
		Message: control.StartSummary(len(started), len(failed)),
	}, nil
}

func (c *Controller) StopAll(stdout, stderr io.Writer) (control.StopReport, error) {
	statuses, err := c.ListStatuses(true, false)
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	for _, status := range statuses {
		bestEffort := false
		if !status.Running {
			if status.StatusCode == StatusCodeOK || (!status.Resource.HasCLI && !status.Resource.HasScript) {
				continue
			}
			bestEffort = true
		}
		if err := c.Run(status.Resource.Name, []string{"stop"}, stdout, stderr); err != nil {
			failed = append(failed, control.Failed(status.Resource.Name, err))
			continue
		}
		message := "Stopped successfully"
		if bestEffort {
			message = "Stopped successfully after degraded status probe"
		}
		stopped = append(stopped, control.Stopped(status.Resource.Name, message))
	}
	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: control.StopSummary(len(stopped), len(failed)),
	}, nil
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
	configPath := filepath.Join(c.Root, filepath.FromSlash(resourceConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Dependencies struct {
			Resources map[string]ConfigEntry `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Dependencies.Resources == nil {
		return map[string]ConfigEntry{}, nil
	}
	return payload.Dependencies.Resources, nil
}

func (c *Controller) filesystemNames() ([]string, error) {
	resourceDir := filepath.Join(c.Root, "resources")
	entries, err := os.ReadDir(resourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (c *Controller) statusForResource(item Resource, fast bool) (Status, error) {
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
	if path, err := lookPathResourceFn("resource-" + name); err == nil {
		return path, true
	}
	return "", false
}

func (c *Controller) commandForResource(name string, args ...string) (*exec.Cmd, error) {
	if path, ok := c.resolveCLIPath(name); ok {
		cmd := exec.Command(path, args...)
		cmd.Dir = c.Root
		cmd.Env = resourceEnv(c.Root, c.Home)
		return cmd, nil
	}

	scriptPath := filepath.Join(c.Root, "resources", name, "cli.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)
		cmd.Dir = c.Root
		cmd.Env = resourceEnv(c.Root, c.Home)
		return cmd, nil
	}

	return nil, &Error{
		Code:      ErrorCodeCommandUnavailable,
		Resource:  name,
		Operation: "invoke",
		Err:       fmt.Errorf("no installed CLI or cli.sh"),
	}
}

func resourceEnv(root, home string) []string {
	env := os.Environ()
	env = setEnvValue(env, "VROOLI_ROOT", root)
	env = setEnvValue(env, "APP_ROOT", root)
	if strings.TrimSpace(home) != "" {
		env = setEnvValue(env, "HOME", home)
	}
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

func fastArgs(fast bool) []string {
	if fast {
		return []string{"--fast"}
	}
	return nil
}

func extractJSONPayload(output []byte) (json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, false
	}
	start := bytes.IndexByte(trimmed, '{')
	end := bytes.LastIndexByte(trimmed, '}')
	if start < 0 || end < start {
		return nil, false
	}
	candidate := bytes.TrimSpace(trimmed[start : end+1])
	if !json.Valid(candidate) {
		return nil, false
	}
	return append(json.RawMessage(nil), candidate...), true
}

func boolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "healthy", "running", "installed":
			return true
		case "false", "no", "stopped", "missing":
			return false
		}
	}
	return fallback
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return strings.Trim(strings.TrimSpace(string(data)), `"`)
	}
}
