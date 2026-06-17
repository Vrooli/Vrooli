package ai

import (
	"log"
	"net/http"

	internalai "image-tools/internal/ai"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/module"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
)

// Module returns the ai domain's contribution: the AIService discovery handler
// (Connect-RPC) plus the REST multipart submit edge (POST /api/v1/ai/{operation}).
// Execution is asynchronous on the durable job Manager's GPU-serialized lane.
func Module(engine *internalai.Engine, registry *models.Registry, store *storage.Store, jobs *internaljobs.Manager, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	rest := &Deps{
		Engine: engine,
		Store:  store,
		Jobs:   jobs,
		Guard:  storage.DefaultGuard(),
		Logger: logger,
	}
	connectPath, connectHandler := aiconnect.NewAIServiceHandler(NewConnectHandler(registry))
	return module.Module{
		Name: "ai",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/ai/{operation}", rest.submitHandler).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the ai domain owns no tables; submitted jobs live in the
// jobs domain's schema.
func Schema() string { return "" }
