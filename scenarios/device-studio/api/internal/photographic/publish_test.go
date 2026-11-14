package photographic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"device-studio/internal/library"
	"device-studio/internal/scenes"
)

type capturePublisher struct{ input library.PublishInput }

func (p *capturePublisher) Publish(_ context.Context, in library.PublishInput) (library.BundleRevision, error) {
	p.input = in
	_, _, manifestSHA, err := library.CanonicalizeManifest(in.Manifest)
	if err != nil {
		return library.BundleRevision{}, err
	}
	return library.BundleRevision{Manifest: in.Manifest, ManifestSHA256: manifestSHA, Review: in.Review, ReviewSHA256: digest([]byte("review"))}, nil
}

type captureSceneSaver struct{ candidate scenes.SceneCandidate }

func (s *captureSceneSaver) SaveCandidate(_ context.Context, in scenes.SceneCandidate, expected uint32) (scenes.Revision, error) {
	if expected != 0 {
		return scenes.Revision{}, scenes.ErrRevisionConflict
	}
	canonical, _, sum, err := scenes.CanonicalizeCandidate(in)
	if err != nil {
		return scenes.Revision{}, err
	}
	s.candidate = canonical
	return scenes.Revision{SceneID: canonical.SceneID, Revision: 1, SHA256: sum, State: scenes.RevisionCandidate, Candidate: canonical}, nil
}

func TestPublishCandidateRetainsReviewedLineageAndGrantsEditOnly(t *testing.T) {
	pub, save := &capturePublisher{}, &captureSceneSaver{}
	result, err := PublishCandidate(context.Background(), photographicFixture(t), PublishClaims{LicenseID: "CC-BY-4.0", Attribution: "Photographer attribution", SourceLabel: "reviewed source photo"}, pub, save)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scene.Revision != 1 || result.Scene.State != scenes.RevisionCandidate {
		t.Fatalf("candidate not saved: %+v", result.Scene)
	}
	if save.candidate.Photograph == nil || save.candidate.Photograph.ScaleStatus != "estimated_monocular_scale_not_measured" {
		t.Fatal("scene candidate lost explicit estimated-scale status")
	}
	if save.candidate.Photograph.SourceReviewSHA256 == "" || save.candidate.Photograph.SourceReviewer != "operator-reviewer" {
		t.Fatal("source visual review lineage was not retained")
	}
	if len(pub.input.Manifest.Artifacts) != 2 || len(pub.input.Sources) != 2 {
		t.Fatalf("expected geometry and clean-plate artifacts, got %+v", pub.input.Manifest)
	}
	for _, artifact := range pub.input.Manifest.Artifacts {
		if len(artifact.Rights) != 1 || artifact.Rights[0] != library.UseEdit {
			t.Fatalf("premature scene use granted: %+v", artifact.Rights)
		}
	}
	if len(save.candidate.CameraFrames) != 1 || len(save.candidate.SupportFrames) != 1 {
		t.Fatal("source camera or support plane frame missing")
	}
	if len(save.candidate.Materials) != 1 || save.candidate.Materials[0].SurfaceProfile != scenes.MaterialEmissionV1 {
		t.Fatalf("photographic plate must preserve baked appearance with emission, got %+v", save.candidate.Materials)
	}
	plateInput := save.candidate.Materials[0].Inputs[0]
	if plateInput.AffineUV == nil || *plateInput.AffineUV != ([6]float64{1, 0, 0, 0, 1, 0}) {
		t.Fatalf("photographic plate must use identity [a,b,tx,c,d,ty] UV mapping, got %+v", plateInput.AffineUV)
	}
	if len(save.candidate.LightFrames) != 1 || save.candidate.LightFrames[0].ID != "estimated-left-window-area-v1" || save.candidate.LightFrames[0].PowerWatts != 220 {
		t.Fatalf("device light must be explicitly authored as a bounded source estimate, got %+v", save.candidate.LightFrames)
	}
	if save.candidate.LightFrames[0].Frame.Translation.X >= 0 {
		t.Fatalf("estimated window light should be placed to image-left of the source camera, got %+v", save.candidate.LightFrames[0].Frame)
	}
}

