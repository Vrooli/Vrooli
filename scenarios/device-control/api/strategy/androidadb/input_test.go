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
