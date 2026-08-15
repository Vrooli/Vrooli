package hostinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	platformgo "github.com/vrooli/platform-go"
)

// IntegrityCommandRunner and IntegrityFileSystem are the control-plane seams
// for host-integrity probes. They keep scenarios from owning host detection
// while still allowing deterministic fixture tests.
type IntegrityCommandRunner interface {
	LookPath(string) (string, error)
	CombinedOutput(context.Context, string, ...string) ([]byte, error)
}

type IntegrityFileSystem interface {
	ReadFile(string) ([]byte, error)
	ReadDir(string) ([]os.DirEntry, error)
	Stat(string) (os.FileInfo, error)
}

type IntegrityCollector interface {
	Collect(context.Context) (HostInventory, error)
}

type IntegrityCollectorOptions struct {
	Commands IntegrityCommandRunner
	Files    IntegrityFileSystem
	GOOS     string
	GOARCH   string
	Now      func() time.Time
}

type IntegrityDefaultCollector struct {
	commands IntegrityCommandRunner
	files    IntegrityFileSystem
	goos     string
	goarch   string
	now      func() time.Time
}

type nativeIntegrityCommands struct{}

func (nativeIntegrityCommands) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (nativeIntegrityCommands) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type nativeIntegrityFiles struct{}

func (nativeIntegrityFiles) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (nativeIntegrityFiles) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }
func (nativeIntegrityFiles) Stat(path string) (os.FileInfo, error)      { return os.Stat(path) }

