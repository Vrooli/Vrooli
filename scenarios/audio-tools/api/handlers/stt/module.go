package stt

import (
	"net/http"

	"audio-tools/internal/modulekit"
	"audio-tools/internal/stt/session"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func Module(d Deps) modulekit.Module {
	if d.Sessions == nil {
		d.Sessions = session.NewRegistry(0)
	}
	h := NewConnectHandler(d)
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(h)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(h)
	return modulekit.Module{
		Name: "stt",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r,
				connectx.ServiceMount{Path: runtimePath, Handler: runtimeHandler},
				connectx.ServiceMount{Path: adminPath, Handler: adminHandler},
			)
			r.Handle("/api/v1/voice/transcribe", MultipartTranscribeHandler(d)).Methods(http.MethodPost)
			r.Handle("/api/v1/voice/stream", StreamWSHandler(d)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
