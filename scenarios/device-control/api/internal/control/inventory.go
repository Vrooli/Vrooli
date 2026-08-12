package control

import (
	"context"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
)

func (s *Service) Strategies(ctx context.Context) []strategy.Declaration { return s.registry.List(ctx) }
func (s *Service) Verify(ctx context.Context, id string) (strategy.ConformanceReport, error) {
	return s.registry.Verify(ctx, id)
}

func (s *Service) Devices(ctx context.Context) []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	declarations := s.registry.List(ctx)
	declarationByID := make(map[string]strategy.Declaration, len(declarations))
	for _, declaration := range declarations {
		declarationByID[declaration.StrategyID] = declaration
	}
	hostNodeID := strings.TrimSpace(os.Getenv("VROOLI_NODE_ID"))
	if hostNodeID == "" {
		hostNodeID, _ = os.Hostname()
	}
	seen := map[string]bool{}
	for _, d := range declarations {
		enumerating := false
		if item, ok := s.registry.Get(d.StrategyID); ok {
			if enumerator, ok := item.(strategy.Enumerator); ok {
				enumerating = true
				if discovered, err := enumerator.Enumerate(ctx); err == nil {
					for _, discoveredDevice := range discovered {
						item := devicedomain.Record{ID: discoveredDevice.ID, Name: discoveredDevice.Model, Kind: "physical", Serial: discoveredDevice.Serial, Model: discoveredDevice.Model, OSVersion: discoveredDevice.OSVersion, StrategyID: discoveredDevice.StrategyID, Status: discoveredDevice.Health, Health: discoveredDevice.Health, HealthReason: discoveredDevice.HealthReason, HostNodeID: hostNodeID, Transport: discoveredDevice.Transport, ObservedAt: discoveredDevice.ObservedAt, Capabilities: mapCaps(d)}
						if item.Name == "" {
							item.Name = item.Serial
						}
						s.devices.Upsert(item)
						seen[item.ID] = true
					}
				}
			}
		}
		if enumerating {
			continue
		}
		// Strategies are exposed by /strategies. The device inventory contains
		// physical identities (and bridge-attached identities) only; a strategy
		// row is not a device and must not be presented as one.
		_ = d
	}
	if len(seen) == 0 {
		seen = map[string]bool{}
	}
	s.devices.MarkAbsentExcept(time.Now().UTC(), seen, func(item devicedomain.Record) string {
		if item.HostNodeID != "" {
			return "device not present on host node " + item.HostNodeID
		}
		return "device not present on host node local"
	})
	out := make([]Device, 0)
	for _, item := range s.devices.List() {
		if seen[item.ID] || item.Kind == "physical" {
			out = append(out, deviceFromRecord(item))
		}
	}
	if s.attached != nil {
		attached, err := s.attached.List(ctx)
		if err != nil {
			// A failed bridge lookup must not manufacture a pseudo-device beside
			// locally enumerated physical devices. Consumers expect one row per
			// real device; retain the diagnostic placeholder only when there is
			// no physical inventory to provide context for the bridge failure.
			if len(out) == 0 {
				out = append(out, Device{ID: "bridge", Name: "Bridge attached-device registry", Kind: "bridge", Status: strategy.StatusUnavailable, Health: "unreachable", HealthReason: "bridge host node is unavailable", Capabilities: make([]strategy.Capability, 0), ObservedAt: time.Now().UTC()})
			}
		} else {
			byID := make(map[string]int, len(out))
			bySerial := make(map[string]int, len(out))
			for i, device := range out {
				if device.Kind != "physical" {
					continue
				}
				byID[device.ID] = i
				if device.Serial != "" {
					bySerial[device.Serial] = i
				}
			}
			for _, d := range attached {
				status := strategy.StatusAvailable
				reason := d.HealthReason
				if d.Reachability != "reachable" || d.TrustState != "trusted" {
					status = strategy.StatusUnavailable
				}
				index, found := byID[d.ID]
				if !found && d.Serial != "" {
					index, found = bySerial[d.Serial]
				}
				if found {
					device := &out[index]
					if device.StrategyID == "" && strings.EqualFold(d.Kind, "android") {
						device.StrategyID = "android-adb"
					}
					if d.Name != "" {
						device.Name = d.Name
						if device.Model == "" {
							device.Model = d.Name
						}
					}
					if d.OSVersion != "" {
						device.OSVersion = d.OSVersion
					}
					if d.Transport != "" {
						device.Transport = d.Transport
					}
					if d.HostNodeID != "" {
						device.HostNodeID = d.HostNodeID
					}
					if len(device.Capabilities) == 0 {
						if declaration, ok := declarationByID[device.StrategyID]; ok {
							device.Capabilities = mapCaps(declaration)
						}
					}
					device.Status, device.Health, device.HealthReason = status, status, reason
					s.devices.Upsert(recordFromDevice(*device))
					continue
				}
				byID[d.ID] = len(out)
				if d.Serial != "" {
					bySerial[d.Serial] = len(out)
				}
				strategyID := ""
				if strings.EqualFold(d.Kind, "android") {
					strategyID = "android-adb"
				}
				capabilities := make([]strategy.Capability, 0)
				if declaration, ok := declarationByID[strategyID]; ok {
					capabilities = mapCaps(declaration)
				}
				// Bridge's attached-device contract carries the platform kind
				// (currently "android"), while device-control's inventory kind
				// answers whether the row is a physical target. Keep those
				// concepts separate: every bridge peripheral is physical, and
				// the strategy id carries the platform-specific driver.
				merged := s.devices.Upsert(devicedomain.Record{ID: d.ID, Name: d.Name, Kind: "physical", Serial: d.Serial, Model: d.Name, OSVersion: d.OSVersion, StrategyID: strategyID, Transport: d.Transport, HostNodeID: d.HostNodeID, Status: status, Health: status, HealthReason: reason, Capabilities: capabilities, ObservedAt: time.Now().UTC()})
				out = append(out, deviceFromRecord(merged))
			}
		}
	}
	return out
}

