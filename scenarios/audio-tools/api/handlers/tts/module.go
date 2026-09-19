package tts

import (
	"context"

	intsumm "audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"
	inttts "audio-tools/internal/tts"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// DefaultCredentialResolver returns a configured server-side BYOK
// credential. Explicit request credentials take priority, and resolved
// secrets never enter health or capability responses.
type DefaultCredentialResolver func(ctx context.Context, capability string) (provider, key string, ok bool)

// Deps wires the seams the TTS handler needs.
type Deps struct {
	Chain             *ttschain.Chain
	SummarizeChain    *intsumm.Chain
	TTSService        *inttts.Service
	Engine            *audioformat.Engine
	Logger            logx.Logger
	Clock             schedule.Clock
	Usage             UsageRecorder
	DefaultCredential DefaultCredentialResolver
	Cache             *inttts.Cache
	ConfigStore       TTSConfigRepository
	Playback          PlaybackRepository
}

// UsageRecorder is the TTS transport's narrow usage submission port.
type UsageRecorder interface{ Enqueue(store.UsageRow) }

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
