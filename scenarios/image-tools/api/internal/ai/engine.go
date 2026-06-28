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

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/resolver"
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
	// DefaultOverride resolves the operator-pinned default model for an op (the
	// settings surface), applied when a request carries no explicit override.
	// Returns "" when no pin exists. nil disables the seam (seed default wins).
	DefaultOverride func(ctx context.Context, operation string) (string, error)
	// ModelInstalled reports whether a model's weights are on disk. Required —
	// the engine refuses to launch a job for an uninstalled model with an
	// actionable hint instead of failing opaquely mid-run.
	ModelInstalled func(modelID string) bool
	// ModelsRoot is the absolute directory model weights live under (per model:
	// <ModelsRoot>/models/<id>). Threaded to providers as Request.ModelDir.
	ModelsRoot string
	// AdapterByID resolves a conditioning adapter by id from the MERGED (seed +
	// custom) catalog. nil disables conditioning (a request that names an adapter
	// is then rejected). Required for adapter support.
	AdapterByID func(id string) (adapters.Adapter, bool)
	// AdapterEnabled resolves the adapter enabled-state overlay. nil ⇒ seed defaults.
	AdapterEnabled func(ctx context.Context) (func(id string) bool, error)
	// AdapterInstalled reports whether an adapter's weights are on disk. nil ⇒
	// treated as not installed (a conditioned request fails honestly until install).
	AdapterInstalled func(id string) bool
	// AdaptersRoot is the absolute directory adapter weights live under (per
	// adapter: <AdaptersRoot>/adapters/<id>). Filled into each ResolvedAdapter.Dir.
	AdaptersRoot string
	// AutoScan classifies generated output when a job requests it. Optional; nil
	// disables the hook.
	AutoScan NSFWScanner
	// Capacity arbitrates op-scoped GPU VRAM against the host capacity broker
	// before a GPU generation (plan §7 Phase 7). Optional; nil disables
	// arbitration and the engine runs exactly as before (advisory by default —
	// the broker can degrade a job from GPU to CPU but never blocks it).
	Capacity CapacityBroker
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

// ModelByID returns a registry model by id for submit-edge trace enrichment.
func (e *Engine) ModelByID(id string) (models.Model, bool) {
	return e.deps.Registry.ByID(id)
}

