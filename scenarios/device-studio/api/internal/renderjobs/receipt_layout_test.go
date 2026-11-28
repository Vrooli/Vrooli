package renderjobs

import (
	"encoding/json"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
)

func TestDecodeClosurePinsPhoneV3ProfileAndRejectsMissingContrastInk(t *testing.T) {
	source := image.NewNRGBA(image.Rect(3, 7, 13, 29))
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 20, G: 26, B: 37, A: 255})
		}
	}
	config := presentations.Config{Target: image.Pt(400, 800), Phone: &presentations.PhoneConfig{
		ProfileVersion: presentations.PhoneProfileVersionAuto, Appearance: presentations.AppearanceAuto,
		ShowStatus: true, ShowHome: true, Clock: "9:41",
	}}
	layout, err := presentations.BuildLayout(source, config)
	if err != nil {
		t.Fatal(err)
	}
	closure := immutableClosure{
		Version: closureVersion, ProjectID: "project", ShotID: "shot", Revision: 2,
		RecipeSHA256: strings.Repeat("a", 64), SceneSHA256: strings.Repeat("b", 64), PresentationSHA256: strings.Repeat("c", 64),
		SceneVersion: "device-studio-scene-v1", PresentationVersion: "device-studio-presentation-v3",
		Profile: presentations.SimulatedCompositorProfile, RendererVersion: presentations.SimulatedCompositorProfile,
		OutputProfile: "png-v1", WorkerKey: "sha256:" + strings.Repeat("d", 64), WorkerSHA256: strings.Repeat("d", 64),
		SurfaceWidth: 400, SurfaceHeight: 800, Presentation: &config, PresentationLayout: &layout,
	}
	for _, role := range []string{"plate", "screen", "active_mask"} {
		digest := digestBytes([]byte(role))
		closure.Inputs = append(closure.Inputs, closureInput{Role: role, MediaID: "media-" + role, MediaSHA: digest, MediaSize: 1, BlobKey: inputBlobPrefix + digest + "/" + role, Name: role + ".bin"})
	}
	marshal := func(c immutableClosure) []byte {
		t.Helper()
		data, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		return append(data, '\n')
	}
	decoded, err := decodeClosure(marshal(closure))
	if err != nil {
		t.Fatalf("valid phone v3 closure rejected: %v", err)
	}
	if decoded.PresentationVersion != "device-studio-presentation-v3" || decoded.PresentationLayout.ProfileVersion != presentations.PhoneProfileVersionAuto {
		t.Fatalf("phone v3 closure identity was not preserved: %+v", decoded)
	}
	badLayout := *closure.PresentationLayout
	badLayout.ResolvedStatusInk = nil
	closure.PresentationLayout = &badLayout
	if _, err := decodeClosure(marshal(closure)); err == nil {
		t.Fatal("phone v3 closure without resolved status ink was accepted")
	}
}

func TestPresentationReceiptMustMatchPinnedLayoutAndFont(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 20; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 40, G: 60, B: 80, A: 255})
		}
	}
	config := presentations.Config{Target: image.Pt(400, 800), Phone: &presentations.PhoneConfig{ShowStatus: true, ShowHome: true}}
	layout, err := presentations.BuildLayout(source, config)
	if err != nil {
		t.Fatal(err)
	}
	manifest := rendering.Manifest{
		Version: 2, JobID: "job-layout", AttemptID: "attempt-layout", InputFingerprint: strings.Repeat("a", 64),
		Profile:      presentations.SimulatedCompositorProfile,
		Plate:        rendering.ImageBinding{Name: "plate.image", SHA256: strings.Repeat("1", 64), SizeBytes: 1},
		Screen:       rendering.ImageBinding{Name: "screen.image", SHA256: strings.Repeat("2", 64), SizeBytes: 1},
		ActiveMask:   rendering.ImageBinding{Name: "active-mask.png", SHA256: strings.Repeat("3", 64), SizeBytes: 1},
		SurfaceWidth: 400, SurfaceHeight: 800, Presentation: &config, PresentationLayout: &layout,
	}
	completed := time.Now().UTC()
	receipt := rendering.Receipt{
		Version: 2, JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint,
		ManifestSHA256: strings.Repeat("4", 64), Profile: manifest.Profile, Algorithm: "test",
		Inputs: []rendering.InputReceipt{
			{Role: "plate", Name: manifest.Plate.Name, SHA256: manifest.Plate.SHA256, SizeBytes: manifest.Plate.SizeBytes},
			{Role: "screen", Name: manifest.Screen.Name, SHA256: manifest.Screen.SHA256, SizeBytes: manifest.Screen.SizeBytes},
			{Role: "active_mask", Name: manifest.ActiveMask.Name, SHA256: manifest.ActiveMask.SHA256, SizeBytes: manifest.ActiveMask.SizeBytes},
		},
		OutputName: rendering.OutputName, OutputSHA256: strings.Repeat("5", 64), OutputSizeBytes: 10, OutputWidth: 400, OutputHeight: 800,
		Composition:  composition.Receipt{Algorithm: "test", PlateWidth: 400, PlateHeight: 800, ProtectedPixelsSHA256: strings.Repeat("6", 64)},
		Presentation: &rendering.PresentationReceipt{SemanticVersion: "device-studio-presentation-v2", Config: config, Layout: layout, DisplayRasterSHA256: strings.Repeat("7", 64)},
		CompletedAt:  completed,
	}
	job := jobs.Job{ID: manifest.JobID, InputFingerprint: manifest.InputFingerprint}
	attempt := jobs.Attempt{ID: manifest.AttemptID}
	if err := validateReceipt(receipt, manifest, receipt.ManifestSHA256, job, attempt); err != nil {
		t.Fatalf("matching pinned layout receipt rejected: %v", err)
	}

	tampered := receipt
	tampered.Presentation = &rendering.PresentationReceipt{SemanticVersion: receipt.Presentation.SemanticVersion, Config: config, Layout: layout, DisplayRasterSHA256: receipt.Presentation.DisplayRasterSHA256}
	tampered.Presentation.Layout.FontSHA256 = strings.Repeat("8", 64)
	if err := validateReceipt(tampered, manifest, receipt.ManifestSHA256, job, attempt); err == nil {
		t.Fatal("receipt with a different pinned font identity was accepted")
	}
}
