package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/renderjobs"
	"device-studio/internal/scenes"
	shotdomain "device-studio/internal/shots"
	"google.golang.org/protobuf/proto"
)

const maxQualificationPackageBytes = 128 << 20

type qualificationPackage struct {
	Version         int                                    `json:"version"`
	PackageSHA      string                                 `json:"package_sha256"`
	Origin          qualificationPackageOrigin             `json:"origin"`
	Candidate       devices.DeviceRevision                 `json:"candidate"`
	SourceBundles   []library.BundleRevision               `json:"source_bundle_revisions"`
	SourceArtifacts []qualificationPackageArtifact         `json:"source_artifacts"`
	Scene           scenes.Revision                        `json:"scene_revision"`
	Motion          motions.Revision                       `json:"motion_revision"`
	Preset          shotdomain.PhysicalPresetRevision      `json:"preset_revision"`
	Review          devices.QualificationReview            `json:"review"`
	Render          renderjobs.QualificationRenderEvidence `json:"render_receipt"`
	Output          []byte                                 `json:"native_output_bytes"`
}

type qualificationPackageArtifact struct {
	Ref   physical.ArtifactRef `json:"ref"`
	Bytes []byte               `json:"bytes"`
}

type qualificationPackageOrigin struct {
	ProjectID string `json:"project_id"`
	JobID     string `json:"job_id"`
	Reviewed  string `json:"reviewed_at"`
}

type qualificationRenderExporter interface {
	ExportQualificationRender(context.Context, string, string) (renderjobs.QualificationRenderEvidence, []byte, error)
}

type qualificationArtifactOpener interface {
	OpenArtifact(context.Context, string, uint32, string, library.Use) (library.ResolvedArtifact, error)
}

// ExportReviewPackage packages an already-appended passed review while its
// source render job is still available. The package carries integrity digests,
// not a cryptographic reviewer signature.
func (s *QualificationService) ExportReviewPackage(ctx context.Context, projectID, deviceID string, qualifiedRevision uint32) ([]byte, error) {
	exporter, ok := s.renders.(qualificationRenderExporter)
	if !ok || s.bundles == nil || s.scenes == nil || s.motions == nil || s.presets == nil || s.devices == nil {
		return nil, errors.New("qualification package export requires render and Library artifact owners")
	}
	qualified, err := s.devices.GetRevision(ctx, deviceID, qualifiedRevision)
	if err != nil {
		return nil, err
	}
	if qualified.State != devices.DeviceRevisionQualified || len(qualified.Candidate.Definition.QualificationReviews) == 0 {
		return nil, fmt.Errorf("%w: export requires a qualified revision containing an explicit review", devices.ErrInvalidCandidate)
	}
	review := qualified.Candidate.Definition.QualificationReviews[len(qualified.Candidate.Definition.QualificationReviews)-1]
	if review.Result != "passed" || review.NativeRenderJobID == "" {
		return nil, fmt.Errorf("%w: only passed native reviews can be exported", devices.ErrInvalidCandidate)
	}
	candidate, err := s.devices.GetRevision(ctx, deviceID, review.CandidateRevision)
	if err != nil {
		return nil, err
	}
	if candidate.State != devices.DeviceRevisionCandidate || candidate.SHA256 != review.CandidateSHA256 {
		return nil, devices.ErrRevisionConflict
	}
	scene, err := s.scenes.GetRevision(ctx, review.SceneID, review.SceneRevision)
	if err != nil {
		return nil, err
	}
	motion, err := s.motions.GetRevision(ctx, review.MotionID, review.MotionRevision)
	if err != nil {
		return nil, err
	}
	preset, err := s.presets.Get(ctx, review.PresetID, review.PresetRevision)
	if err != nil {
		return nil, err
	}
	proof, output, err := exporter.ExportQualificationRender(ctx, projectID, review.NativeRenderJobID)
	if err != nil {
		return nil, err
	}
	bundles, artifacts, err := s.exportQualificationSourceClosure(ctx, candidate.Candidate.Geometry, scene.Candidate.Artifacts)
	if err != nil {
		return nil, err
	}
	if proof.OutputSHA256 != review.NativeOutputSHA256 || proof.OutputKind != review.NativeOutputKind || digestHex(output) != proof.OutputSHA256 {
		return nil, fmt.Errorf("%w: completed render output differs from the accepted review", devices.ErrInvalidCandidate)
	}
	// The scene candidate references its own geometry and texture artifacts.
	// Carry the complete render input closure so an import can verify every
	// source byte, rather than checking only the device body GLB.
	pkg := qualificationPackage{Version: 2, Origin: qualificationPackageOrigin{ProjectID: projectID, JobID: review.NativeRenderJobID, Reviewed: review.ReviewedAt}, Candidate: candidate, SourceBundles: bundles, SourceArtifacts: artifacts, Scene: scene, Motion: motion, Preset: preset, Review: review, Render: proof, Output: output}
	encoded, err := sealQualificationPackage(pkg)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxQualificationPackageBytes {
		return nil, errors.New("qualification package exceeds the 128 MiB limit")
	}
	return encoded, nil
}

