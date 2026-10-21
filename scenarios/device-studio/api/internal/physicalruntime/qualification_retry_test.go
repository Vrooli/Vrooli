package physicalruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/renderjobs"
	shotdomain "device-studio/internal/shots"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

type qualificationDeviceStub struct {
	parent, head devices.DeviceRevision
	applies      int
}

func (s *qualificationDeviceStub) GetRevision(context.Context, string, uint32) (devices.DeviceRevision, error) {
	return s.parent, nil
}
func (s *qualificationDeviceStub) GetHead(context.Context, string) (devices.DeviceRevision, error) {
	return s.head, nil
}
func (s *qualificationDeviceStub) ApplyQualificationReview(_ context.Context, id string, rev uint32, sha string, review devices.QualificationReview, expected uint32) (devices.DeviceRevision, error) {
	s.applies++
	if expected != s.parent.Revision {
		return devices.DeviceRevision{}, devices.ErrRevisionConflict
	}
	s.head = s.parent
	s.head.Revision++
	s.head.State = devices.DeviceRevisionQualified
	s.head.Candidate.Definition.Qualification = "qualified"
	s.head.Candidate.Definition.QualificationReviews = []devices.QualificationReview{review}
	s.head.Candidate.Geometry = s.parent.Candidate.Geometry
	return s.head, nil
}

type qualificationPresetStub struct {
	original, head shotdomain.PhysicalPresetRevision
	failOnce       bool
	saves          int
}

func (s *qualificationPresetStub) Get(context.Context, string, uint32) (shotdomain.PhysicalPresetRevision, error) {
	return s.original, nil
}
func (s *qualificationPresetStub) Head(context.Context, string) (shotdomain.PhysicalPresetRevision, error) {
	return s.head, nil
}
func (s *qualificationPresetStub) Save(_ context.Context, id string, expected uint32, name string, setup *shotsv1.PhysicalShotSetup) (shotdomain.PhysicalPresetRevision, error) {
	s.saves++
	if s.failOnce {
		s.failOnce = false
		return shotdomain.PhysicalPresetRevision{}, errors.New("injected storage failure")
	}
	if expected != s.head.Revision {
		return shotdomain.PhysicalPresetRevision{}, shotdomain.ErrPhysicalPresetConflict
	}
	s.head = s.original
	s.head.Revision++
	s.head.SHA256 = "b" + s.head.SHA256[1:]
	s.head.Setup = proto.Clone(setup).(*shotsv1.PhysicalShotSetup)
	return s.head, nil
}

type qualificationBundleStub struct{ revision library.BundleRevision }

func (s qualificationBundleStub) GetRevision(context.Context, string, uint32) (library.BundleRevision, error) {
	return s.revision, nil
}

type qualificationMotionStub struct{ revision motions.Revision }

func (s qualificationMotionStub) GetRevision(context.Context, string, uint32) (motions.Revision, error) {
	return s.revision, nil
}

type qualificationValidatorStub struct{}

func (qualificationValidatorStub) ValidatePhysicalSetup(context.Context, *shotsv1.PhysicalShotSetup) error {
	return nil
}

type qualificationRenderStub struct {
	evidence renderjobs.QualificationRenderEvidence
}

func (s qualificationRenderStub) VerifyQualificationRender(context.Context, string, string) (renderjobs.QualificationRenderEvidence, error) {
	return s.evidence, nil
}

