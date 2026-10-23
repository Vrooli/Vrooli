package rendering

import (
	"errors"
	"fmt"
	"math"

	"device-studio/internal/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

// PhysicalCameraEndpoint is a typed editorial handoff: it places the active
// screen viewport at a normalized rectangle in the physical output raster.
// Screen corners are TL, TR, BR, BL in world meters after the final device
// pose has been applied. The compiler fits a perspective camera to the planar
// display correspondences; it never warps the rendered image or rotates the
// surrounding scene independently.
type PhysicalCameraEndpoint struct {
	DisplayCorners [4]physical.Vec3
	RasterWidth    uint32
	RasterHeight   uint32
	Target         NormalizedRect
	FocalLengthMM  float64
	SensorWidthMM  float64
	SensorHeightMM float64
}

// PhysicalCameraEndpointGeometry records the physical screen bounds and the
// source-pixel content area after the declared contain fit.
type PhysicalCameraEndpointGeometry struct {
	DisplayTarget  NormalizedRect
	DisplayCorners [4]physical.Vec3
	ContentTarget  NormalizedRect
	ContentCorners [4]physical.Vec3
	ContentWidth   uint32
	ContentHeight  uint32
}

// NormalizedRect uses output-raster boundaries with a top-left origin.
type NormalizedRect struct{ X, Y, Width, Height float64 }

// ContainedContentRect returns the source image's projected rectangle inside
// an active-display rectangle after uniform contain fitting. Letterbox bars
// remain inside the display rectangle and are part of the device surface.
func ContainedContentRect(display NormalizedRect, rasterWidth, rasterHeight, sourceWidth, sourceHeight uint32) (NormalizedRect, error) {
	if !validNormalizedRect(display) || rasterWidth == 0 || rasterHeight == 0 || sourceWidth == 0 || sourceHeight == 0 {
		return NormalizedRect{}, errors.New("contained screen content requires valid display bounds and nonempty raster dimensions")
	}
	displayAspect := display.Width * float64(rasterWidth) / (display.Height * float64(rasterHeight))
	sourceAspect := float64(sourceWidth) / float64(sourceHeight)
	content := display
	if sourceAspect > displayAspect {
		content.Height = display.Width * float64(rasterWidth) / (float64(rasterHeight) * sourceAspect)
		content.Y += (display.Height - content.Height) / 2
	} else {
		content.Width = display.Height * float64(rasterHeight) * sourceAspect / float64(rasterWidth)
		content.X += (display.Width - content.Width) / 2
	}
	return content, nil
}

// DisplayRectForContainedContent is the inverse layout operation: given the
// exact desired projected source rectangle, expand it to the containing
// physical display rectangle without changing source aspect or camera scale.
func DisplayRectForContainedContent(content NormalizedRect, rasterWidth, rasterHeight uint32, displayAspect float64) (NormalizedRect, error) {
	if !validNormalizedRect(content) || rasterWidth == 0 || rasterHeight == 0 || !positiveCameraNumber(displayAspect) {
		return NormalizedRect{}, errors.New("contained screen layout requires valid content bounds, raster dimensions, and display aspect")
	}
	contentAspect := content.Width * float64(rasterWidth) / (content.Height * float64(rasterHeight))
	display := content
	if contentAspect > displayAspect {
		display.Height = content.Width * float64(rasterWidth) / (float64(rasterHeight) * displayAspect)
		display.Y -= (display.Height - content.Height) / 2
	} else {
		display.Width = content.Height * float64(rasterHeight) * displayAspect / float64(rasterWidth)
		display.X -= (display.Width - content.Width) / 2
	}
	if !validNormalizedRect(display) {
		return NormalizedRect{}, errors.New("contained screen display rectangle falls outside the output raster")
	}
	return display, nil
}

func validNormalizedRect(r NormalizedRect) bool {
	return finiteNumber(r.X) && finiteNumber(r.Y) && finiteNumber(r.Width) && finiteNumber(r.Height) &&
		r.X >= 0 && r.Y >= 0 && r.Width > 0 && r.Height > 0 && r.X+r.Width <= 1 && r.Y+r.Height <= 1
}

const defaultCameraEndpointErrorPixels = 1.0

// CompilePhysicalCameraEndpoint returns a camera world pose and the maximum
// corner residual (pixels). It rejects aspect-ratio mismatches rather than
// silently stretching the screen, and rejects endpoints outside the raster.
func CompilePhysicalCameraEndpoint(in PhysicalCameraEndpoint) (physical.TRS, float64, error) {
	if in.RasterWidth == 0 || in.RasterHeight == 0 || !positiveCameraNumber(in.FocalLengthMM) || !positiveCameraNumber(in.SensorWidthMM) || !positiveCameraNumber(in.SensorHeightMM) {
		return physical.TRS{}, 0, errors.New("camera endpoint requires a nonempty raster and positive finite camera intrinsics")
	}
	r := in.Target
	if !finiteNumber(r.X) || !finiteNumber(r.Y) || !finiteNumber(r.Width) || !finiteNumber(r.Height) || r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 || r.X+r.Width > 1 || r.Y+r.Height > 1 {
		return physical.TRS{}, 0, errors.New("camera endpoint rectangle must be finite, nonempty, and contained in the output raster")
	}
	for _, p := range in.DisplayCorners {
		if err := p.Validate(); err != nil {
			return physical.TRS{}, 0, fmt.Errorf("camera endpoint display corner: %w", err)
		}
	}
	add, sub, scale, dot, cross, norm, unit := v3Ops()
	topLeft, topRight, bottomRight, bottomLeft := in.DisplayCorners[0], in.DisplayCorners[1], in.DisplayCorners[2], in.DisplayCorners[3]
	rightEdgeTop := sub(topRight, topLeft)
	rightEdgeBottom := sub(bottomRight, bottomLeft)
	downEdgeLeft := sub(bottomLeft, topLeft)
	downEdgeRight := sub(bottomRight, topRight)
	rightEdge := scale(add(rightEdgeTop, rightEdgeBottom), .5)
	downEdge := scale(add(downEdgeLeft, downEdgeRight), .5)
	w := norm(rightEdge)
	h := norm(downEdge)
	if w < 1e-5 || h < 1e-5 {
		return physical.TRS{}, 0, errors.New("camera endpoint display plane is degenerate")
	}
	right, down := scale(rightEdge, 1/w), scale(downEdge, 1/h)
	_, normalErr := unit(cross(right, down))
	if normalErr != nil {
		return physical.TRS{}, 0, errors.New("camera endpoint display plane is degenerate")
	}
	planeTolerance := math.Max(w, h) * 1e-4
	// Test the fourth point against the plane through three actual corners.
	// The sum of averaged opposite edges always lies in the normal's plane and
	// therefore cannot detect a non-planar fourth corner.
	cornerNormal, cornerNormalErr := unit(cross(rightEdgeTop, downEdgeLeft))
	if cornerNormalErr != nil {
		return physical.TRS{}, 0, errors.New("camera endpoint display plane is degenerate")
	}
	// The UV-sampled active region can be a slightly skewed planar quad even
	// when the display frame's outer bevel is rectangular. Keep the plane check
	// strict; let the bounded corner-residual optimizer account for small edge
	// skew instead of rejecting authored UV geometry up front.
	if math.Abs(dot(right, down)) > w*h*.02 || math.Abs(dot(sub(bottomRight, topLeft), cornerNormal)) > planeTolerance {
		return physical.TRS{}, 0, errors.New("camera endpoint requires a planar near-rectangular display")
	}
	up := scale(down, -1)
	back, err := unit(cross(right, up))
	if err != nil {
		return physical.TRS{}, 0, err
	}
	center := scale(add(add(topLeft, topRight), add(bottomRight, bottomLeft)), .25)
	// Physical scenes pin horizontal sensor fit. Blender keeps horizontal and
	// vertical focal lengths equal in pixel space after deriving the effective
	// sensor height from the output aspect ratio.
	fx := float64(in.RasterWidth) * in.FocalLengthMM / in.SensorWidthMM
	fy := fx
	aspectScreen := w / h
	aspectTarget := (r.Width * float64(in.RasterWidth)) / (r.Height * float64(in.RasterHeight))
	if math.Abs(aspectScreen-aspectTarget) > math.Max(aspectScreen, aspectTarget)*1e-3 {
		return physical.TRS{}, 0, fmt.Errorf("camera endpoint target aspect %.6f does not match contained display viewport aspect %.6f", aspectTarget, aspectScreen)
	}
	distanceW := fx * w / (r.Width * float64(in.RasterWidth))
	distanceH := fy * h / (r.Height * float64(in.RasterHeight))
	distance := (distanceW + distanceH) / 2
	if !positiveCameraNumber(distance) || math.Abs(distanceW-distanceH) > distance*1e-3 {
		return physical.TRS{}, 0, errors.New("camera endpoint raster and display cannot be matched by this camera projection")
	}
	cx := (r.X + r.Width/2 - .5) * in.SensorWidthMM / in.FocalLengthMM * distance
	effectiveSensorHeightMM := in.SensorWidthMM * float64(in.RasterHeight) / float64(in.RasterWidth)
	cy := (r.Y + r.Height/2 - .5) * effectiveSensorHeightMM / in.FocalLengthMM * distance
	position := add(center, add(scale(back, distance), add(scale(right, -cx), scale(up, cy))))
	rotation, err := quaternionFromBasis(right, up, back)
	if err != nil {
		return physical.TRS{}, 0, err
	}
	pose := physical.TRS{Translation: position, Rotation: rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	// Refine the initial aligned solution as a known-camera pose fit. This keeps
	// small source-model trapezoids from forcing screen content off target.
	pose, residual := refineEndpointPose(in, pose, fx, fy)
	return pose, residual, nil
}

func refineEndpointPose(in PhysicalCameraEndpoint, initial physical.TRS, fx, fy float64) (physical.TRS, float64) {
	params := [6]float64{initial.Translation.X, initial.Translation.Y, initial.Translation.Z, 0, 0, 0}
	best := endpointResidualForPose(in, initial, fx, fy)
	lambda := 1e-3
	for iteration := 0; iteration < 40; iteration++ {
		baseResiduals, ok := endpointResidualVector(in, poseFromParameters(initial.Rotation, params), fx, fy)
		if !ok {
			break
		}
		var jac [8][6]float64
		for axis := range params {
			epsilon := 1e-5
			if axis < 3 {
				epsilon = 1e-6
			}
			plus, minus := params, params
			plus[axis] += epsilon
			minus[axis] -= epsilon
			rPlus, okPlus := endpointResidualVector(in, poseFromParameters(initial.Rotation, plus), fx, fy)
			rMinus, okMinus := endpointResidualVector(in, poseFromParameters(initial.Rotation, minus), fx, fy)
			if !okPlus || !okMinus {
				continue
			}
			for row := range baseResiduals {
				jac[row][axis] = (rPlus[row] - rMinus[row]) / (2 * epsilon)
			}
		}
		var normal [6][6]float64
		var gradient [6]float64
		for row, residual := range baseResiduals {
			for a := range params {
				gradient[a] -= jac[row][a] * residual
				for b := range params {
					normal[a][b] += jac[row][a] * jac[row][b]
				}
			}
		}
		for i := range params {
			normal[i][i] += lambda * math.Max(1, normal[i][i])
		}
		step, solved := solveEndpointNormal(normal, gradient)
		if !solved {
			break
		}
		candidateParams := params
		for i := range candidateParams {
			candidateParams[i] += step[i]
		}
		candidate := poseFromParameters(initial.Rotation, candidateParams)
		candidateResidual := endpointResidualForPose(in, candidate, fx, fy)
		if candidateResidual < best {
			params, best, lambda = candidateParams, candidateResidual, math.Max(1e-9, lambda/3)
			if best < .01 {
				break
			}
		} else {
			lambda *= 10
			if lambda > 1e12 {
				break
			}
		}
	}
	pose := poseFromParameters(initial.Rotation, params)
	return pose, endpointResidualForPose(in, pose, fx, fy)
}

func poseFromParameters(base physical.QuatXYZW, p [6]float64) physical.TRS {
	angle := math.Sqrt(p[3]*p[3] + p[4]*p[4] + p[5]*p[5])
	delta := physical.QuatXYZW{W: 1}
	if angle > 1e-12 {
		s := math.Sin(angle/2) / angle
		delta = physical.QuatXYZW{X: p[3] * s, Y: p[4] * s, Z: p[5] * s, W: math.Cos(angle / 2)}
	}
	rotation, err := physical.CanonicalizeQuaternion(multiplyCameraQuaternion(delta, base))
	if err != nil {
		rotation = base
	}
	return physical.TRS{Translation: physical.Vec3{X: p[0], Y: p[1], Z: p[2]}, Rotation: rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

func endpointResidualForPose(in PhysicalCameraEndpoint, camera physical.TRS, fx, fy float64) float64 {
	right := rotateCameraVector(camera.Rotation, physical.Vec3{X: 1})
	up := rotateCameraVector(camera.Rotation, physical.Vec3{Y: 1})
	back := rotateCameraVector(camera.Rotation, physical.Vec3{Z: 1})
	return endpointResidual(in, camera, right, up, back, fx, fy)
}

func endpointResidualVector(in PhysicalCameraEndpoint, camera physical.TRS, fx, fy float64) ([8]float64, bool) {
	var residuals [8]float64
	right := rotateCameraVector(camera.Rotation, physical.Vec3{X: 1})
	up := rotateCameraVector(camera.Rotation, physical.Vec3{Y: 1})
	back := rotateCameraVector(camera.Rotation, physical.Vec3{Z: 1})
	_, sub, _, dot, _, _, _ := v3Ops()
	w, h := in.Target.Width*float64(in.RasterWidth), in.Target.Height*float64(in.RasterHeight)
	cx, cy := (in.Target.X+in.Target.Width/2)*float64(in.RasterWidth), (in.Target.Y+in.Target.Height/2)*float64(in.RasterHeight)
	want := [4][2]float64{{cx - w/2, cy - h/2}, {cx + w/2, cy - h/2}, {cx + w/2, cy + h/2}, {cx - w/2, cy + h/2}}
	for i, point := range in.DisplayCorners {
		d := sub(point, camera.Translation)
		z := -dot(d, back)
		if z <= 1e-9 || !finiteNumber(z) {
			return residuals, false
		}
		px := float64(in.RasterWidth)/2 + fx*dot(d, right)/z
		py := float64(in.RasterHeight)/2 - fy*dot(d, up)/z
		residuals[i*2], residuals[i*2+1] = px-want[i][0], py-want[i][1]
	}
	return residuals, true
}

func solveEndpointNormal(matrix [6][6]float64, vector [6]float64) ([6]float64, bool) {
	for column := 0; column < 6; column++ {
		pivot := column
		for row := column + 1; row < 6; row++ {
			if math.Abs(matrix[row][column]) > math.Abs(matrix[pivot][column]) {
				pivot = row
			}
		}
		if math.Abs(matrix[pivot][column]) < 1e-15 {
			return [6]float64{}, false
		}
		matrix[column], matrix[pivot] = matrix[pivot], matrix[column]
		vector[column], vector[pivot] = vector[pivot], vector[column]
		divisor := matrix[column][column]
		for j := column; j < 6; j++ {
			matrix[column][j] /= divisor
		}
		vector[column] /= divisor
		for row := 0; row < 6; row++ {
			if row == column {
				continue
			}
			factor := matrix[row][column]
			for j := column; j < 6; j++ {
				matrix[row][j] -= factor * matrix[column][j]
			}
			vector[row] -= factor * vector[column]
		}
	}
	return vector, true
}

// CompilePhysicalCameraEndpointBinding is the protobuf boundary used by API,
// CLI and saved shot workflows. Callers provide screen corners from the pinned
// device geometry after its final sampled pose.
func CompilePhysicalCameraEndpointBinding(endpoint *shotsv1.PhysicalCameraEndpoint, corners [4]physical.Vec3, focalMM, sensorWidthMM, sensorHeightMM float64) (physical.TRS, float64, error) {
	return CompilePhysicalCameraEndpointBindingWithContentCorners(endpoint, corners, nil, focalMM, sensorWidthMM, sensorHeightMM)
}

func CompilePhysicalCameraEndpointBindingWithContentCorners(endpoint *shotsv1.PhysicalCameraEndpoint, corners [4]physical.Vec3, mappedContentCorners *[4]physical.Vec3, focalMM, sensorWidthMM, sensorHeightMM float64) (physical.TRS, float64, error) {
	if endpoint == nil || endpoint.GetDisplayId() == "" {
		return physical.TRS{}, 0, errors.New("camera endpoint must identify a display")
	}
	geometry, err := ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint, corners, mappedContentCorners)
	if err != nil {
		return physical.TRS{}, 0, err
	}
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: geometry.ContentCorners, RasterWidth: endpoint.GetRasterWidth(), RasterHeight: endpoint.GetRasterHeight(), Target: geometry.ContentTarget, FocalLengthMM: focalMM, SensorWidthMM: sensorWidthMM, SensorHeightMM: sensorHeightMM})
	if err != nil {
		return physical.TRS{}, 0, err
	}
	tolerance := endpoint.GetMaxErrorPixels()
	if tolerance == 0 {
		tolerance = defaultCameraEndpointErrorPixels
	}
	if !positiveCameraNumber(tolerance) || tolerance > 8 {
		return physical.TRS{}, 0, errors.New("camera endpoint pixel tolerance must be in (0,8]")
	}
	if residual > tolerance {
		return physical.TRS{}, residual, fmt.Errorf("compiled camera endpoint residual %.3fpx exceeds %.3fpx tolerance", residual, tolerance)
	}
	return pose, residual, nil
}

// ResolvePhysicalCameraEndpointGeometry preserves the historical active-
// display target behavior and adds an opt-in source-content target. The latter
// maps 16:9 recording pixels into the active display with a centered contain
// fit, so camera placement respects letterboxing instead of stretching it.
func ResolvePhysicalCameraEndpointGeometry(endpoint *shotsv1.PhysicalCameraEndpoint, displayCorners [4]physical.Vec3) (PhysicalCameraEndpointGeometry, error) {
	return ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint, displayCorners, nil)
}

