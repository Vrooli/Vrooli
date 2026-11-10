package androidadb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) WirelessEndpoint() string { return a.endpoint }

// ReconnectWireless re-establishes the persisted wireless endpoint and, when
// Android wireless debugging has rotated its TLS port, discovers a current
// endpoint through adb's mDNS browser. Every candidate is verified against the
// onboarded hardware serial before it can replace the durable endpoint.
func (a *Adapter) ReconnectWireless(ctx context.Context) error {
	if a.endpoint == "" {
		return fmt.Errorf("wireless reconnect requires a persisted endpoint")
	}
	candidates := []string{a.endpoint}
	if discovered, err := a.runner.Run(ctx, "adb", "mdns", "services"); err == nil {
		candidates = append(candidates, wirelessEndpoints(string(discovered))...)
	}
	var lastErr error
	seen := make(map[string]bool, len(candidates))
	for _, endpoint := range candidates {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" || seen[endpoint] {
			continue
		}
		seen[endpoint] = true
		out, err := a.runner.Run(ctx, "adb", "connect", endpoint)
		if err != nil {
			lastErr = fmt.Errorf("connect wireless endpoint: %w", err)
			continue
		}
		if adbConnectFailed(out) {
			lastErr = fmt.Errorf("connect wireless endpoint %s: %s", endpoint, boundedDiagnostic(out))
			continue
		}
		if err := a.waitForDevice(ctx, endpoint); err != nil {
			lastErr = err
			continue
		}
		if err := a.verifyWirelessEndpoint(ctx, endpoint); err != nil {
			lastErr = err
			continue
		}
		a.endpoint = endpoint
		a.transport = "wireless"
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no wireless ADB endpoint was discovered for the onboarded device")
}

func (a *Adapter) waitForDevice(ctx context.Context, endpoint string) error {
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.runner.Run(waitCtx, "adb", "-s", endpoint, "wait-for-device"); err != nil {
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("wireless_reconnect_timeout: wait-for-device timed out for %s", endpoint)
		}
		return fmt.Errorf("wireless_reconnect_wait_failed: wait-for-device for %s: %w", endpoint, err)
	}
	return nil
}

func (a *Adapter) verifyWirelessEndpoint(ctx context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	serialCheck, err := a.runner.Run(ctx, "adb", "-s", endpoint, "shell", "getprop", "ro.serialno")
	if err != nil {
		return fmt.Errorf("verify wireless endpoint: %w", err)
	}
	if strings.TrimSpace(string(serialCheck)) != a.identitySerial {
		return fmt.Errorf("wireless adb identity mismatch: expected onboarded device")
	}
	devices, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return fmt.Errorf("verify wireless adb device state: %w", err)
	}
	for _, line := range strings.Split(string(devices), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == endpoint && fields[1] == "device" {
			return nil
		}
	}
	return fmt.Errorf("wireless endpoint is not authorized")
}

