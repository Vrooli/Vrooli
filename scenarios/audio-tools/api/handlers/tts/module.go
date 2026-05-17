package tts

import (
	"log"

	intsumm "audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"
	inttts "audio-tools/internal/tts"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// Deps wires the seams the TTS handler needs.
type Deps struct {
	Chain          *ttschain.Chain
	SummarizeChain *intsumm.Chain
	TTSService     *inttts.Service
	Logger         *log.Logger
	Cache          *inttts.Cache
	ConfigStore    *store.TTSConfigStore
	Playback       *store.PlaybackStore
}

// Module returns the TTS domain's contribution to the API.
func Module(d Deps) modulekit.Module {
	connectPath, h := ttsconnect.NewTTSServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "tts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
