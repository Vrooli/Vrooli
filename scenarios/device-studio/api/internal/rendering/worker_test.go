package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type workerFixture struct {
	dir      string
	manifest Manifest
	bytes    map[string][]byte
}

func newWorkerFixture(t *testing.T) workerFixture {
	t.Helper()
	dir := t.TempDir()
	plate := image.NewRGBA(image.Rect(0, 0, 8, 8))
	screen := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	active := image.NewGray(image.Rect(0, 0, 8, 8))
	hardware := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			plate.SetRGBA(x, y, color.RGBA{R: uint8(20 + x), G: uint8(40 + y), B: 80, A: 255})
			if x >= 2 && x < 6 && y >= 2 && y < 6 {
				active.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			screen.SetNRGBA(x, y, color.NRGBA{R: 220, G: uint8(30 + y), B: 10, A: 255})
		}
	}
	hardware.SetGray(3, 3, color.Gray{Y: 255})
	data := map[string][]byte{
		"plate.png":       encodePNG(t, plate),
		"screen.png":      encodePNG(t, screen),
		"active-mask.png": encodePNG(t, active),
		"hardware.png":    encodePNG(t, hardware),
	}
	for name, b := range data {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := Manifest{
		Version: manifestVersion, JobID: "job-17", AttemptID: "attempt-2", InputFingerprint: strings.Repeat("a", 64), Profile: CompositorProfile,
		Plate: bind("plate.png", data["plate.png"]), Screen: bind("screen.png", data["screen.png"]), ActiveMask: bind("active-mask.png", data["active-mask.png"]),
		HardwareMask: ptr(bind("hardware.png", data["hardware.png"])),
		SurfaceWidth: 4, SurfaceHeight: 4,
		Quad: [4]Corner{{X: 2, Y: 2}, {X: 6, Y: 2}, {X: 6, Y: 6}, {X: 2, Y: 6}}, Samples: 2,
	}
	writeManifest(t, dir, manifest)
	return workerFixture{dir: dir, manifest: manifest, bytes: data}
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func bind(name string, data []byte) ImageBinding {
	return ImageBinding{Name: name, SHA256: digest(data), SizeBytes: int64(len(data))}
}

func ptr[T any](value T) *T { return &value }

func writeManifest(t *testing.T, dir string, manifest Manifest) []byte {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRunManifestComposesTypedInputsAndPublishesIdentityBoundPNGReceipt(t *testing.T) {
	fixture := newWorkerFixture(t)
	manifestBytes := writeManifest(t, fixture.dir, fixture.manifest)
	if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err != nil {
		t.Fatal(err)
	}
	outputBytes, err := os.ReadFile(filepath.Join(fixture.dir, OutputName))
	if err != nil {
		t.Fatal(err)
	}
	outputHash := digest(outputBytes)
	decoded, err := png.Decode(bytes.NewReader(outputBytes))
	if err != nil {
		t.Fatalf("published PNG did not fully decode: %v", err)
	}
	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 8 {
		t.Fatalf("output size = %v, want 8x8 plate dimensions", decoded.Bounds())
	}
	plate, err := png.Decode(bytes.NewReader(fixture.bytes["plate.png"]))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []image.Point{{0, 0}, {1, 4}, {7, 7}} {
		if !sameColor(decoded.At(p.X, p.Y), plate.At(p.X, p.Y)) {
			t.Fatalf("outside-active plate pixel %v changed", p)
		}
	}
	if !sameColor(decoded.At(3, 3), plate.At(3, 3)) {
		t.Fatal("fully occluded hardware pixel changed")
	}
	if sameColor(decoded.At(2, 2), plate.At(2, 2)) {
		t.Fatal("active screen pixel was not composed")
	}
	var receipt Receipt
	receiptBytes, err := os.ReadFile(filepath.Join(fixture.dir, ReceiptName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.JobID != fixture.manifest.JobID || receipt.AttemptID != fixture.manifest.AttemptID || receipt.InputFingerprint != fixture.manifest.InputFingerprint {
		t.Fatalf("receipt identity does not match manifest: %+v", receipt)
	}
	if receipt.ManifestSHA256 != digest(manifestBytes) || receipt.OutputSHA256 != outputHash || receipt.OutputSizeBytes != int64(len(outputBytes)) || receipt.OutputName != OutputName {
		t.Fatalf("receipt digest/size does not bind bytes: %+v", receipt)
	}
	if receipt.OutputWidth != 8 || receipt.OutputHeight != 8 || len(receipt.Inputs) != 4 || receipt.Composition.ProtectedChangedPixelCount != 0 {
		t.Fatalf("receipt omits composition verification: %+v", receipt)
	}
	for _, name := range []string{OutputName + ".tmp", ReceiptName + ".tmp"} {
		if _, err := os.Lstat(filepath.Join(fixture.dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("temporary publication file %q remains: %v", name, err)
		}
	}
}

func TestWorkerRejectsMalformedManifestCasesWithoutSuccessReceipt(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"version", func(m *Manifest) { m.Version++ }},
		{"fingerprint", func(m *Manifest) { m.InputFingerprint = "not-a-digest" }},
		{"profile", func(m *Manifest) { m.Profile = "ffmpeg-user-argv" }},
		{"path traversal", func(m *Manifest) { m.Plate.Name = "../plate.png" }},
		{"absolute path", func(m *Manifest) { m.Screen.Name = filepath.Join(string(filepath.Separator), "tmp", "screen.png") }},
		{"duplicate binding", func(m *Manifest) { m.ActiveMask.Name = m.Plate.Name }},
		{"oversized input declaration", func(m *Manifest) { m.Plate.SizeBytes = maxImageBytes + 1 }},
		{"digest mismatch", func(m *Manifest) { m.Plate.SHA256 = strings.Repeat("b", 64) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newWorkerFixture(t)
			tc.mutate(&fixture.manifest)
			writeManifest(t, fixture.dir, fixture.manifest)
			err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName))
			if err == nil {
				t.Fatal("malformed manifest was accepted")
			}
			assertNoSuccess(t, fixture.dir)
			assertFailureDiagnostic(t, fixture.dir)
		})
	}
}

func TestManifestValidationRejectsNonfiniteGeometry(t *testing.T) {
	fixture := newWorkerFixture(t)
	fixture.manifest.Quad[0].X = math.Inf(1)
	if err := validateManifest(fixture.manifest); err == nil {
		t.Fatal("non-finite geometry was accepted")
	}
}

func TestBindingAllowsOrdinaryNamesContainingXAndZero(t *testing.T) {
	for _, name := range []string{"texture0.png", "screen-x.png"} {
		if err := validateBinding(bind(name, []byte("fixture"))); err != nil {
			t.Fatalf("valid filename %q rejected: %v", name, err)
		}
	}
}

func TestWorkerMetadataProfileFailsClosedForTransformsItCannotApply(t *testing.T) {
	for _, marker := range []string{"iCCP", "gAMA", "cHRM", "cICP", "eXIf", "acTL"} {
		data := append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 4)...)
		data = append(data, marker...)
		data = append(data, make([]byte, 4)...)
		if err := validateInputMetadata(data, "png"); err == nil {
			t.Errorf("unsupported PNG %s metadata was accepted", marker)
		}
	}
}

