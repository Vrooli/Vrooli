package rendering

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	phoneTableExportManifestSHA = "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf"
	phoneTableExportScriptSHA   = "2a36b813c050497524a75d170deb53f4d23fb8d3c39054f3d5836a34abfa6659"
	phoneTableExportProfileID   = "device-studio-phone-table-scene-export-v1"
	phoneTableExportGLBMax      = 32 << 20
	phoneTableExportReceiptMax  = 1 << 20
)

var phoneTableExportNodeNames = []string{
	"TableTop", "TableLeg-NW", "TableLeg-NE", "TableLeg-SW", "TableLeg-SE",
	"TeaCup", "TeaCupHandle", "TeaSurface",
}

type phoneTableExportAsset struct {
	source, staged, sha string
	size, max           int64
}

type phoneTableExportReceipt struct {
	Profile     string `json:"profile_id"`
	Status      string `json:"status"`
	ManifestSHA string `json:"manifest_sha256"`
	ScriptSHA   string `json:"export_script_sha256"`
	GLBSHA      string `json:"glb_sha256"`
	GLBBytes    int64  `json:"glb_size_bytes"`
	Coordinate  string `json:"coordinate_convention"`
	Nodes       map[string]struct {
		NodeIndex   *int      `json:"node_index"`
		MeshIndex   *int      `json:"mesh_index"`
		Parent      *int      `json:"parent_index"`
		Translation []float64 `json:"translation"`
		Rotation    []float64 `json:"rotation_xyzw"`
		Scale       []float64 `json:"scale"`
		Primitives  []struct {
			Index          *int   `json:"primitive_index"`
			Mode           *int   `json:"mode"`
			MaterialIndex  *int   `json:"material_index"`
			MaterialName   string `json:"material_name"`
			PositionAccess *int   `json:"position_accessor"`
		} `json:"primitives"`
	} `json:"nodes"`
	Materials []struct {
		Index *int   `json:"index"`
		Name  string `json:"name"`
	} `json:"materials"`
	Extensions struct {
		Used     []string `json:"used"`
		Required []string `json:"required"`
	} `json:"gltf_extensions"`
	Textures []struct {
		ID     string `json:"binding_id"`
		Role   string `json:"library_role"`
		SHA256 string `json:"sha256"`
		Bytes  int64  `json:"size_bytes"`
	} `json:"external_texture_artifacts"`
}

// This opt-in test exports a fixed scene candidate through the owner-managed
// Blender operation. It does not launch Blender directly or change source assets.
func TestOptInPhoneTableSceneExport(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_TABLE_EXPORT") != "1" {
		t.Skip("opt-in phone table scene candidate export")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("qualified Blender profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_PHONE_TABLE_EXPORT_DIR"))
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_TABLE_EXPORT_DIR") == "" || !filepath.IsAbs(root) || !strings.Contains(filepath.ToSlash(root), "/goal/evidence/E3/scene-studies/phone/scene-candidate/runs/") || filepath.Base(root) == "runs" || filepath.Base(root) == "." {
		t.Fatalf("evidence path outside scene-candidate runs: %q", root)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	profileBytes, profile := resolveBlenderProfile(t)
	if err := writeDurable(root, "resolved-profile.json", profileBytes); err != nil {
		t.Fatal(err)
	}
	bin := profile.GetBlender()
	if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || profile.GetProfileId() != blenderProfileID || profile.GetArtifactIdentity() != blenderArtifact || profile.GetExpectedTreeSha256() != blenderTreeSHA || bin == nil || bin.GetVersion() != blenderVersion || bin.GetSha256() != blenderBinarySHA || bin.GetArchiveMemberPath() != blenderArchive+"/blender" || !filepath.IsAbs(bin.GetPath()) {
		t.Fatalf("profile differs from pinned artifact: %+v", profile)
	}
	if filepath.Base(filepath.Dir(bin.GetPath())) != blenderArchive {
		t.Fatalf("binary outside expected archive root: %q", bin.GetPath())
	}
	attempt := filepath.Join(root, "cpu-export")
	if err := os.Mkdir(attempt, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestSHA, scriptSHA := stagePhoneTableExportInputs(t, attempt)
	runOwnedBlenderOperation(t, attempt, bin.GetPath(), profile.GetArtifactIdentity(), "CPU", "phone-table-scene-export", "", manifestSHA, scriptSHA, func() error {
		return verifyPhoneTableExport(attempt, manifestSHA, scriptSHA)
	})
}

