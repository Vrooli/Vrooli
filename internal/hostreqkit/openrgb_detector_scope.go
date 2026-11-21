package hostreqkit

import (
	"errors"
	"strings"
)

// This validation input is the exact source-qualified full all-disabled catalog.
const openRGBQualifiedDetectorCatalogSHA256 = "0f83577a12be49d7e351d307bb54f693feb3013f2561bb1885acf8c477c594df"

// Exact release-profile linkage: resource acquisition independently verifies
// the envelope or adopted whole closure before permission enrollment.
const (
	openRGBLinuxAMD64EnvelopeSHA256 = "d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d"
	openRGBLinuxAMD64PayloadSHA256  = "48739d378cfcd27e8b8c8cfc6d820364dddf31f0a9819065be1a281a4ee98dab"
	openRGBLinuxARM64EnvelopeSHA256 = "0fd6806c265f9dbd9b0a8847cf4b4cb172c86baf2e39f52f869f8c5325c6751b"
	openRGBLinuxARM64PayloadSHA256  = "d947ed41462a6cab99e8689b4ac45ebf32383131a8933ed6a8d3584748913ced"
)

type OpenRGBLinuxNode struct {
	Selector   OpenRGBLinuxDevice
	Node       string
	SysPath    string
	MajorMinor string
}

func ValidateOpenRGBLinuxDetectorScope(policy OpenRGBLinuxPolicy) error {
	return validateOpenRGBDetectorScope(policy)
}

func openRGBKnownDetector(device OpenRGBLinuxDevice) string {
	for _, detector := range []string{"ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC", "ASUS Aura Motherboard"} {
		if openRGBDetectorMatches(detector, device) {
			return detector
		}
	}
	return ""
}

// These footprints come from exact OpenRGB release_1.1 commit 8e16de27858cf2be6af54a33ce003df74cb9fd3e:
// DetectionManager compares all four PCI IDs before the ENE GPU callback and
// VID/PID/usage/interface before the ASUS motherboard HID callback. Generic
// SMBus motherboard/DRAM and Super-IO callbacks stay disabled in the resource.
func openRGBDetectorMatches(detector string, device OpenRGBLinuxDevice) bool {
	parts := strings.Split(device.HardwareID, ":")
	switch detector {
	case "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC":
		return device.Kind == "i2c" && (device.HardwareID == "10de:2705:1043:8957" || device.HardwareID == "10de:2705:1043:8958")
	case "ASUS Aura Motherboard":
		if device.Kind != "hidraw" || len(parts) != 3 || parts[0] != "0b05" {
			return false
		}
		switch parts[1] {
		case "18f3", "1939", "19af", "1aa6", "1bed":
			// For 19af the callback further requires usage ff72/a1. Treating
			// every interface as potentially matching is conservative and keeps
			// enrollment independent of device descriptor I/O.
			return true
		}
	}
	return false
}

func validateOpenRGBDetectorScope(policy OpenRGBLinuxPolicy) error {
	armHIDOnly := false
	switch policy.ArtifactSHA256 {
	case openRGBLinuxAMD64EnvelopeSHA256, openRGBLinuxAMD64PayloadSHA256:
	case openRGBLinuxARM64EnvelopeSHA256, openRGBLinuxARM64PayloadSHA256:
		armHIDOnly = true
	default:
		return errors.New("OpenRGB detector transaction scope is unqualified for this artifact")
	}
	if policy.DetectorSHA256 != openRGBQualifiedDetectorCatalogSHA256 {
		return errors.New("OpenRGB detector transaction scope is unqualified for this detector catalog")
	}
	for _, device := range policy.Devices {
		if armHIDOnly && device.Kind != OpenRGBLinuxDeviceHIDRaw {
			return errors.New("Linux ARM OpenRGB I2C transaction scope is unqualified; source-qualified motherboard HID remains available")
		}
		if !openRGBDetectorMatches(device.Detector, device) {
			return errors.New("selected OpenRGB detector and physical identity have no qualified transaction footprint")
		}
	}
	return nil
}