// Payload is the JSON body submitted as a job's Payload. The runner re-reads it
// to execute the op without re-running selection's host probe from scratch
// (model id is pinned at submit time for determinism).
type Payload struct {
	Operation    string            `json:"operation"`
	InputKey     string            `json:"input_key,omitempty"`
	MaskKey      string            `json:"mask_key,omitempty"`
	ModelID      string            `json:"model_id"`
	Backend      string            `json:"backend,omitempty"`
	Tier         string            `json:"tier,omitempty"`
	GPU          bool              `json:"gpu"`
	AllowBYOK    bool              `json:"allow_byok,omitempty"`
	AutoScanNSFW bool              `json:"auto_scan_nsfw,omitempty"`
	Variations   int               `json:"variations,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
	// Adapters is the validated conditioning stack pinned at submit time (typed,
	// not in Params — decision C2). Empty for an unconditioned op.
	Adapters []adapters.ResolvedAdapter `json:"adapters,omitempty"`
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
	// Weight is the resolved consent weight (none|low|high), ELEVATED to
	// max(op, adapters...) when the request carries a conditioning stack (C4).
	Weight string
	// Adapters is the validated, execution-ready conditioning stack to pin into the
	// job payload (empty for an unconditioned op).
	Adapters []adapters.ResolvedAdapter
}

// PlanRequest drives pre-submit selection.
type PlanRequest struct {
	Operation     string
	ModelOverride string
	AllowBYOK     bool
	// Adapters is the requested conditioning stack (resolved + validated here so an
	// incompatible/not-Ready/uninstalled adapter is rejected before any job).
	Adapters []adapters.AdapterRequest
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
	override := req.ModelOverride
	if override == "" && e.deps.DefaultOverride != nil {
		pinned, derr := e.deps.DefaultOverride(ctx, req.Operation)
		if derr != nil {
			return Plan{}, fmt.Errorf("ai: load op default: %w", derr)
		}
		override = pinned
	}
	// The Resolver is the single home for op→model→technique→backend resolution
	// (it does model selection + native/derived technique derivation + backend-tier
	// selection + the consent weight). The engine adds only the install gate and
	// the ETA, which are submit-edge concerns the read-only explain surface omits.
	adapterEnabled, err := e.adapterEnabledFunc(ctx)
	if err != nil {
		return Plan{}, err
	}
	res, err := resolver.New(e.deps.Registry, e.deps.Backends).Resolve(ctx, resolver.Request{
		Operation:        req.Operation,
		ModelOverride:    override,
		Host:             host,
		AllowBYOK:        req.AllowBYOK,
		IsEnabled:        enabled,
		Adapters:         req.Adapters,
		AdapterByID:      e.deps.AdapterByID,
		AdapterEnabled:   adapterEnabled,
		AdapterInstalled: e.deps.AdapterInstalled,
	})
	if err != nil {
		return Plan{}, err // already actionable (no enabled model / VRAM shortfall / override invalid / no provider / adapter incompatible|not-ready|uninstalled)
	}
	if !e.deps.ModelInstalled(res.Model.ID) {
		return Plan{}, fmt.Errorf("%w: %q — run `image-tools models install %s`", ErrModelNotInstalled, res.Model.ID, res.Model.ID)
	}
	// Fill each resolved adapter's on-disk dir so the pinned payload is execution-
	// ready (no second resolution in the runner).
	resolved := make([]adapters.ResolvedAdapter, len(res.Adapters))
	for i, a := range res.Adapters {
		a.Dir = e.absAdapterDir(a.ID)
		resolved[i] = a
	}
	return Plan{
		ModelID:          res.Model.ID,
		Tier:             res.Tier,
		Warnings:         res.Warnings,
		EstimatedSeconds: estimateSeconds(req.Operation, res.GPUViable),
		GPUViable:        res.GPUViable,
		Weight:           res.Weight,
		Adapters:         resolved,
	}, nil
}

func (e *Engine) adapterEnabledFunc(ctx context.Context) (func(id string) bool, error) {
	if e.deps.AdapterEnabled == nil {
		return nil, nil
	}
	fn, err := e.deps.AdapterEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai: load adapter enabled overlay: %w", err)
	}
	return fn, nil
}

// absAdapterDir is the absolute directory adapter id's weights live under.
func (e *Engine) absAdapterDir(adapterID string) string {
	if e.deps.AdaptersRoot == "" {
		return ""
	}
	return filepath.Join(e.deps.AdaptersRoot, "adapters", adapterID)
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
	// naturalize is a deterministic in-process compositor (no model inference);
	// it finishes in well under a second regardless of GPU, so it does not carry
	// the model-inference ETA or the CPU multiplier.
	if op == "naturalize" {
		return 2
	}
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

	// Capacity arbitration (plan §7 Phase 7): when the selected tier is GPU, make
	// an op-scoped VRAM claim against the host capacity broker. The broker may
	// degrade this batch job to CPU when the GPU is contended by higher-priority
	// work (e.g. an active transcription) — in which case we honor the verdict by
	// re-selecting on CPU. Advisory by default: any broker error leaves the GPU
	// selection untouched (the engine never blocks on the broker).
	if e.deps.Capacity != nil && bsel.Tier == backends.TierLocalGPU {
		lease, cerr := e.deps.Capacity.Claim(ctx, "image-tools:"+job.ID, sdGPUVRAMEstimateBytes)
		if cerr != nil {
			e.deps.Logger.Printf("ai: capacity claim failed (advisory, proceeding on GPU): %v", cerr)
		} else {
			defer e.deps.Capacity.Release(ctx, lease.ClaimID)
			for _, w := range lease.Warnings {
				e.deps.Logger.Printf("ai: capacity: %s", w)
			}
			if lease.DegradeToCPU {
				emit(5, "GPU contended — capacity broker degraded this job to CPU")
				bsel, err = e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
					Operation:    op,
					ModelBackend: model.Backend,
					GPUViable:    false,
					AllowBYOK:    pl.AllowBYOK,
				})
				if err != nil {
					return "", err
				}
			}
		}
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
		// Map this variation's in-flight progress into its slice of the 5–90%
		// band so a long backend run (e.g. sd-cli CPU sampling) advances the bar
		// instead of sitting at a static percent. The first variation starts at 5
		// (the "selected" mark); each subsequent one resumes where the prior ended.
		lo := 5
		if i > 0 {
			lo = 10 + int(float64(i)/float64(variations)*80)
		}
		hi := 10 + int(float64(i+1)/float64(variations)*80)
		progress := func(frac float64, message string) {
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			emit(lo+int(frac*float64(hi-lo)), message)
		}
		key, err := e.runOnce(ctx, op, model, bsel, tmpDir, inputs, pl, i, progress)
		if err != nil {
			return "", err
		}
		outKeys = append(outKeys, key)
		emit(hi, fmt.Sprintf("produced %d/%d", i+1, variations))
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

func (e *Engine) runOnce(ctx context.Context, op string, model models.Model, bsel backends.Selection, tmpDir string, in inputFiles, pl Payload, variation int, progress func(frac float64, message string)) (string, error) {
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
		Adapters:  pl.Adapters,
		Progress:  progress,
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
