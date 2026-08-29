// Package nvidiadriver owns the host-side NVIDIA driver lifecycle, across two
// readiness dimensions that fail independently.
//
// Driver presence: GPU consumers cannot compensate for a missing host kernel
// module. This safeguard detects NVIDIA hardware independently of nvidia-smi
// (which is itself unusable when the module is broken), repairs the module
// package for the running Ubuntu kernel, retains the installed kernel-meta
// package for future kernel upgrades, and reports the unavoidable reboot as a
// typed state.
//
// Device-node durability: a driver that answers NVML is still not a host a GPU
// consumer can rely on if the compute device nodes only materialise while some
// other client holds the driver open. That race is silent in both directions —
// the host reports full health and the consumer reports healthy while running
// on CPU — so it is repaired here rather than left to each consumer.
package nvidiadriver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostcapability"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const pciDevicesPath = "/sys/bus/pci/devices"

// A ready driver is necessary but not sufficient. The distribution unit runs
// nvidia-persistenced with --no-persistence-mode and StopWhenUnneeded=true, so
// the NVIDIA compute device nodes (/dev/nvidia0, /dev/nvidia-uvm) exist only
// while some other client already holds the driver open. Creating them on
// demand needs nvidia-modprobe, which is setuid-root and is not part of the
// -open driver packaging, so an unprivileged process cannot bring them up
// itself. A resource that starts before the first client therefore observes a
// host with no usable GPU, falls back to CPU, and never re-probes — while
// nvidia-smi, the PCI scan and NVML all report perfect health.
//
// Persistence mode keeps the driver initialised and the device nodes present
// from boot, which removes the race for every GPU consumer at once.
const (
	persistencedUnit     = "nvidia-persistenced"
	persistenceDropInDir = "/etc/systemd/system/nvidia-persistenced.service.d"
	persistenceDropIn    = persistenceDropInDir + "/10-vrooli-persistence.conf"
	persistenceContent   = `# Managed by Vrooli -- do not edit manually
# Keeps the NVIDIA compute device nodes present from boot so GPU-declaring
# resources cannot start into a host that looks GPU-less and fall back to CPU.
[Unit]
StopWhenUnneeded=false

[Service]
ExecStart=
ExecStart=/usr/bin/nvidia-persistenced --user nvidia-persistenced --persistence-mode --verbose

[Install]
WantedBy=multi-user.target
`
)

