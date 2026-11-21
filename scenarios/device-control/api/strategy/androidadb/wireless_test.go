package androidadb

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestPromoteWirelessVerifiesSerialAndKeepsIdentity(t *testing.T) { // [REQ:DVC-P0-011]
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 tcpip 5555":                                      []byte("restarting in TCP mode port: 5555"),
		"adb -s serial-1 shell ip route":                                  []byte("wlan0 192.168.1.0/24 src 192.168.1.42\n"),
		"adb connect 192.168.1.42:5555":                                   []byte("connected to 192.168.1.42:5555"),
		"adb -s 192.168.1.42:5555 shell getprop ro.serialno":              []byte("serial-1\n"),
		"adb devices -l":                                                  []byte("List of devices attached\n192.168.1.42:5555\tdevice product:a03s model:SM_A037U\n"),
		"adb -s 192.168.1.42:5555 shell getprop ro.build.version.release": []byte("11\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.PromoteWireless(context.Background()))
	devices, err := adapter.Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "serial-1", devices[0].Serial)
	require.Equal(t, "wireless", devices[0].Transport)
}

func TestPromoteWirelessRefusesSerialMismatch(t *testing.T) { // [REQ:DVC-P0-011]
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 tcpip 5555":                         []byte("ok"),
		"adb -s serial-1 shell ip route":                     []byte("wlan0 192.168.1.0/24 src 192.168.1.42\n"),
		"adb connect 192.168.1.42:5555":                      []byte("connected"),
		"adb -s 192.168.1.42:5555 shell getprop ro.serialno": []byte("another-phone\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.ErrorContains(t, adapter.PromoteWireless(context.Background()), "identity mismatch")
}

type delayedConnectRunner struct {
	*scriptedRunner
	connectAttempts int
}

func (r *delayedConnectRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "adb" && strings.Join(args, " ") == "connect 192.168.1.42:5555" {
		r.connectAttempts++
		if r.connectAttempts == 1 {
			return []byte("failed to connect: Connection refused"), nil
		}
	}
	return r.scriptedRunner.Run(ctx, name, args...)
}

func TestPromoteRetriesListenerStartupAndRequiresExactNetworkTransport(t *testing.T) {
	base := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell ip route":                     []byte("wlan0 192.168.1.0/24 src 192.168.1.42\n"),
		"adb connect 192.168.1.42:5555":                      []byte("connected to 192.168.1.42:5555"),
		"adb -s 192.168.1.42:5555 shell getprop ro.serialno": []byte("serial-1\n"),
		"adb devices -l":                                     []byte("List of devices attached\nserial-1 device usb:1-1\n"),
	}}
	runner := &delayedConnectRunner{scriptedRunner: base}
	adapter := NewWithRunner(runner, "serial-1")
	err := adapter.PromoteWireless(context.Background())
	require.ErrorContains(t, err, "wireless endpoint is not authorized")
	require.Equal(t, 2, runner.connectAttempts)
	require.Empty(t, adapter.WirelessEndpoint())
	base.responses["adb devices -l"] = []byte("List of devices attached\n192.168.1.42:5555 device\n")
	require.NoError(t, adapter.PromoteWireless(context.Background()))
	require.Equal(t, "192.168.1.42:5555", adapter.WirelessEndpoint())
}

func TestRestoredWirelessDescribeUsesSelectedEndpointState(t *testing.T) {
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\n192.168.1.42:5555\tdevice product:a03s model:SM_A037U\n"),
		"adb -s 192.168.1.42:5555 exec-out screencap -p": frame.Bytes(),
	}}
	base := NewWithRunner(runner, "serial-1")
	restored, ok := base.RestoreWireless("192.168.1.42:5555").(*Adapter)
	require.True(t, ok)

	declaration, err := restored.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Status)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapInput].Status)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapScreenshot].Status)
	require.Equal(t, "adb devices -l", runner.calls[0])
}

func TestReconnectWirelessDiscoversRotatedTLSEndpoint(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb mdns services":                                   []byte("List of discovered mdns services\nadb-serial-1._adb-tls-connect._tcp 192.168.1.42:37123\n"),
		"adb connect 192.168.1.179:5555":                      []byte("failed to connect"),
		"adb connect 192.168.1.42:37123":                      []byte("connected to 192.168.1.42:37123"),
		"adb -s 192.168.1.42:37123 wait-for-device":           []byte(""),
		"adb -s 192.168.1.42:37123 shell getprop ro.serialno": []byte("serial-1\n"),
		"adb devices -l":                                      []byte("List of devices attached\n192.168.1.42:37123\tdevice product:a03s model:SM_A037U\n"),
		"adb -s 192.168.1.179:5555 shell getprop ro.serialno": []byte("\n"),
	}}
	base := NewWithRunner(runner, "serial-1")
	restored, ok := base.RestoreWireless("192.168.1.179:5555").(*Adapter)
	require.True(t, ok)

	require.NoError(t, restored.ReconnectWireless(context.Background()))
	require.NotContains(t, runner.calls, "adb -s 192.168.1.179:5555 wait-for-device", "ADB can report a failed connection with exit code zero")
	require.Equal(t, "192.168.1.42:37123", restored.WirelessEndpoint())
	require.Equal(t, "wireless", restored.transport)
}

