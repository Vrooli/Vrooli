package rendering

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	phoneEnvironmentManifestSHA = "bc89ac5b69ef6cd58ef788072eee89fa9c12598c7f76664a838d25623d6617ea"
	phoneEnvironmentScriptSHA   = "1cadc29e06491aae213963d731b7edcc7145ebbb54072baef48522b2ce7564b9"
	phoneStarterManifestSHA     = "bd6e1387f59fc6615b262047ae5d9cd43fc35d991fd0b824f762bd2ba8aa1021"
	phoneStarterScriptSHA       = "2cdc2ae6072459761f94fbaa3b8c284695c9cc371d186f1742f882528ff2f6f2"
	phoneTableGLBSHA            = "d03c288c1f6fa401d920c736e1361189066599b0c514f924b9fed8b50ba16587"
	phoneTableGLBBytes          = 350488
	ipadEnvironmentManifestSHA  = "bc8bb2a1eebe3053cd0ae9e767032ecd7fdaa173a72c36b2b6864c22c2eec5b5"
	ipadEnvironmentScriptSHA    = "a4bc2706dca107973a3355bc921c4b2593abb2d488486b31c1725d44929d2ef1"
	ipadEnvironmentMathSHA      = "c45a8996b631d51d48053ae345a01b8a668db68722fa62ce5b96db7a772610d2"
	ipadEnvironmentSourceSHA    = "c0f1450af8cc83b237159f2c5208bd964c48fc5dc9fe7f2ac2b2ee5409447f4d"
	ipadEnvironmentSourceBytes  = 421404
	ipadSceneDeviceSHA          = "0326fd47e7df4a065978eaae76e6789cb3027f54bab6f3f7fdef1bdfb867e52e"
	ipadSceneDeviceBytes        = 1939592
	ipadCupGeneratorSHA         = "2a36b813c050497524a75d170deb53f4d23fb8d3c39054f3d5836a34abfa6659"
	ipadInventoryHelperSHA      = "1cadc29e06491aae213963d731b7edcc7145ebbb54072baef48522b2ce7564b9"
	assetDerivationGLBMax       = 32 << 20
	assetDerivationReceiptMax   = 1 << 20
)

var phoneEnvironmentNodes = []string{
	"TableLeg-NE", "TableLeg-NW", "TableLeg-SE", "TableLeg-SW", "TableTop", "TeaCup", "TeaCupHandle", "TeaSurface",
	"RoomBackWall", "WindowLightPanel", "WindowFrameUprightLeft", "WindowFrameUprightRight", "WindowFrameRailLower", "WindowFrameRailUpper",
}

type ipadEnvironmentReceipt struct {
	Profile     string `json:"profile_id"`
	Status      string `json:"status"`
	ManifestSHA string `json:"manifest_sha256"`
	ScriptSHA   string `json:"export_script_sha256"`
	SourceSHA   string `json:"source_scene_glb_sha256"`
	SourceBytes int64  `json:"source_scene_glb_bytes"`
	DeviceSHA   string `json:"projection_device_glb_sha256"`
	DeviceBytes int64  `json:"projection_device_glb_bytes"`
	GLBSHA      string `json:"glb_sha256"`
	GLBBytes    int64  `json:"glb_size_bytes"`
	Triangles   uint64 `json:"triangle_count"`
	Identity    struct {
		BundleID   string `json:"bundle_id"`
		ArtifactID string `json:"artifact_id"`
		SceneID    string `json:"scene_id"`
		CameraID   string `json:"camera_id"`
		Status     string `json:"status"`
	} `json:"candidate_identity"`
	Cup struct {
		GeneratorSHA           string    `json:"shared_generator_sha256"`
		InventorySHA           string    `json:"shared_inventory_helper_sha256"`
		TableManifestSHA       string    `json:"base_table_manifest_sha256"`
		EnvironmentManifestSHA string    `json:"base_environment_manifest_sha256"`
		Segments               int       `json:"radial_segments"`
		Target                 []float64 `json:"projected_center_meters_blender_xy"`
		Placement              []float64 `json:"placement_delta_meters_blender_xy"`
		Device                 []float64 `json:"projected_tablet_bounds_normalized"`
		Display                []float64 `json:"projected_display_bounds_normalized"`
		Cup                    []float64 `json:"projected_cup_bounds_normalized"`
		Clearance              float64   `json:"projected_clearance_from_full_tablet"`
		Required               float64   `json:"minimum_required_clearance"`
		Margin                 float64   `json:"frame_margin"`
	} `json:"cup_refinement"`
}

type assetDerivationReceipt struct {
	Profile       string `json:"profile_id"`
	Status        string `json:"status"`
	ManifestSHA   string `json:"manifest_sha256"`
	ScriptSHA     string `json:"export_script_sha256"`
	SourceSHA     string `json:"source_table_glb_sha256"`
	SourceBytes   int64  `json:"source_table_glb_bytes"`
	GLBSHA        string `json:"glb_sha256"`
	GLBBytes      int64  `json:"glb_size_bytes"`
	TriangleCount uint64 `json:"triangle_count"`
	Nodes         map[string]struct {
		NodeIndex  uint32 `json:"node_index"`
		MeshIndex  uint32 `json:"mesh_index"`
		Primitives []struct {
			Index         uint32 `json:"primitive_index"`
			Mode          uint32 `json:"mode"`
			MaterialIndex uint32 `json:"gltf_material_index"`
		} `json:"primitives"`
	} `json:"nodes"`
	Materials []struct {
		Index    uint32    `json:"index"`
		Name     string    `json:"name"`
		Emissive []float64 `json:"emissive_factor_linear_rgb"`
	} `json:"materials"`
	Emission struct {
		Profile string `json:"profile"`
	} `json:"window_emission_conversion"`
}