func TestWorkerRejectsUnknownAndNoncanonicalManifestJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"unknown field", func(data []byte) []byte {
			return append(bytes.TrimSuffix(data, []byte("}")), []byte(`,"shell":"bad"}`)...)
		}},
		{"whitespace form", func(data []byte) []byte { return append([]byte(" "), data...) }},
		{"duplicate key", func(data []byte) []byte {
			return append(bytes.TrimSuffix(data, []byte("}")), []byte(`,"version":1}`)...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newWorkerFixture(t)
			data, err := os.ReadFile(filepath.Join(fixture.dir, ManifestName))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(fixture.dir, ManifestName), tc.edit(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
				t.Fatal("noncanonical/unknown manifest was accepted")
			}
			assertNoSuccess(t, fixture.dir)
		})
	}
}

func TestWorkerRejectsSymlinksAndChangedOrOversizedInput(t *testing.T) {
	t.Run("input symlink", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		outside := filepath.Join(t.TempDir(), "plate.png")
		if err := os.WriteFile(outside, fixture.bytes["plate.png"], 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(fixture.dir, fixture.manifest.Plate.Name)); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(fixture.dir, fixture.manifest.Plate.Name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
			t.Fatal("symlink input was accepted")
		}
		assertNoSuccess(t, fixture.dir)
	})
	t.Run("workspace symlink", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		link := filepath.Join(t.TempDir(), "workspace-link")
		if err := os.Symlink(fixture.dir, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := RunManifest(context.Background(), filepath.Join(link, ManifestName)); err == nil {
			t.Fatal("symlink workspace was accepted")
		}
	})
	t.Run("input changed after manifest binding", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		altered := append([]byte(nil), fixture.bytes[fixture.manifest.Screen.Name]...)
		altered[len(altered)-1] ^= 1
		if err := os.WriteFile(filepath.Join(fixture.dir, fixture.manifest.Screen.Name), altered, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
			t.Fatal("input with changed digest was accepted")
		}
		assertNoSuccess(t, fixture.dir)
	})
	t.Run("oversized manifest", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		if err := os.WriteFile(filepath.Join(fixture.dir, ManifestName), bytes.Repeat([]byte("x"), maxManifestBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
			t.Fatal("oversized manifest was accepted")
		}
		assertNoSuccess(t, fixture.dir)
	})
}

