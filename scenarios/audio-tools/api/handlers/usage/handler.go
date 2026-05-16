// Package usage hosts the UsageService Connect-RPC handler.
//
// Stubs until the SQLite usage-row store + async usage reporter land
// (Phase D / Phase G).
package usage

import (
	"log"

	"audio-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	usageconnect.UnimplementedUsageServiceHandler
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "usage.list_recent", Path: "/vrooli.audio_tools.v1.usage.UsageService/ListRecent", Method: "POST", Category: "usage"},
	{ID: "usage.get_summary", Path: "/vrooli.audio_tools.v1.usage.UsageService/GetSummary", Method: "POST", Category: "usage"},
}

func Module(logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := usageconnect.NewUsageServiceHandler(NewConnectHandler(Deps{Logger: logger}))
	return module.Module{
		Name: "usage",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