func TestQualificationReviewRetryResumesPresetAfterExactImmutableAppend(t *testing.T) {
	ctx := context.Background()
	source := physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ManifestSHA256: repeated('a'), ReviewSHA256: repeated('b'), ArtifactID: "glb", Role: library.RoleGeometryGLB, SizeBytes: 10, SHA256: repeated('c')}
	parent := devices.DeviceRevision{DeviceID: "device", Revision: 2, SHA256: repeated('d'), State: devices.DeviceRevisionCandidate, Candidate: devices.DeviceCandidate{Geometry: source, Definition: devices.Definition{ID: "device", Qualification: "unqualified"}}}
	identity := &physicalv1.TRS{TranslationM: &physicalv1.Vec3{}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}}
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: "device", Revision: 2, Sha256: parent.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: "scene", Revision: 3, Sha256: repeated('e')}, Motion: &shotsv1.ComponentRevisionRef{Id: "motion", Revision: 4, Sha256: repeated('f')}, DisplayId: "display", DeviceSupportId: "base", SceneSupportId: "table", CameraId: "camera", SceneSupportWorld: proto.Clone(identity).(*physicalv1.TRS), DeviceRootRest: proto.Clone(identity).(*physicalv1.TRS), SupportPivotM: &physicalv1.Vec3{}}
	preset := shotdomain.PhysicalPresetRevision{PresetID: "preset", Revision: 5, SHA256: repeated('9'), Name: "review", Setup: setup}
	review := devices.QualificationReview{Scope: "physical_geometry", Result: "passed", Reviewer: "operator", Evidence: []devices.Evidence{{Source: "job", Kind: "physical_still_png"}}, CandidateRevision: 2, CandidateSHA256: parent.SHA256, SourceBundleID: source.BundleID, SourceBundleRevision: source.BundleRevision, SourceManifestSHA256: source.ManifestSHA256, SourceReviewSHA256: source.ReviewSHA256, SourceArtifactID: source.ArtifactID, SourceArtifactSHA256: source.SHA256, SourceAllowedUses: []string{"render"}, DimensionStates: []string{"default"}, DisplayIDs: []string{"display"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"base"}, PresetID: preset.PresetID, PresetRevision: preset.Revision, PresetSHA256: preset.SHA256, SceneID: "scene", SceneRevision: 3, SceneSHA256: repeated('e'), MotionID: "motion", MotionRevision: 4, MotionSHA256: repeated('f'), NativeRenderJobID: "job", NativeOutputSHA256: repeated('8'), NativeOutputKind: "physical_still_png", RightsClaimsAcknowledged: true, ReviewedAt: "2026-10-21T00:00:00Z"}
	deviceOwner := &qualificationDeviceStub{parent: parent, head: parent}
	presetOwner := &qualificationPresetStub{original: preset, head: preset, failOnce: true}
	bundle := library.BundleRevision{Manifest: library.BundleManifest{BundleID: source.BundleID, Revision: 1, Artifacts: []library.Artifact{{ID: source.ArtifactID, Role: source.Role, SizeBytes: source.SizeBytes, SHA256: source.SHA256, Rights: []library.Use{library.UseRender}}}}, ManifestSHA256: source.ManifestSHA256, Review: library.ReviewBinding{Decision: "approved", ManifestSHA256: source.ManifestSHA256, AllowedUses: []library.Use{library.UseRender}}, ReviewSHA256: source.ReviewSHA256}
	render := qualificationRenderStub{evidence: renderjobs.QualificationRenderEvidence{DeviceID: "device", DeviceRevision: 2, DeviceSHA256: parent.SHA256, SceneID: "scene", SceneRevision: 3, SceneSHA256: repeated('e'), MotionID: "motion", MotionRevision: 4, MotionSHA256: repeated('f'), DisplayID: "display", DeviceSupportID: "base", SceneSupportID: "table", CameraID: "camera", OutputSHA256: review.NativeOutputSHA256, OutputKind: review.NativeOutputKind, DeviceRootRest: physical.TRS{Translation: physical.Vec3{}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, SceneSupportWorld: physical.TRS{Translation: physical.Vec3{}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}}}
	service := QualificationService{devices: deviceOwner, bundles: qualificationBundleStub{bundle}, motions: qualificationMotionStub{motions.Revision{SHA256: review.MotionSHA256, State: motions.RevisionCandidate}}, presets: presetOwner, validator: qualificationValidatorStub{}, renders: render}
	request := devices.QualificationReviewRequest{ProjectID: "project", DeviceID: parent.DeviceID, CandidateRevision: parent.Revision, CandidateSHA256: parent.SHA256, ExpectedHead: parent.Revision, Review: review}
	if _, err := service.ReviewCandidate(ctx, request); err == nil {
		t.Fatal("expected the first preset save to fail")
	}
	result, err := service.ReviewCandidate(ctx, request)
	if err != nil {
		t.Fatalf("exact retry should complete the existing qualification and refresh the preset: %v", err)
	}
	if deviceOwner.applies != 1 || presetOwner.saves != 2 || result.Revision.Revision != 3 || result.PresetRevision != 6 {
		t.Fatalf("retry duplicated or failed to complete the immutable operation: device applies=%d preset saves=%d result=%+v", deviceOwner.applies, presetOwner.saves, result)
	}
}

