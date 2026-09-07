package hostdesktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestHostDesktopStrategyCapabilityTier(t *testing.T) { // [REQ:DVC-P1-004]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	t.Setenv("DISPLAY", "")
	declaration, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Contains(t, declaration.SupportedHostOS, "darwin")
	require.Contains(t, declaration.SupportedHostOS, "windows")
	require.NotEqual(t, strategy.StatusUnsupported, declaration.Status)
}

func TestHostDesktopEnumeratesCapturedLocalTarget(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-01]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	t.Setenv("DISPLAY", ":test")
	t.Setenv("VROOLI_NODE_ID", "node-1")
	t.Setenv("XDG_SESSION_ID", "session-1")
	a := New()
	a.capture = func(context.Context) (strategy.Frame, error) {
		return strategy.Frame{Width: 8, Height: 6, Bytes: []byte("captured")}, nil
	}
	devices, err := a.Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "host-desktop", devices[0].ID)
	require.Equal(t, "Local desktop", devices[0].Name)
	require.Equal(t, "node-1:session-1", devices[0].Serial)
	require.Equal(t, "local", devices[0].Transport)
	require.Equal(t, strategy.StatusAvailable, devices[0].Health)
}

func TestHostDesktopUsesManagedHelperDisplayBinding(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	t.Setenv("DISPLAY", "")
	t.Setenv("XAUTHORITY", "")
	configPath := filepath.Join(t.TempDir(), "helper.json")
	config := []byte(`{"display":0,"xauthority_file":"/run/user/1000/gdm/Xauthority"}`)
	require.NoError(t, os.WriteFile(configPath, config, 0o600))
	t.Setenv("DEVICE_CONTROL_DESKTOP_HELPER_CONFIG", configPath)

	a := New()
	require.Equal(t, ":0", a.display)
	require.Equal(t, "/run/user/1000/gdm/Xauthority", a.xauthority)
}

func TestHostDesktopIgnoresHelperConfigWithoutDisplayBinding(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	t.Setenv("DISPLAY", "")
	t.Setenv("XAUTHORITY", "")
	configPath := filepath.Join(t.TempDir(), "helper.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"xauthority_file":"/run/user/1000/gdm/Xauthority"}`), 0o600))
	t.Setenv("DEVICE_CONTROL_DESKTOP_HELPER_CONFIG", configPath)

	a := New()
	require.Empty(t, a.display)
	require.Equal(t, "/run/user/1000/gdm/Xauthority", a.xauthority)
}

func TestHostDesktopWindowsCaptureUsesBoundedNativeProbe(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-03]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "windows"
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 4, 5))))
	a := New()
	a.command = func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		require.NotEmpty(t, args)
		return []byte("  " + base64.StdEncoding.EncodeToString(pixels.Bytes()) + "\n"), nil
	}
	declaration, err := a.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Status)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapScreenshot].Status)
	require.Equal(t, strategy.StatusUnavailable, declaration.Capabilities[strategy.CapSemanticTree].Status)
	require.Contains(t, declaration.Capabilities[strategy.CapSemanticTree].Reason, "accessibility adapter")
	require.Contains(t, declaration.Capabilities[strategy.CapScreenshot].ProbeEvidence, "decoded PNG 4x5")
}

func TestHostDesktopActuatesKeyboardAndTextAcrossHosts(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-08]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	for _, host := range []string{"linux", "darwin", "windows"} {
		t.Run(host, func(t *testing.T) {
			strategy.HostOS = host
			a := New()
			a.display = ":test"
			var calls [][]string
			a.run = func(_ context.Context, name string, args ...string) error {
				calls = append(calls, append([]string{name}, args...))
				return nil
			}
			require.NoError(t, a.Actuate(context.Background(), strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: "CTRL+L"}}))
			require.NoError(t, a.Actuate(context.Background(), strategy.Actuation{Text: "hello 世界"}))
			require.Len(t, calls, 2)
			if host == "linux" {
				require.Equal(t, []string{"xdotool", "key", "CTRL+L"}, calls[0])
				require.Equal(t, []string{"xdotool", "type", "--delay", "0", "--", "hello 世界"}, calls[1])
			} else if host == "darwin" {
				require.Equal(t, "osascript", calls[0][0])
				require.Contains(t, calls[1][2], "hello 世界")
			} else {
				require.Equal(t, "powershell.exe", calls[0][0])
				require.Equal(t, "powershell.exe", calls[1][0])
			}
		})
	}
}

