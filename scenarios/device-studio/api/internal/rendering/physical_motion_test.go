package rendering

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhysicalMotionReceiptFrameDigestsBindExactOrderedSchedule(t *testing.T) {
	plan := PhysicalOutputPlan{
		Kind: "motion_mp4", FrameCount: 2,
		Samples: []PhysicalMotionSample{{OutputOrdinal: 0}, {OutputOrdinal: 1}},
	}
	valid := []PhysicalFrameRGBDigest{{OutputOrdinal: 0, RGBSHA256: strings.Repeat("a", 64)}, {OutputOrdinal: 1, RGBSHA256: strings.Repeat("b", 64)}}
	if err := validatePhysicalFrameRGBDigests(plan, valid); err != nil {
		t.Fatalf("valid ordered frame digests rejected: %v", err)
	}
	tests := []struct {
		name    string
		plan    PhysicalOutputPlan
		digests []PhysicalFrameRGBDigest
	}{
		{name: "missing frame", plan: plan, digests: valid[:1]},
		{name: "wrong ordinal", plan: plan, digests: []PhysicalFrameRGBDigest{{OutputOrdinal: 1, RGBSHA256: valid[0].RGBSHA256}, valid[1]}},
		{name: "schedule ordinal changed", plan: PhysicalOutputPlan{Kind: "motion_mp4", FrameCount: 2, Samples: []PhysicalMotionSample{{OutputOrdinal: 1}, {OutputOrdinal: 0}}}, digests: valid},
		{name: "malformed digest", plan: plan, digests: []PhysicalFrameRGBDigest{{OutputOrdinal: 0, RGBSHA256: "bad"}, valid[1]}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validatePhysicalFrameRGBDigests(test.plan, test.digests); err == nil {
				t.Fatal("invalid per-frame digest binding was accepted")
			}
		})
	}
}