// ForgetDevice removes a retained identity only when the owner explicitly
// requests it. Device discovery never calls this path, preserving stable ids
// across disconnects and adb-server restarts.
func (s *Service) ForgetDevice(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.devices.Forget(strings.TrimSpace(id))
}

func mapCaps(d strategy.Declaration) []strategy.Capability {
	names := make([]string, 0, len(d.Capabilities))
	for n := range d.Capabilities {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]strategy.Capability, 0, len(names))
	for _, n := range names {
		capability := d.Capabilities[n]
		capability.Name = n
		out = append(out, capability)
	}
	return out
}

func recordFromDevice(device Device) devicedomain.Record {
	capabilities := make([]strategy.Capability, len(device.Capabilities))
	copy(capabilities, device.Capabilities)
	return devicedomain.Record{
		ID: device.ID, Name: device.Name, Kind: device.Kind, Serial: device.Serial,
		Model: device.Model, OSVersion: device.OSVersion, StrategyID: device.StrategyID,
		Status: device.Status, Health: device.Health, HealthReason: device.HealthReason,
		HostNodeID: device.HostNodeID, Transport: device.Transport, Capabilities: capabilities,
		ObservedAt: device.ObservedAt, FirstSeenAt: device.FirstSeenAt, LastSeenAt: device.LastSeenAt,
	}
}

