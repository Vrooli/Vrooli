package exec

import (
	"fmt"
	"path/filepath"
	"strings"

	"workspace-sandbox/internal/rlimitexec"
	"workspace-sandbox/internal/types"
)

// This file holds the pure, OS-neutral core of the macOS Seatbelt
// containment backend: the sandbox-exec profile generator and the argv
// assembly. Keeping them build-tag-free lets golden tests pin the profile
// and command line on the Linux dev host; only the platformContainmentBackend
// wiring (backend_darwin.go) is darwin-tagged.
//
// The profile is deliberately default-allow with targeted denies — this is
// safety-from-accidents (a workload straying outside its workspace), not
// adversarial sandboxing. It enforces exactly two of the platform-neutral
// guarantees: filesystem-write-containment (writes confined to the sandbox
// writable set) and network-deny (when the profile disallows network). It
// provides NO path illusion and NO pid namespace; the darwin containment
// probe reports that honestly.

// BuildSeatbeltCommand assembles the sandbox-exec command line that runs cmd
// under the generated Seatbelt profile. It returns (executable, args) where
// the executable is always "sandbox-exec", mirroring BuildExecCommand's
// (executable, args) shape on Linux.
//
// When cfg.ResourceLimits has entries, the target is wrapped by the api
// binary's rlimit self-exec shim ("<shimPath> rlimit-exec … --"), exactly as
// BuildExecCommand prepends prlimit ahead of bwrap on Linux. shimPath must be
// the path to the api binary; it is ignored when no limits are configured.
func BuildSeatbeltCommand(shimPath string, s *types.Sandbox, cfg BwrapConfig, cmd string, cmdArgs ...string) (string, []string) {
	args := []string{"-p", SeatbeltProfile(s, cfg)}
	if shim := BuildRlimitShimArgs(shimPath, cfg.ResourceLimits); shim != nil {
		args = append(args, shim...)
	}
	args = append(args, cmd)
	args = append(args, cmdArgs...)
	return "sandbox-exec", args
}

// BuildRlimitShimArgs constructs the "<shimPath> rlimit-exec … --" prefix
// that applies portable resource limits before the target runs. Returns nil
// when no limits are configured, in which case the target runs directly under
// sandbox-exec. The flag names come from the rlimitexec package so producer
// and consumer never drift.
func BuildRlimitShimArgs(shimPath string, limits ResourceLimits) []string {
	if !limits.HasLimits() {
		return nil
	}
	args := []string{shimPath, rlimitexec.Subcommand}
	if limits.MemoryLimitMB > 0 {
		bytes := int64(limits.MemoryLimitMB) * 1024 * 1024
		args = append(args, fmt.Sprintf("--%s=%d", rlimitexec.FlagAddressSpace, bytes))
	}
	if limits.CPUTimeSec > 0 {
		args = append(args, fmt.Sprintf("--%s=%d", rlimitexec.FlagCPUTime, limits.CPUTimeSec))
	}
	if limits.MaxProcesses > 0 {
		args = append(args, fmt.Sprintf("--%s=%d", rlimitexec.FlagMaxProcesses, limits.MaxProcesses))
	}
	if limits.MaxOpenFiles > 0 {
		args = append(args, fmt.Sprintf("--%s=%d", rlimitexec.FlagMaxOpenFiles, limits.MaxOpenFiles))
	}
	args = append(args, "--")
	return args
}

// SeatbeltProfile generates the sandbox-exec profile (an SBPL s-expression)
// for a sandbox. Pure function of (sandbox, cfg): the writable set is derived
// from the sandbox overlay dirs plus the system temp/dev basics, and network
// is denied unless cfg.AllowNetwork is set. Later rules override earlier ones
// in SBPL, so the global `deny file-write*` is re-opened for the writable set.
func SeatbeltProfile(s *types.Sandbox, cfg BwrapConfig) string {
	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(allow default)\n")
	b.WriteString("(deny file-write*)\n")
	b.WriteString("(allow file-write*\n")
	for _, p := range seatbeltWritablePaths(s, cfg) {
		fmt.Fprintf(&b, "    (subpath %s)\n", seatbeltString(p))
	}
	// Device basics every workload expects to write: the null sink and the
	// controlling tty family (/dev/tty, /dev/ttys###).
	b.WriteString("    (literal \"/dev/null\")\n")
	b.WriteString("    (regex #\"^/dev/tty\"))\n")
	if !cfg.AllowNetwork {
		b.WriteString("(deny network*)\n")
	}
	return b.String()
}

// seatbeltWritablePaths returns the absolute paths the profile grants write
// access to, in a stable order, each accompanied by its /private symlink twin
// when applicable. On macOS /tmp and /var are symlinks into /private, so a
// (subpath "/tmp") rule alone would miss writes that resolve to
// /private/tmp; emitting both twins closes that gap.
func seatbeltWritablePaths(s *types.Sandbox, cfg BwrapConfig) []string {
	var paths []string
	add := func(p string) {
		if p == "" || !filepath.IsAbs(p) {
			return
		}
		clean := filepath.Clean(p)
		paths = append(paths, clean)
		if twin, ok := privateTwin(clean); ok {
			paths = append(paths, twin)
		}
	}

	// Sandbox overlay dirs: the merged view the workload sees plus the upper
	// and work dirs the overlay writes land in.
	add(s.MergedDir)
	add(s.UpperDir)
	add(s.WorkDir)
	// Per-sandbox HOME overlay, when one is mounted (mirrors the bwrap
	// HomeMergedDir bind).
	if s.HomeMergedDir != "" && cfg.HostHome != "" {
		add(s.HomeMergedDir)
	}

	// System temp roots every toolchain scribbles into.
	add("/tmp")
	add("/var/folders")
	return paths
}

// privateTwin returns the /private-prefixed path for macOS temp roots that
// live behind the /private symlink (/tmp -> /private/tmp, /var -> /private/var).
// Returns ("", false) for paths that have no such twin.
func privateTwin(clean string) (string, bool) {
	for _, pfx := range []string{"/tmp", "/var"} {
		if clean == pfx || strings.HasPrefix(clean, pfx+"/") {
			return "/private" + clean, true
		}
	}
	return "", false
}

// seatbeltString renders a Go string as an SBPL string literal, escaping the
// backslash and double-quote characters that would otherwise break the
// s-expression.
func seatbeltString(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