func TestWorkerCancellationAndPublicationFailuresNeverCreateSuccessReceipt(t *testing.T) {
	t.Run("cancel during composition", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		ctx := &cancelAfterChecks{cancelAt: 6}
		if err := RunManifest(ctx, filepath.Join(fixture.dir, ManifestName)); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation was not returned: %v", err)
		}
		assertNoSuccess(t, fixture.dir)
		assertFailureDiagnostic(t, fixture.dir)
	})
	t.Run("cancel after output but before receipt", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		ctx := outputAppearsContext{dir: fixture.dir}
		err := RunManifest(ctx, filepath.Join(fixture.dir, ManifestName))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("post-output cancellation was not returned: %v", err)
		}
		if _, err := os.Stat(filepath.Join(fixture.dir, OutputName)); err != nil {
			t.Fatalf("expected complete output for parent-side failure diagnosis: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(fixture.dir, ReceiptName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cancellation published success receipt: %v", err)
		}
		assertFailureDiagnostic(t, fixture.dir)
	})
	t.Run("existing output preserved", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		existing := []byte("preexisting bytes")
		if err := os.WriteFile(filepath.Join(fixture.dir, OutputName), existing, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
			t.Fatal("existing output was overwritten")
		}
		got, err := os.ReadFile(filepath.Join(fixture.dir, OutputName))
		if err != nil || !bytes.Equal(got, existing) {
			t.Fatalf("existing output changed: %q err=%v", got, err)
		}
		if _, err := os.Lstat(filepath.Join(fixture.dir, ReceiptName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failure produced success receipt: %v", err)
		}
	})
	t.Run("receipt publication conflict leaves no success receipt", func(t *testing.T) {
		fixture := newWorkerFixture(t)
		if err := os.WriteFile(filepath.Join(fixture.dir, ReceiptName+".tmp"), []byte("reserved"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunManifest(context.Background(), filepath.Join(fixture.dir, ManifestName)); err == nil {
			t.Fatal("receipt conflict was reported as success")
		}
		if _, err := os.Stat(filepath.Join(fixture.dir, OutputName)); err != nil {
			t.Fatalf("completed output was not retained for diagnosis: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(fixture.dir, ReceiptName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed worker published success receipt: %v", err)
		}
		assertFailureDiagnostic(t, fixture.dir)
	})
}

func TestFailureDiagnosticAlwaysFitsItsEnvelope(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	long := strings.Repeat("a", 40<<10)
	if err := writeFailure(root, Manifest{JobID: "job-1", AttemptID: "attempt-1"}, errors.New(long)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FailureName))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxDiagnosticBytes {
		t.Fatalf("diagnostic exceeds bound: %d", len(data))
	}
	var diagnostic failureDiagnostic
	if err := json.Unmarshal(data, &diagnostic); err != nil || diagnostic.Message == "" {
		t.Fatalf("bounded diagnostic invalid: %+v err=%v", diagnostic, err)
	}
}

