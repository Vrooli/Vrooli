// Package exec runs commands inside sandboxes. It is the single canonical
// implementation of process isolation for all driver types; drivers no
// longer implement Exec/StartProcess themselves. Mode-keyed dispatch
// (DriverModeFor) chooses how strongly to isolate the process.
package exec

import (
	"io"
	"os"
	"strings"
)

// IsolationLevel selects between the legacy "full" and "vrooli-aware" presets.
// Newer call sites should configure isolation via IsolationProfile (loaded
// from config.ProfileStore); IsolationLevel remains as a default switch
// when no profile is supplied.
type IsolationLevel string

const (
	// IsolationFull is the maximum-isolation preset: only /workspace and
	// basic system paths are visible.
	IsolationFull IsolationLevel = "full"

	// IsolationVrooliAware allows access to Vrooli CLIs and localhost.
	IsolationVrooliAware IsolationLevel = "vrooli-aware"
)

// ResourceLimits configures process resource constraints via prlimit.
// Zero values mean unlimited (no limit applied).
type ResourceLimits struct {
	// MemoryLimitMB sets the maximum address space in megabytes.
	MemoryLimitMB int

	// CPUTimeSec sets the maximum CPU time in seconds.
	CPUTimeSec int

	// MaxProcesses sets the maximum number of child processes.
	MaxProcesses int

	// MaxOpenFiles sets the maximum number of open file descriptors.
	MaxOpenFiles int

	// TimeoutSec sets the wall-clock timeout in seconds. Enforced via
	// context, not prlimit.
	TimeoutSec int
}

// HasLimits returns true if any prlimit-backed limit is set.
func (r ResourceLimits) HasLimits() bool {
	return r.MemoryLimitMB > 0 || r.CPUTimeSec > 0 || r.MaxProcesses > 0 || r.MaxOpenFiles > 0
}

// BwrapConfig configures bubblewrap execution parameters. The name is kept
// for continuity with prior driver code; in ModeNone (copy driver) the
// bwrap-specific fields are simply ignored.
type BwrapConfig struct {
	AllowNetwork  bool
	AllowDevices  bool
	ReadOnlyPaths []string
	ReadWritePaths []string
	Env           map[string]string
	WorkingDir    string
	SharePID      bool
	Hostname      string
	IsolationLevel IsolationLevel
	ResourceLimits ResourceLimits

	// StdoutWriter / StderrWriter receive the process's stdout/stderr.
	// Required for StartProcess; Exec uses its own buffers.
	StdoutWriter io.Writer
	StderrWriter io.Writer

	// StdinReader, if non-nil, is wired to the process's stdin pipe.
	StdinReader io.Reader

	// OnExit, if non-nil, fires exactly once after cmd.Wait() returns
	// for a background-started process. Dispatched from a goroutine
	// owned by the exec package; callers must not assume synchronisation
	// with StartProcess returning.
	OnExit func(exitCode int, signal int, oomKilled bool)
}

// DefaultBwrapConfig returns the secure default configuration.
func DefaultBwrapConfig() BwrapConfig {
	return BwrapConfig{
		AllowNetwork:   false,
		AllowDevices:   false,
		SharePID:       false,
		Hostname:       "sandbox",
		IsolationLevel: IsolationFull,
		Env: map[string]string{
			"PATH":         "/usr/local/bin:/usr/bin:/bin",
			"HOME":         "/tmp",
			"SHELL":        "/bin/sh",
			"PROJECT_PATH": "/workspace",
		},
	}
}

// IsolationProfile mirrors config.IsolationProfile for exec-package use.
// Avoids a circular import between the driver layer and the config layer.
type IsolationProfile struct {
	ID             string
	Name           string
	Description    string
	Builtin        bool
	NetworkAccess  string            // "none", "localhost", "full"
	ReadOnlyBinds  map[string]string // host path -> sandbox path
	ReadWriteBinds map[string]string // host path -> sandbox path
	Environment    map[string]string // env var -> value (supports $VAR expansion)
	Hostname       string
}

// ApplyIsolationProfile configures BwrapConfig from a profile. Profile
// binds are appended to the existing ReadOnlyPaths/ReadWritePaths arrays;
// BuildBwrapArgs picks them up.
func ApplyIsolationProfile(cfg *BwrapConfig, profile *IsolationProfile) {
	if profile == nil {
		return
	}

	switch profile.NetworkAccess {
	case "none":
		cfg.AllowNetwork = false
	case "localhost", "full":
		cfg.AllowNetwork = true
	}

	if profile.Hostname != "" {
		cfg.Hostname = profile.Hostname
	}

	for src := range profile.ReadOnlyBinds {
		expandedSrc := expandPathPlaceholders(src)
		if expandedSrc != "" {
			cfg.ReadOnlyPaths = append(cfg.ReadOnlyPaths, expandedSrc)
		}
	}

	for src := range profile.ReadWriteBinds {
		expandedSrc := expandPathPlaceholders(src)
		if expandedSrc != "" {
			cfg.ReadWritePaths = append(cfg.ReadWritePaths, expandedSrc)
		}
	}

	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	for k, v := range profile.Environment {
		cfg.Env[k] = expandEnvPlaceholders(v)
	}
}

// ApplyVrooliAwareConfig augments cfg for the legacy "vrooli-aware" preset.
// Used as a fallback when the ProfileStore does not have a matching profile.
func ApplyVrooliAwareConfig(cfg *BwrapConfig) {
	cfg.IsolationLevel = IsolationVrooliAware
	for k, v := range GetVrooliEnvVars() {
		cfg.Env[k] = v
	}
	cfg.AllowNetwork = true
}

// GetVrooliEnvVars returns environment variables for Vrooli-aware isolation.
// The agent inside the sandbox sees /vrooli mapped to the host VROOLI_ROOT
// via a bwrap bind set up in BuildBwrapArgs.
func GetVrooliEnvVars() map[string]string {
	vars := make(map[string]string)
	if vrooliRoot := os.Getenv("VROOLI_ROOT"); vrooliRoot != "" {
		vars["VROOLI_ROOT"] = "/vrooli"
	}
	envsToCopy := []string{
		"VROOLI_ENV",
		"VROOLI_LOG_LEVEL",
		"API_MANAGER_URL",
		"SCENARIO_REGISTRY_URL",
		"RESOURCE_REGISTRY_URL",
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
	}
	for _, env := range envsToCopy {
		if val := os.Getenv(env); val != "" {
			vars[env] = val
		}
	}
	return vars
}

// expandPathPlaceholders expands $HOME, $USER, $VROOLI_ROOT in paths.
// Returns empty if any placeholder cannot be resolved (skips bind).
func expandPathPlaceholders(path string) string {
	if path == "" {
		return ""
	}
	home := os.Getenv("HOME")
	user := os.Getenv("USER")
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	result := path
	result = strings.ReplaceAll(result, "$HOME", home)
	result = strings.ReplaceAll(result, "$USER", user)
	result = strings.ReplaceAll(result, "$VROOLI_ROOT", vrooliRoot)
	if strings.Contains(result, "$") {
		return ""
	}
	return result
}

func expandEnvPlaceholders(value string) string {
	return os.ExpandEnv(value)
}
