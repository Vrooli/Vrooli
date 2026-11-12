package hostinventory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	devicesUnavailable        = "unavailable"
	devicesUnrecognizedOutput = "unrecognized_output"
)

// DeviceClass is the applicability-relevant category of a host device. It
// answers "what kind of thing is this", not "how is it doing" — anything with
// a plural or a tense in it belongs to a monitoring scenario, not here.
type DeviceClass string

const (
	// DeviceClassGraphics covers every PCI display controller (VGA, 3D and
	// display base class 0x03) and every non-PCI device that exposes a DRM
	// card node. Compute-only accelerators with no display output are
	// deliberately included: they are graphics-class devices to the kernel.
	DeviceClassGraphics DeviceClass = "graphics"
)

// Device is one physical device as the operating system's own device tree
// reports it. Devices are discovered from the tree, never from a vendor tool:
// a vendor tool can only see its own vendor's hardware, so using one for
// discovery makes every other vendor's device invisible to the whole stack.
// Vendor tools may only enrich a device the tree already found.
//
// ID is a stable, platform-durable address. It must survive a reboot and must
// never depend on enumeration order, so a bare index or a kernel node name
// (card0, renderD128) is never an identity. The address form per platform is:
//
//   - Linux: "pci:<domain>:<bus>:<device>.<function>" (for example
//     "pci:tuning.PermNone:01:00.0"), taken from the device's PCI_SLOT_NAME in sysfs.
//     Devices that are not on a PCI bus — SoC GPUs on ARM boards, for example
//     — use "sysfs:<path under /sys/devices>", which is the firmware-assigned
//     device-tree path and is equally durable.
//   - Windows: "pnp:<PNP device instance ID>" (for example
//     "pnp:PCI\VEN_10DE&DEV_2705&SUBSYS_89571043&REV_A1\4&2E5A2E5B&0&0008").
//     The PnP manager derives the instance ID from bus topology, so it is
//     Windows' own equivalent of the PCI address and is stable across reboots.
//   - macOS: "ioreg:<IORegistry IOService path>" (for example
//     "ioreg:/AppleACPIPlatformExpert/PCI0@0/AppleACPIPCI/GFX0@1"), with
//     "pci:<domain>:<bus>:<device>.<function>" preferred on Intel Macs where
//     an IOPCIDevice exposes a bus address. The IORegistry path is built from
//     the firmware device tree, so it does not depend on probe order. macOS
//     enumeration uses the shared native IORegistry adapter.
type Device struct {
	ID     string      `json:"id"`
	Class  DeviceClass `json:"class"`
	Parent string      `json:"parent,omitempty"`

	Vendor   string `json:"vendor,omitempty"`
	VendorID string `json:"vendor_id,omitempty"`
	Model    string `json:"model,omitempty"`
	ModelID  string `json:"model_id,omitempty"`

	// Driver is the kernel driver or adapter bound to the device. An empty
	// Driver on an enumerated device means no driver is bound, which is a
	// real and reportable state, not a probe failure.
	Driver        string `json:"driver,omitempty"`
	DriverVersion string `json:"driver_version,omitempty"`

	// Nodes are the kernel device nodes the device exposes (Linux DRM card
	// and render nodes). Connectors such as card1-DP-1 are outputs of a card,
	// not devices, and never appear here or as devices of their own.
	Nodes []string `json:"nodes,omitempty"`

	// DiscoveredBy names the probe that enumerated the device. A value naming
	// a vendor tool means the device has no device-tree identity, which is a
	// finding rather than a normal result.
	DiscoveredBy string `json:"discovered_by"`
	// EnrichedBy names every probe that added fields to an already-discovered
	// device, in the order they ran.
	EnrichedBy []string `json:"enriched_by,omitempty"`
}

