package devicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// macOS probe binaries. All three ship with the operating system; none of them
// requires privilege for the data this package reads.
const (
	darwinProfilerTool = "system_profiler"
	darwinNetworkTool  = "networksetup"
	darwinNetstatTool  = "netstat"
)

const darwinProbeTimeout = 15 * time.Second

// collectDarwinGraph builds the device graph from the macOS system profiler and
// the BSD network tools. It is a genuine but narrower backend than the Linux
// one: macOS exposes no unprivileged thermal interface and no ECC counters, and
// both of those absences are reported rather than skipped.
func collectDarwinGraph(ctx context.Context, b *builder) {
	collectDarwinBuses(ctx, b)
	collectDarwinStorage(ctx, b)
	collectDarwinNetwork(ctx, b)
	darwinThermal(b)
	darwinMemoryErrors(b)
}

// runDarwinProfiler asks the system profiler for one or more data types and
// flattens the nested "_items" structure it returns.
func runDarwinProfiler(ctx context.Context, env Env, dataType string) ([]map[string]any, error) {
	binary, err := env.LookPath(darwinProfilerTool)
	if err != nil {
		return nil, fmt.Errorf("%s is not on PATH: %w", darwinProfilerTool, err)
	}
	output, runErr := env.Run(ctx, darwinProbeTimeout, binary, "-json", dataType)
	if len(strings.TrimSpace(string(output))) == 0 {
		if runErr != nil {
			return nil, fmt.Errorf("%s %s produced no output: %w", darwinProfilerTool, dataType, runErr)
		}
		return nil, fmt.Errorf("%s %s produced no output", darwinProfilerTool, dataType)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(output, &document); err != nil {
		return nil, fmt.Errorf("%s %s output was not JSON: %w", darwinProfilerTool, dataType, err)
	}
	raw, present := document[dataType]
	if !present {
		return nil, fmt.Errorf("%s returned no %s section", darwinProfilerTool, dataType)
	}
	var roots []map[string]any
	if err := json.Unmarshal(raw, &roots); err != nil {
		return nil, fmt.Errorf("%s %s section was not a list: %w", darwinProfilerTool, dataType, err)
	}
	return flattenProfilerItems(roots), nil
}

func collectDarwinBuses(ctx context.Context, b *builder) {
	pciItems, pciErr := runDarwinProfiler(ctx, b.env, "SPPCIDataType")
	if pciErr != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "pci-bus", pciErr.Error(), darwinProfilerTool+" -json SPPCIDataType"))
	} else {
		for _, item := range pciItems {
			name := jsonString(item, "_name")
			slot := jsonString(item, "sppci_slot_name")
			identity := slot
			if identity == "" {
				identity = name
			}
			if identity == "" {
				continue
			}
			device := Device{
				ID:     "pci-slot:" + sanitizeIdentity(identity),
				Class:  ClassPCIDevice,
				Vendor: jsonString(item, "sppci_vendor"),
				Model:  name,
				Driver: jsonString(item, "sppci_driver_installed"),
				Rungs:  map[Rung]RungState{},
			}
			deviceType := jsonString(item, "sppci_device_type")
			setAttribute(&device, "device_type", deviceType)
			setAttribute(&device, "slot", slot)
			if strings.Contains(strings.ToLower(deviceType), "display") {
				device.Class = ClassGraphicsDevice
			}
			enumerationRungs(b, &device, darwinProfilerTool+" -json SPPCIDataType")
			b.add(device)
		}
	}

	usbItems, usbErr := runDarwinProfiler(ctx, b.env, "SPUSBDataType")
	if usbErr != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "usb-bus", usbErr.Error(), darwinProfilerTool+" -json SPUSBDataType"))
		return
	}
	for _, item := range usbItems {
		location := jsonString(item, "location_id")
		if location == "" {
			continue
		}
		device := Device{
			ID:     "usb-location:" + sanitizeIdentity(location),
			Class:  ClassUSBDevice,
			Vendor: jsonString(item, "manufacturer"),
			Model:  jsonString(item, "_name"),
			Rungs:  map[Rung]RungState{},
		}
		setAttribute(&device, "vendor_id", normalizeHexID(firstField(jsonString(item, "vendor_id"))))
		setAttribute(&device, "product_id", normalizeHexID(firstField(jsonString(item, "product_id"))))
		enumerationRungs(b, &device, darwinProfilerTool+" -json SPUSBDataType")
		b.add(device)
	}
}

