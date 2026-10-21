package devices

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"

	"device-studio/internal/library"
	"device-studio/internal/physical"
)

func setupDeviceRevisions(t *testing.T) (context.Context, *sql.DB, *DeviceRevisionService) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteDeviceRevisionRepository(db)
	service, err := NewDeviceRevisionService(repo, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	return context.Background(), db, service
}

func TestDeviceCandidateConcurrentDifferentWritersHaveOneWinner(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	other, err := NewDeviceRevisionService(service.repo, service.now)
	if err != nil {
		t.Fatal(err)
	}
	left, right := candidateFixture(), candidateFixture()
	left.Definition.Description = "left winner candidate"
	right.Definition.Description = "right winner candidate"
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, item := range []struct {
		service *DeviceRevisionService
		input   DeviceCandidate
	}{{service, left}, {other, right}} {
		wg.Add(1)
		go func(service *DeviceRevisionService, input DeviceCandidate) {
			defer wg.Done()
			<-start
			_, err := service.SaveCandidate(ctx, input, 0)
			results <- err
		}(item.service, item.input)
	}
	close(start)
	wg.Wait()
	close(results)
	winners, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent write error: %v", err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("concurrent different candidates: winners=%d conflicts=%d", winners, conflicts)
	}
	head, err := service.GetHead(ctx, "fixture")
	if err != nil || head.Revision != 1 {
		t.Fatalf("expected one immutable winner, head=%d err=%v", head.Revision, err)
	}
}

func candidateFixture() DeviceCandidate {
	return DeviceCandidate{
		Version:              DeviceCandidateVersion,
		Definition:           validFixture(),
		Geometry:             physicalRefFixture(),
		GeometryUnit:         CanonicalUnit,
		CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical:    identityTRS(),
		Parts:                []SemanticPart{{ID: "body", Role: "assembly", NodeIndex: 0, NodeName: "unverified-glb-root", LocalFrame: identityTRS()}},
		DisplayBindings:      []DisplayPartBinding{{DisplayID: "front", PartID: "body", LocalFrame: identityTRS()}},
	}
}

func physicalRefFixture() physical.ArtifactRef {
	return physical.ArtifactRef{
		BundleID: "reviewed-phone", BundleRevision: 1,
		ManifestSHA256: strings.Repeat("a", 64), ReviewSHA256: strings.Repeat("b", 64),
		ArtifactID: "phone-glb", Role: library.RoleGeometryGLB, SizeBytes: 128, SHA256: strings.Repeat("c", 64),
	}
}

func identityTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

