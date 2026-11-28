package androidadb

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

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

func TestScreenshotProbeIsReusedAcrossCloneAdaptersButOnlyOnSuccess(t *testing.T) {
	const screencap = "adb -s tv exec-out screencap -p"
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	t.Run("Given a decodable frame, when two clones probe, then the device is asked for a frame once", func(t *testing.T) {
		const endpointScreencap = "adb -s 192.168.1.158:5555 exec-out screencap -p"
		runner := &scriptedRunner{responses: map[string][]byte{screencap: frame.Bytes(), endpointScreencap: frame.Bytes()}}
		base := NewWithRunner(runner, "tv")
		for _, adapter := range []*Adapter{base, base.ForDevice("tv").(*Adapter), base.ForDevice("tv").(*Adapter), base.RestoreWireless("192.168.1.158:5555").(*Adapter)} {
			ok, _ := adapter.probeScreenshot(context.Background())
			require.True(t, ok)
		}
		require.Equal(t, 1, countCalls(runner, screencap), "clones of one serial share the probe")
		require.Equal(t, 1, countCalls(runner, endpointScreencap), "a different adb selector probes for itself")
	})
	t.Run("Given a failed probe, when probed again, then the device is asked again", func(t *testing.T) {
		runner := &scriptedRunner{responses: map[string][]byte{screencap: []byte("not an image")}}
		adapter := NewWithRunner(runner, "tv")
		for i := 0; i < 2; i++ {
			ok, _ := adapter.probeScreenshot(context.Background())
			require.False(t, ok)
		}
		require.Equal(t, 2, countCalls(runner, screencap))
	})
}