func TestCaptureReadinessRequiresDecodedPixels(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-01]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("DISPLAY", ":test")
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 2, 3))))
	good := append([]byte(nil), pixels.Bytes()...)
	for _, tc := range []struct {
		name    string
		content []byte
		exit    string
		ready   bool
	}{
		{"permission denied", nil, "exit 1", false},
		{"valid header with missing pixels", good[:33], "exit 0", false},
		{"corrupt pixels", []byte("not a screenshot"), "exit 0", false},
		{"actual capture", good, "exit 0", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The executable exists in every case; only successful full image decoding
			// may produce capture readiness. No real desktop is accessed by this test.
			dataPath := filepath.Join(dir, "frame.png")
			require.NoError(t, os.WriteFile(dataPath, tc.content, 0o600))
			script := "#!/bin/sh\n/bin/cat '" + dataPath + "'\n" + tc.exit + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "import"), []byte(script), 0o700))
			d, err := New().Describe(context.Background())
			require.NoError(t, err)
			if tc.ready {
				require.Equal(t, strategy.StatusAvailable, d.Capabilities[strategy.CapScreenshot].Status)
				require.Contains(t, d.Capabilities[strategy.CapScreenshot].ProbeEvidence, "decoded PNG 2x3")
			} else {
				require.Equal(t, strategy.StatusUnavailable, d.Capabilities[strategy.CapScreenshot].Status)
				require.Empty(t, d.Capabilities[strategy.CapScreenshot].ProbeEvidence)
			}
			require.Equal(t, strategy.StatusUnavailable, d.Capabilities[strategy.CapInput].Status)
			require.NotContains(t, d.Tiers, "observer")
		})
	}
}

func TestCaptureReadinessFallsBackToXWDWhenImportIsUnavailable(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-01]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	dir := t.TempDir()
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("DISPLAY", ":fallback")
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 7, 4))))
	pngPath := filepath.Join(dir, "frame.png")
	require.NoError(t, os.WriteFile(pngPath, pixels.Bytes(), 0o600))
	writeExecutable := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o700))
	}
	// Force the ImageMagick probe to fail, then provide the standard X11/ffmpeg
	// pipeline. The scripts stand in for the native tools without touching a
	// real display, while preserving the same process and pipe boundaries.
	writeExecutable("import", "exit 1")
	writeExecutable("xwd", "printf xwd-frame")
	writeExecutable("ffmpeg", "/bin/cat >/dev/null\n/bin/cat '"+pngPath+"'")

	declaration, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapScreenshot].Status)
	require.Contains(t, declaration.Capabilities[strategy.CapScreenshot].ProbeEvidence, "decoded PNG 7x4")
}

func TestCaptureProbeHonorsCancellation(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	a := New()
	a.capture = func(ctx context.Context) (strategy.Frame, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), captureTimeout)
		<-ctx.Done()
		return strategy.Frame{}, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err := a.Describe(ctx)
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, d.Capabilities[strategy.CapScreenshot].Status)
}

func TestCaptureOutputHasMemoryBound(t *testing.T) {
	var output boundedCapture
	_, err := output.Write(make([]byte, maxCaptureBytes+1))
	require.Error(t, err)
	require.Zero(t, output.Len())
}

func TestWaylandSessionDoesNotFallbackToX11(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", ":99")
	d, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, d.Capabilities[strategy.CapScreenshot].Status)
	require.Contains(t, d.Capabilities[strategy.CapScreenshot].Reason, "portal-backed")
	require.Equal(t, strategy.StatusUnavailable, d.Capabilities[strategy.CapSemanticTree].Status)
	_, err = New().Observe(context.Background())
	require.Error(t, err)
	var availability *strategy.AvailabilityError
	require.ErrorAs(t, err, &availability)
	require.Contains(t, availability.Reason, "portal-backed")
}
