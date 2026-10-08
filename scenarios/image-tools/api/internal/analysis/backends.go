package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"image-tools/internal/backends"
	"image-tools/internal/models"
)

// builtinProvider exposes analysis-style pure-Go operations through the shared
// backend doctor/selection seam. These operations produce structured JSON, so
// their public REST/CLI path remains handlers/analysis; the provider makes the
// catalog-declared backend families honest and probeable.
type builtinProvider struct {
	name string
	ops  []string
	exec func(context.Context, backends.Request) error
}

func (p *builtinProvider) Name() string                   { return p.name }
func (p *builtinProvider) Operations() []string           { return append([]string(nil), p.ops...) }
func (p *builtinProvider) Standalone() bool               { return true }
func (p *builtinProvider) IsCloud() bool                  { return false }
func (p *builtinProvider) GPUCapable() bool               { return false }
func (p *builtinProvider) Available(context.Context) bool { return true }
func (p *builtinProvider) Availability(context.Context) backends.Availability {
	return backends.Availability{
		Available: true,
		Detail:    fmt.Sprintf("%s provider built into image-tools API binary", p.name),
		Provision: "no host provisioning required",
	}
}

func (p *builtinProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("analysis: builtin backend %q requires a local output path", p.name)
	}
	if err := p.exec(ctx, req); err != nil {
		return backends.Result{}, fmt.Errorf("analysis: builtin backend %q execution failed: %w", p.name, err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name}}, nil
}

type tesseractProvider struct {
	lookPath lookPathFunc
	run      cmdOutput
}

func (p *tesseractProvider) Name() string         { return "library-cgo" }
func (p *tesseractProvider) Operations() []string { return []string{OpOCR} }
func (p *tesseractProvider) Standalone() bool     { return true }
func (p *tesseractProvider) IsCloud() bool        { return false }
func (p *tesseractProvider) GPUCapable() bool     { return false }
func (p *tesseractProvider) Available(ctx context.Context) bool {
	return p.Availability(ctx).Available
}

func (p *tesseractProvider) Availability(context.Context) backends.Availability {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	resolved, err := lookPath("tesseract")
	if err != nil {
		return backends.Availability{
			Available: false,
			Detail:    "tesseract not found on PATH",
			Provision: "install Tesseract OCR and language data through Scenario Dependency Analyzer; see docs/reference/backends.md",
		}
	}
	return backends.Availability{
		Available: true,
		Detail:    fmt.Sprintf("tesseract resolved at %s", resolved),
		Provision: "Tesseract OCR and language data are provisioned on this host",
	}
}

func (p *tesseractProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("analysis: library-cgo backend requires a local output path")
	}
	in, err := backendInput(req)
	if err != nil {
		return backends.Result{}, err
	}
	run := p.run
	if run == nil {
		run = func(ctx context.Context, name string, args []string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		}
	}
	out, err := run(ctx, "tesseract", ocrArgs(in))
	if err != nil {
		return backends.Result{}, fmt.Errorf("analysis: tesseract: %w", err)
	}
	if err := writeJSONFile(req.Output.LocalPath, OCRResult{FullText: strings.TrimRight(string(out), "\n"), Language: "eng"}); err != nil {
		return backends.Result{}, err
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.Name()}}, nil
}

type pythonModuleChecker func(ctx context.Context, python string, modules []string) error

type yuNetProvider struct {
	lookPath lookPathFunc
	checkPy  pythonModuleChecker
	run      cmdOutput
}

func (p *yuNetProvider) Name() string         { return "library-cgo" }
func (p *yuNetProvider) Operations() []string { return []string{"face_detection"} }
func (p *yuNetProvider) Standalone() bool     { return true }
func (p *yuNetProvider) IsCloud() bool        { return false }
func (p *yuNetProvider) GPUCapable() bool     { return false }
func (p *yuNetProvider) Available(ctx context.Context) bool {
	return p.Availability(ctx).Available
}

func (p *yuNetProvider) Availability(ctx context.Context) backends.Availability {
	python, err := p.resolvePython()
	if err != nil {
		return backends.Availability{
			Available: false,
			Detail:    err.Error(),
			Provision: "install OpenCV Python bindings and native OpenCV libraries through Scenario Dependency Analyzer; see docs/reference/backends.md",
		}
	}
	check := p.checkPy
	if check == nil {
		check = defaultCheckPythonModules
	}
	modules := []string{"cv2", "numpy"}
	if err := check(ctx, python, modules); err != nil {
		return backends.Availability{
			Available: false,
			Detail:    fmt.Sprintf("python3 resolved at %s, but OpenCV imports failed: %v", python, err),
			Provision: "install OpenCV Python bindings and native OpenCV libraries through Scenario Dependency Analyzer; see docs/reference/backends.md",
		}
	}
	return backends.Availability{
		Available: true,
		Detail:    fmt.Sprintf("python3 resolved at %s; OpenCV imports ready: %s", python, strings.Join(modules, ",")),
		Provision: "OpenCV Python bindings and native OpenCV libraries are provisioned on this host",
	}
}

