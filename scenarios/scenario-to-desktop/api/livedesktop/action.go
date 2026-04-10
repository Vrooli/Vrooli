package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
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
