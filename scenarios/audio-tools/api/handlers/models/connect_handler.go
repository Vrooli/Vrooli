// Package models hosts the ModelRegistryService Connect-RPC handler. It is a
// projection of internal/modelcatalog: the registry owns the facts and the
// selection policy, this package owns the wire shape and the error codes.
package models

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/modelpromotion"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
)

// Deps is the handler's narrow dependency set. Catalog is required; a handler
// without a registry cannot answer and would otherwise return an empty list
// that looks like "no models exist".
type Deps struct {
	Catalog               *modelcatalog.Catalog
	QualificationEvidence modelpromotion.QualificationEvidenceStore
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler. It panics when Catalog is nil so a
// forgotten wire-up fails at boot instead of at the first request.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Catalog == nil {
		panic("models.NewConnectHandler requires Deps.Catalog")
	}
	return &connectHandler{deps: d}
}

// ListModels returns the registry, optionally filtered. Ordering is canonical
// capability order, then model id; the registry's default per capability is
// flagged rather than hoisted so a client can render one stable table.
func (h *connectHandler) ListModels(ctx context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	msg := req.Msg
	capability, err := parseCapabilityFilter(msg.GetCapability())
	if err != nil {
		return nil, err
	}
	language := strings.TrimSpace(msg.GetLanguage())
	platform := platformFromProto(msg.GetPlatform())

	verifiedFor := h.verificationCache(ctx)
	defaults := h.defaults(language, platform, verifiedFor, msg.GetCommercialOnly())
	records := make([]*modelsv1.ModelRecord, 0, 32)
	for _, model := range h.deps.Catalog.SortedForDisplay() {
		if capability != "" && string(model.Capability) != capability {
			continue
		}
		if !model.Speaks(language) {
			continue
		}
		if msg.GetCommercialOnly() && model.Commercial != modelcatalog.CommercialYes {
			continue
		}
		if msg.GetOnlyReady() && model.Readiness != modelcatalog.ReadinessAvailable {
			continue
		}
		eligible := model.DefaultEligible()
		if !eligible && model.HasRole("default") {
			eligible = verifiedFor(model)
		}
		records = append(records, toProtoModel(model, defaults, eligible))
	}
	return connect.NewResponse(&modelsv1.ListModelsResponse{Models: records}), nil
}

// ResolveModel returns the model a caller should use. Commercial requests fail
// closed: a non-commercial or unknown-licence record is never returned, even
// when it ranks highest.
func (h *connectHandler) ResolveModel(ctx context.Context, req *connect.Request[modelsv1.ResolveModelRequest]) (*connect.Response[modelsv1.ResolveModelResponse], error) {
	capability, err := parseCapabilityFilter(req.Msg.GetCapability())
	if err != nil {
		return nil, err
	}
	if capability == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("capability is required"))
	}
	language := strings.TrimSpace(req.Msg.GetLanguage())
	engineID := strings.TrimSpace(req.Msg.GetEngineId())
	platform := platformFromProto(req.Msg.GetPlatform())

	var (
		model modelcatalog.Model
		rerr  error
	)
	verifiedFor := h.verificationCache(ctx)
	if engineID != "" && modelcatalog.Capability(capability) != modelcatalog.CapabilitySTT {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("engine_id is supported only for stt model resolution"))
	}
	if engineID != "" {
		if req.Msg.GetCommercial() {
			model, rerr = h.deps.Catalog.CommercialDefaultForEngineVerified(modelcatalog.Capability(capability), engineID, language, platform, verifiedFor)
		} else {
			model, rerr = h.deps.Catalog.DefaultForEngineVerified(modelcatalog.Capability(capability), engineID, language, platform, verifiedFor)
		}
	} else if req.Msg.GetCommercial() {
		model, rerr = h.deps.Catalog.CommercialDefaultForVerified(modelcatalog.Capability(capability), language, platform, verifiedFor)
	} else {
		model, rerr = h.deps.Catalog.DefaultForVerified(modelcatalog.Capability(capability), language, platform, verifiedFor)
	}
	if rerr != nil {
		if errors.Is(rerr, modelcatalog.ErrNoEligibleModel) || errors.Is(rerr, modelcatalog.ErrNoCommercialModel) {
			return nil, connect.NewError(connect.CodeNotFound, rerr)
		}
		return nil, connect.NewError(connect.CodeInternal, rerr)
	}

	defaults := h.defaults(language, platform, verifiedFor, req.Msg.GetCommercial())
	if engineID != "" {
		defaults[modelcatalog.Capability(capability)] = model
	}
	defaultEligible := model.DefaultEligible() || verifiedFor(model)
	if engineID != "" && model.HasEngineDefault(engineID) && !model.HasRole("default") {
		defaultEligible = model.EngineDefaultEligible() || verifiedFor(model)
	}
	return connect.NewResponse(&modelsv1.ResolveModelResponse{
		Model:       toProtoModel(model, defaults, defaultEligible),
		Explanation: explain(model, language, req.Msg.GetCommercial(), engineID),
	}), nil
}

