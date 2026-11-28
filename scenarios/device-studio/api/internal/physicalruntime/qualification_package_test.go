package physicalruntime

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

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

type packageSceneStub struct{ revision scenes.Revision }

func (s packageSceneStub) GetRevision(context.Context, string, uint32) (scenes.Revision, error) {
	return s.revision, nil
}

func TestQualificationPackageSealDetectsTamperingAndUnknownFields(t *testing.T) {
	ref := physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ArtifactID: "glb", SizeBytes: 3, SHA256: digestHex([]byte("glb"))}
	pkg := qualificationPackage{Version: 2, Origin: qualificationPackageOrigin{ProjectID: "project", JobID: "job", Reviewed: "2026-10-21T00:00:00Z"}, SourceArtifacts: []qualificationPackageArtifact{{Ref: ref, Bytes: []byte("glb")}}, Output: []byte("png")}
	data, err := sealQualificationPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openQualificationPackage(data); err != nil {
		t.Fatalf("open valid package: %v", err)
	}
	tampered := append([]byte(nil), data...)
	needle := []byte(`"bytes":"`)
	for i := 0; i < len(tampered)-len(needle)-1; i++ {
		if string(tampered[i:i+len(needle)]) == string(needle) {
			tampered[i+len(needle)] = 'j'
			break
		}
	}
	if _, err := openQualificationPackage(tampered); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("tampered package accepted: %v", err)
	}

	withUnknown := strings.TrimSuffix(string(data), "}") + `,"signed_by":"not-a-signature"}`
	if _, err := openQualificationPackage([]byte(withUnknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field accepted: %v", err)
	}
}

type packageSourceStub struct {
	bytes    []byte
	revision library.BundleRevision
	openErr  error
	assets   map[string][]byte
	bundles  map[string]library.BundleRevision
	missing  map[string]error
}

func (s packageSourceStub) GetRevision(_ context.Context, bundleID string, revision uint32) (library.BundleRevision, error) {
	if bundle, ok := s.bundles[qualificationBundleKey(bundleID, revision)]; ok {
		return bundle, nil
	}
	if s.revision.Manifest.BundleID == bundleID && s.revision.Manifest.Revision == revision {
		return s.revision, nil
	}
	return library.BundleRevision{}, library.ErrRevisionNotFound
}

func (s packageSourceStub) OpenArtifact(_ context.Context, bundleID string, revision uint32, artifactID string, _ library.Use) (library.ResolvedArtifact, error) {
	if s.openErr != nil {
		return library.ResolvedArtifact{}, s.openErr
	}
	key := qualificationArtifactKey(physical.ArtifactRef{BundleID: bundleID, BundleRevision: revision, ArtifactID: artifactID})
	if err := s.missing[key]; err != nil {
		return library.ResolvedArtifact{}, err
	}
	bundle, err := s.GetRevision(context.Background(), bundleID, revision)
	if err != nil {
		return library.ResolvedArtifact{}, err
	}
	for _, artifact := range bundle.Manifest.Artifacts {
		if artifact.ID == artifactID {
			data, ok := s.assets[key]
			if !ok && bundle.Manifest.BundleID == s.revision.Manifest.BundleID {
				data = s.bytes
			}
			if !ok && data == nil {
				return library.ResolvedArtifact{}, library.ErrArtifactNotFound
			}
			return library.ResolvedArtifact{BundleID: bundleID, Revision: revision, Artifact: artifact, Reader: io.NopCloser(bytes.NewReader(data))}, nil
		}
	}
	return library.ResolvedArtifact{}, library.ErrArtifactNotFound
}

