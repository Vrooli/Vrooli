package hostinventory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// HCI registration is weaker than a usable BlueZ controller or a verified peer
// connection. Read only the enumerated device's Bluetooth functions, never a
// second machine-wide roster or a command that powers or probes the radio.
func readSysfsBluetoothInterfaces(files IntegrityFileSystem, device *DeviceInfo) error {
	var probeErr error
	applicable, allBlocked := false, true
	for _, function := range device.Functions {
		if !isLinuxBluetoothFunction(function) {
			continue
		}
		applicable = true
		root := filepath.Join(device.SysPath, function.ID, "bluetooth")
		entries, err := files.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			probeErr = errors.Join(probeErr, fmt.Errorf("Bluetooth function %s HCI enumeration: %w", function.ID, err))
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if !isLinuxHCIName(name) {
				continue
			}
			device.Readiness.InterfaceNames = append(device.Readiness.InterfaceNames, name)
			blocked, status, err := readHCIRadioBlock(files, filepath.Join(root, name))
			probeErr = errors.Join(probeErr, err)
			allBlocked = allBlocked && blocked
			if device.Readiness.InterfaceStatus == nil {
				device.Readiness.InterfaceStatus = map[string]string{}
			}
			device.Readiness.InterfaceStatus[name] = status
		}
	}
	if !applicable {
		return nil
	}
	slices.Sort(device.Readiness.InterfaceNames)
	switch {
	case probeErr != nil:
		device.Readiness.InterfaceState = DeviceReadinessUnknown
		device.Readiness.InterfaceReason = "Bluetooth interface or radio-block observation failed: " + probeErr.Error()
	case len(device.Readiness.InterfaceNames) == 0:
		device.Readiness.InterfaceState = DeviceReadinessAbsent
		device.Readiness.InterfaceReason = "no HCI interface is exposed for the enumerated Bluetooth functions"
	case allBlocked:
		device.Readiness.InterfaceState = DeviceReadinessDisabled
		device.Readiness.InterfaceReason = "all registered HCI interfaces are explicitly blocked by rfkill; controller usability and peer function are not verified"
	default:
		device.Readiness.InterfaceState = DeviceReadinessInitialized
		device.Readiness.InterfaceReason = "a kernel HCI interface is registered; rfkill status does not establish power, usable OS controller, firmware initialization or peer function"
	}
	return probeErr
}

func isLinuxBluetoothFunction(function DeviceFunction) bool {
	return function.BoundDriver == "btusb" || strings.Contains(strings.ToLower(function.Modalias), "ice0isc01ip01")
}

func isLinuxHCIName(name string) bool {
	if !strings.HasPrefix(name, "hci") || len(name) == len("hci") {
		return false
	}
	for _, c := range name[len("hci"):] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func readHCIRadioBlock(files IntegrityFileSystem, root string) (bool, string, error) {
	entries, err := files.ReadDir(root)
	if err != nil {
		return false, DeviceReadinessUnknown, err
	}
	found, blocked := false, false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "rfkill") {
			continue
		}
		found = true
		for _, field := range []string{"soft", "hard"} {
			path := filepath.Join(root, entry.Name(), field)
			data, err := files.ReadFile(path)
			if err != nil {
				return false, DeviceReadinessUnknown, fmt.Errorf("read %s: %w", path, err)
			}
			switch strings.TrimSpace(string(data)) {
			case "0":
			case "1":
				blocked = true
			default:
				return false, DeviceReadinessUnknown, fmt.Errorf("invalid rfkill %s value", field)
			}
		}
	}
	if !found {
		return false, "radio_block_unknown", nil
	}
	if blocked {
		return true, "rfkill_blocked", nil
	}
	return false, "rfkill_unblocked", nil
}

// ObserveLinuxDeviceReadiness fills readiness for already-enumerated Linux
// devices. Each input carries the bus address, class and resolved sysfs path.
// The caller supplies a bounded command runner for one kernel journal read
// per observation. Enumeration, binding and telemetry stay with their owners.
func ObserveLinuxDeviceReadiness(ctx context.Context, devices []DeviceInfo, run func(context.Context, string, ...string) ([]byte, error)) {
	for i := range devices {
		_ = observeLinuxDeviceInterfaces(nativeIntegrityFiles{}, &devices[i])
	}
	observeLinuxBluetoothControllers(ctx, devices, run)
	if len(devices) == 0 || run == nil || ctx.Err() != nil {
		return
	}
	raw, err := run(ctx, "journalctl", integrityJournalArgs()...)
	applyMissingFirmwareEvidence(devices, parseIntegritySignals(raw, err))
}

func observeLinuxBluetoothControllers(ctx context.Context, devices []DeviceInfo, run func(context.Context, string, ...string) ([]byte, error)) {
	hasHCI := false
	for _, device := range devices {
		if !slices.ContainsFunc(device.Functions, isLinuxBluetoothFunction) {
			continue
		}
		for _, name := range device.Readiness.InterfaceNames {
			hasHCI = hasHCI || isLinuxHCIName(name)
		}
	}
	if !hasHCI {
		return
	}
	var controllers map[string]LinuxBluetoothController
	var err error
	if run == nil {
		err = errors.New("no passive BlueZ command observer is available")
	} else {
		probeCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		var raw []byte
		raw, err = run(probeCtx, "/usr/bin/busctl", "--system", "--no-pager", "--timeout=1", "--auto-start=no", "--allow-interactive-authorization=no", "--json=short", "call", "org.bluez", "/", "org.freedesktop.DBus.ObjectManager", "GetManagedObjects")
		if err == nil {
			controllers, err = ParseLinuxBluetoothControllers(raw)
		}
	}
	for i := range devices {
		if !slices.ContainsFunc(devices[i].Functions, isLinuxBluetoothFunction) {
			continue
		}
		readiness := &devices[i].Readiness
		var details []string
		allDisabled, hasController := true, false
		for _, name := range readiness.InterfaceNames {
			if !isLinuxHCIName(name) {
				continue
			}
			hasController = true
			status := "controller_unknown"
			controller, exists := controllers[name]
			switch {
			case err != nil:
				details = append(details, name+": usable OS controller observation unknown: "+err.Error())
			case !exists:
				status = "controller_absent"
				details = append(details, name+": kernel HCI registered but BlueZ reports no OS controller")
			case !controller.PowerKnown:
				details = append(details, name+": BlueZ controller present, power unknown")
			case !controller.Powered:
				status = "controller_disabled"
				details = append(details, name+": BlueZ controller present with Powered=false")
			default:
				status = "controller_powered"
				details = append(details, name+": BlueZ controller powered; peer function remains unverified")
			}
			allDisabled = allDisabled && (status == "controller_disabled" || readiness.InterfaceStatus[name] == "rfkill_blocked")
			// Preserve independent HCI/rfkill evidence in the same per-interface
			// observation rather than replacing registration with OS usability.
			readiness.InterfaceStatus[name] += ";" + status
		}
		if !hasController {
			continue
		}
		if allDisabled && readiness.InterfaceState != DeviceReadinessUnknown {
			readiness.InterfaceState = DeviceReadinessDisabled
		}
		readiness.InterfaceReason += "; " + strings.Join(details, "; ")
		readiness.FunctionReason = strings.Join(details, "; ") + "; end-to-end Bluetooth function has not been verified"
	}
}