// TestV6PhoneWorkerFixture opts into a retained end-to-end manifest replay.
// Original fixture files are read only; exact byte copies are staged under the
// authorized E2 worker evidence directory before the package worker runs.
func TestV6PhoneWorkerFixture(t *testing.T) {
	fixtureDir := os.Getenv("DEVICE_STUDIO_V6_PHONE_DIR")
	evidenceDir := os.Getenv("DEVICE_STUDIO_E2_WORKER_EVIDENCE_DIR")
	if fixtureDir == "" || evidenceDir == "" {
		t.Skip("set DEVICE_STUDIO_V6_PHONE_DIR and DEVICE_STUDIO_E2_WORKER_EVIDENCE_DIR to retain the v6 worker replay")
	}
	cleanEvidence := filepath.Clean(evidenceDir)
	if !strings.Contains(filepath.ToSlash(cleanEvidence), "/goal/evidence/E2/worker/v6-replay/") {
		t.Fatalf("output must be a unique run directory under goal/evidence/E2/worker/v6-replay, got %q", evidenceDir)
	}
	if err := os.MkdirAll(cleanEvidence, 0o700); err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		return data
	}
	var calibration struct {
		PlateSHA     string      `json:"plate_sha256"`
		PlateSize    []int       `json:"plate_size"`
		SourceCanvas []int       `json:"source_canvas"`
		Homography   [][]float64 `json:"homography_source_to_plate"`
	}
	if err := json.Unmarshal(read("calibration.json"), &calibration); err != nil {
		t.Fatal(err)
	}
	plateBytes := read("plate.png")
	if digest(plateBytes) != calibration.PlateSHA || len(calibration.PlateSize) != 2 || len(calibration.SourceCanvas) != 2 || len(calibration.Homography) != 3 {
		t.Fatal("v6 calibration is incomplete or plate digest mismatches")
	}
	inputs := map[string][]byte{
		"plate.png": plateBytes, "source-screen.png": read("source-screen.png"), "active-mask.png": read("active-mask.png"),
	}
	for name, data := range inputs {
		if err := os.WriteFile(filepath.Join(cleanEvidence, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	quad := [4]Corner{}
	boundary := [4]struct{ x, y float64 }{{0, 0}, {float64(calibration.SourceCanvas[0]), 0}, {float64(calibration.SourceCanvas[0]), float64(calibration.SourceCanvas[1])}, {0, float64(calibration.SourceCanvas[1])}}
	for i, point := range boundary {
		x, y, ok := projectFixtureHomography(calibration.Homography, point.x-0.5, point.y-0.5)
		if !ok {
			t.Fatalf("project calibrated corner %d", i)
		}
		quad[i] = Corner{X: x + 0.5, Y: y + 0.5}
	}
	manifest := Manifest{
		Version: manifestVersion, JobID: "v6-phone-worker-replay", AttemptID: "attempt-1", InputFingerprint: digest([]byte("read-only v6 phone plate + historical composed screen + active mask")), Profile: CompositorProfile,
		Plate: bind("plate.png", inputs["plate.png"]), Screen: bind("source-screen.png", inputs["source-screen.png"]), ActiveMask: bind("active-mask.png", inputs["active-mask.png"]),
		SurfaceWidth: calibration.SourceCanvas[0], SurfaceHeight: calibration.SourceCanvas[1], Quad: quad, Samples: 2,
	}
	writeManifest(t, cleanEvidence, manifest)
	if err := RunManifest(context.Background(), filepath.Join(cleanEvidence, ManifestName)); err != nil {
		t.Fatal(err)
	}
	var receipt Receipt
	receiptBytes, err := os.ReadFile(filepath.Join(cleanEvidence, ReceiptName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.JobID != manifest.JobID || receipt.AttemptID != manifest.AttemptID || receipt.InputFingerprint != manifest.InputFingerprint || receipt.OutputWidth != calibration.PlateSize[0] || receipt.OutputHeight != calibration.PlateSize[1] {
		t.Fatalf("worker receipt did not bind expected replay: %+v", receipt)
	}
	if len(receipt.Inputs) != 3 || receipt.Inputs[0].Role != "plate" || receipt.Inputs[1].Role != "screen" || receipt.Inputs[2].Role != "active_mask" {
		t.Fatalf("worker receipt input roles are incomplete: %+v", receipt.Inputs)
	}
	output, err := os.ReadFile(filepath.Join(cleanEvidence, OutputName))
	if err != nil || digest(output) != receipt.OutputSHA256 {
		t.Fatalf("worker output digest mismatch: %v", err)
	}
	report := map[string]any{
		"fixture":         "v6 phone; plate and active mask are original fixture bytes; source-screen.png is the historical precomposed source frame, not a raw app capture",
		"manifest_sha256": receipt.ManifestSHA256, "plate_sha256": digest(inputs["plate.png"]),
		"source_screen_sha256": digest(inputs["source-screen.png"]), "active_mask_sha256": digest(inputs["active-mask.png"]),
		"output_sha256": receipt.OutputSHA256, "output_width": receipt.OutputWidth, "output_height": receipt.OutputHeight,
		"receipt": ReceiptName, "output": OutputName,
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cleanEvidence, "README.json"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func projectFixtureHomography(matrix [][]float64, x, y float64) (float64, float64, bool) {
	if len(matrix) != 3 || len(matrix[0]) != 3 || len(matrix[1]) != 3 || len(matrix[2]) != 3 {
		return 0, 0, false
	}
	w := matrix[2][0]*x + matrix[2][1]*y + matrix[2][2]
	if math.Abs(w) < 1e-12 {
		return 0, 0, false
	}
	return (matrix[0][0]*x + matrix[0][1]*y + matrix[0][2]) / w, (matrix[1][0]*x + matrix[1][1]*y + matrix[1][2]) / w, true
}

type outputAppearsContext struct{ dir string }

func (c outputAppearsContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c outputAppearsContext) Done() <-chan struct{}       { return nil }
func (c outputAppearsContext) Value(any) any               { return nil }
func (c outputAppearsContext) Err() error {
	if _, err := os.Stat(filepath.Join(c.dir, OutputName)); err == nil {
		return context.Canceled
	}
	return nil
}

func TestWorkerCommandIsFixedAndFailClosed(t *testing.T) {
	if IsWorkerInvocation(nil) || IsWorkerInvocation([]string{"serve"}) || !IsWorkerInvocation([]string{"render-worker", "--bad"}) {
		t.Fatal("worker invocation selector did not reserve only the worker command")
	}
	for _, args := range [][]string{{"render-worker"}, {"render-worker", "--other", "x"}, {"render-worker", "--manifest", "relative.json", "extra"}} {
		if err := RunWorkerCommand(args); err == nil {
			t.Fatalf("invalid worker command was accepted: %q", args)
		}
	}
}

func assertNoSuccess(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{OutputName, ReceiptName} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failure left success artifact %q: %v", name, err)
		}
	}
}

func assertFailureDiagnostic(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, FailureName))
	if err != nil {
		t.Fatalf("bounded failure diagnostic missing: %v", err)
	}
	if len(data) > maxDiagnosticBytes {
		t.Fatalf("diagnostic exceeded bound: %d bytes", len(data))
	}
	var diagnostic failureDiagnostic
	if err := json.Unmarshal(data, &diagnostic); err != nil || diagnostic.Code != "render_failed" {
		t.Fatalf("invalid failure diagnostic: %+v err=%v", diagnostic, err)
	}
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

type cancelAfterChecks struct {
	cancelAt int
	checks   int
}

func (c *cancelAfterChecks) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterChecks) Done() <-chan struct{}       { return nil }
func (c *cancelAfterChecks) Value(any) any               { return nil }
func (c *cancelAfterChecks) Err() error {
	c.checks++
	if c.checks >= c.cancelAt {
		return context.Canceled
	}
	return nil
}