// verificationCache memoizes owner checks for one request. The experiment
// service is useful even in product-only composition; only the missing facet
// owners remain blockers in modelpromotion.Verify.
func (h *connectHandler) verificationCache(ctx context.Context) func(modelcatalog.Model) bool {
	cache := make(map[string]bool)
	checked := make(map[string]bool)
	return func(model modelcatalog.Model) bool {
		if checked[model.ID] {
			return cache[model.ID]
		}
		checked[model.ID] = true
		resolvers := map[string]modelpromotion.Resolver{}
		if h.deps.QualificationEvidence != nil {
			resolver := modelpromotion.ExperimentResolver{
				Store: h.deps.QualificationEvidence, Catalog: h.deps.Catalog,
			}
			resolvers[modelpromotion.FacetCorpusQuality] = resolver
			resolvers[modelpromotion.FacetLatency] = resolver
		}
		cache[model.ID] = modelpromotion.Verify(ctx, model, resolvers).VerifiedFor(model)
		return cache[model.ID]
	}
}

func (h *connectHandler) defaults(language string, platform modelcatalog.Platform, verifiedFor func(modelcatalog.Model) bool, commercial bool) map[modelcatalog.Capability]modelcatalog.Model {
	defaults := make(map[modelcatalog.Capability]modelcatalog.Model)
	for _, capability := range modelcatalog.DefaultCapabilities {
		var model modelcatalog.Model
		var err error
		if commercial {
			model, err = h.deps.Catalog.CommercialDefaultForVerified(capability, language, platform, verifiedFor)
		} else {
			model, err = h.deps.Catalog.DefaultForVerified(capability, language, platform, verifiedFor)
		}
		if err == nil {
			defaults[capability] = model
		}
	}
	return defaults
}

// parseCapabilityFilter validates the capability vocabulary. An unknown value
// is a caller mistake, not an empty result: silently returning nothing makes a
// typo look like "the registry has no such capability".
func parseCapabilityFilter(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	if _, ok := modelcatalog.CapabilityOrder(modelcatalog.Capability(value)); !ok {
		known := make([]string, 0, len(modelcatalog.Capabilities))
		for _, capability := range modelcatalog.Capabilities {
			known = append(known, string(capability))
		}
		sort.Strings(known)
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown capability %q (known: %s)", raw, strings.Join(known, ", ")))
	}
	return value, nil
}

// platformFromProto maps the caller-supplied host description. A nil message
// means "unconstrained", which is the same zero value the catalog expects.
func platformFromProto(p *modelsv1.Platform) modelcatalog.Platform {
	if p == nil {
		return modelcatalog.Platform{}
	}
	return modelcatalog.Platform{
		OS:          strings.TrimSpace(p.GetOs()),
		Arch:        strings.TrimSpace(p.GetArch()),
		Accelerated: p.GetAccelerated(),
		FreeVRAMGB:  p.GetFreeVramGb(),
		RAMGB:       int(p.GetRamGb()),
		CPULimit:    int(p.GetCpuLimit()),
	}
}