// ImportReviewPackage applies an explicit transferred human review only when
// all immutable destination references and included bytes match exactly.
func (s *QualificationService) ImportReviewPackage(ctx context.Context, data []byte, expectedHead uint32) (devices.QualificationReviewResult, error) {
	pkg, err := openQualificationPackage(data)
	if err != nil {
		return devices.QualificationReviewResult{}, err
	}
	if pkg.Version != 2 || pkg.Origin.ProjectID == "" || pkg.Origin.JobID == "" || pkg.Origin.JobID != pkg.Review.NativeRenderJobID || pkg.Origin.Reviewed != pkg.Review.ReviewedAt || pkg.Review.Result != "passed" || pkg.Review.Reviewer == "" || (pkg.Review.Scope != "visual_setup" && pkg.Review.Scope != "physical_geometry") || len(pkg.Review.DimensionStates) != 1 || len(pkg.Review.DisplayIDs) != 1 || len(pkg.Review.SupportIDs) != 1 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: package provenance or human review decision is incomplete", devices.ErrInvalidCandidate)
	}
	if pkg.Candidate.DeviceID == "" || pkg.Candidate.Revision != pkg.Review.CandidateRevision || pkg.Candidate.SHA256 != pkg.Review.CandidateSHA256 || pkg.Candidate.State != devices.DeviceRevisionCandidate || pkg.Candidate.Candidate.Definition.Qualification != "unqualified" {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: package candidate binding is invalid", devices.ErrInvalidCandidate)
	}
	if pkg.Scene.SceneID != pkg.Review.SceneID || pkg.Scene.Revision != pkg.Review.SceneRevision || pkg.Scene.SHA256 != pkg.Review.SceneSHA256 || pkg.Motion.MotionID != pkg.Review.MotionID || pkg.Motion.Revision != pkg.Review.MotionRevision || pkg.Motion.SHA256 != pkg.Review.MotionSHA256 || pkg.Preset.PresetID != pkg.Review.PresetID || pkg.Preset.Revision != pkg.Review.PresetRevision || pkg.Preset.SHA256 != pkg.Review.PresetSHA256 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: package scene, motion, or preset revision bindings are inconsistent", devices.ErrInvalidCandidate)
	}
	if pkg.Render.OutputSHA256 != pkg.Review.NativeOutputSHA256 || pkg.Render.OutputKind != pkg.Review.NativeOutputKind || digestHex(pkg.Output) != pkg.Render.OutputSHA256 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: package source or output bytes do not match their exact references", devices.ErrInvalidCandidate)
	}
	if len(pkg.Output) == 0 || len(pkg.Output) > 64<<20 {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: native output is empty or exceeds the package limit", devices.ErrInvalidCandidate)
	}
	current, err := s.devices.GetRevision(ctx, pkg.Candidate.DeviceID, pkg.Candidate.Revision)
	currentBytes, _ := json.Marshal(current.Candidate)
	packageBytes, _ := json.Marshal(pkg.Candidate.Candidate)
	if err != nil || current.SHA256 != pkg.Candidate.SHA256 || !bytes.Equal(currentBytes, packageBytes) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: exact candidate revision is missing or changed in destination", devices.ErrInvalidCandidate)
	}
	if s.scenes == nil || s.motions == nil || s.presets == nil {
		return devices.QualificationReviewResult{}, errors.New("qualification package import requires scene, motion, and preset owners")
	}
	if err := s.verifyQualificationSourceClosure(ctx, pkg.Candidate.Candidate.Geometry, pkg.Scene.Candidate.Artifacts, pkg.SourceBundles, pkg.SourceArtifacts); err != nil {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: %v", devices.ErrInvalidCandidate, err)
	}
	destinationScene, err := s.scenes.GetRevision(ctx, pkg.Scene.SceneID, pkg.Scene.Revision)
	if err != nil || destinationScene.SHA256 != pkg.Scene.SHA256 || !sameJSON(destinationScene.Candidate, pkg.Scene.Candidate) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: exact scene revision is missing or changed in destination", devices.ErrInvalidCandidate)
	}
	destinationMotion, err := s.motions.GetRevision(ctx, pkg.Motion.MotionID, pkg.Motion.Revision)
	if err != nil || destinationMotion.SHA256 != pkg.Motion.SHA256 || !sameJSON(destinationMotion.Candidate, pkg.Motion.Candidate) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: exact motion revision is missing or changed in destination", devices.ErrInvalidCandidate)
	}
	destinationPreset, err := s.presets.Get(ctx, pkg.Preset.PresetID, pkg.Preset.Revision)
	if err != nil || destinationPreset.SHA256 != pkg.Preset.SHA256 || destinationPreset.Name != pkg.Preset.Name || !proto.Equal(destinationPreset.Setup, pkg.Preset.Setup) {
		return devices.QualificationReviewResult{}, fmt.Errorf("%w: exact preset revision is missing or changed in destination", devices.ErrInvalidCandidate)
	}
	req := devices.QualificationReviewRequest{ProjectID: pkg.Origin.ProjectID, DeviceID: pkg.Candidate.DeviceID, CandidateRevision: pkg.Candidate.Revision, CandidateSHA256: pkg.Candidate.SHA256, ExpectedHead: expectedHead, Review: pkg.Review}
	return s.reviewCandidateWithEvidence(ctx, req, pkg.Render)
}

