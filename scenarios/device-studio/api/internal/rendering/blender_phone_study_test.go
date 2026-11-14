package rendering

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	phoneStudyManifestSHA = "124bd6629befc71ad6e551fea7ae70f6f9c1de3050fb214a77439ac2b0214b2c"
	phoneStudyScriptSHA   = "b8e77d8292d96e6af154084e42be6e937fa0519092aaf84374d576a641500195"
	phoneStudyProfileID   = "device-studio-phone-scene-study-v1"
)

type pinnedPhoneAsset struct {
	source, stage, sha string
	size               int64
}

type phoneStudyOutput struct {
	Path              string    `json:"path"`
	Bytes             int64     `json:"bytes"`
	SHA               string    `json:"sha256"`
	Dimensions        []int     `json:"dimensions_pixels"`
	Yaw               float64   `json:"yaw_degrees"`
	FrameIndex        *int      `json:"frame_index,omitempty"`
	PTSNum            *int      `json:"pts_num,omitempty"`
	PTSDen            *int      `json:"pts_den,omitempty"`
	AnchorTranslation []float64 `json:"anchor_world_translation_meters,omitempty"`
	Contacts          struct {
		CameraEnd struct {
			Count int       `json:"vertex_count"`
			Z     []float64 `json:"world_z_bounds_meters"`
		} `json:"camera_end"`
		ConnectorEnd struct {
			Count int       `json:"vertex_count"`
			Z     []float64 `json:"world_z_bounds_meters"`
		} `json:"connector_end"`
	} `json:"support_contact_regions"`
	Bounds struct {
		Min []float64 `json:"min_meters"`
		Max []float64 `json:"max_meters"`
	} `json:"phone_bounds_after_pose"`
	ScreenNormal []float64 `json:"screen_front_normal_world"`
}

type phoneStudyTimeline struct {
	Profile       string `json:"profile"`
	FrameCount    int    `json:"frame_count"`
	RateNum       int    `json:"frame_rate_num"`
	RateDen       int    `json:"frame_rate_den"`
	DurationTicks int    `json:"duration_ticks"`
	LastIndex     int    `json:"last_frame_index"`
	FinalHold     int    `json:"final_pose_hold_ticks"`
}

type phoneStudyCamera struct {
	SensorWidthMM       float64     `json:"sensor_width_mm"`
	SensorHeightMM      float64     `json:"sensor_height_mm"`
	SensorFit           string      `json:"sensor_fit"`
	ClipStartMeters     float64     `json:"clip_start_meters"`
	ClipEndMeters       float64     `json:"clip_end_meters"`
	CameraMatrixWorld   [][]float64 `json:"camera_matrix_world"`
	FocusMatrixWorld    [][]float64 `json:"focus_object_matrix_world"`
	TargetedPhoneAnchor bool        `json:"targeted_phone_anchor"`
	DOFFStop            float64     `json:"dof_fstop"`
}

type phoneStudyReceipt struct {
	Schema   int    `json:"schema_version"`
	Profile  string `json:"profile_id"`
	Selected string `json:"selected_profile"`
	Manifest string `json:"manifest_sha256"`
	Script   string `json:"script_sha256"`
	Blender  struct {
		Profile string `json:"profile_id"`
		Tree    string `json:"expected_tree_sha256"`
		Version []int  `json:"version_numeric"`
	} `json:"blender_profile"`
	Version     string `json:"blender_version"`
	BuildHash   string `json:"blender_build_hash"`
	Backend     string `json:"backend_requested"`
	Effective   string `json:"backend_effective"`
	ComputeType string `json:"backend_compute_device_type"`
	Engine      string `json:"engine"`
	Model       struct {
		SHA       string `json:"sha256"`
		Bytes     int64  `json:"bytes"`
		Meshes    int    `json:"mesh_count"`
		Triangles int    `json:"triangle_count"`
	} `json:"model"`
	Screen struct {
		SHA         string `json:"replacement_sha256"`
		Bytes       int64  `json:"replacement_bytes"`
		E2Receipt   string `json:"e2_composition_receipt_sha256"`
		EmbeddedSHA string `json:"glb_original_embedded_sha256"`
	} `json:"screen"`
	Material struct {
		ID   string `json:"asset_id"`
		Maps map[string]struct {
			SHA   string `json:"sha256"`
			Bytes int64  `json:"bytes"`
		} `json:"maps"`
	} `json:"table_material"`
	Render struct {
		Resolution []int  `json:"resolution_pixels"`
		Samples    int    `json:"samples"`
		Threads    int    `json:"threads"`
		ThreadMode string `json:"threads_mode"`
	} `json:"render"`
	Camera   phoneStudyCamera    `json:"camera"`
	Outputs  []phoneStudyOutput  `json:"outputs"`
	Timeline *phoneStudyTimeline `json:"timeline,omitempty"`
}