func NewIntegrityCollector(options IntegrityCollectorOptions) *IntegrityDefaultCollector {
	if options.Commands == nil {
		options.Commands = nativeIntegrityCommands{}
	}
	if options.Files == nil {
		options.Files = nativeIntegrityFiles{}
	}
	if options.GOOS == "" {
		options.GOOS = runtime.GOOS
	}
	if options.GOARCH == "" {
		options.GOARCH = runtime.GOARCH
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &IntegrityDefaultCollector{commands: options.Commands, files: options.Files, goos: options.GOOS, goarch: options.GOARCH, now: options.Now}
}

type CachedIntegrityCollector struct {
	inner IntegrityCollector
	ttl   time.Duration
	mu    sync.Mutex
	last  HostInventory
	err   error
}

func NewCachedIntegrityCollector(inner IntegrityCollector, ttl time.Duration) *CachedIntegrityCollector {
	return &CachedIntegrityCollector{inner: inner, ttl: ttl}
}

func (c *CachedIntegrityCollector) Collect(ctx context.Context) (HostInventory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.last.CollectedAt.IsZero() && time.Since(c.last.CollectedAt) < c.ttl {
		return c.last, c.err
	}
	c.last, c.err = c.inner.Collect(ctx)
	return c.last, c.err
}

func (c *IntegrityDefaultCollector) Collect(ctx context.Context) (HostInventory, error) {
	now := c.now().UTC()
	inv := HostInventory{CollectedAt: now, Platform: c.goos, OS: c.goos, Arch: c.goarch, ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	if c.goos != "linux" {
		setIntegrityProbe(&inv, "host", IntegrityProbeUnsupported, nil)
		inv.Unsupported = append(inv.Unsupported, "detailed host-integrity probes are currently implemented on Linux")
		inv.Fingerprint = Fingerprint(inv)
		return inv, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	inv.Kernel.Release = commandText(ctx, c.commands, "uname", "-r")
	inv.Kernel.Version = commandText(ctx, c.commands, "uname", "-v")
	if inv.Kernel.Release != "" {
		_, err := c.files.Stat(filepath.Join("/lib/modules", inv.Kernel.Release))
		inv.Kernel.ModuleTreePresent = err == nil
		setIntegrityProbe(&inv, "kernelModuleTree", probeForError(err), err)
	}
	if data, err := c.files.ReadFile("/proc/sys/kernel/random/boot_id"); err == nil {
		inv.BootID = strings.TrimSpace(string(data))
		setIntegrityProbe(&inv, "bootId", IntegrityProbeOK, nil)
	} else {
		setIntegrityProbe(&inv, "bootId", IntegrityProbeDegraded, err)
	}
	if entries, err := c.files.ReadDir("/lib/modules"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				inv.Kernel.InstalledModuleTrees = append(inv.Kernel.InstalledModuleTrees, entry.Name())
			}
		}
		sort.Strings(inv.Kernel.InstalledModuleTrees)
		setIntegrityProbe(&inv, "installedModuleTrees", IntegrityProbeOK, nil)
	} else {
		setIntegrityProbe(&inv, "installedModuleTrees", IntegrityProbeDegraded, err)
	}
	if data, err := c.files.ReadFile("/proc/modules"); err == nil {
		inv.Kernel.LoadedModules = parseIntegrityModules(string(data))
		setIntegrityProbe(&inv, "loadedModules", IntegrityProbeOK, nil)
	} else {
		setIntegrityProbe(&inv, "loadedModules", IntegrityProbeDegraded, err)
	}
	inv.Devices = collectIntegrityDevices(ctx, c, &inv)
	inv.Runtimes = collectIntegrityRuntimes(ctx, c)
	inv.Packages = collectIntegrityPackages(ctx, c, inv.Kernel.Release)
	enrichIntegrityDrivers(ctx, c, &inv)
	inv.SecureBoot = collectIntegritySecureBoot(ctx, c)
	inv.CrashEvidence = collectIntegrityCrashEvidence(c)
	inv.Signals = collectIntegritySignals(ctx)
	inv.ResetReasons = integrityResetReasons(inv.Signals)
	setIntegrityProbe(&inv, "journal", integrityProbeForSignals(inv.Signals), nil)
	setIntegrityProbe(&inv, "host", IntegrityProbeOK, nil)
	if len(inv.ProbeErrors) == 0 {
		inv.ProbeErrors = nil
	}
	inv.Fingerprint = Fingerprint(inv)
	return inv, nil
}

func Fingerprint(inv HostInventory) string {
	stable := struct {
		Platform string
		OS       string
		Arch     string
		BootID   string
		Kernel   KernelInfo
		Devices  []DeviceInfo
		Runtimes []RuntimeToolInfo
		Packages PackageState
	}{inv.Platform, inv.OS, inv.Arch, inv.BootID, inv.Kernel, append([]DeviceInfo(nil), inv.Devices...), append([]RuntimeToolInfo(nil), inv.Runtimes...), inv.Packages}
	sort.Slice(stable.Kernel.LoadedModules, func(i, j int) bool { return stable.Kernel.LoadedModules[i] < stable.Kernel.LoadedModules[j] })
	sort.Slice(stable.Devices, func(i, j int) bool { return stable.Devices[i].Address < stable.Devices[j].Address })
	sort.Slice(stable.Runtimes, func(i, j int) bool { return stable.Runtimes[i].Name < stable.Runtimes[j].Name })
	b, _ := json.Marshal(stable)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

func setIntegrityProbe(inv *HostInventory, name string, state IntegrityProbeState, err error) {
	inv.ProbeStatus[name] = state
	if err != nil {
		inv.ProbeErrors[name] = err.Error()
	}
}

func probeForError(err error) IntegrityProbeState {
	if err == nil {
		return IntegrityProbeOK
	}
	return IntegrityProbeDegraded
}

func commandText(ctx context.Context, commands IntegrityCommandRunner, name string, args ...string) string {
	out, err := commands.CombinedOutput(ctx, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseIntegrityModules(content string) []string {
	var modules []string
	for _, line := range strings.Split(content, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			modules = append(modules, fields[0])
		}
	}
	sort.Strings(modules)
	return modules
}

var integrityPCILineRE = regexp.MustCompile(`^([0-9a-fA-F:.]+)\s+([^:]+):\s+(.+?)(?:\s+\[([0-9a-fA-F]{4}):([0-9a-fA-F]{4})\])?$`)

func collectIntegrityDevices(ctx context.Context, c *IntegrityDefaultCollector, inv *HostInventory) []DeviceInfo {
	if _, err := c.commands.LookPath("lspci"); err != nil {
		setIntegrityProbe(inv, "devices", IntegrityProbeUnsupported, err)
		return nil
	}
	out, err := c.commands.CombinedOutput(ctx, "lspci", "-nnk")
	if err != nil {
		setIntegrityProbe(inv, "devices", IntegrityProbeDegraded, err)
		return nil
	}
	devices := parseIntegrityPCI(string(out))
	setIntegrityProbe(inv, "devices", IntegrityProbeOK, nil)
	return devices
}

func parseIntegrityPCI(output string) []DeviceInfo {
	var devices []DeviceInfo
	var current *DeviceInfo
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
			match := integrityPCILineRE.FindStringSubmatch(line)
			device := DeviceInfo{BusType: "pci"}
			if len(match) > 0 {
				device.Address, device.Class, device.DeviceName, device.VendorID, device.DeviceID = match[1], strings.TrimSpace(match[2]), strings.TrimSpace(match[3]), match[4], match[5]
			} else if fields := strings.Fields(line); len(fields) > 0 {
				device.Address = fields[0]
				device.DeviceName = strings.TrimSpace(strings.TrimPrefix(line, device.Address))
			}
			devices = append(devices, device)
			current = &devices[len(devices)-1]
			continue
		}
		if current == nil {
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Kernel driver in use:") {
			current.BoundDriver = strings.TrimSpace(strings.TrimPrefix(line, "Kernel driver in use:"))
		}
		if strings.HasPrefix(line, "Kernel modules:") {
			for _, module := range strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Kernel modules:")), ",") {
				if module = strings.TrimSpace(module); module != "" {
					current.AvailableModules = append(current.AvailableModules, module)
				}
			}
		}
	}
	return devices
}

