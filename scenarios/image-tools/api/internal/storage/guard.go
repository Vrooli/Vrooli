package storage

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"

	// Register the standard decoders so DecodeConfig can peek their headers.
	// Phase 2 registers the remaining formats (webp/avif/heic/tiff/bmp) via
	// image.RegisterFormat; the guard picks them up automatically — no change
	// here is needed when a new decoder is registered.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// Decompression-bomb / oversize defaults. Recorded in docs/internal/DECISIONS.md.
// A "decompression bomb" is a tiny encoded file that expands to an enormous
// raster (e.g. a 1 KB PNG declaring 100000x100000 px → ~40 GB decoded). The
// byte cap bounds the encoded payload; the megapixel/dimension caps bound the
// DECODED allocation, checked from the header before any pixels are decoded.
const (
	// DefaultMaxBytes caps the encoded upload size (64 MiB).
	DefaultMaxBytes int64 = 64 << 20
	// DefaultMaxMegapixels caps decoded area (width*height), in megapixels.
	DefaultMaxMegapixels = 128
	// DefaultMaxDimension caps either side, in pixels.
	DefaultMaxDimension = 30000
)

// Guard enforces ingest limits. The zero value is unusable; use NewGuard or
// DefaultGuard.
type Guard struct {
	MaxBytes      int64
	MaxMegapixels int
	MaxDimension  int
}

// DefaultGuard returns a Guard with the documented default bounds.
func DefaultGuard() Guard {
	return Guard{
		MaxBytes:      DefaultMaxBytes,
		MaxMegapixels: DefaultMaxMegapixels,
		MaxDimension:  DefaultMaxDimension,
	}
}

// Ingest errors. Callers map these to 413 (too large) / 422 (bad image).
var (
	// ErrTooLarge is returned when the encoded payload exceeds MaxBytes.
	ErrTooLarge = errors.New("ingest: payload exceeds byte limit")
	// ErrTooManyPixels is returned when the decoded image would exceed the
	// megapixel or per-dimension cap (decompression-bomb guard).
	ErrTooManyPixels = errors.New("ingest: image dimensions exceed limit")
	// ErrEmpty is returned for a zero-byte upload.
	ErrEmpty = errors.New("ingest: empty payload")
)

// Inspected is the verified result of an ingest: the buffered bytes (safe to
// store/decode) plus the peeked header facts.
type Inspected struct {
	// Bytes is the full encoded payload (already bounded by MaxBytes).
	Bytes []byte
	// Format is the detected image format ("png", "jpeg", …); empty when the
	// format is not registered/recognized (byte cap still enforced).
	Format string
	// Width/Height are the decoded dimensions when Format is recognized; 0
	// otherwise.
	Width, Height int
}

// Inspect reads up to MaxBytes+1 from r, rejects oversize/empty payloads, peeks
// the image header to enforce the megapixel/dimension caps, and returns the
// buffered, verified bytes. It never decodes pixels — only the header config —
// so a bomb is rejected before any large allocation.
//
// Unrecognized formats (no registered decoder) pass the dimension check by
// design: the byte cap is the backstop, and Phase 2 registers more decoders.
func (g Guard) Inspect(r io.Reader) (Inspected, error) {
	if r == nil {
		return Inspected{}, ErrEmpty
	}
	limit := g.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxBytes
	}
	// Read one byte past the limit to detect overflow.
	buf, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return Inspected{}, fmt.Errorf("ingest: read: %w", err)
	}
	if int64(len(buf)) > limit {
		return Inspected{}, fmt.Errorf("%w (max %d bytes)", ErrTooLarge, limit)
	}
	if len(buf) == 0 {
		return Inspected{}, ErrEmpty
	}

	out := Inspected{Bytes: buf}
	cfg, format, derr := image.DecodeConfig(bytes.NewReader(buf))
	if derr != nil {
		// Unknown/unregistered format: accept on the byte-cap backstop. The
		// caller's decode step will reject genuinely invalid images.
		return out, nil
	}
	out.Format = format
	out.Width = cfg.Width
	out.Height = cfg.Height

	maxDim := g.MaxDimension
	if maxDim <= 0 {
		maxDim = DefaultMaxDimension
	}
	maxMP := g.MaxMegapixels
	if maxMP <= 0 {
		maxMP = DefaultMaxMegapixels
	}
	if cfg.Width > maxDim || cfg.Height > maxDim {
		return Inspected{}, fmt.Errorf("%w: %dx%d exceeds %d px per side", ErrTooManyPixels, cfg.Width, cfg.Height, maxDim)
	}
	// width*height can overflow int on 32-bit; compute in int64.
	megapixels := (int64(cfg.Width) * int64(cfg.Height)) / 1_000_000
	if megapixels > int64(maxMP) {
		return Inspected{}, fmt.Errorf("%w: %d MP exceeds %d MP", ErrTooManyPixels, megapixels, maxMP)
	}
	return out, nil
}
