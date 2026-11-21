package hostinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	integrityCollectorParameterA = 2
	integrityCollectorParameterB = 300
	integrityCollectorParameterC = 500
)

// IntegrityCommandRunner and IntegrityFileSystem are the control-plane seams
// for host-integrity probes. They keep scenarios from owning host detection
// while still allowing deterministic fixture tests.
type IntegrityCommandRunner = shell.Runner

type IntegrityFileSystem interface {
	ReadFile(string) ([]byte, error)
	ReadDir(string) ([]os.DirEntry, error)
	Readlink(string) (string, error)
	Stat(string) (os.FileInfo, error)
}

type IntegrityCollector interface {
	Collect(context.Context) (HostInventory, error)
}

type IntegrityCollectorOptions struct {
	Commands      IntegrityCommandRunner
	Files         IntegrityFileSystem
	NativeDevices func(context.Context, IntegrityCommandRunner) NativeDeviceObservation
	SysRoot       string
	GOOS          string
	GOARCH        string
	Now           func() time.Time
}

type IntegrityDefaultCollector struct {
	commands IntegrityCommandRunner
	files    IntegrityFileSystem
	native   func(context.Context, IntegrityCommandRunner) NativeDeviceObservation
	sysRoot  string
	goos     string
	goarch   string
	now      func() time.Time
}

type nativeIntegrityFiles struct{}

func (nativeIntegrityFiles) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (nativeIntegrityFiles) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }
func (nativeIntegrityFiles) Readlink(path string) (string, error)       { return os.Readlink(path) }
func (nativeIntegrityFiles) Stat(path string) (os.FileInfo, error)      { return os.Stat(path) }

func NewIntegrityCollector(options IntegrityCollectorOptions) *IntegrityDefaultCollector {
	if options.Commands == nil {
		options.Commands = shell.OSRunner{}
	}
	if options.Files == nil {
		options.Files = nativeIntegrityFiles{}
	}
	if options.NativeDevices == nil {
		options.NativeDevices = CollectNativeDevices
	}
	if options.GOOS == "" {
		options.GOOS = runtime.GOOS
	}
	if options.GOARCH == "" {
		options.GOARCH = runtime.GOARCH
	}
	if options.SysRoot == "" {
		options.SysRoot = "/sys"
	}
	if options.Now == nil {
		options.Now = clock.Real{}.Now
	}
	return &IntegrityDefaultCollector{commands: options.Commands, files: options.Files, native: options.NativeDevices, sysRoot: options.SysRoot, goos: options.GOOS, goarch: options.GOARCH, now: options.Now}
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
	if hostreqspec.PlatformFromGOOS(c.goos) != hostreqspec.PlatformLinux {
		populateNativeDeviceInventory(&inv, c.native(ctx, c.commands))
		inv.Fingerprint = Fingerprint(inv)
		return inv, nil
	}
	ctx, cancel := context.WithTimeout(ctx, tuning.IntegrityCollectionTimeout())
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
		slices.Sort(inv.Kernel.InstalledModuleTrees)
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
	observeLinuxBluetoothControllers(ctx, inv.Devices, c.commands.Run)
	inv.Runtimes = collectIntegrityRuntimes(ctx, c)
	inv.Packages = collectIntegrityPackages(ctx, c, inv.Kernel.Release)
	enrichIntegrityDrivers(ctx, c, &inv)
	inv.SecureBoot = collectIntegritySecureBoot(ctx, c)
	inv.CrashEvidence = collectIntegrityCrashEvidence(c)
	inv.Signals = collectIntegritySignals(ctx)
	applyMissingFirmwareEvidence(inv.Devices, inv.Signals)
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
		Platform    string
		OS          string
		Arch        string
		BootID      string
		Kernel      KernelInfo
		Devices     []DeviceInfo
		Runtimes    []RuntimeToolInfo
		Packages    PackageState
		ProbeStatus map[string]IntegrityProbeState
		ProbeErrors map[string]string
	}{inv.Platform, inv.OS, inv.Arch, inv.BootID, inv.Kernel, append([]DeviceInfo(nil), inv.Devices...), append([]RuntimeToolInfo(nil), inv.Runtimes...), inv.Packages, inv.ProbeStatus, inv.ProbeErrors}
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
	out, err := commands.Run(ctx, name, args...)
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
	slices.Sort(modules)
	return modules
}

