package rendering

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSceneViewRetainedDeviceGLBProbe is an opt-in parser audit for the pinned
// E4 research models. It establishes parser coverage only; it does not import,
// dimension-qualify, or approve either model for the starter library.
func TestSceneViewRetainedDeviceGLBProbe(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E4_SCENEVIEW_PROBE") != "1" {
		t.Skip("set DEVICE_STUDIO_E4_SCENEVIEW_PROBE=1 to inspect retained E4 assets")
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../docs/internal/goal/evidence/E4/free-models/sceneview-research"))
	for _, tc := range []struct {
		name, file, digest string
		bytes              int
		meshes             int
		triangles          uint64
	}{
		{"Apple Watch Series 7", "apple_watch_series_7.glb", "ac3c33a32bc27c562e6655245e20327d17c77c988f11aab3016496d851e4e5e3", 5904504, 15, 156139},
		{"iPad mini 6", "ipad_mini_6.glb", "a462088b7bd84b7752802c109beee04110047c3b1a4f355624c863f790e93d5e", 1986244, 13, 39462},
	} {
		data, err := os.ReadFile(filepath.Join(root, tc.file))
		if err != nil {
			t.Fatalf("read %s: %v", tc.name, err)
		}
		if len(data) != tc.bytes || sha256Bytes(data) != tc.digest {
			t.Fatalf("%s retained input changed: bytes=%d sha256=%s", tc.name, len(data), sha256Bytes(data))
		}
		inv, err := InspectPhysicalGLB(data)
		if err != nil {
			// Preserve the observed parser boundary without making rejection a
			// contract: parser support can improve independently of these assets.
			t.Logf("%s pinned source verified; parser inventory unavailable: %v", tc.name, err)
			continue
		}
		if len(inv.Meshes) != tc.meshes || inv.TriangleCount != tc.triangles || len(inv.Extensions) != 0 {
			t.Fatalf("%s parser inventory changed: meshes=%d triangles=%d extensions=%v", tc.name, len(inv.Meshes), inv.TriangleCount, inv.Extensions)
		}
		t.Logf("%s: nodes=%d meshes=%d materials=%d vertices=%d triangles=%d", tc.name, len(inv.Nodes), len(inv.Meshes), len(inv.Materials), inv.VertexCount, inv.TriangleCount)
		for _, mesh := range inv.Meshes {
			for _, primitive := range mesh.Primitives {
				if primitive.UV0 != nil {
					t.Logf("%s mesh[%d]=%q primitive[%d] material=%d uv=[%v,%v] triangles=%d", tc.name, mesh.Index, mesh.Name, primitive.Index, primitive.Material, primitive.UVMin, primitive.UVMax, primitive.TriangleCount)
				}
			}
		}
	}
}

// TestPhysicalGLBRetainedAssetProbe is opt-in because the retained study files
// are evidence inputs, not package test fixtures. Supply exact files through
// the two environment variables to reproduce the source-compatibility check.
func TestPhysicalGLBRetainedAssetProbe(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_GLB") == "" || os.Getenv("DEVICE_STUDIO_E3_SCENE_GLB") == "" {
		t.Skip("set both DEVICE_STUDIO_E3_PHONE_GLB and DEVICE_STUDIO_E3_SCENE_GLB to inspect retained assets")
	}
	for _, item := range []struct {
		name string
		env  string
		sha  string
	}{
		{"authored phone", "DEVICE_STUDIO_E3_PHONE_GLB", "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36"},
		{"exported table scene", "DEVICE_STUDIO_E3_SCENE_GLB", "d03c288c1f6fa401d920c736e1361189066599b0c514f924b9fed8b50ba16587"},
	} {
		path := os.Getenv(item.env)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", item.name, err)
		}
		if got := sha256Bytes(data); got != item.sha {
			t.Fatalf("%s digest changed: got %s want %s", item.name, got, item.sha)
		}
		inventory, err := InspectPhysicalGLB(data)
		if err != nil {
			t.Fatalf("inspect %s: %v", item.name, err)
		}
		t.Logf("%s bytes=%d nodes=%d meshes=%d materials=%d extensions=%v vertices=%d triangles=%d", item.name, len(data), len(inventory.Nodes), len(inventory.Meshes), len(inventory.Materials), inventory.Extensions, inventory.VertexCount, inventory.TriangleCount)
		if item.name == "authored phone" {
			for _, node := range inventory.Nodes {
				if strings.Contains(strings.ToLower(node.Name), "screen") && node.Mesh != nil {
					t.Logf("display_node=%d mesh=%d primitives=%+v", node.Index, *node.Mesh, inventory.Meshes[*node.Mesh].Primitives)
				}
			}
		}
	}
}
