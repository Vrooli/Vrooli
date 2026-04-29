package driver

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
)

// canCreateUserNamespace reports whether the host can create unprivileged
// user namespaces. Used by capability probes; not the same thing as
// "currently inside a user namespace" (see InUserNamespace).
func canCreateUserNamespace() bool {
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return false
		}
	}
	cmd := osexec.Command("unshare", "--user", "true")
	return cmd.Run() == nil
}

// commandExists checks if a command is available in PATH.
func commandExists(name string) bool {
	_, err := osexec.LookPath(name)
	return err == nil
}

// getCommandVersion runs a command with a version flag and returns the first
// non-empty line of output.
func getCommandVersion(name string, versionFlag string) string {
	cmd := osexec.Command(name, versionFlag)
	out, err := cmd.Output()
	if err != nil {
		return "installed"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
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
// treat that case as "no": OverlayfsRoot must not advertise as available
// when the API is wrapped by `unshare -U -m -r` for OverlayfsUserNS.
func checkCapSysAdmin() bool {
	if InUserNamespace() {
		return false
	}
	cmd := osexec.Command("unshare", "--mount", "true")
	return cmd.Run() == nil
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
//   - Boot-time self-check in main.go (fatal if OverlayfsDriver selected
//     without userns wrapper).
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
func IsBwrapAvailable(ctx context.Context) (bool, string, error) {
	bwrapPath, err := osexec.LookPath("bwrap")
	if err != nil {
		return false, "", fmt.Errorf("bwrap not found in PATH: %w", err)
	}
	cmd := osexec.CommandContext(ctx, bwrapPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return false, bwrapPath, fmt.Errorf("bwrap version check failed: %w", err)
	}
	return true, strings.TrimSpace(string(output)), nil
}

// BwrapInfo describes the bwrap installation and namespace capabilities.
type BwrapInfo struct {
	Available            bool   `json:"available"`
	Version              string `json:"version,omitempty"`
	Path                 string `json:"path,omitempty"`
	UserNamespaceEnabled bool   `json:"userNamespaceEnabled"`
	OverlayfsInUserNS    bool   `json:"overlayfsInUserNS"`
	Error                string `json:"error,omitempty"`
}

// GetBwrapInfo returns information about the bwrap installation.
func GetBwrapInfo(ctx context.Context) (*BwrapInfo, error) {
	available, version, err := IsBwrapAvailable(ctx)
	if !available {
		return &BwrapInfo{
			Available: false,
			Error:     err.Error(),
		}, nil
	}
	info := &BwrapInfo{
		Available: true,
		Version:   version,
		Path:      mustExecLookPath("bwrap"),
	}
	if data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil {
		info.UserNamespaceEnabled = strings.TrimSpace(string(data)) == "1"
	}
	info.OverlayfsInUserNS = checkOverlayfsUserNS()
	return info, nil
}

func mustExecLookPath(name string) string {
	path, _ := osexec.LookPath(name)
	return path
}

func checkOverlayfsUserNS() bool {
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "overlay")
}