// DeviceRoots overrides the filesystem locations device-tree enumeration reads.
// The zero value means the live host.
type DeviceRoots struct {
	// Sysfs is the sysfs mount point. Empty means "/sys".
	Sysfs string
	// PCIIDs lists candidate pci.ids databases used to turn numeric PCI
	// vendor and device IDs into names. Empty means the well-known
	// distribution locations.
	PCIIDs []string
}

// DevicesOfClass returns every enumerated device of a class, in enumeration
// order (which is sorted by identity, not by discovery order).
func (s Snapshot) DevicesOfClass(class DeviceClass) []Device {
	matched := make([]Device, 0, len(s.Devices))
	for _, device := range s.Devices {
		if device.Class == class {
			matched = append(matched, device)
		}
	}
	return matched
}

// Device returns the enumerated device with a stable identity.
func (s Snapshot) Device(id string) (Device, bool) {
	for _, device := range s.Devices {
		if device.ID == id {
			return device, true
		}
	}
	return Device{}, false
}

// collectDevices enumerates host devices from the operating system's device
// tree. Every platform reports a probe status: a platform without an
// implementation reports that it could not look, which is a different fact
// from looking and finding nothing.
func (c Collector) collectDevices(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	switch hostreqspec.PlatformFromGOOS(snap.OS) {
	case hostreqspec.PlatformLinux:
		enumerator := sysfsEnumerator(c.DeviceRoots)
		result := enumerator.enumerateGraphics()
		snap.Warnings = append(snap.Warnings, result.Warnings...)
		snap.ProbeStatuses["device_tree"] = result.Status
		snap.ProbeStatuses["device_tree_names"] = result.NamesStatus
		if len(result.Devices) == 0 {
			return
		}
		snap.Devices = append(snap.Devices, result.Devices...)
		snap.FieldProvenance["devices"] = Provenance{
			SourceKind: SourceKindFile,
			Source:     "linux sysfs device tree",
			ObservedAt: observedAt,
			Confidence: "high",
			File:       enumerator.devicesDir(),
		}
		if result.NamesStatus == "ok" {
			snap.FieldProvenance["devices.model"] = Provenance{
				SourceKind: SourceKindFile,
				Source:     "pci.ids",
				ObservedAt: observedAt,
				Confidence: "medium",
				File:       result.NamesFile,
			}
		}
	case hostreqspec.PlatformWindows, hostreqspec.PlatformMacOS:
		c.collectNativeGraphics(ctx, snap, observedAt)
	default:
		snap.ProbeStatuses["device_tree"] = string(IntegrityProbeUnsupported)
	}
}

// collectNativeGraphics projects the same identity adapter used by integrity
// and System Monitor. It carries scope failures even when useful rows survived.
func (c Collector) collectNativeGraphics(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	probe := c.NativeDevices
	if probe == nil {
		probe = CollectNativeDevices
	}
	observation := probe(ctx, c.Commands)
	state, ok := observation.ProbeStatus["devices"]
	if !ok {
		state = IntegrityProbeFailed
	}
	snap.ProbeStatuses["device_tree"] = string(state)
	for scope, status := range observation.ProbeStatus {
		snap.ProbeStatuses["device_tree."+scope] = string(status)
	}
	for scope, problem := range observation.ProbeErrors {
		snap.Warnings = append(snap.Warnings, "native devices "+scope+": "+problem)
	}
	for _, device := range observation.Devices {
		if device.Class != "display controller" {
			continue
		}
		snap.Devices = append(snap.Devices, Device{
			ID: device.ID, Class: DeviceClassGraphics, Vendor: device.VendorName,
			VendorID: device.VendorID, Model: device.DeviceName, ModelID: device.DeviceID,
			Driver: device.BoundDriver, DiscoveredBy: observation.Mechanism,
		})
	}
	snap.FieldProvenance["devices"] = Provenance{SourceKind: SourceKindCommand,
		Source: observation.Mechanism, ObservedAt: observedAt, Confidence: "high"}
	snap.FieldProvenance["gpus"] = snap.FieldProvenance["devices"]
}

