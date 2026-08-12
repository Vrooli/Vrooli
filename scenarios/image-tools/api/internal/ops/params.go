package ops

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

// Params is the flat, transport-agnostic parameter set for every deterministic
// operation. Each op reads only the fields it needs and validates them; the
// flat shape keeps the job payload a single JSON object and the runner generic.
// The typed proto oneof (ops.proto) is translated into this struct at the REST
// edge, so the wire contract stays per-op-typed while execution stays uniform.
type Params struct {
	// Geometry (resize / thumbnail / canvas / crop targets).
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Fit mode for resize: "fit" (contain, preserve aspect), "fill" (cover +
	// center-crop), "stretch" (exact, ignore aspect). Empty defaults to "fit"
	// when only one of width/height is 0, else "stretch".
	Fit string `json:"fit,omitempty"`

	// Crop rectangle (pixels). Gravity selects an anchored crop when X/Y are
	// unset and only Width/Height are given.
	X       int    `json:"x,omitempty"`
	Y       int    `json:"y,omitempty"`
	Gravity string `json:"gravity,omitempty"` // center,top,bottom,left,right,top-left,...

	// Rotate / straighten.
	Angle  float64 `json:"angle,omitempty"`  // degrees, counter-clockwise
	Expand bool    `json:"expand,omitempty"` // expand canvas to fit a rotated image

	// Flip.
	Axis string `json:"axis,omitempty"` // "horizontal" | "vertical"

	// Convert / compress.
	Format      string `json:"format,omitempty"`
	Quality     int    `json:"quality,omitempty"`
	Lossless    bool   `json:"lossless,omitempty"`
	TargetBytes int64  `json:"target_bytes,omitempty"` // compress to <= this size (best quality fit)

	// Adjust (percent deltas; 0 = no change). Gamma 0 = no change (1.0 identity).
	Brightness float64 `json:"brightness,omitempty"`
	Contrast   float64 `json:"contrast,omitempty"`
	Saturation float64 `json:"saturation,omitempty"`
	Gamma      float64 `json:"gamma,omitempty"`
	Hue        float64 `json:"hue,omitempty"` // degrees, -180..180

	// Filter.
	Filter string  `json:"filter,omitempty"` // grayscale|sepia|invert|blur|sharpen
	Amount float64 `json:"amount,omitempty"` // sigma for blur/sharpen (and canny smoothing)

	// Deterministic treatment parameters.
	Dark               string  `json:"dark,omitempty"`
	Light              string  `json:"light,omitempty"`
	Mid                string  `json:"mid,omitempty"`
	MidLow             float64 `json:"mid_low,omitempty"`
	MidHigh            float64 `json:"mid_high,omitempty"`
	Levels             int     `json:"levels,omitempty"`
	LPI                int     `json:"lpi,omitempty"`
	Dot                string  `json:"dot,omitempty"`
	Seed               int64   `json:"seed,omitempty"`
	ContrastMultiplier float64 `json:"contrast_multiplier,omitempty"`
	ScrimColor         string  `json:"scrim_color,omitempty"`
	Direction          string  `json:"direction,omitempty"`
	// Normalize auto-levels the source's p1-p99 tonal range onto the full ink
	// ramp before mapping, so a low-contrast source still uses the whole ramp.
	// It makes the result depend on whole-image statistics; see treatments.Params.
	Normalize  bool    `json:"normalize,omitempty"`
	Spacing    float64 `json:"spacing,omitempty"`
	Radius     int     `json:"radius,omitempty"`
	BladeCount int     `json:"blade_count,omitempty"`
	Distance   int     `json:"distance,omitempty"`
	Amplitude  float64 `json:"amplitude,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
	Curve      float64 `json:"curve,omitempty"`
	BlockSize  int     `json:"block_size,omitempty"`

	// Relative spatial parameters: a fraction of the image's SHORT edge,
	// resolved to the pixel fields above by ResolveRelative once the image
	// geometry is known. A relative value wins over its absolute twin. See
	// relative.go for the conversion and the per-operation minimums.
	SpacingRel   float64 `json:"spacing_rel,omitempty"`
	RadiusRel    float64 `json:"radius_rel,omitempty"`
	DistanceRel  float64 `json:"distance_rel,omitempty"`
	AmplitudeRel float64 `json:"amplitude_rel,omitempty"`
	BlockSizeRel float64 `json:"block_size_rel,omitempty"`

	// Canny edge-preprocessor hysteresis bounds on the 0..255 gradient magnitude
	// (0 = defaults 50 / 150). The ControlNet "canny" preprocessor reads these.
	LowThreshold  float64 `json:"low_threshold,omitempty"`
	HighThreshold float64 `json:"high_threshold,omitempty"`

	// Canvas / background.
	Background string `json:"background,omitempty"` // hex (#rrggbb / #rrggbbaa)

	// Overlay / watermark / annotate.
	Text     string  `json:"text,omitempty"`
	Position string  `json:"position,omitempty"` // center,top-left,bottom-right,...
	Opacity  float64 `json:"opacity,omitempty"`  // 0..1 (default 1)
	Color    string  `json:"color,omitempty"`    // text color hex
	FontSize float64 `json:"font_size,omitempty"`
	// OverlayImage is raw bytes of an image to composite (watermark). Carried in
	// the job payload; set by the handler from a second multipart part.
	OverlayImage []byte `json:"overlay_image,omitempty"`

	// Metadata.
	StripAll   bool `json:"strip_all,omitempty"`
	StripGPS   bool `json:"strip_gps,omitempty"`
	AutoOrient bool `json:"auto_orient,omitempty"`
}

// resampleFilter returns the high-quality default resampling kernel.
func resampleFilter() imaging.ResampleFilter { return imaging.Lanczos }

// anchorFor maps a gravity/position name to an imaging.Anchor (default Center).
func anchorFor(name string) imaging.Anchor {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "top":
		return imaging.Top
	case "bottom":
		return imaging.Bottom
	case "left":
		return imaging.Left
	case "right":
		return imaging.Right
	case "top-left", "topleft":
		return imaging.TopLeft
	case "top-right", "topright":
		return imaging.TopRight
	case "bottom-left", "bottomleft":
		return imaging.BottomLeft
	case "bottom-right", "bottomright":
		return imaging.BottomRight
	default:
		return imaging.Center
	}
}

// parseHexColor parses "#rgb", "#rrggbb", or "#rrggbbaa" (leading # optional).
// Returns opaque black on empty input.
func parseHexColor(s string) (color.NRGBA, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return color.NRGBA{A: 255}, nil
	}
	s = strings.TrimPrefix(s, "#")
	expand := func(h string) (uint8, error) {
		v, err := strconv.ParseUint(h, 16, 8)
		return uint8(v), err
	}
	switch len(s) {
	case 3: // rgb
		r, e1 := expand(strings.Repeat(s[0:1], 2))
		g, e2 := expand(strings.Repeat(s[1:2], 2))
		b, e3 := expand(strings.Repeat(s[2:3], 2))
		if e1 != nil || e2 != nil || e3 != nil {
			return color.NRGBA{}, fmt.Errorf("ops: invalid hex color %q", s)
		}
		return color.NRGBA{R: r, G: g, B: b, A: 255}, nil
	case 6, 8:
		r, e1 := expand(s[0:2])
		g, e2 := expand(s[2:4])
		b, e3 := expand(s[4:6])
		a := uint8(255)
		var e4 error
		if len(s) == 8 {
			a, e4 = expand(s[6:8])
		}
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			return color.NRGBA{}, fmt.Errorf("ops: invalid hex color %q", s)
		}
		return color.NRGBA{R: r, G: g, B: b, A: a}, nil
	default:
		return color.NRGBA{}, fmt.Errorf("ops: invalid hex color %q", s)
	}
}