type phoneStarterReceipt struct {
	Profile        string `json:"profile_id"`
	Status         string `json:"status"`
	SourceSHA      string `json:"source_glb_sha256"`
	SourceBytes    int64  `json:"source_size_bytes"`
	SourceImageSHA string `json:"source_embedded_display_sha256"`
	DisplayRemoved bool   `json:"display_pixels_removed"`
	DisplaySHA     string `json:"derived_display_png_sha256"`
	DisplaySize    []int  `json:"display_dimensions_pixels"`
	GLBSHA         string `json:"derived_glb_sha256"`
	GLBBytes       int64  `json:"derived_size_bytes"`
	ManifestSHA    string `json:"manifest_sha256"`
	ScriptSHA      string `json:"export_script_sha256"`
	Meshes         int    `json:"mesh_node_count"`
	Triangles      int    `json:"triangle_count"`
	Island         struct {
		Node     int `json:"island_node_index"`
		Mesh     int `json:"island_mesh_index"`
		Material int `json:"island_material_index"`
	} `json:"island_binding"`
	IslandMaterial struct {
		Name string `json:"name"`
		PBR  struct {
			Metallic  float64 `json:"metallicFactor"`
			Roughness float64 `json:"roughnessFactor"`
		} `json:"pbrMetallicRoughness"`
		Extensions map[string]json.RawMessage `json:"extensions"`
	} `json:"island_material"`
}

func TestOptInPhoneStudyEnvironmentExport(t *testing.T) {
	runAssetDerivation(t, "DEVICE_STUDIO_E3_PHONE_ENVIRONMENT_EXPORT", "DEVICE_STUDIO_E3_PHONE_ENVIRONMENT_EXPORT_DIR", "phone-environment", "phone-study-environment-export", stagePhoneEnvironment, verifyPhoneEnvironment)
}

func TestOptInIPadStudyEnvironmentExport(t *testing.T) {
	runAssetDerivation(t, "DEVICE_STUDIO_E3_IPAD_ENVIRONMENT_EXPORT", "DEVICE_STUDIO_E3_IPAD_ENVIRONMENT_EXPORT_DIR", "ipad-environment", "ipad-study-environment-export", stageIPadEnvironment, verifyIPadEnvironment)
}