const qualificationPackageArtifactLimit = 64 << 20

func qualificationArtifactKey(ref physical.ArtifactRef) string {
	return fmt.Sprintf("%s\x00%d\x00%s", ref.BundleID, ref.BundleRevision, ref.ArtifactID)
}

func qualificationBundleKey(bundleID string, revision uint32) string {
	return fmt.Sprintf("%s\x00%d", bundleID, revision)
}

func qualificationSourceRefs(device physical.ArtifactRef, scene []scenes.SceneArtifactBinding) ([]physical.ArtifactRef, error) {
	refs := map[string]physical.ArtifactRef{}
	add := func(ref physical.ArtifactRef) error {
		if ref.BundleID == "" || ref.BundleRevision == 0 || ref.ArtifactID == "" || ref.SizeBytes <= 0 || ref.SizeBytes > qualificationPackageArtifactLimit || ref.SHA256 == "" {
			return errors.New("qualification source artifact reference is incomplete or exceeds 64 MiB")
		}
		key := qualificationArtifactKey(ref)
		if existing, ok := refs[key]; ok && !sameJSON(existing, ref) {
			return fmt.Errorf("artifact key %q has conflicting pinned references", key)
		}
		refs[key] = ref
		return nil
	}
	if err := add(device); err != nil {
		return nil, err
	}
	for _, binding := range scene {
		if err := add(binding.Ref); err != nil {
			return nil, err
		}
	}
	result := make([]physical.ArtifactRef, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref)
	}
	sort.Slice(result, func(i, j int) bool { return qualificationArtifactKey(result[i]) < qualificationArtifactKey(result[j]) })
	return result, nil
}

