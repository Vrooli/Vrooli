// Package audio hosts the AudioProcessingService Connect-RPC handler.
package audio

import (
	"net/http"

	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

type Deps struct {
	Logger logx.Logger
}

// Module returns the audio domain's contribution to the API. logger is
// required; a nil value panics so a forgotten wire-up surfaces at boot.
func Module(logger logx.Logger) modulekit.Module {
	if logger == nil {
		panic("audio.Module requires logger")
	}
	connectPath, h := audioconnect.NewAudioProcessingServiceHandler(NewConnectHandler(Deps{Logger: logger}))
	return modulekit.Module{
		Name: "audio",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
			r.Handle("/api/v1/audio/transcode", multipartTranscodeHandler()).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "audio.transcode", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Transcode", Method: "POST", Summary: "Transcode audio (Connect-RPC)", Category: "audio"},
	{
		ID: "audio.transcode_multipart", Path: "/api/v1/audio/transcode", Method: "POST", Summary: "Multipart audio transcode", Category: "audio",
		RESTException: &modulekit.RESTException{Reason: modulekit.RESTReasonMultipartUpload, Note: "Audio bytes via multipart form-data."},
	},
	{ID: "audio.trim", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Trim", Method: "POST", Category: "audio"},
	{ID: "audio.merge", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Merge", Method: "POST", Category: "audio"},
	{ID: "audio.split", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Split", Method: "POST", Category: "audio"},
	{ID: "audio.fade", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Fade", Method: "POST", Category: "audio"},
	{ID: "audio.volume", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Volume", Method: "POST", Category: "audio"},
	{ID: "audio.normalize", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Normalize", Method: "POST", Category: "audio"},
	{ID: "audio.extract_metadata", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/ExtractMetadata", Method: "POST", Category: "audio"},
}