func TestIPadStudyEnvironmentSourceContract(t *testing.T) {
	_, repo := laptopTestRoots(t)
	assetRoot := filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-study-environment")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "ipad-scene-manifest.json"), ipadEnvironmentManifestSHA, 128<<10)
	readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_ipad_environment.py"), ipadEnvironmentScriptSHA, 1<<20)
	readPinnedPhoneFile(t, filepath.Join(assetRoot, "ipad_environment_math.py"), ipadEnvironmentMathSHA, 128<<10)
	readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-table/export_scene.py"), ipadCupGeneratorSHA, 1<<20)
	readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_environment.py"), ipadInventoryHelperSHA, 1<<20)
	var contract struct {
		Profile string `json:"profile_id"`
		Input   struct {
			Scene struct {
				SHA   string `json:"sha256"`
				Bytes int64  `json:"size_bytes"`
			} `json:"scene_glb"`
			Device struct {
				SHA   string `json:"sha256"`
				Bytes int64  `json:"size_bytes"`
			} `json:"device_glb"`
			Math struct {
				SHA  string `json:"sha256"`
				Name string `json:"staged_name"`
			} `json:"math_helper"`
			Cup map[string]any `json:"cup_generator"`
		} `json:"input"`
		Cup struct {
			Segments int `json:"radial_segments"`
		} `json:"cup"`
		Projection struct {
			CandidateID string  `json:"camera_candidate_id"`
			Margin      float64 `json:"image_margin_normalized"`
			Gap         float64 `json:"minimum_gap_from_device_normalized"`
			Camera      struct {
				Translation any     `json:"translation_m"`
				Rotation    any     `json:"rotation_xyzw"`
				Focal       float64 `json:"focal_length_mm"`
				Sensor      float64 `json:"sensor_width_mm"`
				Aspect      float64 `json:"aspect"`
			} `json:"camera"`
			Placement struct {
				SceneSupportWorld struct {
					Translation any `json:"translation_m"`
				} `json:"scene_support_world"`
				SupportPivot   any `json:"support_pivot_m"`
				DeviceRootRest struct {
					Translation any `json:"translation_m"`
					Rotation    any `json:"rotation_xyzw"`
				} `json:"device_root_rest"`
				SourceToCanonical struct {
					Rotation any            `json:"rotation_xyzw"`
					Scale    map[string]any `json:"scale"`
				} `json:"source_to_canonical"`
			} `json:"device_placement"`
		} `json:"camera_projection"`
		Output struct {
			Nodes []string `json:"expected_mesh_nodes"`
		} `json:"expected_output"`
		Identity struct {
			BundleID   string `json:"bundle_id"`
			ArtifactID string `json:"artifact_id"`
			SceneID    string `json:"scene_id"`
			CameraID   string `json:"camera_id"`
		} `json:"candidate_identity"`
	}
	if err := json.Unmarshal(manifest, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Profile != "device-studio-ipad-study-environment-export-v2" || contract.Input.Scene.SHA != ipadEnvironmentSourceSHA || contract.Input.Scene.Bytes != ipadEnvironmentSourceBytes || contract.Input.Device.SHA != ipadSceneDeviceSHA || contract.Input.Device.Bytes != ipadSceneDeviceBytes || contract.Input.Math.SHA != ipadEnvironmentMathSHA || contract.Input.Math.Name != "ipad_environment_math.py" || contract.Cup.Segments != 128 || contract.Projection.CandidateID != "ipad-study-camera-16x9-fit-screen-uv-oriented-180-v3" || contract.Projection.Margin != .04 || contract.Projection.Gap != .10 || contract.Projection.Camera.Focal != 78 || len(contract.Output.Nodes) != len(phoneEnvironmentNodes) || contract.Identity.BundleID != "device-studio-starter-ipad-mini-6-ipad-environment-v2" || contract.Identity.ArtifactID != "ipad-study-environment-v2-glb" || contract.Identity.SceneID != "ipad-study-environment-screen-uv-oriented-180-v2-cup128-fit-camera-v3" || contract.Identity.CameraID != contract.Projection.CandidateID {
		t.Fatal("iPad scene source contract differs from its fixed scene, device, cup, projection or output census")
	}
	cameraTranslation, cameraErr := manifestComponents(contract.Projection.Camera.Translation, []string{"x", "y", "z"})
	if cameraErr != nil || len(cameraTranslation) != 3 || math.Abs(cameraTranslation[0]-.1002994491448368) > 1e-12 || math.Abs(cameraTranslation[1]-1.409043851488113) > 1e-12 || math.Abs(cameraTranslation[2]-.5300879317248927) > 1e-12 {
		t.Fatal("iPad camera v3 translation differs from its measured projection fit")
	}
	for name, vector := range map[string]any{
		"camera translation":           contract.Projection.Camera.Translation,
		"scene support translation":    contract.Projection.Placement.SceneSupportWorld.Translation,
		"support pivot":                contract.Projection.Placement.SupportPivot,
		"device root-rest translation": contract.Projection.Placement.DeviceRootRest.Translation,
	} {
		if _, err := manifestComponents(vector, []string{"x", "y", "z"}); err != nil {
			t.Errorf("%s must contain three finite numeric components", name)
		}
	}
	for name, quaternion := range map[string]any{
		"camera rotation":              contract.Projection.Camera.Rotation,
		"device root-rest rotation":    contract.Projection.Placement.DeviceRootRest.Rotation,
		"source-to-canonical rotation": contract.Projection.Placement.SourceToCanonical.Rotation,
	} {
		if _, err := manifestUnitQuaternion(quaternion); err != nil {
			t.Errorf("%s must be a finite unit XYZW quaternion", name)
		}
	}
	if !manifestVector3Values(contract.Projection.Placement.SourceToCanonical.Scale) || contract.Projection.Camera.Focal <= 0 || contract.Projection.Camera.Sensor <= 0 || contract.Projection.Camera.Aspect <= 0 {
		t.Fatal("camera optical parameters or source canonical scale are invalid")
	}
	support, supportErr := manifestComponents(contract.Projection.Placement.SceneSupportWorld.Translation, []string{"x", "y", "z"})
	rootRest, rootErr := manifestComponents(contract.Projection.Placement.DeviceRootRest.Translation, []string{"x", "y", "z"})
	if supportErr != nil || rootErr != nil || math.Abs(support[1]-.75) > 1e-9 || math.Abs(rootRest[1]) > .01 {
		t.Fatal("support world elevation and device-relative rest offset are inconsistent")
	}
	script := string(readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_ipad_environment.py"), ipadEnvironmentScriptSHA, 1<<20))
	for _, call := range []string{
		`value = (value["x"], value["y"], value["z"])`,
		`type(component) in (int, float) and math.isfinite(component)`,
		`xyzw = (value["x"], value["y"], value["z"], value["w"])`,
		`abs(math.sqrt(x * x + y * y + z * z + w * w) - 1.0) <= 1e-5`,
		`Quaternion((w, x, y, z))`,
		`manifest_vector3(placement["scene_support_world"]["translation_m"])`,
		`manifest_vector3(placement["support_pivot_m"])`,
		`manifest_vector3(rest["translation_m"])`,
		`manifest_vector3(spec["translation_m"])`,
		`manifest_quaternion(rest["rotation_xyzw"])`,
		`manifest_quaternion(canonical["rotation_xyzw"])`,
		`manifest_quaternion(spec["rotation_xyzw"])`,
	} {
		if !strings.Contains(script, call) {
			t.Errorf("exporter does not convert typed manifest vector at %s", call)
		}
	}
	for _, call := range []string{
		`math_helper.project_camera_local`, `math_helper.blender_to_canonical`,
		`math_helper.canonical_delta_from_blender_xy`, `tabletop_blender_xy_bounds_meters`,
		`imported == source_nodes == output_nodes`,
	} {
		if !strings.Contains(script, call) {
			t.Errorf("exporter is missing audited coordinate/scene contract %q", call)
		}
	}
	makeCup := strings.Index(script, "helper.make_cup(table_manifest, ceramic, tea)")
	placeCup := strings.Index(script, "cup_target(manifest, geometry, device)")
	if makeCup < 0 || placeCup < makeCup {
		t.Fatal("generated cup objects must exist before projected placement reads the cup group")
	}
	if !strings.Contains(script, `MATH_HELPER_SHA = "`+ipadEnvironmentMathSHA+`"`) {
		t.Fatal("exporter math helper pin differs from the manifest and staged helper")
	}
}

func manifestVector3Values(vector any) bool {
	_, err := manifestComponents(vector, []string{"x", "y", "z"})
	return err == nil
}

func manifestComponents(vector any, keys []string) ([]float64, error) {
	var values []any
	switch typed := vector.(type) {
	case map[string]any:
		if len(typed) != len(keys) {
			return nil, fmt.Errorf("expected exactly %d named components", len(keys))
		}
		for _, key := range keys {
			component, ok := typed[key]
			if !ok {
				return nil, fmt.Errorf("missing component %q", key)
			}
			values = append(values, component)
		}
	case []any:
		if len(typed) != len(keys) {
			return nil, fmt.Errorf("expected %d array components", len(keys))
		}
		values = typed
	default:
		return nil, fmt.Errorf("unsupported component type %T", vector)
	}
	result := make([]float64, len(values))
	for index, component := range values {
		value, ok := component.(float64)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("component %d is not finite numeric", index)
		}
		result[index] = value
	}
	return result, nil
}

func manifestUnitQuaternion(value any) ([]float64, error) {
	components, err := manifestComponents(value, []string{"x", "y", "z", "w"})
	if err != nil {
		return nil, err
	}
	norm := math.Sqrt(components[0]*components[0] + components[1]*components[1] + components[2]*components[2] + components[3]*components[3])
	if math.Abs(norm-1) > 1e-5 {
		return nil, fmt.Errorf("quaternion norm %.8g is not one", norm)
	}
	return components, nil
}

func projectedBoundsInsideFrame(bounds []float64, margin float64) bool {
	if len(bounds) != 4 || math.IsNaN(margin) || math.IsInf(margin, 0) || margin < 0 || margin >= .5 {
		return false
	}
	for _, value := range bounds {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return bounds[0] >= margin && bounds[1] >= margin && bounds[2] <= 1-margin && bounds[3] <= 1-margin
}

func TestIPadProjectedBoundsMustKeepFullObjectInsideFrame(t *testing.T) {
	for _, tc := range []struct {
		name   string
		bounds []float64
		margin float64
		want   bool
	}{
		{name: "exact margin accepted", bounds: []float64{.04, .04, .96, .96}, margin: .04, want: true},
		{name: "right edge violation rejected", bounds: []float64{.1, .1, .960001, .9}, margin: .04},
		{name: "top edge violation rejected", bounds: []float64{.1, .039, .9, .9}, margin: .04},
		{name: "malformed bounds rejected", bounds: []float64{.1, .2, .9}, margin: .04},
		{name: "nonfinite coordinate rejected", bounds: []float64{.1, .2, math.Inf(1), .9}, margin: .04},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := projectedBoundsInsideFrame(tc.bounds, tc.margin); got != tc.want {
				t.Fatalf("projectedBoundsInsideFrame(%v, %v) = %v, want %v", tc.bounds, tc.margin, got, tc.want)
			}
		})
	}
}

