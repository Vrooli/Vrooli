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

type scriptedRunner struct {
	responses map[string][]byte
	calls     []string
}

type scriptedProcess struct{ signaled bool }

func (p *scriptedProcess) Start() error     { return nil }
func (p *scriptedProcess) Interrupt() error { p.signaled = true; return nil }
func (p *scriptedProcess) Kill() error      { p.signaled = true; return nil }
func (p *scriptedProcess) Wait() error      { return nil }

type processRunner struct {
	*scriptedRunner
	process *scriptedProcess
}

func (r *processRunner) Start(string, ...string) (Process, error) { return r.process, nil }

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
	runner := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{}}, process: &scriptedProcess{}}
	adapter := NewWithRunner(runner, "serial-1")
	handle, err := adapter.StartRecording(context.Background(), strategy.ClaimAnimation)
	require.NoError(t, err)
	runner.responses["adb -s serial-1 exec-out cat /sdcard/"+handle.ID+".mp4"] = []byte("ftyp-video")
	runner.responses["adb -s serial-1 shell ls /sdcard/"+handle.ID+".mp4"] = []byte(handle.ID)
	artifact, err := adapter.StopRecording(context.Background(), handle)
	require.NoError(t, err)
	require.Equal(t, []byte("ftyp-video"), artifact.Bytes)
	require.Equal(t, strategy.ClaimAnimation, artifact.ClaimClass)
	require.False(t, runner.process.signaled)
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
	var screenshot bytes.Buffer
	require.NoError(t, png.Encode(&screenshot, image.NewRGBA(image.Rect(0, 0, 100, 200))))
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out uiautomator dump /dev/tty": []byte(`<hierarchy><node resource-id="com.example:id/hello-mobile-input" bounds="[10,20][90,180]" /></hierarchy>`),
		"adb -s serial-1 exec-out screencap -p":              screenshot.Bytes(),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	match, err := adapter.ResolveSemantic(context.Background(), "hello-mobile-input")
	require.NoError(t, err)
	require.Equal(t, []float64{.1, .1, .9, .9}, match.Bounds)
	require.Equal(t, 1.0, match.Confidence)
	require.NotContains(t, strings.Join(runner.calls, "\n"), "/sdcard/")
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-target", Value: "hello-mobile-input"}), "resolver-scoped")
}

func TestSemanticAssertVerifiesAccessibilityText(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out uiautomator dump /dev/tty": []byte(`<hierarchy><node resource-id="com.example:id/state" text="Connectivity: offline" bounds="[0,0][100,40]" /></hierarchy>`),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: offline"}))
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: online"}), "expected text")
}

func TestReadStateParsesRecordedAndroidProbes(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell dumpsys activity activities":                []byte("mResumedActivity: ActivityRecord{u0 com.example.hello/.MainActivity t42}"),
		"adb -s serial-1 shell dumpsys power":                              []byte("mWakefulness=Awake\nDisplay Power: state=ON"),
		"adb -s serial-1 shell dumpsys window":                             []byte("mShowingLockscreen=false isStatusBarKeyguard=false"),
		"adb -s serial-1 shell dumpsys input":                              []byte("mCurrentRotation=3"),
		"adb -s serial-1 shell settings get system user_rotation":          []byte("3"),
		"adb -s serial-1 shell settings get system accelerometer_rotation": []byte("1"),
		"adb -s serial-1 shell dumpsys battery":                            []byte("level: 87\nAC powered: true"),
		"adb -s serial-1 shell dumpsys thermalservice":                     []byte("Thermal Status: nominal"),
		"adb -s serial-1 shell wm size":                                    []byte("Physical size: 1080x2400"),
		"adb -s serial-1 shell wm density":                                 []byte("Physical density: 420"),
	}}
	state, err := NewWithRunner(runner, "serial-1").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "com.example.hello", state.ForegroundPackage)
	require.Equal(t, "on", state.ScreenState)
	require.Equal(t, "unlocked", state.LockState)
	require.Equal(t, "reverse-landscape", state.Orientation)
	require.Equal(t, 87, state.BatteryLevel)
	require.True(t, state.Charging)
	require.Equal(t, "nominal", state.ThermalStatus)
	require.Equal(t, 1080, state.DisplayWidth)
	require.Equal(t, 2400, state.DisplayHeight)
	require.Equal(t, 420, state.DisplayDensity)
	require.Empty(t, state.Unavailable)
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
