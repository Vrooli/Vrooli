package posemap

import (
	"bytes"
	"image/png"
	"testing"
)

func TestCOCO18MapsMediaPipeLandmarksAndDerivesMidpoints(t *testing.T) {
	in := make([]Landmark, MediaPipeLandmarkCount)
	for i := range in {
		in[i] = Landmark{X: float64(i) / 40, Y: float64(i) / 40, Visibility: 1}
	}

	points, err := COCO18(in, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != COCOKeypointCount {
		t.Fatalf("points=%d, want %d", len(points), COCOKeypointCount)
	}
	if got, want := points[0], (Point{X: 0, Y: 0, Visible: true}); got != want {
		t.Fatalf("nose=%+v, want %+v", got, want)
	}
	if got, want := points[1], (Point{X: 11.5 / 40, Y: 11.5 / 40, Visible: true}); got != want {
		t.Fatalf("neck=%+v, want %+v", got, want)
	}
	if got, want := points[8], (Point{X: 24.0 / 40, Y: 24.0 / 40, Visible: true}); got != want {
		t.Fatalf("right hip=%+v, want %+v", got, want)
	}
	if got, want := points[11], (Point{X: 23.0 / 40, Y: 23.0 / 40, Visible: true}); got != want {
		t.Fatalf("left hip=%+v, want %+v", got, want)
	}
}

func TestCOCO18OmitsLowConfidenceAndOutOfBoundsLandmarks(t *testing.T) {
	in := make([]Landmark, MediaPipeLandmarkCount)
	for i := range in {
		in[i] = Landmark{X: 0.5, Y: 0.5, Visibility: 1}
	}
	in[0].Visibility = 0.1 // nose
	in[12].X = 1.1         // right shoulder; neck midpoint must also be absent
	in[5].Y = -0.1         // right eye

	points, err := COCO18(in, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 1, 2, 14} {
		if points[index].Visible {
			t.Errorf("point %d should be omitted: %+v", index, points[index])
		}
	}
	if !points[5].Visible {
		t.Fatal("unrelated left shoulder was lost")
	}
}

func TestCOCO18RejectsInvalidInput(t *testing.T) {
	if _, err := COCO18(nil, 0.5); err == nil {
		t.Fatal("expected missing landmark error")
	}
	in := make([]Landmark, MediaPipeLandmarkCount)
	if _, err := COCO18(in, 1.1); err == nil {
		t.Fatal("expected invalid visibility threshold error")
	}
}

func TestWritePNGDrawsBoundedOpenPoseCanvas(t *testing.T) {
	points := make([]Point, COCOKeypointCount)
	points[1] = Point{X: 0.5, Y: 0.2, Visible: true}
	points[2] = Point{X: 0.7, Y: 0.2, Visible: true}
	points[3] = Point{X: 0.8, Y: 0.5, Visible: true}
	var out bytes.Buffer
	if err := WritePNG(&out, 128, 64, points); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got.X != 128 || got.Y != 64 {
		t.Fatalf("size=%v, want 128x64", got)
	}
	colored := 0
	for y := 0; y < 64; y++ {
		for x := 0; x < 128; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r|g|b != 0 {
				colored++
			}
		}
	}
	if colored == 0 {
		t.Fatal("pose map contains no visible joints or bones")
	}
}

func TestWritePNGRejectsInvalidDimensionsAndPoints(t *testing.T) {
	points := make([]Point, COCOKeypointCount)
	var out bytes.Buffer
	if err := WritePNG(&out, 0, 512, points); err == nil {
		t.Fatal("expected invalid dimension error")
	}
	if err := WritePNG(&out, 512, 512, nil); err == nil {
		t.Fatal("expected invalid point count error")
	}
}
