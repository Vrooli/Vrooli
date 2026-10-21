package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path"
	"strings"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

//go:embed assets/**
var starterAssets embed.FS

const (
	starterReviewTime = "2026-10-20T16:46:19Z"
	// The library's stable reviewer identifier uses the safe-ID spelling of
	// the human reviewer label root/device-studio-delivery.
	starterReviewer = "root-device-studio-delivery"
	phoneBundleID   = "device-studio-starter-authored-phone-neutral"
	sceneBundleID   = "device-studio-starter-phone-study-environment"

	ReviewedPhoneStudyPlacementProfileID      = "device-studio-reviewed-phone-study-placement-v1"
	ReviewedPhoneStudyPlacementProfileVersion = uint32(1)
)

type StarterState struct {
	PhoneBundle         library.BundleRevision
	SceneBundle         library.BundleRevision
	Phone               devices.DeviceRevision
	Scene               scenes.Revision
	Motion              motions.Revision
	PhoneStudyPlacement ReviewedPhoneStudyPlacement
	Preset              shots.PhysicalPresetRevision
}

// ReviewedPhoneStudyPlacement is a versioned, candidate-bound starting pose.
// It is returned with the exact candidate revisions from this initializer.
type ReviewedPhoneStudyPlacement struct {
	ProfileID         string
	ProfileVersion    uint32
	DisplayID         string
	DeviceSupportID   string
	SceneSupportID    string
	SelectedCameraID  string
	SceneSupportWorld physical.TRS
	SupportPivotM     physical.Vec3
	DeviceRootRest    physical.TRS
}

type starterInitializer struct {
	db      *database.RoutedDB
	roots   *filerouting.RoutedRoots
	library *library.Service
	devices *devices.DeviceRevisionService
	scenes  *scenes.RevisionService
	motions *motions.RevisionService
	presets *shots.PhysicalPresetService
	fs      fs.FS
}

func newStarterInitializer(db *database.RoutedDB, roots *filerouting.RoutedRoots, libraryOwner *library.Service, deviceOwner *devices.DeviceRevisionService, sceneOwner *scenes.RevisionService, motionOwner *motions.RevisionService, presets *shots.PhysicalPresetService, assets fs.FS) (*starterInitializer, error) {
	if db == nil || roots == nil || libraryOwner == nil || deviceOwner == nil || sceneOwner == nil || motionOwner == nil || presets == nil || assets == nil {
		return nil, errors.New("starter initialization requires library, device, scene, and motion owners")
	}
	return &starterInitializer{db: db, roots: roots, library: libraryOwner, devices: deviceOwner, scenes: sceneOwner, motions: motionOwner, presets: presets, fs: assets}, nil
}

func (s *starterInitializer) Initialize(ctx context.Context) (StarterState, error) {
	ctx, err := s.pinRoute(ctx)
	if err != nil {
		return StarterState{}, err
	}
	return s.initializePinned(ctx)
}

func (s *starterInitializer) initializePinned(ctx context.Context) (StarterState, error) {
	phoneBundle, err := s.publishPhoneBundle(ctx)
	if err != nil {
		return StarterState{}, err
	}
	sceneBundle, err := s.publishSceneBundle(ctx)
	if err != nil {
		return StarterState{}, err
	}
	phone, err := s.savePhoneCandidate(ctx, phoneBundle)
	if err != nil {
		return StarterState{}, err
	}
	scene, err := s.saveSceneCandidate(ctx, sceneBundle)
	if err != nil {
		return StarterState{}, err
	}
	motion, err := s.saveStillMotion(ctx)
	if err != nil {
		return StarterState{}, err
	}
	placement, err := reviewedPhoneStudyPlacement(phone, scene, motion)
	if err != nil {
		return StarterState{}, err
	}
	preset, err := s.ensurePhoneStudyPreset(ctx, phone, scene, motion, placement)
	if err != nil {
		return StarterState{}, err
	}
	return StarterState{PhoneBundle: phoneBundle, SceneBundle: sceneBundle, Phone: phone, Scene: scene, Motion: motion, PhoneStudyPlacement: placement, Preset: preset}, nil
}

func (s *starterInitializer) ensurePhoneStudyPreset(ctx context.Context, phone devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedPhoneStudyPlacement) (shots.PhysicalPresetRevision, error) {
	const presetID = "phone-study-hero"
	if current, err := s.presets.Head(ctx, presetID); err == nil {
		return current, nil
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: phone.DeviceID, Revision: phone.Revision, Sha256: phone.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles:  []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1, shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1},
		MotionProfiles: []shotsv1.PhysicalMotionOutputProfile{shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1, shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2},
	}
	return s.presets.Save(ctx, presetID, 0, "Phone hero on study table", setup)
}