func wirelessEndpoints(output string) []string {
	endpoints := make([]string, 0)
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		endpoint := ""
		if len(fields) >= 3 && fields[1] == "_adb-tls-connect._tcp" {
			endpoint = strings.TrimSpace(fields[2])
		} else if len(fields) >= 2 && strings.Contains(fields[0], "_adb-tls-connect._tcp") {
			endpoint = strings.TrimSpace(fields[1])
		}
		if strings.Contains(endpoint, ":") && !seen[endpoint] {
			seen[endpoint] = true
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func transportForSerial(serial string) string {
	if strings.Contains(serial, ":") {
		return "wireless"
	}
	return "usb"
}

// WirelessPromoter is the narrow service seam for promoting an already
// onboarded USB device. It keeps transport changes out of the inventory store
// until the returned ADB identity has been verified.
type WirelessPromoter interface {
	PromoteWireless(context.Context) error
}

func (a *Adapter) PromoteWireless(ctx context.Context) error {
	if strings.TrimSpace(a.serial) == "" {
		return fmt.Errorf("wireless promotion requires a USB-onboarded device serial")
	}
	if a.transport != "usb" {
		return fmt.Errorf("device is already using %s transport", a.transport)
	}
	// Read the address while the USB transport is still usable. `adb tcpip`
	// restarts adbd and commonly drops the USB-selected endpoint immediately.
	route, err := a.runner.Run(ctx, "adb", a.args("shell", "ip", "route")...)
	if err != nil {
		return fmt.Errorf("read device wireless address: %w", err)
	}
	address := wirelessAddress(string(route))
	if address == "" {
		return fmt.Errorf("device did not report a wireless address")
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("tcpip", wirelessADBPort)...); err != nil {
		return fmt.Errorf("enable adb tcpip mode: %w", err)
	}
	endpoint := address + ":" + wirelessADBPort
	// adbd restarts asynchronously; retry connection failures until the new
	// listener exists. A successful USB transport is never proof that this
	// particular network endpoint is connected to the same hardware.
	var connectErr error
	for attempt := 0; attempt < 20; attempt++ {
		var output []byte
		output, connectErr = a.runner.Run(ctx, "adb", "connect", endpoint)
		if connectErr == nil && !adbConnectFailed(output) {
			break
		}
		if connectErr == nil {
			connectErr = fmt.Errorf("%s", boundedDiagnostic(output))
		}
		if attempt < 19 {
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
	if connectErr != nil {
		return fmt.Errorf("connect wireless adb: %w", connectErr)
	}
	if err := a.waitForDevice(ctx, endpoint); err != nil {
		return err
	}
	if err := a.verifyWirelessEndpoint(ctx, endpoint); err != nil {
		return err
	}
	a.endpoint = endpoint
	a.transport = "wireless"
	return nil
}

// validateADBEndpoint bounds the caller-supplied network address to a host:port
// pair with a numeric port. It rejects shell metacharacters so the value can
// never widen `adb connect` into an arbitrary command; only a transport address
// is ever accepted from the caller.
func validateADBEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("network adb endpoint is required (host:port)")
	}
	if len(endpoint) > 255 {
		return fmt.Errorf("network adb endpoint exceeds bounded length")
	}
	if strings.ContainsAny(endpoint, " \t\r\n;&|$`\"'\\<>()") {
		return fmt.Errorf("network adb endpoint contains unsupported characters")
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("network adb endpoint must be host:port: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("network adb endpoint requires a host")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("network adb endpoint requires a valid TCP port")
	}
	return nil
}

// endpointTransportState reports whether the endpoint currently shows a "device"
// state in `adb devices -l`, returning a typed, owner-actionable reason when it
// is unauthorized, offline, or absent.
func (a *Adapter) endpointTransportState(ctx context.Context, endpoint string) (string, error) {
	out, err := a.runner.Run(ctx, "adb", "devices", "-l")
	if err != nil {
		return "", fmt.Errorf("verify adb endpoint transport: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == endpoint {
			return fields[1], nil
		}
	}
	return "", nil
}

// ConnectNetwork onboards a directly-addressable Android ADB endpoint — classic
// TCP ADB (port 5555, e.g. an Android TV or TV box) or Wireless Debugging/TLS —
// without a prior USB attachment. It runs `adb connect`, waits for the
// transport, confirms authorization, and reads the hardware serial over the
// verified endpoint so the durable identity is the device, never the network
// address. The receiver becomes endpoint-bound and the verified Device is
// returned for the caller to persist as the device's transport strategy.
func adbConnectFailed(out []byte) bool {
	message := strings.ToLower(string(out))
	for _, failure := range []string{"cannot connect", "failed to connect", "unable to connect", "failed to authenticate"} {
		if strings.Contains(message, failure) {
			return true
		}
	}
	return false
}

func (a *Adapter) ConnectNetwork(ctx context.Context, endpoint string) (strategy.Device, error) {
	endpoint = strings.TrimSpace(endpoint)
	if err := validateADBEndpoint(endpoint); err != nil {
		return strategy.Device{}, err
	}
	if out, err := a.runner.Run(ctx, "adb", "connect", endpoint); err != nil {
		return strategy.Device{}, fmt.Errorf("connect adb endpoint %q: %w: %s", endpoint, err, strings.TrimSpace(string(out)))
	} else if adbConnectFailed(out) {
		return strategy.Device{}, fmt.Errorf("connect adb endpoint %q: %s", endpoint, strings.TrimSpace(string(out)))
	}
	if _, err := a.runner.Run(ctx, "adb", "-s", endpoint, "wait-for-device"); err != nil {
		return strategy.Device{}, fmt.Errorf("wait for adb endpoint %q: %w", endpoint, err)
	}
	state, err := a.endpointTransportState(ctx, endpoint)
	if err != nil {
		return strategy.Device{}, err
	}
	switch state {
	case "device":
	case "unauthorized":
		return strategy.Device{}, fmt.Errorf("adb endpoint %q is unauthorized; accept the debugging prompt on the device and retry", endpoint)
	case "":
		return strategy.Device{}, fmt.Errorf("adb endpoint %q did not attach; verify the address and that ADB debugging is enabled", endpoint)
	default:
		return strategy.Device{}, fmt.Errorf("adb endpoint %q is in state %q; expected an authorized device", endpoint, state)
	}
	// The hardware serial (never the transport address) is the durable identity.
	serial := ""
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		out, propErr := a.runner.Run(ctx, "adb", "-s", endpoint, "shell", "getprop", "ro.serialno")
		if propErr == nil {
			if candidate := strings.TrimSpace(string(out)); candidate != "" {
				serial = candidate
				break
			}
		}
		lastErr = propErr
		if attempt < 4 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if serial == "" {
		if lastErr == nil {
			lastErr = fmt.Errorf("device did not report ro.serialno")
		}
		return strategy.Device{}, fmt.Errorf("verify adb endpoint identity: %w", lastErr)
	}
	a.endpoint = endpoint
	a.serial = serial
	a.identitySerial = serial
	a.transport = "wireless"
	devices, err := a.Enumerate(ctx)
	if err != nil {
		return strategy.Device{}, fmt.Errorf("enumerate onboarded endpoint %q: %w", endpoint, err)
	}
	for _, device := range devices {
		if device.Endpoint == endpoint || device.Serial == serial {
			return device, nil
		}
	}
	return strategy.Device{}, fmt.Errorf("onboarded endpoint %q was not present after connect", endpoint)
}

func wirelessAddress(route string) string {
	fields := strings.Fields(route)
	for i, field := range fields {
		if field == "src" && i+1 < len(fields) && isIPv4(fields[i+1]) {
			return fields[i+1]
		}
	}
	for _, field := range fields {
		if isIPv4(field) && !strings.HasPrefix(field, "0.") {
			return field
		}
	}
	return ""
}

func isIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil
}

func (a *Adapter) readWirelessDebuggingState(ctx context.Context) *strategy.WirelessDebuggingState {
	state := &strategy.WirelessDebuggingState{}
	if raw, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "ro.build.version.sdk")...); err == nil {
		if sdk, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && sdk > 0 {
			supported := sdk >= 30
			known := true
			// Android's documented minimum for TV and Wear is API 33, while
			// phones support paired wireless debugging from API 30.
			if sdk >= 30 && sdk < 33 {
				characteristics, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "ro.build.characteristics")...)
				value := strings.ToLower(strings.TrimSpace(string(characteristics)))
				if err != nil || value == "" {
					known = false
				} else if strings.Contains(value, "tv") || strings.Contains(value, "watch") {
					supported = false
				}
			}
			if known {
				state.NativeSupported = &supported
			}
			if known && supported {
				if enabledRaw, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "get", "global", "adb_wifi_enabled")...); err == nil {
					switch strings.TrimSpace(string(enabledRaw)) {
					case "0", "1":
						enabled := strings.TrimSpace(string(enabledRaw)) == "1"
						state.NativeEnabled = &enabled
					}
				}
			}
		}
	}
	if raw, err := a.runner.Run(ctx, "adb", a.args("shell", "getprop", "service.adb.tcp.port")...); err == nil {
		if port, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && port >= -1 && port <= 65535 {
			enabled := port > 0
			state.LegacyTCPEnabled = &enabled
			if enabled {
				state.LegacyTCPPort = port
			}
		}
	}
	if state.NativeSupported == nil && state.LegacyTCPEnabled == nil {
		return nil
	}
	return state
}

func (a *Adapter) SetWirelessDebugging(ctx context.Context, enabled bool) error {
	if a.transport != "usb" || strings.TrimSpace(a.serial) == "" {
		return fmt.Errorf("native wireless debugging changes require an authorized serial-bound USB transport")
	}
	before := a.readWirelessDebuggingState(ctx)
	if before == nil || before.NativeSupported == nil || !*before.NativeSupported {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapWirelessDebugging, Operation: "set"}
	}
	if before.NativeEnabled != nil && *before.NativeEnabled == enabled {
		return nil
	}
	value := "0"
	if enabled {
		value = "1"
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "global", "adb_wifi_enabled", value)...); err != nil {
		return fmt.Errorf("change native wireless debugging: %w", err)
	}
	after := a.readWirelessDebuggingState(ctx)
	if after == nil || after.NativeEnabled == nil || *after.NativeEnabled != enabled {
		return fmt.Errorf("native wireless debugging change was not verified; the OS may require network authorization")
	}
	return nil
}