func (s *QualificationService) exportQualificationSourceClosure(ctx context.Context, device physical.ArtifactRef, scene []scenes.SceneArtifactBinding) ([]library.BundleRevision, []qualificationPackageArtifact, error) {
	artifactOpener, ok := s.bundles.(qualificationArtifactOpener)
	if !ok {
		return nil, nil, errors.New("qualification package export requires source artifact access")
	}
	refs, err := qualificationSourceRefs(device, scene)
	if err != nil {
		return nil, nil, err
	}
	bundleMap := map[string]library.BundleRevision{}
	artifacts := make([]qualificationPackageArtifact, 0, len(refs))
	for _, ref := range refs {
		key := qualificationBundleKey(ref.BundleID, ref.BundleRevision)
		bundle, exists := bundleMap[key]
		if !exists {
			bundle, err = s.bundles.GetRevision(ctx, ref.BundleID, ref.BundleRevision)
			if err != nil {
				return nil, nil, fmt.Errorf("read source bundle %s revision %d: %w", ref.BundleID, ref.BundleRevision, err)
			}
			if bundle.Manifest.BundleID != ref.BundleID || bundle.Manifest.Revision != ref.BundleRevision || bundle.ManifestSHA256 != ref.ManifestSHA256 || bundle.ReviewSHA256 != ref.ReviewSHA256 {
				return nil, nil, errors.New("qualification source artifact does not match its exact Library revision")
			}
			bundleMap[key] = bundle
		}
		resolved, openErr := artifactOpener.OpenArtifact(ctx, ref.BundleID, ref.BundleRevision, ref.ArtifactID, library.UseRender)
		if openErr != nil {
			return nil, nil, fmt.Errorf("open qualification source artifact %s: %w", ref.ArtifactID, openErr)
		}
		data, readErr := io.ReadAll(io.LimitReader(resolved.Reader, qualificationPackageArtifactLimit+1))
		closeErr := resolved.Reader.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) != ref.SizeBytes || digestHex(data) != ref.SHA256 || resolved.Artifact.Role != ref.Role || resolved.Artifact.SizeBytes != ref.SizeBytes || resolved.Artifact.SHA256 != ref.SHA256 {
			return nil, nil, errors.New("qualification source bytes failed size, role, or digest verification")
		}
		artifacts = append(artifacts, qualificationPackageArtifact{Ref: ref, Bytes: data})
	}
	bundles := make([]library.BundleRevision, 0, len(bundleMap))
	for _, bundle := range bundleMap {
		bundles = append(bundles, bundle)
	}
	sort.Slice(bundles, func(i, j int) bool {
		return qualificationBundleKey(bundles[i].Manifest.BundleID, bundles[i].Manifest.Revision) < qualificationBundleKey(bundles[j].Manifest.BundleID, bundles[j].Manifest.Revision)
	})
	return bundles, artifacts, nil
}

