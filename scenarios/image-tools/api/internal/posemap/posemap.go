// Package posemap reads the pose document the pose_extract operation returns and
// draws it as the colored bone image OpenPose ControlNets are trained on. It has
// no model runtime: detection is the caller's, so drawing and parsing are tested
// without weights.
package posemap

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Schema is the document layout version this package reads.
const Schema = "pose-v1"

const (
	// BodyKeypoints is the length of the body array: OpenPose-18 order.
	BodyKeypoints = 18
	// HandKeypoints is the length of each hand array.
	HandKeypoints = 21
	// MaxCanvas bounds the drawn image so a request cannot reserve unbounded scratch.
	MaxCanvas = 4096
	// minHandJoints is how many joints a hand needs before it is drawn: a hand read
	// from two or three stray joints is noise, not a gesture.
	minHandJoints = 6
)

// ModelID is the registry model the pose operation reads with.
const ModelID = "dwpose-ll-384"

// Request asks for the control image of one pose source.
type Request struct {
	// Source and Output are file paths: the picture to read and the PNG to write.
	Source, Output string
	// Width and Height are the control image's size, which is the generation canvas.
	Width, Height int
	// Box limits the reading to one region of the source (a chart cell), in source
	// pixels. Nil reads and shows the whole source.
	Box *[4]float64
	// Prepare is the pose_extract preparation mode ("" lets the operation decide).
	Prepare string
}

// Keypoint is one joint in image pixels with the estimator's confidence.
type Keypoint struct{ X, Y, Confidence float64 }

// UnmarshalJSON reads the [x, y, confidence] triple the document stores.
func (k *Keypoint) UnmarshalJSON(raw []byte) error {
	var triple [3]float64
	if err := json.Unmarshal(raw, &triple); err != nil {
		return fmt.Errorf("posemap: a keypoint is [x, y, confidence]: %w", err)
	}
	*k = Keypoint{triple[0], triple[1], triple[2]}
	return nil
}

// MarshalJSON writes the [x, y, confidence] triple.
func (k Keypoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]float64{k.X, k.Y, k.Confidence})
}

// Person is one figure: its box in image pixels, its body joints in OpenPose-18
// order and both hands.
type Person struct {
	Box       [4]float64 `json:"box"`
	BoxSource string     `json:"box_source"`
	Score     float64    `json:"score"`
	Body      []Keypoint `json:"body"`
	LeftHand  []Keypoint `json:"left_hand"`
	RightHand []Keypoint `json:"right_hand"`
}

