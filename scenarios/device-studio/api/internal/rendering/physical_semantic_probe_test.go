package rendering

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

//go:embed testdata/physical_semantic_probe.py
var physicalSemanticProbe []byte

//go:embed testdata/physical_hinge_probe.glb
var physicalHingeProbeGLB []byte

const physicalHingeProbeGLBSHA256 = "36d6091d2aa782f1f06f7447a6078520c586bcc7c4f3a4f48f697cd9a2ffc512"

type physicalSemanticProbeManifest struct {
	Version       int    `json:"version"`
	AdapterName   string `json:"adapter_name"`
	AdapterSHA256 string `json:"adapter_sha256"`
	ProbeSHA256   string `json:"probe_sha256"`
	TextureName   string `json:"texture_name"`
	HingeGLBName  string `json:"hinge_glb_name"`
	HingeGLBSHA   string `json:"hinge_glb_sha256"`
	HingeGLBSize  int    `json:"hinge_glb_size_bytes"`
}

func TestPhysicalHingeProbeGLBIsPinnedParentedHierarchy(t *testing.T) {
	if len(physicalHingeProbeGLB) != 832 || blenderDigest(physicalHingeProbeGLB) != physicalHingeProbeGLBSHA256 {
		t.Fatal("embedded importer-backed hinge fixture differs from its reviewed identity")
	}
	inventory, err := InspectPhysicalGLB(physicalHingeProbeGLB)
	if err != nil {
		t.Fatalf("inspect importer-backed hinge fixture: %v", err)
	}
	if len(inventory.Nodes) != 2 || inventory.Nodes[0].Name != "DS_device_NODE_00000" || inventory.Nodes[1].Name != "DS_device_NODE_00001" || inventory.Nodes[1].Parent == nil || *inventory.Nodes[1].Parent != 0 || inventory.Nodes[1].Mesh == nil {
		t.Fatalf("fixture does not contain the exact parented mesh hierarchy: %+v", inventory.Nodes)
	}
}

// This evidence-only test uses the existing owned-operation runner. It is
// opt-in because it launches the pinned native Blender installation.
func TestOptInPhysicalBlenderSemanticProbe(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHYSICAL_SEMANTIC_PROBE") != "1" {
		t.Skip("opt-in Blender physical adapter semantic probe")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("pinned physical Blender profile supports Linux/amd64 only")
	}
	base := filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_PHYSICAL_SEMANTIC_EVIDENCE_DIR"))
	if !filepath.IsAbs(base) || !strings.Contains(filepath.ToSlash(base), "/goal/evidence/E3/physical-semantic-probe/") {
		t.Fatalf("evidence directory is outside the admitted goal evidence: %q", base)
	}
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "cpu")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	_, profile, err := ResolvePhysicalBlenderProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetBlender() == nil || profile.GetBlender().GetPath() == "" {
		t.Fatal("managed Blender profile has no verified executable")
	}
	adapter := PhysicalAdapterBytes()
	probe := physicalSemanticProbe
	adapterSHA, probeSHA := blenderDigest(adapter), blenderDigest(probe)
	if err := writeDurable(dir, PhysicalAdapterName, adapter); err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(dir, "physical_semantic_probe.py", probe); err != nil {
		t.Fatal(err)
	}
	if err := writeProbeTexture(dir); err != nil {
		t.Fatal(err)
	}
	if blenderDigest(physicalHingeProbeGLB) != physicalHingeProbeGLBSHA256 {
		t.Fatal("embedded importer-backed hinge fixture changed without a reviewed pin update")
	}
	if err := writeDurable(dir, "physical-hinge-probe.glb", physicalHingeProbeGLB); err != nil {
		t.Fatal(err)
	}
	manifest := physicalSemanticProbeManifest{
		Version: 1, AdapterName: PhysicalAdapterName, AdapterSHA256: adapterSHA,
		ProbeSHA256: probeSHA, TextureName: "probe-texture.png", HingeGLBName: "physical-hinge-probe.glb",
		HingeGLBSHA: physicalHingeProbeGLBSHA256, HingeGLBSize: len(physicalHingeProbeGLB),
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeDurable(dir, "physical-semantic-probe-manifest.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	runOwnedBlenderOperation(t, dir, profile.GetBlender().GetPath(), profile.GetArtifactIdentity(), "CPU", "physical-semantic-probe", "", blenderDigest(manifestBytes), adapterSHA, func() error {
		return verifyPhysicalSemanticProbe(dir, manifest, probeSHA)
	})
}

func writeProbeTexture(dir string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, G: 64, B: 32, A: 0})
	img.Set(1, 0, color.NRGBA{R: 128, G: 160, B: 192, A: 127})
	img.Set(0, 1, color.NRGBA{R: 32, G: 64, B: 96, A: 128})
	img.Set(1, 1, color.NRGBA{R: 220, G: 180, B: 140, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return err
	}
	return writeDurable(dir, "probe-texture.png", b.Bytes())
}

type physicalSemanticProbeReceipt struct {
	Version       int      `json:"version"`
	AdapterSHA256 string   `json:"adapter_sha256"`
	Status        string   `json:"status"`
	Checks        []string `json:"checks"`
}

func verifyPhysicalSemanticProbe(dir string, manifest physicalSemanticProbeManifest, probeSHA string) error {
	probeBytes, err := os.ReadFile(filepath.Join(dir, "physical_semantic_probe.py"))
	if err != nil || blenderDigest(probeBytes) != probeSHA || probeSHA != manifest.ProbeSHA256 {
		return errors.New("staged semantic probe differs from the immutable probe digest")
	}
	b, err := os.ReadFile(filepath.Join(dir, "physical-semantic-probe-receipt.json"))
	if err != nil || len(b) > 16<<10 {
		return fmt.Errorf("bounded semantic receipt: %v", err)
	}
	var receipt physicalSemanticProbeReceipt
	if err := json.Unmarshal(b, &receipt); err != nil {
		return err
	}
	fixture, err := os.ReadFile(filepath.Join(dir, manifest.HingeGLBName))
	if err != nil || len(fixture) != manifest.HingeGLBSize || blenderDigest(fixture) != manifest.HingeGLBSHA || manifest.HingeGLBSHA != physicalHingeProbeGLBSHA256 {
		return errors.New("staged importer-backed hinge GLB differs from the exact embedded fixture")
	}
	want := []string{"empty-factory-scene", "actual-parented-device-landmark", "absolute-hinge-pose-and-child-screen-landmark", "imported-glb-hinge-basis-and-child-landmark", "affine-uv-node-construction", "red-channel-roughness-and-bump", "normal-bump-chain", "grade-order", "mask-cutoff", "pinned-color-management", "canonical-constant-world", "blender-5-photographic-compositor-node-group", "logical-uv0-to-blender-uvmap", "pinned-cycles-look"}
	if receipt.Version != 1 || receipt.AdapterSHA256 != manifest.AdapterSHA256 || receipt.Status != "verified" || !equalStrings(receipt.Checks, want) {
		return errors.New("semantic probe receipt does not prove the pinned adapter checks")
	}
	return nil
}

func equalStrings(a, b []string) bool {
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
