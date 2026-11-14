package hostreqkit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

const OpenRGBLinuxDeviceHIDRaw = "hidraw"

var ErrOpenRGBLiveDeviceAccessUnverified = errors.New("OpenRGB requires a device-access witness from the exact launch context")

// OpenRGBLinuxPolicy is an enrollment from the host permission owner. The
// owner must install and verify a static udev recipe granting only Devices to
// GroupID before supplying the recipe digest. A runtime caller cannot supply
// this enrollment as its policy request.
type OpenRGBLinuxPolicy struct {
	MachineID      string
	ArtifactSHA256 string
	DetectorSHA256 string
	RuleSHA256     string
	GroupID        uint32
	OwnerUID       uint32 `json:"OwnerUID,omitempty"`
	Devices        []OpenRGBLinuxDevice
}

// OpenRGBLinuxDevice identifies a physical attachment independently of the
// character-device number. Topology is relative to /sys/devices. I2C selects
// the PCI function and the adapter's exact name. HID selects the USB interface.
// HardwareID is vendor:device:subsystem_vendor:subsystem_device for I2C and
// vendor:product:interface for HID, using lowercase hex without 0x prefixes.
type OpenRGBLinuxDevice struct {
	// Detector is the exact pinned OpenRGB static detector enabled for this
	// physical selector. It participates in Policy.ID so a detector cannot be
	// substituted while retaining the same device permission grant.
	Detector   string
	Kind       string
	Topology   string
	HardwareID string
	Adapter    string
}

type OpenRGBLinuxCheck struct {
	PolicyID       string
	ArtifactSHA256 string
	DetectorSHA256 string
	EffectiveUID   int
	PID            int
	Incarnation    string
}

// OpenRGBLinuxEvidence distinguishes installed static grant inputs and present
// access from live process evidence. It does not certify a logged-out boot.
type OpenRGBLinuxEvidence struct {
	DeviceGenerationSHA256 string
	StaticGrantConfigured  bool
	SelectedNodes          []string
	CurrentScopeVerified   bool
	LiveCredentialsMatch   bool
}

// ID binds the exact local enrollment, independently of selector order.
func (p OpenRGBLinuxPolicy) ID() (string, error) {
	if !openRGBHex(p.MachineID, 32) || !openRGBHex(p.ArtifactSHA256, 64) || !openRGBHex(p.DetectorSHA256, 64) || !openRGBHex(p.RuleSHA256, 64) || (p.GroupID == 0) == (p.OwnerUID == 0) || len(p.Devices) == 0 || len(p.Devices) > 16 {
		return "", fmt.Errorf("incomplete OpenRGB Linux device policy")
	}
	p.Devices = slices.Clone(p.Devices)
	key := func(d OpenRGBLinuxDevice) string {
		return strings.Join([]string{d.Detector, d.Kind, d.Topology, d.HardwareID, d.Adapter}, "\x00")
	}
	slices.SortFunc(p.Devices, func(a, b OpenRGBLinuxDevice) int { return strings.Compare(key(a), key(b)) })
	for i, d := range p.Devices {
		if strings.TrimSpace(d.Detector) == "" || strings.ContainsAny(d.Detector, "\r\n\x00") || !validOpenRGBTopology(d.Topology) || strings.ContainsAny(d.Adapter, "\r\n\x00") || (i > 0 && d == p.Devices[i-1]) {
			return "", fmt.Errorf("invalid or duplicate OpenRGB device selector")
		}
		parts := strings.Split(d.HardwareID, ":")
		switch d.Kind {
		case "i2c":
			if d.Adapter == "" || len(parts) != 4 || slices.ContainsFunc(parts, func(v string) bool { return !openRGBHex(v, 4) }) {
				return "", fmt.Errorf("invalid OpenRGB PCI adapter selector")
			}
		case OpenRGBLinuxDeviceHIDRaw:
			if d.Adapter != "" || len(parts) != 3 || !openRGBHex(parts[0], 4) || !openRGBHex(parts[1], 4) || !openRGBHex(parts[2], 2) {
				return "", fmt.Errorf("invalid OpenRGB USB HID interface selector")
			}
		default:
			return "", fmt.Errorf("unknown OpenRGB device kind %q", d.Kind)
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func validOpenRGBTopology(path string) bool {
	return path != "" && path != "." && filepath.IsLocal(path) && filepath.Clean(path) == path && !strings.ContainsAny(path, "\\*?[]\r\n\x00")
}

func openRGBHex(s string, length int) bool {
	_, err := hex.DecodeString(s)
	return err == nil && len(s) == length && s == strings.ToLower(s)
}
