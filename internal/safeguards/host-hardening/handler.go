// Package hosthardening installs two diagnostic-friendly host configurations:
//
//  1. Sysctl drop-in at /etc/sysctl.d/99-vrooli-host-hardening.conf that
//     turns kernel hangs and oopses into observable, recoverable events:
//
//     kernel.sysrq                  enables manual `echo c > /proc/sysrq-trigger`
//                                   for kdump validation testing.
//     kernel.hung_task_timeout_secs flag tasks blocked >120s in dmesg.
//     kernel.softlockup_panic       panic on a CPU softlockup so kdump fires
//                                   instead of the kernel sitting wedged
//                                   until a hardware watchdog hard-resets.
//     kernel.unknown_nmi_panic      panic on unknown NMIs (firmware/IPMI
//                                   watchdog NMIs). Same diagnostic
//                                   motivation: silent hard resets defeat
//                                   debugging.
//     kernel.panic_on_oops          turn an oops into a panic so kdump
//                                   captures a vmcore. Without this, the
//                                   kernel logs the oops and limps onward.
//     kernel.panic                  reboot 10s after panic if vmcore capture
//                                   succeeded (or failed) — so the box
//                                   doesn't sit dead at 4am.
//
//  2. Journald rate-limit drop-in at
//     /etc/systemd/journald.conf.d/99-vrooli-ratelimit.conf that raises the
//     burst threshold so kernel-priority panic info isn't dropped under a
//     UFW BLOCK flood. The 2026-05-07 investigation found a 305 MB syslog
//     dominated by UFW lines that made post-crash log inspection slow and
//     could legitimately rate-limit panic messages.
//
// Ordering: this safeguard depends on kdump-tools being armed (Phase 2 of
// the 2026-05-07 work). Without a working kdump capture, panic_on_oops=1
// turns previously-survivable oopses into reboots that produce nothing —
// strictly worse than the status quo. The kdump-tools handler verifies the
// arm state; deploy this safeguard only after that handler reports
// already_present.
package hosthardening

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	sysctlPath  = "/etc/sysctl.d/99-vrooli-host-hardening.conf"
	journaldDir = "/etc/systemd/journald.conf.d"
	journaldPath = journaldDir + "/99-vrooli-ratelimit.conf"
)

// managedSysctls lists kernel parameters and their desired values. Each
// must be readable from /proc/sys/<name with dots → slashes>.
var managedSysctls = []struct {
	Name  string
	Value int
}{
	{"kernel.sysrq", 1},
	{"kernel.hung_task_timeout_secs", 120},
	{"kernel.softlockup_panic", 1},
	{"kernel.unknown_nmi_panic", 1},
	{"kernel.panic_on_oops", 1},
	{"kernel.panic", 10},
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

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

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "host hardening sysctls + journald drop-in are Linux-only")
		return status
	}

	if !host.SupportsSysctl {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not support sysctl")
		return status
	}

	pending := []string{}

	// Live sysctl values reflect what the kernel actually uses, not what's
	// in the drop-in. Drift is possible (operator did a one-shot sysctl -w
	// that didn't get persisted; a subsequent reboot flipped them back).
	for _, p := range managedSysctls {
		if got := readSysctlValue(p.Name); got != p.Value {
			pending = append(pending, fmt.Sprintf("%s=%d (current: %d)", p.Name, p.Value, got))
		}
	}

	if !hostreqkit.FileContentMatches(sysctlPath, buildSysctlContent()) {
		pending = append(pending, sysctlPath+" needs update")
	}

	if !hostreqkit.FileContentMatches(journaldPath, buildJournaldContent()) {
		pending = append(pending, journaldPath+" needs update")
	}

	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "host hardening sysctls and journald rate-limit already in place")
		return status
	}

	status.Notes = append(status.Notes, "host hardening pending: "+strings.Join(pending, ", "))
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

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			"dry-run: would write "+sysctlPath+", "+journaldPath+
				", run `sysctl --system`, and restart systemd-journald")
		return status, nil
	}

	if err := hostreqkit.EnsureManagedDir("/etc/sysctl.d", opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := hostreqkit.InstallManagedContent(sysctlPath, buildSysctlContent(), opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.EnsureManagedDir(journaldDir, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := hostreqkit.InstallManagedContent(journaldPath, buildJournaldContent(), opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	// Apply sysctls live. `sysctl --system` re-reads every drop-in including
	// any conflicting ones; this is what we want — it surfaces conflicts via
	// non-zero exit and matches what `systemd-sysctl` would do at boot.
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sysctl", []string{"--system"}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "files written but `sysctl --system` failed: "+err.Error())
		return status, nil
	}

	// Restart journald so the rate-limit drop-in takes effect immediately.
	// Restart (not reload) is required — journald reloads its own config on
	// SIGHUP, but RateLimit changes specifically require a restart.
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl",
		[]string{"restart", "systemd-journald"}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "files written and sysctls applied, but `systemctl restart systemd-journald` failed: "+err.Error())
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "host hardening sysctls applied and journald rate-limit active")
	return status, nil
}

func readSysctlValue(param string) int {
	procPath := "/proc/sys/" + strings.ReplaceAll(param, ".", "/")
	data, err := hostreqkit.ReadFileFn(procPath)
	if err != nil {
		return -1
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}
	return v
}

func buildSysctlContent() string {
	var b strings.Builder
	b.WriteString("# Managed by Vrooli -- do not edit manually\n")
	b.WriteString("# Make kernel hangs and oopses produce diagnostics rather than silent hard resets.\n")
	b.WriteString("# See internal/safeguards/host-hardening/handler.go for rationale.\n")
	for _, p := range managedSysctls {
		fmt.Fprintf(&b, "%s = %d\n", p.Name, p.Value)
	}
	return b.String()
}

func buildJournaldContent() string {
	return `# Managed by Vrooli -- do not edit manually
# Raise rate-limit so UFW BLOCK floods don't drown kernel-priority panic info.
# See internal/safeguards/host-hardening/handler.go for rationale.
[Journal]
RateLimitIntervalSec=30s
RateLimitBurst=10000
`
}
