package hostinventory

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	parsersParameterA = 10
	parsersParameterB = 1024
	parsersParameterC = 11
	parsersParameterD = 12
	parsersParameterE = 13
	parsersParameterF = 14
	parsersParameterG = 2
	parsersParameterH = 3
	parsersParameterI = 4
	parsersParameterJ = 5
	parsersParameterK = 6
	parsersParameterL = 7
	parsersParameterM = 8
	parsersParameterN = 9
)

func ParseLinuxMeminfo(input string) (Memory, Swap, error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(input, "\n") {
		fields := strings.Fields(line)
		if len(fields) < parsersParameterG {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return Memory{}, Swap{}, fmt.Errorf("parse /proc/meminfo %s: %w", key, err)
		}
		multiplier := uint64(1)
		if len(fields) >= 3 && strings.EqualFold(fields[2], "kB") {
			multiplier = 1024
		}
		values[key] = value * multiplier
	}
	total := values["MemTotal"]
	if total == 0 {
		return Memory{}, Swap{}, fmt.Errorf("parse /proc/meminfo: MemTotal not found")
	}
	return Memory{
			TotalBytes:     total,
			AvailableBytes: values["MemAvailable"],
			BuffersBytes:   values["Buffers"],
			CachedBytes:    values["Cached"],
		}, Swap{
			TotalBytes: values["SwapTotal"],
			FreeBytes:  values["SwapFree"],
		}, nil
}

func ParseLinuxLoadavg(input string, cpuCores int) (Load, error) {
	fields := strings.Fields(input)
	if len(fields) < parsersParameterJ {
		return Load{}, fmt.Errorf("parse /proc/loadavg: unexpected format %q", strings.TrimSpace(input))
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg load1: %w", err)
	}
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg load5: %w", err)
	}
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg load15: %w", err)
	}
	procParts := strings.Split(fields[3], "/")
	if len(procParts) != parsersParameterG {
		return Load{}, fmt.Errorf("parse /proc/loadavg processes: unexpected format %q", fields[3])
	}
	running, err := strconv.Atoi(procParts[0])
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg running processes: %w", err)
	}
	total, err := strconv.Atoi(procParts[1])
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg total processes: %w", err)
	}
	lastPID, err := strconv.Atoi(fields[4])
	if err != nil {
		return Load{}, fmt.Errorf("parse /proc/loadavg last pid: %w", err)
	}
	load := Load{
		Load1:        load1,
		Load5:        load5,
		Load15:       load15,
		RunningProcs: running,
		TotalProcs:   total,
		LastPID:      lastPID,
	}
	if cpuCores > 0 {
		load.NormalizedLoad1 = load.Load1 / float64(cpuCores)
		load.NormalizedLoad5 = load.Load5 / float64(cpuCores)
	}
	return load, nil
}

func ParseNvidiaGPUCSV(input string) []GPU {
	var gpus []GPU
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < parsersParameterH {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(parts[1])
		mb, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
		if err != nil {
			continue
		}
		gpus = append(gpus, GPU{
			Index:     index,
			Name:      name,
			VRAMBytes: mb * 1024 * parsersParameterB,
			Source:    "nvidia-smi",
		})
	}
	return gpus
}

func ParseNvidiaDetailedGPUCSV(input string) ([]GPU, []string, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(input)))
	reader.TrimLeadingSpace = true

	var gpus []GPU
	var warnings []string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, warnings, fmt.Errorf("parse nvidia gpu csv: %w", err)
		}
		if len(record) < parsersParameterF {
			warnings = append(warnings, fmt.Sprintf("unexpected nvidia gpu record length: %d", len(record)))
			continue
		}
		gpu, rowWarnings := parseNvidiaDetailedGPURecord(record)
		warnings = append(warnings, rowWarnings...)
		if gpu == nil {
			continue
		}
		gpus = append(gpus, *gpu)
	}
	return gpus, warnings, nil
}

