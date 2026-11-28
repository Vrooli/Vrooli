package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	internaladapters "image-tools/internal/adapters"
	internalbackends "image-tools/internal/backends"
	internalcaps "image-tools/internal/capabilities"
	internalcapability "image-tools/internal/capability"
	"image-tools/internal/catalogstate"
	internalhfmeta "image-tools/internal/hfmeta"
	internalhosttool "image-tools/internal/hosttool"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/verdicts"

	"connectrpc.com/connect"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// BackendEnsurer ensures a host-tool backend is installed. main.go passes the
// real hosttool.Ensurer; nil in read-only wiring disables EnsureBackend.
type BackendEnsurer interface {
	Inspect(ctx context.Context, tool string) (*internalhosttool.Status, error)
}

// JobSubmitter is the narrow slice of the durable job manager the models handler
// needs to launch a download. main.go passes the real *jobs.Manager.
type JobSubmitter interface {
	Submit(ctx context.Context, spec internaljobs.Spec) (internaljobs.Job, error)
}

// EstimateInstallSecondsFunc estimates model-install ETA from catalog size.
type EstimateInstallSecondsFunc func(sizeMBApprox int) int

// Deps wires the seams the Connect models handler needs.
type Deps struct {
	// CapacityManagedGPU is the same GPU-fit rule the AI submit path uses: true
	// when the host capacity broker arbitrates GPU memory, so a model fits by
	// the GPU's total VRAM and transient contention is decided at admission.
	// Previews (`models select`, `ai --explain`, the picker) must set it from the
	// AI engine (internalai.Engine.CapacityManagedGPU) so they judge the GPU the
	// way a real job does.
	CapacityManagedGPU bool
	// Registry is the loaded, validated seed catalog.
	Registry *internalmodels.Registry
	// Store persists the runtime enabled-state overrides with their provenance.
	Store *internalmodels.Store
	// Verdicts derives each model's per-operation verdict from the evidence seed,
	// job history and operator ratings. Nil leaves every model untested.
	Verdicts *verdicts.Service
	// Previewer answers "which model would run" for SelectModel; the AI discovery
	// handler asks the same previewer, so `models select` and `ai list` cannot
	// disagree. Defaults to one assembled from this Deps.
	Previewer *internalmodels.Previewer
	// Now stamps override provenance. Defaults to time.Now.
	Now func() time.Time
	// Probe reports host hardware facts for SelectModel's hardware-fit preview.
	Probe internalcaps.Probe
	// Installer manages on-disk weights (install/remove/custom) and install state.
	Installer *internalmodels.Installer
	// Backends is the runtime provider registry used by AI selection.
	Backends *internalbackends.Registry
	// RightsEvaluator is the shared current-context evaluator used by model
	// selection previews. A nil value remains fail-closed, matching execution.
	RightsEvaluator internalmodels.RightsEvaluator
	// Jobs submits the durable download job (may be nil in read-only wiring).
	Jobs JobSubmitter
	// OpDefaults persists per-operation default-model pins (settings surface).
	OpDefaults *internalmodels.OpDefaultStore
	// EstimateInstallSeconds estimates model-download ETA for submitted jobs.
	// Defaults to models.EstimateInstallSeconds.
	EstimateInstallSeconds EstimateInstallSecondsFunc
	// Ensurer probes/installs host-tool backends for EnsureBackend + doctor
	// readiness (may be nil in read-only wiring).
	Ensurer BackendEnsurer
	// EnsureEtaSeconds is the initial ETA reported for a backend-ensure job.
	// Defaults to a conservative fixed estimate (host-tool sizes are not in the
	// model catalog).
	EnsureEtaSeconds int
	// Hfmeta inspects an import source (HF repo id / URL / local path) for the
	// guided import flow (InspectModelSource / ImportModel). Defaults to a real
	// hfmeta.HFClient; tests inject a fake.
	Hfmeta internalhfmeta.Fetcher
	// Adapter conditioning seams for ExplainResolution's --explain surface: a merged
	// (seed + custom) catalog lookup, the enabled overlay, and the install check, so
	// a dry-run can validate + report the requested conditioning stack (decision
	// C5). All nil ⇒ ExplainResolution rejects any request that names an adapter.
	AdapterByID      func(id string) (internaladapters.Adapter, bool)
	AdapterEnabled   func(ctx context.Context) (func(id string) bool, error)
	AdapterInstalled func(id string) bool
	// Storage is the model-weight storage lifecycle owner (status, offload,
	// keep pins). nil leaves the storage RPCs unimplemented.
	Storage ModelStorage
	// Consent is the deployment's consent rule (the gate's, shared with the submit
	// edge). ExplainResolution reports through it whether a request resolved to
	// this route needs a consent affirmation, so the UI shows its checkbox exactly
	// when the submit edge would ask. nil reports "not required".
	Consent ConsentRule
	Logger  *log.Logger
}

