// Package audio hosts the AudioProcessingService Connect-RPC handler.
package audio

import (
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

type Deps struct {
	Logger logx.Logger
}

// DefaultMaxRequestBytes bounds one transform request before Connect decodes
// it. Audio transforms accept a whole payload in a single unary message, and
// Connect otherwise buffers the entire body: without a ceiling a client can
// exhaust process memory with one upload. 256 MiB covers a ~190 MiB audio file
// after base64 expansion and is generous for the scenario's largest fixture
// (a 112 MiB 61-minute WAV).
const DefaultMaxRequestBytes = 256 << 20

// Module returns the audio domain's contribution to the API. logger is
// required; a nil value panics so a forgotten wire-up surfaces at boot.
func Module(logger logx.Logger) modulekit.Module {
	return ModuleWithLimit(logger, DefaultMaxRequestBytes)
}

// ModuleWithLimit is Module with an explicit request-body ceiling. The limit
// is a parameter so the transport test can prove the rejection path without
// allocating the production-sized payload.
func ModuleWithLimit(logger logx.Logger, maxRequestBytes int) modulekit.Module {
	if logger == nil {
		panic("audio.Module requires logger")
	}
	if maxRequestBytes <= 0 {
		panic("audio.ModuleWithLimit requires a positive maxRequestBytes")
	}
	connectPath, h := audioconnect.NewAudioProcessingServiceHandler(
		NewConnectHandler(Deps{Logger: logger}),
		connect.WithReadMaxBytes(maxRequestBytes),
	)
	return modulekit.Module{
		Name: "audio",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "audio.transcode", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Transcode", Method: "POST", Summary: "Transcode audio (Connect-RPC)", Category: "audio"},
	{ID: "audio.trim", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Trim", Method: "POST", Category: "audio"},
	{ID: "audio.merge", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Merge", Method: "POST", Category: "audio"},
	{ID: "audio.split", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Split", Method: "POST", Category: "audio"},
	{ID: "audio.fade", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Fade", Method: "POST", Category: "audio"},
	{ID: "audio.volume", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Volume", Method: "POST", Category: "audio"},
	{ID: "audio.normalize", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Normalize", Method: "POST", Category: "audio"},
	{ID: "audio.extract_metadata", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/ExtractMetadata", Method: "POST", Category: "audio"},
}
