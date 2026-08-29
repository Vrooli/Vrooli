// Package vroolilauncher installs a small POSIX shell shim at
// /usr/local/bin/vrooli that forwards every invocation to the real Vrooli
// binary in the calling user's ~/.vrooli/bin directory.
//
// # Why this exists
//
// `make install` writes Vrooli's binaries to ~/.vrooli/bin. That directory
// is on the developer's interactive PATH, so plain `vrooli ...` works. But
// sudo enforces a locked-down `secure_path` ("/usr/local/sbin:/usr/local/bin:
// /usr/sbin:/usr/bin:/sbin:/bin" on Ubuntu and most other distros, similar
// on macOS) that does not include any user home directory. The result:
//
//	$ sudo vrooli setup
//	sudo: vrooli: command not found
//
// That breaks the `sudo vrooli setup` workflow that the setup output's
// "Needs sudo" action block points operators at. We need *something* in
// sudo's PATH that resolves to the real binary.
//
// # Why a shim — and not an alternative
//
// Three alternatives were considered and rejected:
//
//  1. Move the real binary into /usr/local/bin. Cleanest filesystem layout
//     and matches how mainstream distributed CLIs (gh, kubectl, terraform,
//     docker, aws) ship. Rejected because it makes every `make install`
//     require root — which is the dev loop, run dozens of times per hour
//     during development, and run *unattended* by coding agents (Claude
//     Code, Codex) that cannot reliably escalate to sudo. Forcing root on
//     each rebuild would block agent-driven iteration outright.
//
//  2. Drop /etc/sudoers.d/vrooli with a `Defaults secure_path` extension
//     that includes ~/.vrooli/bin. Rejected because it modifies system
//     security policy for one tool, security-conscious admins reject it
//     during review, and no major CLI ships this pattern.
//
//  3. Document `sudo $(which vrooli) ...` in the action block and leave
//     install paths alone. Rejected because it's a workaround dressed up as
//     a fix; users hit it every fresh install, the absolute path is ugly,
//     and the suggestion only works because the operator happens to have
//     vrooli on their non-sudo PATH.
//
// What works is a tiny shim — a 5-line POSIX shell script in /usr/local/bin
// that execs the real binary in ~/.vrooli/bin. This is the same pattern
// pyenv, rbenv, nvm-exec, and volta use for the same reason: a binary that
// rewrites frequently during development must stay in user-writable space,
// but still needs to be reachable from sudo and other constrained PATH
// contexts. Subsequent `make install` runs only rewrite ~/.vrooli/bin/vrooli;
// the shim itself never needs to be touched again.
//
// # Why a shell shim, not a symlink
//
// A symlink at /usr/local/bin/vrooli would point at one specific user's
// ~/.vrooli/bin/vrooli — fine for a single-developer machine, broken on
// multi-user hosts where two users each install Vrooli. The shim instead
// resolves $SUDO_USER (set by sudo) or $USER (otherwise) at exec time and
// looks up that user's home directory in /etc/passwd. Each user's
// `sudo vrooli` then runs *their* binary.
//
// # Why /usr/local/bin
//
// /usr/local/bin is the POSIX-blessed location for locally installed
// software. Sudo's default secure_path includes it on every Linux
// distribution and on macOS. Other candidates (/usr/bin, /opt/local/bin)
// are reserved for distribution packages or specific package managers; we
// avoid them.
//
// # Cross-platform scope
//
// Linux and macOS share the same shim — both honor /etc/passwd for local
// account home-directory lookups, both have /usr/local/bin in sudo's
// secure_path, and POSIX shell is present out of the box. Windows uses a
// fundamentally different elevation model (UAC, no sudo, no secure_path),
// so the handler reports ExecutionUnsupported there and a separate
// platform-specific design will need to land if `sudo vrooli`-equivalent
// is ever needed on Windows.
//
// # Bootstrap chicken-and-egg
//
// On a fresh install the shim does not yet exist, so `sudo vrooli setup`
// cannot install it (the binary is not on sudo's PATH). The setup output's
// action block detects this case and emits the absolute path
// (`sudo /home/user/.vrooli/bin/vrooli setup`) until the shim is in place.
// After the first sudo'd setup run installs the shim, the action block
// reverts to the bare `sudo vrooli ...` form.
//
// # Self-promotion to "required" when running as root
//
// The manifest declares this safeguard as `required: false` so that
// agents and operators who don't care about `sudo vrooli` are not blocked
// at setup time — they see it in the Optional group and can ignore it.
//
// However, when the operator HAS escalated privileges (Geteuid()==0,
// typically because they ran `sudo vrooli setup`), the most useful thing
// we can do is install the shim — they've already paid the sudo cost,
// and skipping the install just to honor "optional" would force them to
// remember `--include-optional` on top. So Inspect promotes the
// requirement to required when running as root: the runtime then applies
// it as part of the normal required-pass, no extra flag needed.
//
// This keeps the framework gate simple (Required vs IncludeOptional is
// the only switch the runtime knows about) and lets the handler express
// situational policy without a new framework concept.
package vroolilauncher

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// LauncherPath is the install location of the shim. /usr/local/bin is
// in sudo's default secure_path on every Linux distribution and on macOS.
// Exported because the action-block fallback in
// internal/setup/requirements_report.go probes the same path to decide
// whether `sudo vrooli ...` will work.
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
	if err := hostreqkit.InstallManagedExecutable(LauncherPath, shimContent, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install launcher shim failed: "+err.Error())
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "launcher shim installed at "+LauncherPath+"; `sudo vrooli` now resolves")
	return status, nil
}
