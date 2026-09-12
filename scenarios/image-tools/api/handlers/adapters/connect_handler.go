package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	internaladapters "image-tools/internal/adapters"
	internalhfmeta "image-tools/internal/hfmeta"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"

	"connectrpc.com/connect"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
)

// JobSubmitter is the narrow slice of the durable job manager the adapters
// handler needs to launch a download. main.go passes the real *jobs.Manager.
type JobSubmitter interface {
	Submit(ctx context.Context, spec internaljobs.Spec) (internaljobs.Job, error)
}

// EstimateInstallSecondsFunc estimates an adapter-install ETA from catalog size.
type EstimateInstallSecondsFunc func(sizeMBApprox int) int

// Deps wires the seams the Connect adapters handler needs.
type Deps struct {
	// Registry is the loaded, validated seed catalog.
	Registry *internaladapters.Registry
	// Store persists the runtime enabled-state overlay.
	Store *internaladapters.Store
	// Installer manages on-disk weights (install/remove/custom) and install state.
	Installer *internaladapters.Installer
	// Models resolves a model id → its architecture for ListCompatibleAdapters.
	Models *internalmodels.Registry
	// Jobs submits the durable download job (may be nil in read-only wiring).
	Jobs JobSubmitter
	// Inspector inspects an import source (HF repo id / URL / local path) for the
	// guided import flow (InspectAdapterSource / ImportAdapter). Defaults to a real
	// hfmeta.HFClient; tests inject a fake.
	Inspector internalhfmeta.Fetcher
	// EstimateInstallSeconds estimates an adapter-download ETA for submitted jobs.
	// Defaults to adapters.EstimateInstallSeconds.
	EstimateInstallSeconds EstimateInstallSecondsFunc
	Logger                 *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the AdaptersService handler. A nil required dep is a
// wiring bug surfaced as an internal error at call time.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.EstimateInstallSeconds == nil {
		d.EstimateInstallSeconds = internaladapters.EstimateInstallSeconds
	}
	if d.Inspector == nil {
		d.Inspector = &internalhfmeta.HFClient{}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) overlay(ctx context.Context) (map[string]bool, error) {
	if h.deps.Store == nil {
		return nil, nil
	}
	return h.deps.Store.LoadOverlay(ctx)
}

// installStates returns the install record per adapter id (empty when no Installer).
func (h *connectHandler) installStates(ctx context.Context) (map[string]internaladapters.InstallRecord, error) {
	out := map[string]internaladapters.InstallRecord{}
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

// customAdapters returns operator-registered / imported entries (nil-safe).
func (h *connectHandler) customAdapters(ctx context.Context) ([]internaladapters.Adapter, error) {
	if h.deps.Installer == nil || h.deps.Installer.Custom == nil {
		return nil, nil
	}
	return h.deps.Installer.Custom.List(ctx)
}

// viewFor builds the runtime adapterView for one adapter.
func viewFor(a internaladapters.Adapter, custom bool, overlay map[string]bool, installs map[string]internaladapters.InstallRecord) adapterView {
	v := adapterView{custom: custom, enabled: internaladapters.EffectiveEnabled(a, overlay)}
	if r, ok := installs[a.ID]; ok {
		rec := r
		v.install = &rec
	}
	return v
}

// resolve finds an adapter by id in the seed or custom store, reporting whether it
// is custom.
func (h *connectHandler) resolve(ctx context.Context, id string) (internaladapters.Adapter, bool, bool) {
	if a, ok := h.deps.Registry.ByID(id); ok {
		return a, false, true
	}
	customs, err := h.customAdapters(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.resolve custom: %v", err)
		return internaladapters.Adapter{}, false, false
	}
	for _, a := range customs {
		if a.ID == id {
			return a, true, true
		}
	}
	return internaladapters.Adapter{}, false, false
}

func (h *connectHandler) ListAdapters(ctx context.Context, req *connect.Request[adaptersv1.ListAdaptersRequest]) (*connect.Response[adaptersv1.ListAdaptersResponse], error) {
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListAdapters overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load adapter state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListAdapters installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}

	kind := internaladapters.Kind(strings.TrimSpace(req.Msg.GetKind()))
	arch := internalmodels.Architecture(strings.TrimSpace(req.Msg.GetArchitecture()))

	resp := &adaptersv1.ListAdaptersResponse{Adapters: make([]*adaptersv1.Adapter, 0, len(h.deps.Registry.Adapters()))}
	appendIfMatch := func(a internaladapters.Adapter, custom bool) {
		if kind != "" && a.Kind != kind {
			return
		}
		if arch != "" && a.Architecture != arch {
			return
		}
		resp.Adapters = append(resp.Adapters, domainToProto(a, viewFor(a, custom, overlay, installs)))
	}
	for _, a := range h.deps.Registry.Adapters() {
		appendIfMatch(a, false)
	}
	customs, err := h.customAdapters(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListAdapters custom: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load custom adapters"))
	}
	for _, a := range customs {
		appendIfMatch(a, true)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetAdapter(ctx context.Context, req *connect.Request[adaptersv1.GetAdapterRequest]) (*connect.Response[adaptersv1.GetAdapterResponse], error) {
	a, custom, ok := h.resolve(ctx, req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("adapter not found"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.GetAdapter overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load adapter state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.GetAdapter installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}
	return connect.NewResponse(&adaptersv1.GetAdapterResponse{
		Adapter: domainToProto(a, viewFor(a, custom, overlay, installs)),
	}), nil
}

func (h *connectHandler) SetAdapterEnabled(ctx context.Context, req *connect.Request[adaptersv1.SetAdapterEnabledRequest]) (*connect.Response[adaptersv1.SetAdapterEnabledResponse], error) {
	id := req.Msg.GetId()
	a, custom, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("adapter not found"))
	}
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("adapter state store unavailable"))
	}
	if err := h.deps.Store.SetEnabled(ctx, id, req.Msg.GetEnabled()); err != nil {
		h.deps.Logger.Printf("adapters.SetAdapterEnabled: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist adapter state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.SetAdapterEnabled installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}
	v := viewFor(a, custom, nil, installs)
	v.enabled = req.Msg.GetEnabled()
	return connect.NewResponse(&adaptersv1.SetAdapterEnabledResponse{
		Adapter: domainToProto(a, v),
	}), nil
}

func (h *connectHandler) InstallAdapter(ctx context.Context, req *connect.Request[adaptersv1.InstallAdapterRequest]) (*connect.Response[adaptersv1.InstallAdapterResponse], error) {
	if h.deps.Installer == nil || h.deps.Jobs == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("adapter installation unavailable"))
	}
	id := req.Msg.GetId()
	a, _, ok := h.resolve(ctx, id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("adapter not found"))
	}

	if h.deps.Installer.Installed(ctx, id) {
		return connect.NewResponse(&adaptersv1.InstallAdapterResponse{
			AlreadyInstalled: true,
			SizeMbApprox:     int32(a.SizeMBApprox),
		}), nil
	}

	payload, err := json.Marshal(internaladapters.InstallPayload{AdapterID: id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("encode install request"))
	}
	eta := h.deps.EstimateInstallSeconds(a.SizeMBApprox)
	job, err := h.deps.Jobs.Submit(ctx, internaljobs.Spec{
		Operation:        internaladapters.InstallJobOperation,
		Lane:             internaljobs.LaneCPU,
		Payload:          payload,
		EstimatedSeconds: eta,
	})
	if err != nil {
		h.deps.Logger.Printf("adapters.InstallAdapter submit: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("submit install job"))
	}
	return connect.NewResponse(&adaptersv1.InstallAdapterResponse{
		JobId:        job.ID,
		EtaSeconds:   int32(eta),
		SizeMbApprox: int32(a.SizeMBApprox),
	}), nil
}

