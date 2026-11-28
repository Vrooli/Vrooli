package rendering

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	laptopStarterManifestSHA  = "642b16a4bcb5ab6cd2d074a0e60a32a1021c4a686a44efeb735eae11cb8a1266"
	laptopStarterScriptSHA    = "c4a54f55678e2b439ab0a9c7aed4c939703e4123b2149a8a2413a6afb9f36fe8"
	laptopStarterSourceSHA    = "4b7a6736e12e105b00556ca6397f3dffe559a1c518954209df08828f186d8a78"
	laptopStarterInventorySHA = "cfe2d0a9e9b03809a07cfdc4d4d823080216c74a1fc025cf73370109be343af4"
	laptopStarterSourceBytes  = 6712244
	laptopStarterGLBMax       = 32 << 20
	laptopSourceOpenTolerance = 1e-7
	laptopStarterPNGMax       = 8 << 20
	laptopStarterPNGTotalMax  = 48 << 20
)

type laptopStarterOutput struct {
	SHA                string    `json:"sha256"`
	Bytes              int64     `json:"size_bytes"`
	Width              int       `json:"width,omitempty"`
	Height             int       `json:"height,omitempty"`
	Materials          string    `json:"materials,omitempty"`
	Diagnostic         string    `json:"diagnostic,omitempty"`
	CameraTargetRule   string    `json:"camera_target_rule,omitempty"`
	CameraTargetWorld  []float64 `json:"camera_target_world_m,omitempty"`
	CameraTargetRegion struct {
		YMin float64 `json:"y_min"`
		YMax float64 `json:"y_max"`
		ZMin float64 `json:"z_min"`
		ZMax float64 `json:"z_max"`
	} `json:"camera_target_region_world_m,omitempty"`
	CameraOrthoWidth float64 `json:"camera_ortho_width_m,omitempty"`
}

type laptopComponentDiagnostic struct {
	SourceObject       string    `json:"source_object"`
	SourceComponent    int       `json:"source_component"`
	SourceNodeIndex    int       `json:"source_node_index"`
	CandidateRole      string    `json:"candidate_role"`
	DerivedMeshName    string    `json:"derived_mesh_name"`
	TriangleCount      int       `json:"triangle_count"`
	Label              string    `json:"label"`
	Color              []float64 `json:"color_rgba"`
	Visibility         string    `json:"visibility"`
	TransformUnchanged bool      `json:"object_transform_unchanged"`
}

type laptopStarterReceipt struct {
	Profile        string               `json:"profile_id"`
	Status         string               `json:"status"`
	SourceSHA      string               `json:"source_sha256"`
	SourceBytes    int64                `json:"source_size_bytes"`
	InventorySHA   string               `json:"component_inventory_sha256"`
	ManifestSHA    string               `json:"manifest_sha256"`
	ScriptSHA      string               `json:"script_sha256"`
	Blender        string               `json:"blender_version"`
	Triangles      int                  `json:"total_source_triangles"`
	MeshNodes      int                  `json:"mesh_node_count"`
	RoleTriangles  map[string]int       `json:"triangles_by_candidate_role"`
	RoleAssignment laptopRoleAssignment `json:"role_assignment"`
	Hierarchy      map[string]string    `json:"hierarchy"`
	NeutralScreen  struct {
		SourceSHA string `json:"source_sha256"`
		ImageSHA  string `json:"derived_png_sha256"`
		Claim     string `json:"claim"`
	} `json:"neutral_screen"`
	WorldBackground struct {
		LinearRGBA []float64 `json:"linear_rgba"`
		Strength   float64   `json:"strength"`
	} `json:"world_background"`
	ClosedPoseFit        laptopStarterClosedFit         `json:"closed_pose_fit"`
	TrimAlignment        laptopStarterTrimAlignment     `json:"hinge_trim_alignment"`
	LocalWitnesses       map[string]laptopLocalWitness  `json:"closed_local_surface_witnesses"`
	Poses                map[string]json.RawMessage     `json:"poses"`
	Outputs              map[string]laptopStarterOutput `json:"outputs"`
	ComponentDiagnostics []laptopComponentDiagnostic    `json:"component_diagnostic_bindings"`
	Limits               string                         `json:"limits"`
}

type laptopRoleOverride struct {
	SourceNodeIndex int    `json:"source_node_index"`
	SourceObject    string `json:"source_object"`
	SourceComponent int    `json:"source_component"`
	SourceRole      string `json:"source_candidate_role"`
	CandidateRole   string `json:"candidate_role"`
	TriangleCount   int    `json:"triangle_count"`
}

type laptopRoleAssignment struct {
	Source               string                     `json:"source"`
	BaseTriangles        int                        `json:"base_triangles"`
	LidTriangles         int                        `json:"lid_triangles"`
	HingeReviewTriangles int                        `json:"hinge_review_triangles"`
	Overrides            []laptopRoleOverride       `json:"lid_component_overrides"`
	Evidence             laptopRoleOverrideEvidence `json:"override_evidence"`
	SourceOpenInvariant  struct {
		Method              string                      `json:"method"`
		PoseRotationDegrees float64                     `json:"pose_rotation_degrees"`
		ToleranceM          float64                     `json:"tolerance_m"`
		AllWithinTolerance  bool                        `json:"all_components_within_tolerance"`
		Components          []laptopSourceOpenComponent `json:"components"`
	} `json:"source_open_geometry_invariance"`
}

type laptopSourceOpenComponent struct {
	SourceObject    string  `json:"source_object"`
	SourceNodeIndex *int    `json:"source_node_index"`
	SourceComponent int     `json:"source_component"`
	TriangleCount   int     `json:"triangle_count"`
	MaxResidualM    float64 `json:"max_vertex_residual_m"`
	RMSResidualM    float64 `json:"rms_vertex_residual_m"`
	ToleranceM      float64 `json:"tolerance_m"`
	WithinTolerance bool    `json:"world_triangle_geometry_within_tolerance"`
	QuantizedEqual  bool    `json:"quantized_signature_equal"`
}

type laptopRoleOverrideEvidence struct {
	Method          string     `json:"method"`
	ScreenNormal    [3]float64 `json:"screen_plane_normal_world"`
	ClosedAngle     float64    `json:"fitted_closed_angle_degrees"`
	FittedAxis      [3]float64 `json:"fitted_axis_glb_m"`
	DeckNormal      [3]float64 `json:"deck_normal_world"`
	AllClosedGapsMM [2]float64 `json:"closed_vertex_gap_mm_range_over_all_overrides"`
	Components      []struct {
		SourceObject string     `json:"source_object"`
		Component    int        `json:"source_component"`
		Triangles    int        `json:"triangle_count"`
		NormalDot    float64    `json:"normal_dot_screen"`
		SourceGapsMM [3]float64 `json:"source_vertex_signed_distance_mm"`
		ClosedDot    float64    `json:"closed_normal_dot_opposite_deck"`
	} `json:"components"`
}

type laptopNegativeSample struct {
	SourceObject        string  `json:"source_object"`
	SourceComponent     int     `json:"source_component"`
	BaseSourceObject    string  `json:"base_source_object"`
	BaseSourceComponent int     `json:"base_source_component"`
	SignedGap           float64 `json:"signed_gap_to_base_normal_m"`
}

