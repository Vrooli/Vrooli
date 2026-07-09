package lifecycle

import (
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lifecycleconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle/lifecycle_v1connect"
)

func Module(logger *log.Logger) module.Module {
	engine, err := templateengine.New("")
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Printf("lifecycle module: template engine unavailable: %v", err)
	}
	handler := NewConnectHandler(engine)
	lifecyclePath, lifecycleHandler := lifecycleconnect.NewTemplateLifecycleServiceHandler(handler)
	designPath, designHandler := lifecycleconnect.NewDesignKitServiceHandler(handler)
	return module.Module{
		Name: "lifecycle",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r,
				connectx.ServiceMount{Path: lifecyclePath, Handler: lifecycleHandler},
				connectx.ServiceMount{Path: designPath, Handler: designHandler},
			)
		},
		Endpoints: Endpoints,
	}
}
