// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"os"
	"runtime"

	"workspace-sandbox/internal/namespace"
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

	// Optional indicates this requirement enhances functionality but isn't required
	Optional bool `json:"optional,omitempty"`
}

// Capabilities describes what isolation/protection features a driver provides.
type Capabilities struct {
	// FilesystemIsolation indicates copy-on-write protection for the canonical repo
	FilesystemIsolation bool `json:"filesystemIsolation"`

	// ProcessIsolation indicates bwrap namespace isolation for spawned processes
	ProcessIsolation bool `json:"processIsolation"`

	// NetworkIsolation indicates ability to restrict network access
	NetworkIsolation bool `json:"networkIsolation"`

	// DirectAccess indicates if merged directory is accessible outside the API
	DirectAccess bool `json:"directAccess"`

	// Notes provides additional context about capabilities
	Notes string `json:"notes,omitempty"`
}

// DriverOption represents a single driver option with its requirements.
type DriverOption struct {
	// ID is the unique identifier for this option
	ID DriverID `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description explains what this driver does and its trade-offs
	Description string `json:"description"`

	// DirectAccess indicates if the merged directory is accessible outside the API process
	DirectAccess bool `json:"directAccess"`

	// Capabilities describes what isolation features this driver provides
	Capabilities Capabilities `json:"capabilities"`

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
	CurrentDriver DriverID `json:"currentDriver"`

	// InUserNamespace indicates if the API is running inside a user namespace
	InUserNamespace bool `json:"inUserNamespace"`

	// Options lists all available driver options with their requirements
	Options []DriverOption `json:"options"`
}

// GetDriverOptions returns all available driver options with their requirements.
func GetDriverOptions(ctx context.Context, currentID DriverID, inUserNS bool) DriverOptionsResponse {
	resp := DriverOptionsResponse{
		OS:              runtime.GOOS,
		InUserNamespace: inUserNS,
		CurrentDriver:   currentID,
		Options:         make([]DriverOption, 0, 4),
	}

	if runtime.GOOS == "linux" {
		resp.Kernel = namespace.Check().KernelVersion
		resp.Options = append(resp.Options,
			buildOverlayfsUserNSOption(),
			buildFuseOverlayfsOption(),
			buildOverlayfsRootOption(),
		)
	}
	resp.Options = append(resp.Options, buildCopyDriverOption())

	markRecommended(resp.Options)
	return resp
}

// buildOverlayfsUserNSOption checks requirements for overlayfs in user namespace.
func buildOverlayfsUserNSOption() DriverOption {
	opt := DriverOption{
		ID:           DriverOverlayfsUserNS,
		Name:         "Overlayfs (User Namespace)",
		Description:  "Secure unprivileged overlayfs using Linux user namespaces. Best performance, no root required. Mounted files are only accessible via the API (exec endpoint or file operations API).",
		DirectAccess: false,
		Capabilities: Capabilities{
			FilesystemIsolation: true,
			ProcessIsolation:    true,
			NetworkIsolation:    true,
			DirectAccess:        false,
			Notes:               "Full isolation via bwrap. Processes see restricted filesystem view.",
		},
		Requirements: make([]Requirement, 0),
	}

	nsStatus := namespace.Check()

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

	usernsOK := canCreateUserNamespace()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "User namespaces enabled",
		Met:  usernsOK,
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

	unshareOK := commandExists("unshare")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "unshare command",
		Met:  unshareOK,
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
	bwrapOK := commandExists("bwrap")

	opt := DriverOption{
		ID:           DriverFuseOverlayfs,
		Name:         "FUSE Overlayfs",
		Description:  "Unprivileged overlayfs via FUSE. Direct filesystem access without root. Process isolation via bwrap when available.",
		DirectAccess: true,
		Capabilities: Capabilities{
			FilesystemIsolation: true,
			ProcessIsolation:    bwrapOK,
			NetworkIsolation:    bwrapOK,
			DirectAccess:        true,
			Notes: func() string {
				if bwrapOK {
					return "Full isolation with bwrap. Direct access to merged directory."
				}
				return "Filesystem isolation only. Install bwrap for process/network isolation."
			}(),
		},
		Requirements: make([]Requirement, 0),
	}

	fuseOverlayfsOK := commandExists("fuse-overlayfs")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "fuse-overlayfs installed",
		Met:  fuseOverlayfsOK,
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

	fuseOK := fuseAvailable()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "FUSE available",
		Met:  fuseOK,
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

	fusermountOK := commandExists("fusermount") || commandExists("fusermount3")
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "fusermount command",
		Met:  fusermountOK,
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

	opt.Requirements = append(opt.Requirements, Requirement{
		Name:     "bubblewrap (bwrap) for process isolation",
		Met:      bwrapOK,
		Optional: true,
		Current: func() string {
			if bwrapOK {
				return getCommandVersion("bwrap", "--version")
			}
			return "not installed"
		}(),
		HowToFix: func() string {
			if !bwrapOK {
				return "sudo apt install bubblewrap"
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
		ID:           DriverOverlayfsRoot,
		Name:         "Overlayfs (Privileged)",
		Description:  "Native kernel overlayfs with root privileges. Best performance, direct filesystem access. Requires running the API as root or with CAP_SYS_ADMIN.",
		DirectAccess: true,
		Capabilities: Capabilities{
			FilesystemIsolation: true,
			ProcessIsolation:    true,
			NetworkIsolation:    true,
			DirectAccess:        true,
			Notes:               "Full isolation with bwrap. Requires elevated privileges.",
		},
		Requirements: make([]Requirement, 0),
	}

	// Inside `unshare -U -r` Geteuid()==0 only in the namespace; we still
	// lack CAP_SYS_ADMIN on the host, so kernel overlayfs would mount via
	// the userns code path — which is what OverlayfsUserNS already covers.
	// Surface OverlayfsRoot only when host-level privileges are genuine.
	inUserNS := InUserNamespace()
	isRoot := os.Geteuid() == 0 && !inUserNS
	hasCapSysAdmin := checkCapSysAdmin()
	privilegedOK := isRoot || hasCapSysAdmin

	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "Root or CAP_SYS_ADMIN (host, not user namespace)",
		Met:  privilegedOK,
		Current: func() string {
			if inUserNS {
				return "running inside a user namespace; use OverlayfsUserNS"
			}
			if isRoot {
				return "running as root"
			}
			if hasCapSysAdmin {
				return "has CAP_SYS_ADMIN"
			}
			return "unprivileged"
		}(),
		HowToFix: func() string {
			if inUserNS {
				return "Switch to OverlayfsUserNS (the recommended driver) or remove the unshare wrapper to use OverlayfsRoot."
			}
			if !privilegedOK {
				return "Run the API as root (sudo), or grant CAP_SYS_ADMIN: sudo setcap cap_sys_admin+ep /path/to/api"
			}
			return ""
		}(),
	})

	overlayOK := overlayfsModuleAvailable()
	opt.Requirements = append(opt.Requirements, Requirement{
		Name: "Overlayfs kernel module",
		Met:  overlayOK,
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
		ID:           DriverCopy,
		Name:         "Copy Driver (Fallback)",
		Description:  "Cross-platform fallback using file copies. Works on any OS, direct filesystem access. Higher disk usage (2x), slower for large directories.",
		DirectAccess: true,
		Capabilities: Capabilities{
			FilesystemIsolation: true,
			ProcessIsolation:    false,
			NetworkIsolation:    false,
			DirectAccess:        true,
			Notes:               "Filesystem isolation via file copy. No process isolation (cross-platform).",
		},
		Requirements: []Requirement{},
		Available:    true,
	}
}

// markRecommended marks the best available option as recommended.
//
// Priority order (post-Phase 5):
//  1. OverlayfsUserNS — kernel overlayfs in a user namespace.
//  2. FuseOverlayfs — daemon-per-mount fallback.
//  3. OverlayfsRoot — kernel overlayfs with CAP_SYS_ADMIN.
//  4. Copy — always available, slowest.
func markRecommended(options []DriverOption) {
	priority := []DriverID{
		DriverOverlayfsUserNS,
		DriverFuseOverlayfs,
		DriverOverlayfsRoot,
		DriverCopy,
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
