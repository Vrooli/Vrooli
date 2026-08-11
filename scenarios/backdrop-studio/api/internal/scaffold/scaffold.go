package scaffold

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

type Region struct{ X, Y, Width, Height float64 }

type Request struct {
	Preset, Conditioner, ParamsJSON string
	Width, Height                   int
	Seed                            int64
	Regions                         []Region
}

type Result struct {
	PNG, SHA256   []byte
	Width, Height int
	Conditioner   string
}

type Preset struct {
	ID, Name, Subject string
	Parameters        []string
}

var presets = []Preset{
	{ID: "horizon", Name: "Layered Horizon", Subject: "horizon", Parameters: []string{"horizon", "focal_x", "depth_ramp"}},
	{ID: "arcade", Name: "Arched Arcade", Subject: "statuary_architecture", Parameters: []string{"bays", "focal_x", "depth_ramp"}},
	{ID: "terrain", Name: "Seeded Terrain", Subject: "geological", Parameters: []string{"horizon", "focal_x", "depth_ramp"}},
	{ID: "field", Name: "Blurred Colour Field", Subject: "non_representational", Parameters: []string{"focal_x", "depth_ramp"}},
}

func ListPresets() []Preset { return append([]Preset(nil), presets...) }

func Render(req Request) (Result, error) {
	if req.Width <= 0 || req.Height <= 0 || req.Width > 4096 || req.Height > 4096 {
		return Result{}, fmt.Errorf("scaffold: dimensions must be between 1 and 4096 pixels")
	}
	if req.Conditioner == "" {
		req.Conditioner = "depth"
	}
	if req.Conditioner != "depth" && req.Conditioner != "edge" {
		return Result{}, fmt.Errorf("scaffold: unsupported conditioner %q", req.Conditioner)
	}
	var found bool
	for _, p := range presets {
		if p.ID == req.Preset {
			found = true
			break
		}
	}
	if !found {
		return Result{}, fmt.Errorf("scaffold: unknown preset %q", req.Preset)
	}
	params := map[string]float64{}
	if req.ParamsJSON != "" {
		if err := json.Unmarshal([]byte(req.ParamsJSON), &params); err != nil {
			return Result{}, fmt.Errorf("scaffold: invalid params_json: %w", err)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, req.Width, req.Height))
	seed := uint64(req.Seed) + 0x9e3779b97f4a7c15
	noise := func() float64 { seed ^= seed << 7; seed ^= seed >> 9; return float64(seed%10000) / 10000 }
	horizon := clamp(params["horizon"], .2, .8, .58)
	focalX := clamp(params["focal_x"], .1, .9, .62)
	depth := clamp(params["depth_ramp"], .2, 1, .72)
	switch req.Preset {
	case "horizon":
		drawHorizon(img, horizon, focalX, depth, noise)
	case "arcade":
		drawArcade(img, int(clamp(params["bays"], 1, 8, 3)), focalX, depth, noise)
	case "terrain":
		drawTerrain(img, horizon, focalX, depth, noise)
	case "field":
		drawField(img, focalX, depth, noise)
	}
	if req.Conditioner == "edge" {
		img = edgeImage(img)
	}
	for _, r := range req.Regions {
		flatten(img, r, req.Conditioner)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return Result{}, fmt.Errorf("scaffold: encode PNG: %w", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return Result{PNG: buf.Bytes(), SHA256: []byte(hex.EncodeToString(sum[:])), Width: req.Width, Height: req.Height, Conditioner: req.Conditioner}, nil
}

func clamp(v, min, max, fallback float64) float64 {
	if v == 0 {
		return fallback
	}
	return math.Min(max, math.Max(min, v))
}
func set(img *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetRGBA(x, y, c)
	}
}

func drawHorizon(img *image.RGBA, horizon, focal, depth float64, noise func() float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		c := color.RGBA{uint8(22 + 35*t), uint8(34 + 65*t), uint8(73 + 95*t), 255}
		for x := 0; x < w; x++ {
			set(img, x, y, c)
		}
	}
	line := int(horizon * float64(h))
	for layer := 0; layer < 4; layer++ {
		base := line + int(float64(layer)*float64(h-line)/5)
		for x := 0; x < w; x++ {
			wave := int((math.Sin(float64(x)/float64(w)*math.Pi*float64(3+layer)) + noise()*.3) * float64(h) * .035 * depth)
			top := base - wave
			for y := top; y < h; y++ {
				set(img, x, y, color.RGBA{uint8(20 + layer*18), uint8(45 + layer*12), uint8(48 + layer*8), 255})
			}
		}
	}
	sunX, sunY := int(focal*float64(w)), int(float64(h)*.28)
	for y := sunY - 28; y <= sunY+28; y++ {
		for x := sunX - 28; x <= sunX+28; x++ {
			if math.Hypot(float64(x-sunX), float64(y-sunY)) < 28 {
				set(img, x, y, color.RGBA{245, 190, 95, 255})
			}
		}
	}
}

