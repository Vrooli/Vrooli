package hostinventory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// sysfsDeviceEnumerator reads the Linux device tree. It is a plain filesystem
// reader with an injectable root so the enumeration is exercised by checked-in
// fixtures on hosts that have none of the hardware.
type sysfsDeviceEnumerator struct {
	root       string
	pciIDFiles []string
}

// defaultPCIIDFiles are the locations distributions install the PCI ID
// database at. It is enrichment only: absent, devices still enumerate with
// their numeric vendor and device IDs.
var defaultPCIIDFiles = []string{
	"/usr/share/hwdata/pci.ids",
	"/usr/share/misc/pci.ids",
	"/usr/share/pci.ids",
	"/var/lib/pciutils/pci.ids",
}

// sysfsBusRootPrefixes are the sysfs device-tree roots that can host a
// graphics device. These are kernel conventions, not facts about any
// particular machine: PCI buses on x86 and most servers, platform and SoC
// buses on ARM boards where the GPU is not on a PCI bus at all.
var sysfsBusRootPrefixes = []string{"pci", "platform", "soc"}

// sysfsWalkDepth bounds the descent below a bus root. A PCI endpoint sits at
// most a few bridges deep; the bound keeps enumeration from walking unrelated
// subsystem trees.
const sysfsWalkDepth = 8