// This test is opt-in evidence only. It has no API/job path and invokes the
// fixed reviewed scene script only through the shared owned-operation runner.
func TestOptInBlenderPhoneSceneStudy(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_STUDY") != "1" {
		t.Skip("opt-in Blender phone scene study")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("qualified Blender profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_PHONE_EVIDENCE_DIR"))
	if !filepath.IsAbs(root) || !strings.Contains(filepath.ToSlash(root), "/goal/evidence/E3/scene-studies/phone/runs/") {
		t.Fatalf("evidence path outside phone scene-study runs: %q", root)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	profileBytes, profile := resolveBlenderProfile(t)
	if err := writeDurable(root, "resolved-profile.json", profileBytes); err != nil {
		t.Fatal(err)
	}
	bin := profile.GetBlender()
	if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || profile.GetProfileId() != blenderProfileID || profile.GetArtifactIdentity() != blenderArtifact || profile.GetExpectedTreeSha256() != blenderTreeSHA || bin == nil || bin.GetVersion() != blenderVersion || bin.GetSha256() != blenderBinarySHA || bin.GetArchiveMemberPath() != blenderArchive+"/blender" || !filepath.IsAbs(bin.GetPath()) {
		t.Fatalf("profile differs from pinned artifact: %+v", profile)
	}
	if filepath.Base(filepath.Dir(bin.GetPath())) != blenderArchive {
		t.Fatalf("binary outside expected archive root: %q", bin.GetPath())
	}
	backend := os.Getenv("DEVICE_STUDIO_E3_PHONE_BACKEND")
	if backend != "CPU" && backend != "CUDA" && backend != "OPTIX" {
		t.Fatalf("backend must be explicit CPU, CUDA, or OPTIX: %q", backend)
	}
	selected := os.Getenv("DEVICE_STUDIO_E3_PHONE_PROFILE")
	if selected != "study" && selected != "hero" && selected != "posecontactsheet" && selected != "motion-yaw-24" {
		t.Fatalf("profile must be explicit study, hero, or posecontactsheet: %q", selected)
	}
	attempt := filepath.Join(root, strings.ToLower(backend)+"-"+selected)
	if err := os.Mkdir(attempt, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestSHA, scriptSHA, stagedSHA := stagePhoneStudyInputs(t, attempt, selected, backend)
	runOwnedBlenderOperation(t, attempt, bin.GetPath(), profile.GetArtifactIdentity(), backend, "phone-scene-study", selected, stagedSHA, scriptSHA, func() error {
		return verifyPhoneStudy(attempt, selected, backend, stagedSHA, scriptSHA, manifestSHA)
	})
	if selected == "motion-yaw-24" {
		runOwnedPhoneMotionEncodeOperation(t, attempt)
	}
}

func stagePhoneStudyInputs(t *testing.T, attempt, selected, backend string) (string, string, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate harness source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	scenario := filepath.Join(repo, "scenarios/device-studio")
	study := filepath.Join(scenario, "docs/internal/goal/evidence/E3/scene-studies/phone")
	manifestBytes := readPinnedPhoneFile(t, filepath.Join(study, "phone-scene-study.json"), phoneStudyManifestSHA, 1<<20)
	scriptBytes := readPinnedPhoneFile(t, filepath.Join(study, "phone_scene_study.py"), phoneStudyScriptSHA, 1<<20)
	assets := []pinnedPhoneAsset{
		{"/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/authored/original-iphone16pro-iter03.glb", "inputs/phone.glb", "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36", 2583052},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E2/presentations/phone-v3-auto-1206x2622.png"), "inputs/phone-screen.png", "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8", 611442},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Color.jpg"), "inputs/table-color.jpg", "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", 16152869},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_NormalGL.jpg"), "inputs/table-normal.jpg", "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", 5152752},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Roughness.jpg"), "inputs/table-roughness.jpg", "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", 5302649},
		{filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001_4K-JPG_Displacement.jpg"), "inputs/table-displacement.jpg", "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf", 355941},
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	bindings := make([]map[string]any, 0, len(assets))
	var total int64
	for _, a := range assets {
		data := readPinnedPhoneFile(t, a.source, a.sha, a.size)
		if int64(len(data)) != a.size || a.size > 20<<20 {
			t.Fatalf("pinned phone input size differs: %s", a.source)
		}
		total += int64(len(data))
		if total > 40<<20 {
			t.Fatal("phone input aggregate exceeded fixed 40 MiB bound")
		}
		if err := writeDurable(attempt, a.stage, data); err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, map[string]any{"source": a.source, "stage": a.stage, "sha256": a.sha, "bytes": a.size})
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["backend"], manifest["selected_render_profile"] = backend, selected
	staged, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	staged = append(staged, '\n')
	if err = writeDurable(attempt, "phone-scene-study.json", staged); err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "phone_scene_study.py", scriptBytes); err != nil {
		t.Fatal(err)
	}
	proof, _ := json.MarshalIndent(map[string]any{"source_manifest_sha256": phoneStudyManifestSHA, "staged_manifest_sha256": blenderDigest(staged), "script_sha256": phoneStudyScriptSHA, "backend": backend, "profile": selected, "assets": bindings}, "", "  ")
	if err = writeDurable(attempt, "source-bindings.json", append(proof, '\n')); err != nil {
		t.Fatal(err)
	}
	return phoneStudyManifestSHA, phoneStudyScriptSHA, blenderDigest(staged)
}

