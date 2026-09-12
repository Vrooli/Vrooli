package remotesessionprotection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/daemonreload"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

const (
	percentDivisor = 100
	bytesPerKiB    = 1024
	bytesPerMiB    = bytesPerKiB * bytesPerKiB
	bytesPerGiB    = bytesPerMiB * bytesPerKiB
	miBPerGiB      = 1024
)

// ── Static protection files (always applied on Linux with sysctl/systemd) ────

const (
	sysctlPath    = "/etc/sysctl.d/99-vrooli-remote-session-protection.conf"
	systemdDir    = "/etc/systemd/system/user@.service.d"
	systemdPath   = "/etc/systemd/system/user@.service.d/90-vrooli-remote-session-protection.conf"
	logindDir     = "/etc/systemd/logind.conf.d"
	logindPath    = "/etc/systemd/logind.conf.d/90-vrooli-remote-session-protection.conf"
	sysctlContent = "vm.swappiness = 10\nvm.oom_kill_allocating_task = 0\n"
	unitContent   = "[Service]\nOOMScoreAdjust=-900\n"
	logindContent = "[Login]\nKillUserProcesses=no\n"
)

// ── Desktop-aware protection files (applied when GUI/remote desktop detected) ─

const (
	desktopMinPercent   = 15
	desktopMinMB        = 4096
	desktopBufferMB     = 2048
	workloadHighPercent = 85
	workloadMaxPercent  = 95
	maxSwapGB           = 64
	swapFile            = "/swapfile"
	desktopUID          = 1000

	desktopSliceDir   = "/etc/systemd/system/user-1000.slice.d"
	desktopSlicePath  = "/etc/systemd/system/user-1000.slice.d/50-memory-protect.conf"
	workloadSlicePath = "/etc/systemd/system/workload.slice"
	dockerDaemonJSON  = dockerhost.DaemonConfigPath
)

// GUI processes checked for desktop detection. Display-manager names come
// from hostinventory.DisplayManagerNames, the repository-wide vocabulary.
var guiProcesses = []string{"Xorg", "Xwayland", "gnome-shell", "kde", "xfce"}

// memoryAllocation holds computed thresholds derived from system RAM.
type memoryAllocation struct {
	systemMB       int
	desktopMinMB   int
	desktopLowMB   int
	workloadHighMB int
	workloadMaxMB  int
	targetSwapGB   int
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// ── Inspect ──────────────────────────────────────────────────────────────────

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "remote session protection is only supported on Linux hosts")
		return status
	}
	if !host.SupportsSysctl && !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not expose sysctl or systemd hooks needed for safeguard application")
		return status
	}

	pending := inspectStaticFiles(host)
	desktopPending := inspectDesktopProtection(host)
	pending = append(pending, desktopPending...)

	// Surface competing swap areas (e.g. Ubuntu installer's /swap.img sitting
	// next to the safeguard-managed /swapfile). Informational only — operator
	// decides whether to dedupe.
	if competing := CompetingSwapsFn(); len(competing) > 0 {
		status.Notes = append(status.Notes, fmt.Sprintf(
			"competing swap area(s) active alongside %s: %s — operator may dedupe with `sudo swapoff <path> && sudo sed -i '\\|^<path>\\b|d' /etc/fstab && sudo rm <path>` (replace <path> with each entry)",
			swapFile, strings.Join(competing, ", ")))
	}

	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "all remote session protection files are present and match the managed Vrooli configuration")
		return status
	}

	if host.SupportsSysctl {
		status.Notes = append(status.Notes, "will manage Linux memory-pressure defaults via "+sysctlPath)
	}
	if host.SupportsSystemd {
		status.Notes = append(status.Notes, "will protect user sessions with managed systemd overrides")
	}
	status.Notes = append(status.Notes, "pending managed files: "+strings.Join(pending, ", "))
	return status
}

func inspectStaticFiles(host hostreqkit.Host) []string {
	var pending []string
	if host.SupportsSysctl {
		if !hostreqkit.FileContentMatches(sysctlPath, sysctlContent) {
			pending = append(pending, sysctlPath)
		}
	}
	if host.SupportsSystemd {
		if !hostreqkit.FileContentMatches(systemdPath, unitContent) {
			pending = append(pending, systemdPath)
		}
		if !hostreqkit.FileContentMatches(logindPath, logindContent) {
			pending = append(pending, logindPath)
		}
	}
	return pending
}