func stagePhoneTableExportInputs(t *testing.T, attempt string) (string, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate harness source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	assetsRoot := filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-table")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetsRoot, "export-manifest.json"), phoneTableExportManifestSHA, 128<<10)
	script := readPinnedPhoneFile(t, filepath.Join(assetsRoot, "export_scene.py"), phoneTableExportScriptSHA, 1<<20)
	assets := []phoneTableExportAsset{
		{"/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/authored/original-iphone16pro-iter03.glb", "inputs/phone.glb", "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36", 2583052, 8 << 20},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E2/presentations/phone-v3-auto-1206x2622.png"), "inputs/phone-screen.png", "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8", 611442, 2 << 20},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Color.jpg"), "inputs/table-color.jpg", "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", 16152869, 20 << 20},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_NormalGL.jpg"), "inputs/table-normal.jpg", "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", 5152752, 8 << 20},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Roughness.jpg"), "inputs/table-roughness.jpg", "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", 5302649, 8 << 20},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Displacement.jpg"), "inputs/table-height.jpg", "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", 355941, 1 << 20},
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	bindings := make([]map[string]any, 0, len(assets))
	var aggregate int64
	for _, asset := range assets {
		data := readPinnedPhoneFile(t, asset.source, asset.sha, asset.max)
		if int64(len(data)) != asset.size {
			t.Fatalf("pinned export input size differs: %s", asset.source)
		}
		aggregate += int64(len(data))
		if aggregate > 40<<20 {
			t.Fatal("fixed provenance input aggregate exceeds 40 MiB")
		}
		if err := writeDurable(attempt, asset.staged, data); err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, map[string]any{"source": asset.source, "stage": asset.staged, "sha256": asset.sha, "bytes": asset.size, "purpose": "provenance_only_unless_export_profile_uses_role"})
	}
	if err := writeDurable(attempt, "export-manifest.json", manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "export_scene.py", script); err != nil {
		t.Fatal(err)
	}
	proof, err := json.MarshalIndent(map[string]any{"manifest_sha256": phoneTableExportManifestSHA, "script_sha256": phoneTableExportScriptSHA, "assets": bindings}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "source-bindings.json", append(proof, '\n')); err != nil {
		t.Fatal(err)
	}
	return phoneTableExportManifestSHA, phoneTableExportScriptSHA
}

func verifyPhoneTableExport(dir, manifestSHA, scriptSHA string) error {
	outputs := filepath.Join(dir, "outputs")
	if err := verifyPhoneTableOutputNames(outputs); err != nil {
		return err
	}
	inventoryBytes, err := readPhoneTableRegular(filepath.Join(outputs, "scene-inventory.json"), phoneTableExportReceiptMax)
	if err != nil {
		return fmt.Errorf("scene inventory: %w", err)
	}
	var inventory phoneTableExportReceipt
	if err := json.Unmarshal(inventoryBytes, &inventory); err != nil {
		return err
	}
	glb, err := readPhoneTableRegular(filepath.Join(outputs, "phone-table-scene.glb"), phoneTableExportGLBMax)
	if err != nil {
		return fmt.Errorf("scene GLB: %w", err)
	}
	if inventory.Profile != phoneTableExportProfileID || inventory.Status != "geometry_candidate_unreviewed" || inventory.ManifestSHA != manifestSHA || inventory.ScriptSHA != scriptSHA || inventory.GLBBytes != int64(len(glb)) || inventory.GLBBytes <= 0 || inventory.GLBBytes > phoneTableExportGLBMax || inventory.GLBSHA != blenderDigest(glb) || inventory.Coordinate != "device-studio-right-handed-meters-x-right-y-up-z-front-v1" {
		return errors.New("inventory identity or output bounds differ from fixed export")
	}
	return verifyPhoneTableGLB(glb, inventory)
}

