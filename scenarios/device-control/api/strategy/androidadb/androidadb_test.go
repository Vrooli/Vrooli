package androidadb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestNewWithConfigUsesResolvedEnvironmentValues(t *testing.T) {
	adapter := NewWithConfig(Config{Serial: "usb:device-1", RequireDefaultServer: true})
	require.Equal(t, "usb:device-1", adapter.serial)
	runner, ok := adapter.runner.(commandRunner)
	require.True(t, ok)
	require.True(t, runner.requireDefaultADBServer)
}

type scriptedRunner struct {
	mu        sync.Mutex
	responses map[string][]byte
	errors    map[string]error
	calls     []string
}

func TestScreenControlUsesExplicitAndroidKeys(t *testing.T) {
	runner := &scriptedRunner{}
	adapter := NewWithRunner(runner, "serial")
	require.NoError(t, adapter.SetScreenState(context.Background(), "on"))
	require.NoError(t, adapter.SetScreenState(context.Background(), "off"))
	require.Error(t, adapter.SetScreenState(context.Background(), "toggle"))
	require.NoError(t, adapter.DismissKeyguard(context.Background()))
	require.NoError(t, adapter.GoHome(context.Background()))
	require.Equal(t, []string{"adb -s serial shell input keyevent KEYCODE_WAKEUP", "adb -s serial shell input keyevent KEYCODE_SLEEP", "adb -s serial shell wm dismiss-keyguard", "adb -s serial shell input keyevent KEYCODE_HOME"}, runner.calls)
}

func TestDescribeCaptureProbesOnlyRequestedModality(t *testing.T) {
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb devices -l":                                   []byte("List of devices attached\nserial device\n"),
		"adb -s serial exec-out screencap -p":              frame.Bytes(),
		"adb -s serial shell getprop ro.build.version.sdk": []byte("30\n"),
	}}
	adapter := NewWithRunner(runner, "serial")
	declaration, err := adapter.DescribeCapture(context.Background(), "screenshot")
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapScreenshot].Status)
	require.Equal(t, []string{"adb devices -l", "adb -s serial exec-out screencap -p"}, runner.calls)
	runner.calls = nil
	declaration, err = adapter.DescribeCapture(context.Background(), "recording")
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapNativeRecording].Status)
	require.Equal(t, []string{
		"adb devices -l",
		"adb -s serial shell screenrecord --help",
		"adb -s serial shell getprop ro.build.version.sdk",
		"adb -s serial shell pm path com.vrooli.devicecontrol.capture",
	}, runner.calls)
	require.Equal(t, strategy.StatusUnavailable, declaration.Capabilities[strategy.CapRecordingAudio].Status)
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

func TestApplicationLogsAreBoundedAndScoped(t *testing.T) {
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{"adb -s serial shell pidof com.example.app": []byte("123\n")}}, output: []byte("example log")}
	adapter := NewWithRunner(runner, "serial")
	output, err := adapter.ReadApplicationLogs(context.Background(), "com.example.app")
	require.NoError(t, err)
	require.Equal(t, []byte("example log"), output)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 500 --pid 123", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	_, err = adapter.ReadApplicationLogs(context.Background(), "--all")
	require.Error(t, err)
}

func TestScreenshotFallbackRefusesToInventFrames(t *testing.T) {
	adapter := NewWithRunner(&scriptedRunner{}, "serial")
	_, _, err := adapter.encodeScreenshotFallback(&recordingScreenshots{frames: [][]byte{[]byte("one frame")}, times: []time.Time{time.Now()}}, time.Now(), time.Second)
	require.ErrorContains(t, err, "fewer than two")
}

