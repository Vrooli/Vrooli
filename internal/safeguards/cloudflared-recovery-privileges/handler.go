// Package cloudflaredrecoveryprivileges installs the minimal NOPASSWD sudoers
// grant that lets tunnel-manager's auto-recovery loop restart cloudflared
// without an interactive sudo prompt.
//
// # Why this exists
//
// tunnel-manager runs as the project user (non-root). cloudflared is a root
// systemd service. tunnel-manager's recovery engine actuates a stuck/flapping
// tunnel with `sudo systemctl reset-failed cloudflared && sudo systemctl
// restart cloudflared` — but a non-root, non-interactive process cannot
// satisfy a sudo password prompt, so without a passwordless grant the restart
// fails and the recovery loop is dark. This safeguard writes that grant once,
// during `sudo vrooli setup` (the one sanctioned privilege-escalation point),
// so sudo is paid a single time at onboarding rather than on every recovery.
//
// # Why a safeguard, not the cloudflared tool
//
// The cloudflared *tool* installs the binary (presence). Privilege/policy is a
// separate concern: a host-level grant that outlives any single scenario and
// must be visudo-validated and risk-surfaced at onboarding. That is exactly
// what the safeguard framework models (mirrors ollama-resource-controls, which
// likewise writes a privileged file and self-gates on a systemd unit).
//
// # Why exactly these two commands, no wildcards
//
// The grant is two literal argv lines — `systemctl restart cloudflared` and
// `systemctl reset-failed cloudflared` — scoped to the invoking user. No
// wildcards: there is no injection surface, and the recovery code's command
// literals are hardcoded (not user-derived). reset-failed is included because
// after cloudflared flaps past systemd's StartLimitBurst, a bare restart is
// rejected until the start-limit counter is cleared — the precise case
// tunnel-manager adds value over systemd's own Restart=on-failure.
//
// # Safety
//
// The temp file is validated with `visudo -c -f` before install; on any
// validation error the real /etc/sudoers.d/tunnel-manager is never touched. We
// only ever write a drop-in under /etc/sudoers.d/ (mode 0440), never
// /etc/sudoers itself. Re-applying identical content is a clean no-op
// (idempotent: twice == once).
//
// # Self-promotion to required under root
//
// The manifest declares this optional so agents/operators who don't run the
// recovery loop are not blocked at setup. But when the operator has already
// escalated (Geteuid()==0, i.e. `sudo vrooli setup`), applying the grant is
// the useful thing to do — they've paid the sudo cost — so Inspect promotes it
// to required, matching the vrooli-launcher pattern.
package cloudflaredrecoveryprivileges

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// sudoersPath is the drop-in this safeguard manages. /etc/sudoers.d/ is the
// blessed location for per-tool grants; sudo includes it by default.
const sudoersPath = "/etc/sudoers.d/tunnel-manager"

// systemctlPath is the absolute command the grant authorizes. sudo matches a
// Cmnd against the fully-qualified path it resolves via secure_path; on every
// systemd distro systemctl lives at /usr/bin/systemctl (usrmerge aliases
// /bin/systemctl to the same inode).
const systemctlPath = "/usr/bin/systemctl"

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the constructor wired into the runtime registry under the
// handler name "cloudflared_recovery_privileges" (see
// internal/runtime/registry.go).
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

	// sudoers grants are a Linux+systemd concern. Off-platform, off-systemd,
	// or tunnel-less hosts have nothing to protect — cleanly not-applicable so
	// this never shows as a missing safeguard.
	if host.OS != "linux" {
		return notApplicable(status, "cloudflared recovery sudoers grant is Linux-only")
	}
	if !host.SupportsSystemd {
		return notApplicable(status, "host does not support systemd")
	}
	if !cloudflaredUnitPresent() {
		return notApplicable(status, "cloudflared.service not present on this host; recovery has no tunnel to restart")
	}

	user := hostreqkit.InvokingUser()
	if user == "" {
		return notApplicable(status, "could not resolve invoking user; cannot scope a sudoers grant")
	}

	if hostreqkit.FileContentMatches(sudoersPath, buildSudoersContent(user)) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "cloudflared recovery sudoers grant already in place at "+sudoersPath)
		return status
	}

	// Bootstrap promotion: when already root (typically from `sudo vrooli
	// setup`), apply without requiring --include-optional.
	if hostreqkit.RunningAsRootFn() {
		status.Required = true
	}

	status.Notes = append(status.Notes, "cloudflared recovery sudoers grant missing or stale at "+sudoersPath)
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
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	user := hostreqkit.InvokingUser()
	if user == "" {
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "could not resolve invoking user; cannot scope a sudoers grant")
		return status, nil
	}
	content := buildSudoersContent(user)

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would write "+sudoersPath+" granting NOPASSWD systemctl restart/reset-failed cloudflared to "+user)
		return status, nil
	}

	tmpPath, err := hostreqkit.WriteTempFileFn(content)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "prepare sudoers content: "+err.Error())
		return status, nil
	}
	defer os.Remove(tmpPath)

	// Validate BEFORE install. A malformed sudoers file can lock the operator
	// out of sudo, so visudo-check the exact bytes; abort on any error and
	// never touch the real drop-in.
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "visudo", []string{"-c", "-f", tmpPath}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "sudoers validation failed; "+sudoersPath+" not written: "+err.Error())
		return status, nil
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0440", tmpPath, sudoersPath}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install sudoers grant failed: "+err.Error())
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "cloudflared recovery sudoers grant installed at "+sudoersPath+"; tunnel-manager can now restart cloudflared non-interactively")
	return status, nil
}

func notApplicable(status hostreqkit.ItemStatus, note string) hostreqkit.ItemStatus {
	status.SupportClass = hostreqkit.SupportNotApplicable
	status.ExecutionState = hostreqkit.ExecutionNotApplicable
	status.Notes = append(status.Notes, note)
	return status
}

// cloudflaredUnitPresent reports whether systemd knows about
// cloudflared.service. Uses `systemctl list-unit-files` so units under both
// /etc/systemd/system and /lib/systemd/system are caught without hardcoding
// paths (mirrors the recovery engine's own presence gate).
func cloudflaredUnitPresent() bool {
	out, err := hostreqkit.CombinedOutputFn("systemctl", "list-unit-files", "--no-pager", "--no-legend", "cloudflared.service")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "cloudflared.service")
}

// buildSudoersContent renders the exact drop-in for the given user. A single
// Cmnd line with two literal argv entries, no wildcards. The trailing newline
// keeps visudo happy and matches how `install` writes managed files.
func buildSudoersContent(user string) string {
	return fmt.Sprintf(
		"# Managed by Vrooli -- do not edit manually\n"+
			"# Lets tunnel-manager's recovery loop restart cloudflared non-interactively.\n"+
			"# See internal/safeguards/cloudflared-recovery-privileges/handler.go for rationale.\n"+
			"%s ALL=(root) NOPASSWD: %s restart cloudflared, %s reset-failed cloudflared\n",
		user, systemctlPath, systemctlPath,
	)
}
