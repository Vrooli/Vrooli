package androidadb

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

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
