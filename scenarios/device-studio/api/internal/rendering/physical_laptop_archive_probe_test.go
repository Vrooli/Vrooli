package rendering

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/physical_laptop_archive_probe.py
var physicalLaptopArchiveProbe []byte

const (
	laptopArchiveJobID      = "5743fd38-f4fd-4a87-9392-bf74653135f6"
	laptopArchiveAttemptID  = "9204e058-ff57-41e4-9a5a-20e170627ad6"
	laptopArchiveWorkerSHA  = "7608a3ff8aa19605b3ac8749ddf76f31c0f0d233112a6a4d7cae026cde55d017"
	laptopArchiveDeviceSHA  = "ee9631cedfab92b782ceb76f400980077d38b6c98394722443ce4e6322507407"
	laptopArchiveSceneSHA   = "ec7674e4213cf145d1405422f71af3f852be6f039da834bd0fbc549504527100"
	laptopArchiveScreenSHA  = "cdd4f9ee8263e55ecde1aae5caebdcb3df8bcc431d485a3aadd09720cd18420e"
	laptopArchiveAdapterSHA = "d6e892d3904bf9cfd17253340e0a97cdd936e393c2835ec5227914ab72a001a1"
	laptopPilot3JobID       = "fb358a14-9a06-43cf-aea2-60091bd01974"
	laptopPilot3AttemptID   = "bba52fb5-cb95-4b72-b0e5-ab509cc1565a"
	laptopPilot3WorkerSHA   = "984c4e4a722ec563a94fe7bff780c602c33a6a77af5a28be3611d45d0b2077b6"
	laptopPilot3AdapterSHA  = "f47d4fff0187b27bcb20f5db31a29d22c1f28accd8d8d5b9f21c509547edb742"
)

type laptopDiagnosticFile struct {
	Name string `json:"name"`
	SHA  string `json:"sha256"`
	Size int64  `json:"size_bytes"`
}

type laptopDiagnosticManifest struct {
	Version       int                    `json:"version"`
	Mode          string                 `json:"mode"`
	JobID         string                 `json:"job_id"`
	AttemptID     string                 `json:"attempt_id"`
	AdapterName   string                 `json:"adapter_name"`
	AdapterMode   string                 `json:"adapter_mode"`
	AdapterSHA    string                 `json:"adapter_sha256"`
	ProbeSHA      string                 `json:"probe_sha256"`
	WorkerInput   laptopDiagnosticFile   `json:"worker_input"`
	DeviceGLB     laptopDiagnosticFile   `json:"device_glb"`
	SceneGLB      laptopDiagnosticFile   `json:"scene_glb"`
	ScreenPNG     laptopDiagnosticFile   `json:"screen_png"`
	SceneTextures []laptopDiagnosticFile `json:"scene_textures,omitempty"`
}

type laptopDiagnosticReport struct {
	Version    int    `json:"version"`
	Status     string `json:"status"`
	JobID      string `json:"job_id"`
	AttemptID  string `json:"attempt_id"`
	AdapterSHA string `json:"adapter_sha256"`
	InputSHA   string `json:"worker_input_sha256"`
	Frames     []struct {
		Ordinal int               `json:"ordinal"`
		Parts   []json.RawMessage `json:"parts"`
	} `json:"frames"`
	Hierarchy         []json.RawMessage `json:"frame_zero_hierarchy"`
	RestMatrixError   float64           `json:"zero_pose_matrix_error"`
	ABAMatrixError    float64           `json:"a_b_a_matrix_error"`
	TravelMatrixDelta float64           `json:"travel_matrix_delta_from_open"`
}

