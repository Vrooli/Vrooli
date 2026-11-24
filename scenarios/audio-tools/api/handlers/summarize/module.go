// Package summarize hosts the SummarizeService Connect-RPC handler.
package summarize

import (
	"context"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"connectrpc.com/connect"

	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type Deps struct {
	Chain               *summarizechain.Chain
	Logger              logx.Logger
	Clock               schedule.Clock
	GetSummarizeConfig  func() intsumm.SummarizeConfig
	SetSummarizeConfig  func(intsumm.SummarizeConfig)
	ListSummarizeModels func(context.Context) ([]intsumm.SummarizeModelInfo, error)
	IsDeclaredLocalRole func(context.Context, string) (bool, error)
	Usage               UsageRecorder
}

// UsageRecorder is the summarize transport's narrow usage submission port.
type UsageRecorder interface{ Enqueue(store.UsageRow) }

// DefaultMaxRequestBytes bounds one summarize message before Connect decodes
// it. Summarize input is text; 4 MiB is far above any realistic document while
// preventing an unbounded unary body from being buffered.
const DefaultMaxRequestBytes = 4 << 20

func Module(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), listModels func(context.Context) ([]intsumm.SummarizeModelInfo, error), logger logx.Logger, clk schedule.Clock, usage UsageRecorder) modulekit.Module {
	return ModuleWithRoleResolver(chain, getCfg, setCfg, listModels, nil, logger, clk, usage)
}

// ModuleWithRoleResolver provides the local-role inventory seam to focused
// tests and embedding owners. Production passes nil and resolves via Gateway.
func ModuleWithRoleResolver(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), listModels func(context.Context) ([]intsumm.SummarizeModelInfo, error), roleResolver func(context.Context, string) (bool, error), logger logx.Logger, clk schedule.Clock, usage UsageRecorder) modulekit.Module {
	return moduleWithLimitAndRoleResolver(chain, getCfg, setCfg, listModels, roleResolver, logger, clk, usage, DefaultMaxRequestBytes)
}

// ModuleWithLimit is Module with an explicit request-body ceiling.
func ModuleWithLimit(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), listModels func(context.Context) ([]intsumm.SummarizeModelInfo, error), logger logx.Logger, clk schedule.Clock, usage UsageRecorder, maxRequestBytes int) modulekit.Module {
	return moduleWithLimitAndRoleResolver(chain, getCfg, setCfg, listModels, nil, logger, clk, usage, maxRequestBytes)
}

func moduleWithLimitAndRoleResolver(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), listModels func(context.Context) ([]intsumm.SummarizeModelInfo, error), roleResolver func(context.Context, string) (bool, error), logger logx.Logger, clk schedule.Clock, usage UsageRecorder, maxRequestBytes int) modulekit.Module {
	if logger == nil {
		panic("summarize.Module requires logger")
	}
	if clk == nil {
		panic("summarize.Module requires clock")
	}
	if maxRequestBytes <= 0 {
		panic("summarize.ModuleWithLimit requires a positive maxRequestBytes")
	}
	connectPath, connectHandler := summconnect.NewSummarizeServiceHandler(NewConnectHandler(Deps{
		Chain:               chain,
		Logger:              logger,
		Clock:               clk,
		GetSummarizeConfig:  getCfg,
		SetSummarizeConfig:  setCfg,
		ListSummarizeModels: listModels,
		IsDeclaredLocalRole: roleResolver,
		Usage:               usage,
	}), connect.WithReadMaxBytes(maxRequestBytes))
	return modulekit.Module{
		Name: "summarize",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

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
	{
		ID: "summarize.list_models", Path: "/vrooli.audio_tools.v1.summarize.SummarizeService/ListSummarizeModels",
		Method: "POST", Summary: "List local and recommended summarizer models",
		Category: "summarize",
	},
}
