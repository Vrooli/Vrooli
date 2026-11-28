package inventory

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory/inventory_v1connect"

	"ai-gateway/internal/module"
)

var ProtoFile = inventoryv1.File_ai_gateway_v1_inventory_inventory_proto

func Module(deps Deps) module.Module {
	connectPath, connectHandler := inventoryconnect.NewInventoryServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "inventory",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