func verifyPhoneTableGLB(data []byte, inventory phoneTableExportReceipt) error {
	if len(data) < 20 || string(data[:4]) != "glTF" || blenderDigest(data) != inventory.GLBSHA {
		return errors.New("GLB header or digest differs from inventory")
	}
	version, declared := littleUint32(data[4:8]), littleUint32(data[8:12])
	if version != 2 || declared != uint32(len(data)) {
		return errors.New("GLB version or declared length differs")
	}
	jsonSize, jsonType := littleUint32(data[12:16]), littleUint32(data[16:20])
	if jsonType != 0x4e4f534a || jsonSize > uint32(len(data)-20) || len(data) < 28+int(jsonSize) {
		return errors.New("GLB JSON chunk is absent or out of bounds")
	}
	binHeader := 20 + int(jsonSize)
	binSize, binType := littleUint32(data[binHeader:binHeader+4]), littleUint32(data[binHeader+4:binHeader+8])
	if binType != 0x004e4942 || uint64(binHeader+8)+uint64(binSize) != uint64(len(data)) {
		return errors.New("GLB must end with exactly one bounded BIN chunk")
	}
	doc := struct {
		Nodes []struct {
			Name        string                     `json:"name"`
			Mesh        *int                       `json:"mesh"`
			Children    []int                      `json:"children"`
			Translation []float64                  `json:"translation"`
			Rotation    []float64                  `json:"rotation"`
			Scale       []float64                  `json:"scale"`
			Extensions  map[string]json.RawMessage `json:"extensions"`
		} `json:"nodes"`
		Meshes []struct {
			Primitives []struct {
				Mode       *int           `json:"mode"`
				Material   *int           `json:"material"`
				Attributes map[string]int `json:"attributes"`
			} `json:"primitives"`
		} `json:"meshes"`
		Materials []struct {
			Name string `json:"name"`
		} `json:"materials"`
		Accessors []struct {
			Min []float64 `json:"min"`
			Max []float64 `json:"max"`
		} `json:"accessors"`
		Buffers []struct {
			URI string `json:"uri"`
		} `json:"buffers"`
		Images []struct {
			URI string `json:"uri"`
		} `json:"images"`
		Textures           []json.RawMessage `json:"textures"`
		Cameras            []json.RawMessage `json:"cameras"`
		Animations         []json.RawMessage `json:"animations"`
		Skins              []json.RawMessage `json:"skins"`
		ExtensionsUsed     []string          `json:"extensionsUsed"`
		ExtensionsRequired []string          `json:"extensionsRequired"`
	}{}
	if err := json.Unmarshal(bytes.TrimRight(data[20:20+jsonSize], " \t\r\n\x00"), &doc); err != nil {
		return fmt.Errorf("decode GLB JSON: %w", err)
	}
	if len(doc.Nodes) != len(phoneTableExportNodeNames) || len(doc.Meshes) != len(phoneTableExportNodeNames) || len(doc.Buffers) != 1 || len(doc.Images) != 0 || len(doc.Cameras) != 0 || len(doc.Animations) != 0 || len(doc.Skins) != 0 || len(doc.Textures) != 0 {
		return errors.New("GLB contains an unexpected node, mesh, camera, animation, skin, or texture")
	}
	if len(doc.ExtensionsUsed) != 0 || len(doc.ExtensionsRequired) != 0 {
		return errors.New("GLB contains an unapproved extension")
	}
	for _, node := range doc.Nodes {
		if len(node.Extensions) != 0 {
			return errors.New("GLB node contains unapproved extension data")
		}
	}
	for _, group := range [][]struct {
		URI string `json:"uri"`
	}{doc.Buffers, doc.Images} {
		for _, item := range group {
			if item.URI != "" {
				return errors.New("GLB contains an external URI")
			}
		}
	}
	if doc.Buffers[0].URI != "" {
		return errors.New("GLB buffer must use its embedded BIN chunk")
	}
	wantNodes := append([]string(nil), phoneTableExportNodeNames...)
	sort.Strings(wantNodes)
	gotNodes := make([]string, 0, len(doc.Nodes))
	materialNames := make([]string, 0, len(doc.Materials))
	for _, material := range doc.Materials {
		materialNames = append(materialNames, material.Name)
	}
	sort.Strings(materialNames)
	wantMaterials := []string{"Dark tea scale cue", "Pale warm mineral sides", "Travertine001 top material slot", "Warm ivory glazed ceramic", "Warm pale timber support legs"}
	if !equalStringSlices(materialNames, wantMaterials) {
		return errors.New("GLB material inventory differs")
	}
	if len(inventory.Materials) != len(doc.Materials) {
		return errors.New("inventory material count differs from GLB")
	}
	for i, material := range inventory.Materials {
		if material.Index == nil || *material.Index != i || material.Name != doc.Materials[i].Name {
			return errors.New("inventory material indices/names differ from GLB")
		}
	}
	actualNodeIndices := make(map[string]int, len(doc.Nodes))
	actualParents := make(map[int]int, len(doc.Nodes))
	for nodeIndex, node := range doc.Nodes {
		gotNodes = append(gotNodes, node.Name)
		actualNodeIndices[node.Name] = nodeIndex
		if node.Mesh == nil || *node.Mesh < 0 || *node.Mesh >= len(doc.Meshes) {
			return fmt.Errorf("node %q has no valid mesh", node.Name)
		}
		mesh := doc.Meshes[*node.Mesh]
		if len(mesh.Primitives) == 0 {
			return fmt.Errorf("node %q has no mesh primitives", node.Name)
		}
		for _, primitive := range mesh.Primitives {
			if primitive.Mode != nil && *primitive.Mode != 4 {
				return fmt.Errorf("node %q has non-triangle primitive", node.Name)
			}
			if primitive.Material == nil || *primitive.Material < 0 || *primitive.Material >= len(doc.Materials) {
				return fmt.Errorf("node %q has invalid material slot", node.Name)
			}
		}
		for _, child := range node.Children {
			if child < 0 || child >= len(doc.Nodes) {
				return errors.New("GLB node child index is out of range")
			}
			if _, exists := actualParents[child]; exists {
				return errors.New("GLB node has more than one parent")
			}
			actualParents[child] = nodeIndex
		}
	}
	sort.Strings(gotNodes)
	if !equalStringSlices(gotNodes, wantNodes) {
		return errors.New("GLB mesh-node names differ from fixed eight-node inventory")
	}
	if len(inventory.Nodes) != len(wantNodes) || len(inventory.Materials) != len(wantMaterials) || len(inventory.Textures) != 4 || len(inventory.Extensions.Used) != 0 || len(inventory.Extensions.Required) != 0 {
		return errors.New("inventory component/material/texture/extension census differs")
	}
	for _, name := range wantNodes {
		node, ok := inventory.Nodes[name]
		if !ok || node.NodeIndex == nil || node.MeshIndex == nil || *node.NodeIndex != actualNodeIndices[name] || *node.MeshIndex < 0 || *node.MeshIndex >= len(doc.Meshes) || doc.Nodes[*node.NodeIndex].Mesh == nil || *doc.Nodes[*node.NodeIndex].Mesh != *node.MeshIndex || len(node.Primitives) == 0 {
			return fmt.Errorf("inventory lacks node/primitive binding for %s", name)
		}
		glbNode := doc.Nodes[*node.NodeIndex]
		parent, hasParent := actualParents[*node.NodeIndex]
		if hasParent != (node.Parent != nil) || hasParent && *node.Parent != parent || !equalFloats(node.Translation, defaultFloatSlice(glbNode.Translation, []float64{0, 0, 0})) || !equalFloats(node.Rotation, defaultFloatSlice(glbNode.Rotation, []float64{0, 0, 0, 1})) || !equalFloats(node.Scale, defaultFloatSlice(glbNode.Scale, []float64{1, 1, 1})) {
			return fmt.Errorf("inventory node transform or hierarchy differs for %s", name)
		}
		glbPrimitives := doc.Meshes[*node.MeshIndex].Primitives
		if len(node.Primitives) != len(glbPrimitives) {
			return fmt.Errorf("inventory primitive count differs for %s", name)
		}
		for i, primitive := range node.Primitives {
			glbPrimitive := glbPrimitives[i]
			if primitive.Index == nil || *primitive.Index != i || primitive.Mode == nil || *primitive.Mode != 4 || primitive.MaterialIndex == nil || *primitive.MaterialIndex < 0 || *primitive.MaterialIndex >= len(doc.Materials) || glbPrimitive.Material == nil || *primitive.MaterialIndex != *glbPrimitive.Material || primitive.MaterialName != doc.Materials[*primitive.MaterialIndex].Name || primitive.PositionAccess == nil || *primitive.PositionAccess < 0 || *primitive.PositionAccess >= len(doc.Accessors) || glbPrimitive.Attributes["POSITION"] != *primitive.PositionAccess || len(doc.Accessors[*primitive.PositionAccess].Min) != 3 || len(doc.Accessors[*primitive.PositionAccess].Max) != 3 {
				return fmt.Errorf("inventory has invalid triangle/material binding for %s", name)
			}
			if glbPrimitive.Mode != nil && *glbPrimitive.Mode != 4 {
				return fmt.Errorf("GLB primitive is not triangles for %s", name)
			}
		}
	}
	wantTextures := map[string]struct {
		sha  string
		size int64
	}{
		"travertine-color":     {"e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", 16152869},
		"travertine-normal":    {"68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", 5152752},
		"travertine-roughness": {"a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", 5302649},
		"travertine-height":    {"64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", 355941},
	}
	for _, texture := range inventory.Textures {
		want, ok := wantTextures[texture.ID]
		if !ok || texture.Role != "texture_image" || texture.SHA256 != want.sha || texture.Bytes != want.size {
			return errors.New("external texture inventory binding is malformed")
		}
		delete(wantTextures, texture.ID)
	}
	if len(wantTextures) != 0 {
		return errors.New("external texture inventory is incomplete")
	}
	return nil
}

func littleUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func defaultFloatSlice(values, defaults []float64) []float64 {
	if values == nil {
		return defaults
	}
	return values
}

func verifyPhoneTableOutputNames(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("scene output path is not a regular directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		return errors.New("scene output directory must contain exactly two fixed outputs")
	}
	want := map[string]bool{"phone-table-scene.glb": true, "scene-inventory.json": true}
	for _, entry := range entries {
		item, e := entry.Info()
		if !want[entry.Name()] || e != nil || !item.Mode().IsRegular() || item.Mode()&os.ModeSymlink != 0 {
			return errors.New("scene output directory contains an unexpected or unsafe entry")
		}
	}
	return nil
}

func readPhoneTableRegular(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > max {
		return nil, errors.New("file missing, unsafe, or outside fixed size bound")
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > max {
		return nil, errors.New("bounded file read failed")
	}
	return data, nil
}