// projectNativeGraphics adds identity-only rows only where vendor telemetry has
// not already described the same adapter. Neither order nor generic vendor is a
// safe match; an exact model must identify one device and one telemetry row.
func projectNativeGraphics(snap *Snapshot) {
	for _, device := range snap.DevicesOfClass(DeviceClassGraphics) {
		match := -1
		for i, gpu := range snap.GPUs {
			if gpu.DeviceID == device.ID {
				match = i
				break
			}
			if gpu.DeviceID == "" && device.Model != "" && strings.EqualFold(gpu.Name, device.Model) {
				count := 0
				for _, candidate := range snap.Devices {
					if strings.EqualFold(candidate.Model, device.Model) {
						count++
					}
				}
				if count != 1 || match >= 0 {
					match = -1
					break
				}
				match = i
			}
		}
		if match >= 0 {
			snap.GPUs[match].DeviceID = device.ID
			continue
		}
		snap.GPUs = append(snap.GPUs, GPU{Index: len(snap.GPUs), Name: device.Model, DeviceID: device.ID, Source: SourceNativeDevice})
	}
}

// linkNvidiaDevices matches nvidia-smi output onto devices the tree already
// enumerated. nvidia-smi is an enrichment source only: it may add a driver
// version and bind a GPU telemetry record to a tree identity, but it may never
// introduce a device. A GPU it reports at a PCI address the tree did not
// enumerate is surfaced as a finding.
func (c Collector) linkNvidiaDevices(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	if len(snap.GPUs) == 0 {
		return
	}
	if !snap.RuntimeTools["nvidia-smi"].Present {
		return
	}
	const query = "--query-gpu=index,pci.bus_id"
	out, err := c.Commands.Run(ctx, "nvidia-smi", query, "--format=csv,noheader,nounits")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("nvidia-smi query pci bus id: %v", err))
		snap.ProbeStatuses["device_tree_nvidia_enrichment"] = string(IntegrityProbeFailed)
		return
	}
	addresses := ParseNvidiaPCIBusIDCSV(string(out))
	if len(addresses) == 0 {
		snap.ProbeStatuses["device_tree_nvidia_enrichment"] = devicesUnrecognizedOutput
		return
	}
	deviceIndex := map[string]int{}
	for i := range snap.Devices {
		deviceIndex[snap.Devices[i].ID] = i
	}
	unmatched := make([]string, 0)
	matched := 0
	for i := range snap.GPUs {
		if snap.GPUs[i].Source != SourceNvidiaSMI {
			continue
		}
		address, ok := addresses[snap.GPUs[i].Index]
		if !ok {
			continue
		}
		id := "pci:" + address
		position, found := deviceIndex[id]
		if !found {
			unmatched = append(unmatched, id)
			continue
		}
		snap.GPUs[i].DeviceID = id
		device := &snap.Devices[position]
		if snap.GPUs[i].DriverVersion != "" {
			device.DriverVersion = snap.GPUs[i].DriverVersion
		}
		if !slices.Contains(device.EnrichedBy, "nvidia-smi") {
			device.EnrichedBy = append(device.EnrichedBy, "nvidia-smi")
		}
		matched++
	}
	if len(unmatched) > 0 {
		slices.Sort(unmatched)
		snap.Warnings = append(snap.Warnings, fmt.Sprintf(
			"nvidia-smi reported GPUs at addresses the device tree did not enumerate: %s",
			strings.Join(unmatched, ", ")))
		snap.ProbeStatuses["device_tree_nvidia_enrichment"] = "unmatched_devices"
		return
	}
	if matched == 0 {
		snap.ProbeStatuses["device_tree_nvidia_enrichment"] = collectorNoDevices
		return
	}
	snap.ProbeStatuses["device_tree_nvidia_enrichment"] = "ok"
	snap.FieldProvenance["devices.driver_version"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "nvidia-smi",
		ObservedAt: observedAt,
		Confidence: "high",
		Command:    "nvidia-smi " + query + " --format=csv,noheader,nounits",
	}
}