func collectIntegrityDevices(_ context.Context, c *IntegrityDefaultCollector, inv *HostInventory) []DeviceInfo {
	var devices []DeviceInfo
	busResults := make([]IntegrityProbeState, 0, 2)
	bindingResults := make([]IntegrityProbeState, 0, 2)
	var interfaceErrors error
	for _, bus := range []string{"pci", "usb"} {
		result := collectIntegrityBusDevices(c, bus)
		devices = append(devices, result.devices...)
		interfaceErrors = errors.Join(interfaceErrors, result.interfaceError)
		setIntegrityProbe(inv, bus+"Devices", result.enumerationState, result.enumerationError)
		setIntegrityProbe(inv, bus+"DriverBindings", result.bindingState, result.bindingError)
		if bus == "usb" {
			state, probeErr := probeForError(result.bluetoothError), result.bluetoothError
			if result.enumerationState != IntegrityProbeOK {
				state, probeErr = result.enumerationState, result.enumerationError
			}
			setIntegrityProbe(inv, "bluetoothInterfaces", state, probeErr)
		}
		busResults = append(busResults, result.enumerationState)
		bindingResults = append(bindingResults, result.bindingState)
	}
	switch {
	case busResults[0] == IntegrityProbeUnsupported:
		setIntegrityProbe(inv, "networkInterfaces", IntegrityProbeUnsupported, errors.New("PCI enumeration is unavailable; network-interface presence cannot be evaluated"))
	case busResults[0] != IntegrityProbeOK:
		setIntegrityProbe(inv, "networkInterfaces", IntegrityProbeDegraded, errors.New("PCI enumeration is incomplete; network-interface presence may be incomplete"))
	default:
		setIntegrityProbe(inv, "networkInterfaces", probeForError(interfaceErrors), interfaceErrors)
	}
	if busResults[0] == IntegrityProbeOK && busResults[1] == IntegrityProbeOK {
		setIntegrityProbe(inv, "devices", IntegrityProbeOK, nil)
	} else if busResults[0] == IntegrityProbeUnsupported && busResults[1] == IntegrityProbeUnsupported {
		setIntegrityProbe(inv, "devices", IntegrityProbeUnsupported, errors.New("neither PCI nor USB device enumeration is available through sysfs"))
	} else {
		setIntegrityProbe(inv, "devices", IntegrityProbeDegraded, errors.New("host device inventory is incomplete; inspect the pciDevices and usbDevices probe results"))
	}
	if bindingResults[0] == IntegrityProbeOK && bindingResults[1] == IntegrityProbeOK {
		setIntegrityProbe(inv, "deviceBindings", IntegrityProbeOK, nil)
	} else if bindingResults[0] == IntegrityProbeUnsupported && bindingResults[1] == IntegrityProbeUnsupported {
		setIntegrityProbe(inv, "deviceBindings", IntegrityProbeUnsupported, errors.New("neither PCI nor USB driver binding is observable through sysfs"))
	} else {
		setIntegrityProbe(inv, "deviceBindings", IntegrityProbeDegraded, errors.New("some device driver bindings could not be read; inspect the per-bus DriverBindings probe results"))
	}
	return devices
}

