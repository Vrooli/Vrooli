// Package nvidiadriver owns the host-side NVIDIA kernel-driver lifecycle.
//
// GPU containers cannot compensate for a missing host kernel module. This
// safeguard detects NVIDIA hardware independently of nvidia-smi (which is
// itself unusable when the module is broken), repairs the module package for
// the running Ubuntu kernel, retains the installed kernel-meta package for
// future kernel upgrades, and reports the unavoidable reboot as a typed state.
package nvidiadriver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const pciDevicesPath = "/sys/bus/pci/devices"

var (
	readDirFn = os.ReadDir
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
		out, err := hostreqkit.CombinedOutputFn("dpkg-query", "-W", "-f=${binary:Package}\\t${db:Status-Abbrev}\\n", "nvidia-driver-*", "linux-modules-nvidia-*")
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
)

var driverPackageRE = regexp.MustCompile(`^nvidia-driver-([0-9]+(?:-[a-z]+)?)(?:-(open|server(?:-open)?))?$`)

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
	if host.OS != "linux" {
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
		status.Applied, status.Installed, status.ExecutionState = true, true, hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "NVIDIA PCI hardware and NVML driver are ready")
		return status
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

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.ExecutionState != hostreqkit.ExecutionPending {
		return status, nil
	}
	packages := splitPackages(status.PackageName)
	if len(packages) == 0 {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "no validated NVIDIA package repair plan was available")
		return status, nil
	}
	args := append([]string{"install", "-y", "--no-install-recommends"}, packages...)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would run apt-get "+strings.Join(args, " "))
		return status, nil
	}
	if RemoteDesktopActiveFn() && !opts.MaintenanceWindow {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingNeedsMaintenanceWindow
		status.Notes = append(status.Notes,
			"active remote-desktop server detected; installing this module can immediately reconfigure DRM/VGA and disconnect the session",
			"schedule a maintenance window with console or SSH recovery, then rerun `vrooli host safeguard nvidia_driver --maintenance-window --sudo-mode ask`")
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
		if driverPackageRE.MatchString(pkg) {
			driver = pkg
			break
		}
	}
	if driver == "" {
		return nil, fmt.Errorf("no installed nvidia-driver metapackage identifies the supported driver branch")
	}
	match := driverPackageRE.FindStringSubmatch(driver)
	branch, flavor := match[1], match[2]
	modulePrefix := "linux-modules-nvidia-" + branch
	if flavor != "" {
		modulePrefix += "-" + flavor
	}
	current := modulePrefix + "-" + kernel
	if !PackageAvailableFn(current) {
		return nil, fmt.Errorf("Ubuntu repository has no module package %q for running kernel", current)
	}
	packages := []string{driver, current}
	for _, pkg := range installed {
		if strings.HasPrefix(pkg, modulePrefix+"-") && !strings.Contains(pkg, kernel) && !regexp.MustCompile(`-[0-9]+\.[0-9]+\.[0-9]+-`).MatchString(pkg) && PackageAvailableFn(pkg) {
			packages = append(packages, pkg)
		}
	}
	sort.Strings(packages)
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