func starterVec3(value physical.Vec3) *physicalv1.Vec3 {
	return &physicalv1.Vec3{X: value.X, Y: value.Y, Z: value.Z}
}

func starterTRS(value physical.TRS) *physicalv1.TRS {
	return &physicalv1.TRS{TranslationM: starterVec3(value.Translation), RotationXyzw: &physicalv1.QuatXYZW{X: value.Rotation.X, Y: value.Rotation.Y, Z: value.Rotation.Z, W: value.Rotation.W}, Scale: starterVec3(value.Scale)}
}

func reviewedPhoneStudyPlacement(phone devices.DeviceRevision, scene scenes.Revision, motion motions.Revision) (ReviewedPhoneStudyPlacement, error) {
	if phone.DeviceID != "authored-phone-neutral" || scene.SceneID != "phone-study-environment" || motion.MotionID != "phone-still-identity" {
		return ReviewedPhoneStudyPlacement{}, errors.New("reviewed phone placement is bound to different starter candidates")
	}
	display, displayBound, deviceSupport := false, false, false
	for _, item := range phone.Candidate.Definition.Displays {
		display = display || item.ID == "main_display"
	}
	for _, item := range phone.Candidate.DisplayBindings {
		displayBound = displayBound || item.DisplayID == "main_display" && item.PartID == "phone-screen"
	}
	for _, item := range phone.Candidate.SupportFrames {
		deviceSupport = deviceSupport || item.ID == "device-root-support" && item.PartID == "phone-root"
	}
	sceneSupport, camera := false, false
	for _, item := range scene.Candidate.SupportFrames {
		sceneSupport = sceneSupport || item.ID == "table-top-flat"
	}
	for _, item := range scene.Candidate.CameraFrames {
		camera = camera || item.ID == "phone-study-camera"
	}
	if !display || !displayBound || !deviceSupport || !sceneSupport || !camera {
		return ReviewedPhoneStudyPlacement{}, errors.New("reviewed phone placement IDs are not present in the pinned starter candidates")
	}
	return ReviewedPhoneStudyPlacement{
		ProfileID: ReviewedPhoneStudyPlacementProfileID, ProfileVersion: ReviewedPhoneStudyPlacementProfileVersion,
		DisplayID: "main_display", DeviceSupportID: "device-root-support", SceneSupportID: "table-top-flat", SelectedCameraID: "phone-study-camera",
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		SupportPivotM:     physical.Vec3{Y: .0071073174476624},
		DeviceRootRest:    physical.TRS{Rotation: physical.QuatXYZW{X: -.6885449545442038, Y: -.08161248882759843, Z: -.07844978823902271, W: .7163036213536392}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
	}, nil
}

func (s *starterInitializer) pinRoute(ctx context.Context) (context.Context, error) {
	if !database.IsTestMode(ctx) {
		if _, err := s.db.PoolForContext(ctx); err != nil {
			return nil, fmt.Errorf("select primary starter database route: %w", err)
		}
		if _, err := s.roots.Pick(ctx, storage.ClassData); err != nil {
			return nil, fmt.Errorf("select primary starter file route: %w", err)
		}
		return ctx, nil
	}
	databaseLease := s.db.ActiveLeaseID()
	fileLease := s.roots.ActiveLeaseID()
	if databaseLease == "" || databaseLease != fileLease {
		return nil, errors.New("reviewed starter requires matching active database and file leases")
	}
	pinned, err := s.db.PinTestPool(ctx, databaseLease)
	if err != nil {
		return nil, fmt.Errorf("pin starter database route: %w", err)
	}
	pinned, err = s.roots.PinTestRoots(pinned, fileLease)
	if err != nil {
		return nil, fmt.Errorf("pin starter file route: %w", err)
	}
	if _, err := s.db.PoolForContext(pinned); err != nil {
		return nil, fmt.Errorf("verify pinned starter database route: %w", err)
	}
	if _, err := s.roots.Pick(pinned, storage.ClassData); err != nil {
		return nil, fmt.Errorf("verify pinned starter file route: %w", err)
	}
	return pinned, nil
}

type starterFile struct {
	ID, File, MIME, SHA, License, Attribution string
	Role                                      library.ArtifactRole
	Provenance                                library.ArtifactProvenance
	Data                                      []byte
}

func (s *starterInitializer) publishPhoneBundle(ctx context.Context) (library.BundleRevision, error) {
	files := []starterFile{{
		ID: "authored-phone-neutral-glb", File: "assets/phone/authored-phone-neutral.glb", MIME: "model/gltf-binary",
		SHA: "152c02883f75affc15185194f4347c256fef91cc83541262744678c40c761fca", Role: library.RoleGeometryGLB,
		License: "AGPL-3.0-only", Attribution: "Vrooli-authored generic phone presentation model; not CAD, a device measurement, or manufacturer-endorsed.",
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Embedded Vrooli-authored neutral phone starter; research display pixels removed."},
	}}
	manifest, sources, err := s.buildManifest(ctx, phoneBundleID, files)
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, "device-studio-e3-starter-local-render-review-v1-phone")
}

