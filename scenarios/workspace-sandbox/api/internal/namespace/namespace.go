// Package namespace provides utilities for working with Linux user namespaces
// to enable unprivileged overlayfs mounts.
//
// # Background
//
// Linux kernel 5.11+ supports overlayfs mounts from within unprivileged user
// namespaces. This allows workspace-sandbox to use native kernel overlayfs
// without requiring root privileges or special capabilities.
//
// # How It Works
//
// The portable workspace-sandbox launcher decides whether the API process
// must start through `unshare -U -m -r` before main runs. This package only
// reports namespace diagnostics for driver option reporting.
//
// # Round 4 Phase 7
//
// Every external command invocation routes through process.Starter and
// every overlayfs probe routes through fsmount.Mounter so the syscall
// surface is confined to the canonical seams.
package namespace

import (
	"context"
	"os"
	"strconv"
	"strings"

	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/process"
)

// Status represents the current namespace status
type Status struct {
	// InUserNamespace is true if we're currently in a user namespace
	InUserNamespace bool

	// CanCreateUserNamespace is true if we can create new user namespaces
	CanCreateUserNamespace bool

	// CanMountOverlayfs is true if overlayfs mounts work (tested empirically)
	CanMountOverlayfs bool

	// KernelVersion is the running kernel version
	KernelVersion string

	// Reason explains why certain features aren't available
	Reason string
}

// Check returns the current namespace status. starter is required to
// probe `unshare` availability and routes the test mount through
// fsmount.Mounter; both are constructed once in main.go and threaded
// down so tests can stub them.
func Check(starter process.Starter) Status {
	status := Status{
		InUserNamespace: inUserNamespace(),
		KernelVersion:   getKernelVersion(),
	}

	// Check if we can create user namespaces
	status.CanCreateUserNamespace = canCreateUserNamespace(starter)
	if !status.CanCreateUserNamespace {
		status.Reason = "cannot create user namespaces (kernel config or security policy)"
		return status
	}

	// If we're already in a user namespace, test if overlayfs works
	if status.InUserNamespace {
		status.CanMountOverlayfs = testOverlayfsMount(starter)
		if !status.CanMountOverlayfs {
			status.Reason = "overlayfs mount failed inside user namespace"
		}
		return status
	}

	// We can create namespaces but haven't entered one yet
	status.Reason = "not yet in user namespace"
	return status
}

// canCreateUserNamespace checks if the current process can create user
// namespaces. Routes through Starter for the empirical `unshare --user
// true` probe.
func canCreateUserNamespace(starter process.Starter) bool {
	// Check the sysctl that controls unprivileged user namespaces
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return false
		}
	}
	// If the file doesn't exist, user namespaces are likely enabled by default

	// Check if unshare is available
	if _, err := starter.LookPath("unshare"); err != nil {
		return false
	}

	// Try to actually create a user namespace (most reliable test)
	res, err := process.Run(context.Background(), starter, process.StartOpts{
		Path: "unshare",
		Args: []string{"--user", "true"},
	})
	if err != nil {
		return false
	}
	return res.Exit.ExitCode == 0
}

// testOverlayfsMount tests if we can actually mount overlayfs. Routes
// through fsmount.SystemMounter so the underlying mount syscalls stay
// in the canonical seam.
func testOverlayfsMount(starter process.Starter) bool {
	mounter := fsmount.NewSystemMounter(starter)
	return fsmount.ProbeKernelOverlayMount(context.Background(), mounter) == nil
}

func inUserNamespace() bool {
	data, err := os.ReadFile("/proc/self/uid_map")
	if err != nil {
		return false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return false
	}
	return !(fields[0] == "0" && fields[1] == "0" && fields[2] == "4294967295")
}

// IsKernelAtLeast checks if the kernel version is at least the specified version
func IsKernelAtLeast(major, minor int) bool {
	version := getKernelVersion()
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	kernelMajor, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	kernelMinor, err := strconv.Atoi(strings.Split(parts[1], "-")[0])
	if err != nil {
		return false
	}

	if kernelMajor > major {
		return true
	}
	if kernelMajor == major && kernelMinor >= minor {
		return true
	}
	return false
}

// RequiredKernelVersion is the minimum kernel version for unprivileged overlayfs
const (
	RequiredKernelMajor = 5
	RequiredKernelMinor = 11 // 5.13 recommended for SELinux support
)
