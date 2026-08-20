// Package hosthardening installs two diagnostic-friendly host configurations:
//
//  1. Sysctl drop-in at /etc/sysctl.d/99-vrooli-host-hardening.conf that
//     turns kernel hangs and oopses into observable, recoverable events:
//
//     kernel.sysrq                  enables manual `echo c > /proc/sysrq-trigger`
//     for kdump validation testing.
//     kernel.hung_task_timeout_secs flag tasks blocked in D state for this many
//     seconds in dmesg. Configurable.
//     kernel.softlockup_panic       whether a CPU soft lockup panics the host.
//     Configurable via softlockup_policy;
//     defaults to warn.
//     kernel.unknown_nmi_panic      panic on unknown NMIs (firmware/IPMI
//     watchdog NMIs). Silent hard resets
//     defeat debugging.
//     kernel.panic_on_oops          whether an oops becomes a panic so kdump
//     captures a vmcore. Configurable via
//     oops_policy.
//     kernel.panic                  reboot 10s after panic if vmcore capture
//     succeeded (or failed) — so the box
//     doesn't sit dead at 4am.
//
// The two panic policies are operator configuration rather than fixed values,
// because the right answer differs by role. On a fleet node that can be drained
// and replaced, failing fast and capturing a vmcore is right. On a workstation
// that is also somebody's daily driver, converting every single-task kernel
// fault into a full outage is a poor trade — and `softlockup_panic` in
// particular fires on ordinary saturation, which the 2026-08-19 incident on
// this host showed is not a rare event. The defaults reflect that: oops still
// panics and dumps, soft lockups only warn.
//
//  2. Journald rate-limit drop-in at
//     /etc/systemd/journald.conf.d/99-vrooli-ratelimit.conf that raises the
//     burst threshold so kernel-priority panic info isn't dropped under a
//     UFW BLOCK flood. The 2026-05-07 investigation found a 305 MB syslog
//     dominated by UFW lines that made post-crash log inspection slow and
//     could legitimately rate-limit panic messages.
//
// Ordering: with oops_policy=panic-and-dump this safeguard depends on a loaded
// crash kernel. Without one, panic_on_oops=1 turns previously-survivable oopses
// into reboots that produce nothing — strictly worse than the status quo.
//
// That dependency used to be documented here and enforced nowhere. Inspect now
// checks it directly against /sys/kernel/kexec_crash_loaded and reports
// BlockingPrerequisiteMissing rather than applying a policy the host cannot
// honour, so the ordering holds even when an operator applies this safeguard on
// its own.
package hosthardening

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	sysctlPath   = "/etc/sysctl.d/99-vrooli-host-hardening.conf"
	journaldDir  = "/etc/systemd/journald.conf.d"
	journaldPath = journaldDir + "/99-vrooli-ratelimit.conf"

	// kexecCrashLoadedPath is the kernel's own statement that a crash kernel is
	// loaded and will run on panic. It is a more direct answer than parsing
	// `kdump-config status`: this file is what the panic path actually consults.
	kexecCrashLoadedPath = "/sys/kernel/kexec_crash_loaded"

	oopsPolicyPanicAndDump   = "panic-and-dump"
	oopsPolicyLogAndContinue = "log-and-continue"
	softlockupPolicyPanic    = "panic"
	softlockupPolicyWarn     = "warn"
)

// policy is the resolved operator configuration for this safeguard.
type policy struct {
	OopsPolicy       string
	SoftlockupPolicy string
	HungTaskTimeout  int
}

