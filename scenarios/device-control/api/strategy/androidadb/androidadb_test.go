package androidadb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

type scriptedRunner struct {
	mu        sync.Mutex
	responses map[string][]byte
	errors    map[string]error
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

type frameworkLogRunner struct {
	*scriptedRunner
	output []byte
	call   string
	limit  int
	err    error
}

func (r *frameworkLogRunner) RunBounded(_ context.Context, name string, limit int, args ...string) ([]byte, error) {
	r.call = strings.Join(append([]string{name}, args...), " ")
	r.limit = limit
	return r.output, r.err
}

func TestFrameworkLogsFiltersAndSummarizesWithoutRawContent(t *testing.T) {
	component := "com.example/.service.Home"
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte(strings.Join([]string{
		"W ActivityManager: Permission Denial com.example/.service.Home private-secret",
		"W ActivityManager: Unable to start service com.example/com.example.service.Home",
		"W ActivityManager: com.example/.service.HomeExtra private-secret",
		"W ActivityManager: xcom.example/.service.Home private-secret",
		"W ActivityManager: com.other/.service.Home private-secret",
	}, "\n"))}
	var output []byte
	err := NewWithRunner(runner, "serial").Actuate(context.Background(), strategy.Actuation{Action: "framework-logs", Component: component, LogTag: "ActivityManager", Output: &output})
	require.NoError(t, err)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 250 -s ActivityManager:W", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	require.NotContains(t, string(output), "private-secret")
	var summary struct {
		Matching int            `json:"matching_line_count"`
		Priority string         `json:"priority"`
		Classes  map[string]int `json:"diagnostic_classes"`
	}
	require.NoError(t, json.Unmarshal(output, &summary))
	require.Equal(t, 2, summary.Matching)
	require.Equal(t, "W", summary.Priority, "omitted priority keeps the warning-level default")
	require.Equal(t, 1, summary.Classes["permission_denial"])
	require.Equal(t, 1, summary.Classes["service_start_failure"])
}

func TestFrameworkLogsInfoMinimumIncludesHigherLevelsAndStaysRedacted(t *testing.T) {
	component := "com.example/.service.Home"
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte(strings.Join([]string{
		"I ActivityManager: Binding service com.example/.service.Home private-secret",
		"W ActivityManager: Unable to start service com.example/com.example.service.Home private-secret",
		"I ActivityManager: Binding service com.example/.service.HomeExtra private-secret",
	}, "\n"))}
	var output []byte
	err := NewWithRunner(runner, "serial").Actuate(context.Background(), strategy.Actuation{
		Action: "framework-logs", Component: component, LogTag: "ActivityManager", LogPriority: "I", Output: &output,
	})
	require.NoError(t, err)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 250 -s ActivityManager:I", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	require.NotContains(t, string(output), "private-secret", "raw log text must not enter persisted evidence")
	var summary struct {
		Component string `json:"component"`
		Tag       string `json:"tag"`
		Priority  string `json:"priority"`
		Matching  int    `json:"matching_line_count"`
	}
	require.NoError(t, json.Unmarshal(output, &summary))
	require.Equal(t, component, summary.Component)
	require.Equal(t, "ActivityManager", summary.Tag)
	require.Equal(t, "I", summary.Priority)
	require.Equal(t, 2, summary.Matching, "the INFO minimum also includes higher-priority warnings")
}

func TestFrameworkLogsRejectsInvalidFiltersAndReportsNoMatch(t *testing.T) {
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte("unrelated line")}
	adapter := NewWithRunner(runner, "serial")
	for _, event := range []strategy.Actuation{
		{Action: "framework-logs", Component: "com.example/.Home;evil", LogTag: "ActivityManager"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "OtherTag"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "ActivityManager", LogPriority: "D"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "ActivityManager", LogPriority: "i"},
	} {
		require.Error(t, adapter.Actuate(context.Background(), event))
		require.Empty(t, runner.call)
	}
	var output []byte
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "framework-logs", Component: "com.example/.Home", LogTag: "AccessibilityManagerService", Output: &output}))
	require.Contains(t, string(output), "inconclusive_no_matching_component_lines")
}

