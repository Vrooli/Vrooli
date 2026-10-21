package geometry

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func validRecipe() Recipe {
	return Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: Dimensions{80, 150, 12}, Parts: []Part{
		{ID: "housing", Kind: "rounded_box", Role: "housing", Center: [3]float64{0, 0, 0}, Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"},
		{ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"},
	}, Screen: Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
}

func TestDecodeRejectsUnknownFieldsAndInvalidLocks(t *testing.T) {
	r := validRecipe()
	b, err := jsonMarshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Decode(append(b[:len(b)-1], []byte(`,"prompt":"run code"}`)...)); err == nil {
		t.Fatal("unknown model-authored fields must be rejected")
	}
	r.Parts[0].Center[0] = 50
	if err := Validate(r); err == nil {
		t.Fatal("part outside immutable envelope was accepted")
	}
	r = validRecipe()
	r.Screen.PartID = "housing"
	if err := Validate(r); err == nil {
		t.Fatal("screen not mapped to the display role was accepted")
	}
	r = validRecipe()
	r.Parts[0].Rotation[2] = -90
	if err := Validate(r); err == nil {
		t.Fatal("non-canonical rotation was accepted")
	}
}

func TestBevelAtHalfSmallestDimensionStaysRejectedAndBecomesTyped(t *testing.T) {
	r := validRecipe()
	r.Parts[0].Size[2] = 2
	r.Parts[0].Bevel = 1
	err := Validate(r)
	if err == nil {
		t.Fatal("bevel equal to half the smallest dimension must remain invalid")
	}
	issue, ok := IssueFromError(err)
	want := SemanticIssue{Code: "bevel_exceeds_part_dimensions", PartID: "housing", BevelMM: 1, MinimumDimensionMM: 2, MaxBevelExclusiveMM: 1, Message: "bevel must be less than half the smallest part dimension"}
	if !ok || issue != want {
		t.Fatalf("expected bounded exact bevel issue %+v, got %+v (typed=%v, err=%v)", want, issue, ok, err)
	}
	r.Parts[0].Bevel = 0.999
	if err := Validate(r); err != nil {
		t.Fatalf("a positive bevel strictly below half the smallest dimension should remain valid: %v", err)
	}
}

func TestDecodeRequiresExactTuplesAndRejectsDuplicateKeys(t *testing.T) {
	base := `{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[80,150,12],"parts":[{"id":"housing","kind":"box","role":"housing","center_mm":[0,0,0],"size_mm":[78,148,10],"rotation_deg":[0,0,0],"bevel_mm":0,"material":"matte_polymer"},{"id":"display","kind":"box","role":"display","center_mm":[0,0,5],"size_mm":[64,124,1],"rotation_deg":[0,0,0],"bevel_mm":0,"material":"glass"}],"screen":{"part_id":"display","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.9,0.9],"rotation_deg":0}}`
	for name, value := range map[string]string{"short tuple": strings.Replace(base, `"envelope_mm":[80,150,12]`, `"envelope_mm":[80,150]`, 1), "long tuple": strings.Replace(base, `"envelope_mm":[80,150,12]`, `"envelope_mm":[80,150,12,1]`, 1), "duplicate key": strings.Replace(base, `"profile":"detailed_device_v1"`, `"profile":"detailed_device_v1","profile":"detailed_device_v1"`, 1), "missing required": strings.Replace(base, `"bevel_mm":0,`, "", 1)} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Decode([]byte(value)); err == nil {
				t.Fatal("non-canonical model JSON must be rejected")
			}
		})
	}
}