func TestDeviceCandidateRevisionReplayConflictAndImmutableHistory(t *testing.T) {
	ctx, db, service := setupDeviceRevisions(t)
	first, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != DeviceRevisionCandidate {
		t.Fatalf("new device revision state=%q, want candidate", first.State)
	}
	replay, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil || replay.Revision != 1 || replay.SHA256 != first.SHA256 {
		t.Fatalf("exact replay did not converge: revision=%d err=%v", replay.Revision, err)
	}
	changed := candidateFixture()
	changed.Definition.Description = "candidate revision two"
	second, err := service.SaveCandidate(ctx, changed, 1)
	if err != nil || second.Revision != 2 {
		t.Fatalf("save next revision: revision=%d err=%v", second.Revision, err)
	}
	if _, err := service.SaveCandidate(ctx, candidateFixture(), 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("reverting content at expected target revision 2 must conflict; save it at the current head instead: %v", err)
	}
	stale := changed
	stale.Definition.Description = "different stale edit"
	if _, err := service.SaveCandidate(ctx, stale, 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected stale head conflict, got %v", err)
	}
	reverted, err := service.SaveCandidate(ctx, candidateFixture(), 2)
	if err != nil || reverted.Revision != 3 || reverted.SHA256 != first.SHA256 {
		t.Fatalf("A→B→A must create revision 3: revision=%d err=%v", reverted.Revision, err)
	}
	firstReplay, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil || firstReplay.Revision != 1 {
		t.Fatalf("retry targeting revision 1 should return revision 1: revision=%d err=%v", firstReplay.Revision, err)
	}
	if _, err := service.SaveCandidate(ctx, changed, 2); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("different content at occupied target revision should conflict, got %v", err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET candidate_json=candidate_json WHERE device_id='fixture' AND revision=1`); err == nil {
		t.Fatal("immutable revision trigger allowed update")
	}
	items, err := service.ListRevisions(ctx, "fixture", 0, 10)
	if err != nil || len(items) != 3 || items[0].Revision != 1 || items[1].Revision != 2 || items[2].Revision != 3 {
		t.Fatalf("revision listing mismatch: %+v err=%v", items, err)
	}
}

func TestExplicitQualificationAppendsScopedRevisionAndPreservesCandidate(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	parent, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	review := reviewForCandidate(parent)
	qualified, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, review, parent.Revision)
	if err != nil {
		t.Fatalf("explicit scoped review: %v", err)
	}
	if qualified.Revision != 2 || qualified.State != DeviceRevisionQualified || qualified.Candidate.Definition.Qualification != "qualified" {
		t.Fatalf("review did not append qualified revision: %+v", qualified)
	}
	storedParent, err := service.GetRevision(ctx, parent.DeviceID, parent.Revision)
	if err != nil || storedParent.SHA256 != parent.SHA256 || storedParent.Candidate.Definition.Qualification != "unqualified" || len(storedParent.Candidate.Definition.QualificationReviews) != 0 {
		t.Fatalf("review changed original candidate: %+v err=%v", storedParent, err)
	}
	storedQualified, err := service.GetHead(ctx, parent.DeviceID)
	if err != nil || storedQualified.SHA256 != qualified.SHA256 || storedQualified.State != DeviceRevisionQualified {
		t.Fatalf("qualified readback lost state: %+v err=%v", storedQualified, err)
	}
	if got := storedQualified.Candidate.Definition.QualificationReviews[0].CandidateSHA256; got != parent.SHA256 {
		t.Fatalf("qualification parent digest=%q want %q", got, parent.SHA256)
	}
}

func TestQualificationRefusesMissingScopeEvidenceAndStaleHead(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	parent, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	missing := reviewForCandidate(parent)
	missing.NativeOutputSHA256 = ""
	if _, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, missing, parent.Revision); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("missing native output evidence accepted: %v", err)
	}
	wrongSource := reviewForCandidate(parent)
	wrongSource.SourceArtifactSHA256 = strings.Repeat("f", 64)
	if _, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, wrongSource, parent.Revision); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("wrong source digest accepted: %v", err)
	}
	if _, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, reviewForCandidate(parent), parent.Revision+1); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("mismatched expected head accepted: %v", err)
	}
	changed := candidateFixture()
	changed.Definition.Description = "head moved"
	if _, err := service.SaveCandidate(ctx, changed, parent.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, reviewForCandidate(parent), parent.Revision); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale candidate review did not conflict: %v", err)
	}
}

func TestVisualSetupAllowsCandidateSupportButPhysicalGeometryStillRequiresQualifiedSupport(t *testing.T) {
	candidate := candidateFixture()
	candidate.Parts = append(candidate.Parts, SemanticPart{ID: "stand", Role: "support", NodeIndex: 1, NodeName: "stand", LocalFrame: identityTRS()})
	candidate.SupportFrames = []DeviceSupportFrame{{ID: "table-contact", InterfaceID: "tabletop", PartID: "stand", LocalFrame: identityTRS()}}
	candidate.Definition.Supports = []SupportCapability{{Kind: "tabletop", InterfaceID: "tabletop", Status: "candidate_only"}}
	review := QualificationReview{Scope: "visual_setup", DimensionStates: []string{"default"}, DisplayIDs: []string{"front"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"table-contact"}}
	if err := validateQualificationScope(candidate, review); err != nil {
		t.Fatalf("visual setup should bind the exact candidate-only support without qualifying it: %v", err)
	}

	physicalReview := review
	physicalReview.Scope = "physical_geometry"
	if err := validateQualificationScope(candidate, physicalReview); err == nil || !strings.Contains(err.Error(), "qualified evidenced support") {
		t.Fatalf("physical geometry must retain the qualified-support guard, got %v", err)
	}
}

func TestVisualSetupReviewAcceptsSourcedUnknownUncertaintyWithoutQualifyingPhysicalFacts(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	input := candidateFixture()
	input.Definition.Dimensions["default"] = derivedUnknownUncertaintyDimensions()
	height := input.Definition.Dimensions["default"].Height
	height.Method = "image_estimated"
	input.Definition.Dimensions["default"] = Dimensions{Width: input.Definition.Dimensions["default"].Width, Height: height, Depth: input.Definition.Dimensions["default"].Depth}
	input.Definition.Displays[0].PhysicalBoundary = Region{Status: "unknown", Shape: "unknown"}
	input.Definition.Displays[0].ActiveArea = Region{Status: "unknown", Shape: "unknown"}
	input.Definition.Displays[0].AppSafeArea = Region{Status: "unknown", Shape: "unknown"}
	parent, err := service.SaveCandidate(ctx, input, 0)
	if err != nil {
		t.Fatal(err)
	}
	review := reviewForCandidate(parent)
	review.Scope = "visual_setup"
	qualified, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, review, parent.Revision)
	if err != nil {
		t.Fatalf("scoped visual setup review: %v", err)
	}
	if qualified.State != DeviceRevisionQualified || qualified.Revision != 2 {
		t.Fatalf("visual setup review did not append scoped revision: %+v", qualified)
	}
	if got := qualified.Candidate.Definition.Dimensions["default"].Width.Uncertainty.Status; got != "unknown" {
		t.Fatalf("review changed dimension uncertainty to %q", got)
	}
	if got := qualified.Candidate.Definition.Displays[0].ActiveArea.Status; got != "unknown" {
		t.Fatalf("review qualified unknown active area: %q", got)
	}
	if status := qualified.Candidate.Definition.PhysicalQualification().Status; status != NeedsReview {
		t.Fatalf("visual setup review made physical qualification %q, want needs-review", status)
	}
	storedParent, err := service.GetRevision(ctx, parent.DeviceID, parent.Revision)
	if err != nil || storedParent.SHA256 != parent.SHA256 || storedParent.Candidate.Definition.Qualification != "unqualified" {
		t.Fatalf("visual review changed parent candidate: %+v err=%v", storedParent, err)
	}
	if qualified.Candidate.Definition.Dimensions["default"].Height.Method != "image_estimated" {
		t.Fatal("visual review did not preserve image-estimated provenance")
	}

	_, _, assumptionService := setupDeviceRevisions(t)
	assumption := candidateFixture()
	assumption.Definition.ID = "fixture-assumption"
	assumption.Definition.Dimensions["default"] = derivedUnknownUncertaintyDimensions()
	widths := assumption.Definition.Dimensions["default"]
	widths.Width.Method = "design_assumption"
	assumption.Definition.Dimensions["default"] = widths
	assumptionParent, err := assumptionService.SaveCandidate(ctx, assumption, 0)
	if err != nil {
		t.Fatal(err)
	}
	visualReview := reviewForCandidate(assumptionParent)
	visualReview.Scope = "visual_setup"
	if _, err := assumptionService.ApplyQualificationReview(ctx, assumptionParent.DeviceID, assumptionParent.Revision, assumptionParent.SHA256, visualReview, assumptionParent.Revision); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("visual setup review accepted design-assumption dimensions: %v", err)
	}
	missingEvidence := candidateFixture()
	missingEvidence.Definition.ID = "fixture-missing-dimension-evidence"
	missingEvidence.Definition.Dimensions["default"] = derivedUnknownUncertaintyDimensions()
	missing := missingEvidence.Definition.Dimensions["default"]
	missing.Depth.Evidence = nil
	missingEvidence.Definition.Dimensions["default"] = missing
	if _, err := assumptionService.SaveCandidate(ctx, missingEvidence, 0); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("dimension without source evidence entered the reviewable catalog: %v", err)
	}
}

func TestPhysicalGeometryReviewStillRejectsUnknownUncertaintyAndDisplayRegions(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	input := candidateFixture()
	input.Definition.Dimensions["default"] = derivedUnknownUncertaintyDimensions()
	input.Definition.Displays[0].PhysicalBoundary = Region{Status: "unknown", Shape: "unknown"}
	input.Definition.Displays[0].ActiveArea = Region{Status: "unknown", Shape: "unknown"}
	input.Definition.Displays[0].AppSafeArea = Region{Status: "unknown", Shape: "unknown"}
	parent, err := service.SaveCandidate(ctx, input, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyQualificationReview(ctx, parent.DeviceID, parent.Revision, parent.SHA256, reviewForCandidate(parent), parent.Revision); !errors.Is(err, ErrInvalidCandidate) {
		t.Fatalf("physical geometry review accepted unknown dimensions/display regions: %v", err)
	}
}

func TestLease25LaptopVisualSetupUsesCapturedCandidatePresetAndStaticScope(t *testing.T) {
	readbacksPath := "../../../docs/internal/goal/evidence/E5/qualification-review/live-laptop-1/public-readbacks.json"
	requestPath := "../../../docs/internal/goal/evidence/E5/qualification-review/identity-still-motion/review-request-lease25-static.draft.json"
	presetPath := "../../../docs/internal/goal/evidence/E4/family-native-review/native-runs/lease25-laptop-screen-vflip-1/preset.json"
	readbacksData, err := os.ReadFile(readbacksPath)
	if err != nil {
		t.Fatal(err)
	}
	var readbacks map[string]any
	if err := json.Unmarshal(readbacksData, &readbacks); err != nil {
		t.Fatal(err)
	}
	deviceRow := readbacks["device_candidate"].(map[string]any)
	candidateData, err := json.Marshal(snakeJSONKeys(deviceRow["candidate"]))
	if err != nil {
		t.Fatal(err)
	}
	var candidate DeviceCandidate
	if err := json.Unmarshal(candidateData, &candidate); err != nil {
		t.Fatal(err)
	}
	requestData, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var reviewJSON any
	if err := json.Unmarshal(requestData, &reviewJSON); err != nil {
		t.Fatal(err)
	}
	reviewData, err := json.Marshal(snakeJSONKeys(reviewJSON))
	if err != nil {
		t.Fatal(err)
	}
	var review QualificationReview
	if err := json.Unmarshal(reviewData, &review); err != nil {
		t.Fatal(err)
	}
	if candidate.Definition.ID != "jack-baeten-macbook-pro-m3-16-screen-vflip-candidate-v1" || review.CandidateRevision != 1 || review.CandidateSHA256 != deviceRow["sha256"] || deviceRow["revision"] != float64(1) {
		t.Fatalf("captured laptop candidate/request identity drifted: candidate=%s@%v request=%s@%d", candidate.Definition.ID, deviceRow["revision"], review.CandidateSHA256, review.CandidateRevision)
	}
	presetData, err := os.ReadFile(presetPath)
	if err != nil {
		t.Fatal(err)
	}
	var preset struct {
		PresetID string `json:"presetId"`
		Revision uint32 `json:"revision"`
		SHA256   string `json:"sha256"`
		Setup    struct {
			Device struct {
				ID       string `json:"id"`
				Revision uint32 `json:"revision"`
				SHA256   string `json:"sha256"`
			} `json:"device"`
			Scene struct {
				ID       string `json:"id"`
				Revision uint32 `json:"revision"`
				SHA256   string `json:"sha256"`
			} `json:"scene"`
			Motion struct {
				ID       string `json:"id"`
				Revision uint32 `json:"revision"`
				SHA256   string `json:"sha256"`
			} `json:"motion"`
		} `json:"setup"`
	}
	if err := json.Unmarshal(presetData, &preset); err != nil {
		t.Fatal(err)
	}
	if preset.Setup.Device.ID != candidate.Definition.ID || preset.Setup.Device.SHA256 != review.CandidateSHA256 || preset.Setup.Device.Revision != review.CandidateRevision || preset.PresetID != review.PresetID || preset.Revision != review.PresetRevision || preset.SHA256 != review.PresetSHA256 || preset.Setup.Scene.ID != review.SceneID || preset.Setup.Scene.SHA256 != review.SceneSHA256 || preset.Setup.Motion.ID != review.MotionID || preset.Setup.Motion.SHA256 != review.MotionSHA256 {
		t.Fatal("review request does not bind the retained lease-25 laptop preset")
	}
	if err := validateQualificationScope(candidate, review); err != nil {
		t.Fatalf("exact visual setup should accept the captured source geometry, unknown regions, and candidate-only support: %v", err)
	}
	ctx, _, service := setupDeviceRevisions(t)
	stored, err := service.SaveCandidate(ctx, candidate, 0)
	if err != nil || stored.SHA256 != review.CandidateSHA256 {
		t.Fatalf("captured candidate must retain its exact immutable digest: stored=%s expected=%s err=%v", stored.SHA256, review.CandidateSHA256, err)
	}
	qualified, err := service.ApplyQualificationReview(ctx, stored.DeviceID, stored.Revision, stored.SHA256, review, stored.Revision)
	if err != nil {
		t.Fatalf("full captured visual-setup review transition: %v", err)
	}
	parent, err := service.GetRevision(ctx, stored.DeviceID, stored.Revision)
	if err != nil || parent.SHA256 != stored.SHA256 || parent.State != DeviceRevisionCandidate {
		t.Fatalf("visual review must preserve captured candidate parent: %+v err=%v", parent, err)
	}
	if qualified.State != DeviceRevisionQualified || qualified.Revision != stored.Revision+1 || qualified.Candidate.Definition.QualificationReviews[0].AllowedMotionIDs[0] != "static" {
		t.Fatalf("review did not append exact static visual scope: %+v", qualified)
	}
	if review.AllowedMotionIDs[0] != "static" || review.NativeRenderJobID != "8bce7833-e6ca-45c1-bec3-cca1648b7253" || review.NativeOutputSHA256 != "a6672148d4f47b9ece17cfda02ad7e06d77ccfe2eacb077be930be6f7c43517f" {
		t.Fatal("captured visual review must bind the static scope and exact successful render")
	}
	physicalReview := review
	physicalReview.Scope = "physical_geometry"
	if err := validateQualificationScope(candidate, physicalReview); err == nil {
		t.Fatal("physical_geometry must still reject the candidate's unknown dimensions and display regions")
	}
	wrongMotion := review
	wrongMotion.AllowedMotionIDs = []string{"rigid_body"}
	if err := validateQualificationScope(candidate, wrongMotion); err == nil {
		t.Fatal("visual_setup must not claim generic rigid-body capability")
	}
}

func snakeJSONKeys(value any) any {
	switch current := value.(type) {
	case map[string]any:
		converted := make(map[string]any, len(current))
		for key, item := range current {
			var name strings.Builder
			for _, r := range key {
				if r >= 'A' && r <= 'Z' {
					name.WriteByte('_')
					name.WriteRune(r + ('a' - 'A'))
				} else {
					name.WriteRune(r)
				}
			}
			if name.String() == "size_bytes" {
				if encoded, ok := item.(string); ok {
					parsed, err := strconv.ParseInt(encoded, 10, 64)
					if err != nil {
						return value
					}
					item = parsed
				}
			}
			converted[name.String()] = snakeJSONKeys(item)
		}
		return converted
	case []any:
		converted := make([]any, len(current))
		for i, item := range current {
			converted[i] = snakeJSONKeys(item)
		}
		return converted
	default:
		return value
	}
}

func derivedUnknownUncertaintyDimensions() Dimensions {
	measurement := func(value float64, axis string) Measurement {
		return Measurement{Value: &value, Unit: "mm", Meaning: axis + " span from source geometry", Configuration: "default", Method: "derived", Uncertainty: Uncertainty{Status: "unknown"}, Evidence: []Evidence{{Source: "fixture://source-glb", Kind: "derived_asset_geometry"}}}
	}
	return Dimensions{Width: measurement(70, "width"), Height: measurement(140, "height"), Depth: measurement(8, "depth")}
}

func reviewForCandidate(parent DeviceRevision) QualificationReview {
	return QualificationReview{Scope: "physical_geometry", Result: "passed", Reviewer: "operator-1", Evidence: []Evidence{{Source: "native-job", Kind: "human_review"}}, CandidateRevision: parent.Revision, CandidateSHA256: parent.SHA256,
		SourceBundleID: parent.Candidate.Geometry.BundleID, SourceBundleRevision: parent.Candidate.Geometry.BundleRevision, SourceManifestSHA256: parent.Candidate.Geometry.ManifestSHA256, SourceReviewSHA256: parent.Candidate.Geometry.ReviewSHA256, SourceArtifactID: parent.Candidate.Geometry.ArtifactID, SourceArtifactSHA256: parent.Candidate.Geometry.SHA256, SourceAllowedUses: []string{"render"},
		DimensionStates: []string{"default"}, DisplayIDs: []string{"front"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"none"}, PresetID: "preset", PresetRevision: 1, PresetSHA256: strings.Repeat("d", 64), SceneID: "scene", SceneRevision: 1, SceneSHA256: strings.Repeat("e", 64), MotionID: "motion", MotionRevision: 1, MotionSHA256: strings.Repeat("f", 64), NativeRenderJobID: "job", NativeOutputSHA256: strings.Repeat("1", 64), NativeOutputKind: "physical_still_png", RightsClaimsAcknowledged: true, ReviewedAt: "2026-10-21T00:00:00Z"}
}

func TestDeviceCandidateUsesLibraryStableIdentifierContract(t *testing.T) {
	input := candidateFixture()
	input.Geometry.BundleID = strings.Repeat("B", 70)
	input.Geometry.ArtifactID = "Geometry-Original_01"
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("library-valid uppercase/long reference ID was rejected: %v", err)
	}
}

func TestDeviceCandidateCanonicalizationDoesNotMutateInputAndIsIdempotent(t *testing.T) {
	input := candidateFixture()
	input.Parts[0].LocalFrame.Rotation = physical.QuatXYZW{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4}
	original := input.Parts[0].LocalFrame.Rotation
	first, _, firstDigest, err := CanonicalizeDeviceCandidate(input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Parts[0].LocalFrame.Rotation != original {
		t.Fatal("canonicalization mutated caller part transform")
	}
	second, _, secondDigest, err := CanonicalizeDeviceCandidate(first)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest || first.Parts[0].LocalFrame.Rotation != second.Parts[0].LocalFrame.Rotation {
		t.Fatalf("canonicalization is not idempotent: %s != %s", firstDigest, secondDigest)
	}
}

func TestDeviceCandidateRejectsRevisionOverflow(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	if _, err := service.SaveCandidate(ctx, candidateFixture(), ^uint32(0)); err == nil {
		t.Fatal("expected exhausted revision number to be rejected")
	}
}

func TestDeviceCandidateConcurrentIdenticalWritersConverge(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	other, err := NewDeviceRevisionService(service.repo, service.now)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan DeviceRevision, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, writer := range []*DeviceRevisionService{service, other} {
		wg.Add(1)
		go func(writer *DeviceRevisionService) {
			defer wg.Done()
			<-start
			result, err := writer.SaveCandidate(ctx, candidateFixture(), 0)
			results <- result
			errs <- err
		}(writer)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("identical concurrent writer failed to converge: %v", err)
		}
	}
	for result := range results {
		if result.Revision != 1 {
			t.Fatalf("concurrent replay created unexpected revision %d", result.Revision)
		}
	}
}

func TestDeviceCandidateStoredDigestAndCanonicalJSONAreVerified(t *testing.T) {
	ctx, db, service := setupDeviceRevisions(t)
	created, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TRIGGER device_revisions_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET sha256=? WHERE device_id=? AND revision=1`, strings.Repeat("d", 64), created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, created.DeviceID, 1); err == nil {
		t.Fatal("mismatched stored digest passed integrity verification")
	}
	if _, err := db.Exec(`UPDATE device_revisions SET sha256=? WHERE device_id=? AND revision=1`, created.SHA256, created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET candidate_json=candidate_json || ' ' WHERE device_id=? AND revision=1`, created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, created.DeviceID, 1); err == nil {
		t.Fatal("noncanonical stored bytes passed integrity verification")
	}
}

func TestDeviceCandidateWrapperPreservesLegacyDefinitionJSON(t *testing.T) {
	definition := validFixture()
	before, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	input := candidateFixture()
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(input.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("candidate wrapper changed the legacy Definition JSON representation")
	}
}

func TestDisplayBindingScreenSelectorPairIsOptionalButAtomic(t *testing.T) {
	input := candidateFixture()
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("legacy display binding without a selector must remain valid: %v", err)
	}
	primitive, material := uint32(0), uint32(0)
	input.DisplayBindings[0].ScreenPrimitiveIndex = &primitive
	input.DisplayBindings[0].ScreenMaterialIndex = &material
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("explicit primitive/material zeroes must remain distinguishable from omitted selectors: %v", err)
	}
	input.DisplayBindings[0].ScreenMaterialIndex = nil
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err == nil || !strings.Contains(err.Error(), "must pin both") {
		t.Fatalf("partial screen selector pair was accepted: %v", err)
	}
}

func TestDeviceCandidateRejectsInvalidReferenceGraphsAndTransforms(t *testing.T) {
	tests := map[string]func(*DeviceCandidate){
		"duplicate part": func(c *DeviceCandidate) { c.Parts = append(c.Parts, c.Parts[0]) },
		"parent cycle": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "screen", Role: "display", ParentID: "body", NodeIndex: 1, NodeName: "screen", LocalFrame: identityTRS()})
			c.Parts[0].ParentID = "screen"
		},
		"unknown display": func(c *DeviceCandidate) { c.DisplayBindings[0].DisplayID = "missing" },
		"unknown joint part": func(c *DeviceCandidate) {
			c.Definition.Joints = []Joint{{ID: "hinge", ParentPart: "body", ChildPart: "missing"}}
		},
		"joint cycle": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "lid", Role: "display", ParentID: "body", NodeIndex: 1, NodeName: "lid", LocalFrame: identityTRS()})
			c.Definition.Joints = []Joint{{ID: "hinge-a", ParentPart: "body", ChildPart: "lid"}, {ID: "hinge-b", ParentPart: "lid", ChildPart: "body"}}
		},
		"joint contradicts part hierarchy": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts,
				SemanticPart{ID: "base", Role: "base", NodeIndex: 1, NodeName: "base", LocalFrame: identityTRS()},
				SemanticPart{ID: "lid", Role: "lid", ParentID: "base", NodeIndex: 2, NodeName: "lid", LocalFrame: identityTRS()},
			)
			c.Definition.Joints = []Joint{{ID: "hinge", ParentPart: "body", ChildPart: "lid"}}
		},
		"multiple joints for child": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "lid", Role: "lid", ParentID: "body", NodeIndex: 1, NodeName: "lid", LocalFrame: identityTRS()})
			c.Definition.Joints = []Joint{{ID: "hinge-x", ParentPart: "body", ChildPart: "lid"}, {ID: "hinge-y", ParentPart: "body", ChildPart: "lid"}}
		},
		"nonfinite transform":   func(c *DeviceCandidate) { c.Parts[0].LocalFrame.Translation.X = math.Inf(1) },
		"claimed qualification": func(c *DeviceCandidate) { c.Definition.Qualification = "qualified" },
		"claimed qualification review": func(c *DeviceCandidate) {
			c.Definition.QualificationReviews = []QualificationReview{{Scope: "physical_geometry", Result: "passed", Reviewer: "caller"}}
		},
		"wrong geometry role": func(c *DeviceCandidate) { c.Geometry.Role = library.RoleTextureImage },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			input := candidateFixture()
			mutate(&input)
			if _, _, _, err := CanonicalizeDeviceCandidate(input); err == nil {
				t.Fatal("invalid candidate was accepted")
			}
		})
	}
}