func TestBoundedCommandRejectsOversizedOutputAndCancellation(t *testing.T) {
	if os.Getenv("DEVICE_CONTROL_BOUNDED_HELPER") == "output" {
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), 1000))
		return
	}
	if os.Getenv("DEVICE_CONTROL_BOUNDED_HELPER") == "block" {
		select {}
	}
	t.Setenv("DEVICE_CONTROL_BOUNDED_HELPER", "output")
	_, err := (commandRunner{}).RunBounded(context.Background(), os.Args[0], 100, "-test.run=^TestBoundedCommandRejectsOversizedOutputAndCancellation$")
	require.ErrorContains(t, err, "exceeded configured limit")
	t.Setenv("DEVICE_CONTROL_BOUNDED_HELPER", "block")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = (commandRunner{}).RunBounded(ctx, os.Args[0], 100, "-test.run=^TestBoundedCommandRejectsOversizedOutputAndCancellation$")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func (r *processRunner) Start(string, ...string) (Process, error) { return r.process, nil }

func TestCommandRunnerRunReportsContextCancellation(t *testing.T) {
	if os.Getenv("DEVICE_CONTROL_COMMAND_RUNNER_BLOCK") == "1" {
		select {}
	}

	t.Setenv("DEVICE_CONTROL_COMMAND_RUNNER_BLOCK", "1")
	tests := []struct {
		name    string
		context func() (context.Context, func(), error)
	}{
		{
			name: "deadline",
			context: func() (context.Context, func(), error) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				return ctx, cancel, context.DeadlineExceeded
			},
		},
		{
			name: "cancel",
			context: func() (context.Context, func(), error) {
				ctx, cancel := context.WithCancel(context.Background())
				timer := time.AfterFunc(100*time.Millisecond, cancel)
				return ctx, func() { timer.Stop(); cancel() }, context.Canceled
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel, want := tt.context()
			defer cancel()
			_, err := (commandRunner{}).Run(ctx, os.Args[0], "-test.run=^TestCommandRunnerRunReportsContextCancellation$")
			require.ErrorIs(t, err, want)
		})
	}
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, call)
	if err := r.errors[call]; err != nil {
		return r.responses[call], err
	}
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

func TestDescribeUsesConnectivityFallbackForNetworkControl(t *testing.T) {
	runner := &scriptedRunner{
		responses: map[string][]byte{
			"adb devices -l": []byte("List of devices attached\nemulator-5554\tdevice product:sdk_gphone model:sdk_gphone64_x86_64 device:generic_x86_64\n"),
		},
		errors: map[string]error{
			"adb -s emulator-5554 shell svc wifi": errors.New("usage: svc wifi [enable|disable]"),
		},
	}
	declaration, err := NewWithRunner(runner, "emulator-5554").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapNetworkControl].Status)
	require.Contains(t, runner.calls, "adb -s emulator-5554 shell cmd connectivity airplane-mode")
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
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell settings get system user_rotation":          []byte("null"),
		"adb -s serial-1 shell settings get system accelerometer_rotation": []byte("1"),
	}}
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

