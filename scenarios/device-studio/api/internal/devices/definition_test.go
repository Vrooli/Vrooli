package devices

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedRegistryPreservesKnownAndUnknownFacts(t *testing.T) {
	definitions, err := LoadDefinitions(seedAssetsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 11 {
		t.Fatalf("loaded %d seed definitions, want 11", len(definitions))
	}
	groups := make(map[Group]bool)
	for _, definition := range definitions {
		groups[definition.Group] = true
		if definition.Qualification != "unqualified" {
			t.Errorf("%s unexpectedly claims qualification", definition.ID)
		}
		if got := definition.PhysicalQualification(); got.Status == Compatible {
			t.Errorf("%s unexpectedly passes physical qualification", definition.ID)
		}
	}
	for _, group := range []Group{GroupPhone, GroupLaptop, GroupTablet, GroupFoldable, GroupWatch, GroupMonitorTV, GroupSmartDisplayFrame, GroupVehicleHost} {
		if !groups[group] {
			t.Errorf("missing required device group %q", group)
		}
	}

	byID := make(map[string]Definition)
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	concept := byID["foldable-iphone-concept"]
	if concept.IdentityStatus != IdentityConcept {
		t.Fatalf("foldable iPhone must be labeled concept, got %q", concept.IdentityStatus)
	}
	for _, measurement := range []Measurement{concept.Dimensions["open"].Width, concept.Dimensions["open"].Height, concept.Dimensions["open"].Depth} {
		if measurement.Value != nil || measurement.Method != "unknown" {
			t.Fatalf("concept must not contain invented dimensions: %+v", measurement)
		}
	}
	vehicle := byID["bmw-i4-2022-curved-display-host"]
	for _, measurement := range []Measurement{vehicle.Dimensions["dashboard_assembly"].Width, vehicle.Dimensions["dashboard_assembly"].Height, vehicle.Dimensions["dashboard_assembly"].Depth} {
		if measurement.Value != nil {
			t.Fatalf("vehicle assembly dimensions should remain unknown: %+v", measurement)
		}
	}
	if CheckMotion(vehicle, "camera-orbit").Status != NeedsReview {
		t.Fatal("unknown vehicle camera motion must need review")
	}
}

func TestValidateRejectsAppSafeAreaThatPaintsHardwareOcclusion(t *testing.T) {
	d := validFixture()
	d.Displays[0].AppSafeArea.Bounds = &Rect{X: 0.1, Y: 0.05, Width: 0.8, Height: 0.8}
	d.Displays[0].HardwareOcclusions = []HardwareOcclusion{{
		ID: "camera-island", Kind: "camera_cutout",
		Region: Region{Status: "qualified", Shape: "rect", Bounds: &Rect{X: 0.4, Y: 0.05, Width: 0.2, Height: 0.1}, Evidence: []Evidence{{Source: "fixture", Kind: "test"}}},
	}}
	if err := d.Validate(); err == nil {
		t.Fatal("expected app-safe region that overlaps a physical cutout to be rejected")
	}
}

func TestCircleBoundsAreNotTreatedAsAnActiveRectangle(t *testing.T) {
	d := validFixture()
	d.Displays[0].ActiveArea = Region{Status: "qualified", Shape: "circle", GeometryRef: "active-circle.svg", GeometrySpace: "physical_mm", Bounds: &Rect{X: 0.3, Y: 0.1, Width: 0.4, Height: 0.8}, Evidence: []Evidence{{Source: "fixture", Kind: "test"}}}
	d.Displays[0].AppSafeArea = Region{Status: "qualified", Shape: "rect", Bounds: &Rect{X: 0.35, Y: 0.2, Width: 0.3, Height: 0.6}, Evidence: []Evidence{{Source: "fixture", Kind: "test"}}}
	if err := d.Validate(); err == nil {
		t.Fatal("non-square UV bounds alone must not qualify physical circle or safe-area containment")
	}
	d.Displays[0].GeometryRelations = []GeometryRelation{
		{Subject: "active_area", Relation: "contained_by", Object: "physical_boundary", Status: "qualified", Evidence: []Evidence{{Source: "fixture://geometry-review", Kind: "review"}}},
		{Subject: "app_safe_area", Relation: "contained_by", Object: "active_area", Status: "qualified", Evidence: []Evidence{{Source: "fixture://geometry-review", Kind: "review"}}},
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("physical-mm geometry and reviewed containment relation should validate: %v", err)
	}
}

func TestQualifiedNonRectangularRegionsNeedPhysicalGeometry(t *testing.T) {
	for _, shape := range []string{"circle", "polygon", "mask", "rounded_rect", "unknown"} {
		t.Run(shape, func(t *testing.T) {
			d := validFixture()
			d.Displays[0].ActiveArea = Region{Status: "qualified", Shape: shape, Bounds: &Rect{X: 0.2, Y: 0.1, Width: 0.5, Height: 0.7}, Evidence: []Evidence{{Source: "fixture", Kind: "test"}}}
			if err := d.Validate(); err == nil {
				t.Fatalf("qualified %s without physical geometry should be rejected", shape)
			}
		})
	}
}

func TestLoadDefinitionsRejectsUnknownFieldsTrailingJSONAndDuplicateIdentity(t *testing.T) {
	base, err := json.Marshal(validFixture())
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(base, &object); err != nil {
		t.Fatal(err)
	}
	object["unexpected_field"] = json.RawMessage(`true`)
	unknownField, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}

	duplicateIdentity := validFixture()
	duplicateIdentity.ID = "fixture-2"
	duplicateIdentityJSON, err := json.Marshal(duplicateIdentity)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		files    map[string][]byte
		wantText string
	}{
		{name: "unknown field", files: map[string][]byte{"one.json": unknownField}, wantText: "unknown field"},
		{name: "trailing JSON value", files: map[string][]byte{"one.json": append(append([]byte(nil), base...), []byte("\n{}")...)}, wantText: "trailing JSON value"},
		{name: "duplicate ID", files: map[string][]byte{"one.json": base, "two.json": base}, wantText: "duplicate device definition ID"},
		{name: "duplicate identity", files: map[string][]byte{"one.json": base, "two.json": duplicateIdentityJSON}, wantText: "duplicate device identity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := LoadDefinitions(dir); err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("LoadDefinitions error = %v, want substring %q", err, tc.wantText)
			}
		})
	}
}

