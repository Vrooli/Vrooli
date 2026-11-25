package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"image-tools/internal/models"
	"image-tools/internal/posemap"
)

// makePNG renders a small solid-color PNG for probe tests.
func makePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// TestProbe_StructuredResult is the IMG-P0-004 probe unit: the pure-Go probe
// returns structured info with no model or GPU.
func TestProbe_StructuredResult(t *testing.T) {
	src := makePNG(t, 1500, 1000, color.RGBA{R: 200, G: 30, B: 40, A: 255})
	res, err := Probe(src)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if res.Width != 1500 || res.Height != 1000 {
		t.Errorf("dimensions = %dx%d, want 1500x1000", res.Width, res.Height)
	}
	if res.Format != "png" {
		t.Errorf("format = %q, want png", res.Format)
	}
	if res.ColorModel != "rgba" {
		t.Errorf("color model = %q, want rgba", res.ColorModel)
	}
	if res.Megapixels <= 0 {
		t.Errorf("megapixels = %v, want > 0", res.Megapixels)
	}
	if res.SizeBytes != int64(len(src)) {
		t.Errorf("size = %d, want %d", res.SizeBytes, len(src))
	}
	if len(res.DominantColors) == 0 {
		t.Fatal("expected at least one dominant color")
	}
	if res.DominantColors[0].Hex == "" {
		t.Error("dominant color hex is empty")
	}
}

// TestOCR_StructuredResult drives OCR with a fake tesseract returning text.
func TestOCR_StructuredResult(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "/usr/bin/tesseract", nil },
		Run: func(_ context.Context, name string, args []string) ([]byte, error) {
			if name != "tesseract" {
				t.Errorf("ran %q, want tesseract", name)
			}
			return []byte("hello world\n"), nil
		},
	})
	res, err := svc.OCR(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("ocr: %v", err)
	}
	if res.FullText != "hello world" {
		t.Errorf("full text = %q, want %q", res.FullText, "hello world")
	}
	if res.Language != "eng" {
		t.Errorf("language = %q, want eng", res.Language)
	}
}

// TestOCR_BackendUnavailable refuses cleanly when tesseract is absent.
func TestOCR_BackendUnavailable(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "", errors.New("not found") },
	})
	_, err := svc.OCR(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err = %v, want ErrBackendUnavailable", err)
	}
}

// TestNSFW_StructuredResult drives NSFW with a fake onnx sidecar returning JSON.
func TestNSFW_StructuredResult(t *testing.T) {
	const isolatedPython = "/runtime/pyenv/bin/python"
	var gotProgram string
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		NSFWThreshold:  0.5,
		PythonProgram:  isolatedPython,
		LookPath: func(program string) (string, error) {
			if program != isolatedPython {
				t.Fatalf("looked up %q, want isolated interpreter %q", program, isolatedPython)
			}
			return program, nil
		},
		Run: func(_ context.Context, program string, _ []string) ([]byte, error) {
			gotProgram = program
			return []byte(`{"score":0.92,"categories":[{"label":"nsfw","score":0.92},{"label":"sfw","score":0.08}]}`), nil
		},
	})
	res, err := svc.NSFW(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("nsfw: %v", err)
	}
	if gotProgram != isolatedPython {
		t.Fatalf("ran %q, want isolated interpreter %q", gotProgram, isolatedPython)
	}
	if !res.NSFW {
		t.Errorf("nsfw = false, want true (score %.2f >= %.2f)", res.Score, res.Threshold)
	}
	if res.Label != "nsfw" {
		t.Errorf("label = %q, want nsfw", res.Label)
	}
	if len(res.Categories) != 2 {
		t.Errorf("categories = %d, want 2", len(res.Categories))
	}
}