func TestQualificationPackageRejectsWrongScopeAndChangedDestinationCandidate(t *testing.T) {
	source := physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ManifestSHA256: repeated('a'), ReviewSHA256: repeated('b'), ArtifactID: "glb", Role: library.RoleGeometryGLB, SizeBytes: 3, SHA256: digestHex([]byte("glb"))}
	parent := devices.DeviceRevision{DeviceID: "device", Revision: 1, SHA256: repeated('c'), State: devices.DeviceRevisionCandidate, Candidate: devices.DeviceCandidate{Geometry: source, Definition: devices.Definition{ID: "device", Qualification: "unqualified"}}}
	baseReview := devices.QualificationReview{Scope: "visual_setup", Result: "passed", Reviewer: "operator", CandidateRevision: 1, CandidateSHA256: parent.SHA256, SourceBundleID: source.BundleID, SourceBundleRevision: source.BundleRevision, SourceManifestSHA256: source.ManifestSHA256, SourceReviewSHA256: source.ReviewSHA256, SourceArtifactID: source.ArtifactID, SourceArtifactSHA256: source.SHA256, NativeRenderJobID: "job", NativeOutputSHA256: digestHex([]byte("png")), NativeOutputKind: "physical_still_png", ReviewedAt: "2026-10-21T00:00:00Z", DimensionStates: []string{"default"}, DisplayIDs: []string{"display"}, SupportIDs: []string{"support"}, PresetID: "preset", PresetRevision: 1, PresetSHA256: repeated('e'), SceneID: "scene", SceneRevision: 1, SceneSHA256: repeated('f'), MotionID: "motion", MotionRevision: 1, MotionSHA256: repeated('a')}
	bundle := library.BundleRevision{Manifest: library.BundleManifest{BundleID: source.BundleID, Revision: 1}, ManifestSHA256: source.ManifestSHA256, Review: library.ReviewBinding{ManifestSHA256: source.ManifestSHA256}, ReviewSHA256: source.ReviewSHA256}
	base := qualificationPackage{Version: 2, Origin: qualificationPackageOrigin{ProjectID: "project", JobID: "job", Reviewed: baseReview.ReviewedAt}, Candidate: parent, SourceBundles: []library.BundleRevision{bundle}, SourceArtifacts: []qualificationPackageArtifact{{Ref: source, Bytes: []byte("glb")}}, Scene: scenes.Revision{SceneID: baseReview.SceneID, Revision: 1, SHA256: baseReview.SceneSHA256}, Motion: motions.Revision{MotionID: baseReview.MotionID, Revision: 1, SHA256: baseReview.MotionSHA256}, Preset: shotdomain.PhysicalPresetRevision{PresetID: baseReview.PresetID, Revision: 1, SHA256: baseReview.PresetSHA256}, Review: baseReview, Render: renderjobs.QualificationRenderEvidence{OutputSHA256: baseReview.NativeOutputSHA256, OutputKind: baseReview.NativeOutputKind}, Output: []byte("png")}

	wrongScope := base
	wrongScope.Review.Scope = "everything"
	data, err := sealQualificationPackage(wrongScope)
	if err != nil {
		t.Fatal(err)
	}
	service := &QualificationService{}
	if _, err := service.ImportReviewPackage(context.Background(), data, 1); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("wrong scope accepted: %v", err)
	}

	stale := base
	stale.Candidate.SHA256 = repeated('d')
	stale.Review.CandidateSHA256 = stale.Candidate.SHA256
	service = &QualificationService{devices: &qualificationDeviceStub{parent: parent, head: parent}, bundles: packageSourceStub{bytes: []byte("wrong"), revision: bundle}}
	data, err = sealQualificationPackage(stale)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ImportReviewPackage(context.Background(), data, 1); err == nil || !strings.Contains(err.Error(), "destination") {
		t.Fatalf("changed destination candidate accepted: %v", err)
	}
}