func TestValidateRejectsUnknownDimensionWithUnstatedMethod(t *testing.T) {
	d := validFixture()
	d.Dimensions["default"] = Dimensions{Width: Measurement{Value: floatPointer(70), Unit: "mm", Meaning: "width", Configuration: "default", Method: "unknown", Uncertainty: Uncertainty{Status: "unknown"}, Evidence: []Evidence{{Source: "x", Kind: "source"}}}, Height: knownMeasurement(140), Depth: knownMeasurement(8)}
	if err := d.Validate(); err == nil {
		t.Fatal("unknown method with a numeric value must not masquerade as a measurement")
	}
}

func TestCheckSupportRequiresGroupInterfaceAndEnvelope(t *testing.T) {
	d := validFixture()
	d.Group = GroupPhone
	d.Supports = []SupportCapability{{Kind: "tabletop", InterfaceID: "base-a", Status: "qualified"}}
	compatible := SupportSlot{ID: "slot", Kind: "tabletop", AllowedGroups: []Group{GroupPhone}, RequiredInterface: "base-a", DimensionState: "default", RequiresKnownEnvelope: true}
	if got := CheckSupport(d, compatible); got.Status != Compatible {
		t.Fatalf("expected exact supported slot compatibility, got %+v", got)
	}

	wrongGroup := compatible
	wrongGroup.AllowedGroups = []Group{GroupTablet}
	if got := CheckSupport(d, wrongGroup); got.Status != Incompatible {
		t.Fatalf("expected group mismatch rejection, got %+v", got)
	}

	wrongInterface := compatible
	wrongInterface.RequiredInterface = "base-b"
	if got := CheckSupport(d, wrongInterface); got.Status != Incompatible {
		t.Fatalf("expected qualified interface mismatch rejection, got %+v", got)
	}

	d.Dimensions["default"] = Dimensions{Width: unknownMeasurement(), Height: knownMeasurement(140), Depth: knownMeasurement(8)}
	if got := CheckSupport(d, compatible); got.Status != NeedsReview {
		t.Fatalf("unknown envelope must prevent physical compatibility, got %+v", got)
	}

	d.Supports[0].Status = "candidate_only"
	compatible.RequiresKnownEnvelope = false
	if got := CheckSupport(d, compatible); got.Status != NeedsReview {
		t.Fatalf("candidate-only support cannot be called compatible, got %+v", got)
	}
}

