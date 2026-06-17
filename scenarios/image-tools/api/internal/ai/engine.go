package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/google/uuid"
)

// BlobStore is the persistence surface the engine needs (satisfied by
// *internal/storage.Store). Declared at the consumer so tests inject a fake.
type BlobStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Put(ctx context.Context, key string, r io.Reader, mime string) error
}

// NSFWScanner classifies image bytes for the optional auto-scan hook. It is
// injected (implemented by internal/analysis) so the ai package does not depend
// on analysis and stays unit-testable with a fake.
type NSFWScanner func(ctx context.Context, img []byte) (nsfw bool, score float64, err error)

// Deps wires the AI engine's seams.
type Deps struct {
	// Registry is the validated model registry (hardware-fit source of truth).
	Registry *models.Registry
	// Backends is the provider registry (selection ladder + boot invariant).
	Backends *backends.Registry
	// Probe reports host hardware for model selection.
	Probe capabilities.Probe
	// Store persists input + output blobs.
	Store BlobStore
	// Enabled resolves a model's runtime enabled-state (SQLite overlay). nil ⇒
	// seed defaults.
	Enabled func(ctx context.Context) (models.EnabledFunc, error)
	// ModelInstalled reports whether a model's weights are on disk. Required —
	// the engine refuses to launch a job for an uninstalled model with an
	// actionable hint instead of failing opaquely mid-run.
	ModelInstalled func(modelID string) bool
	// ModelsRoot is the absolute directory model weights live under (per model:
	// <ModelsRoot>/models/<id>). Threaded to providers as Request.ModelDir.
	ModelsRoot string
	// AutoScan classifies generated output when a job requests it. Optional; nil
	// disables the hook.
	AutoScan NSFWScanner
	// Logger for diagnostics. Defaults to log.Default().
	Logger *log.Logger
}

// Engine builds and runs AI-op jobs over the configured seams.
type Engine struct {
	deps Deps
}

// NewEngine validates deps and returns the engine.
func NewEngine(deps Deps) (*Engine, error) {
	if deps.Registry == nil || deps.Backends == nil || deps.Probe == nil || deps.Store == nil {
		return nil, errors.New("ai: Registry, Backends, Probe, and Store are required")
	}
	if deps.ModelInstalled == nil {
		return nil, errors.New("ai: ModelInstalled is required")
	}
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &Engine{deps: deps}, nil
}

