package devicefit

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"device-studio/internal/physical"
)

func testRequest() Request {
	cam := Camera{WorldFrame: physical.TRS{Translation: physical.Vec3{Z: 4}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, RasterWidth: 1280, RasterHeight: 720, FocalLengthMM: 50, SensorWidthMM: 36}
	truePose := physical.TRS{Translation: physical.Vec3{X: .12, Y: -.08, Z: .15}, Rotation: physical.QuatXYZW{Y: math.Sin(.08), W: math.Cos(.08)}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	pts := []physical.Vec3{{X: -.4, Y: -.25, Z: -.1}, {X: .4, Y: -.25, Z: -.1}, {X: .4, Y: .25, Z: -.1}, {X: -.4, Y: .25, Z: -.1}, {X: -.4, Y: -.25, Z: .1}, {X: .4, Y: -.25, Z: .1}, {X: .4, Y: .25, Z: .1}, {X: -.4, Y: .25, Z: .1}}
	r := Request{SceneID: "fit-fixture", SceneRevision: 1, CameraFrameID: "known-camera", DeviceID: "fit-device", DeviceRevision: 1, DeviceRevisionSHA256: strings.Repeat("b", 64), GeometryAssetID: "fixture-mesh", GeometrySHA256: strings.Repeat("a", 64), Camera: cam, InitialPose: physical.TRS{Translation: physical.Vec3{X: .08, Y: -.04, Z: .1}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, Hinge: HingeAssumption{JointID: "lid-hinge", AngleRadians: .55, Rule: "caller-supplied fixed hinge angle applied before rigid landmark coordinates"}, MaxRMSPixels: .1, MaxPointPixels: .2}
	for i, p := range pts {
		q, e := project(r, truePose, p)
		if e != nil {
			panic(e)
		}
		r.Landmarks = append(r.Landmarks, Landmark{ID: string(rune('a' + i)), LocalM: p, Pixel: q})
	}
	return r
}

func TestFitSyntheticKnownCameraFromImmutableLaptopScreenVertices(t *testing.T) {
	asset := filepath.Join("..", "physicalruntime", "assets", "laptop", "laptop-neutral-rig.glb")
	data, err := os.ReadFile(asset)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if digest != "3005e1e8b12e2017565292712aaad20dfb35cda2c42519a937d2c74afb6f5925" {
		t.Fatalf("laptop asset identity changed: %s", digest)
	}
	// These six supplied points are exact vertices sampled from GLB mesh 3
	// (LaptopScreenSurface, primitive 0) and transformed through its immutable
	// node/pivot chain into canonical device-local meters. Pixels below are
	// analytically projected under the declared known camera, not observed from
	// a photograph and not produced by a keypoint detector.
	local := []physical.Vec3{
		{X: -.17200352833878618, Y: -.2072399899074675, Z: -.22028916783146713},
		{X: .17200287827848193, Y: -.2072399899074675, Z: -.22028916783146713},
		{X: .17233949479956046, Y: -.13185775682005918, Z: -.012951190883740082},
		{X: -.1723401448598647, Y: -.13185775682005918, Z: -.012951190883740082},
		{X: .000010190651240193915, Y: -.2051992209897655, Z: -.21467178590689073},
		{X: .02041787793221239, Y: -.2051992209897655, Z: -.21467178590689073},
	}
	r := Request{SceneID: "laptop-analytic-camera", SceneRevision: 1, CameraFrameID: "laptop-candidate-camera", DeviceID: "jack-baeten-macbook-pro-m3-16-derive16-candidate", DeviceRevision: 1, DeviceRevisionSHA256: strings.Repeat("c", 64), GeometryAssetID: "jack-baeten-laptop-neutral-rig-glb", GeometrySHA256: digest, Camera: Camera{WorldFrame: physical.TRS{Translation: physical.Vec3{Y: -.17, Z: .6}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, RasterWidth: 1280, RasterHeight: 720, FocalLengthMM: 50, SensorWidthMM: 36}, InitialPose: physical.TRS{Translation: physical.Vec3{X: .02, Y: .015, Z: .02}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, Hinge: HingeAssumption{JointID: "laptop_lid_hinge", AngleRadians: 0, Rule: "screen mesh vertices are supplied in this selected open/rest pose; hinge held fixed"}, MaxRMSPixels: .01, MaxPointPixels: .02}
	for i, p := range local {
		q, e := project(r, physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, p)
		if e != nil {
			t.Fatal(e)
		}
		r.Landmarks = append(r.Landmarks, Landmark{ID: []string{"screen-left-lower", "screen-right-lower", "screen-right-upper", "screen-left-upper", "screen-lower-center", "screen-lower-quarter-right"}[i], LocalM: p, Pixel: q})
	}
	result, err := Fit(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("asset=%s bytes=%d sha256=%s landmarks=%d rms_px=%.9f max_px=%.9f iterations=%d", r.GeometryAssetID, len(data), digest, len(r.Landmarks), result.RMSPixels, result.MaxPixels, result.Iterations)
	if result.RMSPixels > 1e-4 || result.GeometrySHA256 != digest || result.GeometryAssetID != r.GeometryAssetID {
		t.Fatalf("unexpected asset-bound fit result: %+v", result)
	}
}

func TestFitKnownCameraPoseAndPreserveSeparateHinge(t *testing.T) {
	r := testRequest()
	out, err := Fit(r)
	if err != nil {
		t.Fatal(err)
	}
	if out.RMSPixels > 1e-4 || out.MaxPixels > 1e-4 {
		t.Fatalf("known pose residuals rms=%g max=%g", out.RMSPixels, out.MaxPixels)
	}
	if out.Hinge != r.Hinge {
		t.Fatalf("hinge assumption changed: got %+v want %+v", out.Hinge, r.Hinge)
	}
	if len(out.Residuals) != len(r.Landmarks) || !strings.Contains(out.OverlaySVG, "#ef476f") || !strings.Contains(out.OverlaySVG, "#06d6a0") {
		t.Fatal("fit omitted point residuals or inspectable observed/projected overlay")
	}
	if math.Abs(out.FittedPose.Translation.X-.12) > 1e-4 || math.Abs(out.FittedPose.Translation.Y+.08) > 1e-4 || math.Abs(out.FittedPose.Translation.Z-.15) > 1e-4 {
		t.Fatalf("fitted translation %+v", out.FittedPose.Translation)
	}
}

func TestFitRejectsDegenerateAndPoorMatches(t *testing.T) {
	r := testRequest()
	for i := range r.Landmarks {
		r.Landmarks[i].LocalM = physical.Vec3{X: float64(i) * .1}
	}
	if _, err := Fit(r); err == nil || !strings.Contains(err.Error(), "degenerate") {
		t.Fatalf("collinear landmarks accepted: %v", err)
	}
	r = testRequest()
	r.Landmarks[0].Pixel.X += 40
	if _, err := Fit(r); err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("poor correspondence fit accepted: %v", err)
	}
}