func TestIPadManifestVectorAndQuaternionForms(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		keys  []string
		valid bool
	}{
		{"vector-array", []any{float64(1), float64(2), float64(3)}, []string{"x", "y", "z"}, true},
		{"vector-object", map[string]any{"x": float64(1), "y": float64(2), "z": float64(3)}, []string{"x", "y", "z"}, true},
		{"quaternion-array", []any{float64(0), float64(0), float64(0), float64(1)}, []string{"x", "y", "z", "w"}, true},
		{"quaternion-object", map[string]any{"x": float64(0), "y": float64(0), "z": float64(0), "w": float64(1)}, []string{"x", "y", "z", "w"}, true},
		{"nonunit-quaternion", []any{float64(0), float64(0), float64(0), float64(2)}, []string{"x", "y", "z", "w"}, false},
		{"wrong-length", []any{float64(1), float64(2)}, []string{"x", "y", "z"}, false},
		{"string-component", map[string]any{"x": "0", "y": float64(0), "z": float64(0)}, []string{"x", "y", "z"}, false},
		{"nonfinite-component", []any{float64(0), math.Inf(1), float64(0)}, []string{"x", "y", "z"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if len(tc.keys) == 4 {
				_, err = manifestUnitQuaternion(tc.value)
			} else {
				_, err = manifestComponents(tc.value, tc.keys)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("manifest component validation err=%v, valid=%v", err, tc.valid)
			}
		})
	}
}

func TestStageIPadStudyEnvironmentPinnedInputs(t *testing.T) {
	attempt := t.TempDir()
	manifestSHA, scriptSHA := stageIPadEnvironment(t, attempt, "")
	if manifestSHA != ipadEnvironmentManifestSHA || scriptSHA != ipadEnvironmentScriptSHA {
		t.Fatal("iPad export stage returned different fixed manifest/script pins")
	}
	for path, want := range map[string]string{
		"ipad-scene-manifest.json":                       ipadEnvironmentManifestSHA,
		"export_ipad_environment.py":                     ipadEnvironmentScriptSHA,
		"ipad_environment_math.py":                       ipadEnvironmentMathSHA,
		"phone_table_export_source.py":                   ipadCupGeneratorSHA,
		"phone_environment_inventory_source.py":          ipadInventoryHelperSHA,
		"phone_table_manifest.json":                      "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf",
		"phone_environment_manifest.json":                phoneEnvironmentManifestSHA,
		"inputs/phone-study-environment.glb":             ipadEnvironmentSourceSHA,
		"inputs/ipad-mini-screen-uv-oriented-180-v2.glb": ipadSceneDeviceSHA,
	} {
		data, err := readPhoneTableRegular(filepath.Join(attempt, path), assetDerivationGLBMax)
		if err != nil || blenderDigest(data) != want {
			t.Fatalf("staged input %s differs from its exact pin: %v", path, err)
		}
	}
}

func TestOptInPhoneStarterDerivationExport(t *testing.T) {
	runAssetDerivation(t, "DEVICE_STUDIO_E3_PHONE_STARTER_DERIVE", "DEVICE_STUDIO_E3_PHONE_STARTER_DERIVE_DIR", "phone-starter", "phone-starter-derive", stagePhoneStarter, verifyPhoneStarter)
}

func TestOptInVerifyRetainedPhoneStarterDerivation(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_STARTER_VERIFY") != "1" {
		t.Skip("opt-in offline verification of retained phone derivation; does not launch Blender")
	}
	dir := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_PHONE_STARTER_VERIFY_DIR"))
	if !filepath.IsAbs(dir) || !strings.Contains(filepath.ToSlash(dir), "/goal/evidence/E3/scene-studies/phone/asset-derivation/runs/") {
		t.Fatalf("retained output path outside E3 phone derivation evidence: %q", dir)
	}
	if err := verifyPhoneStarter(dir, phoneStarterManifestSHA, phoneStarterScriptSHA); err != nil {
		t.Fatal(err)
	}
}

type derivationStager func(*testing.T, string, string) (string, string)
type derivationVerifier func(string, string, string) error

