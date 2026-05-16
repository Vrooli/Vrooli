// Package summarize hosts the SummarizeService Connect-RPC handler.
package summarize

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type Deps struct {
	Chain  *summarizechain.Chain
	Logger *log.Logger
}

type connectHandler struct {
	summconnect.UnimplementedSummarizeServiceHandler
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
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	if err != nil {
		return nil, mapChainError(err)
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

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "summarize.summarize", Path: "/vrooli.audio_tools.v1.summarize.SummarizeService/Summarize",
		Method: "POST", Summary: "Summarize text via the summarization provider chain",
		Category: "summarize",
	},
}

func Module(chain *summarizechain.Chain, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := summconnect.NewSummarizeServiceHandler(NewConnectHandler(Deps{Chain: chain, Logger: logger}))
	return module.Module{
		Name: "summarize",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
