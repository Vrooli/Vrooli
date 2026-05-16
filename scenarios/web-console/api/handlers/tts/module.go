// Package tts is the HTTP-handler home for the tts domain. It exposes
// the generated Connect-RPC TTSService (proto schema:
// packages/proto/schemas/web-console/v1/tts).
package tts

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts/tts_v1connect"

	"web-console/internal/module"
	inttts "web-console/internal/tts"
)

type (
	Service              = inttts.HandlerService
	Config               = inttts.Config
	ConfigPatch          = inttts.ConfigPatch
	SummarizeConfig      = inttts.SummarizeConfig
	SummarizeConfigPatch = inttts.SummarizeConfigPatch
	AppendResult         = inttts.AppendResult
	ClientAck            = inttts.ClientAck
	PlaybackEvent        = inttts.PlaybackEvent
	Status               = inttts.Status
	SynthesizeInput      = inttts.SynthesizeInput
	SynthesizeResult     = inttts.SynthesizeResult
	CacheLookup          = inttts.CacheLookup
	Voice                = inttts.Voice
)

// Module wires the tts domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := ttsconnect.NewTTSServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "tts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
