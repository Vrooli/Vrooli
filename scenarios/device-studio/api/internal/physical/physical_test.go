package physical_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/physical"
)

func TestTRSColumnVectorOrderAndQuaternionCanonicalization(t *testing.T) {
	positive, err := physical.CanonicalizeQuaternion(physical.QuatXYZW{Z: 2, W: 2})
	if err != nil {
		t.Fatal(err)
	}
	negative, err := physical.CanonicalizeQuaternion(physical.QuatXYZW{Z: -2, W: -2})
	if err != nil || positive != negative {
		t.Fatalf("q/-q canonicalization = %+v %+v, %v", positive, negative, err)
	}
	trs := physical.TRS{Translation: physical.Vec3{X: 1, Y: 2, Z: 3}, Rotation: positive, Scale: physical.Vec3{X: 2, Y: 1, Z: 1}}
	matrix, err := trs.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	point, err := physical.TransformPoint(matrix, physical.Vec3{X: 1})
	if err != nil || math.Abs(point.X-1) > 1e-12 || math.Abs(point.Y-4) > 1e-12 || math.Abs(point.Z-3) > 1e-12 {
		t.Fatalf("T·R·S column-vector result = %+v, %v", point, err)
	}
	parent, _ := (physical.TRS{Translation: physical.Vec3{X: 10}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	composed, err := physical.Compose(parent, matrix)
	if err != nil {
		t.Fatal(err)
	}
	point, err = physical.TransformPoint(composed, physical.Vec3{X: 1})
	if err != nil || math.Abs(point.X-11) > 1e-12 || math.Abs(point.Y-4) > 1e-12 {
		t.Fatalf("parent·local result = %+v, %v", point, err)
	}
}

func TestCanonicalBlenderBasisAndDeviceVectorJSONCompatibility(t *testing.T) {
	for _, tc := range []struct{ in, want physical.Vec3 }{
		{physical.Vec3{X: 1}, physical.Vec3{X: 1}},
		{physical.Vec3{Y: 1}, physical.Vec3{Z: 1}},
		{physical.Vec3{Z: 1}, physical.Vec3{Y: -1}},
	} {
		got, err := physical.CanonicalToBlenderVector(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("basis conversion %+v = %+v, %v", tc.in, got, err)
		}
	}
	canonical, _ := (physical.TRS{Translation: physical.Vec3{X: 1, Y: 2, Z: 3}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	blender, err := physical.CanonicalToBlenderMatrix(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if blender[12] != 1 || blender[13] != -3 || blender[14] != 2 {
		t.Fatalf("converted translation = %v", blender[12:15])
	}
	rotation, err := (physical.TRS{Rotation: physical.QuatXYZW{Z: 1, W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		t.Fatal(err)
	}
	blenderRotation, err := physical.CanonicalToBlenderMatrix(rotation)
	if err != nil {
		t.Fatal(err)
	}
	canonicalPoint, err := physical.TransformPoint(rotation, physical.Vec3{X: 1})
	if err != nil {
		t.Fatal(err)
	}
	wantBlenderPoint, err := physical.CanonicalToBlenderVector(canonicalPoint)
	if err != nil {
		t.Fatal(err)
	}
	gotBlenderPoint, err := physical.TransformPoint(blenderRotation, physical.Vec3{X: 1})
	if err != nil || gotBlenderPoint != wantBlenderPoint {
		t.Fatalf("rotation basis conjugation = %+v, want %+v, %v", gotBlenderPoint, wantBlenderPoint, err)
	}
	encoded, err := json.Marshal(devices.Vector3{X: 1, Y: 2, Z: 3})
	if err != nil || string(encoded) != `{"x":1,"y":2,"z":3}` {
		t.Fatalf("legacy vector JSON changed: %s, %v", encoded, err)
	}
}

func TestPhysicalValuesRejectInvalidCoordinates(t *testing.T) {
	if _, err := physical.CanonicalizeQuaternion(physical.QuatXYZW{}); err == nil {
		t.Fatal("zero quaternion accepted")
	}
	if _, err := (physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: -1, Z: 1}}).Matrix(); err == nil {
		t.Fatal("negative scale accepted")
	}
	if err := (physical.Vec3{X: math.NaN()}).Validate(); err == nil {
		t.Fatal("NaN vector accepted")
	}
	if err := (physical.DisplayUV{U: 0.5, V: 1.01}).Validate(); err == nil {
		t.Fatal("out-of-range display UV accepted")
	}
	overflow := physical.Matrix4{}
	overflow[0] = math.MaxFloat64
	double := physical.Matrix4{}
	double[0] = 2
	if _, err := physical.Compose(overflow, double); err == nil {
		t.Fatal("overflowed composed transform accepted")
	}
}

func TestTransformPointRejectsOverflowAfterHomogeneousDivide(t *testing.T) {
	matrix := physical.Matrix4{
		0: 1, 5: 1, 10: 1, 15: math.SmallestNonzeroFloat64,
	}
	if _, err := physical.TransformPoint(matrix, physical.Vec3{X: 1}); err == nil {
		t.Fatal("finite numerator divided by finite tiny w returned an infinite point")
	}
}

func TestCanonicalizeTRSNormalizesNegativeZeroTranslation(t *testing.T) {
	trs := physical.TRS{
		Translation: physical.Vec3{X: math.Copysign(0, -1), Y: 2, Z: math.Copysign(0, -1)},
		Rotation:    physical.QuatXYZW{W: 1},
		Scale:       physical.Vec3{X: 1, Y: 1, Z: 1},
	}
	got, err := physical.CanonicalizeTRS(trs)
	if err != nil {
		t.Fatal(err)
	}
	if math.Signbit(got.Translation.X) || math.Signbit(got.Translation.Z) {
		t.Fatalf("canonical translation retained signed zero: %+v", got.Translation)
	}
	encoded, err := json.Marshal(got)
	if err != nil || bytes.Contains(encoded, []byte("-0")) {
		t.Fatalf("canonical TRS JSON = %s, %v", encoded, err)
	}
}

func TestCanonicalizeQuaternionIsIdempotentForNontrivialRotation(t *testing.T) {
	first, err := physical.CanonicalizeQuaternion(physical.QuatXYZW{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4})
	if err != nil {
		t.Fatal(err)
	}
	second, err := physical.CanonicalizeQuaternion(first)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("canonical quaternion changed on second normalization: first=%+v second=%+v", first, second)
	}
}

type fakeLibrary struct {
	revision  library.BundleRevision
	opened    library.ResolvedArtifact
	openCalls int
}

func (f *fakeLibrary) GetRevision(context.Context, string, uint32) (library.BundleRevision, error) {
	return f.revision, nil
}
func (f *fakeLibrary) OpenArtifact(_ context.Context, bundle string, revision uint32, artifact string, use library.Use) (library.ResolvedArtifact, error) {
	f.openCalls++
	if !allowed(f.revision.Review.AllowedUses, use) {
		return library.ResolvedArtifact{}, library.ErrUseDenied
	}
	return f.opened, nil
}
func allowed(values []library.Use, use library.Use) bool {
	for _, v := range values {
		if v == use {
			return true
		}
	}
	return false
}

func TestArtifactResolverPinsReviewArtifactAndUse(t *testing.T) {
	body := []byte("reviewed glb bytes")
	artifact := library.Artifact{ID: "phone", Role: library.RoleGeometryGLB, SizeBytes: int64(len(body)), SHA256: hash(body), Rights: []library.Use{library.UseRender}}
	ref := physical.ArtifactRef{BundleID: "bundle", BundleRevision: 2, ManifestSHA256: hash([]byte("manifest")), ReviewSHA256: hash([]byte("review")), ArtifactID: artifact.ID, Role: artifact.Role, SizeBytes: artifact.SizeBytes, SHA256: artifact.SHA256}
	revision := library.BundleRevision{Manifest: library.BundleManifest{BundleID: ref.BundleID, Revision: ref.BundleRevision, Artifacts: []library.Artifact{artifact}}, ManifestSHA256: ref.ManifestSHA256, Review: library.ReviewBinding{Decision: "approved", AllowedUses: []library.Use{library.UseRender}}, ReviewSHA256: ref.ReviewSHA256}
	fake := &fakeLibrary{revision: revision, opened: library.ResolvedArtifact{BundleID: ref.BundleID, Revision: ref.BundleRevision, Artifact: artifact, Reader: io.NopCloser(bytes.NewReader(body))}}
	resolver, err := physical.NewLibraryArtifactResolver(fake)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve(context.Background(), ref, library.UseRender)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resolved.Reader)
	if err != nil || !bytes.Equal(data, body) {
		t.Fatalf("resolved reviewed stream = %q, %v", data, err)
	}
	_ = resolved.Reader.Close()

	for name, mutate := range map[string]func(*physical.ArtifactRef){
		"bundle":          func(v *physical.ArtifactRef) { v.BundleID = "other-bundle" },
		"manifest":        func(v *physical.ArtifactRef) { v.ManifestSHA256 = hash([]byte("other manifest")) },
		"review":          func(v *physical.ArtifactRef) { v.ReviewSHA256 = hash([]byte("different review")) },
		"artifact role":   func(v *physical.ArtifactRef) { v.Role = library.RoleTextureImage },
		"artifact digest": func(v *physical.ArtifactRef) { v.SHA256 = hash([]byte("other bytes")) },
	} {
		t.Run(name, func(t *testing.T) {
			wrong := ref
			mutate(&wrong)
			if _, err := resolver.Resolve(context.Background(), wrong, library.UseRender); err == nil {
				t.Fatalf("mismatched %s reference resolved", name)
			}
		})
	}
	if _, err := resolver.Resolve(context.Background(), ref, library.UseRedistribute); !errors.Is(err, library.ErrUseDenied) {
		t.Fatalf("unreviewed use error = %v", err)
	}
	if fake.openCalls != 1 {
		t.Fatalf("library opened bytes %d times for mismatching or denied refs", fake.openCalls)
	}
}

func hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
