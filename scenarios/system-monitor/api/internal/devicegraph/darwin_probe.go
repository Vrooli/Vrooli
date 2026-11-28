package devicegraph

import (
	"context"
	"encoding/json"
	"errors"
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
	collectNativeBusDevices(ctx, b)
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
		return nil, fmt.Errorf("%w: %s is not on PATH: %w", errProbeMechanismUnavailable, darwinProfilerTool, err)
	}
	output, runErr := env.Run(ctx, darwinProbeTimeout, binary, "-json", dataType)
	if runErr != nil {
		return nil, fmt.Errorf("%s %s failed: %w", darwinProfilerTool, dataType, runErr)
	}
	if len(strings.TrimSpace(string(output))) == 0 {
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

func collectDarwinStorage(ctx context.Context, b *builder) {
	mechanism := darwinProfilerTool + " -json SPNVMeDataType/SPSerialATADataType"
	items := make([]map[string]any, 0, 4)
	failures := make([]error, 0, 2)
	for _, dataType := range []string{"SPNVMeDataType", "SPSerialATADataType"} {
		found, err := runDarwinProfiler(ctx, b.env, dataType)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		items = append(items, found...)
	}
	if len(items) == 0 && len(failures) > 0 {
		b.graph.addSubsystem(probeFailureSubsystem(b, "block-storage", errors.Join(failures...), mechanism))
		return
	}
	if len(items) == 0 {
		b.graph.addSubsystem(Subsystem{
			Name: "block-storage", Attributes: map[string]string{"enumerated_disks": "0", "enumeration": "ok"},
			Rungs: rungSet(
				b.grader.measured(RungIdentity, mechanism),
				b.grader.notApplicable(RungTelemetry, "storage telemetry is graded per device"),
				b.grader.notApplicable(RungEvidence, "storage telemetry is retained per device"),
				b.grader.notApplicable(RungControl, "SMART access is graded per device"),
				b.grader.notApplicable(RungAnticipation, "wear and error accrual are graded per device"),
			),
		})
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
	if len(failures) > 0 {
		b.graph.addSubsystem(unmeasurableEnumerationSubsystem(b, "block-storage",
			fmt.Sprintf("%s returned partial storage inventory: %v", mechanism, errors.Join(failures...)), mechanism))
		return
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
	if runErr != nil && len(ports) == 0 {
		b.graph.addSubsystem(unmeasurableEnumerationSubsystem(b, "network-interfaces",
			fmt.Sprintf("%s failed before a complete hardware-port enumeration: %v", darwinNetworkTool, runErr), mechanism))
		return
	}

	counters, counterToolAvailable, counterErr := darwinInterfaceCounters(ctx, b.env)
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
		} else if !counterToolAvailable {
			device.Rungs[RungTelemetry] = b.grader.unavailable(RungTelemetry,
				fmt.Sprintf("%s is unavailable: %v", darwinNetstatTool, counterErr), darwinNetstatTool+" -ibn")
		} else if counterErr != nil {
			device.Rungs[RungTelemetry] = b.grader.unmeasurable(RungTelemetry,
				fmt.Sprintf("%s failed before a complete interface-counter sample: %v", darwinNetstatTool, counterErr),
				darwinNetstatTool+" -ibn")
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
	if runErr != nil {
		b.graph.addSubsystem(unmeasurableEnumerationSubsystem(b, "network-interfaces",
			fmt.Sprintf("%s returned partial hardware-port output before failing: %v", darwinNetworkTool, runErr), mechanism))
		return
	}
	b.graph.addSubsystem(Subsystem{
		Name:       "network-interfaces",
		Attributes: map[string]string{"physical_interfaces": strconv.Itoa(len(ports)), "enumeration": "ok"},
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
func darwinInterfaceCounters(ctx context.Context, env Env) (map[string]map[string]float64, bool, error) {
	counters := map[string]map[string]float64{}
	binary, err := env.LookPath(darwinNetstatTool)
	if err != nil {
		return nil, false, err
	}
	output, runErr := env.Run(ctx, darwinProbeTimeout, binary, "-ibn")
	if runErr != nil {
		return nil, true, runErr
	}
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
	return counters, true, nil
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