func (s *QualificationService) verifyQualificationSourceClosure(ctx context.Context, device physical.ArtifactRef, scene []scenes.SceneArtifactBinding, bundles []library.BundleRevision, artifacts []qualificationPackageArtifact) error {
	artifactOpener, ok := s.bundles.(qualificationArtifactOpener)
	if !ok {
		return errors.New("qualification package import requires source artifact access")
	}
	refs, err := qualificationSourceRefs(device, scene)
	if err != nil {
		return err
	}
	if len(refs) != len(artifacts) {
		return errors.New("qualification package source artifact set is incomplete or contains extras")
	}
	artifactMap := make(map[string]qualificationPackageArtifact, len(artifacts))
	for _, artifact := range artifacts {
		key := qualificationArtifactKey(artifact.Ref)
		if _, exists := artifactMap[key]; exists {
			return errors.New("qualification package contains duplicate source artifact references")
		}
		artifactMap[key] = artifact
	}
	bundleMap := make(map[string]library.BundleRevision, len(bundles))
	for _, bundle := range bundles {
		key := qualificationBundleKey(bundle.Manifest.BundleID, bundle.Manifest.Revision)
		if _, exists := bundleMap[key]; exists {
			return errors.New("qualification package contains duplicate source bundle revisions")
		}
		bundleMap[key] = bundle
	}
	wantBundles := map[string]bool{}
	for _, ref := range refs {
		artifact, ok := artifactMap[qualificationArtifactKey(ref)]
		if !ok || !sameJSON(artifact.Ref, ref) || int64(len(artifact.Bytes)) != ref.SizeBytes || digestHex(artifact.Bytes) != ref.SHA256 {
			return fmt.Errorf("package bytes for source artifact %s are missing or do not match the pinned reference", ref.ArtifactID)
		}
		bundleKey := qualificationBundleKey(ref.BundleID, ref.BundleRevision)
		wantBundles[bundleKey] = true
		packagedBundle, ok := bundleMap[bundleKey]
		if !ok || packagedBundle.ManifestSHA256 != ref.ManifestSHA256 || packagedBundle.ReviewSHA256 != ref.ReviewSHA256 || packagedBundle.Manifest.BundleID != ref.BundleID || packagedBundle.Manifest.Revision != ref.BundleRevision {
			return fmt.Errorf("exact source Library revision %s is missing from package", bundleKey)
		}
		destinationBundle, err := s.bundles.GetRevision(ctx, ref.BundleID, ref.BundleRevision)
		if err != nil || destinationBundle.ManifestSHA256 != packagedBundle.ManifestSHA256 || destinationBundle.ReviewSHA256 != packagedBundle.ReviewSHA256 || !sameJSON(destinationBundle.Manifest, packagedBundle.Manifest) || !sameJSON(destinationBundle.Review, packagedBundle.Review) {
			return fmt.Errorf("exact source Library manifest or review %s is missing or changed in destination", bundleKey)
		}
		resolved, err := artifactOpener.OpenArtifact(ctx, ref.BundleID, ref.BundleRevision, ref.ArtifactID, library.UseRender)
		if err != nil {
			return fmt.Errorf("exact source artifact %s is unavailable in destination: %w", ref.ArtifactID, err)
		}
		actual, readErr := io.ReadAll(io.LimitReader(resolved.Reader, qualificationPackageArtifactLimit+1))
		closeErr := resolved.Reader.Close()
		if readErr != nil || closeErr != nil || int64(len(actual)) != ref.SizeBytes || digestHex(actual) != ref.SHA256 || !bytes.Equal(actual, artifact.Bytes) {
			return fmt.Errorf("destination source bytes for artifact %s differ from package", ref.ArtifactID)
		}
	}
	if len(bundleMap) != len(wantBundles) {
		return errors.New("qualification package contains unreferenced source bundle revisions")
	}
	return nil
}

func sealQualificationPackage(pkg qualificationPackage) ([]byte, error) {
	pkg.PackageSHA = ""
	body, err := json.Marshal(pkg)
	if err != nil {
		return nil, err
	}
	pkg.PackageSHA = digestHex(body)
	return json.Marshal(pkg)
}

func openQualificationPackage(data []byte) (qualificationPackage, error) {
	var pkg qualificationPackage
	if len(data) == 0 || len(data) > maxQualificationPackageBytes {
		return pkg, errors.New("qualification package is empty or exceeds the 128 MiB limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pkg); err != nil {
		return pkg, fmt.Errorf("decode qualification package: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return pkg, errors.New("qualification package has trailing data")
	}
	claimed := pkg.PackageSHA
	pkg.PackageSHA = ""
	canonical, err := json.Marshal(pkg)
	if err != nil || claimed == "" || claimed != digestHex(canonical) {
		return pkg, errors.New("qualification package digest is missing or invalid")
	}
	pkg.PackageSHA = claimed
	return pkg, nil
}

func digestHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func sameJSON(left, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