func explain(model modelcatalog.Model, language string, commercial bool, engineID string) string {
	lang := "any language"
	if language != "" {
		lang = language
	}
	mode := "any licence"
	if commercial {
		mode = "commercial-only"
	}
	scope := fmt.Sprintf("capability=%s", model.Capability)
	if engineID != "" {
		scope += fmt.Sprintf(" engine=%s", engineID)
	}
	return fmt.Sprintf("selected %s for %s language=%s (%s): readiness=%s roles=%s commercial=%s",
		model.ID, scope, lang, mode, model.Readiness,
		strings.Join(model.Roles, ","), model.Commercial)
}

func toProtoModel(model modelcatalog.Model, defaults map[modelcatalog.Capability]modelcatalog.Model, verifiedEligible bool) *modelsv1.ModelRecord {
	isDefault := false
	if winner, ok := defaults[model.Capability]; ok {
		isDefault = winner.ID == model.ID
	}
	return &modelsv1.ModelRecord{
		Id:               model.ID,
		Name:             model.Name,
		ProviderIds:      append([]string(nil), model.ProviderIDs...),
		EngineIds:        append([]string(nil), model.EngineIDs...),
		EngineDefaults:   append([]string(nil), model.EngineDefaults...),
		Capability:       string(model.Capability),
		Runtime:          model.Runtime,
		RuntimeModelId:   model.RuntimeModelID,
		ArtifactResource: artifactResource(model.Artifact),
		ArtifactName:     artifactName(model.Artifact),
		Languages:        append([]string(nil), model.Languages...),
		SizeMbApprox:     model.SizeMBApprox,
		Params:           model.Params,
		License:          model.License,
		CommercialUse:    commercialToProto(model.Commercial),
		LicenseNotes:     model.LicenseNotes,
		SourceUrl:        model.Source.URL,
		SourceRepo:       model.Source.Repo,
		Readiness:        readinessToProto(model.Readiness),
		Roles:            append([]string(nil), model.Roles...),
		DefaultEligible:  model.DefaultEligible() || verifiedEligible,
		IsDefault:        isDefault,
		AllowDefault:     model.AllowDefault,
		Hardware: &modelsv1.Hardware{
			CpuCapable:  model.Hardware.CPUCapable,
			GpuRequired: model.Hardware.GPURequired,
			MinVramGb:   model.Hardware.MinVRAMGB,
			MinRamGb:    float64(model.Hardware.MinRAMGB),
			OsArch:      append([]string(nil), model.Hardware.OSArch...),
		},
		Benchmarks: &modelsv1.Benchmarks{
			Wer:   derefFloat(model.Benchmarks.WER),
			Sim:   derefFloat(model.Benchmarks.SIM),
			Rtf:   derefFloat(model.Benchmarks.RTF),
			Notes: model.Benchmarks.Notes,
		},
		Notes: model.Notes,
	}
}

func artifactResource(artifact *modelcatalog.Artifact) string {
	if artifact == nil {
		return ""
	}
	return artifact.Resource
}

func artifactName(artifact *modelcatalog.Artifact) string {
	if artifact == nil {
		return ""
	}
	return artifact.Name
}

func derefFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func readinessToProto(readiness modelcatalog.Readiness) modelsv1.Readiness {
	switch readiness {
	case modelcatalog.ReadinessAvailable:
		return modelsv1.Readiness_READINESS_AVAILABLE
	case modelcatalog.ReadinessCandidate:
		return modelsv1.Readiness_READINESS_CANDIDATE
	case modelcatalog.ReadinessExperimental:
		return modelsv1.Readiness_READINESS_EXPERIMENTAL
	case modelcatalog.ReadinessWatch:
		return modelsv1.Readiness_READINESS_WATCH
	default:
		return modelsv1.Readiness_READINESS_UNSPECIFIED
	}
}

func commercialToProto(use modelcatalog.CommercialUse) modelsv1.CommercialUse {
	switch use {
	case modelcatalog.CommercialYes:
		return modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED
	case modelcatalog.CommercialNo:
		return modelsv1.CommercialUse_COMMERCIAL_USE_DISALLOWED
	case modelcatalog.CommercialUnknown:
		return modelsv1.CommercialUse_COMMERCIAL_USE_UNKNOWN
	default:
		return modelsv1.CommercialUse_COMMERCIAL_USE_UNSPECIFIED
	}
}
