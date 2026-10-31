package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// pngHeader builds a valid PNG signature + IHDR chunk (with correct CRC)
// declaring the given dimensions. DecodeConfig reads only the header, so this is
// a faithful "decompression bomb" probe: ~45 bytes that claim a huge raster.
func pngHeader(width, height uint32) []byte {
	var b bytes.Buffer
	b.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	data := make([]byte, 13)
	binary.BigEndian.PutUint32(data[0:4], width)
	binary.BigEndian.PutUint32(data[4:8], height)
	data[8] = 8 // bit depth
	data[9] = 6 // color type RGBA
	// data[10..12] = 0 (compression, filter, interlace)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], 13)
	b.Write(lenBuf[:])
	chunk := append([]byte("IHDR"), data...)
	b.Write(chunk)
	var crcBuf [4]byte
	binary.BigEndian.PutUint32(crcBuf[:], crc32.ChecksumIEEE(chunk))
	b.Write(crcBuf[:])
	return b.Bytes()
}

func smallPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInspectValidImage(t *testing.T) {
	g := DefaultGuard()
	in, err := g.Inspect(bytes.NewReader(smallPNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	if in.Format != "png" || in.Width != 4 || in.Height != 4 {
		t.Fatalf("got %+v", in)
	}
	if len(in.Bytes) == 0 {
		t.Fatal("expected buffered bytes")
	}
}

func TestInspectRejectsDimensionBomb(t *testing.T) {
	g := DefaultGuard()
	_, err := g.Inspect(bytes.NewReader(pngHeader(100000, 100000)))
	if !errors.Is(err, ErrTooManyPixels) {
		t.Fatalf("want ErrTooManyPixels, got %v", err)
	}
}

func TestInspectRejectsMegapixelBomb(t *testing.T) {
	// 20000x20000 = 400 MP: under the 30000 per-side cap but over 128 MP.
	g := DefaultGuard()
	_, err := g.Inspect(bytes.NewReader(pngHeader(20000, 20000)))
	if !errors.Is(err, ErrTooManyPixels) || !strings.Contains(err.Error(), "MP") {
		t.Fatalf("want megapixel rejection, got %v", err)
	}
}

func TestInspectRejectsOversize(t *testing.T) {
	g := Guard{MaxBytes: 16, MaxMegapixels: 128, MaxDimension: 30000}
	_, err := g.Inspect(bytes.NewReader(make([]byte, 64)))
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
}

func TestInspectRejectsEmpty(t *testing.T) {
	g := DefaultGuard()
	if _, err := g.Inspect(bytes.NewReader(nil)); !errors.Is(err, ErrEmpty) {
		t.Fatalf("want ErrEmpty, got %v", err)
	}
	if _, err := g.Inspect(nil); !errors.Is(err, ErrEmpty) {
		t.Fatalf("want ErrEmpty for nil reader, got %v", err)
	}
}

func TestInspectUnknownFormatPassesOnByteCap(t *testing.T) {
	g := DefaultGuard()
	in, err := g.Inspect(strings.NewReader("not an image but small"))
	if err != nil {
		t.Fatalf("unknown format under byte cap should pass: %v", err)
	}
	if in.Format != "" {
		t.Fatalf("expected empty format, got %q", in.Format)
	}
}

func TestInspectAtByteLimitBoundary(t *testing.T) {
	data := smallPNG(t)
	g := Guard{MaxBytes: int64(len(data)), MaxMegapixels: 128, MaxDimension: 30000}
	if _, err := g.Inspect(bytes.NewReader(data)); err != nil {
		t.Fatalf("exactly-at-limit should pass: %v", err)
	}
	g.MaxBytes = int64(len(data)) - 1
	if _, err := g.Inspect(bytes.NewReader(data)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("one-over-limit should fail: %v", err)
	}
}