func TestSupportDoesNotInferOpenFitFromClosedEnvelope(t *testing.T) {
	d := validFixture()
	d.Dimensions = map[string]Dimensions{"closed": {Width: knownMeasurement(300), Height: knownMeasurement(200), Depth: knownMeasurement(20)}}
	d.Supports = []SupportCapability{{Kind: "tabletop", InterfaceID: "base-a", Status: "qualified"}}
	slot := SupportSlot{ID: "open-slot", Kind: "tabletop", RequiredInterface: "base-a", DimensionState: "open", RequiresKnownEnvelope: true}
	if got := CheckSupport(d, slot); got.Status != NeedsReview {
		t.Fatalf("closed-state dimensions cannot establish open-state fit, got %+v", got)
	}
}

func TestCheckPoseAndMotionRespectJointLimits(t *testing.T) {
	d := validFixture()
	lo, hi := -0.25, 1.5
	d.Joints = []Joint{{ID: "lid", ParentPart: "base", ChildPart: "display", Axis: &Vector3{X: 1}, PivotMeters: &Vector3{X: 0.1}, LowerRadians: &lo, UpperRadians: &hi, Status: "qualified"}}
	d.SupportedPoses = []Pose{{Name: "open", Status: "qualified", JointState: map[string]float64{"lid": 1.2}}}
	d.SupportedMotion = []Capability{{Name: "lid_hinge_open_close", Status: "qualified", MotionKind: "articulated", JointIDs: []string{"lid"}}}
	if err := d.Validate(); err != nil {
		t.Fatalf("fixture should validate: %v", err)
	}
	if got := CheckPose(d, "open"); got.Status != Compatible {
		t.Fatalf("in-range qualified pose should be compatible, got %+v", got)
	}
	if got := CheckMotion(d, "lid_hinge_open_close"); got.Status != Compatible {
		t.Fatalf("qualified articulation should be compatible, got %+v", got)
	}
	d.SupportedPoses[0].JointState["lid"] = 1.6
	if got := CheckPose(d, "open"); got.Status != Incompatible {
		t.Fatalf("pose outside joint range should be rejected, got %+v", got)
	}
	d.SupportedPoses = []Pose{{Name: "candidate", Status: "candidate_only"}}
	if got := CheckPose(d, "candidate"); got.Status != NeedsReview {
		t.Fatalf("candidate pose should require review, got %+v", got)
	}
	if got := CheckMotion(d, "missing"); got.Status != Incompatible {
		t.Fatalf("undeclared motion should be rejected, got %+v", got)
	}
}

func TestQualifiedArticulationCannotOmitJointEvidence(t *testing.T) {
	d := validFixture()
	d.SupportedMotion = []Capability{{Name: "laptop_hinge", Status: "qualified", MotionKind: "articulated", JointIDs: []string{}}}
	if got := CheckMotion(d, "laptop_hinge"); got.Status != NeedsReview {
		t.Fatalf("hinge motion without qualified joint must remain unqualified, got %+v", got)
	}
	d.SupportedMotion[0].Status = "unsupported"
	if got := CheckMotion(d, "laptop_hinge"); got.Status != Incompatible {
		t.Fatalf("explicitly unsupported motion must be rejected, got %+v", got)
	}
}

func TestQualifiedStatusRequiresIndependentReviewEvidence(t *testing.T) {
	d := validFixture()
	d.Qualification = "qualified"
	if err := d.Validate(); err == nil {
		t.Fatal("qualified status without a review receipt must be rejected")
	}
	d.QualificationReviews = []QualificationReview{{Scope: "physical_geometry", Result: "passed", Reviewer: "reviewer-1", Evidence: []Evidence{{Source: "review://physical-geometry/1", Kind: "human_review"}}}}
	if err := d.Validate(); err == nil {
		t.Fatal("a reviewer name alone must not qualify a candidate without source, preset, native output, and scope bindings")
	}
	d.QualificationReviews[0] = completeQualificationReview()
	if err := d.Validate(); err != nil {
		t.Fatalf("qualified definition with review evidence should validate: %v", err)
	}
	if got := d.PhysicalQualification(); got.Status != Compatible {
		t.Fatalf("qualified fully-specified fixture should pass physical check, got %+v", got)
	}
}