func TestNSFW_HoldsModelUseLeaseThroughSidecar(t *testing.T) {
	gate := models.NewModelUseGate(t.TempDir())
	const modelID = "adamcodd-vit-nsfw"
	svc := mustService(t, Config{
		ModelInstalled:  func(string) bool { return true },
		AcquireModelUse: gate.Acquire,
		PythonProgram:   "/runtime/pyenv/bin/python",
		LookPath:        func(program string) (string, error) { return program, nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			if _, err := gate.AcquireExclusive(modelID); !errors.Is(err, models.ErrModelInUse) {
				t.Errorf("sidecar should hold shared model lease; exclusive acquire err=%v", err)
			}
			return []byte(`{"score":0.1,"categories":[]}`), nil
		},
	})
	if _, err := svc.NSFW(context.Background(), []byte("img")); err != nil {
		t.Fatalf("nsfw: %v", err)
	}
	release, err := gate.AcquireExclusive(modelID)
	if err != nil {
		t.Fatalf("model lease remained after sidecar: %v", err)
	}
	release()
}

// TestNSFW_EnsembleReportsTheDecidingClassifier keeps the analysis surface
// consistent: when the high-recall classifier is the one that crossed, the
// displayed score/threshold are that classifier's, not a precise score that
// looks below its own bar.
func TestNSFW_EnsembleReportsTheDecidingClassifier(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		PythonProgram:  "/runtime/pyenv/bin/python",
		LookPath:       func(program string) (string, error) { return program, nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{
				"score":0.9,"recall_score":0.9,"scores":{"falconsai-nsfw":0.2,"adamcodd-vit-nsfw":0.9},
				"sexual":true,"threshold":0.8,
				"thresholds":{"falconsai-nsfw":0.3,"adamcodd-vit-nsfw":0.8,"age":0.2},
				"reason_code":"sexual_adult","categories":[]
			}`), nil
		},
	})
	res, err := svc.NSFW(context.Background(), []byte("img"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.NSFW || res.Score != 0.9 || res.Threshold != 0.8 {
		t.Fatalf("verdict = %+v, want the recall classifier's 0.9/0.8", res)
	}
	if res.ReasonCode != "sexual_adult" || res.RecallScore != 0.9 {
		t.Fatalf("reason/recall = %q/%v", res.ReasonCode, res.RecallScore)
	}
}

// TestNSFW_ThresholdGovernsVerdict proves the operator-tunable threshold.
func TestNSFW_ThresholdGovernsVerdict(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		NSFWThreshold:  0.95, // strict: 0.92 should read as SFW
		PythonProgram:  "/usr/bin/python3",
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{"score":0.92,"categories":[]}`), nil
		},
	})
	res, err := svc.NSFW(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("nsfw: %v", err)
	}
	if res.NSFW {
		t.Error("expected SFW at a 0.95 threshold with score 0.92")
	}
}

// TestNSFW_ModelNotInstalled refuses when the model weights are absent.
func TestNSFW_ModelNotInstalled(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return false },
		PythonProgram:  "/usr/bin/python3",
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
	})
	_, err := svc.NSFW(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err = %v, want ErrBackendUnavailable", err)
	}
}

func TestNSFW_DoesNotFallbackToHostPython(t *testing.T) {
	lookPathCalls := 0
	runCalls := 0
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		// Even if host python3 resolves, an absent private venv must fail closed.
		LookPath: func(string) (string, error) {
			lookPathCalls++
			return "/usr/bin/python3", nil
		},
		Run: func(context.Context, string, []string) ([]byte, error) {
			runCalls++
			return []byte(`{"score":0}`), nil
		},
	})
	_, err := svc.NSFW(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err = %v, want ErrBackendUnavailable", err)
	}
	if lookPathCalls != 0 || runCalls != 0 {
		t.Fatalf("host fallback was attempted: lookPath calls=%d run calls=%d", lookPathCalls, runCalls)
	}
}

func TestNSFW_PreservesSidecarStderrOnFailure(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		PythonProgram:  os.Args[0],
		LookPath:       func(string) (string, error) { return os.Args[0], nil },
		Run: func(ctx context.Context, _ string, args []string) ([]byte, error) {
			childArgs := append([]string{"-test.run=TestCommandFailureHelper", "--"}, args...)
			cmd := exec.CommandContext(ctx, os.Args[0], childArgs...)
			cmd.Env = append(os.Environ(), "IMAGE_TOOLS_COMMAND_FAILURE_HELPER=1")
			return cmd.Output()
		},
	})
	_, err := svc.NSFW(context.Background(), []byte("img"))
	if err == nil {
		t.Fatal("nsfw returned nil error, want sidecar failure")
	}
	if got := err.Error(); !strings.Contains(got, "session initialization detail") {
		t.Fatalf("error = %q, want captured sidecar stderr", got)
	}
}

