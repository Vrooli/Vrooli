package stt

import (
	"log"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/module"
	intvoice "audio-tools/internal/voice"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func Module(chain *sttchain.Chain, voice *intvoice.Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := sttconnect.NewSTTServiceHandler(NewConnectHandler(Deps{
		Chain:  chain,
		Voice:  voice,
		Logger: logger,
	}))
	return module.Module{
		Name: "stt",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
