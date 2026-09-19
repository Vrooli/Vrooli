package selection

import (
	"log"
	"net/http"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"
	internalselection "image-tools/internal/selection"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection/selection_v1connect"
)

// Module returns the selection domain's contribution: the SelectionService
// discovery + contextual-edit compiler handler (Connect-RPC) plus the REST
// multipart segment edge (POST /api/v1/selection/segment). Segmentation runs
// synchronously (built-in region-grow) and is recorded as a terminal durable
// job; the produced mask is stored via the blob store.
func Module(blobs BlobStore, jobs *internaljobs.Manager, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	rest := &Deps{
		Service: internalselection.NewService(),
		Store:   blobs,
		Jobs:    jobs,
		Guard:   storage.DefaultGuard(),
		Logger:  logger,
	}
	connectPath, connectHandler := selectionconnect.NewSelectionServiceHandler(NewConnectHandler())
	return module.Module{
		Name: "selection",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/selection/segment", rest.segmentHandler).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the selection domain owns no tables; recorded jobs live in
// the jobs domain's schema.
func Schema() string { return "" }
