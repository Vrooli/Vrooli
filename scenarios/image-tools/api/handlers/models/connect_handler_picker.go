package models

import (
	"context"
	"errors"
	"sort"
	"strings"

	internaladapters "image-tools/internal/adapters"
	internalai "image-tools/internal/ai"
	internalbackends "image-tools/internal/backends"
	internalcaps "image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"
	internalresolver "image-tools/internal/resolver"

	"connectrpc.com/connect"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

const bytesPerGB = 1 << 30

// GetHostSummary returns this machine's AI-relevant hardware snapshot so the
// model-catalog UI can render hardware-fit affirmatively ("Runs on your GPU")
// rather than as a static requirement chip.
func (h *connectHandler) GetHostSummary(ctx context.Context, _ *connect.Request[modelsv1.GetHostSummaryRequest]) (*connect.Response[modelsv1.GetHostSummaryResponse], error) {
	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.GetHostSummary probe: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host hardware"))
	}
	return connect.NewResponse(&modelsv1.GetHostSummaryResponse{Host: hostSummaryToProto(host)}), nil
}

// ListOperationModels returns every model serving an operation, each annotated
// for THIS host (hardware fit + backend readiness + a single ready_state). It is
// the data source for the in-product model picker: where SelectModel answers
// "which one would run", this answers "here is the full menu and exactly what
// each one needs to become usable". It is read-only and never executes anything.
func (h *connectHandler) ListOperationModels(ctx context.Context, req *connect.Request[modelsv1.ListOperationModelsRequest]) (*connect.Response[modelsv1.ListOperationModelsResponse], error) {
	op := strings.TrimSpace(req.Msg.GetOperation())
	if op == "" || !h.deps.Registry.IsOperation(op) {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalmodels.ErrUnknownOperation)
	}

	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListOperationModels probe: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host hardware"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListOperationModels overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	installs, err := h.installStates(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListOperationModels installs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load install state"))
	}

	// Backend software readiness, keyed by provider/backend family name.
	providerStatus := map[string]internalbackends.BackendStatus{}
	if h.deps.Backends != nil {
		for _, b := range h.deps.Backends.DoctorForModels(ctx, h.deps.Registry.Models()).Backends {
			providerStatus[b.Name] = b
		}
	}

	// Candidate models: the FULL menu for the op — every model whose effective op
	// set contains it, native OR architecture-derived (proven or not), so a base
	// checkpoint shows "inpaint ⚠ via workflow" instead of disappearing. Each
	// carries its EffectiveOp (support/technique/caveat/Ready) for annotation.
	candidates := h.deps.Registry.OpCandidates(op)
	customMap := map[string]bool{}
	if customs, cerr := h.customModels(ctx); cerr == nil {
		for _, m := range customs {
			for _, eo := range m.EffectiveOps() {
				if eo.Op == op {
					candidates = append(candidates, internalmodels.OpCandidate{Model: m, Effective: eo})
					customMap[m.ID] = true
					break
				}
			}
		}
	}

	// The model that would actually be selected on this host (for the "selected"
	// flag + reason). A selection error (nothing runnable) leaves it empty.
	selectedID, selectedReason := "", ""
	if sel, serr := h.deps.Registry.Select(internalmodels.SelectRequest{
		Operation: op,
		Host:      host,
	}, h.deps.Registry.EnabledWithOverlay(overlay)); serr == nil {
		selectedID, selectedReason = sel.Model.ID, sel.Reason
	}

	toolCache := map[string]*toolTier{}
	out := &modelsv1.ListOperationModelsResponse{
		Operation:      op,
		Host:           hostSummaryToProto(host),
		SelectedId:     selectedID,
		SelectedReason: selectedReason,
		Candidates:     make([]*modelsv1.CandidateModel, 0, len(candidates)),
	}
	for _, cand := range candidates {
		m := cand.Model
		fit := internalmodels.Fit(m, host)
		fitClass := internalmodels.FitClass(m, host, fit)
		enabled := internalmodels.EffectiveEnabled(m, overlay)
		installed := !m.RequiresWeights()
		if rec, ok := installs[m.ID]; ok && rec.Installed {
			installed = true
		}
		backend := h.backendReadiness(ctx, m, providerStatus, toolCache)
		var smokeStatus internalmodels.SmokeStatus
		if h.deps.Installer != nil && installed {
			dir := h.deps.Installer.ModelDir(m.ID)
			if rec, ok := installs[m.ID]; ok && rec.Path != "" {
				dir = rec.Path
			}
			smokeStatus = h.deps.Installer.SmokeStatusFor(m, dir)
		}
		eo := cand.Effective
		// A derived op whose technique is not yet proven is honestly un-offerable
		// (derived_pipeline_unproven) regardless of weights/backend/hardware — it is
		// shown so the user understands the capability exists, never run.
		ready := readyState(installed, enabled, fitClass, backend, smokeStatus)
		if eo.Support == "derived" && !eo.Ready {
			ready = "derived_pipeline_unproven"
		}
		out.Candidates = append(out.Candidates, &modelsv1.CandidateModel{
			Model:      domainToProto(m, viewFor(m, customMap[m.ID], overlay, installs)),
			Fit:        fitToProto(fit, fitClass),
			Backend:    backend,
			ReadyState: ready,
			Selected:   m.ID == selectedID,
			Support:    eo.Support,
			Technique:  eo.Technique,
			Caveat:     eo.Caveat,
		})
	}
	sortCandidates(out.Candidates)
	return connect.NewResponse(out), nil
}

