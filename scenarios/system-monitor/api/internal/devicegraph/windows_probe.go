package devicegraph

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// windowsShell is the interpreter every Windows probe runs through. Each probe
// is a read-only cmdlet; none of them requires an elevated session.
const windowsShell = "powershell.exe"

const windowsProbeTimeout = 20 * time.Second

// Windows reports thermal-zone temperatures in tenths of a Kelvin.
const (
	windowsDeciKelvinPerKelvin = 10.0
	kelvinOffset               = 273.15
)

// collectWindowsGraph builds the device graph from the storage, network and
// Plug-and-Play cmdlets. Storage reliability counters give Windows a genuine
// rung-five signal without smartctl; thermal and ECC do not have an
// unprivileged equivalent and say so.
func collectWindowsGraph(ctx context.Context, b *builder) {
	shell, err := b.env.LookPath(windowsShell)
	if err != nil {
		unsupportedPlatform(b, fmt.Sprintf("%s is not on PATH, so no Windows device probe can run: %v", windowsShell, err))
		return
	}
	collectWindowsPnP(ctx, b, shell)
	collectWindowsStorage(ctx, b, shell)
	collectWindowsNetwork(ctx, b, shell)
	collectWindowsThermal(ctx, b, shell)
	windowsMemoryErrors(b)
}

// runWindowsJSON runs one cmdlet pipeline and decodes its JSON. ConvertTo-Json
// emits a bare object when the pipeline yields exactly one item and an array
// otherwise, so both shapes are accepted.
func runWindowsJSON(ctx context.Context, env Env, shell, script string) ([]map[string]any, error) {
	output, runErr := env.Run(ctx, windowsProbeTimeout, shell, "-NoProfile", "-NonInteractive", "-Command", script)
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		if runErr != nil {
			return nil, fmt.Errorf("probe produced no output: %w", runErr)
		}
		return nil, fmt.Errorf("probe produced no output")
	}
	return decodeJSONObjects([]byte(trimmed))
}