func TestScreenshotFallbackPreservesElapsedTime(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	var imageBytes bytes.Buffer
	require.NoError(t, png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	started := time.Now()
	samples := &recordingScreenshots{frames: [][]byte{imageBytes.Bytes(), imageBytes.Bytes(), imageBytes.Bytes()}, times: []time.Time{started.Add(100 * time.Millisecond), started.Add(time.Second), started.Add(3 * time.Second)}}
	video, count, err := NewWithRunner(&scriptedRunner{}, "serial").encodeScreenshotFallback(samples, started, 4*time.Second)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	path := filepath.Join(t.TempDir(), "fallback.mp4")
	require.NoError(t, os.WriteFile(path, video, 0o600))
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	require.NoError(t, err)
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	require.NoError(t, err)
	require.InDelta(t, 4, duration, 0.2)
}

func TestRecordingFallbackSamplesStayBoundedAndKeepTimeline(t *testing.T) {
	var samples recordingScreenshots
	start := time.Now()
	for i := 0; i < 5; i++ {
		samples.appendSample([]byte{byte(i), 1, 2, 3}, start.Add(time.Duration(i)*time.Second), 12)
	}
	require.LessOrEqual(t, samples.bytes, 12)
	require.Len(t, samples.frames, len(samples.times))
	require.Equal(t, start, samples.times[0])
	require.Equal(t, start.Add(4*time.Second), samples.times[len(samples.times)-1])
	require.Greater(t, samples.interval, 500*time.Millisecond)
	before := len(samples.frames)
	samples.appendSample(make([]byte, 13), start.Add(5*time.Second), 12)
	require.Len(t, samples.frames, before, "a frame larger than the entire budget cannot be retained")
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
		"adb -s serial-1 shell settings get global stay_on_while_plugged_in": []byte("2\n"),
		"adb -s serial-1 shell settings get system screen_off_timeout":       []byte("60000\n"),
		"adb -s serial-1 shell dumpsys deviceidle":                           []byte("mDeepEnabled=true\nmLightEnabled=true\n"),
	}}, process: &scriptedProcess{}}
	adapter := NewWithRunner(runner, "serial-1")
	handle, err := adapter.StartRecording(context.Background(), strategy.ClaimAnimation)
	require.NoError(t, err)
	runner.responses["adb -s serial-1 shell ps -A -o PID,ARGS"] = []byte("123 screenrecord --bit-rate 1000000 /data/local/tmp/" + handle.ID + ".mp4\n456 screenrecord /sdcard/other.mp4\n")
	runner.responses["adb -s serial-1 exec-out cat /data/local/tmp/"+handle.ID+".mp4"] = []byte("ftyp-video")
	runner.responses["adb -s serial-1 shell ls /data/local/tmp/"+handle.ID+".mp4"] = []byte(handle.ID)
	artifact, err := adapter.StopRecording(context.Background(), handle)
	require.NoError(t, err)
	require.Equal(t, []byte("ftyp-video"), artifact.Bytes)
	require.Equal(t, strategy.ClaimAnimation, artifact.ClaimClass)
	require.True(t, runner.process.signaled)
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell kill -INT 123")
	require.NotContains(t, strings.Join(runner.calls, "\n"), "pkill")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell svc power stayon true")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell settings put global stay_on_while_plugged_in 2")
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
	var validFrame bytes.Buffer
	require.NoError(t, png.Encode(&validFrame, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out screencap -p":               validFrame.Bytes(),
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
	require.Equal(t, validFrame.Bytes(), screenshot)
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

func TestSemanticSnapshotCapturesControlsAndRedactsEditableValues(t *testing.T) {
	tree := []byte(`<hierarchy><node class="android.widget.Button" resource-id="com.example:id/save" text="Save" clickable="true" enabled="true" bounds="[10,20][90,70]"/><node class="android.widget.EditText" text="secret" content-desc="private" password="true" enabled="true" bounds="[1,2][80,40]"/></hierarchy>`)
	snapshot, err := parseSemanticSnapshot(tree)
	require.NoError(t, err)
	require.Len(t, snapshot.Elements, 2)
	require.Equal(t, "Save", snapshot.Elements[0].Text)
	require.Equal(t, [4]int{10, 20, 90, 70}, snapshot.Elements[0].Bounds)
	require.True(t, snapshot.Elements[0].Clickable)
	require.True(t, snapshot.Elements[1].TextRedacted)
	require.Empty(t, snapshot.Elements[1].Text)
	require.Empty(t, snapshot.Elements[1].Description)
}

func TestSemanticSnapshotRetainsReportedSelectionAndCheckState(t *testing.T) {
	tree := []byte(`<hierarchy><node text="Selected" selected="true" checkable="true" checked="false" bounds="[0,0][10,10]"/><node text="Unknown" bounds="[0,10][10,20]"/><node text="Not checkable" checkable="false" checked="false" bounds="[0,20][10,30]"/></hierarchy>`)
	snapshot, err := parseSemanticSnapshot(tree)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Elements[0].Selected)
	require.True(t, *snapshot.Elements[0].Selected)
	require.NotNil(t, snapshot.Elements[0].Checked)
	require.False(t, *snapshot.Elements[0].Checked)
	require.Nil(t, snapshot.Elements[1].Selected)
	require.Nil(t, snapshot.Elements[1].Checked)
	require.Nil(t, snapshot.Elements[2].Checked)
}

func TestSemanticSnapshotBoundsOutputAndMarksTruncation(t *testing.T) {
	var tree strings.Builder
	tree.WriteString("<hierarchy>")
	for i := 0; i < maxSemanticSnapshotNodes+1; i++ {
		fmt.Fprintf(&tree, `<node text="%s" enabled="true" bounds="[0,0][10,10]"/>`, strings.Repeat("x", maxSemanticLabelRunes+1))
	}
	tree.WriteString("</hierarchy>")
	snapshot, err := parseSemanticSnapshot([]byte(tree.String()))
	require.NoError(t, err)
	require.True(t, snapshot.Truncated)
	require.Len(t, snapshot.Elements, maxSemanticSnapshotNodes)
	require.Len(t, []rune(snapshot.Elements[0].Text), maxSemanticLabelRunes)
	single := fmt.Sprintf(`<hierarchy><node text="%s" enabled="true" bounds="[0,0][10,10]"/></hierarchy>`, strings.Repeat("x", maxSemanticLabelRunes+1))
	snapshot, err = parseSemanticSnapshot([]byte(single))
	require.NoError(t, err)
	require.True(t, snapshot.Truncated, "truncated labels must be disclosed even if no nodes are omitted")
}

func TestSemanticSnapshotRefusesMissingOrMalformedHierarchy(t *testing.T) {
	for _, raw := range []string{"ERROR: could not get idle state.", `<hierarchy><node bounds="[0,0][1,1]">`} {
		_, err := parseSemanticSnapshot([]byte(raw))
		require.Error(t, err)
	}
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

type blockedStateRunner struct{}

func (blockedStateRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestAndroidStateObservationHonorsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := NewWithRunner(blockedStateRunner{}, "serial-1").ReadState(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func boolPtr(value bool) *bool { return &value }

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

func TestSemanticTargetRejectsAmbiguousMatches(t *testing.T) {
	tree := []byte(`<hierarchy><node text="Open" bounds="[0,0][20,20]"/><node text="Open" bounds="[0,30][20,50]"/></hierarchy>`)
	_, err := findSemanticNode(tree, "Open")
	require.ErrorContains(t, err, "ambiguous")
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
		"adb -s serial-1 shell getprop ro.build.version.sdk":               []byte("29"),
		"adb -s serial-1 shell getprop service.adb.tcp.port":               []byte("-1"),
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

func TestScreenshotRejectsTruncatedImageOnEveryPath(t *testing.T) {
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 40, 40))))
	truncated := encoded.Bytes()[:len(encoded.Bytes())/2]
	_, _, err := image.DecodeConfig(bytes.NewReader(truncated))
	require.NoError(t, err, "the fixture must contain a valid header")
	_, _, err = image.Decode(bytes.NewReader(truncated))
	require.Error(t, err, "the fixture must omit image data")
	runner := &scriptedRunner{responses: map[string][]byte{"adb -s serial exec-out screencap -p": truncated}}
	adapter := NewWithRunner(runner, "serial")
	_, err = adapter.Observe(context.Background())
	require.Error(t, err, "observe must reject an incomplete image")
	var output []byte
	err = adapter.Actuate(context.Background(), strategy.Actuation{Action: "screenshot", Output: &output})
	require.Error(t, err, "screenshot action must reject an incomplete image")
	require.Empty(t, output)
	available, _ := adapter.probeScreenshot(context.Background())
	require.False(t, available, "probe must not advertise corrupt captures as available")
}

func TestObserveReportsActualScreenshotFormat(t *testing.T) {
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 9, 7)), nil))
	runner := &scriptedRunner{responses: map[string][]byte{"adb -s serial exec-out screencap -p": append([]byte("firmware log\n"), encoded.Bytes()...)}}
	frame, err := NewWithRunner(runner, "serial").Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", frame.MediaType)
	require.Equal(t, 9, frame.Width)
	require.Equal(t, 7, frame.Height)
	require.Equal(t, encoded.Bytes(), frame.Bytes)
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