// ResolvePhysicalCameraEndpointGeometryWithContentCorners accepts exact
// source-content corners sampled from the indexed GLB UV triangles. The
// display quad still determines the contain layout; actual UV samples replace
// the bilinear approximation for camera fitting.
func ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint *shotsv1.PhysicalCameraEndpoint, displayCorners [4]physical.Vec3, mappedContentCorners *[4]physical.Vec3) (PhysicalCameraEndpointGeometry, error) {
	if endpoint == nil || endpoint.GetDisplayId() == "" || endpoint.GetRasterWidth() == 0 || endpoint.GetRasterHeight() == 0 {
		return PhysicalCameraEndpointGeometry{}, errors.New("camera endpoint requires a selected display and nonempty output raster")
	}
	requested := NormalizedRect{X: endpoint.GetX(), Y: endpoint.GetY(), Width: endpoint.GetWidth(), Height: endpoint.GetHeight()}
	geometry := PhysicalCameraEndpointGeometry{DisplayCorners: displayCorners, ContentCorners: displayCorners, DisplayTarget: requested, ContentTarget: requested}
	switch endpoint.GetTargetKind() {
	case shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_ACTIVE_DISPLAY:
		if endpoint.GetContentWidthPixels() != 0 || endpoint.GetContentHeightPixels() != 0 {
			return PhysicalCameraEndpointGeometry{}, errors.New("active-display target cannot declare a recording-content raster")
		}
	case shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT:
		if endpoint.GetContentWidthPixels() == 0 || endpoint.GetContentHeightPixels() == 0 {
			return PhysicalCameraEndpointGeometry{}, errors.New("recording-content target requires its source raster dimensions")
		}
		displayAspect, err := physicalDisplayAspect(displayCorners)
		if err != nil {
			return PhysicalCameraEndpointGeometry{}, err
		}
		displayTarget, err := DisplayRectForContainedContent(requested, endpoint.GetRasterWidth(), endpoint.GetRasterHeight(), displayAspect)
		if err != nil {
			return PhysicalCameraEndpointGeometry{}, err
		}
		contentTarget, err := ContainedContentRect(displayTarget, endpoint.GetRasterWidth(), endpoint.GetRasterHeight(), endpoint.GetContentWidthPixels(), endpoint.GetContentHeightPixels())
		if err != nil {
			return PhysicalCameraEndpointGeometry{}, err
		}
		if math.Abs(contentTarget.X-requested.X)*float64(endpoint.GetRasterWidth()) > .01 || math.Abs(contentTarget.Y-requested.Y)*float64(endpoint.GetRasterHeight()) > .01 ||
			math.Abs(contentTarget.Width-requested.Width)*float64(endpoint.GetRasterWidth()) > .01 || math.Abs(contentTarget.Height-requested.Height)*float64(endpoint.GetRasterHeight()) > .01 {
			return PhysicalCameraEndpointGeometry{}, errors.New("recording-content target aspect must match its source raster")
		}
		contentCorners, err := physicalContainedContentCorners(displayCorners, endpoint.GetContentWidthPixels(), endpoint.GetContentHeightPixels())
		if err != nil {
			return PhysicalCameraEndpointGeometry{}, err
		}
		if mappedContentCorners != nil {
			contentCorners = *mappedContentCorners
			if !physicalPlanarQuad(contentCorners, .02) {
				return PhysicalCameraEndpointGeometry{}, errors.New("UV-mapped recording-content corners must form a planar near-rectangle")
			}
			contentAspect := distance(contentCorners[0], contentCorners[1]) / distance(contentCorners[0], contentCorners[3])
			sourceAspect := float64(endpoint.GetContentWidthPixels()) / float64(endpoint.GetContentHeightPixels())
			if !positiveCameraNumber(contentAspect) || math.Abs(contentAspect-sourceAspect) > sourceAspect*.02 {
				return PhysicalCameraEndpointGeometry{}, errors.New("UV-mapped recording-content geometry differs from its source raster aspect")
			}
		}
		geometry.DisplayTarget = displayTarget
		geometry.ContentTarget = contentTarget
		geometry.ContentCorners = contentCorners
		geometry.ContentWidth = endpoint.GetContentWidthPixels()
		geometry.ContentHeight = endpoint.GetContentHeightPixels()
	default:
		return PhysicalCameraEndpointGeometry{}, errors.New("camera endpoint target kind is unsupported")
	}
	return geometry, nil
}