func collectIntegrityRuntimes(ctx context.Context, c *IntegrityDefaultCollector) []RuntimeToolInfo {
	var tools []RuntimeToolInfo
	for _, spec := range []struct {
		name string
		args []string
	}{{"podman", []string{"version", "--format", "{{.Server.Version}}"}}, {"docker", []string{"version", "--format", "{{.Server.Version}}"}}, {"nvidia-smi", []string{"--query-gpu=driver_version", "--format=csv,noheader"}}} {
		path, err := c.commands.LookPath(spec.name)
		tool := RuntimeToolInfo{Name: spec.name, Path: path}
		if err != nil {
			tool.Error = "not found"
		} else if out, runErr := c.commands.CombinedOutput(ctx, spec.name, spec.args...); runErr != nil {
			tool.Error = truncateIntegrity(string(out)+" "+runErr.Error(), 300)
		} else {
			tool.Callable = true
			tool.Version = truncateIntegrity(string(out), 300)
		}
		tools = append(tools, tool)
	}
	return tools
}

func collectIntegrityPackages(ctx context.Context, c *IntegrityDefaultCollector, runningKernel string) PackageState {
	if _, err := c.commands.LookPath("dpkg"); err == nil {
		state := PackageState{Manager: "dpkg"}
		out, err := c.commands.CombinedOutput(ctx, "dpkg-query", "-W", "-f=${db:Status-Abbrev} ${Package} ${Version}\n", "linux-image*", "linux-modules*", "nvidia-*", "amdgpu*", "mesa-*")
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			status, name := fields[0], fields[1]
			version := ""
			if len(fields) > 2 {
				version = fields[2]
			}
			if strings.HasPrefix(status, "ii") {
				state.Installed = append(state.Installed, name)
				state.InstalledPackages = append(state.InstalledPackages, PackageInfo{Name: name, Version: version, Status: status})
			} else if strings.TrimSpace(status) != "" && !strings.HasPrefix(status, "rc") {
				state.BrokenOrHeld = append(state.BrokenOrHeld, strings.TrimSpace(line))
			}
		}
		if err != nil && len(state.Installed) == 0 {
			state.BrokenOrHeld = append(state.BrokenOrHeld, "dpkg-query failed: "+err.Error())
		}
		if holds, holdErr := c.commands.CombinedOutput(ctx, "apt-mark", "showhold"); holdErr == nil {
			for _, line := range strings.Split(string(holds), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					state.BrokenOrHeld = append(state.BrokenOrHeld, "held: "+line)
				}
			}
		}
		state.Kernel = buildIntegrityKernelPackageState(state.Installed, state.BrokenOrHeld, runningKernel)
		return state
	}
	if _, err := c.commands.LookPath("rpm"); err == nil {
		state := PackageState{Manager: "rpm"}
		if out, runErr := c.commands.CombinedOutput(ctx, "rpm", "-qa", "kernel*", "kmod*", "akmod*", "nvidia*", "mesa*"); runErr == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					state.Installed = append(state.Installed, line)
				}
			}
		}
		state.Kernel = buildIntegrityKernelPackageState(state.Installed, nil, runningKernel)
		return state
	}
	if _, err := c.commands.LookPath("pacman"); err == nil {
		state := PackageState{Manager: "pacman"}
		if out, runErr := c.commands.CombinedOutput(ctx, "pacman", "-Q"); runErr == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "linux") || strings.Contains(line, "nvidia") || strings.Contains(line, "mesa") {
					state.Installed = append(state.Installed, strings.TrimSpace(line))
				}
			}
		}
		if out, runErr := c.commands.CombinedOutput(ctx, "pacman", "-Qu"); runErr == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					state.PendingUpgrades = append(state.PendingUpgrades, line)
				}
			}
		}
		state.Kernel = buildIntegrityKernelPackageState(state.Installed, state.BrokenOrHeld, runningKernel)
		return state
	}
	return PackageState{}
}

