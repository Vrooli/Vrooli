package main

import "time"

type PrecommitConfig struct {
	Enabled          bool                `json:"enabled"`
	Command          string              `json:"command"`
	WorkingDirectory string              `json:"working_directory"`
	TimeoutSeconds   int                 `json:"timeout_seconds"`
	RunBeforeCommit  bool                `json:"run_before_commit"`
	AllowOverride    bool                `json:"allow_override"`
	LastResult       *PrecommitRunResult `json:"last_result,omitempty"`
	Hook             *PrecommitHookState `json:"hook,omitempty"`
}

type PrecommitHookState struct {
	Status              string    `json:"status"`
	Reason              string    `json:"reason,omitempty"`
	ExistingKind        string    `json:"existing_kind,omitempty"`
	ExistingHookPreview string    `json:"existing_hook_preview,omitempty"`
	Path                string    `json:"path,omitempty"`
	HooksPath           string    `json:"hooks_path,omitempty"`
	InstalledAt         time.Time `json:"installed_at,omitempty"`
}

type PrecommitRunRequest struct {
	Command          string `json:"command,omitempty"`
	WorkingDirectory string `json:"working_directory,omitempty"`
	TimeoutSeconds   int    `json:"timeout_seconds,omitempty"`
}

type PrecommitRunResult struct {
	Status          string    `json:"status"`
	Command         string    `json:"command,omitempty"`
	ExitCode        int       `json:"exit_code"`
	Summary         string    `json:"summary"`
	Stdout          string    `json:"stdout,omitempty"`
	Stderr          string    `json:"stderr,omitempty"`
	DurationMs      int64     `json:"duration_ms"`
	OverrideAllowed bool      `json:"override_allowed"`
	Timestamp       time.Time `json:"timestamp"`
}

type PrecommitRunResponse struct {
	Success bool               `json:"success"`
	Result  PrecommitRunResult `json:"result"`
}
