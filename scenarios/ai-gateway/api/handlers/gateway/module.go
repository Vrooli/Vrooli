package gateway

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"
	gatewayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway/gateway_v1connect"

	"ai-gateway/internal/module"
)

var ProtoFile = gatewayv1.File_ai_gateway_v1_gateway_gateway_proto

func Module() module.Module {
	connectPath, connectHandler := gatewayconnect.NewGatewayServiceHandler(NewConnectHandler(Deps{}))
	return module.Module{
		Name: "gateway",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