func TestStillOutputReceiptOmitsMotionFrameDigests(t *testing.T) {
	encoded, err := json.Marshal(PhysicalOutputReceipt{Kind: "still_png", Name: PhysicalStillOutputName, PixelFormat: "rgba8"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("frame_rgb_digests")) {
		t.Fatalf("still receipt acquired a motion-only field: %s", encoded)
	}
}

func TestReadPhysicalFrameRequiresBoundedCompleteOrdinalRecord(t *testing.T) {
	payload := []byte("frame-png")
	var record bytes.Buffer
	var header [12]byte
	binary.BigEndian.PutUint32(header[:4], 7)
	binary.BigEndian.PutUint64(header[4:], uint64(len(payload)))
	record.Write(header[:])
	record.Write(payload)
	ordinal, got, err := readPhysicalFrame(&record)
	if err != nil || ordinal != 7 || !bytes.Equal(got, payload) {
		t.Fatalf("read frame = (%d, %q, %v), want ordinal 7 and exact payload", ordinal, got, err)
	}
	if _, _, err := readPhysicalFrame(bytes.NewReader(record.Bytes()[:10])); err == nil {
		t.Fatal("truncated frame header was accepted")
	}
	oversized := make([]byte, 12)
	binary.BigEndian.PutUint64(oversized[4:], uint64(physicalMotionFrameMax+1))
	if _, _, err := readPhysicalFrame(bytes.NewReader(oversized)); err == nil {
		t.Fatal("frame payload above the fixed bound was accepted")
	}
}

func TestPhysicalMotionPreviewOrdinalsRetainBoundedStartMiddleAndEndFrames(t *testing.T) {
	if got, want := physicalMotionPreviewOrdinals(48), [3]uint32{0, 23, 47}; got != want {
		t.Fatalf("48-frame preview ordinals = %v, want %v", got, want)
	}
	if got, want := physicalMotionPreviewOrdinals(1), [3]uint32{0, 0, 0}; got != want {
		t.Fatalf("single-frame preview ordinals = %v, want %v", got, want)
	}
}

func TestPhysicalPNGToRGBKeepsDisplayEncodedPixelsAndExactRaster(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 16, G: 128, B: 240, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 3, G: 4, B: 5, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	rgb, err := physicalPNGToRGB(encoded.Bytes(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{16, 128, 240, 3, 4, 5}; !bytes.Equal(rgb, want) {
		t.Fatalf("packed RGB = %v, want exact PNG display values %v", rgb, want)
	}
	if _, err := physicalPNGToRGB(encoded.Bytes(), 1, 2); err == nil {
		t.Fatal("frame with a different output raster was accepted")
	}
	transparent := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	transparent.SetNRGBA(0, 0, color.NRGBA{R: 4, G: 5, B: 6, A: 128})
	encoded.Reset()
	if err := png.Encode(&encoded, transparent); err != nil {
		t.Fatal(err)
	}
	if _, err := physicalPNGToRGB(encoded.Bytes(), 1, 1); err == nil {
		t.Fatal("non-opaque frame pixels were silently flattened into RGB")
	}
}

func TestRetainInterruptedPhysicalFrameKeepsOnlyValidatedBoundedPNG(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	encoded := testRGBA8PNG(t, 2, 1, []byte{17, 120, 231, 255, 6, 44, 99, 255})
	if _, format, err := image.DecodeConfig(bytes.NewReader(encoded)); err != nil || format != "png" {
		t.Fatalf("test RGBA PNG invalid: format=%q err=%v", format, err)
	}
	if decoded, format, err := image.Decode(bytes.NewReader(encoded)); err != nil || format != "png" {
		t.Fatalf("test RGBA PNG decode: format=%q err=%v", format, err)
	} else if decoded.Bounds().Dx() != 2 || decoded.Bounds().Dy() != 1 {
		t.Fatalf("test RGBA PNG raster bounds=%v", decoded.Bounds())
	}
	if err := os.WriteFile(filepath.Join(dir, physicalMotionFrameName), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := retainInterruptedPhysicalFrame(root, 2, 1); err != nil {
		t.Fatalf("valid interrupted frame was not retained: %v", err)
	}
	retained, err := os.ReadFile(filepath.Join(dir, physicalMotionInterruptedPreview))
	if err != nil || !bytes.Equal(retained, encoded) {
		t.Fatalf("retained interruption preview changed bytes: err=%v", err)
	}
	if err := retainInterruptedPhysicalFrame(root, 2, 1); err == nil {
		t.Fatal("duplicate interrupted preview overwrote immutable attempt evidence")
	}
	if err := os.Remove(filepath.Join(dir, physicalMotionInterruptedPreview)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, physicalMotionFrameName), []byte("not a PNG"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := retainInterruptedPhysicalFrame(root, 2, 1); err == nil {
		t.Fatal("malformed interrupted scratch was retained")
	}
}

func TestPhysicalMotionCheckpointAtomicallyKeepsOnlyLatestValidatedFrame(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	first := testRGBA8PNG(t, 2, 1, []byte{17, 120, 231, 255, 6, 44, 99, 255})
	latest := testRGBA8PNG(t, 2, 1, []byte{200, 110, 20, 255, 12, 34, 56, 255})
	if err := writePhysicalMotionCheckpoint(root, first, 2, 1); err != nil {
		t.Fatalf("write first checkpoint: %v", err)
	}
	if err := writePhysicalMotionCheckpoint(root, latest, 2, 1); err != nil {
		t.Fatalf("replace checkpoint: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, physicalMotionCheckpoint))
	if err != nil || !bytes.Equal(got, latest) {
		t.Fatalf("checkpoint did not contain latest complete frame: bytes=%d err=%v", len(got), err)
	}
	if _, err := os.Lstat(filepath.Join(dir, physicalMotionCheckpointTemp)); !os.IsNotExist(err) {
		t.Fatalf("atomic checkpoint temporary remains after publish: %v", err)
	}
	if err := writePhysicalMotionCheckpoint(root, []byte("not png"), 2, 1); err == nil {
		t.Fatal("malformed checkpoint was accepted")
	}
}

func testRGBA8PNG(t *testing.T, width, height uint32, pixels []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(append([]byte{0}, pixels...)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	output.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	writeChunk := func(kind string, data []byte) {
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(data)))
		output.Write(length[:])
		output.WriteString(kind)
		output.Write(data)
		checksum := crc32.NewIEEE()
		_, _ = checksum.Write([]byte(kind))
		_, _ = checksum.Write(data)
		binary.BigEndian.PutUint32(length[:], checksum.Sum32())
		output.Write(length[:])
	}
	var header [13]byte
	binary.BigEndian.PutUint32(header[0:4], width)
	binary.BigEndian.PutUint32(header[4:8], height)
	header[8], header[9] = 8, 6
	writeChunk("IHDR", header[:])
	writeChunk("IDAT", compressed.Bytes())
	writeChunk("IEND", nil)
	return output.Bytes()
}
