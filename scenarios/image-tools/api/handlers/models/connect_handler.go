package models

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	internalbackends "image-tools/internal/backends"
	internalcaps "image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"

	"connectrpc.com/connect"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// JobSubmitter is the narrow slice of the durable job manager the models handler
// needs to launch a download. main.go passes the real *jobs.Manager.
type JobSubmitter interface {
	Submit(ctx context.Context, spec internaljobs.Spec) (internaljobs.Job, error)
}

// EstimateInstallSecondsFunc estimates model-install ETA from catalog size.
type EstimateInstallSecondsFunc func(sizeMBApprox int) int

// Deps wires the seams the Connect models handler needs.
type Deps struct {
	// Registry is the loaded, validated seed catalog.
	Registry *internalmodels.Registry
	// Store persists the runtime enabled-state overlay.
	Store *internalmodels.Store
	// Probe reports host hardware facts for SelectModel's hardware-fit preview.
	Probe internalcaps.Probe
	// Installer manages on-disk weights (install/remove/custom) and install state.
	Installer *internalmodels.Installer
	// Backends is the runtime provider registry used by AI selection.
	Backends *internalbackends.Registry
	// Jobs submits the durable download job (may be nil in read-only wiring).
	Jobs JobSubmitter
	// OpDefaults persists per-operation default-model pins (settings surface).
	OpDefaults *internalmodels.OpDefaultStore
	// EstimateInstallSeconds estimates model-download ETA for submitted jobs.
	// Defaults to models.EstimateInstallSeconds.
	EstimateInstallSeconds EstimateInstallSecondsFunc
	Logger                 *log.Logger
}

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
	return &connectHandler{deps: d}
}

func (h *connectHandler) overlay(ctx context.Context) (map[string]bool, error) {
	if h.deps.Store == nil {
		return nil, nil
	}
	return h.deps.Store.LoadOverlay(ctx)
}

// installStates returns the install record per model id (empty when no Installer).
func (h *connectHandler) installStates(ctx context.Context) (map[string]internalmodels.InstallRecord, error) {
	out := map[string]internalmodels.InstallRecord{}
	if h.deps.Installer == nil || h.deps.Installer.State == nil {
		return out, nil
	}
	recs, err := h.deps.Installer.State.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		out[r.ID] = r
	}
	return out, nil
}

// customModels returns operator-registered entries keyed by id (nil-safe).
func (h *connectHandler) customModels(ctx context.Context) ([]internalmodels.Model, error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, nil
	}
	return h.deps.Installer.Custom.List(ctx)
}

// viewFor builds the runtime modelView for one model.
func viewFor(m internalmodels.Model, custom bool, overlay map[string]bool, installs map[string]internalmodels.InstallRecord) modelView {
	v := modelView{custom: custom, enabled: internalmodels.EffectiveEnabled(m, overlay)}
	if r, ok := installs[m.ID]; ok {
		rec := r
		v.install = &rec
	}
	return v
}

func (h *connectHandler) ListModels(ctx context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListModels overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListModels installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
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
		resp.Models = append(resp.Models, domainToProto(m, viewFor(m, false, overlay, installs)))
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
		resp.Models = append(resp.Models, domainToProto(m, viewFor(m, true, overlay, installs)))
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
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.GetModel overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.GetModel installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}
	return connect.NewResponse(&modelsv1.GetModelResponse{
		Model: domainToProto(m, viewFor(m, custom, overlay, installs)),
	}), nil
}

func (h *connectHandler) ListOperations(_ context.Context, _ *connect.Request[modelsv1.ListOperationsRequest]) (*connect.Response[modelsv1.ListOperationsResponse], error) {
	return connect.NewResponse(&modelsv1.ListOperationsResponse{
		Operations: h.deps.Registry.Operations(),
	}), nil
}

func (h *connectHandler) SelectModel(ctx context.Context, req *connect.Request[modelsv1.SelectModelRequest]) (*connect.Response[modelsv1.SelectModelResponse], error) {
	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel probe: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host hardware"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}

	override := req.Msg.GetOverrideId()
	if override == "" && h.deps.OpDefaults != nil {
		if pinned, derr := h.deps.OpDefaults.Get(ctx, req.Msg.GetOperation()); derr == nil {
			override = pinned
		}
	}
	sel, err := h.deps.Registry.Select(internalmodels.SelectRequest{
		Operation:  req.Msg.GetOperation(),
		Host:       host,
		OverrideID: override,
	}, h.deps.Registry.EnabledWithOverlay(overlay))
	if err != nil {
		return nil, selectError(err)
	}

	return connect.NewResponse(&modelsv1.SelectModelResponse{
		Model:     domainToProto(sel.Model, viewFor(sel.Model, false, overlay, installs)),
		GpuViable: sel.GPUViable,
		Reason:    sel.Reason,
		Warnings:  sel.Warnings,
	}), nil
}

func (h *connectHandler) SetModelEnabled(ctx context.Context, req *connect.Request[modelsv1.SetModelEnabledRequest]) (*connect.Response[modelsv1.SetModelEnabledResponse], error) {
	id := req.Msg.GetId()
	m, custom, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("model state store unavailable"))
	}
	if err := h.deps.Store.SetEnabled(ctx, id, req.Msg.GetEnabled()); err != nil {
		h.deps.Logger.Printf("models.SetModelEnabled: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist model state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SetModelEnabled installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}
	v := viewFor(m, custom, nil, installs)
	v.enabled = req.Msg.GetEnabled()
	return connect.NewResponse(&modelsv1.SetModelEnabledResponse{
		Model: domainToProto(m, v),
	}), nil
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

	if h.deps.Installer.Installed(ctx, id) {
		return connect.NewResponse(&modelsv1.InstallModelResponse{
			AlreadyInstalled: true,
			SizeMbApprox:     int32(m.SizeMBApprox),
		}), nil
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

	if err := h.deps.Installer.AddCustom(ctx, m); err != nil {
		return nil, addCustomError(err)
	}
	overlay, _ := h.overlay(ctx)
	installs, _ := h.installStates(ctx)
	return connect.NewResponse(&modelsv1.AddCustomModelResponse{
		Model: domainToProto(m, viewFor(m, true, overlay, installs)),
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

func (h *connectHandler) DoctorCatalog(_ context.Context, _ *connect.Request[modelsv1.DoctorCatalogRequest]) (*connect.Response[modelsv1.DoctorCatalogResponse], error) {
	return connect.NewResponse(doctorReportToProto(h.deps.Registry.DoctorCatalog())), nil
}

func (h *connectHandler) DoctorBackends(ctx context.Context, _ *connect.Request[modelsv1.DoctorBackendsRequest]) (*connect.Response[modelsv1.DoctorBackendsResponse], error) {
	if h.deps.Backends == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("backend registry is not configured"))
	}
	return connect.NewResponse(backendReportToProto(h.deps.Backends.DoctorForModels(ctx, h.deps.Registry.Models()))), nil
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
