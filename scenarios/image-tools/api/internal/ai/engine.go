package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	// CustomModels lists the user-imported (add-only) models so a model imported by
	// `models import` is resolvable for generation right after install. nil ⇒
	// seed-only registry. Merged over Registry via models.Registry.WithCustom per
	// request (a best-effort error leaves the seed registry in place).
	CustomModels func(ctx context.Context) ([]models.Model, error)
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
	// arbitration. Production always wires it; admission/renewal failures stop
	// GPU work, and degraded work requires a real CPU-capable model/backend.
	Capacity CapacityBroker
	// Memory admits local CPU and GPU inference. Production always supplies it;
	// a warm worker must be reaped before this operation's RAM lease is released.
	Memory MemoryAdmission
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

// ModelByID returns a registry model by id for submit-edge trace enrichment,
// resolving user-imported custom models as well as the seed catalog.
func (e *Engine) ModelByID(id string) (models.Model, bool) {
	return e.effectiveRegistry(context.Background()).ByID(id)
}

// effectiveRegistry returns the seed registry merged with the user-imported
// (add-only) custom models, so an imported model is resolvable for generation
// right after install. Falls back to the seed registry when no CustomModels seam
// is wired or the store read / merge fails (logged) — generation of seed models
// is never blocked by a custom-store hiccup.
func (e *Engine) effectiveRegistry(ctx context.Context) *models.Registry {
	if e.deps.CustomModels == nil {
		return e.deps.Registry
	}
	customs, err := e.deps.CustomModels(ctx)
	if err != nil {
		e.deps.Logger.Printf("ai: list custom models (using seed registry): %v", err)
		return e.deps.Registry
	}
	merged, err := e.deps.Registry.WithCustom(customs)
	if err != nil {
		e.deps.Logger.Printf("ai: merge custom models (using seed registry): %v", err)
		return e.deps.Registry
	}
	return merged
}

