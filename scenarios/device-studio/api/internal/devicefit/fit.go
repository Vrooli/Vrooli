// Package devicefit fits a supplied rigid device pose to caller-supplied
// 3D-to-2D landmark correspondences under a fixed known camera.
package devicefit

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"device-studio/internal/physical"
)

type Landmark struct {
	ID     string        `json:"id"`
	LocalM physical.Vec3 `json:"local_m"`
	Pixel  Pixel         `json:"pixel"`
}

type Pixel struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Camera struct {
	WorldFrame    physical.TRS `json:"world_frame"`
	RasterWidth   uint32       `json:"raster_width"`
	RasterHeight  uint32       `json:"raster_height"`
	FocalLengthMM float64      `json:"focal_length_mm"`
	SensorWidthMM float64      `json:"sensor_width_mm"`
}

type HingeAssumption struct {
	JointID      string  `json:"joint_id"`
	AngleRadians float64 `json:"angle_radians"`
	Rule         string  `json:"rule"`
}

type Request struct {
	SceneID              string          `json:"scene_id"`
	SceneRevision        uint32          `json:"scene_revision"`
	CameraFrameID        string          `json:"camera_frame_id"`
	DeviceID             string          `json:"device_id"`
	DeviceRevision       uint32          `json:"device_revision"`
	DeviceRevisionSHA256 string          `json:"device_revision_sha256"`
	GeometryAssetID      string          `json:"geometry_asset_id"`
	GeometrySHA256       string          `json:"geometry_sha256"`
	Camera               Camera          `json:"camera"`
	Landmarks            []Landmark      `json:"landmarks"`
	InitialPose          physical.TRS    `json:"initial_pose_world"`
	Hinge                HingeAssumption `json:"hinge_assumption"`
	MaxRMSPixels         float64         `json:"max_rms_pixels"`
	MaxPointPixels       float64         `json:"max_point_pixels"`
}

type Residual struct {
	LandmarkID string  `json:"landmark_id"`
	Observed   Pixel   `json:"observed"`
	Projected  Pixel   `json:"projected"`
	DX         float64 `json:"dx"`
	DY         float64 `json:"dy"`
	Magnitude  float64 `json:"magnitude"`
}

type Result struct {
	Version              string          `json:"version"`
	SceneID              string          `json:"scene_id"`
	SceneRevision        uint32          `json:"scene_revision"`
	CameraFrameID        string          `json:"camera_frame_id"`
	Camera               Camera          `json:"camera"`
	DeviceID             string          `json:"device_id"`
	DeviceRevision       uint32          `json:"device_revision"`
	DeviceRevisionSHA256 string          `json:"device_revision_sha256"`
	GeometryAssetID      string          `json:"geometry_asset_id"`
	GeometrySHA256       string          `json:"geometry_sha256"`
	FittedPose           physical.TRS    `json:"fitted_pose_world"`
	InitialPose          physical.TRS    `json:"initial_pose_world"`
	Hinge                HingeAssumption `json:"hinge_assumption"`
	Residuals            []Residual      `json:"residuals"`
	RMSPixels            float64         `json:"rms_pixels"`
	MaxPixels            float64         `json:"max_pixels"`
	OverlaySVG           string          `json:"overlay_svg"`
	Iterations           int             `json:"iterations"`
}

const profile = "device-studio-known-camera-landmark-fit-v1"

