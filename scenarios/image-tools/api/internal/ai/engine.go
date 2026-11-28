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
	"slices"
	"strconv"
	"strings"
	"time"

	"image-tools/internal/adapters"
	aigateway "image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/capability"
	"image-tools/internal/capacity"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/posemap"
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
// non-configurable outcome and always fails the job (unless Shadow is set).
type ScanVerdict struct {
	NSFW        bool
	Score       float64
	HardRefusal string
	// Detail is the reviewer's one-line account of what it measured (never the
	// image). It is recorded on the job so a pass is auditable, not just a refusal.
	Detail string
	// Action is the reviewer's structured verdict: allow, flag or refuse. Empty
	// derives from HardRefusal/NSFW so a simple scanner keeps working.
	Action string
	// ReasonCode is the machine-readable why (e.g. sexual_minor_face).
	ReasonCode string
	// Evidence is the reviewer's JSON evidence (thresholds, model ids, per-image
	// scores). It is recorded on the job for passes and refusals alike.
	Evidence string
	// Finding is this output's per-image finding JSON, when the reviewer reports
	// one. A quarantined output keeps it beside the bytes for review.
	Finding string
	// Shadow records a would-refuse verdict without deleting or failing.
	Shadow bool
	// DropVariation deletes only this refused output and returns the others.
	DropVariation bool
}

// ActionOrDerived fills Action for a scanner that did not set it.
func (v ScanVerdict) ActionOrDerived() string {
	if v.Action != "" {
		return v.Action
	}
	switch {
	case v.HardRefusal != "":
		return "refuse"
	case v.NSFW:
		return "flag"
	default:
		return "allow"
	}
}

// ReasonCodeOrDerived fills ReasonCode for a simple scanner.
func (v ScanVerdict) ReasonCodeOrDerived() string {
	if v.ReasonCode != "" {
		return v.ReasonCode
	}
	switch v.ActionOrDerived() {
	case "refuse":
		return "sexual_minor_face"
	case "flag":
		return "sexual_adult"
	default:
		return "clean"
	}
}

// ReviewBatch is one review call over every variation. Batch-level Evidence is
// the reviewer's structured record (thresholds/models/version), stored once.
type ReviewBatch struct {
	Verdicts []ScanVerdict
	Evidence string
}

// ReviewSource describes what produced the bytes under review, so the reviewer can
// ask a harder question of output that came from a model able to make explicit
// images. It rides the context because the review hook is a plain bytes-in function.
type ReviewSource struct {
	NSFWCapable bool
}

type reviewSourceKey struct{}

// WithReviewSource returns a context carrying the producing model's review facts.
func WithReviewSource(ctx context.Context, src ReviewSource) context.Context {
	return context.WithValue(ctx, reviewSourceKey{}, src)
}

// ReviewSourceFromContext returns the review facts set by the engine (zero when none).
func ReviewSourceFromContext(ctx context.Context) ReviewSource {
	src, _ := ctx.Value(reviewSourceKey{}).(ReviewSource)
	return src
}

// OutputScanner classifies generated image bytes. Both hooks are injected so
// the ai package does not depend on analysis or safety and stays unit-testable.
// AutoScan is one image; ReviewOutput is the whole variation batch so a reviewer
// loads its models and starts one process per job, not one per variation.
type ImageScanner func(ctx context.Context, img []byte) (ScanVerdict, error)

// OutputScanner reviews every produced variation in one call.
type OutputScanner func(ctx context.Context, images [][]byte) (ReviewBatch, error)

// QuarantineItem is one refused output handed to the quarantine sink. It carries
// the reviewer evidence so an admin can judge the refusal without the job row.
type QuarantineItem struct {
	JobID        string
	OriginalRef  string
	ReasonCode   string
	SafetyReview string
	Score        float64
	Evidence     string
	Finding      string
	MIME         string
}

// QuarantineFunc copies a refused output into a private, time-bounded store and
// returns its quarantine key. Nil means "delete the refused bytes" (the
// deployment chose no retention).
type QuarantineFunc func(ctx context.Context, item QuarantineItem, data []byte) (string, error)

// PosePreprocessor derives an OpenPose bone image from a pose source: a photo, a
// line-art figure or one cell of a chart. It runs inside the already-admitted AI
// job and writes a concrete PNG under that job's temporary directory. The
// model-backed implementation is injected by the analysis owner; nil fails closed
// for pose-conditioned jobs.
type PosePreprocessor func(ctx context.Context, req posemap.Request) error

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
	// AutoAcquirable reports nil when an uninstalled model may be acquired
	// automatically by its job, or the reason it cannot (switch off, manual or
	// custom source, disabled, rights). nil disables auto-acquisition.
	AutoAcquirable func(ctx context.Context, model models.Model) error
	// AcquireModel makes the model's weights present before admission takes
	// any lease, reporting progress. Required when AutoAcquirable is set.
	AcquireModel func(ctx context.Context, modelID, jobID string, emit func(progress int, message string)) error
	// Revalidatable reports nil when an installed model that is not
	// runtime-ready (reason is its lifecycle reason) may be revalidated by
	// AcquireModel before the job runs.
	Revalidatable func(ctx context.Context, model models.Model, reason string) error
	// InstallEstimateSeconds estimates an acquisition for the submit ETA.
	InstallEstimateSeconds func(sizeMBApprox int) int
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
	AutoScan ImageScanner
	// ReviewOutput runs on every generated image regardless of what the caller
	// asked for, and fails the job when it returns a HardRefusal. A deployment
	// that serves other people wires this; a local install may leave it nil.
	ReviewOutput OutputScanner
	// Quarantine retains a refused output for bounded admin review. Nil deletes
	// the refused bytes immediately.
	Quarantine QuarantineFunc
	// ReviewModels are the registry ids ReviewOutput reads from disk. The engine
	// makes them present (governed acquisition, like any model) and holds their use
	// lease around the review, so adaptive offload cannot remove a weight between the
	// admission check and the review, which would otherwise fail the job closed.
	ReviewModels []string
	// ReviewEscalationModels are the weights the review reads only for an output the
	// classifier calls sexual (face finder, age model). They are acquired up front
	// when the producing model is NSFW-capable, and on demand (ReviewWeightsMissing)
	// otherwise, so a hair edit or a deblur never depends on them.
	ReviewEscalationModels []string
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
	// Route is the cloud routing policy (see route.go). The zero value is inactive.
	Route RouteConfig
	// RouteConsent is the consent rule routing and admission obey (see
	// RouteConsentFunc). nil means no consent rule.
	RouteConsent RouteConsentFunc
	// Jobs is the durable job history the cloud spend ledger reads. Required
	// when Route is active.
	Jobs JobSource
	// Now is the clock the wait budget and the rolling spend window use.
	Now func() time.Time
	// Logger for diagnostics. Defaults to log.Default().
	Logger *log.Logger
}