type cancelDuringRecordingSetup struct {
	*processRunner
	cancel             context.CancelFunc
	restoreHasDeadline bool
}

func (r *cancelDuringRecordingSetup) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.Contains(strings.Join(args, " "), "settings put global stay_on_while_plugged_in 2") {
		_, r.restoreHasDeadline = ctx.Deadline()
	}
	out, err := r.processRunner.Run(ctx, name, args...)
	if strings.Contains(strings.Join(args, " "), "svc power stayon true") {
		r.cancel()
		return out, ctx.Err()
	}
	return out, err
}

func TestRecordingStartHonorsCancellationAndRestoresSettings(t *testing.T) {
	base := &processRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
		"adb -s serial shell settings get global stay_on_while_plugged_in": []byte("2\n"),
		"adb -s serial shell settings get system screen_off_timeout":       []byte("60000\n"),
		"adb -s serial shell dumpsys deviceidle":                           []byte("mDeepEnabled=true\nmLightEnabled=false\n"),
	}}, process: &scriptedProcess{}}
	ctx, cancel := context.WithCancel(context.Background())
	runner := &cancelDuringRecordingSetup{processRunner: base, cancel: cancel}
	handle, err := NewWithRunner(runner, "serial").StartRecording(ctx, strategy.ClaimTransition)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, handle.ID)
	calls := strings.Join(base.calls, "\n")
	require.Contains(t, calls, "settings put global stay_on_while_plugged_in 2")
	require.Contains(t, calls, "settings put system screen_off_timeout 60000")
	require.NotContains(t, calls, "screenrecord")
	require.True(t, runner.restoreHasDeadline, "cleanup must have a bounded independent deadline")
}