var integrityNVIDIARE = regexp.MustCompile(`^nvidia-driver-([0-9]+)(-open)?$`)

func enrichIntegrityDrivers(ctx context.Context, c *IntegrityDefaultCollector, inv *HostInventory) {
	if inv == nil || inv.Packages.Manager != "dpkg" || inv.Kernel.Release == "" || !integrityHasNVIDIAEvidence(*inv) {
		return
	}
	driver := DriverPackageState{Vendor: "nvidia", Applicability: "needs_corroboration", LoadedModules: integrityFilterModules(inv.Kernel.LoadedModules, []string{"nvidia", "nouveau"})}
	for _, pkg := range inv.Packages.InstalledPackages {
		match := integrityNVIDIARE.FindStringSubmatch(pkg.Name)
		if len(match) == 0 {
			continue
		}
		driver.Series = match[1]
		if match[2] == "-open" {
			driver.Flavor = "open"
		}
		driver.InstalledPackages = append(driver.InstalledPackages, pkg)
	}
	if driver.Series == "" {
		driver.Applicability = "unsupported"
		inv.Packages.Drivers = append(inv.Packages.Drivers, driver)
		return
	}
	parts := []string{"linux-modules-nvidia", driver.Series}
	if driver.Flavor != "" {
		parts = append(parts, driver.Flavor)
	}
	driver.ExpectedModulePackage = strings.Join(append(parts, inv.Kernel.Release), "-")
	for _, pkg := range inv.Packages.InstalledPackages {
		if pkg.Name == driver.ExpectedModulePackage {
			driver.ExpectedPackageInstalled = true
			driver.InstalledPackages = append(driver.InstalledPackages, pkg)
			break
		}
	}
	if driver.ExpectedPackageInstalled {
		driver.Applicability = "applicable"
	} else {
		driver.MissingModulePackage = driver.ExpectedModulePackage
		driver.Applicability = "applicable"
		candidate := PackageCandidate{Name: driver.ExpectedModulePackage, Source: "apt-cache policy"}
		if _, err := c.commands.LookPath("apt-cache"); err != nil {
			candidate.Error = err.Error()
		} else if out, err := c.commands.CombinedOutput(ctx, "apt-cache", "policy", driver.ExpectedModulePackage); err != nil {
			candidate.Error = truncateIntegrity(string(out)+" "+err.Error(), 300)
		} else {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "Candidate:") {
					version := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Candidate:"))
					if version != "" && version != "(none)" {
						candidate.Version, candidate.Available = version, true
					}
				}
			}
		}
		driver.Candidate = &candidate
	}
	inv.Packages.Drivers = append(inv.Packages.Drivers, driver)
}

func integrityHasNVIDIAEvidence(inv HostInventory) bool {
	for _, device := range inv.Devices {
		if strings.Contains(strings.ToLower(device.VendorName+" "+device.DeviceName), "nvidia") || strings.EqualFold(device.VendorID, "10de") {
			return true
		}
	}
	for _, tool := range inv.Runtimes {
		if tool.Name == "nvidia-smi" && tool.Path != "" {
			return true
		}
	}
	for _, module := range inv.Kernel.LoadedModules {
		if strings.HasPrefix(module, "nvidia") || module == "nouveau" {
			return true
		}
	}
	return false
}

func integrityFilterModules(modules, prefixes []string) []string {
	var filtered []string
	for _, module := range modules {
		for _, prefix := range prefixes {
			if strings.HasPrefix(module, prefix) {
				filtered = append(filtered, module)
				break
			}
		}
	}
	sort.Strings(filtered)
	return filtered
}

var integrityKernelRE = regexp.MustCompile(`\d+\.\d+\.\d+[-._A-Za-z0-9]*`)

