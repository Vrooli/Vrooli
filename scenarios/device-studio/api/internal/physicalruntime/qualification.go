package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/renderjobs"
	"device-studio/internal/scenes"
	shotdomain "device-studio/internal/shots"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

type qualificationRenderReader interface {
	VerifyQualificationRender(context.Context, string, string) (renderjobs.QualificationRenderEvidence, error)
}

type qualificationDeviceOwner interface {
	GetRevision(context.Context, string, uint32) (devices.DeviceRevision, error)
	GetHead(context.Context, string) (devices.DeviceRevision, error)
	ApplyQualificationReview(context.Context, string, uint32, string, devices.QualificationReview, uint32) (devices.DeviceRevision, error)
}
type qualificationPresetOwner interface {
	Get(context.Context, string, uint32) (shotdomain.PhysicalPresetRevision, error)
	Head(context.Context, string) (shotdomain.PhysicalPresetRevision, error)
	Save(context.Context, string, uint32, string, *shotsv1.PhysicalShotSetup) (shotdomain.PhysicalPresetRevision, error)
}
type qualificationBundleReader interface {
	GetRevision(context.Context, string, uint32) (library.BundleRevision, error)
}
type qualificationMotionReader interface {
	GetRevision(context.Context, string, uint32) (motions.Revision, error)
}
type qualificationSceneReader interface {
	GetRevision(context.Context, string, uint32) (scenes.Revision, error)
}

type QualificationService struct {
	devices   qualificationDeviceOwner
	bundles   qualificationBundleReader
	motions   qualificationMotionReader
	scenes    qualificationSceneReader
	presets   qualificationPresetOwner
	validator shotdomain.PhysicalPresetValidator
	renders   qualificationRenderReader
}

func NewQualificationService(owners Owners, renders qualificationRenderReader) (*QualificationService, error) {
	if owners.Devices == nil || owners.Scenes == nil || owners.Motions == nil || owners.Presets == nil || owners.Shots == nil || renders == nil {
		return nil, errors.New("qualification requires device, scene, motion, library, physical setup, preset, and render owners")
	}
	validator, ok := owners.Shots.(shotdomain.PhysicalPresetValidator)
	if !ok {
		return nil, errors.New("qualification requires physical setup validation")
	}
	return &QualificationService{devices: owners.Devices, bundles: owners.LibraryCatalog, scenes: owners.Scenes, motions: owners.Motions, presets: owners.Presets, validator: validator, renders: renders}, nil
}

func (s *QualificationService) ReviewCandidate(ctx context.Context, req devices.QualificationReviewRequest) (devices.QualificationReviewResult, error) {
	if req.ProjectID == "" || req.DeviceID == "" {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: project and device IDs are required", devices.ErrInvalidCandidate)
	}
	proof, err := s.renders.VerifyQualificationRender(ctx, req.ProjectID, req.Review.NativeRenderJobID)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	return s.reviewCandidateWithEvidence(ctx, req, proof)
}

