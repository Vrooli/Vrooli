package exec

import (
	"fmt"
	"os"
	"path/filepath"
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

// BuildBwrapArgs constructs the bubblewrap argument list. Used by
// BuildExecCommand and by interactive-session callers that wrap the
// resulting argv themselves.
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

	// Network isolation depends on isolation level.
	if cfg.IsolationLevel == IsolationVrooliAware {
		// vrooli-aware: leave network namespace alone (allow localhost).
	} else if !cfg.AllowNetwork {
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
	if s.HomeMergedDir != "" {
		if hostHome := os.Getenv("HOME"); hostHome != "" && filepath.IsAbs(hostHome) {
			addDirHierarchy(&args, hostHome)
			args = append(args, "--bind", s.HomeMergedDir, filepath.Clean(hostHome))
		}
	}

	// Optional: mirror the workspace at the project's host path so prompts
	// using host-absolute paths work. Toggled via env var.
	if shouldMirrorProjectRoot() && s.ProjectRoot != "" && filepath.IsAbs(s.ProjectRoot) {
		addDirHierarchy(&args, s.ProjectRoot)
		args = append(args, "--bind", s.MergedDir, filepath.Clean(s.ProjectRoot))
	}

	// Lower layer mounted read-only as /workspace-readonly.
	args = append(args, "--ro-bind", s.LowerDir, "/workspace-readonly")

	// Essential system directories (read-only).
	args = append(args, "--ro-bind", "/usr", "/usr")
	args = append(args, "--ro-bind", "/lib", "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		args = append(args, "--ro-bind", "/lib64", "/lib64")
	}
	args = append(args, "--ro-bind", "/bin", "/bin")

	// Read-only /etc essentials.
	args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
	args = append(args, "--ro-bind", "/etc/hosts", "/etc/hosts")
	args = append(args, "--ro-bind", "/etc/passwd", "/etc/passwd")
	args = append(args, "--ro-bind", "/etc/group", "/etc/group")

	// Vrooli-aware preset adds VROOLI_ROOT bind.
	if cfg.IsolationLevel == IsolationVrooliAware {
		addVrooliAwareBinds(&args)
	}

	args = append(args, "--proc", "/proc")

	if cfg.AllowDevices {
		args = append(args, "--dev", "/dev")
	} else {
		args = append(args, "--dev", "/dev")
	}

	args = append(args, "--tmpfs", "/tmp")

	for _, path := range cfg.ReadOnlyPaths {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--ro-bind", path, path)
		}
	}

	for _, path := range cfg.ReadWritePaths {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--bind", path, path)
		}
	}

	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = "/workspace"
	}
	args = append(args, "--chdir", workDir)

	// Separator between bwrap args and command.
	args = append(args, "--")
	return args
}

func shouldMirrorProjectRoot() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

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

// addVrooliAwareBinds adds the VROOLI_ROOT bind. Most agent-config
// visibility (claude/codex configs, etc.) flows through the per-sandbox
// HOME overlay set up by Driver.Mount and bound at the host $HOME path
// inside the namespace by BuildBwrapArgs. This function only handles the
// VROOLI_ROOT entry, which is independent of $HOME.
func addVrooliAwareBinds(args *[]string) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot != "" {
		if _, err := os.Stat(vrooliRoot); err == nil {
			*args = append(*args, "--ro-bind", vrooliRoot, "/vrooli")
		}
	}
}