func collectWindowsPnP(ctx context.Context, b *builder, shell string) {
	const script = `Get-PnpDevice -PresentOnly | Select-Object InstanceId,FriendlyName,Class,Manufacturer,Service,Status | ConvertTo-Json -Compress -Depth 3`
	mechanism := "powershell Get-PnpDevice -PresentOnly"
	items, err := runWindowsJSON(ctx, b.env, shell, script)
	if err != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "pci-bus", err.Error(), mechanism))
		b.graph.addSubsystem(unavailableSubsystem(b, "usb-bus", err.Error(), mechanism))
		return
	}
	pci, usb := 0, 0
	for _, item := range items {
		instance := jsonString(item, "InstanceId")
		if instance == "" {
			continue
		}
		prefix, _, _ := strings.Cut(instance, `\`)
		class := ClassPCIDevice
		switch strings.ToUpper(prefix) {
		case "PCI":
			pci++
		case "USB":
			class = ClassUSBDevice
			usb++
		default:
			continue
		}
		// The identity is the PnP instance id verbatim, which is the same
		// platform-durable address the shared host inventory uses on Windows,
		// so the two enumerations join without a translation table.
		identity := "pnp:" + instance
		device := Device{
			ID:     identity,
			Class:  class,
			Vendor: jsonString(item, "Manufacturer"),
			Model:  jsonString(item, "FriendlyName"),
			Driver: jsonString(item, "Service"),
			Rungs:  map[Rung]RungState{},
		}
		setAttribute(&device, "instance_id", instance)
		setAttribute(&device, "pnp_class", jsonString(item, "Class"))
		setAttribute(&device, "pnp_status", jsonString(item, "Status"))
		if strings.EqualFold(jsonString(item, "Class"), "Display") {
			device.Class = ClassGraphicsDevice
		}
		enumerationRungs(b, &device, mechanism)
		b.add(device)
	}
	b.graph.addSubsystem(Subsystem{
		Name:       "pci-bus",
		Attributes: map[string]string{"enumerated_devices": strconv.Itoa(pci)},
		Rungs:      windowsEnumerationSubsystem(b, mechanism),
	})
	b.graph.addSubsystem(Subsystem{
		Name:       "usb-bus",
		Attributes: map[string]string{"enumerated_devices": strconv.Itoa(usb)},
		Rungs:      windowsEnumerationSubsystem(b, mechanism),
	})
}

func windowsEnumerationSubsystem(b *builder, mechanism string) map[Rung]RungState {
	return rungSet(
		b.grader.measured(RungIdentity, mechanism),
		b.grader.notApplicable(RungTelemetry, "bus enumeration reports configuration, not a live measurement"),
		b.grader.notApplicable(RungEvidence, "bus enumeration is retained per device"),
		b.grader.notApplicable(RungControl, "driver binding is graded per device"),
		b.grader.notApplicable(RungAnticipation, "bus enumeration carries no forward-looking signal"),
	)
}

func collectWindowsStorage(ctx context.Context, b *builder, shell string) {
	const script = `Get-PhysicalDisk | Select-Object DeviceId,SerialNumber,FriendlyName,Manufacturer,Model,MediaType,BusType,Size,HealthStatus | ConvertTo-Json -Compress -Depth 3`
	const reliabilityScript = `Get-PhysicalDisk | ForEach-Object { $c = $_ | Get-StorageReliabilityCounter; ` +
		`[pscustomobject]@{DeviceId=$_.DeviceId;Wear=$c.Wear;PowerOnHours=$c.PowerOnHours;Temperature=$c.Temperature;` +
		`ReadErrorsTotal=$c.ReadErrorsTotal;WriteErrorsTotal=$c.WriteErrorsTotal} } | ConvertTo-Json -Compress -Depth 3`
	mechanism := "powershell Get-PhysicalDisk"

	items, err := runWindowsJSON(ctx, b.env, shell, script)
	if err != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "block-storage", err.Error(), mechanism))
		return
	}

	reliability := map[string]map[string]any{}
	reliabilityItems, reliabilityErr := runWindowsJSON(ctx, b.env, shell, reliabilityScript)
	for _, item := range reliabilityItems {
		if id := jsonString(item, "DeviceId"); id != "" {
			reliability[id] = item
		}
	}

	disks := 0
	for _, item := range items {
		deviceID := jsonString(item, "DeviceId")
		if deviceID == "" {
			continue
		}
		device := Device{
			ID:     "block:physicaldisk" + deviceID,
			Class:  ClassBlockDevice,
			Vendor: jsonString(item, "Manufacturer"),
			Model:  firstNonEmpty(jsonString(item, "Model"), jsonString(item, "FriendlyName")),
			Rungs:  map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", "PhysicalDisk"+deviceID)
		setAttribute(&device, "transport", strings.ToLower(jsonString(item, "BusType")))
		setAttribute(&device, "media_type", jsonString(item, "MediaType"))
		setAttribute(&device, "health_status", jsonString(item, "HealthStatus"))
		if size, ok := jsonNumber(item, "Size"); ok {
			setReading(&device, "capacity_bytes", size)
		}
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		telemetry := b.grader.unavailable(RungTelemetry,
			"Windows reports per-disk I/O through the performance counter subsystem, which this probe does not sample",
			mechanism)
		device.Rungs[RungTelemetry] = telemetry
		device.Rungs[RungEvidence] = b.grader.evidenceFor(telemetry)
		applyWindowsReliability(b, &device, reliability[deviceID], reliabilityErr)
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
			b.grader.notApplicable(RungControl, "reliability-counter access is graded per device"),
			b.grader.notApplicable(RungAnticipation, "wear and error accrual are graded per device"),
		),
	})
}

// applyWindowsReliability grades control and anticipation from the storage
// reliability counters, which are Windows' equivalent of the SMART attributes
// the Linux backend reads through smartctl.
func applyWindowsReliability(b *builder, device *Device, counters map[string]any, probeErr error) {
	const mechanism = "powershell Get-StorageReliabilityCounter"
	setAttribute(device, "smart_mechanism", mechanism)
	if counters == nil {
		reason := "Get-StorageReliabilityCounter returned no counters for this disk; many consumer controllers do not implement the interface"
		if probeErr != nil {
			reason = "Get-StorageReliabilityCounter failed: " + probeErr.Error()
		}
		device.Rungs[RungControl] = remediated(
			b.grader.unmeasurable(RungControl, reason, mechanism),
			"the storage driver must implement the reliability-counter interface; no commissioning step can add it")
		device.Rungs[RungAnticipation] = b.grader.unmeasurable(RungAnticipation,
			"no wear or error-accrual signal: "+reason, mechanism)
		return
	}

	recorded := 0
	for key, reading := range map[string]string{
		"Wear":             "smart_wear_percent_used",
		"PowerOnHours":     "smart_power_on_hours",
		"Temperature":      "smart_temperature_celsius",
		"ReadErrorsTotal":  "smart_read_errors_total",
		"WriteErrorsTotal": "smart_write_errors_total",
	} {
		value, ok := jsonNumber(counters, key)
		if !ok {
			continue
		}
		setReading(device, reading, value)
		recorded++
	}
	if recorded == 0 {
		const reason = "the storage driver returned an empty reliability-counter set for this disk"
		device.Rungs[RungControl] = b.grader.unmeasurable(RungControl, reason, mechanism)
		device.Rungs[RungAnticipation] = b.grader.unmeasurable(RungAnticipation,
			"no wear or error-accrual signal: "+reason, mechanism)
		return
	}
	device.Rungs[RungControl] = b.grader.measured(RungControl, mechanism)
	device.Rungs[RungAnticipation] = b.grader.measured(RungAnticipation, mechanism)
}

// collectWindowsNetwork separates hardware from virtual adapters using the
// adapter's own HardwareInterface and Virtual properties, which is Windows'
// structural statement about what is real.
func collectWindowsNetwork(ctx context.Context, b *builder, shell string) {
	const script = `Get-NetAdapter | Select-Object Name,ifIndex,InterfaceDescription,DriverName,Status,LinkSpeed,Speed,Virtual,HardwareInterface,MacAddress | ConvertTo-Json -Compress -Depth 3`
	const statsScript = `Get-NetAdapterStatistics | Select-Object Name,ReceivedBytes,SentBytes,ReceivedPackets,SentPackets,ReceivedDiscardedPackets,OutboundDiscardedPackets,ReceivedPacketErrors,OutboundPacketErrors | ConvertTo-Json -Compress -Depth 3`
	mechanism := "powershell Get-NetAdapter"

	items, err := runWindowsJSON(ctx, b.env, shell, script)
	if err != nil {
		b.graph.addSubsystem(unavailableSubsystem(b, "network-interfaces", err.Error(), mechanism))
		return
	}
	stats := map[string]map[string]any{}
	statItems, statsErr := runWindowsJSON(ctx, b.env, shell, statsScript)
	for _, item := range statItems {
		if name := jsonString(item, "Name"); name != "" {
			stats[name] = item
		}
	}

	physical := 0
	for _, item := range items {
		name := jsonString(item, "Name")
		if name == "" {
			continue
		}
		if isTrue(item["Virtual"]) || !isTrue(item["HardwareInterface"]) {
			b.graph.VirtualNetworkInterfaces = append(b.graph.VirtualNetworkInterfaces, name)
			continue
		}
		device := Device{
			ID:     "net:" + sanitizeIdentity(name),
			Class:  ClassNetworkInterface,
			Model:  jsonString(item, "InterfaceDescription"),
			Driver: jsonString(item, "DriverName"),
			Rungs:  map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", name)
		setAttribute(&device, "link_state", jsonString(item, "Status"))
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		device.Rungs[RungTelemetry] = windowsInterfaceTelemetry(b, &device, stats[name], statsErr)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		if speed, ok := jsonNumber(item, "Speed"); ok && speed > 0 {
			setReading(&device, "link_speed_mbps", speed/1_000_000)
			device.Rungs[RungControl] = b.grader.measured(RungControl, mechanism)
		} else {
			device.Rungs[RungControl] = b.grader.unmeasurable(RungControl,
				"the adapter reported no negotiated link speed, which Windows does while the link is down", mechanism)
		}
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "interface error")
		b.add(device)
		physical++
	}
	b.graph.addSubsystem(Subsystem{
		Name: "network-interfaces",
		Attributes: map[string]string{
			"physical_interfaces": strconv.Itoa(physical),
			"virtual_interfaces":  strconv.Itoa(len(b.graph.VirtualNetworkInterfaces)),
		},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, mechanism),
			b.grader.notApplicable(RungTelemetry, "counters are graded on each interface"),
			b.grader.notApplicable(RungEvidence, "counters are retained per interface"),
			b.grader.notApplicable(RungControl, "link configuration is graded per interface"),
			b.grader.notApplicable(RungAnticipation, "error accrual is graded per interface"),
		),
	})
}

func windowsInterfaceTelemetry(b *builder, device *Device, counters map[string]any, probeErr error) RungState {
	const mechanism = "powershell Get-NetAdapterStatistics"
	if counters == nil {
		reason := "Get-NetAdapterStatistics returned no counters for this adapter"
		if probeErr != nil {
			reason = "Get-NetAdapterStatistics failed: " + probeErr.Error()
		}
		return b.grader.unmeasurable(RungTelemetry, reason, mechanism)
	}
	mapping := map[string]string{
		"ReceivedBytes":            "rx_bytes",
		"SentBytes":                "tx_bytes",
		"ReceivedPackets":          "rx_packets",
		"SentPackets":              "tx_packets",
		"ReceivedPacketErrors":     "rx_errors",
		"OutboundPacketErrors":     "tx_errors",
		"ReceivedDiscardedPackets": "rx_dropped",
		"OutboundDiscardedPackets": "tx_dropped",
	}
	for source, key := range mapping {
		value, ok := jsonNumber(counters, source)
		if !ok {
			continue
		}
		setReading(device, key, value)
	}
	errorTotal := 0.0
	found := 0
	for _, key := range interfaceErrorCounters {
		value, ok := device.Readings[key]
		if !ok {
			continue
		}
		errorTotal += value
		found++
	}
	if found == 0 {
		return b.grader.unmeasurable(RungTelemetry,
			"the adapter exposes no error or discard counter, so failed traffic is not observable", mechanism)
	}
	setReading(device, readingInterfaceErrors, errorTotal)
	return b.grader.measured(RungTelemetry, mechanism)
}

func collectWindowsThermal(ctx context.Context, b *builder, shell string) {
	const script = `Get-CimInstance -Namespace root/wmi -ClassName MSAcpi_ThermalZoneTemperature -ErrorAction Stop | ` +
		`Select-Object InstanceName,CurrentTemperature,CriticalTripPoint | ConvertTo-Json -Compress -Depth 3`
	mechanism := "powershell MSAcpi_ThermalZoneTemperature"

	items, err := runWindowsJSON(ctx, b.env, shell, script)
	if err != nil {
		reason := "no ACPI thermal zone is exposed through WMI on this host: " + err.Error()
		b.graph.addSubsystem(Subsystem{
			Name: "thermal",
			Rungs: rungSet(
				b.grader.unmeasurable(RungIdentity, reason, mechanism),
				b.grader.unmeasurable(RungTelemetry, reason, mechanism),
				b.grader.unmeasurable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
				remediated(b.grader.unmeasurable(RungControl, reason, mechanism),
					"ACPI thermal-zone reporting is a firmware feature; where the firmware omits it, no commissioning step can add it"),
				b.grader.unmeasurable(RungAnticipation, "no temperature trend: "+reason, trendMechanism),
			),
		})
		return
	}

	used := map[string]int{}
	zones := 0
	for _, item := range items {
		instance := jsonString(item, "InstanceName")
		if instance == "" {
			continue
		}
		device := Device{
			ID:    uniqueSensorID(used, "acpi-thermal-zone", instance, instance),
			Class: ClassThermalSensor,
			Model: instance,
			Rungs: map[Rung]RungState{},
		}
		setAttribute(&device, "sensor_name", instance)
		setAttribute(&device, "sensor_kind", "thermal-zone")
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		if deciKelvin, ok := jsonNumber(item, "CurrentTemperature"); ok && deciKelvin > 0 {
			setReading(&device, readingTemperature, deciKelvinToCelsius(deciKelvin))
			if trip, ok := jsonNumber(item, "CriticalTripPoint"); ok && trip > 0 {
				setReading(&device, readingSetpointCritical, deciKelvinToCelsius(trip))
			}
			device.Rungs[RungTelemetry] = b.grader.measured(RungTelemetry, mechanism)
		} else {
			device.Rungs[RungTelemetry] = b.grader.unmeasurable(RungTelemetry,
				"the thermal zone reported no current temperature", mechanism)
		}
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.notApplicable(RungControl,
			"ACPI thermal zones are readable without elevation; cooling actuation is owned by the platform")
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "temperature")
		b.add(device)
		zones++
	}
	b.graph.addSubsystem(Subsystem{
		Name:       "thermal",
		Attributes: map[string]string{"thermal_zones": strconv.Itoa(zones)},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, mechanism),
			b.grader.notApplicable(RungTelemetry, "temperatures are graded on each zone"),
			b.grader.notApplicable(RungEvidence, "temperatures are retained per zone"),
			b.grader.notApplicable(RungControl, "ACPI thermal zones are readable without elevation"),
			b.grader.notApplicable(RungAnticipation, "temperature trend is graded per zone"),
		),
	})
}

func windowsMemoryErrors(b *builder) {
	const reason = "Windows exposes corrected-memory-error events only as WHEA records in the event log, " +
		"not as running per-controller counters; this probe reads no counter, so ECC accrual is not observable here"
	b.graph.addSubsystem(Subsystem{
		Name:       SubsystemMemoryErrors,
		Attributes: map[string]string{"registered_controllers": "0"},
		Rungs: rungSet(
			b.grader.unmeasurable(RungIdentity, reason, "WHEA event log"),
			b.grader.unmeasurable(RungTelemetry, reason, "WHEA event log"),
			b.grader.unmeasurable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
			remediated(b.grader.unmeasurable(RungControl, reason, "WHEA event log"),
				"a WHEA event-log reader would have to be declared and commissioned before ECC accrual becomes observable on Windows"),
			b.grader.unmeasurable(RungAnticipation, "no memory-error trend: "+reason, trendMechanism),
		),
	})
}

func deciKelvinToCelsius(deciKelvin float64) float64 {
	return deciKelvin/windowsDeciKelvinPerKelvin - kelvinOffset
}

func isTrue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	default:
		return false
	}
}
