// Package summarize hosts the SummarizeService Connect-RPC handler.
package summarize

import (
	"context"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"
	"audio-tools/internal/usagereport"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type Deps struct {
	Chain              *summarizechain.Chain
	Logger             *log.Logger
	GetSummarizeConfig func() intsumm.SummarizeConfig
	SetSummarizeConfig func(intsumm.SummarizeConfig)
	Usage              usagereport.Recorder
}

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
	chainReq := summarizechain.Request{
		Text:           req.Msg.Text,
		Level:          req.Msg.Level,
		Model:          req.Msg.Model,
		TimeoutSeconds: int(req.Msg.TimeoutSeconds),
		BYOKProvider:   req.Header().Get("X-Audio-BYOK-Provider"),
		BYOKKey:        req.Header().Get("X-Audio-BYOK-Key"),
		LPBSToken:      req.Header().Get("X-Audio-LPBS-Token"),
		UserIdentity:   req.Header().Get("X-Audio-User-Identity"),
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

func toProtoSummarizeConfig(c intsumm.SummarizeConfig) *summv1.SummarizeConfig {
	return &summv1.SummarizeConfig{
		Enabled:        c.Enabled,
		CharThreshold:  int32(c.CharThreshold),
		Level:          c.Level,
		Model:          c.Model,
		TimeoutSeconds: int32(c.TimeoutSeconds),
	}
}

func mapChainError(err error) error {
	switch err {
	case summarizechain.ErrInsufficientCredits:
		return connect.NewError(connect.CodeResourceExhausted, err)
	case summarizechain.ErrUnknownBYOKProvider, summarizechain.ErrMissingBYOKProvider:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case summarizechain.ErrAllProvidersFailed:
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "summarize.summarize", Path: "/vrooli.audio_tools.v1.summarize.SummarizeService/Summarize",
		Method: "POST", Summary: "Summarize text via the summarization provider chain",
		Category: "summarize",
	},
	{
		ID: "summarize.get_config", Path: "/vrooli.audio_tools.v1.summarize.SummarizeService/GetSummarizeConfig",
		Method: "POST", Summary: "Get the persisted summarizer configuration",
		Category: "summarize",
	},
	{
		ID: "summarize.update_config", Path: "/vrooli.audio_tools.v1.summarize.SummarizeService/UpdateSummarizeConfig",
		Method: "POST", Summary: "Update the persisted summarizer configuration",
		Category: "summarize",
	},
}

func Module(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), logger *log.Logger, usage usagereport.Recorder) modulekit.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := summconnect.NewSummarizeServiceHandler(NewConnectHandler(Deps{
		Chain:              chain,
		Logger:             logger,
		GetSummarizeConfig: getCfg,
		SetSummarizeConfig: setCfg,
		Usage:              usage,
	}))
	return modulekit.Module{
		Name: "summarize",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