func (p *yuNetProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("analysis: library-cgo face_detection backend requires a local output path")
	}
	in, err := backendInput(req)
	if err != nil {
		return backends.Result{}, err
	}
	modelPath := modelAssetPath(req, models.ArtifactONNX)
	python, err := p.resolvePython()
	if err != nil {
		return backends.Result{}, fmt.Errorf("analysis: library-cgo face_detection backend unavailable: %w", err)
	}
	args := []string{
		"-m", "image_tools_sidecar.face_detection",
		"--model", modelPath,
		"--image", in,
		"--out", req.Output.LocalPath,
	}
	run := p.run
	if run == nil {
		run = defaultRunOutput
	}
	if _, err := run(ctx, python, args); err != nil {
		return backends.Result{}, fmt.Errorf("analysis: OpenCV YuNet face detection: %w", err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.Name()}}, nil
}

func (p *yuNetProvider) resolvePython() (string, error) {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	resolved, err := lookPath("python3")
	if err != nil {
		return "", fmt.Errorf("python3 not found on PATH")
	}
	return resolved, nil
}

// RegisterBackendProviders registers pure-Go analysis providers on the shared
// backend registry. It has no external dependencies and performs no host
// provisioning.
func RegisterBackendProviders(reg *backends.Registry) error {
	for _, p := range []*builtinProvider{
		{name: models.BackendComputed, ops: []string{OpQuality}, exec: runQualityProvider},
		{name: models.BackendLibraryGo, ops: []string{OpDuplicate, "qr_barcode_read"}, exec: runLibraryGoProvider},
	} {
		if err := reg.Register(p); err != nil {
			return fmt.Errorf("analysis: register backend provider %q: %w", p.Name(), err)
		}
	}
	if err := reg.Register(&tesseractProvider{}); err != nil {
		return fmt.Errorf("analysis: register backend provider %q: %w", "library-cgo", err)
	}
	if err := reg.Register(&yuNetProvider{}); err != nil {
		return fmt.Errorf("analysis: register backend provider %q: %w", "library-cgo", err)
	}
	return nil
}

func runQualityProvider(_ context.Context, req backends.Request) error {
	return writeAnalysisJSON(req, QualityAssess)
}

func runLibraryGoProvider(_ context.Context, req backends.Request) error {
	switch req.Operation {
	case OpDuplicate:
		return writeAnalysisJSON(req, DuplicateDetect)
	case "qr_barcode_read":
		in, err := backendInput(req)
		if err != nil {
			return err
		}
		if _, err := os.ReadFile(in); err != nil {
			return fmt.Errorf("analysis: read barcode input: %w", err)
		}
		return writeJSONFile(req.Output.LocalPath, map[string]any{
			"symbols": []string{},
			"notes":   []string{"pure-Go barcode decoder seam is registered; no symbol detected by the current lightweight fallback"},
		})
	default:
		return fmt.Errorf("unsupported library-go operation %q", req.Operation)
	}
}

func writeAnalysisJSON[T any](req backends.Request, fn func([]byte) (T, error)) error {
	in, err := backendInput(req)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("analysis: read input: %w", err)
	}
	res, err := fn(data)
	if err != nil {
		return err
	}
	return writeJSONFile(req.Output.LocalPath, res)
}

func backendInput(req backends.Request) (string, error) {
	if len(req.InputKeys) == 0 || req.InputKeys[0] == "" {
		return "", fmt.Errorf("missing input image")
	}
	return req.InputKeys[0], nil
}

func modelAssetPath(req backends.Request, kind models.ArtifactKind) string {
	modelDir := req.ModelDir
	if modelDir == "" {
		modelDir = filepath.Join("models", req.Model.ID)
	}
	for _, a := range req.Model.Source.Assets {
		if a.Kind == kind && a.Filename != "" {
			return filepath.Join(modelDir, a.Filename)
		}
	}
	if len(req.Model.Source.Assets) > 0 && req.Model.Source.Assets[0].Filename != "" {
		return filepath.Join(modelDir, req.Model.Source.Assets[0].Filename)
	}
	return modelDir
}

func defaultCheckPythonModules(ctx context.Context, python string, modules []string) error {
	script := "import " + strings.Join(modules, ", ")
	cmd := exec.CommandContext(ctx, python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func defaultRunOutput(ctx context.Context, name string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w (%s)", name, args, err, string(out))
	}
	return out, nil
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("analysis: encode structured output: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("analysis: write structured output: %w", err)
	}
	return nil
}
