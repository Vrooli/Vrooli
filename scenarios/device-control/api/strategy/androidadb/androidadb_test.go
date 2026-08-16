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

func TestEnumerateKeysWirelessPhysicalDeviceByHardwareSerial(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\n192.168.1.179:34483\tdevice product:a03susq model:SM_A037U device:a03su\n"),
		"adb -s 192.168.1.179:34483 shell getprop ro.serialno":              []byte("R9TT608Q6MH\n"),
		"adb -s 192.168.1.179:34483 shell getprop ro.build.version.release": []byte("13\n"),
	}}
	devices, err := NewWithRunner(runner, "").Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "R9TT608Q6MH", devices[0].Serial)
	require.Equal(t, "192.168.1.179:34483", devices[0].Endpoint)
	require.Equal(t, "android-024665203bca17fa", devices[0].ID)
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

func TestRestoredWirelessDescribeUsesSelectedEndpointState(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\n192.168.1.42:5555\tdevice product:a03s model:SM_A037U\n"),
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

func TestLockStateParsesVendorKeyguardFormats(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "explicit legacy field", output: "mShowingLockscreen=true", want: "locked"},
		{name: "spaced explicit field", output: "isKeyguardShowing: false", want: "unlocked"},
		{name: "samsung showing state", output: "mShowingState=SHOWING", want: "locked"},
		{name: "samsung not showing state", output: "mShowingState: NOT_SHOWING", want: "unlocked"},
		{name: "delegate showing", output: "KeyguardDelegate: showing=false, occluded=false", want: "unlocked"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, lockState(test.output))
		})
	}
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

func TestShellQuotePreservesIntentExtraBoundaries(t *testing.T) {
	require.Equal(t, "'Hello Mobile share probe'", shellQuote("Hello Mobile share probe"))
	require.Equal(t, "'operator'\\''s note'", shellQuote("operator's note"))
}

func TestAttachWebViewMatchesPackageProcessAndAllocatesCDPForward(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pidof com.example.hello":                            []byte("4321\n"),
		"adb -s serial-1 shell cat /proc/net/unix":                                 []byte("Num RefCount Protocol Flags Type St Inode Path\n0001 2 0 10000 1 01 7 @webview_devtools_remote_4321\n"),
		"adb -s serial-1 forward tcp:0 localabstract:webview_devtools_remote_4321": []byte("tcp:43123\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	adapter.rendererID = func(context.Context, string) (string, error) { return "renderer-1", nil }
	endpoint, err := adapter.AttachWebView(context.Background(), "com.example.hello")
	require.NoError(t, err)
	require.Equal(t, strategy.WebViewEndpoint{Package: "com.example.hello", Socket: "webview_devtools_remote_4321", CDPEndpoint: "http://127.0.0.1:43123", RendererID: "renderer-1", Transport: "adb-forward"}, endpoint)
}

func TestWebViewSocketMatchingRejectsDifferentProcess(t *testing.T) {
	_, err := webViewSocketForPID([]byte("0001 2 0 10000 1 01 7 @webview_devtools_remote_9876\n"), "4321")
	require.ErrorContains(t, err, "matched process 4321")
}

func TestInstallReusesPackageDataForUpdateMigration(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "install", Value: "/tmp/hello-mobile.apk"}))
	require.Contains(t, runner.calls[0], "install -r /tmp/hello-mobile.apk")
}

func TestActuateScreenrecordRequiresAndFillsOutputSink(t *testing.T) {
	runner := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell dumpsys power":                          []byte("mStayOn=false\n"),
		"adb -s serial-1 shell settings get system screen_off_timeout": []byte("60000\n"),
		"adb -s serial-1 shell dumpsys deviceidle":                     []byte("mDeepEnabled=true\nmLightEnabled=true\n"),
	}}, process: &scriptedProcess{}}
	adapter := NewWithRunner(runner, "serial-1")
	handle, err := adapter.StartRecording(context.Background(), strategy.ClaimAnimation)
	require.NoError(t, err)
	runner.responses["adb -s serial-1 exec-out cat /sdcard/"+handle.ID+".mp4"] = []byte("ftyp-video")
	runner.responses["adb -s serial-1 shell ls /sdcard/"+handle.ID+".mp4"] = []byte(handle.ID)
	artifact, err := adapter.StopRecording(context.Background(), handle)
	require.NoError(t, err)
	require.Equal(t, []byte("ftyp-video"), artifact.Bytes)
	require.Equal(t, strategy.ClaimAnimation, artifact.ClaimClass)
	require.True(t, runner.process.signaled)
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell svc power stayon true")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell svc power stayon false")
	require.NotContains(t, strings.Join(runner.calls, "\n"), "shell input keyevent 224")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell settings put system screen_off_timeout 1800000")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell settings put system screen_off_timeout 60000")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell dumpsys deviceidle disable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell dumpsys deviceidle enable")
}

func TestActuateConformanceControlsUseBoundedAndroidVerbs(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "rotate", Value: "landscape"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "network", Value: "offline"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "network", Value: "online"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "deep-link", Value: "hello-mobile://home", Package: "com.example.hello"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "share", Value: "Hello Mobile share probe", Package: "com.example.hello"}))
	require.Contains(t, strings.Join(runner.calls, "\n"), "settings put system user_rotation 1")
	require.Contains(t, strings.Join(runner.calls, "\n"), "svc wifi disable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "svc data enable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "am start -a android.intent.action.VIEW -d hello-mobile://home -p com.example.hello")
	require.Contains(t, strings.Join(runner.calls, "\n"), "am start -a android.intent.action.SEND -c android.intent.category.DEFAULT -t text/plain -p com.example.hello --es android.intent.extra.TEXT 'Hello Mobile share probe'")
}

func TestActuateObservationClipboardAndLogcatVerbsUseExactADBArguments(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out screencap -p":               []byte("png"),
		"adb -s serial-1 shell cmd clipboard get":             []byte("copied text\n"),
		"adb -s serial-1 logcat -d -v epoch":                  []byte("08-15 12:00:00.000  1  1 I App: ready\n"),
		"adb -s serial-1 shell date +%s.%N":                   []byte("1755259200.125000000\n"),
		"adb -s serial-1 shell cmd clipboard set copied text": []byte(""),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	var screenshot, clipboard, logs, clock []byte
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "screenshot", Output: &screenshot}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "clipboard-write", Value: "copied text"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "clipboard-read", Output: &clipboard}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "logcat-start"}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "logcat-stop", Output: &logs}))
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "clock-sample", Output: &clock}))
	require.Equal(t, []byte("png"), screenshot)
	require.Equal(t, []byte("copied text\n"), clipboard)
	require.Contains(t, string(logs), "App: ready")
	require.Contains(t, string(clock), "1755259200.125000000")
	calls := strings.Join(runner.calls, "\n")
	require.Contains(t, calls, "exec-out screencap -p")
	require.Contains(t, calls, "shell cmd clipboard set copied text")
	require.Contains(t, calls, "shell cmd clipboard get")
	require.Contains(t, calls, "logcat -c")
	require.Contains(t, calls, "logcat -d -v epoch")
	require.Contains(t, calls, "shell date +%s.%N")
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

func indexOfCall(calls []string, fragment string) int {
	for i, call := range calls {
		if strings.Contains(call, fragment) {
			return i
		}
	}
	return len(calls)
}