func drawArcade(img *image.RGBA, bays int, focal, depth float64, noise func() float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{13, 18, 31, 255}}, image.Point{}, draw.Src)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if noise() > .992 {
				set(img, x, y, color.RGBA{30, 38, 61, 255})
			}
		}
	}
	for x := 0; x < w; x++ {
		for y := int(float64(h) * .12); y < h; y++ {
			if (x+y)%23 < 3 {
				set(img, x, y, color.RGBA{87, 71, 61, 255})
			}
		}
	}
	span := float64(w) * .72 / float64(bays)
	left := int(float64(w) * (.14 + focal*.08))
	for i := 0; i < bays; i++ {
		cx := float64(left) + span*(float64(i)+.5)
		width := span * .72
		for y := int(float64(h) * .25); y < h; y++ {
			for x := int(cx - width/2); x <= int(cx+width/2); x++ {
				dx := math.Abs(float64(x)-cx) / (width / 2)
				arch := float64(h)*.25 + math.Sqrt(math.Max(0, 1-dx*dx))*float64(h)*.2
				if float64(y) > arch {
					set(img, x, y, color.RGBA{3, 7, 14, 255})
				}
			}
		}
	}
}

func drawTerrain(img *image.RGBA, horizon, focal, depth float64, noise func() float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / float64(h)
			set(img, x, y, color.RGBA{uint8(31 + 60*t), uint8(54 + 55*t), uint8(44 + 25*t), 255})
		}
	}
	for layer := 0; layer < 7; layer++ {
		base := int(float64(h)*horizon) + layer*h/12
		for x := 0; x < w; x++ {
			top := base - int((math.Sin(float64(x)/float64(w)*math.Pi*float64(2+layer))*.08+noise()*.05)*float64(h)*depth)
			for y := top; y < h; y++ {
				set(img, x, y, color.RGBA{uint8(29 + layer*13), uint8(47 + layer*10), uint8(38 + layer*7), 255})
			}
		}
	}
}

func drawField(img *image.RGBA, focal, depth float64, noise func() float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	fx, fy := focal*float64(w), .48*float64(h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := math.Hypot((float64(x)-fx)/float64(w), (float64(y)-fy)/float64(h))
			n := noise() * .08
			set(img, x, y, color.RGBA{uint8(25 + 100*math.Max(0, 1-d*2) + n*255), uint8(31 + 70*math.Max(0, 1-d) + n*180), uint8(75 + 105*math.Max(0, 1-d) + n*120), 255})
		}
	}
}

func edgeImage(src *image.RGBA) *image.RGBA {
	out := image.NewRGBA(src.Bounds())
	for y := 1; y < src.Bounds().Dy()-1; y++ {
		for x := 1; x < src.Bounds().Dx()-1; x++ {
			a := src.RGBAAt(x, y)
			b := src.RGBAAt(x+1, y)
			d := math.Abs(float64(a.R)-float64(b.R)) + math.Abs(float64(a.G)-float64(b.G)) + math.Abs(float64(a.B)-float64(b.B))
			v := uint8(math.Min(255, d*2))
			out.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return out
}
func flatten(img *image.RGBA, r Region, conditioner string) {
	x0 := int(r.X * float64(img.Bounds().Dx()))
	y0 := int(r.Y * float64(img.Bounds().Dy()))
	x1 := int((r.X + r.Width) * float64(img.Bounds().Dx()))
	y1 := int((r.Y + r.Height) * float64(img.Bounds().Dy()))
	c := color.RGBA{128, 128, 128, 255}
	if conditioner == "edge" {
		c = color.RGBA{0, 0, 0, 255}
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			set(img, x, y, c)
		}
	}
}