// This opt-in probe imports the exact archived framed pilot and records
// hierarchy plus bounded, sampled triangle-level clearance diagnostics.
// Sampled geometry is diagnostic evidence, not continuous mechanical proof.
func TestOptInPhysicalLaptopArchivedRenderDiagnostic(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_LAPTOP_DIAGNOSTIC") != "1" {
		t.Skip("opt-in archived laptop hierarchy diagnostic")
	}
	adapterMode := os.Getenv("DEVICE_STUDIO_E3_LAPTOP_DIAGNOSTIC_ADAPTER")
	if adapterMode == "" {
		adapterMode = "original"
	}
	base := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_LAPTOP_DIAGNOSTIC_EVIDENCE_DIR"))
	evidencePath := "/goal/evidence/E3/scene-studies/laptop/physical-candidate/"
	archivePath := "E3/scene-studies/laptop/physical-candidate/lease13/verified-archive/leased-files/data/device-studio/renderjobs"
	jobID, attemptID := laptopArchiveJobID, laptopArchiveAttemptID
	workerSHA := laptopArchiveWorkerSHA
	workerSize := int64(43376)
	if adapterMode == "archive" {
		evidencePath = "/goal/evidence/E3/scene-studies/laptop/clearance-diagnostic/"
		archivePath = "E3/scene-studies/laptop/framed-scene/lease14/verified-archive/leased-files/data/device-studio/renderjobs"
		jobID, attemptID = laptopPilot3JobID, laptopPilot3AttemptID
		workerSHA, workerSize = laptopPilot3WorkerSHA, 48391
	}
	if !filepath.IsAbs(base) || !strings.Contains(filepath.ToSlash(base), evidencePath) {
		t.Fatalf("evidence directory is outside the admitted laptop evidence: %q", base)
	}
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "cpu")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	_, repoRoot := laptopTestRoots(t)
	archive := filepath.Join(repoRoot, "scenarios/device-studio/docs/internal/goal/evidence", archivePath, jobID, attemptID)
	files := []struct {
		name, sha string
		size      int64
	}{
		{"physical-worker-input.json", workerSHA, workerSize},
		{"device-normalized.glb", laptopArchiveDeviceSHA, 4638192},
		{"scene-normalized.glb", laptopArchiveSceneSHA, 421128},
		{"screen.png", laptopArchiveScreenSHA, 7450},
	}
	if adapterMode == "archive" {
		files = append(files,
			struct {
				name, sha string
				size      int64
			}{"physical-asset-02.png", "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", 16152869},
			struct {
				name, sha string
				size      int64
			}{"physical-asset-03.png", "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", 355941},
			struct {
				name, sha string
				size      int64
			}{"physical-asset-04.png", "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", 5152752},
			struct {
				name, sha string
				size      int64
			}{"physical-asset-05.png", "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", 5302649},
		)
	}
	var bound []laptopDiagnosticFile
	var workerInputBytes []byte
	for _, source := range files {
		path := filepath.Join(archive, source.name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() != source.size {
			t.Fatalf("archived %s is not the pinned regular file: %v", source.name, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || blenderDigest(data) != source.sha {
			t.Fatalf("archived %s differs from the immutable pilot input: %v", source.name, err)
		}
		if err := writeDurable(dir, source.name, data); err != nil {
			t.Fatal(err)
		}
		if source.name == "physical-worker-input.json" {
			workerInputBytes = data
		}
		bound = append(bound, laptopDiagnosticFile{Name: source.name, SHA: source.sha, Size: source.size})
	}
	var sceneTextures []laptopDiagnosticFile
	if adapterMode == "archive" {
		var input struct {
			Assets []struct {
				Name      string `json:"name"`
				Role      string `json:"role"`
				Reference struct {
					Role string `json:"role"`
					SHA  string `json:"sha256"`
					Size int64  `json:"size_bytes"`
				} `json:"reference"`
			} `json:"assets"`
		}
		if err := json.Unmarshal(workerInputBytes, &input); err != nil {
			t.Fatalf("decode pinned pilot-3 asset list: %v", err)
		}
		want := make(map[string]laptopDiagnosticFile, 4)
		for _, entry := range files[4:] {
			want[entry.name] = laptopDiagnosticFile{Name: entry.name, SHA: entry.sha, Size: entry.size}
		}
		for _, asset := range input.Assets {
			if asset.Role != "scene_texture" {
				continue
			}
			expected := want[asset.Name]
			if asset.Reference.Role != "texture_image" || expected.Name == "" || expected.SHA != asset.Reference.SHA || expected.Size != asset.Reference.Size {
				t.Fatalf("pilot-3 scene texture is not one of the pinned manifest assets: %+v", asset)
			}
			delete(want, asset.Name)
			sceneTextures = append(sceneTextures, laptopDiagnosticFile{Name: asset.Name, SHA: asset.Reference.SHA, Size: asset.Reference.Size})
		}
		if len(want) != 0 || len(sceneTextures) != 4 {
			t.Fatalf("pilot-3 scene textures do not match the four pinned archive payloads: missing=%v staged=%d", want, len(sceneTextures))
		}
	}
	var adapter []byte
	switch adapterMode {
	case "original":
		var err error
		adapter, err = os.ReadFile(filepath.Join(archive, "physical_blender.py"))
		if err != nil || blenderDigest(adapter) != laptopArchiveAdapterSHA {
			t.Fatalf("archived diagnostic-1 adapter differs from its retained identity: %v", err)
		}
	case "archive":
		var err error
		adapter, err = os.ReadFile(filepath.Join(archive, PhysicalAdapterName))
		if err != nil || blenderDigest(adapter) != laptopPilot3AdapterSHA {
			t.Fatalf("framed pilot-3 adapter differs from its retained identity: %v", err)
		}
	case "current":
		adapter = PhysicalAdapterBytes()
	default:
		t.Fatalf("unsupported diagnostic adapter mode %q", adapterMode)
	}
	if err := writeDurable(dir, PhysicalAdapterName, adapter); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(dir, "physical_semantic_probe.py", physicalLaptopArchiveProbe); err != nil {
		t.Fatal(err)
	}
	probeSHA := blenderDigest(physicalLaptopArchiveProbe)
	manifestMode := "archived-laptop-hierarchy-v1"
	if adapterMode == "archive" {
		manifestMode = "archived-laptop-clearance-v1"
	}
	manifest := laptopDiagnosticManifest{
		Version: 1, Mode: manifestMode, JobID: jobID,
		AttemptID: attemptID, AdapterName: PhysicalAdapterName,
		AdapterMode: adapterMode, AdapterSHA: blenderDigest(adapter), ProbeSHA: probeSHA,
		WorkerInput: bound[0], DeviceGLB: bound[1], SceneGLB: bound[2], ScreenPNG: bound[3],
		SceneTextures: sceneTextures,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeDurable(dir, "physical-semantic-probe-manifest.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	_, profile, err := ResolvePhysicalBlenderProfile(context.Background())
	if err != nil || profile.GetBlender() == nil || profile.GetBlender().GetPath() == "" {
		t.Fatalf("resolve pinned Blender profile: %v", err)
	}
	runOwnedBlenderOperation(t, dir, profile.GetBlender().GetPath(), profile.GetArtifactIdentity(), "CPU", "physical-semantic-probe", "", blenderDigest(manifestBytes), manifest.AdapterSHA, func() error {
		if err := verifyLaptopDiagnosticReport(dir, manifest); err != nil {
			return err
		}
		if adapterMode == "archive" {
			return verifyLaptopClearanceReport(dir, manifest)
		}
		return nil
	})
}

func verifyLaptopClearanceReport(dir string, manifest laptopDiagnosticManifest) error {
	data, err := os.ReadFile(filepath.Join(dir, "physical-laptop-clearance-diagnostics.json"))
	if err != nil || len(data) > 1<<20 {
		return fmt.Errorf("bounded laptop clearance diagnostics unavailable: %v", err)
	}
	var report struct {
		Version    int       `json:"version"`
		JobID      string    `json:"job_id"`
		AttemptID  string    `json:"attempt_id"`
		InputSHA   string    `json:"worker_input_sha256"`
		AdapterSHA string    `json:"adapter_sha256"`
		Range      []float64 `json:"angle_range_rad"`
		Samples    int       `json:"angular_samples"`
		Ownership  map[string]struct {
			PartID    string `json:"part_id"`
			NodeIndex int    `json:"node_index"`
			Motion    string `json:"motion"`
			Triangles int    `json:"triangle_count"`
		} `json:"triangle_ownership"`
		Broadphase []struct {
			Moving          string `json:"moving_part_id"`
			Fixed           string `json:"fixed_part_id"`
			MovingTriangles int    `json:"moving_triangle_count"`
			FixedTriangles  int    `json:"fixed_triangle_count"`
			Candidate       bool   `json:"candidate_pair"`
		} `json:"analytic_swept_aabb_broadphase"`
		Measurements []struct {
			Angle     float64 `json:"angle_rad"`
			Moving    string  `json:"moving_part_id"`
			Fixed     string  `json:"fixed_part_id"`
			PairCount int     `json:"bvh_intersection_pair_count"`
			Pairs     []struct {
				Moving int `json:"moving_triangle_id"`
				Fixed  int `json:"fixed_triangle_id"`
			} `json:"bvh_intersection_triangle_pairs"`
			Separation *float64 `json:"sampled_raw_vertex_to_opposite_surface_separation_m"`
		} `json:"sampled_bvh_measurements"`
		Claim string `json:"claim_limit"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}
	if report.Version != 1 || report.JobID != manifest.JobID || report.AttemptID != manifest.AttemptID || report.InputSHA != manifest.WorkerInput.SHA || report.AdapterSHA != manifest.AdapterSHA || len(report.Range) != 2 || report.Range[0] != 0 || math.Abs(report.Range[1]-1.919) > 1e-12 || report.Samples != 49 || len(report.Ownership) != 4 || len(report.Broadphase) != 4 || len(report.Measurements) < 4 || !strings.Contains(report.Claim, "do not prove continuous") {
		return errors.New("laptop clearance report does not bind the framed pilot, four component ownerships, bounded full-range samples, and claim limit")
	}
	wantOwners := map[string]struct {
		node   int
		motion string
	}{
		"laptop-base-geometry":         {0, "fixed_base"},
		"laptop-hinge-review-geometry": {2, "unresolved_review_fixed_for_diagnostic"},
		"laptop-lid-geometry":          {4, "moving_with_lid"},
		"laptop-screen-surface":        {5, "moving_with_lid"},
	}
	for id, want := range wantOwners {
		got, ok := report.Ownership[id]
		if !ok || got.PartID != id || got.NodeIndex != want.node || got.Motion != want.motion || got.Triangles < 1 {
			return fmt.Errorf("clearance ownership for %s differs from the pinned pilot: %+v", id, got)
		}
	}
	wantPairs := map[string]bool{
		"laptop-lid-geometry/laptop-base-geometry":           true,
		"laptop-lid-geometry/laptop-hinge-review-geometry":   true,
		"laptop-screen-surface/laptop-base-geometry":         true,
		"laptop-screen-surface/laptop-hinge-review-geometry": true,
	}
	for _, pair := range report.Broadphase {
		key := pair.Moving + "/" + pair.Fixed
		if !wantPairs[key] || pair.MovingTriangles != report.Ownership[pair.Moving].Triangles || pair.FixedTriangles != report.Ownership[pair.Fixed].Triangles {
			return fmt.Errorf("analytic swept-AABB broadphase differs from pinned component pair %q", key)
		}
		delete(wantPairs, key)
	}
	if len(wantPairs) != 0 {
		return errors.New("analytic swept-AABB broadphase omitted a moving/fixed component pair")
	}
	var sawStart, sawEnd bool
	for _, measurement := range report.Measurements {
		if measurement.Angle < report.Range[0] || measurement.Angle > report.Range[1] || measurement.Separation == nil || *measurement.Separation < 0 || measurement.PairCount < len(measurement.Pairs) {
			return errors.New("sampled clearance measurement has an invalid angle, separation, or intersection count")
		}
		sawStart = sawStart || measurement.Angle == report.Range[0]
		sawEnd = sawEnd || math.Abs(measurement.Angle-report.Range[1]) < 1e-12
		if _, ok := wantOwners[measurement.Moving]; !ok || (measurement.Moving != "laptop-lid-geometry" && measurement.Moving != "laptop-screen-surface") {
			return errors.New("sampled clearance measurement names an unexpected moving part")
		}
		if measurement.Fixed != "laptop-base-geometry" && measurement.Fixed != "laptop-hinge-review-geometry" {
			return errors.New("sampled clearance measurement names an unexpected fixed review part")
		}
		for _, pair := range measurement.Pairs {
			if pair.Moving < 0 || pair.Moving >= report.Ownership[measurement.Moving].Triangles || pair.Fixed < 0 || pair.Fixed >= report.Ownership[measurement.Fixed].Triangles {
				return errors.New("BVH intersection references a triangle outside its pinned owner mesh")
			}
		}
	}
	if !sawStart || !sawEnd {
		return errors.New("sampled clearance measurements do not include both ends of the admitted range")
	}
	return nil
}

func verifyLaptopDiagnosticReport(dir string, manifest laptopDiagnosticManifest) error {
	data, err := os.ReadFile(filepath.Join(dir, "physical-laptop-hierarchy-diagnostics.json"))
	if err != nil || len(data) > 256<<10 {
		return fmt.Errorf("bounded laptop hierarchy diagnostics unavailable: %v", err)
	}
	var report laptopDiagnosticReport
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}
	wantStatus := "diagnostic-only"
	if manifest.AdapterMode == "current" {
		wantStatus = "corrected-adapter-measurements"
	} else if manifest.AdapterMode == "archive" {
		wantStatus = "pilot3-archive-diagnostic"
	}
	if report.Version != 1 || report.Status != wantStatus || report.JobID != manifest.JobID || report.AttemptID != manifest.AttemptID || report.AdapterSHA != manifest.AdapterSHA || report.InputSHA != manifest.WorkerInput.SHA || len(report.Frames) != 3 || len(report.Hierarchy) < 8 {
		return errors.New("laptop diagnostic receipt does not bind the archived attempt or all three samples")
	}
	for i, want := range []int{0, 24, 47} {
		if report.Frames[i].Ordinal != want || len(report.Frames[i].Parts) < 3 {
			return fmt.Errorf("laptop diagnostic frame %d lacks the pinned pose or component bounds", want)
		}
	}
	if manifest.AdapterMode == "current" {
		if report.RestMatrixError > 1e-6 || report.ABAMatrixError > 1e-6 || report.TravelMatrixDelta < .1 {
			return fmt.Errorf("corrected adapter failed rest/travel/A-B-A checks: rest %.9g travel %.9g A-B-A %.9g", report.RestMatrixError, report.TravelMatrixDelta, report.ABAMatrixError)
		}
		base0, base47 := findLaptopBounds(report.Frames[0].Parts, "laptop-base-geometry"), findLaptopBounds(report.Frames[2].Parts, "laptop-base-geometry")
		lid0, lid47 := findLaptopBounds(report.Frames[0].Parts, "laptop-lid-geometry"), findLaptopBounds(report.Frames[2].Parts, "laptop-lid-geometry")
		screen0, screen47 := findLaptopBounds(report.Frames[0].Parts, "laptop-screen-surface"), findLaptopBounds(report.Frames[2].Parts, "laptop-screen-surface")
		if base0 == nil || base47 == nil || lid0 == nil || lid47 == nil || screen0 == nil || screen47 == nil || laptopBoundsDelta(base0, base47) > 1e-6 || laptopBoundsDelta(lid0, lid47) < .1 || laptopBoundsDelta(screen0, screen47) < .1 {
			return errors.New("corrected hinge pose does not preserve base while moving lid and screen")
		}
	}
	return nil
}

func findLaptopBounds(parts []json.RawMessage, id string) [][]float64 {
	for _, raw := range parts {
		var part struct {
			PartID string      `json:"part_id"`
			Bounds [][]float64 `json:"world_bounds"`
		}
		if json.Unmarshal(raw, &part) == nil && part.PartID == id {
			return part.Bounds
		}
	}
	return nil
}

func laptopBoundsDelta(a, b [][]float64) float64 {
	if len(a) != len(b) {
		return 1e9
	}
	var delta float64
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return 1e9
		}
		for j := range a[i] {
			if d := math.Abs(a[i][j] - b[i][j]); d > delta {
				delta = d
			}
		}
	}
	return delta
}