// ConsentRule reports whether an operation at consent weight (none|low|high),
// resolved to a route tier, needs a consent affirmation: which rule asks
// ("tier" or "route") and a plain-language reason.
type ConsentRule func(op, weight, tier string) (required bool, basis, reason string)

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the ModelsService handler. Registry, Store, and Probe
// are required (a nil dep is a wiring bug surfaced as an internal error).
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.EstimateInstallSeconds == nil {
		d.EstimateInstallSeconds = internalmodels.EstimateInstallSeconds
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.EnsureEtaSeconds <= 0 {
		d.EnsureEtaSeconds = 90
	}
	if d.Hfmeta == nil {
		d.Hfmeta = &internalhfmeta.HFClient{}
	}
	return &connectHandler{deps: d}
}

// previewer is the shared previewer main.go wired, or one assembled from the
// handler's current seams (read at use, so a test that adjusts a seam after
// construction is previewed against it).
func (h *connectHandler) previewer() *internalmodels.Previewer {
	if h.deps.Previewer != nil {
		return h.deps.Previewer
	}
	return h.defaultPreviewer()
}

// defaultPreviewer assembles the shared previewer from this handler's own seams
// (main.go wires one explicitly so the AI discovery handler shares it).
func (h *connectHandler) defaultPreviewer() *internalmodels.Previewer {
	d := h.deps
	p := &internalmodels.Previewer{
		Registry:           d.Registry,
		Probe:              d.Probe,
		Overlay:            h.overlay,
		Customs:            h.customModels,
		RightsEvaluator:    d.RightsEvaluator,
		CapacityManagedGPU: d.CapacityManagedGPU,
	}
	if d.OpDefaults != nil {
		p.Pin = d.OpDefaults.Get
	}
	return p
}

func (h *connectHandler) overlay(ctx context.Context) (map[string]bool, error) {
	if h.deps.Store == nil {
		return nil, nil
	}
	return h.deps.Store.LoadOverlay(ctx)
}

// installStates returns the install record per model id (empty when no Installer).
// Weight installs are host-wide artifacts, not user data: a routed test DB must
// not hide a real installed model and send an isolated UI into a false install
// gate. Read this inventory from the host's primary catalog; mutations remain
// owned by the installer lifecycle and job admission.
func (h *connectHandler) installStates(_ context.Context) (map[string]internalmodels.InstallRecord, error) {
	out := map[string]internalmodels.InstallRecord{}
	if h.deps.Installer == nil || h.deps.Installer.State == nil {
		return out, nil
	}
	recs, err := h.deps.Installer.State.List(context.Background())
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		out[r.ID] = r
	}
	return out, nil
}

func (h *connectHandler) publicInstallState(m internalmodels.Model, record internalmodels.InstallRecord, found bool) (internalmodels.InstallRecord, bool) {
	if !found || m.BuildProfile == nil || h.deps.Installer == nil {
		return record, found
	}
	record.Installed = h.deps.Installer.BuildReadyForRecord(m, record)
	return record, true
}

// customModels returns operator-registered entries keyed by id (nil-safe).
func (h *connectHandler) customModels(ctx context.Context) ([]internalmodels.Model, error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, nil
	}
	return h.deps.Installer.Custom.List(ctx)
}

// effectiveRegistry gives read-side model selectors the same seed+custom
// catalog that the AI planner uses. Keeping the id set alongside the merged
// registry preserves custom provenance in picker responses.
func (h *connectHandler) effectiveRegistry(ctx context.Context) (*internalmodels.Registry, map[string]bool, error) {
	customs, err := h.customModels(ctx)
	if err != nil {
		return nil, nil, err
	}
	registry, err := h.deps.Registry.WithCustom(customs)
	if err != nil {
		return nil, nil, err
	}
	customIDs := make(map[string]bool, len(customs))
	for _, model := range customs {
		customIDs[model.ID] = true
	}
	return registry, customIDs, nil
}