type laptopLocalWitness struct {
	Method            string                 `json:"method"`
	Samples           int                    `json:"sample_count"`
	Within20mm        int                    `json:"within_20mm_count"`
	Min               float64                `json:"min_distance_m"`
	Median            float64                `json:"median_distance_m"`
	P95               float64                `json:"p95_distance_m"`
	Max               float64                `json:"max_distance_m"`
	SignedMin         float64                `json:"signed_gap_min_m"`
	SignedMedian      float64                `json:"signed_gap_median_m"`
	SignedP95         float64                `json:"signed_gap_p95_m"`
	SignedNegative    int                    `json:"signed_gap_negative_sample_count"`
	NegativeTruncated bool                   `json:"negative_sample_records_truncated"`
	NegativeSamples   []laptopNegativeSample `json:"negative_sample_records"`
	ScreenIncluded    bool                   `json:"screen_mesh_included"`
	Qualification     string                 `json:"qualification"`
	Witnesses         []json.RawMessage      `json:"closest_witnesses"`
}

type laptopStarterTrimAlignment struct {
	Status           string    `json:"status"`
	SourceObject     string    `json:"source_object"`
	SourceComponent  int       `json:"source_component"`
	TriangleCount    int       `json:"triangle_count"`
	Parent           string    `json:"parent"`
	RotatesWithLid   bool      `json:"rotates_with_lid"`
	Translation      []float64 `json:"translation_world_m"`
	MatchesAxisDelta bool      `json:"translation_matches_fitted_axis_delta"`
	Claim            string    `json:"claim"`
}

