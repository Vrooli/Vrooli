package devicegraph

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// hwmon reports temperatures in millidegrees Celsius.
const milliDegreesPerDegree = 1000.0

var (
	hwmonTempInput = regexp.MustCompile(`^temp([0-9]+)_input$`)
	tripPointTemp  = regexp.MustCompile(`^trip_point_([0-9]+)_temp$`)
)

// collectThermalSensors enumerates every hwmon sensor and every thermal zone.
// A sensor that exposes no readable temperature reports unmeasurable with the
// reason; it is never reported as zero degrees, which would read as a very
// cold, very healthy part.
func collectThermalSensors(b *builder) {
	hwmonCount := b.collectHwmonSensors()
	zoneCount := b.collectThermalZones()

	b.graph.addSubsystem(Subsystem{
		Name: "thermal",
		Attributes: map[string]string{
			"hwmon_sensors": strconv.Itoa(hwmonCount),
			"thermal_zones": strconv.Itoa(zoneCount),
		},
		Rungs: rungSet(
			thermalSubsystemIdentity(b, hwmonCount, zoneCount),
			b.grader.notApplicable(RungTelemetry, "temperatures are graded on each sensor"),
			b.grader.notApplicable(RungEvidence, "temperatures are retained per sensor"),
			b.grader.notApplicable(RungControl, "reading hwmon and thermal-zone temperatures needs no privileged control path"),
			b.grader.notApplicable(RungAnticipation, "temperature trend is graded per sensor"),
		),
	})
}

func thermalSubsystemIdentity(b *builder, hwmonCount, zoneCount int) RungState {
	if hwmonCount+zoneCount > 0 {
		return b.grader.measured(RungIdentity, "sysfs /class/hwmon and /class/thermal")
	}
	return b.grader.unavailable(RungIdentity,
		"this host exposes neither a hwmon sensor nor a thermal zone",
		"sysfs /class/hwmon and /class/thermal")
}

func (b *builder) collectHwmonSensors() int {
	root := b.env.sys("class", "hwmon")
	names, ok := b.env.listDir(root)
	if !ok {
		return 0
	}
	used := map[string]int{}
	count := 0
	for _, hwmonName := range names {
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, hwmonName))
		if !resolvedOK {
			b.graph.warn("hwmon sensor %s could not be resolved to a sysfs path", hwmonName)
			continue
		}
		sensorName, hasName := b.env.readText(filepath.Join(resolved, "name"))
		if !hasName {
			sensorName = hwmonName
		}

		device := Device{
			Class:   ClassThermalSensor,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}
		setAttribute(&device, "sensor_name", sensorName)
		setAttribute(&device, "kernel_name", hwmonName)

		ownerPath := resolved
		if owner, ok := b.env.resolve(filepath.Join(resolved, "device")); ok {
			ownerPath = owner
			setAttribute(&device, "measures_sys_path", owner)
			device.ParentID = b.ownerOf(owner, false)
		}
		device.ID = uniqueSensorID(used, sensorName, ownerPath, hwmonName)
		if driver, ok := b.env.linkBase(filepath.Join(ownerPath, "driver")); ok {
			device.Driver = driver
		}
		device.Model = sensorName

		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, "sysfs /class/hwmon")
		device.Rungs[RungTelemetry] = b.readHwmonTemperatures(resolved, &device)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.notApplicable(RungControl,
			"hwmon temperatures are readable without privilege; this sensor exposes no control surface to the monitor")
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "temperature")

		b.add(device)
		count++
	}
	return count
}

// readHwmonTemperatures reads every tempN_input the sensor exposes, along with
// the sensor's own declared max and critical setpoints. Reading the setpoints
// from the hardware is what lets the trend be banded without hard-coding a
// threshold that would be wrong on different silicon.
func (b *builder) readHwmonTemperatures(resolved string, device *Device) RungState {
	const mechanism = "sysfs /class/hwmon/*/temp*_input"
	entries, ok := b.env.listDir(resolved)
	if !ok {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s is not readable", resolved), mechanism)
	}
	indexes := make([]int, 0, 4)
	for _, entry := range entries {
		match := hwmonTempInput.FindStringSubmatch(entry)
		if match == nil {
			continue
		}
		index, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		indexes = append(indexes, index)
	}
	if len(indexes) == 0 {
		return b.grader.unmeasurable(RungTelemetry,
			"this hwmon sensor exposes no temp*_input, so it reports no temperature", mechanism)
	}
	sort.Ints(indexes)

	readCount := 0
	for _, index := range indexes {
		prefix := fmt.Sprintf("temp%d", index)
		milli, ok := b.env.readInt(filepath.Join(resolved, prefix+"_input"))
		if !ok {
			continue
		}
		celsius := float64(milli) / milliDegreesPerDegree
		key := prefix + "_celsius"
		setReading(device, key, celsius)
		if label, ok := b.env.readText(filepath.Join(resolved, prefix+"_label")); ok {
			setAttribute(device, prefix+"_label", label)
		}
		if readCount == 0 {
			setReading(device, readingTemperature, celsius)
			b.recordSetpoints(resolved, prefix, device)
		}
		readCount++
	}
	if readCount == 0 {
		return b.grader.unmeasurable(RungTelemetry,
			"this hwmon sensor declares a temp*_input that could not be read", mechanism)
	}
	setAttribute(device, "temperature_input_count", strconv.Itoa(readCount))
	return b.grader.measured(RungTelemetry, mechanism)
}

