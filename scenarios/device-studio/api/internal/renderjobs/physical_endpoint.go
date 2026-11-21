package renderjobs

import (
	"errors"
	"fmt"
	"math"

	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

// applyPhysicalCameraEndpoint compiles the editorial screen handoff against
// the final sampled device pose, then inserts a deterministic camera path into
// the native worker schedule. Joint motion is deliberately rejected: panel
// corners must currently be recoverable from a rigid, static-lid candidate.
func applyPhysicalCameraEndpoint(
	endpoint *shotsv1.PhysicalCameraEndpoint,
	displayCorners [4]physical.Vec3,
	mappedContentCorners *[4]physical.Vec3,
	cameras []scenes.SceneCameraFrame,
	cameraID string,
	sceneWorld physical.TRS,
	pivot physical.Vec3,
	deviceRest physical.TRS,
	samples []rendering.PhysicalMotionSample,
	width, height uint32,
) (*rendering.PhysicalCameraEndpointPlan, error) {
	if endpoint == nil {
		return nil, nil
	}
	if len(samples) < 2 {
		return nil, errors.New("physical camera endpoint requires at least two scheduled motion samples")
	}
	var camera *scenes.SceneCameraFrame
	for i := range cameras {
		if cameras[i].ID == cameraID {
			camera = &cameras[i]
			break
		}
	}
	if camera == nil || camera.Projection != scenes.ProjectionPerspective || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil || camera.SensorHeightMM == nil || camera.SensorFit != scenes.SensorFitHorizontal {
		return nil, errors.New("physical camera endpoint requires the selected perspective camera with horizontal sensor fit and explicit intrinsics")
	}
	if endpoint.GetRasterWidth() != width || endpoint.GetRasterHeight() != height {
		return nil, errors.New("physical camera endpoint raster differs from the compiled output")
	}
	last := samples[len(samples)-1]
	if len(last.JointAngles) != 0 {
		return nil, errors.New("physical camera endpoint cannot be compiled with joint motion; qualify a static-lid device candidate first")
	}
	if _, exists := last.CameraTransforms[cameraID]; exists {
		return nil, errors.New("physical camera endpoint conflicts with an authored track for the selected camera")
	}

	root := identityTRS()
	if last.DeviceRoot != nil {
		root = *last.DeviceRoot
	}
	deviceWorld, err := physicalComposeTRS(sceneWorld, physical.TRS{Translation: pivot, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, root, deviceRest)
	if err != nil {
		return nil, fmt.Errorf("compose final device pose for camera endpoint: %w", err)
	}
	corners := displayCorners
	deviceMatrix, err := deviceWorld.Matrix()
	if err != nil {
		return nil, err
	}
	for i := range corners {
		corners[i], err = physical.TransformPoint(deviceMatrix, corners[i])
		if err != nil {
			return nil, err
		}
	}
	var posedContentCorners *[4]physical.Vec3
	if mappedContentCorners != nil {
		posed := *mappedContentCorners
		for i := range posed {
			posed[i], err = physical.TransformPoint(deviceMatrix, posed[i])
			if err != nil {
				return nil, fmt.Errorf("transform UV-mapped recording-content corner: %w", err)
			}
		}
		posedContentCorners = &posed
	}
	endpointGeometry, err := rendering.ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint, corners, posedContentCorners)
	if err != nil {
		return nil, fmt.Errorf("resolve physical camera endpoint content geometry: %w", err)
	}
	target, residual, err := rendering.CompilePhysicalCameraEndpointBindingWithContentCorners(endpoint, corners, posedContentCorners, *camera.FocalLengthMM, *camera.SensorWidthMM, *camera.SensorHeightMM)
	if err != nil {
		return nil, fmt.Errorf("compile camera endpoint (residual %.3fpx): %w", residual, err)
	}
	delta, err := rendering.CameraEndpointRelativeTransform(camera.Frame, target)
	if err != nil {
		return nil, fmt.Errorf("express camera endpoint relative to scene camera frame: %w", err)
	}
	plan := &rendering.PhysicalCameraEndpointPlan{
		CameraID: cameraID, DisplayID: endpoint.GetDisplayId(), TargetKind: endpoint.GetTargetKind().String(),
		RasterWidth: endpoint.GetRasterWidth(), RasterHeight: endpoint.GetRasterHeight(),
		Target:        rendering.NormalizedRect{X: endpoint.GetX(), Y: endpoint.GetY(), Width: endpoint.GetWidth(), Height: endpoint.GetHeight()},
		DisplayTarget: endpointGeometry.DisplayTarget, ContentTarget: endpointGeometry.ContentTarget,
		ContentWidth: endpointGeometry.ContentWidth, ContentHeight: endpointGeometry.ContentHeight,
		DisplayCorners: endpointGeometry.DisplayCorners, ContentCorners: endpointGeometry.ContentCorners,
		ContentMappingProfile: func() string {
			if posedContentCorners != nil {
				return rendering.PhysicalScreenTriangleUVMappingProfile
			}
			return ""
		}(),
		CameraFrame: camera.Frame, FocalLengthMM: *camera.FocalLengthMM, SensorWidthMM: *camera.SensorWidthMM, SensorHeightMM: *camera.SensorHeightMM,
		MaxErrorPixels: endpoint.GetMaxErrorPixels(), ResidualPixels: residual, ArrivalMotionTick: endpoint.GetArrivalMotionTick(),
	}
	progress, err := rendering.PhysicalCameraEndpointSampleProgress(endpoint.GetArrivalMotionTick(), samples)
	if err != nil {
		return nil, fmt.Errorf("compile camera endpoint arrival schedule: %w", err)
	}
	for i := range samples {
		if len(samples[i].JointAngles) != 0 {
			return nil, errors.New("physical camera endpoint cannot be compiled with joint motion; qualify a static-lid device candidate first")
		}
		if _, exists := samples[i].CameraTransforms[cameraID]; exists {
			return nil, errors.New("physical camera endpoint conflicts with an authored track for the selected camera")
		}
		if samples[i].CameraTransforms == nil {
			samples[i].CameraTransforms = make(map[string]physical.TRS)
		}
		samples[i].CameraTransforms[cameraID] = rendering.InterpolateCameraTransform(delta, progress[i])
	}
	return plan, nil
}

func identityTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

func physicalComposeTRS(transforms ...physical.TRS) (physical.TRS, error) {
	result := physical.Matrix4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	for _, transform := range transforms {
		matrix, err := transform.Matrix()
		if err != nil {
			return physical.TRS{}, err
		}
		result, err = physical.Compose(result, matrix)
		if err != nil {
			return physical.TRS{}, err
		}
	}
	return matrixToTRS(result)
}

func matrixToTRS(m physical.Matrix4) (physical.TRS, error) {
	// This is only used for the existing scene/device TRS chain, which is
	// required to be rigid. Reject shear or nonuniform scale rather than silently
	// changing the endpoint pose.
	col := [3]physical.Vec3{{X: m[0], Y: m[1], Z: m[2]}, {X: m[4], Y: m[5], Z: m[6]}, {X: m[8], Y: m[9], Z: m[10]}}
	norm := func(v physical.Vec3) float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
	dot := func(a, b physical.Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
	scale := physical.Vec3{X: norm(col[0]), Y: norm(col[1]), Z: norm(col[2])}
	if scale.X <= 0 || scale.Y <= 0 || scale.Z <= 0 || math.Abs(scale.X-scale.Y) > 1e-6 || math.Abs(scale.Y-scale.Z) > 1e-6 {
		return physical.TRS{}, errors.New("camera endpoint pose chain has nonuniform scale")
	}
	for i := range col {
		col[i].X /= scale.X
		col[i].Y /= scale.Y
		col[i].Z /= scale.Z
	}
	if math.Abs(dot(col[0], col[1])) > 1e-6 || math.Abs(dot(col[0], col[2])) > 1e-6 || math.Abs(dot(col[1], col[2])) > 1e-6 {
		return physical.TRS{}, errors.New("camera endpoint pose chain contains shear")
	}
	q, err := quaternionFromMatrixBasis(col[0], col[1], col[2])
	if err != nil {
		return physical.TRS{}, err
	}
	return physical.CanonicalizeTRS(physical.TRS{Translation: physical.Vec3{X: m[12], Y: m[13], Z: m[14]}, Rotation: q, Scale: scale})
}

func quaternionFromMatrixBasis(x, y, z physical.Vec3) (physical.QuatXYZW, error) {
	// Matrix columns are the camera's local +X, +Y, +Z axes in world space.
	m00, m01, m02 := x.X, y.X, z.X
	m10, m11, m12 := x.Y, y.Y, z.Y
	m20, m21, m22 := x.Z, y.Z, z.Z
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
