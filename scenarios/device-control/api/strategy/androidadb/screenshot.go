package androidadb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/strategy"
)

// screencapImageSignatures are the encoded-frame prefixes device-control
// accepts from `adb screencap -p`. Some Android and Android-TV firmware images
// print startup log lines to stdout before the encoded frame — observed on a
// SmartTV 4K where 65 bytes ("Init wrapper sys mutex successful...\nopen mma
// dev failed\n") precede the PNG signature. Left unstripped, that preamble
// makes an otherwise valid frame decode as "image: unknown format".
var screencapImageSignatures = [][]byte{
	{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, // PNG
	{0xff, 0xd8, 0xff},                            // JPEG
}

// sanitizeCapturedImage returns the byte slice starting at the first decodable
// image signature. It fast-paths a frame that already decodes at offset 0 and,
// when a firmware log preamble is present, trims to the earliest signature that
// yields a valid image header. Genuinely corrupt output is returned unchanged
// so the caller's decode error still surfaces the real failure.
func sanitizeCapturedImage(raw []byte) []byte {
	if _, _, err := image.DecodeConfig(bytes.NewReader(raw)); err == nil {
		return raw
	}
	for _, sig := range screencapImageSignatures {
		if idx := bytes.Index(raw, sig); idx > 0 {
			if _, _, err := image.DecodeConfig(bytes.NewReader(raw[idx:])); err == nil {
				return raw[idx:]
			}
		}
	}
	return raw
}

// validateCapturedImage checks all compressed image data, not merely the header.
// Bound decoded size before allocation so a malformed device response cannot
// exhaust the service while advertising a plausible image header.
func validateCapturedImage(raw []byte) ([]byte, image.Rectangle, string, error) {
	data := sanitizeCapturedImage(raw)
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, image.Rectangle{}, "", err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > (64*1024*1024)/int64(config.Height) {
		return nil, image.Rectangle{}, "", fmt.Errorf("screenshot dimensions exceed supported limit")
	}
	decoded, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, image.Rectangle{}, "", err
	}
	return data, decoded.Bounds(), format, nil
}

func displayFrameVisible(raw []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(sanitizeCapturedImage(raw)))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	stepX := maxPositiveInt(1, bounds.Dx()/16)
	stepY := maxPositiveInt(1, bounds.Dy()/16)
	startY := bounds.Min.Y + bounds.Dy()/10
	endY := bounds.Min.Y + bounds.Dy()*9/10
	var total float64
	var maximum float64
	count := 0
	for y := startY; y < endY; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			luma := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 257
			total += luma
			if luma > maximum {
				maximum = luma
			}
			count++
		}
	}
	return count > 0 && total/float64(count) > 18 && maximum > 32
}

func maxPositiveInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *Adapter) encodeScreenshotFallback(samples *recordingScreenshots, started time.Time, elapsed time.Duration) ([]byte, int, error) {
	samples.mu.Lock()
	frames := append([][]byte(nil), samples.frames...)
	times := append([]time.Time(nil), samples.times...)
	samples.mu.Unlock()
	if len(frames) != len(times) || elapsed <= 0 {
		return nil, 0, errors.New("screenshot timing is unavailable")
	}
	end := started.Add(elapsed)
	for len(times) > 0 && times[len(times)-1].After(end) {
		times, frames = times[:len(times)-1], frames[:len(frames)-1]
	}
	if len(frames) < 2 {
		return nil, 0, fmt.Errorf("fewer than two visible screenshot frames were captured")
	}
	dir, err := os.MkdirTemp("", "device-control-recording-fallback-")
	if err != nil {
		return nil, 0, fmt.Errorf("create screenshot fallback workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	var manifest strings.Builder
	manifest.WriteString("ffconcat version 1.0\n")
	for i, frame := range frames {
		path := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := os.WriteFile(path, frame, 0o600); err != nil {
			return nil, 0, fmt.Errorf("write screenshot fallback frame: %w", err)
		}
		begin := times[i]
		if i == 0 {
			begin = started
		}
		finish := end
		if i+1 < len(times) {
			finish = times[i+1]
		}
		if !finish.After(begin) {
			return nil, 0, errors.New("screenshot timestamps are not ordered")
		}
		fmt.Fprintf(&manifest, "file 'frame-%04d.png'\nduration %.6f\n", i, finish.Sub(begin).Seconds())
	}
	fmt.Fprintf(&manifest, "file 'frame-%04d.png'\n", len(frames)-1)
	manifestPath := filepath.Join(dir, "frames.ffconcat")
	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o600); err != nil {
		return nil, 0, fmt.Errorf("write screenshot timing: %w", err)
	}
	outputPath := filepath.Join(dir, "recording.mp4")
	args := []string{"-y", "-loglevel", "error", "-f", "concat", "-safe", "1", "-i", manifestPath, "-t", strconv.FormatFloat(elapsed.Seconds(), 'f', 6, 64), "-c:v", "libx264", "-r", "15", "-pix_fmt", "yuv420p", outputPath}
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, 0, fmt.Errorf("encode screenshot fallback: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	encoded, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, 0, fmt.Errorf("read screenshot fallback: %w", err)
	}
	return encoded, len(frames), nil
}

// probeScreenshot verifies that the device can produce a frame the observer
// path will actually decode. It applies the same log-preamble sanitization used
// at capture time so an Android-TV firmware that prefixes screencap output is
// reported as available, while a device that returns non-image bytes is
// reported unavailable with a decode-specific next action.
// screenshotProbeTTL bounds how long a successful screenshot probe is reused.
// The probe takes a whole decodable frame (seconds on a TV), and a single
// capture request runs it through both capture preflight and flow validation.
const screenshotProbeTTL = 45 * time.Second

// probeCache remembers successful screenshot probes. One cache is shared by an
// adapter and every endpoint- or serial-bound clone made from it, so the
// per-request clones the control service creates still benefit. A nil cache
// disables caching.
type probeCache struct {
	mu      sync.Mutex
	okUntil map[string]time.Time
}

func newProbeCache() *probeCache { return &probeCache{okUntil: map[string]time.Time{}} }

func (c *probeCache) fresh(key string, now time.Time) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return now.Before(c.okUntil[key])
}

func (c *probeCache) remember(key string, until time.Time) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.okUntil[key] = until
	c.mu.Unlock()
}

// probeScreenshot proves the device returns a decodable frame. Only a success is
// cached, per adb selector: a failure is always re-probed so a recovered device
// is never reported unavailable from memory.
func (a *Adapter) probeScreenshot(ctx context.Context) (bool, string) {
	selector := strings.Join(a.args(), "|")
	if selector == "" {
		selector = "default"
	}
	if a.probes.fresh(selector, time.Now()) {
		return true, "adb screencap is unavailable; verify the device is authorized"
	}
	out, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
	if err != nil {
		return false, "adb screencap is unavailable; verify the device is authorized"
	}
	if _, _, _, decodeErr := validateCapturedImage(out); decodeErr != nil {
		return false, "adb screencap returned bytes device-control cannot decode as an image; the device may lack a display surface (use Cast/Remote for screenless control)"
	}
	a.probes.remember(selector, time.Now().Add(screenshotProbeTTL))
	return true, "adb screencap is unavailable; verify the device is authorized"
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	data, err := a.runner.Run(ctx, "adb", a.args("exec-out", "screencap", "-p")...)
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("adb screencap: %w", err)
	}
	data, bounds, format, err := validateCapturedImage(data)
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("decode adb screenshot: %w", err)
	}
	return strategy.Frame{Width: bounds.Dx(), Height: bounds.Dy(), Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/" + format, Bytes: data}, nil
}
