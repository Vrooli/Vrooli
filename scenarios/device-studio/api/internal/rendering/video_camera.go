package rendering

import (
	"errors"
	"image"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
)

const (
	ScreenEntryCameraProfileID           = "screen-entry-projective-contain-v1"
	ScreenEntryCameraProfileVersion      = 1
	StationaryScreenEntryCameraProfileID = "screen-entry-stationary-scene-v1"
	videoCameraMaxFrames                 = 360
)

// Camera pins the bounded photographic screen-entry motion. It contains no
// source geometry: the viewport comes from the presentation profile.
type Camera struct {
	ProfileID            string                   `json:"profile_id"`
	ProfileVersion       uint32                   `json:"profile_version"`
	DurationOutputFrames uint32                   `json:"duration_output_frames"`
	MatteRGB             *presentations.RGB       `json:"matte_rgb"`
	TargetRect           *presentations.BasisRect `json:"target_rect,omitempty"`
	SceneRect            *presentations.BasisRect `json:"scene_rect,omitempty"`
}

type videoCameraPath struct {
	from, target [4]composition.Point
	viewport     image.Rectangle
	duration     uint32
	matte        presentations.RGB
	frames       [][4]composition.Point
}

func newVideoCameraPath(m VideoManifest, plateBounds image.Rectangle, layout *presentations.Layout) (*videoCameraPath, error) {
	if m.Camera == nil || m.Camera.MatteRGB == nil || m.Presentation == nil || m.Scene.PlateWidth <= 0 || m.Scene.PlateHeight <= 0 {
		return nil, errors.New("camera profile and pinned plate dimensions are required")
	}
	if plateBounds.Dx() != m.Scene.PlateWidth || plateBounds.Dy() != m.Scene.PlateHeight {
		return nil, errors.New("decoded camera plate dimensions differ from pinned scene metadata")
	}
	fit, err := cameraSceneFit(image.Pt(m.Scene.PlateWidth, m.Scene.PlateHeight), image.Pt(m.Width, m.Height), m.Camera.SceneRect)
	if err != nil {
		return nil, errors.New("camera plate cannot be fit into the output raster: " + err.Error())
	}
	frames, err := plannedVideoFrameCount(m)
	if err != nil {
		return nil, err
	}
	if frames > videoCameraMaxFrames {
		return nil, errors.New("screen-entry camera video is limited to 360 output frames")
	}
	if m.Camera.DurationOutputFrames < 2 || m.Camera.DurationOutputFrames > videoCameraMaxFrames || int64(m.Camera.DurationOutputFrames) > frames {
		return nil, errors.New("camera transition duration must be 2..360 output frames and fit the clip")
	}
	if err := composition.ValidatePlateQuad(m.Scene.PlateWidth, m.Scene.PlateHeight, m.Scene.SurfaceWidth, m.Scene.SurfaceHeight, videoSceneQuad(m.Scene)); err != nil {
		return nil, err
	}
	var viewport image.Rectangle
	sourceBounds := image.Rect(0, 0, m.Inspection.Width, m.Inspection.Height)
	if layout != nil {
		if layout.DisplaySize != image.Pt(m.Scene.SurfaceWidth, m.Scene.SurfaceHeight) {
			return nil, errors.New("camera presentation layout dimensions differ from scene surface")
		}
		viewport = layout.AppViewport
	} else {
		viewport, err = presentations.BuildVideoViewport(sourceBounds, *m.Presentation)
		if err != nil {
			return nil, err
		}
	}
	if viewport.Empty() || viewport.Min.X < 0 || viewport.Min.Y < 0 || viewport.Max.X > m.Scene.SurfaceWidth || viewport.Max.Y > m.Scene.SurfaceHeight {
		return nil, errors.New("camera active viewport is outside the presentation raster")
	}
	var outputQuad [4]composition.Point
	for i, point := range videoSceneQuad(m.Scene) {
		outputQuad[i] = composition.Point{X: point.X*fit.scale + fit.offsetX, Y: point.Y*fit.scale + fit.offsetY}
	}
	projected, err := composition.ProjectRasterRect(image.Pt(m.Scene.SurfaceWidth, m.Scene.SurfaceHeight), image.Pt(m.Width, m.Height), outputQuad, image.Rect(0, 0, m.Scene.SurfaceWidth, m.Scene.SurfaceHeight), viewport)
	if err != nil {
		return nil, err
	}
	target := containedViewportQuad(viewport, image.Pt(m.Width, m.Height))
	if r := m.Camera.TargetRect; r != nil {
		if !r.Valid() {
			return nil, errors.New("camera target rectangle must have positive size and lie within basis 10000")
		}
		// Keep subpixel boundaries until rasterization, matching the following
		// composition's normalized placement without integer-rounding drift.
		x, y := float64(r.X)*float64(m.Width)/10000, float64(r.Y)*float64(m.Height)/10000
		w, h := float64(r.Width)*float64(m.Width)/10000, float64(r.Height)*float64(m.Height)/10000
		scale := min(w/float64(viewport.Dx()), h/float64(viewport.Dy()))
		vw, vh := float64(viewport.Dx())*scale, float64(viewport.Dy())*scale
		left, top := x+(w-vw)/2, y+(h-vh)/2
		target = [4]composition.Point{{X: left, Y: top}, {X: left + vw, Y: top}, {X: left + vw, Y: top + vh}, {X: left, Y: top + vh}}
	}
	path := &videoCameraPath{from: projected, target: target, viewport: viewport, duration: m.Camera.DurationOutputFrames, matte: *m.Camera.MatteRGB}
	path.frames = make([][4]composition.Point, frames)
	for ordinal := int64(0); ordinal < frames; ordinal++ {
		q := path.quad(uint32(ordinal))
		if err := composition.ValidatePlateQuad(m.Width, m.Height, viewport.Dx(), viewport.Dy(), q); err != nil {
			return nil, errors.New("camera interpolation leaves a finite convex output path: " + err.Error())
		}
		if m.Camera.ProfileID == ScreenEntryCameraProfileID {
			if err := composition.ValidateProjectiveWarp(image.Pt(m.Width, m.Height), path.from, q); err != nil {
				return nil, err
			}
		}
		path.frames[ordinal] = q
	}
	return path, nil
}