func runAssetDerivation(t *testing.T, optIn, evidenceDirEnv, leaf, purpose string, stage derivationStager, verify derivationVerifier) {
	t.Helper()
	if os.Getenv(optIn) != "1" {
		t.Skip("opt-in fixed E3 asset derivation export")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("pinned Blender profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv(evidenceDirEnv))
	rawRoot := os.Getenv(evidenceDirEnv)
	if rawRoot == "" || !filepath.IsAbs(root) || !strings.Contains(filepath.ToSlash(root), "/goal/evidence/E3/scene-studies/") || filepath.Base(root) == "." || filepath.Base(root) == "runs" {
		t.Fatalf("evidence path is outside the fixed E3 studies area: %q", rawRoot)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	profileBytes, profile := resolveBlenderProfile(t)
	if err := writeDurable(root, "resolved-profile.json", profileBytes); err != nil {
		t.Fatal(err)
	}
	bin := profile.GetBlender()
	if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || profile.GetProfileId() != blenderProfileID || profile.GetArtifactIdentity() != blenderArtifact || profile.GetExpectedTreeSha256() != blenderTreeSHA || bin == nil || bin.GetVersion() != blenderVersion || bin.GetSha256() != blenderBinarySHA || bin.GetArchiveMemberPath() != blenderArchive+"/blender" || !filepath.IsAbs(bin.GetPath()) || filepath.Base(filepath.Dir(bin.GetPath())) != blenderArchive {
		t.Fatalf("profile differs from exact admitted Blender artifact: %+v", profile)
	}
	attempt := filepath.Join(root, "cpu-export")
	if err := os.Mkdir(attempt, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestSHA, scriptSHA := stage(t, attempt, leaf)
	runOwnedBlenderOperation(t, attempt, bin.GetPath(), profile.GetArtifactIdentity(), "CPU", purpose, "", manifestSHA, scriptSHA, func() error {
		return verify(attempt, manifestSHA, scriptSHA)
	})
}

func stagePhoneEnvironment(t *testing.T, attempt, _ string) (string, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate rendering package source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	assetRoot := filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-study-environment")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "full-scene-manifest.json"), phoneEnvironmentManifestSHA, 128<<10)
	script := readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_environment.py"), phoneEnvironmentScriptSHA, 1<<20)
	source := readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/api/internal/physicalruntime/assets/phone-table/phone-table-scene.glb"), phoneTableGLBSHA, 2<<20)
	if len(source) != phoneTableGLBBytes {
		t.Fatal("pinned portable table GLB size differs")
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "inputs/phone-table-scene.glb", source); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "full-scene-manifest.json", manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "export_environment.py", script); err != nil {
		t.Fatal(err)
	}
	writeAssetDerivationBindings(t, attempt, phoneEnvironmentManifestSHA, phoneEnvironmentScriptSHA, []map[string]any{{"role": "table_scene_source", "stage": "inputs/phone-table-scene.glb", "sha256": phoneTableGLBSHA, "bytes": phoneTableGLBBytes}})
	return phoneEnvironmentManifestSHA, phoneEnvironmentScriptSHA
}

func stageIPadEnvironment(t *testing.T, attempt, _ string) (string, string) {
	t.Helper()
	_, repo := laptopTestRoots(t)
	assetRoot := filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-study-environment")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "ipad-scene-manifest.json"), ipadEnvironmentManifestSHA, 128<<10)
	script := readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_ipad_environment.py"), ipadEnvironmentScriptSHA, 1<<20)
	cupGenerator := readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-table/export_scene.py"), ipadCupGeneratorSHA, 1<<20)
	inventoryHelper := readPinnedPhoneFile(t, filepath.Join(assetRoot, "export_environment.py"), ipadInventoryHelperSHA, 1<<20)
	mathHelper := readPinnedPhoneFile(t, filepath.Join(assetRoot, "ipad_environment_math.py"), ipadEnvironmentMathSHA, 128<<10)
	scene := readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/api/internal/physicalruntime/assets/phone-study-environment/phone-study-environment.glb"), ipadEnvironmentSourceSHA, 2<<20)
	device := readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/api/internal/physicalruntime/assets/ipad/ipad-mini-screen-uv-oriented-180-v2.glb"), ipadSceneDeviceSHA, 8<<20)
	if int64(len(scene)) != ipadEnvironmentSourceBytes || int64(len(device)) != ipadSceneDeviceBytes {
		t.Fatal("pinned iPad export input size differs")
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"inputs/phone-study-environment.glb": scene, "inputs/ipad-mini-screen-uv-oriented-180-v2.glb": device} {
		if err := writeDurable(attempt, name, data); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range map[string][]byte{"ipad-scene-manifest.json": manifest, "export_ipad_environment.py": script, "ipad_environment_math.py": mathHelper, "phone_table_export_source.py": cupGenerator, "phone_environment_inventory_source.py": inventoryHelper} {
		if err := writeDurable(attempt, name, data); err != nil {
			t.Fatal(err)
		}
	}
	tableManifest := readPinnedPhoneFile(t, filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-table/export-manifest.json"), "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf", 128<<10)
	environmentManifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "full-scene-manifest.json"), phoneEnvironmentManifestSHA, 128<<10)
	if err := writeDurable(attempt, "phone_table_manifest.json", tableManifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "phone_environment_manifest.json", environmentManifest); err != nil {
		t.Fatal(err)
	}
	inputs := []map[string]any{
		{"role": "source_environment_only", "stage": "inputs/phone-study-environment.glb", "sha256": ipadEnvironmentSourceSHA, "bytes": ipadEnvironmentSourceBytes, "purpose": "preserve all non-cup scene meshes; original phone environment artifact remains unchanged"},
		{"role": "projection_device_only", "stage": "inputs/ipad-mini-screen-uv-oriented-180-v2.glb", "sha256": ipadSceneDeviceSHA, "bytes": ipadSceneDeviceBytes, "purpose": "project actual candidate geometry for cup placement; not emitted or changed"},
		{"role": "shared_cup_generator", "stage": "phone_table_export_source.py", "sha256": ipadCupGeneratorSHA},
		{"role": "shared_inventory_builder", "stage": "phone_environment_inventory_source.py", "sha256": ipadInventoryHelperSHA},
		{"role": "fixed_coordinate_math", "stage": "ipad_environment_math.py", "sha256": ipadEnvironmentMathSHA},
		{"role": "shared_table_manifest", "stage": "phone_table_manifest.json", "sha256": "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf"},
		{"role": "shared_environment_manifest", "stage": "phone_environment_manifest.json", "sha256": phoneEnvironmentManifestSHA},
	}
	writeAssetDerivationBindings(t, attempt, ipadEnvironmentManifestSHA, ipadEnvironmentScriptSHA, inputs)
	return ipadEnvironmentManifestSHA, ipadEnvironmentScriptSHA
}

func stagePhoneStarter(t *testing.T, attempt, _ string) (string, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate rendering package source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	assetRoot := filepath.Join(repo, "scenarios/device-studio/assets/sources/phone-starter")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "phone-starter-manifest.json"), phoneStarterManifestSHA, 128<<10)
	script := readPinnedPhoneFile(t, filepath.Join(assetRoot, "derive_phone_starter.py"), phoneStarterScriptSHA, 1<<20)
	sourcePath := "/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/authored/original-iphone16pro-iter03.glb"
	source := readPinnedPhoneFile(t, sourcePath, "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36", 8<<20)
	if len(source) != 2583052 {
		t.Fatal("pinned authored phone source size differs")
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "inputs/authored-phone-source.glb", source); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "phone-starter-manifest.json", manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "derive_phone_starter.py", script); err != nil {
		t.Fatal(err)
	}
	writeAssetDerivationBindings(t, attempt, phoneStarterManifestSHA, phoneStarterScriptSHA, []map[string]any{{"role": "research_source_only", "stage": "inputs/authored-phone-source.glb", "sha256": "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36", "bytes": 2583052, "private_display_pixels": "neutralized in derived output; source input is not copied to starter package"}})
	return phoneStarterManifestSHA, phoneStarterScriptSHA
}