func TestCommandFailureHelper(t *testing.T) {
	if os.Getenv("IMAGE_TOOLS_COMMAND_FAILURE_HELPER") != "1" {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, "session initialization detail")
	os.Exit(5)
}

// poseDocument is a pose_extract result for a 400x800 image with one figure.
func poseDocument(t *testing.T) []byte {
	t.Helper()
	var doc posemap.Document
	doc.Schema, doc.KeypointFloor = posemap.Schema, 0.3
	doc.Image.Width, doc.Image.Height = 400, 800
	person := posemap.Person{Body: make([]posemap.Keypoint, posemap.BodyKeypoints), LeftHand: make([]posemap.Keypoint, posemap.HandKeypoints), RightHand: make([]posemap.Keypoint, posemap.HandKeypoints)}
	for i := range person.Body {
		person.Body[i] = posemap.Keypoint{X: 100 + float64(i)*10, Y: 100 + float64(i)*20, Confidence: 0.9}
	}
	doc.People = []posemap.Person{person}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestPoseMapReadsThePoseOnTheSidecarAndDrawsItOnTheRequestedCanvas: the pose
// operation's own sidecar module reads the source on the CPU (the generator holds
// the GPU), the region and preparation arrive as its params, and the document it
// writes comes out as an OpenPose PNG of the requested size.
func TestPoseMapReadsThePoseOnTheSidecarAndDrawsItOnTheRequestedCanvas(t *testing.T) {
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "source.png")
	outputPath := filepath.Join(tmp, "pose.png")
	if err := os.WriteFile(inputPath, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotArgs []string
	svc := mustService(t, Config{
		ModelInstalled: func(id string) bool { return id == posemap.ModelID },
		PythonProgram:  "/managed/pyenv/bin/python",
		ModelsRoot:     "/data",
		LookPath:       func(path string) (string, error) { return path, nil },
		Run: func(_ context.Context, name string, args []string) ([]byte, error) {
			gotArgs = args
			var out string
			for i, a := range args {
				if a == "--out" {
					out = args[i+1]
				}
			}
			return nil, os.WriteFile(out, poseDocument(t), 0o600)
		},
	})
	box := [4]float64{50, 60, 350, 760}
	req := posemap.Request{Source: inputPath, Output: outputPath, Width: 96, Height: 160, Box: &box, Prepare: "line_art"}
	if err := svc.PoseMap(context.Background(), req); err != nil {
		t.Fatalf("PoseMap: %v", err)
	}
	args := strings.Join(gotArgs, " ")
	for _, want := range []string{"-m image_tools_sidecar.pose_extract", "--model " + filepath.Join("/data", "models", posemap.ModelID), "--image " + inputPath, `"box":[50,60,350,760]`, `"prepare":"line_art"`} {
		if !strings.Contains(args, want) {
			t.Errorf("sidecar args %q lack %q", args, want)
		}
	}
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode pose map: %v", err)
	}
	if got := img.Bounds(); got.Dx() != 96 || got.Dy() != 160 {
		t.Fatalf("pose bounds = %v, want 96x160", got)
	}
	if _, err := os.Stat(outputPath + ".pose.json"); !os.IsNotExist(err) {
		t.Errorf("the pose document was left beside the output: %v", err)
	}
}