func TestCompileIsDeterministicAndProducesBoundedGLBWithScreen(t *testing.T) {
	r := validRecipe()
	a, err := Compile(r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compile(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("same recipe produced different GLB bytes")
	}
	if len(a) > MaxGLBBytes || binary.LittleEndian.Uint32(a[:4]) != 0x46546c67 || int(binary.LittleEndian.Uint32(a[8:12])) != len(a) {
		t.Fatal("invalid or oversized GLB container")
	}
	jsonLen := int(binary.LittleEndian.Uint32(a[12:16]))
	var doc glbJSON
	if err := json.Unmarshal(a[20:20+jsonLen], &doc); err != nil {
		t.Fatal(err)
	}
	var display glbMesh
	for _, mesh := range doc.Meshes {
		if mesh.Name == "display" {
			display = mesh
		}
	}
	if len(display.Primitives) != 2 {
		t.Fatalf("display mesh must contain housing-facing geometry and a separate screen primitive, got %d", len(display.Primitives))
	}
	if display.Primitives[1].Extras["semantic"] != "independent_display_surface" || display.Primitives[1].Attributes["TEXCOORD_0"] == 0 {
		t.Fatal("display screen surface must retain semantic mapping and UVs")
	}
	if doc.Nodes[1].Mesh != 1 {
		t.Fatal("screen must be included in the display node mapping")
	}
}

func TestCompileKeepsSubMillimeterDetailsAndRejectsFloat32CollapsedTriangles(t *testing.T) {
	r := validRecipe()
	r.Parts = append(r.Parts, Part{ID: "speaker-slot", Kind: "box", Role: "speaker", Center: [3]float64{30, 0, 0}, Size: [3]float64{0.6, 0.4, 0.2}, Material: "rubber"})
	if _, err := Compile(r); err != nil {
		t.Fatalf("small but representable device features must compile: %v", err)
	}
	r.Parts = append(r.Parts, Part{ID: "collapsed-detail", Kind: "box", Role: "detail", Center: [3]float64{25, 0, 0}, Size: [3]float64{0.000001, 0.5, 0.5}, Material: "neutral_metal"})
	if err := Validate(r); err != nil {
		t.Fatalf("positive input geometry should reach post-float32 area validation: %v", err)
	}
	if _, err := Compile(r); err == nil || !strings.Contains(err.Error(), "degenerate triangle") {
		t.Fatalf("float32-collapsed triangle must be rejected, got %v", err)
	}
}

func TestCylinderNormalsKeepCapsCrispFromSides(t *testing.T) {
	vertices, indices := cylinder([3]float64{}, [3]float64{2, 2, 1})
	orientTriangles(vertices, indices, [3]float64{})
	normals, err := vertexNormals(vertices, indices)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2*16; i++ {
		if math.Abs(float64(normals[i][2])) > 1e-6 {
			t.Fatalf("side normal %d blended with a cap: %v", i, normals[i])
		}
	}
	for i := 2 * 16; i < 3*16; i++ {
		if normals[i][2] > -0.999 {
			t.Fatalf("bottom cap normal %d is not crisp: %v", i, normals[i])
		}
	}
	for i := 3 * 16; i < 4*16; i++ {
		if normals[i][2] < 0.999 {
			t.Fatalf("top cap normal %d is not crisp: %v", i, normals[i])
		}
	}
}

func TestScreenStandOffCannotEscapeLockedEnvelope(t *testing.T) {
	r := validRecipe()
	r.Parts[1].Center[2] = 5.6
	if err := Validate(r); err == nil || !strings.Contains(err.Error(), "screen surface exceeds locked envelope") {
		t.Fatalf("screen stand-off beyond tolerance must fail, got %v", err)
	}
}

func TestScreenRectangleMustFitActualFlatDisplaySurface(t *testing.T) {
	t.Run("rounded box stops before its curved corner", func(t *testing.T) {
		r := validRecipe()
		display := &r.Parts[1]
		display.Kind = "rounded_box"
		display.Bevel = 3.9
		display.Center[2] = 0
		display.Size[2] = 8
		r.Screen.Size = [2]float64{.9, .9}
		if err := Validate(r); err == nil || !strings.Contains(err.Error(), "actual flat display face") {
			t.Fatalf("screen extending into rounded display corner must fail, got %v", err)
		}
		r.Screen.Size = [2]float64{.8, .8}
		if err := Validate(r); err != nil {
			t.Fatalf("screen fully inside the planar face should pass: %v", err)
		}
	})
	t.Run("cylinder curved side is unsupported", func(t *testing.T) {
		r := validRecipe()
		display := &r.Parts[1]
		display.Kind = "cylinder"
		display.Size = [3]float64{64, 64, 1}
		r.Screen.Face = "right:+x"
		if err := Validate(r); err == nil || !strings.Contains(err.Error(), "actual flat display face") {
			t.Fatalf("flat screen on curved cylinder side must fail, got %v", err)
		}
	})
}

// Avoid making the contract test depend on production canonicalization internals.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