func (s *starterInitializer) publishSceneBundle(ctx context.Context) (library.BundleRevision, error) {
	files := []starterFile{
		{ID: "phone-study-environment-glb", File: "assets/phone-study-environment/phone-study-environment.glb", MIME: "model/gltf-binary", SHA: "c0f1450af8cc83b237159f2c5208bd964c48fc5dc9fe7f2ac2b2ee5409447f4d", Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored static phone-study environment; not measured room or furniture CAD.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Embedded Vrooli-authored full phone-study environment GLB."}},
		{ID: "phone-table-scene-glb", File: "assets/phone-table/phone-table-scene.glb", MIME: "model/gltf-binary", SHA: "d03c288c1f6fa401d920c736e1361189066599b0c514f924b9fed8b50ba16587", Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored procedural table/cup geometry; not measured furniture or CAD.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Embedded Vrooli-authored phone-table scene geometry."}},
		{ID: "travertine-color", File: "assets/phone-table/textures/travertine-color.jpg", MIME: "image/jpeg", SHA: "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 color texture, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8"}},
		{ID: "travertine-normal-gl", File: "assets/phone-table/textures/travertine-normal-gl.jpg", MIME: "image/jpeg", SHA: "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 OpenGL tangent +Y normal map, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b"}},
		{ID: "travertine-roughness", File: "assets/phone-table/textures/travertine-roughness.jpg", MIME: "image/jpeg", SHA: "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 roughness map, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90"}},
		{ID: "travertine-height", File: "assets/phone-table/textures/travertine-height.jpg", MIME: "image/jpeg", SHA: "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", Role: library.RoleTextureImage, License: "CC0-1.0", Attribution: "ambientCG Travertine001 height map used only for subtle bump, CC0-1.0.", Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAcquired, SourceLabel: "ambientCG Travertine001; acquisition record 4268b08091b930230a90de78c6e82f793c887fb870ec0adc28bf923317153f4a.", SourceSHA256: "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf"}},
	}
	for i := range files {
		files[i].Role = library.RoleTextureImage
		files[i].Provenance.Kind = library.ProvenanceAcquired
	}
	files[0].Role, files[1].Role = library.RoleGeometryGLB, library.RoleGeometryGLB
	files[0].Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Embedded Vrooli-authored full phone-study environment GLB."}
	files[1].Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: "Embedded Vrooli-authored phone-table scene geometry."}
	manifest, sources, err := s.buildManifest(ctx, sceneBundleID, files)
	if err != nil {
		return library.BundleRevision{}, err
	}
	return s.publish(ctx, manifest, sources, "device-studio-e3-starter-local-render-review-v1-scene")
}