func populateNativeDeviceInventory(inv *HostInventory, observation NativeDeviceObservation) {
	inv.Devices = observation.Devices
	if observation.ProbeStatus == nil {
		observation.ProbeStatus = make(map[string]IntegrityProbeState)
	}
	if observation.ProbeErrors == nil {
		observation.ProbeErrors = make(map[string]string)
	}
	for _, scope := range []string{"devices", "deviceBindings", "pciDevices", "usbDevices", "networkInterfaces"} {
		if _, observed := observation.ProbeStatus[scope]; !observed {
			observation.setProbe(scope, IntegrityProbeFailed, fmt.Errorf("native adapter omitted %s probe coverage", scope))
		}
	}
	for scope, state := range observation.ProbeStatus {
		setIntegrityProbe(inv, scope, state, nil)
	}
	for scope, reason := range observation.ProbeErrors {
		if reason != "" {
			setIntegrityProbe(inv, scope, observation.ProbeStatus[scope], errors.New(reason))
		}
	}
	setIntegrityProbe(inv, "host", IntegrityProbeUnsupported,
		errors.New("native device enumeration is available, but detailed kernel, module, package and host-integrity probes are unsupported on this platform"))
	inv.Unsupported = append(inv.Unsupported,
		"detailed kernel, module, package and host-integrity probes are not implemented for this platform")
}

type integrityBusResult struct {
	devices          []DeviceInfo
	enumerationState IntegrityProbeState
	enumerationError error
	bindingState     IntegrityProbeState
	bindingError     error
	interfaceError   error
	bluetoothError   error
}

func collectIntegrityBusDevices(c *IntegrityDefaultCollector, bus string) integrityBusResult {
	root := filepath.Join(c.sysRoot, "bus", bus, "devices")
	entries, err := c.files.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return integrityBusResult{enumerationState: IntegrityProbeUnsupported, enumerationError: err, bindingState: IntegrityProbeUnsupported, bindingError: err}
		}
		return integrityBusResult{enumerationState: IntegrityProbeDegraded, enumerationError: err, bindingState: IntegrityProbeUnsupported}
	}
	devices := make([]DeviceInfo, 0, len(entries))
	result := integrityBusResult{devices: devices, enumerationState: IntegrityProbeOK, bindingState: IntegrityProbeOK}
	deviceIndexes := make(map[string]int, len(entries))
	var enumerationErr error
	var bindingErr error
	for _, entry := range entries {
		name := entry.Name()
		if bus == "usb" && strings.Contains(name, ":") {
			continue // sysfs names USB interfaces <device>:<configuration>.<interface>.
		}
		device := DeviceInfo{BusType: bus, Address: name, SysPath: filepath.Join(root, name)}
		if bus == "pci" {
			device.ID = CanonicalPCIIdentity(name)
			if device.ID == "" {
				enumerationErr = errors.Join(enumerationErr, fmt.Errorf("ignoring malformed PCI sysfs address %q", name))
				continue
			}
			if err := readSysfsDeviceFields(c, &device); err != nil {
				enumerationErr = errors.Join(enumerationErr, fmt.Errorf("PCI device %s identity: %w", name, err))
			}
		} else {
			device.ID = "usb:" + name
			if err := readSysfsUSBFields(c, &device); err != nil {
				enumerationErr = errors.Join(enumerationErr, fmt.Errorf("USB device %s identity: %w", name, err))
			}
		}
		if bus == "pci" {
			if bindingReadErr := readDeviceBinding(c, &device); bindingReadErr != nil {
				bindingErr = errors.Join(bindingErr, fmt.Errorf("PCI device %s driver link: %w", name, bindingReadErr))
			}
		}
		deviceIndexes[name] = len(devices)
		devices = append(devices, device)
	}
	if bus == "usb" {
		for _, entry := range entries {
			name := entry.Name()
			if !strings.Contains(name, ":") {
				continue
			}
			parent, _, _ := strings.Cut(name, ":")
			index, ok := deviceIndexes[parent]
			if !ok {
				continue
			}
			function := DeviceFunction{ID: name, BindingState: DeviceBindingUnknown}
			functionRoot := filepath.Join(root, name)
			if data, readErr := c.files.ReadFile(filepath.Join(functionRoot, "bInterfaceClass")); readErr == nil {
				function.Class = strings.TrimSpace(string(data))
			}
			if data, readErr := c.files.ReadFile(filepath.Join(functionRoot, "modalias")); readErr == nil {
				function.Modalias = strings.TrimSpace(string(data))
			}
			if link, readErr := c.files.Readlink(filepath.Join(functionRoot, "driver")); readErr == nil {
				function.BoundDriver = filepath.Base(filepath.Clean(link))
				function.BindingState = DeviceBindingBound
			} else if errors.Is(readErr, os.ErrNotExist) {
				function.BindingState = DeviceBindingUnbound
			} else {
				bindingErr = errors.Join(bindingErr, fmt.Errorf("USB function %s driver link: %w", name, readErr))
			}
			devices[index].Functions = append(devices[index].Functions, function)
		}
		for index := range devices {
			deriveUSBDeviceBinding(&devices[index])
		}
	}
	for index := range devices {
		if err := observeLinuxDeviceInterfaces(c.files, &devices[index]); err != nil {
			if bus == "usb" {
				result.bluetoothError = errors.Join(result.bluetoothError, err)
			} else {
				result.interfaceError = errors.Join(result.interfaceError, err)
			}
		}
	}
	result.devices = devices
	if enumerationErr != nil {
		result.enumerationState = IntegrityProbeDegraded
		result.enumerationError = enumerationErr
	}
	if bindingErr != nil {
		result.bindingState = IntegrityProbeDegraded
		result.bindingError = bindingErr
	}
	return result
}