// Engine builds and runs AI-op jobs over the configured seams.
type Engine struct {
	deps Deps
	// yield bounds how long a GPU claim is retried after an idle resident was
	// unloaded; see claimGPU. The zero value retries once, immediately.
	yield yieldPacing
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
	return &Engine{deps: deps, yield: defaultYieldPacing}, nil
}

// CapacityManagedGPU reports the GPU-fit rule jobs use: true when the host
// capacity broker arbitrates GPU memory, so a model fits by the GPU's total
// VRAM and transient contention is decided at admission. Previews must read it
// from here rather than choose their own.
func (e *Engine) CapacityManagedGPU() bool { return e.deps.Capacity != nil }

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
	// ReferenceKey is the reference image of an op that takes one (the source
	// face of face_swap).
	ReferenceKey string `json:"reference_key,omitempty"`
	ModelID      string `json:"model_id"`
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
	// AutoAcquire lets admission acquire the pinned model's weights when they
	// are missing (resolved at submission from the caller's opt-out and the
	// model's eligibility).
	AutoAcquire bool `json:"auto_acquire,omitempty"`
	// Route is the cloud routing record: where the job runs, why, and its caps.
	// The jobs scheduler reads it to re-route a job that waited past its budget.
	Route *RouteState `json:"route,omitempty"`
	// PreserveIdentity is the caller's choice for edit_instruct; nil means on.
	PreserveIdentity *bool `json:"preserve_identity,omitempty"`
	// ConsentAffirmed records that the caller affirmed consent at submit. A route
	// that leaves this machine needs it for a high-weight operation, including a
	// re-route chosen after the job was queued.
	ConsentAffirmed bool `json:"consent_affirmed,omitempty"`
	// ConsentWaived records that the submit edge looked at the input and found no
	// people, which clears the consent rule for a low-weight operation. It is only
	// ever set from that check, never from the caller, and lets a re-route chosen
	// after the job was queued skip a second look.
	ConsentWaived bool `json:"consent_waived,omitempty"`
	// ConsentBasis is why consent was or was not needed (faces_detected=N,
	// detector_unavailable, not_required_local, ...). It is reported as the job's
	// consent_basis.
	ConsentBasis string `json:"consent_basis,omitempty"`
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
	// AutoAcquire is true when the model's weights are missing and the job will
	// acquire them during admission (pin into Payload.AutoAcquire).
	AutoAcquire bool
	// AllowBYOK is the effective cloud permission to pin into the payload: the
	// caller's own, or true when the plan chose the cloud so admission may
	// revalidate it.
	AllowBYOK bool
	// Route is the routing record to pin into Payload.Route (nil when the
	// routing policy is inactive or has nothing to say about this operation).
	Route *RouteState
	// Params are request params the plan decided (for example the cloud role);
	// merge them over the caller's params when they are not already set.
	Params map[string]string
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
	// AutoAcquire is the caller's choice; nil uses the service default (on).
	// false keeps the "not installed" refusal for missing weights.
	AutoAcquire *bool
	// Prompt, Variations and EditStyle feed the cloud role choice and the cost
	// estimate. MaxWaitSeconds and MaxCostUSD are the caller's routing limits;
	// nil uses the deployment default (see RouteConfig).
	Prompt         string
	Variations     int
	EditStyle      string
	MaxWaitSeconds *int
	MaxCostUSD     *float64
	// ConsentAffirmed is the caller's consent affirmation. Automatic cloud
	// routing of a high-weight operation is only considered when it is set.
	ConsentAffirmed bool
	// InputFaces counts the people the request's input images show, for the
	// consent rule of low-weight operations: a route that leaves this machine asks
	// for consent only when someone is shown. It runs at most once per request (the
	// submit edge memoises it) and only when a cloud route is being weighed. nil
	// means nobody can look, which keeps the requirement.
	InputFaces func(ctx context.Context) (int, error)
}