// viewFor builds the runtime modelView for one model from the request's
// viewState: effective enabled state, override provenance, install record and
// verdicts.
func (h *connectHandler) viewFor(m internalmodels.Model, custom bool, st *viewState) modelView {
	v := modelView{custom: custom, enabled: internalmodels.EffectiveEnabled(m, st.overlay)}
	if o, ok := st.overrides[m.ID]; ok {
		v.override = overrideToProto(&o, m.Enabled)
	}
	r, ok := st.installs[m.ID]
	if !ok {
		r, ok = internalmodels.WeightlessInstallRecord(m)
	}
	if ok {
		rec, _ := h.publicInstallState(m, r, true)
		v.install = &rec
	}
	per, rollup := st.modelVerdicts(m)
	v.verdict = verdicts.ToProto(rollup)
	for _, p := range per {
		v.verdicts = append(v.verdicts, verdicts.ToProto(p))
	}
	return v
}

func (h *connectHandler) ListModels(ctx context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	st, err := h.loadViewState(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListModels state: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}

	op := req.Msg.GetOperation()
	var entries []internalmodels.Model
	if op != "" {
		if !h.deps.Registry.IsOperation(op) {
			return nil, connect.NewError(connect.CodeInvalidArgument, internalmodels.ErrUnknownOperation)
		}
		entries = h.deps.Registry.ForOperation(op)
	} else {
		entries = h.deps.Registry.Models()
	}

	resp := &modelsv1.ListModelsResponse{Models: make([]*modelsv1.Model, 0, len(entries))}
	for _, m := range entries {
		resp.Models = append(resp.Models, domainToProto(m, h.viewFor(m, false, st)))
	}

	// Merge operator-registered custom entries (filtered to the op when set).
	customs, err := h.customModels(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListModels custom: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load custom models"))
	}
	for _, m := range customs {
		if op != "" && !m.ServesOperation(op) {
			continue
		}
		resp.Models = append(resp.Models, domainToProto(m, h.viewFor(m, true, st)))
	}
	return connect.NewResponse(resp), nil
}

// resolve finds a model by id in the seed or custom store, reporting whether it
// is custom.
func (h *connectHandler) resolve(ctx context.Context, id string) (internalmodels.Model, bool, bool) {
	if m, ok := h.deps.Registry.ByID(id); ok {
		return m, false, true
	}
	customs, err := h.customModels(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.resolve custom: %v", err)
		return internalmodels.Model{}, false, false
	}
	for _, m := range customs {
		if m.ID == id {
			return m, true, true
		}
	}
	return internalmodels.Model{}, false, false
}

