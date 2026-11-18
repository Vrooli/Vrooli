// Package vroolilauncher manages the optional POSIX command shim at
// /usr/local/bin/vrooli. The shim resolves the invoking user's executable
// under ~/.vrooli/bin, allowing commands such as `sudo vrooli host wireless`
// when sudo's PATH cannot see the user's home directory. Its installation
// path and content are owned here; platform support is Linux and macOS.
package vroolilauncher

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// LauncherPath is the managed shim location.
const LauncherPath = "/usr/local/bin/vrooli"

// shimContent is the POSIX shell script written to LauncherPath. The header
// comment in the script itself is intentional — an admin reading
// /usr/local/bin/vrooli should be able to understand what it is and why
// it exists without having to find this Go file.
//
// The script:
//   - Reads $SUDO_USER (set by sudo on the invoking user's behalf) or falls
//     back to $USER for non-sudo invocations.
//   - Looks up that user's home directory from /etc/passwd via awk. We
//     deliberately use awk-on-passwd rather than `getent` because `getent`
//     is glibc-only — it does not exist on macOS. /etc/passwd is consulted
//     by both Linux and macOS for local accounts.
//   - Falls back to $HOME if the passwd lookup yields nothing (covers
//     edge cases like AD-bound macs where local /etc/passwd is sparse;
//     the invoking user's $HOME is still correct for them, even if the
//     cross-user case won't work without dscl).
//   - prefers the canonical binary, but temporarily falls back to the last
//     known-good copy if a repair or foreign remover made the canonical path
//     unavailable. The fallback is maintained by the binary installers.
const shimContent = `#!/bin/sh
# Vrooli launcher shim — installed by ` + "`vrooli setup --include-optional`" + `.
# See: internal/safeguards/vrooli-launcher/handler.go (package doc).
#
# This file exists so ` + "`sudo vrooli ...`" + ` works. The real binary lives in
# ~/.vrooli/bin/vrooli (where ` + "`make install`" + ` writes it without sudo, so the
# dev loop and coding-agent automation work). sudo's secure_path doesn't
# include user home dirs, so we forward through this shim instead.
#
# Multi-user note: $SUDO_USER points at the invoking user when run via
# sudo; for normal invocations $USER works. Each user's invocation
# resolves to their own ~/.vrooli/bin/vrooli.
set -e
user="${SUDO_USER:-$USER}"
home=$(awk -F: -v u="$user" '$1==u{print $6; exit}' /etc/passwd)
: "${home:=$HOME}"
primary="$home/.vrooli/bin/vrooli"
fallback="$home/.vrooli/libexec/vrooli.previous"
if [ -x "$primary" ]; then
    exec "$primary" "$@"
fi
if [ -x "$fallback" ]; then
    echo "vrooli: canonical executable is unavailable; using last known-good fallback" >&2
    exec "$fallback" "$@"
fi
echo "vrooli: no usable control-plane executable at $primary or $fallback" >&2
exit 127
`

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the constructor wired into the runtime registry under the
// handler name "vrooli_launcher" (see internal/runtime/registry.go).
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	// Linux and macOS share the shim implementation. Windows has no sudo
	// equivalent and would need its own design (likely a .cmd file in a
	// user-PATH directory plus UAC manifest); flag as Unsupported so this
	// safeguard cleanly disappears on Windows hosts instead of failing.
	if host.OS != string(hostreqspec.PlatformLinux) && host.OS != string(hostreqspec.PlatformDarwin) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "vrooli launcher shim is POSIX-only (Linux + macOS); Windows needs a separate design")
		return status
	}

	// FileContentMatches returns true only when the file exists *and* its
	// contents match — covers both "missing" and "outdated shim from an
	// older Vrooli version" in one branch. Stale shims get rewritten by
	// Apply.
	if hostreqkit.FileContentMatches(LauncherPath, shimContent) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "launcher shim installed at "+LauncherPath)
		return status
	}

	// Bootstrap promotion: when the operator is already running as root
	// (typically from `sudo vrooli setup`), promote the requirement to
	// required so the runtime applies it without `--include-optional`.
	// See package doc "Self-promotion to required when running as root".
	if hostreqkit.RunningAsRootFn() {
		status.Required = true
	}

	status.Notes = append(status.Notes, "launcher shim missing or stale at "+LauncherPath+"; `sudo vrooli` may not resolve")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install POSIX launcher shim to "+LauncherPath)
		return status, nil
	}

	// InstallManagedExecutable handles the temp-file + sudo install pattern
	// (mode 755, since this is an executable shim). On --sudo-mode=skip
	// the call returns ErrSudoSkipped, which the runtime's apply loop
	// detects and tags as BlockingNeedsSudo so the renderer routes the
	// item into the "Needs sudo" group with an actionable hint.
	if err := hostreqkit.InstallManagedExecutable(LauncherPath, shimContent, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install launcher shim failed: "+err.Error())
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "launcher shim installed at "+LauncherPath+"; `sudo vrooli` now resolves")
	return status, nil
}