func (s *starterInitializer) buildManifest(ctx context.Context, bundleID string, files []starterFile) (library.BundleManifest, map[string]io.Reader, error) {
	manifest := library.BundleManifest{Version: library.ManifestVersion, BundleID: bundleID, Revision: 1}
	sources := make(map[string]io.Reader, len(files))
	for _, file := range files {
		if !strings.HasPrefix(file.File, "assets/") || path.Clean(file.File) != file.File {
			return library.BundleManifest{}, nil, errors.New("starter asset path is not a fixed embedded path")
		}
		data := file.Data
		if data == nil {
			var err error
			data, err = fs.ReadFile(s.fs, file.File)
			if err != nil {
				return library.BundleManifest{}, nil, fmt.Errorf("read embedded starter asset %s: %w", file.ID, err)
			}
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA {
			return library.BundleManifest{}, nil, fmt.Errorf("embedded starter asset %s differs from its reviewed source digest", file.ID)
		}
		artifact := library.Artifact{
			ID: file.ID, Role: file.Role, MIME: file.MIME, SizeBytes: int64(len(data)), SHA256: file.SHA,
			LicenseID: file.License, Attribution: file.Attribution, Provenance: file.Provenance,
			Rights: []library.Use{library.UseEdit, library.UseRender},
		}
		manifest.Artifacts = append(manifest.Artifacts, artifact)
		sources[file.ID] = bytes.NewReader(data)
	}
	return manifest, sources, nil
}

func (s *starterInitializer) publish(ctx context.Context, manifest library.BundleManifest, sources map[string]io.Reader, reviewID string) (library.BundleRevision, error) {
	_, _, digest, err := library.CanonicalizeManifest(manifest)
	if err != nil {
		return library.BundleRevision{}, err
	}
	reviewTime, err := time.Parse(time.RFC3339, starterReviewTime)
	if err != nil {
		return library.BundleRevision{}, err
	}
	revision, err := s.library.Publish(ctx, library.PublishInput{Manifest: manifest, Sources: sources, Review: library.ReviewBinding{
		ID: reviewID, ReviewerID: starterReviewer, ReviewedAt: reviewTime, Decision: "approved", ManifestSHA256: digest,
		AllowedUses: []library.Use{library.UseEdit, library.UseRender},
	}})
	if err != nil {
		return library.BundleRevision{}, fmt.Errorf("publish reviewed starter bundle %s: %w", manifest.BundleID, err)
	}
	return revision, nil
}

func (s *starterInitializer) savePhoneCandidate(ctx context.Context, bundle library.BundleRevision) (devices.DeviceRevision, error) {
	ref := artifactRef(bundle, "authored-phone-neutral-glb")
	data, err := fs.ReadFile(s.fs, "assets/phone/authored-phone-neutral.glb")
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("inspect embedded starter phone: %w", err)
	}
	byName := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		byName[node.Name] = node
	}
	root, rootOK := byName["IPHONE_16_PRO_ORIGINAL_MASTER"]
	screen, screenOK := byName["actual frontend capture - contained no stretch"]
	island, islandOK := byName["Dynamic Island black sensor cutout"]
	if !rootOK || !screenOK || !islandOK {
		return devices.DeviceRevision{}, errors.New("starter phone semantic nodes differ from reviewed inventory")
	}
	assemblyBounds, err := physicalGLBWorldBounds(inventory)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive starter phone envelope from pinned GLB: %w", err)
	}
	screenBounds, err := physicalGLBMeshBounds(inventory, *screen.Mesh)
	if err != nil {
		return devices.DeviceRevision{}, fmt.Errorf("derive starter display metric from pinned GLB: %w", err)
	}
	widthM := assemblyBounds.max.X - assemblyBounds.min.X
	heightM := assemblyBounds.max.Y - assemblyBounds.min.Y
	depthM := assemblyBounds.max.Z - assemblyBounds.min.Z
	if widthM <= 0 || heightM <= 0 || depthM <= 0 || screenBounds.max.X <= screenBounds.min.X || screenBounds.max.Y <= screenBounds.min.Y {
		return devices.DeviceRevision{}, fmt.Errorf("starter phone geometry has an empty envelope or display metric: body=%+v display=%+v", assemblyBounds, screenBounds)
	}
	geometryEvidence := []devices.Evidence{{Source: "embedded authored-phone-neutral.glb exact SHA-256 " + ref.SHA256, Kind: "derived_asset_geometry", Qualifier: "geometry-derived candidate values; not manufacturer measurements or physical qualification"}}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: "authored-phone-neutral", Group: devices.GroupPhone,
		Kind: "generic_smartphone", IdentityStatus: devices.IdentityGeneric, Model: "Vrooli authored generic phone", Variant: "neutral-display starter",
		Description: "Authored generic presentation model, not manufacturer CAD or a physical-device measurement.",
		Dimensions: map[string]devices.Dimensions{"portrait_unrotated": {
			Width: derivedGeometryMeasurement(widthM*1000, "body X span"), Height: derivedGeometryMeasurement(heightM*1000, "body Y span"), Depth: derivedGeometryMeasurement(depthM*1000, "body Z span"),
		}},
		Displays: []devices.DisplaySurface{{
			ID: "main_display", Topology: "planar",
			CoordinateMetric: &devices.DisplayMetric{WidthMM: (screenBounds.max.X - screenBounds.min.X) * 1000, HeightMM: (screenBounds.max.Y - screenBounds.min.Y) * 1000, Evidence: geometryEvidence},
			PhysicalBoundary: unknownRegion(), ActiveArea: unknownRegion(), AppSafeArea: unknownRegion(),
			HardwareOcclusions: []devices.HardwareOcclusion{{ID: "dynamic_island", Kind: "sensor_cutout", Region: unknownRegion()}},
		}},
		SupportedMotion: []devices.Capability{{Name: "rigid_root_transform", Status: "candidate_only", MotionKind: "rigid_body"}},
		Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only"}},
		ModelCandidates: []devices.ModelCandidate{{ID: "embedded-neutral-phone", Status: "candidate_only", SHA256: ref.SHA256, Author: "Vrooli", License: "AGPL-3.0-only", RightsNotes: "Generic authored starter; not CAD or manufacturer-endorsed."}},
		Provenance:      []devices.Evidence{{Source: "embedded phone/asset-manifest.json and phone-starter-inventory.json", Kind: "authored_asset_manifest", Qualifier: "derived geometry only; no device measurement claim"}},
		Qualification:   "unqualified",
	}
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: physical.TRS{
			Translation: physical.Vec3{X: -(assemblyBounds.min.X + assemblyBounds.max.X) / 2, Y: -(assemblyBounds.min.Y + assemblyBounds.max.Y) / 2, Z: -(assemblyBounds.min.Z + assemblyBounds.max.Z) / 2},
			Rotation:    physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1},
		},
		Parts: []devices.SemanticPart{
			{ID: "phone-root", Role: "device_root", NodeIndex: root.Index, NodeName: root.Name, LocalFrame: root.Frame},
			{ID: "phone-screen", Role: "display_surface", ParentID: "phone-root", NodeIndex: screen.Index, NodeName: screen.Name, LocalFrame: screen.Frame},
			{ID: "phone-island", Role: "sensor_cutout", ParentID: "phone-root", NodeIndex: island.Index, NodeName: island.Name, LocalFrame: island.Frame},
		},
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: "main_display", PartID: "phone-screen", LocalFrame: identityTRS()}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: "device-root-support", InterfaceID: "horizontal_surface", PartID: "phone-root", LocalFrame: identityTRS()}},
	}
	existing, err := s.devices.GetHead(ctx, candidate.Definition.ID)
	if err == nil {
		_, _, digest, canonErr := devices.CanonicalizeDeviceCandidate(candidate)
		if canonErr != nil {
			return devices.DeviceRevision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return devices.DeviceRevision{}, fmt.Errorf("starter phone head differs from its fixed candidate; refusing implicit revision: %w", devices.ErrRevisionConflict)
	}
	if !errors.Is(err, devices.ErrRevisionNotFound) {
		return devices.DeviceRevision{}, err
	}
	return s.devices.SaveCandidate(ctx, candidate, 0)
}

