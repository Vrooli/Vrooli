package rendering

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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
