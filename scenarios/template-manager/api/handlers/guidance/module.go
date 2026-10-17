package guidance

import (
	"log"

	guidancesvc "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/guidance"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	guidanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance/guidance_v1connect"
)

func Module(logger *log.Logger) module.Module {
	path, handler := guidanceconnect.NewGuidanceServiceHandler(NewConnectHandler(Deps{
		Service: guidancesvc.NewService(nil),
		Logger:  logger,
	}))
	return module.Module{
		Name: "guidance",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