func TestPoseMapRefusesWhatItCannotRead(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.png")
	if err := os.WriteFile(source, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := `{"schema":"pose-v1","image":{"width":10,"height":10},"keypoint_floor":0.3,"people":[]}`
	cases := map[string]struct {
		cfg  Config
		want error
	}{
		"no runtime":    {Config{ModelInstalled: func(string) bool { return true }}, ErrBackendUnavailable},
		"model missing": {Config{ModelInstalled: func(string) bool { return false }, PythonProgram: "/py", LookPath: func(p string) (string, error) { return p, nil }}, ErrBackendUnavailable},
		"nobody in view": {Config{ModelInstalled: func(string) bool { return true }, PythonProgram: "/py", LookPath: func(p string) (string, error) { return p, nil }, Run: func(_ context.Context, _ string, args []string) ([]byte, error) {
			return nil, os.WriteFile(args[slices.Index(args, "--out")+1], []byte(empty), 0o600)
		}}, ErrNoPoseDetected},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := mustService(t, c.cfg).PoseMap(context.Background(), posemap.Request{Source: source, Output: filepath.Join(tmp, "out.png"), Width: 64, Height: 64})
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func mustService(t *testing.T, cfg Config) *Service {
	t.Helper()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

// ocrPaddleModel is the selected model for the detector + recogniser path.
var ocrPaddleModel = models.Model{ID: "paddleocr", Backend: "onnxruntime"}

// TestOCR_RunsTheModelTheCatalogSelects: with PaddleOCR selected, OCR runs the
// sidecar's ocr module over the model directory and decodes its lines with
// confidence and boxes; Tesseract is not touched.
func TestOCR_RunsTheModelTheCatalogSelects(t *testing.T) {
	const isolatedPython = "/runtime/pyenv/bin/python"
	var gotArgs []string
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		PythonProgram:  isolatedPython,
		ModelsRoot:     "/data",
		SelectModel:    func(context.Context, string) (models.Model, error) { return ocrPaddleModel, nil },
		LookPath:       func(program string) (string, error) { return program, nil },
		Run: func(_ context.Context, program string, args []string) ([]byte, error) {
			if program != isolatedPython {
				t.Fatalf("ran %q, want the isolated interpreter", program)
			}
			gotArgs = args
			out := args[indexOf(t, args, "--out")+1]
			payload := `{"full_text":"HARDWARE AND METAL\nYour stock","language":"english","blocks":[{"text":"HARDWARE AND METAL","confidence":0.99,"box":{"x":10,"y":20,"width":300,"height":40}},{"text":"Your stock","confidence":0.97,"box":{"x":12,"y":90,"width":200,"height":60}}]}`
			return nil, os.WriteFile(out, []byte(payload), 0o600)
		},
	})
	res, err := svc.OCR(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("ocr: %v", err)
	}
	if args := strings.Join(gotArgs, " "); !strings.Contains(args, "-m image_tools_sidecar.ocr") || !strings.Contains(args, filepath.Join("/data", "models", "paddleocr")) {
		t.Fatalf("sidecar args = %s", args)
	}
	if res.FullText != "HARDWARE AND METAL\nYour stock" || res.Language != "english" || len(res.Blocks) != 2 {
		t.Fatalf("result = %+v", res)
	}
	if b := res.Blocks[1]; b.Text != "Your stock" || b.Confidence != 0.97 || b.Box != (Box{X: 12, Y: 90, Width: 200, Height: 60}) {
		t.Fatalf("second block = %+v", b)
	}
}

func TestOCR_ReportsAModelThatIsNotInstalled(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return false },
		PythonProgram:  "/runtime/pyenv/bin/python",
		SelectModel:    func(context.Context, string) (models.Model, error) { return ocrPaddleModel, nil },
		LookPath:       func(program string) (string, error) { return program, nil },
	})
	_, err := svc.OCR(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) || !strings.Contains(err.Error(), "models install paddleocr") {
		t.Fatalf("err = %v, want an unavailable error naming `models install paddleocr`", err)
	}
}

// TestOperationsReportTheLiveOCRDefault: discovery shows the model a run would
// use, so `analyze list` cannot disagree with `analyze ocr` (the ai list bug).
func TestOperationsReportTheLiveOCRDefault(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		SelectModel:    func(context.Context, string) (models.Model, error) { return ocrPaddleModel, nil },
	})
	defaults := map[string]string{}
	for _, op := range svc.Operations(context.Background()) {
		defaults[op.Name] = op.DefaultModelID
	}
	if defaults[OpOCR] != "paddleocr" || defaults[OpNSFW] != "falconsai-nsfw" {
		t.Fatalf("defaults = %v", defaults)
	}
	failing := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		SelectModel:    func(context.Context, string) (models.Model, error) { return models.Model{}, errors.New("none") },
	})
	for _, op := range failing.Operations(context.Background()) {
		if op.Name == OpOCR && op.DefaultModelID != "" {
			t.Fatalf("an unselectable ocr op reports default %q", op.DefaultModelID)
		}
	}
}