// ErrModelNotInstalled is returned by Plan when the selected model's weights are
// not on disk and the job cannot acquire them automatically (caller opt-out,
// service switch off, manual/custom source, disabled or rights refusal), and by
// admission when an automatic acquisition fails.
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
	base, err := e.resolverRequest(ctx, req)
	if err != nil {
		return Plan{}, err
	}
	res, resolveErr := resolver.New(e.effectiveRegistry(ctx), e.deps.Backends, e.deps.RightsEvaluator).Resolve(ctx, base)
	routed, err := e.planRoute(ctx, req, base, res, resolveErr, base.ModelOverride != "")
	if err != nil {
		return Plan{}, err // already actionable (no enabled model / VRAM shortfall / override invalid / no provider / adapter incompatible|not-ready|uninstalled / spend cap)
	}
	res = routed.res
	if req.RequireGPU && (res.Tier != backends.TierLocalGPU.String() || !res.GPUViable) {
		return Plan{}, fmt.Errorf("%w for operation %q with model %q", ErrGPURequired, req.Operation, res.Model.ID)
	}
	estimate := estimateSecondsForTier(req.Operation, res.GPUViable, res.Tier)
	warnings := res.Warnings
	autoAcquire := false
	if !e.deps.ModelInstalled(res.Model.ID) {
		if err := e.autoAcquireDecision(ctx, req, res.Model); err != nil {
			return Plan{}, err
		}
		autoAcquire = true
		if e.deps.InstallEstimateSeconds != nil {
			estimate += e.deps.InstallEstimateSeconds(res.Model.SizeMBApprox)
		}
		warnings = append(append([]string(nil), warnings...), fmt.Sprintf("model %s is not installed; the job will acquire its weights (~%d MB) before running", res.Model.ID, res.Model.SizeMBApprox))
	} else if e.deps.ModelRuntimeReadiness != nil {
		if ready, state := e.deps.ModelRuntimeReadiness(ctx, res.Model.ID); !ready {
			if err := e.revalidateDecision(ctx, req, res.Model, state); err != nil {
				return Plan{}, err
			}
			autoAcquire = true
			warnings = append(append([]string(nil), warnings...), fmt.Sprintf("model %s is installed but not runtime-ready (%s); the job will revalidate it before running", res.Model.ID, state))
		}
	}
	// Fill each resolved adapter's on-disk dir so the pinned payload is execution-
	// ready (no second resolution in the runner).
	resolved := make([]adapters.ResolvedAdapter, len(res.Adapters))
	for i, a := range res.Adapters {
		a.Dir = e.absAdapterDir(a.ID)
		resolved[i] = a
	}
	if len(routed.warnings) > 0 {
		warnings = append(append([]string(nil), warnings...), routed.warnings...)
	}
	return Plan{
		ModelID:          res.Model.ID,
		Provider:         res.Provider,
		Tier:             res.Tier,
		Warnings:         warnings,
		EstimatedSeconds: estimate,
		GPUViable:        res.GPUViable,
		Weight:           res.Weight,
		Adapters:         resolved,
		AutoAcquire:      autoAcquire,
		AllowBYOK:        req.AllowBYOK || routed.allowBYOK,
		Route:            routed.state,
		Params:           routed.params,
	}, nil
}

// resolverRequest builds the resolver input for req from the probed host, the
// enabled-state overlay and the operator's pinned op default. It is the one
// place the request is assembled, shared by planning and by the scheduler's
// re-route, so both see the same host.
func (e *Engine) resolverRequest(ctx context.Context, req PlanRequest) (resolver.Request, error) {
	host, err := e.deps.Probe.Inventory(ctx)
	if err != nil {
		return resolver.Request{}, fmt.Errorf("ai: host probe: %w", err)
	}
	enabledOverlay, err := e.enabledOverlay(ctx)
	if err != nil {
		return resolver.Request{}, err
	}
	override := req.ModelOverride
	if override == "" && e.deps.DefaultOverride != nil {
		pinned, derr := e.deps.DefaultOverride(ctx, req.Operation)
		if derr != nil {
			return resolver.Request{}, fmt.Errorf("ai: load op default: %w", derr)
		}
		override = pinned
	}
	// The Resolver is the single home for op→model→technique→backend resolution
	// (it does model selection + native/derived technique derivation + backend-tier
	// selection + the consent weight). The engine adds only the install gate and
	// the ETA, which are submit-edge concerns the read-only explain surface omits.
	adapterEnabled, err := e.adapterEnabledFunc(ctx)
	if err != nil {
		return resolver.Request{}, err
	}
	registry := e.effectiveRegistry(ctx)
	return resolver.Request{
		Operation:          req.Operation,
		ModelOverride:      override,
		Host:               host,
		AllowBYOK:          req.AllowBYOK,
		FallbackPolicy:     req.FallbackPolicy,
		QualityPolicy:      req.QualityPolicy,
		PreferRemote:       strings.TrimSpace(req.Role) != "",
		CapacityManagedGPU: e.deps.Capacity != nil,
		ForceCPU:           req.ForceCPU,
		IsEnabled:          registry.EnabledWithOverlay(enabledOverlay),
		RightsEvaluator:    e.deps.RightsEvaluator,
		Adapters:           req.Adapters,
		AdapterByID:        e.deps.AdapterByID,
		AdapterEnabled:     adapterEnabled,
		AdapterInstalled:   e.deps.AdapterInstalled,
	}, nil
}

// autoAcquireDecision returns nil when the job may acquire the missing model,
// or an ErrModelNotInstalled refusal naming why it cannot and how to recover.
func (e *Engine) autoAcquireDecision(ctx context.Context, req PlanRequest, model models.Model) error {
	if req.AutoAcquire != nil && !*req.AutoAcquire {
		return fmt.Errorf("%w: %q (automatic acquisition was declined by the caller) — run `image-tools models install %s`", ErrModelNotInstalled, model.ID, model.ID)
	}
	if e.deps.AutoAcquirable == nil || e.deps.AcquireModel == nil {
		return fmt.Errorf("%w: %q — run `image-tools models install %s`", ErrModelNotInstalled, model.ID, model.ID)
	}
	if err := e.deps.AutoAcquirable(ctx, model); err != nil {
		return fmt.Errorf("%w: %w", ErrModelNotInstalled, err)
	}
	return nil
}