func readDeviceBinding(c *IntegrityDefaultCollector, device *DeviceInfo) error {
	link, err := c.files.Readlink(filepath.Join(device.SysPath, "driver"))
	if err == nil {
		device.BoundDriver = filepath.Base(filepath.Clean(link))
		device.BindingState = DeviceBindingBound
	} else if errors.Is(err, os.ErrNotExist) {
		device.BindingState = DeviceBindingUnbound
	} else {
		device.BindingState = DeviceBindingUnknown
		return err
	}
	return nil
}

func deriveUSBDeviceBinding(device *DeviceInfo) {
	if len(device.Functions) == 0 {
		device.BindingState = DeviceBindingUnknown
		return
	}
	var drivers []string
	hasUnbound, hasUnknown := false, false
	for _, function := range device.Functions {
		switch function.BindingState {
		case DeviceBindingBound:
			drivers = append(drivers, function.BoundDriver)
		case DeviceBindingUnbound:
			hasUnbound = true
		case DeviceBindingUnknown:
			hasUnknown = true
		}
	}
	if len(drivers) > 0 {
		slices.Sort(drivers)
		device.BoundDriver = strings.Join(slices.Compact(drivers), ",")
		if hasUnbound || hasUnknown {
			device.BindingState = DeviceBindingPartial
		} else {
			device.BindingState = DeviceBindingBound
		}
	} else if hasUnknown {
		device.BindingState = DeviceBindingUnknown
	} else if hasUnbound {
		device.BindingState = DeviceBindingUnbound
	}
}

func readSysfsDeviceFields(c *IntegrityDefaultCollector, device *DeviceInfo) error {
	read := func(name string) (string, error) {
		data, err := c.files.ReadFile(filepath.Join(device.SysPath, name))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", name, err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("read %s: attribute is empty", name)
		}
		return value, nil
	}
	var readErr error
	vendor, err := read("vendor")
	readErr = errors.Join(readErr, err)
	device.VendorID = strings.TrimPrefix(strings.ToLower(vendor), "0x")
	product, err := read("device")
	readErr = errors.Join(readErr, err)
	device.DeviceID = strings.TrimPrefix(strings.ToLower(product), "0x")
	class, err := read("class")
	readErr = errors.Join(readErr, err)
	classCode := strings.TrimPrefix(strings.ToLower(class), "0x")
	device.Class = pciClassName(classCode)
	device.DeviceName = device.DeviceID
	device.Modalias, _ = read("modalias") // Optional for stable sysfs identity and class.
	device.VendorName = device.VendorID
	return readErr
}