var (
	readDirFn = os.ReadDir
	// PersistenceModeReadyFn reports whether every NVIDIA GPU has persistence
	// mode enabled. It is deliberately queried from the driver rather than
	// inferred from the drop-in file: an installed unit override that has not
	// been loaded yet is not a working persistence mode.
	PersistenceModeReadyFn = func() bool {
		out, err := hostreqkit.CombinedOutputFn("nvidia-smi", "--query-gpu=persistence_mode", "--format=csv,noheader")
		if err != nil {
			return false
		}
		lines := 0
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lines++
			if !strings.EqualFold(line, "Enabled") {
				return false
			}
		}
		return lines > 0
	}
	// ComputeDeviceAccessFn reports whether the invoking user can open the
	// NVIDIA compute device nodes. It is deliberately an open attempt rather
	// than a presence check: a node the user cannot open exists, is a character
	// device, and still cannot serve compute. That gap is the whole silent-CPU
	// failure mode, and nvidia-smi answering does not close it — nvidia-smi
	// opens /dev/nvidiactl, not /dev/nvidia0 and /dev/nvidia-uvm, which are the
	// nodes a CUDA context needs.
	ComputeDeviceAccessFn = func() (nodes []string, openable []string) {
		snapshot, err := hostinventory.SystemCollector().CollectGPUFacts(context.Background())
		if err != nil {
			return nil, nil
		}
		return snapshot.NvidiaDeviceNodes, snapshot.OpenableDeviceNodes
	}
	// PersistencedPresentFn reports whether the persistence daemon this
	// safeguard configures actually exists on the host.
	PersistencedPresentFn = func() bool {
		_, err := hostreqkit.LookPathFn(persistencedUnit)
		return err == nil
	}
	// DriverReadyFn is deliberately based on the driver API rather than a
	// module filename: it proves the kernel module, userspace driver and NVML
	// are mutually compatible.
	DriverReadyFn = func() bool {
		if _, err := hostreqkit.LookPathFn("nvidia-smi"); err != nil {
			return false
		}
		_, err := hostreqkit.CombinedOutputFn("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
		return err == nil
	}
	RunningKernelFn = func() (string, error) {
		out, err := hostreqkit.CombinedOutputFn("uname", "-r")
		return strings.TrimSpace(string(out)), err
	}
	InstalledPackagesFn = func() ([]string, error) {
		args := []string{"-W", "-f=${binary:Package}\\t${db:Status-Abbrev}\\n"}
		args = append(args, hostcapability.NvidiaPackageQueryPatterns()...)
		out, err := hostreqkit.CombinedOutputFn("dpkg-query", args...)
		if err != nil {
			return nil, err
		}
		var packages []string
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.HasPrefix(fields[1], "ii") {
				packages = append(packages, fields[0])
			}
		}
		return packages, nil
	}
	PackageAvailableFn = func(name string) bool {
		_, err := hostreqkit.CombinedOutputFn("apt-cache", "show", name)
		return err == nil
	}
	RemoteDesktopActiveFn = func() bool {
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if _, err := strconv.ParseUint(entry.Name(), 10, 32); err != nil {
				continue
			}
			comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
			if err != nil {
				continue
			}
			switch strings.TrimSpace(string(comm)) {
			case "gnome-remote-de", "xrdp", "xrdp-sesman", "wayvnc", "Xvnc", "Xtigervnc", "vino-server", "sunshine":
				return true
			}
		}
		return false
	}
	// RemoteDesktopStateFn keeps an unreadable session state distinct from a
	// confirmed inactive session. Unknown live state needs operator consent.
	RemoteDesktopStateFn = func() (bool, bool) { return RemoteDesktopActiveFn(), true }
)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if requirement.Manual {
		status.SupportClass, status.ExecutionState = hostreqkit.SupportManualOnly, hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass, status.ExecutionState = hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "NVIDIA kernel-driver repair is currently implemented for Ubuntu/Linux hosts")
		return status
	}
	if !hasNvidiaDisplayController() {
		status.SupportClass, status.ExecutionState = hostreqkit.SupportNotApplicable, hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "no NVIDIA display controller found in PCI sysfs; CPU and other accelerator paths remain unaffected")
		return status
	}
	if DriverReadyFn() {
		status.Installed = true
		status.Notes = append(status.Notes, "NVIDIA PCI hardware and NVML driver are ready")
		// Device access is reported before persistence, because it is true or
		// false independently of it: a host with persistence enabled can still
		// deny this user the compute nodes, and a host without persistence can
		// still be granting them right now.
		status = inspectComputeDeviceAccess(status)
		return inspectDeviceNodeDurability(host, status)
	}
	if host.PackageManager != "apt" && host.PackageManager != "apt-get" {
		status.SupportClass, status.ExecutionState = hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "NVIDIA hardware is present but the driver is not ready; automatic repair currently requires Ubuntu apt packages")
		return status
	}
	packages, err := repairPackages()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("cannot derive a safe NVIDIA repair transaction: %v", err))
		return status
	}
	status.PackageName = strings.Join(packages, ", ")
	status.InstallSupported = true
	status.Notes = append(status.Notes, "NVIDIA PCI hardware is present but NVML cannot communicate with the kernel driver", "will install the driver and kernel-module packages for the running kernel: "+strings.Join(packages, ", "))
	return status
}