func (s *starterInitializer) saveSceneCandidate(ctx context.Context, bundle library.BundleRevision) (scenes.Revision, error) {
	envRef := artifactRef(bundle, "phone-study-environment-glb")
	textureIDs := map[string]string{"base_color": "travertine-color", "normal": "travertine-normal-gl", "roughness": "travertine-roughness", "bump_height": "travertine-height"}
	data, err := fs.ReadFile(s.fs, "assets/phone-study-environment/phone-study-environment.glb")
	if err != nil {
		return scenes.Revision{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return scenes.Revision{}, fmt.Errorf("inspect embedded scene geometry: %w", err)
	}
	topMaterialIndex, topFound := namedMaterialIndex(inventory.Materials, "Travertine001 top material slot")
	windowMaterialIndex, windowFound := namedMaterialIndex(inventory.Materials, "Soft luminous window")
	if !topFound || !windowFound {
		return scenes.Revision{}, errors.New("starter scene material names differ from the pinned authored inventory")
	}
	artifactBindings := []scenes.SceneArtifactBinding{{ID: "environment-geometry", Ref: envRef, SourceUnit: "m", SourceToCanonical: ptrTRS(identityTRS())}}
	for _, artifactID := range textureIDs {
		ref := artifactRef(bundle, artifactID)
		artifactBindings = append(artifactBindings, scenes.SceneArtifactBinding{ID: artifactID, Ref: ref})
	}
	materialID := func(i uint32) string { return fmt.Sprintf("material-%d", i) }
	materials := starterTableMaterials(inventory.Materials, materialID, topMaterialIndex, windowMaterialIndex)
	parts := make([]scenes.ScenePart, 0, len(inventory.Nodes))
	partIDByNode := make(map[uint32]string, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		partIDByNode[node.Index] = fmt.Sprintf("scene-node-%d", node.Index)
	}
	for _, node := range inventory.Nodes {
		role := "scene_geometry"
		if node.Name == "TableTop" {
			role = "support_surface"
		}
		parent := ""
		if node.Parent != nil {
			parent = partIDByNode[*node.Parent]
		}
		parts = append(parts, scenes.ScenePart{ID: partIDByNode[node.Index], Role: role, ArtifactBindingID: "environment-geometry", ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	slots := make([]scenes.SceneMaterialSlot, 0)
	for _, node := range inventory.Nodes {
		if node.Mesh == nil {
			continue
		}
		mesh := inventory.Meshes[*node.Mesh]
		for _, primitive := range mesh.Primitives {
			slots = append(slots, scenes.SceneMaterialSlot{PartID: partIDByNode[node.Index], PrimitiveIndex: primitive.Index, SlotIndex: primitive.Material, MaterialID: materialID(primitive.Material)})
		}
	}
	candidate := scenes.SceneCandidate{
		Version: scenes.CandidateVersion, SceneID: "phone-study-environment", CoordinateConvention: physical.CoordinateConvention,
		Artifacts: artifactBindings, Materials: materials, MaterialSlots: slots, Parts: parts,
		SupportFrames: []scenes.SceneSupportFrame{{ID: "table-top-flat", InterfaceID: "horizontal_surface", FramePartID: partIDByNode[5], LocalFrame: physical.TRS{Translation: physical.Vec3{Y: .0170000196}, Rotation: identityTRS().Rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, ContactRegionPartID: partIDByNode[5], LoadDirectionLocal: &physical.Vec3{Y: -1}}},
		CameraFrames:  []scenes.SceneCameraFrame{{ID: "phone-study-camera", Frame: lookAtFrame(physical.Vec3{X: .14, Y: 1.22, Z: .42}, physical.Vec3{Y: .75}), Projection: scenes.ProjectionPerspective, FocalLengthMM: floatPtr(85), SensorWidthMM: floatPtr(36), SensorHeightMM: floatPtr(24), SensorFit: scenes.SensorFitAuto, NearClipM: .1, FarClipM: 1000, Focus: &scenes.FocusTarget{Mode: scenes.FocusSelectedDevice}, DOFFNumber: floatPtr(11)}},
		LightFrames:   starterLights(), Environment: scenes.SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{.53, .56, .60}, Strength: .2},
	}
	existing, err := s.scenes.GetHead(ctx, candidate.SceneID)
	if err == nil {
		_, _, digest, canonErr := scenes.CanonicalizeCandidate(candidate)
		if canonErr != nil {
			return scenes.Revision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return scenes.Revision{}, fmt.Errorf("starter scene head differs from its fixed candidate; refusing implicit revision: %w", scenes.ErrRevisionConflict)
	}
	if !errors.Is(err, scenes.ErrRevisionNotFound) {
		return scenes.Revision{}, err
	}
	return s.scenes.SaveCandidate(ctx, candidate, 0)
}

func (s *starterInitializer) saveStillMotion(ctx context.Context) (motions.Revision, error) {
	zero := motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}
	candidate := motions.MotionCandidate{Version: motions.CandidateVersion, MotionID: "phone-still-identity", Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 24, Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Step, Keys: []motions.MotionKey{{Tick: 0, Transform: &zero}, {Tick: 24, Transform: &zero}}}}}
	existing, err := s.motions.GetHead(ctx, candidate.MotionID)
	if err == nil {
		_, _, digest, canonErr := motions.CanonicalizeCandidate(candidate)
		if canonErr != nil {
			return motions.Revision{}, canonErr
		}
		if existing.SHA256 == digest {
			return existing, nil
		}
		return motions.Revision{}, fmt.Errorf("starter motion head differs from its fixed candidate; refusing implicit revision: %w", motions.ErrRevisionConflict)
	}
	if !errors.Is(err, motions.ErrRevisionNotFound) {
		return motions.Revision{}, err
	}
	return s.motions.SaveCandidate(ctx, candidate, 0)
}

func artifactRef(bundle library.BundleRevision, id string) physical.ArtifactRef {
	for _, artifact := range bundle.Manifest.Artifacts {
		if artifact.ID == id {
			return physical.ArtifactRef{BundleID: bundle.Manifest.BundleID, BundleRevision: bundle.Manifest.Revision, ManifestSHA256: bundle.ManifestSHA256, ReviewSHA256: bundle.ReviewSHA256, ArtifactID: artifact.ID, Role: artifact.Role, SizeBytes: artifact.SizeBytes, SHA256: artifact.SHA256}
		}
	}
	return physical.ArtifactRef{}
}

func derivedGeometryMeasurement(valueMM float64, axis string) devices.Measurement {
	value := valueMM
	return devices.Measurement{
		Value: &value, Unit: "mm", Meaning: "derived " + axis + " from pinned authored GLB geometry",
		Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown"},
		Evidence: []devices.Evidence{{Source: "embedded authored-phone-neutral.glb", Kind: "derived_asset_geometry", Qualifier: "candidate geometry span only; not a measured physical-device dimension"}},
	}
}

func namedMaterialIndex(materials []rendering.PhysicalGLBMaterial, name string) (uint32, bool) {
	var found bool
	var index uint32
	for _, material := range materials {
		if material.Name != name {
			continue
		}
		if found {
			return 0, false
		}
		found, index = true, material.Index
	}
	return index, found
}

func unknownRegion() devices.Region { return devices.Region{Status: "unknown", Shape: "unknown"} }

type physicalBounds struct{ min, max physical.Vec3 }

func physicalGLBMeshBounds(inventory rendering.PhysicalGLBInventory, meshIndex uint32) (physicalBounds, error) {
	var bounds physicalBounds
	bounds.min = physical.Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	bounds.max = physical.Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, mesh := range inventory.Meshes {
		if mesh.Index != meshIndex {
			continue
		}
		for _, primitive := range mesh.Primitives {
			bounds.include(physical.Vec3{X: primitive.PositionMin[0], Y: primitive.PositionMin[1], Z: primitive.PositionMin[2]})
			bounds.include(physical.Vec3{X: primitive.PositionMax[0], Y: primitive.PositionMax[1], Z: primitive.PositionMax[2]})
		}
		if !bounds.valid() {
			return physicalBounds{}, errors.New("GLB mesh has no bounded position primitives")
		}
		return bounds, nil
	}
	return physicalBounds{}, fmt.Errorf("GLB mesh %d is absent", meshIndex)
}

func physicalGLBWorldBounds(inventory rendering.PhysicalGLBInventory) (physicalBounds, error) {
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	meshes := make(map[uint32]rendering.PhysicalGLBMesh, len(inventory.Meshes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	for _, mesh := range inventory.Meshes {
		meshes[mesh.Index] = mesh
	}
	world := make(map[uint32]physical.Matrix4, len(nodes))
	visiting := make(map[uint32]bool, len(nodes))
	var matrixFor func(uint32) (physical.Matrix4, error)
	matrixFor = func(index uint32) (physical.Matrix4, error) {
		if matrix, ok := world[index]; ok {
			return matrix, nil
		}
		if visiting[index] {
			return physical.Matrix4{}, errors.New("GLB node hierarchy contains a cycle")
		}
		node, ok := nodes[index]
		if !ok {
			return physical.Matrix4{}, fmt.Errorf("GLB parent node %d is absent", index)
		}
		visiting[index] = true
		local, err := node.Frame.Matrix()
		if err != nil {
			return physical.Matrix4{}, err
		}
		if node.Parent != nil {
			parent, err := matrixFor(*node.Parent)
			if err != nil {
				return physical.Matrix4{}, err
			}
			local, err = physical.Compose(parent, local)
			if err != nil {
				return physical.Matrix4{}, err
			}
		}
		visiting[index] = false
		world[index] = local
		return local, nil
	}
	bounds := physicalBounds{min: physical.Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, max: physical.Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
	for _, node := range inventory.Nodes {
		if node.Mesh == nil {
			continue
		}
		mesh, ok := meshes[*node.Mesh]
		if !ok {
			return physicalBounds{}, fmt.Errorf("GLB node %d references absent mesh %d", node.Index, *node.Mesh)
		}
		matrix, err := matrixFor(node.Index)
		if err != nil {
			return physicalBounds{}, err
		}
		for _, primitive := range mesh.Primitives {
			for mask := 0; mask < 8; mask++ {
				point := physical.Vec3{}
				for axis := 0; axis < 3; axis++ {
					value := primitive.PositionMin[axis]
					if mask&(1<<axis) != 0 {
						value = primitive.PositionMax[axis]
					}
					switch axis {
					case 0:
						point.X = value
					case 1:
						point.Y = value
					case 2:
						point.Z = value
					}
				}
				transformed, err := physical.TransformPoint(matrix, point)
				if err != nil {
					return physicalBounds{}, err
				}
				bounds.include(transformed)
			}
		}
	}
	if !bounds.valid() {
		return physicalBounds{}, errors.New("GLB has no bounded mesh geometry")
	}
	return bounds, nil
}

func (b *physicalBounds) include(point physical.Vec3) {
	b.min.X = math.Min(b.min.X, point.X)
	b.min.Y = math.Min(b.min.Y, point.Y)
	b.min.Z = math.Min(b.min.Z, point.Z)
	b.max.X = math.Max(b.max.X, point.X)
	b.max.Y = math.Max(b.max.Y, point.Y)
	b.max.Z = math.Max(b.max.Z, point.Z)
}

func (b physicalBounds) valid() bool {
	return finite(b.min.X) && finite(b.min.Y) && finite(b.min.Z) && finite(b.max.X) && finite(b.max.Y) && finite(b.max.Z)
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func identityTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}
func ptrTRS(value physical.TRS) *physical.TRS { return &value }
func floatPtr(value float64) *float64         { return &value }

func starterLights() []scenes.SceneLightFrame {
	return []scenes.SceneLightFrame{
		{ID: "warm-window-key", Frame: lookAtFrame(physical.Vec3{X: -.48, Y: 1.62, Z: .12}, physical.Vec3{Y: .75}), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: scenes.LightRectangle, DimensionsM: [2]float64{.95, 1.25}, PowerWatts: 12, LinearRGB: [3]float64{1, .91, .8}},
		{ID: "neutral-broad-fill", Frame: lookAtFrame(physical.Vec3{X: .22, Y: 1.30, Z: .70}, physical.Vec3{Y: .75}), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: scenes.LightRectangle, DimensionsM: [2]float64{.75, .55}, PowerWatts: 2.8, LinearRGB: [3]float64{.91, .94, 1}},
		{ID: "cool-edge", Frame: lookAtFrame(physical.Vec3{X: .48, Y: 1.22, Z: -.22}, physical.Vec3{Y: .75}), LocalAxisProfile: "AREA_XY_MINUS_Z_V1", Shape: scenes.LightRectangle, DimensionsM: [2]float64{.24, .85}, PowerWatts: 1.6, LinearRGB: [3]float64{.8, .88, 1}},
	}
}

// lookAtFrame constructs a rigid +Y-up frame whose local -Z axis points at target.
func lookAtFrame(position, target physical.Vec3) physical.TRS {
	forward := normalize(physical.Vec3{X: target.X - position.X, Y: target.Y - position.Y, Z: target.Z - position.Z})
	up := physical.Vec3{Y: 1}
	right := normalize(cross(forward, up))
	actualUp := cross(right, forward)
	zAxis := physical.Vec3{X: -forward.X, Y: -forward.Y, Z: -forward.Z}
	q := quaternionFromColumns(right, actualUp, zAxis)
	return physical.TRS{Translation: position, Rotation: q, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

func quaternionFromColumns(x, y, z physical.Vec3) physical.QuatXYZW {
	m00, m01, m02 := x.X, y.X, z.X
	m10, m11, m12 := x.Y, y.Y, z.Y
	m20, m21, m22 := x.Z, y.Z, z.Z
	trace := m00 + m11 + m22
	var q physical.QuatXYZW
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1) * 2
		q = physical.QuatXYZW{W: .25 * s, X: (m21 - m12) / s, Y: (m02 - m20) / s, Z: (m10 - m01) / s}
	case m00 > m11 && m00 > m22:
		s := math.Sqrt(1+m00-m11-m22) * 2
		q = physical.QuatXYZW{W: (m21 - m12) / s, X: .25 * s, Y: (m01 + m10) / s, Z: (m02 + m20) / s}
	case m11 > m22:
		s := math.Sqrt(1+m11-m00-m22) * 2
		q = physical.QuatXYZW{W: (m02 - m20) / s, X: (m01 + m10) / s, Y: .25 * s, Z: (m12 + m21) / s}
	default:
		s := math.Sqrt(1+m22-m00-m11) * 2
		q = physical.QuatXYZW{W: (m10 - m01) / s, X: (m02 + m20) / s, Y: (m12 + m21) / s, Z: .25 * s}
	}
	canonical, err := physical.CanonicalizeQuaternion(q)
	if err != nil {
		return physical.QuatXYZW{W: 1}
	}
	return canonical
}

func normalize(v physical.Vec3) physical.Vec3 {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return physical.Vec3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

func cross(a, b physical.Vec3) physical.Vec3 {
	return physical.Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