func inspectDesktopProtection(host hostreqkit.Host) []string {
	if !host.SupportsSystemd {
		return nil
	}
	if !isDesktopInstalled() {
		return nil
	}

	alloc, err := calculateMemory()
	if err != nil {
		return []string{"(unable to read host inventory memory)"}
	}

	var pending []string

	// Desktop slice
	want := desktopSliceContent(alloc)
	if !hostreqkit.FileContentMatches(desktopSlicePath, want) {
		pending = append(pending, desktopSlicePath)
	}

	// Workload slice
	if !hostreqkit.FileContentMatches(workloadSlicePath, workloadSliceContent()) {
		pending = append(pending, workloadSlicePath)
	}

	// Swap sufficiency
	currentSwapGB := readCurrentSwapGB()
	if currentSwapGB < alloc.targetSwapGB {
		pending = append(pending, swapFile)
	}

	// Docker config (only when Docker is installed)
	if hostreqkit.CommandAvailable("docker") {
		if !dockerConfigApplied() {
			pending = append(pending, dockerDaemonJSON)
		}
	}

	return pending
}

// ── Apply ────────────────────────────────────────────────────────────────────

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
		status.Notes = append(status.Notes, "dry-run: would apply remote session protection")
		return status, nil
	}

	// Phase 1: static sysctl + systemd files (always)
	if err := applyStaticFiles(host, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	// Phase 2: desktop-aware memory protection (when GUI detected)
	if host.SupportsSystemd && isDesktopInstalled() {
		if err := applyDesktopProtection(host, opts); err != nil {
			// Desktop protection is best-effort; don't fail the entire safeguard
			status.Notes = append(status.Notes, "desktop protection partially applied: "+err.Error())
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "applied managed sysctl and systemd safeguards for remote Linux sessions")
	status.Notes = append(status.Notes, "existing login sessions may need to reconnect before all systemd protections take effect")
	return status, nil
}

func applyStaticFiles(host hostreqkit.Host, opts hostreqkit.EnsureOptions) error {
	if host.SupportsSysctl {
		if err := hostreqkit.EnsureManagedDir(filepath.Dir(sysctlPath), opts.SudoMode, opts); err != nil {
			return fmt.Errorf("create sysctl dir: %w", err)
		}
		if err := hostreqkit.InstallManagedContent(sysctlPath, sysctlContent, opts.SudoMode, opts); err != nil {
			return fmt.Errorf("install sysctl config: %w", err)
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sysctl", []string{"--system"}, opts); err != nil {
			return fmt.Errorf("apply sysctl: %w", err)
		}
	}
	if host.SupportsSystemd {
		for _, dir := range []string{systemdDir, logindDir} {
			if err := hostreqkit.EnsureManagedDir(dir, opts.SudoMode, opts); err != nil {
				return fmt.Errorf("create dir %s: %w", dir, err)
			}
		}
		for _, file := range []struct {
			path    string
			content string
		}{
			{systemdPath, unitContent},
			{logindPath, logindContent},
		} {
			if err := hostreqkit.InstallManagedContent(file.path, file.content, opts.SudoMode, opts); err != nil {
				return fmt.Errorf("install %s: %w", file.path, err)
			}
		}
		if _, err := daemonreload.Reload(context.Background(), daemonreload.CurrentRoot(), opts); err != nil {
			return err
		}
	}
	return nil
}

func applyDesktopProtection(host hostreqkit.Host, opts hostreqkit.EnsureOptions) error {
	alloc, err := calculateMemory()
	if err != nil {
		return fmt.Errorf("calculate memory: %w", err)
	}

	// 1. Swap
	if err := ensureSwap(alloc, opts); err != nil {
		return fmt.Errorf("configure swap: %w", err)
	}

	// 2. Desktop slice
	if err := hostreqkit.EnsureManagedDir(desktopSliceDir, opts.SudoMode, opts); err != nil {
		return fmt.Errorf("create desktop slice dir: %w", err)
	}
	if err := hostreqkit.InstallManagedContent(desktopSlicePath, desktopSliceContent(alloc), opts.SudoMode, opts); err != nil {
		return fmt.Errorf("install desktop slice config: %w", err)
	}

	// Apply to running cgroup v2 if available (best-effort).
	//
	// Why not InstallManagedContent: install(1) replaces the destination by
	// unlink + rename, which fails on /sys/fs/cgroup pseudo-files
	// ("Operation not permitted") and leaks the error to stderr even when
	// the caller discards the return value. cgroup attribute files accept
	// a single write of the new value — stream the bytes via tee with
	// stderr suppressed and skip the call entirely if the file is missing.
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice", desktopUID)
	minBytes := strconv.Itoa(alloc.desktopMinMB * bytesPerMiB)
	lowBytes := strconv.Itoa(alloc.desktopLowMB * bytesPerMiB)
	writeCgroupValue := func(path, value string) {
		if _, err := os.Stat(path); err != nil {
			return
		}
		_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sh",
			[]string{"-c", fmt.Sprintf("printf %%s %s > %s 2>/dev/null", value, path)}, opts)
	}
	writeCgroupValue(cgroupPath+"/memory.min", minBytes)
	writeCgroupValue(cgroupPath+"/memory.low", lowBytes)

	// 3. Workload slice
	if err := hostreqkit.InstallManagedContent(workloadSlicePath, workloadSliceContent(), opts.SudoMode, opts); err != nil {
		return fmt.Errorf("install workload slice: %w", err)
	}

	// 4. Docker daemon.json (only if Docker is installed)
	if hostreqkit.CommandAvailable("docker") {
		if err := applyDockerConfig(opts); err != nil {
			// Docker config is best-effort — don't fail desktop protection
			return nil
		}
	}

	// Final daemon-reload to pick up new slices and repair any GPU containers
	// affected by the reload.
	if _, err := daemonreload.Reload(context.Background(), daemonreload.CurrentRoot(), opts); err != nil {
		return err
	}

	return nil
}