func writeAssetDerivationBindings(t *testing.T, attempt, manifestSHA, scriptSHA string, inputs []map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(map[string]any{"manifest_sha256": manifestSHA, "script_sha256": scriptSHA, "inputs": inputs}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "source-bindings.json", append(b, '\n')); err != nil {
		t.Fatal(err)
	}
}

func verifyPhoneEnvironment(dir, manifestSHA, scriptSHA string) error {
	outputs := filepath.Join(dir, "outputs")
	if err := verifyAssetOutputNames(outputs, "phone-study-environment.glb", "environment-inventory.json"); err != nil {
		return err
	}
	receiptBytes, err := readPhoneTableRegular(filepath.Join(outputs, "environment-inventory.json"), assetDerivationReceiptMax)
	if err != nil {
		return fmt.Errorf("environment inventory: %w", err)
	}
	var receipt assetDerivationReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return err
	}
	glb, err := readPhoneTableRegular(filepath.Join(outputs, "phone-study-environment.glb"), assetDerivationGLBMax)
	if err != nil {
		return fmt.Errorf("environment GLB: %w", err)
	}
	if receipt.Profile != "device-studio-phone-study-environment-export-v1" || receipt.Status != "geometry_candidate_unreviewed" || receipt.ManifestSHA != manifestSHA || receipt.ScriptSHA != scriptSHA || receipt.SourceSHA != phoneTableGLBSHA || receipt.SourceBytes != phoneTableGLBBytes || receipt.GLBBytes != int64(len(glb)) || receipt.GLBBytes <= 0 || receipt.GLBBytes > assetDerivationGLBMax || receipt.GLBSHA != blenderDigest(glb) || receipt.Emission.Profile != "EMISSION_V1" {
		return errors.New("environment inventory identity differs from the fixed operation")
	}
	physical, err := InspectPhysicalGLB(glb)
	if err != nil {
		return fmt.Errorf("physical GLB inspection: %w", err)
	}
	if len(physical.Nodes) != len(phoneEnvironmentNodes) || len(receipt.Nodes) != len(phoneEnvironmentNodes) || physical.TriangleCount != receipt.TriangleCount {
		return errors.New("environment node or triangle census differs")
	}
	want := append([]string(nil), phoneEnvironmentNodes...)
	sort.Strings(want)
	got := make([]string, 0, len(physical.Nodes))
	nodeIndexes := make(map[string]uint32, len(physical.Nodes))
	for _, node := range physical.Nodes {
		got = append(got, node.Name)
		nodeIndexes[node.Name] = node.Index
	}
	sort.Strings(got)
	if !equalStringSlices(got, want) {
		return errors.New("environment GLB mesh names differ from fixed 14-node profile")
	}
	for name, index := range nodeIndexes {
		row, ok := receipt.Nodes[name]
		if !ok || row.NodeIndex != index || int(row.MeshIndex) >= len(physical.Meshes) || len(row.Primitives) != len(physical.Meshes[row.MeshIndex].Primitives) {
			return fmt.Errorf("environment inventory node/mesh binding differs for %s", name)
		}
		for i, primitive := range row.Primitives {
			actual := physical.Meshes[row.MeshIndex].Primitives[i]
			if primitive.Index != uint32(i) || primitive.Mode != 4 || primitive.MaterialIndex != actual.Material {
				return fmt.Errorf("environment primitive/material binding differs for %s", name)
			}
		}
	}
	return verifyEnvironmentEmission(glb)
}

