// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"workspace-sandbox/internal/namespace"
)

// DriverOptionID identifies a specific driver option.
type DriverOptionID string

const (
	DriverOptionOverlayfsUserNS   DriverOptionID = "overlayfs-userns"
	DriverOptionOverlayfsRoot     DriverOptionID = "overlayfs-root"
	DriverOptionFuseOverlayfs     DriverOptionID = "fuse-overlayfs"
	DriverOptionCopy              DriverOptionID = "copy"
)

// Requirement represents a single requirement for a driver option.
type Requirement struct {
	// Name is a human-readable description of the requirement
	Name string `json:"name"`

	// Met indicates whether the requirement is currently satisfied
	Met bool `json:"met"`

	// Current is the current value/state of the requirement (e.g., kernel version)
	Current string `json:"current,omitempty"`

	// HowToFix provides actionable steps to satisfy the requirement
	HowToFix string `json:"howToFix,omitempty"`
}

// DriverOption represents a single driver option with its requirements.
type DriverOption struct {
	// ID is the unique identifier for this option
	ID DriverOptionID `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description explains what this driver does and its trade-offs
	Description string `json:"description"`

	// DirectAccess indicates if the merged directory is accessible outside the API process
	DirectAccess bool `json:"directAccess"`

	// Requirements lists what's needed to use this driver
	Requirements []Requirement `json:"requirements"`

	// Available indicates whether all requirements are met
	Available bool `json:"available"`

	// Recommended indicates if this is the recommended option for the current system
	Recommended bool `json:"recommended,omitempty"`
}

// DriverOptionsResponse is the response from the /driver/options endpoint.
type DriverOptionsResponse struct {
	// OS is the current operating system
	OS string `json:"os"`

	// Kernel is the kernel version (Linux only)
	Kernel string `json:"kernel,omitempty"`

	// CurrentDriver is the ID of the currently active driver
	CurrentDriver DriverOptionID `json:"currentDriver"`

	// InUserNamespace indicates if the API is running inside a user namespace
	InUserNamespace bool `json:"inUserNamespace"`

	// Options lists all available driver options with their requirements
	Options []DriverOption `json:"options"`
}

// GetDriverOptions returns all available driver options with their requirements.
func GetDriverOptions(ctx context.Context, currentDriverType DriverType, inUserNS bool) DriverOptionsResponse {
	resp := DriverOptionsResponse{
		OS:              runtime.GOOS,
		InUserNamespace: inUserNS,
		Options:         make([]DriverOption, 0),
	}

	// Map current driver type to option ID
	switch currentDriverType {
	case DriverTypeOverlayfs:
		if inUserNS {
			resp.CurrentDriver = DriverOptionOverlayfsUserNS
		} else {
			resp.CurrentDriver = DriverOptionOverlayfsRoot
		}
	case DriverTypeCopy:
		resp.CurrentDriver = DriverOptionCopy
	default:
		resp.CurrentDriver = DriverOptionCopy
	}

	if runtime.GOOS == "linux" {
		resp.Kernel = namespace.Check().KernelVersion

		// Option 1: Overlayfs in User Namespace (current default for unprivileged)
		resp.Options = append(resp.Options, buildOverlayfsUserNSOption())

		// Option 2: FUSE Overlayfs
		resp.Options = append(resp.Options, buildFuseOverlayfsOption())

		// Option 3: Overlayfs as Root
		resp.Options = append(resp.Options, buildOverlayfsRootOption())
	}

	// Option 4: Copy Driver (always available)
	resp.Options = append(resp.Options, buildCopyDriverOption())

	// Mark recommended option
	markRecommended(resp.Options)

	return resp
}

// buildOverlayfsUserNSOption checks requirements for overlayfs in user namespace.
func buildOverlayfsUserNSOption() DriverOption {
	opt := DriverOption{
		ID:           DriverOptionOverlayfsUserNS,
		Name:         "Overlayfs (User Namespace)",
		Description:  "Secure unprivileged overlayfs using Linux user namespaces. Best performance, no root required. Mounted files are only accessible via the API (exec endpoint or file operations API).",
		DirectAccess: false,
		Requirements: make([]Requirement, 0),
	}

	nsStatus := namespace.Check()

	// Requirement 1: Linux kernel 5.11+
	kernelOK := namespace.IsKernelAtLeast(5, 11)
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "Linux kernel 5.11+",
		Met:     kernelOK,
		Current: nsStatus.KernelVersion,
		HowToFix: func() string {
			if !kernelOK {
				return "Upgrade to a Linux kernel version 5.11 or later"
			}
			return ""
		}(),
	})

	// Requirement 2: User namespaces enabled
	usernsOK := canCreateUserNamespace()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "User namespaces enabled",
		Met:     usernsOK,
		Current: func() string {
			if usernsOK {
				return "enabled"
			}
			return "disabled"
		}(),
		HowToFix: func() string {
			if !usernsOK {
				return "Enable unprivileged user namespaces: echo 1 | sudo tee /proc/sys/kernel/unprivileged_userns_clone"
			}
			return ""
		}(),
	})

	// Requirement 3: unshare command available
	unshareOK := commandExists("unshare")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "unshare command",
		Met:     unshareOK,
		Current: func() string {
			if unshareOK {
				return "installed"
			}
			return "not found"
		}(),
		HowToFix: func() string {
			if !unshareOK {
				return "sudo apt install util-linux"
			}
			return ""
		}(),
	})

	opt.Available = kernelOK && usernsOK && unshareOK
	return opt
}

// buildFuseOverlayfsOption checks requirements for fuse-overlayfs.
func buildFuseOverlayfsOption() DriverOption {
	opt := DriverOption{
		ID:           DriverOptionFuseOverlayfs,
		Name:         "FUSE Overlayfs",
		Description:  "Unprivileged overlayfs via FUSE. Direct filesystem access without root. Slightly lower performance than kernel overlayfs.",
		DirectAccess: true,
		Requirements: make([]Requirement, 0),
	}

	// Requirement 1: fuse-overlayfs installed
	fuseOverlayfsOK := commandExists("fuse-overlayfs")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "fuse-overlayfs installed",
		Met:     fuseOverlayfsOK,
		Current: func() string {
			if fuseOverlayfsOK {
				return getCommandVersion("fuse-overlayfs", "--version")
			}
			return "not installed"
		}(),
		HowToFix: func() string {
			if !fuseOverlayfsOK {
				return "sudo apt install fuse-overlayfs"
			}
			return ""
		}(),
	})

	// Requirement 2: FUSE available
	fuseOK := fuseAvailable()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "FUSE available",
		Met:     fuseOK,
		Current: func() string {
			if fuseOK {
				return "/dev/fuse exists"
			}
			return "/dev/fuse not found"
		}(),
		HowToFix: func() string {
			if !fuseOK {
				return "sudo modprobe fuse && sudo apt install fuse3"
			}
			return ""
		}(),
	})

	// Requirement 3: fusermount available
	fusermountOK := commandExists("fusermount") || commandExists("fusermount3")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "fusermount command",
		Met:     fusermountOK,
		Current: func() string {
			if fusermountOK {
				return "installed"
			}
			return "not found"
		}(),
		HowToFix: func() string {
			if !fusermountOK {
				return "sudo apt install fuse3"
			}
			return ""
		}(),
	})

	opt.Available = fuseOverlayfsOK && fuseOK && fusermountOK
	return opt
}

// buildOverlayfsRootOption checks requirements for privileged overlayfs.
func buildOverlayfsRootOption() DriverOption {
	opt := DriverOption{
		ID:           DriverOptionOverlayfsRoot,
		Name:         "Overlayfs (Privileged)",
		Description:  "Native kernel overlayfs with root privileges. Best performance, direct filesystem access. Requires running the API as root or with CAP_SYS_ADMIN.",
		DirectAccess: true,
		Requirements: make([]Requirement, 0),
	}

	// Requirement 1: Running as root or has CAP_SYS_ADMIN
	isRoot := os.Geteuid() == 0
	hasCapSysAdmin := checkCapSysAdmin()
	privilegedOK := isRoot || hasCapSysAdmin

	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "Root or CAP_SYS_ADMIN",
		Met:     privilegedOK,
		Current: func() string {
			if isRoot {
				return "running as root"
			}
			if hasCapSysAdmin {
				return "has CAP_SYS_ADMIN"
			}
			return "unprivileged"
		}(),
		HowToFix: func() string {
			if !privilegedOK {
				return "Run the API as root (sudo), or grant CAP_SYS_ADMIN: sudo setcap cap_sys_admin+ep /path/to/api"
			}
			return ""
		}(),
	})

	// Requirement 2: Overlayfs module available
	overlayOK := overlayfsModuleAvailable()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name:    "Overlayfs kernel module",
		Met:     overlayOK,
		Current: func() string {
			if overlayOK {
				return "loaded"
			}
			return "not available"
		}(),
		HowToFix: func() string {
			if !overlayOK {
				return "sudo modprobe overlay"
			}
			return ""
		}(),
	})

	opt.Available = privilegedOK && overlayOK
	return opt
}

// buildCopyDriverOption checks requirements for the copy driver.
func buildCopyDriverOption() DriverOption {
	return DriverOption{
		ID:           DriverOptionCopy,
		Name:         "Copy Driver (Fallback)",
		Description:  "Cross-platform fallback using file copies. Works on any OS, direct filesystem access. Higher disk usage (2x), slower for large directories.",
		DirectAccess: true,
		Requirements: []Requirement{}, // No requirements - always available
		Available:    true,
	}
}

// markRecommended marks the best available option as recommended.
func markRecommended(options []DriverOption) {
	// Priority order for recommendation:
	// 1. FUSE overlayfs (direct access, no root, good performance)
	// 2. Overlayfs + root (direct access, best performance, but needs root)
	// 3. Overlayfs + user namespace (no root, best performance, but no direct access)
	// 4. Copy driver (always works, but slower)

	priority := []DriverOptionID{
		DriverOptionFuseOverlayfs,
		DriverOptionOverlayfsRoot,
		DriverOptionOverlayfsUserNS,
		DriverOptionCopy,
	}

	for _, id := range priority {
		for i := range options {
			if options[i].ID == id && options[i].Available {
				options[i].Recommended = true
				return
			}
		}
	}
}

// --- Helper functions ---

// canCreateUserNamespace checks if user namespaces can be created.
func canCreateUserNamespace() bool {
	// Check the sysctl
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return false
		}
	}

	// Try to actually create one
	cmd := exec.Command("unshare", "--user", "true")
	return cmd.Run() == nil
}

// commandExists checks if a command is available in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// getCommandVersion runs a command with a version flag and returns output.
func getCommandVersion(name string, versionFlag string) string {
	cmd := exec.Command(name, versionFlag)
	out, err := cmd.Output()
	if err != nil {
		return "installed"
	}
	// Return first line only
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "installed"
}

// fuseAvailable checks if FUSE is available.
func fuseAvailable() bool {
	_, err := os.Stat("/dev/fuse")
	return err == nil
}

// checkCapSysAdmin checks if the process has CAP_SYS_ADMIN capability.
func checkCapSysAdmin() bool {
	// Try a simple test that requires CAP_SYS_ADMIN
	// Creating a mount namespace requires this capability
	cmd := exec.Command("unshare", "--mount", "true")
	return cmd.Run() == nil
}

// overlayfsModuleAvailable checks if the overlayfs kernel module is available.
func overlayfsModuleAvailable() bool {
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "overlay")
}

// testMountOverlayfs tries to actually mount overlayfs to verify it works.
func testMountOverlayfs() bool {
	// Create temporary directories
	tmpDir, err := os.MkdirTemp("", "overlay-test-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)

	lower := tmpDir + "/lower"
	upper := tmpDir + "/upper"
	work := tmpDir + "/work"
	merged := tmpDir + "/merged"

	for _, dir := range []string{lower, upper, work, merged} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false
		}
	}

	opts := "lowerdir=" + lower + ",upperdir=" + upper + ",workdir=" + work
	err = syscall.Mount("overlay", merged, "overlay", 0, opts)
	if err != nil {
		return false
	}

	syscall.Unmount(merged, 0)
	return true
}
