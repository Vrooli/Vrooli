package livedesktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ActionExecutor executes a desktop control action against a session.
type ActionExecutor interface {
	Execute(ctx context.Context, session *Session, svc *Service, params json.RawMessage) (*ActionResult, error)
}

// ActionResult is the standardized response from any control action.
type ActionResult struct {
	Status  string         `json:"status"`
	Data    map[string]any `json:"data,omitempty"`
	Message string         `json:"message,omitempty"`
}

// ShellFunc abstracts shell command execution for testability.
type ShellFunc func(ctx context.Context, env []string, name string, args ...string) (stdout []byte, err error)

// defaultShell executes a command using exec.CommandContext.
func defaultShell(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// actionRegistry maps action names to their executors.
var actionRegistry = map[string]ActionExecutor{}

func init() {
	// App control
	actionRegistry["launch_app"] = &LaunchAppAction{}
	actionRegistry["quit_app"] = &QuitAppAction{}
	actionRegistry["screenshot"] = &ScreenshotAction{}
	actionRegistry["start_recording"] = &StartRecordingAction{}
	actionRegistry["stop_recording"] = &StopRecordingAction{}

	// Environment
	actionRegistry["offline_mode"] = &OfflineModeAction{}
	actionRegistry["slow_connection"] = &SlowConnectionAction{}
	actionRegistry["inject_env"] = &InjectEnvAction{}
	actionRegistry["resize_display"] = &ResizeDisplayAction{}

	// Advanced
	actionRegistry["clipboard_read"] = &ClipboardReadAction{}
	actionRegistry["clipboard_write"] = &ClipboardWriteAction{}
	actionRegistry["dark_mode"] = &DarkModeAction{}
	actionRegistry["locale"] = &LocaleAction{}
}

// lookupAction returns the executor for a given action name.
func lookupAction(name string) (ActionExecutor, error) {
	executor, ok := actionRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	return executor, nil
}