func verifyIPadEnvironment(dir, manifestSHA, scriptSHA string) error {
	outputs := filepath.Join(dir, "outputs")
	if err := verifyAssetOutputNames(outputs, "ipad-study-environment.glb", "ipad-environment-inventory.json"); err != nil {
		return err
	}
	receiptBytes, err := readPhoneTableRegular(filepath.Join(outputs, "ipad-environment-inventory.json"), assetDerivationReceiptMax)
	if err != nil {
		return fmt.Errorf("iPad scene inventory: %w", err)
	}
	var receipt ipadEnvironmentReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return err
	}
	glb, err := readPhoneTableRegular(filepath.Join(outputs, "ipad-study-environment.glb"), assetDerivationGLBMax)
	if err != nil {
		return fmt.Errorf("iPad scene GLB: %w", err)
	}
	if receipt.Profile != "device-studio-ipad-study-environment-export-v2" || receipt.Status != "geometry_candidate_unreviewed" || receipt.ManifestSHA != manifestSHA || receipt.ScriptSHA != scriptSHA || receipt.SourceSHA != ipadEnvironmentSourceSHA || receipt.SourceBytes != ipadEnvironmentSourceBytes || receipt.DeviceSHA != ipadSceneDeviceSHA || receipt.DeviceBytes != ipadSceneDeviceBytes || receipt.GLBSHA != blenderDigest(glb) || receipt.GLBBytes != int64(len(glb)) || receipt.GLBBytes <= 0 || receipt.GLBBytes > assetDerivationGLBMax || receipt.Identity.BundleID != "device-studio-starter-ipad-mini-6-ipad-environment-v2" || receipt.Identity.ArtifactID != "ipad-study-environment-v2-glb" || receipt.Identity.SceneID != "ipad-study-environment-screen-uv-oriented-180-v2-cup128-fit-camera-v3" || receipt.Identity.CameraID != "ipad-study-camera-16x9-fit-screen-uv-oriented-180-v3" || receipt.Identity.Status != "candidate_only_until_native_export_and_visual_contact_review" {
		return errors.New("iPad scene inventory identity differs from fixed operation")
	}
	if receipt.Cup.GeneratorSHA != ipadCupGeneratorSHA || receipt.Cup.InventorySHA != ipadInventoryHelperSHA || receipt.Cup.TableManifestSHA != "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf" || receipt.Cup.EnvironmentManifestSHA != phoneEnvironmentManifestSHA || receipt.Cup.Segments != 128 || len(receipt.Cup.Target) != 2 || len(receipt.Cup.Placement) != 2 || len(receipt.Cup.Device) != 4 || len(receipt.Cup.Display) != 4 || len(receipt.Cup.Cup) != 4 || receipt.Cup.Required != .10 || receipt.Cup.Clearance < receipt.Cup.Required || receipt.Cup.Margin != .04 || !projectedBoundsInsideFrame(receipt.Cup.Device, receipt.Cup.Margin) || !projectedBoundsInsideFrame(receipt.Cup.Cup, receipt.Cup.Margin) {
		return errors.New("iPad cup tessellation or projection fit receipt violates the fixed profile")
	}
	physical, err := InspectPhysicalGLB(glb)
	if err != nil {
		return fmt.Errorf("iPad physical GLB inspection: %w", err)
	}
	if len(physical.Nodes) != len(phoneEnvironmentNodes) || physical.TriangleCount != receipt.Triangles {
		return errors.New("iPad scene node or triangle census differs")
	}
	want := append([]string(nil), phoneEnvironmentNodes...)
	sort.Strings(want)
	got := make([]string, 0, len(physical.Nodes))
	for _, node := range physical.Nodes {
		got = append(got, node.Name)
	}
	sort.Strings(got)
	if !equalStringSlices(got, want) {
		return errors.New("iPad scene does not preserve the fixed 14-node scene hierarchy")
	}
	return verifyEnvironmentEmission(glb)
}

func verifyEnvironmentEmission(glb []byte) error {
	chunks, err := splitGLBChunks(glb)
	if err != nil || len(chunks) != 2 {
		return errors.New("environment GLB chunks are invalid")
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return err
	}
	for _, material := range doc.Materials {
		if material.Name != "Soft luminous window" {
			continue
		}
		want := []float64{0.624, 0.656, 0.704}
		if len(material.EmissiveFactor) != 3 {
			return errors.New("window material lacks typed emissive RGB factor")
		}
		for i, v := range want {
			if math.Abs(material.EmissiveFactor[i]-v) > 1e-5 {
				return errors.New("window emission factor differs from the documented strength conversion")
			}
		}
		return nil
	}
	return errors.New("window emission material is absent")
}

func verifyPhoneStarter(dir, manifestSHA, scriptSHA string) error {
	outputs := filepath.Join(dir, "outputs")
	if err := verifyAssetOutputNames(outputs, "authored-phone-neutral.glb", "phone-starter-inventory.json"); err != nil {
		return err
	}
	receiptBytes, err := readPhoneTableRegular(filepath.Join(outputs, "phone-starter-inventory.json"), assetDerivationReceiptMax)
	if err != nil {
		return fmt.Errorf("phone inventory: %w", err)
	}
	var receipt phoneStarterReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return err
	}
	glb, err := readPhoneTableRegular(filepath.Join(outputs, "authored-phone-neutral.glb"), 8<<20)
	if err != nil {
		return fmt.Errorf("phone GLB: %w", err)
	}
	if receipt.Profile != "device-studio-authored-phone-starter-v1" || receipt.Status != "neutralized_authored_model_unreviewed" || receipt.ManifestSHA != manifestSHA || receipt.ScriptSHA != scriptSHA || receipt.SourceSHA != "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36" || receipt.SourceBytes != 2583052 || receipt.SourceImageSHA != "f4b6c7ed95981727bbb94abf1b3958a27c27faf5618e86e2b06b3d8047f3fb6d" || !receipt.DisplayRemoved || receipt.GLBSHA != blenderDigest(glb) || receipt.GLBBytes != int64(len(glb)) || receipt.GLBBytes <= 0 || receipt.GLBBytes > 8<<20 || receipt.Meshes != 57 || receipt.Triangles != 58196 || !equalInts(receipt.DisplaySize, []int{444, 965}) {
		return errors.New("phone derivative inventory identity differs from fixed source/profile")
	}
	physical, err := InspectPhysicalGLB(glb)
	if err != nil {
		return fmt.Errorf("phone physical GLB inspection: %w", err)
	}
	if err := verifyNeutralPhoneNodeCensus(physical.Nodes, physical.TriangleCount); err != nil || len(physical.Materials) <= receipt.Island.Material {
		return errors.New("phone derivative geometry census differs")
	}
	if int(receipt.Island.Node) >= len(physical.Nodes) || physical.Nodes[receipt.Island.Node].Name != "Dynamic Island black sensor cutout" || int(receipt.Island.Material) >= len(physical.Materials) {
		return errors.New("phone island semantic binding does not match emitted GLB")
	}
	material := physical.Materials[receipt.Island.Material]
	if material.Name != "Matte black sensor cutout" || math.Abs(material.Metallic) > 1e-6 || math.Abs(material.Roughness-0.82) > 1e-5 || math.Abs(material.SpecularFactor) > 1e-5 {
		return errors.New("phone island material factors/extensions differ")
	}
	if err := verifyIslandSpecularExtension(glb, receipt.Island.Material); err != nil {
		return err
	}
	sourceGLB, err := readPhoneTableRegular(filepath.Join(dir, "inputs/authored-phone-source.glb"), 8<<20)
	if err != nil {
		return fmt.Errorf("staged phone source: %w", err)
	}
	return verifyNeutralEmbeddedDisplay(glb, receipt, sourceGLB)
}

