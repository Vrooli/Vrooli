package ops

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"testing"
)

func plane(t *testing.T, c color.NRGBA, d float64) PlateSpec {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			im.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return PlateSpec{Image: b.Bytes(), Opacity: 1, Distance: &d}
}

func TestParallaxNearLayerTravelsFurtherAndIdentityPreservesPixels(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 60))
	src.SetNRGBA(50, 30, color.NRGBA{R: 255, A: 255})
	identity, err := renderLayeredView(src, image.Pt(100, 60), 1, LayeredCamera{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(identity.Pix, src.Pix) {
		t.Fatal("zero camera changes source pixels")
	}
	centroid := func(im *image.NRGBA) float64 {
		var sx, a float64
		for y := 0; y < 60; y++ {
			for x := 0; x < 100; x++ {
				v := float64(im.NRGBAAt(x, y).A)
				a += v
				sx += float64(x) * v
			}
		}
		return sx / a
	}
	near, err := renderLayeredView(src, image.Pt(100, 60), 1, LayeredCamera{X: .04})
	if err != nil {
		t.Fatal(err)
	}
	far, err := renderLayeredView(src, image.Pt(100, 60), 2, LayeredCamera{X: .04})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(centroid(near)-46) > .1 || math.Abs(centroid(far)-48) > .1 {
		t.Fatalf("parallax is not depth dependent: near=%v far=%v", centroid(near), centroid(far))
	}
}

func TestParallaxRejectsUnfilledViewsAndInvalidGeometry(t *testing.T) {
	p := &Params{Width: 80, Height: 60, Camera: &LayeredCamera{X: .04, RequireOpaque: true}, Plates: []PlateSpec{plane(t, color.NRGBA{R: 50, A: 255}, 1)}}
	if _, err := Composite(nil, p); err == nil {
		t.Fatal("missing edge scenery silently accepted")
	}
	p.Background = "#071422"
	if _, err := Composite(nil, p); err != nil {
		t.Fatal(err)
	}
	p.Plates[0].Distance = nil
	if _, err := Composite(nil, p); err == nil {
		t.Fatal("missing depth accepted")
	}
	p.Plates = []PlateSpec{plane(t, color.NRGBA{A: 255}, 1), plane(t, color.NRGBA{A: 255}, 2)}
	p.Plates[1].Depth = 1
	if _, err := Composite(nil, p); err == nil {
		t.Fatal("back-to-front geometry contradiction accepted")
	}
	p.Plates = p.Plates[:1]
	for _, c := range []LayeredCamera{{X: math.NaN()}, {Y: math.Inf(1)}, {YawDegrees: 6}, {Z: .09}, {FocalLengthPixels: 1}} {
		p.Camera = &c
		if _, err := Composite(nil, p); err == nil {
			t.Fatalf("invalid camera accepted: %+v", c)
		}
	}
}

func TestParallaxSamplingDoesNotLeakTransparentRGB(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 0})
	src.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	got := layeredSample(src, .5, 0)
	if got.R != 0 || got.B != 255 || got.A < 127 || got.A > 128 {
		t.Fatalf("alpha halo: %+v", got)
	}
}

func TestParallaxSequenceIsLosslessDeterministicAndCancellable(t *testing.T) {
	source := plane(t, color.NRGBA{R: 40, G: 60, B: 80, A: 255}, 2)
	p := &Params{Width: 80, Height: 60, Background: "#071422", Plates: []PlateSpec{source}, Parallax: &ParallaxSequence{Start: LayeredCamera{RequireOpaque: true}, End: LayeredCamera{X: .02, YawDegrees: .5, RequireOpaque: true}, Frames: 3, FPS: 30}}
	a, err := ExecuteContext(context.Background(), "parallax", source.Image, p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ExecuteContext(context.Background(), "parallax", source.Image, p)
	if err != nil {
		t.Fatal(err)
	}
	if a.Mime != "application/zip" || !bytes.Equal(a.Bytes, b.Bytes) {
		t.Fatal("sequence format or reproducibility changed")
	}
	z, err := zip.NewReader(bytes.NewReader(a.Bytes), int64(len(a.Bytes)))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != 4 {
		t.Fatalf("got %d archive entries", len(z.File))
	}
	var manifest struct {
		FPS    uint32 `json:"fps"`
		Frames []struct {
			Name   string `json:"name"`
			SHA256 string `json:"sha256"`
		} `json:"frames"`
	}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == "manifest.json" {
			if err = json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
		} else {
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil || img.Bounds() != image.Rect(0, 0, 80, 60) {
				t.Fatal("bad PNG frame")
			}
		}
	}
	if manifest.FPS != 30 || len(manifest.Frames) != 3 || len(manifest.Frames[2].SHA256) != 64 {
		t.Fatal("incomplete clock/frame provenance")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExecuteContext(ctx, "parallax", source.Image, p); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel ignored: %v", err)
	}
	p.Parallax.Frames = 181
	if _, err := Execute("parallax", source.Image, p); err == nil {
		t.Fatal("unbounded frame count accepted")
	}
	p.Parallax.Frames = 180
	p.Width = 1920
	p.Height = 1080
	if _, err := Execute("parallax", source.Image, p); err == nil {
		t.Fatal("unbounded total frame pixels accepted")
	}
}
