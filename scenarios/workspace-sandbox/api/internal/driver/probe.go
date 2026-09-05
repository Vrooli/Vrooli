// Package driver — capability probes.
//
// Every external command invocation in this file goes through
// process.Starter so the syscalls are confined to the canonical seam
// (Round 4 Phase 7). Tests inject FakeStarter from procmocks.
package driver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"workspace-sandbox/internal/process"
)

// canCreateUserNamespace reports whether the host can create unprivileged
// user namespaces. Used by capability probes; not the same thing as
// "currently inside a user namespace" (see InUserNamespace).
func canCreateUserNamespace(starter process.Starter) bool {
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return false
		}
	}
	res, runErr := process.Run(context.Background(), starter, process.StartOpts{
		Path: "unshare",
		Args: []string{"--user", "true"},
	})
	if runErr != nil {
		return false
	}
	return res.Exit.ExitCode == 0
}

// commandExists checks if a command is available in PATH. Routes through
// Starter so tests can inject deterministic LookPath results.
func commandExists(starter process.Starter, name string) bool {
	return process.CommandExists(starter, name)
}

// getCommandVersion runs a command with a version flag and returns the
// first non-empty line of output. Returns "installed" on any error so
// callers always get a non-empty version string.
func getCommandVersion(starter process.Starter, name string, versionFlag string) string {
	res, err := process.Run(context.Background(), starter, process.StartOpts{
		Path: name,
		Args: []string{versionFlag},
	})
	if err != nil || res.Exit.ExitCode != 0 {
		return "installed"
	}
	lines := strings.Split(strings.TrimSpace(string(res.Stdout)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "installed"
}

// fuseAvailable reports whether /dev/fuse exists.
func fuseAvailable() bool {
	_, err := os.Stat("/dev/fuse")
	return err == nil
}

// checkCapSysAdmin returns true when the process has effective CAP_SYS_ADMIN
// in the host namespace. Inside a user namespace `unshare --mount` succeeds
// even though the caller lacks CAP_SYS_ADMIN on the host, so we explicitly
// treat that case as "no": the overlayfs-root flavor must not advertise
// as available when the API is wrapped by `unshare -U -m -r` for the
// overlayfs-userns flavor.
func checkCapSysAdmin(starter process.Starter) bool {
	if InUserNamespace() {
		return false
	}
	res, err := process.Run(context.Background(), starter, process.StartOpts{
		Path: "unshare",
		Args: []string{"--mount", "true"},
	})
	if err != nil {
		return false
	}
	return res.Exit.ExitCode == 0
}

// overlayfsModuleAvailable reports whether the overlayfs filesystem is
// registered with the kernel.
func overlayfsModuleAvailable() bool {
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "overlay")
}

// InUserNamespace reports whether the calling process is inside a Linux
// user namespace (i.e. mapped via /proc/self/uid_map). The init namespace
// shows the identity mapping `0 0 4294967295`; any other mapping (or a
// shorter range) means we're sub-namespaced.
//
// Used by:
//   - Boot-time self-check in main.go (fatal if the overlayfs-userns
//     driver is selected without a userns wrapper).
//   - checkCapSysAdmin to suppress false-positives in `unshare -U -r`.
//   - DriverOptionsResponse.InUserNamespace for the UI.
func InUserNamespace() bool {
	data, err := os.ReadFile("/proc/self/uid_map")
	if err != nil {
		return false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return false
	}
	// Identity mapping inside the host (init) userns is "0 0 4294967295".
	// Any deviation means we're inside a sub-namespace.
	if fields[0] == "0" && fields[1] == "0" && fields[2] == "4294967295" {
		return false
	}
	return true
}

// IsBwrapAvailable checks if bubblewrap is installed and usable.
func IsBwrapAvailable(ctx context.Context, starter process.Starter) (bool, string, error) {
	bwrapPath, err := starter.LookPath("bwrap")
	if err != nil {
		return false, "", fmt.Errorf("bwrap not found in PATH: %w", err)
	}
	res, runErr := process.Run(ctx, starter, process.StartOpts{
		Path: bwrapPath,
		Args: []string{"--version"},
	})
	if runErr != nil {
		return false, bwrapPath, fmt.Errorf("bwrap version check failed: %w", runErr)
	}
	if res.Exit.ExitCode != 0 {
		return false, bwrapPath, fmt.Errorf("bwrap version check returned exit %d", res.Exit.ExitCode)
	}
	return true, strings.TrimSpace(string(res.Stdout)), nil
}

// Containment enforcement vocabulary — platform-neutral names for the
// guarantees a containment backend provides. Reported by
// GetContainmentInfo so callers reason about isolation strength without
// knowing the OS mechanism (bwrap on Linux, Seatbelt on macOS, …).
const (
	EnforcementFilesystemWriteContainment = "filesystem-write-containment"
	EnforcementNetworkDeny                = "network-deny"
	EnforcementPIDNamespace               = "pid-namespace"
	EnforcementPathIllusion               = "path-illusion"

	// EnforcementNetworkLoopbackOnly names the loopback-only network
	// guarantee a "localhost" NetworkAccess profile implies. NO current
	// backend claims it: bwrap and Seatbelt network control is binary
	// (deny-all via --unshare-net / (deny network*), or allow-all), so
	// "localhost" profiles actually run with unrestricted network (see
	// exec.ApplyIsolationProfile). The name exists in the vocabulary so
	// consumers can detect and surface that gap instead of inferring
	// loopback-only from the profile name.
	EnforcementNetworkLoopbackOnly = "network-loopback-only"
)

// ContainmentInfo is the per-OS report of the process-containment backend
// used for isolated execution. Backend is the backend id ("bwrap" on
// Linux, "none" when no OS containment backend is present); Available
// reports whether it can run on this host; Enforcements lists the
// guarantees the backend provides using the platform-neutral vocabulary
// above. Version/Path are backend-specific diagnostics (empty when not
// applicable). Served by the /driver/containment endpoint.
type ContainmentInfo struct {
	Backend      string   `json:"backend"`
	Available    bool     `json:"available"`
	Version      string   `json:"version,omitempty"`
	Path         string   `json:"path,omitempty"`
	Enforcements []string `json:"enforcements"`
	Error        string   `json:"error,omitempty"`
}
