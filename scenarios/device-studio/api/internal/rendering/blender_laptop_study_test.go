package rendering

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	laptopStudyManifestSHA  = "528069ca045bfbcd8c2fd4019ec1747ed2bcf32cd3d6257faac5d98acd9032f9"
	laptopStudyScriptSHA    = "c923d7d197caecb424bd22bb9573fcfe83d81dd4e3cb73ef35ba01d808718dba"
	laptopStudyProfileID    = "device-studio-laptop-partition-diagnostic-v1"
	laptopStudySourceSHA    = "4b7a6736e12e105b00556ca6397f3dffe559a1c518954209df08828f186d8a78"
	laptopStudyInventorySHA = "cfe2d0a9e9b03809a07cfdc4d4d823080216c74a1fc025cf73370109be343af4"
)

type laptopStudyOutput struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"size_bytes"`
}

type laptopStudyReceipt struct {
	Profile      string                       `json:"profile_id"`
	Source       string                       `json:"source_path"`
	SourceSHA    string                       `json:"source_sha256"`
	InventorySHA string                       `json:"component_inventory_sha256"`
	ManifestSHA  string                       `json:"manifest_sha256"`
	ScriptSHA    string                       `json:"script_sha256"`
	Version      string                       `json:"blender_version"`
	Triangles    int                          `json:"triangle_count"`
	Components   int                          `json:"component_count"`
	Closure      []int                        `json:"closure_from_source_open_degrees"`
	Outputs      map[string]laptopStudyOutput `json:"outputs"`
	Summary      struct {
		ProjectionMargins map[string]float64 `json:"projection_margin_fraction"`
	} `json:"summary"`
}

// This is opt-in evidence only. It stages fixed, reviewed inputs and runs the
// diagnostic through the owned Blender operation; it never edits the source GLB.
func TestOptInBlenderLaptopPartitionStudy(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STUDY") != "1" {
		t.Skip("opt-in Blender laptop partition study")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("qualified Blender profile supports Linux/amd64 only")
	}
	root := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_LAPTOP_STUDY_DIR"))
	portable := filepath.ToSlash(root)
	if !filepath.IsAbs(root) || !strings.Contains(portable, "/goal/evidence/E3/scene-studies/laptop/runs/") || filepath.Base(root) == "." || filepath.Base(root) == "runs" {
		t.Fatalf("evidence path outside laptop scene-study runs: %q", root)
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
	attempt := filepath.Join(root, "cpu-partition-diagnostic")
	if err := os.Mkdir(attempt, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestSHA, scriptSHA := stageLaptopStudyInputs(t, attempt)
	runOwnedBlenderOperation(t, attempt, bin.GetPath(), profile.GetArtifactIdentity(), "CPU", "laptop-partition-study", "", manifestSHA, scriptSHA, func() error {
		return verifyLaptopStudy(attempt, manifestSHA, scriptSHA)
	})
}

func stageLaptopStudyInputs(t *testing.T, attempt string) (string, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate harness source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	study := filepath.Join(repo, "scenarios/device-studio/docs/internal/goal/evidence/E3/scene-studies/laptop")
	manifest := readPinnedPhoneFile(t, filepath.Join(study, "laptop-partition-diagnostic.json"), laptopStudyManifestSHA, 1<<20)
	script := readPinnedPhoneFile(t, filepath.Join(study, "laptop_partition_diagnostic.py"), laptopStudyScriptSHA, 1<<20)
	assets := []pinnedPhoneAsset{
		{"/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/imported-laptop/normalized-macbook.glb", "inputs/laptop.glb", laptopStudySourceSHA, 7 << 20},
		{filepath.Join(study, "component-candidates.json"), "inputs/component-candidates.json", laptopStudyInventorySHA, 1 << 20},
	}
	if err := os.Mkdir(filepath.Join(attempt, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	bindings := make([]map[string]any, 0, len(assets))
	var total int64
	for _, asset := range assets {
		data := readPinnedPhoneFile(t, asset.source, asset.sha, asset.size)
		total += int64(len(data))
		if total > 8<<20 {
			t.Fatal("laptop input aggregate exceeded fixed 8 MiB bound")
		}
		if err := writeDurable(attempt, asset.stage, data); err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, map[string]any{"source": asset.source, "stage": asset.stage, "sha256": asset.sha, "bytes": len(data)})
	}
	if err := writeDurable(attempt, "laptop-partition-diagnostic.json", manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(attempt, "laptop_partition_diagnostic.py", script); err != nil {
		t.Fatal(err)
	}
	proof, err := json.MarshalIndent(map[string]any{"manifest_sha256": laptopStudyManifestSHA, "script_sha256": laptopStudyScriptSHA, "assets": bindings}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "source-bindings.json", append(proof, '\n')); err != nil {
		t.Fatal(err)
	}
	return laptopStudyManifestSHA, laptopStudyScriptSHA
}

func verifyLaptopStudy(dir, manifestSHA, scriptSHA string) error {
	receiptPath := filepath.Join(dir, "outputs/diagnostic-receipt.json")
	data, err := readLaptopRegular(receiptPath, 1<<20)
	if err != nil {
		return fmt.Errorf("laptop receipt: %w", err)
	}
	var receipt laptopStudyReceipt
	if err = json.Unmarshal(data, &receipt); err != nil {
		return err
	}
	if receipt.Profile != laptopStudyProfileID || receipt.SourceSHA != laptopStudySourceSHA || receipt.Source != "/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/imported-laptop/normalized-macbook.glb" || receipt.InventorySHA != laptopStudyInventorySHA || receipt.ManifestSHA != manifestSHA || receipt.ScriptSHA != scriptSHA || receipt.Version != "5.2.2 LTS" || receipt.Triangles != 112502 || receipt.Components != 587 || !equalInts(receipt.Closure, []int{0, 30, 110}) {
		return errors.New("laptop receipt does not match pinned source, profile, or census")
	}
	want := laptopStudyOutputNames()
	if len(receipt.Outputs) != len(want) {
		return errors.New("laptop receipt output set differs")
	}
	if len(receipt.Summary.ProjectionMargins) != len(want) {
		return errors.New("laptop receipt lacks projected-frame checks")
	}
	var aggregate int64
	for _, name := range want {
		out, ok := receipt.Outputs[name]
		if !ok || out.Bytes <= 0 || out.Bytes > 8<<20 || len(out.SHA) != 64 {
			return fmt.Errorf("laptop receipt has invalid output facts for %s", name)
		}
		aggregate += out.Bytes
		if aggregate > 104<<20 {
			return errors.New("laptop PNG aggregate exceeds 104 MiB")
		}
		if err := verifyLaptopPNG(filepath.Join(dir, "outputs", name), out); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	for _, key := range laptopProjectionCheckNames() {
		margin, ok := receipt.Summary.ProjectionMargins[key]
		if !ok || margin < 0.04 || margin > 1 {
			return fmt.Errorf("laptop projection check is absent or below the fixed margin: %s", key)
		}
	}
	if err := verifyLaptopOutputDirectory(filepath.Join(dir, "outputs"), want); err != nil {
		return err
	}
	return nil
}

func laptopProjectionCheckNames() []string {
	names := []string{"source-authored-open-side"}
	for _, ownership := range []string{"reference_split", "bar_only"} {
		for _, angle := range []int{0, 30, 110} {
			for _, view := range []string{"side", "rear"} {
				names = append(names, fmt.Sprintf("%s-%03d-%s", ownership, angle, view))
			}
		}
	}
	return names
}

func laptopStudyOutputNames() []string {
	names := []string{"source-authored-open.png"}
	for _, ownership := range []string{"reference_split", "bar_only"} {
		for _, angle := range []int{0, 30, 110} {
			for _, view := range []string{"side", "rear"} {
				names = append(names, fmt.Sprintf("%s-closure-%03d-%s.png", ownership, angle, view))
			}
		}
	}
	return names
}

func verifyLaptopPNG(path string, output laptopStudyOutput) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != output.Bytes || info.Size() <= 0 || info.Size() > 8<<20 {
		return errors.New("PNG missing, unsafe, or outside fixed size bound")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, (8<<20)+1)); err != nil || hex.EncodeToString(h.Sum(nil)) != output.SHA {
		return errors.New("PNG digest mismatch")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	cfg, err := png.DecodeConfig(io.LimitReader(f, (8<<20)+1))
	if err != nil || cfg.Width != 960 || cfg.Height != 540 {
		return errors.New("PNG dimensions or header invalid")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	img, err := png.Decode(io.LimitReader(f, (8<<20)+1))
	if err != nil || img.Bounds().Dx() != 960 || img.Bounds().Dy() != 540 {
		return errors.New("PNG full decode failed")
	}
	return nil
}

func verifyLaptopOutputDirectory(dir string, expected []string) error {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("laptop output path is not a regular directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != len(expected)+1 {
		return errors.New("laptop output directory has missing or unexpected files")
	}
	allowed := map[string]bool{"diagnostic-receipt.json": true}
	for _, name := range expected {
		allowed[name] = true
	}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if !allowed[entry.Name()] || infoErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("laptop output directory contains an unapproved entry")
		}
	}
	return nil
}

func readLaptopRegular(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > max {
		return nil, errors.New("file missing or unsafe")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) != info.Size() || int64(len(b)) > max {
		return nil, errors.New("bounded file read failed")
	}
	return b, nil
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
