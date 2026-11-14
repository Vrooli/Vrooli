package control

import (
	"context"
	authdomain "device-control/internal/auth"
	"fmt"
	"os/exec"
	"strings"

	"device-control/strategy"
)

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
			map[string]string{"id": "wireless-adb", "prerequisite": "An Android device is visible through authorized wireless ADB.", "owner": "owner", "status": "unavailable", "next_action": "Enable Wireless debugging and authorize this host."},
			map[string]string{"id": "unattended-unlock", "prerequisite": "A locked phone can be used without unlocking it by hand: an active unlock profile with a provisioned PIN (the lock screen's security settings are never changed).", "owner": "owner", "status": "unavailable", "next_action": "Once per phone: device-control auth create --device <device-id> --method pin --credential-identity device-control/<device-id>/unlock --credential-field pin, then read -rs PIN && printf %s \"$PIN\" | device-control auth provision <profile-id>."},
			map[string]string{"id": "usb-debugging", "prerequisite": "USB debugging is enabled and the device authorizes this host.", "owner": "owner", "status": "unavailable", "next_action": "Enable Developer Options and USB debugging, then accept the RSA prompt."})
	}
	if kind == "google-tv" || kind == "googletv" {
		return append(common,
			map[string]string{"id": "lan-multicast", "prerequisite": "The control host and television share a multicast-reachable LAN segment.", "owner": "owner", "status": "available", "next_action": "No action required."},
			map[string]string{"id": "google-tv-discovery", "prerequisite": "The television advertises Android TV Remote or Google Cast DNS-SD services.", "owner": "owner", "status": "unavailable", "next_action": "Enable network access on the television and run device discover."},
			map[string]string{"id": "remote-pairing", "prerequisite": "The owner has opened the Android TV Remote pairing prompt and has its six-character hexadecimal code.", "owner": "owner", "status": "unavailable", "next_action": "Open the pairing prompt on the television, then run device pair with the displayed six-character hexadecimal code."},
			map[string]string{"id": "paired-transport", "prerequisite": "The Android TV Remote certificate is stored in the credential authority and reconnects.", "owner": "scenario", "status": "unavailable", "next_action": "Complete pairing once; subsequent discovery should show the remote as paired."})
	}
	if kind == "ios" {
		return append(common, map[string]string{"id": "xcode", "prerequisite": "Xcode and the requested simulator runtime are installed on a macOS node.", "owner": "owner", "status": "unavailable", "next_action": "Install Xcode and an iOS Simulator runtime."}, map[string]string{"id": "device-trust", "prerequisite": "The iPhone is attached and trusted.", "owner": "owner", "status": "unavailable", "next_action": "Connect the iPhone to the macOS node and tap Trust."})
	}
	return append(common, map[string]string{"id": "kind", "prerequisite": "A supported device kind is selected.", "owner": "owner", "status": "unavailable", "next_action": "Select a supported onboarding kind."})
}

func (s *Service) OnboardingLive(ctx context.Context, kind string) []map[string]string {
	rungs := s.Onboarding(kind)
	if strings.ToLower(strings.TrimSpace(kind)) == "google-tv" || strings.ToLower(strings.TrimSpace(kind)) == "googletv" {
		services, err := s.DiscoverLAN(ctx)
		return applyGoogleTVOnboardingProbe(rungs, services, err)
	}
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
		bus := sampleUSBLink()
		busStatus, busReason := bus.Status, bus.Reason
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
			for i := range rungs {
				if rungs[i]["id"] == "wireless-adb" && device.Transport == "wireless" && status == "available" {
					rungs[i]["status"] = "available"
					rungs[i]["next_action"] = "No action required; authorized wireless ADB is reachable."
				}
			}
			break
		}
		for i := range rungs {
			if rungs[i]["id"] == "unattended-unlock" && s.auth != nil {
				ready := 0
				for _, p := range s.auth.List(ctx) {
					if _, reason := authdomain.UnlockCandidate([]authdomain.Profile{p}, p.DeviceID); reason == "" && p.Status == "active" {
						ready++
					}
				}
				if ready > 0 {
					rungs[i]["status"] = "available"
					rungs[i]["next_action"] = "No action required; at least one phone has a usable unlock profile (device-control auth list)."
				}
			}
			if rungs[i]["id"] == "usb-bus" {
				rungs[i]["status"] = busStatus
				rungs[i]["next_action"] = busReason
				rungs[i]["flap_count"] = fmt.Sprintf("%d", bus.FlapCount)
			}
			if rungs[i]["id"] == "usb-debugging" {
				rungs[i]["status"] = status
				rungs[i]["next_action"] = reason
			}
		}
	}
	return rungs
}