// recordSetpoints copies the sensor's own limits into the device so the trend
// band compares against the hardware's declared bar, not an invented one.
func (b *builder) recordSetpoints(resolved, prefix string, device *Device) {
	if maximum, ok := b.env.readInt(filepath.Join(resolved, prefix+"_max")); ok {
		setReading(device, readingSetpointMax, float64(maximum)/milliDegreesPerDegree)
	}
	if critical, ok := b.env.readInt(filepath.Join(resolved, prefix+"_crit")); ok {
		setReading(device, readingSetpointCritical, float64(critical)/milliDegreesPerDegree)
	}
}

func (b *builder) collectThermalZones() int {
	root := b.env.sys("class", "thermal")
	names, ok := b.env.listDir(root)
	if !ok {
		return 0
	}
	used := map[string]int{}
	count := 0
	for _, zoneName := range names {
		// The thermal class also contains cooling_device* nodes, which are
		// actuators, not sensors. A zone is identified by exposing a "temp"
		// reading and a "type"; cooling devices expose neither.
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, zoneName))
		if !resolvedOK {
			continue
		}
		zoneType, hasType := b.env.readText(filepath.Join(resolved, "type"))
		_, hasTemp := b.env.readText(filepath.Join(resolved, "temp"))
		if !hasType || !hasTemp {
			continue
		}

		device := Device{
			Class:   ClassThermalSensor,
			SysPath: resolved,
			Model:   zoneType,
			Rungs:   map[Rung]RungState{},
		}
		setAttribute(&device, "sensor_name", zoneType)
		setAttribute(&device, "kernel_name", zoneName)
		setAttribute(&device, "sensor_kind", "thermal-zone")
		ownerPath := resolved
		if owner, ok := b.env.resolve(filepath.Join(resolved, "device")); ok {
			ownerPath = owner
			setAttribute(&device, "measures_sys_path", owner)
			device.ParentID = b.ownerOf(owner, false)
		}
		device.ID = uniqueSensorID(used, zoneType, ownerPath, zoneName)

		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, "sysfs /class/thermal")
		device.Rungs[RungTelemetry] = b.readThermalZone(resolved, &device)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.notApplicable(RungControl,
			"thermal-zone temperatures are readable without privilege; cooling actuation is owned by the kernel governor")
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "temperature")

		b.add(device)
		count++
	}
	return count
}

func (b *builder) readThermalZone(resolved string, device *Device) RungState {
	const mechanism = "sysfs /class/thermal/thermal_zone*/temp"
	milli, ok := b.env.readInt(filepath.Join(resolved, "temp"))
	if !ok {
		return b.grader.unmeasurable(RungTelemetry,
			"this thermal zone declares a temp file that could not be read", mechanism)
	}
	setReading(device, readingTemperature, float64(milli)/milliDegreesPerDegree)
	b.recordTripPoints(resolved, device)
	return b.grader.measured(RungTelemetry, mechanism)
}

// recordTripPoints reads the zone's declared trip points and uses the hottest
// non-critical one as the elevated bar and the critical one as the ceiling.
func (b *builder) recordTripPoints(resolved string, device *Device) {
	entries, ok := b.env.listDir(resolved)
	if !ok {
		return
	}
	for _, entry := range entries {
		match := tripPointTemp.FindStringSubmatch(entry)
		if match == nil {
			continue
		}
		milli, ok := b.env.readInt(filepath.Join(resolved, entry))
		if !ok {
			continue
		}
		tripType, _ := b.env.readText(filepath.Join(resolved, fmt.Sprintf("trip_point_%s_type", match[1])))
		celsius := float64(milli) / milliDegreesPerDegree
		switch strings.ToLower(strings.TrimSpace(tripType)) {
		case "critical", "hot":
			if current, exists := device.Readings[readingSetpointCritical]; !exists || celsius < current {
				setReading(device, readingSetpointCritical, celsius)
			}
		case "passive", "active":
			if current, exists := device.Readings[readingSetpointMax]; !exists || celsius < current {
				setReading(device, readingSetpointMax, celsius)
			}
		}
	}
}

// uniqueSensorID builds a stable sensor identity from the sensor's own name and
// the device it is attached to. Two sensors of the same kind (two memory
// temperature sensors, say) differ by their owner, so the identity stays unique
// without depending on the kernel's enumeration order.
func uniqueSensorID(used map[string]int, sensorName, ownerPath, kernelName string) string {
	owner := filepath.Base(ownerPath)
	if owner == "" || owner == "." || owner == "/" {
		owner = kernelName
	}
	base := "sensor:" + sensorName + "@" + owner
	count := used[base]
	used[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s#%d", base, count)
}
