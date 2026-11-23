package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image-tools/internal/modelenv"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/models"
	"image-tools/internal/posemap"
	"image-tools/internal/sidecar"
)

// Config configures a Service.
type Config struct {
	// ModelInstalled reports whether a model's weights are on disk. Required for
	// the model-backed ops (ocr/nsfw); probe never consults it.
	ModelInstalled func(modelID string) bool
	// AcquireModel makes a missing model's governed weights present before
	// admission takes any lease (nil keeps the "not installed" refusal).
	AcquireModel func(ctx context.Context, modelID, jobID string, emit func(progress int, message string)) error
	// AcquireModelUse holds shared ownership while NSFW inference reads model
	// files; production shares this gate with the Installer and AI engine.
	AcquireModelUse func(modelID string) (release func(), err error)
	// AcquireCapacity is the shared host admission owner. Production supplies
	// capacity.AcquireScoped; nil is reserved for isolated domain tests.
	AcquireCapacity func(context.Context, sharedcapacity.Request) (*sharedcapacity.Scope, error)
	ModelRAMBytes   func(string) (int64, error)
	ModelReady      func(context.Context, string) (bool, string)
	// ModelsRoot is the absolute directory model weights live under (per model:
	// <ModelsRoot>/models/<id>).
	ModelsRoot string
	// SelectModel is the catalog's selection for an operation (the same code
	// `models select` runs: effective enabled state, the operator's pin, rights and
	// host fit). Nil keeps the built-in defaults, tesseract for OCR.
	SelectModel func(ctx context.Context, operation string) (models.Model, error)
	// PythonProgram is the absolute interpreter path from image-tools' private
	// lock-pinned venv. An empty value leaves model-backed analysis unavailable;
	// NSFW must never fall back to the host's shared Python installation.
	PythonProgram string
	// NSFWThreshold is the decision threshold for the boolean NSFW verdict.
	// AdamCodd's classifier over-flags skin tones, so this is operator-tunable;
	// defaults to 0.5 when <= 0.
	NSFWThreshold float64
	// LookPath / Run are injected for testing; nil ⇒ real os/exec.
	LookPath lookPathFunc
	Run      cmdOutput
	Logger   *log.Logger
}

// Service runs the model-backed analysis ops (OCR, NSFW). Probe is a free
// function (no deps).
type Service struct {
	modelInstalled  func(string) bool
	acquireModel    func(context.Context, string, string, func(int, string)) error
	acquireCapacity func(context.Context, sharedcapacity.Request) (*sharedcapacity.Scope, error)
	modelRAMBytes   func(string) (int64, error)
	modelReady      func(context.Context, string) (bool, string)
	acquireModelUse func(string) (func(), error)
	modelsRoot      string
	selectModel     func(context.Context, string) (models.Model, error)
	pythonProgram   string
	threshold       float64
	lookPath        lookPathFunc
	run             cmdOutput
	logger          *log.Logger
}

// NewService validates cfg and returns the service.
func NewService(cfg Config) (*Service, error) {
	if cfg.ModelInstalled == nil {
		return nil, fmt.Errorf("analysis: ModelInstalled is required")
	}
	thr := cfg.NSFWThreshold
	if thr <= 0 {
		thr = 0.5
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		modelInstalled:  cfg.ModelInstalled,
		acquireModel:    cfg.AcquireModel,
		acquireCapacity: cfg.AcquireCapacity,
		modelRAMBytes:   cfg.ModelRAMBytes,
		modelReady:      cfg.ModelReady,
		acquireModelUse: cfg.AcquireModelUse,
		modelsRoot:      cfg.ModelsRoot,
		selectModel:     cfg.SelectModel,
		pythonProgram:   strings.TrimSpace(cfg.PythonProgram),
		threshold:       thr,
		lookPath:        cfg.LookPath,
		run:             cfg.Run,
		logger:          logger,
	}, nil
}

// Operations is the analysis catalog with each model-backed operation's default
// taken from the catalog's live selection (the model a run would use), not from
// the built-in table. An operation with no selectable model reports none.
func (s *Service) Operations(ctx context.Context) []OpInfo {
	ops := List()
	if s == nil || s.selectModel == nil {
		return ops
	}
	for i := range ops {
		if !ops[i].ModelBacked || ops[i].Name != OpOCR {
			continue
		}
		model, err := s.selectModel(ctx, ops[i].Name)
		if err != nil {
			ops[i].DefaultModelID = ""
			continue
		}
		ops[i].DefaultModelID = model.ID
	}
	return ops
}

func (s *Service) resolveLookPath() lookPathFunc {
	if s.lookPath != nil {
		return s.lookPath
	}
	return exec.LookPath
}