// Payload is the JSON body submitted as a job's Payload. The runner re-reads it
// to execute the op without re-running selection's host probe from scratch
// (model id is pinned at submit time for determinism).
type Payload struct {
	Operation        string            `json:"operation"`
	InputKey         string            `json:"input_key,omitempty"`
	MaskKey          string            `json:"mask_key,omitempty"`
	ModelID          string            `json:"model_id"`
	Backend          string            `json:"backend,omitempty"`
	Tier             string            `json:"tier,omitempty"`
	GPU              bool              `json:"gpu"`
	AllowBYOK        bool              `json:"allow_byok,omitempty"`
	CapacityPriority string            `json:"capacity_priority,omitempty"`
	AllowReclaim     bool              `json:"allow_reclaim,omitempty"`
	AutoScanNSFW     bool              `json:"auto_scan_nsfw,omitempty"`
	Variations       int               `json:"variations,omitempty"`
	Params           map[string]string `json:"params,omitempty"`
	// Adapters is the validated conditioning stack pinned at submit time (typed,
	// not in Params — decision C2). Empty for an unconditioned op.
	Adapters []adapters.ResolvedAdapter `json:"adapters,omitempty"`
	// Execution-only binding selected from a fresh inventory before admission.
	GPUUUID string `json:"-"`
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
	Operation      string
	ModelOverride  string
	AllowBYOK      bool
	QualityPolicy  string
	FallbackPolicy string
	Priority       string
	AllowReclaim   bool
	// Role is the caller's OpenRouter role, if any. A named role ranks the gateway
	// model first when the cloud tier is permitted.
	Role string
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
	res, err := resolver.New(e.effectiveRegistry(ctx), e.deps.Backends).Resolve(ctx, resolver.Request{
		Operation:          req.Operation,
		ModelOverride:      override,
		Host:               host,
		AllowBYOK:          req.AllowBYOK,
		FallbackPolicy:     req.FallbackPolicy,
		QualityPolicy:      req.QualityPolicy,
		PreferRemote:       strings.TrimSpace(req.Role) != "",
		CapacityManagedGPU: e.deps.Capacity != nil,
		IsEnabled:          enabled,
		Adapters:           req.Adapters,
		AdapterByID:        e.deps.AdapterByID,
		AdapterEnabled:     adapterEnabled,
		AdapterInstalled:   e.deps.AdapterInstalled,
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
		EstimatedSeconds: estimateSecondsForTier(req.Operation, res.GPUViable, res.Tier),
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

// estimateSecondsForTier uses the network ETA for a remote tier (30 s when none
// is declared) instead of the CPU multiplier, which described the wrong machine.
func estimateSecondsForTier(op string, gpuViable bool, tier string) int {
	if tier == "byok-cloud" {
		return 30
	}
	return estimateSeconds(op, gpuViable)
}

// LaneForTier returns the job lane for the selected tier. A remote BYOK-cloud
// tier runs on the network lane, never behind the single local GPU worker, which
// otherwise serialized cloud jobs (F7: ~70 s queued per OpenRouter job). Local
// GPU and CPU tiers stay on the local lanes.
func LaneForTier(tier string) internaljobs.Lane {
	switch tier {
	case "byok-cloud":
		return internaljobs.LaneNetwork
	default:
		return internaljobs.LaneGPU
	}
}

// OpRunner is the per-operation execution function the dispatcher registers. It
// matches both jobrunner.OpRunner and jobs.Runner structurally (an unnamed func
// type is assignable to either named type).
type OpRunner = func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error)

// BuildRunners returns one runner per AI op, ready to register on the
// dispatcher. Each runner materializes inputs, selects + executes the backend,
// persists outputs, and runs the optional NSFW auto-scan.
func (e *Engine) BuildRunners() map[string]OpRunner {
	runners := make(map[string]OpRunner, len(catalog))
	for _, name := range Names() {
		op := name
		runners[op] = func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
			return e.run(ctx, op, job, emit)
		}
	}
	return runners
}

func (e *Engine) run(ctx context.Context, op string, job internaljobs.Job, emit func(progress int, message string)) (result internaljobs.Result, runErr error) {
	jobContext := ctx
	defer func() {
		if runErr != nil {
			if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
				if !errors.Is(runErr, cause) {
					runErr = errors.Join(runErr, cause)
				}
			} else if jobContext.Err() != nil {
				if cause := context.Cause(jobContext); !errors.Is(runErr, cause) {
					runErr = errors.Join(runErr, cause)
				}
			}
		}
	}()
	var pl Payload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return internaljobs.Result{}, fmt.Errorf("ai: decode payload: %w", err)
	}
	model, ok := e.effectiveRegistry(ctx).ByID(pl.ModelID)
	if !ok {
		return internaljobs.Result{}, fmt.Errorf("ai: model %q not in registry", pl.ModelID)
	}
	if !e.deps.ModelInstalled(model.ID) {
		return internaljobs.Result{}, fmt.Errorf("%w: %q", ErrModelNotInstalled, model.ID)
	}

	bsel, err := e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
		Operation:       op,
		ModelBackend:    model.Backend,
		GPUViable:       pl.GPU,
		AllowBYOK:       pl.AllowBYOK,
		RequireAdapters: len(pl.Adapters) > 0,
	})
	if err != nil {
		return internaljobs.Result{}, err
	}
	if bsel.Tier == backends.TierLocalCPU && !model.Hardware.CPUCapable {
		return internaljobs.Result{}, fmt.Errorf("ai: model %q has no CPU execution path on the current backend", model.ID)
	}
	if bsel.Tier == backends.TierLocalGPU && modelVRAMClaimBytes(model) <= 0 {
		if !model.Hardware.CPUCapable {
			return internaljobs.Result{}, fmt.Errorf("ai: model %q has no declared GPU memory budget or CPU execution path", model.ID)
		}
		e.deps.Logger.Printf("ai: model %q has no declared GPU memory budget; using CPU", model.ID)
		bsel, err = e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
			Operation: op, ModelBackend: model.Backend, GPUViable: false,
			AllowBYOK: pl.AllowBYOK, RequireAdapters: len(pl.Adapters) > 0,
		})
		if err != nil {
			return internaljobs.Result{}, err
		}
	}

	if e.deps.Memory != nil && (bsel.Tier == backends.TierLocalCPU || bsel.Tier == backends.TierLocalGPU) {
		if model.Hardware.MinRAMGB <= 0 || model.Hardware.MinRAMGB > 1024 {
			return internaljobs.Result{}, fmt.Errorf("ai: model %q requires a bounded host RAM budget before local execution", model.ID)
		}
		lease, err := e.deps.Memory(ctx, "image-tools:"+job.ID+":ram", int64(model.Hardware.MinRAMGB)<<30)
		if err != nil {
			return internaljobs.Result{}, fmt.Errorf("ai: local RAM capacity: %w", err)
		}
		if lease == nil {
			return internaljobs.Result{}, errors.New("ai: local RAM admission returned no lease")
		}
		ctx = context.WithValue(ctx, operationMemoryContextKey{}, true)
		var stopMemory func()
		ctx, stopMemory = keepMemoryAlive(ctx, lease)
		defer func() {
			stopMemory()
			if result.Meta == nil {
				result.Meta = map[string]string{}
			}
			result.Meta["ram_capacity_claim_bytes"] = strconv.FormatInt(int64(model.Hardware.MinRAMGB)<<30, 10)
			result.Meta["ram_capacity_policy"] = "shared-broker-no-reclaim"
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(jobContext), 10*time.Second)
			defer cancel()
			if err := lease.Release(cleanup); err != nil {
				e.deps.Logger.Printf("ai: job %s RAM capacity release: %v", job.ID, err)
				result.Meta["ram_capacity_release_warning"] = err.Error()
				result.Meta["ram_capacity_release"] = "unknown; retained native TTL recovery"
				runErr = errors.Join(runErr, fmt.Errorf("ai: release local RAM capacity: %w", err))
			} else {
				result.Meta["ram_capacity_release"] = "released"
			}
		}()
	}

	// Capacity arbitration (plan §7 Phase 7): when the selected tier is GPU, make
	// an op-scoped VRAM claim against the host capacity broker. The broker may
	// degrade this batch job to CPU when the GPU is contended by higher-priority
	// work (e.g. an active transcription) — in which case we honor the verdict by
	// re-selecting on CPU. Broker failures refuse unadmitted GPU work; a valid
	// grant is renewed until execution stops, then released independently of
	// job cancellation.
	if e.deps.Capacity != nil && bsel.Tier == backends.TierLocalGPU {
		host, err := e.deps.Probe.Inventory(ctx)
		if err != nil {
			return internaljobs.Result{}, fmt.Errorf("ai: GPU capacity inventory: %w", err)
		}
		gpu, err := capacityGPU(host, modelVRAMClaimBytes(model), bsel.Provider)
		if err != nil {
			return internaljobs.Result{}, err
		}
		lease, cerr := e.deps.Capacity.Claim(ctx, "image-tools:"+job.ID, modelVRAMClaimBytes(model), CapacityClaimOptions{
			Priority:     pl.CapacityPriority,
			AllowReclaim: pl.AllowReclaim,
			GPUUUID:      gpu.UUID,
		})
		if cerr != nil {
			return internaljobs.Result{}, fmt.Errorf("ai: GPU capacity admission failed: %w", cerr)
		} else {
			defer func() {
				if err := e.deps.Capacity.Release(jobContext, lease); err != nil {
					e.deps.Logger.Printf("ai: job %s: %v", job.ID, err)
					if result.Meta == nil {
						result.Meta = map[string]string{}
					}
					result.Meta["capacity_release_warning"] = err.Error()
				}
			}()
			for _, w := range lease.Warnings {
				e.deps.Logger.Printf("ai: capacity: %s", w)
			}
			if lease.DegradeToCPU {
				if !model.Hardware.CPUCapable {
					return internaljobs.Result{}, fmt.Errorf("ai: GPU capacity unavailable; model %q has no CPU execution path", model.ID)
				}
				emit(5, "GPU contended — capacity broker degraded this job to CPU")
				bsel, err = e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
					Operation:       op,
					ModelBackend:    model.Backend,
					GPUViable:       false,
					AllowBYOK:       pl.AllowBYOK,
					RequireAdapters: len(pl.Adapters) > 0,
				})
				if err != nil {
					return internaljobs.Result{}, err
				}
			} else {
				pl.GPUUUID = gpu.UUID
				var stop func()
				ctx, stop = keepCapacityAlive(ctx, e.deps.Capacity, lease, capacityClaimTTL/3)
				defer stop()
			}
		}
	}
	emit(5, fmt.Sprintf("selected %s on %s", model.ID, bsel.Tier))

	tmpDir, err := os.MkdirTemp("", "imgtools-ai-*")
	if err != nil {
		return internaljobs.Result{}, fmt.Errorf("ai: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	inputs, err := e.materializeInputs(ctx, tmpDir, pl)
	if err != nil {
		return internaljobs.Result{}, err
	}
	// Materialize each conditioning adapter's control/reference image (ControlNet /
	// IP-Adapter) from its blob key to a local file, rewriting the key in place so
	// the technique arg-builders hand the sidecar concrete paths. LoRAs carry no
	// image and pass through untouched.
	pl.Adapters, err = e.materializeAdapterImages(ctx, tmpDir, pl.Adapters)
	if err != nil {
		return internaljobs.Result{}, err
	}

	variations := pl.Variations
	if variations < 1 {
		variations = 1
	}
	outKeys := make([]string, 0, variations)
	// meta is the backend's own record of the run — the model that served it,
	// the tier it ran on and what the route cost. It is carried out of the
	// engine rather than dropped at persistOutput, because a caller billed for
	// a cloud generation has no other way to learn the charge.
	var meta map[string]string
	for i := 0; i < variations; i++ {
		select {
		case <-ctx.Done():
			return internaljobs.Result{}, ctx.Err()
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
		key, runMeta, err := e.runOnce(ctx, op, model, bsel, tmpDir, inputs, pl, i, progress)
		if err != nil {
			return internaljobs.Result{}, err
		}
		// The first variation's record is the job's record: every variation runs
		// the same model on the same tier, and a per-variation cost is summed
		// below rather than overwritten.
		if i == 0 {
			// Copied, not aliased: the accumulator below writes to it, and the
			// map belongs to the provider that produced it.
			meta = make(map[string]string, len(runMeta))
			for k, v := range runMeta {
				meta[k] = v
			}
		} else if cost, ok := parseCostUSD(runMeta); ok {
			addCostUSD(meta, cost)
		}
		outKeys = append(outKeys, key)
		emit(hi, fmt.Sprintf("produced %d/%d", i+1, variations))
	}

	primary := outKeys[0]
	if e.deps.AutoScan != nil && pl.AutoScanNSFW && op != "scene_geometry" && op != "image_to_object" {
		emit(92, "auto-scanning output for NSFW content")
		if msg := e.autoScan(ctx, primary); msg != "" {
			emit(96, msg)
		}
	}
	if len(outKeys) > 1 {
		if meta == nil {
			meta = map[string]string{}
		}
		// Persist every variation key so `jobs get` can list them; result_ref
		// stays the first entry for single-result callers.
		meta["result_refs"] = strings.Join(outKeys, ",")
		emit(98, fmt.Sprintf("variations: %v", outKeys))
	}
	return internaljobs.Result{Ref: primary, Meta: meta}, nil
}

func modelVRAMClaimBytes(model models.Model) int64 {
	if model.Hardware.MinVRAMGB > 0 {
		return int64(model.Hardware.MinVRAMGB) << 30
	}
	return 0
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

// materializeAdapterImages downloads each resolved adapter's conditioning image
// (ControlNet control map / IP-Adapter reference) to a local file under tmpDir and
// returns a copy of the stack with ConditioningImageKey rewritten to the local
// path. Adapters with no image (LoRA, or a ControlNet whose map was already
// materialized) pass through unchanged. Fails closed on a fetch error so a
// conditioned job never runs missing a requested control image.
func (e *Engine) materializeAdapterImages(ctx context.Context, tmpDir string, in []adapters.ResolvedAdapter) ([]adapters.ResolvedAdapter, error) {
	if len(in) == 0 {
		return in, nil
	}
	out := make([]adapters.ResolvedAdapter, len(in))
	for i, a := range in {
		out[i] = a
		if a.ConditioningImageKey == "" {
			continue
		}
		p, err := e.fetchToFile(ctx, tmpDir, fmt.Sprintf("cond-%d", i), a.ConditioningImageKey)
		if err != nil {
			return nil, fmt.Errorf("ai: materialize conditioning image for adapter %q: %w", a.ID, err)
		}
		out[i].ConditioningImageKey = p
	}
	return out, nil
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

func (e *Engine) runOnce(ctx context.Context, op string, model models.Model, bsel backends.Selection, tmpDir string, in inputFiles, pl Payload, variation int, progress func(frac float64, message string)) (string, map[string]string, error) {
	ext := ".png"
	if op == "scene_geometry" || op == "image_to_object" {
		ext = ".zip"
	}
	outPath := filepath.Join(tmpDir, fmt.Sprintf("out-%d%s", variation, ext))
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
	sourceProfileDir := ""
	if op == "image_to_object" {
		cli, err := exec.LookPath("vrooli")
		if err != nil {
			return "", nil, fmt.Errorf("image_to_object requires the native TripoSR source profile command: %w", err)
		}
		cmd := exec.CommandContext(ctx, cli, "host", "triposr-source", "profile", "--json")
		out, err := cmd.Output()
		if err != nil {
			return "", nil, fmt.Errorf("resolve managed TripoSR source profile: %w", err)
		}
		var profile struct {
			SourceDir string `json:"source_dir"`
		}
		if err := json.Unmarshal(out, &profile); err != nil || strings.TrimSpace(profile.SourceDir) == "" {
			return "", nil, errors.New("native TripoSR source profile returned invalid JSON")
		}
		sourceProfileDir = profile.SourceDir
	}
	req := backends.Request{
		Operation:        op,
		Model:            model,
		ModelDir:         e.absModelDir(model.ID),
		SourceProfileDir: sourceProfileDir,
		GPU:              bsel.Tier == backends.TierLocalGPU,
		GPUUUID:          pl.GPUUUID,
		InputKeys:        collectInputs(in),
		Output:           storage.OutputTarget{LocalPath: outPath},
		Params:           params,
		Adapters:         pl.Adapters,
		Progress:         progress,
	}
	res, err := bsel.Provider.Execute(ctx, req)
	if err != nil {
		return "", nil, err
	}
	key, err := e.persistOutput(ctx, outPath, res.Meta)
	if err != nil {
		return "", nil, err
	}
	return key, res.Meta, nil
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

func (e *Engine) persistOutput(ctx context.Context, outPath string, meta map[string]string) (string, error) {
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("ai: read backend output: %w", err)
	}
	mime, ext := outputFormat(data, meta)
	key := "out/" + uuid.NewString() + ext
	if err := e.deps.Store.Put(ctx, key, bytes.NewReader(data), mime); err != nil {
		return "", fmt.Errorf("ai: store output: %w", err)
	}
	return key, nil
}

func outputFormat(data []byte, meta map[string]string) (mime, ext string) {
	mime = strings.TrimSpace(meta["media_type"])
	if mime == "" {
		trimmed := strings.TrimSpace(string(data[:min(len(data), 256)]))
		if strings.HasPrefix(trimmed, "<svg") || strings.HasPrefix(trimmed, "<?xml") && strings.Contains(trimmed, "<svg") {
			mime = "image/svg+xml"
		} else {
			mime = http.DetectContentType(data)
		}
	}
	switch strings.ToLower(strings.Split(mime, ";")[0]) {
	case "application/zip":
		return "application/zip", ".zip"
	case "image/svg+xml":
		return "image/svg+xml", ".svg"
	case "image/jpeg":
		return "image/jpeg", ".jpg"
	case "image/webp":
		return "image/webp", ".webp"
	default:
		return "image/png", ".png"
	}
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
