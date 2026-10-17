package ops

import (
	"log"
	"net/http"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	opsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops/ops_v1connect"
)

// Module returns the ops domain's contribution: the OpsService discovery
// handler (Connect-RPC) plus the two REST edges — the multipart execution edge
// (POST /api/v1/ops/{operation}) and the result-blob serve (GET
// /api/v1/blobs/{key}). Execution runs synchronously (deterministic ops are
// instant) and is recorded as a terminal durable job for uniform observability.
func Module(store *storage.Store, jobs *internaljobs.Manager, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	rest := &Deps{
		Store:  store,
		Jobs:   jobs,
		Guard:  storage.DefaultGuard(),
		Logger: logger,
	}
	connectPath, connectHandler := opsconnect.NewOpsServiceHandler(NewConnectHandler())
	return module.Module{
		Name: "ops",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/ops/{operation}", rest.runHandler).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/blobs/{key:.*}", rest.blobHandler).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the ops domain owns no tables; recorded jobs live in the
// jobs domain's schema.
func Schema() string { return "" }