func (s *Service) resolveRun() cmdOutput {
	if s.run != nil {
		return s.run
	}
	return func(ctx context.Context, name string, args []string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		threads := 1
		if admitted, _ := ctx.Value(admittedThreads{}).(int); admitted > 0 {
			threads = admitted
		}
		count := strconv.Itoa(threads)
		env := append(os.Environ(), "CUDA_VISIBLE_DEVICES=", "OMP_NUM_THREADS="+count, "MKL_NUM_THREADS="+count, "OPENBLAS_NUM_THREADS="+count, "IMAGE_TOOLS_CPU_THREADS="+count)
		env = withRuntimeLibraries(env, name)
		cmd.Env = modelenv.Apply(env, modelenv.ModeInference)
		return cmd.Output()
	}
}

// withRuntimeLibraries puts the NVIDIA runtime libraries of the venv that owns
// python on LD_LIBRARY_PATH. CUDA-enabled wheels (onnxruntime-gpu) load them at
// import time even for a CPU run, and they are not on the service's ambient loader
// path; without them the sidecar exits 3 with "libcudart.so.13: cannot open shared
// object file". Every analysis process that starts a venv interpreter goes through
// this, so no launch path can forget it.
func withRuntimeLibraries(env []string, python string) []string {
	libraries := sidecar.RuntimeLibraries(python)
	if len(libraries) == 0 {
		return env
	}
	if current := strings.TrimSpace(os.Getenv("LD_LIBRARY_PATH")); current != "" {
		libraries = append(libraries, current)
	}
	return append(env, "LD_LIBRARY_PATH="+strings.Join(libraries, string(os.PathListSeparator)))
}

const maxCommandStderrBytes = 4096

var ErrNoPoseDetected = errors.New("analysis: no visible pose found in source image")

func commandError(prefix string, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if len(stderr) > maxCommandStderrBytes {
			stderr = stderr[:maxCommandStderrBytes] + "… [truncated]"
		}
		if stderr != "" {
			return fmt.Errorf("%s: %w; stderr: %s", prefix, err, stderr)
		}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// PoseMap derives an OpenPose bone image from a source picture: a photograph, a
// line-art mannequin, or one cell of a pose chart. The caller holds the pose
// model's use lease and the job's admission, so reading the pose stays inside the
// same whole-job admission as generation.
func (s *Service) PoseMap(ctx context.Context, req posemap.Request) error {
	if strings.TrimSpace(s.pythonProgram) == "" {
		return fmt.Errorf("%w: isolated Python environment is not provisioned; install uv and restart image-tools", ErrBackendUnavailable)
	}
	python, err := s.resolveLookPath()(s.pythonProgram)
	if err != nil {
		return fmt.Errorf("%w: isolated Python interpreter is unavailable at %q", ErrBackendUnavailable, s.pythonProgram)
	}
	if !s.modelInstalled(posemap.ModelID) {
		return fmt.Errorf("%w: model %q not installed — run `image-tools models install %s`", ErrBackendUnavailable, posemap.ModelID, posemap.ModelID)
	}
	info, err := os.Stat(req.Source)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: pose source image is unavailable at %q", ErrBackendUnavailable, req.Source)
	}
	options := map[string]any{}
	if req.Box != nil {
		options["box"] = req.Box
	}
	if req.Prepare != "" {
		options["prepare"] = req.Prepare
	}
	params, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("analysis: pose options: %w", err)
	}
	docPath := req.Output + ".pose.json"
	defer func() { _ = os.Remove(docPath) }()
	args := []string{"-m", "image_tools_sidecar.pose_extract", "--model", s.modelDir(posemap.ModelID), "--image", req.Source, "--out", docPath, "--params", string(params)}
	if _, err := s.resolveRun()(ctx, python, args); err != nil {
		return commandError("analysis: pose estimation", err)
	}
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("analysis: read pose document: %w", err)
	}
	doc, err := posemap.Parse(raw)
	if err != nil {
		return err
	}
	person, err := doc.Primary()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNoPoseDetected, err)
	}
	view := doc.Whole()
	if req.Box != nil {
		view = posemap.View{X0: req.Box[0], Y0: req.Box[1], X1: req.Box[2], Y1: req.Box[3]}
	}
	if view, err = posemap.Fit([4]float64{view.X0, view.Y0, view.X1, view.Y1}, float64(req.Width)/float64(req.Height)); err != nil {
		return err
	}
	file, err := os.OpenFile(req.Output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("analysis: create pose map: %w", err)
	}
	encodeErr := posemap.Render(file, doc, person, req.Width, req.Height, view)
	return errors.Join(encodeErr, file.Close())
}