// Document is the pose_extract result.
type Document struct {
	Schema string `json:"schema"`
	Image  struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"image"`
	BodyKeypoints []string `json:"body_keypoints"`
	KeypointFloor float64  `json:"keypoint_floor"`
	People        []Person `json:"people"`
}

// ErrNoPose is returned when a document holds nobody with a readable body.
var ErrNoPose = errors.New("posemap: no person with a readable pose in the document")

// Parse reads and validates a pose_extract document.
func Parse(raw []byte) (Document, error) {
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doc, fmt.Errorf("posemap: decode pose document: %w", err)
	}
	if doc.Schema != Schema {
		return doc, fmt.Errorf("posemap: pose document schema %q, want %q", doc.Schema, Schema)
	}
	if doc.KeypointFloor <= 0 || doc.KeypointFloor >= 1 {
		return doc, fmt.Errorf("posemap: keypoint_floor %v must be between 0 and 1", doc.KeypointFloor)
	}
	for i, p := range doc.People {
		if len(p.Body) != BodyKeypoints || len(p.LeftHand) != HandKeypoints || len(p.RightHand) != HandKeypoints {
			return doc, fmt.Errorf("posemap: person %d has %d body, %d + %d hand keypoints, want %d, %d + %d",
				i, len(p.Body), len(p.LeftHand), len(p.RightHand), BodyKeypoints, HandKeypoints, HandKeypoints)
		}
	}
	return doc, nil
}

// Visible counts the body joints at or above the document's confidence floor.
func (d Document) Visible(p Person) int {
	n := 0
	for _, k := range p.Body {
		if k.Confidence >= d.KeypointFloor {
			n++
		}
	}
	return n
}

// Primary is the person with the most readable body joints.
func (d Document) Primary() (Person, error) {
	best, most := -1, 0
	for i, p := range d.People {
		if n := d.Visible(p); n > most {
			best, most = i, n
		}
	}
	if best < 0 {
		return Person{}, ErrNoPose
	}
	return d.People[best], nil
}

// View is the part of the source image the canvas shows, in source pixels.
type View struct{ X0, Y0, X1, Y1 float64 }

// Whole is the view of an entire image.
func (d Document) Whole() View {
	return View{0, 0, float64(d.Image.Width), float64(d.Image.Height)}
}

// viewMargin is the share of the box added on each side, so a figure that fills its
// box does not touch the control image's edge (a ControlNet draws limbs it sees
// cut off by the edge as cut off).
const viewMargin = 0.06

// Fit is the view that shows `box` (x0, y0, x1, y1 in source pixels) whole, with a
// margin, on a canvas of the given aspect ratio (width / height). The view grows
// along the short side instead of stretching the box, and may reach past the image:
// what lies outside it is drawn black.
func Fit(box [4]float64, aspect float64) (View, error) {
	w, h := box[2]-box[0], box[3]-box[1]
	if w <= 0 || h <= 0 || aspect <= 0 || math.IsNaN(aspect) || math.IsInf(aspect, 0) {
		return View{}, errors.New("posemap: the box and the canvas aspect must have positive size")
	}
	w, h = w*(1+2*viewMargin), h*(1+2*viewMargin)
	if w/h < aspect {
		w = h * aspect
	} else {
		h = w / aspect
	}
	cx, cy := (box[0]+box[2])/2, (box[1]+box[3])/2
	return View{cx - w/2, cy - h/2, cx + w/2, cy + h/2}, nil
}

// canvasStep is the multiple generation canvases snap to (latent-space models need
// 8; 64 keeps the size friendly to every pipeline).
const canvasStep = 64

// Canvas is the width and height, multiples of 64 within 512..2048 and about
// `area` pixels, for a view of the given aspect ratio.
func Canvas(aspect float64, area int) (width, height int) {
	if aspect <= 0 || math.IsNaN(aspect) || math.IsInf(aspect, 0) {
		aspect = 1
	}
	h := math.Sqrt(float64(area) / aspect)
	snap := func(v float64) int {
		return min(2048, max(512, int(math.Round(v/canvasStep))*canvasStep))
	}
	return snap(h * aspect), snap(h)
}

type bone struct{ a, b int }

var bodyBones = [...]bone{
	{1, 2}, {1, 5}, {2, 3}, {3, 4}, {5, 6}, {6, 7}, {1, 8}, {8, 9}, {9, 10},
	{1, 11}, {11, 12}, {12, 13}, {1, 0}, {0, 14}, {14, 16}, {0, 15}, {15, 17},
}

// handBones are the 20 edges of a hand: the wrist, then four joints up each finger.
var handBones = [...]bone{
	{0, 1}, {1, 2}, {2, 3}, {3, 4}, {0, 5}, {5, 6}, {6, 7}, {7, 8}, {0, 9}, {9, 10},
	{10, 11}, {11, 12}, {0, 13}, {13, 14}, {14, 15}, {15, 16}, {0, 17}, {17, 18}, {18, 19}, {19, 20},
}

var boneColors = [...]color.RGBA{
	{255, 0, 0, 255}, {255, 85, 0, 255}, {255, 170, 0, 255}, {255, 255, 0, 255},
	{170, 255, 0, 255}, {85, 255, 0, 255}, {0, 255, 0, 255}, {0, 255, 85, 255},
	{0, 255, 170, 255}, {0, 255, 255, 255}, {0, 170, 255, 255}, {0, 85, 255, 255},
	{0, 0, 255, 255}, {85, 0, 255, 255}, {170, 0, 255, 255}, {255, 0, 255, 255},
	{255, 0, 170, 255}, {255, 0, 85, 255},
}

// limbShade dims a limb against its joint markers, as OpenPose draws them, so the
// joints stand out from the bones that meet at them.
const limbShade = 0.6

// Render draws the person's body and hands on a black canvas of width x height,
// showing `view` of the source image. Line weight follows the canvas, so a larger
// control image keeps the same proportions. Joints under the confidence floor are
// left out, and so is every bone that touches one.
func Render(w io.Writer, d Document, p Person, width, height int, view View) error {
	if width < 1 || height < 1 || width > MaxCanvas || height > MaxCanvas {
		return fmt.Errorf("posemap: canvas dimensions must be between 1 and %d", MaxCanvas)
	}
	if view.X1 <= view.X0 || view.Y1 <= view.Y0 {
		return errors.New("posemap: the view must have positive size")
	}
	sx, sy := float64(width)/(view.X1-view.X0), float64(height)/(view.Y1-view.Y0)
	place := func(k Keypoint) (image.Point, bool) {
		if k.Confidence < d.KeypointFloor || math.IsNaN(k.X) || math.IsNaN(k.Y) || math.IsInf(k.X, 0) || math.IsInf(k.Y, 0) {
			return image.Point{}, false
		}
		return image.Pt(int(math.Round((k.X-view.X0)*sx)), int(math.Round((k.Y-view.Y0)*sy))), true
	}
	stick := max(3, int(math.Round(float64(min(width, height))/160)))
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := range canvas.Pix {
		if i%4 == 3 {
			canvas.Pix[i] = 255
		}
	}
	body := make([]image.Point, len(p.Body))
	seen := make([]bool, len(p.Body))
	for i, k := range p.Body {
		body[i], seen[i] = place(k)
	}
	for i, b := range bodyBones {
		if seen[b.a] && seen[b.b] {
			drawLine(canvas, body[b.a], body[b.b], stick, shade(boneColors[i], limbShade))
		}
	}
	for i := range body {
		if seen[i] {
			drawDisk(canvas, body[i], stick, boneColors[i])
		}
	}
	for _, hand := range [][]Keypoint{p.LeftHand, p.RightHand} {
		drawHand(canvas, hand, place, max(1, stick/2))
	}
	if err := png.Encode(w, canvas); err != nil {
		return fmt.Errorf("posemap: encode PNG: %w", err)
	}
	return nil
}

func drawHand(dst *image.RGBA, hand []Keypoint, place func(Keypoint) (image.Point, bool), radius int) {
	points := make([]image.Point, len(hand))
	seen := make([]bool, len(hand))
	joints := 0
	for i, k := range hand {
		points[i], seen[i] = place(k)
		if seen[i] {
			joints++
		}
	}
	if joints < minHandJoints {
		return
	}
	for i, b := range handBones {
		if seen[b.a] && seen[b.b] {
			drawLine(dst, points[b.a], points[b.b], max(0, radius-1), hue(float64(i)/float64(len(handBones))))
		}
	}
	for i := range points {
		if seen[i] {
			drawDisk(dst, points[i], radius, color.RGBA{0, 0, 255, 255})
		}
	}
}

func shade(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), 255}
}

// hue is the fully saturated color at fraction h of the color wheel.
func hue(h float64) color.RGBA {
	h = h * 6
	x := 1 - math.Abs(math.Mod(h, 2)-1)
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g, b = 1, x, 0
	case 1:
		r, g, b = x, 1, 0
	case 2:
		r, g, b = 0, 1, x
	case 3:
		r, g, b = 0, x, 1
	case 4:
		r, g, b = x, 0, 1
	default:
		r, g, b = 1, 0, x
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func drawLine(dst *image.RGBA, from, to image.Point, radius int, c color.RGBA) {
	dx, dy := abs(to.X-from.X), abs(to.Y-from.Y)
	sx, sy := -1, -1
	if from.X < to.X {
		sx = 1
	}
	if from.Y < to.Y {
		sy = 1
	}
	err := dx - dy
	for {
		drawDisk(dst, from, radius, c)
		if from == to {
			return
		}
		twice := 2 * err
		if twice > -dy {
			err -= dy
			from.X += sx
		}
		if twice < dx {
			err += dx
			from.Y += sy
		}
	}
}

func drawDisk(dst *image.RGBA, center image.Point, radius int, c color.RGBA) {
	for y := center.Y - radius; y <= center.Y+radius; y++ {
		for x := center.X - radius; x <= center.X+radius; x++ {
			dx, dy := x-center.X, y-center.Y
			if dx*dx+dy*dy > radius*radius || !image.Pt(x, y).In(dst.Bounds()) {
				continue
			}
			dst.SetRGBA(x, y, c)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