func (s *QualificationService) reviewCandidateWithEvidence(ctx context.Context, req devices.QualificationReviewRequest, proof renderjobs.QualificationRenderEvidence) (devices.QualificationReviewResult, error) {
	if req.ProjectID == "" || req.DeviceID == "" {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: project and device IDs are required", devices.ErrInvalidCandidate)
	}
	r := req.Review
	candidate, err := s.devices.GetRevision(ctx, req.DeviceID, req.CandidateRevision)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if candidate.SHA256 != req.CandidateSHA256 || candidate.State != devices.DeviceRevisionCandidate || candidate.Candidate.Definition.Qualification != "unqualified" {
		return devices.QualificationReviewResult{}, devices.ErrRevisionConflict
	}
	if r.CandidateRevision != candidate.Revision || r.CandidateSHA256 != candidate.SHA256 || !sameSourceRef(r, candidate.Candidate.Geometry) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: review does not name the exact candidate and source artifact", devices.ErrInvalidCandidate)
	}
	if r.Result == "rejected" {
		revision, saveErr := s.devices.ApplyQualificationReview(ctx, req.DeviceID, candidate.Revision, candidate.SHA256, r, req.ExpectedHead)
		return devices.QualificationReviewResult{Revision: revision}, saveErr
	}
	if r.Result != "passed" {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: decision must be passed or rejected", devices.ErrInvalidCandidate)
	}
	bundle, err := s.bundles.GetRevision(ctx, candidate.Candidate.Geometry.BundleID, candidate.Candidate.Geometry.BundleRevision)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if bundle.ManifestSHA256 != candidate.Candidate.Geometry.ManifestSHA256 || bundle.ReviewSHA256 != candidate.Candidate.Geometry.ReviewSHA256 || bundle.Review.Decision != "approved" || bundle.Review.ManifestSHA256 != bundle.ManifestSHA256 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: source review binding is stale or unapproved", devices.ErrInvalidCandidate)
	}
	var source *library.Artifact
	for i := range bundle.Manifest.Artifacts {
		if bundle.Manifest.Artifacts[i].ID == candidate.Candidate.Geometry.ArtifactID {
			source = &bundle.Manifest.Artifacts[i]
			break
		}
	}
	if source == nil || source.SHA256 != candidate.Candidate.Geometry.SHA256 || source.Role != candidate.Candidate.Geometry.Role || source.SizeBytes != candidate.Candidate.Geometry.SizeBytes || !hasLibraryUse(bundle.Review.AllowedUses, library.UseRender) || !hasLibraryUse(source.Rights, library.UseRender) || !sameStringSet(r.SourceAllowedUses, libraryUses(bundle.Review.AllowedUses)) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: source artifact/use rights do not match the reviewed Library revision", devices.ErrInvalidCandidate)
	}
	preset, err := s.presets.Get(ctx, r.PresetID, r.PresetRevision)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if preset.SHA256 != r.PresetSHA256 || preset.Setup == nil || preset.Setup.Device == nil || preset.Setup.Scene == nil || preset.Setup.Motion == nil ||
		preset.Setup.Device.Id != candidate.DeviceID || preset.Setup.Device.Revision != candidate.Revision || preset.Setup.Device.Sha256 != candidate.SHA256 ||
		preset.Setup.Scene.Id != r.SceneID || preset.Setup.Scene.Revision != r.SceneRevision || preset.Setup.Scene.Sha256 != r.SceneSHA256 ||
		preset.Setup.Motion.Id != r.MotionID || preset.Setup.Motion.Revision != r.MotionRevision || preset.Setup.Motion.Sha256 != r.MotionSHA256 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: saved preset does not bind the reviewed device, scene, and motion revisions", devices.ErrInvalidCandidate)
	}
	if err := s.validator.ValidatePhysicalSetup(ctx, preset.Setup); err != nil {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: selected preset is not physically resolvable: %v", devices.ErrInvalidCandidate, err)
	}
	motionRevision, err := s.motions.GetRevision(ctx, r.MotionID, r.MotionRevision)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if motionRevision.SHA256 != r.MotionSHA256 || motionRevision.State != motions.RevisionCandidate {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: exact motion candidate is unavailable or changed", devices.ErrInvalidCandidate)
	}
	if r.Scope == "visual_setup" {
		if !constantIdentityMotion(motionRevision.Candidate) || !sameStringSet(r.AllowedMotionIDs, []string{"static"}) {
			return devices.QualificationReviewResult{}, fmt.Errorf("%w: visual setup requires a constant identity motion and static-only scope", devices.ErrInvalidCandidate)
		}
	} else {
		motionScope := motionCapabilityScope(motionRevision.Candidate)
		if len(motionScope) != len(r.AllowedMotionIDs) || !sameStringSet(motionScope, r.AllowedMotionIDs) {
			return devices.QualificationReviewResult{}, fmt.Errorf("%w: allowed motion scope must match the exact motion job's target capabilities", devices.ErrInvalidCandidate)
		}
	}
	deviceRest, transformErr := qualificationTRS(preset.Setup.DeviceRootRest)
	if transformErr != nil {
		return devices.QualificationReviewResult{}, transformErr
	}
	sceneWorld, transformErr := qualificationTRS(preset.Setup.SceneSupportWorld)
	if transformErr != nil {
		return devices.QualificationReviewResult{}, transformErr
	}
	pivot := preset.Setup.SupportPivotM
	if pivot == nil || proof.DeviceID != candidate.DeviceID || proof.DeviceRevision != candidate.Revision || proof.DeviceSHA256 != candidate.SHA256 || proof.SceneID != r.SceneID || proof.SceneRevision != r.SceneRevision || proof.SceneSHA256 != r.SceneSHA256 || proof.MotionID != r.MotionID || proof.MotionRevision != r.MotionRevision || proof.MotionSHA256 != r.MotionSHA256 || proof.DisplayID != preset.Setup.DisplayId || proof.DeviceSupportID != preset.Setup.DeviceSupportId || proof.SceneSupportID != preset.Setup.SceneSupportId || proof.CameraID != preset.Setup.CameraId || proof.OutputSHA256 != r.NativeOutputSHA256 || proof.OutputKind != r.NativeOutputKind || proof.DeviceRootRest != deviceRest || proof.SceneSupportWorld != sceneWorld || proof.SupportPivot != (physical.Vec3{X: pivot.X, Y: pivot.Y, Z: pivot.Z}) || !containsScope(r.DisplayIDs, preset.Setup.DisplayId) || !containsScope(r.SupportIDs, preset.Setup.DeviceSupportId) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: completed native render does not match the exact reviewed preset scope/output", devices.ErrInvalidCandidate)
	}
	if len(r.DimensionStates) != 1 || len(r.DisplayIDs) != 1 || len(r.SupportIDs) != 1 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: qualification scope must name only the exact rendered dimension, display, motion, and support", devices.ErrInvalidCandidate)
	}
	if r.Scope != "visual_setup" && r.Scope != "physical_geometry" {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: review scope must be visual_setup or physical_geometry", devices.ErrInvalidCandidate)
	}
	qualified, replayed, err := s.existingExactQualification(ctx, req, candidate, r)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if !replayed {
		qualified, err = s.devices.ApplyQualificationReview(ctx, req.DeviceID, candidate.Revision, candidate.SHA256, r, req.ExpectedHead)
		if err != nil {
			if !errors.Is(err, devices.ErrRevisionConflict) {
				return devices.QualificationReviewResult{}, err
			}
			applyErr := err
			qualified, replayed, err = s.existingExactQualification(ctx, req, candidate, r)
			if err != nil {
				return devices.QualificationReviewResult{}, err
			}
			if !replayed {
				return devices.QualificationReviewResult{}, applyErr
			}
		}
	}
	updatedSetup := proto.Clone(preset.Setup).(*shotsv1.PhysicalShotSetup)
	updatedSetup.Device.Revision = qualified.Revision
	updatedSetup.Device.Sha256 = qualified.SHA256
	if replayed {
		currentPreset, headErr := s.presets.Head(ctx, preset.PresetID)
		if headErr != nil {
			return devices.QualificationReviewResult{}, headErr
		}
		if currentPreset.Setup != nil && currentPreset.Setup.Device != nil && currentPreset.Setup.Device.Revision == qualified.Revision && currentPreset.Setup.Device.Sha256 == qualified.SHA256 {
			if !proto.Equal(currentPreset.Setup, updatedSetup) {
				return devices.QualificationReviewResult{}, devices.ErrRevisionConflict
			}
			return devices.QualificationReviewResult{Revision: qualified, PresetID: currentPreset.PresetID, PresetRevision: currentPreset.Revision, PresetSHA256: currentPreset.SHA256}, nil
		}
		if currentPreset.Revision != preset.Revision || currentPreset.SHA256 != preset.SHA256 {
			return devices.QualificationReviewResult{}, devices.ErrRevisionConflict
		}
	}
	updatedPreset, err := s.presets.Save(ctx, preset.PresetID, preset.Revision, preset.Name, updatedSetup)
	if err != nil {
		return devices.QualificationReviewResult{}, fmt.Errorf("qualified revision persisted but preset refresh failed; retry exact review after checking the device head: %w", err)
	}
	return devices.QualificationReviewResult{Revision: qualified, PresetID: updatedPreset.PresetID, PresetRevision: updatedPreset.Revision, PresetSHA256: updatedPreset.SHA256}, nil
}

