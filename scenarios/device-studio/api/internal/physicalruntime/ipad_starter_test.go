package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	_ "modernc.org/sqlite"
)

func ipadNativeBackend(value string) (shotsv1.PhysicalRenderBackend, error) {
	switch value {
	case "CPU":
		return shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU, nil
	case "CUDA":
		return shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CUDA, nil
	case "OPTIX":
		return shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX, nil
	default:
		return shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_UNSPECIFIED, errors.New("DEVICE_STUDIO_E4_IPAD_BACKEND must explicitly be CPU, CUDA, or OPTIX")
	}
}

func TestIPadNativeBackendRequiresExplicitSupportedChoice(t *testing.T) {
	for name, want := range map[string]shotsv1.PhysicalRenderBackend{
		"CPU":   shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU,
		"CUDA":  shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CUDA,
		"OPTIX": shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX,
	} {
		got, err := ipadNativeBackend(name)
		if err != nil || got != want {
			t.Fatalf("explicit backend %q: got=%v err=%v want=%v", name, got, err, want)
		}
	}
	for _, invalid := range []string{"", "auto", "CPU ", "METAL"} {
		if _, err := ipadNativeBackend(invalid); err == nil {
			t.Fatalf("backend %q was not rejected", invalid)
		}
	}
}

