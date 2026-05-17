package summarize

import (
	"context"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/byok/envelope"
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
		Level:          req.Msg.Level,
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
		ProviderTier: string(res.Tier),
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

func (h *connectHandler) UpdateSummarizeConfig(_ context.Context, req *connect.Request[summv1.UpdateSummarizeConfigRequest]) (*connect.Response[summv1.UpdateSummarizeConfigResponse], error) {
	if h.deps.GetSummarizeConfig == nil || h.deps.SetSummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("summarize config accessor not configured"))
	}
	patch := intsumm.SummarizeConfigPatch{}
	if req.Msg.HasEnabled {
		v := req.Msg.Enabled
		patch.Enabled = &v
	}
	if req.Msg.HasCharThreshold {
		v := int(req.Msg.CharThreshold)
		patch.CharThreshold = &v
	}
	if req.Msg.HasLevel {
		v := req.Msg.Level
		patch.Level = &v
	}
	if req.Msg.HasModel {
		v := req.Msg.Model
		patch.Model = &v
	}
	if req.Msg.HasTimeoutSeconds {
		v := int(req.Msg.TimeoutSeconds)
		patch.TimeoutSeconds = &v
	}
	updated := patch.Apply(h.deps.GetSummarizeConfig())
	h.deps.SetSummarizeConfig(updated)
	return connect.NewResponse(&summv1.UpdateSummarizeConfigResponse{Config: toProtoSummarizeConfig(updated)}), nil
}