// ── Desktop detection ────────────────────────────────────────────────────────

func isDesktopInstalled() bool {
	// Check for display manager systemd units
	out, err := hostreqkit.CombinedOutputFn("systemctl", "list-unit-files", "--no-pager", "--no-legend")
	if err == nil {
		units := string(out)
		for _, dm := range hostinventory.DisplayManagerNames {
			if strings.Contains(units, dm+".service") {
				return true
			}
		}
	}

	// Check for running GUI processes
	for _, proc := range guiProcesses {
		if _, err := hostreqkit.CombinedOutputFn("pgrep", "-x", proc); err == nil {
			return true
		}
	}

	return false
}

// ── Memory calculation ───────────────────────────────────────────────────────

func calculateMemory() (memoryAllocation, error) {
	snapshot, err := hostSnapshot()
	if err != nil {
		return memoryAllocation{}, err
	}
	if snapshot.Memory.TotalBytes == 0 {
		return memoryAllocation{}, fmt.Errorf("host inventory reported no total memory")
	}

	memMB := int(snapshot.Memory.TotalBytes / bytesPerMiB)

	dMin := memMB * desktopMinPercent / percentDivisor
	if dMin < desktopMinMB {
		dMin = desktopMinMB
	}
	dLow := dMin + desktopBufferMB

	wHigh := memMB * workloadHighPercent / percentDivisor
	wMax := memMB * workloadMaxPercent / percentDivisor

	swapGB := memMB / miBPerGiB
	if swapGB > maxSwapGB {
		swapGB = maxSwapGB
	}

	return memoryAllocation{
		systemMB:       memMB,
		desktopMinMB:   dMin,
		desktopLowMB:   dLow,
		workloadHighMB: wHigh,
		workloadMaxMB:  wMax,
		targetSwapGB:   swapGB,
	}, nil
}

func readCurrentSwapGB() int {
	snapshot, err := hostSnapshot()
	if err != nil {
		return 0
	}
	return int(snapshot.Swap.TotalBytes / bytesPerGiB)
}

func hostSnapshot() (hostinventory.Snapshot, error) {
	collector := hostinventory.SystemCollector()
	collector.Commands = &shelltest.Fake{}
	collector.Files = hostReqFileReader{}
	collector.GOOS = string(hostreqspec.PlatformLinux)
	return collector.Collect(context.Background())
}

type hostReqFileReader struct{}

func (hostReqFileReader) ReadFile(path string) ([]byte, error) {
	return hostreqkit.ReadFileFn(path)
}

// ── Content generators ───────────────────────────────────────────────────────

func desktopSliceContent(alloc memoryAllocation) string {
	return fmt.Sprintf(`[Slice]
# Vrooli Remote Session Protection
# Hard reservation: cannot be reclaimed under memory pressure
MemoryMin=%dM
# Soft preference: try to keep at least this much
MemoryLow=%dM
# Don't kill desktop processes first
ManagedOOMPreference=omit
`, alloc.desktopMinMB, alloc.desktopLowMB)
}