func TestRotationRestorationPreservesAbsentSettings(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell settings get system user_rotation":          []byte("null"),
		"adb -s serial-1 shell settings get system accelerometer_rotation": []byte("1"),
		"adb -s serial-1 shell dumpsys input":                              []byte("mCurrentRotation=0"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	state, err := adapter.ReadState(context.Background())
	require.NoError(t, err)
	require.NoError(t, adapter.RestoreState(context.Background(), state))
	require.Contains(t, runner.calls, "adb -s serial-1 shell settings delete system user_rotation")
	require.Contains(t, runner.calls, "adb -s serial-1 shell settings put system accelerometer_rotation 1")
	require.NotContains(t, runner.calls, "adb -s serial-1 shell settings put system user_rotation 0")
}

func TestKeyguardCleanupDoesNotWriteRotation(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	require.NoError(t, NewWithRunner(runner, "serial-1").RestoreState(context.Background(), strategy.DeviceState{}))
	require.Empty(t, runner.calls)
}

func TestRotateRefusesUnknownRecoverySettings(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	err := NewWithRunner(runner, "serial-1").Actuate(context.Background(), strategy.Actuation{Action: "rotate", Value: "landscape"})
	require.ErrorContains(t, err, "exact raw settings")
	require.NotContains(t, strings.Join(runner.calls, "\n"), "settings put")
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
		"adb -s serial-1 exec-out uiautomator dump /dev/stdout": []byte(`<hierarchy><node resource-id="com.example:id/hello-mobile-input" bounds="[10,20][90,180]" /></hierarchy>`),
		"adb -s serial-1 exec-out screencap -p":                 screenshot.Bytes(),
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
		"adb -s serial-1 exec-out uiautomator dump /dev/stdout": []byte(`<hierarchy><node resource-id="com.example:id/state" text="Connectivity: offline" bounds="[0,0][100,40]" /></hierarchy>`),
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
		"adb -s serial-1 shell dumpsys dreams":                             []byte("mCurrentDreamToken=null"),
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

func TestSanitizeCapturedImageStripsFirmwareLogPreamble(t *testing.T) {
	var clean bytes.Buffer
	require.NoError(t, png.Encode(&clean, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	pngBytes := clean.Bytes()

	var jpg bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
	jpgBytes := jpg.Bytes()

	cases := []struct {
		name    string
		raw     []byte
		want    []byte
		decodes bool
	}{
		{name: "clean png passes through", raw: pngBytes, want: pngBytes, decodes: true},
		{
			name:    "smarttv log preamble stripped",
			raw:     append([]byte("Init wrapper sys mutex successful. Pid:14180\nopen mma dev failed\n"), pngBytes...),
			want:    pngBytes,
			decodes: true,
		},
		{name: "jpeg preamble stripped", raw: append([]byte("noise\n"), jpgBytes...), want: jpgBytes, decodes: true},
		{name: "non-image returned unchanged", raw: []byte("not an image"), want: []byte("not an image"), decodes: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeCapturedImage(tc.raw)
			require.Equal(t, tc.want, got)
			_, _, err := image.DecodeConfig(bytes.NewReader(got))
			if tc.decodes {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestProbeScreenshotRequiresDecodableFrame(t *testing.T) {
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	preambled := append([]byte("Init wrapper sys mutex successful\n"), frame.Bytes()...)

	// A firmware log preamble must still declare screenshot available.
	okRunner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\nemulator-5554\tdevice model:sdk\n"),
		"adb -s emulator-5554 exec-out screencap -p": preambled,
	}}
	decl, err := NewWithRunner(okRunner, "emulator-5554").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, decl.Capabilities[strategy.CapScreenshot].Status)

	// A device that returns non-image bytes must declare screenshot unavailable
	// without hiding the other probed capabilities.
	badRunner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\nemulator-5554\tdevice model:sdk\n"),
		"adb -s emulator-5554 exec-out screencap -p":  []byte("no display surface"),
		"adb -s emulator-5554 shell pm list packages": []byte("package:com.example\n"),
	}}
	decl, err = NewWithRunner(badRunner, "emulator-5554").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, decl.Capabilities[strategy.CapScreenshot].Status)
	require.Equal(t, strategy.StatusAvailable, decl.Capabilities[strategy.CapAppLifecycle].Status)
}

func TestAppLifecyclePackageStateVerifiesInstallation(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages org.smarttube.stable": []byte("package:org.smarttube.stable\n"),
		"adb -s serial-1 shell dumpsys package org.smarttube.stable":  []byte("Packages:\n  Package [org.smarttube.stable]\n    versionName=25.10\n"),
		"adb -s serial-1 shell pm list packages com.example.absent":   []byte(""),
		"adb -s serial-1 shell pm list packages org.smart":            []byte("package:org.smarttube.stable\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")

	present, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.True(t, present.Installed)
	require.Equal(t, "25.10", present.Version)
	require.Equal(t, strategy.StatusAvailable, present.Status)

	absent, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "com.example.absent"})
	require.NoError(t, err)
	require.False(t, absent.Installed)
	require.Equal(t, strategy.StatusAvailable, absent.Status)

	// A prefix collision must not report a shorter package present.
	prefix, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "org.smart"})
	require.NoError(t, err)
	require.False(t, prefix.Installed)
}

