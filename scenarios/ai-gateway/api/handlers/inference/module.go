package inference

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"

	"ai-gateway/internal/inference"
	"ai-gateway/internal/module"
)

var ProtoFile = inferencev1.File_ai_gateway_v1_inference_inference_proto

type Deps struct {
	Service *inference.Service
}

func Module(deps Deps) module.Module {
	if deps.Service == nil {
		deps.Service = inference.NewService(nil)
	}
	connectPath, connectHandler := inferenceconnect.NewInferenceServiceHandler(NewConnectHandler(deps.Service))
	return module.Module{
		Name: "inference",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