// resolvePolicy reads declared config, falling back to the manifest defaults.
// The defaults are duplicated here rather than read from the manifest because a
// handler must still behave correctly when invoked with no resolved config at
// all; the invariant is covered by TestDefaultsMatchManifest.
func resolvePolicy(config map[string]any) policy {
	p := policy{
		OopsPolicy:       oopsPolicyPanicAndDump,
		SoftlockupPolicy: softlockupPolicyWarn,
		HungTaskTimeout:  120,
	}
	if config == nil {
		return p
	}
	if v, ok := config["oops_policy"].(string); ok && strings.TrimSpace(v) != "" {
		p.OopsPolicy = strings.TrimSpace(v)
	}
	if v, ok := config["softlockup_policy"].(string); ok && strings.TrimSpace(v) != "" {
		p.SoftlockupPolicy = strings.TrimSpace(v)
	}
	// JSON numbers decode as float64; accept int for programmatic callers.
	switch v := config["hung_task_timeout_secs"].(type) {
	case float64:
		p.HungTaskTimeout = int(v)
	case int:
		p.HungTaskTimeout = v
	}
	return p
}

type sysctlSetting struct {
	Name  string
	Value int
}

// managedSysctls lists kernel parameters and their desired values for a
// resolved policy. Each must be readable from /proc/sys/<name with dots →
// slashes>.
//
// kernel.sysrq, kernel.unknown_nmi_panic and kernel.panic are not configurable:
// sysrq is a diagnostic entry point with no downside, an unknown NMI is always
// a hardware-level event worth capturing, and the 10-second post-panic reboot
// only matters once a panic has already happened.
func managedSysctls(p policy) []sysctlSetting {
	boolToInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	return []sysctlSetting{
		{"kernel.sysrq", 1},
		{"kernel.hung_task_timeout_secs", p.HungTaskTimeout},
		{"kernel.softlockup_panic", boolToInt(p.SoftlockupPolicy == softlockupPolicyPanic)},
		{"kernel.unknown_nmi_panic", 1},
		{"kernel.panic_on_oops", boolToInt(p.OopsPolicy == oopsPolicyPanicAndDump)},
		{"kernel.panic", 10},
	}
}

// kdumpArmed reports whether a crash kernel is loaded. Stubbed in tests.
var kdumpArmed = func() bool {
	data, err := hostreqkit.ReadFileFn(kexecCrashLoadedPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
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

	resolved := resolvePolicy(requirement.Config)

	// panic_on_oops without an armed crash kernel is strictly worse than the
	// default: it converts survivable oopses into reboots that produce no
	// vmcore. Refuse to apply the policy rather than silently degrading the
	// host's ability to explain its own crashes.
	if resolved.OopsPolicy == oopsPolicyPanicAndDump && !kdumpArmed() {
		status.BlockingReason = hostreqkit.BlockingPrerequisiteMissing
		status.Notes = append(status.Notes,
			"oops_policy=panic-and-dump requires an armed crash kernel, but "+kexecCrashLoadedPath+
				" does not report one; apply the kdump_tools requirement first, or set oops_policy=log-and-continue")
		return status
	}

	pending := []string{}

	// Live sysctl values reflect what the kernel actually uses, not what's
	// in the drop-in. Drift is possible (operator did a one-shot sysctl -w
	// that didn't get persisted; a subsequent reboot flipped them back).
	for _, p := range managedSysctls(resolved) {
		if got := readSysctlValue(p.Name); got != p.Value {
			pending = append(pending, fmt.Sprintf("%s=%d (current: %d)", p.Name, p.Value, got))
		}
	}

	if !hostreqkit.FileContentMatches(sysctlPath, buildSysctlContent(resolved)) {
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
	if err := hostreqkit.InstallManagedContent(sysctlPath, buildSysctlContent(resolvePolicy(status.Config)), opts.SudoMode, opts); err != nil {
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

func buildSysctlContent(p policy) string {
	var b strings.Builder
	b.WriteString("# Managed by Vrooli -- do not edit manually\n")
	b.WriteString("# Make kernel hangs and oopses produce diagnostics rather than silent hard resets.\n")
	b.WriteString("# See internal/safeguards/host-hardening/handler.go for rationale.\n")
	fmt.Fprintf(&b, "# oops_policy=%s softlockup_policy=%s\n", p.OopsPolicy, p.SoftlockupPolicy)
	for _, setting := range managedSysctls(p) {
		fmt.Fprintf(&b, "%s = %d\n", setting.Name, setting.Value)
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