func TestLaunchUsesTVActivityAndReportsObservedForeground(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p org.smarttube.stable": []byte("org.smarttube.stable/.MainActivity\n"),
		"adb -s serial-1 shell am start -W -n org.smarttube.stable/.MainActivity":                                                                                       []byte("Status: ok\nActivity: org.smarttube.stable/.MainActivity\n"),
		"adb -s serial-1 shell dumpsys window": []byte("mCurrentFocus=Window{abc u0 org.smarttube.stable/.MainActivity}"),
	}}
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "launch", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.Equal(t, "foreground_observed", result.Outcome)
	require.Equal(t, "org.smarttube.stable", result.ForegroundPackage)
	require.Contains(t, result.Diagnostic, "Status: ok")
	require.False(t, result.ObservedAt.IsZero())
	require.NotContains(t, strings.Join(runner.calls, "\n"), "monkey")
}

func TestLaunchFallsBackToOrdinaryLauncherAndRejectsZeroExitError(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p com.android.tv.settings": []byte("No activity found"),
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -p com.android.tv.settings":          []byte("com.android.tv.settings/.MainSettings"),
		"adb -s serial-1 shell am start -W -n com.android.tv.settings/.MainSettings":                                                                                       []byte("Error: Activity not started, permission denied"),
	}}
	err := NewWithRunner(runner, "serial-1").Actuate(context.Background(), strategy.Actuation{Action: "launch", Package: "com.android.tv.settings"})
	require.ErrorContains(t, err, "permission denied")
}

func TestLaunchDoesNotClaimForegroundFromAcceptanceAlone(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p org.smarttube.stable": []byte("org.smarttube.stable/.MainActivity"),
		"adb -s serial-1 shell am start -W -n org.smarttube.stable/.MainActivity":                                                                                       []byte("Status: ok"),
		"adb -s serial-1 shell dumpsys window": []byte("mCurrentFocus=Window{abc u0 com.google.android.apps.tv.dreamx/.DreamActivity}"),
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(ctx, strategy.AppLifecycleRequest{Operation: "launch", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.Equal(t, "accepted", result.Outcome)
	require.Equal(t, "com.google.android.apps.tv.dreamx", result.ForegroundPackage)
}

func TestCapabilityProbeKeepsIndependentCapabilitiesAfterScreenshotDecodeFailure(t *testing.T) { // [REQ:DVC-P0-011]
	// A device whose screencap returns non-image bytes must report screenshot
	// unavailable while every independently-probed capability stays available.
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l": []byte("List of devices attached\nemulator-5554\tdevice model:sdk\n"),
		"adb -s emulator-5554 exec-out screencap -p":     []byte("no display surface here"),
		"adb -s emulator-5554 shell input --help":        []byte("usage: input"),
		"adb -s emulator-5554 shell uiautomator help":    []byte("usage: uiautomator"),
		"adb -s emulator-5554 shell pm list packages":    []byte("package:com.example\n"),
		"adb -s emulator-5554 shell wm size":             []byte("Physical size: 1920x1080"),
		"adb -s emulator-5554 shell cmd clipboard help":  []byte("clipboard"),
		"adb -s emulator-5554 logcat -d -t 1":            []byte("log"),
		"adb -s emulator-5554 shell screenrecord --help": []byte("usage: screenrecord"),
	}}
	decl, err := NewWithRunner(runner, "emulator-5554").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, decl.Capabilities[strategy.CapScreenshot].Status)
	for _, cap := range []string{strategy.CapInput, strategy.CapSemanticTree, strategy.CapAppLifecycle, strategy.CapPermissions, strategy.CapOrientation, strategy.CapDeviceLogs} {
		require.Equalf(t, strategy.StatusAvailable, decl.Capabilities[cap].Status, "capability %q must survive a screenshot decode failure", cap)
	}
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