// modelDir is the absolute installed-weights directory for a model id.
func (s *Service) modelDir(modelID string) string {
	if s.modelsRoot == "" {
		return "models/" + modelID
	}
	return filepath.Join(s.modelsRoot, "models", modelID)
}

// withTempInput writes src to a temp file and invokes fn with its path.
func withTempInput(src []byte, fn func(path string) error) error {
	dir, err := os.MkdirTemp("", "imgtools-analysis-*")
	if err != nil {
		return fmt.Errorf("analysis: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "input-"+uuid.NewString())
	if err := os.WriteFile(path, src, 0o600); err != nil {
		return fmt.Errorf("analysis: write temp input: %w", err)
	}
	return fn(path)
}

// OCR extracts text via tesseract. Refuses with ErrBackendUnavailable when the
// tesseract program is absent or the model is not installed.
func (s *Service) OCR(ctx context.Context, src []byte) (result OCRResult, runErr error) {
	ctx, finish, err := s.execution(ctx, OpOCR)
	if err != nil {
		return result, err
	}
	defer func() { runErr = errors.Join(runErr, finish()) }()
	model, err := s.ocrModel(ctx)
	if err != nil {
		return OCRResult{}, err
	}
	if model.Backend == models.BackendTesseract {
		return s.ocrTesseract(ctx, src)
	}
	return s.ocrONNX(ctx, model, src)
}

// ocrModel is the model this OCR run uses: the one admission acquired (so the
// model that was installed and leased is the model that runs), else the
// catalog's current selection.
func (s *Service) ocrModel(ctx context.Context) (models.Model, error) {
	if admitted, ok := ctx.Value(admittedModel{}).(models.Model); ok {
		return admitted, nil
	}
	return s.selectOperationModel(ctx, OpOCR)
}

// selectOperationModel asks the catalog which model serves operation, falling back
// to the built-in OCR default when no selector is wired (tests, read-only wiring).
func (s *Service) selectOperationModel(ctx context.Context, operation string) (models.Model, error) {
	if s.selectModel != nil {
		model, err := s.selectModel(ctx, operation)
		if err != nil {
			return models.Model{}, fmt.Errorf("%w: select %s model: %v", ErrBackendUnavailable, operation, err)
		}
		return model, nil
	}
	if operation == OpOCR {
		return models.Model{ID: "tesseract", Backend: models.BackendTesseract}, nil
	}
	return models.Model{}, fmt.Errorf("%w: no model selector for %s", ErrBackendUnavailable, operation)
}

func (s *Service) ocrTesseract(ctx context.Context, src []byte) (OCRResult, error) {
	if _, err := s.resolveLookPath()("tesseract"); err != nil {
		return OCRResult{}, fmt.Errorf("%w: install tesseract (the `ocr` backend of the selected model)", ErrBackendUnavailable)
	}
	var out []byte
	err := withTempInput(src, func(path string) error {
		args := ocrArgs(path)
		if s.modelsRoot != "" {
			args = append(args, "--tessdata-dir", s.modelDir("tesseract"))
		}
		b, err := s.resolveRun()(ctx, "tesseract", args)
		if err != nil {
			return fmt.Errorf("analysis: tesseract: %w", err)
		}
		out = b
		return nil
	})
	if err != nil {
		return OCRResult{}, err
	}
	return OCRResult{FullText: strings.TrimRight(string(out), "\n"), Language: "eng"}, nil
}

// ocrONNX runs a detector + recogniser model (PaddleOCR) through the sidecar's
// ocr module and decodes its structured result: text lines with confidence and
// pixel boxes in reading order.
func (s *Service) ocrONNX(ctx context.Context, model models.Model, src []byte) (OCRResult, error) {
	if s.pythonProgram == "" {
		return OCRResult{}, fmt.Errorf("%w: isolated Python environment is not provisioned; install uv and restart image-tools", ErrBackendUnavailable)
	}
	python, err := s.resolveLookPath()(s.pythonProgram)
	if err != nil {
		return OCRResult{}, fmt.Errorf("%w: isolated Python interpreter is unavailable at %q", ErrBackendUnavailable, s.pythonProgram)
	}
	if !s.modelInstalled(model.ID) {
		return OCRResult{}, fmt.Errorf("%w: model %q not installed — run `image-tools models install %s`", ErrBackendUnavailable, model.ID, model.ID)
	}
	var raw []byte
	err = withTempInput(src, func(path string) error {
		outPath := path + ".ocr.json"
		defer func() { _ = os.Remove(outPath) }()
		args := []string{"-m", "image_tools_sidecar.ocr", "--model", s.modelDir(model.ID), "--image", path, "--out", outPath}
		if _, err := s.resolveRun()(ctx, python, args); err != nil {
			return commandError("analysis: onnx ocr", err)
		}
		b, readErr := os.ReadFile(outPath)
		if readErr != nil {
			return fmt.Errorf("analysis: read ocr output: %w", readErr)
		}
		raw = b
		return nil
	})
	if err != nil {
		return OCRResult{}, err
	}
	return parseONNXOCR(raw)
}

func parseONNXOCR(raw []byte) (OCRResult, error) {
	var payload struct {
		FullText string `json:"full_text"`
		Language string `json:"language"`
		Blocks   []struct {
			Text       string  `json:"text"`
			Confidence float64 `json:"confidence"`
			Box        struct {
				X, Y, Width, Height int
			} `json:"box"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return OCRResult{}, fmt.Errorf("analysis: parse ocr output: %w", err)
	}
	res := OCRResult{FullText: payload.FullText, Language: payload.Language}
	for _, b := range payload.Blocks {
		res.Blocks = append(res.Blocks, OCRBlock{Text: b.Text, Confidence: b.Confidence, Box: Box{X: b.Box.X, Y: b.Box.Y, Width: b.Box.Width, Height: b.Box.Height}})
	}
	return res, nil
}

func ocrArgs(path string) []string {
	return []string{path, "stdout", "-l", "eng"}
}

// NSFW classifies src for unsafe content via the onnxruntime Python sidecar.
// It uses the isolated lock-pinned interpreter supplied by the app; a host
// Python fallback would bypass the runtime's dependency and provenance contract.
func (s *Service) NSFW(ctx context.Context, src []byte) (result NSFWResult, runErr error) {
	ctx, finish, admissionErr := s.execution(ctx, OpNSFW)
	if admissionErr != nil {
		return result, admissionErr
	}
	defer func() { runErr = errors.Join(runErr, finish()) }()
	const modelID = "adamcodd-vit-nsfw"
	if s.pythonProgram == "" {
		return NSFWResult{}, fmt.Errorf("%w: isolated Python environment is not provisioned; install uv and restart image-tools", ErrBackendUnavailable)
	}
	python, err := s.resolveLookPath()(s.pythonProgram)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("%w: isolated Python interpreter is unavailable at %q", ErrBackendUnavailable, s.pythonProgram)
	}
	if !s.modelInstalled(modelID) {
		return NSFWResult{}, fmt.Errorf("%w: model %q not installed — run `image-tools models install %s`", ErrBackendUnavailable, modelID, modelID)
	}
	var raw []byte
	err = withTempInput(src, func(path string) error {
		b, err := s.resolveRun()(ctx, python, nsfwArgs(s.modelDir(modelID), path))
		if err != nil {
			return commandError("analysis: nsfw classifier", err)
		}
		raw = b
		return nil
	})
	if err != nil {
		return NSFWResult{}, err
	}
	return s.parseNSFW(raw)
}

// nsfwArgs is the documented onnx sidecar invocation. Pure for unit-testing.
func nsfwArgs(modelDir, path string) []string {
	return []string{"-m", "image_tools_sidecar.nsfw", "--model", modelDir, "--image", path}
}

// parseNSFW interprets the sidecar's JSON output: {"score":0.9,"categories":[...]}.
func (s *Service) parseNSFW(raw []byte) (NSFWResult, error) {
	var payload struct {
		Score      float64 `json:"score"`
		Categories []struct {
			Label string  `json:"label"`
			Score float64 `json:"score"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return NSFWResult{}, fmt.Errorf("analysis: parse nsfw output: %w", err)
	}
	res := NSFWResult{
		Score:     payload.Score,
		NSFW:      payload.Score >= s.threshold,
		Threshold: s.threshold,
		Label:     "sfw",
	}
	if res.NSFW {
		res.Label = "nsfw"
	}
	for _, c := range payload.Categories {
		res.Categories = append(res.Categories, NSFWCategory{Label: c.Label, Score: c.Score})
	}
	return res, nil
}

// ScanNSFW is the advisory half of the generation auto-scan. The engine's
// hard-refusal review is separate and fails closed; this advisory hook degrades
// to "not flagged" with a logged note when the classifier is unavailable, so a
// local install without the model still generates.
func (s *Service) ScanNSFW(ctx context.Context, img []byte) (bool, float64, error) {
	res, err := s.NSFW(ctx, img)
	if err != nil {
		s.logger.Printf("analysis: auto-scan unavailable: %v", err)
		return false, 0, nil
	}
	return res.NSFW, res.Score, nil
}