func TestIPadJobExecutionReleasedRequiresTerminalAndEveryAttemptReleased(t *testing.T) {
	claimed := &jobs.ExecutionClaim{ID: "execution"}
	for _, tc := range []struct {
		name string
		job  jobs.Job
		want bool
	}{
		{name: "queued job is not settled", job: jobs.Job{State: jobs.Queued}, want: false},
		{name: "terminal attempt without execution is settled", job: jobs.Job{State: jobs.Cancelled, Attempts: []jobs.Attempt{{State: jobs.AttemptCancelled}}}, want: true},
		{name: "terminal claimed attempt needs release proof", job: jobs.Job{State: jobs.Cancelled, Attempts: []jobs.Attempt{{Execution: claimed}}}, want: false},
		{name: "terminal released attempt is settled", job: jobs.Job{State: jobs.Failed, Attempts: []jobs.Attempt{{Execution: claimed, ExecutionReleasedAt: ptrTime(time.Now())}}}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ipadJobExecutionReleased(tc.job); got != tc.want {
				t.Fatalf("ipadJobExecutionReleased()=%v want %v", got, tc.want)
			}
		})
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestIPadEnvironmentV3SceneBindsActualExportInventory(t *testing.T) {
	data, err := starterAssets.ReadFile("assets/ipad-environment-v3/ipad-study-environment.glb")
	if err != nil {
		t.Fatal(err)
	}
	texture := func(id, sha string, size int64) library.Artifact {
		return library.Artifact{ID: id, Role: library.RoleTextureImage, MIME: "image/jpeg", SizeBytes: size, SHA256: sha}
	}
	bundle := library.BundleRevision{
		Manifest: library.BundleManifest{Version: library.ManifestVersion, BundleID: ipadEnvironmentV3BundleID, Revision: 1, Artifacts: []library.Artifact{
			{ID: ipadEnvironmentV3ArtifactID, Role: library.RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: ipadEnvironmentV3Size, SHA256: ipadEnvironmentV3SHA256},
			texture("travertine-color", "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", 16152869),
			texture("travertine-normal-gl", "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", 5152752),
			texture("travertine-roughness", "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", 5302649),
			texture("travertine-height", "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", 355941),
		}},
		ManifestSHA256: strings.Repeat("a", 64), ReviewSHA256: strings.Repeat("b", 64),
	}
	candidate, err := buildIPadEnvironmentV3Candidate(bundle, data, scenes.SceneCandidate{LightFrames: starterLights(), Environment: scenes.SceneEnvironment{Model: "CONSTANT_V1", Strength: .2}})
	if err != nil {
		t.Fatalf("build iPad environment candidate from exported inventory: %v", err)
	}
	inventory, err := rendering.ValidateScenePhysicalGLB(data, candidate)
	if err != nil {
		t.Fatalf("candidate must match the pinned exported GLB inventory: %v", err)
	}
	partsByIndex := make(map[uint32]scenes.ScenePart, len(candidate.Parts))
	for _, part := range candidate.Parts {
		partsByIndex[part.NodeIndex] = part
		if int(part.NodeIndex) >= len(inventory.Nodes) || inventory.Nodes[part.NodeIndex].Name != part.NodeName {
			t.Fatalf("scene part %q is not bound to actual node index %d", part.ID, part.NodeIndex)
		}
	}
	var tabletop rendering.PhysicalGLBNode
	for _, node := range inventory.Nodes {
		if node.Name == "TableTop" {
			tabletop = node
		}
	}
	if tabletop.Name == "" || partsByIndex[tabletop.Index].Role != "support_surface" || partsByIndex[tabletop.Index].NodeName != "TableTop" {
		t.Fatalf("support surface was not resolved from the exported node inventory: %+v", tabletop)
	}
	if len(candidate.CameraFrames) != 1 || candidate.CameraFrames[0].ID != ipadEnvironmentV3CameraID || candidate.SceneID != ipadEnvironmentV3SceneID {
		t.Fatalf("iPad v3 scene/camera identity differs: %+v", candidate)
	}
	if candidate.CameraFrames[0].Frame.Translation != (physical.Vec3{X: .1002994491448368, Y: 1.409043851488113, Z: .5300879317248927}) {
		t.Fatalf("camera fit translation changed from export6 manifest: %+v", candidate.CameraFrames[0].Frame.Translation)
	}
	for _, slot := range candidate.MaterialSlots {
		part := partsByIndex[slotNodeIndex(candidate.Parts, slot.PartID)]
		if int(part.NodeIndex) >= len(inventory.Nodes) || inventory.Nodes[part.NodeIndex].Mesh == nil {
			t.Fatalf("material slot is not bound to an actual mesh node: %+v", slot)
		}
		mesh := meshByIndex(inventory.Meshes, *inventory.Nodes[part.NodeIndex].Mesh)
		if mesh == nil || int(slot.PrimitiveIndex) >= len(mesh.Primitives) || mesh.Primitives[slot.PrimitiveIndex].Material != slot.SlotIndex {
			t.Fatalf("material slot does not match the exported primitive/material indices: %+v", slot)
		}
	}
}

func slotNodeIndex(parts []scenes.ScenePart, id string) uint32 {
	for _, part := range parts {
		if part.ID == id {
			return part.NodeIndex
		}
	}
	return ^uint32(0)
}

func cleanupIPadJob(t *testing.T, coordinator *renderjobs.Coordinator, db *database.RoutedDB, roots *filerouting.RoutedRoots, evidence, projectID, jobID, profileSHA, leaseID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(database.WithTestMode(context.Background()), 11*time.Minute)
	defer cancel()
	var err error
	ctx, err = db.PinTestPool(ctx, leaseID)
	if err != nil {
		t.Errorf("pin iPad job cleanup database: %v", err)
		return
	}
	ctx, err = roots.PinTestRoots(ctx, leaseID)
	if err != nil {
		t.Errorf("pin iPad job cleanup file roots: %v", err)
		return
	}
	var lastErr error
	for {
		job, getErr := coordinator.GetProjectJob(ctx, projectID, jobID)
		if getErr != nil {
			lastErr = fmt.Errorf("read iPad job during owner cleanup: %w", getErr)
		} else if ipadJobExecutionReleased(job) {
			if err := writeIPadExclusive(filepath.Join(evidence, "native-job-terminal.json"), marshalIPadJSON(map[string]any{
				"project_id": projectID, "job": job, "profile_sha256": profileSHA,
				"execution_released": true, "cleanup_owner": "renderjobs.Coordinator.GetProjectJob/Cancel/Advance",
			})); err != nil {
				t.Errorf("retain terminal iPad job and release evidence: %v", err)
			}
			return
		} else {
			if job.State == jobs.Queued || job.State == jobs.Running {
				_, lastErr = coordinator.Cancel(ctx, jobID, job.Version)
			} else {
				_, lastErr = coordinator.Advance(ctx, jobID)
			}
		}
		select {
		case <-ctx.Done():
			t.Errorf("iPad job owner cleanup did not settle before its deadline: last owner error=%v deadline=%v", lastErr, ctx.Err())
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func ipadJobExecutionReleased(job jobs.Job) bool {
	if job.State != jobs.Cancelled && job.State != jobs.Failed && job.State != jobs.Succeeded {
		return false
	}
	for _, attempt := range job.Attempts {
		if attempt.Execution != nil && attempt.ExecutionReleasedAt == nil {
			return false
		}
	}
	return true
}

func TestIPadCandidateUsesPinnedDerivativeAndCandidateOnlyMeasurements(t *testing.T) {
	b, err := starterAssets.ReadFile("assets/ipad/ipad-mini-screen-uv-normalized.glb")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1938936 || sha256Hex(b) != ipadDerivativeSHA256 {
		t.Fatalf("embedded iPad bytes differ: size=%d sha=%s", len(b), sha256Hex(b))
	}
	oriented, err := starterAssets.ReadFile("assets/ipad/ipad-mini-screen-uv-oriented-180-v2.glb")
	if err != nil || len(oriented) != 1939592 || sha256Hex(oriented) != ipadOrientedGLBSHA256 {
		t.Fatalf("embedded iPad oriented bytes differ: size=%d sha=%s err=%v", len(oriented), sha256Hex(oriented), err)
	}
	beforeUV, err := ipadScreenUVs(b, 36)
	if err != nil {
		t.Fatal(err)
	}
	afterUV, err := ipadScreenUVs(oriented, 36)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeUV) != 74 || len(afterUV) != len(beforeUV) {
		t.Fatalf("screen UV vertex counts differ: before=%d after=%d", len(beforeUV), len(afterUV))
	}
	for index := range beforeUV {
		if math.Abs(afterUV[index][0]-(1-beforeUV[index][0])) > 1e-7 || math.Abs(afterUV[index][1]-(1-beforeUV[index][1])) > 1e-7 {
			t.Fatalf("screen UV vertex %d was not rotated 180 degrees: before=%v after=%v", index, beforeUV[index], afterUV[index])
		}
	}
	inventory, err := rendering.InspectPhysicalGLB(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Nodes) != 18 || inventory.TriangleCount != 39462 || inventory.Nodes[14].Name != "iPad Mini_Screen_0" || inventory.Nodes[14].Mesh == nil || *inventory.Nodes[14].Mesh != 10 {
		t.Fatalf("neutral iPad semantic inventory changed: nodes=%d triangles=%d screen=%+v", len(inventory.Nodes), inventory.TriangleCount, inventory.Nodes[14])
	}
	manifest, err := starterAssets.ReadFile("assets/ipad/asset-manifest-uv-normalized.json")
	if err != nil || !json.Valid(manifest) || !strings.Contains(string(manifest), ipadDerivativeSHA256) || !strings.Contains(string(manifest), "CC BY 4.0") || !strings.Contains(string(manifest), "private aligned TEXCOORD_0") {
		t.Fatalf("iPad package manifest lacks exact candidate provenance: err=%v", err)
	}
	orientedInventory, err := rendering.InspectPhysicalGLB(oriented)
	if err != nil || len(orientedInventory.Nodes) != len(inventory.Nodes) || orientedInventory.TriangleCount != inventory.TriangleCount || orientedInventory.Nodes[14].Name != inventory.Nodes[14].Name || orientedInventory.Nodes[14].Mesh == nil || *orientedInventory.Nodes[14].Mesh != 10 || orientedInventory.Nodes[13].Name != inventory.Nodes[13].Name || orientedInventory.Nodes[13].Mesh == nil || *orientedInventory.Nodes[13].Mesh != 9 {
		t.Fatalf("oriented candidate must preserve display and port semantic nodes: screen=%+v port=%+v err=%v", orientedInventory.Nodes[14], orientedInventory.Nodes[13], err)
	}
	orientedManifest, err := starterAssets.ReadFile("assets/ipad/asset-manifest-uv-oriented-180-v2.json")
	if err != nil || !json.Valid(orientedManifest) || !strings.Contains(string(orientedManifest), ipadOrientedGLBSHA256) || !strings.Contains(string(orientedManifest), "(u,v) to (1-u,1-v)") {
		t.Fatalf("oriented iPad manifest is missing the exact correction/provenance: err=%v", err)
	}
	placement := ReviewedIPadPlacement{
		SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		DeviceRootRest:    physical.TRS{Translation: physical.Vec3{Y: .002631086703401778}, Rotation: physical.QuatXYZW{X: math.Sin((-89.430831 * math.Pi / 180) / 2), W: math.Cos((-89.430831 * math.Pi / 180) / 2)}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
	}
	lowest, err := ipadGLBPlacedMinimumY(b, placement)
	if err != nil || math.Abs(lowest-.75) > 1e-8 || placement.DeviceRootRest.Translation.Y > .01 {
		t.Fatalf("all actual source vertices must rest at the scene's 0.75 m support plane without a second world-height offset: minY=%.12f err=%v placement=%+v", lowest, err, placement.DeviceRootRest)
	}
}

func TestIPadLandscapeCandidateRotatesOnlyScreenUVsAndPreservesTableContact(t *testing.T) {
	source, err := starterAssets.ReadFile("assets/ipad/ipad-mini-screen-uv-normalized.glb")
	if err != nil {
		t.Fatal(err)
	}
	derived, err := rotateScreenUVQuarterTurn(source, "iPad Mini_Screen_0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(derived) != 1939668 || sha256Hex(derived) != ipadLandscapeGLBSHA256 {
		t.Fatalf("quarter-turn derivative changed: size=%d sha=%s", len(derived), sha256Hex(derived))
	}
	before, err := ipadScreenUVs(source, 36)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ipadScreenUVs(derived, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("screen UV count changed: %d to %d", len(before), len(after))
	}
	for i := range before {
		if math.Abs(after[i][0]-before[i][1]) > 1e-7 || math.Abs(after[i][1]-(1-before[i][0])) > 1e-7 {
			t.Fatalf("UV %d was not quarter-turned: before=%v after=%v", i, before[i], after[i])
		}
	}
	ctx := database.WithTestMode(context.Background())
	owners := newReviewedStarterInstallTestOwners(t, ctx)
	state, err := owners.InitializeReviewedIPadLandscapeStarter(ctx)
	if err != nil {
		t.Fatalf("initialize landscape candidate: %v", err)
	}
	if state.Device.DeviceID != ipadLandscapeDeviceID || state.Device.State != devices.DeviceRevisionCandidate || state.Device.Candidate.Definition.Qualification != "unqualified" || state.Scene.SceneID != ipadLandscapeSceneID || state.Preset.PresetID != ipadLandscapePresetID {
		t.Fatalf("landscape candidate identity or qualification differs: device=%+v scene=%s preset=%s", state.Device, state.Scene.SceneID, state.Preset.PresetID)
	}
	if state.Device.Candidate.Definition.Displays[0].CoordinateMetric.WidthMM <= state.Device.Candidate.Definition.Displays[0].CoordinateMetric.HeightMM || state.Preset.Setup.DeviceRootRest == nil || state.Preset.Setup.SceneSupportId != state.Placement.SceneSupportID || state.Preset.Setup.CameraId != ipadLandscapeCameraID {
		t.Fatalf("landscape display axes/root/support/camera were not versioned together: display=%+v setup=%+v", state.Device.Candidate.Definition.Displays[0], state.Preset.Setup)
	}
	minimum, err := ipadGLBPlacedMinimumY(derived, state.Placement)
	if err != nil || math.Abs(minimum-.75) > .0005 {
		t.Fatalf("landscape root rest must preserve full-body tabletop contact after quarter-turn: minimumY=%.9f err=%v", minimum, err)
	}
}

func TestIPadLandscapeV2UsesInverseUVTurnAndPreservesV1Candidate(t *testing.T) {
	source, err := starterAssets.ReadFile("assets/ipad/ipad-mini-screen-uv-normalized.glb")
	if err != nil {
		t.Fatal(err)
	}
	derived, err := rotateScreenUVQuarterTurnCounterClockwise(source, "iPad Mini_Screen_0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(derived) != 1939668 || sha256Hex(derived) != ipadLandscapeV2GLBSHA256 {
		t.Fatalf("inverse quarter-turn derivative changed: size=%d sha=%s", len(derived), sha256Hex(derived))
	}
	before, err := ipadScreenUVs(source, 36)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ipadScreenUVs(derived, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("screen UV count changed: %d to %d", len(before), len(after))
	}
	for i := range before {
		if math.Abs(after[i][0]-(1-before[i][1])) > 1e-7 || math.Abs(after[i][1]-before[i][0]) > 1e-7 {
			t.Fatalf("UV %d was not inverse-quarter-turned: before=%v after=%v", i, before[i], after[i])
		}
	}
	ctx := database.WithTestMode(context.Background())
	owners := newReviewedStarterInstallTestOwners(t, ctx)
	v1, err := owners.InitializeReviewedIPadLandscapeStarter(ctx)
	if err != nil {
		t.Fatalf("initialize historical landscape v1: %v", err)
	}
	v2, err := owners.InitializeReviewedIPadLandscapeV2Starter(ctx)
	if err != nil {
		t.Fatalf("initialize corrected landscape v2: %v", err)
	}
	if v1.Device.DeviceID != ipadLandscapeDeviceID || v1.Device.SHA256 == v2.Device.SHA256 || v2.Device.DeviceID != ipadLandscapeV2DeviceID || v2.Device.State != devices.DeviceRevisionCandidate || v2.Device.Candidate.Definition.Qualification != "unqualified" || v2.Scene.SceneID != ipadLandscapeV2SceneID || v2.Preset.PresetID != ipadLandscapeV2PresetID {
		t.Fatalf("versioned candidates must preserve v1 and keep corrected v2 unqualified: v1=%+v v2=%+v", v1.Device, v2.Device)
	}
	if v2.Device.Candidate.Definition.Displays[0].CoordinateMetric.WidthMM <= v2.Device.Candidate.Definition.Displays[0].CoordinateMetric.HeightMM || v2.Preset.Setup.DeviceRootRest == nil || v2.Preset.Setup.SceneSupportId != v2.Placement.SceneSupportID || v2.Preset.Setup.CameraId != ipadLandscapeV2CameraID {
		t.Fatalf("landscape v2 display axes/root/support/camera were not versioned together: display=%+v setup=%+v", v2.Device.Candidate.Definition.Displays[0], v2.Preset.Setup)
	}
	minimum, err := ipadGLBPlacedMinimumY(derived, v2.Placement)
	if err != nil || math.Abs(minimum-.75) > .0005 {
		t.Fatalf("corrected landscape root rest must preserve tabletop contact: minimumY=%.9f err=%v", minimum, err)
	}
}

func ipadScreenUVs(glb []byte, accessorIndex int) ([][2]float64, error) {
	if len(glb) < 20 || string(glb[:4]) != "glTF" || binary.LittleEndian.Uint32(glb[4:8]) != 2 || int(binary.LittleEndian.Uint32(glb[8:12])) != len(glb) {
		return nil, errors.New("invalid pinned iPad GLB header")
	}
	offset := 12
	var doc struct {
		BufferViews []struct {
			ByteOffset uint32 `json:"byteOffset"`
			ByteLength uint32 `json:"byteLength"`
		} `json:"bufferViews"`
		Accessors []struct {
			BufferView    uint32 `json:"bufferView"`
			ByteOffset    uint32 `json:"byteOffset"`
			ComponentType uint32 `json:"componentType"`
			Count         uint32 `json:"count"`
			Type          string `json:"type"`
		} `json:"accessors"`
	}
	var bin []byte
	for offset < len(glb) {
		if offset+8 > len(glb) {
			return nil, errors.New("truncated pinned iPad GLB chunk")
		}
		size := int(binary.LittleEndian.Uint32(glb[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(glb[offset+4 : offset+8])
		offset += 8
		if size < 0 || size > len(glb)-offset {
			return nil, errors.New("pinned iPad GLB chunk exceeds file")
		}
		payload := glb[offset : offset+size]
		switch kind {
		case 0x4e4f534a:
			if err := json.Unmarshal(bytes.TrimRight(payload, " \t\r\n\x00"), &doc); err != nil {
				return nil, err
			}
		case 0x004e4942:
			bin = payload
		}
		offset += size
	}
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, errors.New("pinned screen UV accessor is absent")
	}
	a := doc.Accessors[accessorIndex]
	if a.ComponentType != 5126 || a.Type != "VEC2" || a.BufferView >= uint32(len(doc.BufferViews)) || a.Count != 74 {
		return nil, errors.New("pinned screen UV accessor layout differs")
	}
	v := doc.BufferViews[a.BufferView]
	start := uint64(v.ByteOffset) + uint64(a.ByteOffset)
	end := start + uint64(a.Count)*8
	if end > uint64(len(bin)) || uint64(v.ByteOffset)+uint64(v.ByteLength) > uint64(len(bin)) || end > uint64(v.ByteOffset)+uint64(v.ByteLength) {
		return nil, errors.New("pinned screen UV view exceeds BIN")
	}
	rows := make([][2]float64, a.Count)
	for i := range rows {
		rows[i] = [2]float64{float64(math.Float32frombits(binary.LittleEndian.Uint32(bin[start+uint64(i)*8:]))), float64(math.Float32frombits(binary.LittleEndian.Uint32(bin[start+uint64(i)*8+4:])))}
	}
	return rows, nil
}

func TestIPadReviewRasterMatchesMeasuredDisplayAspect(t *testing.T) {
	config, err := png.DecodeConfig(bytes.NewReader(ipadReviewScreenPNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	got := float64(config.Width) / float64(config.Height)
	want := 117.281 / 177.229
	if math.Abs(got-want) > .0003 {
		t.Fatalf("synthetic review raster aspect %.8f differs from measured screen %.8f", got, want)
	}
}

func assertIPadCameraContainsVertices(t *testing.T, vertices []physical.Vec3, camera scenes.SceneCameraFrame, aspect, margin float64) {
	t.Helper()
	if camera.Projection != scenes.ProjectionPerspective || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil || camera.SensorFit != scenes.SensorFitHorizontal {
		t.Fatalf("iPad camera fit test needs the fixed horizontal perspective camera: %+v", camera)
	}
	matrix, err := camera.Frame.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	origin, err := physical.TransformPoint(matrix, physical.Vec3{})
	if err != nil {
		t.Fatal(err)
	}
	axis := func(local physical.Vec3) physical.Vec3 {
		point, err := physical.TransformPoint(matrix, local)
		if err != nil {
			t.Fatal(err)
		}
		return normalize(subVec(point, origin))
	}
	right := axis(physical.Vec3{X: 1})
	up := axis(physical.Vec3{Y: 1})
	forward := axis(physical.Vec3{Z: -1})
	sensorWidth := *camera.SensorWidthMM
	sensorHeight := sensorWidth / aspect
	allowed := .5 - margin
	maxX, maxY, minDepth := 0.0, 0.0, math.Inf(1)
	for _, point := range vertices {
		delta := subVec(point, origin)
		depth := dotVec(delta, forward)
		minDepth = math.Min(minDepth, depth)
		if depth <= camera.NearClipM {
			t.Fatalf("camera fit puts an actual mesh vertex behind/inside its near plane: depth=%g near=%g point=%+v", depth, camera.NearClipM, point)
		}
		maxX = math.Max(maxX, math.Abs(dotVec(delta, right)*(*camera.FocalLengthMM/sensorWidth)/depth))
		maxY = math.Max(maxY, math.Abs(dotVec(delta, up)*(*camera.FocalLengthMM/sensorHeight)/depth))
	}
	if maxX > allowed+1e-7 || maxY > allowed+1e-7 {
		t.Fatalf("actual iPad geometry escapes the fitted 16:9 frame: maxNormalized=(%.6f,%.6f) allowed=%.6f minDepth=%.4fm vertices=%d", maxX, maxY, allowed, minDepth, len(vertices))
	}
	t.Logf("iPad camera fit: actual_vertices=%d max_normalized=(%.6f,%.6f), allowed=%.6f, minimum_depth=%.4fm", len(vertices), maxX, maxY, allowed, minDepth)
}

type ipadGLBNode struct {
	Name        string    `json:"name"`
	Mesh        *uint32   `json:"mesh"`
	Children    []uint32  `json:"children"`
	Translation []float64 `json:"translation"`
	Rotation    []float64 `json:"rotation"`
	Scale       []float64 `json:"scale"`
}

type ipadGLBAccessor struct {
	BufferView    uint32 `json:"bufferView"`
	ComponentType uint32 `json:"componentType"`
	Count         uint32 `json:"count"`
	Type          string `json:"type"`
	ByteOffset    uint32 `json:"byteOffset"`
}

type ipadGLBBufferView struct {
	Buffer     uint32 `json:"buffer"`
	ByteOffset uint32 `json:"byteOffset"`
	ByteLength uint32 `json:"byteLength"`
	ByteStride uint32 `json:"byteStride"`
}

type ipadGLBPrimitive struct {
	Attributes map[string]uint32 `json:"attributes"`
}

type ipadGLBMesh struct {
	Primitives []ipadGLBPrimitive `json:"primitives"`
}

type ipadGLBScene struct {
	Nodes []uint32 `json:"nodes"`
}

type ipadGLBDocument struct {
	Scene       int                 `json:"scene"`
	Scenes      []ipadGLBScene      `json:"scenes"`
	Nodes       []ipadGLBNode       `json:"nodes"`
	Meshes      []ipadGLBMesh       `json:"meshes"`
	Accessors   []ipadGLBAccessor   `json:"accessors"`
	BufferViews []ipadGLBBufferView `json:"bufferViews"`
}

func ipadGLBPlacedMinimumY(data []byte, placement ReviewedIPadPlacement) (float64, error) {
	points, err := ipadGLBPlacedVertices(data, placement)
	if err != nil {
		return 0, err
	}
	minimum := math.Inf(1)
	for _, point := range points {
		minimum = math.Min(minimum, point.Y)
	}
	return minimum, nil
}

func ipadGLBPlacedVertices(data []byte, placement ReviewedIPadPlacement) ([]physical.Vec3, error) {
	byNode, err := ipadGLBPlacedNodeVertices(data, placement)
	if err != nil {
		return nil, err
	}
	points := make([]physical.Vec3, 0)
	for _, nodePoints := range byNode {
		points = append(points, nodePoints...)
	}
	return points, nil
}

func ipadGLBPlacedNodeVertices(data []byte, placement ReviewedIPadPlacement) (map[string][]physical.Vec3, error) {
	if len(data) < 20 || string(data[:4]) != "glTF" || binary.LittleEndian.Uint32(data[4:8]) != 2 || int(binary.LittleEndian.Uint32(data[8:12])) != len(data) {
		return nil, errors.New("invalid fixed iPad GLB header")
	}
	var documentBytes, binaryBytes []byte
	for offset := 12; offset+8 <= len(data); {
		length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || length > len(data)-offset {
			return nil, errors.New("iPad GLB chunk exceeds file bounds")
		}
		chunk := data[offset : offset+length]
		switch kind {
		case 0x4e4f534a:
			documentBytes = chunk
		case 0x004e4942:
			binaryBytes = chunk
		}
		offset += length
	}
	var document ipadGLBDocument
	if err := json.Unmarshal(documentBytes, &document); err != nil || len(document.Nodes) == 0 || len(document.Scenes) == 0 || document.Scene < 0 || document.Scene >= len(document.Scenes) || len(binaryBytes) == 0 {
		return nil, errors.New("iPad GLB has no usable scene, JSON, or binary chunk")
	}
	world := make([]physical.Matrix4, len(document.Nodes))
	seen := make([]bool, len(document.Nodes))
	var visit func(uint32, physical.Matrix4) error
	visit = func(index uint32, parent physical.Matrix4) error {
		if int(index) >= len(document.Nodes) {
			return errors.New("iPad GLB scene references an absent node")
		}
		node := document.Nodes[index]
		local := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
		if len(node.Translation) != 0 {
			if len(node.Translation) != 3 {
				return errors.New("invalid iPad node translation")
			}
			local.Translation = physical.Vec3{X: node.Translation[0], Y: node.Translation[1], Z: node.Translation[2]}
		}
		if len(node.Rotation) != 0 {
			if len(node.Rotation) != 4 {
				return errors.New("invalid iPad node rotation")
			}
			local.Rotation = physical.QuatXYZW{X: node.Rotation[0], Y: node.Rotation[1], Z: node.Rotation[2], W: node.Rotation[3]}
		}
		if len(node.Scale) != 0 {
			if len(node.Scale) != 3 {
				return errors.New("invalid iPad node scale")
			}
			local.Scale = physical.Vec3{X: node.Scale[0], Y: node.Scale[1], Z: node.Scale[2]}
		}
		localMatrix, err := local.Matrix()
		if err != nil {
			return err
		}
		world[index], err = physical.Compose(parent, localMatrix)
		if err != nil {
			return err
		}
		seen[index] = true
		for _, child := range node.Children {
			if err := visit(child, world[index]); err != nil {
				return err
			}
		}
		return nil
	}
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	identityMatrix, _ := identity.Matrix()
	for _, root := range document.Scenes[document.Scene].Nodes {
		if err := visit(root, identityMatrix); err != nil {
			return nil, err
		}
	}
	sourceMatrix, err := (physical.TRS{Rotation: physical.QuatXYZW{Y: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		return nil, err
	}
	rootRest, err := placement.DeviceRootRest.Matrix()
	if err != nil {
		return nil, err
	}
	pivotMatrix, err := (physical.TRS{Translation: placement.SupportPivotM, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		return nil, err
	}
	sceneSupport, err := placement.SceneSupportWorld.Matrix()
	if err != nil {
		return nil, err
	}
	placementMatrix, err := physical.Compose(sceneSupport, pivotMatrix)
	if err == nil {
		placementMatrix, err = physical.Compose(placementMatrix, rootRest)
	}
	if err == nil {
		placementMatrix, err = physical.Compose(placementMatrix, sourceMatrix)
	}
	if err != nil {
		return nil, err
	}
	points := make(map[string][]physical.Vec3)
	for nodeIndex, node := range document.Nodes {
		if !seen[nodeIndex] || node.Mesh == nil || int(*node.Mesh) >= len(document.Meshes) {
			continue
		}
		for _, primitive := range document.Meshes[*node.Mesh].Primitives {
			accessorIndex, ok := primitive.Attributes["POSITION"]
			if !ok || int(accessorIndex) >= len(document.Accessors) {
				return nil, errors.New("iPad mesh lacks a valid POSITION accessor")
			}
			accessor := document.Accessors[accessorIndex]
			if accessor.ComponentType != 5126 || accessor.Type != "VEC3" || int(accessor.BufferView) >= len(document.BufferViews) {
				return nil, errors.New("iPad POSITION accessor is outside the fixed float VEC3 profile")
			}
			view := document.BufferViews[accessor.BufferView]
			if view.Buffer != 0 {
				return nil, errors.New("iPad POSITION accessor references a nonzero buffer")
			}
			stride := int(view.ByteStride)
			if stride == 0 {
				stride = 12
			}
			start := int(view.ByteOffset + accessor.ByteOffset)
			if stride < 12 || start < 0 || int(accessor.Count) > 0 && start+(int(accessor.Count)-1)*stride+12 > len(binaryBytes) || int(view.ByteOffset+view.ByteLength) > len(binaryBytes) {
				return nil, errors.New("iPad POSITION accessor exceeds its binary view")
			}
			combined, err := physical.Compose(placementMatrix, world[nodeIndex])
			if err != nil {
				return nil, err
			}
			for vertex := uint32(0); vertex < accessor.Count; vertex++ {
				offset := start + int(vertex)*stride
				point := physical.Vec3{X: float64(math.Float32frombits(binary.LittleEndian.Uint32(binaryBytes[offset : offset+4]))), Y: float64(math.Float32frombits(binary.LittleEndian.Uint32(binaryBytes[offset+4 : offset+8]))), Z: float64(math.Float32frombits(binary.LittleEndian.Uint32(binaryBytes[offset+8 : offset+12])))}
				placed, err := physical.TransformPoint(combined, point)
				if err != nil {
					return nil, err
				}
				points[node.Name] = append(points[node.Name], placed)
			}
		}
	}
	if len(points) == 0 {
		return nil, errors.New("iPad GLB yielded no transformed mesh vertices")
	}
	return points, nil
}

// This opt-in test initializes the candidate on isolated routed stores and
// exercises the real render-job/native-operation owner for its supported
// tabletop hero pose. A no-table rear scene is not packaged yet.
func TestOptInRenderIPadStarterSupportedHero(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E4_IPAD_NATIVE") != "1" {
		t.Skip("set DEVICE_STUDIO_E4_IPAD_NATIVE=1 to run the owned iPad still-render journey")
	}
	if os.Getenv("DEVICE_STUDIO_E4_IPAD_API_BIN") == "" {
		t.Fatal("DEVICE_STUDIO_E4_IPAD_API_BIN must point to the reviewed Device Studio API executable")
	}
	apiBin := filepath.Clean(os.Getenv("DEVICE_STUDIO_E4_IPAD_API_BIN"))
	if !filepath.IsAbs(apiBin) || apiBin != os.Getenv("DEVICE_STUDIO_E4_IPAD_API_BIN") {
		t.Fatalf("API binary path must be absolute and clean: %q", apiBin)
	}
	info, err := os.Lstat(apiBin)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 {
		t.Fatalf("API binary is not a regular file: %v", err)
	}
	evidence := filepath.Clean(os.Getenv("DEVICE_STUDIO_E4_IPAD_EVIDENCE_DIR"))
	if !filepath.IsAbs(evidence) || !strings.Contains(filepath.ToSlash(evidence), "/goal/evidence/E4/free-models/sceneview-research/ipad-source-preserving/native-runs/") || filepath.Base(evidence) == "native-runs" {
		t.Fatalf("evidence directory is outside the admitted iPad run tree: %q", evidence)
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatalf("create exclusive evidence directory: %v", err)
	}
	backendName := os.Getenv("DEVICE_STUDIO_E4_IPAD_BACKEND")
	backend, err := ipadNativeBackend(backendName)
	if err != nil {
		t.Fatal(err)
	}

	const nativeTestDeadline = 11 * time.Minute
	const routeLeaseTTL = 25 * time.Minute // 11m execution + 11m owner cleanup + 3m margin.
	ctx, cancel := context.WithTimeout(context.Background(), nativeTestDeadline)
	defer cancel()
	primaryData := filepath.Join(evidence, "primary-data")
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(evidence, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	providers := []database.SchemaProvider{database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema), database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema), database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema)}
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(initCtx, pool, providers...)
	})
	roots := filerouting.New(storage.Paths{DataDir: primaryData})
	const leaseID = "ipad-native-review"
	if err := db.InstallTestPool(ctx, filepath.Join(evidence, "leased.db"), leaseID, routeLeaseTTL); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(evidence, "leased-data")}, leaseID, routeLeaseTTL); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stats := db.LeaseStats()
		rootStats := roots.LeaseStats()
		clearRootsErr := roots.ClearTestRoots(leaseID)
		clearPoolErr := db.ClearTestPool(leaseID)
		receipt := marshalIPadJSON(map[string]any{
			"lease_id": leaseID, "database_test_requests": stats.TestPoolRequests,
			"database_primary_requests_during_test_mode": stats.PrimaryDuringTestModeRequests,
			"file_test_writes":                           rootStats.TestRootWrites,
			"file_primary_writes_during_test_mode":       rootStats.PrimaryWritesDuringTestMode,
			"database_clear_error":                       errorText(clearPoolErr), "file_roots_clear_error": errorText(clearRootsErr),
			"leased_database_path":  filepath.Join(evidence, "leased.db"),
			"leased_file_root_path": filepath.Join(evidence, "leased-data"),
		})
		if err := writeIPadExclusive(filepath.Join(evidence, "routing-cleanup-receipt.json"), receipt); err != nil {
			t.Errorf("retain routed cleanup receipt: %v", err)
		}
		if clearRootsErr != nil || clearPoolErr != nil {
			t.Errorf("clear isolated test routing after job cleanup: file roots=%v database=%v", clearRootsErr, clearPoolErr)
		}
	})
	testCtx := database.WithTestMode(ctx)
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, mediaService)
	if err != nil {
		t.Fatal(err)
	}
	var state IPadStarterState
	var sceneBundle library.BundleRevision
	if os.Getenv("DEVICE_STUDIO_E4_IPAD_ENVIRONMENT_V3") == "1" {
		v3, err := owners.InitializeReviewedIPadEnvironmentV3(testCtx)
		if err != nil {
			t.Fatalf("publish and save iPad environment v3 candidate: %v", err)
		}
		state, sceneBundle = v3.IPadStarterState, v3.SceneBundle
		if state.Device.DeviceID != ipadOrientedCandidateID || state.Device.Candidate.Geometry.SHA256 != ipadOrientedGLBSHA256 || state.Scene.SceneID != ipadEnvironmentV3SceneID || state.Placement.SelectedCameraID != ipadEnvironmentV3CameraID || sceneBundle.Manifest.BundleID != ipadEnvironmentV3BundleID {
			t.Fatalf("iPad environment v3 candidate identities differ: device=%s scene=%s camera=%s bundle=%s", state.Device.DeviceID, state.Scene.SceneID, state.Placement.SelectedCameraID, sceneBundle.Manifest.BundleID)
		}
	} else {
		v1, err := owners.InitializeReviewedIPadStarter(testCtx)
		if err != nil {
			t.Fatalf("publish and save iPad candidate: %v", err)
		}
		oriented, err := owners.InitializeReviewedIPadOrientedStarter(testCtx)
		if err != nil {
			t.Fatalf("publish and save oriented iPad candidate: %v", err)
		}
		state = oriented
		if state.Device.DeviceID != ipadOrientedCandidateID || state.Device.Candidate.Geometry.SHA256 != ipadOrientedGLBSHA256 || state.Scene.SceneID != ipadOrientedSceneID || state.Scene.Candidate.CameraFrames[0].ID != ipadOrientedCameraID {
			t.Fatalf("oriented candidate/scene/camera identity bindings differ: device=%s scene=%s", state.Device.DeviceID, state.Scene.SceneID)
		}
		if unchanged, err := owners.Devices.GetHead(testCtx, ipadCandidateID); err != nil || unchanged.SHA256 != v1.Device.SHA256 || unchanged.Candidate.Geometry.SHA256 != ipadDerivativeSHA256 {
			t.Fatalf("v1 candidate changed while adding oriented revision: err=%v head=%+v", err, unchanged)
		}
	}
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "DatSketch iPad candidate review"})
	if err != nil {
		t.Fatal(err)
	}
	screenBytes := ipadNativeReviewScreenPNG(t)
	screen, err := mediaService.Import(testCtx, project.ID, "ipad-app-screen-proof.png", int64(len(screenBytes)), bytes.NewReader(screenBytes))
	if err != nil {
		t.Fatalf("import neutral project-owned screen: %v", err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	workerInfo, err := os.Stat(apiBin)
	if err != nil || workerInfo.Size() <= 0 || workerInfo.Size() > 256<<20 {
		t.Fatalf("reviewed worker executable is outside the 256 MiB bound: %v", err)
	}
	worker, err := os.ReadFile(apiBin)
	if err != nil || len(worker) == 0 {
		t.Fatalf("read reviewed worker executable: %v", err)
	}
	profileBytes, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		t.Fatalf("resolve managed Blender profile: %v", err)
	}
	if profile.GetBlender().GetPath() == "" {
		t.Fatal("managed Blender profile does not resolve an executable")
	}
	profileDigest := sha256.Sum256(profileBytes)
	profileSHA := hex.EncodeToString(profileDigest[:])
	renderBlobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{
		Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: renderBlobs, Roots: roots,
		Exe: func() (string, error) { return apiBin, nil }, PhysicalArtifacts: owners.Library,
		PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil },
		PhysicalAdapter: rendering.PhysicalAdapterBytes,
	})
	if err != nil {
		t.Fatal(err)
	}

	const poseName = "hero-supported-candidate-tabletop-pose"
	snapshot := ipadStillSnapshot(state, screen.ID, state.Placement, backend)
	snapshot.PhysicalBinding.ScreenWidthPixels = 1536
	snapshot.PhysicalBinding.ScreenHeightPixels = 2322
	shot, revision, err := shotService.Create(testCtx, project.ID, poseName, snapshot)
	if err != nil {
		t.Fatalf("create %s shot: %v", poseName, err)
	}
	if !revision.RenderEligible {
		t.Fatalf("%s candidate shot is not render eligible: %+v", poseName, revision.Issues)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: 1, IdempotencyKey: "ipad-candidate-" + poseName, OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !created {
		t.Fatalf("submit %s native still: created=%v err=%v", poseName, created, err)
	}
	t.Cleanup(func() {
		cleanupIPadJob(t, coordinator, db, roots, evidence, project.ID, job.ID, profileSHA, "ipad-native-review")
	})
	workerSum := sha256.Sum256(worker)
	if err := writeIPadExclusive(filepath.Join(evidence, "native-job-submission.json"), marshalIPadJSON(map[string]any{
		"project_id": project.ID, "shot_id": shot.ID, "shot_revision": revision.Revision,
		"job": job, "created": created, "device_id": state.Device.DeviceID, "device_revision": state.Device.Revision,
		"device_sha256": state.Device.SHA256, "scene_id": state.Scene.SceneID, "scene_revision": state.Scene.Revision,
		"motion_id": state.Motion.MotionID, "bundle_manifest_sha256": state.Bundle.ManifestSHA256,
		"bundle_review_sha256": state.Bundle.ReviewSHA256, "profile_sha256": profileSHA,
		"api_worker_sha256": hex.EncodeToString(workerSum[:]), "backend_requested": backendName,
		"scene_bundle_id": sceneBundle.Manifest.BundleID, "scene_bundle_manifest_sha256": sceneBundle.ManifestSHA256, "scene_bundle_review_sha256": sceneBundle.ReviewSHA256,
	})); err != nil {
		t.Fatalf("retain native iPad submission before execution: %v", err)
	}
	for job.State != jobs.Succeeded && job.State != jobs.Failed && job.State != jobs.Cancelled {
		job, err = coordinator.Advance(testCtx, job.ID)
		if err != nil {
			t.Fatalf("reconcile %s native still: %v", poseName, err)
		}
		if job.State != jobs.Succeeded && job.State != jobs.Failed && job.State != jobs.Cancelled {
			select {
			case <-ctx.Done():
				t.Fatalf("%s native still exceeded bounded test deadline: %v", poseName, ctx.Err())
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	if job.State != jobs.Succeeded {
		t.Fatalf("%s native still finished %s: %+v", poseName, job.State, job.Failure)
	}
	openedJob, output, err := coordinator.OpenOutput(testCtx, project.ID, job.ID)
	if err != nil {
		t.Fatalf("open verified %s PNG: %v", poseName, err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || config.Width != 2560 || config.Height != 1440 {
		t.Fatalf("%s output dimensions=%+v err=%v", poseName, config, err)
	}
	if _, err := png.Decode(bytes.NewReader(output)); err != nil {
		t.Fatalf("fully decode %s PNG: %v", poseName, err)
	}
	if err := writeIPadNativeEvidence(evidence, poseName, openedJob, state, profileSHA, output); err != nil {
		t.Fatalf("retain %s evidence: %v", poseName, err)
	}
	if err := writeIPadRunReceipt(evidence, project.ID, state, sceneBundle, apiBin, worker, profileSHA, backendName); err != nil {
		t.Fatal(err)
	}
}

func ipadStillSnapshot(state IPadStarterState, sourceMediaID string, placement ReviewedIPadPlacement, backend shotsv1.PhysicalRenderBackend) *shotsv1.ShotSnapshot {
	return &shotsv1.ShotSnapshot{
		PhysicalBinding: &shotsv1.PhysicalShotBinding{
			Device:    &shotsv1.ComponentRevisionRef{Id: state.Device.DeviceID, Revision: state.Device.Revision, Sha256: state.Device.SHA256},
			Scene:     &shotsv1.ComponentRevisionRef{Id: state.Scene.SceneID, Revision: state.Scene.Revision, Sha256: state.Scene.SHA256},
			Motion:    &shotsv1.ComponentRevisionRef{Id: state.Motion.MotionID, Revision: state.Motion.Revision, Sha256: state.Motion.SHA256},
			DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
			SceneSupportWorld: ipadv1TRS(placement.SceneSupportWorld), SupportPivotM: &physicalv1.Vec3{X: placement.SupportPivotM.X, Y: placement.SupportPivotM.Y, Z: placement.SupportPivotM.Z}, DeviceRootRest: ipadv1TRS(placement.DeviceRootRest),
			ScreenWidthPixels: 512, ScreenHeightPixels: 774, RendererProfileId: "device-studio-physical-cycles-v1", RendererProfileVersion: 1,
			OutputProfile: shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1, MotionSampleTick: &shotsv1.RationalTime{Numerator: 0, Denominator: 1}, Backend: backend,
		},
		Presentation: &shotsv1.ScreenPresentation{SourceMediaId: sourceMediaID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN},
		MotionMode:   shotsv1.MotionMode_MOTION_MODE_FIXED,
	}
}

func ipadv1TRS(value physical.TRS) *physicalv1.TRS {
	return &physicalv1.TRS{TranslationM: &physicalv1.Vec3{X: value.Translation.X, Y: value.Translation.Y, Z: value.Translation.Z}, RotationXyzw: &physicalv1.QuatXYZW{X: value.Rotation.X, Y: value.Rotation.Y, Z: value.Rotation.Z, W: value.Rotation.W}, Scale: &physicalv1.Vec3{X: value.Scale.X, Y: value.Scale.Y, Z: value.Scale.Z}}
}

func ipadReviewScreenPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 512, 774))
	for y := 0; y < 774; y++ {
		for x := 0; x < 512; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(210 + (x/128)%30), G: uint8(225 + (y/256)%20), B: 235, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// ipadNativeReviewScreenPNG reads one approved, local-only Device Studio UI
// capture. It remains test evidence and is never included in starterAssets.
func ipadNativeReviewScreenPNG(t *testing.T) []byte {
	t.Helper()
	const (
		originalPath = "../../../docs/internal/goal/evidence/E4/free-models/sceneview-research/ipad-app-source/current-projects-ui/f5da4931-3fe0-45bc-85c8-811d08f6b5b3/screenshots/step-04-5ccbcd6c-ec26-4282-aa24-7fcad0b39b28.png"
		originalSHA  = "cf1a75656ccdb4e4d1d0cebc49f074a4c0c8a18b75531f0b8d1932e9f9221c3c"
		proofPath    = "../../../docs/internal/goal/evidence/E4/free-models/sceneview-research/ipad-screen-uv-orientation/proof/current-projects-ui-contained-1536x2322.png"
		proofSHA     = "c2cbaa2baee01343ffb8712d836a3b1a34bff0b6be3edc0fe0048a7f450fd38b"
	)
	original, err := os.ReadFile(filepath.Clean(originalPath))
	if err != nil || sha256Hex(original) != originalSHA {
		t.Fatalf("read or verify pinned private iPad app-screen source: err=%v sha256=%s", err, sha256Hex(original))
	}
	data, err := os.ReadFile(filepath.Clean(proofPath))
	if err != nil {
		t.Fatalf("read pinned private iPad app-screen proof raster: %v", err)
	}
	if got := sha256Hex(data); got != proofSHA {
		t.Fatalf("private iPad proof raster changed: sha256=%s want=%s", got, proofSHA)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != 1536 || config.Height != 2322 {
		t.Fatalf("private iPad proof raster dimensions=%+v err=%v; want aspect-padded 1536x2322", config, err)
	}
	sourceImage, err := png.Decode(bytes.NewReader(original))
	if err != nil {
		t.Fatalf("decode pinned private app source: %v", err)
	}
	proofImage, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode aspect-padded private app proof: %v", err)
	}
	for y := 0; y < 2320; y++ {
		for x := 0; x < 1536; x++ {
			proofPixel := color.NRGBAModel.Convert(proofImage.At(x, y+1)).(color.NRGBA)
			sourcePixel := color.NRGBAModel.Convert(sourceImage.At(x, y)).(color.NRGBA)
			if proofPixel != sourcePixel {
				t.Fatalf("proof image changed source pixel at (%d,%d)", x, y)
			}
		}
	}
	return data
}

func writeIPadNativeEvidence(root, pose string, job jobs.Job, state IPadStarterState, profileSHA string, output []byte) error {
	name := pose + "-2560x1440.png"
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return errors.New("invalid fixed iPad evidence name")
	}
	f, err := os.OpenFile(filepath.Join(root, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(output); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	sum := sha256.Sum256(output)
	row := map[string]any{"job": job, "shot_pose": pose, "device_id": state.Device.DeviceID, "device_revision": state.Device.Revision, "device_sha256": state.Device.SHA256, "scene_id": state.Scene.SceneID, "scene_revision": state.Scene.Revision, "motion_id": state.Motion.MotionID, "profile_sha256": profileSHA, "png_path": name, "png_size_bytes": len(output), "png_sha256": hex.EncodeToString(sum[:]), "preview_status": "candidate_still_only_unqualified"}
	data, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return err
	}
	return writeIPadExclusive(filepath.Join(root, pose+"-receipt.json"), append(data, '\n'))
}

func writeIPadRunReceipt(root, projectID string, state IPadStarterState, sceneBundle library.BundleRevision, apiBin string, worker []byte, profileSHA, backend string) error {
	sum := sha256.Sum256(worker)
	row := map[string]any{"project_id": projectID, "device_id": state.Device.DeviceID, "device_revision": state.Device.Revision, "device_sha256": state.Device.SHA256, "scene_id": state.Scene.SceneID, "scene_revision": state.Scene.Revision, "motion_id": state.Motion.MotionID, "bundle_manifest_sha256": state.Bundle.ManifestSHA256, "bundle_review_sha256": state.Bundle.ReviewSHA256, "scene_bundle_id": sceneBundle.Manifest.BundleID, "scene_bundle_manifest_sha256": sceneBundle.ManifestSHA256, "scene_bundle_review_sha256": sceneBundle.ReviewSHA256, "creator": "DatSketch", "license": "CC BY 4.0", "source_sha256": ipadSourceSHA256, "device_derivative_sha256": state.Device.Candidate.Geometry.SHA256, "screen_uv_correction": "(u,v) -> (1-u,1-v); accessor 36 only; screen pixels and device pose unchanged", "screen_preview_proof": map[string]any{"source_path": "docs/internal/goal/evidence/E4/free-models/sceneview-research/ipad-app-source/current-projects-ui/f5da4931-3fe0-45bc-85c8-811d08f6b5b3/screenshots/step-04-5ccbcd6c-ec26-4282-aa24-7fcad0b39b28.png", "source_sha256": "cf1a75656ccdb4e4d1d0cebc49f074a4c0c8a18b75531f0b8d1932e9f9221c3c", "source_dimensions": []int{1536, 2320}, "proof_path": "docs/internal/goal/evidence/E4/free-models/sceneview-research/ipad-screen-uv-orientation/proof/current-projects-ui-contained-1536x2322.png", "proof_sha256": "c2cbaa2baee01343ffb8712d836a3b1a34bff0b6be3edc0fe0048a7f450fd38b", "proof_dimensions": []int{1536, 2322}, "padding": "copy first and last source rows to add one pixel vertically; original rows preserved; no scaling or stretching; proof raster is not packaged"}, "api_binary_path": apiBin, "api_worker_sha256": hex.EncodeToString(sum[:]), "blender_profile_sha256": profileSHA, "backend_requested": backend, "placement_profile": state.Placement.ProfileID, "placement_pose": "candidate supported tabletop hero using actual-vertex-derived pose; visual result does not establish physical stability", "rendered_views": []string{"hero-supported-candidate-tabletop-pose"}, "rear_view": "deferred until a separate no-table scene is available", "status": "candidate_native_hero_only_unqualified"}
	data, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return err
	}
	return writeIPadExclusive(filepath.Join(root, "run-receipt.json"), append(data, '\n'))
}

func writeIPadExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func marshalIPadJSON(value any) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return []byte("{\"marshal_error\":\"" + err.Error() + "\"}\n")
	}
	return append(data, '\n')
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
