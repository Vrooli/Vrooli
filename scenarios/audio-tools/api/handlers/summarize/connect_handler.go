package summarize

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

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
	start := time.Now()
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	row := store.UsageRow{
		OperationID:  opID,
		EmittedAt:    time.Now().UTC(),
		Capability:   "summarize",
		Operation:    "summarize",
		LatencyMs:    float64(time.Since(start).Milliseconds()),
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
	row.PromptTokens = int32(res.PromptTokens)
	row.OutputTokens = int32(res.OutputTokens)
	if h.deps.Usage != nil {
		h.deps.Usage.Enqueue(row)
	}
	return connect.NewResponse(&summv1.SummarizeResponse{
		Text:         res.Text,
		PromptTokens: int32(res.PromptTokens),
		OutputTokens: int32(res.OutputTokens),
		ProviderTier: protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:   res.ProviderID,
		ModelId:      res.ModelID,
		LatencyMs:    float64(res.Latency.Milliseconds()),
	}), nil
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