func (p *videoCameraPath) quad(ordinal uint32) [4]composition.Point {
	if ordinal >= p.duration-1 {
		return p.target
	}
	if int(ordinal) < len(p.frames) && p.frames[ordinal] != ([4]composition.Point{}) {
		return p.frames[ordinal]
	}
	u := float64(ordinal) / float64(p.duration-1)
	eased := u * u * (3 - 2*u)
	q, _ := composition.InterpolateQuad(p.from, p.target, eased)
	return q
}

func (p *videoCameraPath) blend(ordinal uint32) (float64, float64) {
	if ordinal >= p.duration-1 {
		return 1, 1
	}
	u := float64(ordinal) / float64(p.duration-1)
	if u <= 0.75 {
		return 0, 0
	}
	v := (u - 0.75) / 0.25
	weight := v * v * (3 - 2*v)
	return weight, weight
}

func containedViewportQuad(viewport image.Rectangle, output image.Point) [4]composition.Point {
	xScale := float64(output.X) / float64(viewport.Dx())
	yScale := float64(output.Y) / float64(viewport.Dy())
	var x, y, w, h float64
	if xScale <= yScale {
		scale := xScale
		w, h = float64(output.X), float64(viewport.Dy())*scale
		y = (float64(output.Y) - h) / 2
	} else {
		scale := yScale
		w, h = float64(viewport.Dx())*scale, float64(output.Y)
		x = (float64(output.X) - w) / 2
	}
	return [4]composition.Point{{X: x, Y: y}, {X: x + w, Y: y}, {X: x + w, Y: y + h}, {X: x, Y: y + h}}
}

func videoSceneQuad(scene VideoScene) [4]composition.Point {
	var out [4]composition.Point
	for i, point := range scene.Quad {
		out[i] = composition.Point{X: point.X, Y: point.Y}
	}
	return out
}

func cameraSceneFit(source, output image.Point, r *presentations.BasisRect) (videoFit, error) {
	if r == nil {
		return videoContainFit(source, output)
	}
	if !r.Valid() {
		return videoFit{}, errors.New("camera scene rectangle must have positive size and lie within basis 10000")
	}
	w, h := float64(output.X)*float64(r.Width)/10000, float64(output.Y)*float64(r.Height)/10000
	scale := min(w/float64(source.X), h/float64(source.Y))
	return videoFit{scale: scale, offsetX: float64(output.X)*float64(r.X)/10000 + (w-float64(source.X)*scale)/2, offsetY: float64(output.Y)*float64(r.Y)/10000 + (h-float64(source.Y)*scale)/2}, nil
}