func TestLaptopComponentDiagnosticSourcePins(t *testing.T) {
	scenarioRoot, _ := laptopTestRoots(t)
	assets := filepath.Join(scenarioRoot, "assets/sources/laptop-starter")
	manifest := readPinnedPhoneFile(t, filepath.Join(assets, "laptop-starter-manifest.json"), laptopStarterManifestSHA, 128<<10)
	readPinnedPhoneFile(t, filepath.Join(assets, "derive_laptop_starter.py"), laptopStarterScriptSHA, 1<<20)
	var cfg struct {
		Profile        string               `json:"profile_id"`
		RoleAssignment laptopRoleAssignment `json:"role_assignment"`
		Output         struct {
			PNGs     []string                    `json:"diagnostic_pngs"`
			Bindings []laptopComponentDiagnostic `json:"component_diagnostic_bindings"`
		} `json:"output"`
		Render struct {
			Anchor struct {
				Rule                string  `json:"rule"`
				BaseZPercentile     float64 `json:"base_z_percentile"`
				BaseZOffset         float64 `json:"base_z_offset_m"`
				UsesKeyedComponents bool    `json:"target_uses_keyed_components"`
				SidewallContext     string  `json:"full_sidewall_context_component"`
			} `json:"component_macro_anchor"`
		} `json:"render"`
	}
	if err := json.Unmarshal(manifest, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != "device-studio-laptop-starter-derive-v9" || len(cfg.Output.PNGs) != 10 || len(cfg.Output.Bindings) != 6 ||
		cfg.Render.Anchor.Rule != "closed-hinge-material-macro-v1" || cfg.Render.Anchor.BaseZPercentile != 0.9 ||
		cfg.Render.Anchor.BaseZOffset != 0.006 || cfg.Render.Anchor.UsesKeyedComponents || cfg.Render.Anchor.SidewallContext != "Object_22/0" ||
		cfg.RoleAssignment.BaseTriangles != 99866 || cfg.RoleAssignment.LidTriangles != 8470 || cfg.RoleAssignment.HingeReviewTriangles != 4166 {
		t.Fatal("fixed laptop profile does not declare the component diagnostic outputs and bindings")
	}
	if err := verifyLaptopRoleOverrides(cfg.RoleAssignment.Overrides); err != nil {
		t.Fatalf("manifest role overrides differ from the exact plane-supported component set: %v", err)
	}
	if err := verifyLaptopRoleOverrideEvidence(cfg.RoleAssignment.Evidence); err != nil {
		t.Fatalf("manifest role override evidence is incomplete or changed: %v", err)
	}
	want := []struct {
		object                     string
		node, component, triangles int
		role                       string
		mesh                       string
	}{
		{"Object_10", 0, 310, 184, "lid", "LaptopLidGeometry"},
		{"Object_10", 0, 313, 148, "lid", "LaptopLidGeometry"},
		{"Object_20", 2, 92, 456, "lid", "LaptopLidGeometry"},
		{"Object_42", 8, 12, 584, "hinge-review", "LaptopHingeTrimCandidate"},
		{"Object_22", 3, 0, 3582, "hinge-review", "LaptopHingeReviewGeometry"},
		{"Object_10", 0, 316, 298, "lid", "LaptopLidGeometry"},
	}
	for index, row := range cfg.Output.Bindings {
		if row.SourceObject != want[index].object || row.SourceNodeIndex != want[index].node || row.SourceComponent != want[index].component || row.TriangleCount != want[index].triangles || row.CandidateRole != want[index].role || row.DerivedMeshName != want[index].mesh {
			t.Fatalf("component binding %d differs from the pinned source inventory", index)
		}
	}
}

func TestLaptopRoleOverrideReceiptRequiresExactComponentsAndSourceOpenInvariance(t *testing.T) {
	assignment := laptopRoleAssignment{
		Source:        "component-candidates labels plus exact normal/plane-supported lid overrides",
		BaseTriangles: 99866, LidTriangles: 8470, HingeReviewTriangles: 4166,
		Overrides: []laptopRoleOverride{
			{SourceNodeIndex: 0, SourceObject: "Object_10", SourceComponent: 310, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 184},
			{SourceNodeIndex: 0, SourceObject: "Object_10", SourceComponent: 313, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 148},
			{SourceNodeIndex: 2, SourceObject: "Object_20", SourceComponent: 92, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 456},
		},
	}
	assignment.Evidence = expectedLaptopRoleOverrideEvidence()
	assignment.SourceOpenInvariant.Method = "source triangle index matched before/after parenting; raw world-space vertex residual measured per vertex; rounded 1e-7 m signatures retained as diagnostics only"
	assignment.SourceOpenInvariant.PoseRotationDegrees = 0
	assignment.SourceOpenInvariant.ToleranceM = laptopSourceOpenTolerance
	assignment.SourceOpenInvariant.AllWithinTolerance = true
	for _, row := range assignment.Overrides {
		nodeIndex := row.SourceNodeIndex
		assignment.SourceOpenInvariant.Components = append(assignment.SourceOpenInvariant.Components, laptopSourceOpenComponent{
			SourceObject: row.SourceObject, SourceNodeIndex: &nodeIndex,
			SourceComponent: row.SourceComponent, TriangleCount: row.TriangleCount,
			MaxResidualM: 2.8e-8, RMSResidualM: 2.6e-8,
			ToleranceM: laptopSourceOpenTolerance, WithinTolerance: true, QuantizedEqual: false,
		})
	}
	if err := verifyLaptopRoleAssignment(assignment); err != nil {
		t.Fatalf("valid exact lid-role override receipt rejected: %v", err)
	}
	legacyWithoutNestedNode := assignment
	legacyWithoutNestedNode.SourceOpenInvariant.Components = append([]laptopSourceOpenComponent(nil), assignment.SourceOpenInvariant.Components...)
	legacyWithoutNestedNode.SourceOpenInvariant.Components[2].SourceNodeIndex = nil
	if err := verifyLaptopRoleAssignment(legacyWithoutNestedNode); err != nil {
		t.Fatalf("component identity bound by the exact top-level override list should allow omitted nested node index: %v", err)
	}
	atTolerance := assignment
	atTolerance.SourceOpenInvariant.Components = append([]laptopSourceOpenComponent(nil), assignment.SourceOpenInvariant.Components...)
	atTolerance.SourceOpenInvariant.Components[0].MaxResidualM = laptopSourceOpenTolerance
	atTolerance.SourceOpenInvariant.Components[0].RMSResidualM = laptopSourceOpenTolerance / 2
	if err := verifyLaptopRoleAssignment(atTolerance); err != nil {
		t.Fatalf("residual exactly at the 100 nm threshold should pass: %v", err)
	}
	wrongComponent := assignment
	wrongComponent.Overrides = append([]laptopRoleOverride(nil), assignment.Overrides...)
	wrongComponent.Overrides[2].SourceComponent = 93
	if err := verifyLaptopRoleAssignment(wrongComponent); err == nil {
		t.Fatal("wrong source component was accepted as a lid override")
	}
	wrongCount := assignment
	wrongCount.Overrides = append([]laptopRoleOverride(nil), assignment.Overrides...)
	wrongCount.Overrides[0].TriangleCount++
	if err := verifyLaptopRoleAssignment(wrongCount); err == nil {
		t.Fatal("changed component triangle count was accepted")
	}
	movedAtOpen := assignment
	movedAtOpen.SourceOpenInvariant.AllWithinTolerance = false
	if err := verifyLaptopRoleAssignment(movedAtOpen); err == nil {
		t.Fatal("source-open geometry change was accepted")
	}
	tooFarAtOpen := assignment
	tooFarAtOpen.SourceOpenInvariant.Components = append([]laptopSourceOpenComponent(nil), assignment.SourceOpenInvariant.Components...)
	tooFarAtOpen.SourceOpenInvariant.Components[1].MaxResidualM = laptopSourceOpenTolerance + 1e-12
	tooFarAtOpen.SourceOpenInvariant.Components[1].WithinTolerance = false
	tooFarAtOpen.SourceOpenInvariant.AllWithinTolerance = false
	if err := verifyLaptopRoleAssignment(tooFarAtOpen); err == nil {
		t.Fatal("source-open residual above the pinned 100 nm tolerance was accepted")
	}
	missingMeasurement := assignment
	missingMeasurement.SourceOpenInvariant.Components = assignment.SourceOpenInvariant.Components[:2]
	if err := verifyLaptopRoleAssignment(missingMeasurement); err == nil {
		t.Fatal("incomplete three-component source-open measurements were accepted")
	}
	wrongNestedNode := assignment
	wrongNestedNode.SourceOpenInvariant.Components = append([]laptopSourceOpenComponent(nil), assignment.SourceOpenInvariant.Components...)
	wrongNestedIndex := 99
	wrongNestedNode.SourceOpenInvariant.Components[2].SourceNodeIndex = &wrongNestedIndex
	if err := verifyLaptopRoleAssignment(wrongNestedNode); err == nil {
		t.Fatal("contradictory nested source node index was accepted")
	}
}

func TestLaptopOpenGeometryGuardPairsOriginalTriangleIndicesBeforeJoin(t *testing.T) {
	scenarioRoot, _ := laptopTestRoots(t)
	script, err := os.ReadFile(filepath.Join(scenarioRoot, "assets/sources/laptop-starter/derive_laptop_starter.py"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(script)
	layer := strings.Index(source, `bm.faces.layers.int.new("source_triangle_index")`)
	assign := strings.Index(source, `face[source_triangle_layer] = face.index`)
	delete := strings.Index(source, `bmesh.ops.delete(bm, geom=[face for face in bm.faces if roles[face.index] != role]`)
	persist := strings.Index(source, `persist_diagnostic(ROOT / "source-open-invariance-diagnostic.json"`)
	assert := strings.Index(source, `require(all_within, "source-open world-space geometry exceeds the pinned 1e-7 m tolerance")`)
	if layer < 0 || assign < layer || delete < assign ||
		!strings.Contains(source, `set(source_by_triangle) == set(derived_by_triangle)`) ||
		!strings.Contains(source, `"quantized_signature_equal": original == derived`) ||
		!strings.Contains(source, `"max_vertex_residual_m": max_residual`) ||
		!strings.Contains(source, `all_within = len(checks) == len(EXPECTED_LID_COMPONENT_OVERRIDES)`) ||
		!strings.Contains(source, `SOURCE_OPEN_INVARIANCE_DIAGNOSTIC`) ||
		!strings.Contains(source, `"source_triangle_index"`) || persist < 0 || assert < persist ||
		!strings.Contains(source, `"failure-diagnostic.json"`) || !strings.Contains(source, `traceback.format_exc(limit=4)[:3072]`) {
		t.Fatal("source triangle identity must survive filtering/join for the strict source-open geometry comparison")
	}
}

func TestLaptopMacroAnchorCallsPassManifestConfig(t *testing.T) {
	scenarioRoot, _ := laptopTestRoots(t)
	script, err := os.ReadFile(filepath.Join(scenarioRoot, "assets/sources/laptop-starter/derive_laptop_starter.py"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(script)
	if strings.Count(source, "closed_hinge_macro_target(") != 3 ||
		strings.Count(source, "closed_hinge_macro_target(base_points, fitted_axis_y, cfg)") != 3 {
		t.Fatal("rear macro helper definition and both call sites must pass the manifest config")
	}
}

type laptopStarterClosedFit struct {
	Method                string    `json:"method"`
	SourceAxis            []float64 `json:"source_axis_blender_m"`
	FittedAxis            []float64 `json:"fitted_axis_blender_m"`
	FittedAxisGLB         []float64 `json:"fitted_axis_glb_m"`
	PlaneAngle            float64   `json:"selected_closed_angle_degrees"`
	OppositePlaneAngle    float64   `json:"screen_to_opposite_base_plane_alignment_angle_degrees"`
	GapTarget             float64   `json:"target_screen_deck_gap_m"`
	GapActual             float64   `json:"screen_deck_gap_median_m"`
	GapMin                float64   `json:"screen_deck_gap_min_m"`
	GapP95                float64   `json:"screen_deck_gap_p95_m"`
	LidGeometryMin        float64   `json:"lid_geometry_lowest_point_relative_to_deck_m"`
	LidGeometryDiagnostic bool      `json:"lid_geometry_minimum_is_diagnostic_only"`
	DepthCenterError      float64   `json:"realized_depth_center_error_at_diagnostic_pose_m"`
	DiagnosticAngle       float64   `json:"diagnostic_angle_degrees"`
	Uncertainty           string    `json:"uncertainty"`
}

type laptopStarterPose struct {
	Label        string  `json:"pose_label"`
	Angle        float64 `json:"closure_from_source_open_degrees"`
	OverlapPairs int     `json:"candidate_triangle_surface_overlap_pairs"`
	Status       string  `json:"sampled_envelope_status"`
	Views        map[string]struct {
		Margin float64 `json:"projected_margin_fraction"`
		Output string  `json:"output"`
	} `json:"views"`
}

func TestLaptopStarterSourcePinsMatchTrackedInputs(t *testing.T) {
	scenarioRoot, _ := laptopTestRoots(t)
	root := filepath.Join(scenarioRoot, "assets/sources/laptop-starter")
	for _, item := range []struct{ name, path, want string }{
		{"manifest", filepath.Join(root, "laptop-starter-manifest.json"), laptopStarterManifestSHA},
		{"script", filepath.Join(root, "derive_laptop_starter.py"), laptopStarterScriptSHA},
	} {
		data, err := os.ReadFile(item.path)
		if err != nil {
			t.Fatalf("read pinned %s: %v", item.name, err)
		}
		if got := blenderDigest(data); got != item.want {
			t.Errorf("pinned %s digest mismatch: got %s want %s", item.name, got, item.want)
		}
	}
}

func TestLaptopLocalWitnessReceiptSeparatesScreenFromLid(t *testing.T) {
	base := laptopLocalWitness{Method: "nearest points from actual selected world-space triangle centroids", Samples: 5,
		Within20mm: 5, Min: 0.0003, Median: 0.001, P95: 0.002, Max: 0.003,
		SignedMin: -0.002, SignedMedian: 0.0004, SignedP95: 0.002, SignedNegative: 1,
		NegativeSamples: []laptopNegativeSample{{SourceObject: "Object_42", SourceComponent: 12,
			BaseSourceObject: "Object_22", BaseSourceComponent: 0, SignedGap: -0.002}},
		Qualification: "discrete centroid proximity only", Witnesses: []json.RawMessage{json.RawMessage(`{"distance_m":0.0003}`)}}
	items := map[string]laptopLocalWitness{
		"base_to_lid_underside": base,
		"base_to_screen_surface": {Method: base.Method, Samples: base.Samples, Within20mm: base.Within20mm,
			Min: base.Min, Median: base.Median, P95: base.P95, Max: base.Max, ScreenIncluded: true,
			SignedMin: 0.0001, SignedMedian: 0.0002, SignedP95: 0.002,
			Qualification: base.Qualification, Witnesses: base.Witnesses},
	}
	if err := verifyLaptopLocalWitnesses(items); err != nil {
		t.Fatalf("valid separated local witnesses rejected: %v", err)
	}
	delete(items, "base_to_screen_surface")
	if err := verifyLaptopLocalWitnesses(items); err == nil {
		t.Fatal("receipt without a separate screen witness was accepted")
	}
}

func TestLaptopClosedFitUsesScreenDeckGapNotWholeLidMinimum(t *testing.T) {
	fit := laptopStarterClosedFit{
		Method: "screen plane fit", SourceAxis: []float64{0, 0, 0}, FittedAxis: []float64{0, 0, 0},
		FittedAxisGLB: []float64{0, 0, 0}, PlaneAngle: 110, OppositePlaneAngle: 110,
		GapTarget: 0.00045, GapMin: -0.001, GapActual: 0.00045, GapP95: 0.001,
		LidGeometryMin: -0.006, LidGeometryDiagnostic: true, DiagnosticAngle: 110,
		Uncertainty: "source geometry is approximate",
	}
	if err := verifyLaptopStarterClosedFit(fit); err != nil {
		t.Fatalf("screen/deck fit with an unresolved lower hinge trim diagnostic rejected: %v", err)
	}
	fit.LidGeometryDiagnostic = false
	if err := verifyLaptopStarterClosedFit(fit); err == nil {
		t.Fatal("fit without explicit whole-lid diagnostic labeling was accepted")
	}
}

func TestLaptopTrimAlignmentReceiptIsExplicitAndFixed(t *testing.T) {
	trim := laptopStarterTrimAlignment{Status: "authored_fixed_base_trim_alignment_hypothesis", SourceObject: "Object_42",
		SourceComponent: 12, TriangleCount: 584, Parent: "LaptopHingeReview", Translation: []float64{0, -0.0066, -0.0044},
		MatchesAxisDelta: true, Claim: "visual alignment experiment only"}
	if err := verifyLaptopTrimAlignment(trim); err != nil {
		t.Fatalf("valid fixed trim hypothesis rejected: %v", err)
	}
	trim.RotatesWithLid = true
	if err := verifyLaptopTrimAlignment(trim); err == nil {
		t.Fatal("trim hypothesis that rotates with lid was accepted")
	}
}

func verifyLaptopStarterClosedFit(fit laptopStarterClosedFit) error {
	derived := []float64{fit.PlaneAngle, fit.OppositePlaneAngle, fit.GapTarget, fit.GapActual, fit.GapMin, fit.GapP95, fit.LidGeometryMin, fit.DiagnosticAngle, fit.DepthCenterError}
	for _, value := range derived {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("closed pose fit contains a non-finite measurement")
		}
	}
	if fit.Method == "" || len(fit.SourceAxis) != 3 || len(fit.FittedAxis) != 3 || len(fit.FittedAxisGLB) != 3 || fit.PlaneAngle < 70 || fit.PlaneAngle > 130 || absFloat(fit.PlaneAngle-fit.OppositePlaneAngle) > 1e-6 || fit.GapTarget != 0.00045 || fit.GapActual < 0.0003 || fit.GapActual > 0.0006 || fit.GapMin > fit.GapActual || fit.GapP95 < fit.GapActual || !fit.LidGeometryDiagnostic || absFloat(fit.DiagnosticAngle-fit.PlaneAngle) > 1e-6 || absFloat(fit.DepthCenterError) > 0.0005 || fit.Uncertainty == "" {
		return errors.New("closed pose fit lacks bounded plane, gap, footprint, or uncertainty evidence")
	}
	return nil
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func verifyLaptopRoleAssignment(assignment laptopRoleAssignment) error {
	want := expectedLaptopRoleOverrides()
	if assignment.Source == "" || assignment.BaseTriangles != 99866 || assignment.LidTriangles != 8470 || assignment.HingeReviewTriangles != 4166 || len(assignment.Overrides) != len(want) {
		return errors.New("role assignment census or exact override list is incomplete")
	}
	if err := verifyLaptopRoleOverrides(assignment.Overrides); err != nil {
		return err
	}
	if err := verifyLaptopRoleOverrideEvidence(assignment.Evidence); err != nil {
		return err
	}
	invariant := assignment.SourceOpenInvariant
	if !invariant.AllWithinTolerance || invariant.PoseRotationDegrees != 0 ||
		!strings.Contains(invariant.Method, "source triangle index matched") ||
		invariant.ToleranceM != laptopSourceOpenTolerance || len(invariant.Components) != len(want) {
		return errors.New("source-open role reassignment lacks unchanged world-triangle evidence")
	}
	for i, component := range invariant.Components {
		if !component.WithinTolerance || component.ToleranceM != laptopSourceOpenTolerance ||
			math.IsNaN(component.MaxResidualM) || math.IsInf(component.MaxResidualM, 0) || component.MaxResidualM < 0 ||
			math.IsNaN(component.RMSResidualM) || math.IsInf(component.RMSResidualM, 0) || component.RMSResidualM < 0 ||
			component.MaxResidualM > laptopSourceOpenTolerance || component.RMSResidualM > component.MaxResidualM ||
			component.SourceObject != want[i].SourceObject ||
			component.SourceComponent != want[i].SourceComponent ||
			component.TriangleCount != want[i].TriangleCount {
			return errors.New("source-open geometry evidence does not match the exact override set")
		}
		if component.SourceNodeIndex != nil && *component.SourceNodeIndex != want[i].SourceNodeIndex {
			return errors.New("source-open component declares a contradictory source node index")
		}
	}
	return nil
}

func expectedLaptopRoleOverrideEvidence() laptopRoleOverrideEvidence {
	evidence := laptopRoleOverrideEvidence{
		Method:       "Source GLB transformed component triangles into world space; area-weighted normals compared with Object_123 screen-plane normal; vertex signed distances measured to that plane; closed-pose normals and vertex signed gaps compared with fitted deck plane.",
		ScreenNormal: [3]float64{-0.0000076, 0.341690, 0.939813}, ClosedAngle: 109.9807869,
		FittedAxis: [3]float64{0, -0.0007040767, -0.1219446287}, DeckNormal: [3]float64{0, 1, 0},
		AllClosedGapsMM: [2]float64{0.28, 0.64},
	}
	for _, row := range expectedLaptopRoleOverrides() {
		component := struct {
			SourceObject string     `json:"source_object"`
			Component    int        `json:"source_component"`
			Triangles    int        `json:"triangle_count"`
			NormalDot    float64    `json:"normal_dot_screen"`
			SourceGapsMM [3]float64 `json:"source_vertex_signed_distance_mm"`
			ClosedDot    float64    `json:"closed_normal_dot_opposite_deck"`
		}{SourceObject: row.SourceObject, Component: row.SourceComponent, Triangles: row.TriangleCount}
		switch row.SourceComponent {
		case 310:
			component.NormalDot, component.SourceGapsMM = .999929, [3]float64{-.042, -.030, -.018}
		case 313:
			component.NormalDot, component.SourceGapsMM = .999913, [3]float64{.006, .159, .170}
		case 92:
			component.NormalDot, component.SourceGapsMM = .998811, [3]float64{-.185, -.164, .173}
		}
		component.ClosedDot = component.NormalDot
		evidence.Components = append(evidence.Components, component)
	}
	return evidence
}

func verifyLaptopRoleOverrideEvidence(got laptopRoleOverrideEvidence) error {
	want := expectedLaptopRoleOverrideEvidence()
	if got.Method != want.Method || got.ClosedAngle != want.ClosedAngle || got.Components == nil ||
		len(got.Components) != len(want.Components) || got.ScreenNormal != want.ScreenNormal ||
		got.FittedAxis != want.FittedAxis || got.DeckNormal != want.DeckNormal || got.AllClosedGapsMM != want.AllClosedGapsMM {
		return errors.New("plane and fitted-pose measurement evidence differs")
	}
	for index := range want.Components {
		if got.Components[index] != want.Components[index] {
			return errors.New("component normal or signed-distance evidence differs")
		}
	}
	return nil
}

func expectedLaptopRoleOverrides() []laptopRoleOverride {
	return []laptopRoleOverride{
		{SourceNodeIndex: 0, SourceObject: "Object_10", SourceComponent: 310, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 184},
		{SourceNodeIndex: 0, SourceObject: "Object_10", SourceComponent: 313, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 148},
		{SourceNodeIndex: 2, SourceObject: "Object_20", SourceComponent: 92, SourceRole: "hinge-review", CandidateRole: "lid", TriangleCount: 456},
	}
}

func verifyLaptopRoleOverrides(got []laptopRoleOverride) error {
	want := expectedLaptopRoleOverrides()
	if len(got) != len(want) {
		return errors.New("role override component count differs")
	}
	for i := range want {
		if got[i] != want[i] {
			return errors.New("role assignment changed an unapproved source component or triangle count")
		}
	}
	return nil
}

// Opt-in only: the fixed adapter stages pinned research inputs, runs through the
// owned Blender operation, and retains an independently checked local candidate.
func TestOptInLaptopStarterDerivation(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STARTER") != "1" {
		t.Skip("opt-in fixed laptop starter derivation; no Blender process is launched by default")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("pinned Blender profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STARTER_DIR"))
	if os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STARTER_DIR") == "" || !filepath.IsAbs(root) || !strings.Contains(filepath.ToSlash(root), "/goal/evidence/E3/scene-studies/laptop/asset-derivation/runs/") || filepath.Base(root) == "runs" {
		t.Fatalf("laptop starter attempt outside its fixed evidence root: %q", root)
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
		t.Fatalf("Blender profile differs from exact admitted artifact: %+v", profile)
	}
	attempt := filepath.Join(root, "cpu-laptop-starter")
	if err := os.Mkdir(attempt, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestSHA, scriptSHA := stageLaptopStarterInputs(t, attempt)
	runOwnedBlenderOperation(t, attempt, bin.GetPath(), profile.GetArtifactIdentity(), "CPU", "laptop-starter-derive", "", manifestSHA, scriptSHA, func() error {
		return verifyLaptopStarter(attempt, manifestSHA, scriptSHA)
	})
}

func TestOptInVerifyRetainedLaptopDerive16(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STARTER_VERIFY_DERIVE16") != "1" {
		t.Skip("set DEVICE_STUDIO_E3_LAPTOP_STARTER_VERIFY_DERIVE16=1 to verify the retained derive16 output without rerunning Blender")
	}
	_, repoRoot := laptopTestRoots(t)
	attempt := filepath.Join(repoRoot, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/laptop/asset-derivation/runs/derive-16/cpu-laptop-starter")
	for _, name := range []string{"derive_laptop_starter.py", "laptop-starter-manifest.json"} {
		path := filepath.Join(attempt, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("retained staged %s is not a regular file: %v", name, err)
		}
	}
	manifest, err := readPhoneTableRegular(filepath.Join(attempt, "laptop-starter-manifest.json"), 128<<10)
	if err != nil {
		t.Fatal(err)
	}
	script, err := readPhoneTableRegular(filepath.Join(attempt, "derive_laptop_starter.py"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyLaptopStarter(attempt, blenderDigest(manifest), blenderDigest(script)); err != nil {
		t.Fatalf("retained derive16 output failed independent verification: %v", err)
	}
}

func stageLaptopStarterInputs(t *testing.T, attempt string) (string, string) {
	t.Helper()
	scenarioRoot, _ := laptopTestRoots(t)
	assetRoot := filepath.Join(scenarioRoot, "assets/sources/laptop-starter")
	evidenceRoot := filepath.Join(scenarioRoot, "docs/internal/goal/evidence/E3/scene-studies/laptop")
	manifest := readPinnedPhoneFile(t, filepath.Join(assetRoot, "laptop-starter-manifest.json"), laptopStarterManifestSHA, 128<<10)
	script := readPinnedPhoneFile(t, filepath.Join(assetRoot, "derive_laptop_starter.py"), laptopStarterScriptSHA, 1<<20)
	sourcePath := "/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/imported-laptop/normalized-macbook.glb"
	source := readPinnedPhoneFile(t, sourcePath, laptopStarterSourceSHA, 8<<20)
	inventory := readPinnedPhoneFile(t, filepath.Join(evidenceRoot, "component-candidates.json"), laptopStarterInventorySHA, 2<<20)
	if len(source) != laptopStarterSourceBytes {
		t.Fatal("pinned laptop source size differs")
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"inputs/normalized-macbook.glb": source, "inputs/component-candidates.json": inventory} {
		if err := writeDurable(attempt, name, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeDurable(attempt, "laptop-starter-manifest.json", manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "derive_laptop_starter.py", script); err != nil {
		t.Fatal(err)
	}
	bindings, err := json.MarshalIndent(map[string]any{
		"manifest_sha256": laptopStarterManifestSHA, "script_sha256": laptopStarterScriptSHA,
		"inputs": []map[string]any{
			{"role": "CC_BY_4_0_research_model", "path": sourcePath, "stage": "inputs/normalized-macbook.glb", "sha256": laptopStarterSourceSHA, "bytes": len(source), "rights": "Jack Baeten, CC BY 4.0; attribution required; candidate not yet published"},
			{"role": "derived_partition_inventory", "stage": "inputs/component-candidates.json", "sha256": laptopStarterInventorySHA, "bytes": len(inventory)},
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "source-bindings.json", append(bindings, '\n')); err != nil {
		t.Fatal(err)
	}
	return laptopStarterManifestSHA, laptopStarterScriptSHA
}

// laptopTestRoots uses Go's package working directory, which is stable under
// -trimpath. runtime.Caller paths may be module-relative in Test Genie builds.
func laptopTestRoots(t *testing.T) (scenarioRoot, repoRoot string) {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get rendering package working directory: %v", err)
	}
	apiRoot := filepath.Clean(filepath.Join(workingDir, "../.."))
	if _, err := os.Stat(filepath.Join(apiRoot, "go.mod")); err != nil {
		t.Fatalf("rendering package working directory has no API go.mod at %s: %v", apiRoot, err)
	}
	scenarioRoot = filepath.Dir(apiRoot)
	repoRoot = filepath.Clean(filepath.Join(apiRoot, "../../.."))
	if filepath.Clean(filepath.Join(repoRoot, "scenarios/device-studio")) != scenarioRoot {
		t.Fatalf("unexpected Device Studio package root: %s", scenarioRoot)
	}
	return scenarioRoot, repoRoot
}

func verifyLaptopStarter(dir, manifestSHA, scriptSHA string) error {
	outputs := filepath.Join(dir, "outputs")
	if err := verifyLaptopStarterOutputNames(outputs); err != nil {
		return err
	}
	receiptBytes, err := readPhoneTableRegular(filepath.Join(outputs, "laptop-starter-inventory.json"), 1<<20)
	if err != nil {
		return fmt.Errorf("laptop starter receipt: %w", err)
	}
	var receipt laptopStarterReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return err
	}
	glb, err := readPhoneTableRegular(filepath.Join(outputs, "laptop-neutral-rig.glb"), laptopStarterGLBMax)
	if err != nil {
		return fmt.Errorf("laptop neutral rig GLB: %w", err)
	}
	if receipt.Profile != "device-studio-laptop-starter-derive-v9" || receipt.Status != "derived_candidate_unreviewed" || receipt.SourceSHA != laptopStarterSourceSHA || receipt.SourceBytes != laptopStarterSourceBytes || receipt.InventorySHA != laptopStarterInventorySHA || receipt.ManifestSHA != manifestSHA || receipt.ScriptSHA != scriptSHA || receipt.Blender != "5.2.2 LTS" || receipt.Triangles != 112502 || receipt.MeshNodes != 4 || receipt.RoleTriangles["base"] != 99866 || receipt.RoleTriangles["lid"] != 8470 || receipt.RoleTriangles["hinge-review"] != 4166 || receipt.NeutralScreen.SourceSHA != "b717b7a795896812a34dcd8dc3514fde3da4199052fca537e9105488a94ee63f" || receipt.NeutralScreen.Claim == "" || len(receipt.Poses) != 3 {
		return errors.New("laptop starter receipt differs from pinned source/profile/role census")
	}
	if err := verifyLaptopRoleAssignment(receipt.RoleAssignment); err != nil {
		return err
	}
	if err := verifyLaptopLocalWitnesses(receipt.LocalWitnesses); err != nil {
		return err
	}
	if err := verifyLaptopStarterClosedFit(receipt.ClosedPoseFit); err != nil {
		return err
	}
	if err := verifyLaptopTrimAlignment(receipt.TrimAlignment); err != nil {
		return err
	}
	if len(receipt.WorldBackground.LinearRGBA) != 4 || receipt.WorldBackground.LinearRGBA[0] != 0.06 || receipt.WorldBackground.LinearRGBA[1] != 0.06 || receipt.WorldBackground.LinearRGBA[2] != 0.06 || receipt.WorldBackground.LinearRGBA[3] != 1 || receipt.WorldBackground.Strength != 1 {
		return errors.New("laptop diagnostic world background differs from pinned linear values")
	}
	if !strings.Contains(receipt.Limits, "not a continuous swept-clearance proof") || receipt.Hierarchy["base"] != "LaptopBaseCandidate" || receipt.Hierarchy["lid_pivot"] != "LaptopLidPivotCandidate" || receipt.Hierarchy["unresolved_fixed"] != "LaptopHingeReview" || receipt.Hierarchy["lid_screen_node"] != "LaptopScreenSurface" {
		return errors.New("laptop starter receipt overclaims its candidate rig or omits unresolved hinge status")
	}
	if receipt.Outputs["laptop-neutral-rig.glb"].SHA != blenderDigest(glb) || receipt.Outputs["laptop-neutral-rig.glb"].Bytes != int64(len(glb)) {
		return errors.New("candidate GLB digest/size differs from its receipt")
	}
	for _, sample := range []struct {
		label string
		angle float64
	}{{"source-open", 0}, {"intermediate", 30}, {"closed", receipt.ClosedPoseFit.PlaneAngle}} {
		var pose laptopStarterPose
		if err := json.Unmarshal(receipt.Poses[sample.label], &pose); err != nil {
			return err
		}
		if pose.Label != sample.label || absFloat(pose.Angle-sample.angle) > 1e-6 || pose.OverlapPairs < 0 || len(pose.Views) != 2 || pose.Status == "" {
			return fmt.Errorf("pose %s lacks sampled status or both fixed views", sample.label)
		}
		for _, view := range []string{"side", "rear"} {
			facts, ok := pose.Views[view]
			wantName := fmt.Sprintf("%s-%s.png", sample.label, view)
			if !ok || facts.Output != wantName || facts.Margin < 0.04 || facts.Margin > 1 {
				return fmt.Errorf("pose %s %s view is missing, cropped, or misnamed", sample.label, view)
			}
		}
	}
	if err := verifyLaptopStarterGLB(glb, receipt); err != nil {
		return err
	}
	var pngBytes int64
	for _, label := range []string{"source-open", "intermediate", "closed"} {
		for _, view := range []string{"side", "rear"} {
			name := fmt.Sprintf("%s-%s.png", label, view)
			facts, ok := receipt.Outputs[name]
			if !ok || facts.Width != 960 || facts.Height != 540 || facts.Bytes <= 0 || facts.Bytes > laptopStarterPNGMax || len(facts.SHA) != 64 {
				return fmt.Errorf("diagnostic image facts invalid for %s", name)
			}
			pngBytes += facts.Bytes
			if pngBytes > laptopStarterPNGTotalMax {
				return errors.New("diagnostic PNG aggregate exceeds fixed 48 MiB limit")
			}
			if err := verifyLaptopStarterPNG(filepath.Join(outputs, name), facts); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	for _, name := range []string{"closed-hinge-material-macro.png", "closed-front-material-macro.png"} {
		facts, ok := receipt.Outputs[name]
		if !ok || facts.Width != 1280 || facts.Height != 720 || facts.Bytes <= 0 || facts.Bytes > laptopStarterPNGMax || !strings.Contains(facts.Materials, "original non-screen source slots") {
			return fmt.Errorf("source-material macro facts invalid for %s", name)
		}
		pngBytes += facts.Bytes
		if pngBytes > 64<<20 {
			return errors.New("diagnostic PNG aggregate exceeds fixed 64 MiB limit")
		}
		if err := verifyLaptopStarterPNGDimensions(filepath.Join(outputs, name), facts, 1280, 720); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if len(receipt.ComponentDiagnostics) != 6 {
		return errors.New("component-keyed diagnostic receipt is incomplete")
	}
	wantComponents := []struct {
		object                     string
		node, component, triangles int
		role                       string
		mesh                       string
		label                      string
		color                      []float64
	}{
		{"Object_10", 0, 310, 184, "lid", "LaptopLidGeometry", "Object_10 component 310", []float64{0.9, 0.15, 0.7, 1}},
		{"Object_10", 0, 313, 148, "lid", "LaptopLidGeometry", "Object_10 component 313", []float64{0.1, 0.75, 0.9, 1}},
		{"Object_20", 2, 92, 456, "lid", "LaptopLidGeometry", "Object_20 component 92", []float64{0.4, 0.9, 0.18, 1}},
		{"Object_42", 8, 12, 584, "hinge-review", "LaptopHingeTrimCandidate", "Object_42 component 12", []float64{1.0, 0.38, 0.06, 1}},
		{"Object_22", 3, 0, 3582, "hinge-review", "LaptopHingeReviewGeometry", "Object_22 component 0", []float64{0.62, 0.27, 0.94, 1}},
		{"Object_10", 0, 316, 298, "lid", "LaptopLidGeometry", "Object_10 component 316", []float64{0.98, 0.82, 0.08, 1}},
	}
	for index, want := range wantComponents {
		got := receipt.ComponentDiagnostics[index]
		if got.SourceObject != want.object || got.SourceNodeIndex != want.node || got.SourceComponent != want.component || got.CandidateRole != want.role || got.DerivedMeshName != want.mesh || got.TriangleCount != want.triangles || got.Label != want.label || got.Visibility != "visible" || !got.TransformUnchanged || len(got.Color) != 4 {
			return errors.New("component-keyed diagnostic binding differs from exact source component")
		}
		for channel := range want.color {
			if math.Abs(got.Color[channel]-want.color[channel]) > 1e-6 {
				return errors.New("component diagnostic color differs from fixed key")
			}
		}
	}
	for _, spec := range []struct {
		name          string
		width, height int
	}{
		{"closed-component-key-side.png", 960, 540},
		{"closed-component-key-hinge-macro.png", 1280, 720},
	} {
		facts, ok := receipt.Outputs[spec.name]
		if !ok || facts.Width != spec.width || facts.Height != spec.height || facts.Bytes <= 0 || facts.Bytes > laptopStarterPNGMax || facts.Diagnostic == "" {
			return fmt.Errorf("component diagnostic output facts invalid for %s", spec.name)
		}
		pngBytes += facts.Bytes
		if pngBytes > 64<<20 {
			return errors.New("diagnostic PNG aggregate exceeds fixed 64 MiB limit")
		}
		if err := verifyLaptopStarterPNGDimensions(filepath.Join(outputs, spec.name), facts, spec.width, spec.height); err != nil {
			return fmt.Errorf("%s: %w", spec.name, err)
		}
		if spec.name == "closed-component-key-hinge-macro.png" {
			region := facts.CameraTargetRegion
			if facts.CameraTargetRule != "closed-hinge-material-macro-v1" || len(facts.CameraTargetWorld) != 3 ||
				math.Abs(facts.CameraOrthoWidth-0.16) > 1e-9 || math.Abs((region.YMax-region.YMin)-0.16) > 1e-9 ||
				math.Abs((region.YMin+region.YMax)/2-facts.CameraTargetWorld[1]) > 1e-9 ||
				math.Abs((region.ZMin+region.ZMax)/2-facts.CameraTargetWorld[2]) > 1e-9 ||
				region.ZMax <= region.ZMin {
				return errors.New("keyed hinge macro lacks a bounded camera target tied to the material-macro rear band")
			}
			for _, value := range facts.CameraTargetWorld {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					return errors.New("keyed hinge macro camera target is non-finite")
				}
			}
		}
	}
	return nil
}

func verifyLaptopLocalWitnesses(items map[string]laptopLocalWitness) error {
	for name, item := range items {
		wantScreen := name == "base_to_screen_surface"
		if (name != "base_to_lid_underside" && !wantScreen) || item.Samples <= 0 || item.Within20mm != item.Samples || item.Min < 0 || item.Median < item.Min || item.P95 < item.Median || item.Max < item.P95 || item.SignedMedian < item.SignedMin || item.SignedP95 < item.SignedMedian || item.SignedNegative < 0 || item.SignedNegative > item.Samples || len(item.NegativeSamples) > item.SignedNegative || (!item.NegativeTruncated && len(item.NegativeSamples) != item.SignedNegative) || math.IsNaN(item.SignedMin) || math.IsNaN(item.SignedMedian) || math.IsNaN(item.SignedP95) || math.IsInf(item.SignedMin, 0) || math.IsInf(item.SignedMedian, 0) || math.IsInf(item.SignedP95, 0) || len(item.Witnesses) == 0 || item.ScreenIncluded != wantScreen || item.Qualification == "" || !strings.Contains(item.Method, "world-space triangle centroids") {
			return fmt.Errorf("local surface witness %q is incomplete", name)
		}
		for _, sample := range item.NegativeSamples {
			if sample.SourceObject == "" || sample.BaseSourceObject == "" || sample.SourceComponent < 0 || sample.BaseSourceComponent < 0 || !(sample.SignedGap < 0) || math.IsNaN(sample.SignedGap) || math.IsInf(sample.SignedGap, 0) {
				return fmt.Errorf("local surface witness %q has an invalid negative sample identity", name)
			}
		}
	}
	if len(items) != 2 {
		return errors.New("local surface witnesses must separate lid underside and screen")
	}
	return nil
}

func verifyLaptopTrimAlignment(trim laptopStarterTrimAlignment) error {
	if trim.Status != "authored_fixed_base_trim_alignment_hypothesis" || trim.SourceObject != "Object_42" || trim.SourceComponent != 12 || trim.TriangleCount != 584 || trim.Parent != "LaptopHingeReview" || trim.RotatesWithLid || len(trim.Translation) != 3 || !trim.MatchesAxisDelta || trim.Claim == "" {
		return errors.New("fixed trim alignment receipt is incomplete or changes ownership")
	}
	for _, value := range trim.Translation {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("fixed trim alignment has a non-finite transform")
		}
	}
	return nil
}

func verifyLaptopStarterGLB(glb []byte, receipt laptopStarterReceipt) error {
	chunks, err := splitGLBChunks(glb)
	if err != nil || len(chunks) != 2 {
		return errors.New("candidate GLB has invalid chunks")
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return err
	}
	if len(doc.Nodes) != 8 || len(doc.Meshes) != 4 || len(doc.Images) == 0 || len(doc.Cameras) != 0 || len(doc.Animations) != 0 || len(doc.Skins) != 0 || len(doc.Buffers) != 1 || doc.Buffers[0].URI != "" {
		return errors.New("candidate GLB hierarchy or closed feature set differs")
	}
	triangleCount := uint64(0)
	for _, node := range doc.Nodes {
		if node.Camera != nil || node.Skin != nil {
			return errors.New("candidate GLB contains an unsupported camera or skin")
		}
		if node.Mesh == nil {
			continue
		}
		if int(*node.Mesh) >= len(doc.Meshes) {
			return errors.New("candidate node references a missing mesh")
		}
		for _, primitive := range doc.Meshes[*node.Mesh].Primitives {
			if primitive.Mode != nil && *primitive.Mode != 4 || primitive.Indices == nil || int(*primitive.Indices) >= len(doc.Accessors) {
				return errors.New("candidate primitive is not indexed triangles")
			}
			indices := doc.Accessors[*primitive.Indices].Count
			if indices%3 != 0 {
				return errors.New("candidate primitive has incomplete triangle indices")
			}
			triangleCount += uint64(indices / 3)
		}
	}
	if triangleCount != 112502 {
		return errors.New("candidate GLB source triangle census differs")
	}
	required := map[string]bool{"LaptopStarterRoot": false, "LaptopBaseCandidate": false, "LaptopLidPivotCandidate": false, "LaptopHingeReview": false, "LaptopBaseGeometry": false, "LaptopLidGeometry": false, "LaptopScreenSurface": false, "LaptopHingeReviewGeometry": false}
	nodeIndex := map[string]int{}
	for i, node := range doc.Nodes {
		if _, ok := required[node.Name]; ok {
			required[node.Name] = true
			nodeIndex[node.Name] = i
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("candidate GLB hierarchy lacks %s", name)
		}
	}
	children := func(parent, child string) bool {
		for _, index := range doc.Nodes[nodeIndex[parent]].Children {
			if int(index) == nodeIndex[child] {
				return true
			}
		}
		return false
	}
	if !children("LaptopStarterRoot", "LaptopBaseCandidate") || !children("LaptopStarterRoot", "LaptopLidPivotCandidate") || !children("LaptopStarterRoot", "LaptopHingeReview") || !children("LaptopBaseCandidate", "LaptopBaseGeometry") || !children("LaptopLidPivotCandidate", "LaptopLidGeometry") || !children("LaptopLidPivotCandidate", "LaptopScreenSurface") || !children("LaptopHingeReview", "LaptopHingeReviewGeometry") {
		return errors.New("candidate GLB part-parent hierarchy differs")
	}
	screenNodeIndex, meshIndex := -1, -1
	for i, node := range doc.Nodes {
		if node.Name == "LaptopScreenSurface" {
			if node.Mesh == nil || screenNodeIndex >= 0 {
				return errors.New("screen surface node is absent or ambiguous")
			}
			screenNodeIndex, meshIndex = i, int(*node.Mesh)
		}
	}
	if screenNodeIndex < 0 || meshIndex >= len(doc.Meshes) || len(doc.Meshes[meshIndex].Primitives) != 1 {
		return errors.New("screen surface has unexpected primitive structure")
	}
	primitive := doc.Meshes[meshIndex].Primitives[0]
	if primitive.Material == nil || int(*primitive.Material) >= len(doc.Materials) {
		return errors.New("screen surface material is absent")
	}
	material := doc.Materials[*primitive.Material]
	if material.EmissiveTexture == nil || int(material.EmissiveTexture.Index) >= len(doc.Textures) {
		return errors.New("screen surface no longer binds its emissive texture")
	}
	texture := doc.Textures[material.EmissiveTexture.Index]
	if texture.Source == nil || int(*texture.Source) >= len(doc.Images) {
		return errors.New("screen texture has no embedded image source")
	}
	image := doc.Images[*texture.Source]
	if image.URI != "" || image.BufferView == nil || int(*image.BufferView) >= len(doc.BufferViews) {
		return errors.New("screen image must remain embedded in candidate GLB")
	}
	view := doc.BufferViews[*image.BufferView]
	if uint64(view.ByteOffset)+uint64(view.ByteLength) > uint64(len(chunks[1])) || view.ByteLength == 0 {
		return errors.New("screen image range exceeds candidate GLB")
	}
	data := chunks[1][view.ByteOffset : view.ByteOffset+view.ByteLength]
	if blenderDigest(data) == receipt.NeutralScreen.SourceSHA || blenderDigest(data) != receipt.NeutralScreen.ImageSHA {
		return errors.New("screen image bytes are original or differ from receipt")
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil || decoded.Bounds().Dx() != 1024 || decoded.Bounds().Dy() != 512 {
		return errors.New("neutral screen PNG cannot be decoded at pinned dimensions")
	}
	bounds := decoded.Bounds()
	first := decoded.At(bounds.Min.X, bounds.Min.Y)
	r0, g0, b0, a0 := first.RGBA()
	if r0 != g0 || g0 != b0 {
		return errors.New("neutral screen raster is not achromatic")
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			if r != r0 || g != g0 || b != b0 || a != a0 || a != 0xffff {
				return errors.New("neutral screen raster is not uniform opaque gray")
			}
		}
	}
	return nil
}

func verifyLaptopStarterPNG(path string, facts laptopStarterOutput) error {
	return verifyLaptopStarterPNGDimensions(path, facts, 960, 540)
}

func verifyLaptopStarterPNGDimensions(path string, facts laptopStarterOutput, width, height int) error {
	data, err := readPhoneTableRegular(path, laptopStarterPNGMax)
	if err != nil || int64(len(data)) != facts.Bytes || blenderDigest(data) != facts.SHA {
		return errors.New("diagnostic PNG bytes/hash differ from receipt")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != width || config.Height != height {
		return errors.New("diagnostic PNG header dimensions differ")
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil || image.Bounds().Dx() != width || image.Bounds().Dy() != height {
		return errors.New("diagnostic PNG full decode failed")
	}
	return nil
}

func verifyLaptopStarterOutputNames(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("laptop starter outputs are not a regular directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 12 {
		return errors.New("laptop outputs must be one GLB, ten fixed PNGs, and one receipt")
	}
	want := map[string]bool{"laptop-neutral-rig.glb": true, "laptop-starter-inventory.json": true,
		"closed-hinge-material-macro.png": true, "closed-front-material-macro.png": true,
		"closed-component-key-side.png": true, "closed-component-key-hinge-macro.png": true}
	for _, label := range []string{"source-open", "intermediate", "closed"} {
		for _, view := range []string{"side", "rear"} {
			want[fmt.Sprintf("%s-%s.png", label, view)] = true
		}
	}
	for _, entry := range entries {
		mode, e := entry.Info()
		if !want[entry.Name()] || e != nil || !mode.Mode().IsRegular() || mode.Mode()&os.ModeSymlink != 0 {
			return errors.New("laptop output directory contains unsafe or unapproved files")
		}
	}
	return nil
}
