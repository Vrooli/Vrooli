// Package usage hosts the UsageService Connect-RPC handler.
package usage

import (
	"log"

	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

type Deps struct {
	Logger *log.Logger
	Store  *store.UsageStore
}

func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	connectPath, h := usageconnect.NewUsageServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "usage",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "usage.list_recent", Path: "/vrooli.audio_tools.v1.usage.UsageService/ListRecent", Method: "POST", Category: "usage"},
	{ID: "usage.get_summary", Path: "/vrooli.audio_tools.v1.usage.UsageService/GetSummary", Method: "POST", Category: "usage"},
}