func (h *connectHandler) GetModel(ctx context.Context, req *connect.Request[modelsv1.GetModelRequest]) (*connect.Response[modelsv1.GetModelResponse], error) {
	m, custom, ok := h.resolve(ctx, req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	st, err := h.loadViewState(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.GetModel state: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	return connect.NewResponse(&modelsv1.GetModelResponse{
		Model: domainToProto(m, h.viewFor(m, custom, st)),
	}), nil
}

func (h *connectHandler) ListOperations(_ context.Context, _ *connect.Request[modelsv1.ListOperationsRequest]) (*connect.Response[modelsv1.ListOperationsResponse], error) {
	return connect.NewResponse(&modelsv1.ListOperationsResponse{
		Operations: h.deps.Registry.Operations(),
	}), nil
}

func (h *connectHandler) SelectModel(ctx context.Context, req *connect.Request[modelsv1.SelectModelRequest]) (*connect.Response[modelsv1.SelectModelResponse], error) {
	// The shared previewer merges the catalog, the enabled overlay, the operator's
	// pin, rights and hardware fit exactly as a submit does, and is the same code
	// `ai list` asks for each operation's default, so the two cannot disagree.
	preview, err := h.previewer().Load(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	st, err := h.loadViewState(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel state: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	op := req.Msg.GetOperation()
	selectOp := op
	override := req.Msg.GetOverrideId()
	if runs, presetModel, _, ok := internalcapability.PresetTarget(op); ok {
		selectOp = runs
		if override == "" {
			override = presetModel
		}
	}
	sel, err := preview.Select(ctx, internalmodels.PreviewRequest{
		Operation: selectOp,
		// The caller's routing policy, so the preview resolves the model a
		// submit with the same policy would resolve. Without it a BYOK-routed
		// caller was previewed the local default's identity and geometry, and
		// sized its request from a model that was never going to draw it.
		QualityPolicy: req.Msg.GetQualityPolicy(),
		AllowBYOK:     req.Msg.GetAllowByok(),
		OverrideID:    override,
		ForceCPU:      req.Msg.GetForceCpu(),
	})
	if err != nil {
		return nil, selectError(err)
	}

	return connect.NewResponse(&modelsv1.SelectModelResponse{
		Model:     domainToProto(sel.Model, h.viewFor(sel.Model, preview.CustomIDs[sel.Model.ID], st)),
		GpuViable: sel.GPUViable,
		Reason:    sel.Reason,
		Warnings:  sel.Warnings,
		Verdict:   verdicts.ToProto(st.verdictFor(sel.Model, selectOp)),
	}), nil
}

func (h *connectHandler) SetModelEnabled(ctx context.Context, req *connect.Request[modelsv1.SetModelEnabledRequest]) (*connect.Response[modelsv1.SetModelEnabledResponse], error) {
	id := req.Msg.GetId()
	m, custom, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if req.Msg.GetEnabled() {
		if err := internalmodels.RequireLocalUsePermitted(ctx, h.deps.RightsEvaluator, m); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("model cannot be enabled: %w", err))
		}
	}
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("model state store unavailable"))
	}
	actor := strings.TrimSpace(req.Msg.GetActor())
	if actor == "" {
		actor = "api"
	}
	reason := strings.TrimSpace(req.Msg.GetReason())
	if req.Msg.GetPinned() && reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a pinned override is a standing choice: say why with a reason"))
	}
	override := catalogstate.Override{ID: id, Enabled: req.Msg.GetEnabled(), SetBy: actor, Reason: reason, Pinned: req.Msg.GetPinned()}
	if err := h.deps.Store.SetOverride(ctx, override, m.Enabled, h.deps.Now()); err != nil {
		h.deps.Logger.Printf("models.SetModelEnabled: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist model state"))
	}
	return modelResponse(h, ctx, m, custom, "SetModelEnabled", func(m *modelsv1.Model) *modelsv1.SetModelEnabledResponse {
		return &modelsv1.SetModelEnabledResponse{Model: m}
	})
}

// ClearModelOverride deletes an override so the model follows the seed again.
func (h *connectHandler) ClearModelOverride(ctx context.Context, req *connect.Request[modelsv1.ClearModelOverrideRequest]) (*connect.Response[modelsv1.ClearModelOverrideResponse], error) {
	id := req.Msg.GetId()
	m, custom, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("model state store unavailable"))
	}
	existed, err := h.deps.Store.ClearOverride(ctx, id)
	if err != nil {
		h.deps.Logger.Printf("models.ClearModelOverride: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist model state"))
	}
	if !existed {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("model %q has no override: it already follows the seed", id))
	}
	return modelResponse(h, ctx, m, custom, "ClearModelOverride", func(m *modelsv1.Model) *modelsv1.ClearModelOverrideResponse {
		return &modelsv1.ClearModelOverrideResponse{Model: m}
	})
}

