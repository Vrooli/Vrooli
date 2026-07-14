// Package exec runs commands inside sandboxes. It is the single canonical
// implementation of process isolation for all driver types; drivers no
// longer implement Exec/StartProcess themselves. ContainmentLevel-keyed
// dispatch (buildStartOpts) selects the OS containment backend that
// enforces how strongly the process is isolated.
package exec

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrIsolationProfileRequired is returned when ApplyIsolationProfile is
// called with a nil profile. Callers must always look up a profile by ID
// (typically via config.ProfileStore.Get) and surface a 400-level error
// to the user when the lookup fails — silent fallback to a default
// preset is the bug class this contract eliminates.
var ErrIsolationProfileRequired = errors.New("isolation profile is required")

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
// for continuity with prior driver code; under ContainmentNone (copy
// driver) the bwrap-specific fields are simply ignored.
//
// Construction order at every call site:
//
//	cfg := DefaultBwrapConfig()
//	CaptureEnv().ApplyTo(&cfg)              // host env -> HostHome, MirrorProjectRoot, …
//	if err := ApplyIsolationProfile(&cfg, profile); err != nil { … }
//
// After that, BuildBwrapArgs is a pure function of (sandbox, cfg).
type BwrapConfig struct {
	AllowNetwork bool
	AllowDevices bool
	SharePID     bool
	Hostname     string

	ResourceLimits ResourceLimits

	Env map[string]string

	// ReadOnlyBinds maps host path -> sandbox path (read-only).
	// ReadWriteBinds maps host path -> sandbox path (read-write).
	// Both are populated by ApplyIsolationProfile from the active profile.
	ReadOnlyBinds  map[string]string
	ReadWriteBinds map[string]string

	WorkingDir string

	// HostHome is the host-side $HOME, used to bind the per-sandbox
	// HOME overlay at the same path inside the namespace. Set by
	// CaptureEnv().ApplyTo. Empty disables the home bind.
	HostHome string

	// MirrorProjectRoot, when true, also binds the merged dir at
	// s.ProjectRoot inside the namespace so prompts using host-absolute
	// paths resolve. Toggled via the WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT
	// env var; captured via CaptureEnv().ApplyTo.
	MirrorProjectRoot bool

	// StdoutWriter / StderrWriter receive the process's stdout/stderr.
	// Required for StartProcess; Exec uses its own buffers.
	StdoutWriter io.Writer
	StderrWriter io.Writer

	// StdinReader, if non-nil, is wired to the process's stdin pipe.
	StdinReader io.Reader

	// OnExit, if non-nil, fires exactly once after the spawned process exits
	// for a background-started process. Dispatched from a goroutine
	// owned by the exec package; callers must not assume synchronisation
	// with StartProcess returning.
	OnExit func(exitCode int, signal int, oomKilled bool)
}

// DefaultBwrapConfig returns the secure default configuration. The
// returned config requires a subsequent ApplyIsolationProfile call to be
// usable for execution; the empty bind maps are intentionally
// minimalist so missing profile-application is a fail-closed signal.
func DefaultBwrapConfig() BwrapConfig {
	return BwrapConfig{
		AllowNetwork:   false,
		AllowDevices:   false,
		SharePID:       false,
		Hostname:       "sandbox",
		Env:            map[string]string{},
		ReadOnlyBinds:  map[string]string{},
		ReadWriteBinds: map[string]string{},
	}
}

// EnvSnapshot captures the host-side env vars that influence sandbox
// shape. Centralising the os.Getenv reads here keeps BuildBwrapArgs and
// ApplyIsolationProfile pure-of-process-env, which is the precondition
// for golden-testing the argv contract.
type EnvSnapshot struct {
	Home              string
	User              string
	VrooliRoot        string
	MirrorProjectRoot bool
	// extras carries any other env vars referenced by profile env values
	// via $VAR placeholders (e.g. VROOLI_ENV). Captured at request time.
	extras map[string]string
}

// CaptureEnv reads the relevant host env vars into a snapshot.
func CaptureEnv() EnvSnapshot {
	extras := map[string]string{}
	for _, k := range []string{
		"VROOLI_ENV",
		"VROOLI_LOG_LEVEL",
		"API_MANAGER_URL",
		"SCENARIO_REGISTRY_URL",
		"RESOURCE_REGISTRY_URL",
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
	} {
		if v := os.Getenv(k); v != "" {
			extras[k] = v
		}
	}
	return EnvSnapshot{
		Home:              os.Getenv("HOME"),
		User:              os.Getenv("USER"),
		VrooliRoot:        os.Getenv("VROOLI_ROOT"),
		MirrorProjectRoot: parseBoolEnv(os.Getenv("WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT")),
		extras:            extras,
	}
}

// ApplyTo populates env-derived fields on cfg. Idempotent.
func (e EnvSnapshot) ApplyTo(cfg *BwrapConfig) {
	cfg.HostHome = e.Home
	cfg.MirrorProjectRoot = e.MirrorProjectRoot
}

func parseBoolEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
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

