package tts

import (
	"log"

	intsumm "audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/clock"
	"audio-tools/internal/modulekit"
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
	Clock          clock.Clock
	Cache          *inttts.Cache
	ConfigStore    TTSConfigRepository
	Playback       PlaybackRepository
}

// Module returns the TTS domain's contribution to the API.
func Module(d Deps) modulekit.Module {
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
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