// modelResponse re-reads the catalog state after a write and wraps the model's
// fresh view, so the response shows the state the next read will.
func modelResponse[T any](h *connectHandler, ctx context.Context, m internalmodels.Model, custom bool, op string, wrap func(*modelsv1.Model) *T) (*connect.Response[T], error) {
	st, err := h.loadViewState(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.%s state: %v", op, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	return connect.NewResponse(wrap(domainToProto(m, h.viewFor(m, custom, st)))), nil
}

func (h *connectHandler) ListBlocklist(_ context.Context, _ *connect.Request[modelsv1.ListBlocklistRequest]) (*connect.Response[modelsv1.ListBlocklistResponse], error) {
	entries := h.deps.Registry.Blocklist()
	resp := &modelsv1.ListBlocklistResponse{Entries: make([]*modelsv1.BlocklistEntry, 0, len(entries))}
	for _, b := range entries {
		resp.Entries = append(resp.Entries, blocklistToProto(b))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) InstallModel(ctx context.Context, req *connect.Request[modelsv1.InstallModelRequest]) (*connect.Response[modelsv1.InstallModelResponse], error) {
	if h.deps.Installer == nil || h.deps.Jobs == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model installation unavailable"))
	}
	id := req.Msg.GetId()
	m, _, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if err := internalmodels.RequireLocalUsePermitted(ctx, h.deps.RightsEvaluator, m); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("model cannot be installed: %w", err))
	}

	lifecycle, err := h.deps.Installer.Lifecycle(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read model lifecycle: %w", err))
	}
	if lifecycle.Ready() {
		return connect.NewResponse(&modelsv1.InstallModelResponse{
			AlreadyInstalled: true,
			SizeMbApprox:     int32(m.SizeMBApprox),
		}), nil
	}
	// An absent model or an interrupted install can retry through the normal
	// durable path. Installed artifacts with stale/failed smoke and incomplete
	// native exports use that same path to revalidate without reacquisition.
	switch lifecycle.Reason {
	case "needs_model_install", "install_failed", "smoke_pending", "smoke_failed", "build_not_ready":
	default:
		message := fmt.Sprintf("model %q is not installable from lifecycle state %s (%s)", id, lifecycle.State, lifecycle.Reason)
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(message))
	}

	payload, err := json.Marshal(internalmodels.InstallPayload{ModelID: id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("encode install request"))
	}
	eta := h.deps.EstimateInstallSeconds(m.SizeMBApprox)
	job, err := h.deps.Jobs.Submit(ctx, internaljobs.Spec{
		Operation:        internalmodels.InstallJobOperation,
		Lane:             internaljobs.LaneCPU,
		Payload:          payload,
		EstimatedSeconds: eta,
	})
	if err != nil {
		h.deps.Logger.Printf("models.InstallModel submit: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("submit install job"))
	}
	return connect.NewResponse(&modelsv1.InstallModelResponse{
		JobId:        job.ID,
		EtaSeconds:   int32(eta),
		SizeMbApprox: int32(m.SizeMBApprox),
	}), nil
}

func (h *connectHandler) RemoveModel(ctx context.Context, req *connect.Request[modelsv1.RemoveModelRequest]) (*connect.Response[modelsv1.RemoveModelResponse], error) {
	if h.deps.Installer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model management unavailable"))
	}
	id := req.Msg.GetId()
	if _, _, ok := h.resolve(ctx, id); !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if err := h.deps.Installer.Remove(ctx, id); err != nil {
		h.deps.Logger.Printf("models.RemoveModel: %v", err)
		if errors.Is(err, internalmodels.ErrModelInUse) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("model is in use or being changed; retry after the current operation completes"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New("remove model"))
	}
	return connect.NewResponse(&modelsv1.RemoveModelResponse{Removed: true}), nil
}

func (h *connectHandler) AddCustomModel(ctx context.Context, req *connect.Request[modelsv1.AddCustomModelRequest]) (*connect.Response[modelsv1.AddCustomModelResponse], error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("custom models unavailable"))
	}
	pm := req.Msg.GetModel()
	if pm == nil || pm.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("custom model id is required"))
	}
	m := protoToDomain(pm)
	m.Source.LocalPath = req.Msg.GetLocalPath()
	m.Source.DownloadURL = req.Msg.GetDownloadUrl()
	// Complete what the caller left out so the response shows what was stored
	// (AddCustom completes idempotently).
	m = internalmodels.CompleteCustom(m)

	if err := h.deps.Installer.AddCustom(ctx, m); err != nil {
		return nil, addCustomError(err)
	}
	st, err := h.loadViewState(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.AddCustomModel state: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	return connect.NewResponse(&modelsv1.AddCustomModelResponse{
		Model: domainToProto(m, h.viewFor(m, true, st)),
	}), nil
}

