package hostdesktop

import (
	"bytes"
	"context"
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
	require.NotContains(t, declaration.SupportedHostOS, "windows")
	require.NotEqual(t, strategy.StatusUnsupported, declaration.Status)
}

func TestHostDesktopRejectsUnverifiedWindowsPath(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "windows"
	declaration, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnsupported, declaration.Status)
	require.Contains(t, declaration.Reason, "unsupported")
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
			require.NoError(t, os.WriteFile(dataPath, tc.content, 0600))
			script := "#!/bin/sh\n/bin/cat '" + dataPath + "'\n" + tc.exit + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "import"), []byte(script), 0700))
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