func constantIdentityMotion(candidate motions.MotionCandidate) bool {
	for _, track := range candidate.Tracks {
		if len(track.Keys) == 0 {
			return false
		}
		for _, key := range track.Keys {
			if track.Target.JointID != "" {
				if key.AngleRad == nil || *key.AngleRad != 0 {
					return false
				}
				continue
			}
			if key.Transform == nil || key.Transform.Translation != (physical.Vec3{}) || key.Transform.Rotation != (physical.QuatXYZW{W: 1}) {
				return false
			}
		}
	}
	return true
}

func (s *QualificationService) existingExactQualification(ctx context.Context, req devices.QualificationReviewRequest, candidate devices.DeviceRevision, review devices.QualificationReview) (devices.DeviceRevision, bool, error) {
	head, err := s.devices.GetHead(ctx, req.DeviceID)
	if err != nil {
		return devices.DeviceRevision{}, false, err
	}
	if head.State != devices.DeviceRevisionQualified || head.Revision <= candidate.Revision || len(head.Candidate.Definition.QualificationReviews) == 0 {
		return devices.DeviceRevision{}, false, nil
	}
	last := head.Candidate.Definition.QualificationReviews[len(head.Candidate.Definition.QualificationReviews)-1]
	if !sameQualificationReview(last, review) {
		return devices.DeviceRevision{}, false, nil
	}
	if last.CandidateRevision != candidate.Revision || last.CandidateSHA256 != candidate.SHA256 {
		return devices.DeviceRevision{}, false, devices.ErrRevisionConflict
	}
	return head, true, nil
}