func (s *Service) Onboarding(kind string) []map[string]string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	common := []map[string]string{{"id": "host-node", "prerequisite": "A trusted bridge host node is online.", "owner": "owner", "status": "available", "next_action": "No action required."}}
	if kind == "android" {
		sdkStatus := probeCommand("adb")
		sdkNext := "Run `vrooli resource install android-sdk` and verify `adb version`."
		if sdkStatus == "available" {
			sdkNext = "No action required."
		}
		return append(common,
			map[string]string{"id": "android-sdk", "prerequisite": "android-sdk resource provides adb and platform-tools.", "owner": "scenario", "status": sdkStatus, "next_action": sdkNext},
			map[string]string{"id": "usb-bus", "prerequisite": "An Android device is visible on the host USB bus.", "owner": "owner", "status": "unavailable", "next_action": "Connect a data-capable cable and set USB mode to File Transfer."},
			map[string]string{"id": "usb-debugging", "prerequisite": "USB debugging is enabled and the device authorizes this host.", "owner": "owner", "status": "unavailable", "next_action": "Enable Developer Options and USB debugging, then accept the RSA prompt."})
	}
	if kind == "ios" {
		return append(common, map[string]string{"id": "xcode", "prerequisite": "Xcode and the requested simulator runtime are installed on a macOS node.", "owner": "owner", "status": "unavailable", "next_action": "Install Xcode and an iOS Simulator runtime."}, map[string]string{"id": "device-trust", "prerequisite": "The iPhone is attached and trusted.", "owner": "owner", "status": "unavailable", "next_action": "Connect the iPhone to the macOS node and tap Trust."})
	}
	return append(common, map[string]string{"id": "kind", "prerequisite": "A supported device kind is selected.", "owner": "owner", "status": "unavailable", "next_action": "Use --kind android or --kind ios."})
}

func (s *Service) OnboardingLive(ctx context.Context, kind string) []map[string]string {
	rungs := s.Onboarding(kind)
	if strings.ToLower(strings.TrimSpace(kind)) != "android" {
		return rungs
	}
	for _, item := range s.registry.List(ctx) {
		if item.StrategyID != "android-adb" {
			continue
		}
		adapter, ok := s.registry.Get(item.StrategyID)
		if !ok {
			continue
		}
		enumerator, ok := adapter.(strategy.Enumerator)
		if !ok {
			continue
		}
		busStatus, busReason := usbBusProbe()
		status, reason := "unavailable", "No Android device is visible to adb. Use a data-capable cable and set USB mode to File Transfer."
		if busStatus == "unavailable" {
			reason = busReason
		}
		devices, _ := enumerator.Enumerate(ctx)
		for _, device := range devices {
			status = "available"
			reason = "Device " + device.Serial + " is authorized and reachable."
			if device.Health != strategy.StatusAvailable {
				status = "unavailable"
				reason = device.HealthReason
			}
			break
		}
		for i := range rungs {
			if rungs[i]["id"] == "usb-bus" {
				rungs[i]["status"] = busStatus
				rungs[i]["next_action"] = busReason
			}
			if rungs[i]["id"] == "usb-debugging" {
				rungs[i]["status"] = status
				rungs[i]["next_action"] = reason
			}
		}
	}
	return rungs
}

func probeCommand(name string) string {
	if _, err := execLookPath(name); err != nil {
		return "unavailable"
	}
	return "available"
}

var execLookPath = func(name string) (string, error) { return exec.LookPath(name) }

var usbBusCommand = func() ([]byte, error) { return exec.Command("lsusb").Output() }

// usbBusProbe distinguishes a disconnected/charge-only cable from an adb
// authorization problem. Vendor ids cover the common Android manufacturers;
// adb remains the authority for serial, authorization, and device state.
func usbBusProbe() (string, string) {
	if _, err := execLookPath("lsusb"); err != nil {
		return "unavailable", "Install usbutils so the host can inspect the USB bus."
	}
	out, err := usbBusCommand()
	if err != nil {
		return "unavailable", "USB bus inspection failed; verify host permissions and install usbutils."
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		vendor := strings.ToLower(strings.SplitN(fields[5], ":", 2)[0]) + ":"
		if androidUSBVendorIDs[vendor] {
			return "available", "Android hardware is visible on the USB bus; waiting for adb authorization."
		}
	}
	return "unavailable", "No Android device is visible on the USB bus; check for a data-capable cable and File Transfer USB mode."
}

var androidUSBVendorIDs = map[string]bool{
	"04e8:": true, // Samsung
	"05c6:": true, // Qualcomm
	"0bb4:": true, // HTC
	"0e8d:": true, // MediaTek
	"12d1:": true, // Huawei
	"1782:": true, // Spreadtrum
	"18d1:": true, // Google
	"19d2:": true, // ZTE
	"22b8:": true, // Motorola
	"2717:": true, // Xiaomi
	"2a45:": true, // Meizu
	"2a70:": true, // OnePlus
	"2d95:": true, // Vivo
}
