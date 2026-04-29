package exec

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"workspace-sandbox/internal/types"
)

// BuildExecCommand builds the full command line for executing in a sandbox
// under bwrap-backed isolation modes (ModeBwrapPreferred, ModeBwrapRequired).
// Returns (executable, args). When ResourceLimits has prlimit-backed entries,
// the returned executable is "prlimit" wrapping bwrap; otherwise it is "bwrap".
func BuildExecCommand(s *types.Sandbox, cfg BwrapConfig, cmd string, cmdArgs ...string) (string, []string) {
	bwrapArgs := BuildBwrapArgs(s, cfg)
	bwrapArgs = append(bwrapArgs, cmd)
	bwrapArgs = append(bwrapArgs, cmdArgs...)

	prlimitArgs := BuildPrlimitArgs(cfg.ResourceLimits)
	if prlimitArgs == nil {
		return "bwrap", bwrapArgs
	}

	full := prlimitArgs
	full = append(full, "bwrap")
	full = append(full, bwrapArgs...)
	return "prlimit", full
}

// BuildPrlimitArgs constructs prlimit arguments for resource limiting.
// Returns nil if no limits are configured.
func BuildPrlimitArgs(limits ResourceLimits) []string {
	if !limits.HasLimits() {
		return nil
	}
	args := []string{}
	if limits.MemoryLimitMB > 0 {
		bytes := int64(limits.MemoryLimitMB) * 1024 * 1024
		args = append(args, fmt.Sprintf("--as=%d", bytes))
	}
	if limits.CPUTimeSec > 0 {
		args = append(args, fmt.Sprintf("--cpu=%d", limits.CPUTimeSec))
	}
	if limits.MaxProcesses > 0 {
		args = append(args, fmt.Sprintf("--nproc=%d", limits.MaxProcesses))
	}
	if limits.MaxOpenFiles > 0 {
		args = append(args, fmt.Sprintf("--nofile=%d", limits.MaxOpenFiles))
	}
	args = append(args, "--")
	return args
}

// BuildBwrapArgs constructs the bubblewrap argument list. Pure function
// of (sandbox, cfg) — every input that varies per request lives in cfg,
// captured by CaptureEnv().ApplyTo and ApplyIsolationProfile at the
// wiring boundary. This is the contract a golden test pins.
func BuildBwrapArgs(s *types.Sandbox, cfg BwrapConfig) []string {
	args := []string{}

	// Unshare namespaces for isolation.
	args = append(args, "--unshare-user")
	args = append(args, "--unshare-ipc")
	args = append(args, "--unshare-uts")
	args = append(args, "--unshare-cgroup")

	if !cfg.SharePID {
		args = append(args, "--unshare-pid")
	}

	if !cfg.AllowNetwork {
		args = append(args, "--unshare-net")
	}

	// Die with parent — ensures the bwrap supervisor exits when the API
	// process holding it does.
	args = append(args, "--die-with-parent")

	if cfg.Hostname != "" {
		args = append(args, "--hostname", cfg.Hostname)
	}

	// Bind the sandbox merged directory as /workspace.
	args = append(args, "--bind", s.MergedDir, "/workspace")

	// Bind the per-sandbox home overlay at the host $HOME path inside the
	// namespace so agent CLIs find their host config (auth tokens, tool
	// caches, etc.) via the overlay's lower layer; writes land in the
	// per-sandbox upper layer that's discarded at sandbox teardown.
	if s.HomeMergedDir != "" && cfg.HostHome != "" && filepath.IsAbs(cfg.HostHome) {
		addDirHierarchy(&args, cfg.HostHome)
		args = append(args, "--bind", s.HomeMergedDir, filepath.Clean(cfg.HostHome))
	}

	// Optional: mirror the workspace at the project's host path so prompts
	// using host-absolute paths work. Set by the wiring layer from
	// WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT.
	if cfg.MirrorProjectRoot && s.ProjectRoot != "" && filepath.IsAbs(s.ProjectRoot) {
		addDirHierarchy(&args, s.ProjectRoot)
		args = append(args, "--bind", s.MergedDir, filepath.Clean(s.ProjectRoot))
	}

	// Lower layer mounted read-only as /workspace-readonly.
	args = append(args, "--ro-bind", s.LowerDir, "/workspace-readonly")

	// Profile-driven binds. Iterated in stable (sorted-by-source) order so
	// the argv contract is deterministic for golden tests.
	for _, src := range sortedKeys(cfg.ReadOnlyBinds) {
		args = append(args, "--ro-bind", src, cfg.ReadOnlyBinds[src])
	}
	for _, src := range sortedKeys(cfg.ReadWriteBinds) {
		args = append(args, "--bind", src, cfg.ReadWriteBinds[src])
	}

	args = append(args, "--proc", "/proc")
	args = append(args, "--dev", "/dev")
	args = append(args, "--tmpfs", "/tmp")

	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = "/workspace"
	}
	args = append(args, "--chdir", workDir)

	// Separator between bwrap args and command.
	args = append(args, "--")
	return args
}

// sortedKeys returns the keys of m in lexicographic order. Used to make
// BuildBwrapArgs's output deterministic regardless of map iteration order.
func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// addDirHierarchy emits --dir entries for every parent of absPath
// (excluding well-known system roots). Required so bwrap can create the
// final mount point inside the namespace when the host-side path is not
// already covered by the system root binds.
func addDirHierarchy(args *[]string, absPath string) {
	clean := filepath.Clean(absPath)
	if clean == "" || clean == "/" || !filepath.IsAbs(clean) {
		return
	}
	switch clean {
	case "/bin", "/sbin", "/usr", "/etc", "/lib", "/lib64", "/proc", "/dev", "/tmp":
		return
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), string(filepath.Separator))
	cur := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur = cur + string(filepath.Separator) + p
		*args = append(*args, "--dir", cur)
	}
}