func indexOf(t *testing.T, args []string, flag string) int {
	t.Helper()
	for i, a := range args {
		if a == flag {
			return i
		}
	}
	t.Fatalf("no %s in %v", flag, args)
	return -1
}

// TestDefaultRunnerExposesVenvNvidiaLibraries pins the sidecar loader contract:
// the default runner puts the interpreter's venv NVIDIA library directories on
// LD_LIBRARY_PATH (CUDA wheels load them at import time even for CPU work) and
// keeps the run on the CPU with the model containment overlay applied.
func TestDefaultRunnerExposesVenvNvidiaLibraries(t *testing.T) {
	venv := t.TempDir()
	libDir := filepath.Join(venv, "lib", "python3.12", "site-packages", "nvidia", "cuda_runtime", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(venv, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(bin, "python")
	script := "#!/bin/sh\necho \"$LD_LIBRARY_PATH|$CUDA_VISIBLE_DEVICES|$HF_HUB_OFFLINE\"\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := (&Service{}).resolveRun()(context.Background(), fake, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) != 3 {
		t.Fatalf("unexpected output %q", out)
	}
	if !strings.HasPrefix(parts[0], libDir) {
		t.Errorf("LD_LIBRARY_PATH %q does not start with the venv NVIDIA dir %q", parts[0], libDir)
	}
	if parts[1] != "" {
		t.Errorf("CUDA_VISIBLE_DEVICES = %q, want empty (CPU run)", parts[1])
	}
	if parts[2] != "1" {
		t.Errorf("HF_HUB_OFFLINE = %q, want 1 (containment overlay)", parts[2])
	}
}

// TestRunnerThreadsMatchTheAdmittedCPUReservation pins that the process starts
// with exactly the threads admission reserved: OCR reserves four cores and must
// use four (one thread made a 12 s read take 52 s), and an unadmitted call or a
// light op stays on one.
func TestRunnerThreadsMatchTheAdmittedCPUReservation(t *testing.T) {
	bin := t.TempDir()
	fake := filepath.Join(bin, "python")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"$IMAGE_TOOLS_CPU_THREADS|$OMP_NUM_THREADS\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := (&Service{}).resolveRun()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"unadmitted", context.Background(), "1|1"},
		{"ocr", context.WithValue(context.Background(), admittedThreads{}, cpuThreadsFor(OpOCR)), "4|4"},
		{"light op", context.WithValue(context.Background(), admittedThreads{}, cpuThreadsFor(OpNSFW)), "1|1"},
	} {
		out, err := run(tc.ctx, fake, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := strings.TrimSpace(string(out)); got != tc.want {
			t.Errorf("%s: threads = %q, want %q", tc.name, got, tc.want)
		}
	}
	if cpuThreadsFor(OpOCR) < 2 {
		t.Errorf("OCR must reserve more than one core, got %d", cpuThreadsFor(OpOCR))
	}
}

// TestEveryAnalysisPythonLaunchPathExposesVenvNvidiaLibraries is the class guard
// for the libcudart exit-3 failure: the module probe and the backend runner start
// the same venv interpreter as the OCR runner and need the same loader path.
func TestEveryAnalysisPythonLaunchPathExposesVenvNvidiaLibraries(t *testing.T) {
	venv := t.TempDir()
	libDir := filepath.Join(venv, "lib", "python3.12", "site-packages", "nvidia", "cuda_runtime", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(venv, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(venv, "bin", "python")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"$LD_LIBRARY_PATH\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := defaultRunOutput(context.Background(), fake, nil)
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(out)), libDir) {
		t.Errorf("defaultRunOutput LD_LIBRARY_PATH = %q (err %v), want prefix %q", out, err, libDir)
	}
	failing := filepath.Join(venv, "bin", "python-probe")
	script := "#!/bin/sh\ncase \"$LD_LIBRARY_PATH\" in " + libDir + "*) exit 0;; *) echo missing libs; exit 3;; esac\n"
	if err := os.WriteFile(failing, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := defaultCheckPythonModules(context.Background(), failing, []string{"os"}); err != nil {
		t.Errorf("module probe did not get the venv NVIDIA libraries: %v", err)
	}
}