func TestQualificationPackageSourceClosureCarriesEveryReferencedArtifact(t *testing.T) {
	ctx := context.Background()
	deviceBytes, roomBytes := []byte("device-glb"), []byte("room-and-texture-glb")
	makeRef := func(bundleID, artifactID string, data []byte, manifest, review byte) physical.ArtifactRef {
		return physical.ArtifactRef{BundleID: bundleID, BundleRevision: 1, ManifestSHA256: repeated(manifest), ReviewSHA256: repeated(review), ArtifactID: artifactID, Role: library.RoleGeometryGLB, SizeBytes: int64(len(data)), SHA256: digestHex(data)}
	}
	deviceRef := makeRef("device-bundle", "device", deviceBytes, 'a', 'b')
	roomRef := makeRef("room-bundle", "room", roomBytes, 'c', 'd')
	makeBundle := func(ref physical.ArtifactRef) library.BundleRevision {
		return library.BundleRevision{Manifest: library.BundleManifest{BundleID: ref.BundleID, Revision: ref.BundleRevision, Artifacts: []library.Artifact{{ID: ref.ArtifactID, Role: ref.Role, SizeBytes: ref.SizeBytes, SHA256: ref.SHA256, Rights: []library.Use{library.UseRender}}}}, ManifestSHA256: ref.ManifestSHA256, Review: library.ReviewBinding{Decision: "approved", ManifestSHA256: ref.ManifestSHA256, AllowedUses: []library.Use{library.UseRender}}, ReviewSHA256: ref.ReviewSHA256}
	}
	deviceBundle, roomBundle := makeBundle(deviceRef), makeBundle(roomRef)
	store := packageSourceStub{bundles: map[string]library.BundleRevision{
		qualificationBundleKey(deviceRef.BundleID, 1): deviceBundle,
		qualificationBundleKey(roomRef.BundleID, 1):   roomBundle,
	}, assets: map[string][]byte{qualificationArtifactKey(deviceRef): deviceBytes, qualificationArtifactKey(roomRef): roomBytes}}
	service := QualificationService{bundles: store}
	bundles, artifacts, err := service.exportQualificationSourceClosure(ctx, deviceRef, []scenes.SceneArtifactBinding{{ID: "wall", Ref: roomRef}, {ID: "wall-texture", Ref: roomRef}})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 2 || len(artifacts) != 2 || !bytes.Equal(artifacts[0].Bytes, deviceBytes) || !bytes.Equal(artifacts[1].Bytes, roomBytes) {
		t.Fatalf("source closure did not deduplicate and package both exact references: bundles=%d artifacts=%+v", len(bundles), artifacts)
	}
	store.missing = map[string]error{qualificationArtifactKey(roomRef): library.ErrArtifactNotFound}
	service.bundles = store
	if _, _, err := service.exportQualificationSourceClosure(ctx, deviceRef, []scenes.SceneArtifactBinding{{ID: "wall", Ref: roomRef}}); err == nil {
		t.Fatal("export accepted a scene artifact absent from the source Library owner")
	}
	store.missing = nil
	store.assets[qualificationArtifactKey(roomRef)] = []byte("changed room")
	service.bundles = store
	if _, _, err := service.exportQualificationSourceClosure(ctx, deviceRef, []scenes.SceneArtifactBinding{{ID: "wall", Ref: roomRef}}); err == nil {
		t.Fatal("export accepted scene bytes that differ from the pinned digest")
	}
}

