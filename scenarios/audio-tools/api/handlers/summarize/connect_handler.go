package summarize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/protoint"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. All Deps seams
// (Logger, Clock) are required; nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("summarize.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("summarize.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) now() time.Time { return h.deps.Clock.Now() }

func (h *connectHandler) Summarize(ctx context.Context, req *connect.Request[summv1.SummarizeRequest]) (*connect.Response[summv1.SummarizeResponse], error) {
	if h.deps.Chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("summarize chain not configured"))
	}
	env := envelope.FromConnectRequest(req)
	chainReq := summarizechain.Request{
		Text:           req.Msg.Text,
		Level:          protomap.SummarizeLevelFromProto(req.Msg.GetLevel()),
		Model:          req.Msg.Model,
		TimeoutSeconds: int(req.Msg.TimeoutSeconds),
		BYOKProvider:   env.Provider,
		BYOKKey:        env.Key,
		LPBSToken:      env.LPBSToken,
		UserIdentity:   env.UserIdentity,
	}
	opID := req.Header().Get("X-Audio-Operation-ID")
	if opID == "" {
		opID = uuid.NewString()
	}
	start := h.now()
	resp := connect.NewResponse(&summv1.SummarizeResponse{})
	ctx = tiered.WithOnFallback(ctx, func(ev tiered.FallbackEvent) {
		resp.Header().Set("x-audio-tools-fallback",
			fmt.Sprintf("from=%s;to=%s;reason=%s", ev.From.String(), ev.To.String(), ev.Reason))
	})
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	row := store.UsageRow{
		OperationID:  opID,
		EmittedAt:    h.now().UTC(),
		Capability:   "summarize",
		Operation:    "summarize",
		LatencyMs:    float64(h.now().Sub(start).Milliseconds()),
		UserIdentity: chainReq.UserIdentity,
	}
	if err != nil {
		row.Error = err.Error()
		if h.deps.Usage != nil {
			h.deps.Usage.Enqueue(row)
		}
		return nil, mapChainError(err)
	}
	row.ProviderTier = string(res.Tier)
	row.ProviderID = res.ProviderID
	row.ModelID = res.ModelID
	row.PromptTokens = protoint.FromInt64(int64(res.PromptTokens))
	row.OutputTokens = protoint.FromInt64(int64(res.OutputTokens))
	if h.deps.Usage != nil {
		h.deps.Usage.Enqueue(row)
	}
	resp.Msg = &summv1.SummarizeResponse{
		Text:         res.Text,
		PromptTokens: protoint.FromInt64(int64(res.PromptTokens)),
		OutputTokens: protoint.FromInt64(int64(res.OutputTokens)),
		ProviderTier: protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:   res.ProviderID,
		ModelId:      res.ModelID,
		LatencyMs:    float64(res.Latency.Milliseconds()),
	}
	return resp, nil
}

func (h *connectHandler) GetSummarizeConfig(_ context.Context, _ *connect.Request[summv1.GetSummarizeConfigRequest]) (*connect.Response[summv1.GetSummarizeConfigResponse], error) {
	if h.deps.GetSummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("summarize config accessor not configured"))
	}
	cfg := h.deps.GetSummarizeConfig()
	return connect.NewResponse(&summv1.GetSummarizeConfigResponse{Config: toProtoSummarizeConfig(cfg)}), nil
}

var summarizeConfigAllowedPaths = map[string]struct{}{
	"enabled":         {},
	"char_threshold":  {},
	"level":           {},
	"model":           {},
	"timeout_seconds": {},
}

func (h *connectHandler) UpdateSummarizeConfig(_ context.Context, req *connect.Request[summv1.UpdateSummarizeConfigRequest]) (*connect.Response[summv1.UpdateSummarizeConfigResponse], error) {
	if h.deps.GetSummarizeConfig == nil || h.deps.SetSummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("summarize config accessor not configured"))
	}
	mask := req.Msg.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask required"))
	}
	if bad := protomap.MaskPathsOutsideAllowed(mask, summarizeConfigAllowedPaths); len(bad) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown update_mask paths: %v", bad))
	}
	cfg := req.Msg.GetConfig()
	patch := intsumm.SummarizeConfigPatch{}
	if protomap.MaskHas(mask, "enabled") {
		v := cfg.GetEnabled()
		patch.Enabled = &v
	}
	if protomap.MaskHas(mask, "char_threshold") {
		v := int(cfg.GetCharThreshold())
		patch.CharThreshold = &v
	}
	if protomap.MaskHas(mask, "level") {
		v := protomap.SummarizeLevelFromProto(cfg.GetLevel())
		patch.Level = &v
	}
	if protomap.MaskHas(mask, "model") {
		v := cfg.GetModel()
		patch.Model = &v
	}
	if protomap.MaskHas(mask, "timeout_seconds") {
		v := int(cfg.GetTimeoutSeconds())
		patch.TimeoutSeconds = &v
	}
	updated := patch.Apply(h.deps.GetSummarizeConfig())
	h.deps.SetSummarizeConfig(updated)
	return connect.NewResponse(&summv1.UpdateSummarizeConfigResponse{Config: toProtoSummarizeConfig(updated)}), nil
}

func (h *connectHandler) ListSummarizeModels(ctx context.Context, _ *connect.Request[summv1.ListSummarizeModelsRequest]) (*connect.Response[summv1.ListSummarizeModelsResponse], error) {
	if h.deps.ListSummarizeModels == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("summarize model catalog accessor not configured"))
	}
	models, err := h.deps.ListSummarizeModels(ctx)
	if err != nil {
		// Return the recommended catalog even when Ollama is down so settings UI
		// can still explain which models are missing and how to install them.
		h.deps.Logger.Printf("summarize-models: local ollama list unavailable: %v", err)
	}
	out := make([]*summv1.SummarizeModel, 0, len(models))
	for _, model := range models {
		out = append(out, toProtoSummarizeModel(model))
	}
	return connect.NewResponse(&summv1.ListSummarizeModelsResponse{Models: out}), nil
}