func Fit(req Request) (Result, error) {
	if err := validate(req); err != nil {
		return Result{}, err
	}
	pose, err := physical.CanonicalizeTRS(req.InitialPose)
	if err != nil {
		return Result{}, fmt.Errorf("initial object pose: %w", err)
	}
	if math.Abs(pose.Scale.X-1) > 1e-8 || math.Abs(pose.Scale.Y-1) > 1e-8 || math.Abs(pose.Scale.Z-1) > 1e-8 {
		return Result{}, errors.New("initial device transform must be rigid with unit scale")
	}
	pose.Scale = physical.Vec3{X: 1, Y: 1, Z: 1}
	current, err := residualVector(req, pose)
	if err != nil {
		return Result{}, err
	}
	if len(current) < 12 {
		return Result{}, errors.New("at least six supplied landmarks are required")
	}
	iterations := 0
	for ; iterations < 80; iterations++ {
		j := numericJacobian(req, pose, current)
		var h [6][6]float64
		var g [6]float64
		for row := 0; row < len(current); row++ {
			for a := 0; a < 6; a++ {
				g[a] += j[row][a] * current[row]
				for b := 0; b < 6; b++ {
					h[a][b] += j[row][a] * j[row][b]
				}
			}
		}
		// A full-rank normal matrix is required. Damping is for conditioning,
		// not permission to fit a geometrically ambiguous point set.
		if !fullRank(h) {
			return Result{}, errors.New("landmark geometry is degenerate for a unique fixed-camera rigid pose")
		}
		for d := 0; d < 6; d++ {
			h[d][d] += 1e-8 * (1 + h[d][d])
			g[d] = -g[d]
		}
		step, ok := solve(h, g)
		if !ok {
			return Result{}, errors.New("pose fit normal equations are singular")
		}
		norm := 0.0
		for _, v := range step {
			norm += v * v
		}
		if norm < 1e-16 {
			break
		}
		accepted := false
		for alpha := 1.0; alpha >= 1.0/128; alpha *= .5 {
			candidate := apply(pose, step, alpha)
			values, e := residualVector(req, candidate)
			if e != nil {
				continue
			}
			if sumSquares(values) < sumSquares(current) {
				pose, current, accepted = candidate, values, true
				break
			}
		}
		if !accepted {
			break
		}
	}
	result, err := makeResult(req, pose, iterations)
	if err != nil {
		return Result{}, err
	}
	if req.MaxRMSPixels > 0 && result.RMSPixels > req.MaxRMSPixels {
		return result, fmt.Errorf("pose fit rejected: RMS reprojection error %.3fpx exceeds %.3fpx", result.RMSPixels, req.MaxRMSPixels)
	}
	if req.MaxPointPixels > 0 && result.MaxPixels > req.MaxPointPixels {
		return result, fmt.Errorf("pose fit rejected: maximum landmark error %.3fpx exceeds %.3fpx", result.MaxPixels, req.MaxPointPixels)
	}
	return result, nil
}