func applyGoogleTVOnboardingProbe(rungs []map[string]string, services []DiscoveredService, discoverErr error) []map[string]string {
	// A browse can return useful instances from one transport while another
	// transport reports a timeout. Preserve the successful discovery in the
	// ladder and make the degraded condition explicit instead of hiding the TV.
	found := len(services) > 0
	remoteFound := false
	remotePaired := false
	instances := make([]string, 0, len(services))
	seenInstances := map[string]struct{}{}
	for _, service := range services {
		instance := strings.TrimSpace(service.Name)
		if instance == "" {
			instance = strings.TrimSpace(service.ID)
		}
		if instance != "" {
			if _, alreadySeen := seenInstances[instance]; !alreadySeen {
				instances = append(instances, instance)
				seenInstances[instance] = struct{}{}
			}
		}
		if service.StrategyID == "android-tv-remote" {
			remoteFound = true
			remotePaired = remotePaired || service.Paired
		}
	}
	for i := range rungs {
		switch rungs[i]["id"] {
		case "lan-multicast":
			if discoverErr == nil {
				rungs[i]["status"] = "available"
				rungs[i]["next_action"] = "No action required."
			} else if found {
				rungs[i]["status"] = "degraded"
				rungs[i]["next_action"] = "Multicast browse partially succeeded: " + discoverErr.Error()
			} else {
				rungs[i]["status"] = "unreachable"
				rungs[i]["next_action"] = "Multicast browse failed: " + discoverErr.Error()
			}
		case "google-tv-discovery":
			if found {
				rungs[i]["status"] = "available"
				rungs[i]["next_action"] = "Discovered: " + strings.Join(instances, ", ")
				if discoverErr != nil {
					rungs[i]["next_action"] += " (degraded: " + discoverErr.Error() + ")"
				}
			}
		case "remote-pairing":
			if remotePaired {
				rungs[i]["status"] = "available"
				rungs[i]["next_action"] = "No action required; the Android TV Remote certificate is stored."
			} else if remoteFound {
				rungs[i]["next_action"] = "Enter the six-character hexadecimal code shown by the television with device pair."
			}
		case "paired-transport":
			if remotePaired {
				rungs[i]["status"] = "available"
				rungs[i]["next_action"] = "No action required; the paired transport reconnects without another code."
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

var systemProfilerCommand = func() ([]byte, error) { return exec.Command("system_profiler", "SPUSBDataType").Output() }

var windowsUSBCommand = func() ([]byte, error) {
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-PnpDevice -Class USB | Format-List -Property InstanceId,Status,Class").Output()
}

const (
	usbBusPresent       = "available"
	usbBusAbsent        = "unavailable"
	usbBusCannotInspect = "cannot-inspect"
)

type usbBusInspection struct {
	Status    string
	Reason    string
	FlapCount int
}

type usbBusInspector interface {
	Inspect() usbBusInspection
}

type linuxUSBInspector struct{}

func (linuxUSBInspector) Inspect() usbBusInspection {
	if _, err := execLookPath("lsusb"); err != nil {
		return usbBusInspection{Status: usbBusCannotInspect, Reason: "USB bus inspection is unavailable on Linux; install usbutils so the host can inspect Android hardware."}
	}
	out, err := usbBusCommand()
	if err != nil {
		return usbBusInspection{Status: usbBusCannotInspect, Reason: "USB bus inspection failed on Linux; verify host permissions and install usbutils."}
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			vendor := strings.ToLower(strings.SplitN(fields[5], ":", 2)[0]) + ":"
			if androidUSBVendorIDs[vendor] {
				return usbBusInspection{Status: usbBusPresent, Reason: "Android hardware is visible on the Linux USB bus; waiting for adb authorization."}
			}
		}
	}
	return usbBusInspection{Status: usbBusAbsent, Reason: "No Android device is visible on the Linux USB bus; check for a data-capable cable and File Transfer USB mode."}
}

type darwinUSBInspector struct{}

func (darwinUSBInspector) Inspect() usbBusInspection {
	if _, err := execLookPath("system_profiler"); err != nil {
		return usbBusInspection{Status: usbBusCannotInspect, Reason: "USB bus inspection is unavailable on macOS; use the built-in system_profiler command."}
	}
	out, err := systemProfilerCommand()
	if err != nil {
		return usbBusInspection{Status: usbBusCannotInspect, Reason: "macOS system_profiler could not inspect the USB bus; check host permissions."}
	}
	if containsAndroidUSBVendor(string(out)) {
		return usbBusInspection{Status: usbBusPresent, Reason: "Android hardware is visible through macOS system_profiler; waiting for adb authorization."}
	}
	return usbBusInspection{Status: usbBusAbsent, Reason: "No Android device is visible in macOS system_profiler; check the data cable and USB mode."}
}

type windowsUSBInspector struct{}

func (windowsUSBInspector) Inspect() usbBusInspection {
	if _, err := execLookPath("powershell"); err != nil {
		return usbBusCannotInspectResult("USB bus inspection is unavailable on Windows; use PowerShell PnP device enumeration.")
	}
	out, err := windowsUSBCommand()
	if err != nil {
		return usbBusCannotInspectResult("Windows PnP USB enumeration failed; check device-manager permissions.")
	}
	if containsAndroidUSBVendor(string(out)) || strings.Contains(strings.ToLower(string(out)), "android") {
		return usbBusInspection{Status: usbBusPresent, Reason: "Android hardware is visible through Windows PnP enumeration; waiting for adb authorization."}
	}
	return usbBusInspection{Status: usbBusAbsent, Reason: "No Android device is visible in Windows PnP enumeration; check the data cable and USB mode."}
}

func usbBusCannotInspectResult(reason string) usbBusInspection {
	return usbBusInspection{Status: usbBusCannotInspect, Reason: reason}
}

func containsAndroidUSBVendor(output string) bool {
	lower := strings.ToLower(output)
	for vendor := range androidUSBVendorIDs {
		if strings.Contains(lower, strings.TrimSuffix(vendor, ":")) {
			return true
		}
	}
	return false
}

func selectedUSBInspector() usbBusInspector {
	switch strategy.HostOS {
	case "darwin":
		return darwinUSBInspector{}
	case "windows":
		return windowsUSBInspector{}
	case "linux":
		return linuxUSBInspector{}
	default:
		return unknownUSBInspector{}
	}
}

type unknownUSBInspector struct{}

func (unknownUSBInspector) Inspect() usbBusInspection {
	return usbBusCannotInspectResult(fmt.Sprintf("USB bus inspection is not implemented for host OS %q.", strategy.HostOS))
}

func sampleUSBLink() usbBusInspection {
	const sampleCount = 3
	first := selectedUSBInspector().Inspect()
	if first.Status == usbBusCannotInspect {
		return first
	}
	previousPresent := first.Status == usbBusPresent
	flaps := 0
	for i := 1; i < sampleCount; i++ {
		current := selectedUSBInspector().Inspect()
		if current.Status == usbBusCannotInspect {
			return current
		}
		present := current.Status == usbBusPresent
		if present != previousPresent {
			flaps++
		}
		previousPresent = present
		first = current
	}
	first.FlapCount = flaps
	if flaps > 0 {
		first.Reason = fmt.Sprintf("Android USB link is intermittent (%d flap(s) observed); replace the cable before running a flow.", flaps)
	} else if first.Status == usbBusPresent {
		first.Reason += " Link stability sample passed; flap_count=0."
	}
	return first
}

// usbBusProbe distinguishes a disconnected/charge-only cable from an adb
// authorization problem. Vendor ids cover the common Android manufacturers;
// adb remains the authority for serial, authorization, and device state.
func usbBusProbe() (string, string) {
	result := selectedUSBInspector().Inspect()
	return result.Status, result.Reason
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
