package executionwriter

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"sync"
)

const (
	// The decoded raster itself uses about four bytes per pixel, but overlapping
	// Go heap retention raised resident memory to about eight bytes per pixel in
	// the 10-writer fixture. Charge ten bytes so the 192 MiB budget admits only
	// one measured 16.384 MP full-page raster at a time.
	screenshotDecodeBytesPerPixel = int64(10)
	// The 16.384 MP full-page fixture consumes about 156 MiB of estimated budget.
	// Its measured one-at-a-time resident peak is about 149 MiB including runtime.
	maxActiveScreenshotDecodeBytes = int64(192 << 20)
)

type screenshotDecodeBudget struct {
	mu      sync.Mutex
	limit   int64
	used    int64
	changed chan struct{}
}

func newScreenshotDecodeBudget(limit int64) *screenshotDecodeBudget {
	return &screenshotDecodeBudget{limit: limit, changed: make(chan struct{})}
}

func (b *screenshotDecodeBudget) acquire(ctx context.Context, weight int64) error {
	if b == nil || weight <= 0 {
		return fmt.Errorf("invalid screenshot decode budget request: %d bytes", weight)
	}
	if weight > b.limit {
		return fmt.Errorf("screenshot raster estimate %d bytes exceeds active screenshot decode budget of %d bytes", weight, b.limit)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		b.mu.Lock()
		if b.used <= b.limit-weight {
			b.used += weight
			b.mu.Unlock()
			return nil
		}
		changed := b.changed
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (b *screenshotDecodeBudget) release(weight int64) {
	b.mu.Lock()
	b.used -= weight
	close(b.changed)
	b.changed = make(chan struct{})
	b.mu.Unlock()
}

// screenshotDecodeWeight validates dimensions before image.Decode allocates
// raster-sized buffers. The process-wide budget covers all FileWriter instances.
func screenshotDecodeWeight(width, height int) (int64, error) {
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("screenshot dimensions must be positive, got %dx%d", width, height)
	}
	maxPixels := maxActiveScreenshotDecodeBytes / screenshotDecodeBytesPerPixel
	if int64(width) > maxPixels/int64(height) {
		return 0, fmt.Errorf("screenshot raster %dx%d exceeds active screenshot decode budget of %d bytes", width, height, maxActiveScreenshotDecodeBytes)
	}
	return int64(width) * int64(height) * screenshotDecodeBytesPerPixel, nil
}

var processScreenshotDecodeBudget = newScreenshotDecodeBudget(maxActiveScreenshotDecodeBytes)

func screenshotDecodeConfig(data []byte) (image.Config, string, int64, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return image.Config{}, "", 0, fmt.Errorf("read screenshot dimensions: %w", err)
	}
	if format != "png" && format != "jpeg" {
		return image.Config{}, "", 0, fmt.Errorf("screenshot format %q is unsupported", format)
	}
	weight, err := screenshotDecodeWeight(cfg.Width, cfg.Height)
	if err != nil {
		return image.Config{}, "", 0, err
	}
	return cfg, format, weight, nil
}

func validateScreenshotImage(ctx context.Context, data []byte) (image.Config, string, error) {
	cfg, format, weight, err := screenshotDecodeConfig(data)
	if err != nil {
		return image.Config{}, "", err
	}
	if err := processScreenshotDecodeBudget.acquire(ctx, weight); err != nil {
		return image.Config{}, "", fmt.Errorf("acquire screenshot decode budget: %w", err)
	}
	decodeErr := func() error {
		defer processScreenshotDecodeBudget.release(weight)
		decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("decode screenshot: %w", err)
		}
		if decodedFormat != format || decoded.Bounds().Empty() || decoded.Bounds().Dx() != cfg.Width || decoded.Bounds().Dy() != cfg.Height {
			return fmt.Errorf("screenshot is not a supported encoded image")
		}
		return nil
	}()
	if decodeErr != nil {
		return image.Config{}, "", decodeErr
	}
	return cfg, format, nil
}
