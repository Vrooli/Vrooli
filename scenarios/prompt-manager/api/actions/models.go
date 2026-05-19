// Package actions owns the Action API/domain layer.
//
// DOC: docs/concepts/ACTIONS.md
package actions

import "prompt-manager/store"

type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckWarning CheckStatus = "warning"
	CheckFailed  CheckStatus = "failed"
)

type Check struct {
	Code    string      `json:"code"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
	Path    string      `json:"path,omitempty"`
}

type ValidationResponse struct {
	ActionID string             `json:"actionId"`
	Valid    bool               `json:"valid"`
	Runnable bool               `json:"runnable"`
	// Unvalidated is true when the action references a command whose
	// safety properties (effect, permissions, run-eligibility) are not yet
	// declared by the owning scenario's cli/manifest.json. The action runs
	// but consumers should surface this as an info-level "unvalidated" flag.
	Unvalidated bool               `json:"unvalidated,omitempty"`
	// RequiresConfirmation mirrors the resolved command's confirmation
	// requirement so callers (CLI/UI) can decide whether to prompt before
	// invoking. True for any manifest command with governance.effect ==
	// destructive (default) or governance.requires_confirmation == true.
	RequiresConfirmation bool               `json:"requiresConfirmation,omitempty"`
	Status               string             `json:"status"`
	Command     *CommandResolution `json:"command,omitempty"`
	Checks      []Check            `json:"checks"`
	Action      *store.Action      `json:"action,omitempty"`
}

type ListFilters struct {
	Pack   string
	Status string
	Owner  string
	Tag    string
}

type CreateRequest struct {
	store.Action
	Pack string `json:"pack,omitempty"`
}

type MutationResponse struct {
	Action     *store.Action      `json:"action"`
	Validation ValidationResponse `json:"validation"`
}

type RunRequest struct {
	Input  map[string]any `json:"input,omitempty"`
	DryRun bool           `json:"dryRun,omitempty"`
}

type RunStatus string

const (
	RunStatusDryRun    RunStatus = "dry-run"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusTimedOut  RunStatus = "timed-out"
	RunStatusRejected  RunStatus = "rejected"
	RunStatusThrottled RunStatus = "throttled"
)

type RunResponse struct {
	ActionID        string             `json:"actionId"`
	Status          RunStatus          `json:"status"`
	ExitCode        *int               `json:"exitCode,omitempty"`
	DurationMs      int64              `json:"durationMs"`
	Argv            []string           `json:"argv,omitempty"`
	Stdout          string             `json:"stdout,omitempty"`
	Stderr          string             `json:"stderr,omitempty"`
	StdoutTruncated bool               `json:"stdoutTruncated,omitempty"`
	StderrTruncated bool               `json:"stderrTruncated,omitempty"`
	Output          map[string]any     `json:"output,omitempty"`
	Validation      ValidationResponse `json:"validation"`
	Error           string             `json:"error,omitempty"`
}

type CommandCertainty string

const (
	CertaintyNone      CommandCertainty = "none"
	CertaintyOwnerOnly CommandCertainty = "owner-only"
	CertaintyCommand   CommandCertainty = "command"
	CertaintyOperation CommandCertainty = "operation"
)

type CommandEffect string

const (
	EffectRead        CommandEffect = "read"
	EffectWrite       CommandEffect = "write"
	EffectDestructive CommandEffect = "destructive"
	EffectAdmin       CommandEffect = "admin"
)

type CommandOwner struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type CommandResolution struct {
	Certainty            CommandCertainty `json:"certainty"`
	Owner                CommandOwner     `json:"owner"`
	Target               string           `json:"target"`
	CommandPath          []string         `json:"commandPath,omitempty"`
	Effect               CommandEffect    `json:"effect,omitempty"`
	Permissions          []string         `json:"permissions,omitempty"`
	RunSurfaces          []string         `json:"runSurfaces,omitempty"`
	RequiresConfirmation bool             `json:"requiresConfirmation,omitempty"`
	Message              string           `json:"message,omitempty"`
}