var (
	pciAddressPattern = regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]$`)
	drmCardPattern    = regexp.MustCompile(`^card[0-9]+$`)
	drmRenderPattern  = regexp.MustCompile(`^renderD[0-9]+$`)
)

// pciBaseClassDisplay is the PCI base class for display controllers. It covers
// VGA (0x0300), XGA, 3D controllers with no display output (0x0302) and other
// display controllers (0x0380).
const (
	pciBaseClassDisplay = 0x03
	// pciBaseClassBridge marks the PCI bridges that can have devices beneath
	// them; every other class is a leaf as far as this walk is concerned.
	pciBaseClassBridge = 0x06
)

type deviceEnumerationResult struct {
	Devices     []Device
	Status      string
	NamesStatus string
	NamesFile   string
	Warnings    []string
}

func sysfsEnumerator(roots DeviceRoots) sysfsDeviceEnumerator {
	enumerator := sysfsDeviceEnumerator{root: roots.Sysfs, pciIDFiles: roots.PCIIDs}
	if enumerator.root == "" {
		enumerator.root = "/sys"
	}
	if len(enumerator.pciIDFiles) == 0 {
		enumerator.pciIDFiles = defaultPCIIDFiles
	}
	return enumerator
}

func (e sysfsDeviceEnumerator) devicesDir() string { return filepath.Join(e.root, "devices") }

// enumerateGraphics walks the sysfs device tree and emits one device per
// graphics controller. DRM connectors and render nodes are attributes of a
// card, never devices in their own right.
func (e sysfsDeviceEnumerator) enumerateGraphics() deviceEnumerationResult {
	result := deviceEnumerationResult{NamesStatus: "not_needed"}
	devicesDir := e.devicesDir()
	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		result.Status = "unavailable"
		result.Warnings = append(result.Warnings, fmt.Sprintf("read %s: %v", devicesDir, err))
		return result
	}
	busRoots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		for _, prefix := range sysfsBusRootPrefixes {
			if strings.HasPrefix(entry.Name(), prefix) {
				busRoots = append(busRoots, entry.Name())
				break
			}
		}
	}
	if len(busRoots) == 0 {
		result.Status = "no_devices"
		return result
	}
	sort.Strings(busRoots)

	devices := make([]Device, 0, 4)
	for _, busRoot := range busRoots {
		found, warnings := e.walkBus(devicesDir, busRoot)
		devices = append(devices, found...)
		result.Warnings = append(result.Warnings, warnings...)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].ID < devices[j].ID })
	if len(devices) == 0 {
		result.Status = "no_devices"
		return result
	}
	result.Devices = devices
	result.Status = "ok"
	e.resolveNames(&result)
	return result
}

func (e sysfsDeviceEnumerator) walkBus(devicesDir, busRoot string) ([]Device, []string) {
	var devices []Device
	var warnings []string
	rootDepth := strings.Count(filepath.ToSlash(filepath.Join(devicesDir, busRoot)), "/")

	err := filepath.WalkDir(filepath.Join(devicesDir, busRoot), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			// A sysfs subtree can disappear mid-walk when a device is
			// removed. Skip it rather than abandoning the whole enumeration.
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if strings.Count(filepath.ToSlash(path), "/")-rootDepth > sysfsWalkDepth {
			return filepath.SkipDir
		}
		uevent := readSysfsUevent(path)
		device, ok := e.inspect(devicesDir, path, uevent)
		if ok {
			devices = append(devices, device)
			// A graphics controller has no graphics controller beneath it;
			// its children are nodes and connectors.
			return filepath.SkipDir
		}
		if path != filepath.Join(devicesDir, busRoot) && !mayContainDevices(path, uevent) {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("walk %s: %v", filepath.Join(devicesDir, busRoot), err))
	}
	return devices, warnings
}

// mayContainDevices prunes the walk to the parts of the tree that can hold a
// device. sysfs marks every device directory with a uevent file, so a
// directory without one (power, link, msi_irqs and the other attribute
// groups) holds no devices. A PCI device that is not a bridge is likewise a
// leaf: an endpoint has no PCI devices beneath it. Pruning matters because
// hostinventory sits on hot paths where an unpruned walk of /sys/devices costs
// tens of milliseconds per snapshot.
func mayContainDevices(path string, uevent map[string]string) bool {
	if len(uevent) == 0 {
		if _, err := os.Stat(filepath.Join(path, "uevent")); err != nil {
			return false
		}
		return true
	}
	if raw := uevent["PCI_CLASS"]; raw != "" {
		value, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 32)
		if err == nil && value>>16 != pciBaseClassBridge {
			return false
		}
	}
	return true
}

// inspect decides whether a sysfs directory is a graphics device and builds
// its record. Identity comes from the PCI slot name where the device is on a
// PCI bus and from the device-tree path otherwise; neither depends on the
// order directories were read in.
func (e sysfsDeviceEnumerator) inspect(devicesDir, path string, uevent map[string]string) (Device, bool) {
	nodes := readDRMNodes(path)

	baseClass, hasClass := sysfsBaseClass(path, uevent)
	isGraphics := (hasClass && baseClass == pciBaseClassDisplay) || len(nodes) > 0
	if !isGraphics {
		return Device{}, false
	}

	device := Device{
		Class:        DeviceClassGraphics,
		Driver:       uevent["DRIVER"],
		Nodes:        nodes,
		DiscoveredBy: "linux-sysfs-device-tree",
	}
	if slot := uevent["PCI_SLOT_NAME"]; slot != "" {
		device.ID = "pci:" + strings.ToLower(slot)
	} else {
		device.ID = sysfsDeviceID(devicesDir, path)
	}
	device.Parent = sysfsDeviceID(devicesDir, filepath.Dir(path))

	if vendor, model, ok := splitPCIID(uevent["PCI_ID"]); ok {
		device.VendorID = vendor
		device.ModelID = model
	} else {
		device.VendorID = readSysfsHexID(path, "vendor")
		device.ModelID = readSysfsHexID(path, "device")
	}
	return device, true
}

// sysfsDeviceID renders a stable identity for a sysfs path: the PCI address
// when the directory is a PCI endpoint or bridge, and the device-tree path
// under /sys/devices otherwise.
func sysfsDeviceID(devicesDir, path string) string {
	name := filepath.Base(path)
	if pciAddressPattern.MatchString(name) {
		return "pci:" + strings.ToLower(name)
	}
	relative, err := filepath.Rel(devicesDir, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return ""
	}
	return "sysfs:" + filepath.ToSlash(relative)
}

func sysfsBaseClass(path string, uevent map[string]string) (uint64, bool) {
	if raw := uevent["PCI_CLASS"]; raw != "" {
		if value, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 32); err == nil {
			return value >> 16, true
		}
	}
	data, err := os.ReadFile(filepath.Join(path, "class"))
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(string(data)), "0x"), 16, 32)
	if err != nil {
		return 0, false
	}
	return value >> 16, true
}

func readSysfsUevent(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(filepath.Join(path, "uevent"))
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || key == "" {
			continue
		}
		values[key] = value
	}
	return values
}

// readDRMNodes lists the kernel nodes a graphics device exposes. Only card and
// render nodes are nodes; entries such as card1-DP-1 are connectors nested
// inside a card and are outputs, not devices.
func readDRMNodes(path string) []string {
	entries, err := os.ReadDir(filepath.Join(path, "drm"))
	if err != nil {
		return nil
	}
	nodes := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if drmCardPattern.MatchString(name) || drmRenderPattern.MatchString(name) {
			nodes = append(nodes, name)
		}
	}
	if len(nodes) == 0 {
		return nil
	}
	sort.Strings(nodes)
	return nodes
}

func readSysfsHexID(path, name string) string {
	data, err := os.ReadFile(filepath.Join(path, name))
	if err != nil {
		return ""
	}
	value := strings.ToLower(strings.TrimSpace(string(data)))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "0x") {
		value = "0x" + value
	}
	return value
}

// splitPCIID splits a uevent PCI_ID such as "10DE:2705" into vendor and device
// identifiers.
func splitPCIID(raw string) (string, string, bool) {
	vendor, device, found := strings.Cut(strings.TrimSpace(raw), ":")
	if !found || vendor == "" || device == "" {
		return "", "", false
	}
	return "0x" + strings.ToLower(vendor), "0x" + strings.ToLower(device), true
}

// resolveNames turns numeric PCI IDs into vendor and model names using the
// host's PCI ID database. The database is optional: when it is absent the
// devices keep their numeric identifiers and the probe says so.
func (e sysfsDeviceEnumerator) resolveNames(result *deviceEnumerationResult) {
	wanted := map[string]bool{}
	for _, device := range result.Devices {
		if device.VendorID != "" {
			wanted[device.VendorID] = true
		}
	}
	if len(wanted) == 0 {
		return
	}
	for _, candidate := range e.pciIDFiles {
		vendors, models, err := readPCIIDs(candidate, result.Devices)
		if err != nil {
			continue
		}
		for i := range result.Devices {
			device := &result.Devices[i]
			if name, ok := vendors[device.VendorID]; ok {
				device.Vendor = name
			}
			if name, ok := models[device.VendorID+":"+device.ModelID]; ok {
				device.Model = name
			}
			if device.Vendor != "" || device.Model != "" {
				device.EnrichedBy = appendUnique(device.EnrichedBy, "pci.ids")
			}
		}
		result.NamesStatus = "ok"
		result.NamesFile = candidate
		return
	}
	result.NamesStatus = "unavailable"
}

// readPCIIDs scans the pci.ids database for exactly the vendor and device
// identifiers the enumerated devices carry. It reads line by line and stops as
// soon as every wanted pair is resolved so a 1.4MB database costs little.
func readPCIIDs(path string, devices []Device) (map[string]string, map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	wantedVendors := map[string]bool{}
	wantedModels := map[string]bool{}
	for _, device := range devices {
		if device.VendorID != "" {
			wantedVendors[strings.TrimPrefix(device.VendorID, "0x")] = true
		}
		if device.VendorID != "" && device.ModelID != "" {
			wantedModels[strings.TrimPrefix(device.VendorID, "0x")+":"+strings.TrimPrefix(device.ModelID, "0x")] = true
		}
	}
	vendors := map[string]string{}
	models := map[string]string{}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	currentVendor := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case !strings.HasPrefix(line, "\t"):
			// Device class sections follow the vendor sections; nothing past
			// them can resolve a vendor or device name.
			if strings.HasPrefix(line, "C ") {
				currentVendor = ""
				continue
			}
			id, name, ok := splitPCIIDsEntry(line)
			if !ok {
				currentVendor = ""
				continue
			}
			currentVendor = id
			if wantedVendors[id] {
				vendors["0x"+id] = name
			}
		case !strings.HasPrefix(line, "\t\t"):
			if currentVendor == "" {
				continue
			}
			id, name, ok := splitPCIIDsEntry(strings.TrimPrefix(line, "\t"))
			if !ok {
				continue
			}
			if wantedModels[currentVendor+":"+id] {
				models["0x"+currentVendor+":0x"+id] = name
			}
		}
		if len(vendors) == len(wantedVendors) && len(models) == len(wantedModels) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if len(vendors) == 0 && len(models) == 0 {
		return nil, nil, fmt.Errorf("pci id database %s resolved no identifiers", path)
	}
	return vendors, models, nil
}

func splitPCIIDsEntry(line string) (string, string, bool) {
	id, name, found := strings.Cut(line, "  ")
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if !found || id == "" || name == "" {
		return "", "", false
	}
	if _, err := strconv.ParseUint(id, 16, 32); err != nil {
		return "", "", false
	}
	return strings.ToLower(id), name, true
}