func (h *connectHandler) SetDefaultModel(ctx context.Context, req *connect.Request[modelsv1.SetDefaultModelRequest]) (*connect.Response[modelsv1.SetDefaultModelResponse], error) {
	if h.deps.OpDefaults == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("settings unavailable"))
	}
	op := req.Msg.GetOperation()
	if !h.deps.Registry.IsOperation(op) {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalmodels.ErrUnknownOperation)
	}
	id := req.Msg.GetModelId()
	if id != "" {
		m, _, ok := h.resolve(ctx, id)
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
		}
		if !m.ServesOperation(op) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model does not serve that operation"))
		}
		if err := internalmodels.RequireLocalUsePermitted(ctx, h.deps.RightsEvaluator, m); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("model cannot be pinned: %w", err))
		}
	}
	if err := h.deps.OpDefaults.Set(ctx, op, id); err != nil {
		h.deps.Logger.Printf("models.SetDefaultModel: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist default"))
	}
	return connect.NewResponse(&modelsv1.SetDefaultModelResponse{Operation: op, ModelId: id}), nil
}

func (h *connectHandler) ListDefaults(ctx context.Context, _ *connect.Request[modelsv1.ListDefaultsRequest]) (*connect.Response[modelsv1.ListDefaultsResponse], error) {
	pins := map[string]string{}
	if h.deps.OpDefaults != nil {
		loaded, err := h.deps.OpDefaults.Load(ctx)
		if err != nil {
			h.deps.Logger.Printf("models.ListDefaults: %v", err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("load defaults"))
		}
		pins = loaded
	}
	resp := &modelsv1.ListDefaultsResponse{}
	for _, op := range h.deps.Registry.Operations() {
		d := &modelsv1.OpDefault{Operation: op, Source: "seed"}
		if pinned, ok := pins[op]; ok && pinned != "" {
			d.ModelId = pinned
			d.Source = "override"
		} else if seed, ok := h.deps.Registry.DefaultFor(op); ok {
			d.ModelId = seed.ID
		}
		resp.Defaults = append(resp.Defaults, d)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DoctorCatalog(ctx context.Context, _ *connect.Request[modelsv1.DoctorCatalogRequest]) (*connect.Response[modelsv1.DoctorCatalogResponse], error) {
	report := h.deps.Registry.DoctorCatalog()
	if h.deps.Store != nil {
		overrides, err := h.deps.Store.LoadOverrides(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load model overrides: %w", err))
		}
		custom, err := h.customModels(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		report.Findings = append(report.Findings, internalmodels.DoctorOverrides(overrides, h.deps.Registry.SeedState(custom))...)
	}
	if h.deps.Installer == nil {
		report.Findings = append(report.Findings, internalmodels.CatalogFinding{
			Severity: internalmodels.FindingError,
			Code:     "model_lifecycle_unavailable",
			Message:  "model lifecycle evidence is unavailable; runtime readiness cannot be established",
		})
		report.OK = false
	}
	// Fold in the runtime "enabled ⇒ runnable" findings (env provisioned + smoke
	// passed) that the static catalog check cannot see, so a model that is
	// installed-but-broken or enabled-without-a-ready-venv surfaces here.
	if h.deps.Installer != nil {
		overlay, err := h.overlay(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		catalog := h.deps.Registry.Models()
		custom, err := h.customModels(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		runtime, err := h.deps.Installer.DoctorRuntimeReport(ctx, catalog, overlay)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		report.Findings = append(report.Findings, runtime.Findings...)
		report.Models = append(report.Models, runtime.Models...)
		if !runtime.OK {
			report.OK = false
		}
		if h.deps.Probe == nil {
			report.Findings = append(report.Findings, internalmodels.CatalogFinding{
				Severity: internalmodels.FindingError,
				Code:     "host_inventory_unavailable",
				Message:  "host inventory is unavailable; enabled-capability provisioning readiness cannot be established",
			})
			report.OK = false
		} else {
			plan, err := h.hostFitProvisionPlan(ctx, overlay)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build enabled-capability readiness plan: %w", err))
			}
			if len(plan.CoveredOperations)+len(plan.UncoveredOperations) == 0 {
				report.Findings = append(report.Findings, internalmodels.CatalogFinding{
					Severity: internalmodels.FindingError,
					Code:     "enabled_capability_scope_empty",
					Message:  "no enabled capability operations are in scope; readiness cannot be established",
				})
				report.OK = false
			}
			for _, gap := range plan.UncoveredOperations {
				report.Findings = append(report.Findings, internalmodels.CatalogFinding{
					Severity:  internalmodels.FindingError,
					Code:      "enabled_capability_not_provisionable",
					Operation: gap.Operation,
					Message:   fmt.Sprintf("enabled capability %s has no host-fit, commercial-clean local model: %s", gap.Operation, gap.Reason),
				})
				report.OK = false
			}
		}
		for _, m := range custom {
			state, err := h.deps.Installer.Lifecycle(ctx, m.ID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			report.Models = append(report.Models, state)
		}
	}
	return connect.NewResponse(doctorReportToProto(report)), nil
}

func (h *connectHandler) DoctorBackends(ctx context.Context, _ *connect.Request[modelsv1.DoctorBackendsRequest]) (*connect.Response[modelsv1.DoctorBackendsResponse], error) {
	if h.deps.Backends == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("backend registry is not configured"))
	}
	return connect.NewResponse(backendReportToProto(h.deps.Backends.DoctorForModels(ctx, h.deps.Registry.Models()))), nil
}

// EnsureBackend installs a missing host-tool backend on demand. It first probes
// (dry-run) to short-circuit already-present, manual, and not-applicable tools
// with guidance and no job; otherwise it submits a durable job (mirroring
// InstallModel) that shells `vrooli host install <tool> --json` and survives
// client cancellation.
func (h *connectHandler) EnsureBackend(ctx context.Context, req *connect.Request[modelsv1.EnsureBackendRequest]) (*connect.Response[modelsv1.EnsureBackendResponse], error) {
	if h.deps.Ensurer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("backend provisioning unavailable"))
	}
	tool := strings.TrimSpace(req.Msg.GetTool())
	if tool == "" {
		if op := strings.TrimSpace(req.Msg.GetOperation()); op != "" {
			if resolved, ok := internalhosttool.ToolForOperation(op); ok {
				tool = resolved
			} else {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation %q has no host-tool backend", op))
			}
		}
	}
	if tool == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tool or operation is required"))
	}
	if !internalhosttool.KnownTool(tool) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown backend host tool %q", tool))
	}

	status, err := h.deps.Ensurer.Inspect(ctx, tool)
	if err != nil {
		h.deps.Logger.Printf("models.EnsureBackend inspect %q: %v", tool, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host tool"))
	}
	state := status.GetExecutionState()
	detail := strings.TrimSpace(strings.Join(status.GetNotes(), "; "))

	resp := &modelsv1.EnsureBackendResponse{Tool: tool, State: state, Detail: detail}
	switch state {
	case "already_present":
		resp.AlreadyInstalled = true
		return connect.NewResponse(resp), nil
	case "manual_action_required", "unsupported", "not_applicable":
		// No auto-fetch path; return guidance, no job.
		resp.Manual = true
		return connect.NewResponse(resp), nil
	}

	// would_install / pending → fetchable.
	if req.Msg.GetDryRun() {
		return connect.NewResponse(resp), nil
	}
	if h.deps.Jobs == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("durable jobs unavailable"))
	}
	payload, err := json.Marshal(internalhosttool.EnsurePayload{Tool: tool})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("encode ensure request"))
	}
	job, err := h.deps.Jobs.Submit(ctx, internaljobs.Spec{
		Operation:        internalhosttool.EnsureJobOperation,
		Lane:             internaljobs.LaneCPU,
		Payload:          payload,
		EstimatedSeconds: h.deps.EnsureEtaSeconds,
	})
	if err != nil {
		h.deps.Logger.Printf("models.EnsureBackend submit %q: %v", tool, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("submit ensure job"))
	}
	resp.JobId = job.ID
	resp.EtaSeconds = int32(h.deps.EnsureEtaSeconds)
	return connect.NewResponse(resp), nil
}

// selectError maps selector errors to actionable Connect codes.
func selectError(err error) error {
	switch {
	case errors.Is(err, internalmodels.ErrUnknownOperation),
		errors.Is(err, internalmodels.ErrOverrideInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalmodels.ErrNoEnabledModel),
		errors.Is(err, internalmodels.ErrNotRunnable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// addCustomError maps custom-registration errors to actionable Connect codes.
func addCustomError(err error) error {
	switch {
	case errors.Is(err, internalmodels.ErrCustomShadowsSeed),
		errors.Is(err, internalmodels.ErrLocalPathMissing):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