func workloadSliceContent() string {
	return fmt.Sprintf(`[Unit]
Description=Slice for batch jobs and AI workloads
Documentation=https://github.com/Vrooli/Vrooli

[Slice]
# Start throttling at high watermark
MemoryHigh=%d%%
# Hard cap at maximum
MemoryMax=%d%%
# Prefer killing workload processes under pressure
ManagedOOMMemoryPressure=kill
ManagedOOMMemoryPressureLimit=60%%
# Lower OOM score preference (kill these first)
ManagedOOMPreference=avoid
`, workloadHighPercent, workloadMaxPercent)
}

// ── Swap management ──────────────────────────────────────────────────────────

// CompetingSwapsFn returns the paths of active swap areas that are not the
// safeguard's managed swapFile. Stubbed in tests.
//
// Why this exists: ensureSwap reads SwapTotal from /proc/meminfo, which is
// the *aggregate* across all swap areas. On Ubuntu the installer creates
// /swap.img and `vrooli setup` then writes /swapfile alongside it — both
// stay active, SwapTotal looks correct, and the safeguard never realizes
// the host has two competing swaps until a future resize, fstab edit, or
// disk-cleanup hits inconsistent state. We surface a Note so the operator
// can dedupe; we deliberately do *not* swapoff/rm here because some
// operators legitimately maintain multiple swap areas (encrypted swap on a
// separate device, hibernate swap partition, etc.). Detect-and-warn, not
// detect-and-destroy.
var CompetingSwapsFn = func() []string {
	out, err := hostreqkit.CombinedOutputFn("swapon", "--show=NAME", "--noheadings")
	if err != nil {
		return nil
	}
	var competing []string
	for _, line := range strings.Split(string(out), "\n") {
		path := strings.TrimSpace(line)
		if path == "" || path == swapFile {
			continue
		}
		competing = append(competing, path)
	}
	return competing
}

func ensureSwap(alloc memoryAllocation, opts hostreqkit.EnsureOptions) error {
	currentGB := readCurrentSwapGB()
	if currentGB >= alloc.targetSwapGB {
		return nil
	}

	target := strconv.Itoa(alloc.targetSwapGB)

	// Disable existing swap file if present
	_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "swapoff", []string{swapFile}, opts)

	// Try fallocate first, fall back to dd
	err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "fallocate", []string{"-l", target + "G", swapFile}, opts)
	if err != nil {
		err = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "dd", []string{
			"if=/dev/zero", "of=" + swapFile, "bs=1G", "count=" + target, "status=none",
		}, opts)
		if err != nil {
			return fmt.Errorf("create swap file: %w", err)
		}
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chmod", []string{"600", swapFile}, opts); err != nil {
		return fmt.Errorf("chmod swap file: %w", err)
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "mkswap", []string{swapFile}, opts); err != nil {
		return fmt.Errorf("mkswap: %w", err)
	}

	// Add to fstab if not already there
	fstab, _ := hostreqkit.ReadFileFn("/etc/fstab")
	if !strings.Contains(string(fstab), swapFile) {
		fstabLine := swapFile + " none swap sw 0 0\n"
		if err := hostreqkit.InstallManagedContent("/etc/fstab.vrooli-swap", fstabLine, opts.SudoMode, opts); err == nil {
			// Append via shell to preserve existing fstab
			_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sh", []string{
				"-c", "cat /etc/fstab.vrooli-swap >> /etc/fstab && rm -f /etc/fstab.vrooli-swap",
			}, opts)
		}
	}

	_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "swapon", []string{swapFile}, opts)
	return nil
}

// ── Docker daemon.json ───────────────────────────────────────────────────────

func dockerConfigApplied() bool {
	return dockerhost.ConfigHasWorkloadPolicy(dockerDaemonJSON)
}

func applyDockerConfig(opts hostreqkit.EnsureOptions) error {
	result, err := dockerhost.SanitizeDaemonConfig(dockerDaemonJSON, dockerhost.ConfigOptions{
		ApplyWorkloadCgroupPolicy: true,
	}, opts)
	if err != nil {
		return err
	}
	if result.Changed {
		_ = dockerhost.RestartDockerIfActive(opts)
	}
	return nil
}
