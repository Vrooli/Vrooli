package resourcetemplate

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	resourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template/resource_template_v1connect"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
)

func Module(logger *log.Logger) module.Module {
	engine, err := templateengine.New("")
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Printf("resource-template module: template engine unavailable: %v", err)
	}
	handler := NewConnectHandler(engine)
	path, connectHandler := resourceconnect.NewResourceTemplateServiceHandler(handler)
	return module.Module{
		Name: "resource_template",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
