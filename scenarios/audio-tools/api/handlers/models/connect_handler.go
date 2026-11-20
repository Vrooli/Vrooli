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

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
)

// Deps is the handler's narrow dependency set. Catalog is required; a handler
// without a registry cannot answer and would otherwise return an empty list
// that looks like "no models exist".
type Deps struct {
	Catalog *modelcatalog.Catalog
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
func (h *connectHandler) ListModels(_ context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	msg := req.Msg
	capability, err := parseCapabilityFilter(msg.GetCapability())
	if err != nil {
		return nil, err
	}
	language := strings.TrimSpace(msg.GetLanguage())
	platform := platformFromProto(msg.GetPlatform())

	defaults := h.deps.Catalog.DefaultsByCapability(platform)
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
		records = append(records, toProtoModel(model, defaults))
	}
	return connect.NewResponse(&modelsv1.ListModelsResponse{Models: records}), nil
}

// ResolveModel returns the model a caller should use. Commercial requests fail
// closed: a non-commercial or unknown-licence record is never returned, even
// when it ranks highest.
func (h *connectHandler) ResolveModel(_ context.Context, req *connect.Request[modelsv1.ResolveModelRequest]) (*connect.Response[modelsv1.ResolveModelResponse], error) {
	capability, err := parseCapabilityFilter(req.Msg.GetCapability())
	if err != nil {
		return nil, err
	}
	if capability == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("capability is required"))
	}
	language := strings.TrimSpace(req.Msg.GetLanguage())
	platform := platformFromProto(req.Msg.GetPlatform())

	var (
		model modelcatalog.Model
		rerr  error
	)
	if req.Msg.GetCommercial() {
		model, rerr = h.deps.Catalog.CommercialDefaultFor(modelcatalog.Capability(capability), language, platform)
	} else {
		model, rerr = h.deps.Catalog.DefaultFor(modelcatalog.Capability(capability), language, platform)
	}
	if rerr != nil {
		if errors.Is(rerr, modelcatalog.ErrNoEligibleModel) || errors.Is(rerr, modelcatalog.ErrNoCommercialModel) {
			return nil, connect.NewError(connect.CodeNotFound, rerr)
		}
		return nil, connect.NewError(connect.CodeInternal, rerr)
	}

	defaults := h.deps.Catalog.DefaultsByCapability(platform)
	return connect.NewResponse(&modelsv1.ResolveModelResponse{
		Model:       toProtoModel(model, defaults),
		Explanation: explain(model, language, req.Msg.GetCommercial()),
	}), nil
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
	}
}

func explain(model modelcatalog.Model, language string, commercial bool) string {
	lang := "any language"
	if language != "" {
		lang = language
	}
	mode := "any licence"
	if commercial {
		mode = "commercial-only"
	}
	return fmt.Sprintf("selected %s for capability=%s language=%s (%s): readiness=%s roles=%s commercial=%s",
		model.ID, model.Capability, lang, mode, model.Readiness,
		strings.Join(model.Roles, ","), model.Commercial)
}

func toProtoModel(model modelcatalog.Model, defaults map[modelcatalog.Capability]modelcatalog.Model) *modelsv1.ModelRecord {
	isDefault := false
	if winner, ok := defaults[model.Capability]; ok {
		isDefault = winner.ID == model.ID
	}
	return &modelsv1.ModelRecord{
		Id:              model.ID,
		Name:            model.Name,
		ProviderIds:     append([]string(nil), model.ProviderIDs...),
		EngineIds:       append([]string(nil), model.EngineIDs...),
		Capability:      string(model.Capability),
		Runtime:         model.Runtime,
		Languages:       append([]string(nil), model.Languages...),
		SizeMbApprox:    model.SizeMBApprox,
		Params:          model.Params,
		License:         model.License,
		CommercialUse:   commercialToProto(model.Commercial),
		LicenseNotes:    model.LicenseNotes,
		SourceUrl:       model.Source.URL,
		SourceRepo:      model.Source.Repo,
		Readiness:       readinessToProto(model.Readiness),
		Roles:           append([]string(nil), model.Roles...),
		DefaultEligible: model.DefaultEligible(),
		IsDefault:       isDefault,
		AllowDefault:    model.AllowDefault,
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
