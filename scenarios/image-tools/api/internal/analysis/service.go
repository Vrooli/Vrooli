package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Config configures a Service.
type Config struct {
	// ModelInstalled reports whether a model's weights are on disk. Required for
	// the model-backed ops (ocr/nsfw); probe never consults it.
	ModelInstalled func(modelID string) bool
	// AcquireModelUse holds shared ownership while NSFW inference reads model
	// files; production shares this gate with the Installer and AI engine.
	AcquireModelUse func(modelID string) (release func(), err error)
	// ModelsRoot is the absolute directory model weights live under (per model:
	// <ModelsRoot>/models/<id>).
	ModelsRoot string
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
	acquireModelUse func(string) (func(), error)
	modelsRoot      string
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
		acquireModelUse: cfg.AcquireModelUse,
		modelsRoot:      cfg.ModelsRoot,
		pythonProgram:   strings.TrimSpace(cfg.PythonProgram),
		threshold:       thr,
		lookPath:        cfg.LookPath,
		run:             cfg.Run,
		logger:          logger,
	}, nil
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
		return exec.CommandContext(ctx, name, args...).Output()
	}
}

const maxCommandStderrBytes = 4096

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
func (s *Service) OCR(ctx context.Context, src []byte) (OCRResult, error) {
	if _, err := s.resolveLookPath()("tesseract"); err != nil {
		return OCRResult{}, fmt.Errorf("%w: install tesseract (the `ocr` default backend)", ErrBackendUnavailable)
	}
	var out []byte
	err := withTempInput(src, func(path string) error {
		b, err := s.resolveRun()(ctx, "tesseract", ocrArgs(path))
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

// ocrArgs is the documented tesseract invocation: read <path>, write plain text
// to stdout, English. Pure for unit-testing.
func ocrArgs(path string) []string {
	return []string{path, "stdout", "-l", "eng"}
}

// NSFW classifies src for unsafe content via the onnxruntime Python sidecar.
// It uses the isolated lock-pinned interpreter supplied by the app; a host
// Python fallback would bypass the runtime's dependency and provenance contract.
func (s *Service) NSFW(ctx context.Context, src []byte) (NSFWResult, error) {
	const modelID = "adamcodd-vit-nsfw"
	if s.pythonProgram == "" {
		return NSFWResult{}, fmt.Errorf("%w: isolated Python environment is not provisioned; install uv and restart image-tools", ErrBackendUnavailable)
	}
	python, err := s.resolveLookPath()(s.pythonProgram)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("%w: isolated Python interpreter is unavailable at %q", ErrBackendUnavailable, s.pythonProgram)
	}
	if s.acquireModelUse != nil {
		release, acquireErr := s.acquireModelUse(modelID)
		if acquireErr != nil {
			return NSFWResult{}, fmt.Errorf("analysis: acquire model use %q: %w", modelID, acquireErr)
		}
		if release == nil {
			return NSFWResult{}, fmt.Errorf("analysis: acquire model use %q returned no release function", modelID)
		}
		defer release()
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

// ScanNSFW adapts NSFW to the ai package's NSFWScanner signature for the
// generation auto-scan hook. A backend-unavailable error degrades to "not
// flagged" with a logged note rather than failing the generation job.
func (s *Service) ScanNSFW(ctx context.Context, img []byte) (bool, float64, error) {
	res, err := s.NSFW(ctx, img)
	if err != nil {
		s.logger.Printf("analysis: auto-scan unavailable: %v", err)
		return false, 0, nil
	}
	return res.NSFW, res.Score, nil
}