func sameQualificationReview(left, right devices.QualificationReview) bool {
	left.SourceAllowedUses = sortedCopy(left.SourceAllowedUses)
	left.DimensionStates = sortedCopy(left.DimensionStates)
	left.DisplayIDs = sortedCopy(left.DisplayIDs)
	left.AllowedMotionIDs = sortedCopy(left.AllowedMotionIDs)
	left.SupportIDs = sortedCopy(left.SupportIDs)
	right.SourceAllowedUses = sortedCopy(right.SourceAllowedUses)
	right.DimensionStates = sortedCopy(right.DimensionStates)
	right.DisplayIDs = sortedCopy(right.DisplayIDs)
	right.AllowedMotionIDs = sortedCopy(right.AllowedMotionIDs)
	right.SupportIDs = sortedCopy(right.SupportIDs)
	return reflect.DeepEqual(left, right)
}

func sortedCopy(values []string) []string {
	copy := append([]string(nil), values...)
	sort.Strings(copy)
	return copy
}

func qualificationTRS(value *physicalv1.TRS) (physical.TRS, error) {
	if value == nil || value.TranslationM == nil || value.RotationXyzw == nil || value.Scale == nil {
		return physical.TRS{}, fmt.Errorf("%w: qualification preset transform is incomplete", devices.ErrInvalidCandidate)
	}
	return physical.CanonicalizeTRS(physical.TRS{
		Translation: physical.Vec3{X: value.TranslationM.X, Y: value.TranslationM.Y, Z: value.TranslationM.Z},
		Rotation:    physical.QuatXYZW{X: value.RotationXyzw.X, Y: value.RotationXyzw.Y, Z: value.RotationXyzw.Z, W: value.RotationXyzw.W},
		Scale:       physical.Vec3{X: value.Scale.X, Y: value.Scale.Y, Z: value.Scale.Z},
	})
}

func containsScope(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func motionCapabilityScope(candidate motions.MotionCandidate) []string {
	if len(candidate.Tracks) == 0 {
		return []string{"static"}
	}
	set := map[string]bool{}
	for _, track := range candidate.Tracks {
		switch {
		case track.Target.DeviceRoot:
			set["rigid_body"] = true
		case track.Target.JointID != "":
			set["articulated"] = true
		case track.Target.SceneCameraID != "":
			set["camera"] = true
		}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func sameSourceRef(r devices.QualificationReview, ref physical.ArtifactRef) bool {
	return r.SourceBundleID == ref.BundleID && r.SourceBundleRevision == ref.BundleRevision && r.SourceManifestSHA256 == ref.ManifestSHA256 && r.SourceReviewSHA256 == ref.ReviewSHA256 && r.SourceArtifactID == ref.ArtifactID && r.SourceArtifactSHA256 == ref.SHA256
}

func libraryUses(uses []library.Use) []string {
	result := make([]string, len(uses))
	for i, use := range uses {
		result[i] = string(use)
	}
	sort.Strings(result)
	return result
}
func hasLibraryUse(uses []library.Use, want library.Use) bool {
	for _, use := range uses {
		if use == want {
			return true
		}
	}
	return false
}
func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]bool, len(left))
	for _, value := range left {
		seen[value] = true
	}
	for _, value := range right {
		if !seen[value] {
			return false
		}
		delete(seen, value)
	}
	return len(seen) == 0
}