// Payload is the JSON body submitted as a job's Payload. The runner re-reads it
// to execute the op without re-running selection's host probe from scratch
// (model id is pinned at submit time for determinism).
type Payload struct {
	Operation    string            `json:"operation"`
	InputKey     string            `json:"input_key,omitempty"`
	MaskKey      string            `json:"mask_key,omitempty"`
	ModelID      string            `json:"model_id"`
	GPU          bool              `json:"gpu"`
	AllowBYOK    bool              `json:"allow_byok,omitempty"`
	AutoScanNSFW bool              `json:"auto_scan_nsfw,omitempty"`
	Variations   int               `json:"variations,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
}

// Plan is the selection verdict returned to the submit edge so it can surface
// the chosen model/tier + ETA + warnings before the job runs (and reject an
// unrunnable request up front).
type Plan struct {
	ModelID          string
	Tier             string
	Warnings         []string
	EstimatedSeconds int
	GPUViable        bool
}

// PlanRequest drives pre-submit selection.
type PlanRequest struct {
	Operation     string
	ModelOverride string
	AllowBYOK     bool
}

// ErrModelNotInstalled is returned by Plan when the selected model's weights are
// not yet on disk (the operator must install it via the model management layer).
var ErrModelNotInstalled = errors.New("ai: selected model is not installed")

// Plan selects the model + provider for an op on the current host without
// running it. It fails (before any job is created) when the op is unknown, no
// enabled model fits the host, the model is not installed, or no backend
// provider is available — each with an actionable message.
func (e *Engine) Plan(ctx context.Context, req PlanRequest) (Plan, error) {
	if !Has(req.Operation) {
		return Plan{}, fmt.Errorf("ai: unknown operation %q", req.Operation)
	}
	host, err := e.deps.Probe.Inventory(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("ai: host probe: %w", err)
	}
	enabled, err := e.enabledFunc(ctx)
	if err != nil {
		return Plan{}, err
	}
	sel, err := e.deps.Registry.Select(models.SelectRequest{
		Operation:  req.Operation,
		Host:       host,
		OverrideID: req.ModelOverride,
	}, enabled)
	if err != nil {
		return Plan{}, err // already actionable (no enabled model / VRAM shortfall / override invalid)
	}
	if !e.deps.ModelInstalled(sel.Model.ID) {
		return Plan{}, fmt.Errorf("%w: %q — run `image-tools models install %s`", ErrModelNotInstalled, sel.Model.ID, sel.Model.ID)
	}
	bsel, err := e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
		Operation:    req.Operation,
		ModelBackend: sel.Model.Backend,
		GPUViable:    sel.GPUViable,
		AllowBYOK:    req.AllowBYOK,
	})
	if err != nil {
		return Plan{}, err // "no available provider — install a backend/enable BYOK"
	}
	warnings := append([]string{}, sel.Warnings...)
	warnings = append(warnings, bsel.Warnings...)
	return Plan{
		ModelID:          sel.Model.ID,
		Tier:             bsel.Tier.String(),
		Warnings:         warnings,
		EstimatedSeconds: estimateSeconds(req.Operation, sel.GPUViable),
		GPUViable:        sel.GPUViable,
	}, nil
}

func (e *Engine) enabledFunc(ctx context.Context) (models.EnabledFunc, error) {
	if e.deps.Enabled == nil {
		return nil, nil
	}
	fn, err := e.deps.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai: load enabled overlay: %w", err)
	}
	return fn, nil
}

// estimateSeconds is a coarse initial ETA. GPU paths are far faster; CPU
// defaults carry the "slower" expectation. Measures (Phase 4) replace this with
// observed p50/p95 once real runs accrue.
func estimateSeconds(op string, gpuViable bool) int {
	base := 30
	if o, ok := Get(op); ok && o.Category == CategoryEnhancement {
		base = 15
	}
	if !gpuViable {
		base *= 6
	}
	return base
}

// Lane returns the job lane for an op. All current AI ops are heavy and run on
// the serialized GPU lane (one at a time) to avoid VRAM contention, regardless
// of whether the chosen tier is GPU or CPU — a CPU-bound model still saturates
// the box and should not contend with another heavy run.
func Lane(string) internaljobs.Lane { return internaljobs.LaneGPU }

// OpRunner is the per-operation execution function the dispatcher registers. It
// matches both jobrunner.OpRunner and jobs.Runner structurally (an unnamed func
// type is assignable to either named type).
type OpRunner = func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (string, error)

// BuildRunners returns one runner per AI op, ready to register on the
// dispatcher. Each runner materializes inputs, selects + executes the backend,
// persists outputs, and runs the optional NSFW auto-scan.
func (e *Engine) BuildRunners() map[string]OpRunner {
	runners := make(map[string]OpRunner, len(catalog))
	for _, name := range Names() {
		op := name
		runners[op] = func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (string, error) {
			return e.run(ctx, op, job, emit)
		}
	}
	return runners
}

func (e *Engine) run(ctx context.Context, op string, job internaljobs.Job, emit func(progress int, message string)) (string, error) {
	var pl Payload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return "", fmt.Errorf("ai: decode payload: %w", err)
	}
	model, ok := e.deps.Registry.ByID(pl.ModelID)
	if !ok {
		return "", fmt.Errorf("ai: model %q not in registry", pl.ModelID)
	}
	if !e.deps.ModelInstalled(model.ID) {
		return "", fmt.Errorf("%w: %q", ErrModelNotInstalled, model.ID)
	}

	bsel, err := e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
		Operation:    op,
		ModelBackend: model.Backend,
		GPUViable:    pl.GPU,
		AllowBYOK:    pl.AllowBYOK,
	})
	if err != nil {
		return "", err
	}
	emit(5, fmt.Sprintf("selected %s on %s", model.ID, bsel.Tier))

	tmpDir, err := os.MkdirTemp("", "imgtools-ai-*")
	if err != nil {
		return "", fmt.Errorf("ai: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	inputs, err := e.materializeInputs(ctx, tmpDir, pl)
	if err != nil {
		return "", err
	}

	variations := pl.Variations
	if variations < 1 {
		variations = 1
	}
	outKeys := make([]string, 0, variations)
	for i := 0; i < variations; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		key, err := e.runOnce(ctx, op, model, bsel, tmpDir, inputs, pl, i)
		if err != nil {
			return "", err
		}
		outKeys = append(outKeys, key)
		emit(10+int(float64(i+1)/float64(variations)*80), fmt.Sprintf("produced %d/%d", i+1, variations))
	}

	primary := outKeys[0]
	if e.deps.AutoScan != nil && pl.AutoScanNSFW {
		emit(92, "auto-scanning output for NSFW content")
		if msg := e.autoScan(ctx, primary); msg != "" {
			emit(96, msg)
		}
	}
	if len(outKeys) > 1 {
		emit(98, fmt.Sprintf("variations: %v", outKeys))
	}
	return primary, nil
}

type inputFiles struct {
	image string
	mask  string
}

func (e *Engine) materializeInputs(ctx context.Context, tmpDir string, pl Payload) (inputFiles, error) {
	var in inputFiles
	if pl.InputKey != "" {
		p, err := e.fetchToFile(ctx, tmpDir, "input", pl.InputKey)
		if err != nil {
			return in, err
		}
		in.image = p
	}
	if pl.MaskKey != "" {
		p, err := e.fetchToFile(ctx, tmpDir, "mask", pl.MaskKey)
		if err != nil {
			return in, err
		}
		in.mask = p
	}
	return in, nil
}

func (e *Engine) fetchToFile(ctx context.Context, dir, name, key string) (string, error) {
	rc, _, err := e.deps.Store.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("ai: fetch %q: %w", key, err)
	}
	defer func() { _ = rc.Close() }()
	ext := filepath.Ext(key)
	if ext == "" {
		ext = ".png"
	}
	path := filepath.Join(dir, name+ext)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("ai: create input file: %w", err)
	}
	if _, err := io.Copy(f, rc); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("ai: write input file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("ai: close input file: %w", err)
	}
	return path, nil
}

func (e *Engine) runOnce(ctx context.Context, op string, model models.Model, bsel backends.Selection, tmpDir string, in inputFiles, pl Payload, variation int) (string, error) {
	outPath := filepath.Join(tmpDir, fmt.Sprintf("out-%d.png", variation))
	params := map[string]string{}
	for k, v := range pl.Params {
		params[k] = v
	}
	// Vary the seed per variation so multiple outputs differ deterministically.
	if base, ok := params["seed"]; ok && variation > 0 {
		if n, err := strconv.Atoi(base); err == nil {
			params["seed"] = strconv.Itoa(n + variation)
		}
	}
	req := backends.Request{
		Operation: op,
		Model:     model,
		ModelDir:  e.absModelDir(model.ID),
		GPU:       bsel.Tier == backends.TierLocalGPU,
		InputKeys: collectInputs(in),
		Output:    storage.OutputTarget{LocalPath: outPath},
		Params:    params,
	}
	if _, err := bsel.Provider.Execute(ctx, req); err != nil {
		return "", err
	}
	return e.persistOutput(ctx, outPath)
}

func collectInputs(in inputFiles) []string {
	if in.image == "" {
		return nil
	}
	if in.mask != "" {
		return []string{in.image, in.mask}
	}
	return []string{in.image}
}

func (e *Engine) absModelDir(modelID string) string {
	if e.deps.ModelsRoot == "" {
		return ""
	}
	return filepath.Join(e.deps.ModelsRoot, "models", modelID)
}

func (e *Engine) persistOutput(ctx context.Context, outPath string) (string, error) {
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("ai: read backend output: %w", err)
	}
	key := "out/" + uuid.NewString() + ".png"
	if err := e.deps.Store.Put(ctx, key, bytes.NewReader(data), "image/png"); err != nil {
		return "", fmt.Errorf("ai: store output: %w", err)
	}
	return key, nil
}

func (e *Engine) autoScan(ctx context.Context, key string) string {
	rc, _, err := e.deps.Store.Get(ctx, key)
	if err != nil {
		return ""
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	nsfw, score, err := e.deps.AutoScan(ctx, data)
	if err != nil {
		e.deps.Logger.Printf("ai: auto-scan failed: %v", err)
		return ""
	}
	if nsfw {
		return fmt.Sprintf("NSFW flagged (score %.2f)", score)
	}
	return fmt.Sprintf("NSFW clear (score %.2f)", score)
}
