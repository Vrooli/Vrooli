package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"image-tools/internal/adapters"
	aigateway "image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/capacity"
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
	// Delete removes an artifact the engine refuses to publish. A hard-refusal
	// verdict is only meaningful if the bytes stop being addressable.
	Delete(ctx context.Context, key string) error
}

// ScanVerdict is what an injected output scanner answers. NSFW is advisory and
// only surfaces when a caller asked for it; HardRefusal is the one
// non-configurable outcome and always fails the job.
type ScanVerdict struct {
	NSFW        bool
	Score       float64
	HardRefusal string
}

// OutputScanner classifies generated image bytes. Both hooks are injected so
// the ai package does not depend on analysis or safety and stays unit-testable.
type OutputScanner func(ctx context.Context, img []byte) (ScanVerdict, error)

// PosePreprocessor derives an OpenPose bone image from a source image. It runs
// inside the already-admitted AI job and writes a concrete PNG under that job's
// temporary directory. The model-backed implementation is injected by the
// analysis owner; nil fails closed for pose-conditioned jobs.
type PosePreprocessor func(ctx context.Context, modelPath, inputPath, outputPath string, width, height int) error

// Deps wires the AI engine's seams.
type Deps struct {
	// Registry is the validated model registry (hardware-fit source of truth).
	Registry *models.Registry
	// Backends is the provider registry (selection ladder + boot invariant).
	Backends *backends.Registry
	// RightsEvaluator is the adapter to the shared policy package. It must load
	// trusted current context itself; nil denies model execution.
	RightsEvaluator models.RightsEvaluator
	// Probe reports host hardware for model selection.
	Probe capabilities.Probe
	// Store persists input + output blobs.
	Store BlobStore
	// Enabled resolves a model's runtime enabled-state (SQLite overlay). nil ⇒
	// seed defaults.
	// Enabled returns explicit per-model overrides. The effective registry is
	// built after custom models are loaded, so its defaults and this overlay are
	// combined against the same catalog snapshot during selection.
	Enabled func(ctx context.Context) (map[string]bool, error)
	// DefaultOverride resolves the operator-pinned default model for an op (the
	// settings surface), applied when a request carries no explicit override.
	// Returns "" when no pin exists. nil disables the seam (seed default wins).
	DefaultOverride func(ctx context.Context, operation string) (string, error)
	// ModelInstalled reports whether a model's weights are on disk. Required —
	// the engine refuses to launch a job for an uninstalled model with an
	// actionable hint instead of failing opaquely mid-run.
	ModelInstalled func(modelID string) bool
	// ModelRuntimeReadiness reports whether the model has a complete built artifact
	// and, when applicable, a provisioned Python environment with passing load-smoke.
	ModelRuntimeReadiness func(ctx context.Context, modelID string) (bool, string)
	// AcquireModelUse retains the installed weights while inference consumes
	// them. Production shares this gate with Installer and analysis services.
	AcquireModelUse func(modelID string) (release func(), err error)
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
	// AcquireAdapterUse protects each conditioning artifact through worker reap.
	AcquireAdapterUse func(id string) (func(), error)
	// AdaptersRoot is the absolute directory adapter weights live under (per
	// adapter: <AdaptersRoot>/adapters/<id>). Filled into each ResolvedAdapter.Dir.
	AdaptersRoot string
	// AutoScan reports the advisory NSFW classification when a job requests it.
	// Optional; nil disables the advisory hook.
	AutoScan OutputScanner
	// ReviewOutput runs on every generated image regardless of what the caller
	// asked for, and fails the job when it returns a HardRefusal. A deployment
	// that serves other people wires this; a local install may leave it nil.
	ReviewOutput OutputScanner
	// PosePreprocessor derives OpenPose maps for ControlNet under the same AI job
	// admission. Optional until the governed CPU runtime and pinned model are ready.
	PosePreprocessor PosePreprocessor
	// Capacity arbitrates op-scoped GPU VRAM against the host capacity broker
	// before a GPU generation (plan §7 Phase 7). Optional; nil disables
	// arbitration. Production always wires it; admission/renewal failures stop
	// GPU work, and degraded work requires a real CPU-capable model/backend.
	Capacity capacity.Broker
	// Memory admits local CPU and GPU inference. Production always supplies it;
	// a warm worker must be reaped before this operation's RAM lease is released.
	Memory capacity.MemoryAdmission
	// CPU reserves the bounded inference thread budget independently of warm
	// model residency. It is released after each job, including cancellation.
	CPU capacity.MemoryAdmission
	// Disk reserves measured output and temporary peak bytes on their actual
	// filesystems through the shared host capacity owner for the full job.
	Disk capacity.DiskAdmission
	// Logger for diagnostics. Defaults to log.Default().
	Logger *log.Logger
}

