package tts

import (
	"log"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/module"
	inttts "audio-tools/internal/tts"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// Module returns the TTS domain's contribution to the API.
func Module(chain *ttschain.Chain, summarize *summarizechain.Chain, svc *inttts.Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := ttsconnect.NewTTSServiceHandler(NewConnectHandler(Deps{
		Chain:          chain,
		SummarizeChain: summarize,
		TTSService:     svc,
		Logger:         logger,
	}))
	return module.Module{
		Name: "tts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns an empty SQL string. TTS settings storage uses scenario
// state, not the SQL system schema, so EnsureSchemas has nothing to do here.
func Schema() string { return "" }
