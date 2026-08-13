package androidadb

import (
	"context"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

type scriptedRunner struct {
	responses map[string][]byte
	calls     []string
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, call)
	return r.responses[call], nil
}

func TestEnumerateDerivesStableIdentityAndHealth(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\nR9TT608Q6MH\tdevice usb:1-1 product:a03s model:SM_A037U device:a03s\nunauthorized-1\tunauthorized usb:1-2 model:Pixel_7\noffline-1\toffline usb:1-3 model:Pixel_6\n"),
		"adb -s R9TT608Q6MH shell getprop ro.build.version.release": []byte("11\n"),
	}}
	adapter := NewWithRunner(runner, "")
	devices, err := adapter.Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 3)
	require.Equal(t, "R9TT608Q6MH", devices[0].Serial)
	require.Equal(t, "SM_A037U", devices[0].Model)
	require.Equal(t, "11", devices[0].OSVersion)
	require.Equal(t, "available", devices[0].Health)
	require.Equal(t, "unauthorized", devices[1].Health)
	require.Equal(t, strategy.HealthUnreachable, devices[2].Health)
	require.NotEqual(t, devices[0].ID, devices[1].ID)
	second, err := adapter.Enumerate(context.Background())
	require.NoError(t, err)
	require.Equal(t, devices[0].ID, second[0].ID)
}

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

func TestWirelessAddressRejectsInvalidIPv4(t *testing.T) {
	require.Equal(t, "", wirelessAddress("wlan0 999.999.999.999/24 src 999.999.999.999\n"))
	require.Equal(t, "192.168.1.42", wirelessAddress("wlan0 192.168.1.0/24 src 192.168.1.42\n"))
}

func TestActuateTapTextAndLifecycleUseADBVerbs(t *testing.T) { // [REQ:DVC-P0-011]
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Pointer: &strategy.PointerEvent{X: 12, Y: 34}}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Text: "hello world"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "stop", Package: "com.example.app"}))
	require.Contains(t, runner.calls[0], "shell input tap 12 34")
	require.Contains(t, runner.calls[1], "shell input text hello%sworld")
	require.Contains(t, runner.calls[2], "shell am force-stop com.example.app")
}

func TestInstallReusesPackageDataForUpdateMigration(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "install", Value: "/tmp/hello-mobile.apk"}))
	require.Contains(t, runner.calls[0], "install -r /tmp/hello-mobile.apk")
}

func TestActuateScreenrecordRequiresAndFillsOutputSink(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell screenrecord --time-limit 2 --bit-rate 1000000 /sdcard/device-control-proof.mp4": nil,
		"adb -s serial-1 exec-out cat /sdcard/device-control-proof.mp4":                                         []byte("ftyp-video"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	var output []byte
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "screenrecord", Output: &output}))
	require.Equal(t, []byte("ftyp-video"), output)
}

func TestActuateConformanceControlsUseBoundedAndroidVerbs(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "rotate", Value: "landscape"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "network", Value: "offline"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "network", Value: "online"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "deep-link", Value: "hello-mobile://home", Package: "com.example.hello"}))
	require.Contains(t, strings.Join(runner.calls, "\n"), "settings put system user_rotation 1")
	require.Contains(t, strings.Join(runner.calls, "\n"), "svc wifi disable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "svc data enable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "am start -a android.intent.action.VIEW -d hello-mobile://home -p com.example.hello")
}

func TestSemanticTargetResolvesAccessibilityBoundsAndTapsCenter(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cat /sdcard/window.xml": []byte(`<hierarchy><node resource-id="com.example:id/hello-mobile-input" bounds="[10,20][90,180]" /></hierarchy>`),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-target", Value: "hello-mobile-input"}))
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell input tap 50 100")
}

func TestSemanticAssertVerifiesAccessibilityText(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cat /sdcard/window.xml": []byte(`<hierarchy><node resource-id="com.example:id/state" text="Connectivity: offline" bounds="[0,0][100,40]" /></hierarchy>`),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: offline"}))
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: online"}), "expected text")
}

func TestPackageStateReportsExpectedInstallationState(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages com.example.hello": []byte("package:com.example.hello\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "present"}))
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "absent"}), "still installed")

	runner.responses["adb -s serial-1 shell pm list packages com.example.hello"] = nil
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "absent"}))
}
