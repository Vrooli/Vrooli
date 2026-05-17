// Package summarize hosts the SummarizeService Connect-RPC handler.
package summarize

import (
	"log"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/modulekit"
	intsumm "audio-tools/internal/summarize"
	"audio-tools/internal/usagereport"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type Deps struct {
	Chain              *summarizechain.Chain
	Logger             *log.Logger
	GetSummarizeConfig func() intsumm.SummarizeConfig
	SetSummarizeConfig func(intsumm.SummarizeConfig)
	Usage              usagereport.Recorder
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
