package posemap

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"
)

// standing is a front-facing figure in a 400x800 image: shoulders at y=200,
// hips at y=420, feet at y=700.
func standing() (Document, Person) {
	var d Document
	d.Schema, d.KeypointFloor = Schema, 0.3
	d.Image.Width, d.Image.Height = 400, 800
	at := func(x, y float64) Keypoint { return Keypoint{x, y, 0.9} }
	p := Person{Score: 1, BoxSource: "detector"}
	p.Body = []Keypoint{
		at(200, 100), at(200, 200), at(150, 200), at(120, 300), at(110, 400),
		at(250, 200), at(280, 300), at(290, 400), at(170, 420), at(170, 560),
		at(170, 700), at(230, 420), at(230, 560), at(230, 700), at(190, 90),
		at(210, 90), at(175, 100), at(225, 100),
	}
	p.LeftHand = make([]Keypoint, HandKeypoints)
	p.RightHand = make([]Keypoint, HandKeypoints)
	d.People = []Person{p}
	return d, p
}

func render(t *testing.T, d Document, p Person, w, h int, v View) *image.RGBA {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, d, p, w, h, v); err != nil {
		t.Fatalf("Render: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rgba := image.NewRGBA(img.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

func TestParseRoundTripsTheDocumentThePoseOperationWrites(t *testing.T) {
	d, _ := standing()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.People[0].Body[3] != (Keypoint{120, 300, 0.9}) {
		t.Fatalf("elbow = %+v", got.People[0].Body[3])
	}
}

func TestParseRejectsWrongSchemaFloorAndShapes(t *testing.T) {
	d, _ := standing()
	cases := map[string]func(*Document){
		"schema": func(d *Document) { d.Schema = "pose-v2" },
		"floor":  func(d *Document) { d.KeypointFloor = 0 },
		"body":   func(d *Document) { d.People[0].Body = d.People[0].Body[:17] },
		"hand":   func(d *Document) { d.People[0].LeftHand = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := d
			c.People = []Person{d.People[0]}
			mutate(&c)
			raw, _ := json.Marshal(c)
			if _, err := Parse(raw); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if _, err := Parse([]byte(`{"schema":"pose-v1","keypoint_floor":0.3,"people":[{"body":[[1,2]]}]}`)); err == nil {
		t.Fatal("a keypoint with two numbers should be refused")
	}
}

func TestPrimaryIsTheFigureWithMostReadableJoints(t *testing.T) {
	d, p := standing()
	partial := p
	partial.Body = append([]Keypoint(nil), p.Body...)
	for i := 6; i < 18; i++ {
		partial.Body[i].Confidence = 0.1
	}
	d.People = []Person{partial, p}
	got, err := d.Primary()
	if err != nil || d.Visible(got) != 18 {
		t.Fatalf("Primary = %d visible, err %v", d.Visible(got), err)
	}
	d.People = nil
	if _, err := d.Primary(); err == nil {
		t.Fatal("nobody should be ErrNoPose")
	}
	empty := p
	empty.Body = make([]Keypoint, BodyKeypoints)
	d.People = []Person{empty}
	if _, err := d.Primary(); err == nil {
		t.Fatal("a figure with no readable joint should be ErrNoPose")
	}
}

func TestRenderDrawsLimbsDimmerThanJointsOnBlack(t *testing.T) {
	d, p := standing()
	img := render(t, d, p, 400, 800, d.Whole())
	if got := img.RGBAAt(5, 5); got != (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("background = %v, want black", got)
	}
	// Joint 1 (the neck) wears its own color; the limb halfway down the spine does not.
	if got := img.RGBAAt(200, 200); got != boneColors[1] {
		t.Fatalf("neck joint = %v, want %v", got, boneColors[1])
	}
	limb := img.RGBAAt(185, 310) // between the neck and the right hip
	if limb == (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("expected a limb through the torso, got background")
	}
	if limb.R > 200 || limb.G > 200 || limb.B > 200 {
		t.Fatalf("limb %v is as bright as a joint", limb)
	}
}

func TestRenderLeavesOutLowConfidenceJointsAndTheirBones(t *testing.T) {
	d, p := standing()
	p.Body = append([]Keypoint(nil), p.Body...)
	p.Body[4].Confidence = 0.1 // right wrist
	img := render(t, d, p, 400, 800, d.Whole())
	if got := img.RGBAAt(110, 400); got != (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("the dropped wrist is drawn: %v", got)
	}
	if got := img.RGBAAt(115, 350); got != (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("the forearm to the dropped wrist is drawn: %v", got)
	}
	if got := img.RGBAAt(120, 300); got != boneColors[3] {
		t.Fatalf("the elbow should still be drawn: %v", got)
	}
}

func TestRenderViewCropsAndScalesTheSource(t *testing.T) {
	d, p := standing()
	// Show only the top half of the image on a canvas twice as wide as the crop.
	img := render(t, d, p, 800, 800, View{0, 0, 400, 400})
	if got := img.RGBAAt(400, 400); got != boneColors[1] {
		t.Fatalf("neck at (200,200) should land at (400,400): %v", got)
	}
	if got := img.RGBAAt(100, 780); got == boneColors[2] {
		t.Fatal("a joint outside the view must not be drawn")
	}
}

func TestRenderLineWeightFollowsTheCanvas(t *testing.T) {
	d, p := standing()
	thickness := func(w, h int) int {
		img := render(t, d, p, w, h, d.Whole())
		x := int(float64(w) * 0.5)
		y := int(float64(h) * 0.6)
		n := 0
		for dx := -w / 4; dx < w/4; dx++ {
			if px := img.RGBAAt(x+dx, y); px != (color.RGBA{0, 0, 0, 255}) {
				n++
			}
		}
		return n
	}
	small, large := thickness(400, 800), thickness(1600, 3200)
	if large <= small {
		t.Fatalf("line weight did not grow with the canvas: %d then %d", small, large)
	}
}

func TestRenderDrawsAHandOnlyWhenEnoughJointsAreRead(t *testing.T) {
	d, p := standing()
	hand := func(n int) []Keypoint {
		out := make([]Keypoint, HandKeypoints)
		for i := 0; i < n; i++ {
			out[i] = Keypoint{300 + float64(i)*4, 500, 0.9}
		}
		return out
	}
	count := func(img *image.RGBA) int {
		blue := 0
		for y := 480; y < 520; y++ {
			for x := 290; x < 400; x++ {
				if img.RGBAAt(x, y) == (color.RGBA{0, 0, 255, 255}) {
					blue++
				}
			}
		}
		return blue
	}
	p.LeftHand = hand(3)
	if count(render(t, d, p, 400, 800, d.Whole())) != 0 {
		t.Fatal("a hand of three joints is noise and must not be drawn")
	}
	p.LeftHand = hand(12)
	if count(render(t, d, p, 400, 800, d.Whole())) == 0 {
		t.Fatal("a hand of twelve joints should be drawn")
	}
}

func TestRenderRejectsBadCanvasAndView(t *testing.T) {
	d, p := standing()
	for _, c := range []struct {
		name string
		w, h int
		v    View
	}{
		{"zero width", 0, 100, d.Whole()},
		{"too large", MaxCanvas + 1, 100, d.Whole()},
		{"empty view", 100, 100, View{10, 10, 10, 20}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := Render(&bytes.Buffer{}, d, p, c.w, c.h, c.v); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if err := Render(&bytes.Buffer{}, d, p, 0, 0, d.Whole()); err == nil || !strings.Contains(err.Error(), "canvas") {
		t.Fatalf("error should name the canvas: %v", err)
	}
}

func TestFitShowsTheBoxWholeOnTheCanvasAspect(t *testing.T) {
	for _, aspect := range []float64{0.5, 1, 2} {
		v, err := Fit([4]float64{100, 200, 300, 600}, aspect)
		if err != nil {
			t.Fatal(err)
		}
		w, h := v.X1-v.X0, v.Y1-v.Y0
		if got := w / h; got < aspect*0.999 || got > aspect*1.001 {
			t.Errorf("aspect %v: view %vx%v is %v", aspect, w, h, got)
		}
		if v.X0 > 100 || v.Y0 > 200 || v.X1 < 300 || v.Y1 < 600 {
			t.Errorf("aspect %v: view %+v cuts the box", aspect, v)
		}
		if cx, cy := (v.X0+v.X1)/2, (v.Y0+v.Y1)/2; math.Abs(cx-200) > 1e-6 || math.Abs(cy-400) > 1e-6 {
			t.Errorf("aspect %v: view centre %v,%v moved off the box", aspect, cx, cy)
		}
	}
	for _, box := range [][4]float64{{10, 10, 10, 50}, {50, 10, 10, 50}} {
		if _, err := Fit(box, 1); err == nil {
			t.Errorf("box %v should be refused", box)
		}
	}
	if _, err := Fit([4]float64{0, 0, 10, 10}, 0); err == nil {
		t.Error("a zero aspect should be refused")
	}
}

func TestCanvasKeepsTheAspectAtTheRequestedArea(t *testing.T) {
	cases := []struct {
		aspect float64
		area   int
		w, h   int
	}{
		{1, 1024 * 1024, 1024, 1024},
		{0.5, 1024 * 1024, 704, 1472},
		{2, 1024 * 1024, 1472, 704},
		{1, 512 * 512, 512, 512},
		{0.1, 1024 * 1024, 512, 2048}, // clamped, never a sliver
	}
	for _, c := range cases {
		if w, h := Canvas(c.aspect, c.area); w != c.w || h != c.h {
			t.Errorf("Canvas(%v, %d) = %dx%d, want %dx%d", c.aspect, c.area, w, h, c.w, c.h)
		}
	}
	if w, h := Canvas(math.NaN(), 1<<20); w != 1024 || h != 1024 {
		t.Errorf("a NaN aspect should give the square, got %dx%d", w, h)
	}
}
