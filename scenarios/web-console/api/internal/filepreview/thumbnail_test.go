package filepreview

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeImage(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// solid builds a w×h image whose left half is red and right half is blue, so
// a rotation is visible in where the red ends up.
func solid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{R: 220, A: 255}
			if x >= w/2 {
				c = color.NRGBA{B: 220, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation splices an EXIF APP1 segment carrying one Orientation tag
// right after the JPEG's SOI marker, the way a phone camera writes it.
func withOrientation(jpg []byte, o uint16) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08")
	ifd := make([]byte, 2+12+4)
	binary.BigEndian.PutUint16(ifd[0:], 1)
	binary.BigEndian.PutUint16(ifd[2:], 0x0112)
	binary.BigEndian.PutUint16(ifd[4:], 3)
	binary.BigEndian.PutUint32(ifd[6:], 1)
	binary.BigEndian.PutUint16(ifd[10:], o)
	payload := append([]byte("Exif\x00\x00"), append(tiff, ifd...)...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	out = append(out, payload...)
	return append(out, jpg[2:]...)
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	return img
}

func TestThumbnail_ScalesJPEGToFitTheEdge(t *testing.T) {
	path := writeImage(t, "wide.jpg", encodeJPEG(t, solid(800, 400)))
	data, ct, err := Thumbnail(path, 160)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type %q, want image/jpeg", ct)
	}
	if b := decode(t, data).Bounds(); b.Dx() != 160 || b.Dy() != 80 {
		t.Fatalf("size %dx%d, want 160x80", b.Dx(), b.Dy())
	}
}

func TestThumbnail_KeepsTransparencyAsPNGAndNeverEnlarges(t *testing.T) {
	small := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	var buf bytes.Buffer
	if err := png.Encode(&buf, small); err != nil {
		t.Fatal(err)
	}
	data, ct, err := Thumbnail(writeImage(t, "icon.png", buf.Bytes()), 160)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/png" {
		t.Fatalf("content type %q, want image/png", ct)
	}
	img := decode(t, data)
	if b := img.Bounds(); b.Dx() != 40 || b.Dy() != 20 {
		t.Fatalf("size %dx%d, want the original 40x20", b.Dx(), b.Dy())
	}
	if _, _, _, a := img.At(5, 5).RGBA(); a != 0 {
		t.Fatalf("alpha %d, want transparent", a)
	}
}

func TestThumbnail_AppliesEXIFOrientationSoPhonePhotosAreUpright(t *testing.T) {
	// Orientation 6: the stored pixels must be turned 90° clockwise to view.
	path := writeImage(t, "phone.jpg", withOrientation(encodeJPEG(t, solid(400, 200)), 6))
	data, _, err := Thumbnail(path, 160)
	if err != nil {
		t.Fatal(err)
	}
	img := decode(t, data)
	if b := img.Bounds(); b.Dx() != 80 || b.Dy() != 160 {
		t.Fatalf("size %dx%d, want portrait 80x160", b.Dx(), b.Dy())
	}
	// The stored left (red) half becomes the top after a clockwise turn.
	if r, _, bl, _ := img.At(40, 20).RGBA(); r < bl {
		t.Fatalf("top of the upright image is not red (r=%d b=%d)", r, bl)
	}
}

func TestThumbnail_RefusesFormatsItCannotDecode(t *testing.T) {
	for name, data := range map[string][]byte{
		"notes.txt": []byte("hello"),
		"fake.png":  []byte("\x89PNG\r\n\x1a\nnot really"),
	} {
		_, _, err := Thumbnail(writeImage(t, name, data), 160)
		if !errors.Is(err, ErrThumbnailUnsupported) {
			t.Fatalf("%s: err=%v, want ErrThumbnailUnsupported", name, err)
		}
	}
}

func TestThumbnail_ClampsTheRequestedEdge(t *testing.T) {
	path := writeImage(t, "big.jpg", encodeJPEG(t, solid(2000, 1000)))
	data, _, err := Thumbnail(path, 100_000)
	if err != nil {
		t.Fatal(err)
	}
	if b := decode(t, data).Bounds(); b.Dx() != ThumbnailMaxEdge {
		t.Fatalf("width %d, want the %d cap", b.Dx(), ThumbnailMaxEdge)
	}
}

// BenchmarkThumbnail_PhonePhoto measures the realistic worst case: a 12 MP
// camera JPEG rendered into a row tile.
func BenchmarkThumbnail_PhonePhoto(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 4032, 3024))
	for i := range img.Pix {
		img.Pix[i] = uint8(i * 7)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(b.TempDir(), "photo.jpg")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, _, err := Thumbnail(path, 96); err != nil {
			b.Fatal(err)
		}
	}
}