// ExplainResolution returns the explicit Resolution for an operation on this
// host — which model would run, native-vs-derived support + technique + caveat,
// the backend tier, and the operation's safety consent weight — without
// executing anything. It is the read-only `--explain` / dry-run surface over the
// same resolver the AI submit edge uses.
func (h *connectHandler) ExplainResolution(ctx context.Context, req *connect.Request[modelsv1.ExplainResolutionRequest]) (*connect.Response[modelsv1.ExplainResolutionResponse], error) {
	op := strings.TrimSpace(req.Msg.GetOperation())
	if op == "" || !h.deps.Registry.IsOperation(op) {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalmodels.ErrUnknownOperation)
	}
	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ExplainResolution probe: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host hardware"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ExplainResolution overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	override := req.Msg.GetModelId()
	if override == "" && h.deps.OpDefaults != nil {
		if pinned, derr := h.deps.OpDefaults.Get(ctx, op); derr == nil {
			override = pinned
		}
	}
	adapterReqs := explainAdapterRequests(req.Msg.GetAdapters())
	var adapterEnabled func(id string) bool
	if len(adapterReqs) > 0 && h.deps.AdapterEnabled != nil {
		fn, aerr := h.deps.AdapterEnabled(ctx)
		if aerr != nil {
			h.deps.Logger.Printf("models.ExplainResolution adapter overlay: %v", aerr)
			return nil, connect.NewError(connect.CodeInternal, errors.New("load adapter state"))
		}
		adapterEnabled = fn
	}
	res, err := internalresolver.New(h.deps.Registry, h.deps.Backends).Resolve(ctx, internalresolver.Request{
		Operation:        op,
		ModelOverride:    override,
		Host:             host,
		AllowBYOK:        req.Msg.GetAllowByok(),
		IsEnabled:        h.deps.Registry.EnabledWithOverlay(overlay),
		Adapters:         adapterReqs,
		AdapterByID:      h.deps.AdapterByID,
		AdapterEnabled:   adapterEnabled,
		AdapterInstalled: h.deps.AdapterInstalled,
	})
	if err != nil {
		return nil, selectError(err)
	}
	return connect.NewResponse(&modelsv1.ExplainResolutionResponse{Resolution: resolutionToProto(res)}), nil
}

// explainAdapterRequests converts the explain request's AdapterRef list to the
// resolver's conditioning request shape.
func explainAdapterRequests(refs []*modelsv1.AdapterRef) []internaladapters.AdapterRequest {
	if len(refs) == 0 {
		return nil
	}
	out := make([]internaladapters.AdapterRequest, 0, len(refs))
	for _, r := range refs {
		out = append(out, internaladapters.AdapterRequest{
			ID:                   r.GetAdapterId(),
			Scale:                r.GetScale(),
			ConditioningImageKey: r.GetConditioningImageKey(),
			PreprocessorOverride: internaladapters.Preprocessor(r.GetPreprocessorOverride()),
		})
	}
	return out
}