func TestRecordingStartAlreadyCanceledHasNoDeviceEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &processRunner{scriptedRunner: &scriptedRunner{}, process: &scriptedProcess{}}
	_, err := NewWithRunner(runner, "serial").StartRecording(ctx, strategy.ClaimTransition)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, runner.calls)
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
				"adb -s serial-1 shell settings get global stay_on_while_plugged_in": []byte("2\n"),
				"adb -s serial-1 shell settings get system screen_off_timeout":       []byte("60000\n"),
				"adb -s serial-1 shell dumpsys deviceidle":                           []byte("mDeepEnabled=true\nmLightEnabled=false\n"),
			}, errors: map[string]error{}}, process: &scriptedProcess{}}
			adapter := NewWithRunner(runner, "serial-1")
			handle, err := adapter.StartRecording(context.Background(), strategy.ClaimTransition)
			require.NoError(t, err)
			read := "adb -s serial-1 exec-out cat /data/local/tmp/" + handle.ID + ".mp4"
			runner.mu.Lock()
			runner.responses["adb -s serial-1 shell ps -A -o PID,ARGS"] = []byte("123 screenrecord /data/local/tmp/" + handle.ID + ".mp4\n")
			runner.responses[read] = []byte("ftyp-video")
			if tc.failingCommand == "read" {
				runner.errors[read] = errors.New("injected read failure")
			} else {
				runner.errors["adb -s serial-1 shell settings put global stay_on_while_plugged_in 2"] = errors.New("injected restore failure")
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

func TestStopOwnedScreenrecordRejectsMissingAndAmbiguousProcesses(t *testing.T) {
	const ps = "adb -s serial shell ps -A -o PID,ARGS"
	const path = "/sdcard/owned.mp4"
	for _, tc := range []struct{ name, output string }{
		{"missing", "123 screenrecord /sdcard/other.mp4\n"},
		{"ambiguous", "123 screenrecord /sdcard/owned.mp4\n124 screenrecord /sdcard/owned.mp4\n"},
		{"different executable", "123 sh /sdcard/owned.mp4\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &scriptedRunner{responses: map[string][]byte{ps: []byte(tc.output)}}
			err := NewWithRunner(runner, "serial").stopOwnedScreenrecord(context.Background(), path)
			require.Error(t, err)
			require.NotContains(t, strings.Join(runner.calls, "\n"), "shell kill")
		})
	}
}

func TestRecordingProcessReportsBoundedEncoderFailure(t *testing.T) {
	process, err := (commandRunner{}).Start("sh", "-c", "printf 'encoder unavailable' >&2; exit 2")
	require.NoError(t, err)
	require.NoError(t, process.Start())
	require.ErrorContains(t, process.Wait(), "encoder unavailable")
}

func TestRecordingRetentionRestoresIndependentDozeAndUnsetSettings(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial shell dumpsys deviceidle":                           []byte("mLightEnabled=false mDeepEnabled=true\n"),
		"adb -s serial shell settings get global stay_on_while_plugged_in": []byte("null\n"),
		"adb -s serial shell settings get system screen_off_timeout":       []byte("null\n"),
	}}
	adapter := NewWithRunner(runner, "serial")
	state, err := adapter.readDozeEnabled(context.Background())
	require.NoError(t, err)
	require.True(t, state.deep)
	require.False(t, state.light)
	stay, err := adapter.readStayAwake(context.Background())
	require.NoError(t, err)
	timeout, err := adapter.readScreenOffTimeout(context.Background())
	require.NoError(t, err)
	require.NoError(t, adapter.restoreStayAwake(context.Background(), stay))
	require.NoError(t, adapter.setDisplayRetention(context.Background(), timeout, state))
	calls := strings.Join(runner.calls, "\n")
	require.Contains(t, calls, "settings delete global stay_on_while_plugged_in")
	require.Contains(t, calls, "settings delete system screen_off_timeout")
	require.Contains(t, calls, "dumpsys deviceidle enable deep")
	require.Contains(t, calls, "dumpsys deviceidle disable light")
}