func observeLinuxDeviceInterfaces(files IntegrityFileSystem, device *DeviceInfo) error {
	device.Readiness = DeviceReadiness{
		FirmwareState:  DeviceReadinessUnknown,
		FirmwareReason: "firmware load state is not directly exposed by this device inventory probe",
		InterfaceState: DeviceReadinessNotApplicable,
		FunctionState:  DeviceReadinessUnverified,
		FunctionReason: "device enumeration and driver binding do not verify end-to-end function",
	}
	if strings.Contains(strings.ToLower(device.Class), "network") {
		return readSysfsNetworkInterfaces(files, device)
	}
	return readSysfsBluetoothInterfaces(files, device)
}

func readSysfsNetworkInterfaces(files IntegrityFileSystem, device *DeviceInfo) error {
	interfaces, err := files.ReadDir(filepath.Join(device.SysPath, "net"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			device.Readiness.InterfaceState = DeviceReadinessAbsent
			device.Readiness.InterfaceReason = "no network interface is exposed for this enumerated network controller"
			return nil
		}
		device.Readiness.InterfaceState = DeviceReadinessUnknown
		device.Readiness.InterfaceReason = "network interface enumeration failed: " + err.Error()
		return err
	}
	for _, entry := range interfaces {
		name := entry.Name()
		device.Readiness.InterfaceNames = append(device.Readiness.InterfaceNames, name)
		status, statusErr := files.ReadFile(filepath.Join(device.SysPath, "net", name, "operstate"))
		if device.Readiness.InterfaceStatus == nil {
			device.Readiness.InterfaceStatus = make(map[string]string, len(interfaces))
		}
		if statusErr != nil {
			device.Readiness.InterfaceStatus[name] = DeviceReadinessUnknown
		} else {
			device.Readiness.InterfaceStatus[name] = strings.ToLower(strings.TrimSpace(string(status)))
		}
	}
	slices.Sort(device.Readiness.InterfaceNames)
	if len(device.Readiness.InterfaceNames) == 0 {
		device.Readiness.InterfaceState = DeviceReadinessAbsent
		device.Readiness.InterfaceReason = "network interface directory is readable and empty"
		return nil
	}
	device.Readiness.InterfaceState = DeviceReadinessInitialized
	allDown, allLowerLayerDown := true, true
	for _, status := range device.Readiness.InterfaceStatus {
		allDown = allDown && status == "down"
		allLowerLayerDown = allLowerLayerDown && status == "lowerlayerdown"
	}
	switch {
	case allDown:
		device.Readiness.InterfaceState = DeviceReadinessDisabled
		device.Readiness.InterfaceReason = "all exposed interfaces report operstate=down; the specific cause and radio state are not established"
	case allLowerLayerDown:
		device.Readiness.InterfaceState = DeviceReadinessDisconnected
		device.Readiness.InterfaceReason = "all exposed interfaces report operstate=lowerlayerdown; the specific cause is not established"
	default:
		device.Readiness.InterfaceReason = "an OS network interface is initialized; link, radio and peer function are not verified"
	}
	return nil
}