func completeQualificationReview() QualificationReview {
	digest := strings.Repeat("a", 64)
	return QualificationReview{Scope: "physical_geometry", Result: "passed", Reviewer: "reviewer-1", Evidence: []Evidence{{Source: "native-review:job-1", Kind: "human_review"}},
		CandidateRevision: 1, CandidateSHA256: digest, SourceBundleID: "source-bundle", SourceBundleRevision: 1, SourceManifestSHA256: digest, SourceReviewSHA256: digest, SourceArtifactID: "device-glb", SourceArtifactSHA256: digest,
		SourceAllowedUses: []string{"render"}, DimensionStates: []string{"default"}, DisplayIDs: []string{"front"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"none"},
		PresetID: "preset", PresetRevision: 1, PresetSHA256: digest, SceneID: "scene", SceneRevision: 1, SceneSHA256: digest, MotionID: "motion", MotionRevision: 1, MotionSHA256: digest,
		NativeRenderJobID: "render-job", NativeOutputSHA256: digest, NativeOutputKind: "physical_still_png", RightsClaimsAcknowledged: true, ReviewedAt: "2026-10-21T00:00:00Z"}
}

func TestMeasurementRequiresFiniteValueAndUsableEvidence(t *testing.T) {
	measurement := knownMeasurement(70)
	measurement.Evidence = []Evidence{{Source: "  ", Kind: "test"}}
	if measurement.known() {
		t.Fatal("blank evidence source must not establish a measurement")
	}
	measurement = knownMeasurement(70)
	value := math.Inf(1)
	measurement.Value = &value
	if measurement.known() {
		t.Fatal("infinite measurement must not be treated as known")
	}
}

func validFixture() Definition {
	full := Rect{X: 0, Y: 0, Width: 1, Height: 1}
	active := Rect{X: 0.05, Y: 0.05, Width: 0.9, Height: 0.9}
	safe := Rect{X: 0.1, Y: 0.1, Width: 0.8, Height: 0.8}
	evidence := []Evidence{{Source: "fixture://display-geometry", Kind: "test"}}
	return Definition{
		SchemaVersion: "device-definition/v1", ID: "fixture", Group: GroupPhone, Kind: "test_phone", IdentityStatus: IdentityGeneric, Model: "Fixture", Qualification: "unqualified",
		Dimensions:     map[string]Dimensions{"default": {Width: knownMeasurement(70), Height: knownMeasurement(140), Depth: knownMeasurement(8)}},
		Displays:       []DisplaySurface{{ID: "front", Topology: "planar", CoordinateMetric: &DisplayMetric{WidthMM: 70, HeightMM: 140, Evidence: evidence}, PhysicalBoundary: Region{Status: "qualified", Shape: "rect", Bounds: &full, Evidence: evidence}, ActiveArea: Region{Status: "qualified", Shape: "rect", Bounds: &active, Evidence: evidence}, AppSafeArea: Region{Status: "qualified", Shape: "rect", Bounds: &safe, Evidence: evidence}}},
		SupportedPoses: []Pose{}, SupportedMotion: []Capability{}, Supports: []SupportCapability{}, ModelCandidates: []ModelCandidate{}, Provenance: []Evidence{{Source: "fixture", Kind: "test"}},
	}
}

func knownMeasurement(v float64) Measurement {
	return Measurement{Value: &v, Unit: "mm", Meaning: "fixture dimension", Configuration: "default", Method: "measured", Uncertainty: Uncertainty{Status: "source_unspecified", Explanation: "Test fixture only."}, Evidence: []Evidence{{Source: "fixture", Kind: "test"}}}
}

func unknownMeasurement() Measurement {
	return Measurement{Value: nil, Unit: "mm", Meaning: "fixture dimension", Configuration: "default", Method: "unknown", Uncertainty: Uncertainty{Status: "unknown"}}
}

func floatPointer(v float64) *float64 { return &v }

func seedAssetsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "assets", "devices")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate scenario assets/devices from package working directory %q", dir)
		}
		dir = parent
	}
}
