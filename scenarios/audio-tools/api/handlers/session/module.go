// Package session hosts the SessionService Connect-RPC handler.
package session

import (
	"log"

	"audio-tools/internal/modulekit"
	intsession "audio-tools/internal/session"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session/session_v1connect"
)

type Deps struct {
	Registry *intsession.Registry
	Logger   *log.Logger
}

func Module(registry *intsession.Registry, logger *log.Logger) modulekit.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, h := sessconnect.NewSessionServiceHandler(NewConnectHandler(Deps{Registry: registry, Logger: logger}))
	return modulekit.Module{
		Name: "session",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "session.open", Path: "/vrooli.audio_tools.v1.session.SessionService/OpenSession", Method: "POST", Summary: "Open a voice session", Category: "session"},
	{ID: "session.close", Path: "/vrooli.audio_tools.v1.session.SessionService/CloseSession", Method: "POST", Category: "session"},
	{ID: "session.send_text", Path: "/vrooli.audio_tools.v1.session.SessionService/SendText", Method: "POST", Category: "session"},
	{ID: "session.send_cancel", Path: "/vrooli.audio_tools.v1.session.SessionService/SendCancel", Method: "POST", Summary: "Explicit barge-in", Category: "session"},
	{ID: "session.subscribe", Path: "/vrooli.audio_tools.v1.session.SessionService/Subscribe", Method: "POST", Summary: "Server-streaming session events", Category: "session"},
}