func readPinnedPhoneFile(t *testing.T, path, expectedSHA string, max int64) []byte {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > max {
		t.Fatalf("pinned phone evidence file unavailable/unsafe: %s (%v)", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) != info.Size() || blenderDigest(data) != expectedSHA {
		t.Fatalf("pinned phone evidence digest/size mismatch: %s", path)
	}
	return data
}

func verifyPhoneStudy(dir, selected, backend, stagedSHA, scriptSHA, sourceSHA string) error {
	path := filepath.Join(dir, "outputs/qualification-receipt.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return errors.New("phone qualification receipt missing or unsafe")
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) != info.Size() {
		return errors.New("phone qualification receipt read failed")
	}
	var r phoneStudyReceipt
	if err = json.Unmarshal(data, &r); err != nil {
		return err
	}
	if r.Schema != 1 || r.Profile != phoneStudyProfileID || r.Selected != selected || r.Manifest != stagedSHA || r.Script != scriptSHA || sourceSHA != phoneStudyManifestSHA || r.Blender.Profile != blenderProfileID || r.Blender.Tree != blenderTreeSHA || len(r.Blender.Version) != 3 || r.Blender.Version[0] != 5 || r.Blender.Version[1] != 2 || r.Blender.Version[2] != 2 || r.Version != strings.TrimPrefix(blenderVersion, "Blender ") || r.BuildHash == "" || r.Backend != backend || r.Engine != "CYCLES" || r.Model.SHA != "46e7c79c1b54725e5f5e9ff876a3bd55f8c3eeba45e1d01e1a230ff5e7fd5b36" || r.Model.Bytes != 2583052 || r.Model.Meshes != 57 || r.Model.Triangles <= 0 || r.Screen.SHA != "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8" || r.Screen.Bytes != 611442 || r.Screen.E2Receipt != "aefbf24dae185a9338fd96e2248347e1ff1238e8baaa102674f678dfb2df92db" || r.Screen.EmbeddedSHA != "f4b6c7ed95981727bbb94abf1b3958a27c27faf5618e86e2b06b3d8047f3fb6d" || r.Material.ID != "ambientCG-Travertine001" || r.Render.Threads != 2 || r.Render.ThreadMode != "FIXED" || !validPhoneCameraFacts(r.Camera) {
		return errors.New("phone receipt does not match pinned source, profile, or operation")
	}
	width, height, samples, names, yaws := phoneStudyProfile(selected)
	if len(r.Render.Resolution) != 2 || r.Render.Resolution[0] != width || r.Render.Resolution[1] != height || r.Render.Samples != samples || len(r.Outputs) != len(names) {
		return errors.New("phone receipt output profile differs")
	}
	if selected == "motion-yaw-24" {
		if err := verifyPhoneMotionTimeline(r, yaws); err != nil {
			return err
		}
	} else if r.Timeline != nil {
		return errors.New("static phone receipt unexpectedly contains a motion timeline")
	}
	if backend == "CPU" && (r.Effective != "CPU" || r.ComputeType != "CPU") || backend != "CPU" && (r.Effective != "GPU" || r.ComputeType != backend) {
		return errors.New("phone receipt did not use requested backend exactly")
	}
	var totalMotionPNGBytes int64
	for i, out := range r.Outputs {
		maxBytes := int64(64 << 20)
		if selected == "motion-yaw-24" {
			maxBytes = 7 << 20
		}
		if out.Path != names[i] || math.Abs(out.Yaw-yaws[i]) > 1e-9 || len(out.Dimensions) != 2 || out.Dimensions[0] != width || out.Dimensions[1] != height || out.Bytes <= 0 || out.Bytes > maxBytes || len(out.SHA) != 64 {
			return errors.New("phone receipt output entry differs from fixed profile")
		}
		if selected == "motion-yaw-24" {
			totalMotionPNGBytes += out.Bytes
			if err := validatePhoneMotionOutput(out, i, yaws[i]); err != nil {
				return err
			}
		} else if out.FrameIndex != nil || out.PTSNum != nil || out.PTSDen != nil {
			return errors.New("static phone receipt unexpectedly contains motion frame metadata")
		}
		if err := verifyPhoneStudyPNG(filepath.Join(dir, "outputs", names[i]), out, width, height); err != nil {
			return err
		}
	}
	if selected == "motion-yaw-24" && totalMotionPNGBytes > 336<<20 {
		return errors.New("phone motion PNG sequence exceeds the fixed aggregate size bound")
	}
	wantMaps := map[string]string{"color": "e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8", "normal_gl": "68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b", "roughness": "a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90", "displacement": "64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf"}
	if len(r.Material.Maps) != len(wantMaps) {
		return errors.New("phone receipt lacks all four pinned material maps")
	}
	for role, sha := range wantMaps {
		if r.Material.Maps[role].SHA != sha {
			return errors.New("phone receipt material map digest differs: " + role)
		}
	}
	return verifyPhoneOutputDirectory(dir, names)
}

func validPhoneCameraFacts(camera phoneStudyCamera) bool {
	if !finitePhoneMotion(camera.SensorWidthMM, camera.SensorHeightMM, camera.ClipStartMeters, camera.ClipEndMeters, camera.DOFFStop) || camera.SensorWidthMM <= 0 || camera.SensorHeightMM <= 0 || camera.SensorFit != "AUTO" || camera.ClipStartMeters <= 0 || camera.ClipEndMeters <= camera.ClipStartMeters || !camera.TargetedPhoneAnchor || camera.DOFFStop <= 0 {
		return false
	}
	for _, matrix := range [][][]float64{camera.CameraMatrixWorld, camera.FocusMatrixWorld} {
		if len(matrix) != 4 {
			return false
		}
		for _, row := range matrix {
			if len(row) != 4 || !finitePhoneMotion(row...) {
				return false
			}
		}
	}
	return true
}

func phoneStudyProfile(profile string) (int, int, int, []string, []float64) {
	switch profile {
	case "study":
		return 960, 540, 64, []string{"study-stationary.png"}, []float64{-13}
	case "hero":
		return 2560, 1440, 256, []string{"hero-stationary.png"}, []float64{-13}
	case "posecontactsheet":
		return 960, 540, 96, []string{"pose-yaw-m13.png", "pose-yaw-p32.png", "pose-yaw-p77.png"}, []float64{-13, 32, 77}
	case "motion-yaw-24":
		yaws := make([]float64, 48)
		names := make([]string, 48)
		for i := range yaws {
			u := float64(i) / 47
			s := 3*u*u - 2*u*u*u
			yaws[i] = -13 + 45*s
			names[i] = fmt.Sprintf("motion-yaw-frame-%04d.png", i)
		}
		return 1920, 1080, 128, names, yaws
	default:
		return 0, 0, 0, nil, nil
	}
}

func verifyPhoneMotionTimeline(r phoneStudyReceipt, yaws []float64) error {
	timeline := r.Timeline
	if timeline == nil || timeline.Profile != "rigid-world-z-smoothstep-v1" || timeline.FrameCount != 48 || timeline.RateNum != 24 || timeline.RateDen != 1 || timeline.DurationTicks != 48 || timeline.LastIndex != 47 || timeline.FinalHold != 1 || len(r.Outputs) != 48 || len(yaws) != 48 {
		return errors.New("motion receipt timeline differs from fixed 48-frame, 24fps, two-second profile")
	}
	var placement []float64
	for i, frame := range r.Outputs {
		if err := validatePhoneMotionOutput(frame, i, yaws[i]); err != nil {
			return err
		}
		if i == 0 {
			placement = frame.AnchorTranslation
		} else if len(frame.AnchorTranslation) != len(placement) {
			return errors.New("motion frame placement vector changed dimensions")
		} else {
			for axis := range placement {
				if math.Abs(frame.AnchorTranslation[axis]-placement[axis]) > 1e-9 {
					return errors.New("motion frame placement changed instead of preserving its initial world translation")
				}
			}
		}
	}
	return nil
}

func validatePhoneMotionOutput(frame phoneStudyOutput, ordinal int, yaw float64) error {
	if frame.FrameIndex == nil || *frame.FrameIndex != ordinal || frame.PTSNum == nil || *frame.PTSNum != ordinal || frame.PTSDen == nil || *frame.PTSDen != 24 || !finitePhoneMotion(frame.Yaw) || math.Abs(frame.Yaw-yaw) > 1e-9 {
		return fmt.Errorf("motion receipt frame %d has a gap, duplicate, wrong PTS, or wrong pose", ordinal)
	}
	if len(frame.Bounds.Min) != 3 || len(frame.Bounds.Max) != 3 || len(frame.ScreenNormal) != 3 || len(frame.AnchorTranslation) != 3 || !finitePhoneMotion(frame.Bounds.Min...) || !finitePhoneMotion(frame.Bounds.Max...) || !finitePhoneMotion(frame.ScreenNormal...) || !finitePhoneMotion(frame.AnchorTranslation...) || frame.Bounds.Min[2] < 0.749999 || frame.Bounds.Min[2] > 0.750001 || frame.ScreenNormal[2] <= 0.995 || !validPhoneMotionContact(frame.Contacts.CameraEnd.Count, frame.Contacts.CameraEnd.Z) || !validPhoneMotionContact(frame.Contacts.ConnectorEnd.Count, frame.Contacts.ConnectorEnd.Z) {
		return fmt.Errorf("motion frame %d lacks verified bounds, screen orientation, or separated support", ordinal)
	}
	for axis := 0; axis < 3; axis++ {
		if frame.Bounds.Max[axis] <= frame.Bounds.Min[axis] {
			return fmt.Errorf("motion frame %d has invalid world bounds", ordinal)
		}
	}
	return nil
}

func validPhoneMotionContact(count int, z []float64) bool {
	return count > 0 && len(z) == 2 && finitePhoneMotion(z...) && z[0] >= 0.7499 && z[0] <= 0.7501 && z[1] >= 0.7499 && z[1] <= 0.7501
}

func finitePhoneMotion(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func verifyPhoneOutputDirectory(dir string, expected []string) error {
	outputDir := filepath.Join(dir, "outputs")
	info, err := os.Lstat(outputDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("phone output path is not a regular directory")
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil || len(entries) != len(expected)+1 {
		return errors.New("phone output directory contains missing or unexpected files")
	}
	allowed := map[string]bool{"qualification-receipt.json": true}
	for _, name := range expected {
		allowed[name] = true
	}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if !allowed[entry.Name()] || infoErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("phone output directory contains an unapproved entry")
		}
	}
	return nil
}

func verifyPhoneStudyPNG(path string, out phoneStudyOutput, width, height int) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != out.Bytes || info.Size() <= 0 || info.Size() > 64<<20 {
		return errors.New("phone PNG missing or outside fixed size bound")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, (64<<20)+1)); err != nil || hex.EncodeToString(h.Sum(nil)) != out.SHA {
		return errors.New("phone PNG digest mismatch")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	cfg, err := png.DecodeConfig(io.LimitReader(f, (64<<20)+1))
	if err != nil || cfg.Width != width || cfg.Height != height {
		return errors.New("phone PNG header or dimensions mismatch")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	img, err := png.Decode(io.LimitReader(f, (64<<20)+1))
	if err != nil || img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		return errors.New("phone PNG full decode failed")
	}
	return nil
}