func TestReconnectWaitsForTransportBeforeReadingIdentity(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb connect 192.168.1.42:37123":                      []byte("connected"),
		"adb -s 192.168.1.42:37123 wait-for-device":           []byte(""),
		"adb -s 192.168.1.42:37123 shell getprop ro.serialno": []byte("serial-1\n"),
		"adb devices -l": []byte("List of devices attached\n192.168.1.42:37123\tdevice\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1").RestoreWireless("192.168.1.42:37123").(*Adapter)
	require.NoError(t, adapter.ReconnectWireless(context.Background()))
	require.Less(t, indexOfCall(runner.calls, "wait-for-device"), indexOfCall(runner.calls, "getprop ro.serialno"))
}

func TestWirelessEndpointsOnlyReturnsTLSConnectServices(t *testing.T) {
	got := wirelessEndpoints("" +
		"adb-serial-1._adb-tls-pairing._tcp 192.168.1.42:37123\n" +
		"adb-serial-1._adb-tls-connect._tcp 192.168.1.42:37124\n")
	require.Equal(t, []string{"192.168.1.42:37124"}, got)
	require.Equal(t, []string{"192.168.1.42:37124"}, wirelessEndpoints("adb-R9TT _adb-tls-connect._tcp 192.168.1.42:37124\n"))
}

func TestWirelessAddressRejectsInvalidIPv4(t *testing.T) {
	require.Equal(t, "", wirelessAddress("wlan0 999.999.999.999/24 src 999.999.999.999\n"))
	require.Equal(t, "192.168.1.42", wirelessAddress("wlan0 192.168.1.0/24 src 192.168.1.42\n"))
}

func TestWirelessDebuggingStateDistinguishesNativeAndLegacyModes(t *testing.T) {
	for _, tc := range []struct {
		name, sdk, native, port, characteristics string
		supported, enabled                       *bool
		legacy                                   *bool
	}{
		{name: "android 10 legacy enabled", sdk: "29", port: "5555", supported: boolPtr(false), legacy: boolPtr(true)},
		{name: "android 13 native disabled", sdk: "33", native: "0", port: "-1", supported: boolPtr(true), enabled: boolPtr(false), legacy: boolPtr(false)},
		{name: "android 12 TV", sdk: "32", characteristics: "tv", port: "-1", supported: boolPtr(false), legacy: boolPtr(false)},
		{name: "android 12 phone", sdk: "32", characteristics: "default", native: "1", port: "-1", supported: boolPtr(true), enabled: boolPtr(true), legacy: boolPtr(false)},
		{name: "android 12 unknown device class", sdk: "32", port: "-1", legacy: boolPtr(false)},
		{name: "unknown setting", sdk: "33", native: "null", port: "", supported: boolPtr(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &scriptedRunner{responses: map[string][]byte{
				"adb -s serial-1 shell getprop ro.build.version.sdk":         []byte(tc.sdk),
				"adb -s serial-1 shell getprop ro.build.characteristics":     []byte(tc.characteristics),
				"adb -s serial-1 shell settings get global adb_wifi_enabled": []byte(tc.native),
				"adb -s serial-1 shell getprop service.adb.tcp.port":         []byte(tc.port),
			}}
			state := NewWithRunner(runner, "serial-1").readWirelessDebuggingState(context.Background())
			require.NotNil(t, state)
			require.Equal(t, tc.supported, state.NativeSupported)
			require.Equal(t, tc.enabled, state.NativeEnabled)
			require.Equal(t, tc.legacy, state.LegacyTCPEnabled)
		})
	}
}

type wirelessSettingRunner struct {
	sdk, setting string
	ignoreWrites bool
	writes       int
}

func (r *wirelessSettingRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	call := strings.Join(args, " ")
	switch {
	case strings.HasSuffix(call, "getprop ro.build.version.sdk"):
		return []byte(r.sdk), nil
	case strings.HasSuffix(call, "settings get global adb_wifi_enabled"):
		return []byte(r.setting), nil
	case strings.Contains(call, "settings put global adb_wifi_enabled "):
		r.writes++
		if !r.ignoreWrites {
			r.setting = args[len(args)-1]
		}
		return nil, nil
	case strings.HasSuffix(call, "getprop service.adb.tcp.port"):
		return []byte("-1"), nil
	default:
		return nil, nil
	}
}

