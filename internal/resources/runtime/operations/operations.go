package operations

import "strings"

// Action identifies a control-plane resource operation.
type Action string

const (
	ActionInstall   Action = "install"
	ActionStatus    Action = "status"
	ActionStart     Action = "start"
	ActionRestart   Action = "restart"
	ActionStop      Action = "stop"
	ActionUninstall Action = "uninstall"
	ActionLogs      Action = "logs"
	ActionInvoke    Action = "invoke"
)

// Parse normalizes a raw action name.
func Parse(raw string) Action {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return ActionInvoke
	}
	return Action(normalized)
}

// IsStandard reports whether the action is part of the common resource surface.
func IsStandard(action Action) bool {
	switch action {
	case ActionInstall, ActionStatus, ActionStart, ActionRestart, ActionStop, ActionUninstall, ActionLogs, ActionInvoke:
		return true
	default:
		return false
	}
}