// revalidateDecision accepts an installed, not-ready model when the caller
// allows automatic acquisition and its lifecycle reason is one revalidation
// resolves; otherwise it returns the actionable not-runtime-ready refusal.
func (e *Engine) revalidateDecision(ctx context.Context, req PlanRequest, model models.Model, state string) error {
	refuse := func(detail string) error {
		return fmt.Errorf("%w: %q (%s)%s — re-run `image-tools models install %s` and resolve the reported runtime issue", ErrModelNotRuntimeReady, model.ID, state, detail, model.ID)
	}
	if req.AutoAcquire != nil && !*req.AutoAcquire {
		return refuse(" (automatic revalidation was declined by the caller)")
	}
	if e.deps.Revalidatable == nil || e.deps.AcquireModel == nil {
		return refuse("")
	}
	if err := e.deps.Revalidatable(ctx, model, state); err != nil {
		return fmt.Errorf("%w: %q (%s): %w", ErrModelNotRuntimeReady, model.ID, state, err)
	}
	return nil
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
		run := e.run
		if manifest, ok := capability.Get(op); ok && manifest.Extent != nil {
			run = e.runExtent
		}
		runners[op] = func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
			return run(ctx, op, job, emit)
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
	// routeEvidence is the routing record, kept even when the run fails so a
	// failed job still says where it ran and what it was allowed to spend.
	var routeEvidence map[string]string
	defer func() {
		if runErr != nil {
			// A cancellation with its own cause (a deadline, a lost lease) explains
			// the failure. A plain one is the job manager's to report: it knows
			// whether the request was cancelled or the owner merely released the
			// reservation, which must not be appended to a real reason.
			if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
				if !errors.Is(runErr, cause) {
					runErr = errors.Join(runErr, cause)
				}
			}
		}
		if runErr != nil && (gatewayEvidenceObserved || len(routeEvidence) > 0) {
			// A failed job keeps exactly its routing record and the gateway's
			// partial execution evidence; never output, cost or admission facts.
			failed := make(map[string]string, len(routeEvidence)+len(result.Meta)+1)
			for k, v := range routeEvidence {
				failed[k] = v
			}
			// The reviewer's evidence is what triage and calibration read after a
			// refusal, so it must survive this narrowing too. Replace the result
			// wholesale only for the approved keys; a refused job otherwise loses
			// the very evidence that explains it.
			for k, v := range result.Meta {
				if strings.HasPrefix(k, "safety_") || k == MetaNSFWScore {
					failed[k] = v
				}
			}
			if gatewayEvidenceObserved {
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
					failed[internaljobs.MetaGatewayExecutionEvidence] = string(encoded)
				}
			}
			result = internaljobs.Result{Meta: failed}
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
	routeEvidence = routeMeta(pl.Route, e.deps.Route)
	if pl.ConsentBasis != "" {
		if routeEvidence == nil {
			routeEvidence = map[string]string{}
		}
		routeEvidence[MetaConsentBasis] = pl.ConsentBasis
	}
	// An instruction edit keeps its subject by default; the effective prompt is
	// reported in the meta so the added wording is never hidden.
	var promptMeta map[string]string
	if op == "edit_instruct" {
		pl.Params, promptMeta = editPromptParams(pl.Params, pl.PreserveIdentity)
	}
	if colourParams, colourMeta := colourPromptParams(op, pl.Params); colourMeta != nil {
		pl.Params = colourParams
		if promptMeta == nil {
			promptMeta = map[string]string{}
		}
		for k, v := range colourMeta {
			promptMeta[k] = v
		}
	}
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
	pl.Adapters, err = e.materializeConditioningImages(ctx, tmpDir, job.ID, &pl, inputs.image, emit)
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
			if i == 0 && bsel.Tier == backends.TierBYOK {
				// The first call was refused before any billing (no credits, bad
				// credentials, provider unreachable). A job the policy moved here
				// returns to its local route; a caller who demanded the cloud
				// gets the honest error, and either way the refusal is recorded
				// so the cloud is paused for everyone else.
				if code, refused := cloudRefusal(err); refused {
					if back := e.fellBackLocal(pl, code, err); back != nil {
						return internaljobs.Result{}, back
					}
					if routeEvidence == nil {
						routeEvidence = map[string]string{}
					}
					for k, v := range e.refusalMeta(code, err) {
						routeEvidence[k] = v
					}
				}
			}
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
	if len(promptMeta) > 0 || len(routeEvidence) > 0 {
		if meta == nil {
			meta = map[string]string{}
		}
		for k, v := range routeEvidence {
			meta[k] = v
		}
		for k, v := range promptMeta {
			meta[k] = v
		}
		if pl.Route != nil && allVariationCostsKnown && pl.Tier == backends.TierBYOK.String() && aggregatedCost > pl.Route.MaxCostUSD+1e-9 {
			meta["route_cost_over_cap"] = "true"
		}
	}
	// The hard-refusal review is unconditional when it is configured: it must not
	// depend on a request flag, because the caller cannot opt out of the one
	// non-configurable boundary. The advisory NSFW scan stays opt-in.
	reviewRequired := e.reviewRequired(op)
	advisoryScan := e.deps.AutoScan != nil && pl.AutoScanNSFW && outputIsRaster(op)
	if reviewRequired || advisoryScan {
		// Relinquish execution claims before the next model is admitted. A warm
		// worker retains only its independently admitted resident ownership.
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		err := prepared.finishInference(cleanup)
		stop()
		if err != nil {
			return internaljobs.Result{}, err
		}
		// The admitted context belongs to the claims just released: a warm
		// worker's reservation cancels it on release. The review and scan are
		// the job's own work, so they continue on the context the job was
		// admitted under, which ends only when the job is canceled or the
		// server stops.
		if prepared.jobContext != nil {
			ctx = prepared.jobContext
		}
	}
	if reviewRequired {
		emit(92, "reviewing output for the non-configurable safety boundary")
		report, allowed, err := e.reviewOutputs(ctx, job.ID, outKeys, ReviewSource{NSFWCapable: model.CapabilityLabels.NSFWCapable}, emit)
		if meta == nil {
			meta = map[string]string{}
		}
		report.addTo(meta)
		if err != nil {
			if report.action == "unavailable" {
				meta["safety_warning"] = "the safety reviewer was unavailable; the output was removed and the job failed closed"
			}
			return internaljobs.Result{Meta: meta}, err
		}
		outKeys = allowed
		if len(outKeys) == 0 {
			return internaljobs.Result{Meta: meta}, fmt.Errorf("%w: no variation passed the safety review", ErrSafetyUnavailable)
		}
		if report.action == "partial_refused" {
			meta["safety_warning"] = "one or more variations were refused and removed; the remaining variations are below"
		}
	}
	primary := outKeys[0]
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

// modelResidentVRAMClaimBytes is the larger claim that keeps a staged model's
// stages GPU-resident. It is zero unless the registry declares one above the
// streamed floor.
func modelResidentVRAMClaimBytes(model models.Model) int64 {
	if model.Hardware.ResidentVRAMGB > model.Hardware.MinVRAMGB {
		return int64(model.Hardware.ResidentVRAMGB) << 30
	}
	return 0
}

func modelVRAMClaimBytes(model models.Model) int64 {
	if model.Hardware.MinVRAMGB > 0 {
		return int64(model.Hardware.MinVRAMGB) << 30
	}
	return 0
}

type inputFiles struct {
	image     string
	mask      string
	reference string
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
	if pl.ReferenceKey != "" {
		p, err := e.fetchToFile(ctx, tmpDir, "reference", pl.ReferenceKey)
		if err != nil {
			return in, err
		}
		in.reference = p
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

// materializeConditioningImages is materializeAdapterImages for a stack that may
// hold pose controls: a ControlNet whose preprocessor is "pose" gets its control
// image drawn from the pose source it was given (see derivePoseControl) and every
// other adapter's image is fetched as it is. A pose control that was not given an
// image reads the job's own input.
func (e *Engine) materializeConditioningImages(ctx context.Context, tmpDir, jobID string, pl *Payload, inputImage string, emit func(int, string)) ([]adapters.ResolvedAdapter, error) {
	if len(pl.Adapters) == 0 {
		return pl.Adapters, nil
	}
	out := make([]adapters.ResolvedAdapter, len(pl.Adapters))
	var releasePose func()
	defer func() {
		if releasePose != nil {
			releasePose()
		}
	}()
	for i, adapter := range pl.Adapters {
		out[i] = adapter
		if adapter.Kind == adapters.KindControlNet && adapter.Preprocessor == adapters.PreprocessorPose {
			if adapter.ConditioningImageKey == "" {
				return nil, fmt.Errorf("ai: pose control %q requires a source image", adapter.ID)
			}
			if e.deps.PosePreprocessor == nil {
				return nil, fmt.Errorf("ai: pose preprocessor is unavailable; the pose reader needs the governed Python runtime")
			}
			if releasePose == nil {
				release, err := e.holdModels(ctx, jobID, []string{posemap.ModelID}, "pose reading", emit)
				if err != nil {
					return nil, err
				}
				releasePose = release
			}
			source := inputImage
			if source == "" || adapter.ConditioningImageKey != pl.InputKey {
				var err error
				source, err = e.fetchToFile(ctx, tmpDir, fmt.Sprintf("pose-source-%d", i), adapter.ConditioningImageKey)
				if err != nil {
					return nil, fmt.Errorf("ai: materialize pose source for adapter %q: %w", adapter.ID, err)
				}
			}
			control, err := e.derivePoseControl(ctx, tmpDir, i, adapter, source, pl.Params)
			if err != nil {
				return nil, fmt.Errorf("ai: derive OpenPose map for adapter %q: %w", adapter.ID, err)
			}
			out[i].ConditioningImageKey = control
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
	// A cloud generation is bounded by the job's spend cap: the provider reserves a
	// request's worst-case cost before it runs, so the ceiling has to travel with
	// the request. The cap covers every variation, so each image gets its share.
	if pl.Route != nil && pl.Route.MaxCostUSD > 0 && bsel.Provider.IsCloud() {
		share := pl.Route.MaxCostUSD / float64(max(pl.Variations, 1))
		params[aigateway.ParamMaxCostUSD] = strconv.FormatFloat(share, 'f', -1, 64)
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
		GPURequired:      pl.RequireGPU,
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
	res.Meta = withExecutionDevice(res.Meta, bsel.Tier)
	// A family render stamps its receipt into the PNG; the measured peak belongs in
	// the job record, read before anything rewrites the pixels.
	for key, value := range receiptMeta(outPath) {
		if res.Meta == nil {
			res.Meta = map[string]string{}
		}
		if _, set := res.Meta[key]; !set {
			res.Meta[key] = value
		}
	}
	// An in-place edit keeps the source's frame whichever provider made it.
	policy, rewritten, err := matchSourceFrame(op, in.image, outPath, params)
	if err != nil {
		return "", aigateway.MarkAttemptFailed(res.Meta), err
	}
	if policy != framePolicyModel {
		if res.Meta == nil {
			res.Meta = map[string]string{}
		}
		res.Meta["frame_policy"] = policy
		if rewritten {
			res.Meta["media_type"] = "image/png"
		}
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

// Job meta keys that say where a local job actually ran. The tier a submit
// reports is the plan; admission can still move the job to the CPU when the GPU
// lease is denied, so the result states what happened.
const (
	MetaExecutionTier   = "execution_tier"
	MetaExecutionDevice = "execution_device"
)

// withExecutionDevice records the tier and device a local job ran on. Hosted and
// gateway-served routes carry their own provider evidence and are left alone.
func withExecutionDevice(meta map[string]string, tier backends.Tier) map[string]string {
	device := ""
	switch tier {
	case backends.TierLocalGPU:
		device = "gpu"
	case backends.TierLocalCPU:
		device = "cpu"
	default:
		return meta
	}
	if meta == nil {
		meta = map[string]string{}
	}
	if _, ran := meta[MetaExecutionDevice]; ran {
		// The provider already said where it ran (it moved the job to the CPU
		// after the GPU run failed); the planned tier must not overwrite that.
		return meta
	}
	meta[MetaExecutionTier] = tier.String()
	meta[MetaExecutionDevice] = device
	return meta
}

// collectInputs orders the operation inputs the way backends.Request.InputKeys
// documents them: image, mask, reference. A reference without a mask leaves the
// mask slot empty so a provider reads each role by position, never by guessing.
func collectInputs(in inputFiles) []string {
	if in.image == "" {
		return nil
	}
	switch {
	case in.reference != "":
		return []string{in.image, in.mask, in.reference}
	case in.mask != "":
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
		} else if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid(data) {
			// Text and data outputs (a caption, detections, an embedding) are JSON
			// documents, not rasters: name them so.
			mime = "application/json"
		} else {
			mime = http.DetectContentType(data)
		}
	}
	switch strings.ToLower(strings.Split(mime, ";")[0]) {
	case "application/json":
		return "application/json", ".json"
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

// ReviewWeightsMissing is what an OutputScanner returns when the review needs model
// weights that are not installed. The engine acquires IDs through governed
// acquisition and runs the review again once; if that fails the output is refused
// like any other unavailable review (fail closed, for this output only).
type ReviewWeightsMissing struct{ IDs []string }

func (e *ReviewWeightsMissing) Error() string {
	return "ai: the safety review needs model weights that are not installed: " + strings.Join(e.IDs, ", ")
}

// reviewModelIDs lists the weights a review of this producer's output needs now:
// the classifier always, and the face and age models too when the producing model
// can make explicit images (its output is very likely to need the age check).
func (e *Engine) reviewModelIDs(nsfwCapable bool) []string {
	ids := slices.Clone(e.deps.ReviewModels)
	if nsfwCapable {
		ids = append(ids, e.deps.ReviewEscalationModels...)
	}
	return ids
}

// HardRefusalError is returned when an output review identifies the one class
// this project refuses to produce. It is a terminal job failure, not a warning:
// the artifact is deleted before the error is returned.
type HardRefusalError struct {
	Reason     string
	ReasonCode string
}

func (e *HardRefusalError) Error() string {
	if e.ReasonCode != "" {
		return "ai: output refused (" + e.ReasonCode + "): " + e.Reason
	}
	return "ai: output refused: " + e.Reason
}

// FailureKind marks the refusal as a policy decision, not a fault of the model.
func (e *HardRefusalError) FailureKind() internaljobs.FailureKind { return internaljobs.FailureRefused }

// ErrSafetyUnavailable is the technical-review failure: the boundary failed
// closed, but the model did nothing wrong. It is retryable in principle and is
// deliberately a distinct job failure kind.
var ErrSafetyUnavailable = internaljobs.Kinded(internaljobs.FailureSafetyUnavailable, errors.New("ai: safety review unavailable"))

// Job result_meta keys the output review records. `sensitive` is the flag the API
// and UI act on (blur, tap to reveal); the rest make a pass or refusal auditable.
// The vocabulary lives in internal/jobs so the feedback surface reads the same
// constants the engine writes.
const (
	MetaSensitive             = internaljobs.MetaSensitive
	MetaSensitiveRefs         = internaljobs.MetaSensitiveRefs
	MetaSensitiveReason       = internaljobs.MetaSensitiveReason
	MetaNSFWScore             = internaljobs.MetaNSFWScore
	MetaSafetyReview          = internaljobs.MetaSafetyReview
	MetaSafetyReviewNotes     = internaljobs.MetaSafetyReviewNotes
	MetaSafetyAction          = internaljobs.MetaSafetyAction
	MetaSafetyReason          = internaljobs.MetaSafetyReason
	MetaSafetyEvidence        = internaljobs.MetaSafetyEvidence
	MetaSafetyReviewedRefs    = internaljobs.MetaSafetyReviewedRefs
	MetaSafetyAllowedRefs     = internaljobs.MetaSafetyAllowedRefs
	MetaSafetyRefusedRefs     = internaljobs.MetaSafetyRefusedRefs
	MetaSafetyShadowRefs      = internaljobs.MetaSafetyShadowRefs
	MetaSafetyQuarantineRefs  = internaljobs.MetaSafetyQuarantineRefs
	MetaSafetyQuarantineError = internaljobs.MetaSafetyQuarantineError
)

// reviewReport is what the mandatory review learned about the produced outputs.
type reviewReport struct {
	// action is the aggregate outcome: allow, flag, refuse, partial_refused,
	// shadow_refused, or unavailable.
	action    string
	sensitive []string
	// reasons records why outputs were flagged: "classifier" (the output scored as
	// sexual) and/or "model" (an NSFW-capable model produced it, so every output of
	// that model is treated as sensitive whatever the classifier said).
	reasons []string
	// codes are the machine-readable reason codes (safety_reason).
	codes    []string
	maxScore float64
	details  []string
	evidence string
	// reviewed is every ref in the order the reviewer saw it, so a per-output
	// finding can be mapped back to the blob that produced it.
	reviewed []string
	refused  []string
	shadow   []string
	allowed  []string
	// quarantined holds the private keys that retain refused outputs for
	// bounded admin review; quarantineErrors records a failed retention.
	quarantined      []string
	quarantineErrors []string
}

// addTo records the review on the job's result meta. It is called for passes,
// queued shadow refusals, and refusals that fail the job.
func (r reviewReport) addTo(meta map[string]string) {
	action := r.action
	if action == "" {
		switch {
		case len(r.refused) > 0 && len(r.allowed) == 0:
			action = "refuse"
		case len(r.refused) > 0:
			action = "partial_refused"
		case len(r.shadow) > 0:
			action = "shadow_refused"
		case len(r.sensitive) > 0:
			action = "flag"
		default:
			action = "allow"
		}
	}
	meta[MetaSafetyAction] = action
	meta[MetaSafetyReview] = reviewStatus(action)
	meta[MetaNSFWScore] = strconv.FormatFloat(r.maxScore, 'f', 4, 64)
	if len(r.details) > 0 {
		meta[MetaSafetyReviewNotes] = strings.Join(r.details, " | ")
	}
	if len(r.codes) > 0 {
		meta[MetaSafetyReason] = strings.Join(r.codes, ",")
	} else if action == "flag" && slices.Contains(r.reasons, "model") {
		meta[MetaSafetyReason] = "model"
	} else if action == "allow" || action == "flag" {
		meta[MetaSafetyReason] = "clean"
	}
	if r.evidence != "" {
		meta[MetaSafetyEvidence] = r.evidence
	}
	if len(r.reviewed) > 0 {
		meta[MetaSafetyReviewedRefs] = strings.Join(r.reviewed, ",")
	}
	if len(r.allowed) > 0 {
		meta[MetaSafetyAllowedRefs] = strings.Join(r.allowed, ",")
	}
	if len(r.refused) > 0 {
		meta[MetaSafetyRefusedRefs] = strings.Join(r.refused, ",")
	}
	if len(r.shadow) > 0 {
		meta[MetaSafetyShadowRefs] = strings.Join(r.shadow, ",")
	}
	if len(r.quarantined) > 0 {
		meta[MetaSafetyQuarantineRefs] = strings.Join(r.quarantined, ",")
	}
	if len(r.quarantineErrors) > 0 {
		meta[MetaSafetyQuarantineError] = strings.Join(r.quarantineErrors, " | ")
	}
	if len(r.sensitive) > 0 {
		meta[MetaSensitive] = "true"
		meta[MetaSensitiveRefs] = strings.Join(r.sensitive, ",")
		meta[MetaSensitiveReason] = strings.Join(r.reasons, ",")
	}
}

func reviewStatus(action string) string {
	switch action {
	case "refuse":
		return "refused"
	case "partial_refused":
		return "partial_refused"
	case "shadow_refused":
		return "shadow_refused"
	case "flag":
		return "flagged"
	case "unavailable":
		return "unavailable"
	default:
		return "passed"
	}
}

// flag marks one output sensitive for a reason, once per output and reason.
func (r *reviewReport) flag(key, reason string) {
	if !slices.Contains(r.sensitive, key) {
		r.sensitive = append(r.sensitive, key)
	}
	if !slices.Contains(r.reasons, reason) {
		r.reasons = append(r.reasons, reason)
	}
}

// code records a machine-readable reason once. "clean" is implied by an empty
// reason list, so it is not carried alongside a real refusal/flag.
func (r *reviewReport) code(code string) {
	if code == "" || code == "clean" {
		return
	}
	if !slices.Contains(r.codes, code) {
		r.codes = append(r.codes, code)
	}
}

func (r *reviewReport) detail(detail string) {
	detail = strings.TrimSpace(detail)
	if detail != "" && !slices.Contains(r.details, detail) {
		r.details = append(r.details, detail)
	}
}

// holdReviewModels makes the review's model weights present and holds their use
// leases until the returned release runs. A weight that was offloaded since the
// last job is re-acquired here, so a reviewer that is configured can always run.
func (e *Engine) holdReviewModels(ctx context.Context, jobID string, ids []string, emit func(int, string)) (func(), error) {
	return e.holdModels(ctx, jobID, ids, "safety review", emit)
}

// holdModels makes the weights of the named models present and holds their use
// leases until the returned release runs; purpose names what needs them in the
// progress line and in the error.
func (e *Engine) holdModels(ctx context.Context, jobID string, ids []string, purpose string, emit func(int, string)) (func(), error) {
	var releases []func()
	release := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	for _, id := range ids {
		if e.deps.ModelInstalled != nil && !e.deps.ModelInstalled(id) {
			if e.deps.AcquireModel == nil {
				release()
				return nil, fmt.Errorf("%w: %q (needed for %s)", ErrModelNotInstalled, id, purpose)
			}
			if err := e.deps.AcquireModel(ctx, id, jobID, func(p int, m string) { emit(p, purpose+" weights: "+m) }); err != nil {
				release()
				return nil, fmt.Errorf("%w: %q (needed for %s): %w", ErrModelNotInstalled, id, purpose, err)
			}
		}
		if e.deps.AcquireModelUse != nil {
			held, err := e.deps.AcquireModelUse(id)
			if err != nil {
				release()
				return nil, fmt.Errorf("ai: hold %s model %q: %w", purpose, id, err)
			}
			releases = append(releases, held)
		}
	}
	return release, nil
}

// reviewOutputs runs the configured review on every produced variation in one
// reviewer call. It returns the keys that remain allowed. A technical failure
// deletes the unreviewed artifacts (fail closed) and is classified as
// ErrSafetyUnavailable so it does not count against the model. A genuine refusal
// is classified as a policy refusal.
func (e *Engine) reviewOutputs(ctx context.Context, jobID string, keys []string, source ReviewSource, emit func(int, string)) (reviewReport, []string, error) {
	var report reviewReport
	report.reviewed = slices.Clone(keys)
	// The model work is finished; the review is CPU work plus a local vision
	// question. Drop the job's compute claims first so a local vision model can be
	// admitted while the review runs (the manager still owns and releases the
	// worker and disk claims).
	if err := releaseComputeClaims(ctx); err != nil {
		emit(92, "compute claims could not be released before the safety review; the manager will retry: "+err.Error())
	} else {
		emit(92, "released compute claims before the safety review")
	}
	release, err := e.holdReviewModels(ctx, jobID, e.reviewModelIDs(source.NSFWCapable), emit)
	if err != nil {
		report.action = "unavailable"
		report.code("weights_missing")
		report.detail(err.Error())
		return report, nil, errors.Join(fmt.Errorf("%w: %w", ErrSafetyUnavailable, err), e.deleteOutputs(ctx, keys))
	}
	releases := []func(){release}
	defer func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}()
	ctx = WithReviewSource(ctx, source)

	// Read every variation before the single review call, so one process/model
	// load covers the whole job and the reviewer can measure the batch together.
	images := make([][]byte, 0, len(keys))
	mimes := make([]string, 0, len(keys))
	for _, key := range keys {
		rc, mime, err := e.deps.Store.Get(ctx, key)
		if err != nil {
			report.action = "unavailable"
			report.code("reviewer_unavailable")
			report.detail(err.Error())
			return report, nil, errors.Join(fmt.Errorf("%w: ai: read output for safety review: %w", ErrSafetyUnavailable, err), e.deleteOutputs(ctx, keys))
		}
		// Bound the read: a provider that produced an oversized file must not turn
		// the mandatory review into an unbounded allocation.
		data, readErr := io.ReadAll(io.LimitReader(rc, maxReviewBytes+1))
		closeErr := rc.Close()
		if readErr != nil || closeErr != nil {
			report.action = "unavailable"
			report.code("reviewer_unavailable")
			report.detail(errors.Join(readErr, closeErr).Error())
			return report, nil, errors.Join(fmt.Errorf("%w: ai: read output for safety review: %w", ErrSafetyUnavailable, errors.Join(readErr, closeErr)), e.deleteOutputs(ctx, keys))
		}
		if len(data) > maxReviewBytes {
			err := fmt.Errorf("output exceeds the %d-byte review limit", maxReviewBytes)
			report.action = "unavailable"
			report.code("reviewer_unavailable")
			report.detail(err.Error())
			return report, nil, errors.Join(fmt.Errorf("%w: ai: %w", ErrSafetyUnavailable, err), e.deleteOutputs(ctx, keys))
		}
		images = append(images, data)
		mimes = append(mimes, mime)
	}

	batch, err := e.deps.ReviewOutput(ctx, images)
	var missing *ReviewWeightsMissing
	if errors.As(err, &missing) {
		// The classifier called an output sexual and the face/age weights are not
		// installed (or were offloaded): acquire exactly those, once.
		extra, holdErr := e.holdReviewModels(ctx, jobID, missing.IDs, emit)
		if holdErr != nil {
			report.action = "unavailable"
			report.code("weights_missing")
			report.detail(holdErr.Error())
			return report, nil, errors.Join(fmt.Errorf("%w: %w", ErrSafetyUnavailable, holdErr), e.deleteOutputs(ctx, keys))
		}
		releases = append(releases, extra)
		batch, err = e.deps.ReviewOutput(ctx, images)
	}
	if err != nil {
		report.action = "unavailable"
		if errors.As(err, &missing) {
			report.code("weights_missing")
		} else {
			report.code("reviewer_unavailable")
		}
		report.detail(err.Error())
		return report, nil, errors.Join(fmt.Errorf("%w: ai: required output review failed: %w", ErrSafetyUnavailable, err), e.deleteOutputs(ctx, keys))
	}
	if len(batch.Verdicts) != len(keys) {
		err := fmt.Errorf("reviewer answered for %d of %d outputs", len(batch.Verdicts), len(keys))
		report.action = "unavailable"
		report.code("reviewer_invalid_output")
		report.detail(err.Error())
		return report, nil, errors.Join(fmt.Errorf("%w: ai: %w", ErrSafetyUnavailable, err), e.deleteOutputs(ctx, keys))
	}
	report.evidence = batch.Evidence

	refusalReason := ""
	refusalCode := ""
	failJob := false
	for i, verdict := range batch.Verdicts {
		key := keys[i]
		action := verdict.ActionOrDerived()
		reasonCode := verdict.ReasonCodeOrDerived()
		report.code(reasonCode)
		report.maxScore = max(report.maxScore, verdict.Score)
		report.detail(verdict.Detail)
		switch action {
		case "refuse":
			if verdict.Shadow {
				// Shadow mode records the verdict and protects the UI with the
				// sensitive flag, but does not delete or fail. It is local-only.
				report.shadow = append(report.shadow, key)
				report.allowed = append(report.allowed, key)
				report.flag(key, "shadow")
				report.code("shadow_" + reasonCode)
				// Leave the case so the model-produced reason is also recorded; the
				// shadow verdict already protects the UI with the sensitive flag.
				break
			}
			if refusalReason == "" {
				refusalReason = verdict.HardRefusal
				if strings.TrimSpace(refusalReason) == "" {
					refusalReason = verdict.Detail
				}
				if strings.TrimSpace(refusalReason) == "" {
					refusalReason = "the reviewer refused this output"
				}
				refusalCode = reasonCode
			}
			report.refused = append(report.refused, key)
			if qRef, qErr := e.quarantineRefused(ctx, jobID, key, verdict, mimes[i], images[i]); qErr != nil {
				report.quarantineErrors = append(report.quarantineErrors, key+": "+qErr.Error())
				report.detail("quarantine failed for " + key + ": " + qErr.Error())
			} else if qRef != "" {
				report.quarantined = append(report.quarantined, qRef)
			}
			if verdict.DropVariation {
				if err := e.deleteOutputs(ctx, []string{key}); err != nil {
					report.action = "partial_refused"
					return report, report.allowed, errors.Join(err, e.deleteOutputs(ctx, keys))
				}
				continue
			}
			failJob = true
		case "flag":
			report.flag(key, "classifier")
			report.allowed = append(report.allowed, key)
		default:
			report.allowed = append(report.allowed, key)
		}
		if source.NSFWCapable {
			report.flag(key, "model")
		}
	}

	if failJob {
		report.action = "refuse"
		report.allowed = nil
		return report, nil, errors.Join(&HardRefusalError{Reason: refusalReason, ReasonCode: refusalCode}, e.deleteOutputs(ctx, keys))
	}
	if len(report.refused) > 0 && len(report.allowed) == 0 {
		report.action = "refuse"
		return report, nil, errors.Join(&HardRefusalError{Reason: refusalReason, ReasonCode: refusalCode}, e.deleteOutputs(ctx, keys))
	}
	if len(report.refused) > 0 {
		report.action = "partial_refused"
		return report, report.allowed, nil
	}
	if len(report.shadow) > 0 {
		report.action = "shadow_refused"
		return report, report.allowed, nil
	}
	if len(report.sensitive) > 0 {
		report.action = "flag"
		return report, report.allowed, nil
	}
	report.action = "allow"
	return report, report.allowed, nil
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

// quarantineRefused copies one refused output into the deployment's bounded
// quarantine. A nil sink means the deployment chose immediate deletion. A
// quarantine failure does not turn a refusal into a pass: the caller still
// deletes the original, and the error is recorded on the job.
func (e *Engine) quarantineRefused(ctx context.Context, jobID, key string, verdict ScanVerdict, mime string, data []byte) (string, error) {
	if e.deps.Quarantine == nil {
		return "", nil
	}
	return e.deps.Quarantine(ctx, QuarantineItem{
		JobID:        jobID,
		OriginalRef:  key,
		ReasonCode:   verdict.ReasonCodeOrDerived(),
		SafetyReview: "refused",
		Score:        verdict.Score,
		Evidence:     verdict.Evidence,
		Finding:      verdict.Finding,
		MIME:         mime,
	}, data)
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