func verifyNeutralPhoneNodeCensus(nodes []PhysicalGLBNode, triangles uint64) error {
	if len(nodes) != 58 || triangles != 58196 {
		return errors.New("phone derivative must retain 58 nodes and 58,196 triangles")
	}
	meshNodes, roots := 0, 0
	for _, node := range nodes {
		if node.Mesh != nil {
			meshNodes++
			continue
		}
		roots++
		if node.Name != "IPHONE_16_PRO_ORIGINAL_MASTER" || node.Index != 57 || len(node.Children) != 57 {
			return errors.New("phone derivative empty scene root or child hierarchy differs")
		}
		for i, child := range node.Children {
			if child != uint32(i) {
				return errors.New("phone derivative root children do not preserve mesh-node ordering")
			}
		}
	}
	if meshNodes != 57 || roots != 1 {
		return errors.New("phone derivative must contain 57 mesh nodes and one empty scene root")
	}
	return nil
}

func verifyIslandSpecularExtension(glb []byte, materialIndex int) error {
	chunks, err := splitGLBChunks(glb)
	if err != nil || len(chunks) != 2 {
		return errors.New("phone material GLB chunks are invalid")
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return err
	}
	if materialIndex < 0 || materialIndex >= len(doc.Materials) {
		return errors.New("derived island material index is out of range")
	}
	declared := false
	for _, extension := range doc.ExtensionsUsed {
		if extension == "KHR_materials_specular" {
			declared = true
		}
	}
	if !declared {
		return errors.New("GLB does not declare KHR_materials_specular")
	}
	raw := doc.Materials[materialIndex].Extensions["KHR_materials_specular"]
	var extension struct {
		SpecularFactor *float64 `json:"specularFactor"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &extension) != nil || extension.SpecularFactor == nil || math.Abs(*extension.SpecularFactor) > 1e-6 {
		return errors.New("derived island has no exact zero specularFactor extension")
	}
	return nil
}

func verifyNeutralEmbeddedDisplay(glb []byte, receipt phoneStarterReceipt, sourceGLB []byte) error {
	chunks, err := splitGLBChunks(glb)
	if err != nil || len(chunks) != 2 {
		return errors.New("phone GLB chunks are invalid")
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return err
	}
	if len(doc.Images) != 1 || doc.Images[0].Name != "portable-display" || doc.Images[0].BufferView == nil || doc.Images[0].URI != "" {
		return errors.New("phone GLB must contain exactly one embedded neutral display image")
	}
	viewIndex := *doc.Images[0].BufferView
	if int(viewIndex) >= len(doc.BufferViews) {
		return errors.New("display buffer view is out of range")
	}
	view := doc.BufferViews[viewIndex]
	if uint64(view.ByteOffset)+uint64(view.ByteLength) > uint64(len(chunks[1])) || view.ByteLength == 0 {
		return errors.New("display buffer view exceeds GLB BIN chunk")
	}
	encoded := chunks[1][view.ByteOffset : view.ByteOffset+view.ByteLength]
	sourceImage, err := embeddedGLBImage(sourceGLB, "portable-display")
	if err != nil {
		return fmt.Errorf("source private display: %w", err)
	}
	if blenderDigest(sourceImage) != receipt.SourceImageSHA || bytes.Contains(glb, sourceImage) || blenderDigest(encoded) == receipt.SourceImageSHA || blenderDigest(encoded) != receipt.DisplaySHA {
		return errors.New("private source image bytes remain or embedded display digest differs")
	}
	image, err := png.Decode(bytes.NewReader(encoded))
	if err != nil || image.Bounds().Dx() != 444 || image.Bounds().Dy() != 965 {
		return errors.New("neutral display PNG cannot be fully decoded at expected dimensions")
	}
	var first color.NRGBA
	for y := 0; y < image.Bounds().Dy(); y++ {
		for x := 0; x < image.Bounds().Dx(); x++ {
			pixel := color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)
			if x == 0 && y == 0 {
				first = pixel
			} else if absInt(int(pixel.R)-int(first.R)) > 1 || absInt(int(pixel.G)-int(first.G)) > 1 || absInt(int(pixel.B)-int(first.B)) > 1 || pixel.A != first.A {
				return errors.New("derived display contains non-uniform/private image variation")
			}
		}
	}
	if first.A != 255 || absInt(int(first.R)-int(first.G)) > 1 || absInt(int(first.G)-int(first.B)) > 1 {
		return errors.New("derived display is not an opaque neutral gray")
	}
	// Verify achromatic, opaque pixels without assigning an sRGB meaning to the
	// packed byte value. The Blender-side adapter separately checks its working
	// image values; the starter placeholder only requires a uniform neutral fill.
	return nil
}

func embeddedGLBImage(glb []byte, name string) ([]byte, error) {
	chunks, err := splitGLBChunks(glb)
	if err != nil || len(chunks) != 2 {
		return nil, errors.New("source GLB chunks are invalid")
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return nil, err
	}
	var selected *glbImage
	for i := range doc.Images {
		if doc.Images[i].Name == name {
			if selected != nil {
				return nil, errors.New("source image name is ambiguous")
			}
			selected = &doc.Images[i]
		}
	}
	if selected == nil || selected.BufferView == nil || int(*selected.BufferView) >= len(doc.BufferViews) {
		return nil, errors.New("source image buffer view is absent")
	}
	view := doc.BufferViews[*selected.BufferView]
	if uint64(view.ByteOffset)+uint64(view.ByteLength) > uint64(len(chunks[1])) || view.ByteLength == 0 {
		return nil, errors.New("source image buffer view exceeds binary chunk")
	}
	return chunks[1][view.ByteOffset : view.ByteOffset+view.ByteLength], nil
}

func verifyAssetOutputNames(dir, glbName, receiptName string) error {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("output path is not a regular directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		return errors.New("output directory must contain exactly the fixed GLB and receipt")
	}
	want := map[string]bool{glbName: true, receiptName: true}
	for _, entry := range entries {
		item, e := entry.Info()
		if !want[entry.Name()] || e != nil || !item.Mode().IsRegular() || item.Mode()&os.ModeSymlink != 0 {
			return errors.New("output directory contains an unexpected or unsafe file")
		}
	}
	return nil
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