func readSysfsUSBFields(c *IntegrityDefaultCollector, device *DeviceInfo) error {
	read := func(name string) (string, error) {
		data, err := c.files.ReadFile(filepath.Join(device.SysPath, name))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", name, err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("read %s: attribute is empty", name)
		}
		return value, nil
	}
	var readErr error
	vendor, err := read("idVendor")
	readErr = errors.Join(readErr, err)
	device.VendorID = strings.TrimPrefix(strings.ToLower(vendor), "0x")
	product, err := read("idProduct")
	readErr = errors.Join(readErr, err)
	device.DeviceID = strings.TrimPrefix(strings.ToLower(product), "0x")
	device.VendorName, _ = read("manufacturer") // Optional human-readable metadata.
	device.DeviceName, _ = read("product")      // Optional human-readable metadata.
	device.Modalias, _ = read("modalias")       // Optional for stable USB identity.
	if device.DeviceName == "" {
		device.DeviceName = device.DeviceID
	}
	if device.VendorName == "" {
		device.VendorName = device.VendorID
	}
	device.Class = "usb device"
	return readErr
}

func pciClassName(code string) string {
	if len(code) < 2 {
		return "pci device"
	}
	switch code[:2] {
	case "01":
		return "storage controller"
	case "02":
		return "network controller"
	case "03":
		return "display controller"
	case "04":
		return "multimedia controller"
	case "0c":
		return "serial bus controller"
	default:
		return "pci device"
	}
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
		} else if out, runErr := c.commands.Run(ctx, spec.name, spec.args...); runErr != nil {
			tool.Error = truncateIntegrity(string(out)+" "+runErr.Error(), integrityCollectorParameterB)
		} else {
			tool.Callable = true
			tool.Version = truncateIntegrity(string(out), integrityCollectorParameterB)
		}
		tools = append(tools, tool)
	}
	return tools
}