func (h *connectHandler) RemoveAdapter(ctx context.Context, req *connect.Request[adaptersv1.RemoveAdapterRequest]) (*connect.Response[adaptersv1.RemoveAdapterResponse], error) {
	if h.deps.Installer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("adapter management unavailable"))
	}
	id := req.Msg.GetId()
	if _, _, ok := h.resolve(ctx, id); !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("adapter not found"))
	}
	if err := h.deps.Installer.Remove(ctx, id); err != nil {
		h.deps.Logger.Printf("adapters.RemoveAdapter: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("remove adapter"))
	}
	return connect.NewResponse(&adaptersv1.RemoveAdapterResponse{Removed: true}), nil
}

func (h *connectHandler) ListCompatibleAdapters(ctx context.Context, req *connect.Request[adaptersv1.ListCompatibleAdaptersRequest]) (*connect.Response[adaptersv1.ListCompatibleAdaptersResponse], error) {
	arch := internalmodels.Architecture(strings.TrimSpace(req.Msg.GetArchitecture()))
	if modelID := strings.TrimSpace(req.Msg.GetModelId()); modelID != "" {
		if h.deps.Models == nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("model registry unavailable"))
		}
		m, ok := h.deps.Models.ByID(modelID)
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
		}
		arch = m.Architecture
	}
	if arch == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a model_id or architecture is required"))
	}

	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListCompatibleAdapters overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load adapter state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListCompatibleAdapters installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}

	resp := &adaptersv1.ListCompatibleAdaptersResponse{Architecture: string(arch)}
	for _, a := range h.deps.Registry.Compatible(arch) {
		resp.Adapters = append(resp.Adapters, domainToProto(a, viewFor(a, false, overlay, installs)))
	}
	// Merge compatible operator-registered custom entries.
	customs, err := h.customAdapters(ctx)
	if err != nil {
		h.deps.Logger.Printf("adapters.ListCompatibleAdapters custom: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load custom adapters"))
	}
	for _, a := range customs {
		if a.CompatibleWith(arch) {
			resp.Adapters = append(resp.Adapters, domainToProto(a, viewFor(a, true, overlay, installs)))
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DoctorCatalog(_ context.Context, _ *connect.Request[adaptersv1.DoctorCatalogRequest]) (*connect.Response[adaptersv1.DoctorCatalogResponse], error) {
	report := h.deps.Registry.DoctorCatalog()
	// Fold in the fetch-strategy lint over ALL rows (incl. disabled) so a
	// source-less adapter surfaces here, not at a user's failed install.
	lint := h.deps.Registry.RegistryLint()
	report.Findings = append(report.Findings, lint.Findings...)
	if !lint.OK {
		report.OK = false
	}
	return connect.NewResponse(doctorReportToProto(report)), nil
}

// addCustomError maps custom-registration errors to actionable Connect codes.
func addCustomError(err error) error {
	switch {
	case errors.Is(err, internaladapters.ErrCustomShadowsSeed),
		errors.Is(err, internaladapters.ErrLocalPathMissing):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
