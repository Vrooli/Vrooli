package hostinventory

import (
	"regexp"
	"strings"
)

var pciLineRE = regexp.MustCompile(`^([0-9a-fA-F:.]+)\s+([^:]+):\s+(.+?)(?:\s+\[([0-9a-fA-F]{4}):([0-9a-fA-F]{4})\])?$`)

func ParseLSPCI(output string) []DeviceInfo {
	var devices []DeviceInfo
	var current *DeviceInfo
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
			match := pciLineRE.FindStringSubmatch(line)
			device := DeviceInfo{BusType: "pci"}
			if len(match) > 0 {
				device.Address = match[1]
				device.Class = strings.TrimSpace(match[2])
				device.DeviceName = strings.TrimSpace(match[3])
				device.VendorID = match[4]
				device.DeviceID = match[5]
			} else {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					device.Address = fields[0]
				}
				device.DeviceName = strings.TrimSpace(strings.TrimPrefix(line, device.Address))
			}
			devices = append(devices, device)
			current = &devices[len(devices)-1]
			continue
		}
		if current == nil {
			continue
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Kernel driver in use:"):
			current.BoundDriver = strings.TrimSpace(strings.TrimPrefix(trimmed, "Kernel driver in use:"))
		case strings.HasPrefix(trimmed, "Kernel modules:"):
			modules := strings.TrimSpace(strings.TrimPrefix(trimmed, "Kernel modules:"))
			for _, module := range strings.Split(modules, ",") {
				module = strings.TrimSpace(module)
				if module != "" {
					current.AvailableModules = append(current.AvailableModules, module)
				}
			}
		}
	}
	return devices
}
