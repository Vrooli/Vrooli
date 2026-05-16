// Package audio hosts the AudioProcessingService Connect-RPC handler.
//
// Most processing methods are stubs returning Unimplemented until the next
// phase wires them to internal/audio. Transcode is implemented end-to-end
// because the existing internal/audio/transcode.go ports cleanly.
package audio

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	audioconnect.UnimplementedAudioProcessingServiceHandler
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Transcode(ctx context.Context, req *connect.Request[audiov1.TranscodeRequest]) (*connect.Response[audiov1.TranscodeResponse], error) {
	if len(req.Msg.Audio) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("audio bytes are required"))
	}
	out, err := intaudio.Transcode(ctx, req.Msg.Audio)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&audiov1.TranscodeResponse{
		Audio:       out,
		ContentType: "audio/wav",
	}), nil
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "audio.transcode", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Transcode", Method: "POST", Summary: "Transcode audio (Connect-RPC)", Category: "audio"},
	{ID: "audio.transcode_multipart", Path: "/api/v1/audio/transcode", Method: "POST", Summary: "Multipart audio transcode", Category: "audio",
		RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "Audio bytes via multipart form-data."}},
	{ID: "audio.trim", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Trim", Method: "POST", Category: "audio"},
	{ID: "audio.merge", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Merge", Method: "POST", Category: "audio"},
	{ID: "audio.split", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Split", Method: "POST", Category: "audio"},
	{ID: "audio.fade", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Fade", Method: "POST", Category: "audio"},
	{ID: "audio.volume", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Volume", Method: "POST", Category: "audio"},
	{ID: "audio.normalize", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/Normalize", Method: "POST", Category: "audio"},
	{ID: "audio.extract_metadata", Path: "/vrooli.audio_tools.v1.audio.AudioProcessingService/ExtractMetadata", Method: "POST", Category: "audio"},
}

func Module(logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := audioconnect.NewAudioProcessingServiceHandler(NewConnectHandler(Deps{Logger: logger}))
	return module.Module{
		Name: "audio",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