func physicalDisplayAspect(corners [4]physical.Vec3) (float64, error) {
	width := (distance(corners[0], corners[1]) + distance(corners[3], corners[2])) / 2
	height := (distance(corners[0], corners[3]) + distance(corners[1], corners[2])) / 2
	if !positiveCameraNumber(width) || !positiveCameraNumber(height) {
		return 0, errors.New("camera endpoint active display has degenerate physical bounds")
	}
	return width / height, nil
}

func physicalContainedContentCorners(display [4]physical.Vec3, sourceWidth, sourceHeight uint32) ([4]physical.Vec3, error) {
	var out [4]physical.Vec3
	displayAspect, err := physicalDisplayAspect(display)
	if err != nil {
		return out, err
	}
	if sourceWidth == 0 || sourceHeight == 0 {
		return out, errors.New("contained screen content requires a nonempty source raster")
	}
	sourceAspect := float64(sourceWidth) / float64(sourceHeight)
	x0, x1, y0, y1 := 0.0, 1.0, 0.0, 1.0
	if sourceAspect > displayAspect {
		visibleHeight := displayAspect / sourceAspect
		y0, y1 = (1-visibleHeight)/2, (1+visibleHeight)/2
	} else {
		visibleWidth := sourceAspect / displayAspect
		x0, x1 = (1-visibleWidth)/2, (1+visibleWidth)/2
	}
	lerp := func(a, b physical.Vec3, t float64) physical.Vec3 {
		return physical.Vec3{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t, Z: a.Z + (b.Z-a.Z)*t}
	}
	bilinear := func(u, v float64) physical.Vec3 {
		left := lerp(display[0], display[3], v)
		right := lerp(display[1], display[2], v)
		return lerp(left, right, u)
	}
	out = [4]physical.Vec3{bilinear(x0, y0), bilinear(x1, y0), bilinear(x1, y1), bilinear(x0, y1)}
	return out, nil
}

