package stt

import (
	"net/http"

	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func Module(d Deps) modulekit.Module {
	connectPath, h := sttconnect.NewSTTServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "stt",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
			r.Handle("/api/v1/voice/transcribe", MultipartTranscribeHandler(d.Chain)).Methods(http.MethodPost)
			r.Handle("/api/v1/voice/stream", StreamWSHandler(d)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