// ParseNvidiaComputeCapabilityCSV parses the separately queried CUDA compute
// capability. Keeping this query separate preserves the detailed GPU parser's
// stable field order while using the nvidia-smi field requested by the
// acquisition fact contract.
func ParseNvidiaComputeCapabilityCSV(input string) map[int]string {
	capabilities := map[int]string{}
	for _, line := range strings.Split(strings.TrimSpace(input), "\n") {
		parts := strings.SplitN(line, ",", parsersParameterG)
		if len(parts) != parsersParameterG {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		capability := strings.TrimSpace(parts[1])
		if capability == "" || strings.EqualFold(capability, "N/A") {
			continue
		}
		if _, err := strconv.ParseFloat(capability, 64); err != nil {
			continue
		}
		capabilities[index] = capability
	}
	return capabilities
}

func ParseNvidiaComputeAppsCSV(input string) ([]GPUProcess, []string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.Contains(trimmed, "No running processes found") {
		return nil, nil, nil
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.TrimLeadingSpace = true

	var processes []GPUProcess
	var warnings []string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return processes, warnings, fmt.Errorf("parse nvidia compute apps csv: %w", err)
		}
		if len(record) < parsersParameterI {
			warnings = append(warnings, fmt.Sprintf("unexpected nvidia compute app record length: %d", len(record)))
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("invalid nvidia compute app pid: %v", err))
			continue
		}
		usedMB, err := parseNvidiaUintMB(strings.TrimSpace(record[2]))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("invalid nvidia compute app memory: %v", err))
			continue
		}
		processes = append(processes, GPUProcess{
			PID:         pid,
			ProcessName: strings.TrimSpace(record[1]),
			UsedBytes:   usedMB * 1024 * parsersParameterB,
			GPUUUID:     strings.TrimSpace(record[3]),
		})
	}
	return processes, warnings, nil
}

func ParseWindowsTotalPhysicalMemory(input string) (uint64, error) {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "TotalPhysicalMemory=") {
			continue
		}
		return ParseUintBytes(strings.TrimPrefix(line, "TotalPhysicalMemory="))
	}
	return 0, fmt.Errorf("parse wmic output: TotalPhysicalMemory not found")
}

func ParseSystemProfilerGPUs(input string) []GPU {
	var gpus []GPU
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), "chipset model:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
		if name == "" {
			continue
		}
		gpus = append(gpus, GPU{
			Index:  len(gpus),
			Name:   name,
			Source: "system_profiler",
		})
	}
	return gpus
}

func ParseWindowsGPUNames(input string) []GPU {
	var gpus []GPU
	for _, line := range strings.Split(input, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.EqualFold(name, "Name") {
			continue
		}
		lower := strings.ToLower(name)
		if !strings.Contains(lower, "nvidia") && !strings.Contains(lower, "amd") && !strings.Contains(lower, "intel") {
			continue
		}
		gpus = append(gpus, GPU{
			Index:  len(gpus),
			Name:   name,
			Source: "wmic",
		})
	}
	return gpus
}

// ParseNvidiaPCIBusIDCSV maps each nvidia-smi GPU index to the normalised PCI
// address of the device it describes. nvidia-smi prints an eight-digit domain
// ("00000000:01:00.0"); sysfs and every other consumer use four
// ("tuning.PermNone:01:00.0"), so the address is normalised here rather than at each
// comparison site.
func ParseNvidiaPCIBusIDCSV(input string) map[int]string {
	addresses := map[int]string{}
	for _, line := range strings.Split(strings.TrimSpace(input), "\n") {
		index, address, found := strings.Cut(line, ",")
		if !found {
			continue
		}
		gpuIndex, err := strconv.Atoi(strings.TrimSpace(index))
		if err != nil {
			continue
		}
		normalized := NormalizePCIAddress(address)
		if normalized == "" {
			continue
		}
		addresses[gpuIndex] = normalized
	}
	return addresses
}