func validate(r Request) error {
	if strings.TrimSpace(r.SceneID) == "" || r.SceneRevision == 0 || strings.TrimSpace(r.CameraFrameID) == "" {
		return errors.New("pose-fit manifest requires an exact scene revision and camera-frame ID")
	}
	if strings.TrimSpace(r.GeometryAssetID) == "" || len(r.GeometrySHA256) != 64 {
		return errors.New("pose-fit manifest requires exact device geometry asset ID and SHA-256")
	}
	if strings.TrimSpace(r.DeviceID) == "" || r.DeviceRevision == 0 || len(r.DeviceRevisionSHA256) != 64 {
		return errors.New("pose-fit manifest requires an exact immutable device revision and SHA-256")
	}
	deviceDigest, err := hex.DecodeString(r.DeviceRevisionSHA256)
	if err != nil || len(deviceDigest) != 32 || strings.ToLower(r.DeviceRevisionSHA256) != r.DeviceRevisionSHA256 {
		return errors.New("device revision SHA-256 must be 64 lowercase hexadecimal characters")
	}
	digest, err := hex.DecodeString(r.GeometrySHA256)
	if err != nil || len(digest) != 32 || strings.ToLower(r.GeometrySHA256) != r.GeometrySHA256 {
		return errors.New("geometry SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if len(r.Landmarks) < 6 || len(r.Landmarks) > 256 {
		return errors.New("pose fit requires 6..256 explicit landmarks")
	}
	if r.Camera.RasterWidth == 0 || r.Camera.RasterHeight == 0 || !positive(r.Camera.FocalLengthMM) || !positive(r.Camera.SensorWidthMM) {
		return errors.New("known camera requires raster dimensions and positive finite focal length and sensor width")
	}
	if !finite(r.Hinge.AngleRadians) {
		return errors.New("hinge angle assumption must be finite")
	}
	if strings.TrimSpace(r.Hinge.JointID) == "" || strings.TrimSpace(r.Hinge.Rule) == "" {
		return errors.New("hinge joint and explicit hinge rule are required; hinge is held fixed during rigid-pose fitting")
	}
	if !positive(r.MaxRMSPixels) || !positive(r.MaxPointPixels) {
		return errors.New("positive finite RMS and per-landmark pixel limits are required; poor fits must never be accepted with implicit limits")
	}
	seen := map[string]bool{}
	for _, p := range r.Landmarks {
		if strings.TrimSpace(p.ID) == "" || seen[p.ID] {
			return errors.New("every landmark requires a unique nonempty id")
		}
		seen[p.ID] = true
		if err := p.LocalM.Validate(); err != nil {
			return fmt.Errorf("landmark %s 3D coordinate: %w", p.ID, err)
		}
		if !finite(p.Pixel.X) || !finite(p.Pixel.Y) || p.Pixel.X < 0 || p.Pixel.X >= float64(r.Camera.RasterWidth) || p.Pixel.Y < 0 || p.Pixel.Y >= float64(r.Camera.RasterHeight) {
			return fmt.Errorf("landmark %s pixel must be a finite pixel center inside the raster", p.ID)
		}
	}
	camera, e := physical.CanonicalizeTRS(r.Camera.WorldFrame)
	if e != nil {
		return fmt.Errorf("camera world frame: %w", e)
	}
	if math.Abs(camera.Scale.X-1) > 1e-8 || math.Abs(camera.Scale.Y-1) > 1e-8 || math.Abs(camera.Scale.Z-1) > 1e-8 {
		return errors.New("camera world frame must have unit scale")
	}
	return nil
}

func makeResult(req Request, pose physical.TRS, iterations int) (Result, error) {
	vals, err := residualVector(req, pose)
	if err != nil {
		return Result{}, err
	}
	out := Result{Version: profile, FittedPose: pose, InitialPose: req.InitialPose, Hinge: req.Hinge, Iterations: iterations, Residuals: make([]Residual, 0, len(req.Landmarks))}
	out.SceneID, out.SceneRevision, out.CameraFrameID = req.SceneID, req.SceneRevision, req.CameraFrameID
	out.DeviceID, out.DeviceRevision, out.DeviceRevisionSHA256 = req.DeviceID, req.DeviceRevision, req.DeviceRevisionSHA256
	out.Camera = req.Camera
	out.GeometryAssetID, out.GeometrySHA256 = req.GeometryAssetID, req.GeometrySHA256
	ss := 0.0
	for i, l := range req.Landmarks {
		dx, dy := vals[2*i], vals[2*i+1]
		mag := math.Hypot(dx, dy)
		ss += mag * mag
		if mag > out.MaxPixels {
			out.MaxPixels = mag
		}
		p, e := project(req, pose, l.LocalM)
		if e != nil {
			return Result{}, e
		}
		out.Residuals = append(out.Residuals, Residual{LandmarkID: l.ID, Observed: l.Pixel, Projected: p, DX: dx, DY: dy, Magnitude: mag})
	}
	out.RMSPixels = math.Sqrt(ss / float64(len(req.Landmarks)))
	out.OverlaySVG = svg(req, out.Residuals)
	return out, nil
}

func project(r Request, pose physical.TRS, p physical.Vec3) (Pixel, error) {
	pm, e := pose.Matrix()
	if e != nil {
		return Pixel{}, e
	}
	world, e := physical.TransformPoint(pm, p)
	if e != nil {
		return Pixel{}, e
	}
	cm, e := r.Camera.WorldFrame.Matrix()
	if e != nil {
		return Pixel{}, e
	}
	inv := inverseRigid(cm)
	local, e := physical.TransformPoint(inv, world)
	if e != nil {
		return Pixel{}, e
	}
	// Contract camera conversion: canonical camera-local xyz -> OpenCV xyz.
	x, y, z := local.X, -local.Y, -local.Z
	if z <= 1e-8 {
		return Pixel{}, errors.New("landmark projects behind or on the camera plane")
	}
	fx := float64(r.Camera.RasterWidth) * r.Camera.FocalLengthMM / r.Camera.SensorWidthMM
	// Horizontal sensor fit makes pixel focal lengths equal in both axes.
	u := fx*x/z + float64(r.Camera.RasterWidth)/2
	v := fx*y/z + float64(r.Camera.RasterHeight)/2
	return Pixel{u, v}, nil
}

func residualVector(r Request, p physical.TRS) ([]float64, error) {
	out := make([]float64, 0, 2*len(r.Landmarks))
	for _, l := range r.Landmarks {
		q, e := project(r, p, l.LocalM)
		if e != nil {
			return nil, e
		}
		out = append(out, q.X-l.Pixel.X, q.Y-l.Pixel.Y)
	}
	return out, nil
}
func numericJacobian(r Request, p physical.TRS, base []float64) [][]float64 {
	j := make([][]float64, len(base))
	for i := range j {
		j[i] = make([]float64, 6)
	}
	for k := 0; k < 6; k++ {
		d := [6]float64{}
		eps := 1e-5
		if k < 3 {
			eps = 1e-6
		}
		d[k] = eps
		q := apply(p, d, 1)
		v, e := residualVector(r, q)
		if e != nil {
			continue
		}
		for row := range base {
			j[row][k] = (v[row] - base[row]) / eps
		}
	}
	return j
}
func apply(p physical.TRS, d [6]float64, a float64) physical.TRS {
	p.Translation.X += a * d[0]
	p.Translation.Y += a * d[1]
	p.Translation.Z += a * d[2]
	angle := math.Sqrt(d[3]*d[3]+d[4]*d[4]+d[5]*d[5]) * a
	if angle > 0 {
		scale := math.Sin(angle/2) / math.Sqrt(d[3]*d[3]+d[4]*d[4]+d[5]*d[5])
		q := physical.QuatXYZW{X: d[3] * scale, Y: d[4] * scale, Z: d[5] * scale, W: math.Cos(angle / 2)}
		p.Rotation = mul(q, p.Rotation)
	}
	p, _ = physical.CanonicalizeTRS(p)
	p.Scale = physical.Vec3{X: 1, Y: 1, Z: 1}
	return p
}
func mul(a, b physical.QuatXYZW) physical.QuatXYZW {
	return physical.QuatXYZW{X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y, Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X, Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W, W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z}
}
func inverseRigid(m physical.Matrix4) physical.Matrix4 {
	return physical.Matrix4{m[0], m[4], m[8], 0, m[1], m[5], m[9], 0, m[2], m[6], m[10], 0, -(m[0]*m[12] + m[1]*m[13] + m[2]*m[14]), -(m[4]*m[12] + m[5]*m[13] + m[6]*m[14]), -(m[8]*m[12] + m[9]*m[13] + m[10]*m[14]), 1}
}
func sumSquares(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x * x
	}
	return s
}
func fullRank(a [6][6]float64) bool {
	m := a
	for i := 0; i < 6; i++ {
		pivot := i
		for r := i + 1; r < 6; r++ {
			if math.Abs(m[r][i]) > math.Abs(m[pivot][i]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot][i]) < 1e-10 {
			return false
		}
		m[i], m[pivot] = m[pivot], m[i]
		for r := i + 1; r < 6; r++ {
			f := m[r][i] / m[i][i]
			for c := i; c < 6; c++ {
				m[r][c] -= f * m[i][c]
			}
		}
	}
	return true
}
func solve(a [6][6]float64, b [6]float64) ([6]float64, bool) {
	for i := 0; i < 6; i++ {
		p := i
		for r := i + 1; r < 6; r++ {
			if math.Abs(a[r][i]) > math.Abs(a[p][i]) {
				p = r
			}
		}
		if math.Abs(a[p][i]) < 1e-14 {
			return [6]float64{}, false
		}
		a[i], a[p] = a[p], a[i]
		b[i], b[p] = b[p], b[i]
		d := a[i][i]
		for c := i; c < 6; c++ {
			a[i][c] /= d
		}
		b[i] /= d
		for r := 0; r < 6; r++ {
			if r == i {
				continue
			}
			f := a[r][i]
			for c := i; c < 6; c++ {
				a[r][c] -= f * a[i][c]
			}
			b[r] -= f * b[i]
		}
	}
	return b, true
}
func svg(r Request, rs []Residual) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" preserveAspectRatio=\"none\">", r.Camera.RasterWidth, r.Camera.RasterHeight, r.Camera.RasterWidth, r.Camera.RasterHeight)
	for _, p := range rs {
		fmt.Fprintf(&b, "<line x1=\"%.3f\" y1=\"%.3f\" x2=\"%.3f\" y2=\"%.3f\" stroke=\"#ffc857\"/><circle cx=\"%.3f\" cy=\"%.3f\" r=\"4\" fill=\"#ef476f\"/><circle cx=\"%.3f\" cy=\"%.3f\" r=\"3\" fill=\"#06d6a0\"/>", p.Observed.X, p.Observed.Y, p.Projected.X, p.Projected.Y, p.Observed.X, p.Observed.Y, p.Projected.X, p.Projected.Y)
	}
	b.WriteString("</svg>")
	return b.String()
}
func positive(v float64) bool { return finite(v) && v > 0 }
func finite(v float64) bool   { return !math.IsNaN(v) && !math.IsInf(v, 0) }