func TestQualificationPackageImportReusesExactVerifierAndAppendsIdempotently(t *testing.T) {
	ctx := context.Background()
	geometry := []byte("glb")
	output := []byte("verified-native-png-bytes")
	geometrySHA, outputSHA := digestHex(geometry), digestHex(output)
	source := physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ManifestSHA256: repeated('a'), ReviewSHA256: repeated('b'), ArtifactID: "glb", Role: library.RoleGeometryGLB, SizeBytes: int64(len(geometry)), SHA256: geometrySHA}
	parent := devices.DeviceRevision{DeviceID: "device", Revision: 2, SHA256: repeated('d'), State: devices.DeviceRevisionCandidate, Candidate: devices.DeviceCandidate{Geometry: source, Definition: devices.Definition{ID: "device", Qualification: "unqualified"}}}
	sceneBytes := []byte("scene-glb")
	sceneRef := physical.ArtifactRef{BundleID: "scene-bundle", BundleRevision: 1, ManifestSHA256: repeated('7'), ReviewSHA256: repeated('8'), ArtifactID: "room-glb", Role: library.RoleGeometryGLB, SizeBytes: int64(len(sceneBytes)), SHA256: digestHex(sceneBytes)}
	identity := &physicalv1.TRS{TranslationM: &physicalv1.Vec3{}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}}
	setup := &shotsv1.PhysicalShotSetup{Device: &shotsv1.ComponentRevisionRef{Id: "device", Revision: 2, Sha256: parent.SHA256}, Scene: &shotsv1.ComponentRevisionRef{Id: "scene", Revision: 3, Sha256: repeated('e')}, Motion: &shotsv1.ComponentRevisionRef{Id: "motion", Revision: 4, Sha256: repeated('f')}, DisplayId: "display", DeviceSupportId: "base", SceneSupportId: "table", CameraId: "camera", SceneSupportWorld: proto.Clone(identity).(*physicalv1.TRS), DeviceRootRest: proto.Clone(identity).(*physicalv1.TRS), SupportPivotM: &physicalv1.Vec3{}}
	preset := shotdomain.PhysicalPresetRevision{PresetID: "preset", Revision: 5, SHA256: repeated('9'), Name: "review", Setup: setup}
	review := devices.QualificationReview{Scope: "visual_setup", Result: "passed", Reviewer: "operator", Evidence: []devices.Evidence{{Source: "job", Kind: "physical_still_png"}}, CandidateRevision: 2, CandidateSHA256: parent.SHA256,
		SourceBundleID: source.BundleID, SourceBundleRevision: source.BundleRevision, SourceManifestSHA256: source.ManifestSHA256, SourceReviewSHA256: source.ReviewSHA256, SourceArtifactID: source.ArtifactID, SourceArtifactSHA256: source.SHA256, SourceAllowedUses: []string{"render"},
		DimensionStates: []string{"default"}, DisplayIDs: []string{"display"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"base"}, PresetID: preset.PresetID, PresetRevision: preset.Revision, PresetSHA256: preset.SHA256,
		SceneID: "scene", SceneRevision: 3, SceneSHA256: repeated('e'), MotionID: "motion", MotionRevision: 4, MotionSHA256: repeated('f'), NativeRenderJobID: "job", NativeOutputSHA256: outputSHA, NativeOutputKind: "physical_still_png", RightsClaimsAcknowledged: true, ReviewedAt: "2026-10-21T00:00:00Z"}
	bundle := library.BundleRevision{Manifest: library.BundleManifest{BundleID: source.BundleID, Revision: 1, Artifacts: []library.Artifact{{ID: source.ArtifactID, Role: source.Role, SizeBytes: source.SizeBytes, SHA256: source.SHA256, Rights: []library.Use{library.UseRender}}}}, ManifestSHA256: source.ManifestSHA256, Review: library.ReviewBinding{Decision: "approved", ManifestSHA256: source.ManifestSHA256, AllowedUses: []library.Use{library.UseRender}}, ReviewSHA256: source.ReviewSHA256}
	sceneBundle := library.BundleRevision{Manifest: library.BundleManifest{BundleID: sceneRef.BundleID, Revision: 1, Artifacts: []library.Artifact{{ID: sceneRef.ArtifactID, Role: sceneRef.Role, SizeBytes: sceneRef.SizeBytes, SHA256: sceneRef.SHA256, Rights: []library.Use{library.UseRender}}}}, ManifestSHA256: sceneRef.ManifestSHA256, Review: library.ReviewBinding{Decision: "approved", ManifestSHA256: sceneRef.ManifestSHA256, AllowedUses: []library.Use{library.UseRender}}, ReviewSHA256: sceneRef.ReviewSHA256}
	proof := renderjobs.QualificationRenderEvidence{DeviceID: "device", DeviceRevision: 2, DeviceSHA256: parent.SHA256, SceneID: "scene", SceneRevision: 3, SceneSHA256: repeated('e'), MotionID: "motion", MotionRevision: 4, MotionSHA256: repeated('f'), DisplayID: "display", DeviceSupportID: "base", SceneSupportID: "table", CameraID: "camera", OutputSHA256: outputSHA, OutputKind: "physical_still_png", DeviceRootRest: physical.TRS{Translation: physical.Vec3{}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, SceneSupportWorld: physical.TRS{Translation: physical.Vec3{}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}}
	sceneRevision := scenes.Revision{SceneID: "scene", Revision: 3, SHA256: review.SceneSHA256, Candidate: scenes.SceneCandidate{SceneID: "scene", Artifacts: []scenes.SceneArtifactBinding{{ID: "room", Ref: sceneRef}}}}
	pkg := qualificationPackage{Version: 2, Origin: qualificationPackageOrigin{ProjectID: "project", JobID: "job", Reviewed: review.ReviewedAt}, Candidate: parent, SourceBundles: []library.BundleRevision{bundle, sceneBundle}, SourceArtifacts: []qualificationPackageArtifact{{Ref: source, Bytes: geometry}, {Ref: sceneRef, Bytes: sceneBytes}}, Review: review, Render: proof, Output: output}
	pkg.Scene, pkg.Motion, pkg.Preset = sceneRevision, motions.Revision{MotionID: "motion", Revision: 4, SHA256: review.MotionSHA256}, preset
	encoded, err := sealQualificationPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	deviceOwner := &qualificationDeviceStub{parent: parent, head: parent}
	presetOwner := &qualificationPresetStub{original: preset, head: preset}
	sourceStub := packageSourceStub{bytes: geometry, revision: bundle, bundles: map[string]library.BundleRevision{qualificationBundleKey(bundle.Manifest.BundleID, bundle.Manifest.Revision): bundle, qualificationBundleKey(sceneBundle.Manifest.BundleID, sceneBundle.Manifest.Revision): sceneBundle}, assets: map[string][]byte{qualificationArtifactKey(source): geometry, qualificationArtifactKey(sceneRef): sceneBytes}}
	service := QualificationService{devices: deviceOwner, bundles: sourceStub, scenes: packageSceneStub{pkg.Scene}, motions: qualificationMotionStub{motions.Revision{MotionID: "motion", Revision: 4, SHA256: review.MotionSHA256, State: motions.RevisionCandidate}}, presets: presetOwner, validator: qualificationValidatorStub{}, renders: qualificationRenderStub{}}
	missing := service
	missingStub := sourceStub
	missingStub.missing = map[string]error{qualificationArtifactKey(sceneRef): library.ErrArtifactNotFound}
	missing.bundles = missingStub
	if _, err := missing.ImportReviewPackage(ctx, encoded, parent.Revision); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("missing destination source reference accepted: %v", err)
	}
	tamperedSource := service
	tamperedStub := sourceStub
	tamperedStub.assets = map[string][]byte{qualificationArtifactKey(source): geometry, qualificationArtifactKey(sceneRef): []byte("bad")}
	tamperedSource.bundles = tamperedStub
	if _, err := tamperedSource.ImportReviewPackage(ctx, encoded, parent.Revision); err == nil || !strings.Contains(err.Error(), "source bytes") {
		t.Fatalf("different destination source bytes accepted: %v", err)
	}
	first, err := service.ImportReviewPackage(ctx, encoded, parent.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision.State != devices.DeviceRevisionQualified || deviceOwner.applies != 1 || presetOwner.saves != 1 {
		t.Fatalf("package did not append one scoped immutable review and refresh its preset: result=%+v apply=%d save=%d", first, deviceOwner.applies, presetOwner.saves)
	}
	second, err := service.ImportReviewPackage(ctx, encoded, parent.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if second.Revision.Revision != first.Revision.Revision || deviceOwner.applies != 1 || presetOwner.saves != 1 {
		t.Fatalf("exact package retry appended another revision: first=%+v second=%+v applies=%d saves=%d", first, second, deviceOwner.applies, presetOwner.saves)
	}
}