func TestPublishCandidateGrantsOnlyExplicitLocalRenderAndClaimsChangeIdentity(t *testing.T) {
	firstPub, firstSave := &capturePublisher{}, &captureSceneSaver{}
	raw := photographicFixture(t)
	claims := PublishClaims{LicenseID: "CC-BY-4.0", Attribution: "Photographer attribution", SourceLabel: "reviewed source photo", AllowLocalRender: true, SourceUseRationale: "Source license permits local compositing and render evaluation."}
	first, err := PublishCandidate(context.Background(), raw, claims, firstPub, firstSave)
	if err != nil {
		t.Fatal(err)
	}
	if first.Conversion.SourceUseClaimSHA256 == "" || !first.Conversion.LocalRenderAuthorized {
		t.Fatal("source-use claim was not retained in conversion metadata")
	}
	if firstPub.input.Review.Rationale != claims.SourceUseRationale {
		t.Fatalf("Library review did not retain the source-use rationale: %+v", firstPub.input.Review)
	}
	var adaptation struct {
		AssemblyProfile string `json:"assembly_profile"`
		RenderMode      string `json:"render_mode"`
	}
	if err := json.Unmarshal([]byte(firstPub.input.Manifest.Artifacts[0].Provenance.Adaptation), &adaptation); err != nil {
		t.Fatal(err)
	}
	if adaptation.AssemblyProfile != sceneAssemblyProfile || !strings.Contains(adaptation.AssemblyProfile, "gltf-top-origin-uv-row-major-affine-identity") || !strings.Contains(adaptation.AssemblyProfile, "source-camera-plate-no-parallax") {
		t.Fatalf("adaptation omitted assembly profile: %+v", adaptation)
	}
	if adaptation.RenderMode != "SOURCE_CAMERA_PLATE_V1" {
		t.Fatalf("adaptation omitted explicit render mode: %+v", adaptation)
	}
	if first.Scene.Candidate.Photograph == nil || first.Scene.Candidate.Photograph.RenderMode != "SOURCE_CAMERA_PLATE_V1" {
		t.Fatalf("typed source camera plate mode missing from scene candidate: %+v", first.Scene.Candidate.Photograph)
	}
	for _, artifact := range firstPub.input.Manifest.Artifacts {
		if len(artifact.Rights) != 2 || artifact.Rights[0] != library.UseEdit || artifact.Rights[1] != library.UseRender {
			t.Fatalf("explicit local render claim did not grant exactly edit+render: %+v", artifact.Rights)
		}
	}
	if len(firstPub.input.Review.AllowedUses) != 2 || firstPub.input.Review.AllowedUses[1] != library.UseRender {
		t.Fatalf("review did not bind the local render use: %+v", firstPub.input.Review.AllowedUses)
	}
	secondClaims := claims
	secondClaims.SourceUseRationale = "Different bounded source-use basis."
	second, err := PublishCandidate(context.Background(), raw, secondClaims, &capturePublisher{}, &captureSceneSaver{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Library.Manifest.BundleID == second.Library.Manifest.BundleID || first.Scene.SceneID == second.Scene.SceneID || first.Conversion.SourceUseClaimSHA256 == second.Conversion.SourceUseClaimSHA256 {
		t.Fatal("changed rights claims reused an immutable candidate identity")
	}
	sourceSHA := strings.Repeat("a", 64)
	profileV1 := photographicCandidateIdentity(sourceSHA, "same-rights", "assembly-v1")
	profileV2 := photographicCandidateIdentity(sourceSHA, "same-rights", "assembly-v2")
	if profileV1 == profileV2 {
		t.Fatal("changed conversion profile reused immutable candidate identity")
	}
	if photographicReviewBindingID(sourceSHA, profileV1) == photographicReviewBindingID(sourceSHA, profileV2) {
		t.Fatal("changed conversion profile reused the immutable Library review identity")
	}
	if !strings.Contains(first.Library.Manifest.Artifacts[0].Provenance.Adaptation, sceneAssemblyProfile) {
		t.Fatal("published provenance does not describe the exact photographic assembly profile")
	}
	replay, err := PublishCandidate(context.Background(), raw, claims, &capturePublisher{}, &captureSceneSaver{})
	if err != nil {
		t.Fatal(err)
	}
	if replay.Library.Manifest.BundleID != first.Library.Manifest.BundleID || replay.Scene.SceneID != first.Scene.SceneID || replay.Conversion.SourceUseClaimSHA256 != first.Conversion.SourceUseClaimSHA256 {
		t.Fatal("same source, claims, and assembly profile did not replay the same immutable identity")
	}
}

func TestPublishCandidateRejectsRationaleWithoutLocalRenderChoice(t *testing.T) {
	_, err := PublishCandidate(context.Background(), photographicFixture(t), PublishClaims{LicenseID: "CC-BY-4.0", Attribution: "Photographer attribution", SourceLabel: "reviewed source photo", SourceUseRationale: "Local rendering permitted"}, &capturePublisher{}, &captureSceneSaver{})
	if err == nil {
		t.Fatal("rationale without the explicit local render choice was accepted")
	}
}

func TestPublishCandidateRequiresOperatorRightsClaims(t *testing.T) {
	_, err := PublishCandidate(context.Background(), photographicFixture(t), PublishClaims{LicenseID: "CC-BY-4.0"}, &capturePublisher{}, &captureSceneSaver{})
	if err == nil {
		t.Fatal("incomplete rights claims accepted")
	}
}