func TestNativeWirelessDebuggingRequiresUSBAndVerifiedState(t *testing.T) {
	runner := &wirelessSettingRunner{sdk: "33", setting: "0"}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.SetWirelessDebugging(context.Background(), true))
	require.Equal(t, "1", runner.setting)
	require.Equal(t, 1, runner.writes)
	require.NoError(t, adapter.SetWirelessDebugging(context.Background(), true))
	require.Equal(t, 1, runner.writes, "already enabled must be a no-op")
	require.NoError(t, adapter.SetWirelessDebugging(context.Background(), false))
	require.Equal(t, "0", runner.setting)
	runner.setting = "null"
	require.NoError(t, adapter.SetWirelessDebugging(context.Background(), true), "an unset setting must be writable and verified")
	require.Equal(t, "1", runner.setting)
	runner.setting = "0"

	runner.ignoreWrites = true
	require.ErrorContains(t, adapter.SetWirelessDebugging(context.Background(), true), "not verified")
	runner.sdk = "29"
	require.Error(t, adapter.SetWirelessDebugging(context.Background(), true))
	before := runner.writes
	wireless := NewWithRunner(runner, "192.0.2.1:5555")
	require.ErrorContains(t, wireless.SetWirelessDebugging(context.Background(), true), "USB")
	require.Equal(t, before, runner.writes)
}

func TestValidateADBEndpoint(t *testing.T) {
	cases := []struct {
		endpoint string
		ok       bool
	}{
		{"192.168.1.158:5555", true},
		{"tv.local:5555", true},
		{"[2600:4040::1]:5555", true},
		{"", false},
		{"192.168.1.158", false},
		{"192.168.1.158:0", false},
		{"192.168.1.158:99999", false},
		{"192.168.1.158:5555; rm -rf /", false},
		{"$(reboot):5555", false},
	}
	for _, tc := range cases {
		err := validateADBEndpoint(tc.endpoint)
		if tc.ok {
			require.NoErrorf(t, err, "endpoint %q should be accepted", tc.endpoint)
		} else {
			require.Errorf(t, err, "endpoint %q should be rejected", tc.endpoint)
		}
	}
}

func TestConnectNetworkVerifiesSerialIdentity(t *testing.T) {
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	endpoint := "192.168.1.158:5555"
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb connect 192.168.1.158:5555":                                   []byte("connected to 192.168.1.158:5555"),
		"adb -s 192.168.1.158:5555 wait-for-device":                        []byte(""),
		"adb devices -l":                                                   []byte("List of devices attached\n192.168.1.158:5555\tdevice product:tv model:SmartTV_4K\n"),
		"adb -s 192.168.1.158:5555 shell getprop ro.serialno":              []byte("AE70A4D38B\n"),
		"adb -s 192.168.1.158:5555 shell getprop ro.build.version.release": []byte("12\n"),
		"adb -s 192.168.1.158:5555 exec-out screencap -p":                  frame.Bytes(),
	}}
	adapter := NewWithRunner(runner, "")
	device, err := adapter.ConnectNetwork(context.Background(), endpoint)
	require.NoError(t, err)
	require.Equal(t, "AE70A4D38B", device.Serial)
	require.Equal(t, endpoint, device.Endpoint)
	require.Equal(t, "wireless", device.Transport)
	// Identity is derived from the hardware serial, not the network address.
	require.NotContains(t, device.ID, endpoint)
	require.Equal(t, "AE70A4D38B", adapter.identitySerial)
}

func TestConnectNetworkRejectsUnauthorizedEndpoint(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb connect 192.168.1.158:5555":            []byte("connected to 192.168.1.158:5555"),
		"adb -s 192.168.1.158:5555 wait-for-device": []byte(""),
		"adb devices -l":                            []byte("List of devices attached\n192.168.1.158:5555\tunauthorized\n"),
	}}
	adapter := NewWithRunner(runner, "")
	_, err := adapter.ConnectNetwork(context.Background(), "192.168.1.158:5555")
	require.ErrorContains(t, err, "unauthorized")
}

func TestConnectNetworkRejectsFailedConnect(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb connect 192.168.1.158:5555": []byte("failed to connect to '192.168.1.158:5555'"),
	}}
	adapter := NewWithRunner(runner, "")
	_, err := adapter.ConnectNetwork(context.Background(), "192.168.1.158:5555")
	require.ErrorContains(t, err, "failed to connect")
}