func distance(a, b physical.Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// CameraEndpointRelativeTransform expresses an absolute camera world pose as
// the local delta applied after the authored scene camera frame.
func CameraEndpointRelativeTransform(base, target physical.TRS) (physical.TRS, error) {
	base, err := physical.CanonicalizeTRS(base)
	if err != nil {
		return physical.TRS{}, err
	}
	target, err = physical.CanonicalizeTRS(target)
	if err != nil {
		return physical.TRS{}, err
	}
	if math.Abs(base.Scale.X-1) > 1e-8 || math.Abs(base.Scale.Y-1) > 1e-8 || math.Abs(base.Scale.Z-1) > 1e-8 {
		return physical.TRS{}, errors.New("selected scene camera frame must have unit scale")
	}
	inv := physical.QuatXYZW{X: -base.Rotation.X, Y: -base.Rotation.Y, Z: -base.Rotation.Z, W: base.Rotation.W}
	deltaPosition := rotateCameraVector(inv, physical.Vec3{X: target.Translation.X - base.Translation.X, Y: target.Translation.Y - base.Translation.Y, Z: target.Translation.Z - base.Translation.Z})
	return physical.CanonicalizeTRS(physical.TRS{Translation: deltaPosition, Rotation: multiplyCameraQuaternion(inv, target.Rotation), Scale: physical.Vec3{X: 1, Y: 1, Z: 1}})
}

// InterpolateCameraTransform creates a deterministic identity-to-endpoint
// local camera path. The output schedule supplies exact ordinal/time binding.
func InterpolateCameraTransform(target physical.TRS, progress float64) physical.TRS {
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	if progress <= 0 {
		return identity
	}
	if progress >= 1 {
		return target
	}
	q := slerpCamera(identity.Rotation, target.Rotation, progress)
	return physical.TRS{Translation: physical.Vec3{X: target.Translation.X * progress, Y: target.Translation.Y * progress, Z: target.Translation.Z * progress}, Rotation: q, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

// PhysicalCameraEndpointSampleProgress derives the camera's normalized
// identity-to-endpoint progress from the exact requested motion ticks. A zero
// arrival tick keeps the legacy behavior of reaching the target at the final
// scheduled sample; otherwise the camera reaches it at the requested tick and
// holds for all later samples.
func PhysicalCameraEndpointSampleProgress(arrivalTick uint32, samples []PhysicalMotionSample) ([]float64, error) {
	if len(samples) < 2 {
		return nil, errors.New("camera endpoint schedule requires at least two motion samples")
	}
	ticks := make([]float64, len(samples))
	for i, sample := range samples {
		if sample.RequestedMotionTick.Den <= 0 {
			return nil, fmt.Errorf("camera endpoint sample %d has a nonpositive motion-tick denominator", i)
		}
		tick := float64(sample.RequestedMotionTick.Num) / float64(sample.RequestedMotionTick.Den)
		if math.IsNaN(tick) || math.IsInf(tick, 0) {
			return nil, fmt.Errorf("camera endpoint sample %d has a nonfinite motion tick", i)
		}
		if i > 0 && tick <= ticks[i-1] {
			return nil, errors.New("camera endpoint motion samples must have strictly increasing ticks")
		}
		ticks[i] = tick
	}
	first, last := ticks[0], ticks[len(ticks)-1]
	arrival := float64(arrivalTick)
	if arrivalTick == 0 {
		arrival = last
	}
	if arrival < first || arrival > last {
		return nil, fmt.Errorf("camera endpoint arrival tick %.3f must be within scheduled motion range [%.3f, %.3f]", arrival, first, last)
	}
	progress := make([]float64, len(samples))
	for i, tick := range ticks {
		if tick >= arrival || arrival == first {
			progress[i] = 1
		} else {
			progress[i] = (tick - first) / (arrival - first)
		}
	}
	return progress, nil
}

func multiplyCameraQuaternion(a, b physical.QuatXYZW) physical.QuatXYZW {
	return physical.QuatXYZW{X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y, Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X, Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W, W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z}
}

func rotateCameraVector(q physical.QuatXYZW, v physical.Vec3) physical.Vec3 {
	qv := physical.QuatXYZW{X: v.X, Y: v.Y, Z: v.Z}
	r := multiplyCameraQuaternion(multiplyCameraQuaternion(q, qv), physical.QuatXYZW{X: -q.X, Y: -q.Y, Z: -q.Z, W: q.W})
	return physical.Vec3{X: r.X, Y: r.Y, Z: r.Z}
}

func slerpCamera(a, b physical.QuatXYZW, t float64) physical.QuatXYZW {
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
	if dot < 0 {
		b = physical.QuatXYZW{X: -b.X, Y: -b.Y, Z: -b.Z, W: -b.W}
		dot = -dot
	}
	if dot > .9995 {
		q, _ := physical.CanonicalizeQuaternion(physical.QuatXYZW{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t, Z: a.Z + (b.Z-a.Z)*t, W: a.W + (b.W-a.W)*t})
		return q
	}
	theta := math.Acos(math.Max(-1, math.Min(1, dot)))
	denom := math.Sin(theta)
	wa, wb := math.Sin((1-t)*theta)/denom, math.Sin(t*theta)/denom
	q, _ := physical.CanonicalizeQuaternion(physical.QuatXYZW{X: wa*a.X + wb*b.X, Y: wa*a.Y + wb*b.Y, Z: wa*a.Z + wb*b.Z, W: wa*a.W + wb*b.W})
	return q
}

type vectorOps struct {
	add, sub, scale func(physical.Vec3, float64) physical.Vec3
	dot             func(physical.Vec3, physical.Vec3) float64
	cross           func(physical.Vec3, physical.Vec3) physical.Vec3
	norm            func(physical.Vec3) float64
	unit            func(physical.Vec3) (physical.Vec3, error)
}

func v3Ops() (func(physical.Vec3, physical.Vec3) physical.Vec3, func(physical.Vec3, physical.Vec3) physical.Vec3, func(physical.Vec3, float64) physical.Vec3, func(physical.Vec3, physical.Vec3) float64, func(physical.Vec3, physical.Vec3) physical.Vec3, func(physical.Vec3) float64, func(physical.Vec3) (physical.Vec3, error)) {
	add := func(a, b physical.Vec3) physical.Vec3 { return physical.Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
	sub := func(a, b physical.Vec3) physical.Vec3 { return physical.Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }
	scale := func(a physical.Vec3, s float64) physical.Vec3 {
		return physical.Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
	}
	dot := func(a, b physical.Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
	cross := func(a, b physical.Vec3) physical.Vec3 {
		return physical.Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
	}
	norm := func(a physical.Vec3) float64 { return math.Sqrt(dot(a, a)) }
	unit := func(a physical.Vec3) (physical.Vec3, error) {
		n := norm(a)
		if !positiveCameraNumber(n) {
			return physical.Vec3{}, errors.New("camera endpoint basis is degenerate")
		}
		return scale(a, 1/n), nil
	}
	return add, sub, scale, dot, cross, norm, unit
}

func quaternionFromBasis(right, up, back physical.Vec3) (physical.QuatXYZW, error) {
	// Matrix columns are the camera's local +X, +Y, +Z axes in world space.
	m00, m01, m02 := right.X, up.X, back.X
	m10, m11, m12 := right.Y, up.Y, back.Y
	m20, m21, m22 := right.Z, up.Z, back.Z
	trace := m00 + m11 + m22
	var q physical.QuatXYZW
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1) * 2
		q = physical.QuatXYZW{X: (m21 - m12) / s, Y: (m02 - m20) / s, Z: (m10 - m01) / s, W: .25 * s}
	case m00 > m11 && m00 > m22:
		s := math.Sqrt(1+m00-m11-m22) * 2
		q = physical.QuatXYZW{X: .25 * s, Y: (m01 + m10) / s, Z: (m02 + m20) / s, W: (m21 - m12) / s}
	case m11 > m22:
		s := math.Sqrt(1+m11-m00-m22) * 2
		q = physical.QuatXYZW{X: (m01 + m10) / s, Y: .25 * s, Z: (m12 + m21) / s, W: (m02 - m20) / s}
	default:
		s := math.Sqrt(1+m22-m00-m11) * 2
		q = physical.QuatXYZW{X: (m02 + m20) / s, Y: (m12 + m21) / s, Z: .25 * s, W: (m10 - m01) / s}
	}
	return physical.CanonicalizeQuaternion(q)
}

func endpointResidual(in PhysicalCameraEndpoint, camera physical.TRS, right, up, back physical.Vec3, fx, fy float64) float64 {
	_, sub, _, dot, _, _, _ := v3Ops()
	w, h := in.Target.Width*float64(in.RasterWidth), in.Target.Height*float64(in.RasterHeight)
	cx, cy := (in.Target.X+in.Target.Width/2)*float64(in.RasterWidth), (in.Target.Y+in.Target.Height/2)*float64(in.RasterHeight)
	want := [4][2]float64{{cx - w/2, cy - h/2}, {cx + w/2, cy - h/2}, {cx + w/2, cy + h/2}, {cx - w/2, cy + h/2}}
	max := 0.0
	for i, p := range in.DisplayCorners {
		d := sub(p, camera.Translation)
		// Camera-local forward is -Z; back is its +Z world axis.
		z := -dot(d, back)
		x := dot(d, right)
		y := dot(d, up)
		if z <= 0 {
			return math.Inf(1)
		}
		px := float64(in.RasterWidth)/2 + fx*x/z
		py := float64(in.RasterHeight)/2 - fy*y/z
		e := math.Hypot(px-want[i][0], py-want[i][1])
		if e > max {
			max = e
		}
	}
	return max
}

func positiveCameraNumber(value float64) bool { return finiteNumber(value) && value > 0 }
