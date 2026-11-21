package vectorgenerations

import (
	"log"

	vectorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/vector"
	vectorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/vector/vector_v1connect"
	"source-ledger/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
)

func Module(service lifecycle, logger *log.Logger) module.Module {
	path, handler := vectorconnect.NewVectorGenerationServiceHandler(NewConnectHandler(service, logger))
	return module.Module{
		Name: "vector-generations",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

var ProtoFile = vectorv1.File_source_ledger_v1_vector_generations_proto