func collectDarwinStorage(ctx context.Context, b *builder) {
	mechanism := darwinProfilerTool + " -json SPNVMeDataType/SPSerialATADataType"
	items := make([]map[string]any, 0, 4)
	failures := make([]string, 0, 2)
	for _, dataType := range []string{"SPNVMeDataType", "SPSerialATADataType"} {
		found, err := runDarwinProfiler(ctx, b.env, dataType)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		items = append(items, found...)
	}
	if len(items) == 0 {
		reason := "macOS reported no storage devices"
		if len(failures) > 0 {
			reason = strings.Join(failures, "; ")
		}
		b.graph.addSubsystem(unavailableSubsystem(b, "block-storage", reason, mechanism))
		return
	}

	disks := 0
	for _, item := range items {
		bsdName := jsonString(item, "bsd_name")
		if bsdName == "" {
			continue
		}
		device := Device{
			ID:    "block:" + bsdName,
			Class: ClassBlockDevice,
			Model: firstNonEmpty(jsonString(item, "device_model"), jsonString(item, "_name")),
			Rungs: map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", bsdName)
		setAttribute(&device, "transport", darwinTransport(item))
		if size, ok := jsonNumber(item, "size_in_bytes"); ok {
			setReading(&device, "capacity_bytes", size)
		}
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		telemetry := b.grader.unavailable(RungTelemetry,
			"macOS exposes no per-device I/O counter equivalent to the Linux block stat interface without a privileged iostat sample",
			mechanism)
		device.Rungs[RungTelemetry] = telemetry
		device.Rungs[RungEvidence] = b.grader.evidenceFor(telemetry)

		smart := readSMART(ctx, b.env, filepath.Join(b.env.DevRoot, bsdName))
		applySMART(b, &device, smart)
		b.add(device)
		disks++
	}
	b.graph.addSubsystem(Subsystem{
		Name:       "block-storage",
		Attributes: map[string]string{"enumerated_disks": strconv.Itoa(disks)},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, mechanism),
			b.grader.notApplicable(RungTelemetry, "storage telemetry is graded per device"),
			b.grader.notApplicable(RungEvidence, "storage telemetry is retained per device"),
			b.grader.notApplicable(RungControl, "SMART access is graded per device"),
			b.grader.notApplicable(RungAnticipation, "wear and error accrual are graded per device"),
		),
	})
}

func darwinTransport(item map[string]any) string {
	switch {
	case jsonString(item, "spnvme_trim_support") != "" || strings.Contains(strings.ToLower(jsonString(item, "_parent_name")), "nvm"):
		return "nvme"
	case jsonString(item, "spsata_medium_type") != "":
		return "sata"
	default:
		return ""
	}
}

// collectDarwinNetwork uses networksetup's hardware-port list, which is macOS's
// own structural statement about which interfaces are hardware. Interfaces
// absent from that list are virtual and are recorded, not graded.
func collectDarwinNetwork(ctx context.Context, b *builder) {
	mechanism := darwinNetworkTool + " -listallhardwareports"
	binary, err := b.env.LookPath(darwinNetworkTool)
	if err != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "network-interfaces",
			fmt.Sprintf("%s is not on PATH: %v", darwinNetworkTool, err), mechanism))
		return
	}
	output, runErr := b.env.Run(ctx, darwinProbeTimeout, binary, "-listallhardwareports")
	ports := parseDarwinHardwarePorts(string(output))
	if len(ports) == 0 {
		reason := "macOS reported no hardware network ports"
		if runErr != nil {
			reason = fmt.Sprintf("%s failed: %v", darwinNetworkTool, runErr)
		}
		b.graph.addSubsystem(unavailableSubsystem(b, "network-interfaces", reason, mechanism))
		return
	}

	counters := darwinInterfaceCounters(ctx, b.env)
	for _, port := range ports {
		device := Device{
			ID:    "net:" + port.device,
			Class: ClassNetworkInterface,
			Model: port.name,
			Rungs: map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", port.device)
		setAttribute(&device, "hardware_port", port.name)
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		if stats, ok := counters[port.device]; ok {
			for key, value := range stats {
				setReading(&device, key, value)
			}
			setReading(&device, readingInterfaceErrors, stats["rx_errors"]+stats["tx_errors"])
			device.Rungs[RungTelemetry] = b.grader.measured(RungTelemetry, darwinNetstatTool+" -ibn")
		} else {
			device.Rungs[RungTelemetry] = b.grader.unmeasurable(RungTelemetry,
				fmt.Sprintf("%s -ibn reported no counters for %s", darwinNetstatTool, port.device),
				darwinNetstatTool+" -ibn")
		}
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.unavailable(RungControl,
			"macOS reports negotiated link speed only through a privileged IOKit query, which this scenario does not perform",
			mechanism)
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "interface error")
		b.add(device)
	}
	b.graph.addSubsystem(Subsystem{
		Name:       "network-interfaces",
		Attributes: map[string]string{"physical_interfaces": strconv.Itoa(len(ports))},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, mechanism),
			b.grader.notApplicable(RungTelemetry, "counters are graded on each interface"),
			b.grader.notApplicable(RungEvidence, "counters are retained per interface"),
			b.grader.notApplicable(RungControl, "link configuration is graded per interface"),
			b.grader.notApplicable(RungAnticipation, "error accrual is graded per interface"),
		),
	})
}