// inspectDeviceNodeDurability is the second readiness dimension: a working
// driver that only materialises its compute device nodes on demand is still a
// host where a GPU consumer can start too early and silently land on CPU.
func inspectDeviceNodeDurability(host hostreqkit.Host, status hostreqkit.ItemStatus) hostreqkit.ItemStatus {
	alreadyPresent := func(note string) hostreqkit.ItemStatus {
		status.Applied, status.ExecutionState = true, hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, note)
		return status
	}
	if !host.SupportsSystemd {
		return alreadyPresent("device-node persistence is managed through a systemd unit override; this host has no systemd, so GPU consumers must verify their own placement")
	}
	if !PersistencedPresentFn() {
		return alreadyPresent("nvidia-persistenced is not installed, so device-node persistence cannot be configured here; GPU consumers must verify their own placement")
	}
	if hostreqkit.FileContentMatches(persistenceDropIn, persistenceContent) && PersistenceModeReadyFn() {
		return alreadyPresent("NVIDIA persistence mode is enabled, so the compute device nodes survive from boot with no client attached")
	}
	status.InstallSupported = true
	status.Notes = append(status.Notes,
		"NVIDIA persistence mode is not durably enabled: the compute device nodes exist only while another client holds the driver open",
		"a resource that starts before the first client observes a GPU-less host, falls back to CPU, and never re-probes",
		"will install "+persistenceDropIn+" and reload nvidia-persistenced with --persistence-mode")
	return status
}

// inspectComputeDeviceAccess is the third readiness dimension. Driver presence
// and persistence say the device nodes will exist; this says the invoking user
// can actually open them. A resource that cannot open the node falls back to
// the CPU while every other dimension reports green, so the dimension is
// reported even when nothing here can repair it: an unopenable node is a group
// membership or udev rule question, and both belong to the operator.
func inspectComputeDeviceAccess(status hostreqkit.ItemStatus) hostreqkit.ItemStatus {
	nodes, openable := ComputeDeviceAccessFn()
	if len(nodes) == 0 {
		status.Notes = append(status.Notes, "no NVIDIA compute device nodes are present yet; they appear when a client first opens the driver, and a resource that starts before that lands on the CPU")
		return status
	}
	if len(openable) > 0 {
		status.Notes = append(status.Notes, fmt.Sprintf("compute device nodes are openable by this user: %s", strings.Join(openable, ", ")))
		return status
	}
	status.BlockingReason = hostreqkit.BlockingManual
	status.Notes = append(status.Notes,
		fmt.Sprintf("NVIDIA compute device nodes exist but this user cannot open any of them: %s", strings.Join(nodes, ", ")),
		"presence is not access: a resource will start, fail to create a CUDA context, and silently run on the CPU",
		"add this user to the group that owns the device nodes, or install a udev rule that grants access, then re-run `vrooli setup`")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.ExecutionState != hostreqkit.ExecutionPending {
		return status, nil
	}
	// Inspect produces exactly two pending plans and they are distinguishable
	// by PackageName: the driver repair names the packages it will install,
	// the device-node persistence plan names none. applyPersistence re-verifies
	// its own preconditions, so an unrecognised empty plan fails cleanly rather
	// than acting on a host it did not inspect.
	packages := splitPackages(status.PackageName)
	if len(packages) == 0 {
		return applyPersistence(status, opts)
	}
	remoteDesktopActive, remoteDesktopDetermined := RemoteDesktopStateFn()
	if !remoteDesktopDetermined && !opts.MaintenanceWindow {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingUndeterminedNeedsConsent
		status.Notes = append(status.Notes, "remote-desktop state could not be determined; operator consent is required before changing the live driver")
		return status, nil
	}
	if remoteDesktopActive && !opts.MaintenanceWindow {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingNeedsMaintenanceWindow
		status.Notes = append(status.Notes,
			"active remote-desktop server detected; installing this module can immediately reconfigure DRM/VGA and disconnect the session",
			"schedule a maintenance window with console or SSH recovery, then rerun `vrooli host safeguard nvidia_driver --maintenance-window --sudo-mode ask`")
		return status, nil
	}
	args := append([]string{"install", "-y", "--no-install-recommends"}, packages...)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would run apt-get "+strings.Join(args, " "))
		return status, nil
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "apt-get", args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("NVIDIA driver repair failed: %v", err))
		if hostreqkit.IsSudoSkipped(err) {
			status.Notes = append(status.Notes, "privilege is required: rerun `vrooli host safeguard nvidia_driver --sudo-mode ask` from an interactive terminal")
		}
		return status, nil
	}
	status.Applied, status.Installed = true, true
	status.ExecutionState = hostreqkit.ExecutionRebootRequired
	status.Notes = append(status.Notes, "NVIDIA packages were installed for the running kernel; reboot, then rerun `vrooli setup --resources ollama` to verify NVML and GPU-container readiness")
	return status, nil
}

