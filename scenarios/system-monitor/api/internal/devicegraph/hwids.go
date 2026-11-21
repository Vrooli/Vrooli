package devicegraph

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// hardwareIDs resolves numeric vendor/device ids to human names using the
// system PCI/USB id databases. The databases are optional: when none is
// present the graph degrades to raw ids rather than inventing a name.
type hardwareIDs struct {
	vendors map[string]string
	devices map[string]string
	source  string
}

// loadHardwareIDs parses the first readable database named `file` (pci.ids or
// usb.ids) from the configured search paths.
func loadHardwareIDs(env Env, file string) hardwareIDs {
	for _, dir := range env.HardwareIDPaths {
		handle, err := os.Open(filepath.Join(dir, file))
		if err != nil {
			continue
		}
		ids := parseHardwareIDs(handle)
		handle.Close()
		ids.source = filepath.Join(dir, file)
		return ids
	}
	return hardwareIDs{}
}

// parseHardwareIDs reads the hwdata format: a vendor line at column zero, its
// devices indented one tab, subsystems indented two. Lines below the numeric
// section (device classes) start with a letter and are ignored.
func parseHardwareIDs(reader interface{ Read([]byte) (int, error) }) hardwareIDs {
	ids := hardwareIDs{vendors: map[string]string{}, devices: map[string]string{}}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	vendor := ""
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "\t\t"):
			// Subsystem entry; the graph does not model subsystem ids.
			continue
		case strings.HasPrefix(line, "\t"):
			if vendor == "" {
				continue
			}
			id, name, ok := splitIDLine(strings.TrimPrefix(line, "\t"))
			if !ok {
				continue
			}
			ids.devices[vendor+":"+id] = name
		default:
			id, name, ok := splitIDLine(line)
			if !ok {
				// The class/subclass section that follows the vendor list is
				// keyed by letters, not hex; stop trusting the vendor cursor.
				vendor = ""
				continue
			}
			vendor = id
			ids.vendors[id] = name
		}
	}
	return ids
}

func splitIDLine(line string) (id, name string, ok bool) {
	fields := strings.SplitN(line, "  ", 2)
	if len(fields) != 2 {
		return "", "", false
	}
	id = strings.ToLower(strings.TrimSpace(fields[0]))
	name = strings.TrimSpace(fields[1])
	if id == "" || name == "" || len(id) != 4 {
		return "", "", false
	}
	for _, char := range id {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return "", "", false
		}
	}
	return id, name, true
}

func (h hardwareIDs) vendorName(vendorID string) string {
	return h.vendors[normalizeHexID(vendorID)]
}

func (h hardwareIDs) deviceName(vendorID, deviceID string) string {
	return h.devices[normalizeHexID(vendorID)+":"+normalizeHexID(deviceID)]
}

func (h hardwareIDs) present() bool { return len(h.vendors) > 0 }

// normalizeHexID accepts both sysfs form ("0x10de") and database form ("10de").
func normalizeHexID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "0x")
}
