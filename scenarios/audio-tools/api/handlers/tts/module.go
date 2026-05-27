package tts

import (
	intsumm "audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// Deps wires the seams the TTS handler needs.
type Deps struct {
	Chain          *ttschain.Chain
	SummarizeChain *intsumm.Chain
	TTSService     *inttts.Service
	Engine         *audioformat.Engine
	Logger         logx.Logger
	Clock          clock.Clock
	Usage          usagereport.Recorder
	Cache          *inttts.Cache
	ConfigStore    TTSConfigRepository
	Playback       PlaybackRepository
}

// Module returns the TTS domain's contribution to the API. Deps.Logger
// and Deps.Clock are required seams; nil values panic via the
// NewConnectHandler check.
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