func buildIntegrityKernelPackageState(installed, held []string, running string) KernelPackageState {
	state := KernelPackageState{RunningKernel: running, HeldOrBlocked: held}
	if running == "" {
		return state
	}
	for _, pkg := range installed {
		if strings.Contains(pkg, running) {
			state.InstalledMatching = append(state.InstalledMatching, pkg)
			continue
		}
		if integrityKernelRE.MatchString(pkg) {
			state.InstalledOtherKernel = append(state.InstalledOtherKernel, pkg)
		}
	}
	for _, expected := range []string{"linux-image-" + running, "linux-modules-" + running} {
		if !containsIntegrity(installed, expected) {
			state.MissingMatching = append(state.MissingMatching, expected)
		}
	}
	return state
}

func containsIntegrity(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func collectIntegritySecureBoot(ctx context.Context, c *IntegrityDefaultCollector) SecureBootState {
	state := SecureBootState{Source: "mokutil --sb-state"}
	if _, err := c.commands.LookPath("mokutil"); err != nil {
		state.Error = err.Error()
		return state
	}
	state.Supported = true
	out, err := c.commands.CombinedOutput(ctx, "mokutil", "--sb-state")
	if err != nil {
		state.Error = truncateIntegrity(string(out)+" "+err.Error(), 300)
		return state
	}
	state.Enabled = strings.Contains(strings.ToLower(string(out)), "secureboot enabled")
	return state
}

func collectIntegrityCrashEvidence(c *IntegrityDefaultCollector) CrashEvidenceProbeState {
	var state CrashEvidenceProbeState
	if _, err := c.files.ReadDir("/sys/fs/pstore"); err == nil {
		state.PstoreSupported, state.PstoreReadable = true, true
	} else {
		state.PstoreError = err.Error()
		state.PstoreSupported = !errors.Is(err, os.ErrNotExist)
	}
	if _, err := c.files.ReadDir("/var/lib/vrooli/host-observability/pstore"); err == nil {
		state.PstoreExportReadable = true
	} else {
		state.PstoreExportError = err.Error()
	}
	state.PstoreCoverageGap = state.PstoreSupported && !state.PstoreReadable && !state.PstoreExportReadable
	if path, err := c.commands.LookPath("ras-mc-ctl"); err == nil {
		state.RasdaemonPresent, state.RasdaemonPath = true, path
	}
	return state
}

func collectIntegritySignals(ctx context.Context) []HostSignal {
	result, err := platformgo.ReadHostLogs(platformgo.HostLogOptions{Arguments: []string{"--no-pager", "-k", "--since", "24 hours ago", "-p", "0..4", "-n", "50", "-r"}})
	if err != nil {
		return []HostSignal{{Source: "journal", Severity: "warning", Category: "probe_error", Message: truncateIntegrity(err.Error(), 300)}}
	}
	var signals []HostSignal
	for _, line := range strings.Split(string(result.Raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		category := integritySignalCategory(line)
		if category == "" {
			continue
		}
		signals = append(signals, HostSignal{Source: "journal", Severity: "warning", Category: category, Message: truncateIntegrity(line, 500)})
	}
	return signals
}

func integritySignalCategory(message string) string {
	lower := strings.ToLower(message)
	for _, pair := range []struct{ needle, category string }{
		{"machine check", "machine_check"},
		{"mce", "machine_check"},
		{"reset", "device_reset"},
		{"timeout", "device_timeout"},
		{"i/o error", "io_error"},
		{"corrupt", "filesystem_corruption"},
		{"xid", "device_runtime"},
		{"aer", "bus_error"},
		{"pcie", "bus_error"},
		{"gpu", "display_or_accelerator"},
		{"drm", "display_or_accelerator"},
		{"amdgpu", "display_or_accelerator"},
		{"nvidia", "display_or_accelerator"},
	} {
		if strings.Contains(lower, pair.needle) {
			return pair.category
		}
	}
	return ""
}

func integrityResetReasons(signals []HostSignal) []ResetReason {
	var reasons []ResetReason
	for _, signal := range signals {
		if signal.Category != "device_reset" {
			continue
		}
		reasons = append(reasons, ResetReason{Source: signal.Source, RawMessage: signal.Message, Category: signal.Category, Criticality: "warning"})
	}
	return reasons
}

func integrityProbeForSignals(signals []HostSignal) IntegrityProbeState {
	if len(signals) == 1 && signals[0].Category == "probe_error" {
		return IntegrityProbeDegraded
	}
	return IntegrityProbeOK
}

func truncateIntegrity(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "...[truncated]"
}
