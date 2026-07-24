// Package summarize hosts the SummarizeService Connect-RPC handler.
package summarize

import (
	"context"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type Deps struct {
	Chain               *summarizechain.Chain
	Logger              logx.Logger
	Clock               clock.Clock
	GetSummarizeConfig  func() intsumm.SummarizeConfig
	SetSummarizeConfig  func(intsumm.SummarizeConfig)
	ListSummarizeModels func(context.Context) ([]intsumm.SummarizeModelInfo, error)
	Usage               UsageRecorder
}

// UsageRecorder is the summarize transport's narrow usage submission port.
type UsageRecorder interface{ Enqueue(store.UsageRow) }

func Module(chain *summarizechain.Chain, getCfg func() intsumm.SummarizeConfig, setCfg func(intsumm.SummarizeConfig), listModels func(context.Context) ([]intsumm.SummarizeModelInfo, error), logger logx.Logger, clk clock.Clock, usage UsageRecorder) modulekit.Module {
	if logger == nil {
		panic("summarize.Module requires logger")
	}
	if clk == nil {
		panic("summarize.Module requires clock")
	}
	connectPath, connectHandler := summconnect.NewSummarizeServiceHandler(NewConnectHandler(Deps{
		Chain:               chain,
		Logger:              logger,
		Clock:               clk,
		GetSummarizeConfig:  getCfg,
		SetSummarizeConfig:  setCfg,
		ListSummarizeModels: listModels,
		Usage:               usage,
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