func resolutionToProto(r internalresolver.Resolution) *modelsv1.Resolution {
	return &modelsv1.Resolution{
		Operation:     r.Operation,
		ModelId:       r.Model.ID,
		ModelName:     r.Model.Name,
		Support:       r.Support,
		Technique:     r.Technique,
		PipelineClass: r.PipelineClass,
		Caveat:        r.Caveat,
		Weight:        r.Weight,
		Tier:          r.Tier,
		GpuViable:     r.GPUViable,
		Warnings:      r.Warnings,
		Adapters:      resolvedAdaptersToProto(r.Adapters),
	}
}

func resolvedAdaptersToProto(in []internaladapters.ResolvedAdapter) []*modelsv1.ResolvedAdapter {
	if len(in) == 0 {
		return nil
	}
	out := make([]*modelsv1.ResolvedAdapter, 0, len(in))
	for _, a := range in {
		out = append(out, &modelsv1.ResolvedAdapter{
			Id:                   a.ID,
			Name:                 a.Name,
			Kind:                 string(a.Kind),
			Architecture:         string(a.Architecture),
			Scale:                a.Scale,
			Weight:               string(a.Weight),
			Preprocessor:         string(a.Preprocessor),
			ConditioningImageKey: a.ConditioningImageKey,
		})
	}
	return out
}

// toolTier is the cached install-posture classification for one host tool.
type toolTier struct {
	tier        string
	remediation string
	manualHint  string
}

// backendReadiness resolves the provisioning posture of the host program model m
// needs to execute, separate from whether its weights are downloaded.
func (h *connectHandler) backendReadiness(ctx context.Context, m internalmodels.Model, providerStatus map[string]internalbackends.BackendStatus, cache map[string]*toolTier) *modelsv1.BackendReadiness {
	br := &modelsv1.BackendReadiness{Backend: m.Backend}
	if st, ok := providerStatus[m.Backend]; ok {
		br.Ready = st.Available
		br.Detail = st.Detail
	}
	tool, hasTool := internalai.HostToolForProvider(m.Backend)
	if !hasTool {
		// In-process (builtin/computed/library-go) or cloud provider: nothing to
		// fetch. Ready ⇒ builtin; otherwise there is no install path here.
		if br.Ready || !m.RequiresWeights() {
			br.Ready = true
			br.InstallTier = "builtin"
		} else {
			br.InstallTier = "unsupported"
		}
		return br
	}
	br.HostTool = tool
	tt := h.toolTierFor(ctx, tool, cache)
	br.InstallTier = tt.tier
	br.Remediation = tt.remediation
	br.ManualHint = tt.manualHint
	return br
}

// toolTierFor classifies (and caches) how a host tool can be provisioned, via a
// dry-run inspect through the platform host-tool installer.
func (h *connectHandler) toolTierFor(ctx context.Context, tool string, cache map[string]*toolTier) *toolTier {
	if t, ok := cache[tool]; ok {
		return t
	}
	t := &toolTier{tier: "manual"} // safe default if we cannot probe
	if h.deps.Ensurer != nil {
		if st, err := h.deps.Ensurer.Inspect(ctx, tool); err == nil {
			t = classifyTier(tool, st)
		} else {
			h.deps.Logger.Printf("models.ListOperationModels inspect %q: %v", tool, err)
		}
	}
	cache[tool] = t
	return t
}

// classifyTier maps a host-install inspect verdict to a picker install tier.
func classifyTier(tool string, st *cliv1.CliHostInstallStatus) *toolTier {
	notes := strings.TrimSpace(strings.Join(st.GetNotes(), "\n"))
	switch st.GetExecutionState() {
	case "would_install", "already_present":
		return &toolTier{tier: "auto", remediation: "vrooli host install " + tool}
	case "manual_action_required":
		return &toolTier{tier: "manual", manualHint: notes}
	case "not_applicable":
		return &toolTier{tier: "not_applicable", manualHint: notes}
	case "unsupported":
		return &toolTier{tier: "unsupported", manualHint: notes}
	default:
		if st.GetSupportClass() == "manual_only" {
			return &toolTier{tier: "manual", manualHint: notes}
		}
		if st.GetSupportClass() == "supported" {
			return &toolTier{tier: "auto", remediation: "vrooli host install " + tool}
		}
		return &toolTier{tier: "manual", manualHint: notes}
	}
}

