// Package posemap converts MediaPipe Pose Landmarker landmarks to the COCO-18
// colored bone image consumed by the existing OpenPose ControlNet adapter.
// It contains no model runtime or external dependency; detection is supplied by
// the caller and can therefore be validated independently from inference.
package posemap

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

const (
	MediaPipeLandmarkCount = 33
	COCOKeypointCount      = 18
)

// Landmark is a normalized MediaPipe image-space point. Visibility is the
// detector's visibility score, in [0,1]. Coordinates outside the image or
// points below the configured visibility threshold are omitted.
type Landmark struct {
	X, Y       float64
	Visibility float64
}

// Point is a normalized COCO-18 keypoint. Invalid points have Visible=false.
type Point struct {
	X, Y    float64
	Visible bool
}

var ErrLandmarkCount = errors.New("posemap: expected 33 MediaPipe pose landmarks")

// COCO18 maps MediaPipe's 33 body landmarks into the 18 OpenPose body joints.
// Neck and mid-hip are derived from the corresponding shoulder/hip pairs.
func COCO18(in []Landmark, minVisibility float64) ([]Point, error) {
	if len(in) != MediaPipeLandmarkCount {
		return nil, fmt.Errorf("%w, got %d", ErrLandmarkCount, len(in))
	}
	if math.IsNaN(minVisibility) || math.IsInf(minVisibility, 0) || minVisibility < 0 || minVisibility > 1 {
		return nil, errors.New("posemap: visibility threshold must be finite and between 0 and 1")
	}
	valid := func(index int) Point {
		p := in[index]
		if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) ||
			math.IsNaN(p.Visibility) || math.IsInf(p.Visibility, 0) || p.Visibility < minVisibility ||
			p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
			return Point{}
		}
		return Point{X: p.X, Y: p.Y, Visible: true}
	}
	midpoint := func(a, b int) Point {
		left, right := valid(a), valid(b)
		if !left.Visible || !right.Visible {
			return Point{}
		}
		return Point{X: (left.X + right.X) / 2, Y: (left.Y + right.Y) / 2, Visible: true}
	}
	return []Point{
		valid(0),                        // nose
		midpoint(11, 12),                // neck: shoulders
		valid(12), valid(14), valid(16), // right shoulder, elbow, wrist
		valid(11), valid(13), valid(15), // left shoulder, elbow, wrist
		valid(24), valid(26), valid(28), // right hip, knee, ankle
		valid(23), valid(25), valid(27), // left hip, knee, ankle
		valid(5), valid(2), valid(8), valid(7), // right/left eye and ear
	}, nil
}

type bone struct{ a, b int }

var coco18Bones = [...]bone{
	{1, 2}, {1, 5}, {2, 3}, {3, 4}, {5, 6}, {6, 7}, {1, 8}, {8, 9}, {9, 10},
	{1, 11}, {11, 12}, {12, 13}, {0, 14}, {14, 16}, {0, 15}, {15, 17}, {2, 16}, {5, 17},
}

var boneColors = [...]color.RGBA{
	{255, 0, 0, 255}, {255, 85, 0, 255}, {255, 170, 0, 255}, {255, 255, 0, 255},
	{170, 255, 0, 255}, {85, 255, 0, 255}, {0, 255, 0, 255}, {0, 255, 85, 255},
	{0, 255, 170, 255}, {0, 255, 255, 255}, {0, 170, 255, 255}, {0, 85, 255, 255},
	{0, 0, 255, 255}, {85, 0, 255, 255}, {170, 0, 255, 255}, {255, 0, 255, 255},
	{255, 0, 170, 255}, {255, 0, 85, 255},
}

// WritePNG draws the valid COCO-18 joints and limbs on the black background
// expected by OpenPose ControlNet. Output dimensions are explicit so callers
// can match the generation canvas and reserve bounded scratch space.
func WritePNG(w io.Writer, width, height int, points []Point) error {
	if width < 1 || height < 1 || width > 4096 || height > 4096 {
		return errors.New("posemap: output dimensions must be between 1 and 4096")
	}
	if len(points) != COCOKeypointCount {
		return fmt.Errorf("posemap: expected 18 COCO keypoints, got %d", len(points))
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	coords := make([]image.Point, len(points))
	for i, p := range points {
		if !p.Visible || !finiteUnit(p.X) || !finiteUnit(p.Y) {
			continue
		}
		coords[i] = image.Pt(int(math.Round(p.X*float64(width-1))), int(math.Round(p.Y*float64(height-1))))
	}
	for i, b := range coco18Bones {
		if points[b.a].Visible && points[b.b].Visible && finiteUnit(points[b.a].X) && finiteUnit(points[b.a].Y) && finiteUnit(points[b.b].X) && finiteUnit(points[b.b].Y) {
			drawLine(canvas, coords[b.a], coords[b.b], 3, boneColors[i])
		}
	}
	for i, p := range points {
		if p.Visible && finiteUnit(p.X) && finiteUnit(p.Y) {
			drawDisk(canvas, coords[i], 4, boneColors[i%len(boneColors)])
		}
	}
	if err := png.Encode(w, canvas); err != nil {
		return fmt.Errorf("posemap: encode PNG: %w", err)
	}
	return nil
}

func finiteUnit(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }

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