// Engine builds and runs AI-op jobs over the configured seams.
type Engine struct {
	deps Deps
}

// NewEngineWithRightsEvaluator is the production constructor for model
// execution. Callers inject the shared evaluator explicitly; nil preserves the
// fail-closed behavior while the governed first-party module is unavailable.
func NewEngineWithRightsEvaluator(deps Deps, evaluator models.RightsEvaluator) (*Engine, error) {
	deps.RightsEvaluator = evaluator
	return NewEngine(deps)
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
	Operation string `json:"operation"`
	InputKey  string `json:"input_key,omitempty"`
	MaskKey   string `json:"mask_key,omitempty"`
	ModelID   string `json:"model_id"`
	// Backend is the selected provider identity pinned at submission. Jobs stored
	// before this contract used the model's declared backend; restart recovery
	// fails their nonterminal rows before this runner can admit them.
	Backend          string            `json:"backend,omitempty"`
	Tier             string            `json:"tier,omitempty"`
	GPU              bool              `json:"gpu"`
	AllowBYOK        bool              `json:"allow_byok,omitempty"`
	CapacityPriority string            `json:"capacity_priority,omitempty"`
	AllowReclaim     bool              `json:"allow_reclaim,omitempty"`
	RequireGPU       bool              `json:"require_gpu,omitempty"`
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
	Provider         string
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
	RequireGPU     bool
	ForceCPU       bool
	// Role is the caller's OpenRouter role, if any. A named role ranks the gateway
	// model first when the cloud tier is permitted.
	Role string
	// Adapters is the requested conditioning stack (resolved + validated here so an
	// incompatible/not-Ready/uninstalled adapter is rejected before any job).
	Adapters []adapters.AdapterRequest
}

// ErrModelNotInstalled is returned by Plan when the selected model's weights are
// not yet on disk (the operator must install it via the model management layer).
var (
	ErrModelNotInstalled    = errors.New("ai: selected model is not installed")
	ErrModelNotRuntimeReady = errors.New("ai: selected model is not runtime-ready")
	ErrModelUnsupportedHost = errors.New("ai: selected model does not support the current host")
	ErrGPURequired          = errors.New("ai: requested local GPU execution is unavailable")
)

