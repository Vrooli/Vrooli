package routing

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

	"ai-gateway/internal/module"
	internalrouting "ai-gateway/internal/routing"
)

var ProtoFile = routingv1.File_ai_gateway_v1_routing_routing_proto

func Module(deps Deps) module.Module {
	connectPath, connectHandler := routingconnect.NewRoutingServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "routing",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return internalrouting.Schema() }