// applyPersistence installs the unit override that keeps the NVIDIA compute
// device nodes present from boot.
//
// Unlike the driver-package repair this is not gated behind a maintenance
// window: enabling persistence mode is an online driver-state change that does
// not reconfigure DRM/VGA and cannot disconnect a display or remote session.
func applyPersistence(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if !PersistencedPresentFn() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "nvidia-persistenced is not installed, so no device-node persistence plan could be applied")
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install "+persistenceDropIn+", then reload and restart "+persistencedUnit)
		return status, nil
	}
	if err := hostreqkit.EnsureManagedDir(persistenceDropInDir, opts.SudoMode, opts); err != nil {
		return failPersistence(status, err)
	}
	if err := hostreqkit.InstallManagedContent(persistenceDropIn, persistenceContent, opts.SudoMode, opts); err != nil {
		return failPersistence(status, err)
	}
	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", persistencedUnit},
		{"restart", persistencedUnit},
	} {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", args, opts); err != nil {
			return failPersistence(status, fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), err))
		}
	}
	status.Applied = true
	if !PersistenceModeReadyFn() {
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes, persistenceDropIn+" is installed but the driver does not report persistence mode yet; it takes effect on the next boot")
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "NVIDIA persistence mode is enabled; the compute device nodes now survive from boot with no client attached")
	return status, nil
}

func failPersistence(status hostreqkit.ItemStatus, err error) (hostreqkit.ItemStatus, error) {
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "device-node persistence repair failed: "+err.Error())
	if hostreqkit.IsSudoSkipped(err) {
		status.Notes = append(status.Notes, "privilege is required: rerun `vrooli host safeguard nvidia_driver --sudo-mode ask` from an interactive terminal")
	}
	return status, nil
}

func hasNvidiaDisplayController() bool {
	entries, err := readDirFn(pciDevicesPath)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		base := filepath.Join(pciDevicesPath, entry.Name())
		vendor, vendorErr := hostreqkit.ReadFileFn(filepath.Join(base, "vendor"))
		class, classErr := hostreqkit.ReadFileFn(filepath.Join(base, "class"))
		if vendorErr == nil && classErr == nil && strings.EqualFold(strings.TrimSpace(string(vendor)), "0x10de") && strings.HasPrefix(strings.TrimSpace(string(class)), "0x03") {
			return true
		}
	}
	return false
}

func repairPackages() ([]string, error) {
	kernel, err := RunningKernelFn()
	if err != nil || kernel == "" {
		return nil, fmt.Errorf("read running kernel: %w", err)
	}
	installed, err := InstalledPackagesFn()
	if err != nil {
		return nil, fmt.Errorf("query installed NVIDIA packages: %w", err)
	}
	driver := ""
	for _, pkg := range installed {
		if hostcapability.IsNvidiaDriverPackage(pkg) {
			driver = pkg
			break
		}
	}
	if driver == "" {
		return nil, fmt.Errorf("no installed nvidia-driver metapackage identifies the supported driver branch")
	}
	modulePrefix, ok := hostcapability.NvidiaModulePackagePrefix(driver)
	if !ok {
		return nil, fmt.Errorf("installed driver package %q has no supported module-package mapping", driver)
	}
	current, ok := hostcapability.DeriveNvidiaModulePackage(driver, kernel)
	if !ok {
		return nil, fmt.Errorf("running kernel %q has no supported module-package mapping", kernel)
	}
	if !PackageAvailableFn(current) {
		return nil, fmt.Errorf("Ubuntu repository has no module package %q for running kernel", current)
	}
	packages := []string{driver, current}
	for _, pkg := range installed {
		if strings.HasPrefix(pkg, modulePrefix+"-") && !strings.Contains(pkg, kernel) && !regexp.MustCompile(`-[0-9]+\.[0-9]+\.[0-9]+-`).MatchString(pkg) && PackageAvailableFn(pkg) {
			packages = append(packages, pkg)
		}
	}
	slices.Sort(packages)
	return unique(packages), nil
}

func splitPackages(value string) []string {
	return unique(strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' }))
}

func unique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