// readyState collapses fit + enable + install + backend + Python-env/smoke into
// the single verdict the picker styles and acts on. The env/smoke overrides only
// apply to a model whose weights are installed and whose backend is otherwise
// ready — they surface "installed but not runnable" (env not provisioned, or the
// install-time load-smoke failed) BEFORE the user picks the model.
func readyState(installed, enabled bool, fitClass string, br *modelsv1.BackendReadiness, smoke internalmodels.SmokeStatus) string {
	switch fitClass {
	case "unsupported_os":
		return "unsupported"
	case "no_gpu", "insufficient_vram":
		return "insufficient"
	}
	if !enabled {
		return "disabled"
	}
	backendReady := br.GetReady()
	backendAuto := br.GetInstallTier() == "auto"
	switch {
	case installed && backendReady:
		if smoke.Applicable {
			if !smoke.EnvProvisioned {
				return "env_not_provisioned"
			}
			if smoke.HasVerdict && !smoke.Verdict.Pass {
				return "smoke_failed"
			}
		}
		return "ready"
	case !installed && backendReady:
		return "needs_model_install"
	case installed && !backendReady:
		if backendAuto {
			return "needs_backend"
		}
		return "needs_backend_manual"
	default: // !installed && !backendReady
		if !backendAuto {
			return "needs_backend_manual"
		}
		return "needs_both"
	}
}

// readyRank orders ready_states from most- to least-usable for picker display.
func readyRank(state string) int {
	switch state {
	case "ready":
		return 0
	case "needs_model_install":
		return 1
	case "needs_backend":
		return 2
	case "needs_both":
		return 3
	case "needs_backend_manual":
		return 4
	case "env_not_provisioned":
		return 5
	case "smoke_failed":
		return 6
	case "disabled":
		return 7
	case "derived_pipeline_unproven":
		return 8
	case "insufficient":
		return 9
	case "unsupported":
		return 10
	default:
		return 11
	}
}

// sortCandidates orders the picker menu: the selected model first, then by
// usability (ready_state rank), then GPU-viable, then by name for determinism.
func sortCandidates(cs []*modelsv1.CandidateModel) {
	sort.SliceStable(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		if a.GetSelected() != b.GetSelected() {
			return a.GetSelected()
		}
		if ra, rb := readyRank(a.GetReadyState()), readyRank(b.GetReadyState()); ra != rb {
			return ra < rb
		}
		if a.GetFit().GetGpuViable() != b.GetFit().GetGpuViable() {
			return a.GetFit().GetGpuViable()
		}
		return a.GetModel().GetName() < b.GetModel().GetName()
	})
}

func fitToProto(fit internalmodels.HardwareFit, fitClass string) *modelsv1.ModelFit {
	return &modelsv1.ModelFit{
		Runnable:        fit.Runnable,
		GpuViable:       fit.GPUViable,
		FitClass:        fitClass,
		VramShortfallGb: int32(fit.VRAMShortfallGB),
	}
}

func hostSummaryToProto(h internalcaps.Host) *modelsv1.HostSummary {
	s := &modelsv1.HostSummary{
		HasGpu:   h.HasGPU(),
		GpuCount: int32(len(h.GPUs)),
		RamGb:    int32(h.TotalMemoryBytes / bytesPerGB),
		CpuCores: int32(h.Cores),
		Os:       h.OS,
		Arch:     h.Arch,
	}
	if len(h.GPUs) > 0 {
		s.GpuName = h.GPUs[0].Name
	}
	if total, known := h.MaxVRAMBytes(); known {
		s.VramKnown = true
		s.VramTotalGb = int32(total / bytesPerGB)
		if free, ok := h.MaxFreeVRAMBytes(); ok {
			s.VramFreeGb = int32(free / bytesPerGB)
		}
	}
	return s
}