// Plan selects the model + provider for an op on the current host without
// running it. It fails (before any job is created) when the op is unknown, no
// enabled model fits the host, the model is not installed, or no backend
// provider is available — each with an actionable message.
func (e *Engine) Plan(ctx context.Context, req PlanRequest) (Plan, error) {
	if !Has(req.Operation) {
		return Plan{}, fmt.Errorf("ai: unknown operation %q", req.Operation)
	}
	if req.RequireGPU && req.ForceCPU {
		return Plan{}, errors.New("ai: require_gpu and force_cpu cannot both be set")
	}
	host, err := e.deps.Probe.Inventory(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("ai: host probe: %w", err)
	}
	enabledOverlay, err := e.enabledOverlay(ctx)
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
	registry := e.effectiveRegistry(ctx)
	enabled := registry.EnabledWithOverlay(enabledOverlay)
	res, err := resolver.New(registry, e.deps.Backends, e.deps.RightsEvaluator).Resolve(ctx, resolver.Request{
		Operation:          req.Operation,
		ModelOverride:      override,
		Host:               host,
		AllowBYOK:          req.AllowBYOK,
		FallbackPolicy:     req.FallbackPolicy,
		QualityPolicy:      req.QualityPolicy,
		PreferRemote:       strings.TrimSpace(req.Role) != "",
		CapacityManagedGPU: e.deps.Capacity != nil,
		ForceCPU:           req.ForceCPU,
		IsEnabled:          enabled,
		RightsEvaluator:    e.deps.RightsEvaluator,
		Adapters:           req.Adapters,
		AdapterByID:        e.deps.AdapterByID,
		AdapterEnabled:     adapterEnabled,
		AdapterInstalled:   e.deps.AdapterInstalled,
	})
	if err != nil {
		return Plan{}, err // already actionable (no enabled model / VRAM shortfall / override invalid / no provider / adapter incompatible|not-ready|uninstalled)
	}
	if req.RequireGPU && (res.Tier != backends.TierLocalGPU.String() || !res.GPUViable) {
		return Plan{}, fmt.Errorf("%w for operation %q with model %q", ErrGPURequired, req.Operation, res.Model.ID)
	}
	if !e.deps.ModelInstalled(res.Model.ID) {
		return Plan{}, fmt.Errorf("%w: %q — run `image-tools models install %s`", ErrModelNotInstalled, res.Model.ID, res.Model.ID)
	}
	if e.deps.ModelRuntimeReadiness != nil {
		if ready, state := e.deps.ModelRuntimeReadiness(ctx, res.Model.ID); !ready {
			return Plan{}, fmt.Errorf("%w: %q (%s) — re-run `image-tools models install %s` and resolve the reported runtime issue", ErrModelNotRuntimeReady, res.Model.ID, state, res.Model.ID)
		}
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
		Provider:         res.Provider,
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

func (e *Engine) enabledOverlay(ctx context.Context) (map[string]bool, error) {
	if e.deps.Enabled == nil {
		return nil, nil
	}
	overlay, err := e.deps.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai: load enabled overlay: %w", err)
	}
	return overlay, nil
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
	if o, ok := Get(op); ok && o.Family == "enhance" {
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

// LaneForTier returns the scheduler queue for the selected tier. Remote
// BYOK-cloud work uses the bounded network pool; every local tier uses the
// serialized local queue. The internal LaneGPU name is a legacy queue name and
// does not assert that the selected backend executes on GPU hardware.
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
	requestedVariations := 0
	nextVariation := 0
	var gatewayVariations []aigateway.EvidenceVariation
	gatewayEvidenceObserved := false
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
		if runErr != nil && gatewayEvidenceObserved {
			for index := nextVariation; index < requestedVariations; index++ {
				gatewayVariations = append(gatewayVariations, aigateway.EvidenceVariation{
					Index: index, Outcome: "not_started", CostState: "unknown",
				})
			}
			evidence := aigateway.Evidence{
				SchemaVersion: 1, Complete: false, RequestedVariations: requestedVariations,
				Variations: gatewayVariations,
			}
			if encoded, err := json.Marshal(evidence); err == nil {
				result = internaljobs.Result{Meta: map[string]string{internaljobs.MetaGatewayExecutionEvidence: string(encoded)}}
			}
		}
	}()
	prepared, ok := ctx.Value(executionAdmissionKey{}).(*preparedExecution)
	if !ok {
		if job.Operation == "" {
			job.Operation = op
		}
		admission, err := e.Admit(ctx, job)
		if admission.Release != nil {
			defer func() {
				cleanup, stop := context.WithTimeout(context.WithoutCancel(jobContext), 10*time.Second)
				defer stop()
				if err := admission.Release(cleanup); err != nil {
					runErr = errors.Join(runErr, err)
				}
				if len(admission.Meta) > 0 {
					if result.Meta == nil {
						result.Meta = map[string]string{}
					}
					for key, value := range admission.Meta {
						result.Meta[key] = value
					}
				}
			}()
		}
		if err != nil {
			return internaljobs.Result{}, err
		}
		ctx = admission.Context
		prepared, _ = ctx.Value(executionAdmissionKey{}).(*preparedExecution)
	}
	if prepared == nil {
		return internaljobs.Result{}, errors.New("ai: execution was not admitted")
	}
	model, bsel, pl := prepared.model, prepared.selection, prepared.payload
	emit(5, fmt.Sprintf("selected %s on %s", model.ID, bsel.Tier))

	tmpDir, err := os.MkdirTemp("", "imgtools-ai-*")
	if err != nil {
		return internaljobs.Result{}, fmt.Errorf("ai: temp dir: %w", err)
	}
	if prepared.markDiskCleanupVerified != nil {
		prepared.markDiskCleanupVerified(false)
	}
	defer func() {
		if cleanupErr := removeVerifiedAITempDir(tmpDir, os.RemoveAll); cleanupErr != nil {
			if prepared.markDiskCleanupVerified != nil {
				prepared.markDiskCleanupVerified(false)
			}
			runErr = errors.Join(runErr, cleanupErr)
			return
		}
		if prepared.markDiskCleanupVerified != nil {
			prepared.markDiskCleanupVerified(true)
		}
	}()

	inputs, err := e.materializeInputs(ctx, tmpDir, pl)
	if err != nil {
		return internaljobs.Result{}, err
	}
	// Materialize each conditioning adapter's control/reference image (ControlNet /
	// IP-Adapter) from its blob key to a local file, rewriting the key in place so
	// the technique arg-builders hand the sidecar concrete paths. LoRAs carry no
	// image and pass through untouched.
	pl.Adapters, err = e.materializeConditioningImages(ctx, tmpDir, pl, inputs.image)
	if err != nil {
		return internaljobs.Result{}, err
	}

	variations := pl.Variations
	if variations < 1 {
		variations = 1
	}
	requestedVariations = variations
	outKeys := make([]string, 0, variations)
	// meta is the backend's own record of the run — the model that served it,
	// the tier it ran on and what the route cost. It is carried out of the
	// engine rather than dropped at persistOutput, because a caller billed for
	// a cloud generation has no other way to learn the charge.
	var meta map[string]string
	aggregatedCost := 0.0
	allVariationCostsKnown := true
	for i := 0; i < variations; i++ {
		select {
		case <-ctx.Done():
			nextVariation = i
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
		if attempt, ok := aigateway.EvidenceVariationFromMeta(i, runMeta); ok {
			gatewayVariations = append(gatewayVariations, attempt)
			gatewayEvidenceObserved = true
		}
		if err != nil {
			if !gatewayEvidenceObserved && bsel.Tier == backends.TierBYOK && bsel.Provider.Name() == models.BackendOpenRouter {
				gatewayVariations = append(gatewayVariations, aigateway.EvidenceVariation{Index: i, Outcome: "failed", CostState: "unknown"})
				gatewayEvidenceObserved = true
			} else if gatewayEvidenceObserved && (len(gatewayVariations) == 0 || gatewayVariations[len(gatewayVariations)-1].Index != i) {
				gatewayVariations = append(gatewayVariations, aigateway.EvidenceVariation{Index: i, Outcome: "failed", CostState: "unknown"})
			}
			nextVariation = i + 1
			return internaljobs.Result{}, err
		}
		nextVariation = i + 1
		// The first variation's record is the job's record: every variation runs
		// the same model on the same tier, and a per-variation cost is summed
		// below rather than overwritten.
		if i == 0 {
			// Copied, not aliased: the accumulator below writes to it, and the
			// map belongs to the provider that produced it.
			meta = make(map[string]string, len(runMeta))
			for k, v := range runMeta {
				if k == aigateway.MetaExecutionAttempt {
					continue
				}
				meta[k] = v
			}
		}
		if cost, ok := aigateway.ParseCostUSD(runMeta); ok {
			aggregatedCost += cost
			if math.IsInf(aggregatedCost, 0) || math.IsNaN(aggregatedCost) {
				allVariationCostsKnown = false
			}
		} else {
			allVariationCostsKnown = false
		}
		outKeys = append(outKeys, key)
		emit(hi, fmt.Sprintf("produced %d/%d", i+1, variations))
	}
	if meta == nil && allVariationCostsKnown {
		meta = make(map[string]string)
	}
	if allVariationCostsKnown {
		meta[aigateway.MetaCostUSD] = strconv.FormatFloat(aggregatedCost, 'f', -1, 64)
		meta[aigateway.MetaCostProvenance] = aigateway.ProviderReportedCostProvenance
	} else if meta != nil {
		delete(meta, aigateway.MetaCostUSD)
		delete(meta, aigateway.MetaCostProvenance)
	}
	primary := outKeys[0]
	// The hard-refusal review is unconditional when it is configured: it must not
	// depend on a request flag, because the caller cannot opt out of the one
	// non-configurable boundary. The advisory NSFW scan stays opt-in.
	reviewRequired := e.deps.ReviewOutput != nil && op != "scene_geometry" && op != "image_to_object"
	advisoryScan := e.deps.AutoScan != nil && pl.AutoScanNSFW && op != "scene_geometry" && op != "image_to_object"
	if reviewRequired || advisoryScan {
		// Relinquish execution claims before the next model is admitted. A warm
		// worker retains only its independently admitted resident ownership.
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		err := prepared.finishInference(cleanup)
		stop()
		if err != nil {
			return internaljobs.Result{}, err
		}
	}
	if reviewRequired {
		emit(92, "reviewing output for the non-configurable safety boundary")
		if err := e.reviewOutputs(ctx, outKeys); err != nil {
			return internaljobs.Result{}, err
		}
	}
	if advisoryScan {
		emit(96, "auto-scanning output for NSFW content")
		if msg := e.autoScan(ctx, primary); msg != "" {
			emit(97, msg)
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

func removeVerifiedAITempDir(path string, removeAll func(string) error) error {
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	if err := removeAll(path); err != nil {
		return fmt.Errorf("%w: remove %s: %w", ErrDiskOutputCleanupUnverified, path, err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		if err == nil {
			err = errors.New("temporary directory still exists after removal")
		}
		return fmt.Errorf("%w: verify removal of %s: %w", ErrDiskOutputCleanupUnverified, path, err)
	}
	return nil
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

func (e *Engine) materializeConditioningImages(ctx context.Context, tmpDir string, pl Payload, inputImage string) ([]adapters.ResolvedAdapter, error) {
	if len(pl.Adapters) == 0 {
		return pl.Adapters, nil
	}
	out := make([]adapters.ResolvedAdapter, len(pl.Adapters))
	for i, adapter := range pl.Adapters {
		out[i] = adapter
		if adapter.Kind == adapters.KindControlNet && adapter.Preprocessor == adapters.PreprocessorPose {
			if adapter.ConditioningImageKey == "" {
				return nil, fmt.Errorf("ai: pose control %q requires a source image", adapter.ID)
			}
			if e.deps.PosePreprocessor == nil {
				return nil, fmt.Errorf("ai: pose preprocessor is unavailable; install the governed MediaPipe runtime and pinned Full model")
			}
			modelPath := filepath.Join(adapter.Dir, "pose", "pose_landmarker_full.task")
			source := inputImage
			if source == "" || adapter.ConditioningImageKey != pl.InputKey {
				var err error
				source, err = e.fetchToFile(ctx, tmpDir, fmt.Sprintf("pose-source-%d", i), adapter.ConditioningImageKey)
				if err != nil {
					return nil, fmt.Errorf("ai: materialize pose source for adapter %q: %w", adapter.ID, err)
				}
			}
			width, height := generationDimensions(pl.Params)
			output := filepath.Join(tmpDir, fmt.Sprintf("pose-map-%d.png", i))
			if err := e.deps.PosePreprocessor(ctx, modelPath, source, output, width, height); err != nil {
				return nil, fmt.Errorf("ai: derive OpenPose map for adapter %q: %w", adapter.ID, err)
			}
			info, err := os.Stat(output)
			if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
				return nil, fmt.Errorf("ai: pose preprocessor for adapter %q produced no PNG", adapter.ID)
			}
			out[i].ConditioningImageKey = output
			continue
		}
		if adapter.ConditioningImageKey == "" {
			continue
		}
		path, err := e.fetchToFile(ctx, tmpDir, fmt.Sprintf("cond-%d", i), adapter.ConditioningImageKey)
		if err != nil {
			return nil, fmt.Errorf("ai: materialize conditioning image for adapter %q: %w", adapter.ID, err)
		}
		out[i].ConditioningImageKey = path
	}
	return out, nil
}

func generationDimensions(params map[string]string) (int, int) {
	parse := func(key string) int {
		value, err := strconv.Atoi(params[key])
		if err != nil || value < 1 || value > 4096 {
			return 512
		}
		return value
	}
	return parse("width"), parse("height")
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
	rights, rightsErr := models.EvaluateCurrentModelRights(ctx, e.deps.RightsEvaluator, model, models.RightsUseLocalInference)
	if rightsErr != nil || rights.Status != "allowed" {
		return "", nil, fmt.Errorf("model %q rights are not allowed at execution: %s", model.ID, rights.Reason)
	}
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
		CPUThreads:       admittedCPUThreads(ctx),
		InputKeys:        collectInputs(in),
		Output:           storage.OutputTarget{LocalPath: outPath},
		Params:           params,
		Adapters:         pl.Adapters,
		Progress:         progress,
	}
	res, err := bsel.Provider.Execute(ctx, req)
	if err != nil {
		return "", res.Meta, err
	}
	// A mesh bundle is a structured artifact, not a raster: validate it before it
	// becomes addressable, so a malformed provider output fails the job instead of
	// publishing a corrupt downloadable.
	if op == "image_to_object" {
		if verr := validateMeshArtifact(outPath); verr != nil {
			return "", aigateway.MarkAttemptFailed(res.Meta), verr
		}
	}
	key, err := e.persistOutput(ctx, outPath, res.Meta)
	if err != nil {
		return "", aigateway.MarkAttemptFailed(res.Meta), err
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

// maxReviewBytes bounds what the mandatory review will read from one artifact.
const maxReviewBytes = 64 << 20

// HardRefusalError is returned when an output review identifies the one class
// this project refuses to produce. It is a terminal job failure, not a warning:
// the artifact is deleted before the error is returned.
type HardRefusalError struct{ Reason string }

func (e *HardRefusalError) Error() string { return "ai: output refused: " + e.Reason }

// reviewOutputs runs the configured review on every produced variation. Any
// review failure deletes the artifacts: a deployment that promised the boundary
// cannot leave unreviewed bytes addressable because its classifier crashed.
func (e *Engine) reviewOutputs(ctx context.Context, keys []string) error {
	for _, key := range keys {
		rc, _, err := e.deps.Store.Get(ctx, key)
		if err != nil {
			return errors.Join(fmt.Errorf("ai: read output for safety review: %w", err), e.deleteOutputs(ctx, keys))
		}
		// Bound the read: a provider that produced an oversized file must not turn
		// the mandatory review into an unbounded allocation.
		data, readErr := io.ReadAll(io.LimitReader(rc, maxReviewBytes+1))
		closeErr := rc.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(fmt.Errorf("ai: read output for safety review: %w", errors.Join(readErr, closeErr)), e.deleteOutputs(ctx, keys))
		}
		if len(data) > maxReviewBytes {
			return errors.Join(fmt.Errorf("ai: output exceeds the %d-byte review limit", maxReviewBytes), e.deleteOutputs(ctx, keys))
		}
		verdict, err := e.deps.ReviewOutput(ctx, data)
		if err != nil {
			return errors.Join(fmt.Errorf("ai: required output review failed: %w", err), e.deleteOutputs(ctx, keys))
		}
		if verdict.HardRefusal != "" {
			return errors.Join(&HardRefusalError{Reason: verdict.HardRefusal}, e.deleteOutputs(ctx, keys))
		}
	}
	return nil
}

func (e *Engine) deleteOutputs(ctx context.Context, keys []string) error {
	var errs []error
	for _, key := range keys {
		if err := e.deps.Store.Delete(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("ai: delete refused output %q: %w", key, err))
		}
	}
	return errors.Join(errs...)
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
	verdict, err := e.deps.AutoScan(ctx, data)
	if err != nil {
		e.deps.Logger.Printf("ai: auto-scan failed: %v", err)
		return ""
	}
	if verdict.NSFW {
		return fmt.Sprintf("NSFW flagged (score %.2f)", verdict.Score)
	}
	return fmt.Sprintf("NSFW clear (score %.2f)", verdict.Score)
}