// NormalizePCIAddress renders a PCI address in the four-digit-domain form the
// Linux device tree uses. It returns an empty string when the input is not a
// PCI address.
func NormalizePCIAddress(input string) string {
	address := strings.ToLower(strings.TrimSpace(input))
	parts := strings.Split(address, ":")
	if len(parts) != parsersParameterH {
		return ""
	}
	domain := strings.TrimLeft(parts[0], "0")
	if domain == "" {
		domain = "0"
	}
	if len(domain) > parsersParameterI {
		return ""
	}
	return fmt.Sprintf("%04s:%s:%s", domain, parts[1], parts[2])
}

// ParseWindowsVideoControllers builds device records from the key=value blocks
// wmic emits for win32_VideoController. Identity is the PNP device instance
// ID, which the PnP manager derives from bus topology and which therefore does
// not depend on enumeration order.
func ParseWindowsVideoControllers(input string) []Device {
	var devices []Device
	current := map[string]string{}
	flush := func() {
		defer func() { current = map[string]string{} }()
		id := strings.TrimSpace(current["PNPDeviceID"])
		if id == "" {
			return
		}
		devices = append(devices, Device{
			ID:            "pnp:" + strings.ToUpper(id),
			Class:         DeviceClassGraphics,
			Vendor:        strings.TrimSpace(current["AdapterCompatibility"]),
			Model:         strings.TrimSpace(current["Name"]),
			DriverVersion: strings.TrimSpace(current["DriverVersion"]),
			DiscoveredBy:  "windows-pnp-device-tree",
		})
	}
	for _, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			flush()
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if _, repeated := current[key]; repeated {
			flush()
		}
		current[key] = value
	}
	flush()
	return devices
}

func ParseUintBytes(input string) (uint64, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(input), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func DockerInfoHasNvidiaRuntime(input string) bool {
	lower := strings.ToLower(input)
	return strings.Contains(lower, "nvidia")
}

func parseNvidiaDetailedGPURecord(record []string) (*GPU, []string) {
	var warnings []string
	trim := func(index int) string { return strings.TrimSpace(record[index]) }

	index, err := strconv.Atoi(trim(0))
	if err != nil {
		return nil, []string{fmt.Sprintf("invalid nvidia gpu index: %v", err)}
	}
	totalMB, err := parseNvidiaUintMB(trim(parsersParameterK))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("invalid nvidia gpu memory total: %v", err))
	}
	usedMB, err := parseNvidiaUintMB(trim(parsersParameterL))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("invalid nvidia gpu memory used: %v", err))
	}

	return &GPU{
		Index:                    index,
		Name:                     trim(1),
		UUID:                     trim(parsersParameterG),
		DriverVersion:            trim(parsersParameterH),
		UtilizationPercent:       parseNvidiaFloat(trim(parsersParameterI)),
		MemoryUtilizationPercent: parseNvidiaFloat(trim(parsersParameterJ)),
		VRAMBytes:                totalMB * 1024 * parsersParameterB,
		VRAMUsedBytes:            usedMB * 1024 * parsersParameterB,
		TemperatureC:             parseNvidiaOptionalFloat(trim(parsersParameterM)),
		FanSpeedPercent:          parseNvidiaOptionalFloat(trim(parsersParameterN)),
		PowerDrawW:               parseNvidiaOptionalFloat(trim(parsersParameterA)),
		PowerLimitW:              parseNvidiaOptionalFloat(trim(parsersParameterC)),
		SMClockMHz:               parseNvidiaOptionalFloat(trim(parsersParameterD)),
		MemoryClockMHz:           parseNvidiaOptionalFloat(trim(parsersParameterE)),
		Source:                   "nvidia-smi",
	}, warnings
}

func parseNvidiaUintMB(input string) (uint64, error) {
	if input == "" || strings.EqualFold(input, "N/A") {
		return 0, nil
	}
	return strconv.ParseUint(input, 10, 64)
}

func parseNvidiaFloat(input string) float64 {
	if input == "" || strings.EqualFold(input, "N/A") {
		return 0
	}
	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseNvidiaOptionalFloat(input string) *float64 {
	if input == "" || strings.EqualFold(input, "N/A") {
		return nil
	}
	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return nil
	}
	return &value
}