type darwinHardwarePort struct {
	name   string
	device string
}

func parseDarwinHardwarePorts(output string) []darwinHardwarePort {
	ports := make([]darwinHardwarePort, 0, 4)
	current := darwinHardwarePort{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Hardware Port:"):
			current = darwinHardwarePort{name: strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))}
		case strings.HasPrefix(line, "Device:"):
			current.device = strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if current.device != "" {
				ports = append(ports, current)
			}
			current = darwinHardwarePort{}
		}
	}
	return ports
}

// darwinInterfaceCounters parses `netstat -ibn`, whose first row per interface
// carries the cumulative packet, error and byte counters.
func darwinInterfaceCounters(ctx context.Context, env Env) map[string]map[string]float64 {
	counters := map[string]map[string]float64{}
	binary, err := env.LookPath(darwinNetstatTool)
	if err != nil {
		return counters
	}
	output, _ := env.Run(ctx, darwinProbeTimeout, binary, "-ibn")
	for index, line := range strings.Split(string(output), "\n") {
		if index == 0 {
			continue
		}
		fields := strings.Fields(line)
		// Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll
		if len(fields) < 11 {
			continue
		}
		name := fields[0]
		if _, seen := counters[name]; seen {
			continue
		}
		parsed := map[string]float64{}
		for key, position := range map[string]int{
			"rx_packets": 4, "rx_errors": 5, "rx_bytes": 6,
			"tx_packets": 7, "tx_errors": 8, "tx_bytes": 9, "collisions": 10,
		} {
			value, err := strconv.ParseFloat(fields[position], 64)
			if err != nil {
				continue
			}
			parsed[key] = value
		}
		if len(parsed) > 0 {
			counters[name] = parsed
		}
	}
	return counters
}

func darwinThermal(b *builder) {
	const reason = "macOS exposes SMC temperatures only through the privileged powermetrics interface; " +
		"this scenario never escalates at runtime, so no temperature is observable"
	b.graph.addSubsystem(Subsystem{
		Name: "thermal",
		Rungs: rungSet(
			b.grader.unavailable(RungIdentity, reason, "powermetrics"),
			b.grader.unavailable(RungTelemetry, reason, "powermetrics"),
			b.grader.unavailable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
			remediated(b.grader.unavailable(RungControl, reason, "powermetrics"),
				"grant the monitor user access to powermetrics at commissioning time, or accept that macOS thermal telemetry stays unobservable"),
			b.grader.unavailable(RungAnticipation, "no temperature trend: "+reason, trendMechanism),
		),
	})
}

func darwinMemoryErrors(b *builder) {
	const reason = "macOS exposes no EDAC-equivalent ECC counter interface, so memory errors are not observable on this platform"
	b.graph.addSubsystem(Subsystem{
		Name: SubsystemMemoryErrors,
		Rungs: rungSet(
			b.grader.notApplicable(RungIdentity, reason),
			b.grader.notApplicable(RungTelemetry, reason),
			b.grader.notApplicable(RungEvidence, reason),
			b.grader.notApplicable(RungControl, reason),
			b.grader.notApplicable(RungAnticipation, reason),
		),
	})
}