// ApplyIsolationProfile composes the isolation profile into cfg. This is
// the single source of truth for isolation: there is no preset fallback
// path — callers that don't have a profile to apply must surface a 400.
//
// Placeholder handling:
//   - $HOME, $USER, $VROOLI_ROOT in bind sources expand from the snapshot
//     captured by CaptureEnv (read off cfg.HostHome / os.Getenv).
//   - Source paths whose host-side existence is not visible at apply time
//     are silently dropped from the bind map (e.g. /lib64 on systems
//     without it). BuildBwrapArgs trusts the resulting map.
//   - Env values support $VAR expansion from the captured sandbox
//     environment. PATH is composed instead of overwritten so profiles can
//     declare the policy baseline while callers intentionally add tools.
func ApplyIsolationProfile(cfg *BwrapConfig, profile *IsolationProfile) error {
	if profile == nil {
		return ErrIsolationProfileRequired
	}

	switch profile.NetworkAccess {
	case "none":
		cfg.AllowNetwork = false
	case "localhost", "full":
		// Network control is binary (bwrap: --unshare-net or nothing;
		// Seatbelt: deny network* or nothing), so "localhost" grants
		// unrestricted network exactly like "full". The vocabulary name
		// driver.EnforcementNetworkLoopbackOnly stays unclaimed until a
		// backend can actually restrict egress to loopback
		// (knw-1784006975589682125).
		cfg.AllowNetwork = true
	}

	if profile.Hostname != "" {
		cfg.Hostname = profile.Hostname
	}

	home := cfg.HostHome
	if home == "" {
		home = os.Getenv("HOME")
	}
	// A non-absolute $HOME would expand into a workspace-relative path
	// (e.g. HOME=.home). Combined with `--chdir <workspace>`, every
	// $HOME-relative write — Go build cache, vrooli CLI metrics, tool
	// config — then lands inside the tracked workspace as
	// `<workspace>/.home` instead of a real home. BuildBwrapArgs already
	// refuses to bind a relative HostHome (see args.go); mirror that guard
	// here so a relative HOME never reaches the sandbox env or bind
	// expansions either. Dropping to "" makes $HOME expand to empty, which
	// well-behaved tools resolve via /etc/passwd rather than the workspace.
	if home != "" && !filepath.IsAbs(home) {
		home = ""
	}
	user := os.Getenv("USER")
	vrooliRoot := os.Getenv("VROOLI_ROOT")

	for src, dst := range profile.ReadOnlyBinds {
		if expanded, ok := expandBindSource(src, home, user, vrooliRoot); ok {
			cfg.ReadOnlyBinds[expanded] = expandBindDest(dst, expanded)
		}
	}
	for src, dst := range profile.ReadWriteBinds {
		if expanded, ok := expandBindSource(src, home, user, vrooliRoot); ok {
			cfg.ReadWriteBinds[expanded] = expandBindDest(dst, expanded)
		}
	}

	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	for k, v := range profile.Environment {
		expanded := expandEnvValue(v, home, user, vrooliRoot)
		if k == "PATH" {
			cfg.Env[k] = mergePathLists(expanded, cfg.Env[k])
			continue
		}
		cfg.Env[k] = expanded
	}

	return nil
}

func expandEnvValue(value, home, user, vrooliRoot string) string {
	return os.Expand(value, func(key string) string {
		switch key {
		case "HOME":
			return home
		case "USER":
			return user
		case "VROOLI_ROOT":
			return vrooliRoot
		default:
			return os.Getenv(key)
		}
	})
}

func mergePathLists(primary, secondary string) string {
	seen := map[string]bool{}
	entries := make([]string, 0)
	add := func(pathList string) {
		for _, entry := range strings.Split(pathList, ":") {
			entry = strings.TrimSpace(entry)
			if entry == "" || seen[entry] {
				continue
			}
			seen[entry] = true
			entries = append(entries, entry)
		}
	}
	add(primary)
	add(secondary)
	return strings.Join(entries, ":")
}

// expandBindSource resolves $-placeholders in a profile bind source and
// verifies the resulting host path exists. Returns (expanded, true) on
// success and ("", false) when expansion fails (placeholder not set) or
// the path is absent on this host. Profile binds are best-effort by
// design: a missing /lib64 should not block sandbox creation.
func expandBindSource(src, home, user, vrooliRoot string) (string, bool) {
	if src == "" {
		return "", false
	}
	s := src
	s = strings.ReplaceAll(s, "$HOME", home)
	s = strings.ReplaceAll(s, "$USER", user)
	s = strings.ReplaceAll(s, "$VROOLI_ROOT", vrooliRoot)
	if strings.Contains(s, "$") {
		return "", false
	}
	if _, err := os.Stat(s); err != nil {
		return "", false
	}
	return s, true
}

// expandBindDest mirrors expandBindSource but does not stat. An empty
// destination defaults to "same as source" — preserving the
// "host:host"-style bind shape that BuildBwrapArgs emits.
func expandBindDest(dst, expandedSrc string) string {
	if dst == "" {
		return expandedSrc
	}
	d := dst
	// $HOME / $USER / $VROOLI_ROOT in destinations are not currently
	// used by builtin profiles; expand them anyway for forward-compat.
	d = os.ExpandEnv(d)
	return d
}