//nolint:gocyclo // integrity collection merges package-manager, kernel, and verification state outcomes.
func collectIntegrityPackages(ctx context.Context, c *IntegrityDefaultCollector, runningKernel string) PackageState {
	if _, err := c.commands.LookPath("dpkg"); err == nil {
		state := PackageState{Manager: "dpkg"}
		out, err := c.commands.Run(ctx, "dpkg-query", "-W", "-f=${db:Status-Abbrev} ${Package} ${Version}\n", "linux-image*", "linux-modules*", "nvidia-*", "amdgpu*", "mesa-*")
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < integrityCollectorParameterA {
				continue
			}
			status, name := fields[0], fields[1]
			version := ""
			if len(fields) > integrityCollectorParameterA {
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
		if holds, holdErr := c.commands.Run(ctx, "apt-mark", "showhold"); holdErr == nil {
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
		if out, runErr := c.commands.Run(ctx, "rpm", "-qa", "kernel*", "kmod*", "akmod*", "nvidia*", "mesa*"); runErr == nil {
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
		if out, runErr := c.commands.Run(ctx, "pacman", "-Q"); runErr == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, string(hostreqspec.PlatformLinux)) || strings.Contains(line, "nvidia") || strings.Contains(line, "mesa") {
					state.Installed = append(state.Installed, strings.TrimSpace(line))
				}
			}
		}
		if out, runErr := c.commands.Run(ctx, "pacman", "-Qu"); runErr == nil {
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

func enrichIntegrityDrivers(ctx context.Context, c *IntegrityDefaultCollector, inv *HostInventory) {
	if inv == nil || inv.Packages.Manager != "dpkg" || inv.Kernel.Release == "" || !integrityHasNVIDIAEvidence(*inv) {
		return
	}
	driver := DriverPackageState{Vendor: "nvidia", VendorID: "10de", Applicability: "needs_corroboration", LoadedModules: integrityFilterModules(inv.Kernel.LoadedModules, []string{"nvidia", "nouveau"})}
	driverPackage := ""
	for _, pkg := range inv.Packages.InstalledPackages {
		if !hostreqkit.IsNvidiaDriverPackage(pkg.Name) {
			continue
		}
		if driverPackage == "" {
			driverPackage = pkg.Name
		}
		driver.Series, driver.Flavor = hostreqkit.NvidiaDriverPackageIdentity(pkg.Name)
		driver.InstalledPackages = append(driver.InstalledPackages, pkg)
	}
	if driver.Series == "" {
		driver.Applicability = "unsupported"
		inv.Packages.Drivers = append(inv.Packages.Drivers, driver)
		return
	}
	driver.ExpectedModulePackage, _ = hostreqkit.DeriveNvidiaModulePackage(driverPackage, inv.Kernel.Release)
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
		} else if out, err := c.commands.Run(ctx, "apt-cache", "policy", driver.ExpectedModulePackage); err != nil {
			candidate.Error = truncateIntegrity(string(out)+" "+err.Error(), integrityCollectorParameterB)
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
		if tool.Name == ToolNvidiaSMI && tool.Path != "" {
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
	slices.Sort(filtered)
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
		if !slices.Contains(installed, expected) {
			state.MissingMatching = append(state.MissingMatching, expected)
		}
	}
	return state
}

func collectIntegritySecureBoot(ctx context.Context, c *IntegrityDefaultCollector) SecureBootState {
	state := SecureBootState{Source: "mokutil --sb-state"}
	if _, err := c.commands.LookPath("mokutil"); err != nil {
		state.Error = err.Error()
		return state
	}
	state.Supported = true
	out, err := c.commands.Run(ctx, "mokutil", "--sb-state")
	if err != nil {
		state.Error = truncateIntegrity(string(out)+" "+err.Error(), integrityCollectorParameterB)
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
	result, err := platformgo.ReadHostLogs(platformgo.HostLogOptions{Arguments: integrityJournalArgs()})
	return parseIntegritySignals(result.Raw, err)
}

func integrityJournalArgs() []string {
	return []string{"--no-pager", "-k", "--since", "24 hours ago", "-p", "0..4", "-n", "50", "-r"}
}

func parseIntegritySignals(raw []byte, err error) []HostSignal {
	if err != nil {
		return []HostSignal{{Source: "journal", Severity: "warning", Category: "probe_error", Message: truncateIntegrity(err.Error(), integrityCollectorParameterB)}}
	}
	var signals []HostSignal
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		category := integritySignalCategory(line)
		if category == "" {
			continue
		}
		signals = append(signals, HostSignal{Source: "journal", Severity: "warning", Category: category, Message: truncateIntegrity(line, integrityCollectorParameterC)})
	}
	return signals
}

func integritySignalCategory(message string) string {
	lower := strings.ToLower(message)
	if missingFirmwareMessage(lower) {
		return "firmware_missing"
	}
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

func missingFirmwareMessage(lower string) bool {
	if !strings.Contains(lower, "firmware") {
		return false
	}
	return strings.Contains(lower, "failed with error -2") ||
		strings.Contains(lower, "no such file or directory") ||
		strings.Contains(lower, "firmware") && strings.Contains(lower, "not found")
}

func applyMissingFirmwareEvidence(devices []DeviceInfo, signals []HostSignal) {
	for i := range devices {
		address := strings.ToLower(strings.TrimSpace(devices[i].Address))
		if address == "" {
			continue
		}
		for _, signal := range signals {
			if signal.Category != "firmware_missing" || !containsDeviceAddress(signal.Message, address) {
				continue
			}
			devices[i].Readiness.FirmwareState = DeviceReadinessMissing
			devices[i].Readiness.FirmwareReason = "kernel log reports a missing firmware file for this device: " + truncateIntegrity(signal.Message, integrityCollectorParameterC)
			break
		}
	}
}

func containsDeviceAddress(message, address string) bool {
	lowerMessage := strings.ToLower(message)
	for offset := 0; offset < len(lowerMessage); {
		index := strings.Index(lowerMessage[offset:], address)
		if index < 0 {
			return false
		}
		index += offset
		end := index + len(address)
		leftOK := index == 0 || !isAddressCharacter(lowerMessage[index-1])
		rightOK := end == len(lowerMessage) || !isAddressCharacter(lowerMessage[end])
		if leftOK && rightOK {
			return true
		}
		offset = index + 1
	}
	return false
}

func isAddressCharacter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '-' || char == '.'
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
