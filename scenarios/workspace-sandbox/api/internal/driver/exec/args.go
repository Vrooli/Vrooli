package exec

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// BuildExecCommand builds the full command line for executing in a sandbox
// under the bwrap containment backend (ContainmentPreferred, ContainmentRequired).
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

	// Bind the sandbox merged directory as the agent-visible workspace path.
	args = append(args, "--bind", s.MergedDir, driver.NamespaceWorkspacePath)

	// Bind the per-sandbox home overlay at the host $HOME path inside the
	// namespace so agent CLIs find their host config (auth tokens, tool
	// caches, etc.) via the overlay's lower layer; writes land in the
	// per-sandbox upper layer that's discarded at sandbox teardown.
	if s.HomeMergedDir != "" && cfg.HostHome != "" && filepath.IsAbs(cfg.HostHome) {
		addDirHierarchy(&args, cfg.HostHome)
		args = append(args, "--bind", s.HomeMergedDir, filepath.Clean(cfg.HostHome))

		// Re-bind the merged dir at its own host path, AFTER the home
		// overlay bind so it mounts over it. The merged dir usually lives
		// under $HOME, and overlayfs does not surface submounts of a lower
		// layer — without this bind a leaked host-side merged path resolves
		// through the home overlay to the EMPTY underlying directory
		// instead of failing loudly (root cause of the 2026-07-20
		// empty-workspace incident, sandbox cc371116).
		if filepath.IsAbs(s.MergedDir) {
			addDirHierarchy(&args, s.MergedDir)
			args = append(args, "--bind", s.MergedDir, filepath.Clean(s.MergedDir))
		}
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

	// Mask paths: empty tmpfs over-binds emitted after every other bind so
	// deny beats allow. Hides host state the home overlay would otherwise
	// expose (e.g. unrelated checkouts under $HOME). Sorted for a
	// deterministic argv contract. A mask that would cover the workspace,
	// the merged dir, or the lower layer is skipped — masks restrict what a
	// workload can see beyond its workspace, never the workspace itself.
	masks := append([]string{}, cfg.MaskPaths...)
	sort.Strings(masks)
	for _, mask := range masks {
		if mask == "" || !filepath.IsAbs(mask) {
			continue
		}
		mask = filepath.Clean(mask)
		if pathsOverlap(mask, driver.NamespaceWorkspacePath) ||
			pathsOverlap(mask, s.MergedDir) ||
			pathsOverlap(mask, "/workspace-readonly") {
			continue
		}
		args = append(args, "--tmpfs", mask)
	}

	args = append(args, "--proc", "/proc")
	args = append(args, "--dev", "/dev")
	args = append(args, "--tmpfs", "/tmp")

	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = driver.NamespaceWorkspacePath
	}
	args = append(args, "--chdir", workDir)

	// Separator between bwrap args and command.
	args = append(args, "--")
	return args
}

// pathsOverlap reports whether a and b are equal or one is an ancestor
// directory of the other. Used to refuse masks that would shadow any part
// of the workspace — in either direction: a mask over the workspace root
// hides it entirely, and a mask inside it would hide part of one alias of
// the overlay while other aliases (e.g. /workspace vs the host merged
// path) still show it, which is worse than not masking.
func pathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	return a == b ||
		strings.HasPrefix(b, a+string(filepath.Separator)) ||
		strings.HasPrefix(a, b+string(filepath.Separator))
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