func indexOfCall(calls []string, fragment string) int {
	for i, call := range calls {
		if strings.Contains(call, fragment) {
			return i
		}
	}
	return len(calls)
}

func TestRestartUsesOnlyExactBoundTransportAndReturnsAcceptance(t *testing.T) {
	runner := &scriptedRunner{}
	require.Error(t, NewWithRunner(runner, "").RestartDevice(context.Background()))
	require.Empty(t, runner.calls)
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.RestartDevice(context.Background()))
	require.Equal(t, []string{"adb -s serial-1 reboot"}, runner.calls)
	runner.errors = map[string]error{"adb -s serial-1 reboot": errors.New("transport refused")}
	require.ErrorContains(t, adapter.RestartDevice(context.Background()), "transport refused")
	require.Len(t, runner.calls, 2, "owner must not retry a restart with uncertain effect")
}

func TestRecordingStopReportsArtifactAndRestoreSeparately(t *testing.T) {
	for _, tc := range []struct {
		name, failingCommand string
		artifactAvailable    bool
		want                 string
	}{
		{name: "artifact read failed", failingCommand: "read", want: "read screenrecord artifact"},
		{name: "restore failed after artifact", failingCommand: "restore", artifactAvailable: true, want: "restore display keep-awake state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
				"adb -s serial-1 shell dumpsys power":                          []byte("mStayOn=false\n"),
				"adb -s serial-1 shell settings get system screen_off_timeout": []byte("60000\n"),
				"adb -s serial-1 shell dumpsys deviceidle":                     []byte("mDeepEnabled=true\n"),
			}, errors: map[string]error{}}, process: &scriptedProcess{}}
			adapter := NewWithRunner(runner, "serial-1")
			handle, err := adapter.StartRecording(context.Background(), strategy.ClaimTransition)
			require.NoError(t, err)
			read := "adb -s serial-1 exec-out cat /sdcard/" + handle.ID + ".mp4"
			runner.mu.Lock()
			runner.responses[read] = []byte("ftyp-video")
			if tc.failingCommand == "read" {
				runner.errors[read] = errors.New("injected read failure")
			} else {
				runner.errors["adb -s serial-1 shell svc power stayon false"] = errors.New("injected restore failure")
			}
			runner.mu.Unlock()
			artifact, err := adapter.StopRecording(context.Background(), handle)
			require.ErrorContains(t, err, tc.want)
			require.True(t, runner.process.signaled, "host wrapper was signaled; this alone does not prove remote encoder termination")
			if tc.artifactAvailable {
				require.Equal(t, []byte("ftyp-video"), artifact.Bytes)
			} else {
				require.Empty(t, artifact.Bytes)
			}
			runner.mu.Lock()
			calls := strings.Join(runner.calls, "\n")
			runner.mu.Unlock()
			require.Contains(t, calls, "shell settings put system screen_off_timeout 60000", "restoration is attempted even after an artifact error")
			_, err = adapter.StopRecording(context.Background(), handle)
			require.ErrorContains(t, err, "not active", "native stop is one-shot")
		})
	}
}