func repeated(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func TestQualificationReplayComparisonIncludesScope(t *testing.T) {
	visual := devices.QualificationReview{Scope: "visual_setup", Result: "passed", Reviewer: "operator", Evidence: []devices.Evidence{{Source: "job", Kind: "physical_still_png"}}}
	if !sameQualificationReview(visual, visual) {
		t.Fatal("exact visual_setup review must compare equal for retry")
	}
	physical := visual
	physical.Scope = "physical_geometry"
	if sameQualificationReview(visual, physical) {
		t.Fatal("visual_setup replay must not match a physical_geometry review")
	}
}

func TestLease25LaptopMotionEvidenceIsIdentityAcrossEveryKey(t *testing.T) {
	readbacksPath := "../../../docs/internal/goal/evidence/E5/qualification-review/live-laptop-1/public-readbacks.json"
	requestPath := "../../../docs/internal/goal/evidence/E5/qualification-review/identity-still-motion/review-request-lease25-static.draft.json"
	data, err := os.ReadFile(readbacksPath)
	if err != nil {
		t.Fatal(err)
	}
	var readbacks map[string]any
	if err := json.Unmarshal(data, &readbacks); err != nil {
		t.Fatal(err)
	}
	motionRow := readbacks["motion"].(map[string]any)
	motionData, err := json.Marshal(qualificationSnakeJSONKeys(motionRow["candidate"]))
	if err != nil {
		t.Fatal(err)
	}
	var motion motions.MotionCandidate
	if err := json.Unmarshal(motionData, &motion); err != nil {
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
	reviewData, err := json.Marshal(qualificationSnakeJSONKeys(reviewJSON))
	if err != nil {
		t.Fatal(err)
	}
	var review devices.QualificationReview
	if err := json.Unmarshal(reviewData, &review); err != nil {
		t.Fatal(err)
	}
	if review.MotionID != motion.MotionID || review.MotionSHA256 != motionRow["sha256"] || review.NativeRenderJobID != "8bce7833-e6ca-45c1-bec3-cca1648b7253" {
		t.Fatal("retained review request does not bind the captured laptop motion and native still")
	}
	if !constantIdentityMotion(motion) {
		t.Fatal("captured phone-still-identity laptop motion should be constant identity")
	}
	moving := motion
	moving.Tracks = append([]motions.MotionTrack(nil), motion.Tracks...)
	last := len(moving.Tracks) - 1
	moving.Tracks[last].Keys = append([]motions.MotionKey(nil), motion.Tracks[last].Keys...)
	if moving.Tracks[last].Keys[len(moving.Tracks[last].Keys)-1].Transform != nil {
		key := moving.Tracks[last].Keys[len(moving.Tracks[last].Keys)-1]
		transform := *key.Transform
		transform.Translation.X = 0.001
		key.Transform = &transform
		moving.Tracks[last].Keys[len(moving.Tracks[last].Keys)-1] = key
	} else {
		angle := 0.01
		moving.Tracks[last].Keys[len(moving.Tracks[last].Keys)-1].AngleRad = &angle
	}
	if constantIdentityMotion(moving) {
		t.Fatal("non-identity end key must not pass based on its identity first frame")
	}
}

func qualificationSnakeJSONKeys(value any) any {
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
			if name.String() == "duration_ticks" || name.String() == "tick" || name.String() == "num" || name.String() == "den" {
				if encoded, ok := item.(string); ok {
					parsed, err := strconv.ParseInt(encoded, 10, 64)
					if err != nil {
						return value
					}
					item = parsed
				}
			}
			converted[name.String()] = qualificationSnakeJSONKeys(item)
		}
		return converted
	case []any:
		converted := make([]any, len(current))
		for i, item := range current {
			converted[i] = qualificationSnakeJSONKeys(item)
		}
		return converted
	default:
		return value
	}
}
